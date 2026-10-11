package mcpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/argrules"
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

// ErrParentCannotDelegate is returned when the parent token's signature does
// not verify under this server's ordinary key (a hosted sign-in token), so this
// server cannot extend it into a child token. Its text is what a client sees,
// so it names no key, signature or caveat.
var ErrParentCannotDelegate = errors.New("This token can't create child tokens here. Ask the owner to create a token for this agent on the Tokens page.")

// ErrChildTokenFailed is returned when the child token could not be created for
// a reason the owner cannot fix: decoding, a caveat check, sealing, a missing
// root key, creating the child's budget, or a budget store that gave no usable
// answer. The cause goes only to the server log.
var ErrChildTokenFailed = errors.New("This child token couldn't be created because of a problem on our side. Try again in a few minutes.")

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
	// A delegating parent may carry argument rules; the child keeps them
	// because caveats are cumulative. Only the enforcing proxy builds this.
	return &Delegator{
		macaroonSvc: macaroonSvc.AcceptingArgumentRules(),
		budget:      budget,
		events:      &NoOpPublisher{},
	}
}

// SetEventPublisher sets the event publisher for delegation events.
func (d *Delegator) SetEventPublisher(ep EventPublisher) {
	d.events = ep
}

// refundTimeout bounds a give-back. It runs on a context detached from the
// request, so a cancelled or timed-out request still gets its carve back.
const refundTimeout = 5 * time.Second

// Delegate creates a child token with a carved budget from the parent.
//
// Order: the child token is minted and sealed first and held only in a local
// variable. Then the budget is carved. The only step after the carve that can
// fail is creating the child's budget, and a failure there gives the carve back.
// A token minted for a request that fails is never returned, logged or published.
func (d *Delegator) Delegate(ctx context.Context, parent *TokenInfo, params *DelegateParams) (*DelegateResult, error) {
	if params.Budget < 0 {
		return nil, fmt.Errorf("budget must be non-negative")
	}

	if parent.RawToken == "" {
		return nil, fmt.Errorf("delegation requires macaroon auth (mode=header)")
	}

	// Enforce delegation depth limit
	if parent.DelegationDepth > 0 && parent.Depth >= parent.DelegationDepth {
		return nil, fmt.Errorf("delegation depth limit reached (%d)", parent.DelegationDepth)
	}

	// Enforce per-delegation budget cap
	if parent.DelegationBudget > 0 && params.Budget > parent.DelegationBudget {
		return nil, fmt.Errorf("delegation budget exceeds cap: requested %d, max %d per delegation", params.Budget, parent.DelegationBudget)
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

	// Nothing has been taken from the parent yet, so a failure up to the carve
	// has nothing to give back.
	childMac, err := d.macaroonSvc.Delegate(parent.RawToken, caveats)
	if err != nil {
		return nil, d.refuseMint(parent, err, "parent token cannot be extended")
	}
	// Seal the child budget_id with the shared writer. A budget_id is
	// accepted only when this seal covers it.
	if childBudgetID != "" {
		sealed, sealErr := d.macaroonSvc.AppendBudgetBind(d.macaroonSvc.Encode(childMac))
		if sealErr != nil {
			return nil, d.refuseMint(parent, sealErr, "child token could not be sealed")
		}
		childMac = sealed
	}

	// Budget transfer: skip for observe-only (0-budget) tokens. Such a request
	// calls neither Spend nor Initialize on the parent.
	var parentResult *BudgetResult
	var delegationID string
	if params.Budget > 0 {
		// Auto-initialize parent budget from macaroon caveat if not yet initialized
		if parent.BudgetLimit > 0 && parent.BudgetID != "" {
			_ = d.budget.Initialize(ctx, parent.BudgetID, parent.BudgetLimit)
		}

		// Carve budget from parent: atomically check & decrement parent, set child.
		// Each delegation gets a unique idempotency key (nonce) so repeated calls
		// are treated as new spends, not retries of the same operation.
		nonce := make([]byte, 8)
		_, _ = rand.Read(nonce)
		delegationID = fmt.Sprintf("delegate-%s-%s-%d-%s", parent.BudgetID, params.Label, params.Budget, hex.EncodeToString(nonce))
		res, spendErr := d.budget.Spend(ctx, parent.BudgetID, "_delegation", params.Budget, delegationID)
		if err := d.carveOutcome(parent, res, spendErr); err != nil {
			// Nothing was carved (see carveOutcome), so nothing is given back.
			return nil, err
		}
		parentResult = res
	}

	// Initialize child budget (skip for observe-only 0-budget tokens)
	if params.Budget > 0 && childBudgetID != "" {
		// Pass delegation ID via context for atomic TransferBudget in enterprise
		initCtx := context.WithValue(ctx, CtxDelegationID, delegationID)
		if err := d.budget.Initialize(initCtx, childBudgetID, params.Budget); err != nil {
			d.giveBack(ctx, parent, delegationID, params.Budget)
			if errors.Is(err, ErrInsufficientBudget) {
				// The budget said no when the money was to move: a real refusal.
				return nil, fmt.Errorf("insufficient parent budget: %w", err)
			}
			log.Error().Err(err).Str("parent", parent.TokenID).Str("childBudgetId", childBudgetID).Msg("delegation refused: child budget could not be created")
			return nil, ErrChildTokenFailed
		}
	}

	childToken := d.macaroonSvc.Encode(childMac)
	childTokenID := hashToken(childMac.Identifier + childMac.Signature)
	inheritedRules, rulesKnown, childScopes := childAuthorityForEvent(d.macaroonSvc, childMac, childToken)

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
		TenantID:  parent.TenantID,
		Data: map[string]interface{}{
			"childTokenId":              childTokenID,
			"childBudgetId":             childBudgetID,
			"childBudget":               params.Budget,
			"parentRemaining":           parentRemaining,
			"label":                     params.Label,
			"scope":                     params.Scope,
			EventInheritedRulesKnown:    rulesKnown,
			EventInheritedArgumentRules: inheritedRules,
			EventChildScopes:            childScopes,
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

// refuseMint turns a failure to mint or seal the child into the refusal the
// client sees. Only a parent whose signature does not verify under this
// server's ordinary key gets ErrParentCannotDelegate (the owner can fix that on
// the Tokens page). Anything else is on the server's side. The raw error goes
// only to the server log, never to the client, and never includes token text.
func (d *Delegator) refuseMint(parent *TokenInfo, err error, what string) error {
	if errors.Is(err, macaroon.ErrInvalidSignature) {
		log.Debug().Err(err).Str("parent", parent.TokenID).Msgf("delegation refused: %s", what)
		return ErrParentCannotDelegate
	}
	log.Error().Err(err).Str("parent", parent.TokenID).Msgf("delegation refused: %s", what)
	return ErrChildTokenFailed
}

// refusalCodes are the ErrorCodes of a Spend result that mean "the budget said
// no", as opposed to "the budget could not answer".
var refusalCodes = map[string]bool{
	"":                       true, // Allowed false with no code is still a no
	"budget_exhausted":       true,
	"insufficient_budget":    true,
	"budget_not_initialized": true,
	"token_not_found":        true,
}

// carveOutcome decides what a Spend answer for the carve means. The carve is
// approved only by a non-nil result with Allowed true and no error; it returns
// nil then. Every other answer took nothing from the parent, so there is
// nothing to give back, and the client sees one of two refusals:
//
//   - the budget said no (Allowed false with a refusal code): the existing
//     "insufficient parent budget" text;
//   - the budget could not answer (nil result, a store error, an answer that
//     contradicts itself): ErrChildTokenFailed. The raw cause is logged only.
func (d *Delegator) carveOutcome(parent *TokenInfo, res *BudgetResult, err error) error {
	switch {
	case res != nil && res.Allowed && err == nil:
		return nil
	case res != nil && !res.Allowed && refusalCodes[res.ErrorCode]:
		if err != nil {
			return fmt.Errorf("insufficient parent budget: %w", err)
		}
		return fmt.Errorf("insufficient parent budget: %s", refusalReason(res))
	}
	if err == nil {
		err = errors.New("the budget store gave no usable answer")
	}
	log.Error().Err(err).Str("parent", parent.TokenID).Str("budgetId", parent.BudgetID).Msg("delegation refused: the carve was not approved by the budget store")
	return ErrChildTokenFailed
}

func refusalReason(res *BudgetResult) string {
	if res.ErrorCode != "" {
		return res.ErrorCode
	}
	return "not allowed"
}

// giveBack returns one request's carve to the parent through the store's
// Refund. It runs on a context detached from the request's cancellation. If the
// store cannot give back, or the give-back fails, that is logged at error level
// with the budget ID, request ID and amount; the caller still returns its refusal.
func (d *Delegator) giveBack(ctx context.Context, parent *TokenInfo, requestID string, amount int64) {
	if amount <= 0 {
		return
	}
	refunder, ok := d.budget.(BudgetRefunder)
	if !ok {
		log.Error().Str("budgetId", parent.BudgetID).Str("requestId", requestID).Int64("amount", amount).Msg("delegation give-back failed: the budget store cannot give back a carve")
		return
	}
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), refundTimeout)
	defer cancel()
	if err := refunder.Refund(rctx, parent.BudgetID, requestID, amount); err != nil {
		log.Error().Err(err).Str("budgetId", parent.BudgetID).Str("requestId", requestID).Int64("amount", amount).Msg("delegation give-back failed")
	}
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

// childAuthorityForEvent is what the gateway persistence writer needs in
// order to store the child's own rules and scope, not a routes-only row.
// rulesKnown is false when the child could not be read; the writer then
// keeps the parent copy rather than dropping rules it could not see.
func childAuthorityForEvent(svc *macaroon.Service, child *macaroon.Macaroon, token string) (rules string, known bool, scopes []string) {
	if child == nil || svc == nil {
		return "", false, nil
	}
	info, err := FillTokenInfo(svc, child, token)
	if err != nil || info == nil {
		return "", false, nil
	}
	known = true
	if len(info.ArgumentRules) > 0 {
		raw, mErr := argrules.Marshal(info.ArgumentRules)
		if mErr != nil {
			return "", false, nil
		}
		rules = string(raw)
	}
	for _, c := range child.Caveats {
		if !strings.HasPrefix(c, "scope = ") {
			continue
		}
		v := strings.TrimPrefix(c, "scope = ")
		if argrules.IsRuleScope(v) {
			continue
		}
		scopes = append(scopes, v)
	}
	return rules, known, scopes
}
