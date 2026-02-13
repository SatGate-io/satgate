package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"time"

	"github.com/rs/zerolog/log"
)

// Proxy is the MCP proxy gateway. It sits between an MCP client and one or
// more upstream MCP servers, enforcing budgets and attributing costs.
type Proxy struct {
	config    *Config
	client    Transport       // client-facing transport
	upstream  *UpstreamManager
	budget    BudgetEnforcer
	costs     CostResolver
	auth      Authenticator
	delegator *Delegator
	events    EventPublisher
	tokenID   string // default session token (from config or first auth)
	rootToken string // auto-minted root token (for delegation demos)
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
	upstream := NewUpstreamManager(cfg.Upstreams, cfg.Routing, cfg.DefaultUpstream)

	p := &Proxy{
		config:   cfg,
		upstream: upstream,
		budget:   budget,
		costs:    costs,
		auth:     auth,
		events:   &NoOpPublisher{},
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
				rootMac, mintErr := macAuth.Service.Mint("api:*", time.Now().Add(24*time.Hour))
				if mintErr != nil {
					return nil, fmt.Errorf("auto-mint root token: %w", mintErr)
				}
				rootToken := macAuth.Service.Encode(rootMac)
				// Must match MacaroonAuthenticator.Verify — identifier + signature
				tokenID = hashToken(rootMac.Identifier + rootMac.Signature)
				p.rootToken = rootToken
				log.Info().Str("tokenId", tokenID).Msg("auto-minted root token")
				fmt.Fprintf(os.Stderr, "ROOT_TOKEN=%s\n", rootToken)
				fmt.Fprintf(os.Stderr, "TOKEN_ID=%s\n", tokenID)
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

// SetCostResolver replaces the cost resolver (e.g., with enterprise per-tenant costs).
func (p *Proxy) SetCostResolver(r CostResolver) {
	p.costs = r
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
		return p.handleToolsList(req)

	case MethodToolsCall, MethodSatGateDelegate, MethodSatGateBudget:
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
func (p *Proxy) handleToolsList(req *Request) (*Response, error) {
	tools := p.upstream.AllTools()

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

	// Resolve cost
	cost := p.costs.Resolve(tc.Name)

	// Determine budget identity
	budgetID := tokenInfo.BudgetID
	if budgetID == "" {
		budgetID = p.tokenID
	}

	// Generate request ID for idempotency
	requestID := generateRequestID(budgetID, tc.Name, req.ID)

	log.Debug().
		Str("tool", tc.Name).
		Int64("cost", cost).
		Str("enforcement", p.config.Enforcement.Mode).
		Str("budgetId", budgetID).
		Str("tokenId", tokenInfo.TokenID).
		Msg("tool call intercepted")

	// Enforce budget (unless shadow mode)
	if p.config.Enforcement.Mode != "shadow" && budgetID != "" && cost > 0 {
		result, err := p.budget.Spend(ctx, budgetID, tc.Name, cost, requestID)
		if err != nil {
			// Distinguish budget exhaustion (result has ErrorCode) from backend failures.
			isBudgetExhausted := result != nil && result.ErrorCode == "budget_exhausted"

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
					Msg("budget exhausted")

				p.events.Publish(Event{
					Type:      EventBudgetExhaust,
					Timestamp: time.Now(),
					TokenID:   tokenInfo.TokenID,
					BudgetID:  budgetID,
					Data: map[string]interface{}{
						"tool":      tc.Name,
						"cost":      cost,
						"remaining": remaining,
					},
				})

				if p.config.Enforcement.Mode == "hard" {
					return NewErrorResponseWithData(req.ID, CodeBudgetExhausted, "Budget exhausted", map[string]interface{}{
						"error":             "budget_exhausted",
						"tool":              tc.Name,
						"cost_credits":      cost,
						"remaining_credits": remaining,
						"token_id":          tokenInfo.TokenID,
						"budget_id":         budgetID,
					}), nil
				}
				// Soft mode: warn but continue
				log.Warn().Str("tool", tc.Name).Msg("soft enforcement: allowing despite budget exhaustion")
			} else {
				// Backend failure (Redis down, network error, etc.)
				log.Error().Err(err).
					Str("tool", tc.Name).
					Str("budgetId", budgetID).
					Str("failMode", p.config.Budget.FailMode).
					Msg("budget backend error")

				if p.config.Budget.FailMode != "open" {
					// fail-closed (default): deny on backend error
					return NewErrorResponseWithData(req.ID, CodeInternalError, "Budget service unavailable", map[string]interface{}{
						"error":    "budget_backend_error",
						"tool":     tc.Name,
						"failMode": "closed",
					}), nil
				}
				// fail-open: log and allow through
				log.Warn().Str("tool", tc.Name).Msg("fail-open: allowing despite budget backend error")
			}
		} else if result != nil {
			log.Info().
				Str("tool", tc.Name).
				Int64("cost", cost).
				Int64("remaining", result.Remaining).
				Str("budgetId", budgetID).
				Msg("budget spent")

			p.events.Publish(Event{
				Type:      EventBudgetSpend,
				Timestamp: time.Now(),
				TokenID:   tokenInfo.TokenID,
				BudgetID:  budgetID,
				Data: map[string]interface{}{
					"tool":      tc.Name,
					"cost":      cost,
					"remaining": result.Remaining,
				},
			})
		}
	} else if p.config.Enforcement.Mode == "shadow" && cost > 0 {
		log.Info().
			Str("tool", tc.Name).
			Int64("cost", cost).
			Msg("shadow mode: would have charged")
	}

	// Publish tool call event
	p.events.Publish(Event{
		Type:      EventToolCall,
		Timestamp: time.Now(),
		TokenID:   tokenInfo.TokenID,
		BudgetID:  budgetID,
		Data: map[string]interface{}{
			"tool":        tc.Name,
			"cost":        cost,
			"enforcement": p.config.Enforcement.Mode,
		},
	})

	// Forward to upstream
	timeout := 30 * time.Second
	if ucfg, ok := p.config.Upstreams[p.config.DefaultUpstream]; ok {
		timeout = ucfg.Timeout
	}

	resp, err := p.upstream.ForwardToolCall(ctx, tc.Name, req.Params, timeout)
	if err != nil {
		return NewErrorResponse(req.ID, CodeUpstreamError, fmt.Sprintf("upstream error: %v", err)), nil
	}

	// Rewrite the response ID to match the client's request ID
	resp.ID = req.ID
	return resp, nil
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
