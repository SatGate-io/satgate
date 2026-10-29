package argrules

import (
	"encoding/json"
	"fmt"
	"math/big"
	"math/rand"
	"strings"
	"testing"

	"github.com/satgate-io/satgate/pkg/argrules/internal/releasedv2"
	"github.com/satgate-io/satgate/pkg/argrules/internal/releasedv3"
)

// A made-up value schema for tests only. It reads an order on an imaginary
// service: buy or sell YES contracts at an optional price. Selling YES at p
// opens the other side at 1 - p; with no price the order is valued at 1 per
// contract. reduce_only with ioc can only close a position.
const testSchema = "testbook_v1"

var testSchemaFields = []string{"market", "action", "count", "price", "tif", "reduce_only", "sizing", "amount", "replaces", "orders", "client_ref"}

func testSchemaRead(args map[string]json.RawMessage) (EventOrder, OrderCostWhy) {
	if _, ok := args["orders"]; ok {
		return EventOrder{}, WhyEventBatch
	}
	if _, ok := args["replaces"]; ok {
		return EventOrder{}, WhyEventAmendOrReplace
	}
	if _, ok := args["amount"]; ok {
		return EventOrder{}, WhyEventByAmount
	}
	if raw, ok := args["sizing"]; ok {
		if s, _ := ReadArgString(raw); s != "contracts" {
			return EventOrder{}, WhyEventByAmount
		}
	}
	tif, _ := ReadArgString(args["tif"])
	switch tif {
	case "ioc", "fok", "day":
	case "gtc":
		return EventOrder{}, WhyEventResting
	default:
		return EventOrder{}, WhyUnreadable
	}
	var o EventOrder
	o.Market, _ = ReadArgString(args["market"])
	cnt, ok := ReadArgDecimal(args["count"])
	if !ok {
		return EventOrder{}, WhyEventBadCount
	}
	o.Contracts = cnt
	if raw, ok := args["reduce_only"]; ok {
		b, ok := ReadArgBool(raw)
		if !ok {
			return EventOrder{}, WhyUnreadable
		}
		if b {
			if tif != "ioc" {
				return EventOrder{}, WhyEventClosingUnsure
			}
			o.Closing = true
			return o, OrderCostValued
		}
	}
	action, _ := ReadArgString(args["action"])
	var price *big.Rat
	if raw, ok := args["price"]; ok {
		p, ok := ReadArgDecimal(raw)
		if !ok {
			return EventOrder{}, WhyBadNumber
		}
		price = p
	}
	switch action {
	case "buy":
		o.Price = price
	case "sell":
		if price != nil {
			o.Price = new(big.Rat).Sub(big.NewRat(1, 1), price)
		}
	default:
		return EventOrder{}, WhyUnreadable
	}
	return o, OrderCostValued
}

func init() {
	must := func(err error) {
		if err != nil {
			panic(err)
		}
	}
	must(RegisterEventSchema(EventSchema{Name: testSchema, Fields: testSchemaFields, Read: testSchemaRead}))
	// A reader that tries to put its own text in a refusal.
	must(RegisterEventSchema(EventSchema{Name: "leaky_v1", Fields: []string{"market"}, Read: func(map[string]json.RawMessage) (EventOrder, OrderCostWhy) {
		return EventOrder{}, OrderCostWhy("market SECRET-123 is closed")
	}}))
	// A reader that panics.
	must(RegisterEventSchema(EventSchema{Name: "panicky_v1", Fields: []string{"market"}, Read: func(map[string]json.RawMessage) (EventOrder, OrderCostWhy) {
		panic("boom")
	}}))
}

func evArgs(t *testing.T, js string) map[string]json.RawMessage {
	t.Helper()
	m, err := objectOf([]byte(js))
	if err != nil {
		t.Fatalf("%s: %v", js, err)
	}
	return m
}

func buy(market, count, price string) string {
	s := `{"market":"` + market + `","action":"buy","count":"` + count + `","tif":"ioc"`
	if price != "" {
		s += `,"price":"` + price + `"`
	}
	return s + `}`
}

func sell(market, count, price string) string {
	s := `{"market":"` + market + `","action":"sell","count":"` + count + `","tif":"ioc"`
	if price != "" {
		s += `,"price":"` + price + `"`
	}
	return s + `}`
}

// The worked values: exact units for buys and sells, the smallest and the
// largest order.
func TestEventOrderMaxCostBuySell(t *testing.T) {
	for _, tc := range []struct {
		args      string
		units     int64
		contracts int64
	}{
		{buy("MKT-1", "10", "0.56"), 560000000, 1000},
		{sell("MKT-1", "10", "0.56"), 440000000, 1000},
		{buy("MKT-1", "2.50", "0.5675"), 141875000, 250},
		{buy("MKT-1", "0.01", "0.0001"), 100, 1},
		{buy("MKT-1", "5", "0.10"), 50000000, 500},
		{sell("MKT-1", "5", "0.90"), 50000000, 500},
		{buy("MKT-1", "500000", "0.9999"), 49995000000000, 50000000},
		{`{"market":"MKT-1","action":"buy","count":7,"price":0.5,"tif":"fok"}`, 350000000, 700},
	} {
		v, why := EventOrderMaxCost(testSchema, evArgs(t, tc.args))
		if !why.Valued() || v.Units != tc.units || v.Contracts != tc.contracts || v.Closing || v.Market != "MKT-1" {
			t.Errorf("%s: got %+v %q, want units %d contracts %d", tc.args, v, why, tc.units, tc.contracts)
		}
	}
}

// An order with no price that bounds it counts 1 per contract.
func TestEventOrderMaxCostNoPriceIsOnePerContract(t *testing.T) {
	for _, js := range []string{buy("M", "7", ""), sell("M", "7", "")} {
		v, why := EventOrderMaxCost(testSchema, evArgs(t, js))
		if !why.Valued() || v.Units != 700000000 {
			t.Errorf("%s: %+v %q", js, v, why)
		}
	}
}

// An order that can only close a position is valued at 0 and still carries
// its contract count.
func TestEventOrderClosingIsZeroWithContracts(t *testing.T) {
	v, why := EventOrderMaxCost(testSchema, evArgs(t, `{"market":"M","action":"sell","count":"3.5","tif":"ioc","reduce_only":true}`))
	if !why.Valued() || !v.Closing || v.Units != 0 || v.Contracts != 350 {
		t.Fatalf("%+v %q", v, why)
	}
	_, why = EventOrderMaxCost(testSchema, evArgs(t, `{"market":"M","action":"sell","count":"3","tif":"fok","reduce_only":true}`))
	if why != WhyEventClosingUnsure {
		t.Fatalf("reduce_only without ioc: %q", why)
	}
}

// Every refused shape returns its fixed reason and no value.
func TestEventOrderMaxCostRefusedShapes(t *testing.T) {
	long := strings.Repeat("M", 65)
	for _, tc := range []struct {
		schema, args string
		why          OrderCostWhy
	}{
		{"not_registered_v1", buy("M", "1", "0.5"), WhyEventUnknownSchema},
		{testSchema, `{"market":"M","action":"buy","count":"1","tif":"ioc","extra":1}`, WhyEventUnknownField},
		{testSchema, `{"market":"M","action":"buy","Count":"1","tif":"ioc"}`, WhyEventUnknownField},
		{testSchema, `{"action":"buy","count":"1","tif":"ioc"}`, WhyEventNoMarket},
		{testSchema, buy(long, "1", "0.5"), WhyEventNoMarket},
		{testSchema, buy("M 1", "1", "0.5"), WhyEventNoMarket},
		{testSchema, buy("M\u00e9", "1", "0.5"), WhyEventNoMarket},
		{testSchema, `{"market":7,"action":"buy","count":"1","tif":"ioc"}`, WhyEventNoMarket},
		{testSchema, `{"market":"M","action":"buy","tif":"ioc"}`, WhyEventBadCount},
		{testSchema, buy("M", "0", "0.5"), WhyEventBadCount},
		{testSchema, buy("M", "-1", "0.5"), WhyEventBadCount},
		{testSchema, buy("M", "1.001", "0.5"), WhyEventBadCount},
		{testSchema, buy("M", "1e3", "0.5"), WhyEventBadCount},
		{testSchema, buy("M", "10000000000", "0.5"), WhyEventTooManyContracts},
		{testSchema, buy("M", "1", "0"), WhyEventBadPrice},
		{testSchema, buy("M", "1", "1"), WhyEventBadPrice},
		{testSchema, buy("M", "1", "1.5"), WhyEventBadPrice},
		{testSchema, sell("M", "1", "1.5"), WhyEventBadPrice},
		{testSchema, buy("M", "1", "abc"), WhyBadNumber},
		{testSchema, `{"market":"M","action":"buy","count":"1","tif":"gtc"}`, WhyEventResting},
		{testSchema, `{"market":"M","action":"buy","count":"1","tif":"forever"}`, WhyUnreadable},
		{testSchema, `{"market":"M","action":"buy","amount":"5","tif":"ioc"}`, WhyEventByAmount},
		{testSchema, `{"market":"M","action":"buy","count":"1","sizing":"dollars","tif":"ioc"}`, WhyEventByAmount},
		{testSchema, `{"market":"M","action":"buy","count":"1","replaces":"o-1","tif":"ioc"}`, WhyEventAmendOrReplace},
		{testSchema, `{"orders":[]}`, WhyEventBatch},
		{testSchema, `{"market":"M","action":"hold","count":"1","tif":"ioc"}`, WhyUnreadable},
		{"leaky_v1", `{"market":"M"}`, WhyUnreadable},
		{"panicky_v1", `{"market":"M"}`, WhyUnreadable},
	} {
		v, why := EventOrderMaxCost(tc.schema, evArgs(t, tc.args))
		if why != tc.why || v != (EventOrderValue{}) {
			t.Errorf("%s %s: got %+v %q, want %q", tc.schema, tc.args, v, why, tc.why)
		}
	}
	// No arguments at all.
	if _, why := EventOrderMaxCost(testSchema, nil); why.Valued() {
		t.Error("an order with no arguments was valued")
	}
}

// Through a whole call: a repeated key or a case twin of name/arguments is
// unreadable, as argument rules read it.
func TestEventOrderMaxCostOfCallReadsStrictly(t *testing.T) {
	ok := `{"name":"place_event_order","arguments":` + buy("M", "1", "0.5") + `}`
	if v, why := EventOrderMaxCostOfCall(json.RawMessage(ok), "place_event_order", testSchema); !why.Valued() || v.Units != 50000000 {
		t.Fatalf("%+v %q", v, why)
	}
	for _, bad := range []string{
		`{"name":"place_event_order","arguments":{"market":"M","market":"N","action":"buy","count":"1","tif":"ioc"}}`,
		`{"name":"place_event_order","Arguments":{},"arguments":` + buy("M", "1", "0.5") + `}`,
		`{"name":"other_tool","arguments":` + buy("M", "1", "0.5") + `}`,
		`{"name":"place_event_order","arguments":[1]}`,
	} {
		if _, why := EventOrderMaxCostOfCall(json.RawMessage(bad), "place_event_order", testSchema); why != WhyUnreadable {
			t.Errorf("%s: %q", bad, why)
		}
	}
}

// Fuzz: the value is never zero for an opening order, never below the exact
// cost, and less than one unit above it.
func TestEventOrderMaxCostRoundsUpNeverZero(t *testing.T) {
	r := rand.New(rand.NewSource(7))
	for i := 0; i < 5000; i++ {
		hundredths := r.Int63n(100000000) + 1
		priceN := r.Int63n(99999999) + 1 // 8 decimals, 0 < p < 1
		count := FormatContracts(hundredths)
		price := fmt.Sprintf("0.%08d", priceN)
		v, why := EventOrderMaxCost(testSchema, evArgs(t, buy("M", count, price)))
		if !why.Valued() {
			t.Fatalf("%s @ %s: %q", count, price, why)
		}
		exact := new(big.Rat).Mul(big.NewRat(hundredths, 100), big.NewRat(priceN, 100000000))
		exact.Mul(exact, big.NewRat(UnitsPerOne, 1))
		u := new(big.Rat).SetInt64(v.Units)
		if v.Units < 1 || u.Cmp(exact) < 0 || new(big.Rat).Sub(u, exact).Cmp(big.NewRat(1, 1)) >= 0 {
			t.Fatalf("%s @ %s: units %d, exact %s", count, price, v.Units, exact.FloatString(4))
		}
	}
}

func TestRegisterEventSchemaRefusals(t *testing.T) {
	read := func(map[string]json.RawMessage) (EventOrder, OrderCostWhy) { return EventOrder{}, WhyUnreadable }
	for _, s := range []EventSchema{
		{Name: "", Fields: []string{"a"}, Read: read},
		{Name: "Upper_v1", Fields: []string{"a"}, Read: read},
		{Name: "1abc", Fields: []string{"a"}, Read: read},
		{Name: strings.Repeat("a", 41), Fields: []string{"a"}, Read: read},
		{Name: "ok_but_no_reader", Fields: []string{"a"}},
		{Name: "ok_but_no_fields", Read: read},
		{Name: "ok_but_twice", Fields: []string{"a", "a"}, Read: read},
		{Name: "ok_but_bad_field", Fields: []string{"a b"}, Read: read},
		{Name: testSchema, Fields: []string{"a"}, Read: read}, // already registered
	} {
		if err := RegisterEventSchema(s); err == nil {
			t.Errorf("%q was registered", s.Name)
		}
	}
	if EventSchemaRegistered("ok_but_no_reader") {
		t.Fatal("a refused schema is registered")
	}
	names := strings.Join(EventSchemaNames(), ",")
	if !strings.Contains(names, testSchema) {
		t.Fatalf("names: %s", names)
	}
}

const eventRuleDoc = `{"v":1,"rules":[{"tool":"place_event_order","shapes":[{"event_order":{"max_contracts":"10","schema":"testbook_v1","max_cost":"5.60"}}]}]}`

// The event_order condition parses, writes canonical bytes and round-trips.
func TestEventOrderRuleParseAndCanonical(t *testing.T) {
	rules, err := ParseJSON([]byte(eventRuleDoc))
	if err != nil {
		t.Fatal(err)
	}
	c := rules[0].Shapes[0].Fields[0]
	if c.Field != EventOrderField || c.EventOrder == nil || c.EventOrder.Schema != testSchema || c.EventOrder.MaxCost.String() != "5.60" || c.EventOrder.MaxContracts.String() != "10" {
		t.Fatalf("%+v", c)
	}
	doc, err := Marshal(rules)
	if err != nil {
		t.Fatal(err)
	}
	want := `{"rules":[{"shapes":[{"event_order":{"max_contracts":"10","max_cost":"5.60","schema":"testbook_v1"}}],"tool":"place_event_order"}],"v":1}`
	if string(doc) != want {
		t.Fatalf("canonical:\n got %s\nwant %s", doc, want)
	}
	if got := Describe(rules)[0]; got != "place_event_order: allowed only when event contract order (value schema testbook_v1) costs at most 5.60 and is at most 10 contracts" {
		t.Fatalf("describe: %s", got)
	}
	if got := AgentLines(rules)[0]; !strings.Contains(got, "costs at most 5.60 and is at most 10 contracts") {
		t.Fatalf("agent line: %s", got)
	}
}

// A rule naming a schema that is not registered does not parse (fail closed),
// and the condition's bounds are enforced.
func TestEventOrderRuleRefusedAtParse(t *testing.T) {
	for _, doc := range []string{
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"schema":"not_registered_v1","max_cost":"5"}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"schema":"","max_cost":"5"}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"schema":"testbook_v1"}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"schema":"testbook_v1","max_cost":"0"}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"schema":"testbook_v1","max_cost":"1.123456789"}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"schema":"testbook_v1","max_cost":"12345678901"}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"schema":"testbook_v1","max_contracts":"1.001"}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"schema":"testbook_v1","max_contracts":"-1"}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"schema":"testbook_v1","max_cost":"5","one_of":["a"]}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"schema":7,"max_cost":"5"}}]}]}`,
		// schema and max_contracts are only read under event_order.
		`{"v":1,"rules":[{"tool":"t","shapes":[{"order":{"schema":"testbook_v1","max_cost":"5"}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"count":{"max_contracts":"5"}}]}]}`,
		// Without schema, event_order with max_cost is the order-cost key on
		// the wrong field, refused as before.
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"max_cost":"5"}}]}]}`,
	} {
		if _, err := ParseJSON([]byte(doc)); err == nil {
			t.Errorf("parsed: %s", doc)
		}
	}
	// An argument that happens to be called event_order keeps its meaning.
	plain := `{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"one_of":["x"]}}]}]}`
	rules, err := ParseJSON([]byte(plain))
	if err != nil || rules[0].Shapes[0].Fields[0].EventOrder != nil || rules[0].Shapes[0].Fields[0].OneOf == nil {
		t.Fatalf("plain event_order field: %+v %v", rules, err)
	}
	if d := Check(rules, "t", json.RawMessage(`{"name":"t","arguments":{"event_order":"x"}}`)); d != nil {
		t.Fatalf("plain event_order field refused: %v", d)
	}
}

func eventCall(args string) json.RawMessage {
	return json.RawMessage(`{"name":"place_event_order","arguments":` + args + `}`)
}

// Boundaries of the cap and the contracts cap; a closing order passes the
// cost cap and is still held to the contracts cap.
func TestEventOrderRuleCheckBoundaries(t *testing.T) {
	rules, err := ParseJSON([]byte(eventRuleDoc))
	if err != nil {
		t.Fatal(err)
	}
	for _, tc := range []struct {
		args string
		pass bool
	}{
		{buy("M", "10", "0.56"), true},
		{buy("M", "10", "0.5601"), false},
		{sell("M", "10", "0.44"), true},    // 10 x 0.56
		{sell("M", "10", "0.4399"), false}, // 10 x 0.5601
		{buy("M", "10.00", "0.01"), true},
		{buy("M", "10.01", "0.01"), false},
		{`{"market":"M","action":"sell","count":"10","tif":"ioc","reduce_only":true}`, true},
		{`{"market":"M","action":"sell","count":"10.01","tif":"ioc","reduce_only":true}`, false},
		{`{"market":"M","action":"buy","count":"1","tif":"gtc","price":"0.1"}`, false},
	} {
		d := Check(rules, "place_event_order", eventCall(tc.args))
		if (d == nil) != tc.pass {
			t.Errorf("%s: denial %v, want pass=%v", tc.args, d, tc.pass)
		}
		if d != nil && (d.Reason != ReasonNoShape || d.Field != EventOrderField) {
			t.Errorf("%s: denial %+v", tc.args, d)
		}
	}
}

// A hand-built rule naming a schema that is not registered refuses at call.
func TestEventOrderRuleUnregisteredSchemaRefusedAtCall(t *testing.T) {
	five, _ := ParseDecimal("5")
	rules := []Rule{{Tool: "place_event_order", Shapes: []Shape{{Fields: []Condition{{Field: EventOrderField, EventOrder: &EventOrderLimit{Schema: "never_registered_v1", MaxCost: five}}}}}}}
	d := Check(rules, "place_event_order", eventCall(buy("M", "1", "0.5")))
	if d == nil || !strings.Contains(d.Allowed, string(WhyEventUnknownSchema)) {
		t.Fatalf("denial %+v", d)
	}
}

// A refusal carries the rule's words and a fixed reason, never the call's
// market, count or price.
func TestEventOrderRefusalNeverEchoesValues(t *testing.T) {
	rules, _ := ParseJSON([]byte(eventRuleDoc))
	for _, args := range []string{
		buy("SECRETMKT-77", "12.34", "0.9876"),
		`{"market":"SECRETMKT-77","action":"buy","count":"12.34","tif":"gtc","price":"0.9876"}`,
		`{"market":"SECRETMKT-77","action":"buy","count":"12.34","tif":"ioc","price":"0.9876","bogus":"x"}`,
	} {
		d := Check(rules, "place_event_order", eventCall(args))
		if d == nil {
			t.Fatalf("%s passed", args)
		}
		text := d.Error() + " " + d.Allowed
		for _, v := range []string{"SECRETMKT", "12.34", "0.9876", "bogus"} {
			if strings.Contains(text, v) {
				t.Errorf("refusal %q echoes %q", text, v)
			}
		}
	}
}

// Every release before the event keys refuses a document that carries one,
// so an older verifier denies the token instead of dropping the cap.
func TestOldArgrulesParsersRefuseEventOrder(t *testing.T) {
	for _, doc := range []string{
		eventRuleDoc,
		`{"v":1,"rules":[{"tool":"place_event_order","shapes":[{"event_order":{"schema":"testbook_v1","max_cost":"5"}}]}]}`,
		`{"v":1,"rules":[{"tool":"place_event_order","shapes":[{"event_order":{"schema":"testbook_v1","max_contracts":"5"}}]}]}`,
	} {
		if _, err := ParseJSON([]byte(doc)); err != nil {
			t.Fatalf("current parser refused %s: %v", doc, err)
		}
		if _, err := releasedv3.ParseJSON([]byte(doc)); err == nil || !strings.Contains(err.Error(), "unknown condition") {
			t.Errorf("the parser at e22202ba read %s: %v", doc, err)
		}
		if _, err := releasedv2.ParseJSON([]byte(doc)); err == nil {
			t.Errorf("the parser at 0e60d5f8 read %s", doc)
		}
		rules, _ := ParseJSON([]byte(doc))
		word, err := ScopeValue(rules)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := releasedv3.ParseScopeValue(word); err == nil {
			t.Errorf("the parser at e22202ba read the word for %s", doc)
		}
		if _, err := releasedv3.Collect([]string{"scope = place_event_order", "scope = " + word}, func(s, tool string) bool { return s == tool }); err == nil {
			t.Errorf("the old Collect read the word for %s", doc)
		}
	}
}

// Every existing kind of document keeps its canonical bytes and its
// decisions: the current code and the snapshot of the release before the
// event keys agree byte for byte and call for call.
func TestExistingRuleDocumentsDecideTheSame(t *testing.T) {
	docs := []string{
		`{"v":1,"rules":[{"tool":"place_equity_order","shapes":[{"account_number":{"one_of":["5RH12345"]},"order":{"max_cost":"100"}}]}]}`,
		`{"v":1,"rules":[{"tool":"place_crypto_order","shapes":[{"dollar_amount":{"max":"100","min":"1"}},{"side":{"one_of":["sell"],"ascii_case_insensitive":true}}]}]}`,
		`{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET"],"ascii_case_insensitive":true}},{"method":{"one_of":["POST"]},"url":{"url":{"hosts":["www.wixapis.com"],"path_suffixes":["/query"]}}}]}]}`,
		`{"v":1,"rules":[{"tool":"t","shapes":[{"event_order":{"one_of":["x"]},"note":{"absent":true}}]}]}`,
	}
	calls := map[string][]string{
		"place_equity_order": {
			`{"account_number":"5RH12345","dollar_amount":"99"}`,
			`{"account_number":"5RH12345","dollar_amount":"101"}`,
			`{"account_number":"5RH12345","quantity":"2","limit_price":"50","side":"buy","type":"limit","time_in_force":"gfd"}`,
			`{"account_number":"other","dollar_amount":"1"}`,
		},
		"place_crypto_order": {`{"dollar_amount":"50"}`, `{"dollar_amount":"500"}`, `{"side":"SELL","dollar_amount":"500"}`, `{"Dollar_amount":"5"}`},
		"CallWixSiteAPI":     {`{"method":"get","url":"https://x"}`, `{"method":"POST","url":"https://www.wixapis.com/a/query"}`, `{"method":"POST","url":"https://evil.example/query"}`},
		"t":                  {`{"event_order":"x"}`, `{"event_order":"y"}`, `{"event_order":"x","note":"n"}`},
	}
	for _, doc := range docs {
		now, err := ParseJSON([]byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		then, err := releasedv3.ParseJSON([]byte(doc))
		if err != nil {
			t.Fatal(err)
		}
		a, _ := Marshal(now)
		b, _ := releasedv3.Marshal(then)
		if string(a) != string(b) {
			t.Fatalf("canonical bytes differ:\n now %s\nthen %s", a, b)
		}
		tool := now[0].Tool
		for _, args := range calls[tool] {
			params := json.RawMessage(`{"name":"` + tool + `","arguments":` + args + `}`)
			dn := Check(now, tool, params)
			dt := releasedv3.Check(then, tool, params)
			if (dn == nil) != (dt == nil) || (dn != nil && (dn.Reason != dt.Reason || dn.Field != dt.Field || dn.Allowed != dt.Allowed)) {
				t.Errorf("%s %s: now %+v, then %+v", tool, args, dn, dt)
			}
		}
	}
}
