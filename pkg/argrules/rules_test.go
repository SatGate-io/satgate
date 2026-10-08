package argrules

import (
	"encoding/json"
	"strings"
	"testing"
)

func mustDec(t *testing.T, s string) *Decimal {
	t.Helper()
	d, err := ParseDecimal(s)
	if err != nil {
		t.Fatalf("ParseDecimal(%q): %v", s, err)
	}
	return d
}

func TestDecimalGrammar(t *testing.T) {
	good := []string{"0", "1", "-1", "0.5", "-0.5", "1.0", "10", "123456789012345678901234567890123456.78"[:38], "0.0000000000000000000000000000000000001"}
	for _, s := range good {
		if _, err := ParseDecimal(s); err != nil {
			t.Errorf("ParseDecimal(%q) refused: %v", s, err)
		}
	}
	bad := []string{
		"", " ", " 1", "1 ", "+1", "1e3", "1E3", "0x10", "NaN", "Inf", "-Inf", "+Inf", "infinity",
		".5", "5.", "-", "-.5", "01", "00", "1_000", "1,5", "--1", "1.2.3", "\t1", "1\n",
		strings.Repeat("1", 39), "0." + strings.Repeat("1", 38), strings.Repeat("9", 20) + "." + strings.Repeat("9", 19),
		"١", "1\u00a0",
	}
	for _, s := range bad {
		if _, err := ParseDecimal(s); err == nil {
			t.Errorf("ParseDecimal(%q) accepted, want refusal", s)
		}
	}
	// Exactly 38 digits is the most.
	if _, err := ParseDecimal(strings.Repeat("9", 38)); err != nil {
		t.Errorf("38 digits refused: %v", err)
	}
}

func TestDecimalCompareIsExact(t *testing.T) {
	// float64 cannot tell these apart.
	a := mustDec(t, "12345678901234567890.000000000000000001")
	b := mustDec(t, "12345678901234567890.000000000000000002")
	if a.Cmp(b) >= 0 {
		t.Fatal("exact compare lost the last digit")
	}
	if mustDec(t, "0.1").Cmp(mustDec(t, "0.10")) != 0 {
		t.Fatal("0.1 and 0.10 must be equal")
	}
	if mustDec(t, "-5").Cmp(mustDec(t, "1")) >= 0 {
		t.Fatal("negative below positive")
	}
}

func rulesFor(t *testing.T, doc string) []Rule {
	t.Helper()
	rules, err := ParseJSON([]byte(doc))
	if err != nil {
		t.Fatalf("ParseJSON: %v\n%s", err, doc)
	}
	return rules
}

func call(tool, args string) json.RawMessage {
	if args == "" {
		return json.RawMessage(`{"name":"` + tool + `"}`)
	}
	return json.RawMessage(`{"name":"` + tool + `","arguments":` + args + `}`)
}

// The decimal table, through a real max/min rule and Check.
func TestMaxMinDecimalTable(t *testing.T) {
	maxRule := rulesFor(t, `{"v":1,"rules":[{"tool":"order","shapes":[{"quantity":{"max":"10"}}]}]}`)
	minRule := rulesFor(t, `{"v":1,"rules":[{"tool":"order","shapes":[{"quantity":{"min":"0"}}]}]}`)
	big39 := strings.Repeat("1", 39)
	cases := []struct {
		arg     string // raw JSON of the quantity value
		maxPass bool
		minPass bool
	}{
		{`"1"`, true, true},
		{`"1.0"`, true, true},
		{`"0.5"`, true, true},
		{`"10"`, true, true},
		{`"10.0"`, true, true},
		{`"10.000000000000000000000000000001"`, false, true},
		{`"11"`, false, true},
		{`"1e3"`, false, false},
		{`"+1"`, false, false},
		{`" 1"`, false, false},
		{`"1 "`, false, false},
		{`""`, false, false},
		{`"0x10"`, false, false},
		{`"NaN"`, false, false},
		{`"Infinity"`, false, false},
		{`1`, true, true},
		{`1.5`, true, true},
		{`10`, true, true},
		{`10.5`, false, true},
		{`1e3`, false, false},
		{`1E3`, false, false},
		{`1e-3`, false, false},
		{`-1`, true, false},
		{`"-1"`, true, false},
		{`"-0.0001"`, true, false},
		{`-0`, true, true}, // exactly zero
		{`"` + big39 + `"`, false, false},
		{big39, false, false},
		{`"0.` + strings.Repeat("0", 36) + `1"`, true, true},
		{`null`, false, false},
		{`true`, false, false},
		{`[1]`, false, false},
		{`{"a":1}`, false, false},
	}
	for _, c := range cases {
		args := `{"quantity":` + c.arg + `}`
		got := Check(maxRule, "order", call("order", args)) == nil
		if got != c.maxPass {
			t.Errorf("max 10, quantity %s: pass=%v, want %v", c.arg, got, c.maxPass)
		}
		got = Check(minRule, "order", call("order", args)) == nil
		if got != c.minPass {
			t.Errorf("min 0, quantity %s: pass=%v, want %v", c.arg, got, c.minPass)
		}
	}
	// Missing field fails both.
	if Check(maxRule, "order", call("order", `{}`)) == nil {
		t.Error("missing quantity passed a max rule")
	}
	if Check(maxRule, "order", call("order", "")) == nil {
		t.Error("missing arguments passed a max rule")
	}
}

func TestMaxIsNotFloat64(t *testing.T) {
	// 9007199254740993 is not representable as float64 (rounds to ...992).
	rules := rulesFor(t, `{"v":1,"rules":[{"tool":"order","shapes":[{"n":{"max":"9007199254740992"}}]}]}`)
	if Check(rules, "order", call("order", `{"n":"9007199254740993"}`)) == nil {
		t.Fatal("9007199254740993 passed max 9007199254740992: compared as float64")
	}
	if Check(rules, "order", call("order", `{"n":9007199254740993}`)) == nil {
		t.Fatal("JSON number 9007199254740993 passed max 9007199254740992: compared as float64")
	}
	if Check(rules, "order", call("order", `{"n":"9007199254740992"}`)) != nil {
		t.Fatal("the limit itself must pass")
	}
}

func TestOneOf(t *testing.T) {
	exact := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"method":{"one_of":["GET","HEAD"]}}]}]}`)
	fold := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"method":{"one_of":["GET"],"ascii_case_insensitive":true}}]}]}`)
	cases := []struct {
		arg         string
		exact, fold bool
	}{
		{`"GET"`, true, true},
		{`"HEAD"`, true, false},
		{`"get"`, false, true},
		{`"Get"`, false, true},
		{`"GET "`, false, false},
		{`" GET"`, false, false},
		{`"POST"`, false, false},
		{`""`, false, false},
		{`"GE\u0054"`, true, true}, // JSON escape of T: the value is GET
		{`"GEТ"`, false, false},    // Cyrillic Т
		{`"gEt"`, false, true},
		{`"\u212a"`, false, false}, // Kelvin sign
		{`"ſ"`, false, false},      // long s
		{`"İ"`, false, false},
		{`"GET\u0000"`, false, false},
		{`null`, false, false},
		{`1`, false, false},
		{`["GET"]`, false, false},
		{`true`, false, false},
	}
	for _, c := range cases {
		args := `{"method":` + c.arg + `}`
		if got := Check(exact, "t", call("t", args)) == nil; got != c.exact {
			t.Errorf("exact, method %s: pass=%v want %v", c.arg, got, c.exact)
		}
		if got := Check(fold, "t", call("t", args)) == nil; got != c.fold {
			t.Errorf("fold, method %s: pass=%v want %v", c.arg, got, c.fold)
		}
	}
	// Folding never lets a non-ASCII byte through, even when the rule value
	// folds to the same bytes under Unicode rules.
	k := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"m":{"one_of":["k"],"ascii_case_insensitive":true}}]}]}`)
	if Check(k, "t", call("t", `{"m":"\u212a"}`)) == nil {
		t.Error("Kelvin sign matched k")
	}
	if Check(fold, "t", call("t", `{}`)) == nil {
		t.Error("missing field passed one_of")
	}
}

func TestAbsent(t *testing.T) {
	r := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"method":{"one_of":["GET"]},"body":{"absent":true}}]}]}`)
	if Check(r, "t", call("t", `{"method":"GET"}`)) != nil {
		t.Error("GET without body refused")
	}
	for _, a := range []string{`{"method":"GET","body":{}}`, `{"method":"GET","body":null}`, `{"method":"GET","body":""}`, `{"method":"GET","body":0}`} {
		if Check(r, "t", call("t", a)) == nil {
			t.Errorf("body present but passed: %s", a)
		}
	}
}

func TestShapesAreAlternativesAndConditionsAreAnded(t *testing.T) {
	r := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[
	  {"method":{"one_of":["GET"]}},
	  {"method":{"one_of":["POST"]},"op":{"one_of":["query","search"]}}
	]}]}`)
	ok := []string{`{"method":"GET"}`, `{"method":"GET","op":"delete"}`, `{"method":"POST","op":"query"}`, `{"method":"POST","op":"search"}`}
	no := []string{`{"method":"POST"}`, `{"method":"POST","op":"delete"}`, `{"method":"DELETE"}`, `{}`, `{"op":"query"}`}
	for _, a := range ok {
		if Check(r, "t", call("t", a)) != nil {
			t.Errorf("should pass: %s", a)
		}
	}
	for _, a := range no {
		if Check(r, "t", call("t", a)) == nil {
			t.Errorf("should fail: %s", a)
		}
	}
}

func TestUnruledToolIsUntouched(t *testing.T) {
	r := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"method":{"one_of":["GET"]}}]}]}`)
	// A tool with no rule passes even with odd arguments.
	for _, a := range []string{`{"method":"DELETE"}`, `{"method":"x","method":"y"}`, `not json`, ``} {
		p := json.RawMessage(`{"name":"other","arguments":` + a + `}`)
		if a == "" {
			p = json.RawMessage(`{"name":"other"}`)
		}
		if a == "not json" {
			continue
		}
		if d := Check(r, "other", p); d != nil {
			t.Errorf("unruled tool refused for %s: %v", a, d)
		}
	}
}

func TestMultipleRulesForOneToolAreAnded(t *testing.T) {
	both := rulesFor(t, `{"v":1,"rules":[
	  {"tool":"t","shapes":[{"method":{"one_of":["GET","POST"]}}]},
	  {"tool":"t","shapes":[{"method":{"one_of":["GET","DELETE"]}}]}
	]}`)
	if Check(both, "t", call("t", `{"method":"GET"}`)) != nil {
		t.Error("GET passes both rules")
	}
	for _, m := range []string{"POST", "DELETE", "PUT"} {
		if Check(both, "t", call("t", `{"method":"`+m+`"}`)) == nil {
			t.Errorf("%s must fail one of the two rules", m)
		}
	}
}

func TestDuplicateKeysRefusedBothOrders(t *testing.T) {
	r := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"method":{"one_of":["GET"]}}]}]}`)
	for _, a := range []string{
		`{"method":"GET","method":"DELETE"}`,
		`{"method":"DELETE","method":"GET"}`,
		`{"method":"GET","method":"GET"}`,
		`{"method":"GET","m\u0065thod":"DELETE"}`,
		`{"method":"GET","body":{"a":1,"a":2}}`,
		`{"method":"GET","body":[{"a":1,"a":2}]}`,
	} {
		if d := Check(r, "t", call("t", a)); d == nil || d.Reason != ReasonArgsUnreadable {
			t.Errorf("duplicate keys passed or wrong reason for %s: %v", a, d)
		}
	}
	// A duplicate at the top of params, in both orders.
	for _, p := range []string{
		`{"name":"t","arguments":{"method":"GET"},"arguments":{"method":"DELETE"}}`,
		`{"name":"t","arguments":{"method":"DELETE"},"arguments":{"method":"GET"}}`,
		`{"name":"t","name":"other","arguments":{"method":"GET"}}`,
		`{"name":"other","name":"t","arguments":{"method":"GET"}}`,
	} {
		if d := Check(r, "t", json.RawMessage(p)); d == nil {
			t.Errorf("duplicate top-level key passed: %s", p)
		}
	}
	// Keys that differ only by case are also refused (a lenient decoder merges them).
	for _, p := range []string{
		`{"name":"t","Arguments":{"method":"DELETE"},"arguments":{"method":"GET"}}`,
		`{"Name":"other","name":"t","arguments":{"method":"GET"}}`,
		`{"name":"t","arguments":{"method":"GET","METHOD":"DELETE"}}`,
		`{"name":"t","arguments":{"Method":"DELETE","method":"GET"}}`,
	} {
		if d := Check(r, "t", json.RawMessage(p)); d == nil {
			t.Errorf("case-folded duplicate passed: %s", p)
		}
	}
}

func TestMalformedParamsRefused(t *testing.T) {
	r := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"method":{"one_of":["GET"]}}]}]}`)
	for _, p := range []string{
		``, `null`, `[]`, `"x"`, `{"name":"t","arguments":[]}`, `{"name":"t","arguments":"GET"}`,
		`{"name":"t","arguments":{"method":"GET"}} {"x":1}`,
		"{\"name\":\"t\",\"arguments\":{\"method\":\"\xff\"}}",
		`{"name":7,"arguments":{}}`, `{"arguments":{"method":"GET"}}`, `{"name":"other","arguments":{"method":"GET"}}`,
	} {
		if d := Check(r, "t", json.RawMessage(p)); d == nil {
			t.Errorf("malformed params passed: %q", p)
		}
	}
	// arguments: null is the same as no arguments, so a required field is missing.
	if Check(r, "t", json.RawMessage(`{"name":"t","arguments":null}`)) == nil {
		t.Error("null arguments passed a method rule")
	}
	deep := strings.Repeat("[", 80) + strings.Repeat("]", 80)
	if Check(r, "t", json.RawMessage(`{"name":"t","arguments":{"method":"GET","x":`+deep+`}}`)) == nil {
		t.Error("deeply nested arguments passed")
	}
}

func TestCaseVariantOfLimitedFieldRefused(t *testing.T) {
	r := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"method":{"one_of":["GET"]}}]}]}`)
	if Check(r, "t", call("t", `{"method":"GET","Method":"DELETE"}`)) == nil {
		t.Error("Method variant next to method passed")
	}
}

func TestDenialNamesFieldButNeverValue(t *testing.T) {
	r := rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"method":{"one_of":["GET"]}}]}]}`)
	d := Check(r, "t", call("t", `{"method":"SECRET-CUSTOMER-VALUE"}`))
	if d == nil || d.Field != "method" {
		t.Fatalf("denial = %+v", d)
	}
	if strings.Contains(d.Error(), "SECRET") || strings.Contains(d.Reason, "SECRET") {
		t.Fatal("denial echoed an argument value")
	}
}

func TestParserRejectsMalformedRules(t *testing.T) {
	bad := map[string]string{
		"empty":               ``,
		"not object":          `[]`,
		"no version":          `{"rules":[{"tool":"t","shapes":[{"a":{"absent":true}}]}]}`,
		"v2":                  `{"v":2,"rules":[{"tool":"t","shapes":[{"a":{"absent":true}}]}]}`,
		"v string":            `{"v":"1","rules":[{"tool":"t","shapes":[{"a":{"absent":true}}]}]}`,
		"no rules":            `{"v":1}`,
		"empty rules":         `{"v":1,"rules":[]}`,
		"unknown doc key":     `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"absent":true}}]}],"x":1}`,
		"unknown rule key":    `{"v":1,"rules":[{"tool":"t","x":1,"shapes":[{"a":{"absent":true}}]}]}`,
		"no tool":             `{"v":1,"rules":[{"shapes":[{"a":{"absent":true}}]}]}`,
		"wildcard tool":       `{"v":1,"rules":[{"tool":"*","shapes":[{"a":{"absent":true}}]}]}`,
		"wildcard prefix":     `{"v":1,"rules":[{"tool":"db:*","shapes":[{"a":{"absent":true}}]}]}`,
		"comma tool":          `{"v":1,"rules":[{"tool":"a,b","shapes":[{"a":{"absent":true}}]}]}`,
		"no shapes":           `{"v":1,"rules":[{"tool":"t"}]}`,
		"empty shapes":        `{"v":1,"rules":[{"tool":"t","shapes":[]}]}`,
		"empty shape":         `{"v":1,"rules":[{"tool":"t","shapes":[{}]}]}`,
		"unknown condition":   `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"matches":"x"}}]}]}`,
		"unknown with known":  `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"one_of":["x"],"regex":"."}}]}]}`,
		"no condition":        `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{}}]}]}`,
		"one_of empty":        `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"one_of":[]}}]}]}`,
		"one_of number":       `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"one_of":[1]}}]}]}`,
		"one_of not list":     `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"one_of":"x"}}]}]}`,
		"flag without one_of": `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"max":"1","ascii_case_insensitive":true}}]}]}`,
		"flag not bool":       `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"one_of":["x"],"ascii_case_insensitive":"yes"}}]}]}`,
		"max number":          `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"max":10}}]}]}`,
		"max bad decimal":     `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"max":"1e3"}}]}]}`,
		"max empty":           `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"max":""}}]}]}`,
		"min above max":       `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"max":"1","min":"2"}}]}]}`,
		"mixed one_of max":    `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"one_of":["x"],"max":"1"}}]}]}`,
		"absent false":        `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"absent":false}}]}]}`,
		"absent with max":     `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"absent":true,"max":"1"}}]}]}`,
		"url no hosts":        `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{}}}]}]}`,
		"url upper host":      `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["Example.com"]}}}]}]}`,
		"url ip host":         `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["10.0.0.1"]}}}]}]}`,
		"url dot host":        `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["example.com."]}}}]}]}`,
		"url unicode host":    `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["bücher.example"]}}}]}]}`,
		"url port zero":       `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["a.com"],"ports":[0]}}}]}]}`,
		"url bad prefix":      `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["a.com"],"path_prefixes":["v1"]}}}]}]}`,
		"url dotdot prefix":   `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["a.com"],"path_prefixes":["/a/../b"]}}}]}]}`,
		"url percent prefix":  `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["a.com"],"path_prefixes":["/a%2Fb"]}}}]}]}`,
		"url trailing slash":  `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["a.com"],"path_suffixes":["/query/"]}}}]}]}`,
		"url root suffix":     `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["a.com"],"path_suffixes":["/"]}}}]}]}`,
		"url unknown key":     `{"v":1,"rules":[{"tool":"t","shapes":[{"u":{"url":{"hosts":["a.com"],"regex":"x"}}}]}]}`,
		"bad field name":      `{"v":1,"rules":[{"tool":"t","shapes":[{"a b":{"absent":true}}]}]}`,
		"empty field name":    `{"v":1,"rules":[{"tool":"t","shapes":[{"":{"absent":true}}]}]}`,
		"duplicate key":       `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"one_of":["x"],"one_of":["y"]}}]}]}`,
		"duplicate field":     `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"absent":true},"a":{"one_of":["x"]}}]}]}`,
		"trailing data":       `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"absent":true}}]}]} {}`,
	}
	for name, doc := range bad {
		if _, err := ParseJSON([]byte(doc)); err == nil {
			t.Errorf("%s: parsed, want refusal", name)
		}
	}
}

func TestParserSizeLimits(t *testing.T) {
	many := func(n int, item string) string {
		parts := make([]string, n)
		for i := range parts {
			parts[i] = item
		}
		return strings.Join(parts, ",")
	}
	shape := `{"a":{"absent":true}}`
	rule := `{"tool":"t","shapes":[` + shape + `]}`
	if _, err := ParseJSON([]byte(`{"v":1,"rules":[` + many(MaxRulesPerToken, rule) + `]}`)); err != nil {
		t.Errorf("at the rule limit: %v", err)
	}
	if _, err := ParseJSON([]byte(`{"v":1,"rules":[` + many(MaxRulesPerToken+1, rule) + `]}`)); err == nil {
		t.Error("over the rule limit parsed")
	}
	if _, err := ParseJSON([]byte(`{"v":1,"rules":[{"tool":"t","shapes":[` + many(MaxShapesPerRule+1, shape) + `]}]}`)); err == nil {
		t.Error("over the shape limit parsed")
	}
	fields := make([]string, MaxFieldsPerShape+1)
	for i := range fields {
		fields[i] = `"f` + string(rune('a'+i)) + `":{"absent":true}`
	}
	if _, err := ParseJSON([]byte(`{"v":1,"rules":[{"tool":"t","shapes":[{` + strings.Join(fields, ",") + `}]}]}`)); err == nil {
		t.Error("over the field limit parsed")
	}
	vals := make([]string, MaxOneOfValues+1)
	for i := range vals {
		vals[i] = `"v` + string(rune('a'+i%26)) + string(rune('a'+i/26)) + `"`
	}
	if _, err := ParseJSON([]byte(`{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"one_of":[` + strings.Join(vals, ",") + `]}}]}]}`)); err == nil {
		t.Error("over the one_of limit parsed")
	}
	long := strings.Repeat("x", MaxOneOfValueBytes+1)
	if _, err := ParseJSON([]byte(`{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"one_of":["` + long + `"]}}]}]}`)); err == nil {
		t.Error("over-long one_of value parsed")
	}
	pad := strings.Repeat(" ", MaxDocBytes)
	if _, err := ParseJSON([]byte(`{"v":1,"rules":[` + rule + `]}` + pad)); err == nil {
		t.Error("over the document limit parsed")
	}
	deep := strings.Repeat("[", 40) + strings.Repeat("]", 40)
	if _, err := ParseJSON([]byte(`{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"one_of":[` + deep + `]}}]}]}`)); err == nil {
		t.Error("deep nesting parsed")
	}
}

func TestMarshalRoundTripAndDeterminism(t *testing.T) {
	doc := `{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[
	  {"method":{"one_of":["GET"],"ascii_case_insensitive":true}},
	  {"method":{"one_of":["POST"],"ascii_case_insensitive":true},"url":{"url":{"hosts":["www.wixapis.com"],"path_suffixes":["/query"]}}},
	  {"qty":{"max":"10","min":"0.5"},"body":{"absent":true}}
	]}]}`
	rules := rulesFor(t, doc)
	a, err := Marshal(rules)
	if err != nil {
		t.Fatal(err)
	}
	again, err := ParseJSON(a)
	if err != nil {
		t.Fatal(err)
	}
	b, err := Marshal(again)
	if err != nil {
		t.Fatal(err)
	}
	if string(a) != string(b) {
		t.Fatalf("encoding is not stable:\n%s\n%s", a, b)
	}
	v, err := ScopeValue(rules)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(v, "argrules:v1:") || strings.ContainsAny(v, ", \t") {
		t.Fatalf("scope value %q must be one word with the reserved prefix", v)
	}
	back, err := ParseScopeValue(v)
	if err != nil || len(back) != 1 || len(back[0].Shapes) != 3 {
		t.Fatalf("round trip: %v %+v", err, back)
	}
	if _, err := Marshal(nil); err == nil {
		t.Error("empty rules marshalled")
	}
}

func TestScopeValueRefusals(t *testing.T) {
	good, _ := ScopeValue(rulesFor(t, `{"v":1,"rules":[{"tool":"t","shapes":[{"a":{"absent":true}}]}]}`))
	for name, v := range map[string]string{
		"unknown version": "argrules:v2:" + strings.TrimPrefix(good, "argrules:v1:"),
		"no version":      "argrules:" + strings.TrimPrefix(good, "argrules:v1:"),
		"padding":         good + "=",
		"not base64":      "argrules:v1:!!!!",
		"empty payload":   "argrules:v1:",
		"huge payload":    "argrules:v1:" + strings.Repeat("A", 20000),
	} {
		if _, err := ParseScopeValue(v); err == nil {
			t.Errorf("%s: parsed", name)
		}
	}
}

func allowAll(string, string) bool { return true }

func TestCollect(t *testing.T) {
	v1, _ := ScopeValue(rulesFor(t, `{"v":1,"rules":[{"tool":"a","shapes":[{"m":{"one_of":["GET","POST"]}}]}]}`))
	v2, _ := ScopeValue(rulesFor(t, `{"v":1,"rules":[{"tool":"a","shapes":[{"m":{"one_of":["GET"]}}]}]}`))
	got, err := Collect([]string{"tenant_id = x", "scope = mcp:*", "scope = " + v1, "scope = " + v2}, allowAll)
	if err != nil || len(got) != 2 {
		t.Fatalf("Collect = %v, %v", got, err)
	}
	if got, err := Collect([]string{"scope = a,b"}, allowAll); err != nil || got != nil {
		t.Fatalf("no rules = %v, %v", got, err)
	}
	// A rule for a tool the scope does not allow.
	only := func(scope, tool string) bool { return scope == tool }
	if _, err := Collect([]string{"scope = b", "scope = " + v1}, only); err == nil {
		t.Error("rule for a tool outside the scope was accepted")
	}
	if _, err := Collect([]string{"scope = a", "scope = " + v1}, only); err != nil {
		t.Errorf("rule for an allowed tool refused: %v", err)
	}
	// A scope that comes after a rule may narrow the token; the rule stays.
	if _, err := Collect([]string{"scope = a", "scope = " + v1, "scope = b"}, only); err != nil {
		t.Errorf("later narrowing refused: %v", err)
	}
	// Too many rule caveats.
	var cav []string
	for i := 0; i <= MaxRuleCaveats; i++ {
		cav = append(cav, "scope = "+v1)
	}
	if _, err := Collect(cav, allowAll); err == nil {
		t.Error("over the rule-caveat limit accepted")
	}
	// Total rules over the limit.
	var rl []string
	for i := 0; i < MaxRulesPerToken; i++ {
		rl = append(rl, `{"tool":"a","shapes":[{"m":{"absent":true}}]}`)
	}
	big, _ := ScopeValue(rulesFor(t, `{"v":1,"rules":[`+strings.Join(rl, ",")+`]}`))
	if _, err := Collect([]string{"scope = " + big, "scope = " + v1}, allowAll); err == nil {
		t.Error("total over the rules-per-token limit accepted")
	}
	// A rule word mixed into a list is refused.
	if _, err := Collect([]string{"scope = a," + v1}, allowAll); err == nil {
		t.Error("rule word hidden in a scope list accepted")
	}
	if _, err := Collect([]string{"scope = " + v1 + ",a"}, allowAll); err == nil {
		t.Error("rule word with a trailing tool accepted")
	}
	if _, err := Collect([]string{"scope = argrules:v9:abc"}, allowAll); err == nil {
		t.Error("unknown rule version accepted")
	}
}

func TestDescribe(t *testing.T) {
	rules := rulesFor(t, `{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[
	  {"method":{"one_of":["GET"],"ascii_case_insensitive":true}},
	  {"method":{"one_of":["POST"]},"url":{"url":{"hosts":["www.wixapis.com"],"path_suffixes":["/query"]}}},
	  {"qty":{"max":"10"}}
	]}]}`)
	lines := Describe(rules)
	if len(lines) != 1 {
		t.Fatalf("lines = %v", lines)
	}
	for _, want := range []string{"CallWixSiteAPI", "method is GET (any letter case)", "www.wixapis.com", "/query", "qty is at most 10"} {
		if !strings.Contains(lines[0], want) {
			t.Errorf("description %q lacks %q", lines[0], want)
		}
	}
}

func jsonMarshalString(s string) (string, error) {
	b, err := json.Marshal(s)
	return string(b), err
}
