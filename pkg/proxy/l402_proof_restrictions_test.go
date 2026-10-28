package proxy

import (
	"encoding/base64"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	macaroonv2 "gopkg.in/macaroon.v2"

	"github.com/satgate-io/satgate/pkg/config"
)

// Tests that every restriction on an L402 proof (scope, routes, methods,
// expiry) is checked before the proof is consumed. A proof narrowed so that
// it does not cover the request gets a fresh 402 and stays unconsumed.

func useL402Method(t *testing.T, gw *Gateway, method, path, authorization string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("Authorization", authorization)
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	return rec
}

func wantFreshChallenge(t *testing.T, rec *httptest.ResponseRecorder, what string) {
	t.Helper()
	if rec.Code != http.StatusPaymentRequired {
		t.Fatalf("%s: want 402, got %d body=%s", what, rec.Code, rec.Body.String())
	}
	if !strings.HasPrefix(rec.Header().Get("WWW-Authenticate"), `L402 macaroon="`) {
		t.Fatalf("%s: refusal carries no fresh L402 challenge", what)
	}
}

// Reviewer cases (both reviews): a holder narrows the proof's scope or
// expiry; plus routes, methods, spacing variants and caveats the gateway
// does not enforce.
func TestL402_NarrowedProofNotCoveringRequestRefusedAndNotConsumed(t *testing.T) {
	future := time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)
	for _, tc := range []struct {
		name    string
		caveats []string
		method  string
		path    string
	}{
		{"scope unrelated", []string{"scope = unrelated"}, "GET", "/cheap/x"},
		{"scope forbidden", []string{"scope = forbidden"}, "GET", "/cheap/x"},
		{"scope narrower than l402", []string{"scope = l402:read"}, "GET", "/cheap/x"},
		{"scope after a covering scope", []string{"scope = l402", "scope = api:read"}, "GET", "/cheap/x"},
		{"scope key with extra spaces", []string{"scope  = forbidden"}, "GET", "/cheap/x"},
		{"scope key with leading space", []string{" scope = forbidden"}, "GET", "/cheap/x"},
		{"expired", []string{"expires = 1"}, "GET", "/cheap/x"},
		{"expired after a later expiry", []string{"expires = " + future, "expires = 1"}, "GET", "/cheap/x"},
		{"relative expiry", []string{"expires = 5m"}, "GET", "/cheap/x"},
		{"routes other path", []string{"routes = /dear/*"}, "GET", "/cheap/x"},
		{"routes other route name", []string{"routes = dear"}, "GET", "/cheap/x"},
		{"routes exact other path", []string{"routes = /cheap/y"}, "GET", "/cheap/x"},
		{"routes empty", []string{"routes = "}, "GET", "/cheap/x"},
		{"routes second caveat narrower", []string{"routes = /cheap/*", "routes = /cheap/y"}, "GET", "/cheap/x"},
		{"methods other", []string{"methods = POST"}, "GET", "/cheap/x"},
		{"unenforced caveat", []string{"source_ip = 10.0.0.0/8"}, "GET", "/cheap/x"},
		{"case variant of scope", []string{"Scope = forbidden"}, "GET", "/cheap/x"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gw := newTestGateway(t, newTwoTierL402Config(okUpstream(t).URL))
			store := newCountingReplayStore()
			gw.l402Seen = store
			paid := payL402At(t, gw, "/cheap/x")

			narrowed := appendBinaryCaveats(t, paid.macaroon, tc.caveats...)
			rec := useL402Method(t, gw, tc.method, tc.path, "L402 "+narrowed+":"+paid.preimage)
			wantFreshChallenge(t, rec, "narrowed proof")
			if store.Calls() != 0 {
				t.Fatalf("narrowed proof reached the consumed-proof store before it was refused")
			}
			// The original, unrestricted proof was not used up.
			if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusOK {
				t.Fatalf("unrestricted proof after the refusal: want 200, got %d", code)
			}
		})
	}
}

// A holder can narrow a proof in a way that still covers the request.
func TestL402_NarrowedProofCoveringRequestAccepted(t *testing.T) {
	future := time.Now().Add(30 * time.Minute).UTC().Format(time.RFC3339)
	for _, tc := range []struct {
		name    string
		caveats []string
		method  string
	}{
		{"scope l402 again", []string{"scope = l402"}, "GET"},
		{"scope wildcard", []string{"scope = *"}, "GET"},
		{"earlier expiry", []string{"expires = " + future}, "GET"},
		{"routes path prefix", []string{"routes = /dear/*,/cheap/*"}, "GET"},
		{"routes exact path", []string{"routes = /cheap/x"}, "GET"},
		{"routes route name", []string{"routes = cheap"}, "GET"},
		{"methods", []string{"methods = get,HEAD"}, "GET"},
		{"all", []string{"scope = l402", "routes = cheap", "methods = POST, GET", "expires = " + future}, "POST"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			gw := newTestGateway(t, newTwoTierL402Config(okUpstream(t).URL))
			paid := payL402At(t, gw, "/cheap/x")
			narrowed := appendBinaryCaveats(t, paid.macaroon, tc.caveats...)
			auth := "L402 " + narrowed + ":" + paid.preimage
			if rec := useL402Method(t, gw, tc.method, "/cheap/x", auth); rec.Code != http.StatusOK {
				t.Fatalf("narrowed proof that covers the request: want 200, got %d body=%s", rec.Code, rec.Body.String())
			}
			if code := useL402(t, gw, "/cheap/x", paid.authorization()); code == http.StatusOK {
				t.Fatalf("the payment was accepted twice")
			}
		})
	}
}

// Reviewer case (security review): a standard libmacaroon client appends a
// scope caveat to the challenge of a single-route gateway.
func TestL402_ScopeAddedByStandardMacaroonClientRefused(t *testing.T) {
	up := okUpstream(t)
	cfg := &config.Config{
		Upstreams: map[string]config.Upstream{"backend": {URL: up.URL}},
		Routes: []config.Route{{
			Name: "paid", Match: config.RouteMatch{PathPrefix: "/paid/"}, Upstream: "backend",
			Policy: config.RoutePolicy{Kind: "l402", PriceSats: 10},
		}},
		Admin: config.AdminConfig{Token: "admin-secret"},
	}
	gw := newTestGateway(t, cfg)
	store := newCountingReplayStore()
	gw.l402Seen = store
	paid := payL402At(t, gw, "/paid/x")

	raw, err := base64.StdEncoding.DecodeString(paid.macaroon)
	if err != nil {
		t.Fatal(err)
	}
	var m macaroonv2.Macaroon
	if err := m.UnmarshalBinary(raw); err != nil {
		t.Fatal(err)
	}
	if err := m.AddFirstPartyCaveat([]byte("scope = forbidden")); err != nil {
		t.Fatal(err)
	}
	out, err := m.MarshalBinary()
	if err != nil {
		t.Fatal(err)
	}
	rec := useL402Method(t, gw, "GET", "/paid/x", "L402 "+base64.StdEncoding.EncodeToString(out)+":"+paid.preimage)
	wantFreshChallenge(t, rec, "scope-restricted proof")
	if store.Calls() != 0 {
		t.Fatal("scope-restricted proof reached the consumed-proof store")
	}
	if code := useL402(t, gw, "/paid/x", paid.authorization()); code != http.StatusOK {
		t.Fatalf("unrestricted proof: want 200, got %d", code)
	}
}
