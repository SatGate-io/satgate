package toolcontrols

import (
	"encoding/json"
	"reflect"
	"testing"
)

func TestRobinhoodSellDollarBudgetRule(t *testing.T) {
	r := RobinhoodSellDollarBudgetRule
	d := r.Definition()
	want := DollarBudgetRuleDefinition{Service: "robinhood", Tools: []string{"place_equity_order", "place_crypto_order"}, Side: "sell"}
	if !reflect.DeepEqual(d, want) {
		t.Fatalf("definition: %+v", d)
	}
	d.Tools[0] = "changed"
	if !reflect.DeepEqual(r.Definition(), want) {
		t.Fatal("mutable policy")
	}
	for _, tool := range want.Tools {
		for _, side := range []string{"sell", "SELL", "SeLl"} {
			params := json.RawMessage(`{"name":"` + tool + `","arguments":{"side":"` + side + `","symbol":"TEST","dollar_amount":"20.01"}}`)
			o, ok := r.ReadOrderOfCall(params, tool, "robinhood", tool)
			if !ok || o.Service != "robinhood" || o.Tool != tool || o.Side != "sell" || o.Symbol != "TEST" || o.DollarAmount == nil || o.DollarAmount.String() != "20.01" || o.Quantity != nil {
				t.Fatalf("parsed: %+v %v", o, ok)
			}
		}
	}
}

func TestDollarBudgetRuleStrictArguments(t *testing.T) {
	for _, args := range []string{
		`{"side":"sell","side":"sell","symbol":"TEST","dollar_amount":"20"}`,
		`{"side":"sell","Side":"sell","symbol":"TEST","dollar_amount":"20"}`,
		`{"Side":"sell","symbol":"TEST","dollar_amount":"20"}`,
		`{"side":"ſell","symbol":"TEST","dollar_amount":"20"}`,
		`{"side":"sеll","symbol":"TEST","dollar_amount":"20"}`,
		`{"side":null,"symbol":"TEST","dollar_amount":"20"}`,
		`{"side":" sell","symbol":"TEST","dollar_amount":"20"}`,
		`{"side":"sell ","symbol":"TEST","dollar_amount":"20"}`,
		`{"symbol":"TEST","dollar_amount":"20"}`,
		`{"side":"sell","symbol":"TEST","dollar_amount":"20","quantity":"1"}`,
		`{"side":"sell","symbol":"TEST","dollar_amount":null}`,
		`{"side":"sell","symbol":"TEST","Dollar_amount":"20"}`,
		`{"side":"sell","symbol":"TEST","dollar_amount":"20","Quantity":"1"}`,
		`{"side":"sell","symbol":"TEST","quantity":"01"}`,
		`{"side":"sell","symbol":"TEST","quantity":1e2}`,
		`{"side":"sell","symbol":"TEST","quantity":" 1"}`,
		`{"side":"sell","symbol":"TEST","quantity":"0"}`,
		`{"side":"sell","symbol":"TEST","quantity":"-1"}`,
		`{"side":"sell","symbol":"TEST"}`,
		`{"side":"sell","symbol":null,"quantity":"1"}`,
		`{"side":"sell","symbol":"TEST","Symbol":"OTHER","quantity":"1"}`,
		`{"side":"sell","quantity":"1"}`,
		`{"side":"sell","symbol":" TEST","quantity":"1"}`,
		`{"side":"sell","symbol":"\ud800","quantity":"1"}`,
		`{"side":"sell","symbol":"TEST","quantity":"1","other":{"x":1,"x":2}}`,
	} {
		t.Run(args, func(t *testing.T) {
			p := json.RawMessage(`{"name":"place_equity_order","arguments":` + args + `}`)
			if _, ok := RobinhoodSellDollarBudgetRule.ReadOrderOfCall(p, "place_equity_order", "robinhood", "place_equity_order"); ok {
				t.Fatal("ambiguous/malformed order matched")
			}
		})
	}
}

func TestDollarBudgetRuleNamesAndQuantity(t *testing.T) {
	const sent = "second_order"
	p := json.RawMessage(`{"name":"second_order","arguments":{"side":"SeLl","symbol":"TEST","quantity":0.125}}`)
	o, ok := RobinhoodSellDollarBudgetRule.ReadOrderOfCall(p, sent, "Robinhood", "PLACE_EQUITY_ORDER")
	if !ok || o.Quantity == nil || o.Quantity.String() != "0.125" || o.DollarAmount != nil || o.Tool != "PLACE_EQUITY_ORDER" {
		t.Fatalf("quantity lost: %+v %v", o, ok)
	}
	for _, service := range []string{"", "alpaca", "kalshi", "robinhood-other", "robinhood ", "robinhοod"} {
		if _, ok := RobinhoodSellDollarBudgetRule.ReadOrderOfCall(p, sent, service, "place_equity_order"); ok {
			t.Fatalf("service %q matched", service)
		}
	}
	for _, tool := range []string{"", "other_order", "second_order", "place_equity_order ", "place_еquity_order"} {
		if _, ok := RobinhoodSellDollarBudgetRule.ReadOrderOfCall(p, sent, "robinhood", tool); ok {
			t.Fatalf("tool %q matched", tool)
		}
	}
	for _, r := range []DollarBudgetRule{"", `{`, `{"service":"robinhood","tools":["place_equity_order"]}`} {
		if _, ok := r.ReadOrderOfCall(p, sent, "robinhood", "place_equity_order"); ok {
			t.Fatal("invalid policy matched")
		}
	}
}

func TestDollarFieldVocabularyUnchanged(t *testing.T) {
	for _, field := range []string{"dollar_amount", "usd", "cost_USD", "DOLLARS", "quantity", "credits", "count"} {
		if IsDollarField(field) != (FormatAmount(field, "1") == "$1") {
			t.Fatalf("vocabulary split: %s", field)
		}
	}
}
