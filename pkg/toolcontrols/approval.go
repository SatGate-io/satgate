package toolcontrols

import (
	"bytes"
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/satgate-io/satgate/pkg/argrules"
)

// "Ask me" approvals: a token can say that calls to some tools whose numeric
// argument is above a threshold are not refused and not forwarded, but held
// until the token's owner approves that exact call.
//
//	{"tools":["place_crypto_order"],"field":"dollar_amount","above":"50"}
//
// A call at or below the threshold is not touched. Argument rules and
// spending limits run first: a call they would refuse is refused, never held.
//
// The rule rides in the same control scope word as spending limits, so a
// verifier that does not know approvals refuses the token instead of ignoring
// the hold.

// Codes of the refusals an approval produces. They are the error code in the
// JSON-RPC data and the denial_code signed into the receipt.
const (
	DenialCodeApprovalRequired    = "APPROVAL_REQUIRED"
	DenialCodeApprovalDenied      = "APPROVAL_DENIED"
	DenialCodeApprovalExpired     = "APPROVAL_EXPIRED"
	DenialCodeApprovalUnreadable  = "APPROVAL_CHECK_FAILED"
	DenialCodeApprovalTooManyOpen = "APPROVAL_QUEUE_FULL"
)

// ApprovalTTL is how long a held call waits for the owner.
const ApprovalTTL = 15 * time.Minute

// MaxPendingApprovals bounds the calls one token can have waiting at once, so
// an agent cannot fill the owner's inbox by changing one argument at a time.
const MaxPendingApprovals = 20

// MaxPendingApprovalsPerTenant bounds the calls a whole tenant can have waiting
// at once across all its tokens. A delegated child is a new token, so the
// per-token bound alone multiplies with the number of tokens.
const MaxPendingApprovalsPerTenant = 50

// Email volume. A tenant is sent at most MaxApprovalEmailsPerHour approval
// emails in any rolling ApprovalMailWindow, however many tokens it has. A call
// over the cap is still held and still on the owner's approvals page; only its
// email is held back. The held-back calls (and the ones whose email failed) are
// announced by one summary, "N more orders are waiting for your approval", with
// no amounts or symbols: at most MaxApprovalSummariesPerHour per window, sent
// once the oldest of them has waited SummaryDelay, so a burst is one summary.
// The summary is not an approval email and does not use one of the ten.
const (
	MaxApprovalEmailsPerHour    = 10
	MaxApprovalSummariesPerHour = 1
	ApprovalMailWindow          = time.Hour
	SummaryDelay                = 2 * time.Minute
)

// ApprovalRule holds calls above a threshold for the owner.
type ApprovalRule struct {
	// Tools are the tools the rule covers, sorted.
	Tools []string
	// Field is the top-level numeric argument that is compared.
	Field string
	// Above is the threshold, as written (a plain decimal). A call whose
	// field is greater than this is held; equal is not.
	Above string

	// ID names the rule (set by ParseScopeValue / CollectApprovals).
	ID string

	above *argrules.Decimal
}

// Covers reports whether the rule applies to tool. Names match without regard
// to letter case, like a spending limit (SpendLimit.Covers): a scope that
// allows "Place_Crypto_Order" must not be a way round a rule written for
// "place_crypto_order", because an upstream may read the two as one tool.
func (r ApprovalRule) Covers(tool string) bool {
	for _, t := range r.Tools {
		if strings.EqualFold(t, tool) {
			return true
		}
	}
	return false
}

// Threshold is Above as an exact decimal.
func (r ApprovalRule) Threshold() *argrules.Decimal { return r.above }

func parseApprovalRule(raw json.RawMessage) (ApprovalRule, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return ApprovalRule{}, ctlErr("an approval rule must be an object")
	}
	var r ApprovalRule
	var sawTools, sawField, sawAbove bool
	for k, v := range obj {
		switch k {
		case "tools":
			if err := json.Unmarshal(v, &r.Tools); err != nil {
				return ApprovalRule{}, ctlErr("tools must be a list of names")
			}
			sawTools = true
		case "field":
			if err := json.Unmarshal(v, &r.Field); err != nil {
				return ApprovalRule{}, ctlErr("field must be a string")
			}
			sawField = true
		case "above":
			// A JSON string, like a spending limit's max, so no tool rounds it.
			if err := json.Unmarshal(v, &r.Above); err != nil {
				return ApprovalRule{}, ctlErr("above must be a string holding a plain decimal")
			}
			sawAbove = true
		default:
			return ApprovalRule{}, ctlErr("unknown approval rule key %q", clip(k))
		}
	}
	if !sawTools || !sawField || !sawAbove {
		return ApprovalRule{}, ctlErr("an approval rule needs tools, field and above")
	}
	if err := r.validate(); err != nil {
		return ApprovalRule{}, err
	}
	return r, nil
}

func (r *ApprovalRule) validate() error {
	if len(r.Tools) == 0 || len(r.Tools) > MaxToolsPerLimit {
		return ctlErr("an approval rule covers 1 to %d tools", MaxToolsPerLimit)
	}
	seen := map[string]bool{}
	for _, t := range r.Tools {
		if !argrules.ValidToolName(t) {
			return ctlErr("tool name is not allowed")
		}
		if seen[t] {
			return ctlErr("a tool is listed twice")
		}
		seen[t] = true
	}
	sort.Strings(r.Tools)
	if !argrules.ValidFieldName(r.Field) {
		return ctlErr("field name is not allowed")
	}
	d, err := argrules.ParseDecimal(r.Above)
	if err != nil {
		return ctlErr("above is not a plain decimal")
	}
	if d.Rat().Sign() < 0 {
		return ctlErr("above must not be negative")
	}
	whole, frac, _ := strings.Cut(r.Above, ".")
	if len(whole) > MaxWholeDigits || len(frac) > MaxScale {
		return ctlErr("above has more than %d digits before or %d after the point", MaxWholeDigits, MaxScale)
	}
	r.above = d
	return nil
}

// CollectApprovals reads every control caveat on a token, in order, and
// returns its approval rules. A rule that names a tool the token's scope does
// not allow is kept and inert, like a spending limit.
func CollectApprovals(caveats []string) ([]ApprovalRule, error) {
	var all []ApprovalRule
	n := 0
	for _, caveat := range caveats {
		value, ok := strings.CutPrefix(caveat, "scope = ")
		if !ok || !argrules.IsControlScope(value) {
			continue
		}
		n++
		if n > MaxControlCaveats {
			return nil, ctlErr("more than %d control caveats", MaxControlCaveats)
		}
		d, err := ParseScopeValue(value)
		if err != nil {
			return nil, err
		}
		all = append(all, d.Approvals...)
		if len(all) > MaxApprovalRules {
			return nil, ctlErr("more than %d approval rules on one token", MaxApprovalRules)
		}
	}
	return all, nil
}

// DescribeApprovals is the plain-words form shown to a token owner.
func DescribeApprovals(rules []ApprovalRule) []string {
	var out []string
	for _, r := range rules {
		out = append(out, fmt.Sprintf("%s: calls with %s above %s wait for your approval",
			strings.Join(r.Tools, " and "), r.Field, FormatAmount(r.Field, r.Above)))
	}
	return out
}

// ---- The call a held approval is bound to ------------------------------------

// ErrCallUnreadable means the call cannot be turned into one canonical form
// without loss (see strictjson.go): repeated keys at any depth, invalid UTF-8,
// an unpaired UTF-16 surrogate escape, a number the canonical form cannot keep
// exactly, nesting too deep, or not an object. The error text names the kind of
// problem and never holds a value from the call.
var ErrCallUnreadable = errors.New("the call could not be read")

// maxCallNesting is the same bound argument rules use (argrules maxCallJSONDepth).
const maxCallNesting = 64

// CallHash binds an approval to one call: it is the SHA-256 of the token id,
// the tool and the canonical form of the call. What is hashed must identify
// exactly what is forwarded, so the call is read by the strict, lossless
// reader in strictjson.go, which refuses what it cannot read without loss
// instead of normalising it. Two calls hash equal only if the strict reader
// reads them as equal values.
//
// The canonical form sorts object keys and drops insignificant whitespace and
// escape spelling; it keeps every number as written ("75", "75.0" and "7.5e1"
// are three calls) and keeps "arguments" absent, null and {} apart. Every
// top-level member is part of the call (arguments, and anything else an
// upstream might read), including _meta, which is forwarded as sent. Two
// members of _meta are left out on purpose: token (the credential; the call
// is already bound to the token id) and progressToken (a per-request
// correlation value a client changes on every send; it names progress
// notifications, not what is ordered). The JSON-RPC id is not in params.
//
// The hash is one-way. It is safe to keep next to an approval; it is not a
// way to read the call back.
func CallHash(tokenID, tool string, params json.RawMessage) (string, error) {
	if tokenID == "" || tool == "" {
		return "", ErrCallUnreadable
	}
	v, err := readStrict(params)
	if err != nil {
		return "", err
	}
	top, ok := v.(objectEntries)
	if !ok {
		return "", ErrCallUnreadable
	}
	// A key that differs from name, arguments or _meta only by letter case may
	// be read as that member by an upstream whose decoder folds case.
	for k := range top {
		if k != "name" && k != "arguments" && k != "_meta" &&
			(strings.EqualFold(k, "name") || strings.EqualFold(k, "arguments") || strings.EqualFold(k, "_meta")) {
			return "", unreadable("a member name differs from a reserved one only by letter case")
		}
	}
	if name, ok := top["name"].(string); !ok || name != tool {
		return "", ErrCallUnreadable
	}
	call := objectEntries{}
	for k, val := range top {
		call[k] = val
	}
	delete(call, "name") // equal to tool, which is hashed below
	if meta, ok := call["_meta"].(objectEntries); ok && len(meta) > 0 {
		kept := objectEntries{}
		for k, val := range meta {
			if k != "token" && k != "progressToken" {
				kept[k] = val
			}
		}
		if len(kept) == 0 {
			delete(call, "_meta")
		} else {
			call["_meta"] = kept
		}
	}
	if args, present := call["arguments"]; present && args != nil {
		if _, isObj := args.(objectEntries); !isObj {
			return "", ErrCallUnreadable
		}
	}
	var buf bytes.Buffer
	if err := writeCanonical(&buf, map[string]interface{}(call)); err != nil {
		return "", ErrCallUnreadable
	}
	h := sha256.New()
	// Length-prefixed, so no token id and tool can be re-split into another pair.
	fmt.Fprintf(h, "satgate-approval-call-v2\n%d:%s\n%d:%s\n", len(tokenID), tokenID, len(tool), tool)
	h.Write(buf.Bytes())
	return hex.EncodeToString(h.Sum(nil)), nil
}

func writeCanonical(buf *bytes.Buffer, v interface{}) error {
	switch x := v.(type) {
	case nil:
		buf.WriteString("null")
	case bool:
		if x {
			buf.WriteString("true")
		} else {
			buf.WriteString("false")
		}
	case numText:
		buf.WriteString(string(x))
	case string:
		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)
		if err := enc.Encode(x); err != nil {
			return err
		}
		buf.Truncate(buf.Len() - 1) // the newline Encode adds
	case []interface{}:
		buf.WriteByte('[')
		for i, e := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, e); err != nil {
				return err
			}
		}
		buf.WriteByte(']')
	case objectEntries:
		return writeCanonical(buf, map[string]interface{}(x))
	case map[string]interface{}:
		keys := make([]string, 0, len(x))
		for k := range x {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, k := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			if err := writeCanonical(buf, k); err != nil {
				return err
			}
			buf.WriteByte(':')
			if err := writeCanonical(buf, x[k]); err != nil {
				return err
			}
		}
		buf.WriteByte('}')
	default:
		return fmt.Errorf("unexpected json type")
	}
	return nil
}

// ---- What the owner sees --------------------------------------------------------

// SummaryItem is one argument of the held call, shown to the owner.
type SummaryItem struct {
	Name  string `json:"name"`
	Value string `json:"value"`
}

// Summary limits: enough to read an order, small enough to keep an email and a
// table row short. A value that is cut is marked, so the owner knows the
// display is not the whole value.
const (
	maxSummaryItems = 16
	maxSummaryValue = 120
)

// OwnerSummary lists every top-level argument of the call, sorted by name,
// with the rule's field first. It is for the owner who is asked to approve:
// the owner is shown what will be sent, including values. It must never be
// put in a receipt, a log or any text the agent reads.
func OwnerSummary(params json.RawMessage, field string) []SummaryItem {
	var top struct {
		Arguments map[string]json.RawMessage `json:"arguments"`
	}
	if json.Unmarshal(params, &top) != nil {
		return nil
	}
	names := make([]string, 0, len(top.Arguments))
	for k := range top.Arguments {
		names = append(names, k)
	}
	sort.Slice(names, func(i, j int) bool {
		if (names[i] == field) != (names[j] == field) {
			return names[i] == field
		}
		return names[i] < names[j]
	})
	var out []SummaryItem
	for _, k := range names {
		if len(out) == maxSummaryItems {
			out = append(out, SummaryItem{Name: "more", Value: fmt.Sprintf("%d more argument(s) not shown", len(names)-maxSummaryItems)})
			break
		}
		out = append(out, SummaryItem{Name: cutText(k, 64), Value: displayValue(top.Arguments[k])})
	}
	return out
}

func displayValue(raw json.RawMessage) string {
	raw = bytes.TrimSpace(raw)
	var s string
	if len(raw) > 0 && raw[0] == '"' && json.Unmarshal(raw, &s) == nil {
		return cutText(s, maxSummaryValue)
	}
	var compact bytes.Buffer
	if json.Compact(&compact, raw) != nil {
		return "(unreadable)"
	}
	return cutText(compact.String(), maxSummaryValue)
}

// cutText shortens s for display. A NUL character cannot be stored in a
// Postgres text or jsonb value, so it is shown as U+2400 (the symbol for NUL);
// this is the owner's display only, the hash binds the real bytes.
func cutText(s string, max int) string {
	s = strings.ReplaceAll(s, "\x00", "\u2400")
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	r := []rune(s)
	return string(r[:max]) + "... (cut)"
}

// ---- The store -----------------------------------------------------------------

// ApprovalOutcome is what a store says about a call that is above a threshold.
type ApprovalOutcome int

const (
	// ApprovalConsumed: an approved, unexpired, unused approval matched and
	// was used up by this call. The call may be forwarded.
	ApprovalConsumed ApprovalOutcome = iota
	// ApprovalHeld: the call is waiting for the owner. Created is true when
	// this call made the waiting approval.
	ApprovalHeld
	// ApprovalDenied: the owner said no.
	ApprovalDenied
	// ApprovalExpired: the waiting or approved approval ran out unused. It is
	// reported once; the next identical call starts a new one.
	ApprovalExpired
	// ApprovalQueueFull: the token already has MaxPendingApprovals waiting, or
	// the tenant has MaxPendingApprovalsPerTenant waiting across its tokens.
	ApprovalQueueFull
)

// ApprovalRequest is one call above a threshold, as the store sees it.
type ApprovalRequest struct {
	TenantID string
	TokenID  string
	Tool     string
	// CallHash is CallHash(TokenID, Tool, params).
	CallHash string
	// Field and Above are the rule that held the call.
	Field string
	Above string
	// Summary is what the owner is shown. It holds the call's values.
	Summary []SummaryItem
	Now     time.Time
	TTL     time.Duration
}

// ApprovalDecision is the store's answer.
type ApprovalDecision struct {
	Outcome    ApprovalOutcome
	ApprovalID string
	ExpiresAt  time.Time
	// NeedsNotice is true when the store has reserved one of the tenant's
	// MaxApprovalEmailsPerHour email slots for this approval: the caller must
	// send the email now. It is false for an approval whose owner was already
	// told, whose email is in flight, or whose email is over the cap (the
	// approval is then held and listed, and counted in a later summary).
	NeedsNotice bool
}

// ApprovalStore is the durable record of held calls. Decide must be atomic: of
// any number of concurrent calls matching one approved approval, exactly one
// gets ApprovalConsumed. A store that cannot answer returns an error and the
// gateway refuses the call.
//
// Decide looks at the newest approval for (TenantID, TokenID, CallHash):
//
//	approved, not expired       -> mark used, ApprovalConsumed
//	waiting, not expired        -> ApprovalHeld (a repeat of the same call)
//	denied, not expired         -> ApprovalDenied
//	waiting or approved, expired-> mark expired, ApprovalExpired (reported once)
//	anything else               -> if the token has MaxPendingApprovals
//	                               waiting, or the tenant has
//	                               MaxPendingApprovalsPerTenant waiting across
//	                               its tokens, ApprovalQueueFull; otherwise make
//	                               a new waiting approval, ApprovalHeld
//
// Decide also decides whether the owner is emailed: NeedsNotice is true only
// when it reserved one of the tenant's MaxApprovalEmailsPerHour slots in the
// last ApprovalMailWindow, atomically with the decision.
type ApprovalStore interface {
	Decide(ctx context.Context, req ApprovalRequest) (ApprovalDecision, error)
	// NoticeFailed is called when the email Decide asked for could not be sent.
	// The slot it used stays used (a failing mail service does not widen the
	// cap), but the approval may be given a new slot by a repeat of the call.
	NoticeFailed(ctx context.Context, tenantID, approvalID string) error
	// MarkNotified records that the owner was told about the approval. The
	// owner is told once, when the approval is made (Decide's NeedsNotice), or
	// in a summary; a repeat of the held call never tells them again.
	MarkNotified(ctx context.Context, tenantID, approvalID string, at time.Time) error
}

// NewApprovalID returns a fresh approval id: "apr_" and 24 hex digits.
func NewApprovalID() (string, error) {
	var b [12]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", fmt.Errorf("approvals: no random id available: %w", err)
	}
	return "apr_" + hex.EncodeToString(b[:]), nil
}

// ValidApprovalID reports whether id has the shape NewApprovalID makes.
func ValidApprovalID(id string) bool {
	rest, ok := strings.CutPrefix(id, "apr_")
	if !ok || len(rest) != 24 {
		return false
	}
	for i := 0; i < len(rest); i++ {
		c := rest[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return false
		}
	}
	return true
}
