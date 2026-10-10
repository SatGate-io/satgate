package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

const spendDoc200Day = `{"v":1,"spend_limits":[{"tools":["place_crypto_order"],"field":"dollar_amount","max":"200","window":"day"}]}`

// scriptedRouter answers each tools/call with the next scripted reply.
type scriptedRouter struct {
	mu      sync.Mutex
	replies []func() (*Response, error)
	n       int
	seen    []string
}

func (r *scriptedRouter) AllToolsForTenant(context.Context, string) []json.RawMessage { return nil }

func (r *scriptedRouter) ForwardToolCallForTenant(_ context.Context, _ string, _ string, params json.RawMessage, _ time.Duration) (*Response, error) {
	r.mu.Lock()
	i := r.n
	r.n++
	r.seen = append(r.seen, string(params))
	var f func() (*Response, error)
	if i < len(r.replies) {
		f = r.replies[i]
	}
	r.mu.Unlock()
	if f == nil {
		return okReply()
	}
	return f()
}

func (r *scriptedRouter) forwarded() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.n
}

func okReply() (*Response, error) {
	return &Response{JSONRPC: "2.0", Result: json.RawMessage(`{"content":[{"type":"text","text":"ok"}],"isError":false}`)}, nil
}
func isErrorReply() (*Response, error) {
	return &Response{JSONRPC: "2.0", Result: json.RawMessage(`{"content":[{"type":"text","text":"rejected"}],"isError":true}`)}, nil
}
func rpcErrorReply() (*Response, error) {
	return &Response{JSONRPC: "2.0", Error: &RPCError{Code: -32000, Message: "upstream said no"}}, nil
}
func forwardFails() (*Response, error) { return nil, errors.New("connection reset") }

func controlWord(t *testing.T, doc string) string {
	t.Helper()
	d, err := toolcontrols.ParseJSON([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	c, err := toolcontrols.Caveat(d)
	if err != nil {
		t.Fatal(err)
	}
	return c
}

type spendHarness struct {
	proxy *Proxy
	rr    *scriptedRouter
	store *toolcontrols.MemoryStore
	rec   *lockedRecorder
	token string
	now   *time.Time
}

func newSpendHarness(t *testing.T, doc string) *spendHarness {
	t.Helper()
	return newSpendHarnessScope(t, doc, "place_crypto_order,place_equity_order")
}

// newSpendHarnessScope is newSpendHarness for a token with the given scope.
func newSpendHarnessScope(t *testing.T, doc, scope string) *spendHarness {
	t.Helper()
	proxy, _, _ := newArgProxy(t)
	rr := &scriptedRouter{}
	proxy.SetUpstreamRouter(rr)
	rec := &lockedRecorder{}
	proxy.SetEvidenceRecorder(rec)
	store := toolcontrols.NewMemoryStore(nil)
	proxy.SetCallGate(NewSpendGate(store))
	svc, tok := mintBudgetToken(t, argTestRoot, scope, "tenant-1", "budget-spend", "50")
	tok = appendCaveat(t, svc, tok, controlWord(t, doc))
	return &spendHarness{proxy: proxy, rr: rr, store: store, rec: rec, token: tok}
}

func (h *spendHarness) call(t *testing.T, id int, tool, args string) *Response {
	t.Helper()
	resp, err := h.proxy.handleRequest(context.Background(), argCall(id, h.token, tool, args))
	if err != nil || resp == nil {
		t.Fatalf("handleRequest: %v %v", resp, err)
	}
	return resp
}

func order(amount string) string {
	return `{"side":"buy","type":"market","symbol":"BTC-USD","dollar_amount":"` + amount + `"}`
}

func errData(t *testing.T, r *Response) map[string]any {
	t.Helper()
	if r.Error == nil {
		t.Fatalf("not refused: %s", r.Result)
	}
	var d map[string]any
	if err := json.Unmarshal(r.Error.Data, &d); err != nil {
		t.Fatalf("data: %v", err)
	}
	return d
}

func (h *spendHarness) counter(t *testing.T) int64 {
	t.Helper()
	var total int64
	for _, k := range h.store.Keys() {
		total += h.store.Total(k)
	}
	return total
}

const units = 100000000

func TestSpendLimitReservesAndRefusesOverTheLimit(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	if r := h.call(t, 1, "place_crypto_order", order("150")); r.Error != nil {
		t.Fatalf("first order refused: %v", r.Error)
	}
	if h.counter(t) != 150*units {
		t.Fatalf("counter = %d", h.counter(t))
	}
	r := h.call(t, 2, "place_crypto_order", order("51"))
	if r.Error == nil || r.Error.Code != CodePolicyDenied {
		t.Fatalf("over-limit order was not refused: %+v", r)
	}
	d := errData(t, r)
	if d["error"] != "TOOL_SPEND_LIMIT_DENIED" || d["tool"] != "place_crypto_order" || d["field"] != "dollar_amount" ||
		d["window"] != "day" || d["limit"] != "200" || d["receipt_id"] == "" || d["evidence_url"] == "" {
		t.Fatalf("data = %v", d)
	}
	if want := "place_crypto_order: this token's limit is $200 a day on dollar_amount; this order would go over it"; r.Error.Message != want {
		t.Fatalf("message = %q", r.Error.Message)
	}
	if h.rr.forwarded() != 1 {
		t.Fatalf("forwarded = %d", h.rr.forwarded())
	}
	// Exactly up to the limit is allowed.
	if r := h.call(t, 3, "place_crypto_order", order("50")); r.Error != nil {
		t.Fatalf("order that exactly reaches the limit refused: %v", r.Error)
	}
	if r := h.call(t, 4, "place_crypto_order", order("0.01")); r.Error == nil {
		t.Fatal("order after the limit was reached was allowed")
	}
	// A tool the limit does not cover is not counted or refused.
	if r := h.call(t, 5, "place_equity_order", order("9999")); r.Error != nil {
		t.Fatalf("uncovered tool refused: %v", r.Error)
	}
}

// The refusal and its receipt never carry the amount the agent sent.
func TestSpendRefusalDoesNotEchoTheAmount(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	r := h.call(t, 1, "place_crypto_order", order("777.31"))
	if r.Error == nil {
		t.Fatal("not refused")
	}
	raw, _ := json.Marshal(r)
	if strings.Contains(string(raw), "777") {
		t.Fatalf("refusal echoes the amount: %s", raw)
	}
	for _, d := range h.rec.decisions {
		b, _ := json.Marshal(d)
		if strings.Contains(string(b), "777") {
			t.Fatalf("receipt input carries the amount: %s", b)
		}
	}
	if len(h.rec.decisions) != 1 {
		t.Fatalf("decisions = %d", len(h.rec.decisions))
	}
	d := h.rec.decisions[0]
	if d.Decision != "denied" || d.DecisionReason != "policy_denied" || d.DenialCode != "TOOL_SPEND_LIMIT_DENIED" ||
		d.ArgumentField != "dollar_amount" || d.ControlWindow != "day" || d.ControlLimit != "200" || !d.BudgetNotEvaluated {
		t.Fatalf("decision = %+v", d)
	}
}

func TestSpendReleasedWhenUpstreamFails(t *testing.T) {
	cases := map[string]func() (*Response, error){
		"json-rpc error": rpcErrorReply,
		"isError result": isErrorReply,
		"forward fails":  forwardFails,
	}
	for name, reply := range cases {
		t.Run(name, func(t *testing.T) {
			h := newSpendHarness(t, spendDoc200Day)
			h.rr.replies = []func() (*Response, error){reply}
			h.call(t, 1, "place_crypto_order", order("200"))
			if h.counter(t) != 0 {
				t.Fatalf("counter after failed order = %d", h.counter(t))
			}
			// The whole allowance is still there.
			if r := h.call(t, 2, "place_crypto_order", order("200")); r.Error != nil {
				t.Fatalf("retry after failure refused: %v", r.Error)
			}
			if h.counter(t) != 200*units {
				t.Fatalf("counter = %d", h.counter(t))
			}
		})
	}
}

func TestSpendKeptWhenUpstreamSucceeds(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	h.call(t, 1, "place_crypto_order", order("120.5"))
	if h.counter(t) != 12050000000 {
		t.Fatalf("counter = %d", h.counter(t))
	}
	// An allowed call's receipt carries the window total after the call.
	var allowed *MCPDecision
	for i := range h.rec.decisions {
		if h.rec.decisions[i].Decision == "allowed" {
			allowed = &h.rec.decisions[i]
		}
	}
	// This token has a budget (cost 1), so an allowed receipt is written.
	if allowed == nil || allowed.ControlWindow != "day" || allowed.ControlLimit != "200" || allowed.ControlWindowTotal != "120.5" {
		t.Fatalf("allowed decision = %+v", allowed)
	}
}

func TestSpendFailsClosedOnMissingOrBadField(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	for i, args := range []string{
		`{"side":"buy"}`,
		`{"side":"buy","dollar_amount":null}`,
		`{"side":"buy","dollar_amount":"abc"}`,
		`{"side":"buy","dollar_amount":"1e2"}`,
		`{"side":"buy","dollar_amount":true}`,
		`{"side":"buy","dollar_amount":["5"]}`,
		`{"side":"buy","dollar_amount":"-5"}`,
		`{"side":"buy","Dollar_Amount":"5"}`,
		`{"side":"buy","dollar_amount":"5","dollar_amount":"1"}`,
	} {
		r := h.call(t, i+1, "place_crypto_order", args)
		if r.Error == nil {
			t.Errorf("accepted %s", args)
			continue
		}
		if errData(t, r)["error"] != "TOOL_SPEND_LIMIT_DENIED" {
			t.Errorf("%s: data = %s", args, r.Error.Data)
		}
	}
	// No arguments at all.
	if r := h.call(t, 99, "place_crypto_order", ""); r.Error == nil {
		t.Error("call without arguments was allowed")
	}
	if h.rr.forwarded() != 0 || h.counter(t) != 0 {
		t.Fatalf("forwarded %d counter %d", h.rr.forwarded(), h.counter(t))
	}
	// A JSON number is read by its literal.
	if r := h.call(t, 100, "place_crypto_order", `{"dollar_amount":25.5}`); r.Error != nil {
		t.Fatalf("number refused: %v", r.Error)
	}
	if h.counter(t) != 2550000000 {
		t.Fatalf("counter = %d", h.counter(t))
	}
}

func TestSpendFailsClosedWhenStoreIsDown(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	h.store.Err = toolcontrols.ErrStoreUnavailable
	r := h.call(t, 1, "place_crypto_order", order("5"))
	if r.Error == nil || h.rr.forwarded() != 0 {
		t.Fatalf("call went through a down store: %+v", r)
	}
	if errData(t, r)["error"] != "tool_controls_unavailable" {
		t.Fatalf("data = %s", r.Error.Data)
	}
	// An uncovered tool does not need the store.
	if r := h.call(t, 2, "place_equity_order", order("5")); r.Error != nil {
		t.Fatalf("uncovered tool refused while store down: %v", r.Error)
	}
	// No gate installed: a limited token is still refused, never sent unchecked.
	h.proxy.SetCallGate(nil)
	if r := h.call(t, 3, "place_crypto_order", order("5")); r.Error == nil {
		t.Fatal("limited token was forwarded with no gate installed")
	}
}

func TestSpendDuplicateRequestIDReservesOnce(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	if r := h.call(t, 7, "place_crypto_order", order("100")); r.Error != nil {
		t.Fatal(r.Error)
	}
	r := h.call(t, 7, "place_crypto_order", order("100")) // retry, same id and params
	if r.Error == nil || errData(t, r)["error"] != DenialDuplicateRequest {
		t.Fatalf("replay = %+v", r)
	}
	if h.counter(t) != 100*units || h.rr.forwarded() != 1 {
		t.Fatalf("counter %d forwarded %d", h.counter(t), h.rr.forwarded())
	}
	// The same id with different params is a different call.
	if r := h.call(t, 7, "place_crypto_order", order("60")); r.Error != nil {
		t.Fatalf("same id, different order refused: %v", r.Error)
	}
	if h.counter(t) != 160*units {
		t.Fatalf("counter = %d", h.counter(t))
	}
	// A replay of a call that failed upstream is not a duplicate: it was released.
	h2 := newSpendHarness(t, spendDoc200Day)
	h2.rr.replies = []func() (*Response, error){isErrorReply}
	h2.call(t, 1, "place_crypto_order", order("100"))
	if r := h2.call(t, 1, "place_crypto_order", order("100")); r.Error != nil {
		t.Fatalf("retry of a failed call refused: %v", r.Error)
	}
	if h2.counter(t) != 100*units {
		t.Fatalf("counter = %d", h2.counter(t))
	}
}

func TestSpendRequestWithoutIDIsNeverADuplicate(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	for i := 0; i < 2; i++ {
		req := argCall(0, h.token, "place_crypto_order", order("10"))
		req.ID = nil
		resp, err := h.proxy.handleRequest(context.Background(), req)
		if err != nil || resp.Error != nil {
			t.Fatalf("call %d: %v %v", i, resp, err)
		}
	}
	if h.counter(t) != 20*units {
		t.Fatalf("counter = %d", h.counter(t))
	}
}

func TestSpendWindowRollover(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	base := time.Date(2026, 10, 9, 12, 0, 0, 0, time.UTC)
	clock := base
	// The store expires counters by its own clock; give it the test's, so the
	// test does not depend on today's date.
	h.store = toolcontrols.NewMemoryStore(func() time.Time { return clock })
	h.proxy.SetCallGate(NewSpendGate(h.store))
	h.proxy.gate.(*SpendGate).now = func() time.Time { return clock }
	if r := h.call(t, 1, "place_crypto_order", order("200")); r.Error != nil {
		t.Fatal(r.Error)
	}
	if r := h.call(t, 2, "place_crypto_order", order("1")); r.Error == nil {
		t.Fatal("limit not enforced inside the day")
	}
	// 23:59 New York, same day.
	clock = time.Date(2026, 10, 10, 3, 59, 0, 0, time.UTC)
	if r := h.call(t, 3, "place_crypto_order", order("1")); r.Error == nil {
		t.Fatal("window rolled over before local midnight")
	}
	// Midnight New York.
	clock = time.Date(2026, 10, 10, 4, 0, 0, 0, time.UTC)
	if r := h.call(t, 4, "place_crypto_order", order("200")); r.Error != nil {
		t.Fatalf("new day refused: %v", r.Error)
	}
}

func TestSpendWeekWindowAndTwoLimits(t *testing.T) {
	doc := `{"v":1,"spend_limits":[
	  {"tools":["place_crypto_order"],"field":"dollar_amount","max":"100","window":"day"},
	  {"tools":["place_crypto_order"],"field":"dollar_amount","max":"150","window":"week"}]}`
	h := newSpendHarness(t, doc)
	clock := time.Date(2026, 10, 5, 15, 0, 0, 0, time.UTC) // Monday
	h.store = toolcontrols.NewMemoryStore(func() time.Time { return clock })
	h.proxy.SetCallGate(&SpendGate{store: h.store, now: func() time.Time { return clock }})
	if r := h.call(t, 1, "place_crypto_order", order("100")); r.Error != nil {
		t.Fatal(r.Error)
	}
	clock = clock.Add(24 * time.Hour) // Tuesday: day is fresh, week has 50 left
	r := h.call(t, 2, "place_crypto_order", order("60"))
	if r.Error == nil || errData(t, r)["window"] != "week" {
		t.Fatalf("week limit not enforced: %+v", r)
	}
	// The refused call left nothing behind in the day counter either.
	if h.counter(t) != 100*units {
		t.Fatalf("counter = %d, want only the first order", h.counter(t))
	}
	if r := h.call(t, 3, "place_crypto_order", order("50")); r.Error != nil {
		t.Fatalf("order within both limits refused: %v", r.Error)
	}
	clock = clock.Add(6 * 24 * time.Hour) // next Monday
	if r := h.call(t, 4, "place_crypto_order", order("100")); r.Error != nil {
		t.Fatalf("new week refused: %v", r.Error)
	}
}

// N parallel calls never take more than the limit.
func TestSpendConcurrentCallsNeverExceedMax(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	const n = 64
	var ok int64
	var wg sync.WaitGroup
	for i := 0; i < n; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			resp, err := h.proxy.handleRequest(context.Background(), argCall(1000+i, h.token, "place_crypto_order", order("30")))
			if err == nil && resp.Error == nil {
				atomic.AddInt64(&ok, 30)
			}
		}(i)
	}
	wg.Wait()
	if ok > 200 || ok != 180 || h.counter(t) != ok*units || int64(h.rr.forwarded())*30 != ok {
		t.Fatalf("allowed %d dollars, counter %d, forwarded %d", ok, h.counter(t), h.rr.forwarded())
	}
}

func TestSpendTokensDoNotShareACounter(t *testing.T) {
	a := newSpendHarness(t, spendDoc200Day)
	b := newSpendHarness(t, spendDoc200Day)
	b.store = a.store
	b.proxy.SetCallGate(NewSpendGate(a.store))
	a.call(t, 1, "place_crypto_order", order("200"))
	if r := b.call(t, 1, "place_crypto_order", order("200")); r.Error != nil {
		t.Fatalf("a second token shares the first one's counter: %v", r.Error)
	}
}

// A token delegated from a limited one carries the same word, so it spends
// from the same counter.
func TestSpendDelegatedChildSharesTheCounter(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	svc, _ := macaroon.NewService(argTestRoot)
	child := appendCaveat(t, svc, h.token, "scope = place_crypto_order")
	h.call(t, 1, "place_crypto_order", order("200"))
	resp, _ := h.proxy.handleRequest(context.Background(), argCall(2, child, "place_crypto_order", order("1")))
	if resp.Error == nil {
		t.Fatal("a child token got a fresh allowance")
	}
}

// A token carrying a control a plain verifier cannot enforce is refused by it.
func TestPlainVerifierRefusesControlWord(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	svc, err := macaroon.NewService(argTestRoot)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := svc.Verify(h.token); err == nil {
		t.Fatal("a verifier that does not enforce controls accepted the token")
	}
	if _, err := svc.AcceptingArgumentRules().Verify(h.token); err != nil {
		t.Fatalf("enforcing verifier refused: %v", err)
	}
}

func TestSpendRulesAreCheckedFirst(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	svc, _ := macaroon.NewService(argTestRoot)
	h.token = appendCaveat(t, svc, h.token, ruleScopeWord(t, robinhoodCapDoc2))
	r := h.call(t, 1, "place_crypto_order", `{"dollar_amount":"300"}`)
	if r.Error == nil || errData(t, r)["error"] != "TOOL_ARGUMENT_DENIED" {
		t.Fatalf("argument rule should refuse first: %+v", r)
	}
	if h.counter(t) != 0 {
		t.Fatalf("counter = %d", h.counter(t))
	}
}

const robinhoodCapDoc2 = `{"v":1,"rules":[{"tool":"place_crypto_order","shapes":[{"dollar_amount":{"max":"100"}}]}]}`

func TestChainGatesSettlesEarlierGatesWhenALaterOneRefuses(t *testing.T) {
	var log []string
	mk := func(name string, refuse bool) CallGate {
		return gateFunc(func(ctx context.Context, c GateCall) (*GateAdmission, error) {
			adm := &GateAdmission{Settle: func(_ context.Context, o GateOutcome) {
				log = append(log, fmt.Sprintf("%s:%v", name, o.Succeeded))
			}}
			if refuse {
				adm.Refusal = &GateRefusal{Code: "X", Message: "m"}
			}
			return adm, nil
		})
	}
	g := ChainGates(mk("a", false), mk("b", true))
	adm, err := g.Admit(context.Background(), GateCall{})
	if err != nil || adm.Refusal == nil {
		t.Fatalf("%v %v", adm, err)
	}
	adm.Settle(context.Background(), GateOutcome{})
	if strings.Join(log, ",") != "a:false,b:false" {
		t.Fatalf("settle log = %v", log)
	}
	if ChainGates(nil, nil) != nil {
		t.Fatal("empty chain is not nil")
	}
}

type gateFunc func(context.Context, GateCall) (*GateAdmission, error)

func (f gateFunc) Admit(ctx context.Context, c GateCall) (*GateAdmission, error) { return f(ctx, c) }

// lockedRecorder is fakeEvidenceRecorder safe for parallel calls.
type lockedRecorder struct {
	mu        sync.Mutex
	decisions []MCPDecision
}

func (r *lockedRecorder) Preflight(context.Context) error { return nil }

func (r *lockedRecorder) RecordMCPDecision(_ context.Context, d MCPDecision) (*MCPEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.decisions = append(r.decisions, d)
	return &MCPEvidence{ReceiptID: fmt.Sprintf("rcpt_%d", len(r.decisions)), EvidenceURL: "https://issuer.example/v1/evidence/evid_test"}, nil
}

// An amount finer than the counter's unit is rounded up, never down.
func TestSpendRoundsFractionsUp(t *testing.T) {
	h := newSpendHarness(t, spendDoc200Day)
	if r := h.call(t, 1, "place_crypto_order", `{"dollar_amount":"0.123456781"}`); r.Error != nil {
		t.Fatal(r.Error)
	}
	if h.counter(t) != 12345679 {
		t.Fatalf("counter = %d", h.counter(t))
	}
}

// Round 2, Grok: a scope that allows a case variant of a limited tool (mcp:*
// allows everything) must not be a way round the limit.
func TestSpendCaseVariantToolNameIsCovered(t *testing.T) {
	h := newSpendHarnessScope(t, spendDoc200Day, "mcp:*")

	// Grok's exact scenario: limit 200/day, Place_Crypto_Order for 500.
	r := h.call(t, 1, "Place_Crypto_Order", order("500"))
	if got := errData(t, r)["error"]; got != toolcontrols.DenialCodeSpendLimit {
		t.Fatalf("500 on a case variant: error = %v, want %s", got, toolcontrols.DenialCodeSpendLimit)
	}
	if h.rr.forwarded() != 0 || h.counter(t) != 0 {
		t.Fatalf("refused call forwarded=%d counter=%d", h.rr.forwarded(), h.counter(t))
	}

	// A mixed-case call inside the limit is forwarded, and it reserves.
	if r := h.call(t, 2, "PLACE_crypto_ORDER", order("150")); r.Error != nil {
		t.Fatalf("150 on a case variant refused: %v", r.Error)
	}
	if h.rr.forwarded() != 1 || h.counter(t) != 150*units {
		t.Fatalf("after 150: forwarded=%d counter=%d", h.rr.forwarded(), h.counter(t))
	}
	// The canonical spelling shares the same counter.
	if r := h.call(t, 3, "place_crypto_order", order("100")); r.Error == nil {
		t.Fatal("100 more under a 200 limit was forwarded")
	}
	if h.rr.forwarded() != 1 {
		t.Fatalf("forwarded=%d", h.rr.forwarded())
	}
}

// Names the upstream might read as a covered tool but that are not plain:
// refused, never forwarded, on a token that carries a limit.
func TestSpendUnmatchableToolNamesAreRefused(t *testing.T) {
	names := map[string]string{
		"trailing space":   "place_crypto_order ",
		"leading space":    " place_crypto_order",
		"newline":          "place_crypto_order\n",
		"kelvin sign":      "place_crypto_order\u212a",
		"dotless i":        "place_crypto_order\u0131",
		"fullwidth":        "\uff50lace_crypto_order",
		"zero width":       "place_crypto\u200b_order",
		"combining":        "place_crypto_orde\u0072\u0301",
		"unrelated spaced": "get quote",
	}
	for label, name := range names {
		t.Run(label, func(t *testing.T) {
			h := newSpendHarnessScope(t, spendDoc200Day, "mcp:*")
			params := `{"name":"` + name + `","arguments":` + order("500") + `,"_meta":{"token":"` + h.token + `"}}`
			req := &Request{JSONRPC: "2.0", ID: json.RawMessage("1"), Method: MethodToolsCall, Params: json.RawMessage(params)}
			resp, err := h.proxy.handleRequest(context.Background(), req)
			if err != nil || resp == nil || resp.Error == nil {
				t.Fatalf("not refused: %+v %v", resp, err)
			}
			if h.rr.forwarded() != 0 {
				t.Fatalf("forwarded %d", h.rr.forwarded())
			}
		})
	}
}

// A top level that reads two ways is refused on a token with a limit, even for
// a tool the limit does not cover.
func TestSpendAmbiguousTopLevelIsRefused(t *testing.T) {
	h := newSpendHarnessScope(t, spendDoc200Day, "mcp:*")
	for label, params := range map[string]string{
		"folded name key": `{"name":"get_quote","Name":"place_crypto_order","arguments":` + order("500") + `}`,
		"repeated name":   `{"name":"get_quote","name":"place_crypto_order","arguments":` + order("500") + `}`,
	} {
		req := &Request{JSONRPC: "2.0", ID: json.RawMessage("7"), Method: MethodToolsCall, Params: json.RawMessage(params)}
		// handleRequest authenticates from _meta; add the token the way argCall does.
		req.Params = json.RawMessage(strings.TrimSuffix(params, "}") + `,"_meta":{"token":"` + h.token + `"}}`)
		resp, err := h.proxy.handleRequest(context.Background(), req)
		if err != nil || resp == nil || resp.Error == nil {
			t.Fatalf("%s: not refused: %+v %v", label, resp, err)
		}
	}
	if h.rr.forwarded() != 0 {
		t.Fatalf("forwarded %d", h.rr.forwarded())
	}
}

// An ordinary uncovered tool on a token with a limit still goes through.
func TestSpendUncoveredPlainToolStillForwarded(t *testing.T) {
	h := newSpendHarnessScope(t, spendDoc200Day, "mcp:*")
	if r := h.call(t, 1, "get_quote", `{"symbol":"BTC-USD"}`); r.Error != nil {
		t.Fatalf("uncovered tool refused: %v", r.Error)
	}
	if h.rr.forwarded() != 1 || h.counter(t) != 0 {
		t.Fatalf("forwarded=%d counter=%d", h.rr.forwarded(), h.counter(t))
	}
}
