package mcpserver

// Sign every action.
//
// With SetSignEveryAction(true) the proxy makes a signed receipt for every
// request an agent makes with a key that verified, and for every refusal of
// one, before the request is passed on or answered. If the receipt cannot be
// made, the request is not passed on, nothing is debited, and the agent gets
// the same "proof_unavailable" answer a refused order already gets.
//
// The switch is off by default. With it off, every answer is exactly what it
// was before this file existed.
//
// What counts as an action, and what does not, is written once, in
// everyActionMethods below and in docs/reference/receipt-schema.md. In short:
// a request that reaches the authenticated methods with a key that verified is
// an action, whether the proxy passes it on or refuses it. The handshake
// (initialize, ping, notifications), tools/list, and requests with no key or a
// key that does not verify are not. A failure where the proxy could not decide
// (a price it could not read, a spending-limit store that was down, a budget
// store that did not answer) is not an action either: there is no decision to
// record, and the request is not passed on.

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
// Turning the switch on with no recorder, or with a recorder that does not
// implement WindowRecorder, returns ErrSignEveryActionNeedsWindowRecorder. The
// switch is still set: until a capable recorder is installed every request
// that would be passed on is refused as proof_unavailable and every refusal
// of a verified key is answered proof_unavailable, so a wiring mistake the
// caller ignores stops traffic and never leaves it unsigned. A caller that
// installs the recorder after the switch checks SignEveryActionReady once the
// wiring is done.
func (p *Proxy) SetSignEveryAction(on bool) error {
	p.signEveryAction = on
	p.wireDelegator()
	return p.SignEveryActionReady()
}

// SignEveryActionReady reports whether the switch can keep its promise with
// what is installed now. It returns nil with the switch off.
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

// UncountedObserveMisses is how many Observe calls went through with no
// receipt and could not be reported to a window recorder, since the proxy
// started. It is zero when the recorder keeps windows. It exists so a gap that
// the recorder could not count is still counted somewhere.
func (p *Proxy) UncountedObserveMisses() int64 { return p.uncountedMisses.Load() }

// ReversalFailures is how many debits the proxy could not give back after a
// receipt failed, since it started. Each is logged with its budget and request
// id. A non-zero value means a store owes a reconciliation.
func (p *Proxy) ReversalFailures() int64 { return p.reversalFailures.Load() }

// wireDelegator gives the delegator the recorder, the switch and the mode
// lookup, so a delegation is signed under the same rules as a tool call.
func (p *Proxy) wireDelegator() {
	if p.delegator == nil {
		return
	}
	p.delegator.SetEvidenceRecorder(p.evidence)
	p.delegator.SetSignEveryAction(p.signEveryAction)
	p.delegator.receiptFailureBlocks = func(ctx context.Context, tenantID string) bool {
		return p.allowReceiptFailureBlocks(p.enforcementModeFor(ctx, tenantID))
	}
	p.delegator.reversalFailed = func() { p.reversalFailures.Add(1) }
	p.delegator.reportMiss = func(ctx context.Context, parent *TokenInfo) {
		p.reportObserveMiss(ctx, parent, MethodSatGateDelegate, "", p.enforcementModeFor(ctx, tenantOf(parent)))
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
// does nothing with the switch off (today's behaviour) or when the recorder
// keeps no windows. It never fails the call.
func (p *Proxy) reportObserveMiss(ctx context.Context, tokenInfo *TokenInfo, method, tool, mode string) {
	if !p.signEveryAction {
		return
	}
	if !reportUnrecorded(ctx, p.evidence, tokenInfo, method, tool, normalizeEnforcementMode(mode)) {
		p.uncountedMisses.Add(1)
	}
}

// reportUnrecorded is reportObserveMiss for any holder of a recorder (the
// proxy and the delegator). It reports whether the recorder took the count.
func reportUnrecorded(ctx context.Context, rec EvidenceRecorder, tokenInfo *TokenInfo, method, tool, mode string) bool {
	wr, ok := rec.(WindowRecorder)
	if !ok {
		log.Error().Str("method", method).Str("tool", receiptTool(tool)).Msg("observe call went through with no receipt, and the recorder keeps no windows to count it")
		return false
	}
	d := identifyDecision(nil, tokenInfo, method, tool, MCPDecision{
		Decision:           "allowed",
		DecisionReason:     DecisionReasonObserveUnrecorded,
		BudgetNotEvaluated: true,
	}, mode)
	if _, err := wr.RecordWindowedCall(ctx, MCPWindowedCall{
		Kind:          WindowKindUnrecorded,
		Decision:      d,
		WindowStart:   RateWindowStart(time.Now()),
		WindowSeconds: RateWindowSeconds,
	}); err != nil {
		log.Error().Err(err).Str("method", method).Str("tool", receiptTool(tool)).Msg("observe miss could not be counted")
		return false
	}
	return true
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
	return recordIdentifiedDecision(ctx, p.evidence, req, tokenInfo, method, tool, decision, p.enforcementModeFor(ctx, tenantOf(tokenInfo)))
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
// not made one: that is an error.
func recordIdentifiedDecision(ctx context.Context, rec EvidenceRecorder, req *Request, tokenInfo *TokenInfo, method, tool string, decision MCPDecision, mode string) (*MCPEvidence, error) {
	ev, err := rec.RecordMCPDecision(ctx, identifyDecision(req, tokenInfo, method, tool, decision, mode))
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
			p.reportObserveMiss(ctx, tokenInfo, method, tool, mode)
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
func (p *Proxy) recordAuthRefusal(ctx context.Context, req *Request, authErr error) *Response {
	var exp ExpiredCredential
	if !errors.As(authErr, &exp) || !exp.CredentialExpired() {
		return nil
	}
	if TenantFromContext(ctx) == "" {
		log.Warn().Str("method", req.Method).Msg("expired key refused; no tenant on the request, so no receipt")
		return nil
	}
	if err := p.recordWindowedRefusal(ctx, req, nil, MCPDecision{
		DecisionReason:       "capability_expired",
		DenialCode:           DenialCodeCredentialExpired,
		NoVerifiedCapability: true,
	}, 0); err != nil {
		log.Error().Err(err).Str("method", req.Method).Str("code", DenialCodeCredentialExpired).Msg("MCP refused-call receipt unavailable")
		return proofUnavailableResponse(req.ID, refusalTool(req))
	}
	return nil
}

// dispatchAuthenticated runs the authenticated methods: tools/call,
// satgate/delegate, satgate/budget, budget/check and, with the switch on, every
// other method a key may ask for. The order is authentication, revocation, the
// request limiter, then the method.
func (p *Proxy) dispatchAuthenticated(ctx context.Context, req *Request) (*Response, error) {
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
		if p.signEveryAction {
			if unavailable := p.recordAuthRefusal(ctx, req, authErr); unavailable != nil {
				return unavailable, nil
			}
		}
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

// canReverseDebit reports whether the budget store can give a request's debit
// back (Compensate or Refund). With the switch on a priced call in a blocking
// mode is refused before it is debited when it cannot.
func (p *Proxy) canReverseDebit() bool { return canReverse(p.budget) }

func canReverse(b BudgetEnforcer) bool {
	switch b.(type) {
	case BudgetCompensator, BudgetRefunder:
		return true
	}
	return false
}

// reversalAttempts is how many times a reversal is tried inside refundTimeout.
const reversalAttempts = 3

var errNoReversal = errors.New("the budget store offers neither Compensate nor Refund")

// reverseDebit gives one request's debit back. It runs on a context detached
// from the request's cancellation (a request that was cancelled while its
// receipt was being made must still be given back) and bounded by
// refundTimeout. It uses Compensate when the store has it and Refund otherwise,
// never both, and tries again a few times: both are keyed by the request id and
// do nothing the second time (see BudgetRefunder).
func reverseDebit(ctx context.Context, b BudgetEnforcer, budgetID, requestID string, amount int64) error {
	switch s := b.(type) {
	case BudgetCompensator:
		return retryUndo(ctx, func(c context.Context) error { return s.Compensate(c, budgetID, requestID, amount) })
	case BudgetRefunder:
		return retryUndo(ctx, func(c context.Context) error { return s.Refund(c, budgetID, requestID, amount) })
	}
	return errNoReversal
}

// retryUndo runs undo on a context detached from ctx's cancellation and bounded
// by refundTimeout, up to reversalAttempts times with a short pause between.
func retryUndo(ctx context.Context, undo func(context.Context) error) error {
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), refundTimeout)
	defer cancel()
	var err error
	for attempt := 0; attempt < reversalAttempts; attempt++ {
		if err = undo(rctx); err == nil {
			return nil
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
// request's context, apart from the request's params. The transports set it
// with the switch on; authenticate reads it when params carry no token. It is
// the credential the connection presented, verified like any other: nothing a
// caller wrote into params can name a different identity through it.
func withTransportToken(ctx context.Context, token string) context.Context {
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
