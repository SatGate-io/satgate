package toolcontrols

import (
	"encoding/json"

	"github.com/satgate-io/satgate/pkg/argrules"
)

// DollarBudgetRule is immutable policy data, not an exemption authorization.
// Definition returns a fresh value so callers cannot mutate the policy.
type DollarBudgetRule string

// RobinhoodSellDollarBudgetRule identifies orders a trusted caller may exempt
// after proving authenticated route provenance and a closing-position bound.
const RobinhoodSellDollarBudgetRule DollarBudgetRule = `{"service":"robinhood","tools":["place_equity_order","place_crypto_order"],"side":"sell"}`

// DollarBudgetRuleDefinition is the public, machine-readable policy shape.
type DollarBudgetRuleDefinition struct {
	Service string   `json:"service"`
	Tools   []string `json:"tools"`
	Side    string   `json:"side"`
}

// Definition decodes policy data only, never call arguments. An invalid rule
// has an empty definition and cannot match an order.
func (r DollarBudgetRule) Definition() DollarBudgetRuleDefinition {
	var d DollarBudgetRuleDefinition
	if json.Unmarshal([]byte(r), &d) != nil {
		return DollarBudgetRuleDefinition{}
	}
	return d
}

// DollarBudgetOrder is a parsed order plus the router's service and original
// tool. Service alone is not proof of broker identity; the trusted hook must
// authenticate the route and prove the size cannot open or increase a short.
type DollarBudgetOrder struct {
	Service string
	Tool    string
	argrules.OrderArguments
}

// ReadOrderOfCall applies the policy through the shared strict argument reader.
// A match is only a candidate; the SpendGate hook must still authorize it.
func (r DollarBudgetRule) ReadOrderOfCall(params json.RawMessage, sentTool, service, originalTool string) (DollarBudgetOrder, bool) {
	d := r.Definition()
	if d.Service == "" || !argrules.ASCIIEqualFold(d.Service, service) {
		return DollarBudgetOrder{}, false
	}
	matched := false
	for _, tool := range d.Tools {
		if argrules.ASCIIEqualFold(tool, originalTool) {
			matched = true
			break
		}
	}
	if !matched {
		return DollarBudgetOrder{}, false
	}
	args, ok := argrules.ReadOrderArgumentsOfCall(params, sentTool)
	if !ok || d.Side == "" || !argrules.ASCIIEqualFold(d.Side, args.Side) {
		return DollarBudgetOrder{}, false
	}
	return DollarBudgetOrder{Service: service, Tool: originalTool, OrderArguments: args}, true
}
