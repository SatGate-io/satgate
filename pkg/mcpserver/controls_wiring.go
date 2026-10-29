package mcpserver

import (
	"fmt"
	"strings"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// ControlGateFromConfig builds the call gate a self-hosted gateway installs
// from cfg.Controls. It returns nil when controls.store is not set: no gate
// is installed and a token that carries a control is refused on every call,
// as before.
//
// With controls.store memory the gate is ChainGates(event, spend, approval)
// over one MemoryStore:
//
//   - the EventGate binds each controls.eventContracts schema to its
//     upstream and looks markets up through the resolver the program
//     registered for the schema;
//   - the SpendGate enforces day and week limits (a shared budget is refused:
//     there is no saved budget to check it against);
//   - the ApprovalGate has no approval store, so a call an "ask me" rule
//     would hold is refused (there is no owner page to approve it on).
//
// Every controls.eventContracts entry must name a value schema the program
// registered (argrules.RegisterEventSchema) with a market lookup
// (RegisterEventMarketResolver); otherwise it returns an error and the
// gateway must not start. satgate-mcp as built from this repository
// registers no schema.
func ControlGateFromConfig(cfg *Config) (CallGate, error) {
	if cfg == nil || cfg.Controls.Store == "" {
		if cfg != nil && len(cfg.Controls.EventContracts) > 0 {
			return nil, fmt.Errorf("controls.eventContracts needs controls.store: memory")
		}
		return nil, nil
	}
	if err := cfg.Controls.validate(cfg.Upstreams); err != nil {
		return nil, err
	}
	bound := map[string]string{}
	for _, e := range cfg.Controls.EventContracts {
		if !argrules.EventSchemaRegistered(e.Schema) {
			registered := strings.Join(argrules.EventSchemaNames(), ", ")
			if registered == "" {
				registered = "none"
			}
			return nil, fmt.Errorf("controls.eventContracts: value schema %q is not registered by this program (registered: %s); event contract limits need a host program that registers its value schemas, so the gateway was not started", e.Schema, registered)
		}
		if EventMarketResolverFor(e.Schema) == nil {
			return nil, fmt.Errorf("controls.eventContracts: value schema %q has no market lookup registered by this program, so the gateway was not started", e.Schema)
		}
		bound[e.Schema] = e.Upstream
	}
	store := toolcontrols.NewMemoryStore(nil)
	return ChainGates(NewEventGate(store, bound), NewSpendGate(store), NewApprovalGate(nil, nil)), nil
}
