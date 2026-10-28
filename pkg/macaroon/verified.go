package macaroon

import (
	"time"
)

// VerifiedMacaroon represents a verified and decoded macaroon token.
type VerifiedMacaroon struct {
	Signature string
	Scope     string
	Caveats   map[string]string
	ExpiresAt time.Time
}

// HasScope reports whether the verified macaroon grants required. An empty
// Scope keeps full access; any other Scope is compared with ScopeGrants.
func (v *VerifiedMacaroon) HasScope(required string) bool {
	if v.Scope == "" {
		return true
	}
	return ScopeGrants(v.Scope, required)
}
