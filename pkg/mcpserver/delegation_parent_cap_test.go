package mcpserver

import (
	"context"
	"encoding/json"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
)

// spendCountingBudget counts the carves that reach the store, so a test can
// tell a refusal made before any carve from one the carve itself made.
type spendCountingBudget struct {
	*InMemoryBudgetEnforcer
	spends atomic.Int64
}

func (b *spendCountingBudget) Spend(ctx context.Context, tokenID, toolName string, cost int64, requestID string) (*BudgetResult, error) {
	b.spends.Add(1)
	return b.InMemoryBudgetEnforcer.Spend(ctx, tokenID, toolName, cost, requestID)
}

// narrowedParent mints a parent whose issuer sealed budget_limit = issued and
// whose holder appended a narrower budget_limit = narrowed, and initialises
// the parent's stored balance to balance.
func narrowedParent(t *testing.T, issued, narrowed string, balance int64) (*MacaroonAuthenticator, *TokenInfo, *spendCountingBudget) {
	t.Helper()
	svc, tok := mintBudgetToken(t, "root", "mcp:*", "", "budget-parent", issued)
	if narrowed != "" {
		tok = appendCaveat(t, svc, tok, "budget_limit = "+narrowed)
	}
	auth := &MacaroonAuthenticator{Service: svc}
	parent, err := auth.Verify(context.Background(), tok)
	if err != nil {
		t.Fatal(err)
	}
	b := &spendCountingBudget{InMemoryBudgetEnforcer: NewInMemoryBudgetEnforcer()}
	if balance > 0 {
		if err := b.Initialize(context.Background(), parent.BudgetID, balance); err != nil {
			t.Fatal(err)
		}
	}
	return auth, parent, b
}

func remainingOf(t *testing.T, b BudgetEnforcer, id string) int64 {
	t.Helper()
	r, err := b.Remaining(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// Reviewer case (direct API): a parent sealed at 100, narrowed to 20 by an
// appended caveat, with a stored balance of 100, asks for 30. The allocation
// is above the parent's effective cap, so it is refused before any mint or
// carve and the parent keeps its whole balance.
func TestDelegate_AllocationAboveInheritedCapRefused(t *testing.T) {
	auth, parent, b := narrowedParent(t, "100", "20", 100)
	result, err := NewDelegator(auth.Service, b).Delegate(context.Background(), parent, &DelegateParams{Budget: 30, Scope: "api:read"})
	if err == nil {
		t.Fatalf("allocation 30 above the parent's cap 20 was accepted (reported %d)", result.Budget)
	}
	if want := "insufficient parent budget"; err.Error() != want {
		t.Fatalf("refusal = %q, want %q", err.Error(), want)
	}
	if n := b.spends.Load(); n != 0 {
		t.Fatalf("the refusal reached the carve (%d spends)", n)
	}
	if got := remainingOf(t, b, parent.BudgetID); got != 100 {
		t.Fatalf("parent balance after refusal = %d, want 100", got)
	}
}

// Every accepted delegation reports the budget its child verifies with: for a
// parent whose effective cap is set by its narrowest budget_limit, by its
// stored balance, or by both, an allocation at or below the cap is accepted
// and verifies as itself; one above it is refused with the parent unchanged.
func TestDelegate_AcceptedAllocationEqualsChildVerifiedBudget(t *testing.T) {
	cases := []struct {
		name              string
		issued, narrowed  string
		balance           int64
		ask               int64
		wantAccepted      bool
		wantRefusalPrefix string
	}{
		{"narrowed cap, larger balance, at cap", "100", "20", 100, 20, true, ""},
		{"narrowed cap, larger balance, below cap", "100", "20", 100, 7, true, ""},
		{"narrowed cap, larger balance, above cap", "100", "20", 100, 21, false, "insufficient parent budget"},
		{"narrowed cap, larger balance, the issued amount", "100", "20", 100, 100, false, "insufficient parent budget"},
		{"issued cap only, uninitialised, at cap", "50", "", 0, 50, true, ""},
		{"issued cap only, uninitialised, above cap", "50", "", 0, 51, false, "insufficient parent budget"},
		{"balance below cap, at balance", "100", "", 10, 10, true, ""},
		{"balance below cap, above balance", "100", "", 10, 11, false, "insufficient parent budget"},
		{"balance below narrowed cap, above balance", "100", "40", 10, 30, false, "insufficient parent budget"},
		{"narrowed cap below balance, between them", "100", "40", 60, 50, false, "insufficient parent budget"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			auth, parent, b := narrowedParent(t, c.issued, c.narrowed, c.balance)
			before := remainingOf(t, b, parent.BudgetID)
			result, err := NewDelegator(auth.Service, b).Delegate(context.Background(), parent, &DelegateParams{Budget: c.ask, Scope: "api:read"})
			if !c.wantAccepted {
				if err == nil {
					t.Fatalf("allocation %d accepted (reported %d), want refused", c.ask, result.Budget)
				}
				if !strings.HasPrefix(err.Error(), c.wantRefusalPrefix) {
					t.Fatalf("refusal = %q, want prefix %q", err.Error(), c.wantRefusalPrefix)
				}
				if n := b.spends.Load(); n != 0 {
					t.Fatalf("the refusal reached the carve (%d spends)", n)
				}
				if got := remainingOf(t, b, parent.BudgetID); got != before {
					t.Fatalf("parent balance changed from %d to %d on a refusal", before, got)
				}
				return
			}
			if err != nil {
				t.Fatalf("allocation %d refused: %v", c.ask, err)
			}
			if result.Budget != c.ask {
				t.Fatalf("reported allocation %d, asked %d", result.Budget, c.ask)
			}
			info, err := auth.Verify(context.Background(), result.Token)
			if err != nil {
				t.Fatalf("accepted child does not verify: %v", err)
			}
			if info.BudgetLimit != result.Budget {
				t.Fatalf("reported allocation %d, child verifies with %d", result.Budget, info.BudgetLimit)
			}
			if got := remainingOf(t, b, info.BudgetID); got != result.Budget {
				t.Fatalf("child balance %d, reported allocation %d", got, result.Budget)
			}
		})
	}
}

// Reviewer case over HTTP: the same narrowed parent asks for 30 through
// satgate/delegate on /mcp. The call is refused and the parent keeps its
// balance; an ask at the cap is then accepted and the
// child verifies with exactly the reported allocation.
func TestDelegate_HTTPAllocationAboveInheritedCapRefused(t *testing.T) {
	svc, tok := mintBudgetToken(t, "root", "mcp:*", "", "budget-parent", "100")
	tok = appendCaveat(t, svc, tok, "budget_limit = 20")
	p := newEvidenceTestProxy(t)
	p.config.Auth.Mode = "header"
	p.SetAuthenticator(&MacaroonAuthenticator{Service: svc})
	p.delegator = NewDelegator(svc, p.budget)
	if err := p.budget.Initialize(context.Background(), "budget-parent", 100); err != nil {
		t.Fatal(err)
	}
	srv := httptest.NewServer(NewSSEServer(p, ":0").Handler())
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, tok)

	call := func(id int, budget int64) Response {
		t.Helper()
		body := map[string]any{"jsonrpc": "2.0", "id": id, "method": MethodSatGateDelegate, "params": map[string]any{"budget": budget, "scope": "api:read"}}
		status, _, raw := streamablePost(t, srv.URL+"/mcp", sid, tok, body)
		if status != 200 {
			t.Fatalf("HTTP status %d", status)
		}
		var resp Response
		if err := json.Unmarshal([]byte(raw), &resp); err != nil {
			t.Fatal(err)
		}
		return resp
	}

	resp := call(2, 30)
	if resp.Error == nil {
		var result DelegateResult
		_ = json.Unmarshal(resp.Result, &result)
		t.Fatalf("HTTP allocation 30 above the parent's cap 20 was accepted (reported %d)", result.Budget)
	}
	if want := "insufficient parent budget"; resp.Error.Message != want {
		t.Fatalf("refusal = %q, want %q", resp.Error.Message, want)
	}
	if got := remainingOf(t, p.budget, "budget-parent"); got != 100 {
		t.Fatalf("parent balance after refusal = %d, want 100", got)
	}

	resp = call(3, 20)
	if resp.Error != nil {
		t.Fatalf("allocation at the parent's cap refused: %s", resp.Error.Message)
	}
	var result DelegateResult
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatal(err)
	}
	info, err := (&MacaroonAuthenticator{Service: svc}).Verify(context.Background(), result.Token)
	if err != nil {
		t.Fatalf("issued child does not verify: %v", err)
	}
	if result.Budget != 20 || info.BudgetLimit != result.Budget {
		t.Fatalf("reported allocation %d, child verifies with %d", result.Budget, info.BudgetLimit)
	}
}

// A hand-built TokenInfo whose BudgetLimit field was left unset is still held
// to the budget_limit caveats its raw token carries.
func TestDelegate_ParentCapReadFromCaveatsNotOnlyTokenInfo(t *testing.T) {
	auth, parent, b := narrowedParent(t, "100", "20", 100)
	parent.BudgetLimit = 0
	_, err := NewDelegator(auth.Service, b).Delegate(context.Background(), parent, &DelegateParams{Budget: 30, Scope: "api:read"})
	if err == nil || !strings.HasPrefix(err.Error(), "insufficient parent budget") {
		t.Fatalf("err = %v, want the parent-cap refusal", err)
	}
	if n := b.spends.Load(); n != 0 {
		t.Fatalf("the refusal reached the carve (%d spends)", n)
	}
}
