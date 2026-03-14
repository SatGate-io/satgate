package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/proxy"
)

// httpTestEnv holds all components for an HTTP proxy integration test.
type httpTestEnv struct {
	Gateway    *proxy.Gateway
	Config     *config.Config
	MacSvc     *macaroon.Service
	GovSvc     *governance.Service
	Lightning  *lightning.MockProvider
	AdminToken string
}

// newHTTPTestEnv creates a fully wired HTTP proxy test environment.
func newHTTPTestEnv(t *testing.T, cfg *config.Config) *httpTestEnv {
	t.Helper()

	// Allow private IPs for test upstreams (localhost)
	if cfg.Cloud == nil {
		cfg.Cloud = &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}}
	} else if cfg.Cloud.SSRF == nil {
		cfg.Cloud.SSRF = &config.CloudSSRFConfig{AllowPrivateIPs: true}
	} else {
		cfg.Cloud.SSRF.AllowPrivateIPs = true
	}

	macSvc, err := macaroon.NewService("http-integration-test-key")
	if err != nil {
		t.Fatalf("macaroon service: %v", err)
	}
	govSvc := governance.NewService(nil)
	mockLN := lightning.NewMockProvider()

	gw, err := proxy.New(proxy.Options{
		Config:     cfg,
		Macaroon:   macSvc,
		Governance: govSvc,
		Lightning:  mockLN,
	})
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}

	return &httpTestEnv{
		Gateway:    gw,
		Config:     cfg,
		MacSvc:     macSvc,
		GovSvc:     govSvc,
		Lightning:  mockLN,
		AdminToken: cfg.Admin.Token,
	}
}

// mintToken mints a capability token with the given scope and duration.
func (e *httpTestEnv) mintToken(t *testing.T, scope string, dur time.Duration) string {
	t.Helper()
	mac, err := e.MacSvc.Mint(scope, time.Now().Add(dur))
	if err != nil {
		t.Fatalf("mint token: %v", err)
	}
	return e.MacSvc.Encode(mac)
}

// doRequest executes an HTTP request against the gateway and returns the response.
func (e *httpTestEnv) doRequest(t *testing.T, method, path string, headers map[string]string, body io.Reader) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(method, path, body)
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	rr := httptest.NewRecorder()
	e.Gateway.ServeHTTP(rr, req)
	return rr
}

// adminMint mints a token via the admin API.
func (e *httpTestEnv) adminMint(t *testing.T, scope, duration string) (token, signature string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{"scope": scope, "duration": duration})
	rr := e.doRequest(t, "POST", "/api/capability/mint",
		map[string]string{
			"X-Admin-Token": e.AdminToken,
			"Content-Type":  "application/json",
		},
		bytes.NewReader(body),
	)
	if rr.Code != http.StatusOK {
		t.Fatalf("admin mint failed: %d %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode mint response: %v", err)
	}
	return resp["token"].(string), resp["signature"].(string)
}

// adminDelegate delegates a token via the admin API.
func (e *httpTestEnv) adminDelegate(t *testing.T, parentToken string, caveats []string) (token, signature string) {
	t.Helper()
	body, _ := json.Marshal(map[string]interface{}{
		"parentToken": parentToken,
		"caveats":     caveats,
	})
	rr := e.doRequest(t, "POST", "/api/capability/delegate",
		map[string]string{"Content-Type": "application/json"},
		bytes.NewReader(body),
	)
	if rr.Code != http.StatusOK {
		t.Fatalf("delegate failed: %d %s", rr.Code, rr.Body.String())
	}
	var resp map[string]interface{}
	if err := json.NewDecoder(rr.Body).Decode(&resp); err != nil {
		t.Fatalf("decode delegate response: %v", err)
	}
	return resp["token"].(string), resp["signature"].(string)
}

// adminBan bans a token via the admin API.
func (e *httpTestEnv) adminBan(t *testing.T, signature, reason string) {
	t.Helper()
	body, _ := json.Marshal(map[string]string{
		"tokenSignature": signature,
		"reason":         reason,
	})
	rr := e.doRequest(t, "POST", "/api/governance/ban",
		map[string]string{
			"X-Admin-Token": e.AdminToken,
			"Content-Type":  "application/json",
		},
		bytes.NewReader(body),
	)
	if rr.Code != http.StatusOK {
		t.Fatalf("ban failed: %d %s", rr.Code, rr.Body.String())
	}
}

// newUpstream creates a test HTTP server that echoes the request path.
func newUpstream(t *testing.T) *httptest.Server {
	t.Helper()
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]string{
			"path":   r.URL.Path,
			"method": r.Method,
		})
	}))
}

// newPathRecordingUpstream creates an upstream that records received paths.
func newPathRecordingUpstream(t *testing.T) (*httptest.Server, *[]string) {
	t.Helper()
	var mu sync.Mutex
	paths := &[]string{}
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		mu.Lock()
		*paths = append(*paths, r.URL.Path)
		mu.Unlock()
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"path":%q}`, r.URL.Path)
	}))
	return srv, paths
}

// ---------- 1. Full auth lifecycle ----------

func TestHTTPProxy_AuthLifecycle(t *testing.T) {
	upstream := newUpstream(t)
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin:  config.AdminConfig{Token: "test-admin-secret"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "api-read",
				Match:    config.RouteMatch{PathPrefix: "/api/data"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "capability", Scope: "api:read"},
			},
			{
				Name:     "api-admin",
				Match:    config.RouteMatch{PathPrefix: "/api/admin"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "capability", Scope: "api:admin"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)

	// Step 1: Mint token with api:read scope — can access read routes
	readToken, _ := env.adminMint(t, "api:read", "1h")

	rr := env.doRequest(t, "GET", "/api/data/users",
		map[string]string{"Authorization": "Bearer " + readToken}, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("read token should access /api/data: %d %s", rr.Code, rr.Body.String())
	}

	// Step 2: Read token should NOT access admin route (scope mismatch)
	rr = env.doRequest(t, "GET", "/api/admin/settings",
		map[string]string{"Authorization": "Bearer " + readToken}, nil)
	if rr.Code != http.StatusForbidden {
		t.Fatalf("read token should NOT access /api/admin: expected 403, got %d %s", rr.Code, rr.Body.String())
	}

	// Step 3: Mint admin-scoped token — can access admin route
	adminToken, adminSig := env.adminMint(t, "api:admin", "1h")
	_ = adminSig

	rr = env.doRequest(t, "GET", "/api/admin/settings",
		map[string]string{"Authorization": "Bearer " + adminToken}, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("admin token should access /api/admin: %d %s", rr.Code, rr.Body.String())
	}

	// Step 4: Delegate admin token — child inherits scope, still works
	childToken, _ := env.adminDelegate(t, adminToken, []string{
		fmt.Sprintf("expires = %d", time.Now().Add(30*time.Minute).UnixMilli()),
	})

	rr = env.doRequest(t, "GET", "/api/admin/settings",
		map[string]string{"Authorization": "Bearer " + childToken}, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("delegated admin token should access /api/admin: %d %s", rr.Code, rr.Body.String())
	}

	// Step 5: No auth at all — should fail
	rr = env.doRequest(t, "GET", "/api/data/users", nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("no auth should be 401: got %d", rr.Code)
	}

	// Step 6: Invalid token — should fail
	rr = env.doRequest(t, "GET", "/api/data/users",
		map[string]string{"Authorization": "Bearer invalid-garbage-token"}, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("invalid token should be 401: got %d", rr.Code)
	}
}

// ---------- 2. L402 payment flow ----------

// accessibleMockProvider wraps lightning.MockProvider to expose invoices.
type accessibleMockProvider struct {
	*lightning.MockProvider
	mu       sync.Mutex
	invoices map[string]*lightning.Invoice // paymentHash → invoice
}

func newAccessibleMockProvider() *accessibleMockProvider {
	return &accessibleMockProvider{
		MockProvider: lightning.NewMockProvider(),
		invoices:     make(map[string]*lightning.Invoice),
	}
}

func (m *accessibleMockProvider) CreateInvoice(amountSats int64, memo string) (*lightning.Invoice, error) {
	inv, err := m.MockProvider.CreateInvoice(amountSats, memo)
	if err != nil {
		return nil, err
	}
	m.mu.Lock()
	m.invoices[inv.PaymentHash] = inv
	m.mu.Unlock()
	return inv, nil
}

func (m *accessibleMockProvider) GetInvoice(paymentHash string) *lightning.Invoice {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.invoices[paymentHash]
}

// newHTTPTestEnvWithAccessibleLN creates an env with an accessible mock provider.
func newHTTPTestEnvWithAccessibleLN(t *testing.T, cfg *config.Config) (*httpTestEnv, *accessibleMockProvider) {
	t.Helper()

	if cfg.Cloud == nil {
		cfg.Cloud = &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}}
	} else if cfg.Cloud.SSRF == nil {
		cfg.Cloud.SSRF = &config.CloudSSRFConfig{AllowPrivateIPs: true}
	} else {
		cfg.Cloud.SSRF.AllowPrivateIPs = true
	}

	macSvc, err := macaroon.NewService("http-integration-test-key")
	if err != nil {
		t.Fatalf("macaroon service: %v", err)
	}
	govSvc := governance.NewService(nil)
	mockLN := newAccessibleMockProvider()

	gw, err := proxy.New(proxy.Options{
		Config:     cfg,
		Macaroon:   macSvc,
		Governance: govSvc,
		Lightning:  mockLN,
	})
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}

	env := &httpTestEnv{
		Gateway:    gw,
		Config:     cfg,
		MacSvc:     macSvc,
		GovSvc:     govSvc,
		Lightning:  mockLN.MockProvider,
		AdminToken: cfg.Admin.Token,
	}
	return env, mockLN
}

// TestHTTPProxy_L402PaymentFlowE2E tests the full L402 flow using JSON-encoded
// macaroons (the binary encoding via gopkg.in/macaroon.v2 uses a different HMAC
// chain than our manual implementation, so we construct the L402 token directly).
func TestHTTPProxy_L402PaymentFlowE2E(t *testing.T) {
	upstream := newUpstream(t)
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin:  config.AdminConfig{Token: "test-admin"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "paid-api",
				Match:    config.RouteMatch{PathPrefix: "/paid/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "l402", PriceSats: 50},
			},
		},
	}

	env, mockLN := newHTTPTestEnvWithAccessibleLN(t, cfg)

	// Step 1: Hit L402 route without auth — get 402 challenge
	rr := env.doRequest(t, "GET", "/paid/data", nil, nil)
	if rr.Code != http.StatusPaymentRequired {
		t.Fatalf("expected 402, got %d", rr.Code)
	}

	// Step 2: Verify challenge header format
	wwwAuth := rr.Header().Get("WWW-Authenticate")
	if !strings.HasPrefix(wwwAuth, "L402 macaroon=") {
		t.Fatalf("bad challenge format: %s", wwwAuth)
	}
	if !strings.Contains(wwwAuth, "invoice=") {
		t.Fatal("challenge missing invoice")
	}

	// Step 3: Parse challenge body
	var challengeBody map[string]interface{}
	json.NewDecoder(rr.Body).Decode(&challengeBody)
	if challengeBody["error"] != "payment_required" {
		t.Errorf("expected payment_required error, got %v", challengeBody["error"])
	}
	paymentHash := challengeBody["payment_hash"].(string)
	if paymentHash == "" {
		t.Fatal("challenge body missing payment_hash")
	}

	// Step 4: "Pay" — get preimage from mock provider
	inv := mockLN.GetInvoice(paymentHash)
	if inv == nil {
		t.Fatalf("no invoice for payment_hash %s", paymentHash)
	}

	// Step 5: Construct L402 token using JSON encoding (which round-trips correctly)
	// Mint macaroon with payment_hash caveat, encode as JSON, combine with preimage
	mac, err := env.MacSvc.Mint("l402", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	mac.AddCaveat("payment_hash", paymentHash)
	mac.Signature = env.MacSvc.RecalculateSignature(mac)
	macaroonStr := env.MacSvc.Encode(mac)

	l402Token := macaroonStr + ":" + inv.Preimage
	rr = env.doRequest(t, "GET", "/paid/data",
		map[string]string{"Authorization": "L402 " + l402Token}, nil)
	if rr.Code != http.StatusOK {
		t.Fatalf("paid request should succeed: %d %s", rr.Code, rr.Body.String())
	}

	// Step 6: Replay the same preimage — should be rejected (replay guard)
	rr = env.doRequest(t, "GET", "/paid/data",
		map[string]string{"Authorization": "L402 " + l402Token}, nil)
	if rr.Code == http.StatusOK {
		t.Fatal("replay should NOT succeed")
	}

	// Step 7: Different preimage for same payment_hash should fail (hash mismatch)
	fakePreimage := "aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa"
	fakeToken := macaroonStr + ":" + fakePreimage
	rr = env.doRequest(t, "GET", "/paid/data",
		map[string]string{"Authorization": "L402 " + fakeToken}, nil)
	if rr.Code == http.StatusOK {
		t.Fatal("fake preimage should not succeed")
	}
}

// ---------- 3. Budget enforcement / ban flow ----------

func TestHTTPProxy_BudgetBanFlow(t *testing.T) {
	upstream := newUpstream(t)
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin:  config.AdminConfig{Token: "admin-secret-123"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "api",
				Match:    config.RouteMatch{PathPrefix: "/api/v1/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "capability"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)

	// Step 1: Mint token
	token, sig := env.adminMint(t, "api:*", "1h")

	// Step 2: Make requests — verify they succeed and metrics track
	for i := 0; i < 3; i++ {
		rr := env.doRequest(t, "GET", "/api/v1/data",
			map[string]string{"Authorization": "Bearer " + token}, nil)
		if rr.Code != http.StatusOK {
			t.Fatalf("request %d should succeed: %d", i, rr.Code)
		}
	}

	// Step 3: Verify usage tracked
	usage := env.GovSvc.GetUsage(sig)
	if usage == nil {
		t.Fatal("expected usage stats")
	}
	if usage.TotalRequests != 3 {
		t.Errorf("expected 3 requests, got %d", usage.TotalRequests)
	}

	// Step 4: Ban the token
	env.adminBan(t, sig, "compromised")

	// Step 5: Token should be rejected
	rr := env.doRequest(t, "GET", "/api/v1/data",
		map[string]string{"Authorization": "Bearer " + token}, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Fatalf("banned token should be rejected: expected 401, got %d", rr.Code)
	}

	// Step 6: Verify governance graph reflects ban
	graphRR := env.doRequest(t, "GET", "/api/governance/graph",
		map[string]string{"X-Admin-Token": env.AdminToken}, nil)
	if graphRR.Code != http.StatusOK {
		t.Fatalf("graph request failed: %d", graphRR.Code)
	}
	var graph map[string]interface{}
	json.NewDecoder(graphRR.Body).Decode(&graph)

	stats := graph["stats"].(map[string]interface{})
	bannedCount := int(stats["banned"].(float64))
	if bannedCount < 1 {
		t.Errorf("expected at least 1 banned token in graph, got %d", bannedCount)
	}
}

// ---------- 4. Multi-route isolation ----------

func TestHTTPProxy_MultiRouteIsolation(t *testing.T) {
	upstream := newUpstream(t)
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin:  config.AdminConfig{Token: "admin-token"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "public-route",
				Match:    config.RouteMatch{PathPrefix: "/public/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "public"},
			},
			{
				Name:     "cap-route",
				Match:    config.RouteMatch{PathPrefix: "/secure/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "capability"},
			},
			{
				Name:     "l402-route",
				Match:    config.RouteMatch{PathPrefix: "/paid/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "l402", PriceSats: 10},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)
	token := env.mintToken(t, "api:*", time.Hour)

	// Public route — no auth needed
	rr := env.doRequest(t, "GET", "/public/data", nil, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("public route should work without auth: %d", rr.Code)
	}

	// Capability route without auth — should fail
	rr = env.doRequest(t, "GET", "/secure/data", nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("capability route without auth should be 401: got %d", rr.Code)
	}

	// Capability route with auth — should succeed
	rr = env.doRequest(t, "GET", "/secure/data",
		map[string]string{"Authorization": "Bearer " + token}, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("capability route with auth should be 200: got %d", rr.Code)
	}

	// L402 route without payment — should be 402
	rr = env.doRequest(t, "GET", "/paid/data", nil, nil)
	if rr.Code != http.StatusPaymentRequired {
		t.Errorf("l402 route without payment should be 402: got %d", rr.Code)
	}

	// Verify metrics isolation
	metrics := env.Gateway.GetMetrics()
	if metrics.TotalPublic.Load() != 1 {
		t.Errorf("expected 1 public request, got %d", metrics.TotalPublic.Load())
	}
	if metrics.TotalCapability.Load() != 2 {
		// Both the unauthorized and authorized requests go through handleCapability
		// (the metric increments before auth check)
		t.Errorf("expected 2 capability requests (auth+unauth), got %d", metrics.TotalCapability.Load())
	}
}

// ---------- 5. Upstream failure handling ----------

func TestHTTPProxy_UpstreamFailure(t *testing.T) {
	var healthy atomic.Bool

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !healthy.Load() {
			// Close connection abruptly
			hj, ok := w.(http.Hijacker)
			if ok {
				conn, _, _ := hj.Hijack()
				conn.Close()
				return
			}
			w.WriteHeader(http.StatusInternalServerError)
			return
		}
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"ok"}`)
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin:  config.AdminConfig{Token: "admin"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "api",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "public"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)

	// Start unhealthy
	healthy.Store(false)
	rr := env.doRequest(t, "GET", "/api/data", nil, nil)
	if rr.Code != http.StatusBadGateway {
		t.Errorf("expected 502 for dead upstream, got %d", rr.Code)
	}

	// Recover
	healthy.Store(true)
	rr = env.doRequest(t, "GET", "/api/data", nil, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("expected 200 after upstream recovery, got %d", rr.Code)
	}
}

// ---------- 6. Concurrent load ----------

func TestHTTPProxy_ConcurrentLoad(t *testing.T) {
	var reqCount atomic.Int64
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		reqCount.Add(1)
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"ok":true}`)
	}))
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin:  config.AdminConfig{Token: "admin"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "public",
				Match:    config.RouteMatch{PathPrefix: "/pub/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "public"},
			},
			{
				Name:     "cap",
				Match:    config.RouteMatch{PathPrefix: "/cap/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "capability"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)
	token := env.mintToken(t, "api:*", time.Hour)

	const goroutines = 50
	var wg sync.WaitGroup
	errors := make([]error, goroutines)

	for i := 0; i < goroutines; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			var rr *httptest.ResponseRecorder
			if idx%2 == 0 {
				rr = env.doRequest(t, "GET", "/pub/data", nil, nil)
			} else {
				rr = env.doRequest(t, "GET", "/cap/data",
					map[string]string{"Authorization": "Bearer " + token}, nil)
			}
			if rr.Code != http.StatusOK {
				errors[idx] = fmt.Errorf("goroutine %d: expected 200, got %d", idx, rr.Code)
			}
		}(i)
	}

	wg.Wait()

	for _, err := range errors {
		if err != nil {
			t.Error(err)
		}
	}

	// All requests should have reached upstream
	if reqCount.Load() != goroutines {
		t.Errorf("expected %d upstream requests, got %d", goroutines, reqCount.Load())
	}

	// Metrics should be consistent
	metrics := env.Gateway.GetMetrics()
	totalMetric := metrics.TotalPublic.Load() + metrics.TotalCapability.Load()
	if totalMetric != goroutines {
		t.Errorf("expected %d total metric requests, got %d (public=%d, cap=%d)",
			goroutines, totalMetric, metrics.TotalPublic.Load(), metrics.TotalCapability.Load())
	}
}

// ---------- 7. CORS end-to-end ----------

func TestHTTPProxy_CORSEndToEnd(t *testing.T) {
	upstream := newUpstream(t)
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin: config.AdminConfig{
			Token:                "admin",
			CORSAllowedOrigins:   []string{"https://dashboard.satgate.io", "https://app.example.com"},
			CORSAllowCredentials: true,
		},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "api",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "public"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)

	// Preflight from allowed origin
	rr := env.doRequest(t, "OPTIONS", "/api/data",
		map[string]string{"Origin": "https://dashboard.satgate.io"}, nil)
	if rr.Code != http.StatusNoContent {
		t.Errorf("preflight should be 204: got %d", rr.Code)
	}
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://dashboard.satgate.io" {
		t.Errorf("expected allowed origin in response, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}
	if rr.Header().Get("Access-Control-Allow-Credentials") != "true" {
		t.Error("expected credentials header")
	}

	// Regular request from allowed origin
	rr = env.doRequest(t, "GET", "/api/data",
		map[string]string{"Origin": "https://app.example.com"}, nil)
	if rr.Header().Get("Access-Control-Allow-Origin") != "https://app.example.com" {
		t.Errorf("expected allowed origin, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}

	// Request from disallowed origin — no CORS headers
	rr = env.doRequest(t, "GET", "/api/data",
		map[string]string{"Origin": "https://evil.com"}, nil)
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Errorf("disallowed origin should NOT get CORS headers, got %q", rr.Header().Get("Access-Control-Allow-Origin"))
	}

	// Request without Origin — no CORS headers (regular non-browser request)
	rr = env.doRequest(t, "GET", "/api/data", nil, nil)
	if rr.Header().Get("Access-Control-Allow-Origin") != "" {
		t.Error("request without Origin should not get CORS headers")
	}
}

// ---------- 8. Admin rate limiting ----------

func TestHTTPProxy_AdminRateLimiting(t *testing.T) {
	upstream := newUpstream(t)
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin: config.AdminConfig{
			Token:              "admin-token",
			RateLimitPerMinute: 5, // Very low limit for testing
		},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "public",
				Match:    config.RouteMatch{PathPrefix: "/data/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "public"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)

	// Fire admin API calls until rate limited
	rateLimited := false
	for i := 0; i < 20; i++ {
		body, _ := json.Marshal(map[string]string{"token": "fake"})
		rr := env.doRequest(t, "POST", "/api/capability/validate",
			map[string]string{"Content-Type": "application/json"},
			bytes.NewReader(body),
		)
		if rr.Code == http.StatusTooManyRequests {
			rateLimited = true
			// Verify Retry-After header
			if rr.Header().Get("Retry-After") == "" {
				t.Error("expected Retry-After header on 429")
			}
			break
		}
	}
	if !rateLimited {
		t.Error("expected rate limiting to kick in on admin API")
	}

	// Non-admin route should be unaffected
	rr := env.doRequest(t, "GET", "/data/test", nil, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("non-admin route should be unaffected by admin rate limit: got %d", rr.Code)
	}
}

// ---------- 9. Strip-prefix + rewrite ----------

func TestHTTPProxy_StripPrefixAndRewrite(t *testing.T) {
	srv, paths := newPathRecordingUpstream(t)
	defer srv.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin:  config.AdminConfig{Token: "admin"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: srv.URL},
		},
		Routes: []config.Route{
			{
				Name:        "strip",
				Match:       config.RouteMatch{PathPrefix: "/gateway/v1/"},
				Upstream:    "backend",
				StripPrefix: true,
				Policy:      config.RoutePolicy{Kind: "public"},
			},
			{
				Name:     "rewrite",
				Match:    config.RouteMatch{PathPrefix: "/old-api/"},
				Upstream: "backend",
				Rewrite:  "/new-api/v2/handler",
				Policy:   config.RoutePolicy{Kind: "public"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)

	// Strip-prefix: /gateway/v1/users → /users
	env.doRequest(t, "GET", "/gateway/v1/users", nil, nil)

	// Rewrite: /old-api/anything → /new-api/v2/handler
	env.doRequest(t, "GET", "/old-api/something", nil, nil)

	if len(*paths) != 2 {
		t.Fatalf("expected 2 upstream requests, got %d", len(*paths))
	}

	if (*paths)[0] != "/users" {
		t.Errorf("strip-prefix: expected upstream path /users, got %s", (*paths)[0])
	}

	if (*paths)[1] != "/new-api/v2/handler" {
		t.Errorf("rewrite: expected upstream path /new-api/v2/handler, got %s", (*paths)[1])
	}
}

// ---------- 10. Health endpoint ----------

func TestHTTPProxy_HealthEndpoint(t *testing.T) {
	cfg := &config.Config{
		Server:    config.ServerConfig{Listen: ":0"},
		Admin:     config.AdminConfig{Token: "admin"},
		Upstreams: map[string]config.Upstream{},
		Routes: []config.Route{
			{
				Name:   "dummy",
				Match:  config.RouteMatch{PathPrefix: "/api/"},
				Policy: config.RoutePolicy{Kind: "public"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)

	for _, path := range []string{"/health", "/healthz"} {
		rr := env.doRequest(t, "GET", path, nil, nil)
		if rr.Code != http.StatusOK {
			t.Errorf("%s: expected 200, got %d", path, rr.Code)
		}
		var resp map[string]interface{}
		json.NewDecoder(rr.Body).Decode(&resp)
		if resp["status"] != "healthy" {
			t.Errorf("%s: expected healthy status, got %v", path, resp["status"])
		}
	}
}

// ---------- Additional: No matching route ----------

func TestHTTPProxy_NoMatchingRoute(t *testing.T) {
	upstream := newUpstream(t)
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin:  config.AdminConfig{Token: "admin"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "api",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "public"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)

	rr := env.doRequest(t, "GET", "/unknown/path", nil, nil)
	if rr.Code != http.StatusNotFound {
		t.Errorf("expected 404 for unmatched route, got %d", rr.Code)
	}
}

// ---------- Capability ping and admin scope endpoints ----------

func TestHTTPProxy_CapabilityPingAndAdmin(t *testing.T) {
	cfg := &config.Config{
		Server:    config.ServerConfig{Listen: ":0"},
		Admin:     config.AdminConfig{Token: "admin"},
		Upstreams: map[string]config.Upstream{},
		Routes: []config.Route{
			{
				Name:   "dummy",
				Match:  config.RouteMatch{PathPrefix: "/x/"},
				Policy: config.RoutePolicy{Kind: "public"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)

	// Mint token with api:capability:admin scope
	adminToken := env.mintToken(t, "api:capability:admin", time.Hour)

	// Ping should succeed
	rr := env.doRequest(t, "GET", "/api/capability/ping",
		map[string]string{"Authorization": "Bearer " + adminToken}, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("ping should succeed: %d %s", rr.Code, rr.Body.String())
	}

	// Admin endpoint should succeed with admin scope
	rr = env.doRequest(t, "GET", "/api/capability/admin",
		map[string]string{"Authorization": "Bearer " + adminToken}, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("admin endpoint should succeed with admin scope: %d %s", rr.Code, rr.Body.String())
	}

	// Mint token with limited scope — admin endpoint should fail
	readToken := env.mintToken(t, "api:read", time.Hour)
	rr = env.doRequest(t, "GET", "/api/capability/admin",
		map[string]string{"Authorization": "Bearer " + readToken}, nil)
	if rr.Code != http.StatusForbidden {
		t.Errorf("admin endpoint should deny limited scope: expected 403, got %d", rr.Code)
	}
}

// ---------- Multi-route capability with exact match priority ----------

func TestHTTPProxy_ExactMatchPriority(t *testing.T) {
	upstream := newUpstream(t)
	defer upstream.Close()

	cfg := &config.Config{
		Server: config.ServerConfig{Listen: ":0"},
		Admin:  config.AdminConfig{Token: "admin"},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstream.URL},
		},
		Routes: []config.Route{
			{
				Name:     "exact",
				Match:    config.RouteMatch{PathExact: "/api/special"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "public"},
			},
			{
				Name:     "prefix",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "capability"},
			},
		},
	}

	env := newHTTPTestEnv(t, cfg)

	// Exact match — public, no auth needed
	rr := env.doRequest(t, "GET", "/api/special", nil, nil)
	if rr.Code != http.StatusOK {
		t.Errorf("exact match should be public: got %d", rr.Code)
	}

	// Prefix match — requires auth
	rr = env.doRequest(t, "GET", "/api/other", nil, nil)
	if rr.Code != http.StatusUnauthorized {
		t.Errorf("prefix match should require auth: expected 401, got %d", rr.Code)
	}
}
