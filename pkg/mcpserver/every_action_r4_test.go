package mcpserver

// Round-4 tests for "sign every action": every admitted call gets a hold of its
// own, so a retried commit or release can never reach another call.
//
// Two sessions of one key legitimately number their requests alike, so two
// calls in flight can carry the same JSON-RPC id. Round 3 named the hold by
// budget, tool and that id, queued same-id holds, and let each commit or
// release end the earliest of them: a retry after a lost answer could commit
// (or release) the other call's hold. These tests drive that interleaving
// through all three roads (direct dispatch, POST /mcp, POST /message), for a
// tool call and for a delegation carve.

import (
	"context"
	"errors"
	"regexp"
	"strings"
	"sync"
	"testing"
	"time"
)

// ---- test doubles ----

// eaScriptStep is what the recorder does with the nth receipt it is asked for:
// wait for gate to close (when set), then sign or fail.
type eaScriptStep struct {
	gate chan struct{}
	fail bool
}

// eaScriptRecorder is a window recorder whose receipts follow a script, in the
// order the calls reach it. Calls past the script sign at once.
type eaScriptRecorder struct {
	*eaWindowRecorder
	mu      sync.Mutex
	steps   []eaScriptStep
	arrived int
	entered chan int
}

func newEAScriptRecorder(w *eaWindowRecorder, steps ...eaScriptStep) *eaScriptRecorder {
	return &eaScriptRecorder{eaWindowRecorder: w, steps: steps, entered: make(chan int, 16)}
}

func (r *eaScriptRecorder) RecordMCPDecision(ctx context.Context, d MCPDecision) (*MCPEvidence, error) {
	r.mu.Lock()
	n := r.arrived
	r.arrived++
	var step eaScriptStep
	if n < len(r.steps) {
		step = r.steps[n]
	}
	r.mu.Unlock()
	r.entered <- n
	if step.gate != nil {
		select {
		case <-step.gate:
		case <-ctx.Done():
			return nil, ctx.Err()
		}
	}
	if step.fail {
		return nil, errors.New("signer failed")
	}
	return r.eaWindowRecorder.eaRecorder.RecordMCPDecision(context.Background(), d)
}

// waitEntered waits until the nth receipt request has reached the recorder.
func (r *eaScriptRecorder) waitEntered(t *testing.T, want int) {
	t.Helper()
	select {
	case n := <-r.entered:
		if n != want {
			t.Fatalf("receipt request %d arrived, expected %d", n, want)
		}
	case <-time.After(5 * time.Second):
		t.Fatalf("receipt request %d never reached the recorder", want)
	}
}

// eaLossyStore is a hold store that records the identity of every hold, commit
// and release it is asked for, and that can lose answers: the call takes
// effect and the answer is an error.
type eaLossyStore struct {
	*eaHoldStore
	lmu                       sync.Mutex
	loseCommits, loseReleases int
	holdIDs                   []string
	commitIDs                 []string
	releaseIDs                []string
}

func (s *eaLossyStore) Hold(ctx context.Context, id, tool string, cost int64, rid string, ttl time.Duration) (*BudgetResult, error) {
	s.lmu.Lock()
	s.holdIDs = append(s.holdIDs, rid)
	s.lmu.Unlock()
	return s.eaHoldStore.Hold(ctx, id, tool, cost, rid, ttl)
}

func (s *eaLossyStore) CommitHold(ctx context.Context, id, rid string) error {
	s.lmu.Lock()
	s.commitIDs = append(s.commitIDs, rid)
	lose := s.loseCommits > 0
	if lose {
		s.loseCommits--
	}
	s.lmu.Unlock()
	err := s.eaHoldStore.CommitHold(ctx, id, rid)
	if err == nil && lose {
		return errors.New("answer lost")
	}
	return err
}

func (s *eaLossyStore) ReleaseHold(ctx context.Context, id, rid string) error {
	s.lmu.Lock()
	s.releaseIDs = append(s.releaseIDs, rid)
	lose := s.loseReleases > 0
	if lose {
		s.loseReleases--
	}
	s.lmu.Unlock()
	err := s.eaHoldStore.ReleaseHold(ctx, id, rid)
	if err == nil && lose {
		return errors.New("answer lost")
	}
	return err
}

func (s *eaLossyStore) ids() (holds, commits, releases []string) {
	s.lmu.Lock()
	defer s.lmu.Unlock()
	return append([]string(nil), s.holdIDs...), append([]string(nil), s.commitIDs...), append([]string(nil), s.releaseIDs...)
}

// ---- sessions ----

// eaSessions opens n sessions of the environment's key over the road and
// returns a sender for each. A direct road has no session: each sender calls
// the proxy.
func eaSessions(t *testing.T, e *eaEnv, road eaRoad, n int) ([]func(*Request) *Response, func()) {
	t.Helper()
	var senders []func(*Request) *Response
	var closers []func()
	for i := 0; i < n; i++ {
		if road.via == nil {
			senders = append(senders, func(req *Request) *Response {
				resp, err := e.proxy.handleRequest(e.ctx, req)
				if err != nil {
					return NewErrorResponse(req.ID, CodeInternalError, err.Error())
				}
				return resp
			})
			continue
		}
		closers = append(closers, road.via(t, e))
		senders = append(senders, e.sender)
		e.sender = nil
	}
	return senders, func() {
		for _, c := range closers {
			c()
		}
	}
}

// eaRequest is the request of the kind: a paid tool call or a delegation
// carve, with the given JSON-RPC id.
func eaRequest(e *eaEnv, kind string, id int) *Request {
	if kind == "tool call" {
		return argCall(id, e.token, "paid_read", "")
	}
	return e.raw(id, MethodSatGateDelegate, eaDelegateWire)
}

// eaOutcome is what came of one call.
type eaOutcome struct {
	resp *Response
	done chan struct{}
}

// eaStart sends req through send in the background.
func eaStart(send func(*Request) *Response, req *Request) *eaOutcome {
	o := &eaOutcome{done: make(chan struct{})}
	go func() {
		defer close(o.done)
		o.resp = send(req)
	}()
	return o
}

func (o *eaOutcome) wait(t *testing.T) *Response {
	t.Helper()
	select {
	case <-o.done:
		return o.resp
	case <-time.After(10 * time.Second):
		t.Fatal("a call never came back")
		return nil
	}
}

// eaWentOut reports whether the answer is one that let the call out: the
// upstream's result for a tool call, a key for a delegation.
func eaWentOut(kind string, r *Response) bool {
	if r == nil || r.Error != nil {
		return false
	}
	if kind == "tool call" {
		return true
	}
	return strings.Contains(string(r.Result), `"token"`)
}

// eaTwoSessionEnv builds the environment the probes share: a store that loses
// answers on demand, a recorder that follows a script, the switch on.
func eaTwoSessionEnv(t *testing.T, road eaRoad, steps ...eaScriptStep) (*eaEnv, *eaLossyStore, *eaClock, *eaScriptRecorder, []func(*Request) *Response, func()) {
	t.Helper()
	e, store, clk, done := eaHold(t, eaOpts{}, nil)
	lossy := &eaLossyStore{eaHoldStore: store}
	e.proxy.SetBudgetEnforcer(lossy)
	script := newEAScriptRecorder(e.rec.(*eaWindowRecorder), steps...)
	e.proxy.SetEvidenceRecorder(script)
	if err := e.proxy.SetSignEveryAction(true); err != nil {
		t.Fatal(err)
	}
	senders, closeSessions := eaSessions(t, e, road, 2)
	return e, lossy, clk, script, senders, func() { closeSessions(); done() }
}

var (
	eaCallIDShape  = regexp.MustCompile(`^call-[0-9a-f]{32}$`)
	eaCarveIDShape = regexp.MustCompile(`^carve-[0-9a-f]{32}$`)
)

func eaIDShape(kind string) *regexp.Regexp {
	if kind == "tool call" {
		return eaCallIDShape
	}
	return eaCarveIDShape
}

// ---- the reviewers' probe ----

// Start 50. Call A (one session) holds and waits in its signer. Call B (another
// session, the same JSON-RPC id) holds, signs and commits; the store applies
// the commit but the answer is lost, and B's retry commits again. Then A's
// signing fails and A is refused.
//
// Exactly one call is let out (B), A is refused with proof_unavailable, and
// the balance is 49 (carve: 45): B's debit only, one commit in the store. The
// retry named B's hold and found it already committed; it could not reach A's.
func TestR4_B1_LostCommitAnswer_TheRetryCannotChargeAnotherCall(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		gateA := make(chan struct{})
		e, store, clk, script, send, done := eaTwoSessionEnv(t, road,
			eaScriptStep{gate: gateA, fail: true}, // call A: waits, then the signer fails
			eaScriptStep{})                        // call B: signs at once
		defer done()
		start, cost := e.remaining(), eaCost(kind)
		store.loseCommits = 1

		a := eaStart(send[0], eaRequest(e, kind, 7))
		script.waitEntered(t, 0)
		if got := e.remaining(); got != start-cost {
			t.Fatalf("with A's hold open the balance is %d, want %d", got, start-cost)
		}
		b := eaStart(send[1], eaRequest(e, kind, 7)) // the same JSON-RPC id, another session
		script.waitEntered(t, 1)
		respB := b.wait(t)
		if !eaWentOut(kind, respB) {
			t.Fatalf("call B = %s, want it let out", eaShape(respB))
		}
		close(gateA)
		respA := a.wait(t)
		if !eaIsProofUnavailable(respA) || eaWentOut(kind, respA) {
			t.Fatalf("call A = %s, want proof_unavailable", eaShape(respA))
		}

		if got := e.remaining(); got != start-cost {
			t.Fatalf("balance %d, want %d", got, start-cost)
		}
		// After the hold's bound nothing changes: no hold is left to lapse.
		clk.advance(DefaultHoldTTL + time.Second)
		if got := e.remaining(); got != start-cost {
			t.Fatalf("balance %d after the hold's bound, want %d", got, start-cost)
		}
		if kind == "tool call" && e.rr.forwarded() != 1 {
			t.Fatalf("forwarded %d, want 1", e.rr.forwarded())
		}
		if n := len(store.SpendLog()); n != 1 {
			t.Fatalf("%d commits took effect, want exactly 1", n)
		}
		holds, commits, releases := store.ids()
		if len(holds) != 2 || holds[0] == holds[1] {
			t.Fatalf("holds %v: want two, with different identities", holds)
		}
		// B is the second hold. Its commit was tried twice, both times by its own
		// identity; A's identity was released, once, and never committed.
		if len(commits) != 2 || commits[0] != holds[1] || commits[1] != holds[1] {
			t.Fatalf("commits %v, want two tries of B's hold %s", commits, holds[1])
		}
		if len(releases) != 1 || releases[0] != holds[0] {
			t.Fatalf("releases %v, want A's hold %s once", releases, holds[0])
		}
		if log := store.SpendLog(); log[0].RequestID != holds[1] {
			t.Fatalf("the commit that took effect is %q, want B's %q", log[0].RequestID, holds[1])
		}
		// One receipt, and it carries B's identity.
		if signed := e.base.decisions; len(signed) != 1 || signed[0].RequestID != holds[1] {
			t.Fatalf("receipts %+v, want one with request_id %s", signed, holds[1])
		}
		// B's commit landed (its repeat found that out): nothing is left uncharged.
		if p := e.proxy.PendingRecords(); len(p) != 0 {
			t.Fatalf("a charged call must leave nothing uncharged: %+v", p)
		}
	})
}

// The same, with a lost release answer. A and B both hold; B waits in its
// signer; A's signing fails and A's release takes effect but its answer is
// lost, so it is repeated. The repeat finds A's hold released and does
// nothing; B's hold, still open, is not touched, so B signs and commits and is
// charged.
func TestR4_B1_LostReleaseAnswer_TheRetryCannotReleaseAnotherCallsHold(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		gateA, gateB := make(chan struct{}), make(chan struct{})
		e, store, clk, script, send, done := eaTwoSessionEnv(t, road,
			eaScriptStep{gate: gateA, fail: true},
			eaScriptStep{gate: gateB})
		defer done()
		start, cost := e.remaining(), eaCost(kind)
		store.loseReleases = 1

		a := eaStart(send[0], eaRequest(e, kind, 7))
		script.waitEntered(t, 0)
		b := eaStart(send[1], eaRequest(e, kind, 7))
		script.waitEntered(t, 1)
		if got := e.remaining(); got != start-2*cost {
			t.Fatalf("both holds open: balance %d, want %d", got, start-2*cost)
		}
		close(gateA)
		respA := a.wait(t)
		if !eaIsProofUnavailable(respA) || eaWentOut(kind, respA) {
			t.Fatalf("call A = %s, want proof_unavailable", eaShape(respA))
		}
		// A's hold is back; B's is still set aside.
		if got := e.remaining(); got != start-cost {
			t.Fatalf("after A's release (answer lost, repeated): balance %d, want %d (B's hold must still be open)", got, start-cost)
		}
		close(gateB)
		respB := b.wait(t)
		if !eaWentOut(kind, respB) {
			t.Fatalf("call B = %s, want it let out", eaShape(respB))
		}

		holds, commits, releases := store.ids()
		if len(holds) != 2 || holds[0] == holds[1] {
			t.Fatalf("holds %v", holds)
		}
		if len(releases) != 2 || releases[0] != holds[0] || releases[1] != holds[0] {
			t.Fatalf("releases %v, want two tries of A's hold %s", releases, holds[0])
		}
		if len(commits) != 1 || commits[0] != holds[1] {
			t.Fatalf("commits %v, want one, of B's hold %s", commits, holds[1])
		}
		if n := len(store.SpendLog()); n != 1 {
			t.Fatalf("%d commits took effect, want exactly 1", n)
		}
		if got := e.remaining(); got != start-cost {
			t.Fatalf("balance %d, want %d", got, start-cost)
		}
		clk.advance(DefaultHoldTTL + time.Second)
		if got := e.remaining(); got != start-cost {
			t.Fatalf("balance %d after the hold's bound, want %d", got, start-cost)
		}
		if p := e.proxy.PendingRecords(); len(p) != 0 {
			t.Fatalf("a charged call must leave nothing uncharged: %+v", p)
		}
	})
}

// Two sessions send the same JSON-RPC id at the same time and both are
// allowed: two holds under two identities, two commits, two debits.
func TestR4_B1_SameJSONRPCID_BothAllowed(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		gateA, gateB := make(chan struct{}), make(chan struct{})
		e, store, _, script, send, done := eaTwoSessionEnv(t, road,
			eaScriptStep{gate: gateA}, eaScriptStep{gate: gateB})
		defer done()
		start, cost := e.remaining(), eaCost(kind)

		a := eaStart(send[0], eaRequest(e, kind, 7))
		script.waitEntered(t, 0)
		b := eaStart(send[1], eaRequest(e, kind, 7))
		script.waitEntered(t, 1)
		close(gateB)
		close(gateA)
		respA, respB := a.wait(t), b.wait(t)
		if !eaWentOut(kind, respA) || !eaWentOut(kind, respB) {
			t.Fatalf("A = %s, B = %s: both want to be let out", eaShape(respA), eaShape(respB))
		}
		holds, commits, releases := store.ids()
		if len(holds) != 2 || holds[0] == holds[1] {
			t.Fatalf("holds %v: want two identities", holds)
		}
		for _, id := range holds {
			if !eaIDShape(kind).MatchString(id) {
				t.Fatalf("hold identity %q is not a server-made %s id", id, kind)
			}
		}
		if len(commits) != 2 || len(releases) != 0 {
			t.Fatalf("commits %v releases %v", commits, releases)
		}
		seen := map[string]bool{}
		for _, c := range commits {
			seen[c] = true
		}
		if !seen[holds[0]] || !seen[holds[1]] {
			t.Fatalf("commits %v do not name both holds %v", commits, holds)
		}
		if n := len(store.SpendLog()); n != 2 {
			t.Fatalf("%d commits took effect, want 2", n)
		}
		if got := e.remaining(); got != start-2*cost {
			t.Fatalf("balance %d, want %d", got, start-2*cost)
		}
		if kind == "tool call" && e.rr.forwarded() != 2 {
			t.Fatalf("forwarded %d, want 2", e.rr.forwarded())
		}
		// Two receipts, with the two identities, which differ.
		signed := e.base.decisions
		if len(signed) != 2 || signed[0].RequestID == signed[1].RequestID {
			t.Fatalf("receipts %+v: want two with different request_ids", signed)
		}
		for _, d := range signed {
			if d.RequestID != holds[0] && d.RequestID != holds[1] {
				t.Fatalf("receipt request_id %q is neither hold %v", d.RequestID, holds)
			}
		}
	})
}

// ---- where identities come from ----

// A call's identity is the server's: it has the server's shape and holds no
// byte the client sent. The client's JSON-RPC id (here one chosen to be found),
// the label of a delegation, and the id's repetition across calls change
// nothing about it.
func TestR4_B1_TheIdentityIsNeverTheClients(t *testing.T) {
	for _, road := range eaRoads {
		for _, kind := range []string{"tool call", "delegation"} {
			road, kind := road, kind
			t.Run(road.name+"/"+kind, func(t *testing.T) {
				e, store, _, _, send, done := eaTwoSessionEnv(t, road)
				defer done()
				for i := 0; i < 3; i++ {
					req := eaRequest(e, kind, 424242)
					if kind == "delegation" {
						req = e.raw(424242, MethodSatGateDelegate, `"budget":2,"scope":"free_read","label":"LABEL-SENTINEL-`+string(rune('a'+i))+`"`)
					}
					if r := send[i%2](req); !eaWentOut(kind, r) {
						t.Fatalf("call %d = %s", i, eaShape(r))
					}
				}
				holds, _, _ := store.ids()
				if len(holds) != 3 {
					t.Fatalf("holds %v", holds)
				}
				seen := map[string]bool{}
				for _, id := range holds {
					if !eaIDShape(kind).MatchString(id) {
						t.Fatalf("identity %q is not a server-made id", id)
					}
					if strings.Contains(id, "424242") || strings.Contains(strings.ToLower(id), "sentinel") || strings.Contains(id, e.budgetID) {
						t.Fatalf("identity %q carries something the client sent", id)
					}
					if id == generateRequestID(e.budgetID, "paid_read", eaRequest(e, kind, 424242).ID) {
						t.Fatalf("identity %q is derived from the JSON-RPC id", id)
					}
					seen[id] = true
				}
				if len(seen) != 3 {
					t.Fatalf("identities repeat: %v", holds)
				}
				// The receipts' request_id is the same identity, one per call.
				for i, d := range e.base.decisions {
					if d.RequestID != holds[i] {
						t.Fatalf("receipt %d request_id %q, hold %q", i, d.RequestID, holds[i])
					}
				}
			})
		}
	}
}

// Every other receipt of a call has a request_id the server made as well: two
// sessions that number their requests alike get receipts that can be told
// apart, for a refusal and for a call that reads no balance too.
func TestR4_B1_ReceiptsOfOtherCallsCarryTheServersIdentity(t *testing.T) {
	for _, road := range eaRoads {
		road := road
		t.Run(road.name, func(t *testing.T) {
			e, _, _, _, send, done := eaTwoSessionEnv(t, road)
			defer done()
			calls := []func(id int) *Request{
				func(id int) *Request { return argCall(id, e.token, "free_read", "") },                         // allowed, free
				func(id int) *Request { return argCall(id, e.token, "not_in_scope_tool", "") },                 // refused: scope
				func(id int) *Request { return e.raw(id, "budget/check", "") },                                 // budget read
				func(id int) *Request { return e.raw(id, MethodSatGateDelegate, `"budget":1,"scope":"nope"`) }, // delegation refused or not
			}
			for ci, call := range calls {
				before := len(e.base.attempts)
				send[0](call(9))
				send[1](call(9))
				got := e.base.attempts[before:]
				if len(got) != 2 {
					t.Fatalf("call %d: %d receipt requests, want 2", ci, len(got))
				}
				if got[0].RequestID == "" || got[0].RequestID == got[1].RequestID {
					t.Fatalf("call %d: request_ids %q and %q must be set and differ", ci, got[0].RequestID, got[1].RequestID)
				}
				for _, d := range got {
					if !eaCallIDShape.MatchString(d.RequestID) && !eaCarveIDShape.MatchString(d.RequestID) {
						t.Fatalf("call %d: request_id %q is not a server-made id", ci, d.RequestID)
					}
				}
			}
		})
	}
}

// What a commit that failed after signing keeps, and what a carve keeps for its
// settlement, is named by the call's own identity: two sessions with the same
// JSON-RPC id keep two entries, each settled once.
func TestR4_B1_PendingAndSettlementAreKeyedByTheCallsIdentity(t *testing.T) {
	for _, road := range eaRoads {
		road := road
		t.Run(road.name+"/tool calls", func(t *testing.T) {
			e, store, clk, _, send, done := eaTwoSessionEnv(t, road)
			defer done()
			start := e.remaining()
			store.failCommit = true
			for i := 0; i < 2; i++ {
				if r := send[i](eaRequest(e, "tool call", 7)); !eaWentOut("tool call", r) {
					t.Fatalf("call %d = %s", i, eaShape(r))
				}
			}
			recs := e.proxy.PendingRecords()
			if len(recs) != 2 || recs[0].RequestID == recs[1].RequestID || recs[0].ID == recs[1].ID {
				t.Fatalf("pending %+v: want two entries with different request ids and ids", recs)
			}
			holds, _, _ := store.ids()
			for _, r := range recs {
				if !eaCallIDShape.MatchString(r.RequestID) || (r.RequestID != holds[0] && r.RequestID != holds[1]) {
					t.Fatalf("pending request id %q is not one of the holds %v", r.RequestID, holds)
				}
			}
			clk.advance(DefaultHoldTTL + time.Second)
			if got := e.remaining(); got != start {
				t.Fatalf("never charged: balance %d, want %d", got, start)
			}
		})
		t.Run(road.name+"/carves", func(t *testing.T) {
			e, store, _, _, send, done := eaTwoSessionEnv(t, road)
			defer done()
			start := e.remaining()
			store.failCommit = true
			var kids []string
			for i := 0; i < 2; i++ {
				r := send[i](eaRequest(e, "delegation", 7))
				if !eaWentOut("delegation", r) {
					t.Fatalf("delegation %d = %s", i, eaShape(r))
				}
				kids = append(kids, string(r.Result))
			}
			recs := e.proxy.PendingRecords()
			if len(recs) != 2 || recs[0].RequestID == recs[1].RequestID || recs[0].ChildBudgetID == recs[1].ChildBudgetID {
				t.Fatalf("pending %+v: want two carves, each with its own identity and child", recs)
			}
			for _, r := range recs {
				if !eaCarveIDShape.MatchString(r.RequestID) {
					t.Fatalf("carve identity %q is not a server-made id", r.RequestID)
				}
			}
			// The store recovers. Each carve is settled once: the parent is
			// charged 5 twice, each child funded 5 once.
			store.failCommit = false
			e.proxy.FlushPending(context.Background())
			e.proxy.FlushPending(context.Background())
			if left := len(e.proxy.PendingRecords()); left != 0 {
				t.Fatalf("%d entries left after the store recovered", left)
			}
			if got := e.remaining(); got != start-2*eaCost("delegation") {
				t.Fatalf("parent %d, want %d", got, start-2*eaCost("delegation"))
			}
			for _, r := range recs {
				if rem, _ := e.proxy.budget.Remaining(context.Background(), r.ChildBudgetID); rem != eaCost("delegation") {
					t.Fatalf("child %s holds %d, want %d", r.ChildBudgetID, rem, eaCost("delegation"))
				}
			}
			if charged, closed := e.proxy.CarveSettlements(); charged != 2 || closed != 0 {
				t.Fatalf("settlements charged %d closed %d, want 2 and 0", charged, closed)
			}
			_ = kids
		})
	}
}

// The ids of the pending store, which are also the ReportID of a gap report
// (the key a recorder counts it once by), cannot be made twice: they carry
// random bits, so two processes, or one before and after a restart, never make
// the same.
func TestR4_B1_PendingIDsDoNotRepeatAcrossProcesses(t *testing.T) {
	seen := map[string]bool{}
	for i := 0; i < 50; i++ {
		s := newPendingStore()
		for _, id := range []string{s.nextID(kindGap), s.nextID(kindUncharged)} {
			if seen[id] {
				t.Fatalf("id %q made twice", id)
			}
			seen[id] = true
		}
	}
	if !regexp.MustCompile(`^observe_gap-[0-9a-f]{16}-1$`).MatchString(newPendingStore().nextID(kindGap)) {
		t.Fatalf("id shape: %q", newPendingStore().nextID(kindGap))
	}
}

// Observe: a miss from each of two sessions with the same JSON-RPC id is two
// reports with two ReportIDs.
func TestR4_B1_ObserveMissesOfTwoSessionsAreTwoReports(t *testing.T) {
	for _, road := range eaRoads {
		road := road
		t.Run(road.name, func(t *testing.T) {
			e := newEAEnv(t, eaOpts{observe: true})
			// A recorder that rejects every report, so each miss is kept.
			rec := &eaMissRecorder{eaRecorder: e.base, reject: -1}
			e.proxy.SetEvidenceRecorder(rec)
			if err := e.proxy.SetSignEveryAction(true); err != nil {
				t.Fatal(err)
			}
			if err := e.proxy.SetEveryActionSettings(EveryActionSettings{PendingRetry: time.Hour}); err != nil {
				t.Fatal(err)
			}
			send, closeSessions := eaSessions(t, e, road, 2)
			defer closeSessions()
			e.base.setDown(true) // no receipt can be made: the calls go through and are reported
			for i := 0; i < 2; i++ {
				if r := send[i](argCall(7, e.token, "paid_read", "")); r.Error != nil {
					t.Fatalf("an Observe call was stopped: %s", eaShape(r))
				}
			}
			recs := e.proxy.PendingRecords()
			if len(recs) != 2 || recs[0].ID == recs[1].ID {
				t.Fatalf("pending %+v: want two misses with different ids", recs)
			}
		})
	}
}
