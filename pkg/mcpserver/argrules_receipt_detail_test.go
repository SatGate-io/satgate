package mcpserver

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"strings"
	"testing"

	"github.com/satgate-io/satgate/pkg/argrules"
)

// sleeveOrderLimitsDoc caps what an agent may order: sells are free, a market
// buy is at most 100 dollars, limit buys follow a ladder of quantity and price.
const sleeveOrderLimitsDoc = `{
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

const sleeveScope = "place_equity_order,place_crypto_order,place_option_order,exercise_option"

// A tools/call refused by an argument rule names the field that blocked it,
// not the first field in name order.
func TestArgumentRefusalNamesTheBlockingField(t *testing.T) {
	proxy, rr, _ := newArgProxy(t)
	_, tok := argToken(t, sleeveScope, sleeveOrderLimitsDoc)
	cases := []struct {
		name, tool, args, field string
	}{
		{"crypto market over cap", "place_crypto_order", `{"side":"buy","type":"market","dollar_amount":"250"}`, "dollar_amount"},
		{"equity market over cap", "place_equity_order", `{"side":"buy","type":"market","dollar_amount":"500"}`, "dollar_amount"},
		{"equity limit between rungs", "place_equity_order", `{"side":"buy","type":"limit","quantity":"10","limit_price":"12"}`, "quantity"},
		{"equity limit over top rung", "place_equity_order", `{"side":"buy","type":"limit","quantity":"1","limit_price":"150"}`, "limit_price"},
	}
	for i, tc := range cases {
		o := runCall(t, proxy, argCall(i+1, tok, tc.tool, tc.args))
		assertArgDenied(t, o, tc.tool, tc.field)
		if !strings.Contains(o.resp.Error.Message, `argument "`+tc.field+`"`) {
			t.Fatalf("%s: message %q does not name %s", tc.name, o.resp.Error.Message, tc.field)
		}
	}
	if rr.calls() != 0 {
		t.Fatalf("refused calls reached the upstream %d times", rr.calls())
	}
	// A call inside the limits still goes through.
	o := runCall(t, proxy, argCall(9, tok, "place_equity_order", `{"side":"buy","type":"limit","quantity":"8","limit_price":"12.5"}`))
	if o.denied {
		t.Fatalf("in-limit order refused: %+v", o.resp.Error)
	}
}

func recordArgProxy(t *testing.T) (*Proxy, *fakeEvidenceRecorder) {
	t.Helper()
	proxy, _, _ := newArgProxy(t)
	rec := &fakeEvidenceRecorder{}
	proxy.SetEvidenceRecorder(rec)
	return proxy, rec
}

// The decision handed to the recorder for an argument refusal carries the
// stable code, the field, and the hash of the rule documents the token has.
func TestArgumentRefusalDecisionCarriesReceiptDetail(t *testing.T) {
	proxy, rec := recordArgProxy(t)
	_, tok := argToken(t, sleeveScope, sleeveOrderLimitsDoc)

	// Independent expectation: the canonical document of the rule, hashed as
	// a one-element JSON array.
	rules, err := argrules.ParseJSON([]byte(sleeveOrderLimitsDoc))
	if err != nil {
		t.Fatal(err)
	}
	canonical, err := argrules.Marshal(rules)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256([]byte("[" + string(canonical) + "]"))
	wantHash := hex.EncodeToString(sum[:])

	o := runCall(t, proxy, argCall(1, tok, "place_crypto_order", `{"side":"buy","type":"market","dollar_amount":"SECRET-250"}`))
	assertArgDenied(t, o, "place_crypto_order", "dollar_amount")
	if len(rec.decisions) != 1 {
		t.Fatalf("recorded %d decisions", len(rec.decisions))
	}
	d := rec.decisions[0]
	if d.Decision != "denied" || d.DecisionReason != "policy_denied" || !d.BudgetNotEvaluated {
		t.Fatalf("decision changed: %+v", d)
	}
	if d.DenialCode != "TOOL_ARGUMENT_DENIED" || d.ArgumentField != "dollar_amount" || d.ArgumentRulesSHA256 != wantHash {
		t.Fatalf("detail = code %q field %q hash %q, want TOOL_ARGUMENT_DENIED dollar_amount %s", d.DenialCode, d.ArgumentField, d.ArgumentRulesSHA256, wantHash)
	}
	blob, _ := json.Marshal(d)
	if strings.Contains(string(blob), "SECRET") || strings.Contains(string(blob), tok) {
		t.Fatalf("decision carries an argument value or the token: %s", blob)
	}
}

// The hash covers every rule document in token order, so two caveats hash
// differently from one, and a different order hashes differently.
func TestArgumentRulesHashFollowsTokenOrder(t *testing.T) {
	capDoc := `{"v":1,"rules":[{"tool":"place_option_order","shapes":[{"quantity":{"max":"1"}}]}]}`
	hashFor := func(docs ...string) string {
		proxy, rec := recordArgProxy(t)
		_, tok := argToken(t, sleeveScope, docs...)
		runCall(t, proxy, argCall(1, tok, "place_option_order", `{"quantity":"9"}`))
		if len(rec.decisions) != 1 {
			t.Fatalf("recorded %d decisions", len(rec.decisions))
		}
		return rec.decisions[0].ArgumentRulesSHA256
	}
	ab := hashFor(capDoc, sleeveOrderLimitsDoc)
	ba := hashFor(sleeveOrderLimitsDoc, capDoc)
	a := hashFor(capDoc)
	if len(ab) != 64 || ab == ba || ab == a || ba == a {
		t.Fatalf("hashes not distinct: ab=%s ba=%s a=%s", ab, ba, a)
	}
	// Same token documents give the same hash (the nonce is not part of it).
	if again := hashFor(capDoc); again != a {
		t.Fatalf("hash depends on the nonce: %s vs %s", again, a)
	}
}

// A scope refusal leaves the decision exactly as it was: none of the new
// fields is set, and none appears in its JSON form.
func TestScopeRefusalDecisionHasNoArgumentDetail(t *testing.T) {
	proxy, rec := recordArgProxy(t)
	_, tok := argToken(t, "place_equity_order", sleeveOrderLimitsDocEquityOnly)
	o := runCall(t, proxy, argCall(1, tok, "delete_everything", `{}`))
	if o.resp.Error == nil || o.resp.Error.Code != CodePolicyDenied {
		t.Fatalf("not refused: %+v", o.resp)
	}
	if len(rec.decisions) != 1 {
		t.Fatalf("recorded %d decisions", len(rec.decisions))
	}
	d := rec.decisions[0]
	if d.DenialCode != "" || d.ArgumentField != "" || d.ArgumentRulesSHA256 != "" {
		t.Fatalf("scope refusal carries argument detail: %+v", d)
	}
	blob, _ := json.Marshal(d)
	for _, key := range []string{"denial_code", "argument_field", "argument_rules_sha256"} {
		if strings.Contains(string(blob), key) {
			t.Fatalf("scope refusal JSON has %s: %s", key, blob)
		}
	}
	if d.Decision != "denied" || d.DecisionReason != "policy_denied" || !d.BudgetNotEvaluated {
		t.Fatalf("scope decision changed: %+v", d)
	}
}

// Other decisions (an allowed call) carry nothing new either.
func TestAllowedDecisionHasNoArgumentDetail(t *testing.T) {
	proxy, rec := recordArgProxy(t)
	_, tok := argToken(t, sleeveScope, sleeveOrderLimitsDoc)
	o := runCall(t, proxy, argCall(1, tok, "place_equity_order", `{"side":"sell"}`))
	if o.denied {
		t.Fatalf("refused: %+v", o.resp.Error)
	}
	for _, d := range rec.decisions {
		if d.DenialCode != "" || d.ArgumentField != "" || d.ArgumentRulesSHA256 != "" {
			t.Fatalf("allowed decision carries argument detail: %+v", d)
		}
	}
}

const sleeveOrderLimitsDocEquityOnly = `{"v":1,"rules":[{"tool":"place_equity_order","shapes":[{"side":{"one_of":["sell"]}}]}]}`
