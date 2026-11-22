package mcpserver

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/satgate-io/satgate/pkg/argrules"
)

// OrderCostComparison is a host hook used only for order.max_cost and
// order_max_cost approval comparisons, after the call has been valued. Origin
// is the same ToolOrigin used in GateCall.Origin: its Service and Original
// come from the router's tool-name table, not from a name supplied by the
// caller. An unlisted origin or one with Err set is not a verified route.
//
// Args is a detached copy of the arguments validated for valuation. Return an
// exact amount in dollars, not OrderCost.Units; it is rounded up to 10^-8
// units. A nil hook, ok=false, an error, a nil amount, a lower cost or an
// amount that overflows when converted to units leaves the base comparison
// unchanged. The hook may be called more than once for a call and concurrently
// for different calls; it should have no side effects.
type OrderCostComparison func(ctx context.Context, origin ToolOrigin, tool string, args map[string]json.RawMessage, base argrules.OrderCost) (adjusted *big.Rat, ok bool, err error)

// SetOrderCostComparison installs the per-order comparison hook. Call it
// before the proxy serves requests. Nil uses the base cost. Hosts can set the
// same hook on ApprovalGate for the approval comparison.
func (p *Proxy) SetOrderCostComparison(hook OrderCostComparison) { p.orderCostComparison = hook }

// ForTool binds the hook to a call. Hosts checking a call outside the proxy can
// pass the result to CheckToolArgumentsWithOrderCostComparison, supplying the
// ToolOrigin from the same route lookup they use to evaluate the call.
func (h OrderCostComparison) ForTool(ctx context.Context, origin ToolOrigin, tool string) argrules.OrderCostAdjustment {
	if h == nil {
		return nil
	}
	return func(args map[string]json.RawMessage, base argrules.OrderCost) (*big.Rat, bool, error) {
		return h(ctx, origin, tool, args, base)
	}
}
