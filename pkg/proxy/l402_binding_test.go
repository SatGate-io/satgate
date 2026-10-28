package proxy

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	macaroonv2 "gopkg.in/macaroon.v2"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
)

// Tests for L402 proofs bound to the route and price they were issued for,
// single-valued server caveats, and the shared consumed-proof store.

func newTwoTierL402Config(upstreamURL string) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Admin:  config.AdminConfig{Token: "admin-secret"},
		Cloud:  &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstreamURL},
		},
		Routes: []config.Route{
			{
				Name:     "cheap",
				Match:    config.RouteMatch{PathPrefix: "/cheap/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "l402", PriceSats: 10},
			},
			{
				Name:     "dear",
				Match:    config.RouteMatch{PathPrefix: "/dear/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "l402", PriceSats: 1000},
			},
		},
	}
}

func okUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	t.Cleanup(srv.Close)
	return srv
}

// l402Paid is a challenge macaroon and the preimage that pays its invoice.
type l402Paid struct {
	macaroon string
	preimage string
}

func (p l402Paid) authorization() string {
	return "L402 " + p.macaroon + ":" + p.preimage
}

// payL402At asks gw for a challenge at path and pays it through the mock
// provider's admin endpoint.
func payL402At(t *testing.T, gw *Gateway, path string) l402Paid {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("challenge at %s: want 402, got %d body=%s", path, rec.Code, rec.Body.String())
	}
	mac := extractL402MacaroonFromChallenge(t, rec.Header().Get("WWW-Authenticate"))
	var body map[string]any
	if err := json.NewDecoder(rec.Body).Decode(&body); err != nil {
		t.Fatalf("decode challenge body: %v", err)
	}
	hash, _ := body["payment_hash"].(string)

	payBody, _ := json.Marshal(map[string]string{"payment_hash": hash, "macaroon": mac})
	payReq := httptest.NewRequest("POST", "/api/l402/mock-pay", strings.NewReader(string(payBody)))
	payReq.Header.Set("Content-Type", "application/json")
	payReq.Header.Set("X-Admin-Token", "admin-secret")
	payRec := httptest.NewRecorder()
	gw.ServeHTTP(payRec, payReq)
	if payRec.Code != http.StatusOK {
		t.Fatalf("mock pay: want 200, got %d body=%s", payRec.Code, payRec.Body.String())
	}
	var paid map[string]string
	if err := json.NewDecoder(payRec.Body).Decode(&paid); err != nil {
		t.Fatalf("decode pay body: %v", err)
	}
	auth := paid["authorization"]
	prefix := "L402 " + mac + ":"
	if !strings.HasPrefix(auth, prefix) {
		t.Fatalf("unexpected authorization shape")
	}
	return l402Paid{macaroon: mac, preimage: strings.TrimPrefix(auth, prefix)}
}

func useL402(t *testing.T, gw *Gateway, path, authorization string) int {
	t.Helper()
	req := httptest.NewRequest("GET", path, nil)
	req.Header.Set("Authorization", authorization)
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	return rec.Code
}

// appendBinaryCaveats appends first-party caveats to a binary challenge
// macaroon the way any holder can, by chaining from its signature.
func appendBinaryCaveats(t *testing.T, mac string, caveats ...string) string {
	t.Helper()
	raw, err := base64.StdEncoding.DecodeString(mac)
	if err != nil {
		t.Fatalf("decode macaroon: %v", err)
	}
	var m macaroonv2.Macaroon
	if err := m.UnmarshalBinary(raw); err != nil {
		t.Fatalf("unmarshal macaroon: %v", err)
	}
	for _, c := range caveats {
		if err := m.AddFirstPartyCaveat([]byte(c)); err != nil {
			t.Fatalf("add caveat: %v", err)
		}
	}
	out, err := m.MarshalBinary()
	if err != nil {
		t.Fatalf("marshal macaroon: %v", err)
	}
	return base64.StdEncoding.EncodeToString(out)
}

func challengeCaveats(t *testing.T, gw *Gateway, mac string) []string {
	t.Helper()
	decoded, err := gw.macaroonSvc.DecodeBinary(mac)
	if err != nil {
		t.Fatalf("decode challenge macaroon: %v", err)
	}
	return decoded.Caveats
}

func TestL402_ChallengeCarriesRouteAndPriceCaveats(t *testing.T) {
	gw := newTestGateway(t, newTwoTierL402Config(okUpstream(t).URL))
	paid := payL402At(t, gw, "/cheap/x")
	caveats := strings.Join(challengeCaveats(t, gw, paid.macaroon), "\n")
	for _, want := range []string{"route = cheap", "price_sats = 10", "invoice_amount_sats = 10"} {
		if !strings.Contains(caveats, want) {
			t.Fatalf("challenge macaroon lacks %q; caveats:\n%s", want, caveats)
		}
	}
}

func TestL402_PayThenUseStillWorks(t *testing.T) {
	gw := newTestGateway(t, newTwoTierL402Config(okUpstream(t).URL))
	paid := payL402At(t, gw, "/cheap/x")
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusOK {
		t.Fatalf("paid proof on its own route: want 200, got %d", code)
	}
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code == http.StatusOK {
		t.Fatalf("a used proof was accepted a second time")
	}
}

func TestL402_ProofFromCheaperRouteRefusedOnDearerRoute(t *testing.T) {
	gw := newTestGateway(t, newTwoTierL402Config(okUpstream(t).URL))
	paid := payL402At(t, gw, "/cheap/x")

	if code := useL402(t, gw, "/dear/x", paid.authorization()); code != http.StatusPaymentRequired {
		t.Fatalf("cheap-route proof on dear route: want 402, got %d", code)
	}
	// The refusal on the other route does not consume the proof.
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusOK {
		t.Fatalf("proof on its own route after a refusal elsewhere: want 200, got %d", code)
	}
}

func TestL402_ProofRefusedAfterPriceChange(t *testing.T) {
	cfg := newTwoTierL402Config(okUpstream(t).URL)
	gw := newTestGateway(t, cfg)
	paid := payL402At(t, gw, "/cheap/x")

	cfg.Routes[0].Policy.PriceSats = 20
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusPaymentRequired {
		t.Fatalf("proof issued at the old price: want 402, got %d", code)
	}
}

func TestL402_RepeatedPaymentHashCaveatRefused(t *testing.T) {
	gw := newTestGateway(t, newTwoTierL402Config(okUpstream(t).URL))
	paid := payL402At(t, gw, "/cheap/x")

	preimage := strings.Repeat("c", 64)
	raw, _ := hex.DecodeString(preimage)
	sum := sha256.Sum256(raw)
	mac := appendBinaryCaveats(t, paid.macaroon, "payment_hash = "+hex.EncodeToString(sum[:]))

	if code := useL402(t, gw, "/cheap/x", "L402 "+mac+":"+preimage); code == http.StatusOK {
		t.Fatalf("macaroon with a second payment_hash caveat was accepted")
	}
	if code := useL402(t, gw, "/cheap/x", "L402 "+mac+":"+paid.preimage); code == http.StatusOK {
		t.Fatalf("macaroon with a second payment_hash caveat was accepted with the paid preimage")
	}
}

func TestL402_RepeatedRouteCaveatRefused(t *testing.T) {
	gw := newTestGateway(t, newTwoTierL402Config(okUpstream(t).URL))
	paid := payL402At(t, gw, "/cheap/x")

	for _, tc := range []struct {
		name, caveat, path string
	}{
		{"other route", "route = dear", "/dear/x"},
		{"same route repeated", "route = cheap", "/cheap/x"},
		{"price repeated", "price_sats = 1000", "/dear/x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mac := appendBinaryCaveats(t, paid.macaroon, tc.caveat)
			if code := useL402(t, gw, tc.path, "L402 "+mac+":"+paid.preimage); code == http.StatusOK {
				t.Fatalf("macaroon with appended %q was accepted on %s", tc.caveat, tc.path)
			}
		})
	}
}

func TestL402_AppendedDuplicateAfterAttenuationRefused(t *testing.T) {
	gw := newTestGateway(t, newTwoTierL402Config(okUpstream(t).URL))
	paid := payL402At(t, gw, "/cheap/x")

	// A narrower expiry is a legitimate restriction and keeps the proof usable.
	narrowed := appendBinaryCaveats(t, paid.macaroon,
		"expires = "+time.Now().Add(30*time.Minute).UTC().Format(time.RFC3339))
	// A server-written key appended after that attenuation is refused.
	dup := appendBinaryCaveats(t, narrowed, "route = dear")
	if code := useL402(t, gw, "/dear/x", "L402 "+dup+":"+paid.preimage); code == http.StatusOK {
		t.Fatalf("route caveat appended after attenuation was accepted")
	}
	if code := useL402(t, gw, "/cheap/x", "L402 "+dup+":"+paid.preimage); code == http.StatusOK {
		t.Fatalf("route caveat appended after attenuation was accepted on the issuing route")
	}
	if code := useL402(t, gw, "/cheap/x", "L402 "+narrowed+":"+paid.preimage); code != http.StatusOK {
		t.Fatalf("attenuated proof without server keys appended: want 200, got %d", code)
	}

	// Every expires caveat is still checked.
	expired := appendBinaryCaveats(t, paid.macaroon,
		"expires = "+time.Now().Add(-time.Minute).UTC().Format(time.RFC3339))
	if code := useL402(t, gw, "/cheap/x", "L402 "+expired+":"+paid.preimage); code == http.StatusOK {
		t.Fatalf("proof with an expired attenuation caveat was accepted")
	}
}

func TestL402_NonChallengeMacaroonRefused(t *testing.T) {
	gw := newTestGateway(t, newTwoTierL402Config(okUpstream(t).URL))

	// A macaroon signed by this gateway that was not issued as an L402
	// challenge (here a capability token) is never an L402 proof, whatever
	// caveats are appended to it.
	base, err := gw.macaroonSvc.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	preimage := strings.Repeat("d", 64)
	raw, _ := hex.DecodeString(preimage)
	sum := sha256.Sum256(raw)
	child, err := gw.macaroonSvc.DelegateWithoutVerify(gw.macaroonSvc.Encode(base), []string{
		"payment_hash = " + hex.EncodeToString(sum[:]), "route = cheap", "price_sats = 10",
	})
	if err != nil {
		t.Fatal(err)
	}
	if code := useL402(t, gw, "/cheap/x", "L402 "+gw.macaroonSvc.Encode(child)+":"+preimage); code == http.StatusOK {
		t.Fatalf("capability token with appended L402 caveats was accepted as a payment proof")
	}
}

// newRedisL402Gateway builds a gateway whose consumed-proof store is the
// Redis at addr. Both gateways in a test share the root key.
func newRedisL402Gateway(t *testing.T, upstreamURL, addr string) (*Gateway, *config.Config) {
	t.Helper()
	cfg := newTwoTierL402Config(upstreamURL)
	cfg.Redis = &config.RedisConfig{Enabled: true, Addr: addr}
	gw, err := New(Options{
		Config:     cfg,
		Macaroon:   newTestMacaroonService(t),
		Governance: governance.NewService(nil),
		Lightning:  lightning.NewMockProvider(),
	})
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}
	return gw, cfg
}

func TestL402_ReplayAcrossInstancesSharingRedisRefused(t *testing.T) {
	mr := miniredis.RunT(t)
	up := okUpstream(t)
	gwA, _ := newRedisL402Gateway(t, up.URL, mr.Addr())
	gwB, _ := newRedisL402Gateway(t, up.URL, mr.Addr())

	paid := payL402At(t, gwA, "/cheap/x")
	if code := useL402(t, gwA, "/cheap/x", paid.authorization()); code != http.StatusOK {
		t.Fatalf("first use on instance A: want 200, got %d", code)
	}
	if code := useL402(t, gwB, "/cheap/x", paid.authorization()); code == http.StatusOK {
		t.Fatalf("proof used on instance A was accepted again on instance B")
	}

	// A restarted instance (fresh process state, same Redis) also refuses it.
	gwC, _ := newRedisL402Gateway(t, up.URL, mr.Addr())
	if code := useL402(t, gwC, "/cheap/x", paid.authorization()); code == http.StatusOK {
		t.Fatalf("proof was accepted again after a restart")
	}

	// The record outlives the macaroon.
	keys := mr.Keys()
	if len(keys) != 1 {
		t.Fatalf("want one consumed-proof key in Redis, got %d", len(keys))
	}
	if ttl := mr.TTL(keys[0]); ttl < l402MacaroonLifetime {
		t.Fatalf("consumed-proof TTL %s is shorter than the macaroon lifetime %s", ttl, l402MacaroonLifetime)
	}
}

func TestL402_RedisUnavailableFailsClosed(t *testing.T) {
	mr := miniredis.RunT(t)
	gw, _ := newRedisL402Gateway(t, okUpstream(t).URL, mr.Addr())
	paid := payL402At(t, gw, "/cheap/x")
	mr.Close()
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusServiceUnavailable {
		t.Fatalf("store unavailable: want 503, got %d", code)
	}
}

func TestL402_ReplayTTLCoversMacaroonLifetime(t *testing.T) {
	if l402ReplayTTL < l402MacaroonLifetime {
		t.Fatalf("replay TTL %s is shorter than the macaroon lifetime %s", l402ReplayTTL, l402MacaroonLifetime)
	}
}
