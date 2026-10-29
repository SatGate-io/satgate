package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// ---- A made-up value schema and market lookup, for tests only ----------------
//
// The schema reads an order on an imaginary service: buy or sell YES
// contracts, an optional price, a time in force. Selling YES at p opens the
// other side at 1 - p; with no price the order is valued at 1 per contract;
// reduce_only with ioc can only close a position.

var gateSchemaFields = []string{"market", "action", "count", "price", "tif", "reduce_only", "sizing", "amount", "replaces", "orders"}

func gateSchemaRead(args map[string]json.RawMessage) (argrules.EventOrder, argrules.OrderCostWhy) {
	if _, ok := args["orders"]; ok {
		return argrules.EventOrder{}, argrules.WhyEventBatch
	}
	if _, ok := args["replaces"]; ok {
		return argrules.EventOrder{}, argrules.WhyEventAmendOrReplace
	}
	if _, ok := args["amount"]; ok {
		return argrules.EventOrder{}, argrules.WhyEventByAmount
	}
	if _, ok := args["sizing"]; ok {
		return argrules.EventOrder{}, argrules.WhyEventByAmount
	}
	tif, _ := argrules.ReadArgString(args["tif"])
	switch tif {
	case "ioc", "fok", "day":
	case "gtc":
		return argrules.EventOrder{}, argrules.WhyEventResting
	default:
		return argrules.EventOrder{}, argrules.WhyUnreadable
	}
	var o argrules.EventOrder
	o.Market, _ = argrules.ReadArgString(args["market"])
	c, ok := argrules.ReadArgDecimal(args["count"])
	if !ok {
		return argrules.EventOrder{}, argrules.WhyEventBadCount
	}
	o.Contracts = c
	if raw, ok := args["reduce_only"]; ok {
		if b, _ := argrules.ReadArgBool(raw); b {
			if tif != "ioc" {
				return argrules.EventOrder{}, argrules.WhyEventClosingUnsure
			}
			o.Closing = true
			return o, argrules.OrderCostValued
		}
	}
	var price *big.Rat
	if raw, ok := args["price"]; ok {
		p, ok := argrules.ReadArgDecimal(raw)
		if !ok {
			return argrules.EventOrder{}, argrules.WhyBadNumber
		}
		price = p
	}
	switch action, _ := argrules.ReadArgString(args["action"]); action {
	case "buy":
		o.Price = price
	case "sell":
		if price != nil {
			o.Price = new(big.Rat).Sub(big.NewRat(1, 1), price)
		}
	default:
		return argrules.EventOrder{}, argrules.WhyUnreadable
	}
	return o, argrules.OrderCostValued
}

type fakeMarkets struct {
	mu      sync.Mutex
	markets map[string]EventMarket
	err     error
	delay   time.Duration
	calls   int
}

func (f *fakeMarkets) ResolveEventMarket(ctx context.Context, _, _, market string) (EventMarket, error) {
	f.mu.Lock()
	f.calls++
	m, ok := f.markets[market]
	err, delay := f.err, f.delay
	f.mu.Unlock()
	if delay > 0 {
		select {
		case <-ctx.Done():
			return EventMarket{}, ctx.Err()
		case <-time.After(delay):
		}
	}
	if err != nil {
		return EventMarket{}, err
	}
	if !ok {
		return EventMarket{}, ErrEventMarketNotFound
	}
	return m, nil
}

func (f *fakeMarkets) lookups() int {
	f.mu.Lock()
	defer f.mu.Unlock()
	return f.calls
}

func testMarkets() map[string]EventMarket {
	open := func(ev, ser, cat string) EventMarket {
		return EventMarket{Event: ev, Series: ser, Category: cat, Binary: true, MaxValueOne: true, Tradable: true}
	}
	m := map[string]EventMarket{
		"MKT-A1": open("EV-A", "SER-A", "Economics"),
		"MKT-A2": open("EV-A", "SER-A", "Economics"),
		"MKT-B1": open("EV-B", "SER-B", " Sports "),
		"MKT-C1": open("EV-C", "SER-C", "Financials"),
	}
	nb := open("EV-X", "SER-A", "ECONOMICS")
	nb.Binary = false
	m["MKT-NB"] = nb
	big := open("EV-X", "SER-A", "ECONOMICS")
	big.MaxValueOne = false
	m["MKT-BIG"] = big
	combo := open("EV-X", "SER-A", "ECONOMICS")
	combo.Combo = true
	m["MKT-COMBO"] = combo
	closed := open("EV-X", "SER-A", "ECONOMICS")
	closed.Tradable = false
	m["MKT-CLOSED"] = closed
	past := open("EV-X", "SER-A", "ECONOMICS")
	past.ClosesAt = time.Now().Add(-time.Hour)
	m["MKT-PAST"] = past
	m["MKT-NOEVENT"] = open("", "SER-A", "ECONOMICS")
	return m
}

var gateSchemaSeq int64

// registerGateSchema registers a fresh made-up schema (and, with lookup, a
// market lookup for it) under a name no other test uses.
func registerGateSchema(t *testing.T, lookup bool) (string, *fakeMarkets) {
	t.Helper()
	name := fmt.Sprintf("gatetest_%d_v1", atomic.AddInt64(&gateSchemaSeq, 1))
	if err := argrules.RegisterEventSchema(argrules.EventSchema{Name: name, Fields: gateSchemaFields, Read: gateSchemaRead}); err != nil {
		t.Fatal(err)
	}
	res := &fakeMarkets{markets: testMarkets()}
	if lookup {
		if err := RegisterEventMarketResolver(name, res); err != nil {
			t.Fatal(err)
		}
	}
	return name, res
}

// eventOriginRouter is a scripted upstream whose tool-name table says every tool
// comes from service.
type eventOriginRouter struct {
	*scriptedRouter
	mu      sync.Mutex
	service string
	err     error
}

func (r *eventOriginRouter) ToolOriginForTenant(_ context.Context, _, tool string) ToolOrigin {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.err != nil {
		return ToolOrigin{Err: r.err}
	}
	return ToolOrigin{Listed: true, Service: r.service, Original: tool}
}

// ---- Harness ---------------------------------------------------------------

type evOpts struct {
	gate      func(store *toolcontrols.MemoryStore, appr *toolcontrols.MemoryApprovalStore, schema string) CallGate
	noLookup  bool
	scheduled []func() (*Response, error)
}

type evHarness struct {
	proxy  *Proxy
	rr     *eventOriginRouter
	store  *toolcontrols.MemoryStore
	appr   *toolcontrols.MemoryApprovalStore
	event  *EventGate
	schema string
	res    *fakeMarkets
	token  string
	ctl    *toolcontrols.Doc
}

func newEvProxy(t *testing.T) *Proxy {
	t.Helper()
	proxy, err := New(&Config{
		Server:      ServerConfig{Transport: "sse", Name: "event-test", Version: "test"},
		Auth:        AuthConfig{Mode: "header", RootKey: argTestRoot},
		Budget:      BudgetConfig{Limit: 1000000, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	})
	if err != nil {
		t.Fatal(err)
	}
	return proxy
}

// newEvHarness builds a proxy with ChainGates(event, spend, approval) and a
// token carrying ctlDoc and ruleDoc ("SCHEMA" in them is the test schema).
func newEvHarness(t *testing.T, ctlDoc, ruleDoc string, o *evOpts) *evHarness {
	t.Helper()
	if o == nil {
		o = &evOpts{}
	}
	h := &evHarness{}
	h.schema, h.res = registerGateSchema(t, !o.noLookup)
	h.proxy = newEvProxy(t)
	h.rr = &eventOriginRouter{scriptedRouter: &scriptedRouter{replies: o.scheduled}, service: "venue"}
	h.proxy.SetUpstreamRouter(h.rr)
	h.store = toolcontrols.NewMemoryStore(nil)
	h.appr = toolcontrols.NewMemoryApprovalStore()
	if o.gate != nil {
		h.proxy.SetCallGate(o.gate(h.store, h.appr, h.schema))
	} else {
		h.event = NewEventGate(h.store, map[string]string{h.schema: "venue"})
		ag := NewApprovalGate(h.appr, nil)
		ag.Synchronous = true
		h.proxy.SetCallGate(ChainGates(h.event, newTestSpendGate(h.store), ag))
	}
	svc, tok := mintBudgetToken(t, argTestRoot, "place_event_order,cancel_order", "tenant-1", "budget-ev", "1000000")
	if ctlDoc != "" {
		doc := strings.ReplaceAll(ctlDoc, "SCHEMA", h.schema)
		word := controlWord(t, doc)
		d, err := toolcontrols.ParseScopeValue(strings.TrimPrefix(word, "scope = "))
		if err != nil {
			t.Fatal(err)
		}
		h.ctl = d
		tok = appendCaveat(t, svc, tok, word)
	}
	if ruleDoc != "" {
		tok = appendCaveat(t, svc, tok, ruleScopeWord(t, strings.ReplaceAll(ruleDoc, "SCHEMA", h.schema)))
	}
	h.token = tok
	return h
}

var evSeq int64

func (h *evHarness) callTool(t *testing.T, tool, args string) *Response {
	t.Helper()
	resp, err := h.proxy.handleRequest(context.Background(), argCall(int(atomic.AddInt64(&evSeq, 1)), h.token, tool, args))
	if err != nil || resp == nil {
		t.Fatalf("handleRequest: %v %v", resp, err)
	}
	return resp
}

func (h *evHarness) call(t *testing.T, args string) *Response {
	t.Helper()
	return h.callTool(t, "place_event_order", args)
}

// stake is the total of stake limit i on event.
func (h *evHarness) stake(t *testing.T, i int, event string) int64 {
	t.Helper()
	n, err := h.store.EventTotal(context.Background(), EventStakeKey("tenant-1", h.ctl.EventStakes[i]), event)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// spent is the sum of every spending-limit counter.
func (h *evHarness) spent() int64 {
	var total int64
	for _, k := range h.store.Keys() {
		total += h.store.TotalOf(k)
	}
	return total
}

func code(t *testing.T, r *Response) string {
	t.Helper()
	if r.Error == nil {
		return ""
	}
	var d map[string]any
	_ = json.Unmarshal(r.Error.Data, &d)
	s, _ := d["error"].(string)
	return s
}

func evBuy(market, count, price string) string {
	s := `{"market":"` + market + `","action":"buy","count":"` + count + `","tif":"ioc"`
	if price != "" {
		s += `,"price":"` + price + `"`
	}
	return s + `}`
}

func evSell(market, count, price string) string {
	return strings.Replace(evBuy(market, count, price), `"buy"`, `"sell"`, 1)
}

func evClose(market, count string) string {
	return `{"market":"` + market + `","action":"sell","count":"` + count + `","tif":"ioc","reduce_only":true}`
}

const (
	stakeDoc   = `{"v":1,"event_stakes":[{"tools":["place_event_order"],"schema":"SCHEMA","max":"%s","reset":"manual"}]}`
	dayDoc     = `{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"%s","window":"%s","value":"event_max_cost","schema":"SCHEMA"}]}`
	approveDoc = `{"v":1,"approvals":[{"tools":["place_event_order"],"field":"dollar_amount","above":"%s","value":"event_max_cost","schema":"SCHEMA"}]}`
	capRuleDoc = `{"v":1,"rules":[{"tool":"place_event_order","shapes":[{"event_order":{"schema":"SCHEMA"%s}}]}]}`
)

const unit = int64(100000000)

// ---- Allow lists and market checks ------------------------------------------

func TestEventMarketsAllowList(t *testing.T) {
	h := newEvHarness(t, `{"v":1,"event_markets":[{"tools":["place_event_order"],"schema":"SCHEMA","categories":["ECONOMICS"],"series":["SER-C"]}]}`, "", nil)
	for _, m := range []string{"MKT-A1", "MKT-C1"} {
		if r := h.call(t, evBuy(m, "1", "0.5")); r.Error != nil {
			t.Fatalf("%s refused: %s", m, r.Error.Message)
		}
	}
	r := h.call(t, evBuy("MKT-B1", "1", "0.5"))
	if code(t, r) != toolcontrols.DenialCodeEventMarket {
		t.Fatalf("not refused by the allow list: %+v", r.Error)
	}
	for _, v := range []string{"MKT-B1", "SER-B", "Sports", "EV-B"} {
		if strings.Contains(r.Error.Message, v) || strings.Contains(string(r.Error.Data), v) {
			t.Fatalf("refusal echoes %q: %s %s", v, r.Error.Message, r.Error.Data)
		}
	}
	if h.rr.forwarded() != 2 {
		t.Fatalf("forwarded %d", h.rr.forwarded())
	}
	// A category is compared after trimming and upper-casing the looked-up one.
	h2 := newEvHarness(t, `{"v":1,"event_markets":[{"tools":["place_event_order"],"schema":"SCHEMA","categories":["SPORTS"]}]}`, "", nil)
	if r := h2.call(t, evBuy("MKT-B1", "1", "0.5")); r.Error != nil {
		t.Fatalf("category ' Sports ' refused: %s", r.Error.Message)
	}
	// Two entries must both pass: a child can only narrow.
	h3 := newEvHarness(t, `{"v":1,"event_markets":[{"tools":["place_event_order"],"schema":"SCHEMA","categories":["ECONOMICS"]},{"tools":["place_event_order"],"schema":"SCHEMA","series":["SER-C"]}]}`, "", nil)
	for _, m := range []string{"MKT-A1", "MKT-C1"} {
		if r := h3.call(t, evBuy(m, "1", "0.5")); code(t, r) != toolcontrols.DenialCodeEventMarket {
			t.Fatalf("%s passed one of two entries: %+v", m, r.Error)
		}
	}
	if h3.rr.forwarded() != 0 {
		t.Fatal("a refused call was forwarded")
	}
}

// Every way a market cannot be confirmed refuses the call: nothing is
// forwarded and nothing is counted.
func TestEventMarketsRefusesWhenLookupFails(t *testing.T) {
	ctl := `{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"100","window":"day","value":"event_max_cost","schema":"SCHEMA"}],` +
		`"event_markets":[{"tools":["place_event_order"],"schema":"SCHEMA","categories":["ECONOMICS"]}],` +
		`"event_stakes":[{"tools":["place_event_order"],"schema":"SCHEMA","max":"100","reset":"manual"}]}`
	type tc struct {
		name   string
		market string
		setup  func(h *evHarness)
		opts   *evOpts
		want   string
		lookup bool // the lookup may be asked
	}
	unavailable := "tool_controls_unavailable"
	for _, c := range []tc{
		{name: "not found", market: "MKT-NONE", want: toolcontrols.DenialCodeEventMarket, lookup: true},
		{name: "not two outcomes", market: "MKT-NB", want: toolcontrols.DenialCodeEventMarket, lookup: true},
		{name: "worth more than 1", market: "MKT-BIG", want: toolcontrols.DenialCodeEventMarket, lookup: true},
		{name: "combination", market: "MKT-COMBO", want: toolcontrols.DenialCodeEventMarket, lookup: true},
		{name: "not tradable", market: "MKT-CLOSED", want: toolcontrols.DenialCodeEventMarket, lookup: true},
		{name: "closed", market: "MKT-PAST", want: toolcontrols.DenialCodeEventMarket, lookup: true},
		{name: "no event", market: "MKT-NOEVENT", want: toolcontrols.DenialCodeEventMarket, lookup: true},
		{name: "lookup error", market: "MKT-A1", want: unavailable, lookup: true, setup: func(h *evHarness) { h.res.err = errors.New("lookup down") }},
		{name: "lookup timeout", market: "MKT-A1", want: unavailable, lookup: true, setup: func(h *evHarness) {
			h.res.delay = time.Second
			h.event.LookupTimeout = 20 * time.Millisecond
		}},
		{name: "wrong upstream", market: "MKT-A1", want: toolcontrols.DenialCodeEventMarket, setup: func(h *evHarness) { h.rr.service = "some-other-service" }},
		{name: "tool table down", market: "MKT-A1", want: unavailable, setup: func(h *evHarness) { h.rr.err = ErrToolTableUnavailable }},
		{name: "schema not bound", market: "MKT-A1", want: toolcontrols.DenialCodeEventMarket, setup: func(h *evHarness) {
			h.event.BoundUpstream = func(context.Context, string, string) (string, bool) { return "", false }
		}},
		{name: "no lookup registered", market: "MKT-A1", want: toolcontrols.DenialCodeEventMarket, opts: &evOpts{noLookup: true}},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newEvHarness(t, ctl, "", c.opts)
			if c.setup != nil {
				c.setup(h)
			}
			r := h.call(t, evBuy(c.market, "2", "0.5"))
			if got := code(t, r); got != c.want {
				t.Fatalf("code %q, want %q (%+v)", got, c.want, r.Error)
			}
			if h.rr.forwarded() != 0 || h.spent() != 0 || h.stake(t, 0, "EV-A") != 0 || h.stake(t, 0, "EV-X") != 0 {
				t.Fatalf("forwarded %d, spent %d", h.rr.forwarded(), h.spent())
			}
			if !c.lookup && h.res.lookups() != 0 {
				t.Fatalf("the market was looked up %d times", h.res.lookups())
			}
		})
	}
}

// ---- Per-event stakes --------------------------------------------------------

func TestEventStakeAcrossMarketsOfOneEvent(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "10"), "", nil)
	if r := h.call(t, evBuy("MKT-A1", "10", "0.56")); r.Error != nil {
		t.Fatal(r.Error.Message)
	}
	if r := h.call(t, evSell("MKT-A2", "10", "0.56")); r.Error != nil { // 10 x 0.44
		t.Fatal(r.Error.Message)
	}
	if got := h.stake(t, 0, "EV-A"); got != 10*unit {
		t.Fatalf("event total %d", got)
	}
	r := h.call(t, evBuy("MKT-A1", "0.01", "0.0001"))
	if code(t, r) != toolcontrols.DenialCodeSpendLimit || !strings.Contains(r.Error.Message, "on any one event") {
		t.Fatalf("not refused at the stake: %+v", r.Error)
	}
	if r := h.call(t, evBuy("MKT-B1", "10", "1")); code(t, r) != toolcontrols.DenialCodeSpendLimit {
		t.Fatal("price 1 is not a price") // valued refusal, never counted
	}
	if r := h.call(t, evBuy("MKT-B1", "10", "")); r.Error != nil { // another event: 10 x 1.00
		t.Fatalf("another event refused: %s", r.Error.Message)
	}
	if h.stake(t, 0, "EV-A") != 10*unit || h.stake(t, 0, "EV-B") != 10*unit || h.rr.forwarded() != 3 {
		t.Fatalf("totals %d/%d forwarded %d", h.stake(t, 0, "EV-A"), h.stake(t, 0, "EV-B"), h.rr.forwarded())
	}
	// The totals add up until they are reset, and only then.
	if err := h.store.ResetEvents(context.Background(), EventStakeKey("tenant-1", h.ctl.EventStakes[0])); err != nil {
		t.Fatal(err)
	}
	if r := h.call(t, evBuy("MKT-A1", "10", "0.56")); r.Error != nil {
		t.Fatalf("after reset: %s", r.Error.Message)
	}
}

func TestEventStakeConcurrentCallsNeverExceedMax(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "50"), "", nil)
	const n = 200
	var ok int64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := h.proxy.handleRequest(context.Background(), argCall(5000+i, h.token, "place_event_order", evBuy("MKT-A1", "1", "0.50")))
			if err == nil && resp.Error == nil {
				atomic.AddInt64(&ok, 1)
			}
		}(i)
	}
	wg.Wait()
	if ok != 100 || h.stake(t, 0, "EV-A") != 50*unit || h.rr.forwarded() != 100 {
		t.Fatalf("admitted %d, total %d, forwarded %d", ok, h.stake(t, 0, "EV-A"), h.rr.forwarded())
	}
}

// spendOnly is a store without per-event totals.
type spendOnly struct{ toolcontrols.SpendStore }

func TestEventStakeStoreDownRefuses(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "50"), "", nil)
	h.store.Err = toolcontrols.ErrStoreUnavailable
	if r := h.call(t, evBuy("MKT-A1", "1", "0.5")); code(t, r) != "tool_controls_unavailable" || h.rr.forwarded() != 0 {
		t.Fatalf("store down: %+v forwarded %d", r.Error, h.rr.forwarded())
	}
}

func TestEventStakeStoreWithoutInterfaceRefuses(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "50"), "", &evOpts{gate: func(store *toolcontrols.MemoryStore, _ *toolcontrols.MemoryApprovalStore, schema string) CallGate {
		return ChainGates(NewEventGate(spendOnly{store}, map[string]string{schema: "venue"}), newTestSpendGate(store))
	}})
	if r := h.call(t, evBuy("MKT-A1", "1", "0.5")); code(t, r) != "tool_controls_unavailable" || h.rr.forwarded() != 0 {
		t.Fatalf("no per-event store: %+v forwarded %d", r.Error, h.rr.forwarded())
	}
}

// A stake is given back when a later gate refuses or holds the call, and when
// the call is known not to have run; it is kept when the outcome is unknown.
func TestEventStakeGivenBackWhenLaterGateRefusesOrHolds(t *testing.T) {
	stake := `"event_stakes":[{"tools":["place_event_order"],"schema":"SCHEMA","max":"40","reset":"manual"}]`
	held := newEvHarness(t, `{"v":1,`+stake+`,"approvals":[{"tools":["place_event_order"],"field":"dollar_amount","above":"1","value":"event_max_cost","schema":"SCHEMA"}]}`, "", nil)
	if r := held.call(t, evBuy("MKT-A1", "10", "0.56")); code(t, r) != toolcontrols.DenialCodeApprovalRequired {
		t.Fatalf("not held: %+v", r.Error)
	}
	if held.stake(t, 0, "EV-A") != 0 {
		t.Fatalf("a held call kept its stake: %d", held.stake(t, 0, "EV-A"))
	}
	over := newEvHarness(t, `{"v":1,`+stake+`,"spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"1","window":"day","value":"event_max_cost","schema":"SCHEMA"}]}`, "", nil)
	if r := over.call(t, evBuy("MKT-A1", "10", "0.56")); code(t, r) != toolcontrols.DenialCodeSpendLimit {
		t.Fatalf("not refused by the day limit: %+v", r.Error)
	}
	if over.stake(t, 0, "EV-A") != 0 {
		t.Fatal("a refused call kept its stake")
	}
	for _, c := range []struct {
		name  string
		reply func() (*Response, error)
		kept  bool
	}{
		{"never sent", notSent, false},
		{"not processed", rpcErrorCode(-32602), false},
		{"lost after send", lostAfterSend, true},
		{"tool error", isErrorReply, true},
		{"succeeded", okReply, true},
	} {
		h := newEvHarness(t, `{"v":1,`+stake+`}`, "", &evOpts{scheduled: []func() (*Response, error){c.reply}})
		h.call(t, evBuy("MKT-A1", "10", "0.56"))
		want := int64(0)
		if c.kept {
			want = 560000000
		}
		if got := h.stake(t, 0, "EV-A"); got != want {
			t.Errorf("%s: stake %d, want %d", c.name, got, want)
		}
	}
}

// ---- Valuation on every path ------------------------------------------------

// Selling one side at p is counted as buying the other side at 1 - p.
func TestEventSellValuedAsBuyingTheOtherSide(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(dayDoc, "4.40", "day"), "", nil)
	if r := h.call(t, evSell("MKT-A1", "10", "0.56")); r.Error != nil {
		t.Fatalf("sell refused: %s", r.Error.Message)
	}
	if h.spent() != 440000000 {
		t.Fatalf("counted %d", h.spent())
	}
	h2 := newEvHarness(t, fmt.Sprintf(dayDoc, "4.40", "day"), "", nil)
	if r := h2.call(t, evBuy("MKT-A1", "10", "0.56")); code(t, r) != toolcontrols.DenialCodeSpendLimit {
		t.Fatal("a buy at 5.60 passed a 4.40 limit")
	}
}

// An order with no price counts 1 per contract.
func TestEventNoPriceValuedAtOnePerContract(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(dayDoc, "7", "day"), "", nil)
	if r := h.call(t, evSell("MKT-A1", "7", "")); r.Error != nil {
		t.Fatal(r.Error.Message)
	}
	if h.spent() != 7*unit {
		t.Fatalf("counted %d", h.spent())
	}
	if r := h.call(t, evBuy("MKT-A1", "0.01", "")); code(t, r) != toolcontrols.DenialCodeSpendLimit {
		t.Fatal("over the limit")
	}
}

// A closing order is not counted on any cost limit, and still passes every
// other check.
func TestEventClosingOrderNotCounted(t *testing.T) {
	ctl := fmt.Sprintf(`{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"5.60","window":"day","value":"event_max_cost","schema":"SCHEMA"}],`+
		`"approvals":[{"tools":["place_event_order"],"field":"dollar_amount","above":"%s","value":"event_max_cost","schema":"SCHEMA"}],`+
		`"event_markets":[{"tools":["place_event_order"],"schema":"SCHEMA","categories":["ECONOMICS"]}],`+
		`"event_stakes":[{"tools":["place_event_order"],"schema":"SCHEMA","max":"5.60","reset":"manual"}]}`, "0")
	h := newEvHarness(t, strings.Replace(ctl, `"above":"0"`, `"above":"6"`, 1), fmt.Sprintf(capRuleDoc, `,"max_cost":"5.60","max_contracts":"10"`), nil)
	if r := h.call(t, evBuy("MKT-A1", "10", "0.56")); r.Error != nil {
		t.Fatal(r.Error.Message)
	}
	if r := h.call(t, evBuy("MKT-A1", "0.01", "0.01")); r.Error == nil {
		t.Fatal("over every limit and admitted")
	}
	spent, stake := h.spent(), h.stake(t, 0, "EV-A")
	if r := h.call(t, evClose("MKT-A1", "10")); r.Error != nil {
		t.Fatalf("closing refused: %s", r.Error.Message)
	}
	if h.spent() != spent || h.stake(t, 0, "EV-A") != stake {
		t.Fatal("a closing order was counted")
	}
	// Still held to the allow list and to the contracts cap.
	if r := h.call(t, evClose("MKT-B1", "1")); code(t, r) != toolcontrols.DenialCodeEventMarket {
		t.Fatalf("closing outside the allow list: %+v", r.Error)
	}
	if r := h.call(t, evClose("MKT-A1", "10.01")); code(t, r) != argrules.DenialCode {
		t.Fatalf("closing over the contracts cap: %+v", r.Error)
	}
	// A closing order is never held for approval, even with a 0 threshold.
	h0 := newEvHarness(t, ctl, "", nil)
	if r := h0.call(t, evClose("MKT-A1", "3")); r.Error != nil {
		t.Fatalf("closing held: %s", r.Error.Message)
	}
	if h.rr.forwarded() != 2 || h0.rr.forwarded() != 1 {
		t.Fatalf("forwarded %d/%d", h.rr.forwarded(), h0.rr.forwarded())
	}
	// A closing order reads and writes no counter, so a store that is down
	// does not trap the agent in a position; an opening order is refused.
	h0.store.Err = toolcontrols.ErrStoreUnavailable
	if r := h0.call(t, evClose("MKT-A1", "3")); r.Error != nil {
		t.Fatalf("closing refused with the store down: %+v", r.Error)
	}
	if r := h0.call(t, evBuy("MKT-A1", "1", "0.01")); code(t, r) != "tool_controls_unavailable" {
		t.Fatalf("opening order with the store down: %+v", r.Error)
	}
}

// Shapes v1 refuses: resting orders, orders by amount, amend or replace,
// batches, and a close that cannot be guaranteed.
func TestEventRefusedShapesNeverForwardedOrCounted(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "100"), "", nil)
	for _, args := range []string{
		`{"market":"MKT-A1","action":"buy","count":"1","price":"0.5","tif":"gtc"}`,
		`{"market":"MKT-A1","action":"buy","amount":"5","tif":"ioc"}`,
		`{"market":"MKT-A1","action":"buy","count":"1","sizing":"amount","tif":"ioc"}`,
		`{"market":"MKT-A1","action":"buy","count":"1","price":"0.5","tif":"ioc","replaces":"ord-1"}`,
		`{"orders":[{"market":"MKT-A1"}]}`,
		`{"market":"MKT-A1","action":"sell","count":"1","tif":"fok","reduce_only":true}`,
		`{"market":"MKT-A1","action":"buy","count":"1","price":"0.5","tif":"ioc","note":"x"}`,
	} {
		r := h.call(t, args)
		if code(t, r) != toolcontrols.DenialCodeSpendLimit || !strings.Contains(r.Error.Message, "counts each order at the most it can cost") {
			t.Errorf("%s: %+v", args, r.Error)
		}
	}
	if h.rr.forwarded() != 0 || h.stake(t, 0, "EV-A") != 0 || h.res.lookups() != 0 {
		t.Fatalf("forwarded %d, lookups %d", h.rr.forwarded(), h.res.lookups())
	}
}

// A tool no event control covers (a cancel) is forwarded and not counted.
func TestEventUncoveredToolNotCounted(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "1"), "", nil)
	if r := h.callTool(t, "cancel_order", `{"order_id":"o-1"}`); r.Error != nil {
		t.Fatalf("cancel refused: %s", r.Error.Message)
	}
	if h.rr.forwarded() != 1 || h.res.lookups() != 0 {
		t.Fatalf("forwarded %d lookups %d", h.rr.forwarded(), h.res.lookups())
	}
}

// Exactly at each limit is admitted; one unit (or 0.01 contract) over is not.
func TestEventBoundaries(t *testing.T) {
	t.Run("per-order cap", func(t *testing.T) {
		h := newEvHarness(t, "", fmt.Sprintf(capRuleDoc, `,"max_cost":"5.60"`), nil)
		if r := h.call(t, evBuy("MKT-A1", "10", "0.56")); r.Error != nil {
			t.Fatal(r.Error.Message)
		}
		if r := h.call(t, evBuy("MKT-A1", "10", "0.5601")); code(t, r) != argrules.DenialCode {
			t.Fatalf("%+v", r.Error)
		}
	})
	t.Run("contracts cap", func(t *testing.T) {
		h := newEvHarness(t, "", fmt.Sprintf(capRuleDoc, `,"max_contracts":"10"`), nil)
		if r := h.call(t, evBuy("MKT-A1", "10.00", "0.01")); r.Error != nil {
			t.Fatal(r.Error.Message)
		}
		if r := h.call(t, evBuy("MKT-A1", "10.01", "0.01")); code(t, r) != argrules.DenialCode {
			t.Fatalf("%+v", r.Error)
		}
	})
	for _, w := range []string{"day", "week"} {
		t.Run(w, func(t *testing.T) {
			h := newEvHarness(t, fmt.Sprintf(dayDoc, "20", w), "", nil)
			for i := 0; i < 3; i++ {
				if r := h.call(t, evBuy("MKT-A1", "10", "0.56")); r.Error != nil {
					t.Fatal(r.Error.Message)
				}
			}
			if r := h.call(t, evBuy("MKT-C1", "4", "0.80")); r.Error != nil {
				t.Fatalf("exactly at the limit: %s", r.Error.Message)
			}
			if h.spent() != 20*unit {
				t.Fatalf("spent %d", h.spent())
			}
			if r := h.call(t, evBuy("MKT-C1", "0.01", "0.0001")); code(t, r) != toolcontrols.DenialCodeSpendLimit {
				t.Fatalf("over: %+v", r.Error)
			}
		})
	}
	t.Run("approval", func(t *testing.T) {
		h := newEvHarness(t, fmt.Sprintf(approveDoc, "5"), "", nil)
		if r := h.call(t, evBuy("MKT-A1", "10", "0.50")); r.Error != nil {
			t.Fatalf("held at the threshold: %s", r.Error.Message)
		}
		if r := h.call(t, evBuy("MKT-A1", "10", "0.5001")); code(t, r) != toolcontrols.DenialCodeApprovalRequired {
			t.Fatalf("not held above: %+v", r.Error)
		}
	})
	t.Run("stake", func(t *testing.T) {
		h := newEvHarness(t, fmt.Sprintf(stakeDoc, "5.60"), "", nil)
		if r := h.call(t, evBuy("MKT-A1", "10", "0.56")); r.Error != nil {
			t.Fatal(r.Error.Message)
		}
		if r := h.call(t, evBuy("MKT-A2", "0.01", "0.0001")); code(t, r) != toolcontrols.DenialCodeSpendLimit {
			t.Fatalf("over: %+v", r.Error)
		}
	})
}

// ---- The drift test: one value on every path ----------------------------------

func gateTokenInfo(t *testing.T, schema, ctlDoc, ruleDoc string) *TokenInfo {
	t.Helper()
	ti := &TokenInfo{TenantID: "tenant-1", TokenID: "tok-drift", BudgetID: "tok-drift"}
	if ctlDoc != "" {
		d, err := toolcontrols.ParseJSON([]byte(strings.ReplaceAll(ctlDoc, "SCHEMA", schema)))
		if err != nil {
			t.Fatal(err)
		}
		v, err := toolcontrols.ScopeValue(d)
		if err != nil {
			t.Fatal(err)
		}
		d, err = toolcontrols.ParseScopeValue(v)
		if err != nil {
			t.Fatal(err)
		}
		ti.SpendLimits, ti.ApprovalRules, ti.EventMarkets, ti.EventStakes = d.SpendLimits, d.Approvals, d.EventMarkets, d.EventStakes
	}
	if ruleDoc != "" {
		rules, err := argrules.ParseJSON([]byte(strings.ReplaceAll(ruleDoc, "SCHEMA", schema)))
		if err != nil {
			t.Fatal(err)
		}
		ti.ArgumentRules = rules
	}
	return ti
}

func gateCall(ti *TokenInfo, args string) GateCall {
	return GateCall{
		Token:     ti,
		Tool:      "place_event_order",
		Params:    json.RawMessage(`{"name":"place_event_order","arguments":` + args + `}`),
		RequestID: fmt.Sprintf("r%d", atomic.AddInt64(&evSeq, 1)),
		Origin:    ToolOrigin{Listed: true, Service: "venue"},
		Now:       time.Now(),
	}
}

// admits runs one gate over a fresh store and reports whether it admitted
// the call, and the refusal code otherwise.
func admits(t *testing.T, mk func(*toolcontrols.MemoryStore) CallGate, call GateCall) (bool, string) {
	t.Helper()
	adm, err := mk(toolcontrols.NewMemoryStore(nil)).Admit(context.Background(), call)
	if err != nil {
		return false, "error"
	}
	if adm == nil || adm.Refusal == nil {
		return true, ""
	}
	return false, adm.Refusal.Code
}

// The argument rule cap, the day and week limits, the approval rule and the
// per-event stake value each call identically: each admits at exactly the
// value and refuses one unit below it, and each refuses the same calls.
func TestEventOneValueOnEveryPath(t *testing.T) {
	schema, _ := registerGateSchema(t, true)
	var orders []string
	counts := []string{"0.01", "1", "2.5", "10", "33.33", "100"}
	prices := []string{"", "0.0001", "0.01", "0.3333", "0.5", "0.56", "0.99", "0.12345678"}
	for i, c := range counts {
		for j, p := range prices {
			if (i+j)%2 == 0 {
				orders = append(orders, evBuy("MKT-A1", c, p))
			} else {
				orders = append(orders, evSell("MKT-A1", c, p))
			}
		}
	}
	orders = append(orders,
		evClose("MKT-A1", "5"),
		`{"market":"MKT-A1","action":"buy","count":"1","price":"0.5","tif":"gtc"}`,
		`{"market":"MKT-A1","action":"buy","amount":"5","tif":"ioc"}`,
		evBuy("MKT-A1", "1", "1.2"),
		evBuy("MKT-A1", "0", "0.5"),
	)
	if len(orders) < 50 {
		t.Fatalf("only %d orders", len(orders))
	}
	approval := func(s *toolcontrols.MemoryStore) CallGate {
		g := NewApprovalGate(toolcontrols.NewMemoryApprovalStore(), nil)
		g.Synchronous = true
		return g
	}
	spend := func(s *toolcontrols.MemoryStore) CallGate { return newTestSpendGate(s) }
	event := func(s *toolcontrols.MemoryStore) CallGate { return NewEventGate(s, map[string]string{schema: "venue"}) }
	for _, args := range orders {
		v, why := argrules.EventOrderMaxCost(schema, mustArgs(t, args))
		check := func(at string) (rule, day, week, held, stake bool, codes []string) {
			ti := gateTokenInfo(t, schema, "", fmt.Sprintf(capRuleDoc, `,"max_cost":"`+at+`"`))
			rule = argrules.Check(ti.ArgumentRules, "place_event_order", gateCall(ti, args).Params) == nil
			var c string
			ti = gateTokenInfo(t, schema, fmt.Sprintf(dayDoc, at, "day"), "")
			day, c = admits(t, spend, gateCall(ti, args))
			codes = append(codes, c)
			ti = gateTokenInfo(t, schema, fmt.Sprintf(dayDoc, at, "week"), "")
			week, c = admits(t, spend, gateCall(ti, args))
			codes = append(codes, c)
			ti = gateTokenInfo(t, schema, fmt.Sprintf(approveDoc, at), "")
			var notHeld bool
			notHeld, c = admits(t, approval, gateCall(ti, args))
			held = !notHeld
			codes = append(codes, c)
			ti = gateTokenInfo(t, schema, fmt.Sprintf(stakeDoc, at), "")
			stake, c = admits(t, event, gateCall(ti, args))
			codes = append(codes, c)
			return
		}
		if !why.Valued() {
			rule, day, week, held, stake, codes := check("1000")
			if rule || day || week || !held || stake {
				t.Errorf("%s (refused: %q): rule %v day %v week %v held %v stake %v", args, why, rule, day, week, held, stake)
			}
			want := []string{toolcontrols.DenialCodeSpendLimit, toolcontrols.DenialCodeSpendLimit, toolcontrols.DenialCodeApprovalUnreadable, toolcontrols.DenialCodeSpendLimit}
			if strings.Join(codes, ",") != strings.Join(want, ",") {
				t.Errorf("%s: codes %v", args, codes)
			}
			continue
		}
		if v.Closing {
			rule, day, week, held, stake, _ := check("0.00000001")
			if !rule || !day || !week || held || !stake {
				t.Errorf("closing %s: rule %v day %v week %v held %v stake %v", args, rule, day, week, held, stake)
			}
			continue
		}
		at := formatUnits(v.Units)
		rule, day, week, held, stake, _ := check(at)
		if !rule || !day || !week || held || !stake {
			t.Errorf("%s at %s: rule %v day %v week %v held %v stake %v", args, at, rule, day, week, held, stake)
		}
		if v.Units > 1 {
			below := formatUnits(v.Units - 1)
			rule, day, week, held, stake, _ := check(below)
			if rule || day || week || !held || stake {
				t.Errorf("%s at %s: rule %v day %v week %v held %v stake %v", args, below, rule, day, week, held, stake)
			}
		}
	}
}

func mustArgs(t *testing.T, js string) map[string]json.RawMessage {
	t.Helper()
	var m map[string]json.RawMessage
	if err := json.Unmarshal([]byte(js), &m); err != nil {
		t.Fatal(err)
	}
	return m
}

// ---- Fail closed --------------------------------------------------------------

// A gateway that knows the event keys but has no EventGate in its chain
// refuses every call an event control covers instead of passing it to the
// spending and approval gates unchecked. With no gate at all, the call is
// refused as any control is.
func TestEventControlsWithoutEventGateRefuse(t *testing.T) {
	noEvent := &evOpts{gate: func(store *toolcontrols.MemoryStore, appr *toolcontrols.MemoryApprovalStore, _ string) CallGate {
		return ChainGates(newTestSpendGate(store), NewApprovalGate(appr, nil))
	}}
	noGate := &evOpts{gate: func(*toolcontrols.MemoryStore, *toolcontrols.MemoryApprovalStore, string) CallGate { return nil }}
	for name, docs := range map[string][2]string{
		"event markets":  {`{"v":1,"event_markets":[{"tools":["place_event_order"],"schema":"SCHEMA","series":["SER-A"]}]}`, ""},
		"event stakes":   {fmt.Sprintf(stakeDoc, "100"), ""},
		"day limit":      {fmt.Sprintf(dayDoc, "100", "day"), ""},
		"approval":       {fmt.Sprintf(approveDoc, "100"), ""},
		"per-order cap":  {"", fmt.Sprintf(capRuleDoc, `,"max_cost":"100"`)},
		"contracts only": {"", fmt.Sprintf(capRuleDoc, `,"max_contracts":"100"`)},
	} {
		for oname, o := range map[string]*evOpts{"no event gate": noEvent, "no gate": noGate} {
			h := newEvHarness(t, docs[0], docs[1], o)
			r := h.call(t, evBuy("MKT-A1", "1", "0.5"))
			if code(t, r) != "tool_controls_unavailable" || h.rr.forwarded() != 0 {
				t.Errorf("%s, %s: %+v forwarded %d", name, oname, r.Error, h.rr.forwarded())
			}
		}
	}
	// A token without event controls is unaffected by the check.
	h := newEvHarness(t, `{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"count","max":"100","window":"day"}]}`, "", noEvent)
	if r := h.call(t, evBuy("MKT-A1", "1", "0.5")); r.Error != nil {
		t.Fatalf("a plain limit refused: %s", r.Error.Message)
	}
}

// NeedsCallGate is true for every kind of event control, including an
// argument rule that is only a per-order cap.
func TestNeedsCallGateLearnsEventControls(t *testing.T) {
	schema, _ := registerGateSchema(t, true)
	for name, ti := range map[string]*TokenInfo{
		"cap rule": gateTokenInfo(t, schema, "", fmt.Sprintf(capRuleDoc, `,"max_contracts":"1"`)),
		"markets":  gateTokenInfo(t, schema, `{"v":1,"event_markets":[{"tools":["x"],"schema":"SCHEMA","series":["S"]}]}`, ""),
		"stakes":   gateTokenInfo(t, schema, fmt.Sprintf(stakeDoc, "1"), ""),
	} {
		if !ti.NeedsCallGate("place_event_order") {
			t.Errorf("%s: no gate needed", name)
		}
	}
	if (&TokenInfo{}).NeedsCallGate("place_event_order") {
		t.Fatal("a plain token needs a gate")
	}
}

// A control naming a schema this process does not register refuses at
// verify, and a hand-built one refuses at call on every path.
func TestEventUnregisteredSchemaRefused(t *testing.T) {
	h := newEvHarness(t, "", "", nil)
	svc, err := macaroon.NewService(argTestRoot)
	if err != nil {
		t.Fatal(err)
	}
	doc := `{"v":1,"event_stakes":[{"tools":["place_event_order"],"schema":"never_registered_v1","max":"40","reset":"manual"}]}`
	word := "scope = " + argrules.ControlScopePrefix + "v1:" + strings.Repeat("cd", 16) + ":" + base64.RawURLEncoding.EncodeToString([]byte(doc))
	h.token = appendCaveat(t, svc, h.token, word)
	if r := h.call(t, evBuy("MKT-A1", "1", "0.5")); r.Error == nil || h.rr.forwarded() != 0 {
		t.Fatalf("a token with an unregistered schema was served: %+v", r)
	}
	// Controls that name the schema without going through the parser: parsed
	// under a registered schema, then renamed.
	ti := gateTokenInfo(t, h.schema, `{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"5","window":"day","value":"event_max_cost","schema":"SCHEMA"}],"approvals":[{"tools":["place_event_order"],"field":"dollar_amount","above":"5","value":"event_max_cost","schema":"SCHEMA"}]}`, "")
	ti.SpendLimits[0].Schema = "never_registered_v1"
	ti.ApprovalRules[0].Schema = "never_registered_v1"
	call := gateCall(ti, evBuy("MKT-A1", "1", "0.5"))
	for name, g := range map[string]CallGate{
		"spend":    NewSpendGate(toolcontrols.NewMemoryStore(nil)),
		"approval": NewApprovalGate(toolcontrols.NewMemoryApprovalStore(), nil),
		"event":    NewEventGate(toolcontrols.NewMemoryStore(nil), map[string]string{"never_registered_v1": "venue"}),
	} {
		adm, err := g.Admit(context.Background(), call)
		if err == nil && (adm == nil || adm.Refusal == nil) {
			t.Errorf("%s admitted a call valued under an unregistered schema", name)
		}
		if adm != nil && adm.Refusal != nil && !strings.Contains(adm.Refusal.Message, string(argrules.WhyEventUnknownSchema)) {
			t.Errorf("%s: %s", name, adm.Refusal.Message)
		}
	}
}

// A token that names two schemas for one tool is refused.
func TestEventTwoSchemasOnOneToolRefused(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "40"), "", nil)
	other, _ := registerGateSchema(t, true)
	svc, _ := macaroon.NewService(argTestRoot)
	h.token = appendCaveat(t, svc, h.token, controlWord(t, strings.ReplaceAll(fmt.Sprintf(dayDoc, "40", "day"), "SCHEMA", other)))
	if r := h.call(t, evBuy("MKT-A1", "1", "0.5")); code(t, r) != toolcontrols.DenialCodeSpendLimit || h.rr.forwarded() != 0 {
		t.Fatalf("%+v", r.Error)
	}
}

// No refusal carries the call's market, count or price, or the looked-up
// event, series or category.
func TestEventRefusalsNeverEchoValues(t *testing.T) {
	ctl := `{"v":1,"event_markets":[{"tools":["place_event_order"],"schema":"SCHEMA","series":["SER-Z"]}],"event_stakes":[{"tools":["place_event_order"],"schema":"SCHEMA","max":"1","reset":"manual"}]}`
	h := newEvHarness(t, ctl, fmt.Sprintf(capRuleDoc, `,"max_contracts":"50"`), nil)
	h.res.markets["SECRETMKT-9"] = EventMarket{Event: "SECRETEV", Series: "SECRETSER", Category: "SECRETCAT", Binary: true, MaxValueOne: true, Tradable: true}
	h.res.markets["SECRETMKT-Z"] = EventMarket{Event: "SECRETEV", Series: "SER-Z", Category: "SECRETCAT", Binary: true, MaxValueOne: true, Tradable: true}
	var texts []string
	for _, args := range []string{
		evBuy("SECRETMKT-9", "12.34", "0.9876"),
		evBuy("SECRETMKT-Z", "12.34", "0.9876"),
		evBuy("SECRETMKT-Z", "77.77", "0.9876"),
		`{"market":"SECRETMKT-9","action":"buy","count":"12.34","price":"0.9876","tif":"gtc"}`,
		evBuy("SECRETMKT-NONE", "12.34", "0.9876"),
	} {
		r := h.call(t, args)
		if r.Error == nil {
			t.Fatalf("%s admitted", args)
		}
		texts = append(texts, r.Error.Message+" "+string(r.Error.Data))
	}
	for _, text := range texts {
		for _, v := range []string{"SECRET", "12.34", "77.77", "0.9876"} {
			if strings.Contains(text, v) {
				t.Errorf("refusal echoes %q: %s", v, text)
			}
		}
	}
}

// A token with no event control goes through the EventGate untouched.
func TestEventGateIgnoresTokensWithoutEventControls(t *testing.T) {
	ti := gateTokenInfo(t, "", spendDoc200Day, "")
	adm, err := NewEventGate(toolcontrols.NewMemoryStore(nil), nil).Admit(context.Background(), GateCall{Token: ti, Tool: "place_crypto_order", Params: json.RawMessage(`{"name":"place_crypto_order","arguments":{"dollar_amount":"5"}}`)})
	if err != nil || adm != nil {
		t.Fatalf("%+v %v", adm, err)
	}
	if !ChainGates(NewEventGate(nil, nil), NewSpendGate(nil)).(EventControlGate).EnforcesEventControls() {
		t.Fatal("a chain with an event gate does not say so")
	}
	if gateEnforcesEventControls(ChainGates(NewSpendGate(nil), NewApprovalGate(nil, nil))) {
		t.Fatal("a chain without an event gate says it enforces event controls")
	}
}

// ---- Self-hosted wiring ------------------------------------------------------

func selfHostedConfig(entries ...EventContractConfig) *Config {
	return &Config{
		Upstreams: map[string]UpstreamConfig{"venue": {Transport: "stdio", Command: []string{"true"}}},
		Controls:  ControlsConfig{Store: "memory", EventContracts: entries},
	}
}

func TestControlGateFromConfig(t *testing.T) {
	// No controls: no gate, as before.
	if g, err := ControlGateFromConfig(&Config{}); g != nil || err != nil {
		t.Fatalf("%v %v", g, err)
	}
	// A memory store with no event entries: spend, approval and an event gate
	// that refuses any schema it has no binding for.
	g, err := ControlGateFromConfig(selfHostedConfig())
	if err != nil || g == nil || !gateEnforcesEventControls(g) {
		t.Fatalf("%v %v", g, err)
	}
	// An entry naming a schema the program did not register refuses to start.
	_, err = ControlGateFromConfig(selfHostedConfig(EventContractConfig{Schema: "never_registered_v1", Upstream: "venue"}))
	if err == nil || !strings.Contains(err.Error(), `value schema "never_registered_v1" is not registered by this program`) || !strings.Contains(err.Error(), "not started") {
		t.Fatalf("unregistered schema: %v", err)
	}
	// Registered with no market lookup: refused too.
	bare, _ := registerGateSchema(t, false)
	if _, err := ControlGateFromConfig(selfHostedConfig(EventContractConfig{Schema: bare, Upstream: "venue"})); err == nil || !strings.Contains(err.Error(), "no market lookup") {
		t.Fatalf("no lookup: %v", err)
	}
	// Registered with a lookup: the gate binds the schema to its upstream.
	full, _ := registerGateSchema(t, true)
	g, err = ControlGateFromConfig(selfHostedConfig(EventContractConfig{Schema: full, Upstream: "venue"}))
	if err != nil || g == nil {
		t.Fatalf("%v %v", g, err)
	}
	ti := gateTokenInfo(t, full, fmt.Sprintf(stakeDoc, "5"), "")
	adm, err := g.Admit(context.Background(), gateCall(ti, evBuy("MKT-A1", "1", "0.5")))
	if err != nil || adm == nil || adm.Refusal != nil {
		t.Fatalf("bound schema: %+v %v", adm, err)
	}
	// Shape errors in the section.
	for _, c := range []*Config{
		{Upstreams: map[string]UpstreamConfig{"venue": {}}, Controls: ControlsConfig{Store: "redis"}},
		{Upstreams: map[string]UpstreamConfig{"venue": {}}, Controls: ControlsConfig{EventContracts: []EventContractConfig{{Schema: full, Upstream: "venue"}}}},
		selfHostedConfig(EventContractConfig{Schema: full, Upstream: "nowhere"}),
		selfHostedConfig(EventContractConfig{Schema: full, Upstream: "venue"}, EventContractConfig{Schema: full, Upstream: "venue"}),
		selfHostedConfig(EventContractConfig{Upstream: "venue"}),
	} {
		if err := c.Controls.validate(c.Upstreams); err == nil {
			if _, err := ControlGateFromConfig(c); err == nil {
				t.Errorf("accepted %+v", c.Controls)
			}
		}
	}
}
