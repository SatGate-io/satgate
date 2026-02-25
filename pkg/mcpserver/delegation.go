package mcpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

type ctxKey string

// CtxDelegationID is the context key for passing delegation request ID
// from Spend to Initialize for enterprise atomic transfer correlation.
const CtxDelegationID ctxKey = "satgate_delegation_id"

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

	// BudgetID is the child's budget tracking key.
	BudgetID string `json:"budgetId,omitempty"`
}

// Delegator handles token delegation with budget carving.
type Delegator struct {
	macaroonSvc *macaroon.Service
	budget      BudgetEnforcer
	events      EventPublisher
}

// NewDelegator creates a new delegator. Requires macaroon auth mode.
func NewDelegator(macaroonSvc *macaroon.Service, budget BudgetEnforcer) *Delegator {
	return &Delegator{
		macaroonSvc: macaroonSvc,
		budget:      budget,
		events:      &NoOpPublisher{},
	}
}

// SetEventPublisher sets the event publisher for delegation events.
func (d *Delegator) SetEventPublisher(ep EventPublisher) {
	d.events = ep
}

// Delegate creates a child token with a carved budget from the parent.
func (d *Delegator) Delegate(ctx context.Context, parent *TokenInfo, params *DelegateParams) (*DelegateResult, error) {
	if params.Budget < 0 {
		return nil, fmt.Errorf("budget must be non-negative")
	}

	if parent.RawToken == "" {
		return nil, fmt.Errorf("delegation requires macaroon auth (mode=header)")
	}

	// Auto-initialize parent budget from macaroon caveat if not yet initialized
	if parent.BudgetLimit > 0 && parent.BudgetID != "" {
		_ = d.budget.Initialize(ctx, parent.BudgetID, parent.BudgetLimit)
	}

	// Budget transfer: skip for observe-only (0-budget) tokens
	var parentResult *BudgetResult
	if params.Budget > 0 {
		// Carve budget from parent: atomically check & decrement parent, set child.
		// Each delegation gets a unique idempotency key (nonce) so repeated calls
		// are treated as new spends, not retries of the same operation.
		nonce := make([]byte, 8)
		_, _ = rand.Read(nonce)
		delegationID := fmt.Sprintf("delegate-%s-%s-%d-%s", parent.BudgetID, params.Label, params.Budget, hex.EncodeToString(nonce))
		var err error
		parentResult, err = d.budget.Spend(ctx, parent.BudgetID, "_delegation", params.Budget, delegationID)
		if err != nil {
			return nil, fmt.Errorf("insufficient parent budget: %w", err)
		}
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
	// Generate child budget_id deterministically before minting
	var childBudgetID string
	if params.Budget > 0 {
		nonceBuf := make([]byte, 8)
		_, _ = rand.Read(nonceBuf)
		h := sha256.New()
		h.Write([]byte(parent.BudgetID))
		h.Write([]byte(params.Label))
		h.Write(nonceBuf)
		childBudgetID = fmt.Sprintf("del-%s", hex.EncodeToString(h.Sum(nil))[:24])
		caveats = append(caveats, fmt.Sprintf("budget_id = %s", childBudgetID))
		caveats = append(caveats, fmt.Sprintf("budget_limit = %.2f", float64(params.Budget)))
	}

	childMac, err := d.macaroonSvc.Delegate(parent.RawToken, caveats)
	if err != nil {
		// Refund parent budget on delegation failure
		_ = d.budget.Initialize(ctx, parent.BudgetID, parentResult.Remaining+params.Budget)
		return nil, fmt.Errorf("macaroon delegation failed: %w", err)
	}

	childToken := d.macaroonSvc.Encode(childMac)
	childTokenID := hashToken(childMac.Identifier + childMac.Signature)

	// Initialize child budget (skip for observe-only 0-budget tokens)
	if params.Budget > 0 && childBudgetID != "" {
		if err := d.budget.Initialize(ctx, childBudgetID, params.Budget); err != nil {
			return nil, fmt.Errorf("initialize child budget: %w", err)
		}
	}

	parentRemaining := int64(0)
	if parentResult != nil {
		parentRemaining = parentResult.Remaining
	}

	log.Info().
		Str("parent", parent.TokenID).
		Str("child", childTokenID).
		Int64("budget", params.Budget).
		Int64("parentRemaining", parentRemaining).
		Str("label", params.Label).
		Msg("token delegated")

	d.events.Publish(Event{
		Type:      EventDelegation,
		Timestamp: time.Now(),
		TokenID:   parent.TokenID,
		BudgetID:  parent.BudgetID,
		Data: map[string]interface{}{
			"childTokenId":    childTokenID,
			"childBudgetId":   childBudgetID,
			"childBudget":     params.Budget,
			"parentRemaining": parentRemaining,
			"label":           params.Label,
			"scope":           params.Scope,
		},
	})

	return &DelegateResult{
		Token:           childToken,
		TokenID:         childTokenID,
		Budget:          params.Budget,
		ParentRemaining: parentRemaining,
		BudgetID:        childBudgetID,
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
