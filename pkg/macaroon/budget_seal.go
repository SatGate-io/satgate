package macaroon

import (
	"crypto/hmac"
	"strings"
)

// BudgetSealedPrefix returns how many leading caveats of mac a valid
// budget_bind seal covers: a seal at position i that matches the chain
// signature after the first i caveats covers caveats 0..i-1. It returns 0 when
// no seal is valid. A holder can append caveats offline, but cannot write a
// seal without the root key, so a caveat at a position at or after the returned
// count was not written by the issuer.
//
// ResolveIssuedBudgetID applies the same rule to budget_id caveats. A control
// that names a shared budget uses this to decide whether its reference counts.
func (s *Service) BudgetSealedPrefix(mac *Macaroon) int {
	if s == nil || mac == nil {
		return 0
	}
	prefixes := s.prefixSignatures(mac)
	covered := 0
	for i, caveat := range mac.Caveats {
		if !strings.HasPrefix(caveat, budgetBindCaveatPrefix) || i == 0 || i >= len(prefixes) {
			continue
		}
		want := strings.TrimPrefix(caveat, budgetBindCaveatPrefix)
		got := s.BudgetBind(prefixes[i])
		if want != "" && hmac.Equal([]byte(want), []byte(got)) && i > covered {
			covered = i
		}
	}
	return covered
}
