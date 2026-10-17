package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// ---- helpers ----

const (
	ocEquity = "place_equity_order"
	ocCrypto = "place_crypto_order"
)

// ocNow is a Wednesday: its day and week windows are open and the next day and
// the next Monday are new ones.
var ocNow = time.Date(2026, 10, 14, 15, 0, 0, 0, time.UTC)

// ocClock is the store's clock: the tests decide "now" per call, and a store
// on the real clock would sweep counters of a day that is already past.
func ocClock() time.Time { return ocNow }

func ocParams(tool, args string) json.RawMessage {
	return json.RawMessage(`{"name":"` + tool + `","arguments":` + args + `}`)
}

func ocSymbol(tool string) string {
	if tool == ocCrypto {
		return "BTC"
	}
	return "AAPL"
}

// ocArgs is an arguments object with the tool's symbol first, so symbol caps
// can read it.
func ocArgs(tool, fields string) string {
	return `{"symbol":"` + ocSymbol(tool) + `",` + fields + `}`
}

const ocDayBuy = `"side":"buy","type":"limit","time_in_force":"gfd"`

// ocToken builds a token from control documents, the way a minted token
// carries them: every word is sealed, so shared-budget limits count.
func ocToken(t *testing.T, docs ...string) *TokenInfo {
	t.Helper()
	var caveats []string
	for _, d := range docs {
		caveats = append(caveats, controlWord(t, d))
	}
	limits, err := toolcontrols.CollectSealed(caveats, len(caveats))
	if err != nil {
		t.Fatal(err)
	}
	rules, err := toolcontrols.CollectApprovals(caveats)
	if err != nil {
		t.Fatal(err)
	}
	return &TokenInfo{TokenID: "tok-1", TenantID: "tenant-1", SpendLimits: limits, ApprovalRules: rules}
}

// ocLimitKinds are the five places a spend limit lives. Each document counts
// the one tool at the one max, with value order_max_cost.
var ocLimitKinds = []struct {
	name  string
	entry func(tool, max string) string
}{
	{"token day", func(tool, max string) string {
		return `{"tools":["` + tool + `"],"field":"dollar_amount","max":"` + max + `","window":"day","value":"order_max_cost"}`
	}},
	{"token week", func(tool, max string) string {
		return `{"tools":["` + tool + `"],"field":"dollar_amount","max":"` + max + `","window":"week","value":"order_max_cost"}`
	}},
	{"shared total", func(tool, max string) string {
		return `{"tools":["` + tool + `"],"field":"dollar_amount","max":"` + max + `","window":"day","budget":"` + budgetA + `","value":"order_max_cost"}`
	}},
	{"shared symbol cap", func(tool, max string) string {
		return `{"tools":["` + tool + `"],"field":"dollar_amount","max":"` + max + `","window":"day","budget":"` + budgetA + `","symbols":["` + ocSymbol(tool) + `"],"value":"order_max_cost"}`
	}},
	{"token symbol cap", func(tool, max string) string {
		return `{"tools":["` + tool + `"],"field":"dollar_amount","max":"` + max + `","window":"day","symbols":["` + ocSymbol(tool) + `"],"value":"order_max_cost"}`
	}},
}

// doc is the one limit as a control document.
func ocLimitDoc(i int, tool, max string) string {
	return `{"v":1,"spend_limits":[` + ocLimitKinds[i].entry(tool, max) + `]}`
}

// ocAllLimitsDoc is all five limits in one document, as a token carries them.
func ocAllLimitsDoc(tool, max string) string {
	var parts []string
	for _, k := range ocLimitKinds {
		parts = append(parts, k.entry(tool, max))
	}
	return `{"v":1,"spend_limits":[` + strings.Join(parts, ",") + `]}`
}

func ocApprovalDoc(tool, above string) string {
	return `{"v":1,"approvals":[{"tools":["` + tool + `"],"field":"dollar_amount","above":"` + above + `","value":"order_max_cost"}]}`
}

func ocCapRules(t *testing.T, tool, max string) []argrules.Rule {
	t.Helper()
	r, err := argrules.ParseJSON([]byte(`{"v":1,"rules":[{"tool":"` + tool + `","shapes":[{"order":{"max_cost":"` + max + `"}}]}]}`))
	if err != nil {
		t.Fatal(err)
	}
	return r
}

// ocAdmitSpend runs the spend gate over a fresh store with a fixed clock.
func ocAdmitSpend(t *testing.T, tok *TokenInfo, tool string, params json.RawMessage) (*GateAdmission, *toolcontrols.MemoryStore) {
	t.Helper()
	store := toolcontrols.NewMemoryStore(ocClock)
	g := newTestSpendGate(store)
	adm, err := g.Admit(context.Background(), GateCall{Token: tok, Tool: tool, Params: params, RequestID: "req-1", Now: ocNow})
	if err != nil {
		t.Fatalf("spend gate: %v", err)
	}
	return adm, store
}

func ocTotals(store *toolcontrols.MemoryStore) []int64 {
	var out []int64
	for _, k := range store.Keys() {
		out = append(out, store.TotalOf(k))
	}
	return out
}

func ocAllEqual(totals []int64, want int64) bool {
	for _, v := range totals {
		if v != want {
			return false
		}
	}
	return len(totals) > 0
}

// ocAdmitApproval runs the approval gate over a fresh store.
func ocAdmitApproval(t *testing.T, tok *TokenInfo, tool string, params json.RawMessage) (*GateAdmission, *toolcontrols.MemoryApprovalStore, *fakeNotifier) {
	t.Helper()
	store := toolcontrols.NewMemoryApprovalStore()
	n := &fakeNotifier{}
	g := NewApprovalGate(store, n)
	g.Synchronous = true
	adm, err := g.Admit(context.Background(), GateCall{Token: tok, Tool: tool, Params: params, RequestID: "req-1", Now: ocNow})
	if err != nil {
		t.Fatalf("approval gate: %v", err)
	}
	return adm, store, n
}

func ocRefusalCode(adm *GateAdmission) string {
	if adm == nil || adm.Refusal == nil {
		return ""
	}
	return adm.Refusal.Code
}

// ---- the drift test ----

type ocCase struct {
	name  string
	tool  string
	args  string                // the arguments object, symbol included
	units int64                 // above zero: the call is valued at this many 10^-8 units
	why   argrules.OrderCostWhy // otherwise: the fixed reason it is refused
}

func ocValued(name, tool, fields string, units int64) ocCase {
	return ocCase{name: name, tool: tool, args: ocArgs(tool, fields), units: units}
}

func ocRefused(name, tool, fields string, why argrules.OrderCostWhy) ocCase {
	return ocCase{name: name, tool: tool, args: ocArgs(tool, fields), why: why}
}

// The units in this table are worked out by hand, apart from the code under
// test, so the test checks the value as well as the agreement.
var ocCases = []ocCase{
	// By dollars.
	ocValued("equity dollars", ocEquity, `"dollar_amount":"100"`, 10000000000),
	ocValued("equity dollars as a json number", ocEquity, `"dollar_amount":12.5`, 1250000000),
	ocValued("equity dollars, one unit", ocEquity, `"dollar_amount":"0.00000001"`, 1),
	ocValued("equity dollars, rounds up", ocEquity, `"dollar_amount":"0.000000011"`, 2),
	ocValued("crypto dollars", ocCrypto, `"dollar_amount":"50"`, 5000000000),
	ocValued("crypto dollars, half", ocCrypto, `"dollar_amount":"0.5"`, 50000000),
	ocValued("sell by dollars is valued as ever", ocEquity, `"side":"sell","dollar_amount":"20"`, 2000000000),
	ocValued("market buy by dollars", ocEquity, `"side":"buy","type":"market","dollar_amount":"33.33"`, 3333000000),
	ocValued("dollars at the top of the range", ocEquity, `"dollar_amount":"9999999999.99999999"`, 999999999999999999),
	// Limit buys by shares (equity, day orders).
	ocValued("1 x 100.00", ocEquity, ocDayBuy+`,"quantity":"1","limit_price":"100.00"`, 10000000000),
	ocValued("1 x 100.01", ocEquity, ocDayBuy+`,"quantity":"1","limit_price":"100.01"`, 10001000000),
	ocValued("0.5 x 199.99", ocEquity, ocDayBuy+`,"quantity":"0.5","limit_price":"199.99"`, 9999500000),
	ocValued("0.00012345 x 67000.12345678", ocEquity, ocDayBuy+`,"quantity":"0.00012345","limit_price":"67000.12345678"`, 827116525),
	ocValued("3 x 33.333", ocEquity, ocDayBuy+`,"quantity":"3","limit_price":"33.333"`, 9999900000),
	ocValued("1 x 0.003", ocEquity, ocDayBuy+`,"quantity":"1","limit_price":"0.003"`, 300000),
	ocValued("0.25 x 400", ocEquity, ocDayBuy+`,"quantity":"0.25","limit_price":"400"`, 10000000000),
	ocValued("1.5 x 66.67", ocEquity, ocDayBuy+`,"quantity":"1.5","limit_price":"66.67"`, 10000500000),
	ocValued("0.5 x 0.00000001 never zero", ocEquity, ocDayBuy+`,"quantity":"0.5","limit_price":"0.00000001"`, 1),
	ocValued("json numbers", ocEquity, ocDayBuy+`,"quantity":2,"limit_price":49.5`, 9900000000),
	ocValued("stop-limit buy, stop_price ignored", ocEquity, `"side":"buy","type":"stop_limit","time_in_force":"gfd","quantity":"2","limit_price":"49.5","stop_price":"99999"`, 9900000000),
	ocValued("upper-case words", ocEquity, `"side":"BUY","type":"LIMIT","time_in_force":"GFD","quantity":"1","limit_price":"100"`, 10000000000),
	ocValued("extended hours", ocEquity, ocDayBuy+`,"market_hours":"extended_hours","quantity":"10","limit_price":"25.5"`, 25500000000),
	ocValued("regular hours", ocEquity, ocDayBuy+`,"market_hours":"regular_hours","quantity":"1","limit_price":"1"`, 100000000),
	ocValued("just under the top of the range", ocEquity, ocDayBuy+`,"quantity":"99999.99999999","limit_price":"100000"`, 999999999999900000),
	// Refused.
	ocRefused("market by shares", ocEquity, `"side":"buy","type":"market","time_in_force":"gfd","quantity":"1"`, argrules.WhyNotALimitBuy),
	ocRefused("stop_market by shares", ocEquity, `"side":"buy","type":"stop_market","time_in_force":"gfd","quantity":"1","stop_price":"5"`, argrules.WhyNotALimitBuy),
	ocRefused("trailing stop by shares", ocEquity, `"side":"buy","type":"trailing_stop","time_in_force":"gfd","quantity":"1"`, argrules.WhyNotALimitBuy),
	ocRefused("market with a stray limit_price", ocEquity, `"side":"buy","type":"market","time_in_force":"gfd","quantity":"1","limit_price":"5"`, argrules.WhyNotALimitBuy),
	ocRefused("sell by shares", ocEquity, `"side":"sell","type":"limit","time_in_force":"gfd","quantity":"1","limit_price":"5"`, argrules.WhyNotABuy),
	ocRefused("no side", ocEquity, `"type":"limit","time_in_force":"gfd","quantity":"1","limit_price":"5"`, argrules.WhyNotABuy),
	ocRefused("good till canceled", ocEquity, `"side":"buy","type":"limit","time_in_force":"gtc","quantity":"1","limit_price":"5"`, argrules.WhyNotADayOrder),
	ocRefused("no time_in_force", ocEquity, `"side":"buy","type":"limit","quantity":"1","limit_price":"5"`, argrules.WhyNoTimeInForce),
	ocRefused("overnight session", ocEquity, ocDayBuy+`,"market_hours":"all_day_hours","quantity":"1","limit_price":"5"`, argrules.WhyOvernightSession),
	ocRefused("no limit_price", ocEquity, ocDayBuy+`,"quantity":"1"`, argrules.WhyNoLimitPrice),
	ocRefused("crypto limit buy by quantity, day order", ocCrypto, ocDayBuy+`,"quantity":"0.01","limit_price":"50000"`, argrules.WhySharesToolNotYet),
	ocRefused("crypto limit buy by quantity, gtc", ocCrypto, `"side":"buy","type":"limit","time_in_force":"gtc","quantity":"0.01","limit_price":"50000"`, argrules.WhySharesToolNotYet),
	ocRefused("crypto stop_limit by quantity", ocCrypto, `"side":"buy","type":"stop_limit","time_in_force":"gfd","quantity":"0.01","limit_price":"50000","stop_price":"49000"`, argrules.WhySharesToolNotYet),
	ocRefused("crypto market by quantity", ocCrypto, `"side":"buy","type":"market","quantity":"0.01"`, argrules.WhySharesToolNotYet),
	ocRefused("both sizes", ocEquity, `"dollar_amount":"10","quantity":"1000","limit_price":"500"`, argrules.WhyBothSizes),
	ocRefused("neither size", ocEquity, `"side":"buy","type":"limit"`, argrules.WhyNoSize),
	ocRefused("dollars with limit_price", ocEquity, `"dollar_amount":"10","limit_price":"500"`, argrules.WhyDollarsWithPrice),
	ocRefused("dollars with stop_price", ocCrypto, `"dollar_amount":"10","stop_price":"500"`, argrules.WhyDollarsWithPrice),
	ocRefused("quantity zero", ocEquity, ocDayBuy+`,"quantity":"0","limit_price":"5"`, argrules.WhyNotAboveZero),
	ocRefused("limit_price zero", ocEquity, ocDayBuy+`,"quantity":"1","limit_price":"0"`, argrules.WhyNotAboveZero),
	ocRefused("dollars zero", ocEquity, `"dollar_amount":"0"`, argrules.WhyNotAboveZero),
	ocRefused("dollars negative", ocCrypto, `"dollar_amount":"-5"`, argrules.WhyNotAboveZero),
	ocRefused("quantity in exponent form", ocEquity, ocDayBuy+`,"quantity":"1e3","limit_price":"5"`, argrules.WhyBadNumber),
	ocRefused("dollars with a leading zero", ocEquity, `"dollar_amount":"01"`, argrules.WhyBadNumber),
	ocRefused("quantity null", ocEquity, ocDayBuy+`,"quantity":null,"limit_price":"5"`, argrules.WhyBadNumber),
	ocRefused("dollars too large", ocEquity, `"dollar_amount":"10000000000"`, argrules.WhyCostTooLarge),
	ocRefused("product too large", ocEquity, ocDayBuy+`,"quantity":"100000","limit_price":"100000"`, argrules.WhyCostTooLarge),
	ocRefused("a case twin of quantity", ocEquity, ocDayBuy+`,"Quantity":"1","limit_price":"5"`, argrules.WhyUnreadable),
	ocRefused("a repeated key", ocEquity, ocDayBuy+`,"quantity":"1","quantity":"1","limit_price":"5"`, argrules.WhyUnreadable),
}

// ocGenerous is a cap, limit or threshold no valued call in the table reaches.
const ocGenerous = "9999999999.99999999"

func ocUnitsText(u int64) string { return formatUnits(u) }

// TestOneValueOnEveryPath is the drift test. Every call goes through the
// per-order cap, the spend gate (token day, token week, shared total, shared
// symbol cap, token symbol cap, each alone and all five together) and the
// approval gate. For each it asserts one value and one refused set: the units
// each limit reserves are the units OrderMaxCost gives and the units worked out
// by hand; a cap, limit or threshold exactly at that value lets the call
// through and one unit under it does not; and a call the function refuses is
// refused by every path.
func TestOneValueOnEveryPath(t *testing.T) {
	if len(ocCases) < 40 {
		t.Fatalf("only %d cases", len(ocCases))
	}
	for _, c := range ocCases {
		c := c
		t.Run(c.name, func(t *testing.T) {
			params := ocParams(c.tool, c.args)
			cost, why := argrules.OrderMaxCostOfCall(params, c.tool, c.tool)
			valued := c.units > 0
			if valued != why.Valued() || cost.Units != c.units || (!valued && why != c.why) {
				t.Fatalf("OrderMaxCost = (%d, %q), table says (%d, %q)", cost.Units, why, c.units, c.why)
			}

			// Generous limits: the refused set is the same on every path.
			refusedBy := map[string]bool{}
			refusedBy["cap"] = argrules.Check(ocCapRules(t, c.tool, ocGenerous), c.tool, params) != nil
			for i, k := range ocLimitKinds {
				adm, _ := ocAdmitSpend(t, ocToken(t, ocLimitDoc(i, c.tool, ocGenerous)), c.tool, params)
				refusedBy["spend/"+k.name] = ocRefusalCode(adm) != ""
				if code := ocRefusalCode(adm); code != "" && code != toolcontrols.DenialCodeSpendLimit {
					t.Errorf("%s refused with %s", k.name, code)
				}
			}
			adm, _, _ := ocAdmitApproval(t, ocToken(t, ocApprovalDoc(c.tool, ocGenerous)), c.tool, params)
			refusedBy["approval"] = ocRefusalCode(adm) != ""
			for path, refused := range refusedBy {
				if refused != !valued {
					t.Errorf("%s: refused = %v for a call the function %s", path, refused, map[bool]string{true: "values", false: "refuses"}[valued])
				}
			}
			if !valued {
				// One reason in the agent's words, never a call value.
				if why != argrules.WhyUnreadable {
					adm, store := ocAdmitSpend(t, ocToken(t, ocLimitDoc(0, c.tool, "300")), c.tool, params)
					if adm == nil || adm.Refusal == nil || !strings.Contains(adm.Refusal.Message, string(why)) {
						t.Errorf("the spend refusal does not say %q: %+v", why, adm)
					}
					if n := len(ocTotals(store)); n != 0 {
						t.Errorf("a refused call touched %d counters", n)
					}
				}
				adm, rows, notices := ocAdmitApproval(t, ocToken(t, ocApprovalDoc(c.tool, "50")), c.tool, params)
				if code := ocRefusalCode(adm); code != toolcontrols.DenialCodeApprovalUnreadable {
					t.Errorf("approval refused with %q, want APPROVAL_CHECK_FAILED", code)
				}
				if len(rows.Rows()) != 0 || notices.count() != 0 {
					t.Error("an unvaluable call made an approval row or an email")
				}
				return
			}

			// The per-order cap: equal passes, one unit under refuses.
			if d := argrules.Check(ocCapRules(t, c.tool, ocUnitsText(c.units)), c.tool, params); d != nil {
				t.Errorf("cap equal to the value refused: %+v", d)
			}
			if c.units > 1 {
				if d := argrules.Check(ocCapRules(t, c.tool, ocUnitsText(c.units-1)), c.tool, params); d == nil || d.Reason != argrules.ReasonNoShape || d.Field != "order" {
					t.Errorf("cap one unit under the value: %+v", d)
				}
			}

			// Every spend limit alone reserves exactly the value; the max is
			// reached, not passed; one unit under refuses and reserves nothing.
			for i, k := range ocLimitKinds {
				adm, store := ocAdmitSpend(t, ocToken(t, ocLimitDoc(i, c.tool, ocUnitsText(c.units))), c.tool, params)
				if adm == nil || adm.Refusal != nil {
					t.Errorf("%s: a limit equal to the value refused: %+v", k.name, adm)
					continue
				}
				if tot := ocTotals(store); len(tot) != 1 || tot[0] != c.units {
					t.Errorf("%s: reserved %v, want [%d]", k.name, tot, c.units)
				}
				if c.units > 1 {
					adm, store := ocAdmitSpend(t, ocToken(t, ocLimitDoc(i, c.tool, ocUnitsText(c.units-1))), c.tool, params)
					if ocRefusalCode(adm) != toolcontrols.DenialCodeSpendLimit {
						t.Errorf("%s: a limit one unit under the value did not refuse: %+v", k.name, adm)
					}
					if n := len(ocTotals(store)); n != 0 && !ocAllEqual(ocTotals(store), 0) {
						t.Errorf("%s: a refused call left %v reserved", k.name, ocTotals(store))
					}
				}
			}
			// All five on one token: five counters, each holding the one value.
			all := func(max string) *TokenInfo { return ocToken(t, ocAllLimitsDoc(c.tool, max)) }
			adm, store := ocAdmitSpend(t, all(ocUnitsText(c.units)), c.tool, params)
			if adm == nil || adm.Refusal != nil {
				t.Fatalf("five limits equal to the value: %+v", adm)
			}
			if tot := ocTotals(store); len(tot) != 5 || !ocAllEqual(tot, c.units) {
				t.Errorf("five limits reserved %v, want five of %d", tot, c.units)
			}
			if c.units > 1 {
				adm, store := ocAdmitSpend(t, all(ocUnitsText(c.units-1)), c.tool, params)
				if ocRefusalCode(adm) != toolcontrols.DenialCodeSpendLimit {
					t.Errorf("five limits one unit under: %+v", adm)
				}
				if tot := ocTotals(store); len(tot) != 0 && !ocAllEqual(tot, 0) {
					t.Errorf("a refused call left %v reserved", tot)
				}
			}

			// The approval: equal is not held, one unit under holds.
			adm, rows, _ := ocAdmitApproval(t, ocToken(t, ocApprovalDoc(c.tool, ocUnitsText(c.units))), c.tool, params)
			if adm != nil && adm.Refusal != nil || len(rows.Rows()) != 0 {
				t.Errorf("a threshold equal to the value held the call: %+v", adm)
			}
			adm, rows, notices := ocAdmitApproval(t, ocToken(t, ocApprovalDoc(c.tool, ocUnitsText(c.units-1))), c.tool, params)
			if ocRefusalCode(adm) != toolcontrols.DenialCodeApprovalRequired || len(rows.Rows()) != 1 || notices.count() != 1 {
				t.Errorf("a threshold one unit under did not hold: %+v", adm)
			}
		})
	}
}

// ---- end to end through the proxy ----

const ocValueDay300 = `{"v":1,"spend_limits":[{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","max":"300","window":"day","value":"order_max_cost"}]}`

func ocBuy(qty, price string) string {
	return `{"symbol":"AAPL",` + ocDayBuy + `,"quantity":"` + qty + `","limit_price":"` + price + `"}`
}

// The worked boundaries (a day limit of 300 reached exactly, a cap of 100, an ask above 50), through the real proxy and a MemoryStore.
func TestOrderCostBoundariesPerOrderDayWeekBudgetApproval(t *testing.T) {
	for _, window := range []string{"day", "week"} {
		t.Run(window+" limit", func(t *testing.T) {
			doc := `{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"dollar_amount","max":"300","window":"` + window + `","value":"order_max_cost"}]}`
			h := newSpendHarness(t, doc)
			for i := 1; i <= 3; i++ { // three buys of 3 x 33.333 = 99.999
				if r := h.call(t, i, ocEquity, ocBuy("3", "33.333")); r.Error != nil {
					t.Fatalf("buy %d refused: %v", i, r.Error)
				}
			}
			if h.counter(t) != 29999700000 {
				t.Fatalf("counter = %d, want 299.997", h.counter(t))
			}
			// 1 x 0.003 brings the total to exactly 300.000: equal is allowed.
			if r := h.call(t, 4, ocEquity, ocBuy("1", "0.003")); r.Error != nil {
				t.Fatalf("the buy that reaches the limit was refused: %v", r.Error)
			}
			if h.counter(t) != 30000000000 {
				t.Fatalf("counter = %d, want 300", h.counter(t))
			}
			// A fifth of any size is refused, and counts nothing.
			r := h.call(t, 5, ocEquity, ocBuy("1", "0.00000001"))
			d := errData(t, r)
			if d["error"] != toolcontrols.DenialCodeSpendLimit {
				t.Fatalf("data = %v", d)
			}
			if got := r.Error.Message; !strings.Contains(got, "counting each order at the most it can cost; this order would go over it") {
				t.Errorf("message = %q", got)
			}
			if h.counter(t) != 30000000000 {
				t.Fatalf("a refused order counted: %d", h.counter(t))
			}
		})
	}

	t.Run("per-order cap", func(t *testing.T) {
		h := newSpendHarness(t, ocValueDay300)
		svcRules := `{"v":1,"rules":[{"tool":"place_equity_order","shapes":[{"order":{"max_cost":"100"}}]}]}`
		h.token = appendCaveatFor(t, h.token, ruleScopeWord(t, svcRules))
		for i, tc := range []struct {
			qty, price string
			ok         bool
		}{{"1", "100.00", true}, {"1", "100.01", false}, {"0.5", "199.99", true}} {
			r := h.call(t, i+1, ocEquity, ocBuy(tc.qty, tc.price))
			if (r.Error == nil) != tc.ok {
				t.Errorf("%s x %s: %+v", tc.qty, tc.price, r.Error)
			}
			if !tc.ok && errData(t, r)["error"] != argrules.DenialCode {
				t.Errorf("%s x %s: %v", tc.qty, tc.price, errData(t, r))
			}
		}
		// The two buys that passed are the only ones counted.
		if h.counter(t) != 10000000000+9999500000 {
			t.Errorf("counter = %d", h.counter(t))
		}
	})

	t.Run("approval", func(t *testing.T) {
		a := newApprovalHarness(t, ocApprovalDoc(ocEquity, "50"))
		for i, tc := range []struct {
			qty, price string
			held       bool
		}{{"1", "50.00", false}, {"1", "50.01", true}, {"0.5", "100.01", true}, {"0.5", "100", false}, {"1", "49.99999999", false}} {
			r := a.call(t, i+1, ocEquity, ocBuy(tc.qty, tc.price))
			if held := r.Error != nil && errData(t, r)["error"] == "APPROVAL_REQUIRED"; held != tc.held {
				t.Errorf("%s x %s: held = %v (%+v)", tc.qty, tc.price, held, r.Error)
			}
		}
	})
}

func appendCaveatFor(t *testing.T, token string, caveats ...string) string {
	t.Helper()
	svc, err := macaroon.NewService(argTestRoot)
	if err != nil {
		t.Fatal(err)
	}
	return appendCaveat(t, svc, token, caveats...)
}

// The D5 table through the real gate and a MemoryStore: the counter holds the
// rounded-up units.
func TestOrderCostFractionalQuantityAndRounding(t *testing.T) {
	for _, c := range ocCases {
		if c.units == 0 || c.tool != ocEquity || c.units > 90000000000 {
			continue
		}
		h := newSpendHarness(t, ocValueDay300)
		r := h.call(t, 1, ocEquity, c.args)
		if r.Error != nil {
			t.Errorf("%s: refused: %v", c.name, r.Error)
			continue
		}
		if h.counter(t) != c.units {
			t.Errorf("%s: counter = %d, want %d", c.name, h.counter(t), c.units)
		}
	}
}

// A limit buy is counted at the call and never refunded for not filling: the
// give-back rules are the dollar orders' rules, outcome by outcome.
func TestOrderCostCountedAtCallNeverRefundedForNotFilling(t *testing.T) {
	h := newSpendHarness(t, ocValueDay300)
	h.rr.replies = []func() (*Response, error){okReply, isErrorReply, lostAfterSend, notSent, rpcErrorCode(-32602), rpcErrorCode(-32603)}
	steps := []struct {
		qty, price string
		want       int64 // the counter after the call
		what       string
	}{
		{"1", "100", 10000000000, "answered ok: kept"},
		{"1", "50", 15000000000, "answered isError: kept"},
		{"1", "20", 17000000000, "lost after the send: kept"},
		{"1", "10", 17000000000, "never sent: given back"},
		{"1", "5", 17000000000, "-32602 (not processed): given back"},
		{"1", "1", 17100000000, "-32603 (unknown): kept"},
	}
	for i, s := range steps {
		h.call(t, i+1, ocEquity, ocBuy(s.qty, s.price))
		if got := h.counter(t); got != s.want {
			t.Fatalf("step %d (%s): counter = %d, want %d", i+1, s.what, got, s.want)
		}
	}
}

// An order that does not fill still counts until the day or the week resets,
// and not a moment longer.
func TestOrderCostCountsUntilTheWindowResets(t *testing.T) {
	for _, tc := range []struct {
		window string
		later  time.Time // a time in the next window
		same   time.Time // the last moment of this window
	}{
		{"day", time.Date(2026, 10, 15, 4, 1, 0, 0, time.UTC), time.Date(2026, 10, 15, 3, 59, 0, 0, time.UTC)},
		{"week", time.Date(2026, 10, 19, 4, 1, 0, 0, time.UTC), time.Date(2026, 10, 19, 3, 59, 0, 0, time.UTC)},
	} {
		t.Run(tc.window, func(t *testing.T) {
			doc := `{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"dollar_amount","max":"100","window":"` + tc.window + `","value":"order_max_cost"}]}`
			tok := ocToken(t, doc)
			g := newTestSpendGate(toolcontrols.NewMemoryStore(ocClock))
			admit := func(id string, now time.Time) *GateAdmission {
				adm, err := g.Admit(context.Background(), GateCall{Token: tok, Tool: ocEquity, Params: ocParams(ocEquity, ocBuy("1", "100")), RequestID: id, Now: now})
				if err != nil {
					t.Fatal(err)
				}
				return adm
			}
			if adm := admit("a", ocNow); adm == nil || adm.Refusal != nil {
				t.Fatalf("first buy refused: %+v", adm)
			}
			// It never fills; it still counts until the window ends.
			if adm := admit("b", tc.same); ocRefusalCode(adm) != toolcontrols.DenialCodeSpendLimit {
				t.Fatalf("a second buy in the same %s was admitted: %+v", tc.window, adm)
			}
			if adm := admit("c", tc.later); adm == nil || adm.Refusal != nil {
				t.Fatalf("a buy in the next %s was refused: %+v", tc.window, adm)
			}
		})
	}
}

func TestOrderCostApprovalUnvaluableIsRefusedNotHeld(t *testing.T) {
	a := newApprovalHarness(t, ocApprovalDoc(ocEquity, "50"))
	before := a.rr.forwarded()
	for i, args := range []string{
		`{"symbol":"AAPL","side":"buy","type":"market","time_in_force":"gfd","quantity":"1"}`,
		`{"symbol":"AAPL","side":"buy","type":"limit","time_in_force":"gtc","quantity":"1","limit_price":"10"}`,
		`{"symbol":"AAPL","dollar_amount":"10","quantity":"1"}`,
		`{"symbol":"AAPL","side":"buy"}`,
		`{"symbol":"AAPL","dollar_amount":"10","limit_price":"10"}`,
	} {
		r := a.call(t, i+1, ocEquity, args)
		d := errData(t, r)
		if d["error"] != "APPROVAL_CHECK_FAILED" {
			t.Errorf("%s: error = %v", args, d["error"])
		}
	}
	if len(a.approvals.Rows()) != 0 || a.notifier.count() != 0 {
		t.Fatalf("rows=%d notices=%d: an unvaluable call was held or emailed", len(a.approvals.Rows()), a.notifier.count())
	}
	if a.rr.forwarded() != before {
		t.Fatal("an unvaluable call was forwarded")
	}

	// A valued call over the threshold is held, with the cost as the owner's
	// first line, and the approved retry goes through.
	args := ocBuy("1", "75")
	r := a.call(t, 10, ocEquity, args)
	if errData(t, r)["error"] != "APPROVAL_REQUIRED" {
		t.Fatalf("not held: %v", r.Error)
	}
	if !strings.Contains(r.Error.Message, "orders that can cost more than $50 need the owner's approval") {
		t.Errorf("message = %q", r.Error.Message)
	}
	row := a.onlyApproval(t)
	if len(row.Summary) == 0 || row.Summary[0].Name != "Counts as at most" || row.Summary[0].Value != "$75" {
		t.Fatalf("summary = %+v", row.Summary)
	}
	if n := a.notifier.notices; len(n) != 1 || n[0].Value != argrules.OrderCostBasis || n[0].Above != "50" || n[0].Summary[0].Value != "$75" {
		t.Fatalf("notices = %+v", n)
	}
	if err := a.approvals.Approve("tenant-1", row.ID, "owner@example.com", a.clock); err != nil {
		t.Fatal(err)
	}
	if r := a.call(t, 11, ocEquity, args); r.Error != nil {
		t.Fatalf("approved retry refused: %v", r.Error)
	}
}

// A legacy rule keeps what it did: a call it cannot read is held.
func TestOrderCostLegacyApprovalStillHoldsAnUnreadableCall(t *testing.T) {
	a := newApprovalHarness(t, askAbove50)
	r := a.call(t, 1, ocCrypto, `{"side":"buy","type":"limit","symbol":"BTC","quantity":"1","limit_price":"10"}`)
	if errData(t, r)["error"] != "APPROVAL_REQUIRED" {
		t.Fatalf("a legacy rule did not hold an unreadable call: %v", r.Error)
	}
	if row := a.onlyApproval(t); len(row.Summary) == 0 || row.Summary[0].Name == "Counts as at most" {
		t.Fatalf("summary = %+v", row.Summary)
	}
}

// A value rule and a legacy rule on one token: the call the value rule cannot
// value is refused even though the legacy rule would hold it.
func TestOrderCostValueRuleRefusesBeforeALegacyRuleHolds(t *testing.T) {
	tok := ocToken(t, askAbove50, ocApprovalDoc(ocCrypto, "50"))
	params := ocParams(ocCrypto, `{"side":"buy","type":"limit","symbol":"BTC","quantity":"1","limit_price":"10"}`)
	adm, rows, notices := ocAdmitApproval(t, tok, ocCrypto, params)
	if ocRefusalCode(adm) != toolcontrols.DenialCodeApprovalUnreadable || len(rows.Rows()) != 0 || notices.count() != 0 {
		t.Fatalf("%+v rows=%d", adm, len(rows.Rows()))
	}
}

// A token with a legacy limit and a value limit applies both. A dollars call is
// counted in both; a limit buy is refused at the legacy one (it has no
// dollar_amount) and counts nothing at the value one, so a mix fails closed.
func TestOrderCostBothTokensAndBudgetsAreIndependent(t *testing.T) {
	legacy := `{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"dollar_amount","max":"300","window":"day"}]}`
	value := ocLimitDoc(0, ocEquity, "300")
	tok := ocToken(t, legacy, value)
	store := toolcontrols.NewMemoryStore(ocClock)
	g := newTestSpendGate(store)
	admit := func(id, args string) *GateAdmission {
		adm, err := g.Admit(context.Background(), GateCall{Token: tok, Tool: ocEquity, Params: ocParams(ocEquity, args), RequestID: id, Now: ocNow})
		if err != nil {
			t.Fatal(err)
		}
		return adm
	}
	if adm := admit("a", `{"symbol":"AAPL","dollar_amount":"40"}`); adm == nil || adm.Refusal != nil {
		t.Fatalf("a dollars call was refused: %+v", adm)
	}
	if tot := ocTotals(store); len(tot) != 2 || !ocAllEqual(tot, 4000000000) {
		t.Fatalf("totals = %v, want two counters of 40", tot)
	}
	adm := admit("b", ocBuy("1", "10"))
	if ocRefusalCode(adm) != toolcontrols.DenialCodeSpendLimit || !strings.Contains(adm.Refusal.Message, "the call has no value for it") {
		t.Fatalf("a limit buy under a legacy limit: %+v", adm)
	}
	if tot := ocTotals(store); !ocAllEqual(tot, 4000000000) {
		t.Fatalf("a refused limit buy changed the totals: %v", tot)
	}
}

// Documents without the new keys keep the meaning they had.
func TestExistingDocumentsKeepTheirMeaning(t *testing.T) {
	limit := `{"v":1,"spend_limits":[{"tools":["place_equity_order","place_crypto_order"],"field":"dollar_amount","max":"300","window":"day"}]}`
	ask := `{"v":1,"approvals":[{"tools":["place_equity_order","place_crypto_order"],"field":"dollar_amount","above":"50"}]}`
	spend := func(args string) (*GateAdmission, []int64) {
		adm, store := ocAdmitSpend(t, ocToken(t, limit), ocEquity, ocParams(ocEquity, args))
		return adm, ocTotals(store)
	}

	// A limit buy by quantity has no dollar_amount: refused, with the hint.
	adm, tot := spend(ocBuy("1", "100"))
	want := "place_equity_order: this token has a limit of $300 a day on dollar_amount, and the call has no value for it, so the limit cannot be checked; send dollar_amount instead of quantity"
	if ocRefusalCode(adm) != toolcontrols.DenialCodeSpendLimit || adm.Refusal.Message != want || len(tot) != 0 {
		t.Fatalf("limit buy: %+v %v", adm, tot)
	}
	// Both fields: counted at dollar_amount alone.
	adm, tot = spend(`{"symbol":"AAPL","dollar_amount":"10","quantity":"1000","limit_price":"500"}`)
	if adm == nil || adm.Refusal != nil || len(tot) != 1 || tot[0] != 1000000000 {
		t.Fatalf("both fields: %+v %v", adm, tot)
	}
	// Zero is counted as zero.
	adm, tot = spend(`{"symbol":"AAPL","dollar_amount":"0"}`)
	if adm == nil || adm.Refusal != nil || len(tot) != 1 || tot[0] != 0 {
		t.Fatalf("zero: %+v %v", adm, tot)
	}
	// Negative is refused, as over the limit.
	adm, _ = spend(`{"symbol":"AAPL","dollar_amount":"-5"}`)
	if ocRefusalCode(adm) != toolcontrols.DenialCodeSpendLimit || adm.Refusal.Message != "place_equity_order: this token's limit is $300 a day on dollar_amount; this order would go over it" {
		t.Fatalf("negative: %+v", adm)
	}
	// A sell by dollars is counted; the side is never read.
	adm, tot = spend(`{"symbol":"AAPL","side":"sell","dollar_amount":"20"}`)
	if adm == nil || adm.Refusal != nil || len(tot) != 1 || tot[0] != 2000000000 {
		t.Fatalf("sell by dollars: %+v %v", adm, tot)
	}
	// Over the limit: the old text.
	adm, _ = spend(`{"symbol":"AAPL","dollar_amount":"300.01"}`)
	if adm == nil || adm.Refusal == nil || adm.Refusal.Message != "place_equity_order: this token's limit is $300 a day on dollar_amount; this order would go over it" {
		t.Fatalf("over: %+v", adm)
	}

	// Approvals: a legacy rule holds an unreadable or negative amount, and does
	// not hold zero, equal to the threshold, or a call with both fields at 10.
	held := func(args string) bool {
		adm, _, _ := ocAdmitApproval(t, ocToken(t, ask), ocEquity, ocParams(ocEquity, args))
		return ocRefusalCode(adm) == toolcontrols.DenialCodeApprovalRequired
	}
	for args, want := range map[string]bool{
		`{"symbol":"AAPL","dollar_amount":"50"}`:                                               false,
		`{"symbol":"AAPL","dollar_amount":"50.01"}`:                                            true,
		`{"symbol":"AAPL","dollar_amount":"0"}`:                                                false,
		`{"symbol":"AAPL","dollar_amount":"-5"}`:                                               true,
		`{"symbol":"AAPL","dollar_amount":"10","quantity":"1000"}`:                             false,
		`{"symbol":"AAPL","quantity":"1","limit_price":"100"}`:                                 true,
		`{"symbol":"AAPL","dollar_amount":"1e2"}`:                                              true,
		`{"symbol":"AAPL","side":"sell","dollar_amount":"10"}`:                                 false,
		`{"symbol":"AAPL","side":"buy","type":"limit","quantity":"1","limit_price":"1000000"}`: true,
	} {
		if got := held(args); got != want {
			t.Errorf("%s: held = %v, want %v", args, got, want)
		}
	}
}

// A tool listed under a renamed name is counted by a shared budget under the
// tool's own name; the call is valued as that order tool and read under the
// name it was sent as.
func TestOrderCostRenamedToolUnderASharedBudget(t *testing.T) {
	tok := ocToken(t, ocLimitDoc(2, ocEquity, "300"))
	const sent = "robinhood-2_place_equity_order"
	store := toolcontrols.NewMemoryStore(ocClock)
	g := newTestSpendGate(store)
	origin := ToolOrigin{Listed: true, Renamed: true, Original: ocEquity, Service: "robinhood-2"}
	admit := func(args string) *GateAdmission {
		adm, err := g.Admit(context.Background(), GateCall{Token: tok, Tool: sent, Params: ocParams(sent, args), RequestID: "r", Origin: origin, Now: ocNow})
		if err != nil {
			t.Fatal(err)
		}
		return adm
	}
	if adm := admit(ocBuy("2", "49.5")); adm == nil || adm.Refusal != nil {
		t.Fatalf("a limit buy under the renamed name: %+v", adm)
	}
	if tot := ocTotals(store); len(tot) != 1 || tot[0] != 9900000000 {
		t.Fatalf("totals = %v", tot)
	}
	// Not a buy: refused for the fixed reason.
	adm := admit(`{"symbol":"AAPL","side":"sell","type":"limit","time_in_force":"gfd","quantity":"1","limit_price":"1"}`)
	if ocRefusalCode(adm) != toolcontrols.DenialCodeSpendLimit || !strings.Contains(adm.Refusal.Message, string(argrules.WhyNotABuy)) {
		t.Fatalf("%+v", adm)
	}
}

// Refusal texts, data and receipt details never echo a value from the call.
func TestOrderCostRefusalsNeverEchoValues(t *testing.T) {
	const marker = "7731.4242"
	mark := func(s string) string { return strings.ReplaceAll(s, "@", marker) }
	calls := []struct{ tool, args string }{
		{ocEquity, mark(`{"symbol":"AAPL","side":"buy","type":"limit","time_in_force":"gtc","quantity":"@","limit_price":"@"}`)},
		{ocEquity, mark(`{"symbol":"AAPL","side":"buy","type":"market","time_in_force":"gfd","quantity":"@"}`)},
		{ocEquity, mark(`{"symbol":"AAPL","dollar_amount":"@","quantity":"@"}`)},
		{ocEquity, mark(`{"symbol":"AAPL","dollar_amount":"@","stop_price":"@"}`)},
		{ocEquity, mark(`{"symbol":"AAPL","side":"buy","type":"limit","time_in_force":"gfd","quantity":"-@","limit_price":"@"}`)},
		{ocCrypto, mark(`{"symbol":"BTC","side":"buy","type":"limit","time_in_force":"gfd","quantity":"@","limit_price":"@"}`)},
		{ocEquity, mark(`{"symbol":"AAPL","dollar_amount":"@99999999999"}`)},
		{ocEquity, mark(`{"symbol":"AAPL","dollar_amount":"@x"}`)},
		{ocEquity, mark(`{"symbol":"AAPL","side":"buy","type":"limit","time_in_force":"gfd","quantity":"@","limit_price":"@","Dollar_Amount":"@"}`)},
		// Over the limit, and at the cap, with the marker as the cost.
		{ocEquity, mark(`{"symbol":"AAPL","dollar_amount":"@"}`)},
	}
	check := func(what, text string) {
		if strings.Contains(text, "7731") || strings.Contains(text, "4242") {
			t.Errorf("%s holds a value from the call: %q", what, text)
		}
	}
	for _, c := range calls {
		params := ocParams(c.tool, c.args)
		for i, k := range ocLimitKinds {
			// 7731.4242 is over a limit of 100, so both the unvaluable and
			// the over-limit texts are checked.
			adm, _ := ocAdmitSpend(t, ocToken(t, ocLimitDoc(i, c.tool, "100")), c.tool, params)
			if adm == nil || adm.Refusal == nil {
				t.Errorf("%s: admitted %s", k.name, c.args)
				continue
			}
			b, _ := json.Marshal(adm.Refusal)
			check("spend/"+k.name, string(b))
		}
		adm, _, _ := ocAdmitApproval(t, ocToken(t, ocApprovalDoc(c.tool, "100")), c.tool, params)
		if adm != nil && adm.Refusal != nil {
			// Held or refused: the agent-facing parts carry no value.
			b, _ := json.Marshal(adm.Refusal)
			check("approval", string(b))
		}
		if d := argrules.Check(ocCapRules(t, c.tool, "100"), c.tool, params); d != nil {
			check("cap", d.Error()+" "+d.Allowed)
		}
	}
}

// A verifier with no controls support refuses a token that carries a value
// word, and one that enforces controls reads it.
func TestPlainVerifierRefusesValueWord(t *testing.T) {
	h := newSpendHarness(t, ocValueDay300)
	svc, err := macaroon.NewService(argTestRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Verify(h.token); err == nil {
		t.Fatal("a verifier that does not enforce controls accepted the token")
	}
	if _, err := svc.AcceptingArgumentRules().Verify(h.token); err != nil {
		t.Fatalf("enforcing verifier refused: %v", err)
	}
}
