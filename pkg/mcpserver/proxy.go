package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/denial"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// ContextKey type for context values.
type contextKey string

// CtxTenantID is the context key for the tenant ID extracted from the token.
const CtxTenantID contextKey = "satgate_tenant_id"
const CtxTokenInfo contextKey = "satgate_token_info"

// TenantFromContext extracts the tenant ID from a context, or returns empty string.
func TenantFromContext(ctx context.Context) string {
	if v, ok := ctx.Value(CtxTenantID).(string); ok {
		return v
	}
	return ""
}

// UpstreamRouter resolves upstreams per-request. The default implementation
// uses the shared UpstreamManager. Enterprise can override this to provide
// per-tenant upstream isolation in multi-tenant deployments.
type UpstreamRouter interface {
	// AllToolsForTenant returns the aggregated tools list for a specific tenant.
	// If tenantID is empty, returns tools from the shared/default upstreams.
	AllToolsForTenant(ctx context.Context, tenantID string) []json.RawMessage

	// ForwardToolCallForTenant forwards a tool call to the appropriate upstream
	// for the given tenant. Falls back to shared upstreams if tenant has none.
	ForwardToolCallForTenant(ctx context.Context, tenantID, toolName string, params json.RawMessage, timeout time.Duration) (*Response, error)
}

// Proxy is the MCP proxy gateway. It sits between an MCP client and one or
// more upstream MCP servers, enforcing budgets and attributing costs.
type Proxy struct {
	config            *Config
	client            Transport // client-facing transport
	upstream          *UpstreamManager
	router            UpstreamRouter // per-tenant routing (nil = use shared upstream)
	guard             ResponseGuard  // optional, wraps every answer (nil = none)
	budget            BudgetEnforcer
	costs             CostResolver
	tenantCosts       TenantCostResolver // optional, per-tenant cost resolution
	auth              Authenticator
	resource          string // MCP resource URL this proxy serves (see resource_binding.go)
	delegator         *Delegator
	events            EventPublisher
	evidence          EvidenceRecorder
	revocation        RevocationChecker // optional, checks if token is revoked
	limiter           RequestLimiter    // optional, per-token request rate limit
	toolsListEnricher ToolsListEnricher // optional, enriches tools/list with cost metadata
	taskTracker       *TaskTracker      // MCP task-level cost aggregation (SEP-1686)
	gate              CallGate          // optional, stateful checks per tools/call (see call_gate.go)
	tokenID           string            // default session token (from config or first auth)
	rootToken         string            // auto-minted root token (for delegation demos)

	// healthDetail adds a note to /health (SetHealthDetail).
	healthDetail func() string

	// modeResolver, when set, decides each tools/call under its tenant's
	// enforcement mode instead of the process-wide one (enforcement_mode.go).
	modeResolver EnforcementModeResolver

	// signEveryAction is the "sign every action" switch (SetSignEveryAction,
	// every_action.go). Off by default: every answer is then what it was
	// before the switch existed.
	signEveryAction bool
	// Separate vocabulary opt-in: module upgrades must not enable outage receipts.
	signUnavailableCheckRefusals bool

	// reversalFailures counts the give-backs of an Observe carve that failed
	// (ReversalFailures). ea is the rest of the switch's state (holds, the
	// pending store, readiness): see every_action_hold.go.
	reversalFailures atomic.Int64
	eaOnce           sync.Once
	ea               *everyAction
}

// ResponseGuard wraps the handling of every request the proxy answers: the
// SSE message endpoint, the Streamable HTTP endpoint and the stdio loop all
// call it, so it is the single point where an answer leaves for the agent.
//
// handle runs the request and returns what the proxy would send. The guard may
// pass a derived context to handle (to collect per-request state that code
// below the proxy, such as an UpstreamRouter, records), and may rewrite the
// response or the error that handle returns before they go out. Off by
// default: with no guard set the proxy behaves as before.
type ResponseGuard func(ctx context.Context, req *Request, handle func(ctx context.Context) (*Response, error)) (*Response, error)

// SetResponseGuard installs g. Call it before the proxy serves requests.
func (p *Proxy) SetResponseGuard(g ResponseGuard) { p.guard = g }

// defaultRouter wraps the shared UpstreamManager for non-multi-tenant use.
type defaultRouter struct {
	mgr *UpstreamManager
}

func (r *defaultRouter) AllToolsForTenant(_ context.Context, _ string) []json.RawMessage {
	return r.mgr.AllTools()
}

func (r *defaultRouter) ForwardToolCallForTenant(ctx context.Context, _ string, toolName string, params json.RawMessage, timeout time.Duration) (*Response, error) {
	return r.mgr.ForwardToolCall(ctx, toolName, params, timeout)
}

// SetUpstreamRouter overrides the upstream routing strategy.
// Use this for multi-tenant per-tenant upstream isolation.
func (p *Proxy) SetUpstreamRouter(router UpstreamRouter) {
	p.router = router
}

// New creates a new MCP proxy from configuration.
func New(cfg *Config) (*Proxy, error) {
	// Build cost resolver
	costs := NewStaticCostResolver(cfg.Tools.Costs, cfg.Tools.DefaultCost)

	// Build budget enforcer
	budget := NewInMemoryBudgetEnforcer()

	// Build authenticator
	auth, err := NewAuthenticator(cfg.Auth)
	if err != nil {
		return nil, fmt.Errorf("auth: %w", err)
	}

	// Build upstream manager
	upstream := NewUpstreamManager(cfg.Upstreams, cfg.Routing, cfg.DefaultUpstream, cfg.AllowPrivateUpstreams)

	// Normalize enforcement mode to canonical names on startup
	cfg.Enforcement.Mode = normalizeEnforcementMode(cfg.Enforcement.Mode)

	p := &Proxy{
		config:      cfg,
		upstream:    upstream,
		router:      &defaultRouter{mgr: upstream},
		budget:      budget,
		costs:       costs,
		auth:        BindResource(auth, ""),
		events:      &NoOpPublisher{},
		taskTracker: NewTaskTracker(),
	}

	// If macaroon auth, create delegator
	if macAuth, ok := auth.(*MacaroonAuthenticator); ok {
		p.delegator = NewDelegator(macAuth.Service, budget)
	}

	// Initialize budget if configured
	if cfg.Budget.Limit > 0 {
		tokenID := "default"
		if cfg.Auth.Token != "" {
			tokenID = hashToken(cfg.Auth.Token)
		}

		// Auto-mint root macaroon for macaroon auth mode
		if cfg.Auth.AutoMintRoot {
			if macAuth, ok := auth.(*MacaroonAuthenticator); ok {
				if cfg.Auth.RootToken != "" {
					// Reuse a pre-supplied root token (stable across restarts)
					rootMac, verifyErr := macAuth.Service.Verify(cfg.Auth.RootToken)
					if verifyErr != nil {
						return nil, fmt.Errorf("verify supplied root token: %w", verifyErr)
					}
					tokenID = hashToken(rootMac.Identifier + rootMac.Signature)
					p.rootToken = cfg.Auth.RootToken
					log.Info().Str("tokenId", tokenID).Msg("reusing supplied root token")
					log.Info().Str("tokenId", tokenID).Msg("root token available programmatically; not printing secret")
					fmt.Fprintf(os.Stderr, "TOKEN_ID=%s\n", tokenID)
				} else {
					rootMac, mintErr := macAuth.Service.Mint("api:*", time.Now().Add(24*time.Hour))
					if mintErr != nil {
						return nil, fmt.Errorf("auto-mint root token: %w", mintErr)
					}
					rootToken := macAuth.Service.Encode(rootMac)
					// Must match MacaroonAuthenticator.Verify — identifier + signature
					tokenID = hashToken(rootMac.Identifier + rootMac.Signature)
					p.rootToken = rootToken
					log.Info().Str("tokenId", tokenID).Msg("auto-minted root token")
					log.Info().Str("tokenId", tokenID).Msg("root token available programmatically; not printing secret")
					fmt.Fprintf(os.Stderr, "TOKEN_ID=%s\n", tokenID)
				}
			}
		}

		p.tokenID = tokenID
		if err := budget.Initialize(context.Background(), tokenID, cfg.Budget.Limit); err != nil {
			return nil, fmt.Errorf("initialize budget: %w", err)
		}
		log.Info().Int64("limit", cfg.Budget.Limit).Str("token", tokenID).Msg("budget initialized")
	}

	return p, nil
}

// SetBudgetEnforcer replaces the budget enforcer (e.g., with enterprise Redis-backed one).
func (p *Proxy) SetBudgetEnforcer(e BudgetEnforcer) {
	p.budget = e
	// Update delegator if it exists
	if p.delegator != nil {
		p.delegator.budget = e
	}
}

// SetCostResolver replaces the global cost resolver.
// In single-tenant mode, this is the only resolver needed.
func (p *Proxy) SetCostResolver(r CostResolver) {
	p.costs = r
}

// SetTenantCostResolver sets a per-tenant cost resolver for multi-tenant mode.
// When set, this takes priority over the global CostResolver for requests
// that have a tenant ID in their token.
func (p *Proxy) SetTenantCostResolver(r TenantCostResolver) {
	p.tenantCosts = r
}

// SetEnforcementMode changes the enforcement mode at runtime (observe, control, charge).
// Accepts legacy aliases: shadow→observe, soft/hard→control, l402→charge.
func (p *Proxy) SetEnforcementMode(mode string) {
	canonical := normalizeEnforcementMode(mode)
	p.config.Enforcement.Mode = canonical
	log.Info().Str("mode", canonical).Str("raw", mode).Msg("enforcement mode updated")
}

// normalizeEnforcementMode maps legacy mode names to canonical (observe/control/charge).
func normalizeEnforcementMode(mode string) string {
	switch mode {
	case "shadow", "chargeback":
		return "observe"
	case "soft", "hard", "fiat402":
		return "control"
	case "l402":
		return "charge"
	default:
		return mode
	}
}

// GetEnforcementMode returns the current enforcement mode.
func (p *Proxy) GetEnforcementMode() string {
	return p.config.Enforcement.Mode
}

// RootToken returns the auto-minted root token (empty if not using autoMintRoot).
func (p *Proxy) RootToken() string {
	return p.rootToken
}

// SetAuthenticator replaces the authenticator.
func (p *Proxy) SetAuthenticator(a Authenticator) {
	p.auth = BindResource(a, p.resource)
}

// SetEventPublisher replaces the event publisher (e.g., with enterprise Redis pub/sub).
func (p *Proxy) SetEventPublisher(ep EventPublisher) {
	p.events = ep
	if p.delegator != nil {
		p.delegator.SetEventPublisher(ep)
	}
}

// SetHelperRevocationID turns the hlp- token_id caveat for zero-credit children
// on or off (off by default). See Delegator.EnableHelperRevocationID: turn it on
// only when every machine that enforces a revoke reads every token_id caveat.
// Call it before the proxy serves requests.
func (p *Proxy) SetHelperRevocationID(on bool) {
	if p.delegator != nil {
		p.delegator.EnableHelperRevocationID(on)
	}
}

// SetEvidenceRecorder wires signed evidence recording for governed MCP decisions.
func (p *Proxy) SetEvidenceRecorder(rec EvidenceRecorder) {
	p.evidence = rec
	p.every().setRecorder(rec)
	if p.signEveryAction && p.SignEveryActionReady() != nil {
		// A recorder that cannot keep windows replaced one that could: the
		// setup is no longer valid, and calls stop until SetSignEveryAction
		// succeeds again.
		p.every().ready.Store(false)
	}
	p.wireDelegator()
}

// SetRevocationChecker sets the revocation checker for token validation.
func (p *Proxy) SetRevocationChecker(rc RevocationChecker) {
	p.revocation = rc
}

// RequestLimiter bounds how often one verified token may make authenticated
// requests (tools/call, satgate/delegate, satgate/budget). The proxy asks it
// after authentication and revocation and before cost resolution, budget
// spend, event publication or any upstream call, so a refused request costs
// nothing and records nothing. toolName is set for tools/call only.
// retryAfter is the suggested wait before the next request may succeed.
type RequestLimiter interface {
	Allow(ctx context.Context, info *TokenInfo, method, toolName string) (allowed bool, retryAfter time.Duration)
}

// SetRequestLimiter installs a per-token request limiter. Nil (the default)
// leaves requests unlimited.
func (p *Proxy) SetRequestLimiter(l RequestLimiter) {
	p.limiter = l
}

// rateLimitedResponse is the JSON-RPC refusal for a request the limiter
// refused. retry_after_seconds is at least 1.
func rateLimitedResponse(id json.RawMessage, retryAfter time.Duration) *Response {
	secs := int64(math.Ceil(retryAfter.Seconds()))
	if secs < 1 {
		secs = 1
	}
	return NewErrorResponseWithData(id, CodeRateLimited, "rate limit exceeded", map[string]interface{}{
		"error":               "rate_limited",
		"retry_after_seconds": secs,
	})
}

// ToolsListEnricher transforms the tools list before returning to clients.
// Used by enterprise layer to inject cost metadata, budget info, etc.
type ToolsListEnricher func(ctx context.Context, tenantID string, tools []json.RawMessage) []json.RawMessage

// SetToolsListEnricher sets a function that enriches tool definitions
// returned by tools/list (e.g., injecting x-satgate cost metadata).
func (p *Proxy) SetToolsListEnricher(e ToolsListEnricher) {
	p.toolsListEnricher = e
}

// ReloadUpstreams adds, removes, or replaces upstreams at runtime without restart.
// It compares the new config against current upstreams and applies the diff.
func (p *Proxy) ReloadUpstreams(ctx context.Context, newUpstreams map[string]UpstreamConfig) error {
	current := make(map[string]bool)
	for _, name := range p.upstream.UpstreamNames() {
		current[name] = true
	}

	var errors []string

	// Add or update upstreams
	for name, cfg := range newUpstreams {
		if current[name] {
			// Replace: remove then add
			p.upstream.RemoveUpstream(name)
		}
		if err := p.upstream.AddUpstream(ctx, name, cfg); err != nil {
			errors = append(errors, fmt.Sprintf("%s: %v", name, err))
			log.Error().Err(err).Str("upstream", name).Msg("failed to add upstream during reload")
		}
		delete(current, name)
	}

	// Remove upstreams that are no longer in the new config
	for name := range current {
		if err := p.upstream.RemoveUpstream(name); err != nil {
			log.Warn().Err(err).Str("upstream", name).Msg("failed to remove upstream during reload")
		}
	}

	if len(errors) > 0 {
		return fmt.Errorf("reload errors: %s", errors)
	}

	log.Info().Int("upstreams", len(newUpstreams)).Msg("upstreams reloaded")
	return nil
}

// GetUpstream returns the underlying UpstreamManager (for enterprise extensions).
func (p *Proxy) GetUpstreamManager() *UpstreamManager {
	return p.upstream
}

// Run starts the proxy. It blocks until ctx is cancelled or an error occurs.
func (p *Proxy) Run(ctx context.Context, clientTransport Transport) error {
	p.client = clientTransport

	// Start upstreams
	if err := p.upstream.Start(ctx); err != nil {
		return fmt.Errorf("start upstreams: %w", err)
	}
	defer p.upstream.Close()

	log.Info().
		Str("transport", p.config.Server.Transport).
		Str("enforcement", p.config.Enforcement.Mode).
		Str("auth", p.config.Auth.Mode).
		Int64("budget", p.config.Budget.Limit).
		Msg("MCP proxy running")

	// Main message loop
	for {
		msg, err := p.client.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil {
				return nil // clean shutdown
			}
			return fmt.Errorf("client read: %w", err)
		}

		// Parse the message
		req, _, err := ParseMessage(msg)
		if err != nil {
			log.Debug().Err(err).Msg("unparseable client message")
			continue
		}

		if req == nil {
			log.Debug().Msg("unexpected response from client")
			continue
		}

		// Handle the request. withClientLoop marks it as read here (stdio),
		// not over SSE or Streamable HTTP (see authenticate).
		resp, err := p.handleRequest(withClientLoop(ctx), req)
		if err != nil {
			log.Error().Err(err).Str("method", req.Method).Msg("handler error")
			resp = NewErrorResponse(req.ID, CodeInternalError, err.Error())
		}

		// Notifications don't get responses
		if resp == nil {
			continue
		}

		// Send response
		data, err := json.Marshal(resp)
		if err != nil {
			log.Error().Err(err).Msg("marshal response")
			continue
		}

		if err := p.client.WriteMessage(ctx, data); err != nil {
			return fmt.Errorf("client write: %w", err)
		}
	}
}

// handleRequest answers a JSON-RPC request. It is the one place every
// agent-facing transport calls; an installed ResponseGuard wraps it.
func (p *Proxy) handleRequest(ctx context.Context, req *Request) (*Response, error) {
	if p.guard == nil {
		return p.dispatchRequest(ctx, req)
	}
	return p.guard(ctx, req, func(ctx context.Context) (*Response, error) {
		return p.dispatchRequest(ctx, req)
	})
}

// dispatchRequest dispatches a JSON-RPC request to the appropriate handler.
func (p *Proxy) dispatchRequest(ctx context.Context, req *Request) (*Response, error) {
	switch req.Method {
	case MethodInitialize:
		return p.handleInitialize(req)

	case MethodInitialized:
		return nil, nil

	case MethodPing:
		return p.handlePing(req), nil

	case MethodToolsList:
		// With auth mode header or config, the tool list is answered only
		// for a token that verifies (stdio excepted).
		if refused := p.refuseUnauthenticated(ctx, req); refused != nil {
			return refused, nil
		}
		return p.handleToolsListWithCtx(ctx, req)

	case MethodToolsCall, MethodSatGateDelegate, MethodSatGateBudget, "budget/check":
		// These methods require authentication (dispatchAuthenticated).
		return p.dispatchAuthenticated(ctx, req)

	case MethodCancelled:
		log.Debug().RawJSON("params", withoutMetaToken(req.Params)).Msg("cancellation received")
		return nil, nil

	default:
		if IsNotification(req.Method) {
			return nil, nil
		}
		if p.signEveryAction {
			// With the switch on, a method the proxy does not know is an
			// action of a key like any other: it is authenticated, limited
			// and signed before it is passed on.
			return p.dispatchAuthenticated(ctx, req)
		}
		// Passed on to an upstream: with auth mode header or config, only
		// for a token that verifies (stdio excepted).
		if refused := p.refuseUnauthenticated(ctx, req); refused != nil {
			return refused, nil
		}
		return p.forwardToDefault(ctx, req)
	}
}

// authenticate extracts token from the request's _meta field or falls back to config.
// If a token is provided but fails verification, authentication fails (fail-closed).
// verified is the credential that Verify accepted, or "" when the request runs
// under the default identity with no credential.
func (p *Proxy) authenticate(ctx context.Context, req *Request) (info *TokenInfo, verified string, err error) {
	// Try to extract token from params._meta.token (MCP convention for metadata)
	token := extractMetaToken(req.Params)
	if token == "" && p.signEveryAction {
		// The transports keep the bearer credential of the connection apart
		// from params (transportTokenFrom), so a request whose params are not
		// an object, or whose _meta cannot be read, still belongs to the key
		// that was verified. It is verified below like any other.
		token = transportTokenFrom(ctx)
	}

	if token != "" {
		info, err := p.auth.Verify(ctx, token)
		if err != nil {
			// Fail-closed: if a token was provided but is invalid, deny the request.
			// Never silently fall back to default identity on auth failure.
			return nil, "", fmt.Errorf("authentication failed: %w", err)
		}
		return info, token, nil
	}

	// No token provided.
	// In header mode, a missing token is an auth bypass — deny it. Config
	// mode (one static token) denies it too, except for a request read by
	// Run (stdio), whose caller is the local process that started the proxy.
	if p.config.Auth.Mode == "header" || (p.config.Auth.Mode == "config" && !fromClientLoop(ctx)) {
		return nil, "", errNoToken
	}

	// Auth mode "none", and config mode over stdio: the default identity.
	return &TokenInfo{
		TokenID:  p.tokenID,
		BudgetID: p.tokenID,
		Scope:    "*",
	}, "", nil
}

// errNoToken is the refusal of a request that carries no token where one is
// required. Its text is the JSON-RPC error message the caller receives.
var errNoToken = errors.New("authentication required: no token provided")

// authModeRequiresToken reports whether an auth mode verifies a token on
// every request: "header" (macaroons) and "config" (one static token).
func authModeRequiresToken(mode string) bool {
	return mode == "header" || mode == "config"
}

type clientLoopKey struct{}

// withClientLoop marks a request read by Run from its client transport (the
// stdio loop), as opposed to one that arrived over SSE or Streamable HTTP.
func withClientLoop(ctx context.Context) context.Context {
	return context.WithValue(ctx, clientLoopKey{}, true)
}

func fromClientLoop(ctx context.Context) bool {
	v, _ := ctx.Value(clientLoopKey{}).(bool)
	return v
}

// refuseUnauthenticated answers a request that reveals the tool list or
// reaches an upstream without the authenticated path (tools/list, and the
// methods forwardToDefault passes on) when the auth mode requires a token
// ("header" or "config") and the request did not come from Run (stdio). The
// credential checked is the connection's (the Authorization header, see
// withTransportToken), which is also the one forwardToDefault may pass on,
// else params._meta.token. A request with none, or one that does not verify
// or is revoked, gets the refusal dispatchAuthenticated gives; nil means it
// may go on. Its context is left as it was: what tools/list and
// forwardToDefault read from it is unchanged.
func (p *Proxy) refuseUnauthenticated(ctx context.Context, req *Request) *Response {
	if p.config == nil || !authModeRequiresToken(p.config.Auth.Mode) || fromClientLoop(ctx) {
		return nil
	}
	token := transportTokenFrom(ctx)
	if token == "" {
		token = extractMetaToken(req.Params)
	}
	var err error
	var info *TokenInfo
	if token == "" {
		err = errNoToken
	} else if info, err = p.auth.Verify(ctx, token); err != nil {
		err = fmt.Errorf("authentication failed: %w", err)
	} else if info == nil {
		err = fmt.Errorf("authentication failed: no identity")
	}
	if err != nil {
		p.publishAuthFailure(req, err)
		return NewErrorResponse(req.ID, CodePolicyDenied, err.Error())
	}
	if p.revocation != nil && info.BudgetID != "" {
		rctx := ctx
		if info.TenantID != "" {
			rctx = context.WithValue(ctx, CtxTenantID, info.TenantID)
		}
		if p.revocation.IsRevoked(rctx, info.BudgetID) {
			return NewErrorResponse(req.ID, CodePolicyDenied, "token revoked")
		}
	}
	return nil
}

// publishAuthFailure logs and publishes a refused authentication.
func (p *Proxy) publishAuthFailure(req *Request, authErr error) {
	log.Warn().Err(authErr).Str("method", req.Method).Msg("authentication failed")
	p.events.Publish(Event{
		Type:      EventAuthFailure,
		Timestamp: time.Now(),
		Data: map[string]interface{}{
			"method": req.Method,
			"error":  authErr.Error(),
		},
	})
}

// extractMetaToken pulls the token from params._meta.token if present.
func extractMetaToken(params json.RawMessage) string {
	if len(params) == 0 {
		return ""
	}
	var meta struct {
		Meta struct {
			Token string `json:"token"`
		} `json:"_meta"`
	}
	if err := json.Unmarshal(params, &meta); err != nil {
		return ""
	}
	return meta.Meta.Token
}

// handleInitialize responds to the MCP initialize request.
func (p *Proxy) handleInitialize(req *Request) (*Response, error) {
	result, _ := json.Marshal(map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities": map[string]interface{}{
			"tools": map[string]interface{}{
				"listChanged": true,
			},
		},
		"serverInfo": map[string]string{
			"name":    p.config.Server.Name,
			"version": p.config.Server.Version,
		},
	})

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}, nil
}

// handlePing responds to ping requests.
func (p *Proxy) handlePing(req *Request) *Response {
	result, _ := json.Marshal(map[string]interface{}{})
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}
}

// handleToolsList returns the aggregated tools from all upstreams.
// In multi-tenant mode with an UpstreamRouter, returns tenant-specific tools.
func (p *Proxy) handleToolsListWithCtx(ctx context.Context, req *Request) (*Response, error) {
	tenantID := TenantFromContext(ctx)
	tools := p.router.AllToolsForTenant(ctx, tenantID)

	// Allow enterprise layer to enrich tools with cost/budget metadata
	if p.toolsListEnricher != nil {
		tools = p.toolsListEnricher(ctx, tenantID, tools)
	}

	tools = dropReservedTools(tools)

	result, err := json.Marshal(map[string]interface{}{
		"tools": tools,
	})
	if err != nil {
		return nil, err
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}, nil
}

// dropReservedTools removes every tool whose name starts with the reserved
// argument-rules prefix. An upstream can advertise any name; one that starts
// with the prefix is never offered to an agent by a runtime that knows rules.
// A tool entry that cannot be read is dropped too: it could hide such a name.
func dropReservedTools(tools []json.RawMessage) []json.RawMessage {
	kept := make([]json.RawMessage, 0, len(tools))
	for _, t := range tools {
		var probe struct {
			Name json.RawMessage `json:"name"`
		}
		if err := json.Unmarshal(t, &probe); err != nil {
			continue
		}
		var name string
		if len(probe.Name) > 0 {
			if err := json.Unmarshal(probe.Name, &name); err != nil {
				continue
			}
		}
		if macaroon.IsReservedToolName(name) {
			continue
		}
		kept = append(kept, t)
	}
	return kept
}

// priceUnknownRefuses reports whether a call whose price could not be read is
// refused: in a mode that debits and blocks (control, hard, soft, fiat402)
// when the token has a budget, because that debit would be at a guessed
// price. Observe-class modes (observe, shadow, chargeback) never block, so
// they go on with the resolver's default price, as before. A token with no
// budget is not debited by this function in any mode, so its price decides
// nothing here.
func (p *Proxy) priceUnknownRefuses(mode, budgetID string) bool {
	switch mode {
	case "control", "hard", "soft", "fiat402":
		return budgetID != ""
	}
	return false
}

// handleToolsCall is the hot path — intercepts tool calls for budget enforcement.
func (p *Proxy) handleToolsCall(ctx context.Context, req *Request, tokenInfo *TokenInfo) (resp *Response, err error) {
	// Sign every action: not set up (see setupBlocks). dispatchAuthenticated
	// stops it first; this keeps a caller that reaches here directly honest.
	if p.setupBlocks() {
		return proofUnavailableResponse(req.ID, refusalTool(req)), nil
	}
	// Parse tool call
	tc, err := ParseToolCall(req.Params)
	if err != nil {
		unreadable := NewErrorResponse(req.ID, CodeInvalidParams, toolCallParseMessage(err))
		if p.signEveryAction {
			unreadable = p.refuseSigned(ctx, req, tokenInfo, MethodToolsCall, "", "INVALID_PARAMS", unreadable)
		}
		return unreadable, nil
	}

	// The enforcement mode this call is decided under: the process-wide mode,
	// or, when a host installed a resolver, the mode of this token's tenant.
	// Every read of the mode below, and the budget enforcer, use this one
	// answer (see enforcement_mode.go).
	ctx, mode := p.withResolvedEnforcementMode(ctx, tokenInfo.TenantID)

	// A tool whose name starts with the reserved argument-rules prefix is never
	// forwarded, whatever the token allows: see macaroon.ArgumentRulesScopePrefix.
	if macaroon.IsReservedToolName(tc.Name) {
		reserved := NewErrorResponse(req.ID, CodePolicyDenied, "tool name uses a reserved prefix and is not available")
		if p.signEveryAction {
			// The receipt names the prefix, never the rest of the word: it can
			// carry the nonce of a rule document.
			reserved = p.refuseSigned(ctx, req, tokenInfo, MethodToolsCall, macaroon.ArgumentRulesScopePrefix+"redacted", "RESERVED_TOOL_NAME", reserved)
		}
		return reserved, nil
	}

	// Every scope caveat must allow the tool. A later caveat can only narrow.
	// matchScope strips a tenant UUID prefix and ORs words inside one caveat.
	if !tokenInfo.AllowsTool(tc.Name) {
		return p.scopeDeniedResponse(ctx, req, tokenInfo, tc.Name), nil
	}

	// A tool listed under a name other than its own is not covered by a token's
	// money rules unless they name it: see tool_names_gate.go.
	if refused := p.renamedToolRefusal(ctx, req, tokenInfo, tc.Name); refused != nil {
		return refused, nil
	}

	// Argument rules: the token may allow the tool but limit what it can do.
	// This runs on the exact params that are forwarded below, before cost,
	// budget, events or any upstream call, so a refusal charges nothing and
	// never reaches the upstream. Every path to an upstream tools/call goes
	// through this function.
	if denied := tokenInfo.CheckToolArguments(tc.Name, req.Params); denied != nil {
		return p.argumentDeniedResponse(ctx, req, tokenInfo, tc.Name, denied), nil
	}

	// The price, for a tenant resolver that can say it does not know it (see
	// CheckedTenantCostResolver). It is asked before the call gate, so a call
	// refused for want of a price takes no spending-limit slot, asks no owner
	// for approval and has nothing to settle. A resolver without the method
	// is asked below, as it always was.
	var (
		cost         int64
		costResolved bool
	)
	if checked, ok := p.tenantCosts.(CheckedTenantCostResolver); ok && tokenInfo.TenantID != "" {
		price, priceErr := checked.ResolveForTenantChecked(ctx, tokenInfo.TenantID, tc.Name)
		cost, costResolved = price, true
		if priceErr != nil {
			if p.priceUnknownRefuses(mode, tokenInfo.BudgetID) {
				log.Error().Err(priceErr).Str("tool", tc.Name).Str("budgetId", tokenInfo.BudgetID).
					Str("mode", mode).
					Msg("tool price unavailable; refusing the call before any charge")
				return p.refuseUnavailableCheck(ctx, tokenInfo, tc.Name, unavailablePrice, NewErrorResponseWithData(req.ID, CodeInternalError, ToolPriceUnavailableMessage, map[string]interface{}{
					"error":     "tool_price_unavailable",
					"tool":      tc.Name,
					"retryable": true,
				})), nil
			}
			log.Warn().Err(priceErr).Str("tool", tc.Name).Str("budgetId", tokenInfo.BudgetID).
				Str("mode", mode).
				Int64("cost", price).
				Msg("tool price unavailable; observe mode never blocks, using the default price")
		}
	}

	// Sign every action: find a dead signer before anything is taken. Asked
	// once, here, before a spending-limit slot is reserved and before a
	// single-use approval is used up, so a signer that is down costs the owner
	// neither. The receipts themselves are made below. In Observe a dead signer
	// does not stop the call (allowReceiptFailureBlocks); the miss is reported
	// where the receipt fails, not here, because the receipt may still be made.
	if p.signEveryAction {
		if p.evidence == nil {
			if p.allowReceiptFailureBlocks(mode) {
				log.Error().Str("tool", tc.Name).Msg("sign every action is on but no evidence recorder is installed; refusing the call")
				return proofUnavailableResponse(req.ID, tc.Name), nil
			}
		} else if err := p.evidence.Preflight(ctx); err != nil {
			log.Error().Err(err).Str("tool", tc.Name).Msg("MCP evidence preflight failed")
			if p.allowReceiptFailureBlocks(mode) {
				return proofUnavailableResponse(req.ID, tc.Name), nil
			}
			log.Warn().Str("tool", tc.Name).Str("mode", mode).Msg("observe mode: passing the call on with no receipt")
		}
	}

	// Stateful controls (spending limits, owner approval): see call_gate.go.
	// After scope and argument rules, before cost, budget and the upstream.
	// Whatever happens next, the gate is told how the call ended.
	if p.gate != nil || tokenInfo.NeedsCallGate(tc.Name) {
		adm, refused := p.admitCall(ctx, req, tokenInfo, tc.Name)
		if refused != nil {
			return refused, nil
		}
		if adm != nil {
			// The record tells settle whether a failure came before or after
			// the request left (see dispatch_record.go).
			ctx, _ = withDispatchRecord(ctx)
			defer func() { adm.settle(ctx, resp, err) }()
			ctx = withGateDetail(ctx, adm.Detail)
		}
	}

	// Resolve cost (per-tenant if available, else global)
	if costResolved {
		// Resolved above, before the call gate.
	} else if p.tenantCosts != nil && tokenInfo.TenantID != "" {
		cost = p.tenantCosts.ResolveForTenant(tokenInfo.TenantID, tc.Name)
	} else {
		cost = p.costs.Resolve(tc.Name)
	}

	// Determine budget identity.
	// If the token has no budget_id caveat, this is an observe-only token —
	// skip budget enforcement entirely (don't fall back to tokenID).
	budgetID := tokenInfo.BudgetID

	// Tenant ID for event routing (multi-tenant)
	tenantID := tokenInfo.TenantID

	// Generate request ID for idempotency
	requestID := generateRequestID(budgetID, tc.Name, req.ID)
	if p.signEveryAction {
		// Sign every action: the call's identity is one the server makes for
		// this call alone. It names the hold, the amount kept if the commit
		// fails, the spend, the events and the receipt's request_id. The
		// JSON-RPC id is the client's and a different session may send the
		// same one, so it is never part of it (see newCallID).
		id, err := newCallID(callIDPrefix)
		if err != nil {
			log.Error().Err(err).Str("tool", receiptTool(tc.Name)).Msg("sign every action: no identity could be made for the call; refusing it before any charge")
			return proofUnavailableResponse(req.ID, tc.Name), nil
		}
		requestID = id
	}
	eventRequestID := requestID

	log.Debug().
		Str("tool", tc.Name).
		Int64("cost", cost).
		Str("enforcement", mode).
		Str("budgetId", budgetID).
		Str("tokenId", tokenInfo.TokenID).
		Msg("tool call intercepted")

	// Budget tracking and enforcement — three canonical modes:
	//   observe = deduct budget, never block (real spend tracking, allows overspend)
	//   control = deduct budget, enforce limits (block on exhaustion)
	//   charge  = L402 Lightning payments (handled separately)
	// Both observe and control always call Spend(). The only difference is what happens when budget runs out.
	// Each allow, deny, spend, delegation, or revocation outcome should emit receipt-ready events for Evidence Pack construction.
	// Legacy aliases (shadow→observe, soft/hard→control) accepted for backward compatibility.
	isObserve := mode == "observe" || mode == "shadow" || mode == "chargeback"
	isControl := mode == "control" || mode == "hard" || mode == "soft" || mode == "fiat402"
	var allowedEvidence *MCPEvidence
	// spendReceiptTried is set once the spend path has asked for its own
	// receipt (budget_authorized). Whatever came of it, no second allow
	// receipt is asked for below: a call has one.
	spendReceiptTried := false

	if (isObserve || isControl) && budgetID != "" && cost > 0 {
		// Auto-initialize budget from macaroon caveat if not yet initialized
		if tokenInfo.BudgetLimit > 0 {
			_ = p.budget.Initialize(ctx, budgetID, tokenInfo.BudgetLimit)
		}
		if p.evidence != nil && isControl {
			if err := p.evidence.Preflight(ctx); err != nil {
				log.Error().Err(err).Str("tool", tc.Name).Str("budgetId", budgetID).Msg("MCP evidence preflight failed")
				return NewErrorResponseWithData(req.ID, CodeInternalError, "Evidence proof unavailable", map[string]interface{}{
					"error": "proof_unavailable",
					"tool":  tc.Name,
				}), nil
			}
		}
		// Sign every action: a priced Control call is held, not debited. The
		// hold lapses by itself unless the receipt is made and the hold is
		// committed, so a call whose receipt fails ends uncharged even if
		// nothing later succeeds. A store with no hold is refused before
		// anything is taken. Room to keep the amount, should the commit fail
		// after the call was signed, is set aside first.
		var (
			holder     BudgetHolder
			holdExpiry time.Time
			slot       *pendingSlot
		)
		if p.signEveryAction && (isControl || p.every().holdObserve(ctx, p.budget)) {
			holder = holdOf(p.budget)
			if holder == nil {
				log.Error().Str("tool", receiptTool(tc.Name)).Str("budgetId", budgetID).Msg("sign every action: the budget store offers no hold; refusing the call before any charge")
				return proofUnavailableResponse(req.ID, tc.Name), nil
			}
			if slot = p.every().pending.reserve(kindUncharged); slot == nil {
				log.Error().Str("tool", receiptTool(tc.Name)).Msg("sign every action: the pending store is full; refusing the call before any charge")
				if isControl {
					return p.refuseUnavailableCheck(ctx, tokenInfo, tc.Name, unavailablePendingCapacity, proofUnavailableResponse(req.ID, tc.Name)), nil
				}
				return proofUnavailableResponse(req.ID, tc.Name), nil
			}
			defer slot.release()
		}
		var result *BudgetResult
		var err error
		if holder != nil {
			holdExpiry = time.Now().Add(p.every().cfg().HoldTTL)
			result, err = holder.Hold(ctx, budgetID, tc.Name, cost, requestID, p.every().cfg().HoldTTL)
		} else {
			result, err = p.budget.Spend(ctx, budgetID, tc.Name, cost, requestID)
		}
		eventRequestID = budgetEventRequestID(requestID, result)
		if err != nil {
			isBudgetExhausted := result != nil && (result.ErrorCode == "budget_exhausted" || result.ErrorCode == "insufficient_budget")

			if isBudgetExhausted {
				remaining := int64(0)
				if result != nil {
					remaining = result.Remaining
				}
				log.Warn().
					Str("tool", tc.Name).
					Int64("cost", cost).
					Int64("remaining", remaining).
					Str("budgetId", budgetID).
					Str("mode", mode).
					Msg("budget exhausted")

				eventType := EventBudgetExhaust
				if result.ErrorCode == "insufficient_budget" {
					eventType = EventBudgetInsufficient
				}
				p.events.Publish(Event{
					Type:      eventType,
					Timestamp: time.Now(),
					TokenID:   tokenInfo.TokenID,
					BudgetID:  budgetID,
					TenantID:  tenantID,
					Data: map[string]interface{}{
						"tool":         tc.Name,
						"cost":         cost,
						"remaining":    remaining,
						"budget_limit": tokenInfo.BudgetLimit,
						"request_id":   eventRequestID,
						"policy_mode":  normalizeEnforcementMode(mode),
						"decision":     "denied",
						"reason":       result.ErrorCode,
					},
				})

				if isObserve {
					// Observe: deducted but never blocks — allows overspend for receipt-backed usage/evidence tracking
					log.Info().Str("tool", tc.Name).Int64("remaining", remaining).
						Msg("observe mode: allowing overspend")
				} else {
					// Control: block the request, with issuer-anchored evidence when configured.
					errMsg := "Budget exhausted"
					errCode := result.ErrorCode
					if errCode == "insufficient_budget" {
						errMsg = fmt.Sprintf("Insufficient budget: tool '%s' costs %d credits, %d remaining", tc.Name, cost, remaining)
					}
					evidence, evidenceErr := p.recordMCPDecision(ctx, req, tokenInfo, tc.Name, MCPDecision{
						Decision:         "denied",
						DecisionReason:   errCode,
						PolicyMode:       normalizeEnforcementMode(mode),
						CostCredits:      cost,
						RemainingCredits: remaining,
						RequestID:        eventRequestID,
					})
					if evidenceErr != nil {
						log.Error().Err(evidenceErr).Str("tool", tc.Name).Str("budgetId", budgetID).Msg("MCP denial evidence recording failed")
						return NewErrorResponseWithData(req.ID, CodeInternalError, "Evidence proof unavailable", map[string]interface{}{
							"error": "proof_unavailable",
							"tool":  tc.Name,
						}), nil
					}
					data := map[string]interface{}{
						"error":             errCode,
						"tool":              tc.Name,
						"cost_credits":      cost,
						"remaining_credits": remaining,
						"token_id":          tokenInfo.TokenID,
						"budget_id":         budgetID,
					}
					mergeEvidenceData(data, evidence)
					return NewErrorResponseWithData(req.ID, CodeBudgetExhausted, errMsg, data), nil
				}
			} else {
				// Backend failure (Redis down, network error, etc.)
				log.Error().Err(err).
					Str("tool", tc.Name).
					Str("budgetId", budgetID).
					Str("failMode", p.config.Budget.FailMode).
					Msg("budget backend error")

				if isObserve || p.config.Budget.FailMode == "open" {
					// Observe always fails open; Control respects failMode config
					log.Warn().Str("tool", tc.Name).Str("mode", mode).
						Msg("allowing despite budget backend error")
				} else {
					return p.refuseUnavailableCheck(ctx, tokenInfo, tc.Name, unavailableBudgetBackend, NewErrorResponseWithData(req.ID, CodeInternalError, "Budget service unavailable", map[string]interface{}{
						"error":    "budget_backend_error",
						"tool":     tc.Name,
						"failMode": "closed",
					})), nil
				}
			}
		} else if result != nil {
			log.Info().
				Str("tool", tc.Name).
				Int64("cost", cost).
				Int64("remaining", result.Remaining).
				Str("budgetId", budgetID).
				Str("mode", mode).
				Msg("budget spent")

			publishSpend := func() {
				p.events.Publish(Event{
					Type:      EventBudgetSpend,
					Timestamp: time.Now(),
					TokenID:   tokenInfo.TokenID,
					BudgetID:  budgetID,
					TenantID:  tenantID,
					Data:      budgetSpendEventData(tc.Name, cost, result.Remaining, tokenInfo.BudgetLimit, eventRequestID, requestID, mode),
				})
			}
			if holder == nil {
				publishSpend()
			}

			var evidenceErr error
			spendReceiptTried = true
			recCtx := ctx
			var prep *pendingPreparation
			if holder != nil {
				recCtx = withHoldExpiry(ctx, holdExpiry)
				var perr error
				recCtx, prep, perr = p.every().preparePending(recCtx, PendingRecord{TokenID: tokenInfo.TokenID, TenantID: tenantID, BudgetID: budgetID, Method: MethodToolsCall, Tool: receiptTool(tc.Name), RequestID: requestID, Amount: cost})
				if perr != nil {
					p.every().release(ctx, holder, budgetID, requestID, cost)
					return proofUnavailableResponse(req.ID, tc.Name), nil
				}
			}
			allowedEvidence, evidenceErr = p.recordMCPDecision(recCtx, req, tokenInfo, tc.Name, MCPDecision{
				Decision:         "allowed",
				DecisionReason:   "budget_authorized",
				PolicyMode:       normalizeEnforcementMode(mode),
				CostCredits:      cost,
				RemainingCredits: result.Remaining,
				RequestID:        eventRequestID,
			})
			if evidenceErr != nil {
				log.Error().Err(evidenceErr).Str("tool", tc.Name).Str("budgetId", budgetID).Msg("MCP allowed evidence recording failed")
				if holder != nil && isControl {
					// The hold is let go of, on a context of its own. If that
					// fails too the hold lapses by itself: nothing more has to
					// succeed for this call to end uncharged. Nothing is sent.
					p.every().release(ctx, holder, budgetID, requestID, cost)
					return NewErrorResponseWithData(req.ID, CodeInternalError, "Evidence proof unavailable", map[string]interface{}{
						"error": "proof_unavailable",
						"tool":  tc.Name,
					}), nil
				}
				// Observe lets the call through with no receipt (today's
				// behaviour); with the switch on the miss is reported, once.
				if !isControl {
					if err := p.reportObserveMiss(recCtx, tokenInfo, MethodToolsCall, tc.Name, mode); err != nil {
						if holder != nil && !errors.Is(err, ErrPendingOutcomeUnknown) {
							p.every().release(ctx, holder, budgetID, requestID, cost)
						}
						return pendingFailureResponse(req.ID, tc.Name, err), nil
					}
				}
				if isControl {
					p.compensateMCPSpend(ctx, budgetID, eventRequestID, cost)
					return NewErrorResponseWithData(req.ID, CodeInternalError, "Evidence proof unavailable", map[string]interface{}{
						"error": "proof_unavailable",
						"tool":  tc.Name,
					}), nil
				}
			} else if holder != nil && prep != nil {
				stored, perr := p.every().keepPrepared(ctx, prep)
				if !stored {
					if errors.Is(perr, ErrPendingOutcomeUnknown) {
						return pendingFailureResponse(req.ID, tc.Name, perr), nil
					}
					p.every().release(ctx, holder, budgetID, requestID, cost)
					return proofUnavailableResponse(req.ID, tc.Name), nil
				}
				if perr != nil {
					log.Warn().Msg("signed call is durably pending settlement")
				} else {
					publishSpend()
				}
			} else if holder != nil {
				// Signed. Now the hold becomes the debit, and only then does
				// the call go out. If the commit fails the call still goes
				// ahead: it already has a signed allowed receipt, and refusing
				// it would make that receipt false. The hold lapses, so the
				// amount is never charged; it is kept, by tenant, key, tool
				// and minute, for reconciliation, once.
				if cerr := p.every().commit(ctx, holder, budgetID, requestID); cerr != nil {
					log.Error().Err(cerr).Str("tool", receiptTool(tc.Name)).Str("budgetId", budgetID).Msg("budget hold could not be committed after the call was signed; the call goes ahead uncharged")
					p.every().keepUncharged(slot, tokenInfo.TokenID, tenantID, budgetID, MethodToolsCall, tc.Name, requestID, cost)
				} else {
					publishSpend()
				}
			}
		}
	}

	// Sign every action: a call that every check passed and that no earlier
	// step signed (a free tool, a key with no budget, an Observe call over
	// budget) gets its one receipt now, before it is passed on. A call the
	// spend path signed already has its receipt (budget_authorized, with the
	// cost and the balance) and gets no second one.
	if p.signEveryAction && allowedEvidence == nil && !spendReceiptTried {
		ev, stopped := p.signAllowed(ctx, req, tokenInfo, MethodToolsCall, tc.Name, eventRequestID, mode)
		if stopped != nil {
			return stopped, nil
		}
		allowedEvidence = ev
	}

	// Publish tool call event
	p.events.Publish(Event{
		Type:      EventToolCall,
		Timestamp: time.Now(),
		TokenID:   tokenInfo.TokenID,
		BudgetID:  budgetID,
		TenantID:  tenantID,
		Data: map[string]interface{}{
			"tool":        tc.Name,
			"cost":        cost,
			"enforcement": normalizeEnforcementMode(mode),
			"request_id":  eventRequestID,
		},
	})

	// Forward to upstream
	timeout := 30 * time.Second
	if ucfg, ok := p.config.Upstreams[p.config.DefaultUpstream]; ok {
		timeout = ucfg.Timeout
	}

	// The caller's token stays with the proxy: it is removed from params. An
	// upstream that opts in (UpstreamConfig.ForwardCallerToken) gets only the
	// credential authenticate verified, which dispatchAuthenticated put on ctx
	// (see upstream_hygiene.go).
	resp, err = p.router.ForwardToolCallForTenant(ctx, tenantID, tc.Name, withoutMetaToken(req.Params), timeout)
	// What the router returned, as the gate must read it: the reply below is
	// rewritten into a JSON-RPC error on a transport failure, which would
	// otherwise look like an answer from the upstream.
	dispatchFrom(ctx).recordForward(resp, err)
	if err != nil {
		transportError := NewErrorResponse(req.ID, CodeUpstreamError, fmt.Sprintf("upstream error: %v", err))
		if p.unavailableCheckRefusalsEnabled() {
			// Admission already happened: deliver its handles, never a new denial.
			transportError = withEvidenceData(transportError, allowedEvidence)
		}
		return transportError, nil
	}

	// Track every tool call as a task
	// Use upstream task ID if available (SEP-1686), otherwise generate one per tool name
	{
		taskID := ""
		if resp.Result != nil {
			taskID = extractTaskID(resp.Result)
		}
		if taskID == "" {
			// Generate a stable task ID per tool — groups all calls to the same tool
			taskID = fmt.Sprintf("tool-%s-%x", tc.Name, fnv32(budgetID+tc.Name))
		}

		if resp.Result != nil {
			log.Info().
				Str("tool", tc.Name).
				Str("taskId", taskID).
				Str("budgetId", budgetID).
				Msg("MCP task detected in upstream response")

			// Extract status from _meta if available
			var meta struct {
				Meta struct {
					Status string `json:"status"`
				} `json:"_meta"`
			}
			if err := json.Unmarshal(resp.Result, &meta); err == nil && meta.Meta.Status != "" {
				p.taskTracker.UpdateStatus(taskID, meta.Meta.Status)
			}
		}

		p.taskTracker.RecordSpend(taskID, tokenInfo.TokenID, budgetID, tenantID, tc.Name, cost)

		// Get accumulated task state for the event (includes total_cost, attempts, timestamps)
		taskState := p.taskTracker.GetTaskSpend(taskID)
		taskData := map[string]interface{}{
			"task_id": taskID,
			"tool":    tc.Name,
			"cost":    cost, // This call's cost
		}
		if taskState != nil {
			taskData["tool_name"] = taskState.ToolName
			taskData["total_cost"] = taskState.TotalCost
			taskData["attempts"] = taskState.Attempts
			taskData["first_seen"] = taskState.FirstSeen.UTC().Format(time.RFC3339)
			taskData["last_seen"] = taskState.LastSeen.UTC().Format(time.RFC3339)
			taskData["status"] = taskState.Status
			taskData["token_id"] = taskState.TokenID
		}

		p.events.Publish(Event{
			Type:      EventTaskSpend,
			Timestamp: time.Now(),
			TokenID:   tokenInfo.TokenID,
			BudgetID:  budgetID,
			TenantID:  tenantID,
			Data:      taskData,
		})
	}

	// Rewrite the response ID to match the client's request ID
	resp.ID = req.ID
	attachMCPEvidenceToResponse(resp, allowedEvidence)
	return resp, nil
}

func (p *Proxy) recordMCPDecision(ctx context.Context, req *Request, tokenInfo *TokenInfo, toolName string, decision MCPDecision) (*MCPEvidence, error) {
	if p.evidence == nil {
		if p.signEveryAction {
			return nil, errNoRecorder
		}
		return nil, nil
	}
	toolName = receiptTool(toolName)
	decision.TenantID = tokenInfo.TenantID
	decision.TokenID = tokenInfo.TokenID
	decision.BudgetID = tokenInfo.BudgetID
	decision.BudgetSubjectID = tokenInfo.BudgetID
	decision.BudgetLimitCredits = tokenInfo.BudgetLimit
	decision.DelegationDepth = int64(tokenInfo.DelegationDepth)
	decision.DelegationBudget = tokenInfo.DelegationBudget
	decision.ParentTokenID = tokenInfo.ParentTokenID
	decision.Scope = tokenInfo.Scope
	decision.MCPMethod = MethodToolsCall
	decision.ToolName = toolName
	if p.signEveryAction && decision.RequestID == "" {
		// A refusal made before the call had an identity: its receipt gets one
		// of the server's own, never one built from the client's JSON-RPC id.
		id, err := newCallID(callIDPrefix)
		if err != nil {
			return nil, err
		}
		decision.RequestID = id
	}
	if d, ok := gateDetailFrom(ctx); ok && decision.Decision == "allowed" {
		// What a call gate recorded about an admitted call (a window total,
		// an approval id). Refusals carry their own detail.
		decision.ControlWindow, decision.ControlLimit = d.Window, d.Limit
		decision.ControlWindowTotal, decision.ApprovalID = d.WindowTotal, d.ApprovalID
		decision.ControlShared, decision.ControlSymbol = d.Shared, d.Symbol
	}
	decision.RouteOrTool = "mcp:tools/call:" + toolName
	decision.JSONRPCID = string(req.ID)
	if decision.PolicyMode == "" {
		decision.PolicyMode = normalizeEnforcementMode(p.enforcementModeFor(ctx, tokenInfo.TenantID))
	}
	if decision.BudgetNotEvaluated {
		// No balance was read: carry identity only, never a limit or balance.
		decision.BudgetLimitCredits = 0
		decision.CostCredits = 0
		decision.RemainingCredits = 0
		decision.RemainingBeforeCredits = 0
	} else {
		if decision.RemainingBeforeCredits == 0 && decision.Decision == "allowed" {
			decision.RemainingBeforeCredits = decision.RemainingCredits + decision.CostCredits
		}
		if decision.RemainingBeforeCredits == 0 && decision.Decision != "allowed" {
			decision.RemainingBeforeCredits = decision.RemainingCredits
		}
	}
	var ev *MCPEvidence
	var err error
	if p.signEveryAction {
		// Under the signing time limit, which is shorter than a hold's.
		ev, err = p.every().record(ctx, p.evidence, decision)
	} else {
		ev, err = p.evidence.RecordMCPDecision(ctx, decision)
	}
	if p.signEveryAction && err == nil && (ev == nil || ev.ReceiptID == "") {
		// An empty answer is not a receipt (see recordIdentifiedDecision).
		return nil, errNoReceipt
	}
	return ev, err
}

// scopeDeniedResponse refuses a verified token whose scope does not cover the
// tool. It never forwards the call and never reads or spends budget.
//
// Without an evidence recorder the reply is byte-for-byte the refusal main
// has always sent. With a recorder, the same code and message carry only the
// signed receipt handles as error data; recording failure stays a refusal
// (proof_unavailable). The signed decision is policy_denied with
// BudgetNotEvaluated, so the receipt claims no limit or balance.
func (p *Proxy) scopeDeniedResponse(ctx context.Context, req *Request, tokenInfo *TokenInfo, toolName string) *Response {
	message := fmt.Sprintf("tool %q not in scope %q", toolName, tokenInfo.Scope)
	return p.policyDeniedResponse(ctx, req, tokenInfo, toolName, message, nil, nil)
}

// argumentDeniedResponse refuses a call whose arguments are outside the
// token's argument rules. It uses the scope refusal shape (CodePolicyDenied,
// the same signed policy_denied decision) and adds the stable code
// TOOL_ARGUMENT_DENIED, the tool name and the field name as error data. It
// never echoes an argument value: the field name comes from the rule.
func (p *Proxy) argumentDeniedResponse(ctx context.Context, req *Request, tokenInfo *TokenInfo, toolName string, denied *argrules.Denial) *Response {
	message := fmt.Sprintf("tool %q: %s", toolName, denied.Reason)
	if denied.Field != "" {
		message = fmt.Sprintf("tool %q argument %q: %s", toolName, denied.Field, denied.Reason)
	}
	data := map[string]interface{}{
		"error": argrules.DenialCode,
		"tool":  toolName,
	}
	if denied.Field != "" {
		data["field"] = denied.Field
	}
	// What the closest shape allows, in words built from the rule. It goes to
	// the caller only (message and data); the signed receipt below is built
	// from detail and never sees it.
	if denied.Allowed != "" {
		message += ": " + denied.Allowed
		data["allowed"] = denied.Allowed
	}
	// The signed receipt says this was an argument-rule refusal and which
	// field blocked it, so it cannot be mistaken for a scope refusal. The
	// rule documents are identified by hash; no argument value is recorded.
	detail := &argumentDenialDetail{
		code:      argrules.DenialCode,
		field:     denied.Field,
		rulesHash: tokenInfo.ArgumentRulesSHA256(),
	}
	return p.policyDeniedResponse(ctx, req, tokenInfo, toolName, message, data, detail)
}

// argumentDenialDetail is what an argument-rule refusal adds to the signed
// decision. A scope refusal passes nil, which leaves the decision as it was.
type argumentDenialDetail struct {
	code      string
	field     string
	rulesHash string
}

// policyDeniedResponse is the shared refusal. extra is merged into the error
// data next to any evidence handles; nil extra and no recorder gives a reply
// with no data. detail is non-nil only for an argument-rule refusal.
func (p *Proxy) policyDeniedResponse(ctx context.Context, req *Request, tokenInfo *TokenInfo, toolName, message string, extra map[string]interface{}, detail *argumentDenialDetail) *Response {
	if p.evidence == nil && !p.signEveryAction {
		if len(extra) == 0 {
			return NewErrorResponse(req.ID, CodePolicyDenied, message)
		}
		return NewErrorResponseWithData(req.ID, CodePolicyDenied, message, extra)
	}
	proofUnavailable := func(err error, stage string) *Response {
		log.Error().Err(err).Str("tool", toolName).Str("stage", stage).Msg("MCP policy denial evidence unavailable")
		return NewErrorResponseWithData(req.ID, CodeInternalError, "Evidence proof unavailable", map[string]interface{}{
			"error": "proof_unavailable",
			"tool":  toolName,
		})
	}
	if p.evidence == nil {
		return proofUnavailable(errNoRecorder, "no_recorder")
	}
	if err := p.evidence.Preflight(ctx); err != nil {
		return proofUnavailable(err, "preflight")
	}
	// Same helper the HTTP proxy uses. Signing takes no lock; the recorder's
	// archive append is serialized by the archive itself. Do not call this
	// while holding a lock of your own.
	recorded, err := denial.RecordSignedDenial(ctx, mcpScopeDenialRecorder{
		proxy:     p,
		req:       req,
		tokenInfo: tokenInfo,
		detail:    detail,
	}, "policy_denied", tokenInfo.TokenID, toolName, time.Now().UTC())
	if err != nil {
		return proofUnavailable(err, "record")
	}
	evidence, _ := recorded.(*MCPEvidence)
	if evidence == nil || evidence.ReceiptID == "" {
		return proofUnavailable(errors.New("recorder returned no receipt"), "record")
	}
	data := map[string]interface{}{}
	for k, v := range extra {
		data[k] = v
	}
	mergeEvidenceData(data, evidence)
	return NewErrorResponseWithData(req.ID, CodePolicyDenied, message, data)
}

// mcpScopeDenialRecorder adapts the MCP evidence recorder to the shared helper.
// It takes no lock of its own.
type mcpScopeDenialRecorder struct {
	proxy     *Proxy
	req       *Request
	tokenInfo *TokenInfo
	detail    *argumentDenialDetail
}

func (m mcpScopeDenialRecorder) RecordSignedDenial(ctx context.Context, in denial.SignedDenial) (any, error) {
	toolName := in.Target
	if toolName == "" && m.req != nil {
		if tc, err := ParseToolCall(m.req.Params); err == nil {
			toolName = tc.Name
		}
	}
	decision := MCPDecision{
		Decision:           "denied",
		DecisionReason:     in.ReasonCode,
		BudgetNotEvaluated: true,
	}
	if m.detail != nil {
		decision.DenialCode = m.detail.code
		decision.ArgumentField = m.detail.field
		decision.ArgumentRulesSHA256 = m.detail.rulesHash
	}
	evidence, err := m.proxy.recordMCPDecision(ctx, m.req, m.tokenInfo, toolName, decision)
	if err != nil {
		return nil, err
	}
	return evidence, nil
}

func budgetSpendEventData(tool string, cost, remaining, budgetLimit int64, requestIdentity, callerRequestID, mode string) map[string]interface{} {
	return map[string]interface{}{
		"tool":              tool,
		"cost":              cost,
		"remaining":         remaining,
		"budget_limit":      budgetLimit,
		"request_id":        requestIdentity,
		"caller_request_id": callerRequestID,
		"policy_mode":       normalizeEnforcementMode(mode),
	}
}

func budgetEventRequestID(callerRequestID string, result *BudgetResult) string {
	if result != nil && result.RequestID != "" {
		return result.RequestID
	}
	return callerRequestID
}

// compensateMCPSpend gives back the debit of a call that is stopped because its
// receipt failed, with the switch off: the budget's Compensate, on the
// request's context, when the budget has one, as it always was.
//
// With the switch on a priced Control call is never debited before its receipt
// (it is held: see handleToolsCall), so there is nothing here to give back.
func (p *Proxy) compensateMCPSpend(ctx context.Context, budgetID, requestID string, cost int64) {
	if budgetID == "" || requestID == "" || cost <= 0 || p.signEveryAction {
		return
	}
	if compensator, ok := p.budget.(BudgetCompensator); ok {
		if err := compensator.Compensate(ctx, budgetID, requestID, cost); err != nil {
			log.Error().Err(err).Str("budgetId", budgetID).Str("requestId", requestID).Msg("failed to compensate MCP debit after evidence failure")
		}
	}
}

func mergeEvidenceData(data map[string]interface{}, evidence *MCPEvidence) {
	if data == nil || evidence == nil {
		return
	}
	if evidence.ReceiptID != "" {
		data["receipt_id"] = evidence.ReceiptID
	}
	if evidence.ReceiptHash != "" {
		data["receipt_hash"] = evidence.ReceiptHash
	}
	if evidence.EvidencePackID != "" {
		data["evidence_pack_id"] = evidence.EvidencePackID
	}
	if evidence.EvidenceURL != "" {
		data["evidence_url"] = evidence.EvidenceURL
	}
	if evidence.VerifyURL != "" {
		data["verify_url"] = evidence.VerifyURL
	}
	if evidence.JWKSURL != "" {
		data["jwks_url"] = evidence.JWKSURL
	}
}

func attachMCPEvidenceToResponse(resp *Response, evidence *MCPEvidence) {
	if resp == nil || evidence == nil || len(resp.Result) == 0 {
		return
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return
	}
	meta, _ := result["_meta"].(map[string]interface{})
	if meta == nil {
		meta = map[string]interface{}{}
	}
	evidenceData := map[string]interface{}{}
	mergeEvidenceData(evidenceData, evidence)
	if len(evidenceData) == 0 {
		return
	}
	meta["satgate_evidence"] = evidenceData
	result["_meta"] = meta
	if encoded, err := json.Marshal(result); err == nil {
		resp.Result = encoded
	}
}

// handleDelegate processes satgate/delegate requests.
func (p *Proxy) handleDelegate(ctx context.Context, req *Request, tokenInfo *TokenInfo) (*Response, error) {
	if p.delegator == nil {
		return NewErrorResponse(req.ID, CodeMethodNotFound, "delegation requires auth.mode=header"), nil
	}
	return p.delegator.HandleDelegate(ctx, req, tokenInfo)
}

// forwardToDefault forwards an unrecognized method to the default upstream.
func (p *Proxy) forwardToDefault(ctx context.Context, req *Request) (*Response, error) {
	// A tools/call never takes this path: handleRequest sends it through
	// authentication, scope and argument rules first. Refuse it here too, so
	// no later edit to the dispatch can forward one unchecked.
	if IsToolCall(req.Method) {
		return NewErrorResponse(req.ID, CodePolicyDenied, "tools/call must go through the governed path"), nil
	}
	timeout := 30 * time.Second

	// As for tools/call: the caller's token is not passed on (see
	// upstream_hygiene.go). An upstream that opted in gets the credential
	// verified for this request: the one authenticate verified (forwardGoverned),
	// else the connection's own credential once it verifies here. Never a
	// value read from params.
	fwdCtx := p.verifyConnectionCredential(ctx)
	resp, err := p.upstream.ForwardRequest(fwdCtx, req.Method, withoutMetaToken(req.Params), timeout)
	if err != nil {
		return NewErrorResponse(req.ID, CodeUpstreamError, fmt.Sprintf("upstream: %v", err)), nil
	}

	resp.ID = req.ID
	return resp, nil
}

// generateRequestID creates an idempotency key from token + tool + request ID.
// matchScope checks if a tool name matches a comma-separated scope string.
// Supports exact match and wildcard prefix (e.g., "db:*" matches "db:query").
// Handles tenant-prefixed scopes (e.g., "tenant-uuid:*" or "tenant-uuid:mcp:*")
// by stripping the tenant prefix before matching.
func matchScope(scope, toolName string) bool {
	// No scope, wildcard included, allows a tool whose name starts with the
	// reserved argument-rules prefix. An older runtime allows a tool whose name
	// equals a rule word; this one cannot, in the scope check itself.
	if macaroon.IsReservedToolName(toolName) {
		return false
	}
	for _, s := range strings.Split(scope, ",") {
		s = strings.TrimSpace(s)

		// Strip tenant UUID prefix if present (format: "uuid:scope")
		// UUIDs are 36 chars with hyphens (8-4-4-4-12)
		if len(s) > 37 && s[36] == ':' && s[8] == '-' && s[13] == '-' {
			s = s[37:]
		}

		if s == "*" || s == "api:*" || s == "mcp:*" {
			return true
		}
		if strings.HasSuffix(s, ":*") {
			prefix := strings.TrimSuffix(s, ":*")
			if strings.HasPrefix(toolName, prefix+":") || strings.HasPrefix(toolName, prefix+"_") {
				return true
			}
		}
		if s == toolName {
			return true
		}
	}
	return false
}

func generateRequestID(tokenID, toolName string, reqID json.RawMessage) string {
	h := sha256.New()
	h.Write([]byte(tokenID))
	h.Write([]byte(toolName))
	h.Write(reqID)
	return hex.EncodeToString(h.Sum(nil))[:16]
}

// hashToken creates a short hash of a token for use as a budget key.
func hashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// handleBudgetCheck returns the current token's budget status.
// This allows economically-aware agents to check their balance before making expensive calls.
func (p *Proxy) handleBudgetCheck(ctx context.Context, req *Request, tokenInfo *TokenInfo) (*Response, error) {

	response := map[string]interface{}{
		"token_id":  tokenInfo.TokenID,
		"tenant_id": tokenInfo.TenantID,
		"scope":     tokenInfo.Scope,
	}

	// If budget enforcement is available, get remaining budget
	if p.budget != nil && tokenInfo.BudgetID != "" {
		remaining, err := p.budget.Remaining(ctx, tokenInfo.BudgetID)
		if err == nil {
			response["budget_id"] = tokenInfo.BudgetID
			response["budget_remaining_credits"] = remaining
		}
	}

	result, err := json.Marshal(response)
	if err != nil {
		return nil, err
	}

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}, nil
}

// TokenInfoFromContext extracts the TokenInfo from context (set during authenticated requests).
func TokenInfoFromContext(ctx context.Context) *TokenInfo {
	if v, ok := ctx.Value(CtxTokenInfo).(*TokenInfo); ok {
		return v
	}
	return nil
}

// TaskTracker returns the proxy's task tracker for MCP task-level cost queries.
func (p *Proxy) TaskTracker() *TaskTracker { return p.taskTracker }

// extractTaskID parses the task ID from an MCP task response.
// MCP Tasks (SEP-1686) include the task ID in result._meta.taskId.
func extractTaskID(result json.RawMessage) string {
	if len(result) == 0 {
		return ""
	}
	var envelope struct {
		Meta struct {
			TaskID string `json:"taskId"`
		} `json:"_meta"`
	}
	if err := json.Unmarshal(result, &envelope); err != nil {
		return ""
	}
	return envelope.Meta.TaskID
}

// fnv32 generates a simple 32-bit hash for creating stable synthetic IDs.
func fnv32(s string) uint32 {
	var h uint32 = 2166136261
	for i := 0; i < len(s); i++ {
		h ^= uint32(s[i])
		h *= 16777619
	}
	return h
}
