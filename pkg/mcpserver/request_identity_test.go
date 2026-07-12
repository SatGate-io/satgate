package mcpserver

import "testing"

func TestBudgetEventRequestIDPrefersCanonicalEnforcerIdentity(t *testing.T) {
	got := budgetEventRequestID("caller-id", &BudgetResult{RequestID: "billreq:v1:canonical"})
	if got != "billreq:v1:canonical" {
		t.Fatalf("got %q", got)
	}
}

func TestBudgetEventRequestIDFallsBackForLegacyEnforcer(t *testing.T) {
	for _, result := range []*BudgetResult{nil, {}} {
		if got := budgetEventRequestID("caller-id", result); got != "caller-id" {
			t.Fatalf("got %q", got)
		}
	}
}
