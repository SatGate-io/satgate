package mcpserver

import (
	"context"
	"testing"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// No Reserve or Release method is supplied by this store: invoking either
// would panic. The unreadable first limit must not suppress the later two.
type overageReadStore struct{ toolcontrols.SpendStore }

func (overageReadStore) Total(context.Context, string) (int64, error) { return 0, nil }

func TestSpendGateReadOveragesDoesNotReserve(t *testing.T) {
	info := ocToken(t, `{"v":1,"spend_limits":[
		{"field":"missing","window":"day","max":"1","tools":["place_crypto_order"]},
		{"field":"dollar_amount","window":"day","max":"50","tools":["place_crypto_order"],"value":"order_max_cost"},
		{"field":"dollar_amount","window":"week","max":"55","tools":["place_crypto_order"],"value":"order_max_cost"}
	]}`)
	g := newTestSpendGate(overageReadStore{})
	call := GateCall{Token: info, Tool: "place_crypto_order", Now: ocNow,
		Params: ocParams("place_crypto_order", `{"symbol":"BTC","side":"buy","type":"limit","time_in_force":"gtc","dollar_amount":"60","limit_price":"30000"}`)}
	over := g.ReadOverages(context.Background(), call)
	if len(over) != 2 {
		t.Fatalf("read %d overages, want 2", len(over))
	}
	windows := map[any]bool{}
	for _, r := range over {
		windows[r.Data["window"]] = true
		if r.Data["cost"] != "60.01" || r.SpendLimitID == "" {
			t.Errorf("missing verified overage facts: window=%s cost=%s", r.Data["window"], r.Data["cost"])
		}
	}
	if !windows["day"] || !windows["week"] {
		t.Error("expected both day and week overages")
	}
}
