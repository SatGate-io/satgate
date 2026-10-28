package macaroon

import (
	"strings"
	"unicode"
)

// ScopeGrants reports whether one granted scope satisfies one required scope.
// Every scope check in this module goes through it.
//
// A scope is one or more segments joined by ":", for example "api:read" or
// "api:users:read". The leading segments name its namespace. A required scope
// is satisfied by exactly three kinds of granted scope:
//
//   - the same scope ("api:read" grants "api:read");
//   - a wildcard "<ns>:*" when the required scope starts with "<ns>:"
//     ("api:*" grants "api:read" and "api:users:read"; "api:users:*" grants
//     "api:users:read" but not "api:orders:read");
//   - the bare wildcard "*", which grants every well-formed scope.
//
// Nothing else matches. A wildcard never leaves its namespace: "api:*" does
// not grant "mcp:search", "l402" or "read", and "mcp:*" does not grant
// "api:read". "api:*" does not grant "api" itself. Comparison is exact and
// case-sensitive.
//
// It fails closed. It returns false when either scope is empty or malformed.
// A scope is malformed when it contains whitespace, a control character or a
// comma, has an empty segment (a leading, trailing or doubled ":"), or has a
// "*" anywhere except as the whole last segment of a granted scope. A
// required scope never contains "*". Callers that read a list of scopes split
// it themselves and call ScopeGrants once per entry.
func ScopeGrants(granted, required string) bool {
	if !wellFormedScope(required, false) || !wellFormedScope(granted, true) {
		return false
	}
	if granted == "*" || granted == required {
		return true
	}
	if ns, ok := strings.CutSuffix(granted, ":*"); ok {
		return strings.HasPrefix(required, ns+":")
	}
	return false
}

// wellFormedScope reports whether s is a scope ScopeGrants can compare. With
// wildcard set, the bare "*" and a last segment of "*" are allowed.
func wellFormedScope(s string, wildcard bool) bool {
	if s == "" {
		return false
	}
	if s == "*" {
		return wildcard
	}
	segments := strings.Split(s, ":")
	for i, seg := range segments {
		if seg == "" {
			return false
		}
		if strings.Contains(seg, "*") {
			if !wildcard || seg != "*" || i != len(segments)-1 {
				return false
			}
		}
		for _, r := range seg {
			if r == ',' || unicode.IsSpace(r) || unicode.IsControl(r) {
				return false
			}
		}
	}
	return true
}
