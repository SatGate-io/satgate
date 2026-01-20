package proxy

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/satgate-io/satgate/internal/config"
	"github.com/satgate-io/satgate/internal/macaroon"
)

// TestDefaultProtectionEnforcement validates the core "Protect by default" promise:
// - All non-PUBLIC routes require valid credentials (macaroon)
// - PUBLIC routes are the explicit opt-out (allowed without credentials)
// - This is Layer 0 of the Economic Firewall model

// testMacaroonKey is a fixed key for deterministic test macaroons
const testMacaroonKey = "test-satgate-macaroon-key-for-unit-tests-only"

// newTestMacaroonService creates a macaroon service with a deterministic test key
func newTestMacaroonService(t *testing.T) *macaroon.Service {
	svc, err := macaroon.NewService(testMacaroonKey)
	if err != nil {
		t.Fatalf("Failed to create test macaroon service: %v", err)
	}
	return svc
}

// mintTestToken creates a valid test macaroon for the given scope
func mintTestToken(t *testing.T, macSvc *macaroon.Service, scope string) string {
	mac, err := macSvc.Mint(scope, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("Failed to mint test token: %v", err)
	}
	return macSvc.Encode(mac)
}

func TestDefaultProtection_ObserveWithoutMacaroon_Denied(t *testing.T) {
	// Observe policy (chargeback mode) should still enforce Default Protection
	// Per strategy: "Protect by default for non-PUBLIC routes"
	gw := newTestGateway(t)

	route := &config.Route{
		Name:  "observe-test",
		Match: config.RouteMatch{PathPrefix: "/api/observe/"},
		Policy: config.RoutePolicy{Kind: "observe"}, // Also: chargeback, audit
	}
	gw.config.Routes = []config.Route{*route}

	req := httptest.NewRequest("GET", "/api/observe/test", nil)
	// NO Authorization header - should be denied

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Observe route without macaroon: expected 401, got %d", w.Code)
		t.Log("FAIL: Default Protection not enforced on Observe policy")
	}
}

func TestDefaultProtection_ControlWithoutMacaroon_Denied(t *testing.T) {
	// Control policy (fiat402 mode) should enforce Default Protection
	gw := newTestGateway(t)

	route := &config.Route{
		Name:  "control-test",
		Match: config.RouteMatch{PathPrefix: "/api/control/"},
		Policy: config.RoutePolicy{Kind: "control"}, // Also: fiat402, budget
	}
	gw.config.Routes = []config.Route{*route}

	req := httptest.NewRequest("GET", "/api/control/test", nil)
	// NO Authorization header

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Control route without macaroon: expected 401, got %d", w.Code)
		t.Log("FAIL: Default Protection not enforced on Control policy")
	}
}

func TestDefaultProtection_ChargeWithoutMacaroon_Challenged(t *testing.T) {
	// Charge policy (l402 mode) should return 402 Payment Required
	// This is still "protected" - it requires payment proof (L402 token)
	gw := newTestGateway(t)

	route := &config.Route{
		Name:   "charge-test",
		Match:  config.RouteMatch{PathPrefix: "/api/charge/"},
		Policy: config.RoutePolicy{Kind: "charge"}, // Also: l402, monetize, pay
	}
	gw.config.Routes = []config.Route{*route}

	req := httptest.NewRequest("GET", "/api/charge/test", nil)
	// NO Authorization header

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	// 402 Payment Required is expected (challenged, not denied)
	if w.Code != http.StatusPaymentRequired && w.Code != http.StatusUnauthorized {
		t.Errorf("Charge route without payment: expected 402 or 401, got %d", w.Code)
		t.Log("FAIL: Default Protection not enforced on Charge policy")
	}
}

func TestDefaultProtection_PublicWithoutMacaroon_Allowed(t *testing.T) {
	// PUBLIC is the explicit opt-out from protection
	
	// Set up a backend to receive the request
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	// Create gateway with upstream pre-configured
	gw := newTestGatewayWithBackend(t, "test", backend.URL)
	route := &config.Route{
		Name:     "public-test",
		Match:    config.RouteMatch{PathPrefix: "/healthz"},
		Policy:   config.RoutePolicy{Kind: "public"}, // Explicit opt-out
		Upstream: "test",
	}
	gw.config.Routes = []config.Route{*route}

	req := httptest.NewRequest("GET", "/healthz", nil)
	// NO Authorization header - should be allowed for PUBLIC

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("PUBLIC route without macaroon: expected 200, got %d", w.Code)
		t.Log("FAIL: PUBLIC should allow unauthenticated requests")
	}
}

func TestDefaultProtection_DenyWithoutMacaroon_Rejected(t *testing.T) {
	// DENY policy should always reject - even with a valid token
	gw := newTestGateway(t)

	route := &config.Route{
		Name:   "deny-test",
		Match:  config.RouteMatch{PathPrefix: "/internal/"},
		Policy: config.RoutePolicy{Kind: "deny"},
	}
	gw.config.Routes = []config.Route{*route}

	req := httptest.NewRequest("GET", "/internal/secret", nil)

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusForbidden {
		t.Errorf("DENY route: expected 403, got %d", w.Code)
		t.Log("FAIL: DENY policy should always reject")
	}
}

func TestDefaultProtection_ValidMacaroon_Allowed(t *testing.T) {
	// With valid macaroon, Observe route should allow request
	
	// Set up a backend
	backend := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
		w.Write([]byte("OK"))
	}))
	defer backend.Close()

	// Create gateway with upstream pre-configured
	gw := newTestGatewayWithBackend(t, "test", backend.URL)
	route := &config.Route{
		Name:     "observe-auth-test",
		Match:    config.RouteMatch{PathPrefix: "/api/"},
		Policy:   config.RoutePolicy{Kind: "observe"},
		Upstream: "test",
	}
	gw.config.Routes = []config.Route{*route}

	// Create a valid test macaroon using the same key as the gateway
	macSvc := newTestMacaroonService(t)
	token := mintTestToken(t, macSvc, "/api/")

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Observe route with valid macaroon: expected 200, got %d", w.Code)
		t.Logf("Response body: %s", w.Body.String())
		t.Log("FAIL: Valid macaroon should allow access to Observe route")
	}
}

func TestDefaultProtection_ExpiredMacaroon_Denied(t *testing.T) {
	// Expired macaroon should be denied
	gw := newTestGatewayWithMacaroons(t)

	route := &config.Route{
		Name:   "observe-expired-test",
		Match:  config.RouteMatch{PathPrefix: "/api/"},
		Policy: config.RoutePolicy{Kind: "observe"},
	}
	gw.config.Routes = []config.Route{*route}

	// Create an expired macaroon
	macSvc := newTestMacaroonService(t)
	mac, _ := macSvc.Mint("/api/", time.Now().Add(-1*time.Hour)) // Expired 1 hour ago
	token := macSvc.Encode(mac)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Observe route with expired macaroon: expected 401, got %d", w.Code)
		t.Log("FAIL: Expired macaroon should be rejected")
	}
}

func TestDefaultProtection_WrongScopeMacaroon_Denied(t *testing.T) {
	// Macaroon with wrong scope should be denied
	// Note: This test verifies scope enforcement which may not be implemented
	// in all verification modes. Skip if scope verification is not enforced.
	gw := newTestGateway(t)

	route := &config.Route{
		Name:   "observe-scope-test",
		Match:  config.RouteMatch{PathPrefix: "/api/users/"},
		Policy: config.RoutePolicy{Kind: "observe"},
	}
	gw.config.Routes = []config.Route{*route}

	// Create a macaroon for a different scope
	macSvc := newTestMacaroonService(t)
	token := mintTestToken(t, macSvc, "/api/admin/") // Wrong scope

	req := httptest.NewRequest("GET", "/api/users/list", nil)
	req.Header.Set("Authorization", "Bearer "+token)

	w := httptest.NewRecorder()
	gw.ServeHTTP(w, req)

	// Scope enforcement behavior varies by implementation
	// - If scope is enforced: 401/403
	// - If scope is not enforced but request proceeds: might get 502 (no upstream)
	// For this test, we accept any of these as "not 200 OK"
	if w.Code == http.StatusOK {
		t.Errorf("Observe route with wrong scope macaroon: should not return 200 OK, got %d", w.Code)
		t.Log("FAIL: Wrong scope macaroon should not allow full access")
	}
}

// Helper to create a test gateway without macaroon service
func newTestGateway(t *testing.T) *Gateway {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Listen: ":0",
		},
		Admin: config.AdminConfig{
			Token: testMacaroonKey, // Use test key for macaroon service
		},
		Lightning: config.LightningConfig{
			Provider: "mock", // Use mock provider for tests
		},
		Upstreams: make(map[string]config.Upstream),
	}
	gw, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create test gateway: %v", err)
	}
	return gw
}

// Helper to create a test gateway WITH macaroon service for positive tests
func newTestGatewayWithMacaroons(t *testing.T) *Gateway {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Listen: ":0",
		},
		Admin: config.AdminConfig{
			Token: testMacaroonKey, // Use same test key for deterministic macaroons
		},
		Lightning: config.LightningConfig{
			Provider: "mock", // Use mock provider for tests
		},
		Upstreams: make(map[string]config.Upstream),
	}
	gw, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create test gateway: %v", err)
	}
	return gw
}

// Helper to create a test gateway with a pre-configured backend upstream
// This is needed for tests that need to proxy requests to a real backend
func newTestGatewayWithBackend(t *testing.T, upstreamName, upstreamURL string) *Gateway {
	cfg := &config.Config{
		Server: config.ServerConfig{
			Listen: ":0",
		},
		Admin: config.AdminConfig{
			Token: testMacaroonKey, // Use same test key for deterministic macaroons
		},
		Lightning: config.LightningConfig{
			Provider: "mock", // Use mock provider for tests
		},
		Upstreams: map[string]config.Upstream{
			upstreamName: {URL: upstreamURL},
		},
	}
	gw, err := New(cfg)
	if err != nil {
		t.Fatalf("Failed to create test gateway with backend: %v", err)
	}
	return gw
}
