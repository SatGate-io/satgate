package argrules

import (
	"bytes"
	"encoding/json"
	"math/big"
	"strings"
	"testing"
)

// args builds a call's arguments from key, raw-JSON pairs.
func args(t testing.TB, pairs ...string) map[string]json.RawMessage {
	t.Helper()
	var b strings.Builder
	b.WriteString("{")
	for i := 0; i < len(pairs); i += 2 {
		if i > 0 {
			b.WriteString(",")
		}
		b.WriteString(`"` + pairs[i] + `":` + pairs[i+1])
	}
	b.WriteString("}")
	m, err := objectOf([]byte(b.String()))
	if err != nil {
		t.Fatalf("bad test arguments %s: %v", b.String(), err)
	}
	return m
}

// dayBuy is an equity limit buy of shares that OrderMaxCost values: a day
// order in regular hours. Pairs given in over replace or add to it; a value of
// "-" removes the key.
func dayBuy(t testing.TB, over ...string) map[string]json.RawMessage {
	t.Helper()
	m := args(t,
		"symbol", `"AAPL"`, "side", `"buy"`, "type", `"limit"`, "time_in_force", `"gfd"`,
		"quantity", `"1"`, "limit_price", `"100.00"`)
	for i := 0; i < len(over); i += 2 {
		if over[i+1] == "-" {
			delete(m, over[i])
			continue
		}
		m[over[i]] = json.RawMessage(over[i+1])
	}
	return m
}

func mustUnits(t *testing.T, tool string, a map[string]json.RawMessage) int64 {
	t.Helper()
	c, why := OrderMaxCost(tool, a)
	if !why.Valued() {
		t.Fatalf("not valued: %s", why)
	}
	return c.Units
}

func TestOrderMaxCostDollarsAlone(t *testing.T) {
	for _, tool := range []string{"place_equity_order", "place_crypto_order"} {
		for _, tc := range []struct {
			name string
			raw  string
			want int64
		}{
			{"string", `"100"`, 10000000000},
			{"json number", `100`, 10000000000},
			{"cents", `"12.34"`, 1234000000},
			{"json number with a fraction", `100.5`, 10050000000},
			{"smallest unit", `"0.00000001"`, 1},
		} {
			got := mustUnits(t, tool, args(t, "dollar_amount", tc.raw, "symbol", `"X"`))
			if got != tc.want {
				t.Errorf("%s %s: %d units, want %d", tool, tc.name, got, tc.want)
			}
		}
	}
	// side and type are not read on the dollars path: a sell by dollars and a
	// market buy by dollars are valued at dollar_amount, as they always were.
	for _, extra := range [][]string{
		{"side", `"sell"`}, {"side", `"buy"`, "type", `"market"`}, {"type", `"stop_market"`}, {"side", `5`},
	} {
		a := args(t, append([]string{"dollar_amount", `"20"`}, extra...)...)
		if got := mustUnits(t, "place_equity_order", a); got != 2000000000 {
			t.Errorf("%v: %d units", extra, got)
		}
	}
}

func TestOrderMaxCostLimitBuyIsQuantityTimesLimitPrice(t *testing.T) {
	for _, tc := range []struct {
		name       string
		qty, price string
		want       int64
	}{
		{"1 x 100.00", `"1"`, `"100.00"`, 10000000000},
		{"1 x 100.01", `"1"`, `"100.01"`, 10001000000},
		{"0.5 x 199.99", `"0.5"`, `"199.99"`, 9999500000},
		{"0.00012345 x 67000.12345678 (rounds up)", `"0.00012345"`, `"67000.12345678"`, 827116525},
		{"3 x 33.333", `"3"`, `"33.333"`, 9999900000},
		{"1 x 0.003", `"1"`, `"0.003"`, 300000},
		{"json numbers", `2`, `49.5`, 9900000000},
		{"0.5 x 0.00000001 never zero", `"0.5"`, `"0.00000001"`, 1},
	} {
		got := mustUnits(t, "place_equity_order", dayBuy(t, "quantity", tc.qty, "limit_price", tc.price))
		if got != tc.want {
			t.Errorf("%s: %d units, want %d", tc.name, got, tc.want)
		}
	}
}

func TestOrderMaxCostStopLimitBuy(t *testing.T) {
	// stop_price does not change the cost: the most a stop-limit can cost is
	// still quantity x limit_price.
	for _, stop := range []string{`"1"`, `"99999"`, `"0"`, `"junk"`, `null`, `[1]`} {
		a := dayBuy(t, "type", `"stop_limit"`, "stop_price", stop)
		if got := mustUnits(t, "place_equity_order", a); got != 10000000000 {
			t.Errorf("stop_price %s: %d units", stop, got)
		}
	}
}

func TestOrderMaxCostCaseInsensitiveMatches(t *testing.T) {
	a := dayBuy(t, "side", `"BUY"`, "type", `"Stop_Limit"`, "time_in_force", `"GFD"`, "market_hours", `"Extended_Hours"`)
	if got := mustUnits(t, "place_equity_order", a); got != 10000000000 {
		t.Errorf("%d units", got)
	}
	// The tool name is matched in any ASCII letter case too.
	if got := mustUnits(t, "Place_Equity_Order", dayBuy(t)); got != 10000000000 {
		t.Errorf("%d units", got)
	}
	// A letter that only folds to an ASCII one under Unicode rules is not it.
	for _, over := range [][]string{
		{"side", `"bu\u017f"`}, {"type", `"limit\u212a"`}, {"time_in_force", `"gf\u0064\u00a0"`}, {"side", `"bu\u0443"`},
	} {
		if _, why := OrderMaxCost("place_equity_order", dayBuy(t, over...)); why.Valued() {
			t.Errorf("%v was valued", over)
		}
	}
}

func TestOrderMaxCostRefusesEveryOtherShape(t *testing.T) {
	for _, tc := range []struct {
		name string
		tool string
		a    map[string]json.RawMessage
		why  OrderCostWhy
	}{
		{"market by shares", "place_equity_order", dayBuy(t, "type", `"market"`), WhyNotALimitBuy},
		{"stop_market by shares", "place_equity_order", dayBuy(t, "type", `"stop_market"`), WhyNotALimitBuy},
		{"stop_loss by shares", "place_equity_order", dayBuy(t, "type", `"stop_loss"`), WhyNotALimitBuy},
		{"trailing type", "place_equity_order", dayBuy(t, "type", `"trailing_stop"`), WhyNotALimitBuy},
		{"unknown type spelling", "place_equity_order", dayBuy(t, "type", `"limit "`), WhyNotALimitBuy},
		{"type is not text", "place_equity_order", dayBuy(t, "type", `1`), WhyNotALimitBuy},
		{"type null", "place_equity_order", dayBuy(t, "type", `null`), WhyNotALimitBuy},
		{"no type", "place_equity_order", dayBuy(t, "type", "-"), WhyNotALimitBuy},
		{"market with a stray limit_price", "place_equity_order", dayBuy(t, "type", `"market"`, "limit_price", `"5"`), WhyNotALimitBuy},
		{"sell by shares", "place_equity_order", dayBuy(t, "side", `"sell"`), WhyNotABuy},
		{"no side", "place_equity_order", dayBuy(t, "side", "-"), WhyNotABuy},
		{"side is not text", "place_equity_order", dayBuy(t, "side", `["buy"]`), WhyNotABuy},
		{"good till canceled", "place_equity_order", dayBuy(t, "time_in_force", `"gtc"`), WhyNotADayOrder},
		{"other time in force", "place_equity_order", dayBuy(t, "time_in_force", `"ioc"`), WhyNotADayOrder},
		{"time in force null", "place_equity_order", dayBuy(t, "time_in_force", `null`), WhyNotADayOrder},
		{"time in force not text", "place_equity_order", dayBuy(t, "time_in_force", `1`), WhyNotADayOrder},
		{"no time in force", "place_equity_order", dayBuy(t, "time_in_force", "-"), WhyNoTimeInForce},
		{"overnight session", "place_equity_order", dayBuy(t, "market_hours", `"all_day_hours"`), WhyOvernightSession},
		{"market_hours null", "place_equity_order", dayBuy(t, "market_hours", `null`), WhyOvernightSession},
		{"market_hours unknown", "place_equity_order", dayBuy(t, "market_hours", `"whenever"`), WhyOvernightSession},
		{"no limit price", "place_equity_order", dayBuy(t, "limit_price", "-"), WhyNoLimitPrice},
		{"crypto limit buy by quantity", "place_crypto_order", dayBuy(t, "time_in_force", `"gtc"`), WhySharesToolNotYet},
		{"crypto limit buy by quantity, gfd", "place_crypto_order", dayBuy(t), WhySharesToolNotYet},
		{"crypto stop_limit by quantity, gfd", "place_crypto_order", dayBuy(t, "type", `"stop_limit"`, "stop_price", `"9"`), WhySharesToolNotYet},
		{"crypto market by quantity", "place_crypto_order", dayBuy(t, "type", `"market"`), WhySharesToolNotYet},
		{"both sizes", "place_equity_order", dayBuy(t, "dollar_amount", `"5"`), WhyBothSizes},
		{"both sizes, one null", "place_equity_order", dayBuy(t, "dollar_amount", `null`), WhyBothSizes},
		{"neither size", "place_equity_order", dayBuy(t, "quantity", "-", "limit_price", "-"), WhyNoSize},
		{"quantity null is present", "place_equity_order", dayBuy(t, "quantity", `null`), WhyBadNumber},
		{"dollars with limit_price", "place_equity_order", args(t, "dollar_amount", `"5"`, "limit_price", `"5"`), WhyDollarsWithPrice},
		{"dollars with stop_price", "place_equity_order", args(t, "dollar_amount", `"5"`, "stop_price", `"5"`), WhyDollarsWithPrice},
		{"dollars with a null price", "place_equity_order", args(t, "dollar_amount", `"5"`, "stop_price", `null`), WhyDollarsWithPrice},
		{"dollars, crypto, with a price", "place_crypto_order", args(t, "dollar_amount", `"5"`, "limit_price", `"5"`), WhyDollarsWithPrice},
		{"an option tool", "place_option_order", dayBuy(t), WhyNotOrderTool},
		{"another tool", "get_quote", args(t, "dollar_amount", `"5"`), WhyNotOrderTool},
		{"no arguments at all", "place_equity_order", nil, WhyNoSize},
	} {
		c, why := OrderMaxCost(tc.tool, tc.a)
		if why != tc.why || c.Units != 0 {
			t.Errorf("%s: got (%d, %q), want reason %q", tc.name, c.Units, why, tc.why)
		}
	}
}

// The allow-lists are the decided ones, written in one place each. Widening
// one is a one-line change to its variable in order_cost.go and to this test.
func TestOrderMaxCostAllowListsAreTheDecidedOnes(t *testing.T) {
	for _, tc := range []struct {
		name string
		got  []string
		want string
	}{
		{"tools", orderCostTools[:], "place_equity_order place_crypto_order"},
		{"shares tools", orderCostSharesTools[:], "place_equity_order"},
		{"shares types", orderCostSharesTypes[:], "limit stop_limit"},
		{"shares time_in_force", orderCostSharesTimeInForce[:], "gfd"},
		{"shares market_hours", orderCostSharesMarketHours[:], "regular_hours extended_hours"},
		{"shares sides", orderCostSharesSides[:], "buy"},
	} {
		if g := strings.Join(tc.got, " "); g != tc.want {
			t.Errorf("%s = %q, want %q", tc.name, g, tc.want)
		}
	}
	// And every list entry is the lower-case ASCII form the match compares to.
	for _, l := range [][]string{orderCostTools[:], orderCostSharesTools[:], orderCostSharesTypes[:],
		orderCostSharesTimeInForce[:], orderCostSharesMarketHours[:], orderCostSharesSides[:]} {
		for _, s := range l {
			if s != asciiLower(s) {
				t.Errorf("%q is not lower case", s)
			}
		}
	}
}

func TestOrderMaxCostRefusesBadNumbers(t *testing.T) {
	bad := []string{
		`null`, `true`, `false`, `[1]`, `{}`, `""`, `"1e3"`, `1E2`, `1e2`, `"01"`, `" 1"`, `"1 "`, `"+1"`, `".5"`, `"5."`,
		`"NaN"`, `"Infinity"`, `"0x10"`, `"1_0"`, `"1,5"`, `"١"`, `"-"`, `"--1"`,
		`"` + strings.Repeat("9", 39) + `"`, `"1.` + strings.Repeat("0", 38) + `"`,
	}
	for _, v := range bad {
		if _, why := OrderMaxCost("place_equity_order", args(t, "dollar_amount", v)); why != WhyBadNumber {
			t.Errorf("dollar_amount %s: %q", v, why)
		}
		if _, why := OrderMaxCost("place_equity_order", dayBuy(t, "quantity", v)); why != WhyBadNumber {
			t.Errorf("quantity %s: %q", v, why)
		}
		if _, why := OrderMaxCost("place_equity_order", dayBuy(t, "limit_price", v)); why != WhyBadNumber {
			t.Errorf("limit_price %s: %q", v, why)
		}
	}
	notAbove := []string{`"0"`, `"0.0"`, `"-0"`, `"-1"`, `"-0.00000001"`, `0`, `-5`, `"0.00000000"`}
	for _, v := range notAbove {
		if _, why := OrderMaxCost("place_equity_order", args(t, "dollar_amount", v)); why != WhyNotAboveZero {
			t.Errorf("dollar_amount %s: %q", v, why)
		}
		if _, why := OrderMaxCost("place_equity_order", dayBuy(t, "quantity", v)); why != WhyNotAboveZero {
			t.Errorf("quantity %s: %q", v, why)
		}
		if _, why := OrderMaxCost("place_equity_order", dayBuy(t, "limit_price", v)); why != WhyNotAboveZero {
			t.Errorf("limit_price %s: %q", v, why)
		}
	}
	// A negative quantity cannot cancel a negative price into a cost.
	if _, why := OrderMaxCost("place_equity_order", dayBuy(t, "quantity", `"-2"`, "limit_price", `"-3"`)); why.Valued() {
		t.Error("two negatives were valued")
	}
}

func TestOrderMaxCostRoundsUpNeverDown(t *testing.T) {
	for _, tc := range []struct {
		qty, price string
		want       int64
	}{
		{`"0.00000001"`, `"0.5"`, 1},                // 5e-9 -> 1 unit, never 0
		{`"0.00000001"`, `"0.00000001"`, 1},         // 1e-16 -> 1 unit
		{`"1.00000001"`, `"1.00000001"`, 100000003}, // 1.0000000200000001 -> 1.00000003
		{`"0.1"`, `"0.1"`, 1000000},                 // exactly 0.01
		{`"3"`, `"0.333333333"`, 99999999 + 1},      // 0.999999999 -> 1.00000000
	} {
		got := mustUnits(t, "place_equity_order", dayBuy(t, "quantity", tc.qty, "limit_price", tc.price))
		if got != tc.want {
			t.Errorf("%s x %s = %d units, want %d", tc.qty, tc.price, got, tc.want)
		}
	}
	// Exact dollars with nine fractional digits round up too.
	if got := mustUnits(t, "place_crypto_order", args(t, "dollar_amount", `"0.000000011"`)); got != 2 {
		t.Errorf("0.000000011 = %d units, want 2", got)
	}
}

func TestOrderMaxCostPastPrecisionIsRefused(t *testing.T) {
	const topUnits = 999999999999999999 // 9,999,999,999.99999999
	if got := mustUnits(t, "place_equity_order", args(t, "dollar_amount", `"9999999999.99999999"`)); got != topUnits {
		t.Errorf("top of the range = %d", got)
	}
	for _, v := range []string{`"10000000000"`, `"10000000000.00000001"`, `"99999999999999"`, `"9999999999.999999991"`} {
		if _, why := OrderMaxCost("place_equity_order", args(t, "dollar_amount", v)); why != WhyCostTooLarge {
			t.Errorf("dollar_amount %s: %q", v, why)
		}
	}
	// By shares the product is what counts: 100000 x 100000 = 10^10.
	if _, why := OrderMaxCost("place_equity_order", dayBuy(t, "quantity", `"100000"`, "limit_price", `"100000"`)); why != WhyCostTooLarge {
		t.Errorf("10^10 by shares: %q", why)
	}
	if _, why := OrderMaxCost("place_equity_order", dayBuy(t, "quantity", `"99999.99999999"`, "limit_price", `"100000"`)); !why.Valued() {
		t.Errorf("just under 10^10 by shares: %q", why)
	}
	// Two 38-digit inputs make a 76-digit product: refused, no overflow.
	big38 := `"` + strings.Repeat("9", 38) + `"`
	if _, why := OrderMaxCost("place_equity_order", dayBuy(t, "quantity", big38, "limit_price", big38)); why != WhyCostTooLarge {
		t.Errorf("76-digit product: %q", why)
	}
}

func TestOrderMaxCostCaseTwinsAndDuplicateKeysAreUnreadable(t *testing.T) {
	call := func(argsJSON string) json.RawMessage {
		return json.RawMessage(`{"name":"place_equity_order","arguments":` + argsJSON + `}`)
	}
	good := `"side":"buy","type":"limit","time_in_force":"gfd","quantity":"1","limit_price":"100"`
	if c, why := OrderMaxCostOfCall(call(`{`+good+`}`), "place_equity_order", "place_equity_order"); !why.Valued() || c.Units != 10000000000 {
		t.Fatalf("baseline call: %d %q", c.Units, why)
	}
	for name, doc := range map[string]string{
		"Quantity beside quantity":    `{` + good + `,"Quantity":"2"}`,
		"only Quantity":               `{"side":"buy","type":"limit","time_in_force":"gfd","Quantity":"1","limit_price":"100"}`,
		"Limit_Price twin":            `{` + good + `,"Limit_Price":"1"}`,
		"SIDE twin":                   `{` + good + `,"SIDE":"sell"}`,
		"Type twin":                   `{` + good + `,"Type":"market"}`,
		"time_in_force twin":          `{` + good + `,"TIME_IN_FORCE":"gtc"}`,
		"market_hours twin":           `{` + good + `,"Market_Hours":"all_day_hours"}`,
		"Dollar_Amount twin":          `{` + good + `,"Dollar_Amount":"1"}`,
		"stop_price twin":             `{` + good + `,"Stop_Price":"1"}`,
		"a repeated key":              `{` + good + `,"quantity":"1"}`,
		"a repeated key, other value": `{` + good + `,"limit_price":"1"}`,
		"arguments is a string":       `"quantity"`,
		"arguments is a list":         `[1]`,
	} {
		if _, why := OrderMaxCostOfCall(call(doc), "place_equity_order", "place_equity_order"); why != WhyUnreadable {
			t.Errorf("%s: %q", name, why)
		}
	}
	// A name that does not match the tool is unreadable.
	if _, why := OrderMaxCostOfCall(json.RawMessage(`{"name":"place_crypto_order","arguments":{`+good+`}}`), "place_equity_order", "place_equity_order"); why != WhyUnreadable {
		t.Errorf("name mismatch: %q", why)
	}
	// So are a repeated top-level key and a Name/Arguments twin.
	for _, doc := range []string{
		`{"name":"place_equity_order","name":"place_equity_order","arguments":{` + good + `}}`,
		`{"name":"place_equity_order","Arguments":{},"arguments":{` + good + `}}`,
		`{"name":"place_equity_order","arguments":{` + good + `}`,
		`[]`,
	} {
		if _, why := OrderMaxCostOfCall(json.RawMessage(doc), "place_equity_order", "place_equity_order"); why != WhyUnreadable {
			t.Errorf("%s: %q", doc, why)
		}
	}
	// Missing or null arguments: the call gives no size.
	for _, doc := range []string{`{"name":"place_equity_order"}`, `{"name":"place_equity_order","arguments":null}`, `{"name":"place_equity_order","arguments":{}}`} {
		if _, why := OrderMaxCostOfCall(json.RawMessage(doc), "place_equity_order", "place_equity_order"); why != WhyNoSize {
			t.Errorf("%s: %q", doc, why)
		}
	}
	// A key named "order" is not a field the cost reads.
	withOrder := `{` + good + `,"order":"x","note":"hello"}`
	if c, why := OrderMaxCostOfCall(call(withOrder), "place_equity_order", "place_equity_order"); !why.Valued() || c.Units != 10000000000 {
		t.Errorf("extra arguments changed the value: %d %q", c.Units, why)
	}
}

// A tool listed under a renamed name is read under the name it was sent as and
// valued as the order tool it stands for.
func TestOrderMaxCostOfCallRenamedTool(t *testing.T) {
	p := json.RawMessage(`{"name":"robinhood-2_place_equity_order","arguments":{"dollar_amount":"7"}}`)
	c, why := OrderMaxCostOfCall(p, "robinhood-2_place_equity_order", "place_equity_order")
	if !why.Valued() || c.Units != 700000000 {
		t.Fatalf("%d %q", c.Units, why)
	}
	if _, why := OrderMaxCostOfCall(p, "robinhood-2_place_equity_order", "robinhood-2_place_equity_order"); why != WhyNotOrderTool {
		t.Errorf("the renamed name is not an order tool: %q", why)
	}
	if _, why := OrderMaxCostOfCall(p, "place_equity_order", "place_equity_order"); why != WhyUnreadable {
		t.Errorf("sent name must equal params.name: %q", why)
	}
}

// Every reason is a fixed phrase. A call is full of distinctive values; none
// reaches a reason.
func TestOrderMaxCostReasonsNeverEchoAValue(t *testing.T) {
	const marker = "7731.4242"
	calls := []map[string]json.RawMessage{
		dayBuy(t, "quantity", `"`+marker+`"`, "limit_price", `"`+marker+`1x"`),
		dayBuy(t, "quantity", `"`+marker+`"`, "limit_price", `"-`+marker+`"`),
		dayBuy(t, "quantity", `"`+marker+`"`, "type", `"market-`+marker+`"`),
		dayBuy(t, "quantity", `"`+marker+`"`, "side", `"sell-`+marker+`"`),
		dayBuy(t, "quantity", `"`+marker+`"`, "time_in_force", `"gtc-`+marker+`"`),
		dayBuy(t, "quantity", `"`+marker+`"`, "market_hours", `"all_day_hours-`+marker+`"`),
		dayBuy(t, "quantity", `"`+marker+`"`, "dollar_amount", `"`+marker+`"`),
		args(t, "dollar_amount", `"`+marker+`"`, "limit_price", `"`+marker+`"`),
		args(t, "dollar_amount", `"`+marker+`9999999999999"`),
		args(t, "dollar_amount", `"`+marker+`x"`),
		dayBuy(t, "Quantity", `"`+marker+`"`),
	}
	for i, a := range calls {
		for _, tool := range []string{"place_equity_order", "place_crypto_order"} {
			_, why := OrderMaxCost(tool, a)
			if why.Valued() {
				continue
			}
			if strings.Contains(string(why), "7731") || strings.Contains(string(why), "4242") {
				t.Errorf("call %d: the reason holds a value: %q", i, why)
			}
		}
	}
}

// bigProduct is the exact quantity x limit_price of a valued call, worked out
// here with math/big alone, apart from the function under test.
func bigProduct(t testing.TB, a map[string]json.RawMessage) *big.Rat {
	t.Helper()
	rat := func(name string) *big.Rat {
		d, ok := argumentDecimal(bytes.TrimSpace(a[name]))
		if !ok {
			t.Fatalf("%s is not a decimal", name)
		}
		return d.Rat()
	}
	if _, ok := a["dollar_amount"]; ok {
		return rat("dollar_amount")
	}
	return new(big.Rat).Mul(rat("quantity"), rat("limit_price"))
}

// FuzzOrderMaxCost: it never panics; it never returns 0 units; and for any
// valued call the units are the exact product x 10^8 rounded up: never below
// it (never an undercount) and less than one unit above it.
func FuzzOrderMaxCost(f *testing.F) {
	// The mask says which keys are present: bit 0 dollar_amount, 1 quantity,
	// 2 limit_price, 3 stop_price, 4 side, 5 type, 6 time_in_force, 7 market_hours.
	f.Add(uint16(0x76), true, "buy", "limit", "gfd", "", "", "1", "100.00", "", false)
	f.Add(uint16(0xfe), true, "BUY", "stop_limit", "GFD", "extended_hours", "", "0.00012345", "67000.12345678", "5", false)
	f.Add(uint16(0x76), true, "buy", "limit", "gfd", "", "", "0.5", "0.00000001", "", true)
	f.Add(uint16(0x76), true, "buy", "limit", "gfd", "", "", "1.00000001", "1.00000001", "", false)
	f.Add(uint16(0x76), true, "buy", "limit", "gfd", "", "", "99999.99999999", "100000", "", true)
	f.Add(uint16(0x01), false, "", "", "", "", "9999999999.99999999", "", "", "", true)
	f.Add(uint16(0x01), true, "", "", "", "", "0.000000011", "", "", "", false)
	// Found by the first fuzz run: a raw JSON number with leading white space is read like any other.
	f.Add(uint16(0x01), false, "", "", "0", "", " 1", "", "", "0", true)
	f.Add(uint16(0x01), false, "sell", "market", "", "", "12.5", "", "", "", true)
	f.Add(uint16(0xff), true, "buy", "limit", "gfd", "", "5", "1", "1", "", false)
	f.Add(uint16(0x76), false, "buy", "limit", "gfd", "", "", "1", "1", "", false)
	f.Add(uint16(0xf6), true, "buy", "limit", "gtc", "all_day_hours", "", "1e3", "-1", "x", false)
	f.Add(uint16(0x76), true, "sell", "market", "gfd", "", "", "3", "33.333", "", true)
	f.Fuzz(func(t *testing.T, mask uint16, equity bool, side, typ, tif, hours, dollars, qty, limit, stop string, asNumbers bool) {
		tool := "place_crypto_order"
		if equity {
			tool = "place_equity_order"
		}
		m := map[string]json.RawMessage{}
		put := func(bit uint, name, v string, numeric bool) {
			if mask&(1<<bit) == 0 {
				return
			}
			if numeric && asNumbers && json.Valid([]byte(v)) && v != "" && v[0] != '"' {
				m[name] = json.RawMessage(v)
				return
			}
			b, _ := json.Marshal(v)
			m[name] = b
		}
		put(0, "dollar_amount", dollars, true)
		put(1, "quantity", qty, true)
		put(2, "limit_price", limit, true)
		put(3, "stop_price", stop, true)
		put(4, "side", side, false)
		put(5, "type", typ, false)
		put(6, "time_in_force", tif, false)
		put(7, "market_hours", hours, false)
		c, why := OrderMaxCost(tool, m)
		if !why.Valued() {
			if c.Units != 0 {
				t.Fatalf("refused (%q) with units %d", why, c.Units)
			}
			return
		}
		if c.Units < 1 {
			t.Fatalf("valued at %d units", c.Units)
		}
		if c.Units >= maxOrderCostUnits {
			t.Fatalf("valued past the bound: %d", c.Units)
		}
		exact := new(big.Rat).Mul(bigProduct(t, m), new(big.Rat).SetInt64(UnitsPerOne))
		got := new(big.Rat).SetInt64(c.Units)
		if got.Cmp(exact) < 0 {
			t.Fatalf("undercount: %d units for exact %s", c.Units, exact.FloatString(12))
		}
		if new(big.Rat).Sub(got, exact).Cmp(big.NewRat(1, 1)) >= 0 {
			t.Fatalf("rounded up by a whole unit or more: %d for exact %s", c.Units, exact.FloatString(12))
		}
		// Valued twice, valued the same: the function is pure.
		c2, why2 := OrderMaxCost(tool, m)
		if c2 != c || why2 != why {
			t.Fatalf("not pure")
		}
	})
}

func TestUnitsCeil(t *testing.T) {
	for _, tc := range []struct {
		in   string
		want int64
		ok   bool
	}{
		{"0", 0, true}, {"1", 100000000, true}, {"0.00000001", 1, true}, {"0.000000001", 1, true},
		{"1.000000001", 100000001, true}, {"-1", 0, false}, {"92233720368.54775807", 9223372036854775807, true},
		{"92233720368.54775808", 0, false},
	} {
		r, _ := new(big.Rat).SetString(tc.in)
		got, ok := UnitsCeil(r)
		if got != tc.want || ok != tc.ok {
			t.Errorf("%s: (%d, %v), want (%d, %v)", tc.in, got, ok, tc.want, tc.ok)
		}
	}
}
