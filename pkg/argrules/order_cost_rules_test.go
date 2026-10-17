package argrules

import (
	"encoding/json"
	"strings"
	"testing"

	old "github.com/satgate-io/satgate/pkg/argrules/internal/releasedv2"
)

const capDoc = `{"v":1,"rules":[` +
	`{"tool":"place_equity_order","shapes":[{"account_number":{"one_of":["5RH12345"]},"order":{"max_cost":"100"}}]},` +
	`{"tool":"place_crypto_order","shapes":[{"order":{"max_cost":"100"},"rhs_account_number":{"one_of":["5RH12345"]}}]}]}`

const capDocCanonical = `{"rules":[` +
	`{"shapes":[{"account_number":{"one_of":["5RH12345"]},"order":{"max_cost":"100"}}],"tool":"place_equity_order"},` +
	`{"shapes":[{"order":{"max_cost":"100"},"rhs_account_number":{"one_of":["5RH12345"]}}],"tool":"place_crypto_order"}],"v":1}`

func mustRules(t *testing.T, doc string) []Rule {
	t.Helper()
	r, err := ParseJSON([]byte(doc))
	if err != nil {
		t.Fatalf("%v\n%s", err, doc)
	}
	return r
}

func eqCall(argsJSON string) json.RawMessage {
	return json.RawMessage(`{"name":"place_equity_order","arguments":` + argsJSON + `}`)
}

func capOf(max string) []Rule {
	return []Rule{{Tool: "place_equity_order", Shapes: []Shape{{Fields: []Condition{{Field: "order", MaxCost: capDecimal(max)}}}}}}
}

func capDecimal(s string) *Decimal {
	d, err := ParseDecimal(s)
	if err != nil {
		panic(err)
	}
	return d
}

func TestMaxCostConditionParseMarshalRoundTrip(t *testing.T) {
	rules := mustRules(t, capDoc)
	if rules[0].Shapes[0].Fields[1].MaxCost == nil || rules[0].Shapes[0].Fields[1].Field != "order" {
		t.Fatalf("not parsed as an order cap: %+v", rules[0].Shapes[0].Fields)
	}
	out, err := Marshal(rules)
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != capDocCanonical {
		t.Fatalf("canonical form:\n got %s\nwant %s", out, capDocCanonical)
	}
	again, err := Marshal(mustRules(t, string(out)))
	if err != nil || string(again) != string(out) {
		t.Fatalf("canonical form is not a fixed point: %s %v", again, err)
	}
	// The word round trips too.
	word, err := ScopeValue(rules)
	if err != nil {
		t.Fatal(err)
	}
	back, err := ParseScopeValue(word)
	if err != nil {
		t.Fatal(err)
	}
	if b, _ := Marshal(back); string(b) != capDocCanonical {
		t.Fatalf("word round trip: %s", b)
	}
	// Owner and agent wording.
	if got := Describe(rules)[0]; got != "place_equity_order: allowed only when account_number is 5RH12345 and order cost is at most 100" {
		t.Errorf("Describe = %q", got)
	}
	want := "place_equity_order: allowed only when account_number is 5RH12345 and the order's cost is at most 100 (dollar_amount, or quantity times limit_price for a day limit buy of shares)"
	if got := AgentLines(rules)[0]; got != want {
		t.Errorf("AgentLines = %q", got)
	}
	if got := rules[0].Shapes[0].Fields[1].AgentText(); got != "the order's cost (dollar_amount, or quantity times limit_price for a day limit buy of shares) must be at most 100" {
		t.Errorf("AgentText = %q", got)
	}
}

func TestMaxCostOnlyOnOrderToolsAndWithinBounds(t *testing.T) {
	doc := func(tool, field, cond string) string {
		return `{"v":1,"rules":[{"tool":"` + tool + `","shapes":[{"` + field + `":` + cond + `}]}]}`
	}
	ok := `{"max_cost":"100"}`
	for _, tool := range []string{"place_equity_order", "place_crypto_order", "Place_Equity_Order", "PLACE_CRYPTO_ORDER"} {
		if _, err := ParseJSON([]byte(doc(tool, "order", ok))); err != nil {
			t.Errorf("%s refused: %v", tool, err)
		}
	}
	// Options and every other tool are refused at parse.
	for _, tool := range []string{"place_option_order", "review_option_order", "review_equity_order", "preview_crypto_order",
		"cancel_equity_order", "exercise_option", "get_quote", "robinhood-2_place_equity_order", "place_equity_order2"} {
		if _, err := ParseJSON([]byte(doc(tool, "order", ok))); err == nil || !strings.Contains(err.Error(), "max_cost is only for") {
			t.Errorf("%s: %v", tool, err)
		}
	}
	// The condition is written only under the field name "order".
	for _, field := range []string{"dollar_amount", "Order", "orders", "quantity", "order.max_cost"} {
		if _, err := ParseJSON([]byte(doc("place_equity_order", field, ok))); err == nil {
			t.Errorf("field %s accepted", field)
		}
	}
	// Mixing with any other condition kind on the field is refused.
	for _, cond := range []string{
		`{"max_cost":"100","max":"5"}`, `{"max_cost":"100","min":"1"}`, `{"max_cost":"100","max":"5","min":"1"}`,
		`{"max_cost":"100","one_of":["a"]}`, `{"max_cost":"100","absent":true}`,
		`{"max_cost":"100","ascii_case_insensitive":true}`, `{"max_cost":"100","url":{"hosts":["a.example"]}}`,
		`{"max_cost":"100","other":"1"}`,
	} {
		if _, err := ParseJSON([]byte(doc("place_equity_order", "order", cond))); err == nil {
			t.Errorf("%s accepted", cond)
		}
	}
	// Bounds: a plain decimal string above zero, at most 10 whole and 8 fractional digits.
	for _, v := range []string{`"0.00000001"`, `"9999999999.99999999"`, `"1"`, `"0.5"`} {
		if _, err := ParseJSON([]byte(doc("place_equity_order", "order", `{"max_cost":`+v+`}`))); err != nil {
			t.Errorf("max_cost %s refused: %v", v, err)
		}
	}
	for _, v := range []string{`"0"`, `"0.0"`, `"-1"`, `"-0"`, `"0.000000001"`, `"10000000000"`, `"99999999999.5"`,
		`"01"`, `"1e2"`, `" 1"`, `"+1"`, `".5"`, `"5."`, `""`, `100`, `null`, `true`, `["1"]`, `{"a":1}`} {
		if _, err := ParseJSON([]byte(doc("place_equity_order", "order", `{"max_cost":`+v+`}`))); err == nil {
			t.Errorf("max_cost %s accepted", v)
		}
	}
	// A condition on "order" that is not max_cost keeps its old meaning.
	for _, cond := range []string{`{"one_of":["x"]}`, `{"max":"5"}`, `{"absent":true}`} {
		if _, err := ParseJSON([]byte(doc("place_equity_order", "order", cond))); err != nil {
			t.Errorf("legacy %s refused: %v", cond, err)
		}
	}
	// A parse error does not repeat the cap's text.
	_, err := ParseJSON([]byte(doc("place_equity_order", "order", `{"max_cost":"123456789012"}`)))
	if err == nil || strings.Contains(err.Error(), "123456789012") {
		t.Errorf("error: %v", err)
	}
}

func TestPerOrderCapBoundaries(t *testing.T) {
	cap100 := capOf("100")
	for _, tc := range []struct {
		name string
		cap  []Rule
		a    string
		pass bool
	}{
		{"1 x 100.00 against 100 (equal)", cap100, `{"side":"buy","type":"limit","time_in_force":"gfd","quantity":"1","limit_price":"100.00"}`, true},
		{"1 x 100.01 against 100", cap100, `{"side":"buy","type":"limit","time_in_force":"gfd","quantity":"1","limit_price":"100.01"}`, false},
		{"0.5 x 199.99 against 100", cap100, `{"side":"buy","type":"limit","time_in_force":"gfd","quantity":"0.5","limit_price":"199.99"}`, true},
		{"0.5 x 199.99 against 99.99", capOf("99.99"), `{"side":"buy","type":"limit","time_in_force":"gfd","quantity":"0.5","limit_price":"199.99"}`, false},
		{"0.5 x 199.99 against 99.995", capOf("99.995"), `{"side":"buy","type":"limit","time_in_force":"gfd","quantity":"0.5","limit_price":"199.99"}`, true},
		{"8.27116525 against 8.27116524", capOf("8.27116524"), `{"side":"buy","type":"limit","time_in_force":"gfd","quantity":"0.00012345","limit_price":"67000.12345678"}`, false},
		{"8.27116525 against 8.27116525", capOf("8.27116525"), `{"side":"buy","type":"limit","time_in_force":"gfd","quantity":"0.00012345","limit_price":"67000.12345678"}`, true},
		{"dollars 100 (equal)", cap100, `{"dollar_amount":"100"}`, true},
		{"dollars 100.01", cap100, `{"dollar_amount":"100.01"}`, false},
		{"dollars json number", cap100, `{"dollar_amount":99.99}`, true},
		{"cap 0.00000001, one unit", capOf("0.00000001"), `{"dollar_amount":"0.00000001"}`, true},
		{"cap 0.00000001, just over", capOf("0.00000001"), `{"dollar_amount":"0.000000011"}`, false},
	} {
		d := Check(tc.cap, "place_equity_order", eqCall(tc.a))
		if (d == nil) != tc.pass {
			t.Errorf("%s: denial = %v", tc.name, d)
		}
	}
}

func TestPerOrderCapRefusesUnvaluableShapesAsArgumentDenied(t *testing.T) {
	const marker = "7731.4242"
	rules := capOf("100")
	for _, tc := range []struct {
		name, a, reason string
	}{
		{"market by shares", `{"side":"buy","type":"market","quantity":"` + marker + `"}`, string(WhyNotALimitBuy)},
		{"gtc limit buy", `{"side":"buy","type":"limit","time_in_force":"gtc","quantity":"1","limit_price":"` + marker + `"}`, string(WhyNotADayOrder)},
		{"no time in force", `{"side":"buy","type":"limit","quantity":"1","limit_price":"` + marker + `"}`, string(WhyNoTimeInForce)},
		{"overnight", `{"side":"buy","type":"limit","time_in_force":"gfd","market_hours":"all_day_hours","quantity":"1","limit_price":"` + marker + `"}`, string(WhyOvernightSession)},
		{"both sizes", `{"dollar_amount":"` + marker + `","quantity":"1"}`, string(WhyBothSizes)},
		{"neither size", `{"symbol":"AAPL"}`, string(WhyNoSize)},
		{"dollars with a price", `{"dollar_amount":"` + marker + `","limit_price":"1"}`, string(WhyDollarsWithPrice)},
		{"sell by shares", `{"side":"sell","type":"limit","time_in_force":"gfd","quantity":"1","limit_price":"` + marker + `"}`, string(WhyNotABuy)},
		{"too large", `{"dollar_amount":"99999999999999"}`, string(WhyCostTooLarge)},
		{"zero", `{"dollar_amount":"0"}`, string(WhyNotAboveZero)},
		{"a twin", `{"dollar_amount":"1","Quantity":"` + marker + `"}`, string(WhyUnreadable)},
	} {
		d := Check(rules, "place_equity_order", eqCall(tc.a))
		if d == nil {
			t.Errorf("%s: passed", tc.name)
			continue
		}
		if d.Field != "order" || d.Reason != ReasonNoShape {
			t.Errorf("%s: field %q reason %q", tc.name, d.Field, d.Reason)
		}
		if !strings.Contains(d.Allowed, "must be at most 100") || !strings.HasSuffix(d.Allowed, tc.reason) {
			t.Errorf("%s: allowed = %q", tc.name, d.Allowed)
		}
		if strings.Contains(d.Allowed, "7731") || strings.Contains(d.Error(), "7731") {
			t.Errorf("%s: the denial holds a call value: %q", tc.name, d.Allowed)
		}
	}
	// Crypto by quantity, through a crypto rule.
	crypto := []Rule{{Tool: "place_crypto_order", Shapes: []Shape{{Fields: []Condition{{Field: "order", MaxCost: capDecimal("100")}}}}}}
	d := Check(crypto, "place_crypto_order", json.RawMessage(`{"name":"place_crypto_order","arguments":{"side":"buy","type":"limit","time_in_force":"gfd","quantity":"0.001","limit_price":"50000"}}`))
	if d == nil || !strings.HasSuffix(d.Allowed, "crypto orders by quantity can't be counted yet") {
		t.Errorf("crypto by quantity: %+v", d)
	}
	// Crypto by dollars passes.
	if d := Check(crypto, "place_crypto_order", json.RawMessage(`{"name":"place_crypto_order","arguments":{"dollar_amount":"50","symbol":"BTC"}}`)); d != nil {
		t.Errorf("crypto by dollars: %+v", d)
	}
}

func TestPerOrderCapWithAccountPinAndOtherArguments(t *testing.T) {
	rules := mustRules(t, capDoc)
	good := `{"account_number":"5RH12345","symbol":"AAPL","side":"buy","type":"limit","time_in_force":"gfd","quantity":"2","limit_price":"49.5"}`
	if d := Check(rules, "place_equity_order", eqCall(good)); d != nil {
		t.Fatalf("denied: %+v", d)
	}
	// Another account: refused, and the denial names the pin.
	d := Check(rules, "place_equity_order", eqCall(strings.Replace(good, "5RH12345", "5RH99999", 1)))
	if d == nil || d.Field != "account_number" {
		t.Fatalf("wrong account: %+v", d)
	}
	// An argument literally named "order" (or "Order") is not read and does not
	// make the cap pass or fail.
	for _, extra := range []string{`,"order":"1"`, `,"Order":"1"`, `,"order":{"max_cost":"1000000"}`} {
		if d := Check(rules, "place_equity_order", eqCall(strings.TrimSuffix(good, "}")+extra+"}")); d != nil {
			t.Errorf("%s: %+v", extra, d)
		}
		over := strings.Replace(good, `"quantity":"2"`, `"quantity":"3"`, 1)
		if d := Check(rules, "place_equity_order", eqCall(strings.TrimSuffix(over, "}")+extra+"}")); d == nil {
			t.Errorf("%s: an over-cap order passed", extra)
		}
	}
	// A case twin of a field the cost reads is refused as unreadable by the
	// cap itself.
	if d := Check(rules, "place_equity_order", eqCall(strings.TrimSuffix(good, "}")+`,"Dollar_Amount":"1"}`)); d == nil {
		t.Error("a twin of dollar_amount passed")
	}
}

func TestMaxCostWordIsAScopeWordAndCollects(t *testing.T) {
	word, err := ScopeValue(mustRules(t, capDoc))
	if err != nil {
		t.Fatal(err)
	}
	rules, err := Collect([]string{"scope = place_equity_order,place_crypto_order", "scope = " + word}, func(scope, tool string) bool { return true })
	if err != nil || len(rules) != 2 {
		t.Fatalf("%d %v", len(rules), err)
	}
	if sum, err := DocumentsSHA256([]string{"scope = " + word}); err != nil || len(sum) != 64 {
		t.Fatalf("%q %v", sum, err)
	}
}

// ---- old verifiers and existing documents ----

func TestOldArgrulesParserRefusesMaxCost(t *testing.T) {
	if _, err := ParseJSON([]byte(capDoc)); err != nil {
		t.Fatalf("the current parser refused it: %v", err)
	}
	// The frozen parser (public commit 0e60d5f8, the last main before max_cost).
	_, err := old.ParseJSON([]byte(capDoc))
	if err == nil || !strings.Contains(err.Error(), `unknown condition "max_cost"`) {
		t.Fatalf("the old parser did not refuse the document: %v", err)
	}
	// Alone, and next to a legacy condition.
	for _, doc := range []string{
		`{"v":1,"rules":[{"tool":"place_equity_order","shapes":[{"order":{"max_cost":"100"}}]}]}`,
		`{"v":1,"rules":[{"tool":"place_crypto_order","shapes":[{"dollar_amount":{"max":"100"},"order":{"max_cost":"100"}}]}]}`,
	} {
		if _, err := ParseJSON([]byte(doc)); err != nil {
			t.Fatalf("current parser: %v", err)
		}
		if _, err := old.ParseJSON([]byte(doc)); err == nil || !strings.Contains(err.Error(), "max_cost") {
			t.Errorf("old parser: %v", err)
		}
	}
	// A whole rule word built by the current code is refused as a word by the
	// old one, and by its Collect.
	word, err := ScopeValue(mustRules(t, capDoc))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := old.ParseScopeValue(word); err == nil {
		t.Error("the old parser read a rule word with max_cost")
	}
	if _, err := old.Collect([]string{"scope = place_equity_order", "scope = " + word}, func(scope, tool string) bool { return true }); err == nil {
		t.Error("the old Collect read a rule word with max_cost")
	}
	// The old parser takes a document without the new key, and the two read it
	// to the same canonical bytes.
	plain := `{"v":1,"rules":[{"tool":"place_equity_order","shapes":[{"account_number":{"one_of":["5RH12345"]},"dollar_amount":{"max":"100"},"limit_price":{"absent":true},"quantity":{"absent":true},"stop_price":{"absent":true}}]}]}`
	a, err := old.ParseJSON([]byte(plain))
	if err != nil {
		t.Fatal(err)
	}
	b := mustRules(t, plain)
	ao, _ := old.Marshal(a)
	bo, _ := Marshal(b)
	if string(ao) != string(bo) {
		t.Fatalf("canonical forms differ:\n%s\n%s", ao, bo)
	}
}

// TestExistingRuleDocumentsKeepTheirCanonicalBytes is the golden for rule
// documents written before max_cost: the recommended dollars cap and the older
// order caps. Each one parses and marshals to exactly these bytes, and to the
// bytes the frozen parser (0e60d5f8) writes.
func TestExistingRuleDocumentsKeepTheirCanonicalBytes(t *testing.T) {
	for _, tc := range []struct{ name, in, want string }{
		{
			"recommended dollars cap",
			`{"rules":[{"shapes":[{"stop_price":{"absent":true},"quantity":{"absent":true},"limit_price":{"absent":true},"dollar_amount":{"max":"100"},"account_number":{"one_of":["5RH12345"]}}],"tool":"place_equity_order"}],"v":1}`,
			`{"rules":[{"shapes":[{"account_number":{"one_of":["5RH12345"]},"dollar_amount":{"max":"100"},"limit_price":{"absent":true},"quantity":{"absent":true},"stop_price":{"absent":true}}],"tool":"place_equity_order"}],"v":1}`,
		},
		{
			"older order cap with a range and a case-insensitive type",
			`{"v":1,"rules":[{"tool":"place_crypto_order","shapes":[{"type":{"one_of":["limit"],"ascii_case_insensitive":true},"quantity":{"min":"0.001","max":"5"},"limit_price":{"max":"70000.5"},"rhs_account_number":{"one_of":["5RH12345"]}},{"type":{"one_of":["market"]},"dollar_amount":{"max":"100"}}]}]}`,
			`{"rules":[{"shapes":[{"limit_price":{"max":"70000.5"},"quantity":{"max":"5","min":"0.001"},"rhs_account_number":{"one_of":["5RH12345"]},"type":{"ascii_case_insensitive":true,"one_of":["limit"]}},{"dollar_amount":{"max":"100"},"type":{"one_of":["market"]}}],"tool":"place_crypto_order"}],"v":1}`,
		},
		{
			"a legacy condition on a field called order",
			`{"v":1,"rules":[{"tool":"place_equity_order","shapes":[{"order":{"max":"5"}}]}]}`,
			`{"rules":[{"shapes":[{"order":{"max":"5"}}],"tool":"place_equity_order"}],"v":1}`,
		},
		{
			"url rule",
			`{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["POST"],"ascii_case_insensitive":true},"url":{"url":{"hosts":["www.wixapis.com"],"path_suffixes":["/query"]}}}]}]}`,
			`{"rules":[{"shapes":[{"method":{"ascii_case_insensitive":true,"one_of":["POST"]},"url":{"url":{"hosts":["www.wixapis.com"],"path_suffixes":["/query"]}}}],"tool":"CallWixSiteAPI"}],"v":1}`,
		},
	} {
		got, err := Marshal(mustRules(t, tc.in))
		if err != nil || string(got) != tc.want {
			t.Errorf("%s:\n got %s (%v)\nwant %s", tc.name, got, err, tc.want)
			continue
		}
		oldRules, err := old.ParseJSON([]byte(tc.in))
		if err != nil {
			t.Errorf("%s: the frozen parser refused it: %v", tc.name, err)
			continue
		}
		if oldBytes, _ := old.Marshal(oldRules); string(oldBytes) != tc.want {
			t.Errorf("%s: the frozen parser writes %s", tc.name, oldBytes)
		}
	}
}

// Documents without the new key are checked as they always were: the same
// verdicts as the frozen verifier on a table of calls.
func TestExistingRulesDecideAsTheFrozenVerifierDid(t *testing.T) {
	doc := `{"v":1,"rules":[{"tool":"place_equity_order","shapes":[{"account_number":{"one_of":["5RH12345"]},"dollar_amount":{"max":"100"},"limit_price":{"absent":true},"quantity":{"absent":true},"stop_price":{"absent":true}}]}]}`
	cur := mustRules(t, doc)
	frozen, err := old.ParseJSON([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	for _, a := range []string{
		`{"account_number":"5RH12345","dollar_amount":"100"}`,
		`{"account_number":"5RH12345","dollar_amount":"100.01"}`,
		`{"account_number":"5RH12345","dollar_amount":"0"}`,
		`{"account_number":"5RH12345","dollar_amount":"-5"}`,
		`{"account_number":"5RH12345","quantity":"1","limit_price":"5"}`,
		`{"account_number":"5RH12345","dollar_amount":"5","quantity":"1"}`,
		`{"account_number":"5RH99999","dollar_amount":"5"}`,
		`{"dollar_amount":"5"}`,
		`{"account_number":"5RH12345","Dollar_Amount":"5","dollar_amount":"5"}`,
		`{"account_number":"5RH12345","dollar_amount":1e2}`,
	} {
		d1 := Check(cur, "place_equity_order", eqCall(a))
		d2 := old.Check(frozen, "place_equity_order", eqCall(a))
		if (d1 == nil) != (d2 == nil) || (d1 != nil && (d1.Field != d2.Field || d1.Reason != d2.Reason || d1.Allowed != d2.Allowed)) {
			t.Errorf("%s: current %+v, frozen %+v", a, d1, d2)
		}
	}
}
