package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// Delegation methods — SatGate extensions to MCP.
// These are namespaced under "satgate/" to avoid conflicts with standard MCP methods.
const (
	// MethodSatGateDelegate creates a sub-agent token with carved budget.
	MethodSatGateDelegate = "satgate/delegate"

	// MethodSatGateBudget returns the current budget for the authenticated token.
	MethodSatGateBudget = "satgate/budget"
)

// DelegateParams is the input for satgate/delegate.
type DelegateParams struct {
	// Budget is the number of credits to allocate to the child token.
	// Carved from the parent's remaining budget.
	Budget int64 `json:"budget"`

	// Scope restricts what the child token can access (optional).
	Scope string `json:"scope,omitempty"`

	// Label is a human-readable name for the child (e.g., "research-agent").
	Label string `json:"label,omitempty"`

	// ExpiresIn is the TTL for the child token in seconds (optional).
	ExpiresIn int64 `json:"expiresIn,omitempty"`
}

// DelegateResult is the response for satgate/delegate.
type DelegateResult struct {
	// Token is the encoded child macaroon token.
	Token string `json:"token"`

	// TokenID is the budget key for the child.
	TokenID string `json:"tokenId"`

	// Budget is the allocated credits.
	Budget int64 `json:"budget"`

	// ParentRemaining is the parent's remaining budget after carving.
	ParentRemaining int64 `json:"parentRemaining"`
}

// Delegator handles token delegation with budget carving.
type Delegator struct {
	macaroonSvc *macaroon.Service
	budget      BudgetEnforcer
}

// NewDelegator creates a new delegator. Requires macaroon auth mode.
func NewDelegator(macaroonSvc *macaroon.Service, budget BudgetEnforcer) *Delegator {
	return &Delegator{
		macaroonSvc: macaroonSvc,
		budget:      budget,
	}
}

// Delegate creates a child token with a carved budget from the parent.
func (d *Delegator) Delegate(ctx context.Context, parent *TokenInfo, params *DelegateParams) (*DelegateResult, error) {
	if params.Budget <= 0 {
		return nil, fmt.Errorf("budget must be positive")
	}

	if parent.RawToken == "" {
		return nil, fmt.Errorf("delegation requires macaroon auth (mode=header)")
	}

	// Carve budget from parent: atomically check & decrement parent, set child
	// This is the critical section — parent budget goes down, child budget appears
	parentResult, err := d.budget.Spend(ctx, parent.BudgetID, "_delegation", params.Budget, "delegate-"+parent.BudgetID)
	if err != nil {
		return nil, fmt.Errorf("insufficient parent budget: %w", err)
	}

	// Create child macaroon with additional caveats
	caveats := []string{
		fmt.Sprintf("parent = %s", parent.TokenID),
	}

	if params.Label != "" {
		caveats = append(caveats, fmt.Sprintf("label = %s", params.Label))
	}
	if params.Scope != "" {
		caveats = append(caveats, fmt.Sprintf("scope = %s", params.Scope))
	}

	childMac, err := d.macaroonSvc.Delegate(parent.RawToken, caveats)
	if err != nil {
		// Refund parent budget on delegation failure
		_ = d.budget.Initialize(ctx, parent.BudgetID, parentResult.Remaining+params.Budget)
		return nil, fmt.Errorf("macaroon delegation failed: %w", err)
	}

	childToken := d.macaroonSvc.Encode(childMac)
	childTokenID := hashToken(childMac.Identifier + childMac.Signature)

	// Initialize child budget
	if err := d.budget.Initialize(ctx, childTokenID, params.Budget); err != nil {
		return nil, fmt.Errorf("initialize child budget: %w", err)
	}

	log.Info().
		Str("parent", parent.TokenID).
		Str("child", childTokenID).
		Int64("budget", params.Budget).
		Int64("parentRemaining", parentResult.Remaining).
		Str("label", params.Label).
		Msg("token delegated")

	return &DelegateResult{
		Token:           childToken,
		TokenID:         childTokenID,
		Budget:          params.Budget,
		ParentRemaining: parentResult.Remaining,
	}, nil
}

// HandleDelegate processes a satgate/delegate JSON-RPC request.
func (d *Delegator) HandleDelegate(ctx context.Context, req *Request, tokenInfo *TokenInfo) (*Response, error) {
	if d == nil {
		return NewErrorResponse(req.ID, CodeMethodNotFound, "delegation not available (requires auth.mode=header)"), nil
	}

	var params DelegateParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return NewErrorResponse(req.ID, CodeInvalidParams, fmt.Sprintf("invalid delegate params: %v", err)), nil
	}

	result, err := d.Delegate(ctx, tokenInfo, &params)
	if err != nil {
		return NewErrorResponseWithData(req.ID, CodeBudgetExhausted, err.Error(), map[string]interface{}{
			"error": "delegation_failed",
		}), nil
	}

	resultJSON, _ := json.Marshal(result)
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultJSON,
	}, nil
}

// HandleBudget returns the current budget for the authenticated token.
func HandleBudget(ctx context.Context, req *Request, budget BudgetEnforcer, tokenInfo *TokenInfo) (*Response, error) {
	remaining, err := budget.Remaining(ctx, tokenInfo.BudgetID)
	if err != nil {
		return NewErrorResponse(req.ID, CodeInternalError, fmt.Sprintf("budget check failed: %v", err)), nil
	}

	result, _ := json.Marshal(map[string]interface{}{
		"tokenId":   tokenInfo.TokenID,
		"budgetId":  tokenInfo.BudgetID,
		"remaining": remaining,
	})

	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  result,
	}, nil
}
