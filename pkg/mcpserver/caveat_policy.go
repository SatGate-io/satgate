package mcpserver

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"github.com/satgate-io/satgate/pkg/argrules"
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
// tenant_id values must agree, budget_limit is the minimum, a budget_id is
// spent only when a budget_bind seals it, and delegation caps are the
// minimum positive value. A non-numeric or zero caveat does not clear a cap.
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
	// Rules are read here so a token with rules that do not parse, that name
	// a tool the scope does not allow, or that are over the size limits is
	// refused at verify, not at the first call.
	rules, err := argrules.Collect(mac.Caveats, matchScope)
	if err != nil {
		return nil, err
	}
	budgetID, err := svc.ResolveIssuedBudgetID(mac)
	if err != nil {
		return nil, err
	}

	tokenID := hashToken(mac.Identifier + mac.Signature)
	info := &TokenInfo{
		TokenID:       tokenID,
		BudgetID:      tokenID,
		Scope:         displayScope(mac),
		TenantID:      tenantID,
		ArgumentRules: rules,
		Raw:           mac,
		RawToken:      token,
	}
	if budgetID != "" {
		info.BudgetID = budgetID
	}
	if limit, ok := narrowestBudgetLimit(mac); ok {
		info.BudgetLimit = limit
	}
	info.DelegationDepth = narrowestPositiveInt(mac, "delegation_depth")
	info.DelegationBudget = narrowestPositiveAmount(mac, "delegation_budget")
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
// previous Scope-string check. Argument rules are not read here; see
// CheckToolArguments.
func (t *TokenInfo) AllowsTool(toolName string) bool {
	if t == nil {
		return false
	}
	// A name with the reserved rule prefix is never a tool a token allows,
	// whatever its scope says: see macaroon.ArgumentRulesScopePrefix.
	if macaroon.IsReservedToolName(toolName) {
		return false
	}
	if t.Raw != nil {
		saw := false
		for _, caveat := range t.Raw.Caveats {
			if !strings.HasPrefix(caveat, "scope = ") {
				continue
			}
			if argrules.IsRuleScope(strings.TrimPrefix(caveat, "scope = ")) {
				// A rule caveat limits arguments, not tool names. The
				// tool-name check of an older verifier compares the whole
				// word to the tool name, which no real tool equals, so it
				// denies every real tool; that is how it fails closed.
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
		if argrules.IsRuleScope(value) {
			continue
		}
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
		if strings.HasPrefix(caveat, "scope = ") && !argrules.IsRuleScope(strings.TrimPrefix(caveat, "scope = ")) {
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

// narrowestPositiveInt is the minimum successfully parsed positive integer.
// A non-numeric value, zero, or a negative value does not set or clear a cap.
// Zero means unlimited only when no positive cap was parsed.
func narrowestPositiveInt(mac *macaroon.Macaroon, key string) int {
	found := false
	min := 0
	prefix := key + " = "
	for _, caveat := range mac.Caveats {
		if !strings.HasPrefix(caveat, prefix) {
			continue
		}
		v, err := strconv.Atoi(strings.TrimPrefix(caveat, prefix))
		if err != nil || v <= 0 {
			continue
		}
		if !found || v < min {
			min = v
			found = true
		}
	}
	if !found {
		return 0
	}
	return min
}

// narrowestPositiveAmount is the minimum successfully parsed positive amount.
// Parse failures and non-positive values are ignored, so they cannot clear
// an earlier cap.
func narrowestPositiveAmount(mac *macaroon.Macaroon, key string) int64 {
	found := false
	var min int64
	prefix := key + " = "
	for _, caveat := range mac.Caveats {
		if !strings.HasPrefix(caveat, prefix) {
			continue
		}
		v, err := strconv.ParseFloat(strings.TrimPrefix(caveat, prefix), 64)
		if err != nil {
			continue
		}
		iv := int64(v)
		if iv <= 0 {
			continue
		}
		if !found || iv < min {
			min = iv
			found = true
		}
	}
	if !found {
		return 0
	}
	return min
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

// CheckToolArguments applies the token's argument rules to a tools/call and
// returns nil when the call may go on. params is the exact params object that
// will be forwarded; nothing is rewritten, so what is checked is what the
// upstream reads. A token with no rules is not parsed at all.
func (t *TokenInfo) CheckToolArguments(toolName string, params json.RawMessage) *argrules.Denial {
	if t == nil {
		return nil
	}
	rules := t.ArgumentRules
	if t.Raw != nil {
		// Recollect from the caveats, so a TokenInfo whose field was not
		// filled (a hand-built or older constructor) cannot skip its rules.
		var err error
		rules, err = argrules.Collect(t.Raw.Caveats, matchScope)
		if err != nil {
			return &argrules.Denial{Reason: argrules.ReasonRulesInvalid}
		}
	}
	return argrules.Check(rules, toolName, params)
}

// CollectArgumentRules reads and validates the argument rules in a caveat
// list, with the same checks a verifier applies: each rule must parse, name a
// tool the scope caveats allow, and stay inside the size limits. A minting
// service calls it on the caveats it is about to sign, so a token it cannot
// verify is never issued.
func CollectArgumentRules(caveats []string) ([]argrules.Rule, error) {
	return argrules.Collect(caveats, matchScope)
}

// ScopeValueAllowsTool reports whether one scope caveat value (the text after
// "scope = ") allows toolName, by the same matching AllowsTool uses.
func ScopeValueAllowsTool(scope, toolName string) bool {
	return matchScope(scope, toolName)
}
