package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// A resolver that panics decides the call as control. The panic does not
// leave tools/call, the over-budget call is refused like any control call,
// and the one log line says it happened without the panic value.

const panicSecret = "SECRET-PANIC-VALUE-tenant-4711-credential"

// captureLog routes the package logger to a buffer for the test.
func captureLog(t *testing.T) *lockedBuffer {
	t.Helper()
	buf := &lockedBuffer{}
	prev := log.Logger
	log.Logger = zerolog.New(buf).Level(zerolog.DebugLevel)
	t.Cleanup(func() { log.Logger = prev })
	return buf
}

type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

func TestResolverPanicDecidesControl(t *testing.T) {
	for name, boom := range map[string]func(){
		"a string":          func() { panic(panicSecret) },
		"an error":          func() { panic(errors.New(panicSecret)) },
		"a runtime error":   func() { var m map[string]string; m[panicSecret] = "x" },
		"a custom value":    func() { panic(struct{ Secret string }{panicSecret}) },
		"nil (PanicNilErr)": func() { panic(nil) },
	} {
		t.Run(name, func(t *testing.T) {
			logs := captureLog(t)
			// The process-wide mode is observe: a panic must not fall back to it.
			h := newModeHarness(t, "observe", true, nil, plainResolver{price: 7})
			h.fund(t, "t1", 5)
			h.proxy.SetEnforcementModeResolver(func(context.Context, string) string {
				boom()
				return "observe"
			})

			var resp *Response
			func() {
				defer func() {
					if r := recover(); r != nil {
						t.Fatalf("the resolver's panic escaped tools/call: %v", r)
					}
				}()
				resp = h.call(t, "t1", `1`)
			}()

			if resp.Error == nil || resp.Error.Code != CodeBudgetExhausted {
				t.Fatalf("over-budget call under a panicking resolver was not refused as control: %+v", resp)
			}
			if h.router.calls() != 0 {
				t.Fatalf("a refused call reached the upstream %d times", h.router.calls())
			}
			if got := h.remaining(t, "t1"); got != 5 {
				t.Fatalf("remaining = %d, want 5 (a refused call charges nothing)", got)
			}
			if got := h.policyModes(); len(got) != 1 || got[0] != "t1:denied:control" {
				t.Fatalf("receipt = %v, want [t1:denied:control]", got)
			}
			out := logs.String()
			if !strings.Contains(out, "enforcement mode resolver panicked; deciding the call as control") {
				t.Fatalf("the fixed log line is missing; log = %q", out)
			}
			if strings.Count(out, "resolver panicked") != 1 {
				t.Fatalf("want exactly one panic log line per call; log = %q", out)
			}
			if strings.Contains(out, panicSecret) || strings.Contains(out, "SECRET") {
				t.Fatalf("the log line carries the panic value: %q", out)
			}
		})
	}
}

// A funded call under a panicking resolver is a control call: it is debited
// and goes through. A panic in one call does not disturb the next call.
func TestResolverPanicThenAHealthyResolver(t *testing.T) {
	call := func(h *modeHarness, id string) *Response {
		t.Helper()
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("the resolver's panic escaped tools/call: %v", r)
			}
		}()
		return h.call(t, "t1", id)
	}
	captureLog(t)
	h := newModeHarness(t, "observe", true, map[string]string{"t1": "observe"}, plainResolver{price: 7})
	h.fund(t, "t1", 20)
	healthy := func(_ context.Context, tenant string) string { return "observe" }
	panicking := func(context.Context, string) string { panic(panicSecret) }

	h.proxy.SetEnforcementModeResolver(panicking)
	if resp := call(h, `1`); resp.Error != nil {
		t.Fatalf("funded control call refused: %+v", resp.Error)
	}
	if got := h.remaining(t, "t1"); got != 13 {
		t.Fatalf("remaining = %d, want 13: a control call is debited (observe would leave 20)", got)
	}

	h.proxy.SetEnforcementModeResolver(healthy)
	if resp := call(h, `2`); resp.Error != nil {
		t.Fatalf("observe call refused: %+v", resp.Error)
	}
	if got := h.policyModes(); len(got) != 2 || got[0] != "t1:allowed:control" || got[1] != "t1:allowed:observe" {
		t.Fatalf("receipts = %v, want the panicked call recorded as control and the next as observe", got)
	}
}

// The same through the dispatcher every transport calls (authentication, the
// context the transports build, then handleToolsCall).
func TestResolverPanicThroughTheDispatcher(t *testing.T) {
	logs := captureLog(t)
	h := newModeHarness(t, "observe", true, nil, plainResolver{price: 7})
	h.proxy.SetEnforcementModeResolver(func(context.Context, string) string { panic(panicSecret) })
	h.fund(t, "t1", 5)
	h.proxy.SetAuthenticator(modeAuth{})

	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`7`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"priced_tool","arguments":{},"_meta":{"token":"tok-t1"}}`)}
	var resp *Response
	var err error
	func() {
		defer func() {
			if r := recover(); r != nil {
				t.Fatalf("the resolver's panic escaped the dispatcher: %v", r)
			}
		}()
		resp, err = h.proxy.handleRequest(context.Background(), req)
	}()
	if err != nil || resp == nil || resp.Error == nil || resp.Error.Code != CodeBudgetExhausted {
		t.Fatalf("dispatcher: want a budget-exhausted refusal, got %+v, %v", resp, err)
	}
	if strings.Contains(logs.String(), "SECRET") {
		t.Fatalf("the log carries the panic value: %q", logs.String())
	}
}

type modeAuth struct{}

func (modeAuth) Verify(context.Context, string) (*TokenInfo, error) {
	return &TokenInfo{TokenID: "tok-t1", BudgetID: "budget-t1", TenantID: "t1", Scope: "*", BudgetLimit: 100}, nil
}

// Nothing in the Proxy needs the resolver's answer to be remembered: the
// resolver is asked exactly once per tools/call, on every path a call can
// take. (If a path asked twice, a mode that changed between the two asks
// could split one call across two modes.)
func TestResolverIsAskedOncePerCall(t *testing.T) {
	h := newModeHarness(t, "observe", true, map[string]string{"obs": "observe", "ctl": "control", "ctlfunded": "control"}, plainResolver{price: 7})
	h.fund(t, "obs", 5)
	h.fund(t, "ctl", 5)
	h.fund(t, "ctlfunded", 50)
	for i, tc := range []struct {
		tenant  string
		refused bool
	}{{"obs", false}, {"ctl", true}, {"ctlfunded", false}} {
		before := h.resolved.Load()
		resp := h.call(t, tc.tenant, []string{`1`, `2`, `3`}[i])
		if (resp.Error != nil) != tc.refused {
			t.Fatalf("%s: refused=%v, want %v: %+v", tc.tenant, resp.Error != nil, tc.refused, resp)
		}
		if got := h.resolved.Load() - before; got != 1 {
			t.Fatalf("%s: the resolver was asked %d times for one tools/call, want 1", tc.tenant, got)
		}
	}
}
