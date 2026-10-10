package mcpserver

import (
	"encoding/json"
	"strings"
	"testing"
)

// A refusal says what the closest shape allows: in error.data.allowed and as
// the end of error.message. The text comes from the rule; the call's own
// values never appear anywhere in the reply.
func TestArgumentRefusalSaysWhatIsAllowed(t *testing.T) {
	const marker = "MARKER-ARG-VALUE-5521"
	proxy, rr, _ := newArgProxy(t)
	_, tok := argToken(t, sleeveScope, sleeveOrderLimitsDoc)
	cases := []struct {
		name, tool, args, field, allowed string
	}{
		{"max", "place_crypto_order", `{"side":"buy","dollar_amount":"250.` + "5521" + `"}`, "dollar_amount", "dollar_amount must be at most 100"},
		{"one_of", "place_crypto_order", `{"side":"` + marker + `","dollar_amount":"5"}`, "side", "side must be one of: buy"},
		{"absent", "place_crypto_order", `{"side":"buy","dollar_amount":"5","quantity":"` + marker + `"}`, "quantity", "quantity must not be set"},
		{"cap on option", "place_option_order", `{"quantity":"` + marker + `"}`, "quantity", "quantity must be at most 1"},
		{"ladder top", "place_equity_order", `{"side":"buy","type":"limit","quantity":"1","limit_price":"150","symbol":"` + marker + `"}`, "limit_price", "limit_price must be at most 100"},
	}
	for i, tc := range cases {
		o := runCall(t, proxy, argCall(i+1, tok, tc.tool, tc.args))
		assertArgDenied(t, o, tc.tool, tc.field)
		var data map[string]any
		if err := json.Unmarshal(o.resp.Error.Data, &data); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if data["allowed"] != tc.allowed {
			t.Fatalf("%s: data.allowed = %v, want %q", tc.name, data["allowed"], tc.allowed)
		}
		wantMessage := `tool "` + tc.tool + `" argument "` + tc.field + `": arguments are outside what this token allows: ` + tc.allowed
		if o.resp.Error.Message != wantMessage {
			t.Fatalf("%s: message = %q, want %q", tc.name, o.resp.Error.Message, wantMessage)
		}
		raw, _ := json.Marshal(o.resp)
		for _, leak := range []string{marker, "250.5521", tok} {
			if strings.Contains(string(raw), leak) {
				t.Fatalf("%s: reply leaks %q: %s", tc.name, leak, raw)
			}
		}
	}
	if rr.calls() != 0 {
		t.Fatalf("refused calls reached the upstream %d times", rr.calls())
	}
}

// The url condition names the host the rule allows and says only that a path
// is limited. It never shows the rule's paths and never the address the call
// sent.
func TestArgumentRefusalDescribesURLRule(t *testing.T) {
	const doc = `{"v":1,"rules":[{"tool":"fetch","shapes":[{"url":{"url":{"hosts":["api.example.com"],"path_prefixes":["/v1","/v2"]}}}]}]}`
	proxy, _, _ := newArgProxy(t)
	_, tok := argToken(t, "fetch", doc)
	o := runCall(t, proxy, argCall(1, tok, "fetch", `{"url":"https://evil.example/MARKER-URL-PATH"}`))
	assertArgDenied(t, o, "fetch", "url")
	want := "url must be an https address on api.example.com with a path this token allows"
	var data map[string]any
	_ = json.Unmarshal(o.resp.Error.Data, &data)
	if data["allowed"] != want || !strings.HasSuffix(o.resp.Error.Message, ": "+want) {
		t.Fatalf("allowed = %v, message = %q", data["allowed"], o.resp.Error.Message)
	}
	raw, _ := json.Marshal(o.resp)
	if strings.Contains(string(raw), "evil.example") || strings.Contains(string(raw), "MARKER-URL-PATH") || strings.Contains(string(raw), "/v1") || strings.Contains(string(raw), "/v2") {
		t.Fatalf("reply leaks the call's address: %s", raw)
	}
}

// Refusals that are not a shape mismatch keep their old reply exactly.
func TestNonShapeRefusalsHaveNoAgentText(t *testing.T) {
	proxy, _, _ := newArgProxy(t)
	_, tok := argToken(t, sleeveScope, sleeveOrderLimitsDoc)
	// Duplicate key: arguments could not be checked.
	o := runCall(t, proxy, argCall(1, tok, "place_crypto_order", `{"side":"buy","side":"sell","dollar_amount":"5"}`))
	if o.resp.Error == nil {
		t.Fatal("not refused")
	}
	var data map[string]any
	_ = json.Unmarshal(o.resp.Error.Data, &data)
	if _, has := data["allowed"]; has {
		t.Fatalf("unreadable-arguments refusal carries allowed: %v", data)
	}
	if o.resp.Error.Message != `tool "place_crypto_order": arguments could not be checked` {
		t.Fatalf("message changed: %q", o.resp.Error.Message)
	}
	// Scope refusal.
	o = runCall(t, proxy, argCall(2, tok, "delete_everything", `{}`))
	if o.resp.Error == nil || strings.Contains(string(o.resp.Error.Data), "allowed") {
		t.Fatalf("scope refusal changed: %+v", o.resp.Error)
	}
}

// The text is for the caller only. What goes to the evidence recorder is the
// same decision as before this change: no field of it holds allowed text, and
// the decision is identical whether the rule text is long or short.
func TestAgentTextIsNotInTheRecordedDecision(t *testing.T) {
	proxy, rec := recordArgProxy(t)
	_, tok := argToken(t, sleeveScope, sleeveOrderLimitsDoc)
	o := runCall(t, proxy, argCall(1, tok, "place_crypto_order", `{"side":"buy","dollar_amount":"250"}`))
	assertArgDenied(t, o, "place_crypto_order", "dollar_amount")
	if len(rec.decisions) != 1 {
		t.Fatalf("recorded %d decisions", len(rec.decisions))
	}
	blob, _ := json.Marshal(rec.decisions[0])
	for _, bad := range []string{"must be", "at most", "allowed", "outside what"} {
		if strings.Contains(string(blob), bad) {
			t.Fatalf("recorded decision carries allowed text %q: %s", bad, blob)
		}
	}
	d := rec.decisions[0]
	if d.Decision != "denied" || d.DecisionReason != "policy_denied" || !d.BudgetNotEvaluated ||
		d.DenialCode != "TOOL_ARGUMENT_DENIED" || d.ArgumentField != "dollar_amount" || d.ArgumentRulesSHA256 == "" {
		t.Fatalf("decision changed: %+v", d)
	}
}

// Astra's scenario at the HTTP boundary: the rule's path prefix is a webhook
// credential. The refusal names the host and never the path, in the message,
// in error.data, or anywhere else in the reply. A one_of that holds a
// credential-shaped value is reported by count only.
func TestIndependentCredentialURLRuleDisclosure(t *testing.T) {
	const (
		pathMarker = "REVIEW-FAKE-WEBHOOK-CREDENTIAL-729a"
		oneOfKey   = "sk-live-REVIEW-FAKE-KEY-5521"
		doc        = `{"v":1,"rules":[` +
			`{"tool":"fetch","shapes":[{"url":{"url":{"hosts":["hooks.example.com"],"path_prefixes":["/services/` + pathMarker + `"]}}}]},` +
			`{"tool":"pay","shapes":[{"account":{"one_of":["` + oneOfKey + `","acct-1"]}}]}]}`
	)
	proxy, rr, _ := newArgProxy(t)
	_, tok := argToken(t, "fetch,pay", doc)
	cases := []struct {
		name, tool, args, field, allowed string
	}{
		{"url", "fetch", `{"url":"https://hooks.example.com/not-permitted"}`, "url", "url must be an https address on hooks.example.com with a path this token allows"},
		{"one_of", "pay", `{"account":"other"}`, "account", "account must be one of the 2 values this token allows"},
	}
	for i, tc := range cases {
		o := runCall(t, proxy, argCall(i+1, tok, tc.tool, tc.args))
		assertArgDenied(t, o, tc.tool, tc.field)
		var data map[string]any
		if err := json.Unmarshal(o.resp.Error.Data, &data); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if data["allowed"] != tc.allowed || !strings.HasSuffix(o.resp.Error.Message, ": "+tc.allowed) {
			t.Fatalf("%s: allowed = %v, message = %q", tc.name, data["allowed"], o.resp.Error.Message)
		}
		raw, _ := json.Marshal(o.resp)
		for _, leak := range []string{pathMarker, "/services", oneOfKey, "sk-live", "REVIEW-FAKE"} {
			if strings.Contains(string(raw), leak) || strings.Contains(o.resp.Error.Message, leak) || strings.Contains(string(o.resp.Error.Data), leak) {
				t.Fatalf("%s: reply discloses rule value %q: %s", tc.name, leak, raw)
			}
		}
	}
	if rr.calls() != 0 {
		t.Fatalf("refused calls reached the upstream %d times", rr.calls())
	}
}

// Round 3 regression (Astra round 2): a 24-character alphanumeric API key with
// no recognised prefix looks like a plain value. Value shape does not decide
// disclosure; the field name does. The key is absent from the whole refusal
// and the count line is present. Allowlisted fields (side, symbol) still show
// their values, and a non-allowlisted field with a plain value is count-only.
func TestOpaqueKeyOneOfIsCountOnlyAtTheHTTPBoundary(t *testing.T) {
	const (
		apiKey = "a7B9c2D4e6F8g1H3j5K7m9N2"
		doc    = `{"v":1,"rules":[` +
			`{"tool":"call_api","shapes":[{"api_key":{"one_of":["` + apiKey + `"]}}]},` +
			`{"tool":"region_api","shapes":[{"region":{"one_of":["us-east-1"]}}]},` +
			`{"tool":"trade","shapes":[{"side":{"one_of":["buy"]},"symbol":{"one_of":["BTC-USD"]},"account_number":{"one_of":["123456789"]}}]}]}`
	)
	proxy, rr, _ := newArgProxy(t)
	_, tok := argToken(t, "call_api,region_api,trade", doc)
	cases := []struct {
		name, tool, args, field, allowed string
		absent                           []string
	}{
		{"opaque key", "call_api", `{"api_key":"wrong"}`, "api_key", "api_key must be one of the 1 values this token allows", []string{apiKey}},
		{"plain value, field not allowlisted", "region_api", `{"region":"eu-west-1"}`, "region", "region must be one of the 1 values this token allows", []string{"us-east-1"}},
		{"allowlisted side", "trade", `{"side":"sell","symbol":"BTC-USD","account_number":"123456789"}`, "side", "side must be one of: buy", nil},
		{"allowlisted symbol", "trade", `{"side":"buy","symbol":"ETH-USD","account_number":"123456789"}`, "symbol", "symbol must be one of: BTC-USD", nil},
		{"allowlisted account_number", "trade", `{"side":"buy","symbol":"BTC-USD","account_number":"000"}`, "account_number", "account_number must be one of: 123456789", nil},
	}
	for i, tc := range cases {
		o := runCall(t, proxy, argCall(i+1, tok, tc.tool, tc.args))
		assertArgDenied(t, o, tc.tool, tc.field)
		var data map[string]any
		if err := json.Unmarshal(o.resp.Error.Data, &data); err != nil {
			t.Fatalf("%s: %v", tc.name, err)
		}
		if data["allowed"] != tc.allowed || !strings.HasSuffix(o.resp.Error.Message, ": "+tc.allowed) {
			t.Fatalf("%s: allowed = %v, message = %q", tc.name, data["allowed"], o.resp.Error.Message)
		}
		raw, _ := json.Marshal(o.resp)
		for _, leak := range tc.absent {
			if strings.Contains(string(raw), leak) {
				t.Fatalf("%s: reply discloses %q: %s", tc.name, leak, raw)
			}
		}
	}
	if rr.calls() != 0 {
		t.Fatalf("refused calls reached the upstream %d times", rr.calls())
	}
}
