package proxy

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"net"
	"net/http"
	"net/http/httputil"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/internal/cloud"
	"github.com/satgate-io/satgate/internal/config"
	"github.com/satgate-io/satgate/internal/governance"
	"github.com/satgate-io/satgate/internal/l402"
	"github.com/satgate-io/satgate/internal/macaroon"
	"github.com/satgate-io/satgate/internal/tenant"
)

// statusResponseWriter wraps http.ResponseWriter to capture the status code.
type statusResponseWriter struct {
	http.ResponseWriter
	statusCode int
}

func (w *statusResponseWriter) WriteHeader(code int) {
	w.statusCode = code
	w.ResponseWriter.WriteHeader(code)
}

// Gateway is the main L402 reverse proxy
type Gateway struct {
	config             *config.Config
	macaroonSvc        *macaroon.Service
	l402Svc            *l402.Service
	governance         *governance.Service
	proxies            map[string]*httputil.ReverseProxy
	dynamicProxies     *DynamicProxyCache // tenant-config dynamic upstreams with TTL eviction
	metrics            *Metrics
	tenantEnforcer     *tenant.IsolationEnforcer
	trustedProxyCIDRs  []*net.IPNet // CIDRs allowed to set X-Tenant-ID
	routeRateLimiter   *RouteRateLimiter  // Per-route rate limiting
	billingManager     BillingManager     // Enterprise billing (optional)
	policyChecker      PolicyChecker      // Plan-based policy gating (optional)
	haCoordinator      HACoordinator      // HA metrics recorder (optional)
	budgetEnforcer     BudgetEnforcer     // Delegation budget enforcement (optional)
	delegationVerifier DelegationVerifier // Delegation token verification (optional)
	settlementService  SettlementService  // L402 → Credits settlement (optional)
}

// HACoordinator interface for recording metrics to the HA coordinator.
type HACoordinator interface {
	RecordRequest(latency time.Duration, isError bool)
	IncrementConnections()
	DecrementConnections()
}

// BillingManager interface for enterprise billing (decoupled from billing package)
type BillingManager interface {
	Authorize(ctx context.Context, r *http.Request, policy *BillingPolicy) (BillingDecision, error)
	RecordUsage(ctx context.Context, ev BillingUsageEvent, policy *BillingPolicy) error
}

// PolicyChecker interface for plan-based policy gating (decoupled from cloud package)
// This enables the "Economic Firewall" tier model where:
// - Observe (chargeback) is unlimited on all plans
// - Control (fiat402) and Charge (l402) require Pro+
type PolicyChecker interface {
	// CheckPolicyAllowed returns (allowed, reason) for a given tenant and policy kind
	CheckPolicyAllowed(ctx context.Context, tenantID string, policyKind string) (bool, string)
}

// BudgetEnforcer interface for delegated token budget enforcement (economic firewall)
// This enforces hard budget limits at the gateway - "charge on attempt" before upstream
type BudgetEnforcer interface {
	// EnforceFunc checks if the request is within budget and atomically decrements
	// Returns (result, nil) if allowed, (result, error) if budget exhausted
	// tokenID: stable token ID (delegation token ID, NOT macaroon signature)
	// tenantID: tenant UUID for ledger attribution
	// requestID: stable idempotency key (from X-Request-ID or Idempotency-Key)
	// costCredits: explicit cost for this request (0 = use default from cost lookup)
	EnforceFunc(ctx context.Context, tokenID, tenantID, route, method, policyMode, requestID string, costCredits int64) (*BudgetResult, error)
}

// BudgetResult contains the result of a budget enforcement check
type BudgetResult struct {
	CostCredits      int64
	RemainingCredits int64
}

// DelegationVerifier interface for verifying delegation tokens (decoupled from delegation package)
// This enables budget-enforced tokens with proper token ID extraction
type DelegationVerifier interface {
	// VerifySecret validates a delegation token secret and returns token details
	// Returns (tokenID, tenantID, scopes, nil) if valid
	// Returns ("", "", nil, error) if invalid/expired/revoked
	VerifySecret(ctx context.Context, secret string) (tokenID, tenantID string, scopes []string, err error)
}

// SettlementService interface for L402 → Credits integration (Phase 3)
// When an L402 payment is verified, this can be called to credit the delegation token
type SettlementService interface {
	// SettleL402 records an L402 payment and adds credits to a token
	// satsAmount: amount paid in sats
	// creditsPerSat: conversion rate (e.g., 10 credits per sat)
	// Returns: credits added, new budget, error
	SettleL402(ctx context.Context, tenantID, tokenID string, satsAmount int64, creditsPerSat int64, preimageHash string) (creditsAdded, newBudget int64, err error)
}

// BillingPolicy represents billing config for a route
type BillingPolicy struct {
	Mode             string
	Price            float64
	Unit             string
	Scope            string
	CostCenterHeader string
	EnforceBudget    bool
	RouteName        string // Set at runtime for invoice/usage tracking
}

// BillingDecision represents the result of billing authorization
type BillingDecision struct {
	Allowed         bool
	StatusCode      int
	WWWAuthenticate string
	Body            []byte
	Headers         map[string]string
	CostRecorded    float64
}

// BillingUsageEvent represents a billable usage event
type BillingUsageEvent struct {
	Time       time.Time
	Duration   time.Duration
	TenantID   string
	CostCenter string
	RouteName  string
	Method     string
	Path       string
	StatusCode int
	BytesIn    int64
	BytesOut   int64
	Unit       string
	Amount     float64
}

// Metrics tracks gateway statistics
type Metrics struct {
	TotalRequests   int64
	TotalL402       int64
	TotalCapability int64
	TotalDenied     int64
	TotalErrors     int64
}

// buildTLSConfig creates a TLS configuration for upstream connections
func buildTLSConfig(tlsCfg *config.UpstreamTLS, hostname string) (*tls.Config, error) {
	tlsConfig := &tls.Config{
		MinVersion: tls.VersionTLS12,
	}

	// Set server name for SNI
	if tlsCfg.ServerName != "" {
		tlsConfig.ServerName = tlsCfg.ServerName
	} else {
		tlsConfig.ServerName = hostname
	}

	// Skip verification (not recommended for production)
	if tlsCfg.InsecureSkipVerify {
		log.Warn().Str("hostname", hostname).Msg("Upstream TLS: InsecureSkipVerify enabled (NOT recommended)")
		tlsConfig.InsecureSkipVerify = true
	}

	// Load custom CA certificate
	if tlsCfg.CACertFile != "" {
		caCert, err := os.ReadFile(tlsCfg.CACertFile)
		if err != nil {
			return nil, err
		}
		caCertPool := x509.NewCertPool()
		if !caCertPool.AppendCertsFromPEM(caCert) {
			return nil, err
		}
		tlsConfig.RootCAs = caCertPool
		log.Debug().Str("file", tlsCfg.CACertFile).Msg("Loaded custom CA certificate")
	}

	// Load client certificate for mTLS
	if tlsCfg.ClientCertFile != "" && tlsCfg.ClientKeyFile != "" {
		cert, err := tls.LoadX509KeyPair(tlsCfg.ClientCertFile, tlsCfg.ClientKeyFile)
		if err != nil {
			return nil, err
		}
		tlsConfig.Certificates = []tls.Certificate{cert}
		log.Debug().Str("cert", tlsCfg.ClientCertFile).Msg("Loaded client certificate for mTLS")
	}

	return tlsConfig, nil
}

// New creates a new Gateway instance
func New(cfg *config.Config) (*Gateway, error) {
	// Initialize macaroon service
	macaroonSvc, err := macaroon.NewService(cfg.Admin.Token)
	if err != nil {
		return nil, err
	}

	// Initialize L402 service (with Redis if configured for HA)
	// Only initialize if a lightning provider is configured - otherwise L402/charge routes won't work
	// but observe/control routes will still function
	var l402Svc *l402.Service
	if cfg.Lightning.Provider != "" {
		if cfg.Redis != nil && cfg.Redis.Enabled {
			log.Info().Str("addr", cfg.Redis.Addr).Msg("Initializing Redis-backed L402 (HA-safe)")
			redisStore, err := l402.NewRedisStore(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
			if err != nil {
				return nil, err
			}
			l402Svc, err = l402.NewServiceWithStore(cfg.Lightning, redisStore, nil)
			if err != nil {
				return nil, err
			}
		} else {
			log.Warn().Msg("Using in-memory L402 store (not HA-safe)")
			l402Svc, err = l402.NewService(cfg.Lightning)
			if err != nil {
				return nil, err
			}
		}
	} else {
		log.Info().Msg("No lightning provider configured - L402/charge routes disabled")
	}

	// Initialize governance service (with Redis if configured)
	var govSvc *governance.Service
	if cfg.Redis != nil && cfg.Redis.Enabled {
		log.Info().Str("addr", cfg.Redis.Addr).Msg("Initializing Redis-backed governance")
		govSvc, err = governance.NewServiceWithRedis(&governance.RedisConfig{
			Addr:     cfg.Redis.Addr,
			Password: cfg.Redis.Password,
			DB:       cfg.Redis.DB,
		})
		if err != nil {
			return nil, err
		}
	} else {
		log.Warn().Msg("Using in-memory governance (not HA-safe)")
		govSvc = governance.NewService(nil)
	}

	// Create reverse proxies for each upstream
	proxies := make(map[string]*httputil.ReverseProxy)
	for name, upstream := range cfg.Upstreams {
		target, err := url.Parse(upstream.URL)
		if err != nil {
			return nil, err
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		proxy.ModifyResponse = func(resp *http.Response) error {
			// Add timing header
			return nil
		}
		upstreamName := name // Capture for closure
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error().Err(err).Str("upstream", upstreamName).Msg("Proxy error")
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}

		timeout := upstream.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		// Build transport with optional TLS/mTLS configuration
		transport := &http.Transport{
			ResponseHeaderTimeout: timeout,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
		}

		// Configure TLS if specified
		if upstream.TLS != nil {
			tlsConfig, err := buildTLSConfig(upstream.TLS, target.Hostname())
			if err != nil {
				log.Error().Err(err).Str("upstream", name).Msg("Failed to configure TLS")
				return nil, err
			}
			transport.TLSClientConfig = tlsConfig
			log.Info().
				Str("upstream", name).
				Bool("mTLS", upstream.TLS.ClientCertFile != "").
				Bool("customCA", upstream.TLS.CACertFile != "").
				Msg("Upstream TLS configured")
		}

		proxy.Transport = transport
		proxies[name] = proxy
	}

	// Parse trusted proxy CIDRs for X-Tenant-ID header
	var trustedCIDRs []*net.IPNet
	for _, cidr := range cfg.Server.TrustedProxies {
		_, network, err := net.ParseCIDR(cidr)
		if err != nil {
			log.Warn().Str("cidr", cidr).Err(err).Msg("Invalid trusted proxy CIDR")
			continue
		}
		trustedCIDRs = append(trustedCIDRs, network)
	}

	// SECURITY WARNING: Dev mode tenant header trust
	if cfg.Server.TrustTenantHeaderInDev {
		log.Warn().Msg("⚠️  SECURITY: trustTenantHeaderInDev=true - X-Tenant-ID trusted from ANY source. DO NOT USE IN PRODUCTION!")
	}

	// Initialize route rate limiter
	routeRL, err := NewRouteRateLimiter(cfg)
	if err != nil {
		log.Warn().Err(err).Msg("Failed to initialize route rate limiter")
		// Continue without rate limiting - fail open
	}

	return &Gateway{
		config:            cfg,
		macaroonSvc:       macaroonSvc,
		l402Svc:           l402Svc,
		governance:        govSvc,
		proxies:           proxies,
		dynamicProxies:    NewDynamicProxyCache(DynamicProxyTTL),
		metrics:           &Metrics{},
		trustedProxyCIDRs: trustedCIDRs,
		routeRateLimiter:  routeRL,
	}, nil
}

// SetTenantEnforcer sets the tenant isolation enforcer
func (g *Gateway) SetTenantEnforcer(enforcer *tenant.IsolationEnforcer) {
	g.tenantEnforcer = enforcer
}

// SetBillingManager sets the enterprise billing manager
func (g *Gateway) SetBillingManager(bm BillingManager) {
	g.billingManager = bm
}

// SetPolicyChecker sets the plan-based policy checker for Economic Firewall tier gating
func (g *Gateway) SetPolicyChecker(pc PolicyChecker) {
	g.policyChecker = pc
}

// SetHACoordinator sets the HA coordinator for metrics recording.
func (g *Gateway) SetHACoordinator(ha HACoordinator) {
	g.haCoordinator = ha
}

// SetBudgetEnforcer sets the budget enforcer for delegation token limits.
// When set, fiat402 (Control) policy requests will enforce budgets before proxying.
func (g *Gateway) SetBudgetEnforcer(be BudgetEnforcer) {
	g.budgetEnforcer = be
}

// SetDelegationVerifier sets the delegation token verifier.
// When set, Bearer tokens starting with "stks_" will be verified as delegation tokens
// and their token ID (not secret) will be used for budget enforcement.
func (g *Gateway) SetDelegationVerifier(dv DelegationVerifier) {
	g.delegationVerifier = dv
}

// SetSettlementService sets the settlement service for L402 → Credits integration
func (g *Gateway) SetSettlementService(ss SettlementService) {
	g.settlementService = ss
}

// ServeHTTP handles incoming requests
func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()
	g.metrics.TotalRequests++

	// Wrap response writer to capture status code for HA metrics
	sw := &statusResponseWriter{ResponseWriter: w, statusCode: http.StatusOK}
	w = sw // Use wrapped writer for all subsequent calls

	// Track active connections for HA metrics
	if g.haCoordinator != nil {
		g.haCoordinator.IncrementConnections()
		defer func() {
			g.haCoordinator.DecrementConnections()
			// Record request metrics
			latency := time.Since(start)
			isError := sw.statusCode >= 400
			g.haCoordinator.RecordRequest(latency, isError)
		}()
	}

	// SECURITY: Host allowlist validation (skip health probes)
	if len(g.config.Server.AllowedHosts) > 0 && r.URL.Path != "/healthz" && r.URL.Path != "/readyz" {
		host := r.Host
		// Strip port if present
		if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
			host = host[:colonIdx]
		}
		allowed := false
		for _, allowedHost := range g.config.Server.AllowedHosts {
			if strings.EqualFold(host, allowedHost) {
				allowed = true
				break
			}
		}
		if !allowed {
			log.Warn().Str("host", r.Host).Strs("allowed", g.config.Server.AllowedHosts).Msg("Request to disallowed host")
			http.Error(w, `{"error":"Host not allowed"}`, http.StatusForbidden)
			return
		}
	}

	// SECURITY: Strip X-Tenant-ID unless from trusted proxy OR dev mode is enabled
	// This prevents tenant spoofing from untrusted sources in production
	if r.Header.Get("X-Tenant-ID") != "" && !g.isFromTrustedProxy(r) {
		if g.config.Server.TrustTenantHeaderInDev {
			// DEV MODE: Trust header from any source (for local testing)
			log.Debug().
				Str("client_ip", r.RemoteAddr).
				Str("tenant_id", r.Header.Get("X-Tenant-ID")).
				Msg("Trusting X-Tenant-ID in dev mode (trustTenantHeaderInDev=true)")
		} else {
			// PRODUCTION: Strip untrusted header
			log.Warn().
				Str("client_ip", r.RemoteAddr).
				Str("tenant_id", r.Header.Get("X-Tenant-ID")).
				Msg("Stripped X-Tenant-ID from untrusted source")
			r.Header.Del("X-Tenant-ID")
		}
	}

	// SINGLE-TENANT DEFAULT: If no tenant context and defaultTenantId is set, inject it
	// This enables TENANT_ISOLATION_REQUIRED=true without breaking clients
	if r.Header.Get("X-Tenant-ID") == "" && g.config.Server.DefaultTenantId != "" {
		r.Header.Set("X-Tenant-ID", g.config.Server.DefaultTenantId)
		log.Debug().Str("default_tenant", g.config.Server.DefaultTenantId).Msg("Injected default tenant")
	}

	// Enforce tenant isolation if configured
	if g.config.Server.TenantIsolationEnabled {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			if g.config.Server.TenantIsolationRequired {
				log.Warn().Msg("Tenant isolation required but no tenant context")
				http.Error(w, `{"error":"Tenant context required"}`, http.StatusForbidden)
				return
			}
			log.Debug().Msg("Tenant isolation: no tenant context (optional mode)")
		} else {
			// Add verified tenant to response headers for tracing
			w.Header().Set("X-Tenant-ID", tenantID)

			// Enforce quotas if enabled and we have a tenant enforcer
			if g.config.Server.TenantQuotasEnabled && g.tenantEnforcer != nil {
				tenantCtx, err := g.tenantEnforcer.GetTenant(r)
				if err == nil {
					if err := g.tenantEnforcer.EnforceQuotas(r.Context(), tenantCtx); err != nil {
						log.Warn().Err(err).Str("tenant", tenantID).Msg("Quota exceeded")
						w.Header().Set("X-RateLimit-Exceeded", "true")
						http.Error(w, `{"error":"Quota exceeded"}`, http.StatusTooManyRequests)
						return
					}
					defer g.tenantEnforcer.RecordUsage(tenantCtx.ID, 0)
				}
			}
		}
	}

	// Extract headers for matching
	headers := make(map[string]string)
	for k, v := range r.Header {
		if len(v) > 0 {
			headers[k] = v[0]
		}
	}

	// Find matching route - use per-tenant config from context if available (cloud mode),
	// otherwise fall back to global config (self-hosted mode)
	var route *config.Route
	if tenantCfg := cloud.TenantRoutingConfig(r.Context()); tenantCfg != nil {
		// Cloud mode: use tenant-specific routes
		route = tenantCfg.GetRouteWithHeaders(r.URL.Path, r.Method, headers)
	} else {
		// Self-hosted mode: use global config
		route = g.config.GetRouteWithHeaders(r.URL.Path, r.Method, headers)
	}
	if route == nil {
		http.Error(w, "Not Found", http.StatusNotFound)
		return
	}

	// Check route-level rate limit (before processing)
	if g.routeRateLimiter != nil {
		allowed, rlHeaders := g.routeRateLimiter.Check(r, route)
		for k, v := range rlHeaders {
			w.Header().Set(k, v)
		}
		if !allowed {
			g.metrics.TotalDenied++
			http.Error(w, `{"error":"Rate limit exceeded"}`, http.StatusTooManyRequests)
			return
		}
	}

	// Normalize policy kind to canonical form
	// This allows strategic aliases (protect, audit, budget, monetize) to work at runtime
	policyKind := config.NormalizePolicyKind(route.Policy.Kind)

	// Check if tenant's plan allows this policy (Economic Firewall tier gating)
	// - Observe (chargeback) is unlimited on all plans
	// - Control (fiat402) and Charge (l402) require Pro+ plan
	if g.policyChecker != nil {
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID != "" && policyKind != "public" && policyKind != "deny" {
			allowed, reason := g.policyChecker.CheckPolicyAllowed(r.Context(), tenantID, policyKind)
			if !allowed {
				log.Warn().
					Str("tenant", tenantID).
					Str("policy", policyKind).
					Str("reason", reason).
					Msg("Policy not allowed for tenant plan")
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusForbidden)
				w.Write([]byte(`{"error":"` + reason + `","code":"PLAN_FEATURE_REQUIRED"}`))
				return
			}
		}
	}

	// Handle based on policy
	switch policyKind {
	case "public":
		g.proxyRequest(w, r, route)

	case "deny":
		g.metrics.TotalDenied++
		http.Error(w, "Forbidden", http.StatusForbidden)

	case "l402":
		// Check if we should use billing manager (unified pay path) or legacy handler
		if route.Policy.Pay != nil && g.billingManager != nil {
			g.handlePay(w, r, route, start)
		} else {
			g.handleL402(w, r, route)
		}

	case "capability":
		g.handleCapability(w, r, route)

	case "pay":
		// Pay policy with budget enforcement (when enforceBudget: true)
		// Layer 0: Default Protection - verify credential (Economic Firewall)
		var ok bool
		r, ok = g.verifyDefaultProtection(w, r, route)
		if !ok {
			return // 401/403 already sent
		}

		// Layer 1.5: Budget enforcement (economic firewall hard stop)
		// Only enforce when route has enforceBudget: true AND we have a budgetEnforcer
		if g.budgetEnforcer != nil && route.Policy.Pay != nil && route.Policy.Pay.EnforceBudget {
			if principal := GetVerifiedPrincipal(r.Context()); principal != nil && principal.BudgetSubjectID != "" {
				// Get stable idempotency key from headers
				requestID := r.Header.Get("X-Request-ID")
				if requestID == "" {
					requestID = r.Header.Get("Idempotency-Key")
				}

				// Use route's configured cost, or 0 to use default (1 credit)
				costCredits := route.Policy.CostCredits

				result, err := g.budgetEnforcer.EnforceFunc(
					r.Context(),
					principal.BudgetSubjectID, // Use budget subject ID (delegation token ID)
					principal.TenantID,
					r.URL.Path,  // Use URL path for cost lookup and idempotency
					r.Method,
					route.Policy.Pay.Mode,
					requestID,
					costCredits, // Per-route cost (0 = use default)
				)
				if err != nil {
					// Budget exhausted - hard stop
					var remaining int64 = 0
					var cost int64 = 1
					if result != nil {
						remaining = result.RemainingCredits
						cost = result.CostCredits
					}

					log.Warn().
						Str("token", principal.BudgetSubjectID).
						Str("tenant", principal.TenantID).
						Str("path", r.URL.Path).
						Int64("remaining", remaining).
						Msg("Budget enforcement: request denied (pay policy)")

					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Budget-Remaining", strconv.FormatInt(remaining, 10))
					w.Header().Set("X-Budget-Token", principal.BudgetSubjectID)
					w.WriteHeader(http.StatusPaymentRequired) // 402
					w.Write([]byte(`{"error":"budget_exhausted","message":"Insufficient credits for this request","token_id":"` + principal.BudgetSubjectID + `","cost_credits":` + strconv.FormatInt(cost, 10) + `,"remaining_credits":` + strconv.FormatInt(remaining, 10) + `}`))
					return
				}

				// Budget OK - add remaining credits header
				if result != nil {
					w.Header().Set("X-Budget-Remaining", strconv.FormatInt(result.RemainingCredits, 10))
				}
			}
		}
		g.handlePay(w, r, route, start)

	case "chargeback":
		// Observe policy (chargeback mode):
		// Layer 0: Default Protection - verify credential (Economic Firewall)
		// Layer 1: Observe - passive metering through billing manager
		// Layer 1.5: Budget enforcement (when enforceBudget: true)
		var ok bool
		r, ok = g.verifyDefaultProtection(w, r, route)
		if !ok {
			return // 401/403 already sent
		}

		// Layer 1.5: Budget enforcement (economic firewall hard stop)
		// Only enforce when route has enforceBudget: true AND we have a budgetEnforcer
		if g.budgetEnforcer != nil && route.Policy.Pay != nil && route.Policy.Pay.EnforceBudget {
			if principal := GetVerifiedPrincipal(r.Context()); principal != nil && principal.BudgetSubjectID != "" {
				// Get stable idempotency key from headers
				requestID := r.Header.Get("X-Request-ID")
				if requestID == "" {
					requestID = r.Header.Get("Idempotency-Key")
				}

				// Use route's configured cost, or 0 to use default (1 credit)
				costCredits := route.Policy.CostCredits

				result, err := g.budgetEnforcer.EnforceFunc(
					r.Context(),
					principal.BudgetSubjectID,
					principal.TenantID,
					r.URL.Path,
					r.Method,
					"chargeback",
					requestID,
					costCredits,
				)
				if err != nil {
					// Budget exhausted - hard stop
					var remaining int64 = 0
					var cost int64 = 1
					if result != nil {
						remaining = result.RemainingCredits
						cost = result.CostCredits
					}

					log.Warn().
						Str("token", principal.BudgetSubjectID).
						Str("tenant", principal.TenantID).
						Str("path", r.URL.Path).
						Int64("remaining", remaining).
						Msg("Budget enforcement: request denied (chargeback policy)")

					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Budget-Remaining", strconv.FormatInt(remaining, 10))
					w.Header().Set("X-Budget-Token", principal.BudgetSubjectID)
					w.WriteHeader(http.StatusPaymentRequired) // 402
					w.Write([]byte(`{"error":"budget_exhausted","message":"Insufficient credits for this request","token_id":"` + principal.BudgetSubjectID + `","cost_credits":` + strconv.FormatInt(cost, 10) + `,"remaining_credits":` + strconv.FormatInt(remaining, 10) + `}`))
					return
				}

				// Budget OK - add remaining credits header
				if result != nil {
					w.Header().Set("X-Budget-Remaining", strconv.FormatInt(result.RemainingCredits, 10))
				}
			}
		}

		// Only call handlePay if billing manager AND pay policy are configured
		// Simple observe routes may not have a pay policy - they just need credential verification
		if g.billingManager != nil && route.Policy.Pay != nil {
			g.handlePay(w, r, route, start)
		} else {
			// Observe mode without pay policy - proxy with credential verification only
			if route.Policy.Pay == nil {
				log.Debug().Str("route", route.Name).Msg("Observe mode: credential verified, proxying without metering")
			} else {
				log.Warn().Str("route", route.Name).Msg("Chargeback mode requires billing manager - proxying without metering")
			}
			g.proxyRequest(w, r, route)
		}

	case "fiat402":
		// Control policy (fiat402 mode):
		// Layer 0: Default Protection - verify credential (Economic Firewall)
		// Layer 1: Control - budget enforcement (hard stop) + billing manager
		var ok bool
		r, ok = g.verifyDefaultProtection(w, r, route)
		if !ok {
			return // 401/403 already sent
		}
		
		// Layer 1.5: Budget enforcement (economic firewall hard stop)
		// Only enforce for delegation tokens (which have BudgetSubjectID set)
		// Macaroons don't have budget allocation and are not enforced here
		if g.budgetEnforcer != nil {
			if principal := GetVerifiedPrincipal(r.Context()); principal != nil && principal.BudgetSubjectID != "" {
				// Get stable idempotency key from headers
				// Use URL path (not route name) for deterministic fallback
				requestID := r.Header.Get("X-Request-ID")
				if requestID == "" {
					requestID = r.Header.Get("Idempotency-Key")
				}
				
				// Use route's configured cost, or 0 to use default (1 credit)
				costCredits := route.Policy.CostCredits
				
				result, err := g.budgetEnforcer.EnforceFunc(
					r.Context(),
					principal.BudgetSubjectID, // Use budget subject ID (delegation token ID)
					principal.TenantID,
					r.URL.Path,  // Use URL path for cost lookup and idempotency
					r.Method,
					"fiat402",
					requestID,
					costCredits, // Per-route cost (0 = use default)
				)
				if err != nil {
					// Budget exhausted - hard stop
					var remaining int64 = 0
					var cost int64 = 1
					if result != nil {
						remaining = result.RemainingCredits
						cost = result.CostCredits
					}
					
					log.Warn().
						Str("token", principal.BudgetSubjectID).
						Str("tenant", principal.TenantID).
						Str("path", r.URL.Path).
						Int64("remaining", remaining).
						Msg("Budget enforcement: request denied")
					
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("X-Budget-Remaining", strconv.FormatInt(remaining, 10))
					w.Header().Set("X-Budget-Token", principal.BudgetSubjectID)
					w.WriteHeader(http.StatusPaymentRequired) // 402
					w.Write([]byte(`{"error":"budget_exhausted","message":"Insufficient credits for this request","token_id":"` + principal.BudgetSubjectID + `","cost_credits":` + strconv.FormatInt(cost, 10) + `,"remaining_credits":` + strconv.FormatInt(remaining, 10) + `}`))
					return
				}
				
				// Budget OK - add remaining credits header
				if result != nil {
					w.Header().Set("X-Budget-Remaining", strconv.FormatInt(result.RemainingCredits, 10))
				}
			}
		}
		
		if g.billingManager != nil {
			g.handlePay(w, r, route, start)
		} else {
			log.Error().Str("route", route.Name).Msg("Fiat402 mode requires billing manager")
			http.Error(w, "Payment infrastructure not configured", http.StatusServiceUnavailable)
		}

	default:
		http.Error(w, "Invalid policy", http.StatusInternalServerError)
	}

	// Structured routing decision log
	// Includes all context needed for debugging and audit
	logEvent := log.Debug().
		Str("method", r.Method).
		Str("path", r.URL.Path).
		Str("route", route.Name).
		Str("upstream", route.Upstream).
		Str("policy", route.Policy.Kind).
		Dur("duration", time.Since(start))

	// Add tenant context if available
	if tenantID := r.Header.Get("X-Tenant-ID"); tenantID != "" {
		logEvent = logEvent.Str("tenant_id", tenantID)
	}
	if tenantSlug := r.Header.Get("X-Tenant-Slug"); tenantSlug != "" {
		logEvent = logEvent.Str("tenant_slug", tenantSlug)
	}

	// Add config version if using tenant routing
	if tenantCfg := cloud.TenantRoutingConfig(r.Context()); tenantCfg != nil {
		if resolved := cloud.TenantContext(r.Context()); resolved != nil {
			logEvent = logEvent.Int("config_version", resolved.Version)
		}
	}

	// Add policy scope if present
	if route.Policy.Scope != "" {
		logEvent = logEvent.Str("policy_scope", route.Policy.Scope)
	}

	logEvent.Msg("Routing decision")
}

// handleL402 processes L402 (paid) requests
func (g *Gateway) handleL402(w http.ResponseWriter, r *http.Request, route *config.Route) {
	g.metrics.TotalL402++

	// Check for existing L402 token
	authHeader := r.Header.Get("Authorization")
	if authHeader != "" && strings.HasPrefix(authHeader, "L402 ") {
		// Verify the L402 token (macaroon:preimage format)
		token := strings.TrimPrefix(authHeader, "L402 ")
		
		// Check if L402 service is initialized
		if g.l402Svc == nil {
			log.Error().Msg("L402 service not initialized - Lightning provider not configured?")
			http.Error(w, `{"error":"L402 service unavailable"}`, http.StatusServiceUnavailable)
			return
		}
		
		l402Token, err := g.l402Svc.ValidateToken(token)
		if err != nil {
			log.Error().Err(err).Str("token_prefix", token[:min(50, len(token))]).Msg("L402 token validation failed")
			// Fall through to create new challenge
		} else {
			// Verify scope matches route requirement
			if route.Policy.Scope != "" && l402Token.Scope != route.Policy.Scope {
				// Check if token scope is sufficient
				if !scopeAllows(l402Token.Scope, route.Policy.Scope) {
					log.Debug().
						Str("tokenScope", l402Token.Scope).
						Str("requiredScope", route.Policy.Scope).
						Msg("L402 token scope insufficient")
					http.Error(w, "Insufficient scope", http.StatusForbidden)
					return
				}
			}

			// L402 → Credits coupling (Phase 3)
			// If X-Credit-Token header is present, add credits to the delegation token
			creditToken := r.Header.Get("X-Credit-Token")
			if creditToken != "" {
				// Debug: indicate which services are available
				if g.settlementService == nil {
					log.Warn().Msg("L402 → Credits: settlement service not configured")
					w.Header().Set("X-Credit-Settlement", "failed:service_not_configured")
				} else if g.delegationVerifier == nil {
					log.Warn().Msg("L402 → Credits: delegation verifier not configured")
					w.Header().Set("X-Credit-Settlement", "failed:verifier_not_configured")
				}
			}
			if creditToken != "" && g.settlementService != nil && g.delegationVerifier != nil {
				// Verify the delegation token
				tokenID, tenantID, _, err := g.delegationVerifier.VerifySecret(r.Context(), creditToken)
				if err != nil {
					log.Warn().Err(err).Str("credit_token_prefix", creditToken[:min(10, len(creditToken))]).Msg("L402 credit token verification failed")
					// Don't fail the request - just don't credit
					w.Header().Set("X-Credit-Settlement", "failed:invalid_token")
				} else {
					// Get credits per sat from route config (default: 1)
					creditsPerSat := int64(1)
					if route.Policy.Pay != nil && route.Policy.Pay.CreditsPerSat > 0 {
						creditsPerSat = route.Policy.Pay.CreditsPerSat
					}
					
					// Settle L402 payment → credits (idempotent by payment hash)
					credits, newBudget, err := g.settlementService.SettleL402(
						r.Context(),
						tenantID,
						tokenID,
						l402Token.PriceSats,
						creditsPerSat,
						l402Token.PaymentHash, // Idempotency key
					)
					if err != nil {
						log.Warn().Err(err).Str("token", tokenID).Msg("L402 → Credits settlement failed")
						// Don't fail the request - payment was valid, just couldn't add credits
						if strings.Contains(err.Error(), "duplicate") {
							w.Header().Set("X-Credit-Settlement", "duplicate")
						} else {
							w.Header().Set("X-Credit-Settlement", "failed:settlement_error")
						}
					} else {
						log.Info().
							Str("token", tokenID).
							Int64("sats", l402Token.PriceSats).
							Int64("credits", credits).
							Int64("new_budget", newBudget).
							Msg("L402 → Credits: payment settled to delegation token")
						
						// Add headers to indicate credits were added
						w.Header().Set("X-Credit-Settlement", "ok")
						w.Header().Set("X-Credits-Added", fmt.Sprintf("%d", credits))
						w.Header().Set("X-Budget-Remaining", fmt.Sprintf("%d", newBudget))
					}
				}
			}

			// Valid token - proxy the request
			g.proxyRequest(w, r, route)
			return
		}
	}

	// No valid token, return 402 with challenge
	// Get price - prefer PriceSats, fallback to Pay.Price
	priceSats := route.Policy.PriceSats
	if priceSats <= 0 && route.Policy.Pay != nil && route.Policy.Pay.Price > 0 {
		priceSats = int64(route.Policy.Pay.Price)
	}
	if priceSats <= 0 {
		priceSats = 1 // Minimum 1 sat
	}

	challenge, err := g.l402Svc.CreateChallenge(priceSats, route.Policy.Scope, route.Policy.Tier)
	if err != nil {
		log.Error().Err(err).Msg("Failed to create L402 challenge")
		http.Error(w, "Internal Server Error", http.StatusInternalServerError)
		return
	}

	w.Header().Set("WWW-Authenticate", challenge)
	w.WriteHeader(http.StatusPaymentRequired)
	w.Write([]byte("Payment Required"))
}

// scopeAllows checks if tokenScope grants access to requiredScope
func scopeAllows(tokenScope, requiredScope string) bool {
	if tokenScope == "*" || tokenScope == requiredScope {
		return true
	}
	if strings.HasSuffix(tokenScope, ":*") {
		prefix := strings.TrimSuffix(tokenScope, "*")
		return strings.HasPrefix(requiredScope, prefix)
	}
	return false
}

// delegationHasScope checks if a delegation token's scopes include the required scope.
// Delegation scopes can be route prefixes or policy modes.
// Supports wildcards: "*" matches everything, "/api/*" matches "/api/anything"
func delegationHasScope(tokenScopes []string, requiredScope string) bool {
	for _, scope := range tokenScopes {
		// Exact match
		if scope == requiredScope {
			return true
		}
		// Wildcard match
		if scope == "*" {
			return true
		}
		// Prefix wildcard: "/api/*" matches "/api/v1/users"
		if strings.HasSuffix(scope, "/*") {
			prefix := strings.TrimSuffix(scope, "/*")
			if strings.HasPrefix(requiredScope, prefix) {
				return true
			}
		}
		// Simple prefix match: "/api/v1" matches "/api/v1/users"
		if strings.HasPrefix(requiredScope, scope) {
			return true
		}
	}
	return false
}

// handleCapability processes capability token requests
// Supports both delegation tokens (stks_*) and macaroons for unified token experience
func (g *Gateway) handleCapability(w http.ResponseWriter, r *http.Request, route *config.Route) {
	g.metrics.TotalCapability++

	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		http.Error(w, "Authorization required", http.StatusUnauthorized)
		return
	}

	var token string
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	} else {
		token = authHeader
	}

	// Try delegation token verification first (if configured and token looks like delegation secret)
	if g.delegationVerifier != nil && strings.HasPrefix(token, "stks_") {
		tokenID, tenantID, scopes, err := g.delegationVerifier.VerifySecret(r.Context(), token)
		if err == nil {
			// Valid delegation token
			log.Debug().
				Str("token_id", tokenID).
				Str("tenant_id", tenantID).
				Str("route", route.Name).
				Strs("scopes", scopes).
				Msg("Capability: delegation token verified")

			// Check scope against route.Policy.Scope if specified
			if route.Policy.Scope != "" {
				if !delegationHasScope(scopes, route.Policy.Scope) {
					log.Debug().
						Str("required_scope", route.Policy.Scope).
						Strs("token_scopes", scopes).
						Msg("Capability: delegation token scope insufficient")
					http.Error(w, "Insufficient scope", http.StatusForbidden)
					return
				}
			}

			// Proxy the request - delegation tokens don't need governance tracking
			g.proxyRequest(w, r, route)
			return
		}
		// Delegation verification failed - log and fall through to macaroon
		log.Debug().Err(err).Str("route", route.Name).Msg("Capability: delegation token verification failed, trying macaroon")
	}

	// Fall back to macaroon verification
	mac, err := g.macaroonSvc.Verify(token)
	if err != nil {
		log.Debug().Err(err).Msg("Macaroon verification failed")
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}

	// Check if token is banned
	if g.governance.IsBanned(mac.Signature) {
		http.Error(w, "Token revoked", http.StatusUnauthorized)
		return
	}

	// Check scope if specified
	if route.Policy.Scope != "" {
		if !mac.HasScope(route.Policy.Scope) {
			http.Error(w, "Insufficient scope", http.StatusForbidden)
			return
		}
	}

	// Record usage
	g.governance.RecordUsage(mac.Signature, r.URL.Path)

	// Proxy the request
	g.proxyRequest(w, r, route)
}

// principalKey is the context key for the verified principal
type principalKey struct{}

// PrincipalType identifies how the caller was authenticated
type PrincipalType string

const (
	PrincipalTypeMacaroon    PrincipalType = "macaroon"
	PrincipalTypeDelegation  PrincipalType = "delegation_v2"
)

// VerifiedPrincipal represents an authenticated caller after Layer 0 protection
type VerifiedPrincipal struct {
	Type            PrincipalType // How the caller was authenticated
	SubjectID       string        // Stable identifier (macaroon signature or delegation token ID)
	BudgetSubjectID string        // ID for budget enforcement (only set for delegation tokens)
	TenantID        string        // Tenant UUID (from token for delegation, header for macaroon)
	Scopes          []string
}

// GetVerifiedPrincipal extracts the verified principal from context
func GetVerifiedPrincipal(ctx context.Context) *VerifiedPrincipal {
	if v := ctx.Value(principalKey{}); v != nil {
		if p, ok := v.(*VerifiedPrincipal); ok {
			return p
		}
	}
	return nil
}

// verifyDefaultProtection ensures a valid credential is present on all protected routes.
// This is Layer 0 - the foundation of "Protect by default" promise.
// All non-PUBLIC routes must have a valid credential before any Layer 1 policy is applied.
// This enables the "Economic Firewall" architecture: verify first, then apply policy.
// On success, stores VerifiedPrincipal in request context for downstream use.
//
// Verification order:
// 1. Try delegation token verification (if verifier configured and token starts with stks_)
// 2. Fall back to macaroon verification
func (g *Gateway) verifyDefaultProtection(w http.ResponseWriter, r *http.Request, route *config.Route) (*http.Request, bool) {
	// Extract token from Authorization header
	authHeader := r.Header.Get("Authorization")
	if authHeader == "" {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Authorization required","code":"DEFAULT_PROTECTION"}`, http.StatusUnauthorized)
		return r, false
	}

	var token string
	if strings.HasPrefix(authHeader, "Bearer ") {
		token = strings.TrimPrefix(authHeader, "Bearer ")
	} else if strings.HasPrefix(authHeader, "L402 ") {
		// Also accept L402 format for charge routes
		token = strings.TrimPrefix(authHeader, "L402 ")
		// For L402, extract just the macaroon part (before :preimage)
		if idx := strings.Index(token, ":"); idx > 0 {
			token = token[:idx]
		}
	} else {
		token = authHeader
	}

	// Try delegation token verification first (if configured and token looks like delegation secret)
	if g.delegationVerifier != nil && strings.HasPrefix(token, "stks_") {
		tokenID, tenantID, scopes, err := g.delegationVerifier.VerifySecret(r.Context(), token)
		if err == nil {
			// Valid delegation token - use token ID for budget enforcement
			// Note: tenantID comes from the token record, not X-Tenant-ID header
			log.Debug().
				Str("token_id", tokenID).
				Str("tenant_id", tenantID).
				Str("route", route.Name).
				Strs("scopes", scopes).
				Msg("Default Protection: delegation token verified")

			// Check scope against route.Policy.Scope if specified
			// Delegation token scopes include route prefixes and policy modes
			if route.Policy.Scope != "" {
				if !delegationHasScope(scopes, route.Policy.Scope) {
					log.Debug().
						Str("required_scope", route.Policy.Scope).
						Strs("token_scopes", scopes).
						Msg("Default Protection: delegation token scope insufficient")
					w.Header().Set("Content-Type", "application/json")
					http.Error(w, `{"error":"Insufficient scope","code":"DEFAULT_PROTECTION"}`, http.StatusForbidden)
					return r, false
				}
			}

			principal := &VerifiedPrincipal{
				Type:            PrincipalTypeDelegation,
				SubjectID:       tokenID,
				BudgetSubjectID: tokenID, // Use token ID for budget enforcement
				TenantID:        tenantID,
				Scopes:          scopes,
			}
			ctx := context.WithValue(r.Context(), principalKey{}, principal)
			return r.WithContext(ctx), true
		}
		// Delegation verification failed - log and fall through to macaroon
		log.Debug().Err(err).Str("route", route.Name).Msg("Default Protection: delegation token verification failed, trying macaroon")
	}

	// Fall back to macaroon verification
	mac, err := g.macaroonSvc.Verify(token)
	if err != nil {
		log.Debug().Err(err).Str("route", route.Name).Msg("Default Protection: macaroon verification failed")
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Invalid credential","code":"DEFAULT_PROTECTION"}`, http.StatusUnauthorized)
		return r, false
	}

	// Check if token is banned
	if g.governance.IsBanned(mac.Signature) {
		w.Header().Set("Content-Type", "application/json")
		http.Error(w, `{"error":"Credential revoked","code":"DEFAULT_PROTECTION"}`, http.StatusUnauthorized)
		return r, false
	}

	// Check scope if specified
	if route.Policy.Scope != "" {
		if !mac.HasScope(route.Policy.Scope) {
			w.Header().Set("Content-Type", "application/json")
			http.Error(w, `{"error":"Insufficient scope","code":"DEFAULT_PROTECTION"}`, http.StatusForbidden)
			return r, false
		}
	}

	// Record usage for governance tracking
	g.governance.RecordUsage(mac.Signature, r.URL.Path)

	// Store verified principal in context for downstream use
	// Macaroons don't have budget enforcement - BudgetSubjectID is empty
	var scopes []string
	if scope := mac.GetScope(); scope != "" {
		scopes = strings.Split(scope, ",")
	}
	principal := &VerifiedPrincipal{
		Type:            PrincipalTypeMacaroon,
		SubjectID:       mac.Signature,
		BudgetSubjectID: "", // Macaroons are not budget-enforced
		TenantID:        r.Header.Get("X-Tenant-ID"),
		Scopes:          scopes,
	}
	ctx := context.WithValue(r.Context(), principalKey{}, principal)
	
	return r.WithContext(ctx), true
}

// handlePay processes requests with enterprise billing policies
func (g *Gateway) handlePay(w http.ResponseWriter, r *http.Request, route *config.Route, start time.Time) {
	// Check if billing manager is configured
	// P0 FIX: Fail-closed when billing manager not configured for pay routes
	if g.billingManager == nil {
		log.Error().Str("route", route.Name).Msg("Billing manager not configured for pay route - denying request")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"Payments system not configured","code":"PAYMENTS_NOT_CONFIGURED"}`))
		return
	}

	// Check if pay policy is configured
	if route.Policy.Pay == nil {
		log.Error().Str("route", route.Name).Msg("Pay policy not configured for pay route - denying request")
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(http.StatusServiceUnavailable)
		w.Write([]byte(`{"error":"Pay policy not configured","code":"PAY_POLICY_MISSING"}`))
		return
	}

	pay := route.Policy.Pay
	policy := &BillingPolicy{
		Mode:             pay.Mode,
		Price:            pay.Price,
		Unit:             pay.Unit,
		Scope:            pay.Scope,
		CostCenterHeader: pay.CostCenterHeader,
		EnforceBudget:    pay.EnforceBudget,
		RouteName:        route.Name, // P0 FIX: Propagate route name
	}

	// Authorize the request
	decision, err := g.billingManager.Authorize(r.Context(), r, policy)
	if err != nil {
		// P0 FIX: Fail-closed on billing errors (manager returns error with decision for 503)
		log.Error().Err(err).Str("route", route.Name).Msg("Billing authorization error - denying request")
		// Use the decision from manager (which contains fail-closed response)
		if decision.StatusCode == 0 {
			decision.StatusCode = http.StatusServiceUnavailable
		}
		if decision.Body == nil {
			decision.Body = []byte(`{"error":"Payments authorization failed","code":"PAYMENTS_AUTH_ERROR"}`)
		}
	}

	// Add billing headers
	for k, v := range decision.Headers {
		w.Header().Set(k, v)
	}

	// L402 → Credits settlement (Phase 3)
	// If this was an L402 payment and X-Credit-Token is present, settle credits
	if decision.Allowed && decision.Headers["X-L402-Payment-Hash"] != "" {
		creditToken := r.Header.Get("X-Credit-Token")
		if creditToken != "" {
			if g.settlementService == nil {
				log.Warn().Msg("L402 → Credits: settlement service not configured")
				w.Header().Set("X-Credit-Settlement", "failed:service_not_configured")
			} else if g.delegationVerifier == nil {
				log.Warn().Msg("L402 → Credits: delegation verifier not configured")
				w.Header().Set("X-Credit-Settlement", "failed:verifier_not_configured")
			} else {
				// Verify the delegation token
				tokenID, tenantID, _, err := g.delegationVerifier.VerifySecret(r.Context(), creditToken)
				if err != nil {
					log.Warn().Err(err).Str("credit_token_prefix", creditToken[:min(10, len(creditToken))]).Msg("L402 credit token verification failed")
					w.Header().Set("X-Credit-Settlement", "failed:invalid_token")
				} else {
					// Get price from decision or route config
					priceSats := int64(decision.CostRecorded)
					if priceSats == 0 && route.Policy.Pay != nil {
						priceSats = int64(route.Policy.Pay.Price)
					}
					
					// Get credits per sat from route config (default: 1)
					creditsPerSat := int64(1)
					if route.Policy.Pay != nil && route.Policy.Pay.CreditsPerSat > 0 {
						creditsPerSat = route.Policy.Pay.CreditsPerSat
					}
					
					// Settle L402 payment → credits
					paymentHash := decision.Headers["X-L402-Payment-Hash"]
					credits, newBudget, err := g.settlementService.SettleL402(
						r.Context(),
						tenantID,
						tokenID,
						priceSats,
						creditsPerSat,
						paymentHash,
					)
					if err != nil {
						log.Warn().Err(err).Str("token", tokenID).Msg("L402 → Credits settlement failed")
						if strings.Contains(err.Error(), "duplicate") {
							w.Header().Set("X-Credit-Settlement", "duplicate")
						} else {
							w.Header().Set("X-Credit-Settlement", "failed:settlement_error")
						}
					} else {
						log.Info().
							Str("token", tokenID).
							Int64("sats", priceSats).
							Int64("credits", credits).
							Int64("new_budget", newBudget).
							Msg("L402 → Credits: payment settled to delegation token")
						
						w.Header().Set("X-Credit-Settlement", "ok")
						w.Header().Set("X-Credits-Added", fmt.Sprintf("%d", credits))
						w.Header().Set("X-Budget-Remaining", fmt.Sprintf("%d", newBudget))
					}
				}
			}
		}
	}

	// Check if allowed
	if !decision.Allowed {
		if decision.WWWAuthenticate != "" {
			w.Header().Set("WWW-Authenticate", decision.WWWAuthenticate)
		}
		w.Header().Set("Content-Type", "application/json")
		w.WriteHeader(decision.StatusCode)
		if decision.Body != nil {
			w.Write(decision.Body)
		}
		return
	}

	// Capture response for usage metering
	captured := &responseCapture{ResponseWriter: w, statusCode: 200}
	g.proxyRequest(captured, r, route)

	// Record usage asynchronously
	go func() {
		tenantID := r.Header.Get("X-Tenant-ID")
		costCenter := ""
		if policy.CostCenterHeader != "" {
			costCenter = r.Header.Get(policy.CostCenterHeader)
		}
		if costCenter == "" {
			costCenter = r.Header.Get("X-Cost-Center")
		}

		ev := BillingUsageEvent{
			Time:       start,
			Duration:   time.Since(start),
			TenantID:   tenantID,
			CostCenter: costCenter,
			RouteName:  route.Name,
			Method:     r.Method,
			Path:       r.URL.Path,
			StatusCode: captured.statusCode,
			BytesIn:    r.ContentLength,
			BytesOut:   int64(captured.bytesWritten),
			Unit:       policy.Unit,
			Amount:     decision.CostRecorded,
		}

		if err := g.billingManager.RecordUsage(context.Background(), ev, policy); err != nil {
			log.Warn().Err(err).Str("route", route.Name).Msg("Failed to record usage")
		}
	}()
}

// responseCapture wraps ResponseWriter to capture status and size for billing
type responseCapture struct {
	http.ResponseWriter
	statusCode   int
	bytesWritten int
}

func (r *responseCapture) WriteHeader(code int) {
	r.statusCode = code
	r.ResponseWriter.WriteHeader(code)
}

func (r *responseCapture) Write(b []byte) (int, error) {
	n, err := r.ResponseWriter.Write(b)
	r.bytesWritten += n
	return n, err
}

// proxyRequest forwards the request to the upstream
func (g *Gateway) proxyRequest(w http.ResponseWriter, r *http.Request, route *config.Route) {
	var proxy *httputil.ReverseProxy

	// Try tenant-specific upstream first (cloud mode), then fall back to global (self-hosted)
	if tenantCfg := cloud.TenantRoutingConfig(r.Context()); tenantCfg != nil {
		// Cloud mode: use tenant-specific upstreams with dynamic proxy cache
		if upstream, ok := tenantCfg.Upstreams[route.Upstream]; ok {
			var err error
			proxy, err = g.getOrCreateDynamicProxy(upstream)
			if err != nil {
				log.Error().Err(err).Str("upstream", route.Upstream).Msg("Failed to create dynamic proxy")
				http.Error(w, "Bad Gateway", http.StatusBadGateway)
				return
			}
		}
	}

	// Fall back to global config upstreams
	if proxy == nil {
		var ok bool
		proxy, ok = g.proxies[route.Upstream]
		if !ok {
			http.Error(w, "Upstream not found", http.StatusBadGateway)
			return
		}
	}

	// Apply transformations
	if route.Transform != nil {
		if route.Transform.StripPrefix != "" {
			r.URL.Path = strings.TrimPrefix(r.URL.Path, route.Transform.StripPrefix)
		}
		for k, v := range route.Transform.AddHeaders {
			r.Header.Set(k, v)
		}
	}

	proxy.ServeHTTP(w, r)
}

// GetMetrics returns current gateway metrics
func (g *Gateway) GetMetrics() *Metrics {
	return g.metrics
}

// GetGovernance returns the governance service
func (g *Gateway) GetGovernance() *governance.Service {
	return g.governance
}

// GetMacaroonService returns the macaroon service
func (g *Gateway) GetMacaroonService() *macaroon.Service {
	return g.macaroonSvc
}

// GetL402Service returns the L402 payment service
func (g *Gateway) GetL402Service() *l402.Service {
	return g.l402Svc
}

// isFromTrustedProxy checks if the request is from a trusted proxy CIDR
func (g *Gateway) isFromTrustedProxy(r *http.Request) bool {
	if len(g.trustedProxyCIDRs) == 0 {
		return false
	}

	// Extract client IP
	clientIP := r.RemoteAddr
	if colonIdx := strings.LastIndex(clientIP, ":"); colonIdx != -1 {
		clientIP = clientIP[:colonIdx]
	}
	// Handle IPv6 bracketed format
	clientIP = strings.TrimPrefix(strings.TrimSuffix(clientIP, "]"), "[")

	ip := net.ParseIP(clientIP)
	if ip == nil {
		return false
	}

	for _, cidr := range g.trustedProxyCIDRs {
		if cidr.Contains(ip) {
			return true
		}
	}

	return false
}


// getOrCreateDynamicProxy builds (and caches) a reverse proxy for a tenant-defined upstream.
// Uses DynamicProxyCache with TTL eviction to prevent unbounded memory growth.
func (g *Gateway) getOrCreateDynamicProxy(upstream config.Upstream) (*httputil.ReverseProxy, error) {
	key := upstream.URL + "|" + upstream.Timeout.String()

	return g.dynamicProxies.GetOrCreate(key, func() (*httputil.ReverseProxy, error) {
		target, err := url.Parse(upstream.URL)
		if err != nil {
			return nil, err
		}

		proxy := httputil.NewSingleHostReverseProxy(target)
		upstreamName := target.Host
		proxy.ErrorHandler = func(w http.ResponseWriter, r *http.Request, err error) {
			log.Error().Err(err).Str("upstream", upstreamName).Msg("Proxy error")
			http.Error(w, "Bad Gateway", http.StatusBadGateway)
		}

		timeout := upstream.Timeout
		if timeout == 0 {
			timeout = 30 * time.Second
		}

		transport := &http.Transport{
			ResponseHeaderTimeout: timeout,
			MaxIdleConns:          100,
			MaxIdleConnsPerHost:   10,
			IdleConnTimeout:       90 * time.Second,
		}
		proxy.Transport = transport

		return proxy, nil
	})
}

// Shutdown gracefully stops the gateway and releases resources.
// Call this during application shutdown to stop background goroutines.
func (g *Gateway) Shutdown() {
	if g.dynamicProxies != nil {
		g.dynamicProxies.Stop()
	}
	log.Info().Msg("Gateway shutdown complete")
}
