package mcpserver

import (
	"context"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// newTestSpendGate is the spend gate with a validator that honours every
// sealed reference: the tests that are about counting, not about the owner's
// saved record, use it. The tests of the validator itself (round 3) build the
// gate with NewSpendGate and set their own.
func newTestSpendGate(store toolcontrols.SpendStore) *SpendGate {
	g := NewSpendGate(store)
	g.SharedBudgetValidator = func(context.Context, string, toolcontrols.SpendLimit) error { return nil }
	return g
}
