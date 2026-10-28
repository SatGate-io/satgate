package mcpserver

// Sign every action.
//
// With SetSignEveryAction(true) the proxy makes a signed receipt for every
// request an agent makes with a key that verified, and for every refusal of
// one, before the request is passed on or answered. If the receipt cannot be
// made, the request is not passed on, nothing is charged, and the agent gets
// the same "proof_unavailable" answer a refused order already gets. A priced
// Control call and a delegation carve are held while the receipt is made and
// charged only once it is signed (see every_action_hold.go).
//
// The switch is off by default. With it off, every answer is exactly what it
// was before this file existed.
//
// What counts as an action, and what does not, is written once, in
// everyActionMethods below and in docs/reference/receipt-schema.md. In short:
// a request that reaches the authenticated methods with a key that verified is
// an action, whether the proxy passes it on or refuses it. The handshake
// (initialize, ping, notifications), tools/list, and requests with no key or a
// key that does not verify are not. Unavailable-check stops stay unsigned unless
// SetSignUnavailableCheckRefusals is also on. That separate, default-off opt-in
// records only the four explicit no-budget refusal reasons below; signer,
// identity, archive and ambiguous allowed-receipt failures stay unsigned.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// everyActionMethods lists every JSON-RPC method the dispatcher knows and says
// whether it is an action. A test reads dispatchRequest's switch and fails if
// a method is missing here, so adding a method forces the decision.
var everyActionMethods = map[string]bool{
	MethodInitialize:      false, // handshake: no service reached, nothing decided
	MethodInitialized:     false,
	MethodPing:            false,
	MethodToolsList:       false, // answered from SatGate's own table, nothing spent or changed
	MethodCancelled:       false, // notification
	MethodToolsCall:       true,
	MethodSatGateDelegate: true,
	MethodSatGateBudget:   true,
	"budget/check":        true,
}

// ErrSignEveryActionNeedsWindowRecorder is what SetSignEveryAction and
// SignEveryActionReady return when the switch is on and the installed
// evidence recorder cannot keep windows (it does not implement
// WindowRecorder), or no recorder is installed. Without windows a repeated
// refusal could only be signed once per call, and an Observe call that went
// through unrecorded could not be counted: neither keeps the promise.
var ErrSignEveryActionNeedsWindowRecorder = errors.New("sign every action needs an evidence recorder that implements WindowRecorder")

// SetSignEveryAction turns "sign every action" on or off. It is off by
// default. Install the recorder first (SetEvidenceRecorder), then call this.
//
// With the switch on it returns nil only when the setup is valid: a recorder is
// installed and it implements WindowRecorder. That success is what makes the
// proxy ready (see SignEveryActionReady). With no recorder, or a recorder
// without windows, it returns ErrSignEveryActionNeedsWindowRecorder, and the
// switch is still set but the proxy is not ready: every call of a key, free or
// priced, Control or Observe, is answered proof_unavailable before anything is
// spent or sent, until SetSignEveryAction(true) is called again and returns
// nil. Ignoring the error therefore stops traffic; it never lets it through
// unsigned. Replacing the recorder later with one that keeps no windows makes
// the proxy not ready again.
func (p *Proxy) SetSignEveryAction(on bool) error {
	p.signEveryAction = on
	p.every().on.Store(on)
	p.wireDelegator()
	err := p.SignEveryActionReady()
	p.every().ready.Store(on && err == nil)
	return err
}

// SignEveryActionReady reports whether the switch can keep its promise with
// what is installed now: it returns nil with the switch off, and with it on
// when a recorder that keeps windows is installed. It only looks; the setting
// that lets calls through is made by SetSignEveryAction.
func (p *Proxy) SignEveryActionReady() error {
	if !p.signEveryAction {
		return nil
	}
	if p.evidence == nil {
		return fmt.Errorf("%w: no recorder is installed", ErrSignEveryActionNeedsWindowRecorder)
	}
	if _, ok := p.evidence.(WindowRecorder); !ok {
		return fmt.Errorf("%w: %T does not", ErrSignEveryActionNeedsWindowRecorder, p.evidence)
	}
	return nil
}

// SignEveryAction reports whether the switch is on.
func (p *Proxy) SignEveryAction() bool { return p.signEveryAction }

// SetSignUnavailableCheckRefusals opts into the unavailable-check-v1 reason
// vocabulary. Off by default, independently of SetSignEveryAction, including on
// module upgrade. Both switches must be on to emit these refusals or deliver
// existing admission handles on a transport error. Call before serving requests.
// Disabling it stops future emission; it does not rewrite archived receipts.
func (p *Proxy) SetSignUnavailableCheckRefusals(on bool) {
	p.signUnavailableCheckRefusals = on
}

func (p *Proxy) unavailableCheckRefusalsEnabled() bool {
	return p.signEveryAction && p.signUnavailableCheckRefusals
}

// ReversalFailures is how many times the proxy could not give back what it had
// taken for a call whose receipt failed, since it started: a hold that could
// not be released (it lapses by itself and the balance comes back), or the
// carve of an Observe delegation that could not be refunded. Each is logged
// with its budget and request id. Priced Control calls and carves under the
// switch are holds, so a non-zero value there is not an owed debit; for an
// Observe carve it is.
func (p *Proxy) ReversalFailures() int64 {
	return p.reversalFailures.Load() + p.every().releaseFailures.Load()
}

// wireDelegator gives the delegator the recorder, the switch and the mode
// lookup, so a delegation is signed under the same rules as a tool call.
func (p *Proxy) wireDelegator() {
	if p.delegator == nil {
		return
	}
	p.delegator.SetEvidenceRecorder(p.evidence)
	p.delegator.SetSignEveryAction(p.signEveryAction)
	p.delegator.ea = p.every()
	p.delegator.receiptFailureBlocks = func(ctx context.Context, tenantID string) bool {
		return p.allowReceiptFailureBlocks(p.enforcementModeFor(ctx, tenantID))
	}
	p.delegator.reversalFailed = func() { p.reversalFailures.Add(1) }
	p.delegator.reportMiss = func(ctx context.Context, parent *TokenInfo) error {
		return p.reportObserveMiss(ctx, parent, MethodSatGateDelegate, "", p.enforcementModeFor(ctx, tenantOf(parent)))
	}
}

// allowReceiptFailureBlocks reports whether failing to make an allow receipt
// stops the call under mode. Every mode but Observe stops it.
//
// Observe is the owner's choice (b): the call still goes through, as it does
// today, and the miss is reported to the recorder (reportObserveMiss) so the
// gateway's window store can count it and later sign one gap receipt per
// window. This is the one place that decides it. TestObserveKeepsTodaysBehaviour
// pins it.
func (p *Proxy) allowReceiptFailureBlocks(mode string) bool {
	return !isObserveMode(mode)
}

// reportObserveMiss tells the recorder that an Observe call went through with
// no receipt: tenant, tool, key and the UTC minute, nothing the agent sent. It
// does nothing with the switch off (today's behaviour). The first report runs
// on a context of its own with a time limit (so a cancelled request is still
// counted, and Observe is not held up); if the recorder rejects it, the miss is
// kept with its tenant, key, tool and minute and retried until it is recorded
// exactly once (see everyAction.reportMiss).
//
// An error stops forwarding. Unknown durable outcomes must not be called refused.
func (p *Proxy) reportObserveMiss(ctx context.Context, tokenInfo *TokenInfo, method, tool, mode string) error {
	if !p.signEveryAction {
		return nil
	}
	return p.every().reportMiss(ctx, p.evidence, missSlotFrom(ctx), unrecordedCall(tokenInfo, method, tool, mode))
}

// unrecordedCall is the report of an Observe call that went through unrecorded.
func unrecordedCall(tokenInfo *TokenInfo, method, tool, mode string) MCPWindowedCall {
	d := identifyDecision(nil, tokenInfo, method, tool, MCPDecision{
		Decision:           "allowed",
		DecisionReason:     DecisionReasonObserveUnrecorded,
		BudgetNotEvaluated: true,
	}, normalizeEnforcementMode(mode))
	return MCPWindowedCall{
		Kind:          WindowKindUnrecorded,
		Decision:      d,
		WindowStart:   RateWindowStart(time.Now()),
		WindowSeconds: RateWindowSeconds,
	}
}

func isObserveMode(mode string) bool {
	return mode == "observe" || mode == "shadow" || mode == "chargeback"
}

// proofUnavailableResponse is the answer a call gets when its receipt could
// not be made. It is the same answer, byte for byte, the refusals that already
// sign give.
func proofUnavailableResponse(id json.RawMessage, tool string) *Response {
	return NewErrorResponseWithData(id, CodeInternalError, "Evidence proof unavailable", map[string]interface{}{
		"error": "proof_unavailable",
		"tool":  tool,
	})
}

// methodRoute is the receipt's route_or_tool for a method.
func methodRoute(method, tool string) string {
	if method == MethodToolsCall {
		if tool == "" {
			return "mcp:tools/call"
		}
		return "mcp:tools/call:" + tool
	}
	return "mcp:" + method
}

// recordMethodDecision is recordMCPDecision for any authenticated method. For
// tools/call the tool is the tool's name; for every other method it is the
// method name, so a receipt always names what was asked. It adds the identity
// the token proved and nothing the agent sent: never the arguments.
func (p *Proxy) recordMethodDecision(ctx context.Context, req *Request, tokenInfo *TokenInfo, method, tool string, decision MCPDecision) (*MCPEvidence, error) {
	if p.evidence == nil {
		return nil, errNoRecorder
	}
	return recordIdentifiedDecision(ctx, p.every(), p.evidence, req, tokenInfo, method, tool, decision, p.enforcementModeFor(ctx, tenantOf(tokenInfo)))
}

func tenantOf(t *TokenInfo) string {
	if t == nil {
		return ""
	}
	return t.TenantID
}

// identifyDecision fills the identity a verified token proved into a decision.
// It adds the method, the tool and, when there is a request, its ids; it never
// adds anything the agent sent.
func identifyDecision(req *Request, tokenInfo *TokenInfo, method, tool string, decision MCPDecision, mode string) MCPDecision {
	tool = receiptTool(tool)
	if tokenInfo != nil {
		decision.TenantID = tokenInfo.TenantID
		decision.TokenID = tokenInfo.TokenID
		decision.BudgetID = tokenInfo.BudgetID
		decision.BudgetSubjectID = tokenInfo.BudgetID
		decision.BudgetLimitCredits = tokenInfo.BudgetLimit
		decision.DelegationDepth = int64(tokenInfo.DelegationDepth)
		decision.DelegationBudget = tokenInfo.DelegationBudget
		decision.ParentTokenID = tokenInfo.ParentTokenID
		decision.Scope = tokenInfo.Scope
	}
	decision.MCPMethod = method
	decision.ToolName = tool
	if method != MethodToolsCall && tool == "" {
		decision.ToolName = method
	}
	decision.RouteOrTool = methodRoute(method, tool)
	if req != nil {
		decision.JSONRPCID = string(req.ID)
		if decision.RequestID == "" {
			budgetID := ""
			if tokenInfo != nil {
				budgetID = tokenInfo.BudgetID
			}
			decision.RequestID = generateRequestID(budgetID, decision.RouteOrTool, req.ID)
		}
	}
	if decision.PolicyMode == "" {
		decision.PolicyMode = normalizeEnforcementMode(mode)
	}
	if decision.BudgetNotEvaluated {
		// No balance was read: carry identity only, never a limit or balance.
		decision.BudgetLimitCredits = 0
		decision.CostCredits = 0
		decision.RemainingCredits = 0
		decision.RemainingBeforeCredits = 0
	}
	return decision
}

var (
	errNoRecorder = errors.New("sign every action is on but no evidence recorder is installed")
	errNoReceipt  = errors.New("recorder returned no receipt")
)

// receiptTool is the tool name a receipt or a window record may carry. It is
// the one place the reserved-name rule runs: a name that starts with the
// reserved argument-rules prefix can carry the nonce of a rule document, so a
// receipt names the prefix and never the rest of the word. Every receipt and
// window envelope is built through identifyDecision, recordMCPDecision or
// recordWindowedRefusal, and each of them passes the tool through here first.
// Any other name is returned as it is.
func receiptTool(tool string) string {
	if macaroon.IsReservedToolName(tool) {
		return macaroon.ArgumentRulesScopePrefix + "redacted"
	}
	return tool
}

// recordIdentifiedDecision identifies a decision and records it. It is shared
// by the proxy and the delegator. A recorder that answers with no receipt has
// not made one: that is an error. The recorder is asked under the signing time
// limit (everyAction.record).
func recordIdentifiedDecision(ctx context.Context, ea *everyAction, rec EvidenceRecorder, req *Request, tokenInfo *TokenInfo, method, tool string, decision MCPDecision, mode string) (*MCPEvidence, error) {
	if ea != nil && req != nil && decision.RequestID == "" && ea.on.Load() {
		// The request_id of the receipt is the server's, one for each call:
		// two sessions that number their requests alike get receipts that
		// can be told apart (see newCallID).
		id, err := newCallID(callIDPrefix)
		if err != nil {
			return nil, err
		}
		decision.RequestID = id
	}
	d := identifyDecision(req, tokenInfo, method, tool, decision, mode)
	var ev *MCPEvidence
	var err error
	if ea != nil {
		ev, err = ea.record(ctx, rec, d)
	} else {
		ev, err = rec.RecordMCPDecision(ctx, d)
	}
	if err != nil {
		return nil, err
	}
	if ev == nil || ev.ReceiptID == "" {
		return nil, errNoReceipt
	}
	return ev, nil
}

// signAllowed makes the one allowed receipt of a call that every check passed,
// before the call is passed on. It returns the receipt's handles, or a
// response that stops the call.
//
// A call whose receipt cannot be made is stopped, in every mode but Observe;
// in Observe it is passed on and the miss is reported (allowReceiptFailureBlocks).
// The receipt is allowed / policy_allowed and says spend_mode not_evaluated:
// nothing was read from a balance, so it claims none.
func (p *Proxy) signAllowed(ctx context.Context, req *Request, tokenInfo *TokenInfo, method, tool, requestID, mode string) (*MCPEvidence, *Response) {
	stop := func(err error, stage string) (*MCPEvidence, *Response) {
		log.Error().Err(err).Str("method", method).Str("tool", tool).Str("stage", stage).Msg("MCP allow receipt unavailable")
		if !p.allowReceiptFailureBlocks(mode) {
			log.Warn().Str("method", method).Str("tool", tool).Str("mode", mode).Msg("observe mode: passing the call on with no receipt")
			if err := p.reportObserveMiss(ctx, tokenInfo, method, tool, mode); err != nil {
				return nil, pendingFailureResponse(req.ID, tool, err)
			}
			return nil, nil
		}
		return nil, proofUnavailableResponse(req.ID, tool)
	}
	if p.evidence == nil {
		return stop(errors.New("sign every action is on but no evidence recorder is installed"), "no_recorder")
	}
	ev, err := p.recordMethodDecision(ctx, req, tokenInfo, method, tool, MCPDecision{
		Decision:           "allowed",
		DecisionReason:     "policy_allowed",
		PolicyMode:         normalizeEnforcementMode(mode),
		BudgetNotEvaluated: true,
		RequestID:          requestID,
	})
	if err != nil {
		return stop(err, "record")
	}
	return ev, nil
}

// refuseSigned signs the refusal of a verified key's call that no earlier
// check signs: denied / policy_denied, no balance claim, with the stable code
// in denial_code. On success it returns resp (the refusal the agent would have
// got anyway) with the receipt's handles added to its error data. If the
// receipt cannot be made, or no recorder is installed, it returns
// proof_unavailable: a refusal of a verified key is signed or it is not given.
func (p *Proxy) refuseSigned(ctx context.Context, req *Request, tokenInfo *TokenInfo, method, tool, code string, resp *Response) *Response {
	ev, err := p.recordMethodDecision(ctx, req, tokenInfo, method, tool, MCPDecision{
		Decision:           "denied",
		DecisionReason:     "policy_denied",
		BudgetNotEvaluated: true,
		DenialCode:         code,
	})
	if err != nil {
		log.Error().Err(err).Str("method", method).Str("tool", tool).Str("code", code).Msg("MCP refusal receipt unavailable")
		return proofUnavailableResponse(req.ID, tool)
	}
	return withEvidenceData(resp, ev)
}

// unavailableCheckReason is deliberately not a string or an MCPDecision: this
// seam accepts no backend text, fallback price, gate detail or raw arguments.
type unavailableCheckReason uint8

const (
	unavailablePrice unavailableCheckReason = iota + 1
	unavailableControls
	unavailableBudgetBackend
	unavailablePendingCapacity
)

func (r unavailableCheckReason) reason() string {
	switch r {
	case unavailablePrice:
		return "tool_price_unavailable"
	case unavailableControls:
		return "tool_controls_unavailable"
	case unavailableBudgetBackend:
		return "budget_backend_error"
	case unavailablePendingCapacity:
		return "pending_capacity_exhausted"
	default:
		return ""
	}
}

// refuseUnavailableCheck signs one stopped call directly, never reserving a
// pending slot. The original response is only an output envelope, never signed.
// It allocates a server identity and recordMethodDecision uses the existing
// signing timeout and recorder archive contract. No failure is retried as a
// refusal: in particular, a possibly persisted allow is never passed here.
func (p *Proxy) refuseUnavailableCheck(ctx context.Context, tokenInfo *TokenInfo, tool string, reason unavailableCheckReason, resp *Response) *Response {
	if !p.unavailableCheckRefusalsEnabled() {
		return resp
	}
	if reason.reason() == "" || tokenInfo == nil || p.setupBlocks() || p.evidence == nil {
		return proofUnavailableResponse(resp.ID, tool)
	}
	requestID, err := p.every().callID(callIDPrefix)
	if err != nil {
		return proofUnavailableResponse(resp.ID, tool)
	}
	// Price failure precedes the ordinary tools/call preflight. Recheck here
	// for all four stops so a failed signer is never itself signed as an outage.
	preCtx, cancel := context.WithTimeout(ctx, p.every().cfg().SignTimeout)
	defer cancel()
	if err := p.evidence.Preflight(preCtx); err != nil {
		return proofUnavailableResponse(resp.ID, tool)
	}
	// No Request is passed: not even the caller's JSON-RPC id is a signed
	// field here. The response correlation id remains only in the envelope.
	ev, err := p.recordMethodDecision(ctx, nil, tokenInfo, MethodToolsCall, tool, MCPDecision{
		Decision:           "denied",
		DecisionReason:     reason.reason(),
		BudgetNotEvaluated: true,
		RequestID:          requestID,
	})
	if err != nil {
		return proofUnavailableResponse(resp.ID, tool)
	}
	return withEvidenceData(resp, ev)
}

// withEvidenceData returns resp with the receipt's handles merged into the
// error data (existing data is kept). A non-error response is returned as is.
func withEvidenceData(resp *Response, ev *MCPEvidence) *Response {
	if resp == nil || resp.Error == nil || ev == nil {
		return resp
	}
	data := map[string]interface{}{}
	if len(resp.Error.Data) > 0 {
		if err := json.Unmarshal(resp.Error.Data, &data); err != nil {
			data = map[string]interface{}{}
		}
	}
	mergeEvidenceData(data, ev)
	out := *resp
	e := *resp.Error
	if raw, err := json.Marshal(data); err == nil {
		e.Data = raw
	}
	out.Error = &e
	return &out
}

// codeShape is what a limiter's reason must look like to be signed as a
// denial code: upper-case words, so it can never be text an agent sent.
var codeShape = regexp.MustCompile(`^[A-Z][A-Z0-9_]{0,63}$`)

// LimiterRefusalReason is an optional interface of a RequestLimiter. When the
// limiter says no, the proxy asks it why, and signs the answer as the
// refusal's denial_code. A plain rate limit gives no answer (or an empty one)
// and is signed as RATE_LIMITED; a policy that refuses through the limiter
// (for example a trial that has ended) can give its own code. An answer that
// is not an upper-case code is ignored.
type LimiterRefusalReason interface {
	RefusalReason(ctx context.Context) string
}

func limiterDenialCode(ctx context.Context, l RequestLimiter) string {
	if r, ok := l.(LimiterRefusalReason); ok {
		if code := r.RefusalReason(ctx); codeShape.MatchString(code) {
			return code
		}
	}
	return DenialCodeRateLimited
}

// refusalTool is the tool a refusal of method names: the tool for tools/call,
// the method for the rest.
func refusalTool(req *Request) string {
	if req.Method != MethodToolsCall {
		return req.Method
	}
	if tc, err := ParseToolCall(req.Params); err == nil {
		return receiptTool(tc.Name)
	}
	return ""
}

// recordWindowedRefusal tells the window recorder that a call was refused and
// is repeatable: a rate-limit refusal, or a call made with a key that verified
// and is revoked or expired.
//
// The recorder (WindowRecorder) signs one opener and one closer per key, tool
// and UTC minute. It returns nil when the call was counted in a window that is
// already open and signed, which is a success. It returns an error when the
// opener or the record could not be made, and when there is no recorder or the
// recorder keeps no windows. The caller then answers proof_unavailable instead
// of the ordinary refusal: a refusal that cannot be recorded is not given as if
// it had been.
func (p *Proxy) recordWindowedRefusal(ctx context.Context, req *Request, tokenInfo *TokenInfo, base MCPDecision, retryAfter time.Duration) error {
	if p.setupBlocks() {
		return ErrSignEveryActionNeedsWindowRecorder
	}
	if p.evidence == nil {
		return errNoRecorder
	}
	wr, ok := p.evidence.(WindowRecorder)
	if !ok {
		return ErrSignEveryActionNeedsWindowRecorder
	}
	tool := refusalTool(req)
	tenant := ""
	if tokenInfo != nil {
		tenant = tokenInfo.TenantID
	} else {
		tenant = TenantFromContext(ctx)
	}
	base.Decision = "denied"
	if base.DecisionReason == "" {
		base.DecisionReason = "policy_denied"
	}
	base.BudgetNotEvaluated = true
	base.TenantID = tenant
	if tokenInfo != nil {
		base.TokenID = tokenInfo.TokenID
		base.BudgetID = tokenInfo.BudgetID
		base.BudgetSubjectID = tokenInfo.BudgetID
		base.DelegationDepth = int64(tokenInfo.DelegationDepth)
		base.DelegationBudget = tokenInfo.DelegationBudget
		base.ParentTokenID = tokenInfo.ParentTokenID
		base.Scope = tokenInfo.Scope
	}
	base.MCPMethod = req.Method
	base.ToolName = tool
	if base.ToolName == "" {
		base.ToolName = req.Method
	}
	base.RouteOrTool = methodRoute(req.Method, tool)
	base.JSONRPCID = string(req.ID)
	if base.PolicyMode == "" {
		base.PolicyMode = normalizeEnforcementMode(p.enforcementModeFor(ctx, tenant))
	}
	ev, err := wr.RecordWindowedCall(ctx, MCPWindowedCall{
		Kind:          WindowKindRefused,
		Decision:      base,
		WindowStart:   RateWindowStart(time.Now()),
		WindowSeconds: RateWindowSeconds,
		RetryAfter:    retryAfter,
	})
	if err != nil {
		return err
	}
	if ev != nil && ev.ReceiptID == "" {
		// A handle with no receipt in it is not a signed opener.
		return errNoReceipt
	}
	return nil
}

// expiredCredentialError marks an authentication error for a key whose
// signature verified and whose time has run out. Its text is the wrapped
// error's, so what an agent reads does not change.
type expiredCredentialError struct{ err error }

func (e *expiredCredentialError) Error() string           { return e.err.Error() }
func (e *expiredCredentialError) Unwrap() error           { return e.err }
func (e *expiredCredentialError) CredentialExpired() bool { return true }

// ExpiredCredential is an optional interface of an authentication error. An
// authenticator implements it on the error it returns for a key whose
// signature verified but whose time has run out. With SetSignEveryAction on,
// the proxy then signs the refusal, as an account-level receipt (no key hash)
// for the tenant the request's context carries (CtxTenantID). An error that
// does not implement it, or a request whose context names no tenant, is a
// request with no key SatGate can name: it gets no receipt.
type ExpiredCredential interface {
	CredentialExpired() bool
}

// markExpired wraps err when the macaroon service says the token verified and
// its time ran out. Without that it returns err as it is.
func markExpired(err error) error {
	if err == nil || errors.Is(err, macaroon.ErrInvalidSignature) {
		return err
	}
	// The macaroon service checks caveats only after the signature chain
	// verified, so this text means a verified key whose time ran out.
	if strings.Contains(err.Error(), "caveat check failed: token expired") {
		return &expiredCredentialError{err: err}
	}
	return err
}

// recordAuthRefusal signs the refusal of a request whose key did not verify,
// but only when the failure says the key verified and expired (see
// ExpiredCredential) and the context names the tenant. It signs nothing for a
// missing key or a key that does not verify: nothing says whose they are. It
// returns proof_unavailable when the window record cannot be made, and nil
// when the ordinary refusal should go out.
//
// A key that verified and that a lineage ban covers (see
// SetLineageRevocationChecker) is signed the same way, as token_revoked.
func (p *Proxy) recordAuthRefusal(ctx context.Context, req *Request, authErr error) *Response {
	decision := MCPDecision{
		DecisionReason:       "capability_expired",
		DenialCode:           DenialCodeCredentialExpired,
		NoVerifiedCapability: true,
	}
	var exp ExpiredCredential
	switch {
	case errors.As(authErr, &exp) && exp.CredentialExpired():
	case errors.Is(authErr, errTokenRevoked):
		decision.DecisionReason = "token_revoked"
		decision.DenialCode = DenialCodeTokenRevoked
	default:
		return nil
	}
	if TenantFromContext(ctx) == "" {
		log.Warn().Str("method", req.Method).Str("code", decision.DenialCode).Msg("key refused; no tenant on the request, so no receipt")
		return nil
	}
	if err := p.recordWindowedRefusal(ctx, req, nil, decision, 0); err != nil {
		log.Error().Err(err).Str("method", req.Method).Str("code", decision.DenialCode).Msg("MCP refused-call receipt unavailable")
		return proofUnavailableResponse(req.ID, refusalTool(req))
	}
	return nil
}

// dispatchAuthenticated runs the authenticated methods: tools/call,
// satgate/delegate, satgate/budget, budget/check and, with the switch on, every
// other method a key may ask for. The order is authentication, revocation, the
// request limiter, then the method. Over stdio with a lineage checker
// installed, every other token the request supplies is checked as part of
// authentication (otherSuppliedStdioTokenError), so nothing is signed, spent
// or forwarded for a request that carries a refused credential.
func (p *Proxy) dispatchAuthenticated(ctx context.Context, req *Request) (*Response, error) {
	tokenInfo, verified, authErr := p.authenticate(ctx, req)
	if authErr == nil {
		authErr = p.otherSuppliedStdioTokenError(ctx, req, verified)
	}
	if authErr != nil {
		p.publishAuthFailure(req, authErr)
		if p.signEveryAction {
			if unavailable := p.recordAuthRefusal(ctx, req, authErr); unavailable != nil {
				return unavailable, nil
			}
		}
		// The token's ban state could not be read: refused, never served.
		if errors.Is(authErr, ErrRevocationUnavailable) {
			return revocationUnavailableResponse(req.ID), nil
		}
		return NewErrorResponse(req.ID, CodePolicyDenied, authErr.Error()), nil
	}
	// Sign every action: the switch is on but its setup has not succeeded (no
	// recorder, or one that keeps no windows). Nothing is let through, free or
	// priced, Control or Observe, and nothing is spent: see setupBlocks.
	if p.setupBlocks() {
		log.Error().Str("method", req.Method).Msg("sign every action is on but its setup is not valid; refusing the call before anything is spent or sent")
		return proofUnavailableResponse(req.ID, refusalTool(req)), nil
	}
	// Inject tenant ID into context for downstream use (multi-tenant budget routing)
	if tokenInfo.TenantID != "" {
		ctx = context.WithValue(ctx, CtxTenantID, tokenInfo.TenantID)
	}
	ctx = context.WithValue(ctx, CtxTokenInfo, tokenInfo)
	// Check revocation (after tenant is in context)
	if p.revocation != nil && tokenInfo.BudgetID != "" {
		if p.revocation.IsRevoked(ctx, tokenInfo.BudgetID) {
			if p.signEveryAction {
				if err := p.recordWindowedRefusal(ctx, req, tokenInfo, MCPDecision{
					DecisionReason:       "token_revoked",
					DenialCode:           DenialCodeTokenRevoked,
					NoVerifiedCapability: true,
				}, 0); err != nil {
					log.Error().Err(err).Str("method", req.Method).Str("code", DenialCodeTokenRevoked).Msg("MCP refused-call receipt unavailable")
					return proofUnavailableResponse(req.ID, refusalTool(req)), nil
				}
			}
			return NewErrorResponse(req.ID, CodePolicyDenied, "token revoked"), nil
		}
	}
	// Per-token rate limit, before any cost, spend, event or upstream call.
	if p.limiter != nil {
		toolName := ""
		if req.Method == MethodToolsCall {
			if tc, err := ParseToolCall(req.Params); err == nil {
				toolName = tc.Name
			}
		}
		if ok, retryAfter := p.limiter.Allow(ctx, tokenInfo, req.Method, toolName); !ok {
			log.Debug().Str("method", req.Method).Str("tool", toolName).Str("tokenId", tokenInfo.TokenID).Msg("request rate limited")
			if p.signEveryAction {
				code := limiterDenialCode(ctx, p.limiter)
				if err := p.recordWindowedRefusal(ctx, req, tokenInfo, MCPDecision{
					DenialCode: code,
				}, retryAfter); err != nil {
					log.Error().Err(err).Str("method", req.Method).Str("code", code).Msg("MCP refused-call receipt unavailable")
					return proofUnavailableResponse(req.ID, refusalTool(req)), nil
				}
			}
			return rateLimitedResponse(req.ID, retryAfter), nil
		}
	}
	// Sign every action, Observe: a call that goes through with no receipt is
	// kept by tenant, key, tool and minute until it is recorded. Room for that
	// is set aside before anything is spent or sent; with none, Observe answers
	// proof_unavailable until the store drains.
	if p.signEveryAction {
		var mode string
		ctx, mode = p.withResolvedEnforcementMode(ctx, tokenInfo.TenantID)
		if !p.allowReceiptFailureBlocks(mode) {
			slot := p.every().pending.reserve(kindGap)
			if slot == nil {
				log.Error().Str("method", req.Method).Msg("the pending store is full; refusing the Observe call until it drains")
				return proofUnavailableResponse(req.ID, refusalTool(req)), nil
			}
			defer slot.release()
			ctx = withMissSlot(ctx, slot)
		}
	}
	switch req.Method {
	case MethodToolsCall:
		return p.handleToolsCall(ctx, req, tokenInfo)
	case MethodSatGateDelegate:
		return p.handleDelegate(ctx, req, tokenInfo)
	case MethodSatGateBudget:
		if p.signEveryAction {
			return p.handleBudgetSigned(ctx, req, tokenInfo)
		}
		return HandleBudget(ctx, req, p.budget, tokenInfo)
	case "budget/check":
		// Before the switch this method has no handler and answers with an
		// internal error (the alias was listed and never wired). With the
		// switch on it is the same read as satgate/budget.
		if p.signEveryAction {
			return p.handleBudgetSigned(ctx, req, tokenInfo)
		}
	default:
		if p.signEveryAction {
			return p.forwardGoverned(ctx, req, tokenInfo)
		}
	}
	return nil, fmt.Errorf("unreachable")
}

// handleBudgetSigned answers a balance read after signing it. The receipt says
// the read was allowed and carries no balance.
func (p *Proxy) handleBudgetSigned(ctx context.Context, req *Request, tokenInfo *TokenInfo) (*Response, error) {
	mode := p.enforcementModeFor(ctx, tokenInfo.TenantID)
	ev, refused := p.signAllowed(ctx, req, tokenInfo, req.Method, "", "", mode)
	if refused != nil {
		return refused, nil
	}
	resp, err := HandleBudget(ctx, req, p.budget, tokenInfo)
	if err == nil {
		attachMCPEvidenceToResponse(resp, ev)
	}
	return resp, err
}

// forwardGoverned is forwardToDefault for a key that verified: a method the
// proxy does not know goes to the default service only after it is signed. With
// no default service it is a refusal, and a signed one.
func (p *Proxy) forwardGoverned(ctx context.Context, req *Request, tokenInfo *TokenInfo) (*Response, error) {
	if IsToolCall(req.Method) {
		return p.refuseSigned(ctx, req, tokenInfo, req.Method, "", "TOOLS_CALL_NOT_GOVERNED",
			NewErrorResponse(req.ID, CodePolicyDenied, "tools/call must go through the governed path")), nil
	}
	if !p.upstream.hasDefaultUpstream() {
		return p.refuseSigned(ctx, req, tokenInfo, req.Method, "", "NO_DEFAULT_UPSTREAM",
			NewErrorResponse(req.ID, CodeUpstreamError, "upstream: no default upstream configured")), nil
	}
	mode := p.enforcementModeFor(ctx, tokenInfo.TenantID)
	ev, refused := p.signAllowed(ctx, req, tokenInfo, req.Method, "", "", mode)
	if refused != nil {
		return refused, nil
	}
	resp, err := p.forwardToDefault(ctx, req)
	if err == nil {
		attachMCPEvidenceToResponse(resp, ev)
	}
	return resp, err
}

// ---- giving a debit back ----

// reversalAttempts is how many times a give-back is tried inside its bound.
const reversalAttempts = 3

// retryUndo runs undo on a context detached from ctx's cancellation and bounded
// by bound, up to reversalAttempts times with a short pause between.
//
// A repeat is safe because of what the store's calls promise (BudgetHolder,
// BudgetRefunder, CarveSettler): each acts on the one hold, or the one
// request's debit, that it names, and a second time it finds that hold in the
// state the first try left it and changes nothing. A try whose answer was
// lost therefore costs nothing when it is repeated, and a repeat can never
// reach another call's hold, because no two calls share a hold's identity
// (newCallID). An answer that says what became of the hold (ErrHoldGone,
// ErrHoldCommitted, ErrHoldIDInUse) is final and is not repeated.
func retryUndo(ctx context.Context, bound time.Duration, undo func(context.Context) error) error {
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), bound)
	defer cancel()
	var err error
	for attempt := 0; attempt < reversalAttempts; attempt++ {
		if err = undo(rctx); err == nil {
			return nil
		}
		if errors.Is(err, ErrHoldGone) || errors.Is(err, ErrHoldCommitted) || errors.Is(err, ErrHoldIDInUse) {
			return err
		}
		select {
		case <-rctx.Done():
			return err
		case <-time.After(time.Duration(attempt+1) * 20 * time.Millisecond):
		}
	}
	return err
}

// ---- the connection's own credential ----

type transportTokenKey struct{}

// withTransportToken keeps the bearer credential of a connection on the
// request's context, apart from the request's params. The transports always
// set it; with the switch on, authenticate reads it when params carry no
// token. It is the credential the connection presented, verified like any
// other before it is used: nothing a caller wrote into params can name a
// different identity through it. It is never passed to an upstream.
func withTransportToken(ctx context.Context, token string) context.Context {
	token = presentedToken(token)
	if token == "" {
		return ctx
	}
	return context.WithValue(ctx, transportTokenKey{}, token)
}

func transportTokenFrom(ctx context.Context) string {
	t, _ := ctx.Value(transportTokenKey{}).(string)
	return t
}

// paramsAreObject reports whether params is a JSON object. The transports add
// the connection's credential to params._meta only then.
func paramsAreObject(params json.RawMessage) bool {
	t := bytes.TrimSpace(params)
	return len(t) > 0 && t[0] == '{'
}
