package mcpserver

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Round 5: a shared budget written for place_crypto_order governs the same
// tool under the name the gateway gave it (robinhood-2_place_crypto_order).
// The gateway's own table says what a name stands for (ToolOriginRouter); the
// name is never cut apart.

type originRouter struct {
	scriptedRouter
	origins map[string]ToolOrigin
}

func (r *originRouter) ToolOriginForTenant(_ context.Context, _ string, tool string) ToolOrigin {
	if o, ok := r.origins[tool]; ok {
		return o
	}
	return ToolOrigin{}
}

func newOriginHarness(t *testing.T, origins map[string]ToolOrigin) (*budgetHarness, *originRouter) {
	t.Helper()
	h := newBudgetHarness(t)
	rr := &originRouter{origins: origins}
	h.proxy.SetUpstreamRouter(rr)
	return h, rr
}

const renamedOrder = "robinhood-2_place_crypto_order"

var renamedOrigins = map[string]ToolOrigin{
	"place_crypto_order": {Listed: true, Service: "robinhood", Original: "place_crypto_order"},
	renamedOrder:         {Listed: true, Renamed: true, Service: "robinhood-2", Original: "place_crypto_order"},
	// A third service lists a tool whose own name is the renamed name of the
	// second service's tool; the table lists it under its own prefix.
	"zed_" + renamedOrder: {Listed: true, Renamed: true, Service: "zed", Original: renamedOrder},
}

func TestRenamedOrderToolSpendsFromTheSavedBudget(t *testing.T) {
	h, rr := newOriginHarness(t, renamedOrigins)
	scope := "place_crypto_order," + renamedOrder
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", scope, budgetDoc(budgetA, "4", "2"))

	if r := h.call(t, tok, 1, renamedOrder, orderSym("BTC", "1")); r.Error != nil {
		t.Fatalf("allowed renamed order refused: %v", r.Error)
	}
	if h.sum() != 2*units { // the $4 total and the BTC cap, once each
		t.Fatalf("counters after one $1 order = %v, want $1 on the total and $1 on the BTC cap", h.totals())
	}
	if r := h.call(t, tok, 2, "place_crypto_order", orderSym("BTC", "1")); r.Error != nil {
		t.Fatalf("plain order refused: %v", r.Error)
	}
	if len(h.totals()) != 2 || h.sum() != 4*units {
		t.Fatalf("both names draw from the same two counters: %v", h.totals())
	}
	// The BTC cap is used up by the two names together.
	before := rr.n
	for i, tool := range []string{renamedOrder, "place_crypto_order"} {
		if r := h.call(t, tok, 10+i, tool, orderSym("BTC", "1")); r.Error == nil {
			t.Fatalf("%s past the BTC cap was admitted", tool)
		}
	}
	if rr.n != before {
		t.Fatalf("a refused order reached the upstream (%d -> %d)", before, rr.n)
	}
	// What was sent upstream is untouched: the router maps the name itself.
	if got := h.sum(); got != 4*units {
		t.Fatalf("a refusal moved a counter: %d", got)
	}
}

func TestRenamedOrderToolWithoutAnAnswerFromTheTableIsRefused(t *testing.T) {
	h, rr := newOriginHarness(t, map[string]ToolOrigin{
		renamedOrder:         {Err: ErrToolTableUnavailable},
		"place_crypto_order": {Err: errors.New("read failed")},
	})
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", "place_crypto_order,"+renamedOrder, budgetDoc(budgetA, "4", "2"))
	for i, tool := range []string{renamedOrder, "place_crypto_order"} {
		r := h.call(t, tok, 1+i, tool, orderSym("BTC", "1"))
		if r.Error == nil || r.Error.Message != MsgSharedBudgetCheckFailed {
			t.Fatalf("%s with no answer from the table: %+v", tool, r.Error)
		}
	}
	if rr.n != 0 || len(h.totals()) != 0 {
		t.Fatalf("upstream calls %d, counters %v: the refusal must come first", rr.n, h.totals())
	}
}

func TestToolNamedLikeARenamedOneIsNotAnOrderTool(t *testing.T) {
	h, rr := newOriginHarness(t, renamedOrigins)
	lit := "zed_" + renamedOrder
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", "place_crypto_order,"+lit, budgetDoc(budgetA, "4", "2"))
	if r := h.call(t, tok, 1, lit, orderSym("BTC", "1")); r.Error != nil {
		t.Fatalf("zed's own tool refused: %v", r.Error)
	}
	if h.sum() != 0 || rr.n != 1 {
		t.Fatalf("a tool that merely looks like a renamed order tool was debited: %v, upstream calls %d", h.totals(), rr.n)
	}
}

// Round 6: an order the token's shared budget does not cover is refused when
// it is placed, on the order the call resolves to, whatever existed when the
// key was made.

// A crypto-only budget; the key's tools name a renamed equity tool that did not
// exist yet. The table now lists it and it resolves to place_equity_order.
func TestOrderTheSharedBudgetDoesNotCoverIsRefusedAtPlacement(t *testing.T) {
	const futureEquity = "fourth_place_equity_order"
	origins := map[string]ToolOrigin{
		"place_crypto_order": {Listed: true, Service: "robinhood", Original: "place_crypto_order"},
		"place_equity_order": {Listed: true, Service: "robinhood", Original: "place_equity_order"},
		futureEquity:         {Listed: true, Renamed: true, Service: "fourth", Original: "place_equity_order"},
	}
	for _, tool := range []string{futureEquity, "place_equity_order"} {
		t.Run(tool, func(t *testing.T) {
			h, rr := newOriginHarness(t, origins)
			_, tok := sealedBudgetToken(t, "tenant-1", "own-1", "place_crypto_order,place_equity_order,"+futureEquity, budgetDoc(budgetA, "4", "2"))
			r := h.call(t, tok, 1, tool, orderSym("AAPL", "1"))
			if r.Error == nil || r.Error.Message != MsgSharedBudgetNotCovered {
				t.Fatalf("%s: %+v", tool, r.Error)
			}
			if rr.n != 0 || len(h.totals()) != 0 {
				t.Fatalf("upstream calls %d, counters %v: the refusal must come first", rr.n, h.totals())
			}
		})
	}
	// The budget's own order still works, and a tool the budget cannot govern
	// (a price quote) is not touched by the rule.
	h, rr := newOriginHarness(t, origins)
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", "place_crypto_order,place_equity_order,get_quote", budgetDoc(budgetA, "4", "2"))
	if r := h.call(t, tok, 1, "place_crypto_order", orderSym("BTC", "1")); r.Error != nil {
		t.Fatalf("covered order refused: %v", r.Error)
	}
	if r := h.call(t, tok, 2, "get_quote", `{"symbol":"BTC"}`); r.Error != nil {
		t.Fatalf("a tool no budget governs was refused: %v", r.Error)
	}
	if rr.n != 2 || h.sum() != 2*units {
		t.Fatalf("upstream %d, counters %v", rr.n, h.totals())
	}
}

// A limit the owner wrote on the token for the name as sent still governs the
// call, so the token's shared budget is not the only thing that can cover it.
func TestOwnLimitOnTheNameAsSentCoversARenamedOrder(t *testing.T) {
	origins := map[string]ToolOrigin{
		"place_crypto_order": {Listed: true, Service: "robinhood", Original: "place_crypto_order"},
		renamedOrder:         {Listed: true, Renamed: true, Service: "robinhood-2", Original: "place_crypto_order"},
	}
	h, rr := newOriginHarness(t, origins)
	own := `{"v":1,"spend_limits":[{"tools":["` + renamedOrder + `"],"field":"dollar_amount","max":"3","window":"day","tz":"America/New_York"}]}`
	// The token's shared budget is for equity orders: only the owner's own limit
	// covers the renamed crypto order.
	equityOnly := strings.ReplaceAll(budgetDoc(budgetB, "4", ""), "place_crypto_order", "place_equity_order")
	_, tok := sealedBudgetToken(t, "tenant-1", "own-1", "place_crypto_order,place_equity_order,"+renamedOrder, equityOnly, own)
	if r := h.call(t, tok, 1, renamedOrder, orderSym("BTC", "2")); r.Error != nil {
		t.Fatalf("an order the token's own limit covers was refused: %v", r.Error)
	}
	if r := h.call(t, tok, 2, renamedOrder, orderSym("BTC", "2")); r.Error == nil {
		t.Fatal("the token's own $3 limit did not govern the second order")
	}
	if rr.n != 1 {
		t.Fatalf("upstream calls %d, want 1", rr.n)
	}
}
