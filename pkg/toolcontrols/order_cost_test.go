package toolcontrols

import (
	"math/big"
	"strings"
	"testing"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/toolcontrols/internal/releasedv1"
	"github.com/satgate-io/satgate/pkg/toolcontrols/internal/releasedv2"
)

const (
	// The documents the recommended trading setup writes for Small trades, with
	// the made-up account 5RH12345 left out (controls carry no account).
	valueSpendDoc = `{"v":1,"spend_limits":[` +
		`{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","max":"300","window":"day","tz":"America/New_York","value":"order_max_cost"},` +
		`{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","max":"1000","window":"week","tz":"America/New_York","value":"order_max_cost"}]}`
	valueApprovalDoc = `{"v":1,"approvals":[{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","above":"50","value":"order_max_cost"}]}`
	valueBothDoc     = `{"v":1,"spend_limits":[` +
		`{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","max":"300","window":"day","tz":"America/New_York","value":"order_max_cost"}],` +
		`"approvals":[{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","above":"50","value":"order_max_cost"}]}`
)

func TestValueKeyParseMarshalAndCanonicalOrder(t *testing.T) {
	for name, doc := range map[string]string{"spend": valueSpendDoc, "approval": valueApprovalDoc, "both": valueBothDoc} {
		d, err := ParseJSON([]byte(doc))
		if err != nil {
			t.Fatalf("%s: %v", name, err)
		}
		out, err := Marshal(d)
		if err != nil || string(out) != doc {
			t.Errorf("%s: canonical form\n got %s (%v)\nwant %s", name, out, err, doc)
		}
	}
	d, _ := ParseJSON([]byte(valueBothDoc))
	if d.SpendLimits[0].Value != argrules.OrderCostBasis || d.Approvals[0].Value != argrules.OrderCostBasis {
		t.Fatalf("value not read: %+v %+v", d.SpendLimits[0], d.Approvals[0])
	}
	// "value" is written last, after budget and symbols; keys may come in any
	// order on input.
	in := `{"v":1,"spend_limits":[{"value":"order_max_cost","symbols":["AAPL"],"budget":"` + testBudget + `","window":"day","max":"200","field":"dollar_amount","tools":["place_equity_order"]}]}`
	want := `{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"dollar_amount","max":"200","window":"day","tz":"America/New_York","budget":"` + testBudget + `","symbols":["AAPL"],"value":"order_max_cost"}]}`
	if got, err := Canonical([]byte(in)); err != nil || string(got) != want {
		t.Fatalf("canonical = %s (%v)", got, err)
	}
	// A word round trips, and every limit keeps its identity and its value.
	dd, _ := ParseJSON([]byte(valueSpendDoc))
	word, err := ScopeValue(dd)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseScopeValue(word)
	if err != nil || len(back.SpendLimits) != 2 || back.SpendLimits[0].Value == "" || back.SpendLimits[0].ID == "" {
		t.Fatalf("%+v %v", back, err)
	}
	// And Collect sees it.
	limits, err := Collect([]string{"scope = place_equity_order", "scope = " + word})
	if err != nil || len(limits) != 2 || limits[1].Value != argrules.OrderCostBasis {
		t.Fatalf("%+v %v", limits, err)
	}
	rules, err := CollectApprovals([]string{"scope = " + mustCaveatWord(t, valueApprovalDoc)})
	if err != nil || len(rules) != 1 || rules[0].Value != argrules.OrderCostBasis {
		t.Fatalf("%+v %v", rules, err)
	}
}

func mustCaveatWord(t *testing.T, doc string) string {
	t.Helper()
	d, err := ParseJSON([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	w, err := ScopeValue(d)
	if err != nil {
		t.Fatal(err)
	}
	return w
}

func TestValueKeyOnlyOnOrderToolsWithDollarField(t *testing.T) {
	spend := func(tools, field, value string) string {
		return `{"v":1,"spend_limits":[{"tools":[` + tools + `],"field":"` + field + `","max":"300","window":"day","value":` + value + `}]}`
	}
	approval := func(tools, field, value string) string {
		return `{"v":1,"approvals":[{"tools":[` + tools + `],"field":"` + field + `","above":"50","value":` + value + `}]}`
	}
	for name, build := range map[string]func(tools, field, value string) string{"spend limit": spend, "approval": approval} {
		const ok = `"order_max_cost"`
		for _, tools := range []string{`"place_equity_order"`, `"place_crypto_order"`, `"place_crypto_order","place_equity_order"`,
			`"Place_Equity_Order"`, `"PLACE_CRYPTO_ORDER"`} {
			if _, err := ParseJSON([]byte(build(tools, "dollar_amount", ok))); err != nil {
				t.Errorf("%s %s refused: %v", name, tools, err)
			}
		}
		// Options and every other tool, alone or beside an order tool, are refused.
		for _, tools := range []string{`"place_option_order"`, `"review_option_order"`, `"exercise_option"`, `"review_equity_order"`,
			`"preview_crypto_order"`, `"cancel_equity_order"`, `"get_quote"`, `"robinhood-2_place_equity_order"`,
			`"place_equity_order","place_option_order"`, `"place_equity_order","get_quote"`} {
			if _, err := ParseJSON([]byte(build(tools, "dollar_amount", ok))); err == nil || !strings.Contains(err.Error(), "can only count") {
				t.Errorf("%s %s: %v", name, tools, err)
			}
		}
		// Only dollar_amount.
		for _, field := range []string{"quantity", "limit_price", "Dollar_Amount", "dollar_amounts", "symbol"} {
			if _, err := ParseJSON([]byte(build(`"place_equity_order"`, field, ok))); err == nil || !strings.Contains(err.Error(), "must have field") {
				t.Errorf("%s field %s: %v", name, field, err)
			}
		}
		// Only the one value.
		for _, v := range []string{`""`, `"ORDER_MAX_COST"`, `"order_max_cost "`, `"order_cost"`, `"max_cost"`, `"dollar_amount"`, `null`, `true`, `1`, `["order_max_cost"]`, `{}`} {
			if _, err := ParseJSON([]byte(build(`"place_equity_order"`, "dollar_amount", v))); err == nil {
				t.Errorf("%s value %s accepted", name, v)
			}
		}
	}
	// The legacy reading of the same document is untouched: no value, any tool.
	if _, err := ParseJSON([]byte(`{"v":1,"spend_limits":[{"tools":["get_quote"],"field":"quantity","max":"3","window":"week"}]}`)); err != nil {
		t.Errorf("legacy limit refused: %v", err)
	}
	// A value limit can carry a symbol cap and a budget, as a limit can.
	if _, err := ParseJSON([]byte(`{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"50","window":"day","budget":"` + testBudget + `","symbols":["BTC"],"value":"order_max_cost"}]}`)); err != nil {
		t.Errorf("value symbol cap in a budget refused: %v", err)
	}
}

// TestExistingControlDocumentsKeepTheirCanonicalBytes is the golden for control
// documents written before the value key: the recommended trading limits, an
// approval, a budget with a symbol cap. Each marshals to exactly these bytes
// and to the bytes the frozen parser (0e60d5f8) writes.
func TestExistingControlDocumentsKeepTheirCanonicalBytes(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{
			"recommended day and week limits with ask-above",
			`{"v":1,"approvals":[{"above":"50","field":"dollar_amount","tools":["place_equity_order","place_crypto_order"]}],"spend_limits":[{"window":"day","max":"300","field":"dollar_amount","tools":["place_equity_order","place_crypto_order"]},{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","max":"1000","window":"week","tz":"America/New_York"}]}`,
			`{"v":1,"spend_limits":[{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","max":"300","window":"day","tz":"America/New_York"},{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","max":"1000","window":"week","tz":"America/New_York"}],"approvals":[{"tools":["place_crypto_order","place_equity_order"],"field":"dollar_amount","above":"50"}]}`,
		},
		{
			"a shared budget with a symbol cap",
			`{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"dollar_amount","max":"200","window":"day","tz":"America/New_York","budget":"` + testBudget + `"},{"symbols":["AAPL"],"budget":"` + testBudget + `","tools":["place_equity_order"],"field":"dollar_amount","max":"50.5","window":"day"}]}`,
			`{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"dollar_amount","max":"200","window":"day","tz":"America/New_York","budget":"` + testBudget + `"},{"tools":["place_equity_order"],"field":"dollar_amount","max":"50.5","window":"day","tz":"America/New_York","budget":"` + testBudget + `","symbols":["AAPL"]}]}`,
		},
		{
			"a limit on some other field",
			`{"v":1,"spend_limits":[{"tools":["transfer"],"field":"amount","max":"1000.25","window":"week","tz":"Europe/London"}]}`,
			`{"v":1,"spend_limits":[{"tools":["transfer"],"field":"amount","max":"1000.25","window":"week","tz":"Europe/London"}]}`,
		},
	} {
		got, err := Canonical([]byte(tc.in))
		if err != nil || string(got) != tc.want {
			t.Errorf("%s:\n got %s (%v)\nwant %s", tc.name, got, err, tc.want)
			continue
		}
		if frozen, err := releasedv2.Canonical([]byte(tc.in)); err != nil || string(frozen) != tc.want {
			t.Errorf("%s: the frozen parser writes %s (%v)", tc.name, frozen, err)
		}
	}
}

// An old verifier cannot read a document with the new key, in any of the
// places it is written: it refuses it as unknown, so the token is denied and
// the limit is never dropped.
func TestOldVerifiersRefuseNewDocuments(t *testing.T) {
	budgetDoc := `{"v":1,"spend_limits":[` +
		`{"tools":["place_equity_order"],"field":"dollar_amount","max":"200","window":"day","tz":"America/New_York","budget":"` + testBudget + `","value":"order_max_cost"},` +
		`{"tools":["place_equity_order"],"field":"dollar_amount","max":"50","window":"day","tz":"America/New_York","budget":"` + testBudget + `","symbols":["AAPL"],"value":"order_max_cost"}]}`
	docs := map[string]string{
		"spend limit":              valueSpendDoc,
		"shared-budget limit":      budgetDoc,
		"approval rule":            valueApprovalDoc,
		"limits and approval":      valueBothDoc,
		"symbol cap with a value":  `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"50","window":"day","symbols":["BTC"],"value":"order_max_cost"}]}`,
		"an approval beside plain": `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"50","window":"day"}],"approvals":[{"tools":["place_crypto_order"],"field":"dollar_amount","above":"5","value":"order_max_cost"}]}`,
	}
	for name, doc := range docs {
		if _, err := ParseJSON([]byte(doc)); err != nil {
			t.Fatalf("%s: the current parser refused it: %v", name, err)
		}
		// The snapshot at 0e60d5f8: spend limits, approvals, symbols, budgets.
		_, err := releasedv2.ParseJSON([]byte(doc))
		if err == nil || !(strings.Contains(err.Error(), `unknown spending limit key "value"`) || strings.Contains(err.Error(), `unknown approval rule key "value"`)) {
			t.Errorf("%s: the parser at 0e60d5f8 did not refuse it with the unknown-key error: %v", name, err)
		}
		// The release at 84998dbd (releasedv1) refuses it too: for a spend limit
		// by the unknown key, for an approval because its snapshot has none.
		if _, err := releasedv1.ParseJSON([]byte(doc)); err == nil {
			t.Errorf("%s: the parser at 84998dbd read it", name)
		}
		// A whole control word built from it is refused as a word.
		d, _ := ParseJSON([]byte(doc))
		word, err := ScopeValue(d)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := releasedv2.ParseScopeValue(word); err == nil {
			t.Errorf("%s: the parser at 0e60d5f8 read the word", name)
		}
		if _, err := releasedv1.ParseScopeValue(word); err == nil {
			t.Errorf("%s: the parser at 84998dbd read the word", name)
		}
		if _, err := releasedv2.CollectSealed([]string{"scope = place_equity_order", "scope = " + word}, 0); err == nil {
			t.Errorf("%s: the old Collect read the word", name)
		}
	}
	// The same parsers take a document without the key, to the same bytes.
	plain := `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"50","window":"day"}]}`
	a, err1 := releasedv2.Canonical([]byte(plain))
	b, err2 := Canonical([]byte(plain))
	if err1 != nil || err2 != nil || string(a) != string(b) {
		t.Fatalf("plain: %s %v / %s %v", a, err1, b, err2)
	}
	if _, err := releasedv1.ParseJSON([]byte(plain)); err != nil {
		t.Fatalf("84998dbd refused a plain document: %v", err)
	}
}

func TestDescribeValueLimit(t *testing.T) {
	d, err := ParseJSON([]byte(valueSpendDoc))
	if err != nil {
		t.Fatal(err)
	}
	const counted = "counting each order at the most it can cost (dollar_amount, or quantity times limit_price for a day limit buy of shares)"
	got := Describe(d.SpendLimits)
	want := []string{
		"place_crypto_order and place_equity_order: at most $300 a day in total, " + counted,
		"place_crypto_order and place_equity_order: at most $1000 a week in total, " + counted,
	}
	if len(got) != 2 || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("Describe = %q", got)
	}
	// Symbol, budget and zone notes keep their places.
	d, _ = ParseJSON([]byte(`{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"dollar_amount","max":"50","window":"day","tz":"Europe/London","budget":"` + testBudget + `","symbols":["AAPL"],"value":"order_max_cost"}]}`))
	line := Describe(d.SpendLimits)[0]
	wantLine := "place_equity_order: at most $50 a day in total, " + counted + ", for AAPL only, shared with your other agents (days start at midnight Europe/London time)"
	if line != wantLine {
		t.Fatalf("Describe = %q", line)
	}
	if strings.Contains(line, testBudget) {
		t.Error("the line names the budget")
	}
	// A legacy limit prints exactly what it printed.
	l, _ := ParseJSON([]byte(`{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"300","window":"day"}]}`))
	if got := Describe(l.SpendLimits)[0]; got != "place_crypto_order: at most $300 a day in total on dollar_amount" {
		t.Errorf("legacy line = %q", got)
	}
}

func TestDescribeApprovalValue(t *testing.T) {
	d, err := ParseJSON([]byte(valueApprovalDoc))
	if err != nil {
		t.Fatal(err)
	}
	want := "place_crypto_order and place_equity_order: orders that can cost more than $50 wait for your approval, counting each order at the most it can cost (dollar_amount, or quantity times limit_price for a day limit buy of shares)"
	if got := DescribeApprovals(d.Approvals); len(got) != 1 || got[0] != want {
		t.Fatalf("DescribeApprovals = %q", got)
	}
	l, _ := ParseJSON([]byte(askDoc))
	if got := DescribeApprovals(l.Approvals)[0]; got != "place_crypto_order: calls with dollar_amount above $50 wait for your approval" {
		t.Errorf("legacy line = %q", got)
	}
}

func TestApprovalThresholdUnitsAndCountsAs(t *testing.T) {
	for _, tc := range []struct {
		above string
		want  int64
	}{{"0", 0}, {"50", 5000000000}, {"0.00000001", 1}, {"9999999999.99999999", 999999999999999999}} {
		d, err := ParseJSON([]byte(`{"v":1,"approvals":[{"tools":["place_equity_order"],"field":"dollar_amount","above":"` + tc.above + `","value":"order_max_cost"}]}`))
		if err != nil {
			t.Fatalf("%s: %v", tc.above, err)
		}
		if got, ok := d.Approvals[0].ThresholdUnits(); !ok || got != tc.want {
			t.Errorf("%s: (%d, %v), want %d", tc.above, got, ok, tc.want)
		}
	}
	if _, ok := (ApprovalRule{}).ThresholdUnits(); ok {
		t.Error("a rule built by hand has a threshold")
	}
	for _, tc := range []struct {
		units int64
		want  string
	}{{10000000000, "$100"}, {9999500000, "$99.995"}, {1, "$0.00000001"}, {827116525, "$8.27116525"}, {300000, "$0.003"}} {
		it := CountsAsItem("dollar_amount", tc.units)
		if it.Name != "Counts as at most" || it.Value != tc.want {
			t.Errorf("%d: %+v", tc.units, it)
		}
	}
}

// The three places that round an amount up to 10^-8 units agree, and the two
// size bounds argrules keeps for a cap are the ones a spending limit keeps.
func TestUnitsCeilAgreesWithTheLegacyRounding(t *testing.T) {
	if argrules.UnitsPerOne != unitsPerOne || argrules.MaxOrderCostScale != MaxScale || argrules.MaxOrderCostWholeDigits != MaxWholeDigits {
		t.Fatal("argrules and toolcontrols disagree on the amount scale")
	}
	for _, s := range []string{"0", "1", "0.5", "0.00000001", "0.000000001", "0.999999999", "1.000000001", "99.995", "8.271165240739491",
		"9999999999.99999999", "92233720368.54775807", "92233720368.54775808", "92233720368.547758071"} {
		r, _ := new(big.Rat).SetString(s)
		a, aok := argrules.UnitsCeil(r)
		b, bok := unitsOf(r, true)
		if a != b || aok != bok {
			t.Errorf("%s: UnitsCeil (%d, %v), unitsOf (%d, %v)", s, a, aok, b, bok)
		}
	}
	for _, name := range []string{toolEquityOrder, toolCryptoOrder} {
		if !argrules.IsOrderTool(name) || !SymbolCapTool(name) {
			t.Errorf("%s is not an order tool in both packages", name)
		}
	}
}
