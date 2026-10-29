package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// ---- Reservations and resets ------------------------------------------------

// A give-back from before ResetEvents never touches a reservation made after
// it, even when both calls carry the same request id.
func TestEventStakeStaleGiveBackAfterResetChangesNothing(t *testing.T) {
	ctx := context.Background()
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "1"), "", nil)
	ti := gateTokenInfo(t, h.schema, fmt.Sprintf(stakeDoc, "1"), "")
	call := gateCall(ti, evBuy("MKT-A1", "2", "0.5")) // 1.00, the whole limit
	call.RequestID = "same-request"
	old, err := h.event.Admit(ctx, call)
	if err != nil || old == nil || old.Refusal != nil {
		t.Fatalf("first order: %+v %v", old, err)
	}
	key := EventStakeKey(ti.TenantID, ti.EventStakes[0])
	if err := h.store.ResetEvents(ctx, key); err != nil {
		t.Fatal(err)
	}
	fresh, err := h.event.Admit(ctx, call)
	if err != nil || fresh == nil || fresh.Refusal != nil {
		t.Fatalf("order after reset: %+v %v", fresh, err)
	}
	fresh.Settle(ctx, GateOutcome{Succeeded: true})
	old.Settle(ctx, GateOutcome{}) // known not to have run: given back
	if total, _ := h.store.EventTotal(ctx, key, "EV-A"); total != unit {
		t.Fatalf("total %d after a give-back from before the reset, want %d", total, unit)
	}
	next := call
	next.RequestID = "next-request"
	adm, err := h.event.Admit(ctx, next)
	if err != nil || adm == nil || adm.Refusal == nil || adm.Refusal.Code != toolcontrols.DenialCodeSpendLimit {
		t.Fatalf("an order past the limit was admitted: %+v %v", adm, err)
	}
}

// evWireCall sends one tools/call over the streamable HTTP transport and
// returns the JSON-RPC response.
func evWireCall(t *testing.T, url, sid, token string, id int, args string) *Response {
	t.Helper()
	params := json.RawMessage(`{"name":"place_event_order","arguments":` + args + `}`)
	status, _, body := streamablePost(t, url+"/mcp", sid, token, map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": params})
	if status != http.StatusOK {
		t.Errorf("HTTP status %d", status)
		return &Response{Error: &RPCError{Code: -1}}
	}
	var resp Response
	if json.Unmarshal([]byte(body), &resp) == nil {
		return &resp
	}
	for _, line := range strings.Split(body, "\n") {
		if strings.HasPrefix(line, "data: ") && json.Unmarshal([]byte(strings.TrimPrefix(line, "data: ")), &resp) == nil {
			return &resp
		}
	}
	t.Errorf("no JSON-RPC response in %q", body)
	return &Response{Error: &RPCError{Code: -1}}
}

// Through the MCP server: an order is in flight, the totals are reset, the
// same request is sent again and admitted, and the first then comes back
// known not to have run. Its give-back leaves the new reservation counted, so
// a further order past the limit is refused.
func TestEventStakeStaleGiveBackAfterResetOverHTTP(t *testing.T) {
	entered, done := make(chan struct{}), make(chan struct{})
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "0.5"), "", &evOpts{scheduled: []func() (*Response, error){
		func() (*Response, error) { close(entered); <-done; return rpcErrorCode(-32602)() },
		okReply, okReply,
	}})
	srv := httptest.NewServer(NewSSEServer(h.proxy, ":0").Handler())
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, h.token)
	args := evBuy("MKT-A1", "1", "0.5") // 0.50, the whole limit
	first := make(chan *Response, 1)
	go func() { first <- evWireCall(t, srv.URL, sid, h.token, 300, args) }()
	<-entered
	if err := h.store.ResetEvents(context.Background(), EventStakeKey("tenant-1", h.ctl.EventStakes[0])); err != nil {
		t.Fatal(err)
	}
	second := evWireCall(t, srv.URL, sid, h.token, 300, args)
	close(done)
	if second.Error != nil {
		t.Fatalf("order after reset refused: %+v", second.Error)
	}
	if r := <-first; r.Error == nil {
		t.Fatal("the first order should have come back refused by the service")
	}
	if got := h.stake(t, 0, "EV-A"); got != unit/2 {
		t.Fatalf("total %d after a give-back from before the reset, want %d", got, unit/2)
	}
	third := evWireCall(t, srv.URL, sid, h.token, 301, args)
	if code(t, third) != toolcontrols.DenialCodeSpendLimit {
		t.Fatalf("an order past the limit was not refused: %+v", third.Error)
	}
	if h.rr.forwarded() != 2 {
		t.Fatalf("forwarded %d, want 2", h.rr.forwarded())
	}
}

// The same, with a real upstream MCP service behind the gateway: the first
// tools/call it receives waits, then answers with a JSON-RPC error (known not
// to have run).
func TestEventStakeStaleGiveBackAfterResetThroughUpstream(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "1"), "", nil)
	started, release := make(chan struct{}), make(chan struct{})
	var calls atomic.Int64
	up := newFakeService(t, "imaginary", fakeTool{Name: "place_event_order"})
	next := up.srv.Config.Handler
	up.srv.Config.Handler = http.HandlerFunc(func(rw http.ResponseWriter, r *http.Request) {
		raw, _ := io.ReadAll(r.Body)
		r.Body = io.NopCloser(bytes.NewReader(raw))
		var req Request
		_ = json.Unmarshal(raw, &req)
		if req.Method == MethodToolsCall && calls.Add(1) == 1 {
			close(started)
			<-release
			rw.Header().Set("Content-Type", "application/json")
			fmt.Fprintf(rw, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32602,"message":"not processed"}}`, req.ID)
			return
		}
		next.ServeHTTP(rw, r)
	})
	mgr := NewUpstreamManager(map[string]UpstreamConfig{"venue": up.cfg("")}, nil, "venue", true)
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := mgr.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer mgr.Close()
	h.proxy.upstream = mgr
	h.proxy.SetUpstreamRouter(&defaultRouter{mgr: mgr})
	srv := httptest.NewServer(NewSSEServer(h.proxy, ":0").Handler())
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, h.token)
	args := evBuy("MKT-A1", "2", "0.5") // 1.00, the whole limit
	first := make(chan *Response, 1)
	go func() { first <- evWireCall(t, srv.URL, sid, h.token, 901, args) }()
	<-started
	if err := h.store.ResetEvents(context.Background(), EventStakeKey("tenant-1", h.ctl.EventStakes[0])); err != nil {
		t.Fatal(err)
	}
	second := evWireCall(t, srv.URL, sid, h.token, 901, args)
	close(release)
	if second.Error != nil {
		t.Fatalf("order after reset refused: %+v", second.Error)
	}
	<-first
	if got := h.stake(t, 0, "EV-A"); got != unit {
		t.Fatalf("total %d after a give-back from before the reset, want %d", got, unit)
	}
	if r := evWireCall(t, srv.URL, sid, h.token, 902, args); code(t, r) != toolcontrols.DenialCodeSpendLimit {
		t.Fatalf("an order past the limit was not refused: %+v", r.Error)
	}
	if calls.Load() != 2 {
		t.Fatalf("the service received %d orders, want 2", calls.Load())
	}
}

// noIdentityStore reserves but returns no reservation identity.
type noIdentityStore struct{ *toolcontrols.MemoryStore }

func (s noIdentityStore) ReserveEvent(ctx context.Context, r toolcontrols.EventReservation) (toolcontrols.EventReserveResult, error) {
	out, err := s.MemoryStore.ReserveEvent(ctx, r)
	out.Reservation = ""
	return out, err
}

// A reservation that cannot be named cannot be given back alone, so the call
// is refused and nothing is forwarded.
func TestEventStakeStoreWithoutReservationIdentityRefuses(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "50"), "", &evOpts{gate: func(s *toolcontrols.MemoryStore, _ *toolcontrols.MemoryApprovalStore, schema string) CallGate {
		return NewEventGate(noIdentityStore{s}, map[string]string{schema: "venue"})
	}})
	r := h.call(t, evBuy("MKT-A1", "1", "0.5"))
	if got := code(t, r); got != "tool_controls_unavailable" {
		t.Fatalf("code %q, want tool_controls_unavailable (%+v)", got, r.Error)
	}
	if h.rr.forwarded() != 0 {
		t.Fatalf("forwarded %d", h.rr.forwarded())
	}
}

// ---- A lookup that answers after its deadline ---------------------------------

// lateMarket answers with an open market only after the lookup's deadline.
type lateMarket struct{ waitForDone bool }

func (l lateMarket) ResolveEventMarket(ctx context.Context, _, _, _ string) (EventMarket, error) {
	if l.waitForDone {
		<-ctx.Done()
	} else if dl, ok := ctx.Deadline(); ok {
		time.Sleep(time.Until(dl) + 5*time.Millisecond) // ignores ctx
	}
	return EventMarket{Event: "EV-A", Series: "SER-A", Category: "ECONOMICS", Binary: true, MaxValueOne: true, Tradable: true}, nil
}

func TestEventLookupAnsweringAfterDeadlineRefused(t *testing.T) {
	ctl := `{"v":1,"spend_limits":[{"tools":["place_event_order"],"field":"dollar_amount","max":"100","window":"day","value":"event_max_cost","schema":"SCHEMA"}],` +
		`"event_stakes":[{"tools":["place_event_order"],"schema":"SCHEMA","max":"100","reset":"manual"}]}`
	for _, c := range []struct {
		name string
		res  lateMarket
	}{{"waits for the deadline", lateMarket{waitForDone: true}}, {"ignores the deadline", lateMarket{}}} {
		t.Run(c.name, func(t *testing.T) {
			h := newEvHarness(t, ctl, "", &evOpts{noLookup: true})
			if err := RegisterEventMarketResolver(h.schema, c.res); err != nil {
				t.Fatal(err)
			}
			h.event.LookupTimeout = 10 * time.Millisecond
			r := h.call(t, evBuy("MKT-A1", "1", "0.5"))
			if got := code(t, r); got != "tool_controls_unavailable" {
				t.Fatalf("code %q, want tool_controls_unavailable (%+v)", got, r.Error)
			}
			if h.rr.forwarded() != 0 || h.stake(t, 0, "EV-A") != 0 || h.spent() != 0 {
				t.Fatalf("forwarded %d, stake %d, spent %d", h.rr.forwarded(), h.stake(t, 0, "EV-A"), h.spent())
			}
		})
	}
}

// A lookup that returns a market while the call's own context is done is not
// used either.
func TestEventLookupAnsweringAfterContextDoneRefused(t *testing.T) {
	h := newEvHarness(t, fmt.Sprintf(stakeDoc, "100"), "", nil)
	ti := gateTokenInfo(t, h.schema, fmt.Sprintf(stakeDoc, "100"), "")
	ctx, cancel := context.WithCancel(context.Background())
	cancel()
	adm, err := h.event.Admit(ctx, gateCall(ti, evBuy("MKT-A1", "1", "0.5")))
	if err == nil {
		t.Fatalf("admitted after its context was done: %+v", adm)
	}
	if total, _ := h.store.EventTotal(context.Background(), EventStakeKey(ti.TenantID, ti.EventStakes[0]), "EV-A"); total != 0 {
		t.Fatalf("stake %d reserved", total)
	}
}

// ---- Closing orders get the same input checks --------------------------------

// closingPriceRead is the test schema, except that a closing order keeps the
// price it was sent with.
func closingPriceRead(args map[string]json.RawMessage) (argrules.EventOrder, argrules.OrderCostWhy) {
	o, why := gateSchemaRead(args)
	if why.Valued() && o.Closing {
		if raw, ok := args["price"]; ok {
			p, ok := argrules.ReadArgDecimal(raw)
			if !ok {
				return argrules.EventOrder{}, argrules.WhyBadNumber
			}
			o.Price = p
		}
	}
	return o, why
}

// A closing order with a price outside (0, 1) is refused on every path that
// values it, and is never forwarded or counted; one with a price inside is
// admitted and not counted.
func TestEventClosingOrderPriceChecked(t *testing.T) {
	ctl := `{"v":1,"event_stakes":[{"tools":["place_event_order"],"schema":"SCHEMA","max":"1","reset":"manual"}]}`
	rule := fmt.Sprintf(capRuleDoc, `,"max_cost":"1","max_contracts":"10"`)
	closing := func(price string) string {
		return `{"market":"MKT-A1","action":"sell","count":"1","price":"` + price + `","tif":"ioc","reduce_only":true}`
	}
	for _, c := range []struct {
		name, ctl, rule, want string
	}{
		{"event gate", ctl, "", toolcontrols.DenialCodeSpendLimit},
		{"argument rule cap", "", rule, argrules.DenialCode},
	} {
		t.Run(c.name, func(t *testing.T) {
			h := newEvHarness(t, "", "", nil)
			schema := fmt.Sprintf("closingprice_%d_v1", atomic.AddInt64(&gateSchemaSeq, 1))
			if err := argrules.RegisterEventSchema(argrules.EventSchema{Name: schema, Fields: gateSchemaFields, Read: closingPriceRead}); err != nil {
				t.Fatal(err)
			}
			if err := RegisterEventMarketResolver(schema, h.res); err != nil {
				t.Fatal(err)
			}
			h.proxy.SetCallGate(ChainGates(NewEventGate(h.store, map[string]string{schema: "venue"}), newTestSpendGate(h.store), NewApprovalGate(nil, nil)))
			svc, err := macaroon.NewService(argTestRoot)
			if err != nil {
				t.Fatal(err)
			}
			if c.ctl != "" {
				doc := strings.ReplaceAll(c.ctl, "SCHEMA", schema)
				word := controlWord(t, doc)
				d, err := toolcontrols.ParseScopeValue(strings.TrimPrefix(word, "scope = "))
				if err != nil {
					t.Fatal(err)
				}
				h.ctl = d
				h.token = appendCaveat(t, svc, h.token, word)
			}
			if c.rule != "" {
				h.token = appendCaveat(t, svc, h.token, ruleScopeWord(t, strings.ReplaceAll(c.rule, "SCHEMA", schema)))
			}
			for _, p := range []string{"2", "1", "1.0", "0", "-0.5"} {
				if r := h.call(t, closing(p)); code(t, r) != c.want {
					t.Fatalf("closing order at price %s: code %q, want %q", p, code(t, r), c.want)
				}
			}
			if h.rr.forwarded() != 0 {
				t.Fatalf("a refused closing order was forwarded (%d)", h.rr.forwarded())
			}
			if r := h.call(t, closing("0.5")); r.Error != nil {
				t.Fatalf("closing order at 0.5 refused: %+v", r.Error)
			}
			if h.rr.forwarded() != 1 {
				t.Fatalf("forwarded %d, want 1", h.rr.forwarded())
			}
			if c.ctl != "" && h.stake(t, 0, "EV-A") != 0 {
				t.Fatal("a closing order was counted")
			}
		})
	}
}
