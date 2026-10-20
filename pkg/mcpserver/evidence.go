package mcpserver

import (
	"context"
	"time"
)

// MCPDecision is the receipt-ready decision envelope emitted for governed MCP
// tools/call budget decisions. Enterprise implementations can turn this into a
// signed, issuer-anchored Evidence Pack without depending on proxy internals.
type MCPDecision struct {
	Decision               string `json:"decision"`
	DecisionReason         string `json:"decision_reason"`
	PolicyMode             string `json:"policy_mode"`
	TenantID               string `json:"tenant_id,omitempty"`
	TokenID                string `json:"token_id,omitempty"`
	BudgetID               string `json:"budget_id,omitempty"`
	BudgetSubjectID        string `json:"budget_subject_id,omitempty"`
	BudgetLimitCredits     int64  `json:"budget_limit_credits,omitempty"`
	DelegationDepth        int64  `json:"delegation_depth,omitempty"`
	DelegationBudget       int64  `json:"delegation_budget,omitempty"`
	ParentTokenID          string `json:"parent_token_id,omitempty"`
	Scope                  string `json:"scope,omitempty"`
	MCPMethod              string `json:"mcp_method"`
	ToolName               string `json:"tool_name"`
	RouteOrTool            string `json:"route_or_tool"`
	RequestID              string `json:"request_id"`
	JSONRPCID              string `json:"jsonrpc_id,omitempty"`
	CostCredits            int64  `json:"cost_credits"`
	RemainingCredits       int64  `json:"remaining_credits"`
	RemainingBeforeCredits int64  `json:"remaining_before_credits"`
	// NoVerifiedCapability marks terminal denials produced before credential
	// verification succeeds. Recorders must not project token, capability, or
	// evaluated-budget authority for this profile.
	NoVerifiedCapability bool `json:"no_verified_capability,omitempty"`
	// BudgetNotEvaluated marks a verified capability refused by policy before
	// any budget lookup (for example a scope miss). Recorders must sign
	// spend_mode not_evaluated and must not project a limit, remaining, or
	// attempted balance, because none was read.
	BudgetNotEvaluated bool `json:"budget_not_evaluated,omitempty"`

	// DenialCode, ArgumentField and ArgumentRulesSHA256 are set only when the
	// call was refused by the token's argument rules (never for a scope
	// refusal or any other decision). Recorders sign them into the receipt so
	// a reader can tell an argument-limit refusal from a scope refusal.
	//
	// DenialCode is the stable code argrules.DenialCode (TOOL_ARGUMENT_DENIED).
	// ArgumentField is the field name from the rule that blocked the call; it
	// is empty when no single field applies (arguments unreadable, rules
	// invalid). ArgumentRulesSHA256 is the lower-case hex SHA-256 identifying
	// the rule documents the token carries (see argrules.DocumentsSHA256).
	// None of them ever holds an argument value.
	DenialCode          string `json:"denial_code,omitempty"`
	ArgumentField       string `json:"argument_field,omitempty"`
	ArgumentRulesSHA256 string `json:"argument_rules_sha256,omitempty"`

	// ControlWindow, ControlLimit, ControlWindowTotal and ApprovalID are set
	// by a CallGate (see call_gate.go): the window name ("day"/"week"), the
	// owner's maximum, the window total after an allowed call, and the
	// approval the call concerns. Recorders sign them into the receipt. They
	// are text the gate chose; none is an argument value the agent sent.
	ControlWindow      string `json:"control_window,omitempty"`
	ControlLimit       string `json:"control_limit,omitempty"`
	ControlWindowTotal string `json:"control_window_total,omitempty"`
	ApprovalID         string `json:"approval_id,omitempty"`

	// ControlShared and ControlSymbol are set only when a limit with a shared
	// budget or a symbol cap counted or refused the call: that the total is
	// shared, and the canonical symbol. Like the fields above they are text
	// the gate chose; none is a budget id or name, and none is an argument
	// value. The room left is not a receipt field: a refusal receipt makes no
	// balance claim (the verifiers refuse one that does); the agent reads it
	// in the refusal's error data.
	ControlShared bool   `json:"control_shared,omitempty"`
	ControlSymbol string `json:"control_symbol,omitempty"`

	// RateLimitPhase, RateLimitWindowStart, RateLimitWindowSeconds and
	// RateLimitRefusedCalls are set only on a window receipt (see
	// WindowRecorder): the one receipt that opens, or the one that
	// closes, a run of refused calls for one key and tool in one UTC minute.
	// Recorders sign them into the receipt as top-level plain strings and
	// integers next to DenialCode. None of them is a balance claim and none
	// holds a value the agent sent.
	//
	// RateLimitPhase is RateWindowOpened or RateWindowClosed.
	// RateLimitWindowStart is the window's first second, RFC 3339 in UTC.
	// RateLimitWindowSeconds is RateWindowSeconds. RateLimitRefusedCalls is
	// set on the closer only and counts every refused call in the window,
	// the one that opened it included (so it is at least 1).
	RateLimitPhase         string `json:"rate_limit_phase,omitempty"`
	RateLimitWindowStart   string `json:"rate_limit_window_start,omitempty"`
	RateLimitWindowSeconds int64  `json:"rate_limit_window_seconds,omitempty"`
	RateLimitRefusedCalls  int64  `json:"rate_limit_refused_calls,omitempty"`

	// UnrecordedCalls, UnrecordedWindowStart and UnrecordedWindowSeconds are set
	// only on a gap receipt (see UnrecordedDecision): how many Observe calls
	// went through with no receipt in one UTC minute. Plain integer and
	// strings, no balance claim, no value the agent sent.
	UnrecordedCalls         int64  `json:"unrecorded_calls,omitempty"`
	UnrecordedWindowStart   string `json:"unrecorded_window_start,omitempty"`
	UnrecordedWindowSeconds int64  `json:"unrecorded_window_seconds,omitempty"`
}

// Window receipts. A refused call that repeats (a rate limit that keeps
// saying no, a key that stays revoked) does not get a receipt each time: the
// first refusal of a key and tool in a UTC minute is signed as the window's
// opener, and one closer, signed after the minute is over, carries how many
// calls were refused in it. A flood of any size costs two signatures a minute.
const (
	// RateWindowSeconds is the length of a window. Windows are aligned to the
	// UTC minute.
	RateWindowSeconds = 60
	// RateWindowGraceSeconds is how long after a window ends a recorder waits
	// before it signs the closer, so late counts are not missed.
	RateWindowGraceSeconds = 15

	RateWindowOpened = "opened"
	RateWindowClosed = "closed"

	// DenialCodeRateLimited is the signed denial code of a call the request
	// limiter refused when the limiter gives no other reason.
	DenialCodeRateLimited = "RATE_LIMITED"
	// DenialCodeTokenRevoked and DenialCodeCredentialExpired are the signed
	// denial codes of a call made with a key that verified and is revoked, or
	// whose time has run out.
	DenialCodeTokenRevoked      = "TOKEN_REVOKED"
	DenialCodeCredentialExpired = "CREDENTIAL_EXPIRED"
)

// RateWindowStart returns the start of the UTC-minute window t falls in.
func RateWindowStart(t time.Time) time.Time {
	return t.UTC().Truncate(RateWindowSeconds * time.Second)
}

// The two kinds of call a WindowRecorder is told about.
const (
	// WindowKindRefused is a call that was refused and will be refused again:
	// a rate-limit refusal, or a call made with a key that verified and is
	// revoked or expired. The recorder signs an opener and a closer.
	WindowKindRefused = "refused"
	// WindowKindUnrecorded is a call that was passed on in Observe because its
	// receipt could not be made. The recorder counts it and, once signing
	// works again, signs one gap receipt for the window (UnrecordedDecision).
	WindowKindUnrecorded = "unrecorded"
)

// DecisionReasonObserveUnrecorded is the decision_reason of a gap receipt: a
// signed count of the calls that went through in Observe with no receipt.
const DecisionReasonObserveUnrecorded = "observe_unrecorded"

// MCPWindowedCall tells a WindowRecorder about one call that gets no receipt
// of its own: a refused call that repeats, or an Observe call that went
// through unrecorded. It never holds call arguments.
type MCPWindowedCall struct {
	// Kind is WindowKindRefused or WindowKindUnrecorded.
	Kind string
	// Decision is the call as a decision envelope: the key (TokenID, BudgetID,
	// TenantID), the method, the tool and, for a refusal, the signed reason and
	// DenialCode.
	Decision MCPDecision
	// WindowStart is the start of the UTC-minute window of this call.
	WindowStart time.Time
	// WindowSeconds is RateWindowSeconds.
	WindowSeconds int64
	// RetryAfter is what the limiter suggested to the agent; zero otherwise.
	RetryAfter time.Duration
}

// OpenerDecision returns the decision to sign as the window's opener: the
// refusal with its window fields set and no count.
func (c MCPWindowedCall) OpenerDecision() MCPDecision {
	d := c.Decision
	d.RateLimitPhase = RateWindowOpened
	d.RateLimitWindowStart = c.WindowStart.UTC().Format(time.RFC3339)
	d.RateLimitWindowSeconds = c.WindowSeconds
	d.RateLimitRefusedCalls = 0
	return d
}

// CloserDecision returns the decision to sign as the closer of the window an
// opener decision opened: the same key, tool and window, phase closed, and the
// number of refused calls (the opener's included). refused below 1 is raised
// to 1, because a closed window had at least its opener.
func CloserDecision(opener MCPDecision, refused int64) MCPDecision {
	if refused < 1 {
		refused = 1
	}
	d := opener
	d.RateLimitPhase = RateWindowClosed
	d.RateLimitRefusedCalls = refused
	return d
}

// UnrecordedDecision returns the decision to sign as the gap receipt of a
// window: allowed / observe_unrecorded for the key and tool of an unrecorded
// call, with the window and how many calls went through with no receipt. calls
// below 1 is raised to 1.
func UnrecordedDecision(call MCPWindowedCall, calls int64) MCPDecision {
	if calls < 1 {
		calls = 1
	}
	d := call.Decision
	d.Decision = "allowed"
	d.DecisionReason = DecisionReasonObserveUnrecorded
	d.BudgetNotEvaluated = true
	d.UnrecordedWindowStart = call.WindowStart.UTC().Format(time.RFC3339)
	d.UnrecordedWindowSeconds = call.WindowSeconds
	d.UnrecordedCalls = calls
	return d
}

// WindowRecorder is an optional interface of an EvidenceRecorder. It is how a
// recorder that keeps windows (the gateway's shared window store) is told about
// calls that get no receipt of their own. One method serves both kinds (see
// MCPWindowedCall.Kind).
//
// The proxy calls RecordWindowedCall once per call:
//
//   - WindowKindRefused: for every call the request limiter refused and, when
//     SetSignEveryAction is on, every call made with a key that verified and
//     is revoked or expired. It is called after the refusal is decided and
//     before it is sent, and the same refusal is sent whatever the method
//     returns: a refused call is not passed on, so a receipt that cannot be
//     made costs the agent nothing (the error is logged).
//   - WindowKindUnrecorded: when SetSignEveryAction is on and an Observe call
//     was passed on because its receipt could not be made. The call has gone
//     on; the method must not fail it. It is called once for that call.
//
// The implementation owns the window. For a refusal it signs one opener
// (OpenerDecision, through its own RecordMCPDecision) for the first refusal of
// a key and tool in a window, counts the rest, and signs one closer
// (CloserDecision) with the count once the window has been over for
// RateWindowGraceSeconds. For an unrecorded call it counts, per tenant, tool,
// key and UTC minute, and signs one gap receipt (UnrecordedDecision) per window
// when signing works again. It returns the receipt's handles on a call that
// signed one and nil, nil on the others.
//
// A recorder that does not implement this interface is given one denied
// receipt for each refused call through RecordMCPDecision instead (complete,
// but one signature per refused call), and nothing for an unrecorded call.
type WindowRecorder interface {
	RecordWindowedCall(ctx context.Context, call MCPWindowedCall) (*MCPEvidence, error)
}

// MCPEvidence is the verifier-facing handle returned after recording an MCP
// decision as a signed artifact.
type MCPEvidence struct {
	ReceiptID      string `json:"receipt_id,omitempty"`
	ReceiptHash    string `json:"receipt_hash,omitempty"`
	EvidencePackID string `json:"evidence_pack_id,omitempty"`
	EvidenceURL    string `json:"evidence_url,omitempty"`
	VerifyURL      string `json:"verify_url,omitempty"`
	JWKSURL        string `json:"jwks_url,omitempty"`
}

// EvidenceRecorder records governed MCP decisions into a durable, independently
// verifiable artifact. Enterprise implementations should fail closed from
// Preflight/RecordMCPDecision when required signing/archive infrastructure is
// unavailable.
type EvidenceRecorder interface {
	Preflight(ctx context.Context) error
	RecordMCPDecision(ctx context.Context, decision MCPDecision) (*MCPEvidence, error)
}

// BudgetCompensator is optionally implemented by budget enforcers that can undo
// a debit when proof generation fails before an upstream tool call is made.
type BudgetCompensator interface {
	Compensate(ctx context.Context, tokenID string, requestID string, cost int64) error
}
