package mcpserver

// A verified key's action whose pending-journal save has an unknown result
// (see the PR "MCP: record an unknown outcome once the journal can tell").
//
// The agent's answer stays outcome_unknown, as before: nothing is released,
// refused, sent or issued at the time. With both signing switches on the call
// is now also kept in the bounded in-memory pending store a gap uses, and the
// retry worker asks the journal again until it can tell:
//
//	row found   the journal holds the work: it is recovered as any journal
//	            row is (the prepared receipt published, or the gap counted).
//	row absent  the journal confirmed it never stored the work (its
//	            cancellation is written, so no late save can store it): the
//	            hold is given back; Observe counts the call as a miss in the
//	            call's own minute; Control signs the stop budget_backend_error.
//
// D10 inventory: every place pkg/mcpserver makes or answers an unknown
// outcome, at main 3e4d275a and at this change. Class: maybe written (the
// journal may hold the work). "Kept" = kept in the in-memory retry and
// resolved as above (unknownOutcome, pending_durable.go).
//
//	src  pending_durable.go:176-178 (both) saveDurable: Save's result was
//	     neither stored nor definitely-not-stored and ResolveSave failed:
//	     ErrPendingOutcomeUnknown. Reached only from keepPrepared (U1, U2)
//	     and durableGap (U3).
//	U1   delegation.go:859-862 (now :862-874) credit-bearing satgate/delegate,
//	     held, prepared receipt: Control and Observe. Was: bare answer,
//	     nothing kept. Now kept (Proxy.keepUnknownPrepared).
//	U2   proxy.go:1273-1276 (now :1273-1282) priced tools/call, held,
//	     prepared receipt: Control and Observe. Was: bare answer, nothing
//	     kept. Now kept (Proxy.keepUnknownPrepared).
//	U3   every_action_hold.go:678-683 (now :688-694) reportMiss, an Observe
//	     gap whose save is unknown. Was: returned, nothing kept. Now kept, in
//	     the call's gap slot (keepUnknownGap). Its callers answer unknown:
//	     U3a delegation.go:843-845 (now :846-848) held delegation whose
//	         receipt failed;
//	     U3b proxy.go:1249-1254 (both) held priced tools/call whose receipt
//	         failed;
//	     U3c every_action.go:376-377 (now :379-380) receiptFailed: free
//	         tools/call, zero-credit helper, satgate/budget, budget/check,
//	         forwarded methods, identity failures.
//	ans  pending_durable.go:186-191 (both) pendingFailureResponse and
//	     delegation.go:1142-1143 (now :1153-1154) HandleDelegate: the
//	     outcome_unknown answer. Unchanged.
//
// What a Save or a Process failure means:
//
//	Save     a failure the store marks ErrPendingNotWritten (before any write:
//	         bad work, no connection, a transaction that cannot start, a lock
//	         or read that fails) is a definite not-stored save only with both
//	         switches on. With either switch off saveDurable takes main's
//	         ambiguous path (ResolveSave, then exactly as main), so no answer,
//	         hold, release, record or count differs from main's. A
//	         written cancellation or a reached bound is the plain
//	         ErrPendingNotStored, definite as on main. An INSERT or Commit
//	         failure is unknown.
//	Process  an error from the store means the callback's effects may or may
//	         not have landed, and, for an error of its final commit, that the
//	         completion itself may have been committed: the row may be
//	         completed, not always open. Publication, counting and settlement are idempotent
//	         by the stored identity, so the retry is safe either way.
//	aged     a prepared (PendingUncharged) row older than PendingHorizon that
//	         was stored but never published. With both switches on nothing
//	         ages out (never a final no-op without a receipt, count or
//	         stop): recovery, ordinary (flushDurable) and the unknown
//	         outcome's (resolveUnknown), publishes its stored signed receipt and settles it however late, and the
//	         row completes only when both have landed; money keeps the
//	         settlement contract of PendingBudget (idempotent beyond the
//	         horizon, never a fresh Spend). With either switch off it ages
//	         out exactly as on main.
//
// Accepted residual (beside the gap memory retry's, every_action_hold.go
// reportMiss): an unknown kept in memory is lost if the process ends before
// the journal answers; a Control unknown finds no room when the gap store is
// at its bound (Observe always has its call's own slot) and is then only
// logged. The enterprise journal makes the common case (Postgres down before
// the transaction) definite instead (satgate-enterprise, mcppending D8).
//
// This file uses no symbol the change adds, so it compiles and runs on main,
// where the kept-and-resolved tests fail.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// unknownJournal is a journal that cannot be reached while down is set: Save
// fails, before any write or (landed) after its row was written and its
// answer lost, and ResolveSave fails too. Once down is cleared it answers.
type unknownJournal struct {
	p2Store
	down   *atomic.Bool
	landed bool
}

func (s unknownJournal) Save(c context.Context, w PendingWork) error {
	if !s.down.Load() {
		return s.p2Store.Save(c, w)
	}
	if s.landed {
		if err := s.p2Store.Save(c, w); err != nil {
			return err
		}
	}
	return errors.New("fixture: journal unreachable")
}

func (s unknownJournal) ResolveSave(c context.Context, w PendingWork) (bool, error) {
	if s.down.Load() {
		return false, errors.New("fixture: journal unreachable")
	}
	return s.p2Store.ResolveSave(c, w)
}

func (s unknownJournal) List(c context.Context) ([]PendingWork, error) {
	if s.down.Load() {
		return nil, errors.New("fixture: journal unreachable")
	}
	return s.p2Store.List(c)
}

func (s unknownJournal) Process(c context.Context, id string, fn func(context.Context, PendingWork) (PendingDisposition, error)) (bool, error) {
	if s.down.Load() {
		return false, errors.New("fixture: journal unreachable")
	}
	return s.p2Store.Process(c, id, fn)
}

// unknownWindows is the journal's recorder, keeping every count it takes;
// failPrepare makes the allowed receipt impossible to prepare.
type unknownWindows struct {
	*p2Recorder
	mu          sync.Mutex
	calls       []MCPWindowedCall
	failPrepare bool
	// windows, when set, takes the counts (so a test can sign them).
	windows WindowRecorder
}

func (r *unknownWindows) RecordWindowedCall(ctx context.Context, c MCPWindowedCall) (*MCPEvidence, error) {
	r.mu.Lock()
	r.calls = append(r.calls, c)
	r.mu.Unlock()
	if r.windows != nil {
		return r.windows.RecordWindowedCall(ctx, c)
	}
	return r.p2Recorder.RecordWindowedCall(ctx, c)
}

func (r *unknownWindows) PreparePendingDecision(ctx context.Context, d MCPDecision) (*MCPEvidence, json.RawMessage, error) {
	if r.failPrepare {
		return nil, nil, errors.New("signer gave up")
	}
	return r.p2Recorder.PreparePendingDecision(ctx, d)
}

func (r *unknownWindows) counted() []MCPWindowedCall {
	r.mu.Lock()
	defer r.mu.Unlock()
	return append([]MCPWindowedCall(nil), r.calls...)
}

// unknownCase is one place an unknown outcome is made (the D10 inventory).
type unknownCase struct {
	name    string
	site    string // U1, U2, U3a, U3b, U3c
	observe bool
	method  string
	act     func(g *gapsRig) *Response
	// failPrepare / failRecord make the allowed receipt fail (U3).
	failPrepare, failRecord bool
	defaultUp               bool
	held                    bool
}

func unknownCases() []unknownCase {
	return []unknownCase{
		{name: "U1/delegate/observe", site: "U1", observe: true, method: MethodSatGateDelegate, act: gapsDelegate(5), held: true},
		{name: "U1/delegate/control", site: "U1", method: MethodSatGateDelegate, act: gapsDelegate(5), held: true},
		{name: "U2/tools-call/observe", site: "U2", observe: true, method: MethodToolsCall, act: gapsTool("paid_read"), held: true},
		{name: "U2/tools-call/control", site: "U2", method: MethodToolsCall, act: gapsTool("paid_read"), held: true},
		{name: "U3a/delegate-held/observe", site: "U3a", observe: true, method: MethodSatGateDelegate, act: gapsDelegate(5), failPrepare: true, held: true},
		{name: "U3b/tools-call-held/observe", site: "U3b", observe: true, method: MethodToolsCall, act: gapsTool("paid_read"), failPrepare: true, held: true},
		{name: "U3c/tools-call-free/observe", site: "U3c", observe: true, method: MethodToolsCall, act: gapsTool("free_read"), failRecord: true},
		{name: "U3c/helper/observe", site: "U3c", observe: true, method: MethodSatGateDelegate, act: gapsDelegate(0), failRecord: true},
		{name: "U3c/budget/observe", site: "U3c", observe: true, method: MethodSatGateBudget, act: gapsMethod(MethodSatGateBudget), failRecord: true},
		{name: "U3c/forwarded-method/observe", site: "U3c", observe: true, method: "resources/list", act: gapsMethod("resources/list"), failRecord: true, defaultUp: true},
	}
}

// unknownRig is a gaps rig whose journal is down (the save's result unknown)
// until up is called.
type unknownRig struct {
	*gapsRig
	down *atomic.Bool
	rec  *unknownWindows
}

func newUnknownRig(t *testing.T, c unknownCase, landed, sign, refusals bool, road eaRoad, extra func(*testing.T, *gapsRig)) (*unknownRig, func()) {
	t.Helper()
	return newJournalRig(t, c, func(g *gapsRig, down *atomic.Bool) PendingStore {
		return unknownJournal{p2Store{g.b}, down, landed}
	}, sign, refusals, road, extra)
}

// newJournalRig is newUnknownRig with the journal made by mk: down is set
// until up is called.
func newJournalRig(t *testing.T, c unknownCase, mk func(*gapsRig, *atomic.Bool) PendingStore, sign, refusals bool, road eaRoad, extra func(*testing.T, *gapsRig)) (*unknownRig, func()) {
	t.Helper()
	down := &atomic.Bool{}
	down.Store(true)
	var rec *unknownWindows
	gc := gapsCase{name: c.name, opts: eaOpts{observe: c.observe, defaultUp: c.defaultUp}, method: c.method, act: c.act,
		setup: func(t *testing.T, g *gapsRig) {
			rec = &unknownWindows{p2Recorder: g.p2(), failPrepare: c.failPrepare}
			g.journal(t, rec)
			g.proxy.SetPendingStore(mk(g, down))
			if c.failRecord {
				g.base.mu.Lock()
				g.base.failRecord = true
				g.base.mu.Unlock()
			}
			if extra != nil {
				extra(t, g)
			}
		}}
	g, done := newGapsRig(t, gc, sign, refusals, road)
	return &unknownRig{gapsRig: g, down: down, rec: rec}, done
}

func (u *unknownRig) up() {
	u.down.Store(false)
	// The base recorder answers again too: a stop can be signed.
	u.base.mu.Lock()
	u.base.failRecord = false
	u.base.mu.Unlock()
}

func (u *unknownRig) published() (receipts, completed, cancelled int) {
	u.b.mu.Lock()
	defer u.b.mu.Unlock()
	return len(u.b.receipts), len(u.b.completed), len(u.b.cancelled)
}

// kept is how many entries the in-memory pending store holds.
func (u *unknownRig) kept() int {
	s := u.proxy.every().pending
	s.mu.Lock()
	defer s.mu.Unlock()
	return len(s.entries)
}

func unknownEventually(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(2 * time.Second)
	for !cond() {
		if time.Now().After(deadline) {
			t.Fatalf("never happened: %s", what)
		}
		time.Sleep(5 * time.Millisecond)
	}
}

// assertUnknownAnswer: the answer is outcome_unknown and nothing happened yet.
func assertUnknownAnswer(t *testing.T, u *unknownRig, r *Response) {
	t.Helper()
	if outageEvidence(r)["error"] != "outcome_unknown" || u.out(r) != 0 {
		t.Fatalf("want outcome_unknown and nothing out: %s", eaShape(r))
	}
	if len(u.decisions()) != 0 || outageEvidence(r)["receipt_id"] != nil || u.misses() != 0 {
		t.Fatalf("an unknown outcome was signed or counted at the time: %v", u.base.signed())
	}
	if receipts, _, _ := u.published(); receipts != 0 {
		t.Fatal("an unknown outcome published its receipt at the time")
	}
	if _, _, _, releases := u.store.counts(); releases != 0 {
		t.Fatal("an unknown outcome released its hold at the time")
	}
}

// Astra's reproducer (round 4), kept: a valid-key Observe credit-bearing
// satgate/delegate prepares its receipt, the journal's Save fails before any
// write and ResolveSave fails too. Once the journal answers, the call is
// counted as a miss, once, in its own minute; nothing is published, refused or
// issued. On main nothing was ever recorded.
func TestUnknown_AbsentObserveDelegateIsCounted(t *testing.T) {
	c := unknownCases()[0]
	for _, road := range eaRoads {
		t.Run(road.name, func(t *testing.T) {
			u, done := newUnknownRig(t, c, false, true, true, road, nil)
			defer done()
			minute := RateWindowStart(time.Now())
			r := c.act(u.gapsRig)
			after := RateWindowStart(time.Now())
			assertUnknownAnswer(t, u, r)
			u.up()
			u.proxy.FlushPending(context.Background())
			unknownEventually(t, "the miss is counted", func() bool { return len(u.rec.counted()) == 1 })
			u.proxy.FlushPending(context.Background())
			got := u.rec.counted()
			if len(got) != 1 {
				t.Fatalf("counted %d times, want once", len(got))
			}
			m := got[0]
			if !m.WindowStart.Equal(minute) && !m.WindowStart.Equal(after) {
				t.Fatalf("counted in minute %s, want the call's (%s)", m.WindowStart, minute)
			}
			if m.Kind != WindowKindUnrecorded || m.Decision.Decision != "allowed" || m.Decision.DecisionReason != DecisionReasonObserveUnrecorded || m.Decision.TokenID == "" || m.Decision.MCPMethod != MethodSatGateDelegate {
				t.Fatalf("not a miss of this call: %+v", m.Decision)
			}
			receipts, _, cancelled := u.published()
			if receipts != 0 || len(u.decisions()) != 0 || u.out(r) != 0 || cancelled != 1 {
				t.Fatalf("published %d signed %v out %d cancelled %d", receipts, u.base.signed(), u.out(r), cancelled)
			}
			_, holds, commits, releases := u.store.counts()
			if commits != 0 || holds != 1 || releases != 1 {
				t.Fatalf("hold: holds %d commits %d releases %d", holds, commits, releases)
			}
			if n := u.kept(); n != 0 {
				t.Fatalf("pending store still holds %d entries", n)
			}
		})
	}
}

// D9, row absent, at every place an unknown is made: the answer is the one
// it was; once the journal answers, Observe counts one miss and Control signs
// one budget_backend_error stop; a held amount is given back; nothing is
// published or issued; nothing is left pending.
func TestUnknown_AbsentEndsRecorded(t *testing.T) {
	for _, c := range unknownCases() {
		for _, road := range eaRoads {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				off, doneOff := newUnknownRig(t, c, false, true, false, road, nil)
				want := eaShape(c.act(off.gapsRig))
				doneOff()
				u, done := newUnknownRig(t, c, false, true, true, road, nil)
				defer done()
				r := c.act(u.gapsRig)
				assertUnknownAnswer(t, u, r)
				if got := eaShape(r); got != want {
					t.Fatalf("answer differs with the refusal switch on:\n got %s\nwant %s", got, want)
				}
				u.up()
				u.proxy.FlushPending(context.Background())
				if c.observe {
					unknownEventually(t, "the miss is counted", func() bool { return u.misses() == 1 })
				} else {
					unknownEventually(t, "the stop is signed", func() bool { return len(u.decisions()) == 1 })
				}
				u.proxy.FlushPending(context.Background())
				receipts, _, cancelled := u.published()
				if receipts != 0 || cancelled != 1 || u.out(r) != 0 {
					t.Fatalf("published %d cancelled %d out %d", receipts, cancelled, u.out(r))
				}
				if c.observe {
					if u.misses() != 1 || len(u.decisions()) != 0 {
						t.Fatalf("Observe: misses %d signed %v", u.misses(), u.base.signed())
					}
				} else {
					ds := u.decisions()
					if len(ds) != 1 || u.misses() != 0 {
						t.Fatalf("Control: signed %v misses %d", u.base.signed(), u.misses())
					}
					d := ds[0]
					if d.Decision != "denied" || d.DecisionReason != "budget_backend_error" || !d.BudgetNotEvaluated || d.MCPMethod != c.method || d.TokenID == "" || !strings.HasPrefix(d.RequestID, callIDPrefix) || d.JSONRPCID != "" {
						t.Fatalf("Control stop: %s method %q request %q", eaLabel(d), d.MCPMethod, d.RequestID)
					}
					if d.CostCredits != 0 || d.RemainingCredits != 0 || d.BudgetLimitCredits != 0 || d.DenialCode != "" {
						t.Fatal("the stop claimed amounts or a code")
					}
				}
				_, holds, commits, releases := u.store.counts()
				if commits != 0 || (c.held && (holds != 1 || releases != 1)) {
					t.Fatalf("hold: holds %d commits %d releases %d", holds, commits, releases)
				}
				if n := u.kept(); n != 0 {
					t.Fatalf("pending store still holds %d entries", n)
				}
			})
		}
	}
}

// D9, row found: the save landed and only its answer was lost. The retry
// worker alone (no host flush) finds the row and recovers it as recovery
// does: the prepared receipt is published, or the gap counted, once.
func TestUnknown_FoundIsRecoveredByTheRetry(t *testing.T) {
	for _, c := range unknownCases() {
		for _, road := range eaRoads {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				u, done := newUnknownRig(t, c, true, true, true, road, func(t *testing.T, g *gapsRig) {
					if err := g.proxy.SetEveryActionSettings(EveryActionSettings{PendingRetry: 5 * time.Millisecond}); err != nil {
						t.Fatal(err)
					}
				})
				defer done()
				r := c.act(u.gapsRig)
				assertUnknownAnswer(t, u, r)
				u.up()
				prepared := c.site == "U1" || c.site == "U2"
				unknownEventually(t, "the row is recovered by the retry worker", func() bool {
					receipts, completed, _ := u.published()
					if prepared {
						return receipts == 1 && completed == 1
					}
					return u.misses() == 1 && completed == 1
				})
				unknownEventually(t, "the retry is done", func() bool { return u.kept() == 0 })
				u.proxy.FlushPending(context.Background())
				u.proxy.FlushPending(context.Background())
				receipts, completed, cancelled := u.published()
				if completed != 1 || cancelled != 0 || len(u.decisions()) != 0 || u.out(r) != 0 {
					t.Fatalf("completed %d cancelled %d signed %v out %d", completed, cancelled, u.base.signed(), u.out(r))
				}
				if prepared && (receipts != 1 || u.misses() != 0) {
					t.Fatalf("prepared: receipts %d misses %d", receipts, u.misses())
				}
				if !prepared && (receipts != 0 || u.misses() != 1) {
					t.Fatalf("gap: receipts %d misses %d", receipts, u.misses())
				}
			})
		}
	}
}

// D9: the miss is counted in the call's own minute, not the minute the
// journal answered. The kept call is made to look five minutes old.
func TestUnknown_CountedInTheCallsMinute(t *testing.T) {
	for _, c := range unknownCases() {
		if !c.observe {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			u, done := newUnknownRig(t, c, false, true, true, eaRoad{name: "direct"}, nil)
			defer done()
			r := c.act(u.gapsRig)
			assertUnknownAnswer(t, u, r)
			s := u.proxy.every().pending
			s.mu.Lock()
			var old time.Time
			for _, e := range s.entries {
				e.Minute = e.Minute.Add(-5 * time.Minute)
				old = e.Minute
			}
			n := len(s.entries)
			s.mu.Unlock()
			if n != 1 {
				t.Fatalf("the unknown call is not kept for retry (%d entries)", n)
			}
			u.up()
			u.proxy.FlushPending(context.Background())
			unknownEventually(t, "the miss is counted", func() bool { return len(u.rec.counted()) == 1 })
			if got := u.rec.counted()[0].WindowStart; !got.Equal(old) {
				t.Fatalf("counted in %s, want the call's minute %s", got, old)
			}
		})
	}
}

// D9 residual: a Control unknown that finds the gap store at its bound is not
// kept; its answer is unchanged and nothing is recorded (logged only). An
// Observe call always has its own slot, so it is never in this case.
func TestUnknown_FullRetryStoreIsTheResidual(t *testing.T) {
	for _, c := range unknownCases() {
		if c.observe {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			u, done := newUnknownRig(t, c, false, true, true, eaRoad{name: "direct"}, func(t *testing.T, g *gapsRig) { g.fill(t, kindGap) })
			defer done()
			r := c.act(u.gapsRig)
			assertUnknownAnswer(t, u, r)
			if n := u.kept(); n != 0 {
				t.Fatalf("kept %d entries past the bound", n)
			}
			u.up()
			u.proxy.FlushPending(context.Background())
			if len(u.decisions()) != 0 || u.misses() != 0 {
				t.Fatalf("recorded past the bound: %v", u.base.signed())
			}
		})
	}
}

// The released checkers (tools/, demo/ and the unavailable-check-v1 bundle,
// identical to tag checker-unavailable-check-v1) accept every new stop and
// count, pack by pack and as one whole record (UNKNOWN_PACK_DIR keeps them).
func TestUnknown_PublicCheckersAcceptThem(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("UNKNOWN_PACK_DIR")
	if dir == "" {
		dir = t.TempDir()
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	cmd := exec.CommandContext(ctx, "python3", "-B", filepath.Join(root, "tools/mcp_every_action_gaps_fixture.py"), dir)
	cmd.Env = append(os.Environ(), "PYTHONDONTWRITEBYTECODE=1", "PYTHONPATH="+filepath.Join(root, "demo/_vendor"))
	in, err := cmd.StdinPipe()
	if err != nil {
		t.Fatal(err)
	}
	outPipe, err := cmd.StdoutPipe()
	if err != nil {
		t.Fatal(err)
	}
	var stderr strings.Builder
	cmd.Stderr = &stderr
	if err = cmd.Start(); err != nil {
		t.Fatal(err)
	}
	defer func() {
		in.Close()
		cancel()
		if err := cmd.Wait(); err != nil && !errors.Is(ctx.Err(), context.Canceled) {
			t.Error("recorder process failed")
		}
	}()
	pack := &outagePackRecorder{input: in, output: json.NewDecoder(outPipe)}
	stops, misses := 0, 0
	// The whole record is in issued_at order: the stops are signed now, a
	// gap's count once its minute is over, so the stops go first.
	var ordered []unknownCase
	for _, observe := range []bool{false, true} {
		for _, c := range unknownCases() {
			if c.observe == observe {
				ordered = append(ordered, c)
			}
		}
	}
	for _, c := range ordered {
		t.Run(c.name, func(t *testing.T) {
			var rec *gapsPackRecorder
			u, done := newUnknownRig(t, c, false, true, true, eaRoads[1], func(t *testing.T, g *gapsRig) {
				rec = &gapsPackRecorder{eaWindowRecorder: newEAWindowRecorder(g.base), pack: pack}
				g.rec = rec.eaWindowRecorder
				jr := g.proxy.evidence.(*unknownWindows)
				jr.windows = rec.eaWindowRecorder
				g.proxy.SetEvidenceRecorder(gapsSignVia{jr, rec})
				if err := g.proxy.SetSignEveryAction(true); err != nil {
					t.Fatal(err)
				}
			})
			defer done()
			r := c.act(u.gapsRig)
			if outageEvidence(r)["error"] != "outcome_unknown" || u.out(r) != 0 {
				t.Fatalf("want outcome_unknown: %s", eaShape(r))
			}
			u.up()
			u.proxy.FlushPending(context.Background())
			if c.observe {
				unknownEventually(t, "the miss is counted", func() bool {
					rec.mu.Lock()
					defer rec.mu.Unlock()
					return len(rec.misses) == 1
				})
				misses += rec.signGaps(t)
				return
			}
			unknownEventually(t, "the stop is signed", func() bool { return len(u.decisions()) == 1 })
			stops++
		})
	}
	raw, err := pack.exchange(map[string]any{"action": "verify"})
	if err != nil {
		t.Fatalf("%v\n%s", err, stderr.String())
	}
	var result struct {
		Packs      int            `json:"packs"`
		Counts     map[string]int `json:"counts_by_decision_reason"`
		Unrecorded int            `json:"unrecorded_calls"`
	}
	if err = json.Unmarshal(raw, &result); err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"budget_backend_error": stops, "observe_unrecorded": misses}
	if stops != 2 || misses != 8 || !reflect.DeepEqual(result.Counts, want) || result.Unrecorded != misses || result.Packs != stops+misses {
		t.Fatalf("record counts %v (want %v), unrecorded %d, packs %d, stops %d, misses %d", result.Counts, want, result.Unrecorded, result.Packs, stops, misses)
	}
	t.Logf("%d packs valid and trusted in tools, demo and the unavailable-check-v1 bundle; whole record valid, trusted and complete; counts %v", result.Packs, result.Counts)
}

// unknownOffRows are the rows TestNotStored_SwitchOffBytes appends for the
// unknown outcomes: with either switch off each is exactly main's (the answer,
// what is signed, sent, charged, counted and journaled once the journal
// answers), and nothing is kept for retry (kept=0 in the case name). Rows
// appended after main's; main's rows unchanged.
func unknownOffRows(t *testing.T) []gapsSnapshot {
	var got []gapsSnapshot
	for _, c := range unknownCases() {
		for _, landed := range []bool{false, true} {
			for _, sw := range []struct {
				name           string
				sign, refusals bool
			}{{"both-off", false, false}, {"sign-only", true, false}, {"refusals-only", false, true}} {
				u, done := newUnknownRig(t, c, landed, sw.sign, sw.refusals, eaRoad{name: "direct"}, nil)
				before := u.remaining()
				r := c.act(u.gapsRig)
				done()
				kept := u.kept()
				u.up()
				u.proxy.FlushPending(context.Background())
				u.b.mu.Lock()
				rows := len(u.b.rows)
				u.b.mu.Unlock()
				got = append(got, gapsSnapshot{Case: fmt.Sprintf("%s/landed=%v/kept=%d", c.name, landed, kept), Switches: sw.name, Response: eaShape(r),
					Signed: strings.Join(u.base.signed(), ","), Out: u.out(r), Debit: before - u.remaining(),
					Misses: u.misses(), Journaled: rows})
			}
		}
	}
	return got
}
