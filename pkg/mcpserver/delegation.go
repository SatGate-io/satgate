package mcpserver

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
	"strings"
	"sync"
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
	// helperIDs turns on the hlp- token_id caveat for zero-credit children.
	// Off by default: with it off a zero-credit child is minted exactly as it
	// was before the caveat existed, so every gateway machine, old or new,
	// accepts and stops it the same way. See EnableHelperRevocationID.
	helperIDs bool

	// recorder, signEveryAction and receiptFailureBlocks sign a delegation
	// (see SetEvidenceRecorder and SetSignEveryAction). With the switch off
	// the delegator never touches the recorder.
	recorder             EvidenceRecorder
	signEveryAction      bool
	receiptFailureBlocks func(ctx context.Context, tenantID string) bool
	reportMiss           func(ctx context.Context, parent *TokenInfo) error
	// keepUnknown keeps a delegation whose prepared receipt has an unknown
	// save until the journal can tell (Proxy.keepUnknownPrepared).
	keepUnknown    func(ctx context.Context, parent *TokenInfo, slot *pendingSlot, prep *pendingPreparation, release func(context.Context))
	reversalFailed func()
	// refuseUnavailable signs a delegation stopped on SatGate's side with an
	// unavailable-check reason (Proxy.refuseUnavailableMethod). The proxy sets
	// it; it signs only with both signing switches on.
	refuseUnavailable func(ctx context.Context, parent *TokenInfo, reason unavailableCheckReason, resp *Response) *Response
	// ea is the switch's shared state (holds, pending store, readiness). The
	// proxy sets it; a delegator used on its own gets a default one.
	ea     *everyAction
	eaOnce sync.Once

	// lineage is the proxy's lineage ban check, run again on the parent
	// before a child is minted (see Proxy.SetLineageRevocationChecker).
	lineage *lineageGate
}

// state returns the switch's shared state.
func (d *Delegator) state() *everyAction {
	d.eaOnce.Do(func() {
		if d.ea == nil {
			d.ea = newEveryAction()
			d.ea.setRecorder(d.recorder)
		}
	})
	return d.ea
}

// EnableHelperRevocationID turns the hlp- token_id caveat for zero-credit
// children on or off. It is off in a new Delegator. Turn it on only when every
// machine that enforces a revoke reads every token_id caveat of a key (an
// older enforcer reads the first one only, and would keep spending a helper
// that carries this caveat after its revoke). Call it before the Delegator
// serves requests; it is not safe to change while Delegate runs.
//
// With it on, a zero-credit child is refused unless the parent's first
// token_id caveat (the one the issuer wrote) is present and not empty.
func (d *Delegator) EnableHelperRevocationID(on bool) {
	d.helperIDs = on
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

// SetEvidenceRecorder gives the delegator the recorder that signs a
// delegation when SetSignEveryAction is on. Proxy.SetEvidenceRecorder calls it.
func (d *Delegator) SetEvidenceRecorder(rec EvidenceRecorder) { d.recorder = rec }

// SetSignEveryAction turns on signing of every delegation, and of every
// refusal of one the delegator decides. Off by default.
func (d *Delegator) SetSignEveryAction(on bool) { d.signEveryAction = on }

// errDelegationProofUnavailable is returned by Delegate when the receipt of a
// delegation could not be made. Nothing was carved: the carve was given back
// and no key leaves. HandleDelegate answers it as proof_unavailable.
var errDelegationProofUnavailable = errors.New("evidence proof unavailable")

// delegationUnavailable marks a delegation stopped on SatGate's side: a store,
// a check or the key could not be used. It wraps the error Delegate returned
// before, so the text and the answer are the same; HandleDelegate also signs
// the stop with reason. It is made only with both signing switches on.
type delegationUnavailable struct {
	reason unavailableCheckReason
	err    error
}

func (e *delegationUnavailable) Error() string { return e.err.Error() }
func (e *delegationUnavailable) Unwrap() error { return e.err }

// unavailable is err marked with reason when the stop will be signed (both
// switches on, wired by a proxy), and err as it is otherwise.
func (d *Delegator) unavailable(reason unavailableCheckReason, err error) error {
	if !d.signsUnavailable() {
		return err
	}
	return &delegationUnavailable{reason: reason, err: err}
}

func (d *Delegator) signsUnavailable() bool {
	return d.signEveryAction && d.refuseUnavailable != nil && d.ea != nil && d.ea.closesGaps()
}

type delegateRequestKey struct{}

// delegationRequest is the JSON-RPC request a delegation answers, when the
// caller kept it on the context (HandleDelegate does).
func delegationRequest(ctx context.Context) *Request {
	r, _ := ctx.Value(delegateRequestKey{}).(*Request)
	return r
}

// blocksOnReceiptFailure reports whether a delegation whose receipt cannot be
// made must stop. Observe does not (see Proxy.allowReceiptFailureBlocks);
// every other mode does.
func (d *Delegator) blocksOnReceiptFailure(ctx context.Context, tenantID string) bool {
	if d.receiptFailureBlocks == nil {
		return true
	}
	return d.receiptFailureBlocks(ctx, tenantID)
}

// signDelegation records the allowed receipt of a delegation. Like every
// allow receipt it says policy_allowed with no balance claim: it names the
// call and the key, not the amount carved.
func (d *Delegator) signDelegation(ctx context.Context, parent *TokenInfo, callID string) error {
	if d.recorder == nil {
		return errors.New("sign every action is on but no evidence recorder is installed")
	}
	_, err := recordIdentifiedDecision(ctx, d.state(), d.recorder, delegationRequest(ctx), parent, MethodSatGateDelegate, "", MCPDecision{
		Decision:           "allowed",
		DecisionReason:     "policy_allowed",
		BudgetNotEvaluated: true,
		// The carve's own identity when there is one; otherwise the receipt
		// is given one by recordIdentifiedDecision.
		RequestID: callID,
	}, "")
	return err
}

// SetEventPublisher sets the event publisher for delegation events.
func (d *Delegator) SetEventPublisher(ep EventPublisher) {
	d.events = ep
}

// ChildPersister is implemented by an EventPublisher that records a delegated
// child where a revoke can find it. Delegate calls PersistChild, for a
// zero-credit helper, and for every child whose handoff a stored row decided
// (HandoffLimiter), before it returns the key; a non-nil error means the
// child was not durably recorded and no key is returned. PersistChild must be safe to call again for the same event (the
// same parent, label and scope give the same key), and must not put the error's
// text in the reply.
type ChildPersister interface {
	PersistChild(ctx context.Context, event Event) error
}

// HandoffLimiter is implemented by an EventPublisher that keeps the handoff
// limit the owner stored for each token (the gateway's delegation tree).
// Delegate asks it before it mints or carves anything.
//
// HandoffRecord is called with the parent the request's authenticator
// verified. It answers with the row the handoff is judged by:
//
//   - rowID "" and no error: no stored limit applies to this parent (a store
//     with no rows, a token that names no tenant). Delegate goes on as it did
//     before this interface existed.
//   - rowID set: depth is the parent's depth below its root, and limited/limit
//     the stored limit (limited false = unlimited, 0 = may not hand off, N = at
//     most N levels). The rule is handoffAllowed. When the handoff goes ahead,
//     rowID, depth and the limit are carried on the delegation event
//     (EventParentRowID, EventParentDepth, EventParentLimit), so the child's
//     row is written from the row that was read.
//   - an error whose HandoffRecordMissing() reports true: the parent should
//     have a stored row and has none. Refused: "this token can't hand off to
//     a helper".
//   - any other error: the limit could not be read. Refused with
//     ErrChildTokenFailed, because a limit that cannot be read is not one
//     Delegate can promise to keep. Only the error's class is logged (see
//     errClass), never its text or the parent's id.
//
// The method uses only basic types so a gateway can implement it while it
// still builds against a version of this package without the interface.
type HandoffLimiter interface {
	HandoffRecord(ctx context.Context, parent *TokenInfo) (rowID string, depth int, limit int, limited bool, err error)
}

// handoffAllowed is the one rule for a stored handoff limit: no limit means
// unlimited, 0 (or less) refuses, and N refuses a parent already N levels
// down. parentDepth is the parent's depth in its chain (0 for a root).
func handoffAllowed(limit int, limited bool, parentDepth int) bool {
	if !limited {
		return true
	}
	if limit <= 0 {
		return false
	}
	return parentDepth+1 <= limit
}

// delegationDepthPrefix starts every depth caveat. A caveat that does not start
// with it (for example "delegation_depth=3") is not a depth caveat here, as on
// /v1/mint/delegate.
const delegationDepthPrefix = "delegation_depth = "

// readDelegationDepth reads one depth caveat exactly as /v1/mint/delegate does
// (mint.go reads it with fmt.Sscanf and macaroon.go's validator judges the
// result): the leading integer after the prefix, after any spaces, with an
// optional sign. isDepth is false for a caveat that is not a depth caveat. ok
// is false when the value does not read (empty, spaces only, letters first, a
// digit that is not 0-9, out of range): that route refuses such a token. The
// rest of the value is not looked at, so "3abc" reads 3, "2.5" reads 2, "1e3"
// reads 1 and "0x10" reads 0.
func readDelegationDepth(caveat string) (depth int, isDepth, ok bool) {
	if !strings.HasPrefix(caveat, delegationDepthPrefix) {
		return 0, false, false
	}
	if _, err := fmt.Sscanf(caveat, "delegation_depth = %d", &depth); err != nil {
		return 0, true, false
	}
	return depth, true, true
}

// readDelegationDepthChain judges every depth caveat in a chain. refused is
// true when /v1/mint/delegate would refuse the token for its depth: any value
// that does not read, reads below zero, or reads as zero (the issuer's or a
// holder's). Otherwise cap is the smallest value read (0 when the chain has no
// depth caveat), which is the cap that route applies.
func readDelegationDepthChain(caveats []string) (cap int, refused bool) {
	for _, caveat := range caveats {
		depth, isDepth, ok := readDelegationDepth(caveat)
		if !isDepth {
			continue
		}
		if !ok || depth <= 0 {
			return 0, true
		}
		if cap == 0 || depth < cap {
			cap = depth
		}
	}
	return cap, false
}

// parentCaveats are the caveats of the parent's chain: the decoded token when
// the caller kept it, else the raw token decoded here. nil when neither reads;
// the checks that follow refuse such a token.
func parentCaveats(svc *macaroon.Service, parent *TokenInfo) []string {
	mac := parent.Raw
	if mac == nil && svc != nil && parent.RawToken != "" {
		decoded, err := svc.Decode(parent.RawToken)
		if err != nil {
			return nil
		}
		mac = decoded
	}
	if mac == nil {
		return nil
	}
	return mac.Caveats
}

// storedHandoff is the record a handoff was decided by (see HandoffLimiter).
type storedHandoff struct {
	rowID   string
	depth   int
	limit   int
	limited bool
}

// errNoStoredHandoff is the refusal for a parent with no stored row, and for
// a stored limit of 0: the parent may not hand off.
var errNoStoredHandoff = &clientError{msg: "this token can't hand off to a helper"}

// checkStoredHandoffLimit applies handoffAllowed to the record the event
// consumer keeps for parent. The refusals are clientErrors, so they leave
// HandleDelegate in the same shape as the depth-limit refusal beside them. A
// nil record with no error means no stored limit applies.
func (d *Delegator) checkStoredHandoffLimit(ctx context.Context, parent *TokenInfo) (*storedHandoff, error) {
	hl, ok := d.events.(HandoffLimiter)
	if !ok {
		return nil, nil
	}
	rowID, depth, limit, limited, err := hl.HandoffRecord(ctx, parent)
	if err != nil {
		var missing interface{ HandoffRecordMissing() bool }
		if errors.As(err, &missing) && missing.HandoffRecordMissing() {
			log.Warn().Msg("delegation refused: the parent has no stored row")
			return nil, errNoStoredHandoff
		}
		log.Error().Err(errClass(err)).Msg("delegation refused: the stored handoff limit could not be read")
		return nil, d.unavailable(unavailableControls, ErrChildTokenFailed)
	}
	if rowID == "" {
		return nil, nil
	}
	if handoffAllowed(limit, limited, depth) {
		return &storedHandoff{rowID: rowID, depth: depth, limit: limit, limited: limited}, nil
	}
	if limit <= 0 {
		return nil, errNoStoredHandoff
	}
	return nil, &clientError{msg: fmt.Sprintf("this token has reached its handoff limit (%d levels)", limit)}
}

// persistChildTimeout bounds PersistChild.
const persistChildTimeout = 10 * time.Second

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
	// Sign every action: the switch is on but its setup has not succeeded.
	// Nothing is minted or carved (see Proxy.setupBlocks).
	if d.signEveryAction && d.ea != nil && !d.ea.ready.Load() {
		log.Error().Msg("delegation refused: sign every action is on but its setup is not valid")
		return nil, errDelegationProofUnavailable
	}
	// Sign every action: find a dead signer before a key is minted or a budget
	// is carved.
	if d.signEveryAction && d.recorder != nil {
		if err := d.recorder.Preflight(ctx); err != nil {
			log.Error().Err(err).Msg("delegation evidence preflight failed")
			if d.blocksOnReceiptFailure(ctx, parent.TenantID) {
				return nil, errDelegationProofUnavailable
			}
			// Observe: the delegation goes on; the miss is reported once, by
			// the receipt step below if it fails too.
		}
	}
	// Sign every action: in a blocking mode a carve is taken as a hold, not a
	// debit, so that a delegation whose receipt cannot be made ends with the
	// parent uncharged even if nothing later succeeds. A store with no hold is
	// refused before the parent is charged. Room to keep the amount, should the
	// commit fail after the receipt was signed, is set aside first.
	var (
		holder     BudgetHolder
		settler    CarveSettler
		holdExpiry time.Time
		slot       *pendingSlot
	)
	if d.signEveryAction && params.Budget > 0 && (d.blocksOnReceiptFailure(ctx, parent.TenantID) || d.state().durable != nil) {
		if holder = holdOf(d.budget); holder == nil {
			log.Error().Str("parent", parent.TokenID).Msg("delegation refused before any carve: the budget store offers no hold")
			return nil, errDelegationProofUnavailable
		}
		// The child's budget is created empty and funded only once the parent
		// has been charged, so the store must be able to do that step (and
		// repeat it) too. Without it no key could be held back.
		if settler = settlerOf(d.budget); settler == nil {
			log.Error().Str("parent", parent.TokenID).Msg("delegation refused before any carve: the budget store cannot fund a child once its parent is charged")
			return nil, errDelegationProofUnavailable
		}
		if slot = d.state().pending.reserve(kindUncharged); slot == nil {
			log.Error().Str("parent", parent.TokenID).Msg("delegation refused before any carve: the pending store is full")
			// Control, and Observe with a journal installed: the stop is signed.
			return nil, d.unavailable(unavailablePendingCapacity, errDelegationProofUnavailable)
		}
		defer slot.release()
	}
	// undoCarve lets go of what the carve took when the delegation stops
	// before its key is issued: the hold is released (and lapses by itself if
	// the release fails), or, with no hold, the carve is refunded. It reports
	// whether a hold is confirmed gone (always true with no hold): a stop
	// signed after it is signed budget_backend_error when it is not.
	undoCarve := func(delegationID string) bool {
		if holder != nil {
			return d.state().release(ctx, holder, parent.BudgetID, delegationID, params.Budget)
		}
		d.giveBack(ctx, parent, delegationID, params.Budget)
		return true
	}
	if params.Budget < 0 {
		return nil, &clientError{msg: "budget must be non-negative"}
	}

	if parent.RawToken == "" {
		return nil, &clientError{msg: "delegation requires macaroon auth (mode=header)"}
	}

	// A ban that covers the parent stops the delegation before anything is
	// minted, carved or recorded.
	if err := d.lineage.check(ctx, parent, d.macaroonSvc); err != nil {
		return nil, err
	}

	// A token whose chain carries a depth /v1/mint/delegate would refuse (zero,
	// below zero, or a value that does not read) allows no handoff, whether the
	// issuer wrote it or a holder appended it, and whether or not a stored limit
	// is wired. This runs before the stored limit is read, before any mint, and
	// before any carve: it only reads the caveats the token already carries. The
	// smallest value that does read is the cap, read the same way.
	depthCap, refused := readDelegationDepthChain(parentCaveats(d.macaroonSvc, parent))
	if refused {
		log.Warn().Msg("delegation refused: this token allows no further handoffs")
		return nil, errNoStoredHandoff
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

	// Enforce the handoff limit the owner stored for this token, when the
	// event consumer can read one (HandoffLimiter). Nothing has been taken
	// from the parent yet, so a refusal here changes nothing.
	decided, err := d.checkStoredHandoffLimit(ctx, parent)
	if err != nil {
		return nil, err
	}

	// Enforce delegation depth limit
	if depthCap == 0 || (parent.DelegationDepth > 0 && parent.DelegationDepth < depthCap) {
		depthCap = parent.DelegationDepth
	}
	if depthCap > 0 && parent.Depth >= depthCap {
		return nil, &clientError{msg: fmt.Sprintf("delegation depth limit reached (%d)", depthCap)}
	}

	// Enforce per-delegation budget cap
	if parent.DelegationBudget > 0 && params.Budget > parent.DelegationBudget {
		return nil, &clientError{msg: fmt.Sprintf("delegation budget exceeds cap: requested %d, max %d per delegation", params.Budget, parent.DelegationBudget)}
	}

	// A child carries every caveat of its parent, so its verified budget is
	// the narrowest of the parent's budget_limit caveats and its own. An
	// allocation above the parent's effective cap (that narrowest
	// budget_limit, and the parent's remaining balance when the store holds
	// one) is refused here, before any mint or carve, so the allocation the
	// caller is told is always the budget the child verifies with.
	if err := d.refuseAboveParentCap(ctx, parent, params.Budget); err != nil {
		return nil, err
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
	// A zero-credit child has no budget id of its own, so nothing the child
	// carries names it once a holder appends a caveat: the copy has a new
	// signature and a new token id. It gets a caveat that names this helper,
	// before the holder's caveats, so a revoke of the helper reaches every copy.
	// Own-credit children are named by their budget id and are not changed.
	var childRevocationID string
	if params.Budget == 0 && d.helperIDs {
		var idErr error
		childRevocationID, idErr = d.helperRevocationID(parent, params)
		if idErr != nil {
			// The parent has no identity the issuer wrote, so no stable id can
			// be derived for its helper. Nothing has been taken from the
			// parent; refuse rather than mint a helper no revoke can name.
			return nil, errDelegationNoParentIdentity
		}
		serverCaveats = append(serverCaveats, "token_id = "+childRevocationID)
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
		serverCaveats = append(serverCaveats, budgetLimitCaveat(params.Budget))
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
	// carveUnidentified is the carve's identity failure in Observe, if any.
	var carveUnidentified error
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
		if d.signEveryAction {
			// Sign every action: the carve's identity is one the server makes
			// for this carve alone, from nothing the client sent (not the
			// label, the amount or the JSON-RPC id). It names the hold, the
			// settlement and the receipt's request_id (see newCallID).
			id, err := d.state().callID(carveIDPrefix)
			if err != nil && (d.blocksOnReceiptFailure(ctx, parent.TenantID) || !d.signsUnavailable()) {
				log.Error().Err(err).Str("parent", parent.TokenID).Msg("delegation refused before any carve: no identity could be made for it")
				return nil, errDelegationProofUnavailable
			}
			if err != nil {
				// Observe: no receipt can name this carve. It goes on, under
				// an identity of this process, and is counted as a miss.
				carveUnidentified = err
				id = d.state().localCallID(carveIDPrefix)
			}
			delegationID = id
		}
		var (
			res      *BudgetResult
			spendErr error
		)
		if holder != nil {
			holdExpiry = time.Now().Add(d.state().cfg().HoldTTL)
			res, spendErr = holder.Hold(ctx, parent.BudgetID, "_delegation", params.Budget, delegationID, d.state().cfg().HoldTTL)
		} else {
			res, spendErr = d.budget.Spend(ctx, parent.BudgetID, "_delegation", params.Budget, delegationID)
		}
		if err := d.carveOutcome(parent, res, spendErr); err != nil {
			// Nothing was carved (see carveOutcome), so nothing is given back.
			return nil, err
		}
		parentResult = res
	}

	// Initialize child budget (skip for observe-only 0-budget tokens)
	if params.Budget > 0 && childBudgetID != "" {
		// Pass delegation ID via context for atomic TransferBudget in enterprise
		//
		// Sign every action (a blocking mode): the child's budget is created
		// at zero, so whatever happens next the child can spend nothing the
		// parent was not charged for. The carve is added to it by SettleCarve
		// once the commit has landed. Nothing is moved at creation, so the
		// delegation id is not passed.
		initCtx := context.WithValue(ctx, CtxDelegationID, delegationID)
		initCredits := params.Budget
		if holder != nil {
			initCtx, initCredits = ctx, 0
		}
		if err := d.budget.Initialize(initCtx, childBudgetID, initCredits); err != nil {
			released := undoCarve(delegationID)
			if errors.Is(err, ErrInsufficientBudget) {
				// The budget said no when the money was to move: a real refusal.
				// The client gets the fixed sentence, never the error's text.
				log.Warn().Err(err).Str("parent", parent.TokenID).Str("budgetId", parent.BudgetID).Str("childBudgetId", childBudgetID).Msg("delegation refused: the parent budget could not afford the carve")
				if !released {
					// The hold is not confirmed gone: the refusal is
					// signed budget_backend_error, the answer unchanged.
					return nil, d.unavailable(unavailableBudgetBackend, errInsufficientParentBudget)
				}
				return nil, errInsufficientParentBudget
			}
			log.Error().Err(err).Str("parent", parent.TokenID).Str("budgetId", parent.BudgetID).Str("childBudgetId", childBudgetID).Msg("delegation refused: child budget could not be created")
			return nil, d.unavailable(unavailableBudgetBackend, ErrChildTokenFailed)
		}
	}

	childToken := d.macaroonSvc.Encode(childMac)
	childTokenID := hashToken(childMac.Identifier + childMac.Signature)
	inheritedRules, rulesKnown, childScopes := childAuthorityForEvent(d.macaroonSvc, childMac, childToken)

	parentRemaining := int64(0)
	if parentResult != nil {
		parentRemaining = parentResult.Remaining
	}

	delegationEvent := Event{
		Type:      EventDelegation,
		Timestamp: time.Now(),
		TokenID:   parent.TokenID,
		BudgetID:  parent.BudgetID,
		TenantID:  parent.TenantID,
		Data: map[string]interface{}{
			"childTokenId":  childTokenID,
			"childBudgetId": childBudgetID,
			// The id this helper's key carries as a second token_id caveat
			// (empty for a child with its own budget, and for a parent that
			// has no token_id of its own). An id, not a secret.
			EventChildRevocationID: childRevocationID,
			"childBudget":          params.Budget,
			"parentRemaining":      parentRemaining,
			"label":                params.Label,
			// Never the scope text the holder sent: a hash and a word count.
			// What the child may do is published as EventChildScopes (tool
			// names only; no rule or control word) and the rule document.
			EventScopeSHA256:            scopeSHA256(params.Scope),
			EventScopeWords:             scopeWordCount(params.Scope),
			EventInheritedRulesKnown:    rulesKnown,
			EventInheritedArgumentRules: inheritedRules,
			EventChildScopes:            childScopes,
		},
	}
	if decided != nil {
		// The row the handoff was decided by: the child's row is written
		// under it, one level deeper, with the same limit.
		delegationEvent.Data[EventParentRowID] = decided.rowID
		delegationEvent.Data[EventParentDepth] = decided.depth
		if decided.limited {
			delegationEvent.Data[EventParentLimit] = decided.limit
		}
	}

	// A zero-credit helper is recorded by the event consumer before its key
	// leaves this function, whether or not it carries an hlp- id. What a revoke
	// of the helper names is its hlp- id (setting on) or its own macaroon
	// signature (setting off); a key whose name was not recorded could never be
	// revoked on its own, so if the publisher can record it (ChildPersister) and
	// does not, no key is returned. The cause goes to the consumer's own log,
	// not here, because it can embed an id.
	//
	// The signature goes only to PersistChild, on a copy of the event that is
	// never published: no other subscriber of the delegation event sees it, and
	// nothing here logs it.
	// When the handoff was decided by a stored row, every child (with credits
	// too) is recorded before its key is returned, so the row that keeps its
	// limit exists before the child can hand off in turn. A child with
	// credits whose row cannot be written gives its carve back.
	if params.Budget == 0 || decided != nil {
		if cp, ok := d.events.(ChildPersister); ok {
			persistEvent := delegationEvent
			persistEvent.Data = make(map[string]interface{}, len(delegationEvent.Data)+1)
			for k, v := range delegationEvent.Data {
				persistEvent.Data[k] = v
			}
			persistEvent.Data[EventChildSignature] = childMac.Signature
			pctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), persistChildTimeout)
			perr := cp.PersistChild(pctx, persistEvent)
			cancel()
			if perr != nil {
				log.Error().Msg("delegation refused: the helper could not be recorded where a revoke finds it")
				released := undoCarve(delegationID)
				return nil, d.unavailable(heldStop(released, unavailableControls), ErrChildTokenFailed)
			}
		}
	}
	// Sign every action: the receipt of the delegation is made last, once
	// nothing can refuse it any more and before the key leaves. Order: hold,
	// sign, commit, then the key. If it cannot be made, the hold is released
	// (or lapses by itself) and no key is returned. In Observe the call goes on
	// unsigned and the miss is reported. If the commit fails after a signature
	// the key is still issued, because the receipt already says it was allowed,
	// but the child's budget holds nothing (it was created at zero) until the
	// parent has been charged: the amount is kept, and the pending worker
	// settles it (SettleCarve), charging the parent exactly once and only then
	// funding the child. If the parent can no longer cover it, the child stays
	// empty and the entry is closed, logged and counted.
	if d.signEveryAction {
		sctx := ctx
		var prep *pendingPreparation
		if holder != nil {
			sctx = withHoldExpiry(ctx, holdExpiry)
			var perr error
			sctx, prep, perr = d.state().preparePending(sctx, PendingRecord{TokenID: parent.TokenID, TenantID: parent.TenantID, BudgetID: parent.BudgetID, ChildBudgetID: childBudgetID, Method: MethodSatGateDelegate, RequestID: delegationID, Amount: params.Budget})
			if perr != nil {
				undoCarve(delegationID)
				var budgetErr *pendingBudgetError
				if errors.As(perr, &budgetErr) {
					// The budget store could not prepare the carve's
					// entry. Nothing was signed; the stop is, after the
					// release: budget_backend_error whether or not the
					// release confirmed the hold gone.
					return nil, d.unavailable(unavailableBudgetBackend, errDelegationProofUnavailable)
				}
				return nil, errDelegationProofUnavailable
			}
		}
		signErr := carveUnidentified
		if signErr == nil {
			signErr = d.signDelegation(sctx, parent, delegationID)
		}
		if err := signErr; err != nil {
			log.Error().Err(err).Str("parent", parent.TokenID).Msg("delegation receipt unavailable")
			if d.blocksOnReceiptFailure(ctx, parent.TenantID) {
				undoCarve(delegationID)
				return nil, errDelegationProofUnavailable
			}
			log.Warn().Msg("observe mode: delegating with no receipt")
			if d.reportMiss != nil {
				if err := d.reportMiss(sctx, parent); err != nil {
					if errors.Is(err, ErrPendingOutcomeUnknown) {
						return nil, err
					}
					released := undoCarve(delegationID)
					if prep != nil && errors.Is(err, ErrPendingNotStored) {
						// The receipt was only prepared, never published,
						// and the hold is let go of: the stop is signed,
						// budget_backend_error if the hold is not
						// confirmed gone (it lapses).
						return nil, d.unavailable(heldStop(released, unavailablePendingCapacity), errDelegationProofUnavailable)
					}
					return nil, errDelegationProofUnavailable
				}
			}
		} else if holder != nil && prep != nil {
			stored, perr := d.state().keepPrepared(ctx, prep)
			if !stored {
				if errors.Is(perr, ErrPendingOutcomeUnknown) {
					// The answer stays outcome_unknown; the delegation is
					// kept until the journal can tell (unknownOutcome).
					if d.keepUnknown != nil {
						budgetID, amount := parent.BudgetID, params.Budget
						d.keepUnknown(ctx, parent, slot, prep, func(c context.Context) {
							d.state().release(c, holder, budgetID, delegationID, amount)
						})
					}
					return nil, perr
				}
				released := undoCarve(delegationID)
				// The receipt was only prepared, never published, and the
				// journal definitely did not keep it: the hold is let go of
				// and the stop is signed, budget_backend_error if the hold
				// is not confirmed gone (it lapses).
				return nil, d.unavailable(heldStop(released, unavailablePendingCapacity), errDelegationProofUnavailable)
			}
			if perr != nil {
				log.Warn().Msg("signed delegation is durably pending settlement")
			}
		} else if holder != nil {
			// The receipt stands, so the key is issued either way. The child's
			// budget is empty until the parent has been charged: the commit
			// lands, then SettleCarve funds the child. If either step fails,
			// the amount is kept and the retry worker runs SettleCarve until
			// it lands (charging the parent if the hold has lapsed, once), or
			// until the parent can no longer cover it (the child stays empty).
			settle := func(sctx context.Context) error {
				res, serr := settler.SettleCarve(sctx, parent.BudgetID, childBudgetID, delegationID, params.Budget)
				return d.carveSettleOutcome(res, serr)
			}
			cerr := d.state().commit(ctx, holder, parent.BudgetID, delegationID)
			if cerr != nil {
				log.Error().Err(cerr).Str("parent", parent.TokenID).Msg("the carve's hold could not be committed after the delegation was signed; the key is issued and the child can spend nothing until the parent is charged")
			} else if cerr = retryUndo(ctx, d.state().cfg().CommitTimeout, settle); cerr != nil {
				log.Error().Err(cerr).Str("parent", parent.TokenID).Msg("the parent was charged but the child's budget could not be funded; the key is issued and the child can spend nothing until it is")
			}
			if cerr != nil {
				d.state().keepUnchargedEntry(slot, parent.TokenID, parent.TenantID, parent.BudgetID, MethodSatGateDelegate, "", delegationID, params.Budget, childBudgetID, settle)
			}
		}
	}
	// Written once the key is going out: a delegation that is refused above
	// (a helper whose id could not be recorded) is not logged as delegated.
	log.Info().
		Str("parent", parent.TokenID).
		Str("child", childTokenID).
		Int64("budget", params.Budget).
		Int64("parentRemaining", parentRemaining).
		Str("label", params.Label).
		Msg("token delegated")
	d.events.Publish(delegationEvent)

	return &DelegateResult{
		Token:           childToken,
		TokenID:         childTokenID,
		Budget:          params.Budget,
		ParentRemaining: parentRemaining,
		BudgetID:        childBudgetID,
	}, nil
}

// carveSettleOutcome reads the answer of SettleCarve. Only a non-nil result
// with Allowed true and no error means the parent is charged and the child
// funded. A refusal (an error matching ErrInsufficientBudget, or a result
// that says no with a code in refusalCodes) matches ErrInsufficientBudget in
// what it returns; everything else is a failure to try again.
func (d *Delegator) carveSettleOutcome(res *BudgetResult, err error) error {
	switch {
	case err != nil && errors.Is(err, ErrInsufficientBudget):
		return ErrInsufficientBudget
	case err != nil:
		return err
	case res == nil:
		return errors.New("the budget store gave no usable answer to the carve")
	case !res.Allowed && refusalCodes[res.ErrorCode]:
		return ErrInsufficientBudget
	case !res.Allowed:
		return errors.New("the budget store did not settle the carve")
	}
	return nil
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
	return d.unavailable(unavailableControls, ErrChildTokenFailed)
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
//     matches ErrInsufficientBudget (always the one fixed sentence, whatever
//     the result holds). The error's own text is never used;
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
		// A typed refusal always gets the one fixed sentence, whatever the
		// result also holds. The result's code and numbers are not used.
		log.Warn().Err(err).Str("parent", parent.TokenID).Str("budgetId", parent.BudgetID).Msg("delegation refused: the parent budget could not afford the carve")
		return errInsufficientParentBudget
	}
	if err == nil {
		err = errors.New("the budget store gave no usable answer")
	}
	log.Error().Err(err).Str("parent", parent.TokenID).Str("budgetId", parent.BudgetID).Msg("delegation refused: the carve was not approved by the budget store")
	return d.unavailable(unavailableBudgetBackend, ErrChildTokenFailed)
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
		if d.reversalFailed != nil {
			d.reversalFailed()
		}
		return
	}
	// Refund is keyed by the request id and does nothing the second time, so
	// with the switch on a failed give-back is tried again inside the bound.
	undo := func(rctx context.Context) error { return refunder.Refund(rctx, parent.BudgetID, requestID, amount) }
	var err error
	if d.signEveryAction {
		err = retryUndo(ctx, refundTimeout, undo)
	} else {
		rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), refundTimeout)
		defer cancel()
		err = undo(rctx)
	}
	if err != nil {
		if d.reversalFailed != nil {
			d.reversalFailed()
		}
		log.Error().Err(err).Str("budgetId", parent.BudgetID).Str("requestId", requestID).Int64("amount", amount).Msg("delegation give-back failed")
	}
}

// errBudgetParamRange is the refusal for a budget param that is not a whole
// number an int64 holds (a fraction, or a value above math.MaxInt64). The
// child's budget_limit is written from that int64, so every budget the
// delegator accepts is one the verifier reads back exactly.
const errBudgetParamRange = "budget must be a whole number of credits from 0 to 9223372036854775807"

// budgetLimitCaveat writes a child's budget_limit caveat as the exact decimal
// integer, with no float conversion, so the verifier
// (toolcontrols.ParseWholeAmount) reads back the same number for every
// budget the delegator accepts, up to math.MaxInt64. Earlier releases wrote
// "budget_limit = N.00"; those still verify.
func budgetLimitCaveat(budget int64) string {
	return "budget_limit = " + strconv.FormatInt(budget, 10)
}

// refuseAboveParentCap returns a refusal when budget credits are more than the
// parent's effective cap allows, and nil otherwise. The cap is the narrowest
// of every budget_limit caveat the parent carries (read from its own caveats,
// the same way the child's are read at verify, and from parent.BudgetLimit)
// and the parent's remaining balance when the budget store reports one. A
// refusal is errInsufficientParentBudget, the sentence a carve the balance
// refuses already gives. A zero-credit child takes nothing from the parent
// and is not checked here.
//
// Only reads: nothing is minted, initialised or carved. The carve that
// follows stays the authority on the balance, since a balance read here can
// change before it.
func (d *Delegator) refuseAboveParentCap(ctx context.Context, parent *TokenInfo, budget int64) error {
	if budget <= 0 {
		return nil
	}
	limit := parent.BudgetLimit
	if caveats := parentCaveats(d.macaroonSvc, parent); caveats != nil {
		v, found, err := narrowestBudgetLimit(&macaroon.Macaroon{Caveats: caveats})
		if err != nil {
			return d.refuseMint(parent, err, "parent token cannot be read")
		}
		if found && (limit <= 0 || v < limit) {
			limit = v
		}
	}
	if limit > 0 && budget > limit {
		// The same fixed sentence as a carve the balance refuses: to the
		// caller both are "the parent cannot fund this allocation".
		log.Warn().Str("parent", parent.TokenID).Int64("budget", budget).Int64("parentCap", limit).Msg("delegation refused: allocation above the parent's budget cap")
		return errInsufficientParentBudget
	}
	if d.budget == nil || parent.BudgetID == "" {
		return nil
	}
	remaining, err := d.budget.Remaining(ctx, parent.BudgetID)
	if err != nil || remaining < 0 {
		// No balance to read (a store error, or no budget stored yet, which
		// the carve initialises from the cap checked above): the carve
		// decides.
		return nil
	}
	if budget > remaining {
		log.Warn().Str("parent", parent.TokenID).Str("budgetId", parent.BudgetID).Int64("budget", budget).Msg("delegation refused: allocation above the parent's remaining balance")
		return errInsufficientParentBudget
	}
	return nil
}

// HandleDelegate processes a satgate/delegate JSON-RPC request.
func (d *Delegator) HandleDelegate(ctx context.Context, req *Request, tokenInfo *TokenInfo) (*Response, error) {
	if d == nil {
		return NewErrorResponse(req.ID, CodeMethodNotFound, "delegation not available (requires auth.mode=header)"), nil
	}

	if d.signEveryAction && d.ea != nil && !d.ea.ready.Load() {
		return proofUnavailableResponse(req.ID, MethodSatGateDelegate), nil
	}
	var params DelegateParams
	if err := json.Unmarshal(req.Params, &params); err != nil {
		msg := fmt.Sprintf("invalid delegate params: %v", err)
		// A budget written as a JSON number that no int64 holds (a fraction,
		// an exponent, or a value above math.MaxInt64) gets a message that
		// says what is allowed. Other unreadable params keep their message.
		var typeErr *json.UnmarshalTypeError
		if errors.As(err, &typeErr) && typeErr.Field == "budget" && strings.HasPrefix(typeErr.Value, "number") {
			msg = "invalid delegate params: " + errBudgetParamRange
		}
		unreadable := NewErrorResponse(req.ID, CodeInvalidParams, msg)
		return d.signRefusal(ctx, req, tokenInfo, "INVALID_PARAMS", unreadable), nil
	}

	ctx = context.WithValue(ctx, delegateRequestKey{}, req)
	result, err := d.Delegate(ctx, tokenInfo, &params)
	if err != nil {
		var stop *delegationUnavailable
		if errors.As(err, &stop) && d.refuseUnavailable != nil {
			// Stopped on SatGate's side, both switches on: the answer it
			// always got, with the signed refusal's handles.
			resp := proofUnavailableResponse(req.ID, MethodSatGateDelegate)
			if !errors.Is(err, errDelegationProofUnavailable) {
				resp = d.refusalResponse(req, tokenInfo, &params, err)
			}
			return d.refuseUnavailable(ctx, tokenInfo, stop.reason, resp), nil
		}
		if errors.Is(err, ErrPendingOutcomeUnknown) {
			return pendingFailureResponse(req.ID, MethodSatGateDelegate, err), nil
		}
		if errors.Is(err, errDelegationProofUnavailable) {
			return proofUnavailableResponse(req.ID, MethodSatGateDelegate), nil
		}
		if errors.Is(err, ErrRevocationUnavailable) {
			return revocationUnavailableResponse(req.ID), nil
		}
		if errors.Is(err, errTokenRevoked) {
			return d.signRefusal(ctx, req, tokenInfo, DenialCodeTokenRevoked, NewErrorResponse(req.ID, CodePolicyDenied, "token revoked")), nil
		}
		return d.signRefusal(ctx, req, tokenInfo, delegationDenialCode(err), d.refusalResponse(req, tokenInfo, &params, err)), nil
	}

	resultJSON, _ := json.Marshal(result)
	return &Response{
		JSONRPC: "2.0",
		ID:      req.ID,
		Result:  resultJSON,
	}, nil
}

// delegationDenialCode is the signed denial_code of a refused delegation, or
// "" when the failure was not a decision: a problem on SatGate's side (the
// child key could not be made, a store gave no usable answer) decides
// nothing, so it has no receipt.
func delegationDenialCode(err error) string {
	var ce *clientError
	switch {
	case delegationBudgetRefusal(err) != nil:
		return "DELEGATION_REFUSED"
	case errors.As(err, &ce):
		return "DELEGATION_REFUSED"
	case errors.Is(err, ErrParentCannotDelegate):
		return "DELEGATION_REFUSED"
	}
	return ""
}

// signRefusal signs the refusal of a verified key's delegation when the switch
// is on and the refusal is a decision (code is not empty). On success it
// returns resp with the receipt's handles added; if the receipt cannot be made
// it returns proof_unavailable. With the switch off, or with no recorder, it
// returns resp unchanged.
func (d *Delegator) signRefusal(ctx context.Context, req *Request, tokenInfo *TokenInfo, code string, resp *Response) *Response {
	if !d.signEveryAction || code == "" {
		return resp
	}
	if d.recorder == nil {
		log.Error().Str("code", code).Msg("delegation refusal receipt unavailable: no recorder is installed")
		return proofUnavailableResponse(req.ID, MethodSatGateDelegate)
	}
	ev, err := recordIdentifiedDecision(ctx, d.state(), d.recorder, req, tokenInfo, MethodSatGateDelegate, "", MCPDecision{
		Decision:           "denied",
		DecisionReason:     "policy_denied",
		BudgetNotEvaluated: true,
		DenialCode:         code,
	}, "")
	if err != nil {
		log.Error().Err(err).Str("code", code).Msg("delegation refusal receipt unavailable")
		return proofUnavailableResponse(req.ID, MethodSatGateDelegate)
	}
	return withEvidenceData(resp, ev)
}

// refusalResponse is the answer to a delegation Delegate refused.
func (d *Delegator) refusalResponse(req *Request, tokenInfo *TokenInfo, params *DelegateParams, err error) *Response {
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
		})
	}
	return NewErrorResponseWithData(req.ID, CodeBudgetExhausted, delegationClientMessage(err), map[string]interface{}{
		"error": "delegation_failed",
	})
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
		v, ok := macaroon.ScopeCaveatValue(c)
		if !ok {
			continue
		}
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

// HelperRevocationIDPrefix starts the id a zero-credit helper carries. No other
// id this system writes starts with it (row ids are 12 hex characters or start
// with "del-"), so a revoke of one helper can never name anything else.
const HelperRevocationIDPrefix = "hlp-"

// EventChildSignature is the field of the event handed to ChildPersister (and
// only that one) that carries a zero-credit helper's own macaroon signature,
// a 64-character hex string. It is the one id the gateway's HTTP check reads
// that names this helper alone. It is a secret: it is never in the published
// event, never in a caveat, and never in a log.
const EventChildSignature = "childSignature"

// EventChildRevocationID is the child_delegated event field that carries the
// helper's revocation id.
const EventChildRevocationID = "childRevocationId"

// errDelegationNoParentIdentity is the fixed refusal for a zero-credit
// delegation (with helper ids on) from a parent whose first token_id caveat is
// missing or empty. It never repeats a caveat.
var errDelegationNoParentIdentity = &clientError{msg: "delegation refused: this token has no identity of its own, so a helper made from it could not be revoked on its own"}

// firstCaveatValue returns the value of the first caveat named key, and whether
// there is one. The first is the one the issuer wrote: caveats only append, so
// a holder can add later ones but cannot put one in front. GetCaveat returns
// the last, which a holder can choose; nothing that names a key for a revoke
// may use it.
func firstCaveatValue(mac *macaroon.Macaroon, key string) (string, bool) {
	if mac == nil {
		return "", false
	}
	prefix := key + " = "
	for _, caveat := range mac.Caveats {
		if strings.HasPrefix(caveat, prefix) {
			return strings.TrimPrefix(caveat, prefix), true
		}
	}
	return "", false
}

// helperRevocationID derives the id a zero-credit helper carries from the
// parent's token id, the label and the scope the holder sent. Nothing random
// goes in: delegating again with the same parent, label and scope gives the same
// key and the same id, so the same database row (no new row per session), and a
// helper revoked once is still refused when it is delegated again.
//
// The parent's identity is its first token_id caveat, read from the parent
// token itself (the one Delegate extends), never from a TokenInfo a caller built
// and never by a last-value lookup a holder could shadow with a later, empty
// token_id caveat. If that first caveat is missing or empty the error is
// returned and the caller refuses to delegate. The id is a second token_id
// caveat, and a reader that takes the first one (receipts, billing) keeps
// getting the parent's.
func (d *Delegator) helperRevocationID(parent *TokenInfo, params *DelegateParams) (string, error) {
	if parent == nil || parent.RawToken == "" {
		return "", errDelegationNoParentIdentity
	}
	mac, err := d.macaroonSvc.Decode(parent.RawToken)
	if err != nil {
		return "", errDelegationNoParentIdentity
	}
	if id, ok := firstCaveatValue(mac, "token_id"); !ok || id == "" {
		return "", errDelegationNoParentIdentity
	}
	h := sha256.New()
	h.Write([]byte("satgate-zero-credit-helper-v1\x00"))
	h.Write([]byte(parent.TokenID))
	h.Write([]byte{0})
	h.Write([]byte(params.Label))
	h.Write([]byte{0})
	h.Write([]byte(params.Scope))
	return HelperRevocationIDPrefix + hex.EncodeToString(h.Sum(nil))[:24], nil
}
