package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// --- helpers ---

func newTestMacaroonService(t *testing.T) *macaroon.Service {
	t.Helper()
	svc, err := macaroon.NewService("test-secret-key-for-tests")
	if err != nil {
		t.Fatalf("failed to create macaroon service: %v", err)
	}
	return svc
}

func newTestGateway(t *testing.T, cfg *config.Config) *Gateway {
	t.Helper()
	macSvc := newTestMacaroonService(t)
	govSvc := governance.NewService(nil)
	mockLN := lightning.NewMockProvider()

	gw, err := New(Options{
		Config:     cfg,
		Macaroon:   macSvc,
		Governance: govSvc,
		Lightning:  mockLN,
	})
	if err != nil {
		t.Fatalf("failed to create gateway: %v", err)
	}
	return gw
}

func newTestGatewayWithUpstream(t *testing.T, upstream *httptest.Server, policyKind string) (*Gateway, *config.Config) {
	t.Helper()
	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Admin:  config.AdminConfig{Token: "admin-secret"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL, Timeout: 10 * time.Second},
		},
		Routes: []config.Route{
			{
				Name:     "test-route",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: policyKind},
			},
		},
	}
	gw := newTestGateway(t, cfg)
	return gw, cfg
}

// --- New() constructor tests ---

func TestNew_RequiresConfig(t *testing.T) {
	macSvc := newTestMacaroonService(t)
	_, err := New(Options{Macaroon: macSvc})
	if err == nil {
		t.Error("expected error when config is nil")
	}
}

func TestNew_RequiresMacaroon(t *testing.T) {
	cfg := &config.Config{
		Upstreams: map[string]config.Upstream{},
		Routes:    []config.Route{},
	}
	_, err := New(Options{Config: cfg})
	if err == nil {
		t.Error("expected error when macaroon service is nil")
	}
}

func TestNew_CreatesProxies(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{},
	}
	macSvc := newTestMacaroonService(t)
	gw, err := New(Options{Config: cfg, Macaroon: macSvc})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if _, ok := gw.proxies["backend"]; !ok {
		t.Error("expected proxy for 'backend' upstream")
	}
}

// --- Health endpoint tests ---

func TestHealthEndpoint(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	for _, path := range []string{"/health", "/healthz"} {
		t.Run(path, func(t *testing.T) {
			req := httptest.NewRequest("GET", path, nil)
			w := httptest.NewRecorder()
			gw.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
			var body map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode response: %v", err)
			}
			if body["status"] != "healthy" {
				t.Errorf("expected status 'healthy', got %v", body["status"])
			}
		})
	}
}

// --- CORS tests ---

func TestCORSHeaders(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Header().Get("Access-Control-Allow-Origin") != "https://example.com" {
		t.Error("expected Access-Control-Allow-Origin to match origin")
	}
}

func TestCORSPreflight(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	req := httptest.NewRequest("OPTIONS", "/api/test", nil)
	req.Header.Set("Origin", "https://example.com")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusNoContent {
		t.Errorf("expected 204 for OPTIONS, got %d", w.Code)
	}
}

// --- Route matching ---

func TestNoMatchingRoute_Returns404(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	req := httptest.NewRequest("GET", "/no-match", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("expected 404, got %d", w.Code)
	}
}

// --- Public policy proxying ---

func TestPublicPolicy_ProxiesRequest(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(map[string]string{"proxied": "true", "path": r.URL.Path})
	}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	req := httptest.NewRequest("GET", "/api/data", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	var body map[string]string
	json.NewDecoder(w.Body).Decode(&body)
	if body["proxied"] != "true" {
		t.Error("expected proxied response")
	}
}

// --- Capability policy tests ---

func TestCapabilityPolicy_NoToken_Returns401(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "capability")

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
	if w.Header().Get("WWW-Authenticate") == "" {
		t.Error("expected WWW-Authenticate header")
	}
}

func TestCapabilityPolicy_InvalidToken_Returns401(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "capability")

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer invalid-token-garbage")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCapabilityPolicy_ValidToken_Proxies(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "capability")
	macSvc := gw.macaroonSvc

	// Mint a valid token
	mac, err := macSvc.Mint("api:read", time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	token := macSvc.Encode(mac)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
}

func TestCapabilityPolicy_BannedToken_Returns401(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "capability")
	macSvc := gw.macaroonSvc

	mac, err := macSvc.Mint("api:read", time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	token := macSvc.Encode(mac)

	// Ban the token
	gw.governance.Ban(mac.Signature, "test ban", "admin")

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401 for banned token, got %d", w.Code)
	}
}

func TestCapabilityPolicy_ScopeEnforcement(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Admin:  config.AdminConfig{Token: "admin-secret"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "scoped",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy: config.RoutePolicy{
					Kind:  "capability",
					Scope: "api:admin",
				},
			},
		},
	}
	gw := newTestGateway(t, cfg)
	macSvc := gw.macaroonSvc

	// Token with wrong scope
	mac, _ := macSvc.Mint("api:read", time.Now().Add(1*time.Hour))
	token := macSvc.Encode(mac)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("expected 403 for wrong scope, got %d", w.Code)
	}
}

// --- L402 policy tests ---

func TestL402Policy_NoToken_Returns402(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Admin:  config.AdminConfig{Token: "admin-secret"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "paid",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy: config.RoutePolicy{
					Kind:      "l402",
					PriceSats: 100,
				},
			},
		},
	}
	gw := newTestGateway(t, cfg)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusPaymentRequired {
		t.Errorf("expected 402, got %d", w.Code)
	}
	authHeader := w.Header().Get("WWW-Authenticate")
	if !strings.Contains(authHeader, "L402") {
		t.Error("expected L402 challenge in WWW-Authenticate")
	}
	if !strings.Contains(authHeader, "macaroon=") {
		t.Error("expected macaroon in L402 challenge")
	}
	if !strings.Contains(authHeader, "invoice=") {
		t.Error("expected invoice in L402 challenge")
	}
}

// --- StripPrefix tests ---

func TestStripPrefix_RemovesPrefixBeforeProxying(t *testing.T) {
	var receivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:        "stripped",
				Match:       config.RouteMatch{PathPrefix: "/gateway/api/"},
				Upstream:    "backend",
				StripPrefix: true,
				Policy:      config.RoutePolicy{Kind: "public"},
			},
		},
	}
	gw := newTestGateway(t, cfg)

	req := httptest.NewRequest("GET", "/gateway/api/v1/users", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if receivedPath != "/v1/users" {
		t.Errorf("expected upstream path '/v1/users', got %q", receivedPath)
	}
}

func TestStripPrefix_WithWildcard(t *testing.T) {
	var receivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:        "wildcard-strip",
				Match:       config.RouteMatch{PathPrefix: "/gw/*"},
				Upstream:    "backend",
				StripPrefix: true,
				Policy:      config.RoutePolicy{Kind: "public"},
			},
		},
	}
	gw := newTestGateway(t, cfg)

	req := httptest.NewRequest("GET", "/gw/hello/world", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if receivedPath != "/hello/world" {
		t.Errorf("expected upstream path '/hello/world', got %q", receivedPath)
	}
}

// --- Rewrite tests ---

func TestRewrite_OverridesPath(t *testing.T) {
	var receivedPath string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedPath = r.URL.Path
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "rewrite",
				Match:    config.RouteMatch{PathPrefix: "/old-api/"},
				Upstream: "backend",
				Rewrite:  "/v2/new-endpoint",
				Policy:   config.RoutePolicy{Kind: "public"},
			},
		},
	}
	gw := newTestGateway(t, cfg)

	req := httptest.NewRequest("GET", "/old-api/anything", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if receivedPath != "/v2/new-endpoint" {
		t.Errorf("expected rewrite path '/v2/new-endpoint', got %q", receivedPath)
	}
}

// --- Upstream error handling ---

func TestUpstreamDown_Returns502(t *testing.T) {
	// Create a server and immediately close it so the upstream is unreachable
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	upstreamURL := upstream.URL
	upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstreamURL},
		},
		Routes: []config.Route{
			{
				Name:     "dead-upstream",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "public"},
			},
		},
	}
	gw := newTestGateway(t, cfg)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

func TestNoUpstreamConfigured_Returns502(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "no-upstream",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "", // no upstream set
				Policy:   config.RoutePolicy{Kind: "public"},
			},
		},
	}
	gw := newTestGateway(t, cfg)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusBadGateway {
		t.Errorf("expected 502, got %d", w.Code)
	}
}

// --- Unknown policy in OSS ---

func TestUnknownPolicy_Returns501(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "enterprise",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "some-enterprise-feature"},
			},
		},
	}
	gw := newTestGateway(t, cfg)

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusNotImplemented {
		t.Errorf("expected 501, got %d", w.Code)
	}
}

// --- Check payment endpoint ---

func TestCheckPayment_InvalidHash(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	req := httptest.NewRequest("GET", "/check-payment/tooshort", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("expected 400, got %d", w.Code)
	}
}

func TestCheckPayment_ValidHash_NotPaid(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	hash := "a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2c3d4e5f6a1b2"
	req := httptest.NewRequest("GET", "/check-payment/"+hash, nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var body map[string]interface{}
	json.NewDecoder(w.Body).Decode(&body)
	if body["paid"] != false {
		t.Errorf("expected paid=false, got %v", body["paid"])
	}
}

// --- Capability mint endpoint ---

func TestCapabilityMint_NoAdminToken_Returns401(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	body := strings.NewReader(`{"scope":"api:read","duration":"1h"}`)
	req := httptest.NewRequest("POST", "/api/capability/mint", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCapabilityMint_ValidAdmin_MintsToken(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	body := strings.NewReader(`{"scope":"api:read","duration":"1h"}`)
	req := httptest.NewRequest("POST", "/api/capability/mint", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "admin-secret")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if _, ok := resp["token"]; !ok {
		t.Error("expected 'token' in response")
	}
	if resp["scope"] != "api:read" {
		t.Errorf("expected scope 'api:read', got %v", resp["scope"])
	}
}

// --- Capability validate endpoint ---

func TestCapabilityValidate_ValidToken(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")
	macSvc := gw.macaroonSvc

	mac, _ := macSvc.Mint("api:read", time.Now().Add(1*time.Hour))
	token := macSvc.Encode(mac)

	body := strings.NewReader(`{"token":"` + token + `"}`)
	req := httptest.NewRequest("POST", "/api/capability/validate", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if resp["valid"] != true {
		t.Errorf("expected valid=true, got %v", resp["valid"])
	}
}

func TestCapabilityValidate_InvalidToken(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	body := strings.NewReader(`{"token":"garbage"}`)
	req := httptest.NewRequest("POST", "/api/capability/validate", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

// --- Capability delegate endpoint ---

func TestCapabilityDelegate_Success(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")
	macSvc := gw.macaroonSvc

	parent, _ := macSvc.Mint("api:*", time.Now().Add(1*time.Hour))
	parentToken := macSvc.Encode(parent)

	body := strings.NewReader(`{"parentToken":"` + parentToken + `","caveats":["scope = api:read"]}`)
	req := httptest.NewRequest("POST", "/api/capability/delegate", body)
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if _, ok := resp["token"]; !ok {
		t.Error("expected 'token' in delegation response")
	}
}

// --- Capability ping endpoint ---

func TestCapabilityPing_NoAuth_Returns401(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	req := httptest.NewRequest("GET", "/api/capability/ping", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestCapabilityPing_ValidToken_ReturnsOK(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")
	macSvc := gw.macaroonSvc

	mac, _ := macSvc.Mint("api:read", time.Now().Add(1*time.Hour))
	token := macSvc.Encode(mac)

	req := httptest.NewRequest("GET", "/api/capability/ping", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}
}

// --- Governance ban endpoint ---

func TestGovernanceBan_NoAdmin_Returns401(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	body := strings.NewReader(`{"tokenSignature":"abc123","reason":"test"}`)
	req := httptest.NewRequest("POST", "/api/governance/ban", body)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("expected 401, got %d", w.Code)
	}
}

func TestGovernanceBan_Valid(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	body := strings.NewReader(`{"tokenSignature":"abc123def456","reason":"compromised"}`)
	req := httptest.NewRequest("POST", "/api/governance/ban", body)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Admin-Token", "admin-secret")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d; body: %s", w.Code, w.Body.String())
	}

	// Verify the token is actually banned
	if !gw.governance.IsBanned("abc123def456") {
		t.Error("expected token to be banned")
	}
}

// --- Governance graph endpoint ---

func TestGovernanceGraph_ReturnsData(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	req := httptest.NewRequest("GET", "/api/governance/graph", nil)
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("expected 200, got %d", w.Code)
	}
	var resp map[string]interface{}
	json.NewDecoder(w.Body).Decode(&resp)
	if _, ok := resp["nodes"]; !ok {
		t.Error("expected 'nodes' in graph response")
	}
	if _, ok := resp["stats"]; !ok {
		t.Error("expected 'stats' in graph response")
	}
}

// --- Metrics ---

func TestMetrics_Tracked(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	// Make a few requests
	for i := 0; i < 3; i++ {
		req := httptest.NewRequest("GET", "/api/test", nil)
		w := httptest.NewRecorder()
		gw.ServeHTTP(w, req)
	}

	metrics := gw.GetMetrics()
	if metrics.TotalRequests < 3 {
		t.Errorf("expected at least 3 total requests, got %d", metrics.TotalRequests)
	}
	if metrics.TotalPublic < 3 {
		t.Errorf("expected at least 3 public requests, got %d", metrics.TotalPublic)
	}
}

// --- Demo response tests ---

func TestDemoRoutes_ReturnMockData(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Error("should not reach upstream for demo routes")
	}))
	defer upstream.Close()

	demoNames := []string{"api-micro", "api-basic", "api-standard", "api-premium"}
	for _, name := range demoNames {
		t.Run(name, func(t *testing.T) {
			cfg := &config.Config{
				Server: config.ServerConfig{Listen: ":8080"},
				Upstreams: map[string]config.Upstream{
					"backend": {URL: upstream.URL},
				},
				Routes: []config.Route{
					{
						Name:     name,
						Match:    config.RouteMatch{PathPrefix: "/api/"},
						Upstream: "backend",
						Policy:   config.RoutePolicy{Kind: "public"},
					},
				},
			}
			gw := newTestGateway(t, cfg)

			req := httptest.NewRequest("GET", "/api/test", nil)
			w := httptest.NewRecorder()
			gw.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("expected 200, got %d", w.Code)
			}
			var body map[string]interface{}
			if err := json.NewDecoder(w.Body).Decode(&body); err != nil {
				t.Fatalf("failed to decode: %v", err)
			}
			if body["tier"] == nil {
				t.Error("expected 'tier' in demo response")
			}
		})
	}
}

// --- extractBearerToken ---

func TestExtractBearerToken(t *testing.T) {
	tests := []struct {
		name  string
		auth  string
		want  string
	}{
		{"bearer", "Bearer abc123", "abc123"},
		{"no prefix", "raw-token", "raw-token"},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			got := extractBearerToken(req)
			if got != tt.want {
				t.Errorf("extractBearerToken() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- extractL402Token ---

func TestExtractL402Token(t *testing.T) {
	tests := []struct {
		name string
		auth string
		want string
	}{
		{"L402 format", "L402 macaroon:preimage", "macaroon:preimage"},
		{"LSAT format", "LSAT macaroon:preimage", "macaroon:preimage"},
		{"bearer (not L402)", "Bearer token", ""},
		{"empty", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := httptest.NewRequest("GET", "/", nil)
			if tt.auth != "" {
				req.Header.Set("Authorization", tt.auth)
			}
			got := extractL402Token(req)
			if got != tt.want {
				t.Errorf("extractL402Token() = %q, want %q", got, tt.want)
			}
		})
	}
}

// --- matchRoute tests ---

func TestMatchRoute_ExactBeforePrefix(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{Name: "exact", Match: config.RouteMatch{PathExact: "/api/special"}, Upstream: "backend", Policy: config.RoutePolicy{Kind: "public"}},
			{Name: "prefix", Match: config.RouteMatch{PathPrefix: "/api/"}, Upstream: "backend", Policy: config.RoutePolicy{Kind: "public"}},
		},
	}
	gw := newTestGateway(t, cfg)

	req := httptest.NewRequest("GET", "/api/special", nil)
	route := gw.matchRoute(req)
	if route == nil {
		t.Fatal("expected a match")
	}
	if route.Name != "exact" {
		t.Errorf("expected 'exact' route, got %q", route.Name)
	}
}

// --- statusWriter tests ---

func TestStatusWriter_CapturesCode(t *testing.T) {
	w := httptest.NewRecorder()
	sw := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}

	sw.WriteHeader(http.StatusCreated)
	if sw.statusCode != http.StatusCreated {
		t.Errorf("expected 201, got %d", sw.statusCode)
	}
}

// --- min helper ---

func TestMin(t *testing.T) {
	if min(3, 5) != 3 {
		t.Error("min(3,5) should be 3")
	}
	if min(5, 3) != 3 {
		t.Error("min(5,3) should be 3")
	}
	if min(4, 4) != 4 {
		t.Error("min(4,4) should be 4")
	}
}

// --- Request forwarding with body ---

func TestPublicPolicy_ForwardsRequestBody(t *testing.T) {
	var receivedBody string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		bodyBytes, _ := io.ReadAll(r.Body)
		receivedBody = string(bodyBytes)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()

	gw, _ := newTestGatewayWithUpstream(t, upstream, "public")

	reqBody := `{"key":"value"}`
	req := httptest.NewRequest("POST", "/api/data", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("expected 200, got %d", w.Code)
	}
	if receivedBody != reqBody {
		t.Errorf("expected body %q, got %q", reqBody, receivedBody)
	}
}
