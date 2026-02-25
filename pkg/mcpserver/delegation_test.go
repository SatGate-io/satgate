package mcpserver

import (
	"context"
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
	mac, err := svc.Mint("api:*", time.Now().Add(time.Hour))
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
		Scope:    "api:*",
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

	mac, err := svc.Mint("api:*", time.Now().Add(time.Hour))
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

	mac, err := svc.Mint("api:*", time.Now().Add(time.Hour))
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
