package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
)

// invalidWholeAmounts are caveat values that are not a whole number of credits
// above zero: zero, negative, fractional, malformed and out of range.
var invalidWholeAmounts = []string{
	"0", "0.00", "-0.00", "-1", "-5.00",
	"0.5", "1.5", "0.99",
	"nope", "", " 5", "+5", "05", "1e3", "0x10", "NaN", "Inf",
	"9223372036854775808", "99999999999999999999",
}

func TestFillTokenInfo_InvalidBudgetLimitRefusesToken(t *testing.T) {
	ctx := context.Background()
	for _, value := range invalidWholeAmounts {
		t.Run("appended "+value, func(t *testing.T) {
			svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "budget-a", "100")
			appended := appendCaveat(t, svc, token, "budget_limit = "+value)
			auth := &MacaroonAuthenticator{Service: svc}
			info, err := auth.Verify(ctx, appended)
			if err == nil || !strings.Contains(err.Error(), "invalid budget_limit caveat") {
				t.Fatalf("budget_limit = %q must refuse the token, got info=%+v err=%v", value, info, err)
			}
		})
	}
	for _, value := range []string{"0.5", "0", "-1", "12.50"} {
		t.Run("issued "+value, func(t *testing.T) {
			svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "budget-a", value)
			auth := &MacaroonAuthenticator{Service: svc}
			if _, err := auth.Verify(ctx, token); err == nil || !strings.Contains(err.Error(), "invalid budget_limit caveat") {
				t.Fatalf("issued budget_limit = %q must refuse the token, got %v", value, err)
			}
		})
	}
	// Another spelling of the caveat is not skipped. The macaroon layer
	// already refuses some spellings; the rest are refused here.
	for _, caveat := range []string{"budget_limit=1", "budget_limit  = 1", "budget_limit"} {
		t.Run("spelled "+caveat, func(t *testing.T) {
			svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "budget-a", "100")
			appended := appendCaveat(t, svc, token, caveat)
			auth := &MacaroonAuthenticator{Service: svc}
			if _, err := auth.Verify(ctx, appended); err == nil {
				t.Fatalf("%q must refuse the token", caveat)
			}
		})
	}
}

func TestFillTokenInfo_BudgetLimitNarrowestWholeWins(t *testing.T) {
	ctx := context.Background()
	for name, tc := range map[string]struct {
		extra []string
		want  int64
	}{
		"lower written with zeros": {[]string{"budget_limit = 5.00"}, 5},
		"higher is ignored":        {[]string{"budget_limit = 250"}, 100},
		"smallest of several":      {[]string{"budget_limit = 7", "budget_limit = 3", "budget_limit = 9"}, 3},
		"same value repeated":      {[]string{"budget_limit = 100"}, 100},
	} {
		t.Run(name, func(t *testing.T) {
			svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "budget-a", "100")
			token = appendCaveat(t, svc, token, tc.extra...)
			info, err := (&MacaroonAuthenticator{Service: svc}).Verify(ctx, token)
			if err != nil {
				t.Fatal(err)
			}
			if info.BudgetLimit != tc.want {
				t.Fatalf("budget limit = %d, want %d", info.BudgetLimit, tc.want)
			}
		})
	}
}

func TestFillTokenInfo_NoBudgetLimitUnchanged(t *testing.T) {
	svc, token := mintBudgetToken(t, "root", "gpt4_chat", "", "budget-a", "")
	info, err := (&MacaroonAuthenticator{Service: svc}).Verify(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	if info.BudgetLimit != 0 || info.DelegationBudget != 0 {
		t.Fatalf("limits = %d/%d, want 0/0 for a token with no budget caveats", info.BudgetLimit, info.DelegationBudget)
	}
}

func TestFillTokenInfo_InvalidDelegationBudgetRefusesToken(t *testing.T) {
	ctx := context.Background()
	for _, value := range invalidWholeAmounts {
		t.Run(value, func(t *testing.T) {
			svc, token := mintBudgetToken(t, "root", "api:*", "", "budget-parent", "100")
			token = appendCaveat(t, svc, token, "delegation_budget = 10", "delegation_budget = "+value)
			if _, err := (&MacaroonAuthenticator{Service: svc}).Verify(ctx, token); err == nil || !strings.Contains(err.Error(), "invalid delegation_budget caveat") {
				t.Fatalf("delegation_budget = %q must refuse the token, got %v", value, err)
			}
		})
	}
}

func TestDelegate_DelegationBudgetWrittenWithZerosNarrows(t *testing.T) {
	assertBudgetCap(t, "10", "delegation_budget = 5.00", 5)
}

// A budget-carrying token whose budget_limit cannot be read is refused at the
// tool call with the authentication-failure response, before any spend and
// before the upstream is called. It never runs with no budget.
func TestToolsCall_UnreadableBudgetLimitNeverRunsUnlimited(t *testing.T) {
	ctx := context.Background()
	call := func(t *testing.T, issued string, extra ...string) (*Response, *fakeMCPRouter) {
		t.Helper()
		svc, token := mintBudgetToken(t, "root", "search", "", "budget-a", issued)
		if len(extra) > 0 {
			token = appendCaveat(t, svc, token, extra...)
		}
		proxy := newEvidenceTestProxy(t)
		router := &fakeMCPRouter{}
		proxy.router = router
		proxy.config.Auth.Mode = "header"
		proxy.SetAuthenticator(&MacaroonAuthenticator{Service: svc})
		params, _ := json.Marshal(map[string]interface{}{
			"name":      "search",
			"arguments": map[string]interface{}{},
			"_meta":     map[string]string{"token": token},
		})
		resp, err := proxy.handleRequest(ctx, &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodToolsCall, Params: params})
		if err != nil {
			t.Fatal(err)
		}
		return resp, router
	}

	// The tool costs 10 (newEvidenceTestProxy).
	if resp, router := call(t, "20"); resp.Error != nil || !router.called {
		t.Fatalf("a call within a budget of 20 must pass, got %+v", resp.Error)
	}
	if resp, router := call(t, "20", "budget_limit = 5"); resp.Error == nil || router.called {
		t.Fatalf("a call over a narrowed budget of 5 must be refused, got %+v", resp.Error)
	}
	for _, value := range []string{"0.5", "0", "-1", "nope"} {
		t.Run(value, func(t *testing.T) {
			resp, router := call(t, "20", "budget_limit = "+value)
			if router.called {
				t.Fatalf("budget_limit = %q: the call reached the upstream", value)
			}
			if resp == nil || resp.Error == nil || resp.Error.Code != CodePolicyDenied || !strings.Contains(resp.Error.Message, "authentication failed") {
				t.Fatalf("budget_limit = %q: want the authentication-failure refusal, got %+v", value, resp)
			}
		})
	}
}
