package mcpserver

// Round-3 tests for "sign every action".
//
//	R1  a refused call can never stay charged: a priced Control call and a
//	    delegation carve are held, signed, then committed; a hold that is never
//	    committed lapses by itself.
//	R2  a setup error cannot let calls through unsigned: until
//	    SetSignEveryAction has succeeded, every call answers proof_unavailable.
//	R3  a failed gap record keeps who and what it was for: it is kept by tenant,
//	    key, tool and minute and retried until it is recorded exactly once.

import (
	"context"
	"errors"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- test doubles ----

// eaClock is the clock of an in-memory budget store in a test.
type eaClock struct {
	mu sync.Mutex
	t  time.Time
}

func (c *eaClock) Now() time.Time {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.t
}

func (c *eaClock) advance(d time.Duration) {
	c.mu.Lock()
	c.t = c.t.Add(d)
	c.mu.Unlock()
}

// eaHoldStore is a budget store with holds, counting what it is asked and able
// to fail the commit and the release on every try.
type eaHoldStore struct {
	*InMemoryBudgetEnforcer
	mu                              sync.Mutex
	spends, holds                   int
	commits, releases               int
	failCommit, failRelease         bool
	releaseCtxAware                 bool
	holdDelay                       time.Duration
	holdTTLs                        []time.Duration
	lastHoldBudget, lastHoldRequest string
}

func (b *eaHoldStore) Spend(ctx context.Context, id, tool string, cost int64, rid string) (*BudgetResult, error) {
	b.mu.Lock()
	b.spends++
	b.mu.Unlock()
	return b.InMemoryBudgetEnforcer.Spend(ctx, id, tool, cost, rid)
}

func (b *eaHoldStore) Hold(ctx context.Context, id, tool string, cost int64, rid string, ttl time.Duration) (*BudgetResult, error) {
	b.mu.Lock()
	b.holds++
	b.holdTTLs = append(b.holdTTLs, ttl)
	b.lastHoldBudget, b.lastHoldRequest = id, rid
	d := b.holdDelay
	b.mu.Unlock()
	if d > 0 {
		time.Sleep(d)
	}
	return b.InMemoryBudgetEnforcer.Hold(ctx, id, tool, cost, rid, ttl)
}

func (b *eaHoldStore) CommitHold(ctx context.Context, id, rid string) error {
	b.mu.Lock()
	b.commits++
	fail := b.failCommit
	b.mu.Unlock()
	if fail {
		return errors.New("store busy")
	}
	return b.InMemoryBudgetEnforcer.CommitHold(ctx, id, rid)
}

func (b *eaHoldStore) ReleaseHold(ctx context.Context, id, rid string) error {
	b.mu.Lock()
	b.releases++
	fail := b.failRelease
	b.mu.Unlock()
	if b.releaseCtxAware && ctx.Err() != nil {
		return ctx.Err()
	}
	if fail {
		return errors.New("store busy")
	}
	return b.InMemoryBudgetEnforcer.ReleaseHold(ctx, id, rid)
}

func (b *eaHoldStore) counts() (spends, holds, commits, releases int) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.spends, b.holds, b.commits, b.releases
}

// eaNoHoldStore is a budget store with Spend and Refund and no hold. Only the
// methods of BudgetEnforcer and Refund are on it.
type eaNoHoldStore struct {
	inner  *InMemoryBudgetEnforcer
	mu     sync.Mutex
	spends int
}

func (b *eaNoHoldStore) Check(ctx context.Context, id string, cost int64) (*BudgetResult, error) {
	return b.inner.Check(ctx, id, cost)
}
func (b *eaNoHoldStore) Spend(ctx context.Context, id, tool string, cost int64, rid string) (*BudgetResult, error) {
	b.mu.Lock()
	b.spends++
	b.mu.Unlock()
	return b.inner.Spend(ctx, id, tool, cost, rid)
}
func (b *eaNoHoldStore) Remaining(ctx context.Context, id string) (int64, error) {
	return b.inner.Remaining(ctx, id)
}
func (b *eaNoHoldStore) Initialize(ctx context.Context, id string, c int64) error {
	return b.inner.Initialize(ctx, id, c)
}
func (b *eaNoHoldStore) Refund(ctx context.Context, id, rid string, amount int64) error {
	return b.inner.Refund(ctx, id, rid, amount)
}

// eaMissRecorder is a window recorder that can reject the reports of Observe
// misses, and counts what lands. A report lands once per ReportID: a report it
// has seen is not counted again.
type eaMissRecorder struct {
	*eaRecorder
	mu       sync.Mutex
	reject   int // reports to reject; negative means every one
	ctxAware bool
	// ambiguous makes the recorder count a report and then answer with an
	// error, once (the answer was lost on the way back).
	ambiguous bool
	attempts  int
	seen      map[string]bool
	landed    []MCPWindowedCall
	uncharged []PendingRecord
}

func (r *eaMissRecorder) RecordWindowedCall(ctx context.Context, call MCPWindowedCall) (*MCPEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.attempts++
	if r.ctxAware && ctx.Err() != nil {
		return nil, ctx.Err()
	}
	if call.Kind != WindowKindUnrecorded {
		return nil, nil
	}
	if r.reject != 0 {
		if r.reject > 0 {
			r.reject--
		}
		return nil, errors.New("window store rejected the report")
	}
	if r.seen == nil {
		r.seen = map[string]bool{}
	}
	if call.ReportID == "" || !r.seen[call.ReportID] {
		r.seen[call.ReportID] = true
		r.landed = append(r.landed, call)
	}
	if r.ambiguous {
		r.ambiguous = false
		return nil, errors.New("answer lost")
	}
	return nil, nil
}

func (r *eaMissRecorder) setReject(n int) {
	r.mu.Lock()
	r.reject = n
	r.mu.Unlock()
}

func (r *eaMissRecorder) landedCalls() []MCPWindowedCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]MCPWindowedCall(nil), r.landed...)
}

// RecordUncharged makes eaMissRecorder an UnchargedRecorder when sinkOn is set
// through eaMissSink.
type eaMissSink struct{ *eaMissRecorder }

func (s eaMissSink) RecordUncharged(_ context.Context, rec PendingRecord) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.uncharged = append(s.uncharged, rec)
	return nil
}

// eaSlowRecorder is a window recorder whose receipts take delay. With ignoreCtx
// it does not look at its context, as a signer that cannot be interrupted.
type eaSlowRecorder struct {
	*eaWindowRecorder
	delay     time.Duration
	ignoreCtx bool
	mu        sync.Mutex
	started   int
}

func (r *eaSlowRecorder) RecordMCPDecision(ctx context.Context, d MCPDecision) (*MCPEvidence, error) {
	r.mu.Lock()
	r.started++
	r.mu.Unlock()
	if r.ignoreCtx {
		time.Sleep(r.delay)
	} else {
		select {
		case <-time.After(r.delay):
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	return r.eaRecorder.RecordMCPDecision(context.Background(), d)
}

// eaCancelWindowRecorder cancels the request's context and then fails to sign.
// Its window reports honour their context, as a real store's do.
type eaCancelWindowRecorder struct {
	*eaMissRecorder
	cancel context.CancelFunc
}

func (r eaCancelWindowRecorder) RecordMCPDecision(context.Context, MCPDecision) (*MCPEvidence, error) {
	r.cancel()
	return nil, errors.New("signer gave up")
}

// ---- environments ----

// eaHold builds an environment whose budget store has holds, its clock under the
// test's control, with the switch on, carried over road.
func eaHold(t *testing.T, o eaOpts, road func(*testing.T, *eaEnv) func()) (*eaEnv, *eaHoldStore, *eaClock, func()) {
	t.Helper()
	e := newEAEnv(t, o)
	inner := e.proxy.budget.(*InMemoryBudgetEnforcer)
	clk := &eaClock{t: time.Now()}
	inner.now = clk.Now
	store := &eaHoldStore{InMemoryBudgetEnforcer: inner}
	e.proxy.SetBudgetEnforcer(store)
	eaSetSwitch(e, true)
	done := func() {}
	if road != nil {
		done = road(t, e)
	}
	return e, store, clk, done
}

const eaDelegateWire = `"budget":5,"scope":"free_read","label":"helper"`

type eaRoad = struct {
	name string
	via  func(*testing.T, *eaEnv) func()
}

// eaBoth runs fn for a tool call and for a delegation, over every road.
func eaBoth(t *testing.T, fn func(t *testing.T, road eaRoad, kind string)) {
	for _, road := range eaRoads {
		for _, kind := range []string{"tool call", "delegation"} {
			road, kind := road, kind
			t.Run(road.name+"/"+kind, func(t *testing.T) { fn(t, road, kind) })
		}
	}
}

// eaAct makes the call of the kind and returns the answer, and whether a
// forward or a key came out of it.
func eaAct(e *eaEnv, kind string, id int) (resp *Response, out int) {
	if kind == "tool call" {
		resp = e.call(id, "paid_read", "")
		return resp, e.rr.forwarded()
	}
	resp = e.send(e.raw(id, MethodSatGateDelegate, eaDelegateWire))
	if resp.Result != nil && resp.Error == nil && strings.Contains(string(resp.Result), `"token"`) {
		out = 1
	}
	return resp, out
}

// eaCost is what the call of the kind takes: 1 for paid_read, 5 for the carve.
func eaCost(kind string) int64 {
	if kind == "tool call" {
		return 1
	}
	return 5
}

// ---- R1 ----

// The reviewers' probe. The early check passes, the hold is taken, the receipt
// fails, and the release fails on every try. Nothing is sent or issued. The hold
// is still there at first (the amount is set aside), and once its bound has
// passed the balance is the starting balance: no later call had to succeed.
func TestR3_R1_ReceiptFailsAndReleaseFailsEveryTry(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		e, store, clk, done := eaHold(t, eaOpts{}, road.via)
		defer done()
		store.failRelease = true
		e.base.mu.Lock()
		e.base.failRecord = true // Preflight passes; the receipt fails
		e.base.mu.Unlock()
		start := e.remaining()

		resp, out := eaAct(e, kind, 1)
		if !eaIsProofUnavailable(resp) {
			t.Fatalf("answer = %s", eaShape(resp))
		}
		if out != 0 || e.rr.forwarded() != 0 {
			t.Fatalf("something came out: forwarded %d, out %d", e.rr.forwarded(), out)
		}
		spends, holds, commits, releases := store.counts()
		if spends != 0 || holds != 1 || commits != 0 {
			t.Fatalf("spends %d holds %d commits %d, want 0 1 0 (the amount is held, never debited)", spends, holds, commits)
		}
		if releases < 1 {
			t.Fatalf("the hold was not given back or even tried (%d)", releases)
		}
		if e.proxy.ReversalFailures() != 1 {
			t.Fatalf("release failures = %d, want 1", e.proxy.ReversalFailures())
		}
		if got := e.remaining(); got != start-eaCost(kind) {
			t.Fatalf("while the hold lives the amount is set aside: remaining %d, want %d", got, start-eaCost(kind))
		}
		// The hold's bound passes. Nothing else is called.
		before := releases
		clk.advance(e.proxy.EveryActionSettings().HoldTTL + time.Second)
		if got := e.remaining(); got != start {
			t.Fatalf("after the hold's bound the balance is %d, want the starting balance %d", got, start)
		}
		if _, _, c, r := store.counts(); c != 0 || r != before {
			t.Fatalf("the store was called again (commits %d, releases %d -> %d)", c, before, r)
		}
		if len(e.proxy.PendingRecords()) != 0 {
			t.Fatalf("a refused call left a pending record: %+v", e.proxy.PendingRecords())
		}
	})
}

// The receipt fails and the release works: the balance is back at once.
func TestR3_R1_ReceiptFailsAndReleaseWorks(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		e, store, _, done := eaHold(t, eaOpts{}, road.via)
		defer done()
		start := e.remaining()
		e.base.mu.Lock()
		e.base.failRecord = true
		e.base.mu.Unlock()
		resp, out := eaAct(e, kind, 1)
		if !eaIsProofUnavailable(resp) || out != 0 || e.rr.forwarded() != 0 {
			t.Fatalf("answer %s forwarded %d out %d", eaShape(resp), e.rr.forwarded(), out)
		}
		if got := e.remaining(); got != start {
			t.Fatalf("remaining %d, want %d", got, start)
		}
		if _, holds, commits, releases := store.counts(); holds != 1 || commits != 0 || releases != 1 {
			t.Fatalf("holds %d commits %d releases %d", holds, commits, releases)
		}
	})
}

// A release that fails twice and works the third time is enough.
func TestR3_R1_ReleaseRetried(t *testing.T) {
	e, store, _, done := eaHold(t, eaOpts{}, nil)
	defer done()
	flaky := &eaFlakyRelease{eaHoldStore: store, failures: 2}
	e.proxy.SetBudgetEnforcer(flaky)
	e.base.mu.Lock()
	e.base.failRecord = true
	e.base.mu.Unlock()
	start := e.remaining()
	if r := e.call(1, "paid_read", ""); !eaIsProofUnavailable(r) {
		t.Fatalf("answer = %s", eaShape(r))
	}
	if flaky.calls != 3 || e.remaining() != start || e.proxy.ReversalFailures() != 0 {
		t.Fatalf("release calls %d (want 3), remaining %d (want %d), failures %d", flaky.calls, e.remaining(), start, e.proxy.ReversalFailures())
	}
}

type eaFlakyRelease struct {
	*eaHoldStore
	failures int
	calls    int
}

func (b *eaFlakyRelease) ReleaseHold(ctx context.Context, id, rid string) error {
	b.calls++
	if b.failures > 0 {
		b.failures--
		return errors.New("store busy")
	}
	return b.InMemoryBudgetEnforcer.ReleaseHold(ctx, id, rid)
}

// The request is cancelled while its receipt is being made, and the signer then
// fails. The store's release honours its context: the release still runs,
// because it does not use the request's.
func TestR3_R1_ReleaseRunsAfterTheRequestIsCancelled(t *testing.T) {
	for _, kind := range []string{"tool call", "delegation"} {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			e, store, _, done := eaHold(t, eaOpts{}, nil)
			defer done()
			store.releaseCtxAware = true
			ctx, cancel := context.WithCancel(e.ctx)
			defer cancel()
			e.ctx = ctx
			e.proxy.SetEvidenceRecorder(eaCancelWindowRecorder{&eaMissRecorder{eaRecorder: e.base}, cancel})
			if err := e.proxy.SetSignEveryAction(true); err != nil {
				t.Fatal(err)
			}
			start := e.remaining()
			resp, out := eaAct(e, kind, 1)
			if ctx.Err() == nil {
				t.Fatal("the recorder did not cancel the request")
			}
			if !eaIsProofUnavailable(resp) || out != 0 || e.rr.forwarded() != 0 {
				t.Fatalf("answer %s forwarded %d", eaShape(resp), e.rr.forwarded())
			}
			if got := e.remaining(); got != start {
				t.Fatalf("remaining %d, want %d", got, start)
			}
			if e.proxy.ReversalFailures() != 0 {
				t.Fatal("the release failed")
			}
		})
	}
}

// The signature is made, the commit fails on every try. The signed call goes
// ahead: its receipt says allowed and refusing it would make the receipt false.
// The hold lapses, so the amount is never charged, and it is kept once with
// tenant, key, tool and minute.
func TestR3_R1_CommitFailsAfterSigning(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		e, store, clk, done := eaHold(t, eaOpts{}, road.via)
		defer done()
		store.failCommit = true
		start := e.remaining()

		resp, out := eaAct(e, kind, 1)
		if resp.Error != nil || out != 1 {
			t.Fatalf("the signed call did not go ahead: %s (out %d)", eaShape(resp), out)
		}
		if got := e.base.signed(); len(got) != 1 {
			t.Fatalf("receipts = %v, want the one allowed receipt", got)
		}
		if _, holds, commits, _ := store.counts(); holds != 1 || commits < 1 {
			t.Fatalf("holds %d commits %d", holds, commits)
		}
		clk.advance(e.proxy.EveryActionSettings().HoldTTL + time.Second)
		if got := e.remaining(); got != start {
			t.Fatalf("the hold lapsed, so nothing was charged: remaining %d, want %d", got, start)
		}
		recs := e.proxy.PendingRecords()
		if len(recs) != 1 {
			t.Fatalf("pending = %+v, want exactly one entry", recs)
		}
		r := recs[0]
		wantTool := "paid_read"
		wantMethod := MethodToolsCall
		if kind == "delegation" {
			wantTool, wantMethod = "", MethodSatGateDelegate
		}
		if r.Kind != PendingUncharged || r.TenantID != eaTenant || r.TokenID == "" || r.BudgetID != e.budgetID ||
			r.Tool != wantTool || r.Method != wantMethod || r.Amount != eaCost(kind) || r.RequestID == "" {
			t.Fatalf("pending entry = %+v", r)
		}
		if !r.Minute.Equal(RateWindowStart(time.Now())) && !r.Minute.Equal(RateWindowStart(time.Now().Add(-time.Minute))) {
			t.Fatalf("minute = %v", r.Minute)
		}
		// Not twice: a flush with no consumer changes nothing.
		if n := e.proxy.FlushPending(context.Background()); n != 1 {
			t.Fatalf("after a flush the store holds %d entries, want 1", n)
		}
		if !e.proxy.AckPending(r.ID) || len(e.proxy.PendingRecords()) != 0 {
			t.Fatal("the host could not take the entry")
		}
	})
}

// A commit that fails once and works on the second try is not a failure.
func TestR3_R1_CommitRetried(t *testing.T) {
	e, store, _, done := eaHold(t, eaOpts{}, nil)
	defer done()
	flaky := &eaFlakyCommit{eaHoldStore: store, failures: 1}
	e.proxy.SetBudgetEnforcer(flaky)
	start := e.remaining()
	if r := e.call(1, "paid_read", ""); r.Error != nil {
		t.Fatalf("answer = %s", eaShape(r))
	}
	if e.remaining() != start-1 || len(e.proxy.PendingRecords()) != 0 {
		t.Fatalf("remaining %d pending %v", e.remaining(), e.proxy.PendingRecords())
	}
}

type eaFlakyCommit struct {
	*eaHoldStore
	failures int
}

func (b *eaFlakyCommit) CommitHold(ctx context.Context, id, rid string) error {
	if b.failures > 0 {
		b.failures--
		return errors.New("store busy")
	}
	return b.InMemoryBudgetEnforcer.CommitHold(ctx, id, rid)
}

// A commit failure is handed to an UnchargedRecorder when the recorder has one,
// once, and the entry then leaves the store.
func TestR3_R1_UnchargedGoesToTheRecorderOnce(t *testing.T) {
	e, store, _, done := eaHold(t, eaOpts{}, nil)
	defer done()
	store.failCommit = true
	rec := &eaMissRecorder{eaRecorder: e.base}
	sink := eaMissSink{rec}
	e.proxy.SetEvidenceRecorder(sink)
	if err := e.proxy.SetSignEveryAction(true); err != nil {
		t.Fatal(err)
	}
	if err := e.proxy.SetEveryActionSettings(EveryActionSettings{PendingRetry: 5 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	if r := e.call(1, "paid_read", ""); r.Error != nil {
		t.Fatalf("answer = %s", eaShape(r))
	}
	eaWait(t, func() bool { return len(e.proxy.PendingRecords()) == 0 })
	time.Sleep(30 * time.Millisecond)
	rec.mu.Lock()
	defer rec.mu.Unlock()
	if len(rec.uncharged) != 1 || rec.uncharged[0].Amount != 1 || rec.uncharged[0].Tool != "paid_read" {
		t.Fatalf("recorder was given %+v, want the one uncharged call", rec.uncharged)
	}
}

func eaWait(t *testing.T, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(3 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(5 * time.Millisecond)
	}
	t.Fatal("timed out waiting")
}

// An enforcer that offers no hold: a priced Control call and a carve are
// refused before anything is debited. A free call, an Observe call and the
// switch off are not changed by it.
func TestR3_R1_NoHold_RefusedBeforeTheDebit(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		e := newEAEnv(t, eaOpts{})
		store := &eaNoHoldStore{inner: e.proxy.budget.(*InMemoryBudgetEnforcer)}
		e.proxy.SetBudgetEnforcer(store)
		eaSetSwitch(e, true)
		if road.via != nil {
			defer road.via(t, e)()
		}
		start := e.remaining()
		resp, out := eaAct(e, kind, 1)
		if !eaIsProofUnavailable(resp) || out != 0 || e.rr.forwarded() != 0 {
			t.Fatalf("answer %s forwarded %d", eaShape(resp), e.rr.forwarded())
		}
		if store.spends != 0 || e.remaining() != start {
			t.Fatalf("spends %d, remaining %d (want %d)", store.spends, e.remaining(), start)
		}
		if len(e.base.signed()) != 0 {
			t.Fatalf("receipts = %v", e.base.signed())
		}
		if kind == "tool call" {
			if r := e.call(2, "free_read", ""); r.Error != nil || e.rr.forwarded() != 1 {
				t.Fatalf("free call: %s forwarded %d", eaShape(r), e.rr.forwarded())
			}
		}
	})
	t.Run("Observe and switch off are as before", func(t *testing.T) {
		for _, on := range []bool{true, false} {
			e := newEAEnv(t, eaOpts{observe: on})
			store := &eaNoHoldStore{inner: e.proxy.budget.(*InMemoryBudgetEnforcer)}
			e.proxy.SetBudgetEnforcer(store)
			if on {
				eaSetSwitch(e, true)
			}
			if r := e.call(1, "paid_read", ""); r.Error != nil || e.rr.forwarded() != 1 || store.spends != 1 {
				t.Fatalf("observe=%v: %s forwarded %d spends %d", on, eaShape(r), e.rr.forwarded(), store.spends)
			}
		}
		// The switch off, Control: the old path, a plain Spend.
		e := newEAEnv(t, eaOpts{})
		store := &eaNoHoldStore{inner: e.proxy.budget.(*InMemoryBudgetEnforcer)}
		e.proxy.SetBudgetEnforcer(store)
		if r := e.call(1, "paid_read", ""); r.Error != nil || store.spends != 1 {
			t.Fatalf("switch off: %s spends %d", eaShape(r), store.spends)
		}
	})
}

// A signer that is too slow hits its time limit: the call is refused, the hold
// is released, and the answer comes back at the limit and not when the signer
// is done. A signer that cannot be interrupted is not waited for either.
func TestR3_R1_SlowSignerHitsItsTimeout(t *testing.T) {
	for _, ignore := range []bool{false, true} {
		ignore := ignore
		eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
			e, store, _, done := eaHold(t, eaOpts{}, road.via)
			defer done()
			slow := &eaSlowRecorder{eaWindowRecorder: e.rec.(*eaWindowRecorder), delay: 4 * time.Second, ignoreCtx: ignore}
			e.proxy.SetEvidenceRecorder(slow)
			if err := e.proxy.SetSignEveryAction(true); err != nil {
				t.Fatal(err)
			}
			if err := e.proxy.SetEveryActionSettings(EveryActionSettings{HoldTTL: 2 * time.Second, SignTimeout: 150 * time.Millisecond, CommitTimeout: 100 * time.Millisecond}); err != nil {
				t.Fatal(err)
			}
			start := e.remaining()
			began := time.Now()
			resp, out := eaAct(e, kind, 1)
			if took := time.Since(began); took > 1500*time.Millisecond {
				t.Fatalf("the answer took %v; the signing limit is 150ms", took)
			}
			if !eaIsProofUnavailable(resp) || out != 0 || e.rr.forwarded() != 0 {
				t.Fatalf("answer %s forwarded %d", eaShape(resp), e.rr.forwarded())
			}
			if got := e.remaining(); got != start {
				t.Fatalf("remaining %d, want %d (the hold is released)", got, start)
			}
			if _, holds, commits, releases := store.counts(); holds != 1 || commits != 0 || releases != 1 {
				t.Fatalf("holds %d commits %d releases %d", holds, commits, releases)
			}
		})
	}
}

// A signer that takes time but finishes inside its limit is signed and
// committed: the hold has not lapsed before the commit.
func TestR3_R1_SignerInsideTheLimitIsCommitted(t *testing.T) {
	e, store, _, done := eaHold(t, eaOpts{}, nil)
	defer done()
	slow := &eaSlowRecorder{eaWindowRecorder: e.rec.(*eaWindowRecorder), delay: 80 * time.Millisecond}
	e.proxy.SetEvidenceRecorder(slow)
	if err := e.proxy.SetSignEveryAction(true); err != nil {
		t.Fatal(err)
	}
	if err := e.proxy.SetEveryActionSettings(EveryActionSettings{HoldTTL: 2 * time.Second, SignTimeout: 500 * time.Millisecond, CommitTimeout: 200 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	start := e.remaining()
	if r := e.call(1, "paid_read", ""); r.Error != nil {
		t.Fatalf("answer = %s", eaShape(r))
	}
	if _, holds, commits, _ := store.counts(); holds != 1 || commits != 1 || e.remaining() != start-1 {
		t.Fatalf("holds %d commits %d remaining %d", holds, commits, e.remaining())
	}
}

// A hold that has too little time left to be signed under and committed is not
// signed under: it is released and the call refused. The signing limit is the
// smaller of its own and what is left of the hold's, minus the commit's.
func TestR3_R1_HoldTooOldToSignUnder(t *testing.T) {
	e, store, _, done := eaHold(t, eaOpts{}, nil)
	defer done()
	if err := e.proxy.SetEveryActionSettings(EveryActionSettings{HoldTTL: 600 * time.Millisecond, SignTimeout: 150 * time.Millisecond, CommitTimeout: 100 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	store.holdDelay = 520 * time.Millisecond // the store is slow to answer the hold
	start := e.remaining()
	r := e.call(1, "paid_read", "")
	if !eaIsProofUnavailable(r) || e.rr.forwarded() != 0 {
		t.Fatalf("answer %s forwarded %d", eaShape(r), e.rr.forwarded())
	}
	if e.remaining() != start || len(e.base.signed()) != 0 {
		t.Fatalf("remaining %d, receipts %v", e.remaining(), e.base.signed())
	}
}

// The hold's time limit and the signing limit are settings with defaults, and
// the signing limit plus the commit limit must be shorter than the hold's.
func TestR3_R1_Settings(t *testing.T) {
	e := newEAEnv(t, eaOpts{})
	got := e.proxy.EveryActionSettings()
	if got.HoldTTL != 60*time.Second || got.SignTimeout != 10*time.Second || got.CommitTimeout != 5*time.Second ||
		got.PendingBound != 1024 {
		t.Fatalf("defaults = %+v", got)
	}
	if got.SignTimeout+got.CommitTimeout >= got.HoldTTL {
		t.Fatalf("the defaults let a hold lapse before it is committed: %+v", got)
	}
	for name, s := range map[string]EveryActionSettings{
		"sign as long as the hold":  {HoldTTL: time.Second, SignTimeout: time.Second},
		"sign plus commit too long": {HoldTTL: time.Second, SignTimeout: 600 * time.Millisecond, CommitTimeout: 500 * time.Millisecond},
		"negative":                  {HoldTTL: -1},
		"sign longer than the hold": {HoldTTL: time.Second, SignTimeout: 2 * time.Second},
		"default sign, tiny hold":   {HoldTTL: time.Second},
	} {
		if err := e.proxy.SetEveryActionSettings(s); !errors.Is(err, ErrEveryActionSettings) {
			t.Errorf("%s: %v", name, err)
		}
	}
	if e.proxy.EveryActionSettings() != got {
		t.Fatal("a refused setting changed the settings")
	}
	if err := e.proxy.SetEveryActionSettings(EveryActionSettings{HoldTTL: 3 * time.Second, SignTimeout: time.Second, CommitTimeout: time.Second, PendingBound: 7}); err != nil {
		t.Fatal(err)
	}
	if s := e.proxy.EveryActionSettings(); s.HoldTTL != 3*time.Second || s.PendingBound != 7 || s.MissReportTimeout != DefaultMissReportTimeout {
		t.Fatalf("settings = %+v", s)
	}
}

// Observe takes a plain spend, not a hold: it is passed on whatever happens and
// is not stopped for want of a receipt.
func TestR3_R1_ObserveSpendsAndDoesNotHold(t *testing.T) {
	e, store, _, done := eaHold(t, eaOpts{observe: true}, nil)
	defer done()
	if r := e.call(1, "paid_read", ""); r.Error != nil {
		t.Fatalf("answer = %s", eaShape(r))
	}
	if spends, holds, _, _ := store.counts(); spends != 1 || holds != 0 {
		t.Fatalf("spends %d holds %d", spends, holds)
	}
}

// The in-memory enforcer's holds, on their own.
func TestR3_R1_InMemoryHold(t *testing.T) {
	ctx := context.Background()
	clk := &eaClock{t: time.Now()}
	b := NewInMemoryBudgetEnforcer()
	b.now = clk.Now
	_ = b.Initialize(ctx, "b", 10)
	if res, err := b.Hold(ctx, "b", "t", 4, "r1", time.Minute); err != nil || res.Remaining != 6 {
		t.Fatalf("hold: %+v %v", res, err)
	}
	if rem, _ := b.Remaining(ctx, "b"); rem != 6 {
		t.Fatalf("remaining while held = %d", rem)
	}
	// Over budget is Spend's refusal.
	if res, err := b.Hold(ctx, "b", "t", 7, "r2", time.Minute); err == nil || res == nil || res.ErrorCode != "budget_exhausted" {
		t.Fatalf("over-budget hold: %+v %v", res, err)
	}
	if err := b.CommitHold(ctx, "b", "r1"); err != nil {
		t.Fatal(err)
	}
	if err := b.CommitHold(ctx, "b", "r1"); !errors.Is(err, ErrHoldGone) {
		t.Fatalf("second commit = %v", err)
	}
	if err := b.Refund(ctx, "b", "r1", 4); err != nil {
		t.Fatal(err)
	}
	if rem, _ := b.Remaining(ctx, "b"); rem != 10 {
		t.Fatalf("after refund of the committed debit = %d", rem)
	}
	// Release, twice: the second is not an error.
	_, _ = b.Hold(ctx, "b", "t", 3, "r3", time.Minute)
	if err := b.ReleaseHold(ctx, "b", "r3"); err != nil {
		t.Fatal(err)
	}
	if err := b.ReleaseHold(ctx, "b", "r3"); err != nil {
		t.Fatal(err)
	}
	if rem, _ := b.Remaining(ctx, "b"); rem != 10 {
		t.Fatalf("after release = %d", rem)
	}
	// A hold nobody ends lapses; a commit after that says so; a release is fine.
	_, _ = b.Hold(ctx, "b", "t", 5, "r4", time.Second)
	clk.advance(2 * time.Second)
	if rem, _ := b.Remaining(ctx, "b"); rem != 10 {
		t.Fatalf("after the lapse = %d", rem)
	}
	if err := b.CommitHold(ctx, "b", "r4"); !errors.Is(err, ErrHoldGone) {
		t.Fatalf("commit after the lapse = %v", err)
	}
	if err := b.ReleaseHold(ctx, "b", "r4"); err != nil {
		t.Fatal(err)
	}
	// A budget with no record is unlimited: nothing held, commit is fine.
	if res, err := b.Hold(ctx, "none", "t", 3, "r5", time.Minute); err != nil || !res.Allowed || res.Remaining != -1 {
		t.Fatalf("unlimited hold: %+v %v", res, err)
	}
	if err := b.CommitHold(ctx, "none", "r5"); err != nil {
		t.Fatalf("unlimited commit: %v", err)
	}
	// Two holds under one request id: each end takes the earliest.
	_, _ = b.Hold(ctx, "b", "t", 1, "same", time.Minute)
	_, _ = b.Hold(ctx, "b", "t", 2, "same", time.Minute)
	_ = b.ReleaseHold(ctx, "b", "same")
	if rem, _ := b.Remaining(ctx, "b"); rem != 9-0 && rem != 10-2 {
		t.Fatalf("after one of two releases = %d", rem)
	}
	_ = b.ReleaseHold(ctx, "b", "same")
	if rem, _ := b.Remaining(ctx, "b"); rem != 10 {
		t.Fatalf("after both = %d", rem)
	}
}

// ---- R2 ----

// A recorder that is not a WindowRecorder.
type eaPlainRecorder struct{ *eaRecorder }

// With the switch on and a setup that failed, whatever the error was and
// whether it was ignored, a call is answered proof_unavailable before anything
// is spent or sent: Control and Observe, free and priced, over every road, and
// the methods that are not tool calls.
func TestR3_R2_FailedSetupLetsNothingThrough(t *testing.T) {
	setups := map[string]func(e *eaEnv){
		"no recorder":              func(e *eaEnv) { e.proxy.SetEvidenceRecorder(nil) },
		"recorder with no windows": func(e *eaEnv) { e.proxy.SetEvidenceRecorder(eaPlainRecorder{e.base}) },
	}
	for sname, setup := range setups {
		for _, observe := range []bool{false, true} {
			for _, road := range eaRoads {
				sname, setup, observe, road := sname, setup, observe, road
				mode := "control"
				if observe {
					mode = "observe"
				}
				t.Run(sname+"/"+mode+"/"+road.name, func(t *testing.T) {
					e := newEAEnv(t, eaOpts{observe: observe})
					setup(e)
					err := e.proxy.SetSignEveryAction(true) // the error is ignored on purpose
					if !errors.Is(err, ErrSignEveryActionNeedsWindowRecorder) {
						t.Fatalf("setup returned %v", err)
					}
					if road.via != nil {
						defer road.via(t, e)()
					}
					start := e.remaining()
					calls := map[string]*Response{
						"free tool":    e.call(1, "free_read", ""),
						"priced tool":  e.call(2, "paid_read", ""),
						"over budget":  e.call(3, "big_order", ""),
						"budget":       e.send(e.raw(4, MethodSatGateBudget, "")),
						"delegation":   e.send(e.raw(5, MethodSatGateDelegate, eaDelegateWire)),
						"other method": e.send(e.raw(6, "resources/list", "")),
					}
					for name, r := range calls {
						if !eaIsProofUnavailable(r) {
							t.Errorf("%s: answer = %s", name, eaShape(r))
						}
					}
					if e.rr.forwarded() != 0 {
						t.Fatalf("forwarded %d call(s) unsigned", e.rr.forwarded())
					}
					if e.remaining() != start {
						t.Fatalf("spent: %d -> %d", start, e.remaining())
					}
					if n := len(e.base.signed()); n != 0 {
						t.Fatalf("%d receipt(s) signed with a failed setup", n)
					}
					if len(e.proxy.PendingRecords()) != 0 {
						t.Fatal("a refused call left a pending record")
					}
				})
			}
		}
	}
}

// Refusals that need a window and the expired key are proof_unavailable too.
func TestR3_R2_FailedSetup_WindowRefusals(t *testing.T) {
	for name, o := range map[string]eaOpts{
		"rate limited": {limited: true},
		"revoked":      {revoked: true},
		"expired":      {expired: true, tenantCtx: true},
	} {
		o := o
		for _, road := range eaRoads {
			road := road
			if road.via != nil && (o.revoked || o.expired) {
				continue // such a key cannot open a session over HTTP
			}
			t.Run(name+"/"+road.name, func(t *testing.T) {
				e := newEAEnv(t, o)
				e.proxy.SetEvidenceRecorder(eaPlainRecorder{e.base})
				_ = e.proxy.SetSignEveryAction(true)
				if road.via != nil {
					defer road.via(t, e)()
				}
				if r := e.call(1, "paid_read", ""); !eaIsProofUnavailable(r) {
					t.Fatalf("answer = %s", eaShape(r))
				}
			})
		}
	}
}

// Setup that succeeds later makes the proxy ready, and Observe then forwards
// and counts a gap. A recorder installed after a failed setup does not make it
// ready by itself: SetSignEveryAction must succeed.
func TestR3_R2_ReadyOnlyAfterASuccessfulSetup(t *testing.T) {
	for _, road := range eaRoads {
		road := road
		t.Run(road.name, func(t *testing.T) {
			e := newEAEnv(t, eaOpts{observe: true})
			e.proxy.SetEvidenceRecorder(nil)
			_ = e.proxy.SetSignEveryAction(true)
			if road.via != nil {
				defer road.via(t, e)()
			}
			if r := e.call(1, "free_read", ""); !eaIsProofUnavailable(r) {
				t.Fatalf("not set up: %s", eaShape(r))
			}
			rec := &eaMissRecorder{eaRecorder: e.base}
			e.proxy.SetEvidenceRecorder(rec)
			if r := e.call(2, "free_read", ""); !eaIsProofUnavailable(r) {
				t.Fatalf("a recorder installed after a failed setup let a call through: %s", eaShape(r))
			}
			if err := e.proxy.SetSignEveryAction(true); err != nil {
				t.Fatal(err)
			}
			if r := e.call(3, "free_read", ""); r.Error != nil || e.rr.forwarded() != 1 {
				t.Fatalf("after a valid setup: %s forwarded %d", eaShape(r), e.rr.forwarded())
			}
			// Observe with the signer down: forwarded, exactly one gap.
			e.base.setDown(true)
			if r := e.call(4, "free_read", ""); r.Error != nil || e.rr.forwarded() != 2 {
				t.Fatalf("Observe after a valid setup, signer down: %s forwarded %d", eaShape(r), e.rr.forwarded())
			}
			if got := rec.landedCalls(); len(got) != 1 {
				t.Fatalf("gaps = %d, want exactly 1", len(got))
			}
		})
	}
}

// A recorder that keeps windows replaced by one that does not ends the setup.
func TestR3_R2_RecorderReplacedWithOneWithoutWindows(t *testing.T) {
	e := newEAEnv(t, eaOpts{})
	eaSetSwitch(e, true)
	if r := e.call(1, "free_read", ""); r.Error != nil {
		t.Fatalf("answer = %s", eaShape(r))
	}
	e.proxy.SetEvidenceRecorder(eaPlainRecorder{e.base})
	if r := e.call(2, "free_read", ""); !eaIsProofUnavailable(r) || e.rr.forwarded() != 1 {
		t.Fatalf("answer %s forwarded %d", eaShape(r), e.rr.forwarded())
	}
}

// ---- R3 ----

// A window recorder that rejects the report and then recovers: the call goes
// through, the miss is kept with who and what it was for, and exactly one count
// lands when the recorder takes it, however many times it is retried.
func TestR3_R3_RejectedReportIsKeptAndLandsOnce(t *testing.T) {
	cases := map[string]func(e *eaEnv) *Response{
		"free tool":   func(e *eaEnv) *Response { return e.call(1, "free_read", "") },
		"priced tool": func(e *eaEnv) *Response { return e.call(1, "paid_read", "") },
		"budget":      func(e *eaEnv) *Response { return e.send(e.raw(1, MethodSatGateBudget, "")) },
		"delegation":  func(e *eaEnv) *Response { return e.send(e.raw(1, MethodSatGateDelegate, eaDelegateWire)) },
	}
	for name, run := range cases {
		for _, road := range eaRoads {
			name, run, road := name, run, road
			t.Run(name+"/"+road.name, func(t *testing.T) {
				e := newEAEnv(t, eaOpts{observe: true})
				rec := &eaMissRecorder{eaRecorder: e.base, reject: -1}
				e.proxy.SetEvidenceRecorder(rec)
				if err := e.proxy.SetSignEveryAction(true); err != nil {
					t.Fatal(err)
				}
				if err := e.proxy.SetEveryActionSettings(EveryActionSettings{PendingRetry: 5 * time.Millisecond}); err != nil {
					t.Fatal(err)
				}
				if road.via != nil {
					defer road.via(t, e)()
				}
				e.base.setDown(true)
				if r := run(e); r.Error != nil {
					t.Fatalf("an Observe call was stopped: %s", eaShape(r))
				}
				recs := e.proxy.PendingRecords()
				if len(recs) != 1 {
					t.Fatalf("pending = %+v, want the one miss", recs)
				}
				p := recs[0]
				wantTool := name2tool(name)
				if p.Kind != PendingGap || p.TenantID != eaTenant || p.TokenID == "" || p.Tool != wantTool || p.Method == "" || p.Minute.IsZero() {
					t.Fatalf("the miss lost its identity: %+v", p)
				}
				if !p.Minute.Equal(p.Minute.Truncate(time.Minute)) {
					t.Fatalf("minute %v is not a UTC minute", p.Minute)
				}
				// Still rejected: nothing has landed, and the miss is still kept.
				time.Sleep(40 * time.Millisecond)
				if len(rec.landedCalls()) != 0 || len(e.proxy.PendingRecords()) != 1 {
					t.Fatalf("landed %d pending %d while the recorder rejects", len(rec.landedCalls()), len(e.proxy.PendingRecords()))
				}
				// The recorder recovers; the retry lands it once.
				rec.setReject(0)
				eaWait(t, func() bool { return len(e.proxy.PendingRecords()) == 0 })
				time.Sleep(30 * time.Millisecond)
				landed := rec.landedCalls()
				if len(landed) != 1 {
					t.Fatalf("landed %d counts, want exactly 1", len(landed))
				}
				d := landed[0].Decision
				if d.TenantID != p.TenantID || d.TokenID != p.TokenID || receiptTool(d.ToolName) != p.Tool || !landed[0].WindowStart.Equal(p.Minute) || landed[0].Kind != WindowKindUnrecorded {
					t.Fatalf("the count that landed is not the miss that was kept: %+v vs %+v", landed[0], p)
				}
				if e.proxy.FlushPending(context.Background()) != 0 || len(rec.landedCalls()) != 1 {
					t.Fatal("a later flush counted it again")
				}
			})
		}
	}
}

func name2tool(name string) string {
	switch name {
	case "free tool":
		return "free_read"
	case "priced tool":
		return "paid_read"
	case "budget":
		return MethodSatGateBudget
	}
	return MethodSatGateDelegate
}

// A report whose answer was lost (the recorder counted it and said it failed)
// is retried with the same ReportID, so a recorder that keeps ids counts it
// once.
func TestR3_R3_AmbiguousReportCountsOnce(t *testing.T) {
	e := newEAEnv(t, eaOpts{observe: true})
	rec := &eaMissRecorder{eaRecorder: e.base, ambiguous: true}
	e.proxy.SetEvidenceRecorder(rec)
	if err := e.proxy.SetSignEveryAction(true); err != nil {
		t.Fatal(err)
	}
	if err := e.proxy.SetEveryActionSettings(EveryActionSettings{PendingRetry: 5 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	e.base.setDown(true)
	if r := e.call(1, "free_read", ""); r.Error != nil {
		t.Fatalf("answer = %s", eaShape(r))
	}
	eaWait(t, func() bool { return len(e.proxy.PendingRecords()) == 0 })
	if n := len(rec.landedCalls()); n != 1 {
		t.Fatalf("landed %d, want 1", n)
	}
	rec.mu.Lock()
	attempts := rec.attempts
	rec.mu.Unlock()
	if attempts < 2 {
		t.Fatalf("attempts = %d; the lost answer was not retried", attempts)
	}
}

// The request is cancelled while the receipt is being made, and the signer
// fails. The window store honours its context. The report runs on a context of
// its own, so the count lands, once, and nothing is kept.
func TestR3_R3_CancelledRequestStillCountsOnce(t *testing.T) {
	for _, kind := range []string{"free tool", "priced tool", "delegation"} {
		kind := kind
		t.Run(kind, func(t *testing.T) {
			e := newEAEnv(t, eaOpts{observe: true})
			ctx, cancel := context.WithCancel(e.ctx)
			defer cancel()
			e.ctx = ctx
			rec := &eaMissRecorder{eaRecorder: e.base, ctxAware: true}
			e.proxy.SetEvidenceRecorder(eaCancelWindowRecorder{rec, cancel})
			if err := e.proxy.SetSignEveryAction(true); err != nil {
				t.Fatal(err)
			}
			var r *Response
			switch kind {
			case "free tool":
				r = e.call(1, "free_read", "")
			case "priced tool":
				r = e.call(1, "paid_read", "")
			default:
				r = e.send(e.raw(1, MethodSatGateDelegate, eaDelegateWire))
			}
			if ctx.Err() == nil {
				t.Fatal("the recorder did not cancel the request")
			}
			if r.Error != nil {
				t.Fatalf("an Observe call was stopped: %s", eaShape(r))
			}
			if n := len(rec.landedCalls()); n != 1 {
				t.Fatalf("landed %d counts, want exactly 1", n)
			}
			if len(e.proxy.PendingRecords()) != 0 {
				t.Fatalf("a count that landed is also kept: %+v", e.proxy.PendingRecords())
			}
		})
	}
}

// A full store: Observe answers proof_unavailable until it drains, and nothing
// goes out that could not be kept. With a healthy signer too: the room is set
// aside before the call, because the receipt may still fail.
func TestR3_R3_FullStoreRefusesObserveUntilItDrains(t *testing.T) {
	for _, road := range eaRoads {
		road := road
		t.Run(road.name, func(t *testing.T) {
			e := newEAEnv(t, eaOpts{observe: true})
			rec := &eaMissRecorder{eaRecorder: e.base, reject: -1}
			e.proxy.SetEvidenceRecorder(rec)
			if err := e.proxy.SetSignEveryAction(true); err != nil {
				t.Fatal(err)
			}
			if err := e.proxy.SetEveryActionSettings(EveryActionSettings{PendingBound: 2, PendingRetry: time.Hour}); err != nil {
				t.Fatal(err)
			}
			if road.via != nil {
				defer road.via(t, e)()
			}
			e.base.setDown(true)
			for i := 1; i <= 2; i++ {
				if r := e.call(i, "free_read", ""); r.Error != nil {
					t.Fatalf("call %d: %s", i, eaShape(r))
				}
			}
			if n := len(e.proxy.PendingRecords()); n != 2 {
				t.Fatalf("pending = %d, want 2 (the bound)", n)
			}
			start := e.remaining()
			for name, r := range map[string]*Response{
				"free tool":  e.call(3, "free_read", ""),
				"priced":     e.call(4, "paid_read", ""),
				"delegation": e.send(e.raw(5, MethodSatGateDelegate, eaDelegateWire)),
				"budget":     e.send(e.raw(6, MethodSatGateBudget, "")),
			} {
				if !eaIsProofUnavailable(r) {
					t.Errorf("%s with a full store: %s", name, eaShape(r))
				}
			}
			if e.rr.forwarded() != 2 || e.remaining() != start {
				t.Fatalf("forwarded %d (want 2), remaining %d (want %d)", e.rr.forwarded(), e.remaining(), start)
			}
			// Even with the signer up: the miss that might happen needs room.
			e.base.setDown(false)
			if r := e.call(7, "free_read", ""); !eaIsProofUnavailable(r) {
				t.Fatalf("full store, signer up: %s", eaShape(r))
			}
			// The recorder recovers and the store drains.
			rec.setReject(0)
			if n := e.proxy.FlushPending(context.Background()); n != 0 {
				t.Fatalf("after the flush %d entries are left", n)
			}
			if got := len(rec.landedCalls()); got != 2 {
				t.Fatalf("landed %d counts, want 2 (the two that were kept)", got)
			}
			if r := e.call(8, "free_read", ""); r.Error != nil {
				t.Fatalf("after it drained: %s", eaShape(r))
			}
		})
	}
}

// A full store of uncharged amounts refuses a priced Control call before its
// hold: there would be nowhere to keep what a failed commit leaves.
func TestR3_R3_FullUnchargedStoreRefusesBeforeTheHold(t *testing.T) {
	e, store, _, done := eaHold(t, eaOpts{}, nil)
	defer done()
	store.failCommit = true
	if err := e.proxy.SetEveryActionSettings(EveryActionSettings{PendingBound: 2}); err != nil {
		t.Fatal(err)
	}
	for i := 1; i <= 2; i++ {
		if r := e.call(i, "paid_read", ""); r.Error != nil {
			t.Fatalf("call %d: %s", i, eaShape(r))
		}
	}
	_, holds, _, _ := store.counts()
	if r := e.call(3, "paid_read", ""); !eaIsProofUnavailable(r) {
		t.Fatalf("answer = %s", eaShape(r))
	}
	if _, h, _, _ := store.counts(); h != holds {
		t.Fatalf("a hold was taken with no room to keep a failed commit (%d -> %d)", holds, h)
	}
	if r := e.send(e.raw(4, MethodSatGateDelegate, eaDelegateWire)); !eaIsProofUnavailable(r) {
		t.Fatalf("delegation = %s", eaShape(r))
	}
	// A free call needs no hold and is not stopped by it.
	if r := e.call(5, "free_read", ""); r.Error != nil {
		t.Fatalf("free call: %s", eaShape(r))
	}
}

// With the switch off none of this exists: nothing is reserved, kept, held or
// reported.
func TestR3_SwitchOffTouchesNone(t *testing.T) {
	e := newEAEnv(t, eaOpts{})
	store := &eaHoldStore{InMemoryBudgetEnforcer: e.proxy.budget.(*InMemoryBudgetEnforcer)}
	e.proxy.SetBudgetEnforcer(store)
	if r := e.call(1, "paid_read", ""); r.Error != nil {
		t.Fatalf("answer = %s", eaShape(r))
	}
	if r := e.send(e.raw(2, MethodSatGateDelegate, eaDelegateWire)); r.Error != nil {
		t.Fatalf("delegation = %s", eaShape(r))
	}
	spends, holds, commits, releases := store.counts()
	if holds != 0 || commits != 0 || releases != 0 || spends != 2 {
		t.Fatalf("spends %d holds %d commits %d releases %d", spends, holds, commits, releases)
	}
	if len(e.proxy.PendingRecords()) != 0 {
		t.Fatal("pending records with the switch off")
	}
}
