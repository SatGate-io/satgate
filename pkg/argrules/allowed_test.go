package argrules

import (
	"encoding/json"
	"strings"
	"testing"
)

// Each condition kind has one plain description, built from the rule only.
func TestConditionAllowedText(t *testing.T) {
	cases := []struct {
		name, cond, want string
	}{
		{"max", `{"max":"100"}`, "amt must be at most 100"},
		{"max decimal", `{"max":"33.33"}`, "amt must be at most 33.33"},
		{"min", `{"min":"5"}`, "amt must be at least 5"},
		{"min negative", `{"min":"-1.5"}`, "amt must be at least -1.5"},
		{"range", `{"min":"1","max":"9"}`, "amt must be between 1 and 9"},
		{"one_of one", `{"one_of":["buy"]}`, "amt must be one of: buy"},
		{"one_of many", `{"one_of":["buy","sell","hold"]}`, "amt must be one of: buy, sell, hold"},
		{"one_of case insensitive", `{"one_of":["GET"],"ascii_case_insensitive":true}`, "amt must be one of: GET (any letter case)"},
		{"one_of awkward value quoted", `{"one_of":["a b","c,d",""]}`, `amt must be one of: "a b", "c,d", ""`},
		{"absent", `{"absent":true}`, "amt must not be set"},
		{"url host only", `{"url":{"hosts":["api.example.com"]}}`, "amt must be an https address on api.example.com"},
		{"url hosts", `{"url":{"hosts":["a.example.com","b.example.com"]}}`, "amt must be an https address on a.example.com or b.example.com"},
		{"url port", `{"url":{"hosts":["a.example.com"],"ports":[8443,9443]}}`, "amt must be an https address on a.example.com (port 8443 or 9443)"},
		{"url prefixes", `{"url":{"hosts":["www.wixapis.com"],"path_prefixes":["/stores/v1","/contacts/v4"]}}`, "amt must be an https address on www.wixapis.com with a path starting /stores/v1 or /contacts/v4"},
		{"url suffixes", `{"url":{"hosts":["www.wixapis.com"],"path_suffixes":["/query","/search"]}}`, "amt must be an https address on www.wixapis.com with a path ending /query or /search"},
		{"url prefix and suffix", `{"url":{"hosts":["h.example.com"],"path_prefixes":["/v1"],"path_suffixes":["/query"]}}`, "amt must be an https address on h.example.com with a path starting /v1 with a path ending /query"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rules := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"amt":`+tc.cond+`}]}]}`)
			got := rules[0].Shapes[0].Fields[0].AllowedText()
			if got != tc.want {
				t.Fatalf("AllowedText = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAllowedTextLongOneOfIsCapped(t *testing.T) {
	vals := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		vals = append(vals, "v"+string(rune('a'+i)))
	}
	c := Condition{Field: "f", OneOf: vals}
	got := c.AllowedText()
	if !strings.HasSuffix(got, ", and 4 more") || strings.Contains(got, "vu") {
		t.Fatalf("long list not capped: %q", got)
	}
}

// Check carries the text of the first failing condition of the closest shape.
func TestDenialAllowedFollowsClosestShape(t *testing.T) {
	rules := rulesFor(t, orderLimitsDoc)
	cases := []struct {
		name, tool, args, field, allowed string
	}{
		{"crypto over cap", "place_crypto_order", `{"side":"buy","dollar_amount":"250"}`, "dollar_amount", "dollar_amount must be at most 100"},
		{"crypto wrong side", "place_crypto_order", `{"side":"hold","dollar_amount":"50"}`, "side", "side must be one of: buy"},
		{"crypto quantity set", "place_crypto_order", `{"side":"buy","dollar_amount":"50","quantity":"1"}`, "quantity", "quantity must not be set"},
		{"option over cap", "place_option_order", `{"quantity":"2"}`, "quantity", "quantity must be at most 1"},
		{"equity over top rung", "place_equity_order", `{"side":"buy","type":"limit","quantity":"1","limit_price":"150"}`, "limit_price", "limit_price must be at most 100"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			d := Check(rules, tc.tool, call(tc.tool, tc.args))
			if d == nil {
				t.Fatal("want a denial")
			}
			if d.Field != tc.field || d.Allowed != tc.allowed {
				t.Fatalf("field %q allowed %q, want %q / %q", d.Field, d.Allowed, tc.field, tc.allowed)
			}
		})
	}
}

// Only a shape refusal that names a field carries allowed text.
func TestOtherDenialsCarryNoAllowedText(t *testing.T) {
	rules := rulesFor(t, orderLimitsDoc)
	for _, d := range []*Denial{
		Check(rules, "place_crypto_order", json.RawMessage(`{"name":"place_crypto_order","name":"x"}`)),
		Check(rules, "place_crypto_order", json.RawMessage(`[]`)),
		Check(rules, "place_crypto_order", call("place_crypto_order", `{"side":"buy","side":"sell"}`)),
		Check([]Rule{{Tool: "ExecuteWixAPI", Shapes: []Shape{{Fields: []Condition{{Field: "a", Absent: true}}}}}}, "x", call("x", `{}`)),
		Rule{Tool: "t"}.check(map[string]json.RawMessage{}),
	} {
		if d == nil || d.Allowed != "" {
			t.Fatalf("got %+v, want a denial with no allowed text", d)
		}
	}
}

// The text is a function of the rule alone: the same rule gives the same text
// whatever the call held, and no call value shows up in it.
func TestAllowedTextNeverHoldsArgumentValues(t *testing.T) {
	rules := rulesFor(t, orderLimitsDoc)
	markers := []string{"MARKER-ACCT-7731", "987654.321", "evil.example"}
	var texts []string
	for _, m := range markers {
		args := `{"side":"buy","type":"` + m + `","dollar_amount":"` + m + `","quantity":"` + m + `","symbol":"` + m + `"}`
		for _, tool := range []string{"place_equity_order", "place_crypto_order"} {
			d := Check(rules, tool, call(tool, args))
			if d == nil {
				t.Fatalf("%s: want a denial", tool)
			}
			for _, other := range markers {
				if strings.Contains(d.Allowed, other) || strings.Contains(d.Error(), other) {
					t.Fatalf("%s: denial echoed %q: %+v", tool, other, d)
				}
			}
			texts = append(texts, tool+"|"+d.Field+"|"+d.Allowed)
		}
	}
	for i := 2; i < len(texts); i++ {
		if texts[i] != texts[i-2] {
			t.Fatalf("text depends on the call: %q vs %q", texts[i], texts[i-2])
		}
	}
}
