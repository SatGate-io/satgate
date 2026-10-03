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

// BudgetBind seals a chain signature with the root key. A budget_id is
// treated as server-issued only when a budget_bind caveat matches this seal
// for the prefix that contains it, including the first budget_id.
func (s *Service) BudgetBind(chainSignature string) string {
	return budgetBindHex(s.rootKey, chainSignature)
}

func budgetBindHex(rootKey []byte, chainSignature string) string {
	mac := hmac.New(sha256.New, rootKey)
	mac.Write([]byte(budgetBindDomain))
	mac.Write([]byte(chainSignature))
	return hex.EncodeToString(mac.Sum(nil))
}

func budgetBindCaveat(rootKey []byte, chainSignature string) string {
	return budgetBindCaveatPrefix + budgetBindHex(rootKey, chainSignature)
}

// AppendBudgetBind is the only writer of a budget_bind caveat. It attenuates
// token so the current signature — which must already cover the budget_id
// this token should spend — is sealed with rootKey. Every issuer calls this.
func AppendBudgetBind(rootKey []byte, token string) (string, error) {
	if len(rootKey) == 0 {
		return "", fmt.Errorf("budget bind root key required")
	}
	if token == "" {
		return "", fmt.Errorf("token required")
	}
	decoder, err := NewService("budget-bind-decode")
	if err != nil {
		return "", err
	}
	mac, err := decoder.Decode(token)
	if err != nil {
		return "", err
	}
	child, err := decoder.DelegateWithoutVerify(token, []string{budgetBindCaveat(rootKey, mac.Signature)})
	if err != nil {
		return "", err
	}
	return decoder.Encode(child), nil
}

// AppendBudgetBind seals token with this service's root key.
func (s *Service) AppendBudgetBind(token string) (*Macaroon, error) {
	if s == nil {
		return nil, fmt.Errorf("budget bind unavailable")
	}
	sealed, err := AppendBudgetBind(s.rootKey, token)
	if err != nil {
		return nil, err
	}
	return s.Decode(sealed)
}

// ResolveIssuedBudgetID returns the budget_id this macaroon may spend.
//
// Zero budget_id caveats returns "" so the caller can fall back to the token
// id. A budget_id is accepted only when a budget_bind caveat seals the chain
// prefix that contains it, including the first one. A repeated copy of that
// sealed id is not a widen. A later different id without a new matching seal
// is rejected. A seal copied from another chain does not match.
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

	prefixes := s.prefixSignatures(mac)
	covered := -1
	for i, caveat := range mac.Caveats {
		if !strings.HasPrefix(caveat, budgetBindCaveatPrefix) || i == 0 || i >= len(prefixes) {
			continue
		}
		want := strings.TrimPrefix(caveat, budgetBindCaveatPrefix)
		got := s.BudgetBind(prefixes[i])
		if want != "" && hmac.Equal([]byte(want), []byte(got)) && i-1 > covered {
			covered = i - 1
		}
	}

	issued := ""
	seenSealed := false
	for _, h := range hits {
		if h.idx <= covered {
			issued = h.id
			seenSealed = true
		}
	}
	if !seenSealed {
		return "", fmt.Errorf("budget_id caveat is not server-issued")
	}
	for _, h := range hits {
		if h.idx > covered && h.id != issued {
			return "", fmt.Errorf("budget_id caveat is not server-issued")
		}
	}
	return issued, nil
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
