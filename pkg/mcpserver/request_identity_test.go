package mcpserver

import "testing"

func TestBudgetEventRequestIDPrefersCanonicalEnforcerIdentity(t *testing.T) {
	got := budgetEventRequestID("caller-id", &BudgetResult{RequestID: "billreq:v1:canonical"})
	if got != "billreq:v1:canonical" {
		t.Fatalf("got %q", got)
	}
}

func TestBudgetSpendEventDataKeepsCanonicalAndCallerIdentity(t *testing.T) {
	data := budgetSpendEventData("search", 3, 7, 10, "scoped:canonical", "caller-id", "control")
	if data["request_id"] != "scoped:canonical" || data["caller_request_id"] != "caller-id" {
		t.Fatalf("identity fields=%v", data)
	}
}

func TestBudgetEventRequestIDFallsBackForLegacyEnforcer(t *testing.T) {
	for _, result := range []*BudgetResult{nil, {}} {
		if got := budgetEventRequestID("caller-id", result); got != "caller-id" {
			t.Fatalf("got %q", got)
		}
	}
}
