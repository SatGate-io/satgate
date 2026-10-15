package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"sync/atomic"
	"testing"
)

// A host that serves many accounts from one Proxy can decide each tools/call
// under the mode of the account that made it (SetEnforcementModeResolver).
// With no resolver the Proxy decides every call under its process-wide mode,
// as it always did (TestPlainTenantResolverIsUnchanged and the rest of the
// suite cover that).

// modeBudget is an InMemoryBudgetEnforcer that notes the mode the Proxy put
// on the context of each Spend, so a store below the Proxy can be shown to
// read the same answer the Proxy decided under.
type modeBudget struct {
	*InMemoryBudgetEnforcer
	mu    sync.Mutex
	modes []string
	found []bool
}

func (b *modeBudget) Spend(ctx context.Context, tokenID, tool string, cost int64, requestID string) (*BudgetResult, error) {
	mode, ok := EnforcementModeFromContext(ctx)
	b.mu.Lock()
	b.modes = append(b.modes, mode)
	b.found = append(b.found, ok)
	b.mu.Unlock()
	return b.InMemoryBudgetEnforcer.Spend(ctx, tokenID, tool, cost, requestID)
}

type modeHarness struct {
	proxy    *Proxy
	router   *recordingRouter
	budget   *modeBudget
	events   *ChannelPublisher
	rec      *lockedRecorder
	resolved atomic.Int64
	modes    map[string]string // tenant -> mode the resolver answers
	mu       sync.Mutex
}

func newModeHarness(t *testing.T, processMode string, withResolver bool, modes map[string]string, prices TenantCostResolver) *modeHarness {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "per-call-mode", Version: "test"},
		Auth:        AuthConfig{Mode: "none"},
		Budget:      BudgetConfig{Limit: 100, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: processMode},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	h := &modeHarness{
		proxy:  proxy,
		router: &recordingRouter{},
		budget: &modeBudget{InMemoryBudgetEnforcer: NewInMemoryBudgetEnforcer()},
		events: NewChannelPublisher(64),
		rec:    &lockedRecorder{},
		modes:  modes,
	}
	proxy.SetUpstreamRouter(h.router)
	proxy.SetBudgetEnforcer(h.budget)
	proxy.SetEventPublisher(h.events)
	proxy.SetEvidenceRecorder(h.rec)
	if prices != nil {
		proxy.SetTenantCostResolver(prices)
	}
	if withResolver {
		proxy.SetEnforcementModeResolver(func(_ context.Context, tenantID string) string {
			h.resolved.Add(1)
			h.mu.Lock()
			defer h.mu.Unlock()
			return h.modes[tenantID]
		})
	}
	return h
}

func (h *modeHarness) setMode(tenant, mode string) {
	h.mu.Lock()
	h.modes[tenant] = mode
	h.mu.Unlock()
}

// fund gives a tenant a budget of limit credits.
func (h *modeHarness) fund(t *testing.T, tenant string, limit int64) {
	t.Helper()
	if err := h.budget.Initialize(context.Background(), "budget-"+tenant, limit); err != nil {
		t.Fatal(err)
	}
}

func (h *modeHarness) call(t *testing.T, tenant string, id string) *Response {
	t.Helper()
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(id), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"priced_tool","arguments":{}}`)}
	resp, err := h.proxy.handleToolsCall(context.Background(), req, &TokenInfo{TokenID: "tok-" + tenant, BudgetID: "budget-" + tenant, TenantID: tenant, Scope: "*", BudgetLimit: 100})
	if err != nil || resp == nil {
		t.Fatalf("handleToolsCall(%s): %v %v", tenant, resp, err)
	}
	return resp
}

func (h *modeHarness) remaining(t *testing.T, tenant string) int64 {
	t.Helper()
	n, err := h.budget.Remaining(context.Background(), "budget-"+tenant)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func (h *modeHarness) policyModes() (out []string) {
	h.rec.mu.Lock()
	defer h.rec.mu.Unlock()
	for _, d := range h.rec.decisions {
		out = append(out, d.TenantID+":"+d.Decision+":"+d.PolicyMode)
	}
	return out
}

// Two accounts, one Proxy. One chose observe, the other control. Each is
// decided under its own mode, whatever mode the process-wide setting holds
// and however it changes.
func TestResolverDecidesEachTenantUnderItsOwnMode(t *testing.T) {
	for _, processMode := range []string{"control", "observe"} {
		t.Run("process_"+processMode, func(t *testing.T) {
			h := newModeHarness(t, processMode, true, map[string]string{"obs": "observe", "ctl": "control"}, plainResolver{price: 7})
			h.fund(t, "obs", 5) // a call costs 7: every call overspends
			h.fund(t, "ctl", 5)

			// The observe account's overspend is allowed; the control account's is refused.
			if resp := h.call(t, "obs", `1`); resp.Error != nil {
				t.Fatalf("observe account refused: %+v", resp.Error)
			}
			resp := h.call(t, "ctl", `2`)
			if resp.Error == nil || resp.Error.Code != CodeBudgetExhausted {
				t.Fatalf("control account's overspend was not refused: %+v", resp)
			}
			if got := h.router.calls(); got != 1 {
				t.Fatalf("dispatches = %d, want 1 (the observe account's)", got)
			}

			// Moving the process-wide setting in either direction changes neither.
			for _, flip := range []string{"observe", "control", "observe"} {
				h.proxy.SetEnforcementMode(flip)
				if resp := h.call(t, "obs", `3`); resp.Error != nil {
					t.Fatalf("after process mode %s: observe account refused: %+v", flip, resp.Error)
				}
				resp := h.call(t, "ctl", `4`)
				if resp.Error == nil || resp.Error.Code != CodeBudgetExhausted {
					t.Fatalf("after process mode %s: control account's overspend was not refused: %+v", flip, resp)
				}
			}

			// A tenant's own change applies to that tenant at its next call.
			h.setMode("ctl", "observe")
			if resp := h.call(t, "ctl", `5`); resp.Error != nil {
				t.Fatalf("control account that moved to observe was refused: %+v", resp.Error)
			}
			h.setMode("obs", "control")
			if resp := h.call(t, "obs", `6`); resp.Error == nil || resp.Error.Code != CodeBudgetExhausted {
				t.Fatalf("observe account that moved to control was not refused: %+v", resp)
			}
		})
	}
}

// Every record a call leaves names the mode of the account that made it.
func TestResolverModeIsOnTheReceiptTheEventAndTheContext(t *testing.T) {
	h := newModeHarness(t, "control", true, map[string]string{"obs": "observe", "ctl": "control"}, plainResolver{price: 2})
	h.fund(t, "obs", 100)
	h.fund(t, "ctl", 100)
	if resp := h.call(t, "obs", `1`); resp.Error != nil {
		t.Fatal(resp.Error)
	}
	if resp := h.call(t, "ctl", `2`); resp.Error != nil {
		t.Fatal(resp.Error)
	}

	got := h.policyModes()
	want := []string{"obs:allowed:observe", "ctl:allowed:control"}
	if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
		t.Fatalf("receipt policy modes = %v, want %v", got, want)
	}

	// The spend events carry policy_mode and the tool-call events carry enforcement.
	seen := map[string]string{}
	for len(h.events.Events()) > 0 {
		e := <-h.events.Events()
		switch e.Type {
		case EventBudgetSpend:
			seen[e.TenantID+"/spend"], _ = e.Data["policy_mode"].(string)
		case EventToolCall:
			seen[e.TenantID+"/call"], _ = e.Data["enforcement"].(string)
		}
	}
	for key, want := range map[string]string{"obs/spend": "observe", "obs/call": "observe", "ctl/spend": "control", "ctl/call": "control"} {
		if seen[key] != want {
			t.Fatalf("%s = %q, want %q (all: %v)", key, seen[key], want, seen)
		}
	}

	// The store under the Proxy was handed the mode the Proxy decided under.
	if len(h.budget.modes) != 2 || h.budget.modes[0] != "observe" || h.budget.modes[1] != "control" || !h.budget.found[0] || !h.budget.found[1] {
		t.Fatalf("modes on the Spend contexts = %v (found %v), want [observe control]", h.budget.modes, h.budget.found)
	}
	// One answer per call: the resolver is asked once for each.
	if got := h.resolved.Load(); got != 2 {
		t.Fatalf("resolver asked %d times for 2 calls", got)
	}
}

// A mode the resolver cannot give is control: the call is blocked on
// exhaustion, never allowed through and never decided under the process-wide
// mode (here observe).
func TestResolverAnswerThatIsNotAModeIsControl(t *testing.T) {
	for name, answer := range map[string]string{"empty": "", "garbage": "banana", "charge": "charge", "unknown tenant": "-"} {
		t.Run(name, func(t *testing.T) {
			modes := map[string]string{"t1": answer}
			if answer == "-" {
				modes = map[string]string{} // the resolver has no answer for t1 and returns ""
			}
			h := newModeHarness(t, "observe", true, modes, plainResolver{price: 7})
			h.fund(t, "t1", 5)
			resp := h.call(t, "t1", `1`)
			if resp.Error == nil || resp.Error.Code != CodeBudgetExhausted {
				t.Fatalf("overspend under an unresolvable mode was not refused: %+v", resp)
			}
			if got := h.policyModes(); len(got) != 1 || got[0] != "t1:denied:control" {
				t.Fatalf("receipt = %v, want [t1:denied:control]", got)
			}
			if h.router.calls() != 0 {
				t.Fatalf("a refused call reached the upstream")
			}
		})
	}
}

// Legacy names the resolver may give are read as their canonical mode.
func TestResolverLegacyNamesAreAccepted(t *testing.T) {
	h := newModeHarness(t, "control", true, map[string]string{"a": "shadow", "b": "hard"}, plainResolver{price: 7})
	h.fund(t, "a", 5)
	h.fund(t, "b", 5)
	if resp := h.call(t, "a", `1`); resp.Error != nil {
		t.Fatalf("shadow refused: %+v", resp.Error)
	}
	if resp := h.call(t, "b", `2`); resp.Error == nil || resp.Error.Code != CodeBudgetExhausted {
		t.Fatalf("hard not refused: %+v", resp)
	}
}

// A budget store that fails: an account in observe goes on, an account in
// control is refused, whatever the process-wide mode says.
type failingBudget struct{ *InMemoryBudgetEnforcer }

func (failingBudget) Spend(context.Context, string, string, int64, string) (*BudgetResult, error) {
	return nil, errors.New("redis down")
}

func TestResolverFailModeFollowsTheTenantsMode(t *testing.T) {
	for _, processMode := range []string{"control", "observe"} {
		t.Run("process_"+processMode, func(t *testing.T) {
			h := newModeHarness(t, processMode, true, map[string]string{"obs": "observe", "ctl": "control"}, plainResolver{price: 1})
			h.fund(t, "obs", 100)
			h.fund(t, "ctl", 100)
			h.proxy.SetBudgetEnforcer(failingBudget{h.budget.InMemoryBudgetEnforcer})
			if resp := h.call(t, "obs", `1`); resp.Error != nil {
				t.Fatalf("observe account refused on a store error: %+v", resp.Error)
			}
			resp := h.call(t, "ctl", `2`)
			if resp.Error == nil {
				t.Fatalf("control account was let through a store error")
			}
		})
	}
}

// A price the resolver cannot read: the control account's call is refused
// before any charge; the observe account's goes on at the default price.
func TestResolverUnknownPriceFollowsTheTenantsMode(t *testing.T) {
	for _, processMode := range []string{"control", "observe"} {
		t.Run("process_"+processMode, func(t *testing.T) {
			prices := &checkedResolver{plainResolver: plainResolver{price: 1}, err: errors.New("price store down")}
			h := newModeHarness(t, processMode, true, map[string]string{"obs": "observe", "ctl": "control"}, prices)
			h.fund(t, "obs", 100)
			h.fund(t, "ctl", 100)

			resp := h.call(t, "ctl", `1`)
			if resp.Error == nil || resp.Error.Message != ToolPriceUnavailableMessage {
				t.Fatalf("control account's call at an unknown price was not refused: %+v", resp)
			}
			if got := h.remaining(t, "ctl"); got != 100 {
				t.Fatalf("control account charged %d at an unknown price", 100-got)
			}
			if resp := h.call(t, "obs", `2`); resp.Error != nil {
				t.Fatalf("observe account refused at an unknown price: %+v", resp.Error)
			}
		})
	}
}

// With no resolver installed nothing changes: the process-wide mode decides,
// the context carries no mode, and SetEnforcementMode still moves it.
func TestNoResolverKeepsTheProcessWideMode(t *testing.T) {
	h := newModeHarness(t, "observe", false, nil, plainResolver{price: 7})
	h.fund(t, "t1", 5)
	if resp := h.call(t, "t1", `1`); resp.Error != nil {
		t.Fatalf("observe refused: %+v", resp.Error)
	}
	h.proxy.SetEnforcementMode("control")
	if resp := h.call(t, "t1", `2`); resp.Error == nil || resp.Error.Code != CodeBudgetExhausted {
		t.Fatalf("control not enforced after SetEnforcementMode: %+v", resp)
	}
	for _, found := range h.budget.found {
		if found {
			t.Fatalf("a mode was put on the context with no resolver installed")
		}
	}
	// An observe overspend leaves no denial receipt; the control refusal does.
	if got := h.policyModes(); len(got) != 1 || got[0] != "t1:denied:control" {
		t.Fatalf("receipts = %v, want [t1:denied:control]", got)
	}
}
