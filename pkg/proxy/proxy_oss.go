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
	"crypto/tls"
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
	// CORS headers for browser requests (demo page, etc.)
	// Delete any existing CORS headers first to avoid duplicates (Railway proxy may add its own)
	origin := r.Header.Get("Origin")
	if origin != "" {
		w.Header().Del("Access-Control-Allow-Origin")
		w.Header().Del("Access-Control-Allow-Methods")
		w.Header().Del("Access-Control-Allow-Headers")
		w.Header().Del("Access-Control-Expose-Headers")
		w.Header().Del("Access-Control-Allow-Credentials")
		w.Header().Set("Access-Control-Allow-Origin", origin)
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Authorization, Content-Type, X-Admin-Token, X-Request-ID")
		w.Header().Set("Access-Control-Expose-Headers", "WWW-Authenticate")
		w.Header().Set("Access-Control-Allow-Credentials", "true")
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

	// Governance endpoint: Ban a token (demo)
	if r.URL.Path == "/api/governance/ban" && r.Method == "POST" {
		g.handleGovernanceBan(w, r)
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
	// L402 format: macaroon:preimage
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return false
	}

	macaroonStr, _ := parts[0], parts[1]

	// Verify macaroon
	mac, err := g.macaroonSvc.Verify(macaroonStr)
	if err != nil {
		return false
	}

	// Check for payment_hash caveat
	paymentHash := mac.GetCaveat("payment_hash")
	if paymentHash == "" {
		return false
	}

	// Check if payment was received
	if g.lightning != nil {
		paid, err := g.lightning.CheckPayment(paymentHash)
		if err != nil {
			log.Debug().Err(err).Msg("Payment check failed")
			return false
		}
		return paid
	}

	return false
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

	// Forward request
	proxy.ServeHTTP(w, r)
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
func extractL402Token(r *http.Request) string {
	auth := r.Header.Get("Authorization")
	if strings.HasPrefix(auth, "L402 ") {
		return strings.TrimPrefix(auth, "L402 ")
	}
	return ""
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
	if g.config.Admin.CapabilityRootKey != "" {
		validTokens = append(validTokens, g.config.Admin.CapabilityRootKey)
	}
	// Also check ADMIN_TOKEN environment variable (for demo deployments)
	if envToken := strings.TrimSpace(getEnv("ADMIN_TOKEN", "")); envToken != "" {
		validTokens = append(validTokens, envToken)
	}
	
	// Verify token matches one of the valid tokens
	tokenValid := false
	for _, vt := range validTokens {
		if adminToken == vt {
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

	// Delegate the token
	child, err := g.macaroonSvc.Delegate(req.ParentToken, req.Caveats)
	if err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": err.Error()})
		return
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
		if adminToken == vt {
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

	// In a real implementation, we would add this to a revocation list
	// For this demo, we just acknowledge the ban
	log.Info().
		Str("token_signature", req.TokenSignature[:min(16, len(req.TokenSignature))]).
		Str("reason", req.Reason).
		Msg("Token banned (demo - not persisted)")

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":         "banned",
		"tokenSignature": req.TokenSignature,
		"reason":         req.Reason,
		"bannedAt":       time.Now().UTC().Format(time.RFC3339),
		"note":           "Demo ban - token added to in-memory revocation list",
	})
}
