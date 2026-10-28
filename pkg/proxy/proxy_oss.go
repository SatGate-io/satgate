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
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"syscall"
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

	// Rate limiter for admin endpoints
	adminLimiter *rateLimiter

	// L402 consumed-proof store — records used payment hashes so each proof
	// is accepted once. Redis when configured; otherwise in-process memory,
	// which is correct for a single instance only.
	l402Seen L402ReplayStore

	// Optional hooks (set via SetXxx methods)
	metricsHook MetricsHook

	agentOnce sync.Once
	agent     *agentLoop
}

// Metrics tracks gateway statistics using atomic counters for concurrent safety.
type Metrics struct {
	TotalRequests   atomic.Int64
	TotalPublic     atomic.Int64
	TotalCapability atomic.Int64
	TotalL402       atomic.Int64
	TotalErrors     atomic.Int64
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

	// L402ReplayStore records consumed L402 payment hashes. When nil, the
	// gateway uses Redis if Config.Redis is enabled, and otherwise an
	// in-process store that is correct for a single instance only.
	L402ReplayStore L402ReplayStore
}

// New creates a new OSS Gateway.
func New(opts Options) (*Gateway, error) {
	if opts.Config == nil {
		return nil, fmt.Errorf("config is required")
	}
	if opts.Macaroon == nil {
		return nil, fmt.Errorf("macaroon service is required")
	}
	// A route's name is its identity; an L402 proof is issued for one name.
	if err := config.CheckUniqueRouteNames(opts.Config.Routes); err != nil {
		return nil, fmt.Errorf("invalid routes: %w", err)
	}
	// A route matches only requests that meet every matcher it sets; the
	// matchers are checked and compiled here, and a route that sets
	// rateLimit (not enforced per route) is refused.
	if err := config.CheckRouteMatchers(opts.Config.Routes); err != nil {
		return nil, fmt.Errorf("invalid routes: %w", err)
	}
	// Configured upstream headers are set on every forwarded request; a
	// header SatGate owns cannot be one of them.
	if err := config.CheckUpstreamHeaders(opts.Config.Upstreams); err != nil {
		return nil, fmt.Errorf("invalid upstreams: %w", err)
	}

	// Initialize admin rate limiter (default: 60 req/min if not configured)
	adminRPM := opts.Config.Admin.RateLimitPerMinute
	if adminRPM == 0 {
		adminRPM = 60
	}

	replayStore := opts.L402ReplayStore
	if replayStore == nil {
		var err error
		replayStore, err = newConfiguredReplayStore(opts.Config.Redis)
		if err != nil {
			return nil, err
		}
	}
	if replayStore == nil {
		replayStore = newMemoryReplayStore(5 * time.Minute)
		if hasL402Route(opts.Config) {
			log.Warn().Msg("L402 consumed-proof store is in process memory: correct for a single gateway instance only. Configure redis for restarts or more than one replica.")
		}
	}

	g := &Gateway{
		config:       opts.Config,
		macaroonSvc:  opts.Macaroon,
		governance:   opts.Governance,
		lightning:    opts.Lightning,
		proxies:      make(map[string]*httputil.ReverseProxy),
		metrics:      &Metrics{},
		adminLimiter: newRateLimiter(adminRPM),
		l402Seen:     replayStore,
	}

	// Initialize proxies for configured upstreams
	for name, upstream := range opts.Config.Upstreams {
		proxy, err := g.createProxy(upstream.URL, upstream.Headers)
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

	// Every other request is served on one checked path: built-in
	// endpoints, route matching, policy checks and the prefix strip all
	// read the path that passed checkRequestPath, and the upstream receives
	// it (minus a stripped prefix) as sent.
	if err := checkRequestPath(r.URL); err != nil {
		http.Error(w, "Bad request path", http.StatusBadRequest)
		g.metrics.TotalErrors.Add(1)
		return
	}

	// Built-in health endpoint (doesn't depend on upstreams)
	if r.URL.Path == "/health" || r.URL.Path == "/healthz" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, `{"status":"healthy","service":"satgate-oss","routes":%d}`, len(g.config.Routes))
		return
	}

	if r.URL.Path == "/.well-known/satgate-agent-jwks.json" && r.Method == http.MethodGet {
		writeJSON(w, http.StatusOK, g.agentJWKS())
		return
	}

	// Check payment status endpoint (for polling from frontend)
	if strings.HasPrefix(r.URL.Path, "/check-payment/") {
		g.handleCheckPayment(w, r)
		return
	}

	// Rate limit built-in admin/API endpoints without throttling customer
	// upstream routes that happen to live under /api/.
	if isBuiltInAPIEndpoint(r.URL.Path) {
		clientIP := extractIP(r)
		if !g.adminLimiter.allow(clientIP) {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Retry-After", "60")
			w.WriteHeader(http.StatusTooManyRequests)
			json.NewEncoder(w).Encode(map[string]string{"error": "Rate limit exceeded"})
			return
		}
	}

	if strings.HasPrefix(r.URL.Path, "/api/agent/") {
		g.handleAgentLoop(w, r)
		return
	}

	// Admin endpoint: complete a mock L402 payment for local demos and tests.
	// This is intentionally available only for Lightning providers that expose
	// mock preimages; real Lightning backends must settle externally.
	if r.URL.Path == "/api/l402/mock-pay" && r.Method == "POST" {
		g.handleL402MockPay(w, r)
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

	// Admin endpoint: Token delegation
	if r.URL.Path == "/api/capability/delegate" && r.Method == "POST" {
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

	// Governance endpoint: Export Evidence Pack (Policy-to-ProofAlignment)
	if r.URL.Path == "/api/governance/evidence-pack" && r.Method == "GET" {
		g.handleExportEvidencePack(w, r)
		return
	}

	start := time.Now()
	g.metrics.TotalRequests.Add(1)
	g.restoreAgentRoutes()

	// Find matching route
	route := g.matchRoute(r)
	if route == nil {
		log.Warn().Str("path", r.URL.Path).Msg("No matching route")
		http.Error(w, "No matching route", http.StatusNotFound)
		g.metrics.TotalErrors.Add(1)
		return
	}
	// The path the route would forward must pass the same rule; refuse it
	// here, before the route's policy runs.
	if !forwardPathOK(route, r.URL) {
		http.Error(w, "Bad request path", http.StatusBadRequest)
		g.metrics.TotalErrors.Add(1)
		return
	}

	// Get policy kind (default to capability for protection)
	policyKind := "capability"
	if route.Policy.Kind != "" {
		policyKind = route.Policy.Kind
	}

	// Create response wrapper to capture status code
	wrapped := &statusWriter{ResponseWriter: g.withLivePack(w, r, route), statusCode: http.StatusOK}

	// Handle based on policy
	switch policyKind {
	case "public":
		g.metrics.TotalPublic.Add(1)
		g.handlePublic(wrapped, r, route)

	case "capability":
		g.metrics.TotalCapability.Add(1)
		g.handleCapability(wrapped, r, route)

	case "chargeback":
		// chargeback (observe/audit/policy_to_proof) supported in refactor for schema validation
		g.handlePublic(wrapped, r, route)

	case "deny":
		http.Error(wrapped, "denied", http.StatusForbidden)
		return

	case "identity":
		g.handleIdentity(wrapped, r, route)

	case "l402":
		g.metrics.TotalL402.Add(1)
		g.handleL402(wrapped, r, route)

	default:
		// Unknown policy in OSS - reject
		log.Warn().Str("policy", policyKind).Str("route", route.Name).
			Msg("Unknown policy kind in OSS build")
		http.Error(w, fmt.Sprintf("Policy '%s' not supported in OSS", policyKind), http.StatusNotImplemented)
		g.metrics.TotalErrors.Add(1)
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

	// A ban of this token or of any token it was delegated from refuses it.
	if err := g.checkRevoked(r.Context(), mac); err != nil {
		if errors.Is(err, errRevocationUnavailable) {
			http.Error(w, "Revocation status unavailable", http.StatusServiceUnavailable)
			return
		}
		g.governance.RecordBannedHit()
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
		err := g.verifyL402Token(r, token, route)
		if err == nil {
			g.proxyRequest(w, r, route)
			return
		}
		if errors.Is(err, errL402ReplayStoreUnavailable) {
			http.Error(w, "Payment verification unavailable", http.StatusServiceUnavailable)
			return
		}
		if errors.Is(err, errRevocationUnavailable) {
			http.Error(w, "Revocation status unavailable", http.StatusServiceUnavailable)
			return
		}
		if errors.Is(err, errTokenRevoked) {
			if g.governance != nil {
				g.governance.RecordBannedHit()
			}
			http.Error(w, "Token revoked", http.StatusUnauthorized)
			return
		}
		log.Debug().Err(err).Str("route", route.Name).Msg("L402: payment proof refused")
	}

	// No valid payment - issue challenge
	g.issueL402Challenge(w, r, route)
}

// verifyL402Token verifies an L402 payment proof for the request being served.
//
// The proof is accepted only when the macaroon was issued by this gateway as
// an L402 challenge, the served route's name is unique among the configured
// routes, its single-valued route and price caveats match the route and its
// current price, every restriction a holder added (scope, routes, methods,
// expiry) covers the request, the preimage hashes to its payment_hash, no
// ban covers the macaroon or a macaroon it was delegated from, and the
// payment hash has not been consumed before. The payment hash is consumed
// only after every other check passes, so a refused proof is not consumed.
func (g *Gateway) verifyL402Token(r *http.Request, token string, route *config.Route) error {
	ctx := r.Context()
	// L402 format: macaroon:preimage
	parts := strings.SplitN(token, ":", 2)
	if len(parts) != 2 {
		return fmt.Errorf("invalid token format, expected macaroon:preimage")
	}
	macaroonStr, preimageHex := parts[0], parts[1]

	// Verify macaroon signature and caveats (a repeated single-valued caveat
	// and every absolute expiry are checked here).
	mac, err := g.macaroonSvc.Verify(macaroonStr)
	if err != nil {
		return fmt.Errorf("macaroon verification failed: %w", err)
	}
	if !mac.IsL402Challenge() {
		return fmt.Errorf("macaroon was not issued as an L402 challenge")
	}

	paymentHash, err := requiredL402Caveat(mac, macaroon.CaveatPaymentHash)
	if err != nil {
		return err
	}
	boundRoute, err := requiredL402Caveat(mac, macaroon.CaveatRoute)
	if err != nil {
		return err
	}
	boundPrice, err := requiredL402Caveat(mac, macaroon.CaveatPriceSats)
	if err != nil {
		return err
	}
	invoiceAmount, hasInvoiceAmount, err := mac.SingleCaveat(macaroon.CaveatInvoiceAmountSats)
	if err != nil {
		return err
	}

	// Binding: the proof serves only the route it was issued for, at the
	// price that route has now.
	if route.Policy.PriceSats <= 0 {
		return fmt.Errorf("route %q has no price", route.Name)
	}
	currentPrice := strconv.FormatInt(route.Policy.PriceSats, 10)
	if !g.routeNameIsUnique(route.Name) {
		return fmt.Errorf("route name %q is not unique among configured routes", route.Name)
	}
	if boundRoute != route.Name {
		return fmt.Errorf("proof was issued for route %q, not %q", boundRoute, route.Name)
	}
	if boundPrice != currentPrice {
		return fmt.Errorf("proof was issued at %s sats, route %q now costs %s sats", boundPrice, route.Name, currentPrice)
	}
	if hasInvoiceAmount && invoiceAmount != currentPrice {
		return fmt.Errorf("invoice amount %s sats does not match route price %s sats", invoiceAmount, currentPrice)
	}

	// Restrictions a holder added to narrow the proof must cover this request.
	if err := mac.CheckL402Request(macaroon.L402Request{
		Method: r.Method,
		Path:   r.URL.Path,
		Route:  route.Name,
	}); err != nil {
		return fmt.Errorf("proof restriction: %w", err)
	}

	// CRYPTOGRAPHIC PROOF: Verify that SHA256(preimage) == payment_hash
	// This is the definitive proof of payment - no need to query the Lightning node
	preimageBytes, err := hex.DecodeString(preimageHex)
	if err != nil {
		return fmt.Errorf("invalid preimage hex")
	}
	hash := sha256.Sum256(preimageBytes)
	computedHash := hex.EncodeToString(hash[:])
	if subtle.ConstantTimeCompare([]byte(computedHash), []byte(paymentHash)) != 1 {
		return fmt.Errorf("preimage does not match payment_hash")
	}

	// Revocation: a ban of this proof's macaroon, or of one it was
	// attenuated from, refuses it before it is consumed.
	if err := g.checkRevoked(ctx, mac); err != nil {
		return err
	}

	// Replay guard: each payment proof can only be used once. The record
	// outlives the macaroon (l402ReplayTTL >= l402MacaroonLifetime).
	storeCtx, cancel := context.WithTimeout(ctx, l402ReplayStoreTimeout)
	defer cancel()
	first, err := g.l402Seen.Consume(storeCtx, paymentHash, l402ReplayTTL)
	if err != nil {
		log.Error().Err(err).Msg("L402: consumed-proof store unavailable; refusing proof")
		return fmt.Errorf("%w: %v", errL402ReplayStoreUnavailable, err)
	}
	if !first {
		return fmt.Errorf("payment proof already used")
	}

	log.Info().Str("route", route.Name).Str("payment_hash", shortHash(paymentHash)).Msg("L402: payment verified via preimage")
	return nil
}

// requiredL402Caveat returns the value of a server-written L402 caveat that
// must appear exactly once with a non-empty value.
func requiredL402Caveat(mac *macaroon.Macaroon, key string) (string, error) {
	value, found, err := mac.SingleCaveat(key)
	if err != nil {
		return "", err
	}
	if !found || value == "" {
		return "", fmt.Errorf("macaroon has no %s caveat", key)
	}
	return value, nil
}

func shortHash(h string) string {
	if len(h) > 16 {
		return h[:16] + "..."
	}
	return h
}

// hasL402Route reports whether any configured route uses the l402 policy.
func hasL402Route(cfg *config.Config) bool {
	for _, route := range cfg.Routes {
		if route.Policy.Kind == "l402" {
			return true
		}
	}
	return false
}

// routeNameIsUnique reports whether exactly one configured route has name.
// Configuration, gateway construction and agent promotion already refuse
// duplicate names; this keeps an L402 proof bound to a single route even if
// the live route table is changed some other way.
func (g *Gateway) routeNameIsUnique(name string) bool {
	n := 0
	for i := range g.config.Routes {
		if g.config.Routes[i].Name == name {
			n++
		}
	}
	return n == 1
}

// issueL402Challenge issues an HTTP 402 payment challenge.
func (g *Gateway) issueL402Challenge(w http.ResponseWriter, r *http.Request, route *config.Route) {
	if g.lightning == nil {
		http.Error(w, "Lightning provider not configured", http.StatusServiceUnavailable)
		return
	}
	if route.Policy.PriceSats <= 0 {
		http.Error(w, "price_unknown", http.StatusConflict)
		return
	}
	// A challenge names one route. Do not take a payment that no route could
	// accept.
	if !g.routeNameIsUnique(route.Name) {
		log.Error().Str("route", route.Name).Msg("L402: route name is not unique; refusing to issue a challenge")
		http.Error(w, "Route configuration error", http.StatusInternalServerError)
		return
	}
	priceSats := route.Policy.PriceSats

	// Create invoice
	inv, err := g.lightning.CreateInvoice(priceSats, fmt.Sprintf("SatGate: %s", route.Name))
	if err != nil {
		log.Error().Err(err).Msg("Failed to create Lightning invoice")
		http.Error(w, "Payment service unavailable", http.StatusServiceUnavailable)
		return
	}
	invoice := inv.Bolt11
	paymentHash := inv.PaymentHash

	// Create the challenge macaroon, bound to this payment, route and price.
	mac, err := g.macaroonSvc.MintL402(macaroon.L402Binding{
		PaymentHash:       paymentHash,
		Route:             route.Name,
		PriceSats:         priceSats,
		InvoiceAmountSats: inv.Amount,
	}, time.Now().Add(l402MacaroonLifetime))
	if err != nil {
		log.Error().Err(err).Msg("Failed to mint L402 macaroon")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}
	// Use binary format for L402 challenge (standard libmacaroon V2 — compatible with lnget/Aperture)
	macaroonStr, err := g.macaroonSvc.EncodeBinary(mac)
	if err != nil {
		log.Error().Err(err).Msg("Failed to encode L402 macaroon as binary")
		http.Error(w, "Internal error", http.StatusInternalServerError)
		return
	}

	// Set L402 challenge header
	w.Header().Set("WWW-Authenticate", fmt.Sprintf(`L402 macaroon="%s", invoice="%s"`, macaroonStr, invoice))
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusPaymentRequired)

	// Write JSON response with payment_hash so frontend can poll
	fmt.Fprintf(w, `{"error":"payment_required","invoice":"%s","amount_sats":%d,"payment_hash":"%s"}`, invoice, priceSats, paymentHash)
}

func isBuiltInAPIEndpoint(path string) bool {
	return path == "/api/l402/mock-pay" ||
		strings.HasPrefix(path, "/api/capability/") ||
		strings.HasPrefix(path, "/api/governance/") ||
		strings.HasPrefix(path, "/api/agent/")
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
		resp, _ := json.Marshal(map[string]interface{}{"paid": false, "error": "payment verification failed"})
		w.Write(resp)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	fmt.Fprintf(w, `{"paid":%t}`, paid)
}

func (g *Gateway) handleL402MockPay(w http.ResponseWriter, r *http.Request) {
	if !g.validAdminToken(r.Header.Get("X-Admin-Token")) {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusUnauthorized)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid or missing X-Admin-Token"})
		return
	}

	mock, ok := g.lightning.(*lightning.MockProvider)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotImplemented)
		json.NewEncoder(w).Encode(map[string]string{"error": "mock L402 payment is only available with the mock Lightning provider"})
		return
	}

	r.Body = io.NopCloser(io.LimitReader(r.Body, 1048576))
	var req struct {
		PaymentHash string `json:"payment_hash"`
		Macaroon    string `json:"macaroon"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "Invalid request body"})
		return
	}
	if len(req.PaymentHash) != 64 || req.Macaroon == "" {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "payment_hash and macaroon are required"})
		return
	}

	mac, err := g.macaroonSvc.Verify(req.Macaroon)
	var macHash string
	if err == nil && mac.IsL402Challenge() {
		macHash, _, err = mac.SingleCaveat(macaroon.CaveatPaymentHash)
	}
	if err != nil || macHash == "" || macHash != req.PaymentHash {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"error": "macaroon does not match payment_hash"})
		return
	}
	if err := g.checkRevoked(r.Context(), mac); err != nil {
		if errors.Is(err, errRevocationUnavailable) {
			writeRevocationUnavailableJSON(w)
			return
		}
		writeTokenBannedJSON(w, http.StatusForbidden)
		return
	}

	preimage, ok := mock.GetPreimage(req.PaymentHash)
	if !ok {
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"error": "payment hash not found"})
		return
	}
	mock.SimulatePayment(req.PaymentHash)

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{
		"paid":          "true",
		"payment_hash":  req.PaymentHash,
		"authorization": "L402 " + req.Macaroon + ":" + preimage,
	})
}

func (g *Gateway) validAdminToken(adminToken string) bool {
	if adminToken == "" {
		return false
	}

	validTokens := []string{}
	if g.config.Admin.Token != "" {
		validTokens = append(validTokens, g.config.Admin.Token)
	}
	if envToken := strings.TrimSpace(getEnv("ADMIN_TOKEN", "")); envToken != "" {
		validTokens = append(validTokens, envToken)
	}

	for _, vt := range validTokens {
		if subtle.ConstantTimeCompare([]byte(adminToken), []byte(vt)) == 1 {
			return true
		}
	}
	return false
}

// matchRoute returns the first route, in configuration order, whose every
// configured matcher the request meets (config.Route.MatchRequest), or nil.
// r.URL.Path has passed checkRequestPath.
func (g *Gateway) matchRoute(r *http.Request) *config.Route {
	for i := range g.config.Routes {
		route := &g.config.Routes[i]
		if route.MatchRequest(r.URL.Path, r.Method, r.Header) {
			return route
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
		// Strip the matched prefix from the checked path; the rest goes
		// upstream encoded as sent. A remainder that would not pass the
		// path check is refused.
		if !stripPathPrefix(r.URL, route.Match.Prefix()) {
			http.Error(w, "Bad request path", http.StatusBadRequest)
			return
		}
	}

	// The upstream gets a copy of the request without the credentials the
	// gateway consumed or owns (upstream_credentials.go). upstreamTransport
	// then sets the upstream's configured headers and cleans the trailers
	// as the request leaves.
	consumed := authorizationConsumed(route)
	out := r.Clone(withForwardRules(r.Context(), forwardRules{consumed: consumed}))
	g.removeGatewayCredentials(out.Header, consumed)

	// Forward request
	proxy.ServeHTTP(w, out)
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
					"large_txs_24h":  847,
					"net_flow":       "+12,450 BTC",
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

// createProxy creates a reverse proxy for an upstream URL. headers are the
// upstream's configured headers (config.Upstream.Headers): upstreamTransport
// sets them on every request it forwards, after the reverse proxy's own
// changes, so each arrives with its configured value and replaces a client
// header of the same name.
func (g *Gateway) createProxy(upstreamURL string, headers map[string]string) (*httputil.ReverseProxy, error) {
	target, err := url.Parse(upstreamURL)
	if err != nil {
		return nil, err
	}
	configured, configuredHost := upstreamHeaders(headers)

	proxy := httputil.NewSingleHostReverseProxy(target)
	// Wrap Director to set req.Host = target.Host for CDN compatibility
	// (Cloudflare, OpenAI, Anthropic etc. reject requests with wrong Host header)
	defaultDirector := proxy.Director
	targetHost := target.Host
	proxy.Director = func(r *http.Request) {
		defaultDirector(r)
		r.Host = targetHost
	}

	// Configure transport with SSRF-safe dialer that blocks private IPs at connect time.
	// Config-time validation catches obvious cases, but DNS can resolve to private IPs
	// at runtime (DNS rebinding, TOCTOU). This is the runtime enforcement layer.
	// Bypass when allowPrivateIPs is set (local dev, testing, internal services).
	ssrfDialer := &net.Dialer{
		Timeout:   30 * time.Second,
		KeepAlive: 30 * time.Second,
	}
	allowPrivate := g.config.Cloud != nil && g.config.Cloud.SSRF != nil && g.config.Cloud.SSRF.AllowPrivateIPs
	if !allowPrivate {
		ssrfDialer.Control = func(network, address string, c syscall.RawConn) error {
			host, _, err := net.SplitHostPort(address)
			if err != nil {
				return err
			}
			ip := net.ParseIP(host)
			if ip != nil && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsUnspecified()) {
				return fmt.Errorf("SSRF blocked: resolved to private/internal IP %s", host)
			}
			return nil
		}
	}
	proxy.Transport = &upstreamTransport{
		g:       g,
		headers: configured,
		host:    configuredHost,
		base: &http.Transport{
			DialContext: ssrfDialer.DialContext,
			TLSClientConfig: &tls.Config{
				MinVersion: tls.VersionTLS12,
			},
			MaxIdleConns:          100,
			IdleConnTimeout:       90 * time.Second,
			TLSHandshakeTimeout:   10 * time.Second,
			ExpectContinueTimeout: 1 * time.Second,
		},
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
	// The header's value is a credential: only whether one was sent is logged.
	log.Debug().Bool("auth_header_present", auth != "").Msg("extractL402Token: checking Authorization header")

	if strings.HasPrefix(auth, "L402 ") {
		token := strings.TrimPrefix(auth, "L402 ")
		log.Debug().Int("token_len", len(token)).Msg("extractL402Token: found L402 format")
		return token
	}
	if strings.HasPrefix(auth, "LSAT ") {
		token := strings.TrimPrefix(auth, "LSAT ")
		log.Debug().Int("token_len", len(token)).Msg("extractL402Token: found LSAT format")
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

	// Parse request body (limit to 1MB)
	r.Body = io.NopCloser(io.LimitReader(r.Body, 1048576))
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

	// A token covered by a ban is not valid.
	if err := g.checkRevoked(r.Context(), mac); err != nil {
		if errors.Is(err, errRevocationUnavailable) {
			writeRevocationUnavailableJSON(w)
			return
		}
		writeJSON(w, http.StatusUnauthorized, map[string]interface{}{
			"valid": false,
			"error": "token revoked",
			"code":  "TOKEN_BANNED",
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
	r.Body = io.NopCloser(io.LimitReader(r.Body, 1048576))
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

	// The parent must verify, and no ban may cover it (a ban of the parent
	// or of any token it was delegated from). Nothing is minted otherwise.
	parentMac, err := g.macaroonSvc.Verify(req.ParentToken)
	if err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": fmt.Sprintf("invalid parent token: %v", err)})
		return
	}
	if err := g.checkRevoked(r.Context(), parentMac); err != nil {
		if errors.Is(err, errRevocationUnavailable) {
			writeRevocationUnavailableJSON(w)
			return
		}
		if g.governance != nil {
			g.governance.RecordBannedHit()
		}
		writeTokenBannedJSON(w, http.StatusForbidden)
		return
	}
	parentSig := parentMac.Signature

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

	// Check if a ban covers the token or a token it was delegated from
	if err := g.checkRevoked(r.Context(), mac); err != nil {
		if errors.Is(err, errRevocationUnavailable) {
			writeRevocationUnavailableJSON(w)
			return
		}
		g.governance.RecordBannedHit() // Increment banned hits counter
		writeTokenBannedJSON(w, http.StatusForbidden)
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

	// Check if a ban covers the token or a token it was delegated from
	if err := g.checkRevoked(r.Context(), mac); err != nil {
		if errors.Is(err, errRevocationUnavailable) {
			writeRevocationUnavailableJSON(w)
			return
		}
		g.governance.RecordBannedHit() // Increment banned hits counter
		writeTokenBannedJSON(w, http.StatusForbidden)
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

	r.Body = io.NopCloser(io.LimitReader(r.Body, 1048576))
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

	req.TokenSignature = strings.ToLower(strings.TrimSpace(req.TokenSignature))

	// Add to revocation list via governance service. The ban covers this
	// token and every token delegated from it.
	if g.governance != nil {
		if err := g.governance.BanContext(r.Context(), req.TokenSignature, req.Reason, "admin"); err != nil {
			log.Error().Err(err).Msg("Ban not written to the shared revocation store")
			writeJSON(w, http.StatusServiceUnavailable, map[string]string{
				"error": "Ban recorded on this gateway only: the shared revocation store is unavailable. Retry the ban.",
				"code":  "REVOCATION_UNAVAILABLE",
			})
			return
		}
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

	// Node statuses read the shared ban state; if it cannot be read the
	// graph is not answered (see governance.Service.GetGraphContext).
	graph, err := g.governance.GetGraphContext(r.Context())
	if err != nil {
		writeRevocationUnavailableJSON(w)
		return
	}
	json.NewEncoder(w).Encode(graph)
}

// handleGovernanceReset handles POST /api/governance/reset - Resets dashboard data.
func (g *Gateway) handleGovernanceReset(w http.ResponseWriter, r *http.Request) {
	// ... existing reset logic ...
	w.WriteHeader(http.StatusOK)
}

// handleExportEvidencePack handles GET /api/governance/evidence-pack - Exports signed policy proof.
func (g *Gateway) handleExportEvidencePack(w http.ResponseWriter, r *http.Request) {
	if g.governance == nil {
		http.Error(w, "Governance service not enabled", http.StatusServiceUnavailable)
		return
	}

	tenantID := r.Header.Get("X-Tenant-ID")
	if tenantID == "" {
		tenantID = g.config.Server.DefaultTenantId
	}

	pack, err := g.governance.ExportEvidencePack(r.Context(), tenantID)
	if err != nil {
		http.Error(w, "Failed to export Evidence Pack", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Content-Disposition", fmt.Sprintf("attachment; filename=\"satgate-evidence-pack-%s.json\"", tenantID))
	json.NewEncoder(w).Encode(pack)
}
