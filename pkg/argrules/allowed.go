package argrules

import (
	"regexp"
	"strconv"
	"strings"
)

// maxAllowedListItems bounds how many rule values one agent-facing text lists.
// A one_of may hold up to MaxOneOfValues values; the rest are counted, not
// listed, so a refusal stays short.
const maxAllowedListItems = 16

// Agent-safe disclosure policy.
//
// Rule values are the owner's configuration, and some of them can be secrets:
// a URL path prefix may be a webhook path or a capability URL, and a one_of
// value may be an API key. The text an agent's model reads (a refusal, the
// satgate_budget_check limits) therefore goes through this policy and never
// copies a rule literal unless it is known to be plain. The owner-facing
// Describe and the mint acknowledgement keep the full values.
//
//   - max, min, between, absent: shown (numbers are not secrets).
//   - url: hosts and ports shown; path prefixes and suffixes never shown.
//   - one_of: values are shown only when BOTH the field name is on the fixed
//     allowlist agentVisibleFields (and has no sensitive word in it) AND every
//     value is plain (see plainOneOf). Otherwise only the count is shown. The
//     value's shape never decides on its own: a 24-character API key looks
//     like a plain symbol, so shape and length cannot tell them apart.
//   - anything else: "<field> is limited by this token".
const (
	maxPlainValueLen = 24
)

var plainValueRE = regexp.MustCompile(`^[A-Za-z0-9._:-]+$`)

// credentialPrefixes are the starts of well-known credential formats. A value
// that starts with one is never shown to an agent, whatever its length.
var credentialPrefixes = []string{
	"sk-", "sk_", "pk_", "rk_", "ghp_", "gho_", "github_pat_", "xox", "eyJ",
	"AKIA", "ASIA", "AIza", "glpat-", "shpat_", "whsec_",
}

// agentVisibleFields is the fixed allowlist of field names whose one_of values
// an agent may be shown (matched case-insensitively, whole name). Each is a
// known, non-secret trading or request parameter: a value the agent must send
// to make a valid call, and one the owner chose in order to restrict the
// agent. Telling the agent "side must be one of: buy" is what lets it fix the
// call. A field that is not here is count-only, however harmless its values
// look. Add a name only if its values can never be a credential.
//
//	side, type, order_type, time_in_force, position_effect, market_hours,
//	direction                 order parameters (enumerations)
//	symbol, symbols, chain_symbol, underlying_type, asset_class, currency
//	                          what is being traded (tickers, instrument kinds)
//	account_number, rhs_account_number
//	                          which account the call acts on; the agent must
//	                          send it and already holds it
//	interval, bounds, span    market-data query parameters (enumerations)
//	method, http_method       the HTTP verb
var agentVisibleFields = map[string]bool{
	"side": true, "type": true, "order_type": true, "symbol": true,
	"symbols": true, "chain_symbol": true, "account_number": true,
	"rhs_account_number": true, "time_in_force": true, "market_hours": true,
	"direction": true, "position_effect": true, "currency": true,
	"asset_class": true, "underlying_type": true, "interval": true,
	"bounds": true, "span": true, "method": true, "http_method": true,
}

// sensitiveFieldWords: a field name containing any of these (case-insensitive)
// is always count-only, even if someone adds it to agentVisibleFields.
var sensitiveFieldWords = []string{
	"key", "token", "secret", "password", "passwd", "auth", "credential",
	"signature", "cookie", "session",
}

// oneOfVisible reports whether a one_of condition's values may be shown to an
// agent: the field is allowlisted, its name is not sensitive, and every value
// is plain.
func oneOfVisible(c Condition) bool {
	name := strings.ToLower(c.Field)
	for _, w := range sensitiveFieldWords {
		if strings.Contains(name, w) {
			return false
		}
	}
	return agentVisibleFields[name] && plainOneOf(c.OneOf)
}

func plainValue(v string) bool {
	if len(v) == 0 || len(v) > maxPlainValueLen || !plainValueRE.MatchString(v) {
		return false
	}
	for _, p := range credentialPrefixes {
		if strings.HasPrefix(v, p) {
			return false
		}
	}
	return true
}

// plainOneOf reports whether every value may be shown to an agent.
func plainOneOf(vals []string) bool {
	for _, v := range vals {
		if !plainValue(v) {
			return false
		}
	}
	return true
}

// AgentText says in plain words what one condition requires of its field, for
// the agent that was refused, for example "dollar_amount must be at most 100"
// or "side must be one of: buy, sell". It is built from the rule alone and it
// takes no call argument, so it cannot carry one. It also follows the
// agent-safe disclosure policy above: it never shows a URL path or a one_of
// value that is not plain.
func (c Condition) AgentText() string {
	switch {
	case c.Absent:
		return c.Field + " must not be set"
	case c.OneOf != nil:
		var s string
		if oneOfVisible(c) {
			s = c.Field + " must be one of: " + listValues(c.OneOf, ", ")
		} else {
			s = c.Field + " must be one of the " + strconv.Itoa(len(c.OneOf)) + " values this token allows"
		}
		if c.ASCIICaseInsensitive {
			s += " (any letter case)"
		}
		return s
	case c.URL != nil:
		return c.Field + " must be " + agentURL(c.URL)
	case c.Max != nil && c.Min != nil:
		return c.Field + " must be between " + c.Min.String() + " and " + c.Max.String()
	case c.Max != nil:
		return c.Field + " must be at most " + c.Max.String()
	case c.Min != nil:
		return c.Field + " must be at least " + c.Min.String()
	}
	return c.Field + " is limited by this token"
}

// agentLine is the same condition as one clause of an AgentLines line, worded
// like the owner-facing clause ("amt is at most 100") so the two read alike.
func (c Condition) agentLine() string {
	switch {
	case c.Absent:
		return c.Field + " is not sent"
	case c.OneOf != nil:
		suffix := ""
		if c.ASCIICaseInsensitive {
			suffix = " (any letter case)"
		}
		if oneOfVisible(c) {
			return c.Field + " is " + listValues(c.OneOf, " or ") + suffix
		}
		return c.Field + " is one of the " + strconv.Itoa(len(c.OneOf)) + " values this token allows" + suffix
	case c.URL != nil:
		return c.Field + " is " + agentURL(c.URL)
	case c.Max != nil && c.Min != nil:
		return c.Field + " is between " + c.Min.String() + " and " + c.Max.String()
	case c.Max != nil:
		return c.Field + " is at most " + c.Max.String()
	case c.Min != nil:
		return c.Field + " is at least " + c.Min.String()
	}
	return c.Field + " is limited by this token"
}

// agentURL shows hosts and ports. A path prefix or suffix can be a credential
// (a webhook path, a capability URL), so paths are never shown.
func agentURL(u *URLCondition) string {
	s := "an https address on " + strings.Join(u.Hosts, " or ")
	if len(u.Ports) > 0 {
		ports := make([]string, 0, len(u.Ports))
		for _, p := range u.Ports {
			ports = append(ports, strconv.Itoa(p))
		}
		s += " (port " + strings.Join(ports, " or ") + ")"
	}
	if len(u.PathPrefixes) > 0 || len(u.PathSuffixes) > 0 {
		s += " with a path this token allows"
	}
	return s
}

// AgentLines is Describe for an agent: one line per rule, in the same words,
// with the agent-safe disclosure policy applied to every condition.
func AgentLines(rules []Rule) []string {
	return describeRules(rules, Condition.agentLine)
}

// listValues joins plain rule values (letters, digits and . _ : -), so none
// needs quoting. Past maxAllowedListItems the rest are counted, not listed.
func listValues(vals []string, sep string) string {
	n := len(vals)
	extra := 0
	if n > maxAllowedListItems {
		extra = n - maxAllowedListItems
		n = maxAllowedListItems
	}
	s := strings.Join(vals[:n], sep)
	if extra > 0 {
		s += ", and " + strconv.Itoa(extra) + " more"
	}
	return s
}
