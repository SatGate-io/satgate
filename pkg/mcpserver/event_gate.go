package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// Event contract orders (see argrules event_order.go and toolcontrols
// event.go).
//
// An event contract order names a market, never its event, series or
// category. The EventGate asks the market lookup the host program registered
// for the order's value schema (RegisterEventMarketResolver) and checks, in
// this order, before the call is forwarded:
//
//  1. the order is valued (argrules.EventOrderMaxCostOfCall) under the one
//     schema every event control on the token names for the tool;
//  2. the tool's service is the one bound to that schema (a schema's prices
//     mean something only on its own service);
//  3. the market is found, has an event, has two outcomes, a contract on it
//     is worth at most 1, it is not a combination of markets, and it is open
//     for trading;
//  4. every event_markets entry that covers the tool allows the market's
//     series or category;
//  5. every event_stakes entry that covers the tool has room for the order on
//     the market's event, and reserves it.
//
// A lookup that errors or times out, a store that cannot answer, and a store
// that cannot hold per-event totals refuse the call (the proxy's "a limit on
// this token could not be checked"). Nothing is forwarded and nothing is
// counted. An order that can only close a position is checked by 1 to 4 and
// is not counted by 5.
//
// The reservation is given back by the SpendGate's rule: only when SatGate
// knows the call did not run. Totals are never reset by time; the host
// program resets them (toolcontrols.EventStakeStore ResetEvents, with the key
// from EventStakeKey).
//
// Refusals use fixed phrases. They never carry the market, the event, the
// series or the category: a category on the agent's screen could be used to
// probe an allow list.
//
// Put the EventGate first in ChainGates, before the SpendGate and the
// ApprovalGate: a call whose market is not allowed is refused before any
// spending limit is reserved or any owner is asked.

// EventMarket is what a market lookup says about one market.
type EventMarket struct {
	// Event is the id of the event the market belongs to. Every market of
	// one event shares one per-event total.
	Event string
	// Series and Category are what event_markets allow lists compare.
	Series   string
	Category string
	// Binary: the market has exactly two outcomes.
	Binary bool
	// MaxValueOne: a contract on the market settles at no more than 1.
	MaxValueOne bool
	// Combo: the market is a combination of other markets.
	Combo bool
	// Tradable: the market accepts new orders now.
	Tradable bool
	// ClosesAt, when set, is when the market stops trading.
	ClosesAt time.Time
}

// EventMarketResolver answers for a market id under one value schema. It must
// read only data the agent cannot influence (the service's own market data).
// It returns ErrEventMarketNotFound for a market that does not exist; any
// other error means it could not answer.
type EventMarketResolver interface {
	ResolveEventMarket(ctx context.Context, tenantID, schema, market string) (EventMarket, error)
}

// ErrEventMarketNotFound is what a resolver returns for an unknown market.
var ErrEventMarketNotFound = errors.New("event market not found")

var (
	eventResolversMu sync.RWMutex
	eventResolvers   = map[string]EventMarketResolver{}
)

// RegisterEventMarketResolver makes r the market lookup for schema. The
// schema must already be registered (argrules.RegisterEventSchema), and a
// schema gets one resolver.
func RegisterEventMarketResolver(schema string, r EventMarketResolver) error {
	if !argrules.EventSchemaRegistered(schema) {
		return fmt.Errorf("event market lookup: value schema %q is not registered", schema)
	}
	if r == nil {
		return fmt.Errorf("event market lookup for %q: nil resolver", schema)
	}
	eventResolversMu.Lock()
	defer eventResolversMu.Unlock()
	if _, dup := eventResolvers[schema]; dup {
		return fmt.Errorf("event market lookup for %q: already registered", schema)
	}
	eventResolvers[schema] = r
	return nil
}

// EventMarketResolverFor returns the registered lookup for schema, or nil.
func EventMarketResolverFor(schema string) EventMarketResolver {
	eventResolversMu.RLock()
	defer eventResolversMu.RUnlock()
	return eventResolvers[schema]
}

// DefaultEventLookupTimeout bounds one market lookup.
const DefaultEventLookupTimeout = 5 * time.Second

// EventGate is the CallGate behind event contract controls.
type EventGate struct {
	store toolcontrols.SpendStore

	// BoundUpstream says which service (ToolOrigin.Service) a schema's
	// orders must go to, for a tenant. A schema with no binding refuses
	// every call valued under it.
	BoundUpstream func(ctx context.Context, tenantID, schema string) (string, bool)

	// CallScope narrows "the same call", as SpendGate.CallScope does.
	CallScope func(ctx context.Context) string

	// LookupTimeout bounds one market lookup. Zero means
	// DefaultEventLookupTimeout.
	LookupTimeout time.Duration

	now func() time.Time
}

// NewEventGate returns a gate over store, with each schema bound to the
// service named in upstreams. A store that does not implement
// toolcontrols.EventStakeStore refuses every call a stake limit covers.
func NewEventGate(store toolcontrols.SpendStore, upstreams map[string]string) *EventGate {
	bound := make(map[string]string, len(upstreams))
	for k, v := range upstreams {
		bound[k] = v
	}
	return &EventGate{
		store: store,
		BoundUpstream: func(_ context.Context, _ string, schema string) (string, bool) {
			u, ok := bound[schema]
			return u, ok && u != ""
		},
	}
}

// EnforcesEventControls reports that this gate enforces event controls (see
// EventControlGate).
func (g *EventGate) EnforcesEventControls() bool { return g != nil }

// EventControlGate is implemented by a gate that enforces event contract
// controls. The proxy refuses a call that an event control covers when its
// gate does not enforce them, so a gateway that knows the keys but has no
// EventGate fails closed instead of skipping the controls.
type EventControlGate interface {
	EnforcesEventControls() bool
}

// EnforcesEventControls reports whether any gate in the chain does.
func (c chainGate) EnforcesEventControls() bool {
	for _, g := range c {
		if gateEnforcesEventControls(g) {
			return true
		}
	}
	return false
}

func gateEnforcesEventControls(g CallGate) bool {
	e, ok := g.(EventControlGate)
	return ok && e.EnforcesEventControls()
}

// EventStakeKey names the counter of one stake limit for a tenant. A host
// program passes it to ResetEvents.
func EventStakeKey(tenantID string, l toolcontrols.EventStakeLimit) string {
	return "satgate:eventstake:" + tenantID + ":" + l.ID
}

// eventControls is every event control on a token, and the ones that cover
// one tool.
type eventControls struct {
	any     bool // the token carries an event control of any kind
	schemas []string
	markets []toolcontrols.EventMarketRule
	stakes  []toolcontrols.EventStakeLimit
}

// collectEventControls reads the token's event controls. A token whose
// controls do not read returns an error, and the call is refused.
func collectEventControls(t *TokenInfo, tool string) (eventControls, error) {
	var out eventControls
	add := func(s string) {
		for _, x := range out.schemas {
			if x == s {
				return
			}
		}
		out.schemas = append(out.schemas, s)
	}
	ruleSchemas, err := t.eventOrderRuleSchemas(tool)
	if err != nil {
		return out, err
	}
	for _, s := range ruleSchemas {
		add(s)
	}
	limits, err := t.AllSpendLimits()
	if err != nil {
		return out, err
	}
	for _, l := range limits {
		if l.Value != argrules.EventCostBasis {
			continue
		}
		out.any = true
		if l.Covers(tool) {
			add(l.Schema)
		}
	}
	approvals, err := t.AllApprovalRules()
	if err != nil {
		return out, err
	}
	for _, r := range approvals {
		if r.Value != argrules.EventCostBasis {
			continue
		}
		out.any = true
		if r.Covers(tool) {
			add(r.Schema)
		}
	}
	markets, err := t.AllEventMarkets()
	if err != nil {
		return out, err
	}
	for _, r := range markets {
		out.any = true
		if r.Covers(tool) {
			out.markets = append(out.markets, r)
			add(r.Schema)
		}
	}
	stakes, err := t.AllEventStakes()
	if err != nil {
		return out, err
	}
	for _, l := range stakes {
		out.any = true
		if l.Covers(tool) {
			out.stakes = append(out.stakes, l)
			add(l.Schema)
		}
	}
	if len(out.schemas) > 0 {
		out.any = true
	}
	return out, nil
}

// Fixed refusal phrases. None carries a value from the call or the lookup.
const (
	msgEventShape        = "this token has event contract limits, and this call's tool name or layout cannot be matched exactly to them, so the call was not sent"
	msgEventTwoSchemas   = "this token values this tool under more than one value schema, so the call was not sent"
	msgEventNotBound     = "this order's value schema is not bound to the service this tool comes from, so the call was not sent"
	msgEventNoLookup     = "markets for this order's value schema cannot be looked up here, so the call was not sent"
	msgEventNotFound     = "this order's market could not be found, so the call was not sent"
	msgEventNoEvent      = "this order's market has no event, so the call was not sent"
	msgEventNotBinary    = "this order's market is not a two-outcome market, so the call was not sent"
	msgEventNotMaxOne    = "a contract on this order's market can be worth more than 1, so the call was not sent"
	msgEventCombo        = "this order's market combines other markets, so the call was not sent"
	msgEventNotOpen      = "this order's market is not open for trading, so the call was not sent"
	msgEventNotAllowed   = "this order's market is not in a category or series this token allows, so it was not sent"
	msgEventStakesFull   = "this token's per-event totals hold too many events to count another; they must be reset first, so the call was not sent"
	msgEventStakeOverFmt = "this token's limit is %s in total on any one event, counting each order at the most it can cost; this order would go over it"
)

// Admit implements CallGate.
func (g *EventGate) Admit(ctx context.Context, call GateCall) (*GateAdmission, error) {
	ctl, err := collectEventControls(call.Token, call.Tool)
	if err != nil {
		return nil, fmt.Errorf("token controls: %w", err)
	}
	if !ctl.any {
		return nil, nil
	}
	// The token carries an event control, so the name this gate matches must
	// be the name the upstream reads (the rule SpendGate applies).
	if !argrules.ValidToolName(call.Tool) || argrules.CheckCallShape(call.Params, call.Tool) != argrules.NumericFieldOK {
		return &GateAdmission{Refusal: eventSpendRefusal(call.Tool, msgEventShape, "")}, nil
	}
	if len(ctl.schemas) == 0 {
		return nil, nil
	}
	if len(ctl.schemas) > 1 {
		return &GateAdmission{Refusal: eventSpendRefusal(call.Tool, msgEventTwoSchemas, "")}, nil
	}
	schema := ctl.schemas[0]

	// 1. Value the order, with the one function every event control uses.
	v, why := argrules.EventOrderMaxCostOfCall(call.Params, call.Tool, schema)
	if !why.Valued() {
		return &GateAdmission{Refusal: eventSpendRefusal(call.Tool, valueRefusal(call.Tool, why), "")}, nil
	}

	// 2. The service the tool comes from must be the one bound to the schema.
	if call.Origin.Err != nil {
		return nil, fmt.Errorf("event controls: the tool's service could not be confirmed: %w", call.Origin.Err)
	}
	tenant := ""
	if call.Token != nil {
		tenant = call.Token.TenantID
	}
	bound, ok := "", false
	if g != nil && g.BoundUpstream != nil {
		bound, ok = g.BoundUpstream(ctx, tenant, schema)
	}
	if !ok || !call.Origin.Listed || call.Origin.Service != bound {
		return &GateAdmission{Refusal: eventMarketRefusal(call.Tool, msgEventNotBound)}, nil
	}

	// 3. Look the market up.
	resolver := EventMarketResolverFor(schema)
	if resolver == nil {
		return &GateAdmission{Refusal: eventMarketRefusal(call.Tool, msgEventNoLookup)}, nil
	}
	timeout := g.LookupTimeout
	if timeout <= 0 {
		timeout = DefaultEventLookupTimeout
	}
	lctx, cancel := context.WithTimeout(ctx, timeout)
	m, err := resolver.ResolveEventMarket(lctx, tenant, schema, v.Market)
	// An answer that comes after the lookup's deadline, or after its context
	// is done, is not used, even when the lookup returned no error.
	late := lctx.Err()
	if dl, ok := lctx.Deadline(); late == nil && ok && !time.Now().Before(dl) {
		late = context.DeadlineExceeded
	}
	cancel()
	if late != nil {
		return nil, fmt.Errorf("event market lookup did not answer in time: %w", late)
	}
	switch {
	case errors.Is(err, ErrEventMarketNotFound):
		return &GateAdmission{Refusal: eventMarketRefusal(call.Tool, msgEventNotFound)}, nil
	case err != nil:
		return nil, fmt.Errorf("event market lookup failed: %w", err)
	}
	now := call.Now
	if g.now != nil {
		now = g.now()
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}
	switch {
	case m.Event == "":
		return &GateAdmission{Refusal: eventMarketRefusal(call.Tool, msgEventNoEvent)}, nil
	case !m.Binary:
		return &GateAdmission{Refusal: eventMarketRefusal(call.Tool, msgEventNotBinary)}, nil
	case !m.MaxValueOne:
		return &GateAdmission{Refusal: eventMarketRefusal(call.Tool, msgEventNotMaxOne)}, nil
	case m.Combo:
		return &GateAdmission{Refusal: eventMarketRefusal(call.Tool, msgEventCombo)}, nil
	case !m.Tradable || (!m.ClosesAt.IsZero() && !now.Before(m.ClosesAt)):
		return &GateAdmission{Refusal: eventMarketRefusal(call.Tool, msgEventNotOpen)}, nil
	}

	// 4. Allow lists: every entry must pass.
	for _, r := range ctl.markets {
		if !r.Allows(m.Series, m.Category) {
			return &GateAdmission{Refusal: eventMarketRefusal(call.Tool, msgEventNotAllowed)}, nil
		}
	}

	// 5. Per-event stakes. A closing order is not counted.
	if len(ctl.stakes) == 0 || v.Closing {
		return nil, nil
	}
	store, ok := g.store.(toolcontrols.EventStakeStore)
	if g.store == nil || !ok {
		return nil, errors.New("event controls: the store cannot hold per-event totals")
	}
	callID := (&SpendGate{CallScope: g.CallScope}).callID(ctx, call)
	var taken []toolcontrols.EventReservation
	giveBack := func() {
		for i := len(taken) - 1; i >= 0; i-- {
			g.release(ctx, store, taken[i])
		}
	}
	for _, l := range ctl.stakes {
		if l.ID == "" {
			giveBack()
			return nil, errors.New("an event stake limit has no counter id")
		}
		res := toolcontrols.EventReservation{
			Key:    EventStakeKey(tenant, l),
			Event:  m.Event,
			CallID: callID,
			Units:  v.Units,
			Max:    l.MaxUnits(),
		}
		out, err := store.ReserveEvent(ctx, res)
		if err != nil {
			giveBack()
			return nil, fmt.Errorf("event stake store: %w", err)
		}
		switch out.Status {
		case toolcontrols.OverLimit:
			giveBack()
			return &GateAdmission{Refusal: eventSpendRefusal(call.Tool,
				fmt.Sprintf("%s: "+msgEventStakeOverFmt, call.Tool, toolcontrols.FormatAmount(toolcontrols.OrderCostField, l.Max)), l.Max)}, nil
		case toolcontrols.EventsFull:
			giveBack()
			return &GateAdmission{Refusal: eventSpendRefusal(call.Tool, call.Tool+": "+msgEventStakesFull, l.Max)}, nil
		case toolcontrols.Duplicate:
			giveBack()
			return &GateAdmission{Refusal: &GateRefusal{
				Code:    DenialDuplicateRequest,
				Message: fmt.Sprintf("%s: this request id was already used for the same call; send it with a new id to try again", call.Tool),
			}}, nil
		case toolcontrols.Reserved:
			if out.Reservation == "" {
				giveBack()
				return nil, errors.New("event stake store returned no reservation identity")
			}
		default:
			giveBack()
			return nil, errors.New("event stake store returned an unknown status")
		}
		// A give-back releases this reservation and no other.
		res.Reservation = out.Reservation
		taken = append(taken, res)
	}
	// The admission carries no receipt detail: the event, its total and a
	// window named for it are not receipt fields in this version, and an
	// admitted call's detail is the SpendGate's.
	return &GateAdmission{
		Settle: func(ctx context.Context, o GateOutcome) {
			if o.Succeeded {
				return
			}
			if o.Unknown {
				log.Warn().Str("tool", call.Tool).Int("limits", len(taken)).
					Msg("event stake: outcome unknown; reservation kept")
				return
			}
			giveBack()
		},
	}, nil
}

func (g *EventGate) release(ctx context.Context, store toolcontrols.EventStakeStore, r toolcontrols.EventReservation) {
	rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 5*time.Second)
	defer cancel()
	if err := store.ReleaseEvent(rctx, r); err != nil {
		log.Error().Err(err).Str("counter", logCounter(r.Key)).Msg("could not release an event stake reservation; the event total stays counted")
	}
}

// eventSpendRefusal is a TOOL_SPEND_LIMIT_DENIED refusal from the event gate.
// limit is the owner's maximum when a stake limit refused, else "".
func eventSpendRefusal(tool, message, limit string) *GateRefusal {
	if !strings.HasPrefix(message, tool+": ") {
		message = tool + ": " + message
	}
	r := &GateRefusal{
		Code:    toolcontrols.DenialCodeSpendLimit,
		Message: message,
		Data:    map[string]interface{}{},
		Detail:  GateReceiptDetail{DenialCode: toolcontrols.DenialCodeSpendLimit},
	}
	if limit != "" {
		r.Data["limit"] = limit
		r.Detail.Limit = limit
	}
	return r
}

// eventMarketRefusal is an EVENT_MARKET_NOT_ALLOWED refusal.
func eventMarketRefusal(tool, message string) *GateRefusal {
	return &GateRefusal{
		Code:    toolcontrols.DenialCodeEventMarket,
		Message: tool + ": " + message,
		Detail:  GateReceiptDetail{DenialCode: toolcontrols.DenialCodeEventMarket},
	}
}
