package argrules

import (
	"encoding/json"
	"strconv"
	"strings"
	"testing"
)

// Each condition kind has one plain description, built from the rule only.
func TestConditionAgentText(t *testing.T) {
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
		{"one_of awkward values hidden", `{"one_of":["a b","c,d",""]}`, "amt must be one of the 3 values this token allows"},
		{"absent", `{"absent":true}`, "amt must not be set"},
		{"url host only", `{"url":{"hosts":["api.example.com"]}}`, "amt must be an https address on api.example.com"},
		{"url hosts", `{"url":{"hosts":["a.example.com","b.example.com"]}}`, "amt must be an https address on a.example.com or b.example.com"},
		{"url port", `{"url":{"hosts":["a.example.com"],"ports":[8443,9443]}}`, "amt must be an https address on a.example.com (port 8443 or 9443)"},
		{"url prefixes hidden", `{"url":{"hosts":["www.wixapis.com"],"path_prefixes":["/stores/v1","/contacts/v4"]}}`, "amt must be an https address on www.wixapis.com with a path this token allows"},
		{"url suffixes hidden", `{"url":{"hosts":["www.wixapis.com"],"path_suffixes":["/query","/search"]}}`, "amt must be an https address on www.wixapis.com with a path this token allows"},
		{"url prefix and suffix hidden", `{"url":{"hosts":["h.example.com"],"path_prefixes":["/v1"],"path_suffixes":["/query"]}}`, "amt must be an https address on h.example.com with a path this token allows"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			rules := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"amt":`+tc.cond+`}]}]}`)
			got := rules[0].Shapes[0].Fields[0].AgentText()
			if got != tc.want {
				t.Fatalf("AgentText = %q, want %q", got, tc.want)
			}
		})
	}
}

func TestAgentTextLongOneOfIsCapped(t *testing.T) {
	vals := make([]string, 0, 20)
	for i := 0; i < 20; i++ {
		vals = append(vals, "v"+string(rune('a'+i)))
	}
	c := Condition{Field: "f", OneOf: vals}
	got := c.AgentText()
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
func TestOtherDenialsCarryNoAgentText(t *testing.T) {
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
func TestAgentTextNeverHoldsArgumentValues(t *testing.T) {
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

// Each one_of branch of the agent-safe policy: plain values are shown; a long
// value, a value with a slash, an @ or a space, or a value that starts like a
// credential hides the whole list and gives only the count.
func TestAgentTextOneOfPolicy(t *testing.T) {
	hidden := func(n int) string { return "f must be one of the " + strconv.Itoa(n) + " values this token allows" }
	cases := []struct {
		name string
		vals []string
		want string
	}{
		{"plain shown", []string{"buy", "sell", "GET", "POST", "BTC-USD", "12345678", "a.b_c:d-e"}, "f must be one of: buy, sell, GET, POST, BTC-USD, 12345678, a.b_c:d-e"},
		{"exactly 24 shown", []string{strings.Repeat("a", 24)}, "f must be one of: " + strings.Repeat("a", 24)},
		{"25 chars hidden", []string{"buy", strings.Repeat("a", 25)}, hidden(2)},
		{"slash hidden", []string{"buy", "a/b"}, hidden(2)},
		{"at sign hidden", []string{"me@example.com"}, hidden(1)},
		{"space hidden", []string{"a b"}, hidden(1)},
		{"comma hidden", []string{"a,b"}, hidden(1)},
		{"empty hidden", []string{""}, hidden(1)},
		{"asia region is not a credential", []string{"asia", "Asia"}, "f must be one of: asia, Asia"},
	}
	for _, p := range []string{"sk-", "sk_", "pk_", "rk_", "ghp_", "gho_", "github_pat_", "xox", "eyJ", "AKIA", "ASIA", "AIza", "glpat-", "shpat_", "whsec_"} {
		cases = append(cases, struct {
			name string
			vals []string
			want string
		}{"credential prefix " + p, []string{"buy", p + "abc123"}, hidden(2)})
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			c := Condition{Field: "f", OneOf: tc.vals}
			got := c.AgentText()
			if got != tc.want {
				t.Fatalf("AgentText = %q, want %q", got, tc.want)
			}
			if got != tc.want || (strings.HasPrefix(tc.want, "f must be one of the") && strings.Contains(got, tc.vals[len(tc.vals)-1]) && tc.vals[len(tc.vals)-1] != "") {
				t.Fatalf("hidden list leaked a value: %q", got)
			}
		})
	}
}

func TestAgentTextOneOfLongPlainListKeepsTheCap(t *testing.T) {
	vals := make([]string, 20)
	for i := range vals {
		vals[i] = "v" + strconv.Itoa(i)
	}
	got := Condition{Field: "f", OneOf: vals}.AgentText()
	if !strings.HasSuffix(got, ", and 4 more") || strings.Contains(got, "v19") {
		t.Fatalf("not capped: %q", got)
	}
}

const credentialPathMarker = "REVIEW-FAKE-WEBHOOK-CREDENTIAL-729a"

// Astra's scenario: a URL rule whose path prefix is a webhook credential. The
// agent-facing text names the host and never the path; the owner-facing
// Describe still shows it.
func TestIndependentCredentialURLRuleDisclosure(t *testing.T) {
	doc := `{"v":1,"rules":[{"tool":"fetch","shapes":[{"url":{"url":{"hosts":["hooks.example.com"],"path_prefixes":["/services/` + credentialPathMarker + `"],"path_suffixes":["/` + credentialPathMarker + `-end"]}}}]}]}`
	rules := rulesFor(t, doc)
	d := Check(rules, "fetch", call("fetch", `{"url":"https://hooks.example.com/not-permitted"}`))
	if d == nil {
		t.Fatal("want a denial")
	}
	want := "url must be an https address on hooks.example.com with a path this token allows"
	if d.Allowed != want {
		t.Fatalf("Allowed = %q, want %q", d.Allowed, want)
	}
	for name, text := range map[string]string{"Allowed": d.Allowed, "Error": d.Error(), "AgentLines": strings.Join(AgentLines(rules), "\n")} {
		if strings.Contains(text, credentialPathMarker) || strings.Contains(text, "/services") {
			t.Fatalf("%s discloses the path: %q", name, text)
		}
	}
	if lines := AgentLines(rules); len(lines) != 1 || lines[0] != "fetch: allowed only when url is an https address on hooks.example.com with a path this token allows" {
		t.Fatalf("AgentLines = %q", lines)
	}
	if owner := strings.Join(Describe(rules), "\n"); !strings.Contains(owner, credentialPathMarker) {
		t.Fatalf("owner-facing Describe lost the path: %q", owner)
	}
}

// AgentLines equals Describe when the rules hold only plain values.
func TestAgentLinesEqualDescribeForPlainRules(t *testing.T) {
	rules := rulesFor(t, orderLimitsDoc)
	a, o := AgentLines(rules), Describe(rules)
	if len(a) == 0 || strings.Join(a, "\n") != strings.Join(o, "\n") {
		t.Fatalf("agent %q != owner %q", a, o)
	}
}

func TestAgentLinesHideNonPlainOneOf(t *testing.T) {
	rules := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"k":{"one_of":["sk-live-abcdef","buy"]},"m":{"one_of":["buy","sell"],"ascii_case_insensitive":true}}]}]}`)
	lines := AgentLines(rules)
	got := strings.Join(lines, "\n")
	if strings.Contains(got, "sk-live") || !strings.Contains(got, "k is one of the 2 values this token allows") || !strings.Contains(got, "m is buy or sell (any letter case)") {
		t.Fatalf("lines = %q", lines)
	}
	if !strings.Contains(strings.Join(Describe(rules), "\n"), "sk-live-abcdef") {
		t.Fatal("owner-facing Describe must keep the value")
	}
}
