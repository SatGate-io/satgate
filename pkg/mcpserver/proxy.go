package mcpserver

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
)

// Proxy is the MCP proxy gateway. It sits between an MCP client and one or
// more upstream MCP servers, enforcing budgets and attributing costs.
type Proxy struct {
	config   *Config
	client   Transport       // client-facing transport
	upstream *UpstreamManager
	budget   BudgetEnforcer
	costs    CostResolver
	tokenID  string // current session token (from config or auth)
}

// New creates a new MCP proxy from configuration.
func New(cfg *Config) (*Proxy, error) {
	// Build cost resolver
	costs := NewStaticCostResolver(cfg.Tools.Costs, cfg.Tools.DefaultCost)

	// Build budget enforcer
	budget := NewInMemoryBudgetEnforcer()

	// Build upstream manager
	upstream := NewUpstreamManager(cfg.Upstreams, cfg.Routing, cfg.DefaultUpstream)

	p := &Proxy{
		config:   cfg,
		upstream: upstream,
		budget:   budget,
		costs:    costs,
	}

	// Initialize budget if configured
	if cfg.Budget.Limit > 0 {
		tokenID := "default"
		if cfg.Auth.Token != "" {
			tokenID = hashToken(cfg.Auth.Token)
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
}

// SetCostResolver replaces the cost resolver (e.g., with enterprise per-tenant costs).
func (p *Proxy) SetCostResolver(r CostResolver) {
	p.costs = r
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
			// Response from client — shouldn't happen in normal flow
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
		// Notification — no response
		return nil, nil

	case MethodPing:
		return p.handlePing(req), nil

	case MethodToolsList:
		return p.handleToolsList(req)

	case MethodToolsCall:
		return p.handleToolsCall(ctx, req)

	case MethodCancelled:
		// TODO: propagate cancellation to upstream
		log.Debug().RawJSON("params", req.Params).Msg("cancellation received")
		return nil, nil

	default:
		// Forward unknown methods to default upstream
		if IsNotification(req.Method) {
			return nil, nil
		}
		return p.forwardToDefault(ctx, req)
	}
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
func (p *Proxy) handleToolsCall(ctx context.Context, req *Request) (*Response, error) {
	// Parse tool call
	tc, err := ParseToolCall(req.Params)
	if err != nil {
		return NewErrorResponse(req.ID, CodeInvalidParams, err.Error()), nil
	}

	// Resolve cost
	cost := p.costs.Resolve(tc.Name)

	// Generate request ID for idempotency
	requestID := generateRequestID(p.tokenID, tc.Name, req.ID)

	log.Debug().
		Str("tool", tc.Name).
		Int64("cost", cost).
		Str("enforcement", p.config.Enforcement.Mode).
		Str("token", p.tokenID).
		Msg("tool call intercepted")

	// Enforce budget (unless shadow mode)
	if p.config.Enforcement.Mode != "shadow" && p.tokenID != "" && cost > 0 {
		result, err := p.budget.Spend(ctx, p.tokenID, tc.Name, cost, requestID)
		if err != nil {
			// Budget exhausted
			log.Warn().
				Str("tool", tc.Name).
				Int64("cost", cost).
				Int64("remaining", result.Remaining).
				Str("token", p.tokenID).
				Msg("budget exhausted — denying tool call")

			if p.config.Enforcement.Mode == "hard" {
				return NewErrorResponseWithData(req.ID, CodeBudgetExhausted, "Budget exhausted", map[string]interface{}{
					"error":             "budget_exhausted",
					"tool":              tc.Name,
					"cost_credits":      cost,
					"remaining_credits": result.Remaining,
					"token_id":          p.tokenID,
				}), nil
			}
			// Soft mode: warn but continue
			log.Warn().Str("tool", tc.Name).Msg("soft enforcement: allowing despite budget exhaustion")
		} else if result != nil {
			log.Info().
				Str("tool", tc.Name).
				Int64("cost", cost).
				Int64("remaining", result.Remaining).
				Msg("budget spent")
		}
	} else if p.config.Enforcement.Mode == "shadow" && cost > 0 {
		log.Info().
			Str("tool", tc.Name).
			Int64("cost", cost).
			Msg("shadow mode: would have charged")
	}

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
