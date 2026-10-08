package mcpserver

import (
	"context"
	"strings"
	"testing"

	"github.com/satgate-io/satgate/pkg/argrules"
)

// B4-1: the gateway writer stores whatever this event publishes. A child of a
// limited parent must publish the inherited rule document, not only routes.
func TestDelegationPublishesInheritedRuleDocuments(t *testing.T) {
	svc, tok := argToken(t, "CallWixSiteAPI,ManageWixSite", wixReadOnlyDoc)
	mac, err := svc.AcceptingArgumentRules().Verify(tok)
	if err != nil {
		t.Fatal(err)
	}
	info, err := FillTokenInfo(svc.AcceptingArgumentRules(), mac, tok)
	if err != nil {
		t.Fatal(err)
	}
	budget := NewInMemoryBudgetEnforcer()
	if err := budget.Initialize(context.Background(), info.BudgetID, 50); err != nil {
		t.Fatal(err)
	}
	pub := NewChannelPublisher(2)
	d := NewDelegator(svc, budget)
	d.SetEventPublisher(pub)
	if _, err := d.Delegate(context.Background(), info, &DelegateParams{Budget: 1, Label: "child"}); err != nil {
		t.Fatal(err)
	}
	ev := <-pub.Events()
	if ev.Data[EventInheritedRulesKnown] != true {
		t.Fatalf("inheritedRulesKnown = %v", ev.Data[EventInheritedRulesKnown])
	}
	doc, _ := ev.Data[EventInheritedArgumentRules].(string)
	if doc == "" || !strings.Contains(doc, "CallWixSiteAPI") {
		t.Fatalf("inheritedArgumentRules = %q", doc)
	}
	if _, err := argrules.ParseJSON([]byte(doc)); err != nil {
		t.Fatalf("published document does not parse: %v", err)
	}
	scopes, _ := ev.Data[EventChildScopes].([]string)
	if len(scopes) == 0 {
		t.Fatal("child scopes were not published")
	}
}
