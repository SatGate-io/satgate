// Package proxy provides the core HTTP reverse proxy for SatGate OSS.
//
// This is the OSS implementation supporting:
// - public: No authentication required
// - capability: Macaroon-based authentication
// - l402: Lightning payment protocol
//
// Enterprise features (observe, control, charge, tenant routing, budgets)
// are NOT included in this implementation.
package proxy

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"crypto/tls"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/mcp"
)

// Gateway is the OSS reverse proxy with capability token support.
type Gateway struct {
	config      *config.Config
	macaroonSvc *macaroon.Service
	governance  *governance.Service
	lightning   lightning.Provider
	proxies     map[string]*httputil.ReverseProxy
	proxyMu     sync.RWMutex
	metrics     *Metrics

	// Optional hooks (set via SetXxx methods)
	metricsHook MetricsHook
}

// Metrics tracks gateway statistics.
type Metrics struct {
	TotalRequests   int64
	TotalPublic     int64
	TotalCapability int64
	TotalL402       int64
	TotalErrors     int64
}

// MetricsHook allows external metrics collection.
type MetricsHook interface {
	RecordRequest(route string, policy string, statusCode int, latency time.Duration)
}

// Options configures the Gateway.
type Options struct {
	Config     *config.Config
	Macaroon   *macaroon.Service
	Governance *governance.Service
	Lightning  lightning.Provider
}

// New creates a new OSS Gateway.
func New(opts Options) (*Gateway, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if opts.Macaroon == nil {
		return nil, fmt.Errorf("macaroon service is required")
	}

	g := &Gateway{
		config:      opts.Config,
		macaroonSvc: opts.Macaroon,
		governance:  opts.Governance,
		lightning:   opts.Lightning,
		proxies:     make(map[string]*httputil.ReverseProxy),
		metrics:     &Metrics{},
	}

	// Initialize proxies for configured upstreams
	for name, upstream := range opts.Config.Upstreams {
		proxy, err := g.createProxy(upstream.URL)
		if err != nil {
			return nil, fmt.Errorf("failed to create proxy for upstream %s: %w", name, err)
		}
		g.proxies[name] = proxy
	}

	return g, nil
}

// SetMetricsHook sets an optional metrics collection hook.
func (g *Gateway) SetMetricsHook(hook MetricsHook) {
	g.metricsHook = hook
}

// ServeHTTP implements http.Handler.
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// CORS headers for browser requests — only allow configured origins.
	// If no AllowedOrigins are configured, CORS headers are not set (safe default).
	origin := r.Header.Get("Origin")
	if origin != "" && len(g.config.Admin.CORSAllowedOrigins) > 0 {
		originAllowed := false
		for _, allowed := range g.config.Admin.CORSAllowedOrigins {
			if allowed == origin {
				originAllowed = true
				break
			}
		}
		if originAllowed {
			w.Header().Del("Access-Control-Allow-Origin")
			w.Header().Del("Access-Control-Allow-Methods")
			w.Header().Del("Access-Control-Allow-Headers")
			w.Header().Del("Access-Control-Expose-Headers")
			w.Header().Del("Access-Control-Allow-Credentials")
			w.Header().Set("Access-Control-Allow-Origin", origin)
			w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
			w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Admin-Token, X-Request-ID")
			w.Header().Set("Access-Control-Expose-Headers", "WWW-Authenticate")
			if g.config.Admin.CORSAllowCredentials {
				w.Header().Set("Access-Control-Allow-Credentials", "true")
			}
		}
	}

	// Handle CORS preflight
	if r.Method == "OPTIONS" {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	// Built-in health endpoint (doesn't depend on upstreams)
	if r.URL.Path == "/health" || r.URL.Path == "/healthz" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"satgate-oss","routes":%d}`, len(g.config.Routes))
		return
	}

	// Check payment status endpoint (for polling from frontend)
	if strings.HasPrefix(r.URL.Path, "/check-payment/") {
		g.handleCheckPayment(w, r)
		return
	}

	// Admin endpoint: Capability token minting
	if r.URL.Path == "/api/capability/mint" && r.Method == "POST" {
		g.handleCapabilityMint(w, r)
		return
	}

	// Admin endpoint: Token validation
	if r.URL.Path == "/api/capability/validate" && r.Method == "POST" {
		g.handleCapabilityValidate(w, r)
		return
	}

	// Admin endpoint: Token delegation (both direct and demo path)
	if (r.URL.Path == "/api/capability/delegate" || r.URL.Path == "/api/capability/demo/delegate") && r.Method == "POST" {
		g.handleCapabilityDelegate(w, r)
		return
	}

	// Capability ping endpoint: Verifies a capability token and returns success
	if r.URL.Path == "/api/capability/ping" && r.Method == "GET" {
		g.handleCapabilityPing(w, r)
		return
	}

	// Capability admin endpoint: Requires scope = api:capability:admin (for scope enforcement demo)
	if r.URL.Path == "/api/capability/admin" && r.Method == "GET" {
		g.handleCapabilityAdmin(w, r)
		return
	}

	// Governance endpoint: Ban a token (demo)
	if r.URL.Path == "/api/governance/ban" && r.Method == "POST" {
		g.handleGovernanceBan(w, r)
		return
	}

	// Governance endpoint: Get token graph for dashboard
	if r.URL.Path == "/api/governance/graph" && r.Method == "GET" {
		g.handleGovernanceGraph(w, r)
		return
	}

	// Governance endpoint: Reset dashboard data
	if r.URL.Path == "/api/governance/reset" && r.Method == "POST" {
		g.handleGovernanceReset(w, r)
		return
	}

	start := time.Now()
	g.metrics.TotalRequests++

	// Find matching route
	route := g.matchRoute(r)
	if route == nil {
		http.Error(w, "No matching route", http.StatusNotFound)
		g.metrics.TotalErrors++
		return
	}

	// Get policy kind (default to capability for protection)
	policyKind := "capability"
	if route.Policy.Kind != "" {
		policyKind = route.Policy.Kind
	}

	// Create response wrapper to capture status code
	wrapped := &statusWriter{ResponseWriter: w, statusCode: http.StatusOK}

	// Handle based on policy
	switch policyKind {
	case "public":
		g.metrics.TotalPublic++
		g.handlePublic(wrapped, r, route)

	case "capability":
		g.metrics.TotalCapability++
		g.handleCapability(wrapped, r, route)

	case "l402":
		g.metrics.TotalL402++
		g.handleL402(wrapped, r, route)

	default:
		// Unknown policy in OSS - reject
		log.Warn().Str("policy", policyKind).Str("route", route.Name).
			Msg("Unknown policy kind in OSS build")
		http.Error(w, fmt.Sprintf("Policy '%s' not supported in OSS", policyKind), http.StatusNotImplemented)
		g.metrics.TotalErrors++
		return
	}

	// Record metrics
	if g.metricsHook != nil {
		g.metricsHook.RecordRequest(route.Name, policyKind, wrapped.statusCode, time.Since(start))
	}
}

// handlePublic allows requests without authentication.
func (g *Gateway) handlePublic(w http.ResponseWriter, r *http.Request, route *config.Route) {
	g.proxyRequest(w, r, route)
}

// handleCapability requires a valid Macaroon token.
func (g *Gateway) handleCapability(w http.ResponseWriter, r *http.Request, route *config.Route) {
	// Extract token from Authorization header
	token := extractBearerToken(r)
	if token == "" {
		w.Header().Set("WWW-Authenticate", `Bearer realm="SatGate"`)
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	// Verify macaroon
	mac, err := g.macaroonSvc.Verify(token)
	if err != nil {
		log.Debug().Err(err).Msg("Macaroon verification failed")
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Check governance ban list
	if g.governance != nil && g.governance.IsBanned(mac.Signature) {
		http.Error(w, "Token revoked", http.StatusUnauthorized)
		return
	}

	// Check scope if required
	if route.Policy.Scope != "" {
		if !mac.HasScope(route.Policy.Scope) {
			http.Error(w, "Insufficient scope", http.StatusForbidden)
			return
		}
	}

	// Record usage for governance
	if g.governance != nil {
		g.governance.RecordUsage(mac.Signature, r.URL.Path)
	}

	// Proxy the request
	g.proxyRequest(w, r, route)
}

// handleL402 requires Lightning payment proof.
func (g *Gateway) handleL402(w http.ResponseWriter, r *http.Request, route *config.Route) {
	// Check for existing L402 token
	token := extractL402Token(r)
	if token != "" {
		// Verify L402 token (macaroon:preimage format)
		if g.verifyL402Token(r.Context(), token, route) {
			g.proxyRequest(w, r, route)
			return
		}
	}

	// No valid payment - issue challenge
	g.issueL402Challenge(w, r, route)
}

// verifyL402Token verifies an L402 payment proof.
func (g *Gateway) verifyL402Token(ctx context.Context, token string, route *config.Route) bool {
	log.Info().Str("token_len", fmt.Sprintf("%d", len(token))).Msg("verifyL402Token: starting verification")
	
	// L402 format: macaroon:preimage
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		log.Warn().Str("token_preview", token[:min(50, len(token))]).Msg("L402: invalid token format, expected macaroon:preimage")
		return false
	}

	macaroonStr, preimageHex := parts[0], parts[1]
	log.Info().
		Int("macaroon_len", len(macaroonStr)).
		Int("preimage_len", len(preimageHex)).
		Str("preimage", preimageHex).
		Msg("verifyL402Token: parsed token components")

	// Verify macaroon signature
	mac, err := g.macaroonSvc.Verify(macaroonStr)
	if err != nil {
		log.Warn().Err(err).Str("macaroon_preview", macaroonStr[:min(30, len(macaroonStr))]).Msg("L402: macaroon verification failed")
		return false
	}

	// Check for payment_hash caveat
	paymentHash := mac.GetCaveat("payment_hash")
	if paymentHash == "" {
		log.Warn().Msg("L402: no payment_hash caveat in macaroon")
		return false
	}
	log.Info().Str("payment_hash", paymentHash).Msg("verifyL402Token: found payment_hash in macaroon")

	// CRYPTOGRAPHIC PROOF: Verify that SHA256(preimage) == payment_hash
	// This is the definitive proof of payment - no need to query the Lightning node
	preimageBytes, err := hex.DecodeString(preimageHex)
	if err != nil {
		log.Warn().Err(err).Str("preimage", preimageHex).Msg("L402: invalid preimage hex")
		return false
	}

	// Compute SHA256 of preimage
	hash := sha256.Sum256(preimageBytes)
	computedHash := hex.EncodeToString(hash[:])

	log.Info().
		Str("computed", computedHash).
		Str("expected", paymentHash).
		Bool("match", computedHash == paymentHash).
		Msg("verifyL402Token: hash comparison")

	if computedHash != paymentHash {
		log.Warn().
			Str("computed", computedHash).
			Str("expected", paymentHash).
			Msg("L402: preimage hash mismatch - payment verification failed")
		return false
	}

	log.Info().Str("payment_hash", paymentHash[:16]+"...").Msg("L402: payment verified via preimage ✓")
	return true
}

// issueL402Challenge issues an HTTP 402 payment challenge.
func (g *Gateway) issueL402Challenge(w http.ResponseWriter, r *http.Request, route *config.Route) {
	if g.lightning == nil {
		http.Error(w, "Lightning provider not configured", http.StatusServiceUnavailable)
		return
	}

	// Get price from route config
	priceSats := int64(100) // Default
	if route.Policy.PriceSats > 0 {
		priceSats = route.Policy.PriceSats
	}

	// Create invoice
	inv, err := g.lightning.CreateInvoice(priceSats, fmt.Sprintf("SatGate: %s", route.Name))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Lightning invoice")
		http.Error(w, "Payment service unavailable", http.StatusServiceUnavailable)
		return
	}
	invoice := inv.Bolt11
	paymentHash := inv.PaymentHash

	// Create macaroon with payment_hash caveat
	mac, err := g.macaroonSvc.Mint("l402", time.Now().Add(1*time.Hour))
	if err != nil {
		log.Error().Err(err).Msg("Failed to mint L402 macaroon")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	mac.AddCaveat("payment_hash", paymentHash)
	// CRITICAL: Recalculate signature after adding caveats!
	mac.Signature = g.macaroonSvc.RecalculateSignature(mac)
	macaroonStr := g.macaroonSvc.Encode(mac)

	// Set L402 challenge header
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`L402 macaroon="%s", invoice="%s"`, macaroonStr, invoice))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)

	// Write JSON response with payment_hash so frontend can poll
	fmt.Fprintf(w, `{"error":"payment_required","invoice":"%s","amount_sats":%d,"payment_hash":"%s"}`, invoice, priceSats, paymentHash)
}

// handleCheckPayment allows frontend to poll for payment status
func (g *Gateway) handleCheckPayment(w http.ResponseWriter, r *http.Request) {
	// Extract payment hash from URL: /check-payment/{payment_hash}
	paymentHash := strings.TrimPrefix(r.URL.Path, "/check-payment/")
	if paymentHash == "" || len(paymentHash) != 64 {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		fmt.Fprintf(w, `{"error":"invalid_payment_hash","message":"payment hash must be 64 hex characters"}`)
		return
	}

	if g.lightning == nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		fmt.Fprintf(w, `{"error":"lightning_unavailable","message":"lightning provider not configured"}`)
		return
	}

	paid, err := g.lightning.CheckPayment(paymentHash)
	if err != nil {
		log.Debug().Err(err).Str("payment_hash", paymentHash).Msg("Error checking payment status")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK) // Return 200 with paid=false on errors
		fmt.Fprintf(w, `{"paid":false,"error":"%s"}`, err.Error())
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"paid":%t}`, paid)
}

// matchRoute finds the route matching the request.
func (g *Gateway) matchRoute(r *http.Request) *config.Route {
	path := r.URL.Path

	for i := range g.config.Routes {
		route := &g.config.Routes[i]

		// Check exact path match
		if route.Match.PathExact != "" {
			if route.Match.PathExact == path {
				return route
			}
			continue
		}

		// Check path prefix match (supports trailing * for wildcard)
		if route.Match.PathPrefix != "" {
			prefix := route.Match.PathPrefix
			if strings.HasSuffix(prefix, "*") {
				prefix = strings.TrimSuffix(prefix, "*")
			}
			if strings.HasPrefix(path, prefix) {
				return route
			}
		}
	}

	return nil
}

// proxyRequest forwards the request to the upstream.
func (g *Gateway) proxyRequest(w http.ResponseWriter, r *http.Request, route *config.Route) {
	// MCP parsing: if the route has MCP enabled, parse the request body
	// to extract tool-level metadata for logging and telemetry.
	// Per-tool cost attribution is available in SatGate Enterprise.
	if route.MCP != nil && route.MCP.Enabled {
		maxBody := route.MCP.MaxBodySize
		if maxBody <= 0 {
			maxBody = mcp.DefaultMaxBodySize
		}
		info := mcp.Parse(r, maxBody)
		if info != nil {
			ctx := mcp.WithInfo(r.Context(), info)
			r = r.WithContext(ctx)
			log.Debug().
				Str("route", route.Name).
				Str("mcp.method", info.Method).
				Str("mcp.tool", info.ToolName).
				Msg("MCP request parsed")
		}
	}

	// Check for demo routes - return mock data instead of proxying
	if demoResponse := g.getDemoResponse(route); demoResponse != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		json.NewEncoder(w).Encode(demoResponse)
		return
	}

	// Get upstream name
	upstreamName := route.Upstream
	if upstreamName == "" {
		http.Error(w, "No upstream configured", http.StatusBadGateway)
		return
	}

	// Get proxy
	g.proxyMu.RLock()
	proxy, exists := g.proxies[upstreamName]
	g.proxyMu.RUnlock()

	if !exists {
		http.Error(w, "Upstream not found", http.StatusBadGateway)
		return
	}

	// Apply path rewrite if configured (static rewrite takes precedence)
	if route.Rewrite != "" {
		r.URL.Path = route.Rewrite
		r.URL.RawPath = route.Rewrite
	} else if route.StripPrefix && route.Match.PathPrefix != "" {
		// Strip the matched prefix from the path before proxying
		prefix := route.Match.PathPrefix
		if strings.HasSuffix(prefix, "*") {
			prefix = strings.TrimSuffix(prefix, "*")
		}
		newPath := strings.TrimPrefix(r.URL.Path, prefix)
		if !strings.HasPrefix(newPath, "/") {
			newPath = "/" + newPath
		}
		r.URL.Path = newPath
		r.URL.RawPath = newPath
	}

	// Forward request
	proxy.ServeHTTP(w, r)
}

// getDemoResponse returns mock demo data for L402 demo routes
func (g *Gateway) getDemoResponse(route *config.Route) map[string]interface{} {
	switch route.Name {
	case "api-micro":
		return map[string]interface{}{
			"tier":    "micro",
			"price":   "1 sat",
			"message": "🏓 Pong! Micro-payment verified.",
			"data": map[string]interface{}{
				"latency_ms": 42,
				"status":     "healthy",
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			},
		}
	case "api-basic":
		return map[string]interface{}{
			"tier":    "basic",
			"price":   "10 sats",
			"message": "📊 Basic quote data unlocked.",
			"data": map[string]interface{}{
				"btc_usd":    105420.50,
				"btc_eur":    97830.25,
				"btc_gbp":    83150.75,
				"change_24h": "+2.4%",
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			},
		}
	case "api-standard":
		return map[string]interface{}{
			"tier":    "standard",
			"price":   "100 sats",
			"message": "📈 Analytics dashboard unlocked.",
			"data": map[string]interface{}{
				"market_cap":       "2.1T USD",
				"volume_24h":       "48.2B USD",
				"dominance":        "52.3%",
				"fear_greed_index": 72,
				"sentiment":        "Greed",
				"top_movers": []map[string]interface{}{
					{"symbol": "BTC", "change": "+2.4%"},
					{"symbol": "ETH", "change": "+3.1%"},
					{"symbol": "SOL", "change": "+5.8%"},
				},
				"timestamp": time.Now().UTC().Format(time.RFC3339),
			},
		}
	case "api-premium":
		return map[string]interface{}{
			"tier":    "premium",
			"price":   "1000 sats",
			"message": "🔮 Premium AI insights unlocked.",
			"data": map[string]interface{}{
				"prediction": map[string]interface{}{
					"btc_7d":     "+8.2%",
					"btc_30d":    "+15.4%",
					"confidence": 0.78,
					"model":      "SatGate-GPT-v2",
				},
				"signals": []map[string]interface{}{
					{"asset": "BTC", "signal": "STRONG_BUY", "score": 0.92},
					{"asset": "ETH", "signal": "BUY", "score": 0.76},
					{"asset": "SOL", "signal": "HOLD", "score": 0.54},
				},
				"whale_activity": map[string]interface{}{
					"large_txs_24h": 847,
					"net_flow":      "+12,450 BTC",
					"exchange_trend": "outflow",
				},
				"risk_score": 0.34,
				"timestamp":  time.Now().UTC().Format(time.RFC3339),
			},
		}
	default:
		return nil // Not a demo route, proxy normally
	}
}

// createProxy creates a reverse proxy for an upstream URL.
func (g *Gateway) createProxy(upstreamURL string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, err
	}

	proxy := httputil.NewSingleHostReverseProxy(target)

	// Configure transport
	proxy.Transport = &http.Transport{
		DialContext: (&net.Dialer{
			Timeout:   30 * time.Second,
			KeepAlive: 30 * time.Second,
		}).DialContext,
		TLSClientConfig: &tls.Config{
			MinVersion: tls.VersionTLS12,
		},
		MaxIdleConns:          100,
		IdleConnTimeout:       90 * time.Second,
		TLSHandshakeTimeout:   10 * time.Second,
		ExpectContinueTimeout: 1 * time.Second,
	}

	// Modify response to strip CORS headers from upstream (we set our own)
	proxy.ModifyResponse = func(resp *http.Response) error {
		resp.Header.Del("Access-Control-Allow-Origin")
		resp.Header.Del("Access-Control-Allow-Methods")
		resp.Header.Del("Access-Control-Allow-Headers")
		resp.Header.Del("Access-Control-Expose-Headers")
		resp.Header.Del("Access-Control-Allow-Credentials")
		resp.Header.Del("Access-Control-Max-Age")
		return nil
	}

	// Error handler
	proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
		log.Error().Err(err).Str("upstream", upstreamURL).Msg("Proxy error")
		http.Error(w, "Bad Gateway", http.StatusBadGateway)
	}

	return proxy, nil
}

// GetMetrics returns current metrics.
func (g *Gateway) GetMetrics() *Metrics {
	return g.metrics
}

// statusWriter wraps ResponseWriter to capture status code.
type statusWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// extractBearerToken extracts token from Authorization header.
func extractBearerToken(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "Bearer ") {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	return auth
}

// extractL402Token extracts L402 token from Authorization header.
// Supports both "L402 macaroon:preimage" and "LSAT macaroon:preimage" formats.
func extractL402Token(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	log.Debug().Str("auth_header", auth).Msg("extractL402Token: checking Authorization header")
	
	if strings.HasPrefix(auth, "L402 ") {
		token := strings.TrimPrefix(auth, "L402 ")
		log.Debug().Str("token_preview", token[:min(30, len(token))]+"...").Msg("extractL402Token: found L402 format")
		return token
	}
	if strings.HasPrefix(auth, "LSAT ") {
		token := strings.TrimPrefix(auth, "LSAT ")
		log.Debug().Str("token_preview", token[:min(30, len(token))]+"...").Msg("extractL402Token: found LSAT format")
		return token
	}
	
	log.Debug().Msg("extractL402Token: no L402/LSAT token found")
	return ""
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// getEnv returns an environment variable value or a default.
func getEnv(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// handleCapabilityMint handles POST /api/capability/mint - Admin endpoint for minting capability tokens.
func (g *Gateway) handleCapabilityMint(w http.ResponseWriter, r *http.Request) {
	// Check admin token - try multiple sources
	adminToken := r.Header.Get("X-Admin-Token")
	
	// Build list of valid tokens from config and environment
	validTokens := []string{}
	if g.config.Admin.Token != "" {
		validTokens = append(validTokens, g.config.Admin.Token)
	}
	// Also check ADMIN_TOKEN environment variable (for demo deployments)
	if envToken := strings.TrimSpace(getEnv("ADMIN_TOKEN", "")); envToken != "" {
		validTokens = append(validTokens, envToken)
	}
	
	// Verify token matches one of the valid tokens
	tokenValid := false
	for _, vt := range validTokens {
		if subtle.ConstantTimeCompare([]byte(adminToken), []byte(vt)) == 1 {
			tokenValid = true
			break
		}
	}
	
	if adminToken == "" || !tokenValid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or missing X-Admin-Token"})
		return
	}

	// Parse request body
	var req struct {
		Scope    string `json:"scope"`
		Duration string `json:"duration"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Default scope and duration
	if req.Scope == "" {
		req.Scope = "api:read"
	}
	if req.Duration == "" {
		req.Duration = "1h"
	}

	// Parse duration
	duration, err := time.ParseDuration(req.Duration)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid duration format"})
		return
	}

	// Mint the token
	expiresAt := time.Now().Add(duration)
	mac, err := g.macaroonSvc.Mint(req.Scope, expiresAt)
	if err != nil {
		log.Error().Err(err).Msg("Failed to mint capability token")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusInternalServerError)
		json.NewEncoder(w).Encode(map[string]string{"error": "Failed to mint token"})
		return
	}

	// Register with governance for dashboard tracking
	if g.governance != nil {
		g.governance.RegisterMintWithLineage(mac.Signature, req.Scope, expiresAt, "", 1, "Agent Token")
	}

	// Encode the token
	token := g.macaroonSvc.Encode(mac)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":     token,
		"scope":     req.Scope,
		"expiresAt": expiresAt.Format(time.RFC3339),
		"signature": mac.Signature, // Return signature for UI display
	})
}

// handleCapabilityValidate handles POST /api/capability/validate - Validate a capability token.
func (g *Gateway) handleCapabilityValidate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		Token string `json:"token"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Verify the token
	mac, err := g.macaroonSvc.Verify(req.Token)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"valid": false,
			"error": err.Error(),
		})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"valid":      true,
		"identifier": mac.Identifier,
		"caveats":    mac.Caveats,
	})
}

// handleCapabilityDelegate handles POST /api/capability/delegate - Delegate a token with additional caveats.
func (g *Gateway) handleCapabilityDelegate(w http.ResponseWriter, r *http.Request) {
	var req struct {
		ParentToken string   `json:"parentToken"`
		Caveats     []string `json:"caveats"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	// Get parent signature for lineage tracking
	parentMac, _ := g.macaroonSvc.Verify(req.ParentToken)
	parentSig := ""
	if parentMac != nil {
		parentSig = parentMac.Signature
	}

	// Delegate the token
	child, err := g.macaroonSvc.Delegate(req.ParentToken, req.Caveats)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
	}

	// Register delegation with governance for dashboard tracking
	if g.governance != nil {
		// Extract scope from caveats
		scope := "api:*"
		expiresAt := time.Now().Add(1 * time.Hour) // Default
		for _, caveat := range child.Caveats {
			if strings.HasPrefix(caveat, "scope = ") {
				scope = strings.TrimPrefix(caveat, "scope = ")
			}
			if strings.HasPrefix(caveat, "expires = ") {
				if ts, err := time.Parse(time.RFC3339, strings.TrimPrefix(caveat, "expires = ")); err == nil {
					expiresAt = ts
				} else if exp, err := time.ParseDuration(strings.TrimPrefix(caveat, "expires = ")); err == nil {
					expiresAt = time.Now().Add(exp)
				}
			}
		}
		g.governance.RegisterDelegation(child.Signature, scope, expiresAt, parentSig)
	}

	// Encode the child token
	token := g.macaroonSvc.Encode(child)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"token":     token,
		"caveats":   child.Caveats,
		"signature": child.Signature, // Return signature for UI display
	})
}

// handleCapabilityPing handles GET /api/capability/ping - validates a capability token and returns success.
func (g *Gateway) handleCapabilityPing(w http.ResponseWriter, r *http.Request) {
	// Extract authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Missing Authorization header",
			"message": "Provide a capability token via 'Authorization: Bearer <token>'",
		})
		return
	}

	// Parse Bearer token
	token := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		token = authHeader
	}

	// Verify the macaroon (includes decode + signature validation)
	mac, err := g.macaroonSvc.Verify(token)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Token verification failed",
			"message": err.Error(),
		})
		return
	}

	// Check if token is banned
	if g.governance != nil && g.governance.IsBanned(mac.Signature) {
		g.governance.RecordBannedHit() // Increment banned hits counter
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "Token Revoked",
			"reason": "This token has been banned by an administrator",
			"code":   "TOKEN_BANNED",
		})
		return
	}

	// Record usage with governance
	if g.governance != nil {
		g.governance.RecordUsage(mac.Signature, "/api/capability/ping")
	}

	// Success - token is valid
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"message":    "Token validated successfully",
		"caveats":    mac.Caveats,
		"identifier": mac.Identifier,
		"validated":  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleCapabilityAdmin handles GET /api/capability/admin - requires admin scope (for scope enforcement demo).
func (g *Gateway) handleCapabilityAdmin(w http.ResponseWriter, r *http.Request) {
	// Extract authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Missing Authorization header",
			"message": "Provide a capability token via 'Authorization: Bearer <token>'",
		})
		return
	}

	// Parse Bearer token
	token := ""
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		token = authHeader
	}

	// Verify the macaroon (includes decode + signature validation)
	mac, err := g.macaroonSvc.Verify(token)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{
			"error":   "Token verification failed",
			"message": err.Error(),
		})
		return
	}

	// Check if token is banned
	if g.governance != nil && g.governance.IsBanned(mac.Signature) {
		g.governance.RecordBannedHit()
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "Token Revoked",
			"reason": "This token has been banned by an administrator",
			"code":   "TOKEN_BANNED",
		})
		return
	}

	// Check scope caveats - ALL scope caveats must allow the required scope
	// (macaroon semantics: adding caveats can only restrict, never expand permissions)
	requiredScope := "api:capability:admin"
	mostRestrictiveScope := "" // Track the most restrictive scope found
	allScopesAllow := true     // Assume allowed until proven otherwise
	
	for _, caveat := range mac.Caveats {
		if strings.HasPrefix(caveat, "scope = ") {
			scopeValue := strings.TrimPrefix(caveat, "scope = ")
			mostRestrictiveScope = scopeValue // Keep track of last (most restrictive) scope
			
			// Check if this specific scope caveat allows the required scope
			scopeAllows := scopeValue == requiredScope || 
			              scopeValue == "api:capability:*" || 
			              scopeValue == "api:*" ||
			              (strings.HasSuffix(scopeValue, ":*") && strings.HasPrefix(requiredScope, strings.TrimSuffix(scopeValue, "*")))
			
			// If ANY scope caveat denies access, the token is denied (macaroon AND semantics)
			if !scopeAllows {
				allScopesAllow = false
				// Don't break - keep looping to find the most restrictive scope for error message
			}
		}
	}

	if !allScopesAllow || mostRestrictiveScope == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusForbidden)
		json.NewEncoder(w).Encode(map[string]interface{}{
			"error":  "Access Denied",
			"reason": fmt.Sprintf("caveat check failed (scope = %s): Scope violation: token has '%s', need '%s'", mostRestrictiveScope, mostRestrictiveScope, requiredScope),
		})
		return
	}

	// Success - token has admin scope
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":     "ok",
		"message":    "Admin access granted",
		"scope":      mostRestrictiveScope,
		"identifier": mac.Identifier,
		"validated":  time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGovernanceBan handles POST /api/governance/ban - Ban a token (demo endpoint).
// In a real implementation, this would add the token to a revocation list.
func (g *Gateway) handleGovernanceBan(w http.ResponseWriter, r *http.Request) {
	// Check admin token
	adminToken := r.Header.Get("X-Admin-Token")
	
	// Build list of valid tokens from config and environment
	validTokens := []string{}
	if g.config.Admin.Token != "" {
		validTokens = append(validTokens, g.config.Admin.Token)
	}
	if envToken := strings.TrimSpace(getEnv("ADMIN_TOKEN", "")); envToken != "" {
		validTokens = append(validTokens, envToken)
	}
	
	// Verify token matches one of the valid tokens
	tokenValid := false
	for _, vt := range validTokens {
		if subtle.ConstantTimeCompare([]byte(adminToken), []byte(vt)) == 1 {
			tokenValid = true
			break
		}
	}
	
	if adminToken == "" || !tokenValid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or missing X-Admin-Token"})
		return
	}

	var req struct {
		TokenSignature string `json:"tokenSignature"`
		Reason         string `json:"reason"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}

	if req.TokenSignature == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "tokenSignature is required"})
		return
	}

	// Add to revocation list via governance service
	if g.governance != nil {
		g.governance.Ban(req.TokenSignature, req.Reason, "admin")
		g.governance.RecordBannedHit() // Increment kill switch counter
	}
	
	log.Info().
		Str("token_signature", req.TokenSignature[:min(16, len(req.TokenSignature))]).
		Str("reason", req.Reason).
		Msg("Token banned")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "banned",
		"tokenSignature": req.TokenSignature,
		"reason":         req.Reason,
		"bannedAt":       time.Now().UTC().Format(time.RFC3339),
	})
}

// handleGovernanceGraph handles GET /api/governance/graph - Returns token lineage for dashboard.
func (g *Gateway) handleGovernanceGraph(w http.ResponseWriter, r *http.Request) {
	// Require admin token
	adminToken := r.Header.Get("X-Admin-Token")

	validTokens := []string{}
	if g.config.Admin.Token != "" {
		validTokens = append(validTokens, g.config.Admin.Token)
	}
	if envToken := strings.TrimSpace(getEnv("ADMIN_TOKEN", "")); envToken != "" {
		validTokens = append(validTokens, envToken)
	}

	tokenValid := false
	for _, vt := range validTokens {
		if subtle.ConstantTimeCompare([]byte(adminToken), []byte(vt)) == 1 {
			tokenValid = true
			break
		}
	}

	if adminToken == "" || !tokenValid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or missing X-Admin-Token"})
		return
	}

	w.Header().Set("Content-Type", "application/json")
	
	if g.governance == nil {
		// Return empty graph if governance not initialized
		json.NewEncoder(w).Encode(map[string]interface{}{
			"nodes": []interface{}{},
			"edges": []interface{}{},
			"stats": map[string]interface{}{
				"active":     0,
				"blocked":    0,
				"banned":     0,
				"bannedHits": 0,
			},
		})
		return
	}
	
	graph := g.governance.GetGraph()
	json.NewEncoder(w).Encode(graph)
}

// handleGovernanceReset handles POST /api/governance/reset - Resets dashboard data.
func (g *Gateway) handleGovernanceReset(w http.ResponseWriter, r *http.Request) {
	// Check admin token
	adminToken := r.Header.Get("X-Admin-Token")
	
	// Build list of valid tokens from config and environment
	validTokens := []string{}
	if g.config.Admin.Token != "" {
		validTokens = append(validTokens, g.config.Admin.Token)
	}
	if envToken := strings.TrimSpace(getEnv("ADMIN_TOKEN", "")); envToken != "" {
		validTokens = append(validTokens, envToken)
	}
	
	// Verify token matches one of the valid tokens
	tokenValid := false
	for _, vt := range validTokens {
		if subtle.ConstantTimeCompare([]byte(adminToken), []byte(vt)) == 1 {
			tokenValid = true
			break
		}
	}
	
	if adminToken == "" || !tokenValid {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or missing X-Admin-Token"})
		return
	}
	
	if g.governance != nil {
		g.governance.Reset()
	}
	
	log.Info().Msg("Dashboard data reset")
	
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "reset",
		"resetAt": time.Now().UTC().Format(time.RFC3339),
	})
}
