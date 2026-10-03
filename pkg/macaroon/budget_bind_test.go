package macaroon

import (
	"strings"
	"testing"
	"time"
)

func TestHasScope_EveryCaveatMustGrant(t *testing.T) {
	added := &Macaroon{Caveats: []string{"scope = api:read", "scope = admin:write"}}
	if added.HasScope("admin:write") {
		t.Fatal("a second scope caveat must not add admin:write")
	}
	narrowed := &Macaroon{Caveats: []string{"scope = api:*", "scope = api:read"}}
	if !narrowed.HasScope("api:read") {
		t.Fatal("narrowed token should keep api:read")
	}
	if narrowed.HasScope("api:write") {
		t.Fatal("narrowed token should lose api:write")
	}
}

func TestResolveIssuedBudgetID_UnboundLaterIDRejected(t *testing.T) {
	svc, err := NewService("root")
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac.AddCaveat("budget_id", "budget-a")
	mac.Signature = svc.RecalculateSignature(mac)
	token := svc.Encode(mac)
	child, err := svc.DelegateWithoutVerify(token, []string{"budget_id = budget-other"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResolveIssuedBudgetID(child); err == nil || !strings.Contains(err.Error(), "not server-issued") {
		t.Fatalf("unbound later budget_id must be rejected, got %v", err)
	}
}

func TestResolveIssuedBudgetID_SealedChildAccepted(t *testing.T) {
	svc, err := NewService("root")
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac.AddCaveat("budget_id", "budget-parent")
	mac.Signature = svc.RecalculateSignature(mac)
	child, err := svc.Delegate(svc.Encode(mac), []string{"budget_id = budget-child", "budget_limit = 5"})
	if err != nil {
		t.Fatal(err)
	}
	sealed, err := svc.Delegate(svc.Encode(child), []string{"budget_bind = " + svc.BudgetBind(child.Signature)})
	if err != nil {
		t.Fatal(err)
	}
	got, err := svc.ResolveIssuedBudgetID(sealed)
	if err != nil {
		t.Fatal(err)
	}
	if got != "budget-child" {
		t.Fatalf("budget id = %s, want budget-child", got)
	}
	retargeted, err := svc.DelegateWithoutVerify(svc.Encode(sealed), []string{"budget_id = budget-other"})
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.ResolveIssuedBudgetID(retargeted); err == nil {
		t.Fatal("a budget_id appended after the seal must be rejected")
	}
}
