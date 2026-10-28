package proxy

import (
	"bytes"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// Tests for upstream_credentials.go: what the gateway removes from a request
// before forwarding it, and the upstream's configured headers.

// headerRecorder is an upstream that keeps the headers of the last request.
type headerRecorder struct {
	mu   sync.Mutex
	last http.Header
	hits int
	srv  *httptest.Server
}

func newHeaderRecorder(t *testing.T) *headerRecorder {
	t.Helper()
	rec := &headerRecorder{}
	rec.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		rec.mu.Lock()
		rec.last = r.Header.Clone()
		rec.hits++
		rec.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(rec.srv.Close)
	return rec
}

func (h *headerRecorder) headers(t *testing.T) http.Header {
	t.Helper()
	h.mu.Lock()
	defer h.mu.Unlock()
	if h.last == nil {
		t.Fatal("the upstream was not contacted")
	}
	return h.last.Clone()
}

// Headers and cookies a client sends that the gateway owns, and others it
// does not. Values are fixtures, not credentials.
func addOwnedAndForeignHeaders(req *http.Request) {
	req.Header.Set("X-Admin-Token", "admin-secret")
	req.Header.Set("X-Tenant-ID", "tenant-fixture")
	req.Header.Set("X-SatGate-Tenant", "tenant-fixture")
	req.Header.Set("X-Satgate-Anything", "fixture")
	req.Header.Add("Cookie", "cloud_session=fixture; theme=dark; satgate_pref=1")
	req.Header.Add("Cookie", "__Host-admin_session=fixture")
	req.Header.Set("X-Request-ID", "req-1")
	req.Header.Set("X-Custom", "kept")
	req.Header.Set("X-Api-Key", "upstream-own-key")
}

func wantOwnedRemovedForeignKept(t *testing.T, what string, got http.Header) {
	t.Helper()
	for name := range got {
		c := http.CanonicalHeaderKey(name)
		if c == "X-Admin-Token" || c == "X-Tenant-Id" || strings.HasPrefix(c, "X-Satgate-") {
			t.Errorf("%s: upstream received gateway header %s", what, name)
		}
	}
	if c := got.Values("Cookie"); len(c) != 1 || c[0] != "theme=dark" {
		t.Errorf("%s: upstream Cookie = %q, want [theme=dark]", what, c)
	}
	for name, want := range map[string]string{"X-Request-Id": "req-1", "X-Custom": "kept", "X-Api-Key": "upstream-own-key"} {
		if got.Get(name) != want {
			t.Errorf("%s: header %s = %q, want it passed on unchanged", what, name, got.Get(name))
		}
	}
}

// On a capability route the Authorization the gateway checked, the headers
// it owns and its cookies stay at the gateway; other headers are passed on.
func TestUpstreamCredentials_CapabilityRouteRemovesConsumedAndOwned(t *testing.T) {
	up := newHeaderRecorder(t)
	gw, _ := newTestGatewayWithUpstream(t, up.srv, "capability")
	token, _ := mintRoot(t, gw)

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	addOwnedAndForeignHeaders(req)
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d: %s", rec.Code, rec.Body.String())
	}
	got := up.headers(t)
	if v := got.Values("Authorization"); len(v) != 0 {
		t.Errorf("upstream received Authorization on a capability route (%d values)", len(v))
	}
	wantOwnedRemovedForeignKept(t, "capability", got)
	// The client's request is not changed.
	if req.Header.Get("Authorization") == "" || req.Header.Get("X-Admin-Token") == "" {
		t.Error("the inbound request's headers were changed")
	}
}

// A raw (scheme-less) capability token and a second Authorization value are
// both removed on a capability route.
func TestUpstreamCredentials_CapabilityRouteRemovesEveryAuthorizationValue(t *testing.T) {
	up := newHeaderRecorder(t)
	gw, _ := newTestGatewayWithUpstream(t, up.srv, "capability")
	token, _ := mintRoot(t, gw)
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.Header.Add("Authorization", token)
	req.Header.Add("Authorization", "Basic dXNlcjpwYXNz")
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	if v := up.headers(t).Values("Authorization"); len(v) != 0 {
		t.Errorf("upstream received %d Authorization values", len(v))
	}
}

// On an L402 route the payment proof (macaroon and preimage) stays at the
// gateway.
func TestUpstreamCredentials_L402RouteRemovesProof(t *testing.T) {
	up := newHeaderRecorder(t)
	gw := newTestGateway(t, newTwoTierL402Config(up.srv.URL))
	paid := payL402At(t, gw, "/cheap/x")
	for _, scheme := range []string{"L402 ", "LSAT "} {
		if scheme == "LSAT " {
			paid = payL402At(t, gw, "/cheap/x")
		}
		req := httptest.NewRequest(http.MethodGet, "/cheap/x", nil)
		req.Header.Set("Authorization", scheme+paid.macaroon+":"+paid.preimage)
		addOwnedAndForeignHeaders(req)
		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", scheme, rec.Code)
		}
		got := up.headers(t)
		if len(got.Values("Authorization")) != 0 {
			t.Errorf("%s: upstream received the payment proof", scheme)
		}
		for _, v := range got {
			for _, s := range v {
				if strings.Contains(s, paid.preimage) {
					t.Errorf("%s: upstream received the preimage", scheme)
				}
			}
		}
		wantOwnedRemovedForeignKept(t, "l402 "+scheme, got)
	}
}

// On a public route the gateway checked nothing. An Authorization value that
// carries a token this gateway issued is removed; any other is the
// upstream's own and is passed on unchanged.
func TestUpstreamCredentials_PublicRouteRemovesOnlyGatewayTokens(t *testing.T) {
	up := newHeaderRecorder(t)
	gw, _ := newTestGatewayWithUpstream(t, up.srv, "public")
	token, _ := mintRoot(t, gw)

	// A token this gateway issued, expired: still the gateway's.
	expired, err := gw.macaroonSvc.Mint("api:read", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	expiredToken := gw.macaroonSvc.Encode(expired)
	// A binary macaroon signed under this gateway's key.
	binary, err := gw.macaroonSvc.EncodeBinary(expired)
	if err != nil {
		t.Fatal(err)
	}
	// A macaroon from a different key is not this gateway's.
	other, err := macaroon.NewService("a-different-root-key-for-tests")
	if err != nil {
		t.Fatal(err)
	}
	foreignMac, err := other.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	foreign := other.Encode(foreignMac)

	cases := []struct {
		name   string
		values []string
		want   []string
	}{
		{"bearer gateway token", []string{"Bearer " + token}, nil},
		{"lowercase scheme", []string{"bearer " + token}, nil},
		{"bare gateway token", []string{token}, nil},
		{"expired gateway token", []string{"Bearer " + expiredToken}, nil},
		{"binary gateway macaroon", []string{"Bearer " + binary}, nil},
		{"l402 gateway macaroon", []string{"L402 " + binary + ":" + strings.Repeat("ab", 32)}, nil},
		{"lsat gateway macaroon", []string{"LSAT " + binary + ":" + strings.Repeat("ab", 32)}, nil},
		{"upstream api key", []string{"Bearer upstream-own-api-key"}, []string{"Bearer upstream-own-api-key"}},
		{"basic", []string{"Basic dXNlcjpwYXNz"}, []string{"Basic dXNlcjpwYXNz"}},
		{"foreign macaroon", []string{"Bearer " + foreign}, []string{"Bearer " + foreign}},
		{"foreign l402", []string{"L402 " + foreign + ":00"}, []string{"L402 " + foreign + ":00"}},
		{"mixed", []string{"Bearer " + token, "Bearer upstream-own-api-key"}, []string{"Bearer upstream-own-api-key"}},
	}
	for _, c := range cases {
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		for _, v := range c.values {
			req.Header.Add("Authorization", v)
		}
		addOwnedAndForeignHeaders(req)
		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", c.name, rec.Code)
		}
		got := up.headers(t)
		if v := got.Values("Authorization"); strings.Join(v, "\n") != strings.Join(c.want, "\n") {
			t.Errorf("%s: upstream Authorization has %d values, want %d (gateway tokens removed, others kept)", c.name, len(v), len(c.want))
		}
		wantOwnedRemovedForeignKept(t, "public "+c.name, got)
	}
}

// On an identity route the identity token the gateway verified stays at the
// gateway.
func TestUpstreamCredentials_IdentityRouteRemovesVerifiedToken(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "OKP", "crv": "Ed25519", "kid": "idp-1", "x": base64.RawURLEncoding.EncodeToString(pub),
		}}})
	}))
	defer jwks.Close()
	up := newHeaderRecorder(t)
	gw := newTestGateway(t, &config.Config{Admin: config.AdminConfig{Token: "admin-secret"}})
	staged := agentPost(t, gw, "/api/agent/policy", map[string]any{
		"name": "who", "kind": "identity", "path_prefix": "/who", "upstream_url": up.srv.URL,
		"identity_issuer": "https://idp.example", "identity_jwks": jwks.URL, "issuer": "https://issuer.example",
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	token := signTestJWT(t, priv, "idp-1", "https://idp.example")
	if sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID, "authorization": "Bearer " + token}); sim.Code != http.StatusOK {
		t.Fatalf("identity simulate: %d", sim.Code)
	}
	if agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": policy.ID}).Code != http.StatusOK {
		t.Fatal("identity promote failed")
	}
	req := httptest.NewRequest(http.MethodGet, "/who", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	addOwnedAndForeignHeaders(req)
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	got := up.headers(t)
	if len(got.Values("Authorization")) != 0 {
		t.Error("upstream received the identity token the gateway verified")
	}
	wantOwnedRemovedForeignKept(t, "identity", got)
}

// An upstream's configured headers are set on every request to it, after the
// removal, and replace a client header of the same name.
func TestUpstreamCredentials_ConfiguredHeadersAppliedAndWin(t *testing.T) {
	for _, kind := range []string{"capability", "public"} {
		up := newHeaderRecorder(t)
		cfg := &config.Config{
			Server: config.ServerConfig{Listen: ":8080"},
			Admin:  config.AdminConfig{Token: "admin-secret"},
			Upstreams: map[string]config.Upstream{"backend": {URL: up.srv.URL, Headers: map[string]string{
				"authorization":  "Bearer upstream-configured",
				"X-Upstream-Key": "configured-key",
				"X-Custom":       "configured",
			}}},
			Routes: []config.Route{{Name: "r", Match: config.RouteMatch{PathPrefix: "/api/"}, Upstream: "backend",
				Policy: config.RoutePolicy{Kind: kind}}},
		}
		gw := newTestGateway(t, cfg)
		token, _ := mintRoot(t, gw)
		req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
		if kind == "capability" {
			req.Header.Set("Authorization", "Bearer "+token)
		} else {
			req.Header.Set("Authorization", "Bearer client-own-key")
		}
		req.Header.Set("X-Custom", "from-client")
		req.Header.Set("X-Other", "from-client")
		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, req)
		if rec.Code != http.StatusOK {
			t.Fatalf("%s: status %d", kind, rec.Code)
		}
		got := up.headers(t)
		if v := got.Values("Authorization"); len(v) != 1 || v[0] != "Bearer upstream-configured" {
			t.Errorf("%s: upstream Authorization is not the configured one (%d values)", kind, len(v))
		}
		if v := got.Values("X-Custom"); len(v) != 1 || v[0] != "configured" {
			t.Errorf("%s: X-Custom = %q, want the configured value only", kind, v)
		}
		if got.Get("X-Upstream-Key") != "configured-key" || got.Get("X-Other") != "from-client" {
			t.Errorf("%s: configured or client header missing: %v", kind, got)
		}
	}
}

// A configured header SatGate owns, or one that cannot be sent as written, is
// refused when the configuration is loaded and when the gateway is built.
func TestUpstreamCredentials_ConfiguredOwnedHeaderRefused(t *testing.T) {
	bad := []map[string]string{
		{"X-Admin-Token": "x"},
		{"x-tenant-id": "x"},
		{"X-SatGate-Tenant": "x"},
		{"x-satgate-anything": "x"},
		{"Bad Name": "x"},
		{"": "x"},
		{"X-Ok": "line\r\nX-Injected: 1"},
		{"X-Dup": "a", "x-dup": "b"},
	}
	for i, headers := range bad {
		var b strings.Builder
		b.WriteString("upstreams:\n  backend:\n    url: https://api.example.com\n    headers:\n")
		for k, v := range headers {
			kj, _ := json.Marshal(k)
			vj, _ := json.Marshal(v)
			b.WriteString("      " + string(kj) + ": " + string(vj) + "\n")
		}
		b.WriteString("routes:\n  - name: r\n    match: {pathPrefix: /api/}\n    upstream: backend\n    policy: {kind: public}\n")
		if _, err := config.ParseYAML([]byte(b.String())); err == nil {
			t.Errorf("case %d: configuration with headers %q was accepted", i, keys(headers))
		}
		cfg := &config.Config{
			Upstreams: map[string]config.Upstream{"backend": {URL: "http://127.0.0.1:1", Headers: headers}},
			Routes: []config.Route{{Name: "r", Match: config.RouteMatch{PathPrefix: "/api/"}, Upstream: "backend",
				Policy: config.RoutePolicy{Kind: "public"}}},
		}
		if _, err := New(Options{Config: cfg, Macaroon: newTestMacaroonService(t)}); err == nil {
			t.Errorf("case %d: gateway built with headers %q", i, keys(headers))
		}
	}
	ok := "upstreams:\n  backend:\n    url: https://api.example.com\n    headers:\n      Authorization: Bearer upstream-configured\n      X-Upstream-Key: k\n" +
		"routes:\n  - name: r\n    match: {pathPrefix: /api/}\n    upstream: backend\n    policy: {kind: public}\n"
	if _, err := config.ParseYAML([]byte(ok)); err != nil {
		t.Errorf("an upstream's own Authorization was refused: %v", err)
	}
}

func keys(m map[string]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// The gateway's own log does not carry the L402 proof it checked.
func TestUpstreamCredentials_L402ProofNotLogged(t *testing.T) {
	var logs bytes.Buffer
	old := zlog.Logger
	oldLevel := zerolog.GlobalLevel()
	zerolog.SetGlobalLevel(zerolog.DebugLevel)
	zlog.Logger = zerolog.New(&logs).Level(zerolog.DebugLevel)
	defer func() { zlog.Logger = old; zerolog.SetGlobalLevel(oldLevel) }()

	up := newHeaderRecorder(t)
	gw := newTestGateway(t, newTwoTierL402Config(up.srv.URL))
	paid := payL402At(t, gw, "/cheap/x")
	logs.Reset()
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusOK {
		t.Fatalf("status %d", code)
	}
	if !strings.Contains(logs.String(), "extractL402Token") {
		t.Fatal("debug log not captured")
	}
	if strings.Contains(logs.String(), paid.preimage) || strings.Contains(logs.String(), paid.macaroon) {
		t.Error("the log carries the L402 proof")
	}
}

// A client cannot keep a configured header from the upstream by naming it in
// its Connection header.
func TestUpstreamCredentials_ConfiguredHeaderSurvivesConnectionHeader(t *testing.T) {
	up := newHeaderRecorder(t)
	cfg := &config.Config{
		Upstreams: map[string]config.Upstream{"backend": {URL: up.srv.URL, Headers: map[string]string{
			"Authorization": "Bearer upstream-configured",
		}}},
		Routes: []config.Route{{Name: "r", Match: config.RouteMatch{PathPrefix: "/api/"}, Upstream: "backend",
			Policy: config.RoutePolicy{Kind: "public"}}},
	}
	gw := newTestGateway(t, cfg)
	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	req.Header.Set("Connection", "authorization, X-Hop")
	req.Header.Set("X-Hop", "dropped as hop-by-hop")
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	if rec.Code != http.StatusOK {
		t.Fatalf("status %d", rec.Code)
	}
	got := up.headers(t)
	if got.Get("Authorization") != "Bearer upstream-configured" {
		t.Error("the configured Authorization did not reach the upstream")
	}
	if got.Get("X-Hop") != "" {
		t.Error("a header the client named in Connection was passed on")
	}
}
