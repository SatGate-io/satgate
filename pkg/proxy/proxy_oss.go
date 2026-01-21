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
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
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
	// Built-in health endpoint (doesn't depend on upstreams)
	if r.URL.Path == "/health" || r.URL.Path == "/healthz" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"satgate-oss","routes":%d}`, len(g.config.Routes))
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

	// Write JSON response
	fmt.Fprintf(w, `{"error":"payment_required","invoice":"%s","amount_sats":%d}`, invoice, priceSats)
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
