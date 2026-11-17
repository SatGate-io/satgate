package argrules

import (
	"bytes"
	"encoding/json"
	"strings"
)

// OrderArguments is the exact size and symbol supplied to a trusted classifier.
// Exactly one size is non-nil. This is not a valuation or a holdings decision.
type OrderArguments struct {
	Side         string
	Symbol       string
	Quantity     *Decimal
	DollarAmount *Decimal
}

// ASCIIEqualFold matches only ASCII text, without trimming or Unicode folding.
func ASCIIEqualFold(a, b string) bool {
	for i := 0; i < len(a); i++ {
		if a[i] >= 0x80 {
			return false
		}
	}
	return inFoldedList([]string{asciiLower(a)}, b)
}

// ReadOrderArgumentsOfCall shares valuation's strict call and string readers.
// Ambiguous keys, non-ASCII sides, whitespace, and absent/unreadable sizes or
// symbols cannot reach a classifier. It does not change OrderMaxCost semantics.
func ReadOrderArgumentsOfCall(params json.RawMessage, sentTool string) (OrderArguments, bool) {
	args, ok := orderArgumentsOfCall(params, sentTool)
	if !ok {
		return OrderArguments{}, false
	}
	for k := range args {
		if k != "symbol" && strings.EqualFold(k, "symbol") {
			return OrderArguments{}, false
		}
		for _, f := range orderCostFields {
			if k != f && strings.EqualFold(k, f) {
				return OrderArguments{}, false
			}
		}
	}
	side, ok := orderString(args, "side")
	if !ok || side == "" || strings.TrimSpace(side) != side || !ASCIIEqualFold(side, side) {
		return OrderArguments{}, false
	}
	symbol, ok := orderString(args, "symbol")
	if !ok || symbol == "" || strings.TrimSpace(symbol) != symbol || strings.ContainsRune(symbol, '\ufffd') {
		return OrderArguments{}, false
	}
	dollars, hasDollars := args["dollar_amount"]
	quantity, hasQuantity := args["quantity"]
	if hasDollars == hasQuantity {
		return OrderArguments{}, false
	}
	raw := quantity
	if hasDollars {
		raw = dollars
	}
	size, ok := argumentDecimal(bytes.TrimSpace(raw))
	if !ok || size.rat.Sign() <= 0 {
		return OrderArguments{}, false
	}
	out := OrderArguments{Side: asciiLower(side), Symbol: symbol}
	if hasDollars {
		out.DollarAmount = size
	} else {
		out.Quantity = size
	}
	return out, true
}
