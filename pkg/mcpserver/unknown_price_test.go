package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync/atomic"
	"testing"
)

// A tenant cost resolver that can say it does not know a price
// (CheckedTenantCostResolver). A paid call is never charged a guessed price:
// when the price is unknown the call is refused before any gate, debit,
// receipt or upstream dispatch. A resolver without the method behaves exactly
// as it always did.

// plainResolver implements only TenantCostResolver, as every resolver did
// before CheckedTenantCostResolver existed.
type plainResolver struct{ price int64 }

func (r plainResolver) ResolveForTenant(string, string) int64 { return r.price }

// checkedResolver implements both. err non-nil means "price unknown"; price is
// then the guess the resolver would have used.
type checkedResolver struct {
	plainResolver
	err     error
	checked atomic.Int64
	plain   atomic.Int64
}

func (r *checkedResolver) ResolveForTenant(t, tool string) int64 {
	r.plain.Add(1)
	return r.plainResolver.ResolveForTenant(t, tool)
}

func (r *checkedResolver) ResolveForTenantChecked(context.Context, string, string) (int64, error) {
	r.checked.Add(1)
	return r.price, r.err
}

type unknownPriceHarness struct {
	proxy  *Proxy
	router *recordingRouter
	budget *InMemoryBudgetEnforcer
	events *ChannelPublisher
	rec    *lockedRecorder
	gate   *countingGate
}

type countingGate struct{ admits, settles atomic.Int64 }

func (g *countingGate) Admit(context.Context, GateCall) (*GateAdmission, error) {
	g.admits.Add(1)
	return &GateAdmission{Settle: func(context.Context, GateOutcome) { g.settles.Add(1) }}, nil
}

func newUnknownPriceHarness(t *testing.T, mode string, r TenantCostResolver) *unknownPriceHarness {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "unknown-price", Version: "test"},
		Auth:        AuthConfig{Mode: "none"},
		Budget:      BudgetConfig{Limit: 100, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: mode},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := &unknownPriceHarness{proxy: proxy, router: &recordingRouter{}, budget: NewInMemoryBudgetEnforcer(), events: NewChannelPublisher(64), rec: &lockedRecorder{}, gate: &countingGate{}}
	proxy.SetUpstreamRouter(h.router)
	proxy.SetBudgetEnforcer(h.budget)
	proxy.SetEventPublisher(h.events)
	proxy.SetEvidenceRecorder(h.rec)
	proxy.SetCallGate(h.gate)
	proxy.SetTenantCostResolver(r)
	if err := h.budget.Initialize(context.Background(), "budget-1", 100); err != nil {
		t.Fatal(err)
	}
	return h
}

func (h *unknownPriceHarness) call(t *testing.T, id string) *Response {
	t.Helper()
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(id), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"priced_tool","arguments":{}}`)}
	resp, err := h.proxy.handleToolsCall(context.Background(), req, &TokenInfo{TokenID: "tok-1", BudgetID: "budget-1", TenantID: "tenant-1", Scope: "*", BudgetLimit: 100})
	if err != nil || resp == nil {
		t.Fatalf("handleToolsCall: %v %v", resp, err)
	}
	return resp
}

func (h *unknownPriceHarness) remaining(t *testing.T) int64 {
	t.Helper()
	n, err := h.budget.Remaining(context.Background(), "budget-1")
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func (h *unknownPriceHarness) spendEvents() (n int) {
	for {
		select {
		case e := <-h.events.Events():
			if e.Type == EventBudgetSpend {
				n++
			}
		default:
			return n
		}
	}
}

// A resolver without the new method: charged its price, one dispatch, the gate
// asked once and settled once. This is the behaviour before this change.
func TestPlainTenantResolverIsUnchanged(t *testing.T) {
	h := newUnknownPriceHarness(t, "control", plainResolver{price: 7})
	resp := h.call(t, `5`)
	if resp.Error != nil {
		t.Fatalf("refused: %+v", resp.Error)
	}
	if h.router.calls() != 1 || h.remaining(t) != 93 {
		t.Fatalf("dispatches=%d remaining=%d, want 1 and 93", h.router.calls(), h.remaining(t))
	}
	if h.gate.admits.Load() != 1 || h.gate.settles.Load() != 1 {
		t.Fatalf("gate admits=%d settles=%d, want 1 and 1", h.gate.admits.Load(), h.gate.settles.Load())
	}
}

// A resolver with the method that knows the price: charged it, through the
// checked method only.
func TestCheckedResolverWithAKnownPriceIsCharged(t *testing.T) {
	r := &checkedResolver{plainResolver: plainResolver{price: 7}}
	h := newUnknownPriceHarness(t, "control", r)
	if resp := h.call(t, `5`); resp.Error != nil {
		t.Fatalf("refused: %+v", resp.Error)
	}
	if h.router.calls() != 1 || h.remaining(t) != 93 {
		t.Fatalf("dispatches=%d remaining=%d, want 1 and 93", h.router.calls(), h.remaining(t))
	}
	if r.checked.Load() != 1 || r.plain.Load() != 0 {
		t.Fatalf("checked=%d plain=%d, want 1 and 0", r.checked.Load(), r.plain.Load())
	}
}

// A resolver with the method that does not know the price: refused in every
// mode that debits and blocks, before the gate, any debit, any receipt, any
// event and the upstream. The error is fixed text with the request's id and
// the error text of the resolver nowhere in it.
func TestCheckedResolverWithAnUnknownPriceIsRefused(t *testing.T) {
	for _, mode := range []string{"control", "hard", "soft", "fiat402"} {
		for _, id := range []string{`302`, `"req-x"`} {
			t.Run(mode+"/"+id, func(t *testing.T) {
				r := &checkedResolver{plainResolver: plainResolver{price: 1}, err: errors.New("dial tcp 10.1.2.3:6379: i/o timeout key satgate:mcp:cost_profile:tenant-1")}
				h := newUnknownPriceHarness(t, mode, r)
				resp := h.call(t, id)
				if resp.Error == nil {
					t.Fatalf("not refused: %s", resp.Result)
				}
				if string(resp.ID) != id {
					t.Fatalf("reply id %s, want %s", resp.ID, id)
				}
				if resp.Error.Code != CodeInternalError || resp.Error.Message != ToolPriceUnavailableMessage {
					t.Fatalf("error %d %q", resp.Error.Code, resp.Error.Message)
				}
				if resp.Error.Message == "Budget service unavailable" {
					t.Fatal("the message must differ from the budget outage message")
				}
				var data map[string]any
				if err := json.Unmarshal(resp.Error.Data, &data); err != nil {
					t.Fatal(err)
				}
				if data["error"] != "tool_price_unavailable" || data["retryable"] != true || data["tool"] != "priced_tool" || len(data) != 3 {
					t.Fatalf("data %v", data)
				}
				raw, _ := json.Marshal(resp)
				for _, bad := range []string{"10.1.2.3", "6379", "timeout", "satgate:mcp", "tenant-1", "budget-1", "dial"} {
					if strings.Contains(string(raw), bad) {
						t.Fatalf("reply carries %q: %s", bad, raw)
					}
				}
				if h.router.calls() != 0 {
					t.Fatalf("%d upstream dispatches, want 0", h.router.calls())
				}
				if got := h.remaining(t); got != 100 {
					t.Fatalf("remaining %d, want 100 (no debit)", got)
				}
				if h.gate.admits.Load() != 0 || h.gate.settles.Load() != 0 {
					t.Fatalf("gate admits=%d settles=%d, want 0 and 0", h.gate.admits.Load(), h.gate.settles.Load())
				}
				if n := h.spendEvents(); n != 0 {
					t.Fatalf("%d budget_spend events, want 0", n)
				}
				h.rec.mu.Lock()
				n := len(h.rec.decisions)
				h.rec.mu.Unlock()
				if n != 0 {
					t.Fatalf("%d receipts, want 0", n)
				}
			})
		}
	}
}

// Observe-class modes never block: the call goes on at the resolver's price.
func TestCheckedResolverWithAnUnknownPriceDoesNotBlockObserve(t *testing.T) {
	for _, mode := range []string{"observe", "shadow", "chargeback"} {
		t.Run(mode, func(t *testing.T) {
			r := &checkedResolver{plainResolver: plainResolver{price: 1}, err: errors.New("redis down")}
			h := newUnknownPriceHarness(t, mode, r)
			resp := h.call(t, `9`)
			if resp.Error != nil {
				t.Fatalf("observe refused: %+v", resp.Error)
			}
			if h.router.calls() != 1 {
				t.Fatalf("dispatches=%d, want 1", h.router.calls())
			}
		})
	}
}

// A token with no budget has nothing to debit; the unknown price decides
// nothing and the call is not refused (as before).
func TestCheckedResolverWithAnUnknownPriceAndNoBudgetIsNotRefused(t *testing.T) {
	r := &checkedResolver{plainResolver: plainResolver{price: 1}, err: errors.New("redis down")}
	h := newUnknownPriceHarness(t, "control", r)
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`4`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"priced_tool","arguments":{}}`)}
	resp, err := h.proxy.handleToolsCall(context.Background(), req, &TokenInfo{TokenID: "tok-1", TenantID: "tenant-1", Scope: "*"})
	if err != nil || resp == nil || resp.Error != nil {
		t.Fatalf("a call with no budget was refused: %v %+v", err, resp)
	}
	if h.router.calls() != 1 {
		t.Fatalf("dispatches=%d, want 1", h.router.calls())
	}
}

// A token with no tenant never reaches the tenant resolver.
func TestCheckedResolverIsNotAskedWithoutATenant(t *testing.T) {
	r := &checkedResolver{plainResolver: plainResolver{price: 7}, err: errors.New("redis down")}
	h := newUnknownPriceHarness(t, "control", r)
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`4`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"priced_tool","arguments":{}}`)}
	resp, err := h.proxy.handleToolsCall(context.Background(), req, &TokenInfo{TokenID: "tok-1", BudgetID: "budget-1", Scope: "*", BudgetLimit: 100})
	if err != nil || resp == nil || resp.Error != nil {
		t.Fatalf("refused: %v %+v", err, resp)
	}
	if r.checked.Load() != 0 || r.plain.Load() != 0 {
		t.Fatalf("tenant resolver asked: checked=%d plain=%d", r.checked.Load(), r.plain.Load())
	}
	if got := h.remaining(t); got != 99 { // the proxy's own cost resolver: default 1
		t.Fatalf("remaining %d, want 99", got)
	}
}
