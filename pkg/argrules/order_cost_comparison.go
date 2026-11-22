package argrules

import (
	"encoding/json"
	"math/big"
	"slices"
)

// OrderCostAdjustment is an optional, call-bound adjustment for comparing an
// already valued order. The adjusted amount is in dollars (not 10^-8 units).
// The callback receives a copy of the decoded arguments and cannot rewrite the
// call. Nil, ok=false, a non-nil error, a nil amount, a lower amount or overflow
// when converting to units all leave the base comparison unchanged.
type OrderCostAdjustment func(args map[string]json.RawMessage, base OrderCost) (adjusted *big.Rat, ok bool, err error)

func (a OrderCostAdjustment) comparisonUnits(args map[string]json.RawMessage, base OrderCost) int64 {
	if a == nil {
		return base.Units
	}
	copyOfArgs := make(map[string]json.RawMessage, len(args))
	for k, v := range args {
		copyOfArgs[k] = slices.Clone(v)
	}
	adjusted, ok, err := a(copyOfArgs, base)
	if err != nil || !ok || adjusted == nil {
		return base.Units
	}
	units, ok := UnitsCeil(adjusted)
	if !ok || units < base.Units {
		return base.Units
	}
	return units
}

// CompareOrderCostOfCall values a call as OrderMaxCostOfCall does and also
// returns units for the approval comparison. Use the unadjusted OrderCost for
// any purpose other than that comparison. A call that cannot be valued never
// reaches adjust.
func CompareOrderCostOfCall(params json.RawMessage, tool string, adjust OrderCostAdjustment) (base OrderCost, comparisonUnits int64, why OrderCostWhy) {
	if adjust == nil {
		base, why = OrderMaxCostOfCall(params, tool, tool)
		return base, base.Units, why
	}
	args, ok := orderArgumentsOfCall(params, tool)
	if !ok {
		return OrderCost{}, 0, WhyUnreadable
	}
	base, why = OrderMaxCost(tool, args)
	if !why.Valued() {
		return base, base.Units, why
	}
	return base, adjust.comparisonUnits(args, base), why
}
