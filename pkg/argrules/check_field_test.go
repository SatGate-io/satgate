package argrules

import (
	"encoding/json"
	"fmt"
	"strings"
	"testing"
)

// orderLimitsDoc is a realistic multi-shape rule document: sells are free,
// buys are capped by dollar amount (market) or by a ladder of quantity and
// limit-price pairs (limit orders). Several shapes share conditions, so which
// field "came closest" is a real question.
const orderLimitsDoc = `{
  "v": 1,
  "rules": [
    {
      "tool": "place_equity_order",
      "shapes": [
        {"side": {"one_of": ["sell"]}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["market"]}, "dollar_amount": {"max": "100"}, "quantity": {"absent": true}, "limit_price": {"absent": true}, "stop_price": {"absent": true}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["limit", "stop_limit"]}, "quantity": {"max": "1"}, "limit_price": {"max": "100"}, "dollar_amount": {"absent": true}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["limit", "stop_limit"]}, "quantity": {"max": "2"}, "limit_price": {"max": "50"}, "dollar_amount": {"absent": true}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["limit", "stop_limit"]}, "quantity": {"max": "3"}, "limit_price": {"max": "33.33"}, "dollar_amount": {"absent": true}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["limit", "stop_limit"]}, "quantity": {"max": "4"}, "limit_price": {"max": "25"}, "dollar_amount": {"absent": true}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["limit", "stop_limit"]}, "quantity": {"max": "5"}, "limit_price": {"max": "20"}, "dollar_amount": {"absent": true}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["limit", "stop_limit"]}, "quantity": {"max": "8"}, "limit_price": {"max": "12.5"}, "dollar_amount": {"absent": true}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["limit", "stop_limit"]}, "quantity": {"max": "10"}, "limit_price": {"max": "10"}, "dollar_amount": {"absent": true}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["limit", "stop_limit"]}, "quantity": {"max": "20"}, "limit_price": {"max": "5"}, "dollar_amount": {"absent": true}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["limit", "stop_limit"]}, "quantity": {"max": "50"}, "limit_price": {"max": "2"}, "dollar_amount": {"absent": true}},
        {"side": {"one_of": ["buy"]}, "type": {"one_of": ["limit", "stop_limit"]}, "quantity": {"max": "100"}, "limit_price": {"max": "1"}, "dollar_amount": {"absent": true}}
      ]
    },
    {
      "tool": "place_crypto_order",
      "shapes": [
        {"side": {"one_of": ["sell"]}},
        {"side": {"one_of": ["buy"]}, "dollar_amount": {"max": "100"}, "quantity": {"absent": true}}
      ]
    },
    {
      "tool": "place_option_order",
      "shapes": [
        {"quantity": {"max": "1"}}
      ]
    },
    {
      "tool": "exercise_option",
      "shapes": [
        {"quantity": {"max": "1"}}
      ]
    }
  ]
}
`

// oldRuleCheck is Rule.check as it was before the field choice changed: it
// stops at the first failing condition of each shape. It is kept here only so
// the tests can prove the pass/deny verdict did not change.
func oldRuleCheck(r Rule, args map[string]json.RawMessage) *Denial {
	var best *Denial
	bestHeld := -1
	for _, shape := range r.Shapes {
		held := 0
		var failed string
		for _, cond := range shape.Fields {
			if cond.holds(args) {
				held++
				continue
			}
			failed = cond.Field
			break
		}
		if failed == "" {
			return nil
		}
		if held > bestHeld {
			bestHeld = held
			best = &Denial{Field: failed, Reason: ReasonNoShape}
		}
	}
	if best == nil {
		best = &Denial{Reason: ReasonNoShape}
	}
	return best
}

// oracleField recomputes the expected field from the spec: the first failing
// field of the shape with the most conditions held, earliest shape on a tie.
func oracleField(r Rule, args map[string]json.RawMessage) (string, bool) {
	bestHeld, field := -1, ""
	for _, shape := range r.Shapes {
		var failing []string
		for _, c := range shape.Fields {
			if !c.holds(args) {
				failing = append(failing, c.Field)
			}
		}
		if len(failing) == 0 {
			return "", true
		}
		if held := len(shape.Fields) - len(failing); held > bestHeld {
			bestHeld, field = held, failing[0]
		}
	}
	return field, false
}

func argsOf(t *testing.T, raw string) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(raw), &m); err != nil {
		t.Fatalf("bad test arguments %s: %v", raw, err)
	}
	return m
}

// The four refusals seen with the order-limit document, plus every shape's
// allowed boundary and a spread of malformed calls. Each row pins the verdict
// and the field named.
func TestDenialNamesTheFieldThatBlocked(t *testing.T) {
	rules := rulesFor(t, orderLimitsDoc)
	cases := []struct {
		name  string
		tool  string
		args  string // "" = no arguments key
		field string // "" = the call must pass
	}{
		// The four reported refusals.
		{"crypto market over the dollar cap", "place_crypto_order", `{"side":"buy","type":"market","dollar_amount":"250"}`, "dollar_amount"},
		{"equity market over the dollar cap", "place_equity_order", `{"side":"buy","type":"market","dollar_amount":"500"}`, "dollar_amount"},
		{"equity limit between two ladder rungs", "place_equity_order", `{"side":"buy","type":"limit","quantity":"10","limit_price":"12"}`, "quantity"},
		{"equity limit over the top rung", "place_equity_order", `{"side":"buy","type":"limit","quantity":"1","limit_price":"150"}`, "limit_price"},

		// Every allowed example in the document passes.
		{"equity sell", "place_equity_order", `{"side":"sell","type":"market","quantity":"1000"}`, ""},
		{"equity sell with only side", "place_equity_order", `{"side":"sell"}`, ""},
		{"equity market at the cap", "place_equity_order", `{"side":"buy","type":"market","dollar_amount":"100"}`, ""},
		{"equity 1 at 100", "place_equity_order", `{"side":"buy","type":"limit","quantity":"1","limit_price":"100"}`, ""},
		{"equity 2 at 50", "place_equity_order", `{"side":"buy","type":"limit","quantity":"2","limit_price":"50"}`, ""},
		{"equity 3 at 33.33", "place_equity_order", `{"side":"buy","type":"limit","quantity":"3","limit_price":"33.33"}`, ""},
		{"equity 4 at 25", "place_equity_order", `{"side":"buy","type":"stop_limit","quantity":"4","limit_price":"25"}`, ""},
		{"equity 5 at 20", "place_equity_order", `{"side":"buy","type":"limit","quantity":"5","limit_price":"20"}`, ""},
		{"equity 8 at 12.5", "place_equity_order", `{"side":"buy","type":"limit","quantity":"8","limit_price":"12.5"}`, ""},
		{"equity 10 at 10", "place_equity_order", `{"side":"buy","type":"limit","quantity":"10","limit_price":"10"}`, ""},
		{"equity 20 at 5", "place_equity_order", `{"side":"buy","type":"limit","quantity":"20","limit_price":"5"}`, ""},
		{"equity 50 at 2", "place_equity_order", `{"side":"buy","type":"limit","quantity":"50","limit_price":"2"}`, ""},
		{"equity 100 at 1", "place_equity_order", `{"side":"buy","type":"limit","quantity":"100","limit_price":"1"}`, ""},
		{"equity numbers not strings", "place_equity_order", `{"side":"buy","type":"limit","quantity":8,"limit_price":12.5}`, ""},
		{"crypto sell", "place_crypto_order", `{"side":"sell","type":"market","quantity":"5"}`, ""},
		{"crypto buy at the cap", "place_crypto_order", `{"side":"buy","dollar_amount":"100"}`, ""},
		{"option at the cap", "place_option_order", `{"quantity":"1"}`, ""},
		{"exercise at the cap", "exercise_option", `{"quantity":1}`, ""},

		// Other refusals and what they name.
		{"crypto quantity buy: quantity is present", "place_crypto_order", `{"side":"buy","dollar_amount":"50","quantity":"1"}`, "quantity"},
		{"crypto buy without amount", "place_crypto_order", `{"side":"buy"}`, "dollar_amount"},
		{"crypto wrong side", "place_crypto_order", `{"side":"hold","dollar_amount":"50"}`, "side"},
		{"crypto no arguments", "place_crypto_order", ``, "dollar_amount"},
		{"crypto null arguments", "place_crypto_order", `null`, "dollar_amount"},
		{"crypto amount wrong type", "place_crypto_order", `{"side":"buy","dollar_amount":true}`, "dollar_amount"},
		{"crypto amount is an object", "place_crypto_order", `{"side":"buy","dollar_amount":{"v":"5"}}`, "dollar_amount"},
		{"crypto side wrong type", "place_crypto_order", `{"side":1,"dollar_amount":"5"}`, "side"},
		{"crypto amount exponent form", "place_crypto_order", `{"side":"buy","dollar_amount":1e1}`, "dollar_amount"},
		{"equity market stop_price present", "place_equity_order", `{"side":"buy","type":"market","dollar_amount":"50","stop_price":"1"}`, "stop_price"},
		{"equity market with limit_price", "place_equity_order", `{"side":"buy","type":"market","dollar_amount":"50","limit_price":"1"}`, "limit_price"},
		{"equity limit with dollar amount", "place_equity_order", `{"side":"buy","type":"limit","quantity":"1","limit_price":"1","dollar_amount":"5"}`, "dollar_amount"},
		{"equity wrong type", "place_equity_order", `{"side":"buy","type":"trailing","quantity":"1","limit_price":"1"}`, "type"},
		{"equity empty arguments", "place_equity_order", `{}`, "dollar_amount"},
		{"option over the cap", "place_option_order", `{"quantity":"2"}`, "quantity"},
		{"option quantity missing", "place_option_order", `{}`, "quantity"},
		{"exercise over the cap", "exercise_option", `{"quantity":"3"}`, "quantity"},
		{"extra unlimited field does not matter", "place_option_order", `{"quantity":"1","symbol":"X","anything":[1,2]}`, ""},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := Check(rules, tc.tool, call(tc.tool, tc.args))
			if tc.field == "" {
				if d != nil {
					t.Fatalf("want pass, got %+v", d)
				}
				return
			}
			if d == nil {
				t.Fatal("want a denial, call passed")
			}
			if d.Field != tc.field {
				t.Fatalf("denial names %q, want %q", d.Field, tc.field)
			}
			if d.Reason != ReasonNoShape {
				t.Fatalf("reason = %q", d.Reason)
			}
			// Same verdict and same denial-or-pass as the old code.
			for _, r := range rules {
				if r.Tool == tc.tool {
					var args map[string]json.RawMessage
					if tc.args != "" && tc.args != "null" {
						args = argsOf(t, tc.args)
					}
					if old := oldRuleCheck(r, args); old == nil {
						t.Fatal("old code passed a call the new code refuses")
					}
				}
			}
		})
	}
}

// Tie: both shapes 10@10 and 8@12.5 hold the same number of conditions for
// 10@12, and quantity (the earlier shape's failing field) or limit_price would
// both be acceptable; the earliest shape wins, so the answer is stable.
func TestDenialTieKeepsEarliestShape(t *testing.T) {
	rules := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[
		{"a":{"max":"1"},"b":{"max":"1"}},
		{"a":{"max":"5"},"b":{"max":"5"}}
	]}]}`)
	// a=3, b=3: shape 1 holds 0 (a, b both fail), shape 2 holds 2 -> passes.
	if d := Check(rules, "t", call("t", `{"a":"3","b":"3"}`)); d != nil {
		t.Fatalf("want pass, got %+v", d)
	}
	// a=9, b=9: both shapes hold 0; earliest shape wins and names its first field, a.
	if d := Check(rules, "t", call("t", `{"a":"9","b":"9"}`)); d == nil || d.Field != "a" {
		t.Fatalf("tie: got %+v, want field a", d)
	}
	// a=0, b=9: shape 1 holds a (1), shape 2 holds a (1): tie -> shape 1, b.
	if d := Check(rules, "t", call("t", `{"a":"0","b":"9"}`)); d == nil || d.Field != "b" {
		t.Fatalf("tie: got %+v, want field b", d)
	}
	// A later shape that holds more beats an earlier one.
	rules = rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[
		{"a":{"max":"1"},"b":{"max":"1"},"c":{"max":"1"}},
		{"a":{"max":"9"},"b":{"max":"9"},"c":{"max":"1"}}
	]}]}`)
	if d := Check(rules, "t", call("t", `{"a":"5","b":"5","c":"5"}`)); d == nil || d.Field != "c" {
		t.Fatalf("later closer shape: got %+v, want field c", d)
	}
}

func TestRuleWithNoShapesKeepsNoShapeDenialWithoutField(t *testing.T) {
	d := Rule{Tool: "t"}.check(map[string]json.RawMessage{})
	if d == nil || d.Reason != ReasonNoShape || d.Field != "" {
		t.Fatalf("got %+v", d)
	}
}

// Pass/deny must not change. Over every combination of values below for the
// fields of the order-limit document, the new check agrees with the old one on
// the verdict, and every denial names the field the spec says.
func TestVerdictUnchangedOverManyArgumentSets(t *testing.T) {
	rules := rulesFor(t, orderLimitsDoc)
	// "" means the key is absent.
	values := map[string][]string{
		"side":          {"", `"buy"`, `"sell"`, `"hold"`, `1`},
		"type":          {"", `"market"`, `"limit"`, `"stop_limit"`, `7`},
		"dollar_amount": {"", `"50"`, `"100"`, `"250"`, `100.5`, `true`},
		"quantity":      {"", `"1"`, `"8"`, `"10"`, `"11"`, `3`, `"x"`},
		"limit_price":   {"", `"1"`, `"12"`, `"12.5"`, `"100"`, `"150"`},
		"stop_price":    {"", `"1"`},
		"symbol":        {"", `"X"`}, // not limited by any shape
	}
	fields := []string{"side", "type", "dollar_amount", "quantity", "limit_price", "stop_price", "symbol"}
	var build func(i int, parts []string, fn func(raw string))
	build = func(i int, parts []string, fn func(raw string)) {
		if i == len(fields) {
			fn("{" + strings.Join(parts, ",") + "}")
			return
		}
		for _, v := range values[fields[i]] {
			next := parts
			if v != "" {
				next = append(append([]string{}, parts...), fmt.Sprintf("%q:%s", fields[i], v))
			}
			build(i+1, next, fn)
		}
	}
	var total, denied int
	for _, tool := range []string{"place_equity_order", "place_crypto_order", "place_option_order", "exercise_option"} {
		var rule Rule
		for _, r := range rules {
			if r.Tool == tool {
				rule = r
			}
		}
		build(0, nil, func(raw string) {
			args := argsOf(t, raw)
			total++
			got := Check(rules, tool, call(tool, raw))
			old := oldRuleCheck(rule, args)
			if (got == nil) != (old == nil) {
				t.Fatalf("%s %s: verdict changed: new=%+v old=%+v", tool, raw, got, old)
			}
			if got == nil {
				return
			}
			denied++
			want, passed := oracleField(rule, args)
			if passed {
				t.Fatalf("%s %s: oracle says pass but call was denied", tool, raw)
			}
			if got.Field != want || got.Field == "" {
				t.Fatalf("%s %s: names %q, want %q", tool, raw, got.Field, want)
			}
			if got.Reason != ReasonNoShape {
				t.Fatalf("%s %s: reason %q", tool, raw, got.Reason)
			}
		})
	}
	if total < 50000 || denied == 0 || denied == total {
		t.Fatalf("table too small or one-sided: total=%d denied=%d", total, denied)
	}
	t.Logf("compared %d argument sets (%d denied)", total, denied)
}

// Malformed and non-object arguments: verdict equal to the old code.
func TestVerdictUnchangedForOddArguments(t *testing.T) {
	rules := rulesFor(t, orderLimitsDoc)
	odd := []string{
		``, `null`, `{}`, `[]`, `"buy"`, `7`, `true`,
		`{"side":"buy","side":"sell"}`,
		`{"SIDE":"buy","dollar_amount":"5"}`,
		`{"side":"buy","dollar_amount":"5","extra":{"deep":[1,2,{"x":null}]}}`,
	}
	for _, tool := range []string{"place_equity_order", "place_crypto_order"} {
		for _, raw := range odd {
			got := Check(rules, tool, call(tool, raw))
			// Old verdict: run the old per-rule check on whatever the arguments decode to.
			var args map[string]json.RawMessage
			_ = json.Unmarshal([]byte(raw), &args)
			var oldDenied bool
			for _, r := range rules {
				if r.Tool == tool && oldRuleCheck(r, args) != nil {
					oldDenied = true
				}
			}
			// Unreadable shapes (duplicate keys, case variants, non-objects) are
			// refused before rule evaluation, in both old and new code.
			if got == nil && oldDenied {
				t.Fatalf("%s %q: new passes, old denied", tool, raw)
			}
			if got != nil && got.Reason == ReasonNoShape && !oldDenied {
				t.Fatalf("%s %q: new denies by shape, old passed", tool, raw)
			}
		}
	}
}

func TestDenialNeverEchoesArgumentValues(t *testing.T) {
	rules := rulesFor(t, orderLimitsDoc)
	secrets := []string{"SECRET-ACCT-123456", "9999.99"}
	args := `{"side":"buy","type":"SECRET-ACCT-123456","dollar_amount":"9999.99","symbol":"SECRET-ACCT-123456"}`
	for _, tool := range []string{"place_equity_order", "place_crypto_order"} {
		d := Check(rules, tool, call(tool, args))
		if d == nil {
			t.Fatalf("%s: want denial", tool)
		}
		blob, _ := json.Marshal(d)
		for _, s := range secrets {
			if strings.Contains(d.Error(), s) || strings.Contains(string(blob), s) || strings.Contains(d.Reason, s) || strings.Contains(d.Field, s) {
				t.Fatalf("%s: denial echoed %q", tool, s)
			}
		}
	}
}
