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
	"github.com/satgate-io/satgate/pkg/toolcontrols"
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

// Delegate creates a child token with a carved budget from the parent.
func (d *Delegator) Delegate(ctx context.Context, parent *TokenInfo, params *DelegateParams) (*DelegateResult, error) {
	if params.Budget < 0 {
		return nil, fmt.Errorf("budget must be non-negative")
	}

	if parent.RawToken == "" {
		return nil, fmt.Errorf("delegation requires macaroon auth (mode=header)")
	}

	// A budget the issuer sealed at mint is the only budget a token can carry.
	// Delegation never adds one, never seals one the holder chose, and never
	// drops or swaps the parent's. This runs before any reservation or other
	// work, and it only reads.
	if err := refuseHolderBudget(d.macaroonSvc, parent.RawToken, params.Scope); err != nil {
		return nil, err
	}

	// Enforce delegation depth limit
	if parent.DelegationDepth > 0 && parent.Depth >= parent.DelegationDepth {
		return nil, fmt.Errorf("delegation depth limit reached (%d)", parent.DelegationDepth)
	}

	// Enforce per-delegation budget cap
	if parent.DelegationBudget > 0 && params.Budget > parent.DelegationBudget {
		return nil, fmt.Errorf("delegation budget exceeds cap: requested %d, max %d per delegation", params.Budget, parent.DelegationBudget)
	}

	// Auto-initialize parent budget from macaroon caveat if not yet initialized
	if parent.BudgetLimit > 0 && parent.BudgetID != "" {
		_ = d.budget.Initialize(ctx, parent.BudgetID, parent.BudgetLimit)
	}

	// Budget transfer: skip for observe-only (0-budget) tokens
	var parentResult *BudgetResult
	var delegationID string
	if params.Budget > 0 {
		// Carve budget from parent: atomically check & decrement parent, set child.
		// Each delegation gets a unique idempotency key (nonce) so repeated calls
		// are treated as new spends, not retries of the same operation.
		nonce := make([]byte, 8)
		_, _ = rand.Read(nonce)
		delegationID = fmt.Sprintf("delegate-%s-%s-%d-%s", parent.BudgetID, params.Label, params.Budget, hex.EncodeToString(nonce))
		var err error
		parentResult, err = d.budget.Spend(ctx, parent.BudgetID, "_delegation", params.Budget, delegationID)
		if err != nil {
			return nil, fmt.Errorf("insufficient parent budget: %w", err)
		}
	}

	refund := func() {
		if parentResult != nil {
			_ = d.budget.Initialize(ctx, parent.BudgetID, parentResult.Remaining+params.Budget)
		}
	}

	// Create child macaroon with additional caveats. The caveats this server
	// writes (the parent link, the child's own credit budget) come first; what
	// the holder sent in this call (label, scope) comes last. When the child
	// gets a credit budget it is sealed between the two, so the seal never
	// covers a word of this request.
	serverCaveats := []string{
		fmt.Sprintf("parent = %s", parent.TokenID),
	}
	var holderCaveats []string
	if params.Label != "" {
		holderCaveats = append(holderCaveats, fmt.Sprintf("label = %s", params.Label))
	}
	if params.Scope != "" {
		holderCaveats = append(holderCaveats, fmt.Sprintf("scope = %s", params.Scope))
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
		serverCaveats = append(serverCaveats, fmt.Sprintf("budget_id = %s", childBudgetID))
		serverCaveats = append(serverCaveats, fmt.Sprintf("budget_limit = %.2f", float64(params.Budget)))
	}

	first := serverCaveats
	if childBudgetID == "" {
		first = append(append([]string{}, serverCaveats...), holderCaveats...)
	}
	childMac, err := d.macaroonSvc.Delegate(parent.RawToken, first)
	if err != nil {
		// Refund parent budget on delegation failure
		refund()
		return nil, fmt.Errorf("macaroon delegation failed: %w", err)
	}
	parentMac, perr := d.macaroonSvc.Decode(parent.RawToken)
	if perr != nil {
		refund()
		return nil, fmt.Errorf("macaroon delegation failed: %w", perr)
	}
	// Seal the child budget_id with the shared writer. A budget_id is
	// accepted only when this seal covers it.
	//
	// A seal covers every caveat before it. Before sealing, prove that the
	// prefix it will cover names no shared budget except the ones already
	// inside the parent's issuer seal, word for word and at the same place.
	// Anything else is refused and the parent's reservation is given back.
	// The checks at the top of Delegate make a failure here unreachable; this
	// is the proof that the seal adds no budget the holder chose.
	if childBudgetID != "" {
		if !childKeepsOnlyTheParentsSealedBudget(d.macaroonSvc, parentMac, childMac) {
			refund()
			return nil, errDelegationCannotProveSeal
		}
		sealed, sealErr := d.macaroonSvc.AppendBudgetBind(d.macaroonSvc.Encode(childMac))
		if sealErr != nil {
			refund()
			return nil, fmt.Errorf("macaroon delegation failed: %w", sealErr)
		}
		childMac = sealed
		if len(holderCaveats) > 0 {
			childMac, err = d.macaroonSvc.DelegateWithoutVerify(d.macaroonSvc.Encode(childMac), holderCaveats)
			if err != nil {
				refund()
				return nil, fmt.Errorf("macaroon delegation failed: %w", err)
			}
		}
	}
	// The finished child: its budget references are the parent's sealed ones,
	// unchanged, and nothing else.
	if !childKeepsOnlyTheParentsSealedBudget(d.macaroonSvc, parentMac, childMac) {
		refund()
		return nil, errDelegationCannotProveSeal
	}

	childToken := d.macaroonSvc.Encode(childMac)
	childTokenID := hashToken(childMac.Identifier + childMac.Signature)
	inheritedRules, rulesKnown, childScopes := childAuthorityForEvent(d.macaroonSvc, childMac, childToken)

	// Initialize child budget (skip for observe-only 0-budget tokens)
	if params.Budget > 0 && childBudgetID != "" {
		// Pass delegation ID via context for atomic TransferBudget in enterprise
		initCtx := context.WithValue(ctx, CtxDelegationID, delegationID)
		if err := d.budget.Initialize(initCtx, childBudgetID, params.Budget); err != nil {
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
		TenantID:  parent.TenantID,
		Data: map[string]interface{}{
			"childTokenId":    childTokenID,
			"childBudgetId":   childBudgetID,
			"childBudget":     params.Budget,
			"parentRemaining": parentRemaining,
			"label":           params.Label,
			// Never the scope text the holder sent: a hash and a word count.
			// What the child may do is published as EventChildScopes (tool
			// names only; no rule or control word) and the rule document.
			EventScopeSHA256:            scopeSHA256(params.Scope),
			EventScopeWords:             scopeWordCount(params.Scope),
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
		if isDelegationBudgetRefusal(err) {
			// A policy refusal, not an empty budget. Nothing was reserved or
			// changed. The log and the reply carry a hash and a word count of
			// the scope, never the text the holder sent.
			log.Warn().
				Str("tokenId", tokenInfo.TokenID).
				Str("method", req.Method).
				Str("scopeSha256", scopeSHA256(params.Scope)).
				Int("scopeWords", scopeWordCount(params.Scope)).
				Str("reason", err.Error()).
				Msg("delegation refused")
			return NewErrorResponseWithData(req.ID, CodePolicyDenied, err.Error(), map[string]interface{}{
				"error": "delegation_failed",
			}), nil
		}
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

// The three refusals of a delegation that touches a shared budget. Each text is
// fixed: it never repeats what the holder sent. They share the "delegation_failed"
// reason code the other delegation failures use.
var (
	errDelegationAddsBudget = errors.New("delegation refused: a delegated token cannot add a shared budget; it keeps the budget its parent was issued with")
	errDelegationUnsealed   = errors.New("delegation refused: this token refers to a shared budget that was not issued with it, so it cannot be delegated")
	errDelegationScopeBad   = errors.New("delegation refused: the scope carries a control word that cannot be read")
	// Not reachable through the checks above; a different text so a test can
	// tell which layer refused.
	errDelegationCannotProveSeal = errors.New("delegation refused: the child's shared budget could not be proven the same as its parent's")
)

func isDelegationBudgetRefusal(err error) bool {
	return errors.Is(err, errDelegationAddsBudget) || errors.Is(err, errDelegationUnsealed) ||
		errors.Is(err, errDelegationScopeBad) || errors.Is(err, errDelegationCannotProveSeal)
}

// refuseHolderBudget is the gate every delegation passes first. It refuses
//
//  1. a budget-bearing control word in the scope the holder sent, and
//  2. a parent that names a shared budget in a caveat the issuer's seal does
//     not cover (the holder appended it offline).
//
// It reads the parent token itself, not a TokenInfo, so a hand-built identity
// cannot skip it. A parent whose budget is sealed passes; the child inherits it
// because caveats are cumulative.
func refuseHolderBudget(svc *macaroon.Service, parentToken, scope string) error {
	if err := scopeAddsBudget(scope); err != nil {
		return err
	}
	mac, err := svc.Decode(parentToken)
	if err != nil {
		return fmt.Errorf("macaroon delegation failed: %w", err)
	}
	if ok, err := budgetRefsAllSealed(mac.Caveats, svc.BudgetSealedPrefix(mac)); err != nil || !ok {
		return errDelegationUnsealed
	}
	return nil
}

// scopeAddsBudget reports a control word in the delegation scope that names a
// shared budget, or that cannot be read. Plain tool scopes pass. Words are
// split the way a scope list is read (commas and spaces).
func scopeAddsBudget(scope string) error {
	for _, part := range strings.Split(scope, ",") {
		for _, word := range strings.Fields(part) {
			if !strings.HasPrefix(word, argrules.ControlScopePrefix) {
				if len(word) >= len(argrules.ControlScopePrefix) && strings.EqualFold(word[:len(argrules.ControlScopePrefix)], argrules.ControlScopePrefix) {
					// A control word in another letter case is read by no
					// verifier as a control. It is refused rather than judged.
					return errDelegationScopeBad
				}
				continue
			}
			doc, err := toolcontrols.ParseScopeValue(word)
			if err != nil {
				return errDelegationScopeBad
			}
			for _, l := range doc.SpendLimits {
				if l.Budget != "" {
					return errDelegationAddsBudget
				}
			}
		}
	}
	return nil
}

// budgetRefsAllSealed reports whether every spending limit in caveats that
// names a shared budget sits inside the first sealedPrefix caveats, the ones a
// budget_bind seal covers.
func budgetRefsAllSealed(caveats []string, sealedPrefix int) (bool, error) {
	limits, err := toolcontrols.CollectSealed(caveats, sealedPrefix)
	if err != nil {
		return false, err
	}
	for _, l := range limits {
		if l.Budget != "" && !l.Sealed {
			return false, nil
		}
	}
	return true, nil
}

// childKeepsOnlyTheParentsSealedBudget is the proof made before the child is
// sealed: the first n caveats of the child, where n is what the parent's own
// seal covers, are the parent's caveats word for word (so its budget is
// carried unchanged), and no caveat after them names a budget. The seal that
// follows then adds no budget a holder chose.
func childKeepsOnlyTheParentsSealedBudget(svc *macaroon.Service, parent, child *macaroon.Macaroon) bool {
	if parent == nil || child == nil {
		return false
	}
	n := svc.BudgetSealedPrefix(parent)
	if len(parent.Caveats) < n || len(child.Caveats) < len(parent.Caveats) {
		return false
	}
	for i := 0; i < len(parent.Caveats); i++ {
		if child.Caveats[i] != parent.Caveats[i] {
			return false
		}
	}
	ok, err := budgetRefsAllSealed(child.Caveats, n)
	return err == nil && ok
}

func scopeSHA256(scope string) string {
	sum := sha256.Sum256([]byte(scope))
	return hex.EncodeToString(sum[:])
}

func scopeWordCount(scope string) int {
	n := 0
	for _, part := range strings.Split(scope, ",") {
		n += len(strings.Fields(part))
	}
	return n
}
