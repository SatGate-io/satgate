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
// (repeated keys, invalid UTF-8, nesting too deep, not an object).
var ErrCallUnreadable = errors.New("the call could not be read")

const maxCallNesting = 16

// CallHash binds an approval to one call: it is the SHA-256 of the token id,
// the tool and the canonical form of the call's arguments. The canonical form
// sorts object keys, drops insignificant whitespace and keeps every number as
// written, so the same call sent again hashes the same however the client
// formats it, and any changed value hashes differently. The JSON-RPC id and
// _meta are not part of the call.
//
// The hash is one-way. It is safe to keep next to an approval; it is not a
// way to read the call back.
func CallHash(tokenID, tool string, params json.RawMessage) (string, error) {
	if tokenID == "" || tool == "" {
		return "", ErrCallUnreadable
	}
	if !utf8.Valid(params) {
		return "", ErrCallUnreadable
	}
	if err := argrules.CheckDuplicateKeys(params, maxCallNesting); err != nil {
		return "", ErrCallUnreadable
	}
	var top struct {
		Name      string          `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &top); err != nil || top.Name != tool {
		return "", ErrCallUnreadable
	}
	args := bytes.TrimSpace(top.Arguments)
	if len(args) == 0 || bytes.Equal(args, []byte("null")) {
		args = []byte("{}")
	}
	if args[0] != '{' {
		return "", ErrCallUnreadable
	}
	var buf bytes.Buffer
	dec := json.NewDecoder(bytes.NewReader(args))
	dec.UseNumber()
	var v interface{}
	if err := dec.Decode(&v); err != nil {
		return "", ErrCallUnreadable
	}
	if err := writeCanonical(&buf, v); err != nil {
		return "", ErrCallUnreadable
	}
	h := sha256.New()
	h.Write([]byte("satgate-approval-call-v1\n"))
	h.Write([]byte(tokenID))
	h.Write([]byte{'\n'})
	h.Write([]byte(tool))
	h.Write([]byte{'\n'})
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
	case json.Number:
		buf.WriteString(x.String())
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

func cutText(s string, max int) string {
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
	// ApprovalQueueFull: the token already has MaxPendingApprovals waiting.
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
	// NeedsNotice is true when the owner has not been told about a waiting
	// approval yet (the first call, or an earlier notice failed).
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
//	                               waiting, ApprovalQueueFull; otherwise make a
//	                               new waiting approval, ApprovalHeld
type ApprovalStore interface {
	Decide(ctx context.Context, req ApprovalRequest) (ApprovalDecision, error)
	// MarkNotified records that the owner was told about the approval, so a
	// repeat of the held call does not tell them again.
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
