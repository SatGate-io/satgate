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

// SetSignEveryAction turns "sign every action" on or off. It is off by
// default. Call it before the proxy serves requests. With it on, a recorder
// must be installed (SetEvidenceRecorder): a request that would be passed on
// when no recorder exists is refused as proof_unavailable, never passed on
// unsigned.
func (p *Proxy) SetSignEveryAction(on bool) {
	p.signEveryAction = on
	p.wireDelegator()
}

// SignEveryAction reports whether the switch is on.
func (p *Proxy) SignEveryAction() bool { return p.signEveryAction }

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
	reportUnrecorded(ctx, p.evidence, tokenInfo, method, tool, normalizeEnforcementMode(mode))
}

// reportUnrecorded is reportObserveMiss for any holder of a recorder (the
// proxy and the delegator).
func reportUnrecorded(ctx context.Context, rec EvidenceRecorder, tokenInfo *TokenInfo, method, tool, mode string) {
	wr, ok := rec.(WindowRecorder)
	if !ok {
		log.Warn().Str("method", method).Str("tool", tool).Msg("observe call went through with no receipt, and the recorder keeps no windows to count it")
		return
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
		log.Error().Err(err).Str("method", method).Str("tool", tool).Msg("observe miss could not be counted")
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
		return nil, errors.New("no evidence recorder is installed")
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

// recordIdentifiedDecision identifies a decision and records it. It is shared
// by the proxy and the delegator. A recorder that answers with no receipt has
// not made one: that is an error.
func recordIdentifiedDecision(ctx context.Context, rec EvidenceRecorder, req *Request, tokenInfo *TokenInfo, method, tool string, decision MCPDecision, mode string) (*MCPEvidence, error) {
	ev, err := rec.RecordMCPDecision(ctx, identifyDecision(req, tokenInfo, method, tool, decision, mode))
	if err != nil {
		return nil, err
	}
	if ev == nil || ev.ReceiptID == "" {
		return nil, errors.New("recorder returned no receipt")
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
// receipt cannot be made it returns proof_unavailable. With no recorder the
// plain refusal goes out, as every signed refusal does.
func (p *Proxy) refuseSigned(ctx context.Context, req *Request, tokenInfo *TokenInfo, method, tool, code string, resp *Response) *Response {
	if p.evidence == nil {
		return resp
	}
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
		return tc.Name
	}
	return ""
}

// recordWindowedRefusal tells the recorder that a call was refused and is
// repeatable: a rate-limit refusal, or a call made with a key that verified
// and is revoked or expired. The refusal goes out whatever happens here.
//
// A recorder that keeps windows (WindowRecorder) signs one opener and
// one closer per key, tool and UTC minute. Any other recorder gets one denied
// receipt for the call.
func (p *Proxy) recordWindowedRefusal(ctx context.Context, req *Request, tokenInfo *TokenInfo, base MCPDecision, retryAfter time.Duration) {
	if p.evidence == nil {
		return
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
	if wr, ok := p.evidence.(WindowRecorder); ok {
		_, err := wr.RecordWindowedCall(ctx, MCPWindowedCall{
			Kind:          WindowKindRefused,
			Decision:      base,
			WindowStart:   RateWindowStart(time.Now()),
			WindowSeconds: RateWindowSeconds,
			RetryAfter:    retryAfter,
		})
		if err != nil {
			log.Error().Err(err).Str("method", req.Method).Str("code", base.DenialCode).Msg("MCP refused-call receipt unavailable")
		}
		return
	}
	if _, err := p.evidence.RecordMCPDecision(ctx, base); err != nil {
		log.Error().Err(err).Str("method", req.Method).Str("code", base.DenialCode).Msg("MCP refused-call receipt unavailable")
	}
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
// missing key or a key that does not verify: nothing says whose they are.
func (p *Proxy) recordAuthRefusal(ctx context.Context, req *Request, authErr error) {
	var exp ExpiredCredential
	if !errors.As(authErr, &exp) || !exp.CredentialExpired() {
		return
	}
	if TenantFromContext(ctx) == "" {
		log.Warn().Str("method", req.Method).Msg("expired key refused; no tenant on the request, so no receipt")
		return
	}
	p.recordWindowedRefusal(ctx, req, nil, MCPDecision{
		DecisionReason:       "capability_expired",
		DenialCode:           DenialCodeCredentialExpired,
		NoVerifiedCapability: true,
	}, 0)
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
			p.recordAuthRefusal(ctx, req, authErr)
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
				p.recordWindowedRefusal(ctx, req, tokenInfo, MCPDecision{
					DecisionReason:       "token_revoked",
					DenialCode:           DenialCodeTokenRevoked,
					NoVerifiedCapability: true,
				}, 0)
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
				p.recordWindowedRefusal(ctx, req, tokenInfo, MCPDecision{
					DenialCode: limiterDenialCode(ctx, p.limiter),
				}, retryAfter)
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
