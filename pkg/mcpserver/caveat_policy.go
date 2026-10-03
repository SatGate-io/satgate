package mcpserver

import (
	"fmt"
	"strconv"
	"strings"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// ConsistentCaveat returns the value of key when every caveat with that name
// carries the same value. Differing values are rejected. A repeated copy of
// the same value is allowed.
func ConsistentCaveat(mac *macaroon.Macaroon, key string) (string, error) {
	if mac == nil {
		return "", nil
	}
	prefix := key + " = "
	var first string
	seen := false
	for _, caveat := range mac.Caveats {
		if !strings.HasPrefix(caveat, prefix) {
			continue
		}
		value := strings.TrimPrefix(caveat, prefix)
		if !seen {
			first = value
			seen = true
			continue
		}
		if value != first {
			return "", fmt.Errorf("conflicting %s caveats", key)
		}
	}
	return first, nil
}

// FillTokenInfo builds the verified identity. Scope, tenant, and budget
// caveats can only narrow the token: every scope caveat must allow a tool,
// tenant_id values must agree, budget_limit is the minimum, and a later
// different budget_id must be sealed by the server that holds the root key.
func FillTokenInfo(svc *macaroon.Service, mac *macaroon.Macaroon, token string) (*TokenInfo, error) {
	if mac == nil {
		return nil, fmt.Errorf("macaroon required")
	}
	tenantID, err := ConsistentCaveat(mac, "tenant_id")
	if err != nil {
		return nil, err
	}
	if err := rejectWideningScope(mac); err != nil {
		return nil, err
	}
	budgetID, err := svc.ResolveIssuedBudgetID(mac)
	if err != nil {
		return nil, err
	}

	tokenID := hashToken(mac.Identifier + mac.Signature)
	info := &TokenInfo{
		TokenID:  tokenID,
		BudgetID: tokenID,
		Scope:    displayScope(mac),
		TenantID: tenantID,
		Raw:      mac,
		RawToken: token,
	}
	if budgetID != "" {
		info.BudgetID = budgetID
	}
	if limit, ok := narrowestBudgetLimit(mac); ok {
		info.BudgetLimit = limit
	}
	if depth := mac.GetCaveat("delegation_depth"); depth != "" {
		if v, err := strconv.Atoi(depth); err == nil {
			info.DelegationDepth = v
		}
	}
	if budget := mac.GetCaveat("delegation_budget"); budget != "" {
		if v, err := strconv.ParseFloat(budget, 64); err == nil {
			info.DelegationBudget = int64(v)
		}
	}
	for _, caveat := range mac.Caveats {
		if strings.HasPrefix(caveat, "parent = ") {
			info.Depth++
		}
	}
	info.ParentTokenID = mac.GetCaveat("parent")
	return info, nil
}

// AllowsTool reports whether every scope caveat on the token allows toolName.
// A token with no scope caveat, and a non-macaroon identity, keeps the
// previous Scope-string check.
func (t *TokenInfo) AllowsTool(toolName string) bool {
	if t == nil {
		return false
	}
	if t.Raw != nil {
		saw := false
		for _, caveat := range t.Raw.Caveats {
			if !strings.HasPrefix(caveat, "scope = ") {
				continue
			}
			saw = true
			if !matchScope(strings.TrimPrefix(caveat, "scope = "), toolName) {
				return false
			}
		}
		if saw {
			return true
		}
	}
	if t.Scope == "" || t.Scope == "*" || t.Scope == "api:*" || t.Scope == "mcp:*" {
		return true
	}
	return matchScope(t.Scope, toolName)
}

func rejectWideningScope(mac *macaroon.Macaroon) error {
	sawRestricted := false
	for _, caveat := range mac.Caveats {
		if !strings.HasPrefix(caveat, "scope = ") {
			continue
		}
		value := strings.TrimPrefix(caveat, "scope = ")
		if scopeCaveatUniversal(value) && sawRestricted {
			return fmt.Errorf("scope caveat widens an earlier restriction")
		}
		if !scopeCaveatUniversal(value) {
			sawRestricted = true
		}
	}
	return nil
}

func scopeCaveatUniversal(scope string) bool {
	for _, part := range strings.Split(scope, ",") {
		for _, word := range strings.Fields(part) {
			if word == "*" || word == "api:*" || word == "mcp:*" {
				return true
			}
		}
	}
	return false
}

func displayScope(mac *macaroon.Macaroon) string {
	var scopes []string
	for _, caveat := range mac.Caveats {
		if strings.HasPrefix(caveat, "scope = ") {
			scopes = append(scopes, strings.TrimPrefix(caveat, "scope = "))
		}
	}
	if len(scopes) == 0 {
		return ""
	}
	for i := len(scopes) - 1; i >= 0; i-- {
		if !scopeCaveatUniversal(scopes[i]) {
			return scopes[i]
		}
	}
	return scopes[0]
}

func narrowestBudgetLimit(mac *macaroon.Macaroon) (int64, bool) {
	found := false
	var min int64
	for _, caveat := range mac.Caveats {
		if !strings.HasPrefix(caveat, "budget_limit = ") {
			continue
		}
		raw := strings.TrimPrefix(caveat, "budget_limit = ")
		v, err := strconv.ParseFloat(raw, 64)
		if err != nil {
			continue
		}
		iv := int64(v)
		if !found || iv < min {
			min = iv
			found = true
		}
	}
	return min, found
}
