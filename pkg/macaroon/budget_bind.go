package macaroon

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"strings"
)

const (
	budgetIDCaveatPrefix   = "budget_id = "
	budgetBindCaveatPrefix = "budget_bind = "
	budgetBindDomain       = "satgate/budget_bind/v1\n"
)

// BudgetBind seals a chain signature with the root key. A later budget_id is
// treated as server-issued only when a budget_bind caveat matches this seal
// for the prefix that contains it.
func (s *Service) BudgetBind(chainSignature string) string {
	mac := hmac.New(sha256.New, s.rootKey)
	mac.Write([]byte(budgetBindDomain))
	mac.Write([]byte(chainSignature))
	return hex.EncodeToString(mac.Sum(nil))
}

// ResolveIssuedBudgetID returns the budget_id this macaroon may spend.
//
// Zero budget_id caveats returns "" so the caller can fall back to the token
// id. One distinct value is the issued id, including a repeated copy of that
// same value. A different later value is used only when a budget_bind caveat
// seals the chain prefix that contains it. An unbound later value is rejected.
func (s *Service) ResolveIssuedBudgetID(mac *Macaroon) (string, error) {
	if s == nil || mac == nil {
		return "", fmt.Errorf("budget_id check unavailable")
	}
	type hit struct {
		idx int
		id  string
	}
	var hits []hit
	for i, caveat := range mac.Caveats {
		if strings.HasPrefix(caveat, budgetIDCaveatPrefix) {
			hits = append(hits, hit{idx: i, id: strings.TrimPrefix(caveat, budgetIDCaveatPrefix)})
		}
	}
	if len(hits) == 0 {
		return "", nil
	}
	distinct := make(map[string]struct{}, len(hits))
	for _, h := range hits {
		distinct[h.id] = struct{}{}
	}
	if len(distinct) == 1 {
		return hits[0].id, nil
	}

	prefixes := s.prefixSignatures(mac)
	covered := -1
	for i, caveat := range mac.Caveats {
		if !strings.HasPrefix(caveat, budgetBindCaveatPrefix) || i == 0 {
			continue
		}
		want := strings.TrimPrefix(caveat, budgetBindCaveatPrefix)
		got := s.BudgetBind(prefixes[i])
		if want != "" && hmac.Equal([]byte(want), []byte(got)) && i-1 > covered {
			covered = i - 1
		}
	}
	last := hits[len(hits)-1]
	if last.idx > covered {
		return "", fmt.Errorf("budget_id caveat is not server-issued")
	}
	return last.id, nil
}

// prefixSignatures[i] is the chained signature after the first i caveats.
func (s *Service) prefixSignatures(mac *Macaroon) []string {
	h := hmac.New(sha256.New, s.rootKey)
	h.Write([]byte(mac.Identifier))
	sig := h.Sum(nil)
	out := make([]string, len(mac.Caveats)+1)
	out[0] = hex.EncodeToString(sig)
	for i, caveat := range mac.Caveats {
		h = hmac.New(sha256.New, sig)
		h.Write([]byte(caveat))
		sig = h.Sum(nil)
		out[i+1] = hex.EncodeToString(sig)
	}
	return out
}
