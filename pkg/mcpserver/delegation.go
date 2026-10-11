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

// clientError is a refusal whose text is built by this package from fixed
// words and numbers. Its text is what the client sees. No error returned by a
// budget store, a macaroon call or any other dependency is ever put in one.
type clientError struct{ msg string }

func (e *clientError) Error() string { return e.msg }

// errInsufficientParentBudget is the one fixed sentence for a budget refusal
// the store reported as an ErrInsufficientBudget error.
var errInsufficientParentBudget = &clientError{msg: "insufficient parent budget"}

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
		return nil, &clientError{msg: "budget must be non-negative"}
	}

	if parent.RawToken == "" {
		return nil, &clientError{msg: "delegation requires macaroon auth (mode=header)"}
	}

	// A budget the issuer sealed at mint is the only budget a token can carry.
	// Delegation never adds one, never seals one the holder chose, and never
	// drops or swaps the parent's. This runs before any reservation or other
	// work, and it only reads.
	if err := refuseHolderBudget(d.macaroonSvc, parent.RawToken, params.Scope); err != nil {
		if r := delegationBudgetRefusal(err); r != nil {
			return nil, r
		}
		return nil, d.refuseMint(parent, err, "parent token cannot be read")
	}

	// Enforce delegation depth limit
	if parent.DelegationDepth > 0 && parent.Depth >= parent.DelegationDepth {
		return nil, &clientError{msg: fmt.Sprintf("delegation depth limit reached (%d)", parent.DelegationDepth)}
	}

	// Enforce per-delegation budget cap
	if parent.DelegationBudget > 0 && params.Budget > parent.DelegationBudget {
		return nil, &clientError{msg: fmt.Sprintf("delegation budget exceeds cap: requested %d, max %d per delegation", params.Budget, parent.DelegationBudget)}
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

	// Nothing has been taken from the parent yet, so a failure up to the carve
	// has nothing to give back.
	first := serverCaveats
	if childBudgetID == "" {
		first = append(append([]string{}, serverCaveats...), holderCaveats...)
	}
	childMac, err := d.macaroonSvc.Delegate(parent.RawToken, first)
	if err != nil {
		return nil, d.refuseMint(parent, err, "parent token cannot be extended")
	}
	parentMac, perr := d.macaroonSvc.Decode(parent.RawToken)
	if perr != nil {
		return nil, d.refuseMint(parent, perr, "parent token cannot be read")
	}
	// Seal the child budget_id with the shared writer. A budget_id is
	// accepted only when this seal covers it.
	//
	// A seal covers every caveat before it. Before sealing, prove that the
	// prefix it will cover names no shared budget except the ones already
	// inside the parent's issuer seal, word for word and at the same place.
	// Anything else is refused. The checks at the top of Delegate make a
	// failure here unreachable; this is the proof that the seal adds no budget
	// the holder chose. No budget has been carved yet, so nothing is given back.
	if childBudgetID != "" {
		if !childKeepsOnlyTheParentsSealedBudget(d.macaroonSvc, parentMac, childMac) {
			log.Error().Str("parent", parent.TokenID).Msg("delegation refused: the child's shared budget could not be proven the same as its parent's")
			return nil, errDelegationCannotProveSeal
		}
		sealed, sealErr := d.macaroonSvc.AppendBudgetBind(d.macaroonSvc.Encode(childMac))
		if sealErr != nil {
			return nil, d.refuseMint(parent, sealErr, "child token could not be sealed")
		}
		childMac = sealed
		if len(holderCaveats) > 0 {
			childMac, err = d.macaroonSvc.DelegateWithoutVerify(d.macaroonSvc.Encode(childMac), holderCaveats)
			if err != nil {
				return nil, d.refuseMint(parent, err, "child token could not take the holder's caveats")
			}
		}
	}
	// The finished child: its budget references are the parent's sealed ones,
	// unchanged, and nothing else.
	if !childKeepsOnlyTheParentsSealedBudget(d.macaroonSvc, parentMac, childMac) {
		log.Error().Str("parent", parent.TokenID).Msg("delegation refused: the finished child's shared budget could not be proven the same as its parent's")
		return nil, errDelegationCannotProveSeal
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
				// The client gets the fixed sentence, never the error's text.
				log.Warn().Err(err).Str("parent", parent.TokenID).Str("budgetId", parent.BudgetID).Str("childBudgetId", childBudgetID).Msg("delegation refused: the parent budget could not afford the carve")
				return nil, errInsufficientParentBudget
			}
			log.Error().Err(err).Str("parent", parent.TokenID).Str("budgetId", parent.BudgetID).Str("childBudgetId", childBudgetID).Msg("delegation refused: child budget could not be created")
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
//   - the budget said no, in a form this package can trust: no error and a
//     non-nil result with Allowed false and a code in refusalCodes (the message
//     is built from the code and the result's numbers), or an error that
//     matches ErrInsufficientBudget (the message is built the same way when the
//     result also refuses, else it is the fixed sentence). The error's own text
//     is never used;
//   - anything else (nil result, a store error, an answer that contradicts
//     itself, a refusal-shaped result that comes with an unrecognised error):
//     ErrChildTokenFailed. The error is logged only.
func (d *Delegator) carveOutcome(parent *TokenInfo, res *BudgetResult, err error) error {
	refusal := res != nil && !res.Allowed && refusalCodes[res.ErrorCode]
	switch {
	case res != nil && res.Allowed && err == nil:
		return nil
	case err == nil && refusal:
		return refusalError(res, false)
	case err != nil && errors.Is(err, ErrInsufficientBudget):
		log.Warn().Err(err).Str("parent", parent.TokenID).Str("budgetId", parent.BudgetID).Msg("delegation refused: the parent budget could not afford the carve")
		if refusal {
			return refusalError(res, true)
		}
		return errInsufficientParentBudget
	}
	if err == nil {
		err = errors.New("the budget store gave no usable answer")
	}
	log.Error().Err(err).Str("parent", parent.TokenID).Str("budgetId", parent.BudgetID).Msg("delegation refused: the carve was not approved by the budget store")
	return ErrChildTokenFailed
}

// refusalError is the budget refusal built from a result's code. numbers adds
// the result's remaining and required amounts to a budget_exhausted refusal; the
// store's own error text is never used.
func refusalError(res *BudgetResult, numbers bool) error {
	switch {
	case numbers && res.ErrorCode == "budget_exhausted":
		return &clientError{msg: fmt.Sprintf("insufficient parent budget: budget exhausted: %d remaining, %d required", res.Remaining, res.Cost)}
	case res.ErrorCode == "":
		return &clientError{msg: "insufficient parent budget: not allowed"}
	}
	return &clientError{msg: "insufficient parent budget: " + res.ErrorCode}
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
		if refusal := delegationBudgetRefusal(err); refusal != nil {
			// A policy refusal, not an empty budget. Nothing was reserved or
			// changed. The log and the reply carry a hash and a word count of
			// the scope, never the text the holder sent. The reply's text is the
			// refusal's own fixed sentence.
			log.Warn().
				Str("tokenId", tokenInfo.TokenID).
				Str("method", req.Method).
				Str("scopeSha256", scopeSHA256(params.Scope)).
				Int("scopeWords", scopeWordCount(params.Scope)).
				Str("reason", refusal.Error()).
				Msg("delegation refused")
			return NewErrorResponseWithData(req.ID, CodePolicyDenied, refusal.Error(), map[string]interface{}{
				"error": "delegation_failed",
			}), nil
		}
		return NewErrorResponseWithData(req.ID, CodeBudgetExhausted, delegationClientMessage(err), map[string]interface{}{
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

// delegationClientMessage is the text the client sees for a failed delegation.
// It is chosen by kind, never taken from an arbitrary error: a refusal this
// package built carries its own text; the two fixed sentences are returned as
// constants; anything else, including an error that wraps one of the fixed
// sentences around a cause, gets the server-side sentence.
func delegationClientMessage(err error) string {
	var ce *clientError
	switch {
	case errors.As(err, &ce):
		return ce.msg
	case errors.Is(err, ErrParentCannotDelegate):
		return ErrParentCannotDelegate.Error()
	case errors.Is(err, ErrChildTokenFailed):
		return ErrChildTokenFailed.Error()
	}
	log.Error().Err(err).Msg("delegation refused: unclassified error")
	return ErrChildTokenFailed.Error()
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

// delegationBudgetRefusal returns the fixed refusal err is, or nil. The caller
// sends the returned sentinel's text, never err's.
func delegationBudgetRefusal(err error) error {
	for _, e := range []error{errDelegationAddsBudget, errDelegationUnsealed, errDelegationScopeBad, errDelegationCannotProveSeal} {
		if errors.Is(err, e) {
			return e
		}
	}
	return nil
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
//
// The errors it returns are the fixed refusals above, or the decode error, which
// the caller treats as an unexpected failure and does not send to the client.
func refuseHolderBudget(svc *macaroon.Service, parentToken, scope string) error {
	if err := scopeAddsBudget(scope); err != nil {
		return err
	}
	mac, err := svc.Decode(parentToken)
	if err != nil {
		return err
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
