package mcpserver

import (
	"context"
	"testing"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

func TestSellDollarBudgetHookGetsParsedOrder(t *testing.T) {
	for _, quantity := range []bool{false, true} {
		g, s := sellGate(t)
		args := `{"symbol":"TEST","side":"SeLl","dollar_amount":"20.01"}`
		if quantity {
			args = `{"symbol":"TEST","side":"SeLl","quantity":"0.125"}`
		}
		c := sellCall(ocToken(t, ocLimitDoc(2, ocEquity, "1")), "second_order", args)
		c.Origin = ToolOrigin{Listed: true, Renamed: true, Service: "robinhood", Original: ocEquity}
		calls := 0
		g.DollarBudgetExempt = func(_ context.Context, call GateCall, o toolcontrols.DollarBudgetOrder) (bool, error) {
			calls++
			if call.Tool != "second_order" || o.Service != "robinhood" || o.Tool != ocEquity || o.Side != "sell" || o.Symbol != "TEST" {
				t.Fatalf("wrong parsed order: %+v", o)
			}
			if quantity {
				if o.Quantity == nil || o.Quantity.String() != "0.125" || o.DollarAmount != nil {
					t.Fatal("quantity lost")
				}
			} else if o.DollarAmount == nil || o.DollarAmount.String() != "20.01" || o.Quantity != nil {
				t.Fatal("dollar amount lost")
			}
			return true, nil
		}
		if a := sellAdmit(t, g, c); ocRefusalCode(a) != "" {
			t.Fatal(a.Refusal)
		}
		if calls != 1 || s.reserves != 0 || s.releases != 0 {
			t.Fatalf("calls=%d reserves=%d releases=%d", calls, s.reserves, s.releases)
		}
	}
}

func TestSellDollarBudgetUnvaluedStillRefusedByOtherControls(t *testing.T) {
	for _, tool := range []string{ocCrypto, ocEquity} {
		g, s := sellGate(t)
		c := sellCall(ocToken(t, ocLimitDoc(0, tool, "1"), ocApprovalDoc(tool, "10")), tool, `{"symbol":"TEST","side":"sell","quantity":"2","type":"market"}`)
		c.Token.ArgumentRules = ocCapRules(t, tool, "100")
		if c.Token.CheckToolArguments(tool, c.Params) == nil {
			t.Fatal("unvalued per-order cap bypassed")
		}
		a := sellAdmit(t, ChainGates(g, NewApprovalGate(toolcontrols.NewMemoryApprovalStore(), nil)), c)
		if ocRefusalCode(a) != toolcontrols.DenialCodeApprovalUnreadable {
			t.Fatalf("unvalued approval: %s", ocRefusalCode(a))
		}
		if s.reserves != 0 || s.releases != 0 {
			t.Fatal("unvalued refusal touched dollars")
		}
	}
}

func TestSellDollarBudgetCreditsStillChargedAndExhaust(t *testing.T) {
	h := newSpendHarness(t, ocLimitDoc(0, ocEquity, "1"))
	rr := &originRouter{origins: map[string]ToolOrigin{ocEquity: {Listed: true, Service: "robinhood", Original: ocEquity}}}
	h.proxy.SetUpstreamRouter(rr)
	g := newTestSpendGate(h.store)
	sellTestHook(t, g, true, nil)
	h.proxy.SetCallGate(g)
	b := NewInMemoryBudgetEnforcer()
	if err := b.Initialize(context.Background(), "budget-spend", 1); err != nil {
		t.Fatal(err)
	}
	h.proxy.SetBudgetEnforcer(b)
	args := `{"symbol":"TEST","side":"sell","dollar_amount":"20"}`
	if r := h.call(t, 1, ocEquity, args); r.Error != nil {
		t.Fatal(r.Error)
	}
	left, err := b.Check(context.Background(), "budget-spend", 0)
	if err != nil || left.Remaining != 0 {
		t.Fatalf("credit charge missing: %+v %v", left, err)
	}
	if r := h.call(t, 2, ocEquity, args); r.Error == nil {
		t.Fatal("exhausted credits allowed a sell")
	}
	if rr.forwarded() != 1 || h.counter(t) != 0 {
		t.Fatalf("forwarded=%d dollars=%d", rr.forwarded(), h.counter(t))
	}
}
