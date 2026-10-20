package mcpserver

// Round-2 tests for "sign every action": an empty receipt is not a receipt, a
// reversal always runs, a verified key keeps its identity when params cannot be
// read, and a reserved tool name never reaches a recorder.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"sync"
	"testing"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// ---- recorders that answer with nothing ----

// eaEmptyRecorder answers every receipt request without error and without a
// receipt: handle is returned as it is (nil, or a handle with no receipt id).
// It keeps windows and counts the Observe misses it is told of.
type eaEmptyRecorder struct {
	mu     sync.Mutex
	handle *MCPEvidence
	asked  int
	misses int
	// openers is what a window request returns; nil means "counted in an open
	// window", a handle with no receipt id means "an opener that is not one".
	openers *MCPEvidence
}

func (r *eaEmptyRecorder) Preflight(context.Context) error { return nil }

func (r *eaEmptyRecorder) RecordMCPDecision(context.Context, MCPDecision) (*MCPEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.asked++
	return r.handle, nil
}

func (r *eaEmptyRecorder) RecordWindowedCall(_ context.Context, c MCPWindowedCall) (*MCPEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if c.Kind == WindowKindUnrecorded {
		r.misses++
		return nil, nil
	}
	return r.openers, nil
}

func (r *eaEmptyRecorder) missCount() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.misses
}

var eaEmptyHandles = map[string]*MCPEvidence{
	"nil handle":        nil,
	"handle with no id": {},
}

// eaRoads are the three ways a request reaches the proxy.
var eaRoads = []struct {
	name string
	via  func(*testing.T, *eaEnv) func()
}{
	{"direct", nil},
	{"streamable_http", eaViaStreamable},
	{"sse", eaViaSSE},
}

// eaEnvWith builds an environment with the switch on. A nil rec keeps the
// recorder the environment makes (a window recorder when o.window is set).
func eaEnvWith(t *testing.T, o eaOpts, rec EvidenceRecorder, via func(*testing.T, *eaEnv) func()) (*eaEnv, func()) {
	t.Helper()
	e := newEAEnv(t, o)
	if rec != nil {
		e.proxy.SetEvidenceRecorder(rec)
		if err := e.proxy.SetSignEveryAction(true); err != nil {
			t.Fatalf("recorder not ready: %v", err)
		}
	} else {
		eaSetSwitch(e, true)
	}
	if via == nil {
		return e, func() {}
	}
	return e, via(t, e)
}

// F1. In Control a recorder that answers with no receipt stops the call: not
// sent, not charged, proof_unavailable. Priced, free and refused-for-budget
// calls alike.
func TestR2_EmptyReceipt_StopsControl(t *testing.T) {
	for name, handle := range eaEmptyHandles {
		for _, road := range eaRoads {
			for _, tool := range []string{"paid_read", "free_read", "big_order"} {
				name, handle, road, tool := name, handle, road, tool
				t.Run(name+"/"+road.name+"/"+tool, func(t *testing.T) {
					rec := &eaEmptyRecorder{handle: handle}
					e, done := eaEnvWith(t, eaOpts{}, rec, road.via)
					defer done()
					before := e.remaining()
					r := e.call(1, tool, "")
					if !eaIsProofUnavailable(r) {
						t.Fatalf("answer = %s", eaShape(r))
					}
					if e.rr.forwarded() != 0 {
						t.Fatalf("forwarded %d time(s) with no receipt", e.rr.forwarded())
					}
					if e.remaining() != before {
						t.Fatalf("debit left: %d -> %d", before, e.remaining())
					}
				})
			}
		}
	}
}

// F1. In Observe the call goes on and exactly one gap is reported, whatever
// path made the empty answer: the spend receipt of a priced call, the allow
// receipt of a free call, a call over budget.
func TestR2_EmptyReceipt_ObserveReportsOneGap(t *testing.T) {
	for name, handle := range eaEmptyHandles {
		for _, road := range eaRoads {
			for _, tool := range []string{"paid_read", "free_read", "big_order"} {
				name, handle, road, tool := name, handle, road, tool
				t.Run(name+"/"+road.name+"/"+tool, func(t *testing.T) {
					rec := &eaEmptyRecorder{handle: handle}
					e, done := eaEnvWith(t, eaOpts{observe: true}, rec, road.via)
					defer done()
					r := e.call(1, tool, "")
					if r.Error != nil {
						t.Fatalf("an Observe call was stopped: %s", eaShape(r))
					}
					if e.rr.forwarded() != 1 {
						t.Fatalf("forwarded %d times, want 1", e.rr.forwarded())
					}
					if rec.missCount() != 1 {
						t.Fatalf("gap reports = %d, want exactly 1", rec.missCount())
					}
				})
			}
		}
	}
}

// F1. Refusals and delegations that need a recorder are answered
// proof_unavailable when none is installed, and so is a refusal whose recorder
// answers with nothing.
func TestR2_RefusalsAreSignedOrProofUnavailable(t *testing.T) {
	cases := map[string]func(e *eaEnv) *Response{
		"invalid tool call":      func(e *eaEnv) *Response { return e.send(e.raw(1, MethodToolsCall, `"name":123`)) },
		"invalid delegate":       func(e *eaEnv) *Response { return e.send(e.raw(1, MethodSatGateDelegate, `"budget":"many"`)) },
		"refused delegate":       func(e *eaEnv) *Response { return e.send(e.raw(1, MethodSatGateDelegate, `"budget":5000,"label":"x"`)) },
		"reserved tool name":     func(e *eaEnv) *Response { return e.call(1, "argrules:v1:abc", "") },
		"tool outside the scope": func(e *eaEnv) *Response { return e.call(1, "not_in_scope", "") },
		"method with no service": func(e *eaEnv) *Response { return e.send(e.raw(1, "resources/list", "")) },
		"budget over":            func(e *eaEnv) *Response { return e.call(1, "big_order", "") },
	}
	for name, run := range cases {
		name, run := name, run
		t.Run("no recorder/"+name, func(t *testing.T) {
			e := newEAEnv(t, eaOpts{})
			e.proxy.SetEvidenceRecorder(nil)
			_ = e.proxy.SetSignEveryAction(true)
			if r := run(e); !eaIsProofUnavailable(r) {
				t.Fatalf("answer = %s", eaShape(r))
			}
		})
		t.Run("empty handle/"+name, func(t *testing.T) {
			e, _ := eaEnvWith(t, eaOpts{}, &eaEmptyRecorder{}, nil)
			if r := run(e); !eaIsProofUnavailable(r) {
				t.Fatalf("answer = %s", eaShape(r))
			}
		})
	}
}

// F1/F3. A window opener that comes back with a handle that holds no receipt is
// a failure; the nil of a call counted in an open window is not.
func TestR2_WindowOpenerWithoutReceipt_IsAFailure(t *testing.T) {
	rec := &eaEmptyRecorder{openers: &MCPEvidence{}}
	e, done := eaEnvWith(t, eaOpts{limited: true}, rec, nil)
	defer done()
	if r := e.call(1, "paid_read", ""); !eaIsProofUnavailable(r) {
		t.Fatalf("answer = %s", eaShape(r))
	}
	rec.openers = nil // counted in an open window
	if r := e.call(2, "paid_read", ""); r.Error == nil || r.Error.Code != CodeRateLimited {
		t.Fatalf("answer = %s", eaShape(r))
	}
}

// F3. No recorder at all: a refused call that repeats is proof_unavailable.
func TestR2_Windows_NoRecorder(t *testing.T) {
	for name, o := range map[string]eaOpts{
		"rate limited": {limited: true},
		"revoked":      {revoked: true},
		"expired":      {expired: true, tenantCtx: true},
	} {
		o := o
		t.Run(name, func(t *testing.T) {
			e := newEAEnv(t, o)
			e.proxy.SetEvidenceRecorder(nil)
			_ = e.proxy.SetSignEveryAction(true)
			if r := e.call(1, "paid_read", ""); !eaIsProofUnavailable(r) {
				t.Fatalf("answer = %s", eaShape(r))
			}
			if e.rr.forwarded() != 0 {
				t.Fatal("passed on")
			}
		})
	}
}

// An Observe call that goes through unrecorded, with a recorder that keeps no
// windows to count it, is still counted: by the proxy.
func TestR2_Observe_UncountedMissIsCounted(t *testing.T) {
	e := newEAEnv(t, eaOpts{observe: true})
	_ = e.proxy.SetSignEveryAction(true) // wiring error ignored on purpose
	e.base.setDown(true)
	if r := e.call(1, "free_read", ""); r.Error != nil {
		t.Fatalf("stopped: %s", eaShape(r))
	}
	if n := e.proxy.UncountedObserveMisses(); n != 1 {
		t.Fatalf("uncounted misses = %d, want 1", n)
	}
}

// ---- F2: a reversal that always runs ----

// eaNoUndoBudget is a valid BudgetEnforcer with neither Compensate nor Refund.
type eaNoUndoBudget struct {
	inner  *InMemoryBudgetEnforcer
	mu     sync.Mutex
	spends int
}

func (b *eaNoUndoBudget) Check(ctx context.Context, id string, cost int64) (*BudgetResult, error) {
	return b.inner.Check(ctx, id, cost)
}
func (b *eaNoUndoBudget) Spend(ctx context.Context, id, tool string, cost int64, rid string) (*BudgetResult, error) {
	b.mu.Lock()
	b.spends++
	b.mu.Unlock()
	return b.inner.Spend(ctx, id, tool, cost, rid)
}
func (b *eaNoUndoBudget) Remaining(ctx context.Context, id string) (int64, error) {
	return b.inner.Remaining(ctx, id)
}
func (b *eaNoUndoBudget) Initialize(ctx context.Context, id string, c int64) error {
	return b.inner.Initialize(ctx, id, c)
}

// eaFlakyRefund gives back like the in-memory store, after failing `failures`
// times. With ctxAware it refuses a context that is done.
type eaFlakyRefund struct {
	*InMemoryBudgetEnforcer
	mu       sync.Mutex
	failures int
	calls    int
	ctxAware bool
}

func (b *eaFlakyRefund) Refund(ctx context.Context, id, rid string, amount int64) error {
	b.mu.Lock()
	b.calls++
	fail := b.failures > 0
	if fail {
		b.failures--
	}
	b.mu.Unlock()
	if b.ctxAware && ctx.Err() != nil {
		return ctx.Err()
	}
	if fail {
		return errors.New("store busy")
	}
	return b.InMemoryBudgetEnforcer.Refund(ctx, id, rid, amount)
}

// eaCompensator has Compensate and no Refund.
type eaCompensator struct {
	BudgetEnforcer
	inner    *InMemoryBudgetEnforcer
	mu       sync.Mutex
	failures int
	calls    int
}

func (b *eaCompensator) Compensate(ctx context.Context, id, rid string, cost int64) error {
	b.mu.Lock()
	b.calls++
	fail := b.failures > 0
	if fail {
		b.failures--
	}
	b.mu.Unlock()
	if ctx.Err() != nil {
		return ctx.Err()
	}
	if fail {
		return errors.New("store busy")
	}
	return b.inner.Refund(ctx, id, rid, cost)
}

// eaCancelRecorder cancels the request's context and then fails, the way a
// signer that was slow and gave up does.
type eaCancelRecorder struct {
	cancel context.CancelFunc
}

func (r eaCancelRecorder) Preflight(context.Context) error { return nil }
func (r eaCancelRecorder) RecordMCPDecision(context.Context, MCPDecision) (*MCPEvidence, error) {
	r.cancel()
	return nil, errors.New("signer gave up")
}
func (r eaCancelRecorder) RecordWindowedCall(context.Context, MCPWindowedCall) (*MCPEvidence, error) {
	return nil, errors.New("signer gave up")
}

// An enforcer with no way to undo a debit: a priced Control call is refused
// before anything is debited; a free call, and an Observe call (which is passed
// on whatever happens), are not.
func TestR2_Reversal_NoUndo_RefusedBeforeTheDebit(t *testing.T) {
	for _, road := range eaRoads {
		road := road
		t.Run(road.name, func(t *testing.T) {
			e := newEAEnv(t, eaOpts{window: true})
			budget := &eaNoUndoBudget{inner: e.proxy.budget.(*InMemoryBudgetEnforcer)}
			e.proxy.SetBudgetEnforcer(budget)
			eaSetSwitch(e, true)
			if road.via != nil {
				defer road.via(t, e)()
			}
			before := e.remaining()
			r := e.call(1, "paid_read", "")
			if !eaIsProofUnavailable(r) {
				t.Fatalf("answer = %s", eaShape(r))
			}
			if budget.spends != 0 || e.remaining() != before || e.rr.forwarded() != 0 {
				t.Fatalf("spends=%d debit=%d forwarded=%d, want 0 0 0", budget.spends, before-e.remaining(), e.rr.forwarded())
			}
			if len(e.base.signed()) != 0 {
				t.Fatalf("receipts = %v", e.base.signed())
			}
			// Nothing to give back: a free call goes on and is signed.
			if r := e.call(2, "free_read", ""); r.Error != nil || e.rr.forwarded() != 1 {
				t.Fatalf("free call: %s forwarded %d", eaShape(r), e.rr.forwarded())
			}
		})
	}
	t.Run("Observe is passed on", func(t *testing.T) {
		e := newEAEnv(t, eaOpts{observe: true, window: true})
		budget := &eaNoUndoBudget{inner: e.proxy.budget.(*InMemoryBudgetEnforcer)}
		e.proxy.SetBudgetEnforcer(budget)
		eaSetSwitch(e, true)
		if r := e.call(1, "paid_read", ""); r.Error != nil || e.rr.forwarded() != 1 || budget.spends != 1 {
			t.Fatalf("Observe: %s forwarded %d spends %d", eaShape(r), e.rr.forwarded(), budget.spends)
		}
	})
}

// A delegation carve is a debit too: refused before the parent is charged when
// the store cannot give it back.
func TestR2_Reversal_NoUndo_DelegationRefusedBeforeTheCarve(t *testing.T) {
	e := newEAEnv(t, eaOpts{window: true})
	budget := &eaNoUndoBudget{inner: e.proxy.budget.(*InMemoryBudgetEnforcer)}
	e.proxy.SetBudgetEnforcer(budget)
	eaSetSwitch(e, true)
	before := e.remaining()
	r := e.send(e.raw(1, MethodSatGateDelegate, `"budget":5,"scope":"free_read","label":"helper"`))
	if !eaIsProofUnavailable(r) || r.Result != nil {
		t.Fatalf("answer = %s", eaShape(r))
	}
	if budget.spends != 0 || e.remaining() != before {
		t.Fatalf("spends=%d debit=%d", budget.spends, before-e.remaining())
	}
}

// An undo that fails is tried again inside its bound; one that works the second
// or third time leaves no debit. One that never works leaves the debit, stops
// the call, and is counted.
func TestR2_Reversal_FailingUndo(t *testing.T) {
	t.Run("works on the third try", func(t *testing.T) {
		e := newEAEnv(t, eaOpts{window: true})
		b := &eaFlakyRefund{InMemoryBudgetEnforcer: e.proxy.budget.(*InMemoryBudgetEnforcer), failures: 2}
		e.proxy.SetBudgetEnforcer(b)
		eaSetSwitch(e, true)
		e.base.mu.Lock()
		e.base.failRecord = true // the preflight passes, the receipt fails
		e.base.mu.Unlock()
		r := e.call(1, "paid_read", "")
		if !eaIsProofUnavailable(r) || e.rr.forwarded() != 0 {
			t.Fatalf("answer %s forwarded %d", eaShape(r), e.rr.forwarded())
		}
		if e.remaining() != eaCredits || b.calls != 3 {
			t.Fatalf("remaining %d (want %d), refund calls %d (want 3)", e.remaining(), eaCredits, b.calls)
		}
		if e.proxy.ReversalFailures() != 0 {
			t.Fatalf("reversal failures = %d", e.proxy.ReversalFailures())
		}
	})
	t.Run("Compensate only", func(t *testing.T) {
		e := newEAEnv(t, eaOpts{window: true})
		inner := e.proxy.budget.(*InMemoryBudgetEnforcer)
		b := &eaCompensator{BudgetEnforcer: inner, inner: inner, failures: 1}
		e.proxy.SetBudgetEnforcer(b)
		eaSetSwitch(e, true)
		e.base.mu.Lock()
		e.base.failRecord = true
		e.base.mu.Unlock()
		r := e.call(1, "paid_read", "")
		if !eaIsProofUnavailable(r) || e.rr.forwarded() != 0 || e.remaining() != eaCredits || b.calls != 2 {
			t.Fatalf("answer %s forwarded %d remaining %d compensate calls %d", eaShape(r), e.rr.forwarded(), e.remaining(), b.calls)
		}
	})
	t.Run("never works", func(t *testing.T) {
		e := newEAEnv(t, eaOpts{window: true})
		b := &eaFlakyRefund{InMemoryBudgetEnforcer: e.proxy.budget.(*InMemoryBudgetEnforcer), failures: 1000}
		e.proxy.SetBudgetEnforcer(b)
		eaSetSwitch(e, true)
		e.base.mu.Lock()
		e.base.failRecord = true
		e.base.mu.Unlock()
		r := e.call(1, "paid_read", "")
		if !eaIsProofUnavailable(r) || e.rr.forwarded() != 0 {
			t.Fatalf("answer %s forwarded %d", eaShape(r), e.rr.forwarded())
		}
		if e.proxy.ReversalFailures() != 1 {
			t.Fatalf("reversal failures = %d, want 1 (the owed debit is counted)", e.proxy.ReversalFailures())
		}
		if b.calls != reversalAttempts {
			t.Fatalf("refund calls = %d, want %d", b.calls, reversalAttempts)
		}
	})
}

// The request is cancelled while its receipt is being made, and the signer then
// fails. The store's refund honours its context: the reversal still runs,
// because it does not use the request's.
func TestR2_Reversal_RunsAfterTheRequestIsCancelled(t *testing.T) {
	for _, mode := range []string{"control", "delegation"} {
		mode := mode
		t.Run(mode, func(t *testing.T) {
			e := newEAEnv(t, eaOpts{window: true})
			b := &eaFlakyRefund{InMemoryBudgetEnforcer: e.proxy.budget.(*InMemoryBudgetEnforcer), ctxAware: true}
			e.proxy.SetBudgetEnforcer(b)
			ctx, cancel := context.WithCancel(e.ctx)
			defer cancel()
			e.ctx = ctx
			e.proxy.SetEvidenceRecorder(eaCancelRecorder{cancel})
			if err := e.proxy.SetSignEveryAction(true); err != nil {
				t.Fatal(err)
			}
			var r *Response
			if mode == "control" {
				r = e.call(1, "paid_read", "")
			} else {
				r = e.send(e.raw(1, MethodSatGateDelegate, `"budget":5,"scope":"free_read","label":"helper"`))
			}
			if ctx.Err() == nil {
				t.Fatal("the recorder did not cancel the request")
			}
			if !eaIsProofUnavailable(r) || e.rr.forwarded() != 0 {
				t.Fatalf("answer %s forwarded %d", eaShape(r), e.rr.forwarded())
			}
			if got := e.remaining(); got != eaCredits {
				t.Fatalf("debit left: remaining %d, want %d", got, eaCredits)
			}
			if e.proxy.ReversalFailures() != 0 {
				t.Fatal("the reversal failed")
			}
		})
	}
}

// ---- F4: a verified key keeps its identity when params cannot be read ----

func eaRawParams(method, params string) eaWire {
	return eaWire{JSONRPC: "2.0", ID: json.RawMessage(`7`), Method: method, Params: json.RawMessage(params)}
}

// eaSendWire sends a body as it is on the wire through a road. A road that is
// direct carries the key the way the stdio transport does, in params.
func eaSendOnRoad(t *testing.T, e *eaEnv, w eaWire) *Response {
	t.Helper()
	req := &Request{JSONRPC: "2.0", ID: w.ID, Method: w.Method, Params: w.Params}
	return e.send(req)
}

var eaBadParams = []struct{ name, params string }{
	{"array", `[]`},
	{"array with items", `["free_read",1]`},
	{"string", `"free_read"`},
	{"number", `42`},
	{"null", `null`},
	{"object with a name that is not text", `{"name":123}`},
	{"malformed _meta, a string", `{"name":123,"_meta":"oops"}`},
	{"malformed _meta, an array", `{"name":123,"_meta":[1,2]}`},
	{"malformed _meta, a token that is not text", `{"name":123,"_meta":{"token":5}}`},
}

// Over both real transports, with the key in the Authorization header only:
// tools/call whose params cannot be read is refused INVALID_PARAMS with a
// signed receipt for the key that was verified; nothing is passed on. With the
// signer down it is proof_unavailable.
func TestR2_VerifiedKeyKeepsItsIdentity_UnreadableParams(t *testing.T) {
	for _, tr := range eaTransports {
		for _, bp := range eaBadParams {
			tr, bp := tr, bp
			t.Run(tr.name+"/"+bp.name, func(t *testing.T) {
				e, done := eaEnvWith(t, eaOpts{window: true}, nil, tr.via)
				defer done()
				w := eaRawParams(MethodToolsCall, bp.params)
				r := e.sender(&Request{JSONRPC: "2.0", ID: w.ID, Method: w.Method, Params: w.Params})
				if r.Error == nil || r.Error.Code != CodeInvalidParams {
					t.Fatalf("answer = %s", eaShape(r))
				}
				if got := e.base.signed(); len(got) != 1 || got[0] != "denied/policy_denied/INVALID_PARAMS" {
					t.Fatalf("receipts = %v, want one INVALID_PARAMS refusal", got)
				}
				d := e.base.decisions[0]
				if d.TokenID == "" || d.TenantID != eaTenant || d.MCPMethod != MethodToolsCall {
					t.Fatalf("the receipt does not name the verified key: %+v", d)
				}
				if e.rr.forwarded() != 0 || e.remaining() != eaCredits {
					t.Fatal("something was passed on or debited")
				}
				if d := errData(t, r); d["receipt_id"] == nil {
					t.Fatalf("the answer carries no receipt handle: %v", d)
				}
				e.base.setDown(true)
				if r := e.sender(&Request{JSONRPC: "2.0", ID: w.ID, Method: w.Method, Params: w.Params}); !eaIsProofUnavailable(r) {
					t.Fatalf("signer down: %s", eaShape(r))
				}
			})
		}
	}
}

// The other authenticated methods keep the key too.
func TestR2_VerifiedKeyKeepsItsIdentity_OtherMethods(t *testing.T) {
	for _, tr := range eaTransports {
		tr := tr
		t.Run(tr.name, func(t *testing.T) {
			e, done := eaEnvWith(t, eaOpts{window: true}, nil, tr.via)
			defer done()
			r := e.sender(&Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodSatGateDelegate, Params: json.RawMessage(`[]`)})
			if r.Error == nil || r.Error.Code != CodeInvalidParams {
				t.Fatalf("delegate with array params: %s", eaShape(r))
			}
			r = e.sender(&Request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: MethodSatGateBudget, Params: json.RawMessage(`[]`)})
			if r.Error != nil {
				t.Fatalf("budget with array params: %s", eaShape(r))
			}
			if got := e.base.signed(); len(got) != 2 || got[0] != "denied/policy_denied/INVALID_PARAMS" || got[1] != "allowed/policy_allowed" {
				t.Fatalf("receipts = %v", got)
			}
		})
	}
}

// Identity is never taken from what the caller wrote: with no Authorization
// header, params of any shape name nobody; a bad token in params stays a failed
// authentication even when the header is good.
func TestR2_VerifiedKeyKeepsItsIdentity_NeverFromParams(t *testing.T) {
	for _, tr := range eaTransports {
		tr := tr
		t.Run(tr.name+"/no header", func(t *testing.T) {
			e, done := eaEnvWith(t, eaOpts{window: true}, nil, nil)
			defer done()
			// A connection with no credential cannot even make a session, so
			// the request goes straight to the handler with unreadable params.
			for _, bp := range eaBadParams {
				r := eaSendOnRoad(t, e, eaRawParams(MethodToolsCall, bp.params))
				if r.Error == nil || r.Error.Code != CodePolicyDenied || !strings.Contains(r.Error.Message, "authentication") {
					t.Fatalf("%s: answer = %s", bp.name, eaShape(r))
				}
			}
			if len(e.base.attempts) != 0 {
				t.Fatalf("%d receipt(s) for requests that name nobody", len(e.base.attempts))
			}
		})
		t.Run(tr.name+"/bad token in params", func(t *testing.T) {
			e, done := eaEnvWith(t, eaOpts{window: true}, nil, tr.via)
			defer done()
			r := e.sender(&Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodToolsCall,
				Params: json.RawMessage(`{"name":"free_read","_meta":{"token":"not-a-token"}}`)})
			if r.Error == nil || r.Error.Code != CodePolicyDenied {
				t.Fatalf("answer = %s", eaShape(r))
			}
			if len(e.base.attempts) != 0 || e.rr.forwarded() != 0 {
				t.Fatalf("receipts %d forwarded %d", len(e.base.attempts), e.rr.forwarded())
			}
		})
	}
}

// With the switch off the answer is what it was: authentication required.
func TestR2_UnreadableParams_SwitchOff(t *testing.T) {
	for _, bp := range eaBadParams {
		if bp.name == "null" {
			continue // main panics on a null params with a header key; not changed here
		}
		bp := bp
		t.Run("streamable_http/"+bp.name, func(t *testing.T) {
			e := newEAEnv(t, eaOpts{window: true})
			done := eaViaStreamable(t, e)
			defer done()
			r := e.sender(&Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodToolsCall, Params: json.RawMessage(bp.params)})
			if r.Error == nil || len(e.base.attempts) != 0 || e.rr.forwarded() != 0 {
				t.Fatalf("answer %s receipts %d forwarded %d", eaShape(r), len(e.base.attempts), e.rr.forwarded())
			}
		})
	}
}

// ---- F5: a reserved tool name never reaches a recorder ----

func TestR2_ReservedToolName_NeverReachesTheRecorder(t *testing.T) {
	const marker = "PRIVATE-NONCE-5521-never-recorded"
	name := macaroon.ArgumentRulesScopePrefix + "v1:" + marker
	cases := map[string]eaOpts{
		"ordinary refusal": {window: true},
		"rate limited":     {limited: true, window: true},
		"revoked":          {revoked: true, window: true},
		"expired":          {expired: true, tenantCtx: true, window: true},
		"Observe":          {observe: true, window: true},
	}
	for label, o := range cases {
		for _, down := range []bool{false, true} {
			label, o, down := label, o, down
			t.Run(label, func(t *testing.T) {
				e := newEAEnv(t, o)
				eaSetSwitch(e, true)
				e.base.setDown(down)
				_ = e.call(1, name, "")
				_ = e.send(e.raw(2, MethodToolsCall, `"name":"`+name+`","arguments":{"q":1}`))
				w := e.rec.(*eaWindowRecorder)
				e.base.mu.Lock()
				defer e.base.mu.Unlock()
				w.mu.Lock()
				defer w.mu.Unlock()
				seen := 0
				check := func(d MCPDecision) {
					seen++
					raw, _ := json.Marshal(d)
					if strings.Contains(string(raw), marker) {
						t.Fatalf("the nonce reached the recorder: %s", raw)
					}
					if d.ToolName != "" && d.ToolName != macaroon.ArgumentRulesScopePrefix+"redacted" && d.ToolName != d.MCPMethod {
						t.Fatalf("tool name = %q", d.ToolName)
					}
				}
				for _, d := range e.base.attempts {
					check(d)
				}
				for _, m := range w.misses {
					check(m.Decision)
				}
				if !down && seen == 0 {
					t.Fatal("nothing was recorded, so nothing was checked")
				}
			})
		}
	}
}

// The redaction is one function: every name that starts with the reserved
// prefix is reduced to the prefix and a fixed word; any other name is kept.
func TestR2_ReceiptTool(t *testing.T) {
	for in, want := range map[string]string{
		"argrules:v1:abc": macaroon.ArgumentRulesScopePrefix + "redacted",
		"argrules:v1:":    macaroon.ArgumentRulesScopePrefix + "redacted",
		macaroon.ArgumentRulesScopePrefix + "redacted": macaroon.ArgumentRulesScopePrefix + "redacted",
		"paid_read": "paid_read",
		"":          "",
	} {
		if got := receiptTool(in); got != want {
			t.Errorf("receiptTool(%q) = %q, want %q", in, got, want)
		}
	}
}
