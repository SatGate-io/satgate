package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

func TestDelegator_Delegate(t *testing.T) {
	ctx := context.Background()

	// Setup macaroon service
	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}

	// Setup budget
	budget := NewInMemoryBudgetEnforcer()

	// Create delegator
	d := NewDelegator(svc, budget)

	// Mint parent token
	mac, err := svc.Mint("mcp:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	parentToken := svc.Encode(mac)
	parentTokenID := hashToken(mac.Identifier)

	// Initialize parent budget
	if err := budget.Initialize(ctx, parentTokenID, 1000); err != nil {
		t.Fatal(err)
	}

	// Create parent token info
	parentInfo := &TokenInfo{
		TokenID:  parentTokenID,
		BudgetID: parentTokenID,
		Scope:    "mcp:*",
		RawToken: parentToken,
	}

	// Delegate 300 credits to child
	result, err := d.Delegate(ctx, parentInfo, &DelegateParams{
		Budget: 300,
		Label:  "research-agent",
		Scope:  "api:read",
	})
	if err != nil {
		t.Fatal(err)
	}

	if result.Budget != 300 {
		t.Errorf("expected child budget 300, got %d", result.Budget)
	}
	if result.ParentRemaining != 700 {
		t.Errorf("expected parent remaining 700, got %d", result.ParentRemaining)
	}
	if result.Token == "" {
		t.Error("expected non-empty child token")
	}
	if result.TokenID == "" {
		t.Error("expected non-empty child token ID")
	}

	// Verify child budget exists
	childRemaining, err := budget.Remaining(ctx, result.BudgetID)
	if err != nil {
		t.Fatal(err)
	}
	if childRemaining != 300 {
		t.Errorf("expected child remaining 300, got %d", childRemaining)
	}

	// Verify parent budget was decremented
	parentRemaining, err := budget.Remaining(ctx, parentTokenID)
	if err != nil {
		t.Fatal(err)
	}
	if parentRemaining != 700 {
		t.Errorf("expected parent remaining 700, got %d", parentRemaining)
	}

	// Verify child token is valid
	childMac, err := svc.Verify(result.Token)
	if err != nil {
		t.Fatal(err)
	}
	if childMac.GetCaveat("parent") != parentTokenID {
		t.Errorf("expected parent caveat %s, got %s", parentTokenID, childMac.GetCaveat("parent"))
	}
	if childMac.GetCaveat("label") != "research-agent" {
		t.Errorf("expected label research-agent, got %s", childMac.GetCaveat("label"))
	}
}

func TestDelegator_InsufficientBudget(t *testing.T) {
	ctx := context.Background()

	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}

	budget := NewInMemoryBudgetEnforcer()
	d := NewDelegator(svc, budget)

	mac, err := svc.Mint("mcp:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	parentToken := svc.Encode(mac)
	parentTokenID := hashToken(mac.Identifier)

	// Only 100 credits
	budget.Initialize(ctx, parentTokenID, 100)

	parentInfo := &TokenInfo{
		TokenID:  parentTokenID,
		BudgetID: parentTokenID,
		RawToken: parentToken,
	}

	// Try to delegate 500 — should fail
	_, err = d.Delegate(ctx, parentInfo, &DelegateParams{Budget: 500})
	if err == nil {
		t.Fatal("expected error for insufficient budget")
	}

	// Parent budget should be unchanged
	remaining, _ := budget.Remaining(ctx, parentTokenID)
	if remaining != 100 {
		t.Errorf("expected parent budget unchanged at 100, got %d", remaining)
	}
}

func TestDelegator_MultipleDelegations(t *testing.T) {
	ctx := context.Background()

	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}

	budget := NewInMemoryBudgetEnforcer()
	d := NewDelegator(svc, budget)

	mac, err := svc.Mint("mcp:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	parentToken := svc.Encode(mac)
	parentTokenID := hashToken(mac.Identifier)
	budget.Initialize(ctx, parentTokenID, 1000)

	parentInfo := &TokenInfo{
		TokenID:  parentTokenID,
		BudgetID: parentTokenID,
		RawToken: parentToken,
	}

	// Delegate 3 times
	var childIDs []string
	for i, label := range []string{"agent-1", "agent-2", "agent-3"} {
		result, err := d.Delegate(ctx, parentInfo, &DelegateParams{
			Budget: 200,
			Label:  label,
		})
		if err != nil {
			t.Fatalf("delegation %d failed: %v", i, err)
		}
		childIDs = append(childIDs, result.BudgetID)
	}

	// Parent should have 400 left (1000 - 3*200)
	remaining, _ := budget.Remaining(ctx, parentTokenID)
	if remaining != 400 {
		t.Errorf("expected parent remaining 400, got %d", remaining)
	}

	// Each child should have 200
	for i, id := range childIDs {
		r, _ := budget.Remaining(ctx, id)
		if r != 200 {
			t.Errorf("child %d: expected 200, got %d", i, r)
		}
	}

	// Spend from child 0
	_, err = budget.Spend(ctx, childIDs[0], "web_search", 50, "req1")
	if err != nil {
		t.Fatal(err)
	}

	// Child 0 should have 150, others unchanged
	r0, _ := budget.Remaining(ctx, childIDs[0])
	r1, _ := budget.Remaining(ctx, childIDs[1])
	if r0 != 150 {
		t.Errorf("child 0: expected 150, got %d", r0)
	}
	if r1 != 200 {
		t.Errorf("child 1: expected 200 (unchanged), got %d", r1)
	}

	// Parent still 400 (child spending doesn't touch parent)
	pr, _ := budget.Remaining(ctx, parentTokenID)
	if pr != 400 {
		t.Errorf("parent: expected 400, got %d", pr)
	}
}

// recordingBudget counts what the delegator asks of the budget store.
type recordingBudget struct {
	*InMemoryBudgetEnforcer
	spends      []int64
	initialized []string
}

func (r *recordingBudget) Spend(ctx context.Context, id, tool string, cost int64, req string) (*BudgetResult, error) {
	r.spends = append(r.spends, cost)
	return r.InMemoryBudgetEnforcer.Spend(ctx, id, tool, cost, req)
}

func (r *recordingBudget) Initialize(ctx context.Context, id string, credits int64) error {
	r.initialized = append(r.initialized, id)
	return r.InMemoryBudgetEnforcer.Initialize(ctx, id, credits)
}

// unextendableParent is a parent token this delegator's service cannot
// extend: it was signed under another key (as a token the authenticator
// accepts under a different key would be).
func unextendableParent(t *testing.T) (*macaroon.Service, *TokenInfo) {
	t.Helper()
	other, err := macaroon.NewService("some-other-root-key")
	if err != nil {
		t.Fatal(err)
	}
	mac, err := other.Mint("mcp:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}
	id := hashToken(mac.Identifier)
	return svc, &TokenInfo{TokenID: id, BudgetID: id, RawToken: other.Encode(mac)}
}

const (
	wantCannotDelegate = "This token can't create child tokens here. Ask the owner to create a token for this agent on the Tokens page."
	wantServerSide     = "This child token couldn't be created because of a problem on our side. Try again in a few minutes."
)

// A parent signed under another key gets the Tokens-page sentence, nothing is
// carved, and the parent's budget is not touched at all.
func TestDelegateRefusesParentItCannotExtend(t *testing.T) {
	ctx := context.Background()
	for _, budget := range []int64{0, 40} {
		svc, parent := unextendableParent(t)
		rb := &recordingBudget{InMemoryBudgetEnforcer: NewInMemoryBudgetEnforcer()}
		_ = rb.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
		parent.BudgetLimit = 100
		res, err := NewDelegator(svc, rb).Delegate(ctx, parent, &DelegateParams{Budget: budget, Label: "x"})
		if err == nil || res != nil {
			t.Fatalf("budget %d: want a refusal, got result %v err %v", budget, res, err)
		}
		if !errors.Is(err, ErrParentCannotDelegate) || err.Error() != wantCannotDelegate {
			t.Fatalf("budget %d: message = %q", budget, err.Error())
		}
		if len(rb.spends) != 0 || len(rb.initialized) != 0 {
			t.Fatalf("budget %d: the parent's budget was touched: spends %v initializes %v", budget, rb.spends, rb.initialized)
		}
		if n, _ := rb.Remaining(ctx, parent.BudgetID); n != 100 {
			t.Fatalf("budget %d: parent = %d, want 100", budget, n)
		}
	}
}

// Through HandleDelegate the same refusal is a JSON-RPC error with the
// existing code and data.
func TestHandleDelegateRefusalIsJSONRPCError(t *testing.T) {
	svc, parent := unextendableParent(t)
	d := NewDelegator(svc, NewInMemoryBudgetEnforcer())
	for _, budget := range []int64{0, 5} {
		raw, _ := json.Marshal(DelegateParams{Budget: budget})
		resp, err := d.HandleDelegate(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodSatGateDelegate, Params: raw}, parent)
		if err != nil || resp == nil || resp.Error == nil {
			t.Fatalf("budget %d: want a JSON-RPC error, got %+v %v", budget, resp, err)
		}
		if resp.Error.Code != CodeBudgetExhausted || resp.Error.Message != wantCannotDelegate {
			t.Fatalf("budget %d: code %d message %q", budget, resp.Error.Code, resp.Error.Message)
		}
		data, _ := json.Marshal(resp.Error.Data)
		if string(data) != `{"error":"delegation_failed"}` {
			t.Fatalf("budget %d: data %s", budget, data)
		}
	}
}

// The other delegation errors keep their messages.
func TestDelegateOtherErrorsKeepTheirMessages(t *testing.T) {
	ctx := context.Background()
	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}
	mac, _ := svc.Mint("mcp:*", time.Now().Add(time.Hour))
	d := NewDelegator(svc, NewInMemoryBudgetEnforcer())
	base := TokenInfo{TokenID: "p", BudgetID: "p", RawToken: svc.Encode(mac)}

	depth := base
	depth.DelegationDepth, depth.Depth = 1, 1
	capped := base
	capped.DelegationBudget = 10
	for _, tc := range []struct {
		name   string
		parent TokenInfo
		params DelegateParams
		want   string
	}{
		{"depth", depth, DelegateParams{Budget: 1}, "delegation depth limit reached (1)"},
		{"cap", capped, DelegateParams{Budget: 11}, "delegation budget exceeds cap: requested 11, max 10 per delegation"},
	} {
		p := tc.parent
		_, err := d.Delegate(ctx, &p, &tc.params)
		if err == nil || err.Error() != tc.want {
			t.Fatalf("%s: %v, want %q", tc.name, err, tc.want)
		}
	}
	// insufficient parent budget is covered by TestDelegator_InsufficientBudget
}
