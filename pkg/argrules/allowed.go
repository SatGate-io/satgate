package argrules

import (
	"strconv"
	"strings"
	"unicode"
)

// maxAllowedListItems bounds how many rule values one allowed-text lists. A
// one_of may hold up to MaxOneOfValues values; the rest are counted, not
// listed, so a refusal stays short.
const maxAllowedListItems = 16

// AllowedText says in plain words what one condition requires of its field,
// for example "dollar_amount must be at most 100" or "side must be one of:
// buy, sell". It is built from the rule alone. It takes no call argument, so
// it cannot carry one: every value in it is the owner's own configuration.
func (c Condition) AllowedText() string {
	switch {
	case c.Absent:
		return c.Field + " must not be set"
	case c.OneOf != nil:
		s := c.Field + " must be one of: " + listValues(c.OneOf)
		if c.ASCIICaseInsensitive {
			s += " (any letter case)"
		}
		return s
	case c.URL != nil:
		return c.Field + " must be " + describeURLCondition(c.URL)
	case c.Max != nil && c.Min != nil:
		return c.Field + " must be between " + c.Min.String() + " and " + c.Max.String()
	case c.Max != nil:
		return c.Field + " must be at most " + c.Max.String()
	case c.Min != nil:
		return c.Field + " must be at least " + c.Min.String()
	}
	return c.Field + " is limited by this token"
}

func describeURLCondition(u *URLCondition) string {
	s := "an https address on " + strings.Join(u.Hosts, " or ")
	if len(u.Ports) > 0 {
		ports := make([]string, 0, len(u.Ports))
		for _, p := range u.Ports {
			ports = append(ports, strconv.Itoa(p))
		}
		s += " (port " + strings.Join(ports, " or ") + ")"
	}
	if len(u.PathPrefixes) > 0 {
		s += " with a path starting " + strings.Join(u.PathPrefixes, " or ")
	}
	if len(u.PathSuffixes) > 0 {
		s += " with a path ending " + strings.Join(u.PathSuffixes, " or ")
	}
	return s
}

// listValues joins rule values for display. A value that is empty or holds a
// control character, a space or a comma is quoted so the list stays readable.
func listValues(vals []string) string {
	n := len(vals)
	extra := 0
	if n > maxAllowedListItems {
		extra = n - maxAllowedListItems
		n = maxAllowedListItems
	}
	out := make([]string, 0, n+1)
	for _, v := range vals[:n] {
		out = append(out, displayValue(v))
	}
	s := strings.Join(out, ", ")
	if extra > 0 {
		s += ", and " + strconv.Itoa(extra) + " more"
	}
	return s
}

func displayValue(v string) string {
	if v == "" {
		return `""`
	}
	for _, r := range v {
		if r == ',' || r == '"' || unicode.IsSpace(r) || unicode.IsControl(r) {
			return strconv.Quote(v)
		}
	}
	return v
}
