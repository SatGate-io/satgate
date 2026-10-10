// Package toolcontrols holds the token constraints that need state across
// calls, where argument rules (pkg/argrules) look at one call only.
//
// A spending limit says: this token may spend at most Max in total, over a
// calendar day or an ISO week in a named time zone, summing one numeric
// argument (for example dollar_amount) of the calls to the listed tools that
// went through. The gateway reserves the amount before it forwards the call
// and gives it back only when it knows the call did not run: it was never
// sent, or the upstream answered with a JSON-RPC protocol error that means the
// request was not processed (-32700, -32600, -32601, -32602). A call that was
// sent and whose outcome is unknown keeps its reservation: a timeout, a
// connection closed after the send, an unreadable reply, any other JSON-RPC
// error, or a result with isError=true. An order the broker rejects therefore
// still counts toward the total.
//
// # Wire format
//
// Controls ride in a scope caveat, like argument rules, and for the same
// reason (a verifier that does not know them must deny, not ignore them):
//
//	scope = argrules:ctl:v1:<nonce>:<base64url(JSON)>
//
// The word starts with argrules:, so every verifier built before controls,
// and every verifier that knows rules but not controls, refuses a token that
// carries one (see macaroon.ArgumentRulesScopePrefix and
// argrules.ControlScopePrefix). The nonce is 32 hex characters, fresh for
// every minted caveat. It is also the identity of the limit: the counter is
// keyed by the caveat word, so a token delegated from this one carries the same
// word and spends from the same counter. A holder cannot get a fresh allowance
// by delegating.
//
// JSON document (version 1):
//
//	{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount",
//	  "max":"200","window":"day","tz":"America/New_York"}]}
//
// Unknown keys, repeated keys, a missing key other than tz, and anything over
// the size limits are errors at mint and at verify.
package toolcontrols

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"sort"
	"strings"
	"time"
	_ "time/tzdata" // zone names must resolve on a minimal image

	"github.com/satgate-io/satgate/pkg/argrules"
)

// Size limits, enforced when controls are minted and parsed.
const (
	MaxDocBytes             = 8 * 1024
	MaxControlCaveats       = 4 // control caveats on one token
	MaxSpendLimits          = 8 // spending limits over all caveats
	MaxToolsPerLimit        = 8
	MaxScale                = 8  // fractional digits kept for amounts
	MaxWholeDigits          = 10 // digits before the point in a limit
	DefaultTimeZone         = "America/New_York"
	version                 = 1
	scopePrefix             = argrules.ControlScopePrefix + "v1:"
	nonceHexLen             = 32
	maxJSONNesting          = 8
	unitsPerOne       int64 = 100000000 // 10^MaxScale
)

// DenialCodeSpendLimit is the stable code of a call refused by a spending
// limit. It is the error code in the JSON-RPC data and the denial_code signed
// into the receipt.
const DenialCodeSpendLimit = "TOOL_SPEND_LIMIT_DENIED"

// ErrInvalid wraps every parse and validation failure.
var ErrInvalid = errors.New("invalid tool controls")

func ctlErr(format string, args ...any) error {
	return fmt.Errorf("%w: %s", ErrInvalid, fmt.Sprintf(format, args...))
}

// Window is how long a spending limit's total runs before it starts again.
type Window string

const (
	WindowDay  Window = "day"
	WindowWeek Window = "week"
)

// SpendLimit caps the sum of one numeric argument over a window.
type SpendLimit struct {
	// Tools are the tools the limit covers, sorted. One limit counts the
	// calls to all of them together.
	Tools []string
	// Field is the top-level argument that is summed.
	Field string
	// Max is the owner's total for the window, as written (a plain decimal).
	Max string
	// Window is "day" or "week".
	Window Window
	// TZ is an IANA zone name. Windows start at midnight in this zone.
	TZ string

	// ID names the limit's counter: the hex SHA-256 prefix of the caveat word
	// it was read from. It is set by Collect and ParseScopeValue; a limit
	// built by hand has no ID and cannot be enforced.
	ID string

	max   *big.Rat
	units int64
	loc   *time.Location
}

// Doc is the decoded controls document.
type Doc struct {
	SpendLimits []SpendLimit
}

// Covers reports whether the limit counts calls to tool. Names match without
// regard to letter case: a scope that allows "Place_Crypto_Order" must not be
// a way round a limit written for "place_crypto_order", because an upstream
// may read the two as one tool. strings.EqualFold also folds the few
// non-ASCII letters that fold to an ASCII one (the Kelvin sign, the long s),
// which only widens what is counted. A tool name with anything outside the
// plain set (see SpendGate.Admit) is refused before this is asked.
func (l SpendLimit) Covers(tool string) bool {
	for _, t := range l.Tools {
		if strings.EqualFold(t, tool) {
			return true
		}
	}
	return false
}

// MaxUnits is Max as a whole number of 10^-8 units.
func (l SpendLimit) MaxUnits() int64 { return l.units }

// Location is the limit's time zone.
func (l SpendLimit) Location() *time.Location { return l.loc }

// ParseJSON parses the version-1 document strictly.
func ParseJSON(doc []byte) (*Doc, error) {
	if len(doc) == 0 {
		return nil, ctlErr("empty document")
	}
	if len(doc) > MaxDocBytes {
		return nil, ctlErr("document is larger than %d bytes", MaxDocBytes)
	}
	if err := argrules.CheckDuplicateKeys(doc, maxJSONNesting); err != nil {
		return nil, ctlErr("%v", err)
	}
	var top map[string]json.RawMessage
	if err := json.Unmarshal(doc, &top); err != nil || top == nil {
		return nil, ctlErr("document must be an object")
	}
	var limitsRaw json.RawMessage
	sawV := false
	for k, v := range top {
		switch k {
		case "v":
			if strings.TrimSpace(string(v)) != "1" {
				return nil, ctlErr("unsupported version")
			}
			sawV = true
		case "spend_limits":
			limitsRaw = v
		default:
			return nil, ctlErr("unknown document key %q", clip(k))
		}
	}
	if !sawV {
		return nil, ctlErr("document needs v")
	}
	out := &Doc{}
	if limitsRaw != nil {
		var list []json.RawMessage
		if err := json.Unmarshal(limitsRaw, &list); err != nil {
			return nil, ctlErr("spend_limits must be a list")
		}
		if len(list) == 0 {
			return nil, ctlErr("spend_limits is empty")
		}
		if len(list) > MaxSpendLimits {
			return nil, ctlErr("more than %d spending limits", MaxSpendLimits)
		}
		for _, raw := range list {
			l, err := parseSpendLimit(raw)
			if err != nil {
				return nil, err
			}
			out.SpendLimits = append(out.SpendLimits, l)
		}
	}
	if len(out.SpendLimits) == 0 {
		return nil, ctlErr("document has no controls")
	}
	return out, nil
}

func parseSpendLimit(raw json.RawMessage) (SpendLimit, error) {
	var obj map[string]json.RawMessage
	if err := json.Unmarshal(raw, &obj); err != nil || obj == nil {
		return SpendLimit{}, ctlErr("a spending limit must be an object")
	}
	var l SpendLimit
	var sawTools, sawField, sawMax, sawWindow bool
	for k, v := range obj {
		switch k {
		case "tools":
			if err := json.Unmarshal(v, &l.Tools); err != nil {
				return SpendLimit{}, ctlErr("tools must be a list of names")
			}
			sawTools = true
		case "field":
			if err := json.Unmarshal(v, &l.Field); err != nil {
				return SpendLimit{}, ctlErr("field must be a string")
			}
			sawField = true
		case "max":
			// Written as a JSON string so no tool rounds it.
			if err := json.Unmarshal(v, &l.Max); err != nil {
				return SpendLimit{}, ctlErr("max must be a string holding a plain decimal")
			}
			sawMax = true
		case "window":
			var w string
			if err := json.Unmarshal(v, &w); err != nil {
				return SpendLimit{}, ctlErr("window must be a string")
			}
			l.Window = Window(w)
			sawWindow = true
		case "tz":
			if err := json.Unmarshal(v, &l.TZ); err != nil {
				return SpendLimit{}, ctlErr("tz must be a string")
			}
		default:
			return SpendLimit{}, ctlErr("unknown spending limit key %q", clip(k))
		}
	}
	if !sawTools || !sawField || !sawMax || !sawWindow {
		return SpendLimit{}, ctlErr("a spending limit needs tools, field, max and window")
	}
	if err := l.validate(); err != nil {
		return SpendLimit{}, err
	}
	return l, nil
}

// validate checks every field and fills the derived values.
func (l *SpendLimit) validate() error {
	if len(l.Tools) == 0 || len(l.Tools) > MaxToolsPerLimit {
		return ctlErr("a spending limit covers 1 to %d tools", MaxToolsPerLimit)
	}
	seen := map[string]bool{}
	for _, t := range l.Tools {
		if !argrules.ValidToolName(t) {
			return ctlErr("tool name is not allowed")
		}
		if seen[t] {
			return ctlErr("a tool is listed twice")
		}
		seen[t] = true
	}
	sort.Strings(l.Tools)
	if !argrules.ValidFieldName(l.Field) {
		return ctlErr("field name is not allowed")
	}
	switch l.Window {
	case WindowDay, WindowWeek:
	default:
		return ctlErr("window must be %q or %q", WindowDay, WindowWeek)
	}
	if l.TZ == "" {
		l.TZ = DefaultTimeZone
	}
	loc, err := loadZone(l.TZ)
	if err != nil {
		return err
	}
	l.loc = loc
	d, err := argrules.ParseDecimal(l.Max)
	if err != nil {
		return ctlErr("max is not a plain decimal")
	}
	rat := d.Rat()
	if rat.Sign() <= 0 {
		return ctlErr("max must be above zero")
	}
	whole, frac, _ := strings.Cut(l.Max, ".")
	if len(whole) > MaxWholeDigits || len(frac) > MaxScale {
		return ctlErr("max has more than %d digits before or %d after the point", MaxWholeDigits, MaxScale)
	}
	units, ok := unitsOf(rat, false)
	if !ok || units <= 0 {
		return ctlErr("max is out of range")
	}
	l.max, l.units = rat, units
	return nil
}

func loadZone(name string) (*time.Location, error) {
	if len(name) > 64 || strings.ContainsAny(name, " \t\n\\") || name == "Local" || name == "UTC0" {
		return nil, ctlErr("tz is not an IANA zone name")
	}
	// An IANA name has a letter first; this keeps LoadLocation away from
	// file paths and the empty name (which means UTC).
	if c := name[0]; !(c >= 'A' && c <= 'Z') || strings.Contains(name, "..") || strings.HasPrefix(name, "/") {
		return nil, ctlErr("tz is not an IANA zone name")
	}
	loc, err := time.LoadLocation(name)
	if err != nil {
		return nil, ctlErr("tz is not an IANA zone name")
	}
	return loc, nil
}

// unitsOf converts an exact value to whole 10^-8 units. With ceil, a value
// with more fractional digits rounds up (an amount is never undercounted);
// without it, such a value is refused.
func unitsOf(r *big.Rat, ceil bool) (int64, bool) {
	scaled := new(big.Rat).Mul(r, new(big.Rat).SetInt64(unitsPerOne))
	q, rem := new(big.Int).QuoRem(scaled.Num(), scaled.Denom(), new(big.Int))
	if rem.Sign() != 0 {
		if !ceil {
			return 0, false
		}
		q.Add(q, big.NewInt(1))
	}
	if !q.IsInt64() {
		return 0, false
	}
	return q.Int64(), true
}

func clip(s string) string {
	if len(s) > 32 {
		return s[:32] + "..."
	}
	return s
}

// Marshal is the canonical document: limits in order, tools sorted, tz always
// written, keys in a fixed order. It is what a minting service acknowledges.
func Marshal(d *Doc) ([]byte, error) {
	if d == nil || len(d.SpendLimits) == 0 {
		return nil, ctlErr("no controls")
	}
	var b strings.Builder
	b.WriteString(`{"v":1,"spend_limits":[`)
	for i, l := range d.SpendLimits {
		cp := l
		cp.Tools = append([]string(nil), l.Tools...)
		if err := cp.validate(); err != nil {
			return nil, err
		}
		if i > 0 {
			b.WriteByte(',')
		}
		tools, _ := json.Marshal(cp.Tools)
		field, _ := json.Marshal(cp.Field)
		max, _ := json.Marshal(cp.Max)
		win, _ := json.Marshal(string(cp.Window))
		tz, _ := json.Marshal(cp.TZ)
		fmt.Fprintf(&b, `{"tools":%s,"field":%s,"max":%s,"window":%s,"tz":%s}`, tools, field, max, win, tz)
	}
	b.WriteString(`]}`)
	out := []byte(b.String())
	if len(out) > MaxDocBytes {
		return nil, ctlErr("document is larger than %d bytes", MaxDocBytes)
	}
	if _, err := ParseJSON(out); err != nil {
		return nil, err
	}
	return out, nil
}

// Canonical parses doc and returns its canonical form.
func Canonical(doc []byte) ([]byte, error) {
	d, err := ParseJSON(doc)
	if err != nil {
		return nil, err
	}
	return Marshal(d)
}

// ScopeValue is the scope caveat value that carries the controls. Each call
// draws a fresh nonce, so the same controls give a different word, and a
// different counter, every time.
func ScopeValue(d *Doc) (string, error) {
	doc, err := Marshal(d)
	if err != nil {
		return "", err
	}
	var n [16]byte
	if _, err := rand.Read(n[:]); err != nil {
		return "", fmt.Errorf("tool controls: no random nonce available: %w", err)
	}
	return scopePrefix + hex.EncodeToString(n[:]) + ":" + base64.RawURLEncoding.EncodeToString(doc), nil
}

// Caveat is the full caveat string, "scope = argrules:ctl:v1:...".
func Caveat(d *Doc) (string, error) {
	v, err := ScopeValue(d)
	if err != nil {
		return "", err
	}
	return "scope = " + v, nil
}

// limitID names the counter of limit index i in the word value.
func limitID(value string, i int) string {
	sum := sha256.Sum256([]byte(fmt.Sprintf("%s#%d", value, i)))
	return hex.EncodeToString(sum[:])[:24]
}

// ParseScopeValue decodes one control scope value. Every returned limit has
// its ID set.
func ParseScopeValue(value string) (*Doc, error) {
	if !strings.HasPrefix(value, scopePrefix) {
		return nil, ctlErr("unsupported controls version")
	}
	rest := strings.TrimPrefix(value, scopePrefix)
	nonce, enc, ok := strings.Cut(rest, ":")
	if !ok || len(nonce) != nonceHexLen {
		return nil, ctlErr("controls word has no nonce")
	}
	for i := 0; i < len(nonce); i++ {
		c := nonce[i]
		if !(c >= '0' && c <= '9' || c >= 'a' && c <= 'f') {
			return nil, ctlErr("controls word has no nonce")
		}
	}
	if len(enc) > base64.RawURLEncoding.EncodedLen(MaxDocBytes) {
		return nil, ctlErr("document is larger than %d bytes", MaxDocBytes)
	}
	doc, err := base64.RawURLEncoding.Strict().DecodeString(enc)
	if err != nil {
		return nil, ctlErr("controls payload is not base64url")
	}
	d, err := ParseJSON(doc)
	if err != nil {
		return nil, err
	}
	for i := range d.SpendLimits {
		d.SpendLimits[i].ID = limitID(value, i)
	}
	return d, nil
}

// Collect reads every control caveat on a token, in order, and returns its
// spending limits. A limit that names a tool the token's scope does not allow
// is kept: it is inert, because the tool cannot be called, and dropping it
// would let a child that narrowed its tool list lose a limit on a tool it
// still has. A mint checks tool names against the scope (see the gateway's
// MCPSpendLimitCaveats); a verifier does not.
func Collect(caveats []string) ([]SpendLimit, error) {
	var all []SpendLimit
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
		all = append(all, d.SpendLimits...)
		if len(all) > MaxSpendLimits {
			return nil, ctlErr("more than %d spending limits on one token", MaxSpendLimits)
		}
	}
	return all, nil
}

// HasControls reports whether any caveat is a control word, without parsing
// it. A word that does not parse still counts: it must be refused, not skipped.
func HasControls(caveats []string) bool {
	for _, c := range caveats {
		if v, ok := strings.CutPrefix(c, "scope = "); ok && argrules.IsControlScope(v) {
			return true
		}
	}
	return false
}

// FormatAmount writes an amount for a person. A field named for dollars
// ("dollar_amount", "usd") gets a $ sign; any other field is left as a plain
// number, since the unit is not known.
func FormatAmount(field, amount string) string {
	f := strings.ToLower(field)
	if strings.Contains(f, "dollar") || strings.Contains(f, "usd") {
		return "$" + amount
	}
	return amount
}

func (w Window) phrase() string {
	if w == WindowWeek {
		return "a week"
	}
	return "a day"
}

// Phrase is "a day" or "a week".
func (w Window) Phrase() string { return w.phrase() }

// Describe is the plain-words form shown to a token owner, one line per limit.
func Describe(limits []SpendLimit) []string {
	var out []string
	for _, l := range limits {
		line := fmt.Sprintf("%s: at most %s %s in total on %s",
			strings.Join(l.Tools, " and "), FormatAmount(l.Field, l.Max), l.Window.phrase(), l.Field)
		if l.TZ != DefaultTimeZone {
			line += " (days start at midnight " + l.TZ + " time)"
		}
		out = append(out, line)
	}
	return out
}

// WindowBounds gives the window containing now: its id (stable text for the
// counter key), start, and end (exclusive). A day is a calendar day in the
// limit's zone; a week is an ISO week (Monday to Sunday) in that zone.
func (l SpendLimit) WindowBounds(now time.Time) (id string, start, end time.Time) {
	t := now.In(l.loc)
	y, m, d := t.Date()
	switch l.Window {
	case WindowWeek:
		off := (int(t.Weekday()) + 6) % 7 // Monday = 0
		start = time.Date(y, m, d-off, 0, 0, 0, 0, l.loc)
		end = time.Date(y, m, d-off+7, 0, 0, 0, 0, l.loc)
		iy, iw := start.ISOWeek()
		id = fmt.Sprintf("%04d-W%02d", iy, iw)
	default:
		start = time.Date(y, m, d, 0, 0, 0, 0, l.loc)
		end = time.Date(y, m, d+1, 0, 0, 0, 0, l.loc)
		id = fmt.Sprintf("%04d-%02d-%02d", y, int(m), d)
	}
	return id, start, end
}
