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

// The url condition names the host and path prefixes the rule allows, never
// the address the call sent.
func TestArgumentRefusalDescribesURLRule(t *testing.T) {
	const doc = `{"v":1,"rules":[{"tool":"fetch","shapes":[{"url":{"url":{"hosts":["api.example.com"],"path_prefixes":["/v1","/v2"]}}}]}]}`
	proxy, _, _ := newArgProxy(t)
	_, tok := argToken(t, "fetch", doc)
	o := runCall(t, proxy, argCall(1, tok, "fetch", `{"url":"https://evil.example/MARKER-URL-PATH"}`))
	assertArgDenied(t, o, "fetch", "url")
	want := "url must be an https address on api.example.com with a path starting /v1 or /v2"
	var data map[string]any
	_ = json.Unmarshal(o.resp.Error.Data, &data)
	if data["allowed"] != want || !strings.HasSuffix(o.resp.Error.Message, ": "+want) {
		t.Fatalf("allowed = %v, message = %q", data["allowed"], o.resp.Error.Message)
	}
	raw, _ := json.Marshal(o.resp)
	if strings.Contains(string(raw), "evil.example") || strings.Contains(string(raw), "MARKER-URL-PATH") {
		t.Fatalf("reply leaks the call's address: %s", raw)
	}
}

// Refusals that are not a shape mismatch keep their old reply exactly.
func TestNonShapeRefusalsHaveNoAllowedText(t *testing.T) {
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
func TestAllowedTextIsNotInTheRecordedDecision(t *testing.T) {
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
