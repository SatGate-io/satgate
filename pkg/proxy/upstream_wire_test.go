package proxy

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// On-the-wire tests: a client sends to the gateway over TCP and a stub
// upstream records what reaches it. Values are fixtures, not credentials.

// requestRecorder is an upstream that keeps the last request it received,
// after reading its body, so its trailers are complete.
type requestRecorder struct {
	mu   sync.Mutex
	last *http.Request
	srv  *httptest.Server
}

func newRequestRecorder(t *testing.T) *requestRecorder {
	t.Helper()
	rec := &requestRecorder{}
	rec.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, _ = io.Copy(io.Discard, r.Body)
		c := r.Clone(r.Context())
		c.Trailer = r.Trailer.Clone()
		rec.mu.Lock()
		rec.last = c
		rec.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(rec.srv.Close)
	return rec
}

func (rr *requestRecorder) take(t *testing.T) *http.Request {
	t.Helper()
	rr.mu.Lock()
	defer rr.mu.Unlock()
	if rr.last == nil {
		t.Fatal("the upstream was not contacted")
	}
	r := rr.last
	rr.last = nil
	return r
}

// issuedForms are Authorization values that each carry a token this
// gateway issued, in every form the gateway accepts one and others a client
// may send. name -> value.
func issuedForms(t *testing.T, gw *Gateway) map[string]string {
	t.Helper()
	tok, _ := mintRoot(t, gw) // JSON macaroon, URL-safe unpadded base64
	tok2, _ := mintRoot(t, gw)
	raw, err := base64.RawURLEncoding.DecodeString(tok)
	if err != nil {
		t.Fatal(err)
	}
	mac, err := gw.macaroonSvc.Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	bin, err := gw.macaroonSvc.EncodeBinary(mac) // binary, standard padded base64
	if err != nil {
		t.Fatal(err)
	}
	binData, err := base64.StdEncoding.DecodeString(bin)
	if err != nil {
		t.Fatal(err)
	}
	jsonStd := base64.StdEncoding.EncodeToString(raw)
	jsonURLPadded := base64.URLEncoding.EncodeToString(raw)
	jsonRawStd := base64.RawStdEncoding.EncodeToString(raw)
	binRaw := base64.RawStdEncoding.EncodeToString(binData)
	binURL := base64.RawURLEncoding.EncodeToString(binData)
	pre := strings.Repeat("ab", 32)
	return map[string]string{
		// Schemes and letter case.
		"Bearer":           "Bearer " + tok,
		"bearer":           "bearer " + tok,
		"BEARER":           "BEARER " + tok,
		"Basic scheme":     "Basic " + tok,
		"Digest scheme":    "Digest " + tok,
		"Token scheme":     "Token " + tok,
		"Macaroon scheme":  "Macaroon " + tok,
		"Bearer twice":     "Bearer Bearer " + tok,
		"bare":             tok,
		"bare binary":      bin,
		"Bearer binary":    "Bearer " + bin,
		"Basic binary":     "Basic " + bin,
		"Digest binary":    "Digest " + bin,
		"lowercase binary": "bearer " + bin,
		// Whitespace.
		"Bearer tab":        "Bearer\t" + tok,
		"Bearer two spaces": "Bearer  " + tok,
		"Bearer space tab":  "Bearer \t " + tok,
		"Bearer nbsp":       "Bearer\u00a0" + tok,
		"Bearer tab binary": "Bearer\t" + bin,
		// Token encodings.
		"json standard padded": "Bearer " + jsonStd,
		"json url padded":      "Bearer " + jsonURLPadded,
		"json standard raw":    "Bearer " + jsonRawStd,
		"binary raw":           "Bearer " + binRaw,
		"binary url":           "Bearer " + binURL,
		"binary percent":       "Bearer " + url.QueryEscape(bin),
		"bare json standard":   jsonStd,
		"bare binary url":      binURL,
		// Quoted and as a parameter.
		"quoted":             `Bearer "` + tok + `"`,
		"param":              "Bearer token=" + tok,
		"l402 params":        `L402 macaroon="` + bin + `", preimage="` + pre + `"`,
		"macaroon parameter": `Macaroon id=1, token="` + tok + `"`,
		// L402 and LSAT with a preimage.
		"L402":           "L402 " + tok + ":" + pre,
		"LSAT":           "LSAT " + tok + ":" + pre,
		"l402 lowercase": "l402 " + tok + ":" + pre,
		"lsat binary":    "lsat " + bin + ":" + pre,
		"L402 binary":    "L402 " + bin + ":" + pre,
		"L402 raw":       "L402 " + binRaw + ":" + pre,
		"L402 tab":       "L402\t" + tok + ":" + pre,
		"L402 no proof":  "L402 " + tok,
		"L402 bare":      tok + ":" + pre,
		// Inside other encodings.
		"Basic user:token": "Basic " + base64.StdEncoding.EncodeToString([]byte("user:"+tok)),
		"Basic token:":     "Basic " + base64.StdEncoding.EncodeToString([]byte(tok+":")),
		// Comma-joined values that are all the gateway's.
		"two tokens":        "Bearer " + tok + ", Bearer " + tok2,
		"token and l402":    "Bearer " + tok + ",L402 " + bin + ":" + pre,
		"two bare comma":    tok + "," + tok2,
		"trailing comma":    "Bearer " + tok + ",",
		"semicolon":         "Bearer " + tok + ";",
		"token in brackets": "Bearer <" + tok + ">",
	}
}

// foreignForms are Authorization values that carry no token this gateway
// issued; each must reach the upstream byte for byte.
func foreignForms(t *testing.T) map[string]string {
	t.Helper()
	other, err := macaroon.NewService("a-different-root-key-for-tests")
	if err != nil {
		t.Fatal(err)
	}
	fm, err := other.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	foreign := other.Encode(fm)
	foreignBin, err := other.EncodeBinary(fm)
	if err != nil {
		t.Fatal(err)
	}
	pre := strings.Repeat("cd", 32)
	return map[string]string{
		"api key":          "Bearer upstream-own-api-key",
		"basic":            "Basic dXNlcjpwYXNz",
		"foreign macaroon": "Bearer " + foreign,
		"foreign bare":     foreign,
		"foreign l402":     "L402 " + foreignBin + ":" + pre,
		"foreign lsat":     "LSAT " + foreign + ":" + pre,
		"jwt":              "Bearer eyJhbGciOiJFZERTQSJ9.eyJzdWIiOiJ4In0.c2lnbmF0dXJlLWZpeHR1cmU",
		"digest":           `Digest username="u", realm="r", nonce="n", uri="/", response="6629fae49393a05397450978507c4ef1"`,
		"two foreign":      "Bearer upstream-key-a, Basic dXNlcjpwYXNz",
		"basic foreign":    "Basic " + base64.StdEncoding.EncodeToString([]byte("user:"+foreign)),
		"tab foreign":      "Bearer\tupstream-own-api-key",
	}
}

// sendAuthorization sends values as Authorization header lines through
// front and returns the request the upstream received.
func sendAuthorization(t *testing.T, front string, up *requestRecorder, values ...string) *http.Request {
	t.Helper()
	req, err := http.NewRequest(http.MethodGet, front+"/api/data", nil)
	if err != nil {
		t.Fatal(err)
	}
	for _, v := range values {
		req.Header.Add("Authorization", v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	return up.take(t)
}

// On public and chargeback routes, an Authorization value that carries a
// token this gateway issued, in any form, does not reach the upstream.
func TestUpstreamCredentials_IssuedTokenRemovedInEveryAuthorizationForm(t *testing.T) {
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind, func(t *testing.T) {
			up := newRequestRecorder(t)
			gw, _ := newTestGatewayWithUpstream(t, up.srv, kind)
			front := httptest.NewServer(gw)
			defer front.Close()
			for name, v := range issuedForms(t, gw) {
				got := sendAuthorization(t, front.URL, up, v)
				if n := len(got.Header.Values("Authorization")); n != 0 {
					t.Errorf("%s: upstream received %d Authorization values carrying a gateway token", name, n)
				}
			}
		})
	}
}

// On public and chargeback routes, a credential the gateway did not issue
// reaches the upstream unchanged, alone, beside a gateway token in another
// value, or joined to one with a comma.
func TestUpstreamCredentials_ForeignCredentialForwardedUnchanged(t *testing.T) {
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind, func(t *testing.T) {
			up := newRequestRecorder(t)
			gw, _ := newTestGatewayWithUpstream(t, up.srv, kind)
			tok, _ := mintRoot(t, gw)
			front := httptest.NewServer(gw)
			defer front.Close()
			for name, v := range foreignForms(t) {
				got := sendAuthorization(t, front.URL, up, v)
				if g := got.Header.Values("Authorization"); len(g) != 1 || g[0] != v {
					t.Errorf("%s: foreign credential was changed or dropped (%d values)", name, len(g))
				}
			}
			mixed := []struct {
				name string
				send []string
				want []string
			}{
				{"separate values", []string{"Bearer " + tok, "Basic Zm9yZWlnbg=="}, []string{"Basic Zm9yZWlnbg=="}},
				{"gateway first, comma", []string{"Bearer " + tok + ", Basic Zm9yZWlnbg=="}, []string{"Basic Zm9yZWlnbg=="}},
				{"gateway last, comma", []string{"Basic Zm9yZWlnbg==,Bearer " + tok}, []string{"Basic Zm9yZWlnbg=="}},
				{"digest beside", []string{`Digest username="u", realm="r", response="abc"`, "Bearer " + tok}, []string{`Digest username="u", realm="r", response="abc"`}},
				{"digest then token, comma", []string{`Digest username="u", realm="r", Bearer ` + tok}, []string{`Digest username="u", realm="r"`}},
				{"l402 and api key", []string{"L402 " + tok + ":" + strings.Repeat("ab", 32), "Bearer upstream-key"}, []string{"Bearer upstream-key"}},
			}
			for _, m := range mixed {
				got := sendAuthorization(t, front.URL, up, m.send...)
				if g := got.Header.Values("Authorization"); strings.Join(g, "\n") != strings.Join(m.want, "\n") {
					t.Errorf("%s: upstream Authorization has %d values, want only the foreign credential", m.name, len(g))
				}
			}
		})
	}
}

// Reviewer case: a gateway-issued token under the Basic or Digest scheme,
// or after "Bearer" and a tab, is removed on public and chargeback routes.
func TestReviewerCase_IssuedTokenUnderBasicDigestOrTabRemoved(t *testing.T) {
	for _, kind := range []string{"public", "chargeback"} {
		t.Run(kind, func(t *testing.T) {
			up := newRequestRecorder(t)
			gw, _ := newTestGatewayWithUpstream(t, up.srv, kind)
			tok, _ := mintRoot(t, gw)
			front := httptest.NewServer(gw)
			defer front.Close()
			for _, prefix := range []string{"Basic ", "Bearer\t", "Digest "} {
				if got := sendAuthorization(t, front.URL, up, prefix+tok); got.Header.Get("Authorization") != "" {
					t.Errorf("issued credential remains for scheme variant %q", prefix)
				}
			}
		})
	}
}

// trailerRequest is a chunked request with the given trailers, half of them
// declared in a Trailer header and the rest not.
func trailerRequest(path, authorization string, trailers [][2]string) string {
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"probe"}}`
	var declared []string
	for i, tr := range trailers {
		if i%2 == 0 {
			declared = append(declared, tr[0])
		}
	}
	var b strings.Builder
	fmt.Fprintf(&b, "POST %s HTTP/1.1\r\nHost: gateway.test\r\nContent-Type: application/json\r\n", path)
	if authorization != "" {
		fmt.Fprintf(&b, "Authorization: %s\r\n", authorization)
	}
	fmt.Fprintf(&b, "Transfer-Encoding: chunked\r\nTrailer: %s\r\nConnection: close\r\n\r\n", strings.Join(declared, ", "))
	fmt.Fprintf(&b, "%x\r\n%s\r\n0\r\n", len(body), body)
	for _, tr := range trailers {
		fmt.Fprintf(&b, "%s: %s\r\n", tr[0], tr[1])
	}
	b.WriteString("\r\n")
	return b.String()
}

// sendRaw writes raw to addr and returns the response status.
func sendRaw(t *testing.T, srvURL, raw string) int {
	t.Helper()
	c, err := net.Dial("tcp", strings.TrimPrefix(srvURL, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	if _, err := io.WriteString(c, raw); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode
}

// Request trailers get the same removal as the header block, and also lose
// hop-by-hop names and names the upstream has configured headers for: on
// every route kind, whether the gateway read the body before forwarding it
// (an MCP route) or the body is read while it is forwarded. Trailers the
// gateway does not own still reach the upstream when they are forwarded.
func TestUpstreamTrailers_GatewayCredentialsRemovedFromTrailers(t *testing.T) {
	for _, kind := range []string{"public", "chargeback", "capability", ""} {
		for _, preRead := range []bool{true, false} {
			name := kind
			if name == "" {
				name = "default"
			}
			if preRead {
				name += "/body read by gateway"
			} else {
				name += "/body read while forwarded"
			}
			t.Run(name, func(t *testing.T) {
				up := newRequestRecorder(t)
				cfg := &config.Config{
					Server: config.ServerConfig{Listen: ":8080"},
					Admin:  config.AdminConfig{Token: "admin-secret"},
					Upstreams: map[string]config.Upstream{"backend": {URL: up.srv.URL, Timeout: 10 * time.Second,
						Headers: map[string]string{"X-Upstream-Key": "configured"}}},
					Routes: []config.Route{{Name: "test-route", Match: config.RouteMatch{PathPrefix: "/api/"},
						Upstream: "backend", Policy: config.RoutePolicy{Kind: kind}}},
				}
				if preRead {
					cfg.Routes[0].MCP = &config.MCPConfig{Enabled: true}
				}
				gw := newTestGateway(t, cfg)
				tok, _ := mintRoot(t, gw)
				front := httptest.NewServer(gw)
				defer front.Close()

				auth := ""
				if kind == "capability" || kind == "" {
					auth = "Bearer " + tok
				}
				trailers := [][2]string{
					{"X-Admin-Token", "fixture"},
					{"X-Tenant-ID", "fixture"},
					{"X-SatGate-Probe", "fixture"},
					{"Cookie", "cloud_session=fixture; theme=dark"},
					{"Authorization", "Bearer " + tok},
					{"X-Benign", "kept"},
					{"Proxy-Authorization", "Basic Zm9v"},
					{"X-Upstream-Key", "from-client"},
					{"X-Satgate-Undeclared", "fixture"},
				}
				if status := sendRaw(t, front.URL, trailerRequest("/api/data", auth, trailers)); status != http.StatusOK {
					t.Fatalf("status %d", status)
				}
				got := up.take(t)
				for k, vs := range got.Trailer {
					if config.IsGatewayOwnedHeader(k) && len(vs) > 0 {
						t.Errorf("trailer %s that SatGate owns reached the upstream", k)
					}
					for _, v := range vs {
						if strings.Contains(v, tok) {
							t.Errorf("trailer %s carried the gateway's token to the upstream", k)
						}
					}
				}
				if v := got.Trailer.Values("Proxy-Authorization"); len(v) > 0 {
					t.Error("a hop-by-hop trailer reached the upstream")
				}
				if v := got.Trailer.Values("X-Upstream-Key"); len(v) > 0 {
					t.Error("a trailer named like a configured header reached the upstream")
				}
				if c := got.Trailer.Values("Cookie"); len(c) > 0 && (len(c) != 1 || c[0] != "theme=dark") {
					t.Errorf("trailer Cookie = %q, want SatGate's cookie removed", c)
				}
				if got.Header.Get("X-Upstream-Key") != "configured" {
					t.Error("the configured header did not arrive")
				}
				// As before this change: trailers the gateway read with the
				// body are forwarded; values that arrive while the body is
				// forwarded are recorded on the inbound request only.
				if preRead && got.Trailer.Get("X-Benign") != "kept" {
					t.Error("a trailer the gateway does not own was not forwarded")
				}
				if !preRead && got.Trailer.Get("X-Benign") != "" {
					t.Error("a trailer value that arrived while the body was forwarded reached the upstream")
				}
			})
		}
	}
}

// On a public route a foreign Authorization trailer is kept, as it would be
// in the header block, while one carrying a gateway token is removed.
func TestUpstreamTrailers_PublicRouteKeepsForeignAuthorizationTrailer(t *testing.T) {
	up := newRequestRecorder(t)
	cfg := &config.Config{
		Upstreams: map[string]config.Upstream{"backend": {URL: up.srv.URL, Timeout: 10 * time.Second}},
		Routes: []config.Route{{Name: "test-route", Match: config.RouteMatch{PathPrefix: "/api/"},
			Upstream: "backend", Policy: config.RoutePolicy{Kind: "public"}, MCP: &config.MCPConfig{Enabled: true}}},
	}
	gw := newTestGateway(t, cfg)
	front := httptest.NewServer(gw)
	defer front.Close()
	raw := trailerRequest("/api/data", "", [][2]string{{"Authorization", "Basic dXNlcjpwYXNz"}, {"X-Other", "x"}})
	if status := sendRaw(t, front.URL, raw); status != http.StatusOK {
		t.Fatalf("status %d", status)
	}
	if got := up.take(t); got.Trailer.Get("Authorization") != "Basic dXNlcjpwYXNz" {
		t.Error("a foreign Authorization trailer was not forwarded unchanged")
	}
}

// Reviewer case: owned trailers, a SatGate session cookie and a gateway
// Authorization sent as trailers, with the body read by the gateway first,
// do not reach the upstream on public, chargeback, capability and default
// routes; nor do owned trailers a client attaches with net/http.
func TestReviewerCase_TrailersCarryNoGatewayCredential(t *testing.T) {
	for _, kind := range []string{"public", "chargeback", "capability", ""} {
		t.Run("raw/"+kind, func(t *testing.T) {
			up := newRequestRecorder(t)
			gw, cfg := newTestGatewayWithUpstream(t, up.srv, kind)
			cfg.Routes[0].MCP = &config.MCPConfig{Enabled: true}
			tok, _ := mintRoot(t, gw)
			front := httptest.NewServer(gw)
			defer front.Close()
			raw := trailerRequest("/api/data", "Bearer "+tok, [][2]string{
				{"X-Admin-Token", "fixture"}, {"X-Tenant-ID", "fixture"}, {"X-SatGate-Probe", "fixture"},
				{"Cookie", "cloud_session=fixture"}, {"Authorization", "Bearer " + tok},
			})
			if status := sendRaw(t, front.URL, raw); status != http.StatusOK {
				t.Fatalf("request did not reach upstream, status %d", status)
			}
			got := up.take(t)
			for _, k := range []string{"X-Admin-Token", "X-Tenant-ID", "X-SatGate-Probe", "Cookie", "Authorization"} {
				if got.Trailer.Get(k) != "" {
					t.Errorf("trailer %s reached the upstream", k)
				}
			}
		})
	}
	t.Run("net/http client", func(t *testing.T) {
		up := newRequestRecorder(t)
		gw, _ := newTestGatewayWithUpstream(t, up.srv, "public")
		front := httptest.NewServer(gw)
		defer front.Close()
		req, _ := http.NewRequest(http.MethodPost, front.URL+"/api/data", io.NopCloser(strings.NewReader("body")))
		req.Trailer = http.Header{"X-Admin-Token": []string{"fixture"}, "X-Satgate-Test": []string{"fixture"}}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		_, _ = io.Copy(io.Discard, resp.Body)
		resp.Body.Close()
		if resp.StatusCode != http.StatusOK {
			t.Fatalf("status %d", resp.StatusCode)
		}
		for name, values := range up.take(t).Trailer {
			if len(values) > 0 && values[0] != "" {
				t.Errorf("gateway-owned trailer reached upstream: %s", name)
			}
		}
	})
}

// configuredGateway returns a gateway with one public route to up whose
// upstream has headers configured.
func configuredGateway(t *testing.T, up *requestRecorder, headers map[string]string) *Gateway {
	t.Helper()
	return newTestGateway(t, &config.Config{
		Upstreams: map[string]config.Upstream{"backend": {URL: up.srv.URL, Timeout: 10 * time.Second, Headers: headers}},
		Routes: []config.Route{{Name: "r", Match: config.RouteMatch{PathPrefix: "/"}, Upstream: "backend",
			Policy: config.RoutePolicy{Kind: "public"}}},
	})
}

// Every accepted configured header, Host included, arrives with exactly its
// configured value and as its only value, whatever the client sends under
// the same name, and even when the client's Connection header names it.
func TestUpstreamHeaders_DeliveredExactlyAsConfigured(t *testing.T) {
	configured := map[string]string{
		"Authorization":   "Bearer upstream-configured",
		"User-Agent":      "configured-agent/1.0",
		"Accept-Encoding": "identity",
		"Cookie":          "upstream=configured",
		"x-custom":        "configured",
		"X-Inner-Tab":     "a\tb",
		"X-Empty":         "",
		"X-Utf8":          "caf\u00e9",
		"Via":             "1.1 configured",
		"Expect":          "100-continue",
		"Host":            "configured.example:8443",
	}
	up := newRequestRecorder(t)
	gw := configuredGateway(t, up, configured)
	front := httptest.NewServer(gw)
	defer front.Close()
	req, _ := http.NewRequest(http.MethodGet, front.URL+"/data", nil)
	for k := range configured {
		// The gateway's own server answers an Expect it does not know with
		// 417, so the client sends none.
		if k != "Host" && k != "Expect" {
			req.Header.Set(k, "from-client")
		}
	}
	req.Header.Set("Connection", "x-custom, User-Agent")
	req.Header.Set("X-Forwarded-For", "192.0.2.10")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("status %d", resp.StatusCode)
	}
	got := up.take(t)
	for k, want := range configured {
		if k == "Host" {
			if got.Host != want {
				t.Errorf("Host = %q, want the configured host", got.Host)
			}
			continue
		}
		if v := got.Header.Values(k); len(v) != 1 || v[0] != want {
			t.Errorf("header %s = %q, want exactly the configured value", k, v)
		}
	}
}

// Reviewer case: transport-managed names in an upstream's headers. Host is
// set on the outgoing request as configured; every other name is refused
// when the configuration is loaded and when the gateway is built, with an
// error naming the upstream and the entry.
func TestReviewerCase_ConfiguredTransportManagedHeadersDeliveredOrRefused(t *testing.T) {
	t.Run("Host", func(t *testing.T) {
		up := newRequestRecorder(t)
		gw := configuredGateway(t, up, map[string]string{"Host": "review-fixture"})
		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "http://gateway.invalid/test", nil))
		if rec.Code != http.StatusOK {
			t.Fatalf("status %d", rec.Code)
		}
		if got := up.take(t); got.Host != "review-fixture" {
			t.Error("configured Host not set on the outgoing request")
		}
	})
	for _, name := range []string{"X-Forwarded-For", "Proxy-Authorization", "Te", "Connection", "Keep-Alive"} {
		t.Run(name, func(t *testing.T) {
			yamlCfg := "upstreams:\n  backend:\n    url: https://example.com\n    headers:\n      " + name +
				": review-fixture\nroutes:\n  - name: r\n    match: {pathPrefix: /}\n    upstream: backend\n    policy: {kind: public}\n"
			_, err := config.ParseYAML([]byte(yamlCfg))
			if err == nil {
				t.Fatalf("configuration with %s accepted, but it would not arrive as configured", name)
			}
			if !strings.Contains(err.Error(), "upstream backend") || !strings.Contains(err.Error(), name) {
				t.Errorf("load error does not name the upstream and the entry: %v", err)
			}
			if strings.Contains(err.Error(), "review-fixture") {
				t.Error("load error carries the configured value")
			}
			_, err = New(Options{Config: &config.Config{
				Upstreams: map[string]config.Upstream{"backend": {URL: "http://127.0.0.1:1", Headers: map[string]string{name: "review-fixture"}}},
				Routes: []config.Route{{Name: "r", Match: config.RouteMatch{PathPrefix: "/"}, Upstream: "backend",
					Policy: config.RoutePolicy{Kind: "public"}}},
			}, Macaroon: newTestMacaroonService(t)})
			if err == nil {
				t.Fatalf("gateway built with %s configured", name)
			}
		})
	}
}

// Reviewer case: compatibility of upstream headers at load. A name SatGate
// owns and two spellings of one name are refused, with an error naming the
// upstream and the entries; a value with a tab between other characters and
// an ordinary header load.
func TestReviewerCase_CompatRefusalsNameUpstreamAndEntry(t *testing.T) {
	cases := []struct {
		entry    string
		accepted bool
		names    []string
	}{
		{"X-Admin-Token: fixture", false, []string{"X-Admin-Token"}},
		{"X-Dup: a\n      x-dup: b", false, []string{"X-Dup", "x-dup"}},
		{`X-Tab: "a\tb"`, true, nil},
		{"X-Ok: fixture", true, nil},
	}
	for _, c := range cases {
		raw := "upstreams:\n  u:\n    url: https://example.com\n    headers:\n      " + c.entry +
			"\nroutes:\n  - name: r\n    match: {pathPrefix: /}\n    upstream: u\n    policy: {kind: public}\n"
		_, err := config.ParseYAML([]byte(raw))
		if (err == nil) != c.accepted {
			t.Errorf("%s: accepted=%v, want %v (%v)", strings.Split(c.entry, ":")[0], err == nil, c.accepted, err)
			continue
		}
		if err != nil {
			msg := err.Error()
			if !strings.Contains(msg, "upstream u:") {
				t.Errorf("error does not name the upstream: %v", err)
			}
			for _, n := range c.names {
				if !strings.Contains(msg, `"`+n+`"`) {
					t.Errorf("error does not name the entry %s: %v", n, err)
				}
			}
			if strings.Contains(msg, "fixture") {
				t.Error("error carries a configured value")
			}
		}
	}
}

// Reviewer case: for each route kind and each kind of upstream response
// (normal, redirect, error, upgrade), the forwarding routes send no gateway
// credential and keep the foreign one where the route does not consume
// Authorization, the refusing routes forward nothing, and a redirect is
// not followed.
func TestReviewerCase_RouteAndResponseMatrixKeepsCredentialBoundary(t *testing.T) {
	for _, kind := range []string{"public", "chargeback", "capability", "", "deny", "fiat402", "unknown"} {
		for _, status := range []int{200, 302, 500, 101} {
			t.Run(kind+http.StatusText(status), func(t *testing.T) {
				got := make(chan http.Header, 2)
				up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					got <- r.Header.Clone()
					switch status {
					case 302:
						w.Header().Set("Location", "/other")
					case 101:
						w.Header().Set("Connection", "Upgrade")
						w.Header().Set("Upgrade", "websocket")
					case 200:
						w.Header().Set("Content-Type", "text/event-stream")
					}
					w.WriteHeader(status)
					if status == 200 {
						_, _ = io.WriteString(w, "data: ok\n\n")
					}
				}))
				defer up.Close()
				gw, _ := newTestGatewayWithUpstream(t, up, kind)
				token, _ := mintRoot(t, gw)
				front := httptest.NewServer(gw)
				defer front.Close()
				req, _ := http.NewRequest(http.MethodGet, front.URL+"/api/data", nil)
				req.Header.Add("aUtHoRiZaTiOn", "bEaReR "+token)
				if kind == "capability" || kind == "" {
					req.Header.Set("Authorization", "Bearer "+token)
				}
				req.Header.Add("Authorization", "Basic foreign-fixture")
				addOwnedAndForeignHeaders(req)
				if status == 101 {
					req.Header.Set("Connection", "Upgrade")
					req.Header.Set("Upgrade", "websocket")
				}
				client := &http.Client{CheckRedirect: func(*http.Request, []*http.Request) error { return http.ErrUseLastResponse }}
				resp, err := client.Do(req)
				if err != nil {
					t.Fatal(err)
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if kind == "deny" || kind == "fiat402" || kind == "unknown" {
					select {
					case <-got:
						t.Error("refused route forwarded")
					default:
					}
					return
				}
				var h http.Header
				select {
				case h = <-got:
				default:
					t.Fatal("no upstream request")
				}
				wantOwnedRemovedForeignKept(t, "wire matrix", h)
				expected := ""
				if kind == "public" || kind == "chargeback" {
					expected = "Basic foreign-fixture"
				}
				if h.Get("Authorization") != expected {
					t.Error("credential boundary mismatch")
				}
				select {
				case <-got:
					t.Error("proxy followed redirect or repeated request")
				default:
				}
			})
		}
	}
}

// Reviewer case: when the transport retries a request on a new connection
// after the upstream closed a reused one, every attempt is stripped.
func TestReviewerCase_TransportRetryStripsEveryAttempt(t *testing.T) {
	for _, kind := range []string{"public", "capability"} {
		t.Run(kind, func(t *testing.T) {
			var count int32
			var mu sync.Mutex
			got := make(chan http.Header, 4)
			up := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				got <- r.Header.Clone()
				mu.Lock()
				count++
				n := count
				mu.Unlock()
				if n == 2 {
					c, _, err := w.(http.Hijacker).Hijack()
					if err != nil {
						t.Error("hijack failed")
						return
					}
					c.Close()
					return
				}
				w.WriteHeader(http.StatusOK)
				_, _ = io.WriteString(w, "ok")
			}))
			defer up.Close()
			gw, _ := newTestGatewayWithUpstream(t, up, kind)
			token, _ := mintRoot(t, gw)
			front := httptest.NewServer(gw)
			defer front.Close()
			for i := 0; i < 2; i++ {
				req, _ := http.NewRequest(http.MethodGet, front.URL+"/api/data", nil)
				req.Header.Set("Authorization", "Bearer "+token)
				addOwnedAndForeignHeaders(req)
				resp, err := http.DefaultClient.Do(req)
				if err != nil {
					t.Fatal("request failed")
				}
				_, _ = io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
				if resp.StatusCode != http.StatusOK {
					t.Fatalf("status %d", resp.StatusCode)
				}
			}
			mu.Lock()
			n := count
			mu.Unlock()
			if n != 3 {
				t.Fatalf("retry not exercised: upstream calls %d", n)
			}
			for i := 0; i < 3; i++ {
				h := <-got
				wantOwnedRemovedForeignKept(t, "retry", h)
				if h.Get("Authorization") != "" {
					t.Error("retry forwarded token")
				}
			}
		})
	}
}
