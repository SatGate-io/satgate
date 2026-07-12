package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
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
	budget            BudgetEnforcer
	costs             CostResolver
	tenantCosts       TenantCostResolver // optional, per-tenant cost resolution
	auth              Authenticator
	delegator         *Delegator
	events            EventPublisher
	evidence          EvidenceRecorder
	revocation        RevocationChecker // optional, checks if token is revoked
	toolsListEnricher ToolsListEnricher // optional, enriches tools/list with cost metadata
	taskTracker       *TaskTracker      // MCP task-level cost aggregation (SEP-1686)
	tokenID           string            // default session token (from config or first auth)
	rootToken         string            // auto-minted root token (for delegation demos)
}

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
		auth:        auth,
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
	p.auth = a
}

// SetEventPublisher replaces the event publisher (e.g., with enterprise Redis pub/sub).
func (p *Proxy) SetEventPublisher(ep EventPublisher) {
	p.events = ep
	if p.delegator != nil {
		p.delegator.SetEventPublisher(ep)
	}
}

// SetEvidenceRecorder wires signed evidence recording for governed MCP decisions.
func (p *Proxy) SetEvidenceRecorder(rec EvidenceRecorder) {
	p.evidence = rec
}

// SetRevocationChecker sets the revocation checker for token validation.
func (p *Proxy) SetRevocationChecker(rc RevocationChecker) {
	p.revocation = rc
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

		// Handle the request
		resp, err := p.handleRequest(ctx, req)
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

// handleRequest dispatches a JSON-RPC request to the appropriate handler.
func (p *Proxy) handleRequest(ctx context.Context, req *Request) (*Response, error) {
	switch req.Method {
	case MethodInitialize:
		return p.handleInitialize(req)

	case MethodInitialized:
		return nil, nil

	case MethodPing:
		return p.handlePing(req), nil

	case MethodToolsList:
		return p.handleToolsListWithCtx(ctx, req)

	case MethodToolsCall, MethodSatGateDelegate, MethodSatGateBudget, "budget/check":
		// These methods require authentication.
		tokenInfo, authErr := p.authenticate(ctx, req)
		if authErr != nil {
			log.Warn().Err(authErr).Str("method", req.Method).Msg("authentication failed")
			p.events.Publish(Event{
				Type:      EventAuthFailure,
				Timestamp: time.Now(),
				Data: map[string]interface{}{
					"method": req.Method,
					"error":  authErr.Error(),
				},
			})
			return NewErrorResponse(req.ID, CodePolicyDenied, authErr.Error()), nil
		}
		// Inject tenant ID into context for downstream use (multi-tenant budget routing)
		if tokenInfo.TenantID != "" {
			ctx = context.WithValue(ctx, CtxTenantID, tokenInfo.TenantID)
		}
		ctx = context.WithValue(ctx, CtxTokenInfo, tokenInfo)
		// Check revocation (after tenant is in context)
		if p.revocation != nil && tokenInfo.BudgetID != "" {
			if p.revocation.IsRevoked(ctx, tokenInfo.BudgetID) {
				return NewErrorResponse(req.ID, CodePolicyDenied, "token revoked"), nil
			}
		}
		switch req.Method {
		case MethodToolsCall:
			return p.handleToolsCall(ctx, req, tokenInfo)
		case MethodSatGateDelegate:
			return p.handleDelegate(ctx, req, tokenInfo)
		case MethodSatGateBudget:
			return HandleBudget(ctx, req, p.budget, tokenInfo)
		}
		return nil, fmt.Errorf("unreachable")

	case MethodCancelled:
		log.Debug().RawJSON("params", req.Params).Msg("cancellation received")
		return nil, nil

	default:
		if IsNotification(req.Method) {
			return nil, nil
		}
		return p.forwardToDefault(ctx, req)
	}
}

// authenticate extracts token from the request's _meta field or falls back to config.
// If a token is provided but fails verification, authentication fails (fail-closed).
func (p *Proxy) authenticate(ctx context.Context, req *Request) (*TokenInfo, error) {
	// Try to extract token from params._meta.token (MCP convention for metadata)
	token := extractMetaToken(req.Params)

	if token != "" {
		info, err := p.auth.Verify(ctx, token)
		if err != nil {
			// Fail-closed: if a token was provided but is invalid, deny the request.
			// Never silently fall back to default identity on auth failure.
			return nil, fmt.Errorf("authentication failed: %w", err)
		}
		return info, nil
	}

	// No token provided.
	// In header mode, a missing token is an auth bypass — deny it.
	if p.config.Auth.Mode == "header" {
		return nil, fmt.Errorf("authentication required: no token provided")
	}

	// For auth modes "none" and "config", fall back to default identity.
	return &TokenInfo{
		TokenID:  p.tokenID,
		BudgetID: p.tokenID,
		Scope:    "*",
	}, nil
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

// handleToolsCall is the hot path — intercepts tool calls for budget enforcement.
func (p *Proxy) handleToolsCall(ctx context.Context, req *Request, tokenInfo *TokenInfo) (*Response, error) {
	// Parse tool call
	tc, err := ParseToolCall(req.Params)
	if err != nil {
		return NewErrorResponse(req.ID, CodeInvalidParams, err.Error()), nil
	}

	// Scope enforcement: if token has a restricted scope, check the tool is allowed
	// Note: scope may be tenant-prefixed (e.g., "tenant-uuid:*") — matchScope handles stripping
	if tokenInfo.Scope != "" && tokenInfo.Scope != "*" && tokenInfo.Scope != "api:*" && tokenInfo.Scope != "mcp:*" {
		if !matchScope(tokenInfo.Scope, tc.Name) {
			return NewErrorResponse(req.ID, CodePolicyDenied,
				fmt.Sprintf("tool %q not in scope %q", tc.Name, tokenInfo.Scope)), nil
		}
	}

	// Resolve cost (per-tenant if available, else global)
	var cost int64
	if p.tenantCosts != nil && tokenInfo.TenantID != "" {
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
	eventRequestID := requestID

	log.Debug().
		Str("tool", tc.Name).
		Int64("cost", cost).
		Str("enforcement", p.config.Enforcement.Mode).
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
	mode := p.config.Enforcement.Mode
	isObserve := mode == "observe" || mode == "shadow" || mode == "chargeback"
	isControl := mode == "control" || mode == "hard" || mode == "soft" || mode == "fiat402"
	var allowedEvidence *MCPEvidence

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
		result, err := p.budget.Spend(ctx, budgetID, tc.Name, cost, requestID)
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
					Str("mode", p.config.Enforcement.Mode).
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
					log.Warn().Str("tool", tc.Name).Str("mode", p.config.Enforcement.Mode).
						Msg("allowing despite budget backend error")
				} else {
					return NewErrorResponseWithData(req.ID, CodeInternalError, "Budget service unavailable", map[string]interface{}{
						"error":    "budget_backend_error",
						"tool":     tc.Name,
						"failMode": "closed",
					}), nil
				}
			}
		} else if result != nil {
			log.Info().
				Str("tool", tc.Name).
				Int64("cost", cost).
				Int64("remaining", result.Remaining).
				Str("budgetId", budgetID).
				Str("mode", p.config.Enforcement.Mode).
				Msg("budget spent")

			p.events.Publish(Event{
				Type:      EventBudgetSpend,
				Timestamp: time.Now(),
				TokenID:   tokenInfo.TokenID,
				BudgetID:  budgetID,
				TenantID:  tenantID,
				Data:      budgetSpendEventData(tc.Name, cost, result.Remaining, tokenInfo.BudgetLimit, eventRequestID, requestID, mode),
			})

			var evidenceErr error
			allowedEvidence, evidenceErr = p.recordMCPDecision(ctx, req, tokenInfo, tc.Name, MCPDecision{
				Decision:         "allowed",
				DecisionReason:   "budget_authorized",
				PolicyMode:       normalizeEnforcementMode(mode),
				CostCredits:      cost,
				RemainingCredits: result.Remaining,
				RequestID:        eventRequestID,
			})
			if evidenceErr != nil {
				log.Error().Err(evidenceErr).Str("tool", tc.Name).Str("budgetId", budgetID).Msg("MCP allowed evidence recording failed")
				if isControl {
					p.compensateMCPSpend(ctx, budgetID, eventRequestID, cost)
					return NewErrorResponseWithData(req.ID, CodeInternalError, "Evidence proof unavailable", map[string]interface{}{
						"error": "proof_unavailable",
						"tool":  tc.Name,
					}), nil
				}
			}
		}
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
			"enforcement": p.config.Enforcement.Mode,
			"request_id":  eventRequestID,
		},
	})

	// Forward to upstream
	timeout := 30 * time.Second
	if ucfg, ok := p.config.Upstreams[p.config.DefaultUpstream]; ok {
		timeout = ucfg.Timeout
	}

	resp, err := p.router.ForwardToolCallForTenant(ctx, tenantID, tc.Name, req.Params, timeout)
	if err != nil {
		return NewErrorResponse(req.ID, CodeUpstreamError, fmt.Sprintf("upstream error: %v", err)), nil
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
		return nil, nil
	}
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
	decision.RouteOrTool = "mcp:tools/call:" + toolName
	decision.JSONRPCID = string(req.ID)
	if decision.PolicyMode == "" {
		decision.PolicyMode = normalizeEnforcementMode(p.config.Enforcement.Mode)
	}
	if decision.RemainingBeforeCredits == 0 && decision.Decision == "allowed" {
		decision.RemainingBeforeCredits = decision.RemainingCredits + decision.CostCredits
	}
	if decision.RemainingBeforeCredits == 0 && decision.Decision != "allowed" {
		decision.RemainingBeforeCredits = decision.RemainingCredits
	}
	return p.evidence.RecordMCPDecision(ctx, decision)
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

func (p *Proxy) compensateMCPSpend(ctx context.Context, budgetID, requestID string, cost int64) {
	compensator, ok := p.budget.(BudgetCompensator)
	if !ok || budgetID == "" || requestID == "" || cost <= 0 {
		return
	}
	if err := compensator.Compensate(ctx, budgetID, requestID, cost); err != nil {
		log.Error().Err(err).Str("budgetId", budgetID).Str("requestId", requestID).Msg("failed to compensate MCP debit after evidence failure")
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
	timeout := 30 * time.Second

	resp, err := p.upstream.ForwardRequest(ctx, req.Method, req.Params, timeout)
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
