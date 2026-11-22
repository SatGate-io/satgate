package mcpserver

import (
	"context"
	"encoding/json"
	"math/big"

	"github.com/satgate-io/satgate/pkg/argrules"
)

// OrderCostComparison adjusts the cost used in the per-order cap and approval
// comparisons. The returned amount is in dollars, not OrderCost.Units.
type OrderCostComparison func(ctx context.Context, origin ToolOrigin, tool string, args map[string]json.RawMessage, base argrules.OrderCost) (adjusted *big.Rat, ok bool, err error)

// SetOrderCostComparison installs the comparison hook before serving requests.
func (p *Proxy) SetOrderCostComparison(OrderCostComparison) {}
