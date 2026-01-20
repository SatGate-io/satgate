package macaroon

import (
	"strings"
	"time"
)

// VerifiedMacaroon represents a verified and decoded macaroon token.
type VerifiedMacaroon struct {
	Signature string
	Scope     string
	Caveats   map[string]string
	ExpiresAt time.Time
}

// HasScope checks if the verified macaroon has a specific scope.
func (v *VerifiedMacaroon) HasScope(required string) bool {
	if v.Scope == "" || v.Scope == "*" {
		return true
	}
	if v.Scope == required {
		return true
	}
	// Check wildcard patterns
	if strings.HasSuffix(v.Scope, "*") {
		prefix := strings.TrimSuffix(v.Scope, "*")
		return strings.HasPrefix(required, prefix)
	}
	return false
}
