package mcpserver

// A refusal signed after the call's budget hold was given back (see the PR
// "MCP: sign the stop when a delegation's prepared receipt is not stored",
// round 2).
//
// With both signing switches on, each stop below takes a budget hold, gives it
// back, and then signs its refusal:
//
//	not-stored/tools-call   proxy.go: keepPrepared definitely not stored
//	not-stored/delegate     delegation.go: keepPrepared definitely not stored
//	prepare/tools-call      proxy.go: the held amount's entry not prepared
//	prepare/delegate        delegation.go: the carve's entry not prepared
//	held-gap/tools-call     proxy.go: Observe, the held call's gap not stored
//	held-gap/delegate       delegation.go: Observe, the held carve's gap not stored
//	child-budget/delegate   delegation.go: the child's budget not created
//	child-refused/delegate  delegation.go: the store refused the child's budget
//	child-row/delegate      delegation.go: the credit-bearing child's row not written
//
// When release confirms the hold gone (released, or the store says it no
// longer exists), the refusal is signed with the stop's own reason, and at the
// moment the signer is called the hold is already gone. When it does not, the
// refusal is signed budget_backend_error, the failure is counted, nothing is
// issued, forwarded, charged or committed, and the held amount comes back when
// the hold lapses (HoldTTL); the request does not wait for that. Either way the
// answer is the one the call gets with the refusal switch off, plus the
// receipt's handles. This file uses no symbol the change adds, so it compiles
// and runs on main.

import (
	"context"
	"strings"
	"sync"
	"testing"
	"time"
)

// heldCase is one stop signed after a hold is given back. reason is the
// reason it is signed with when the release succeeds; amount is what is held.
type heldCase struct {
	gapsCase
	amount int64
}

// heldTTL is the short hold of the release-fails cases: the held amount must
// come back by itself within it.
const heldTTL = 2 * time.Second

// heldStore is the rig's budget store with two more answers: a refusal to
// create a child's budget (initRefused), and ErrHoldGone after a release that
// took effect (answersGone), as a store that finds the hold gone answers.
type heldStore struct {
	*gapsStore
	initRefused, answersGone bool
}

func (s *heldStore) Initialize(ctx context.Context, id string, credits int64) error {
	if s.initRefused && strings.HasPrefix(id, "del-") {
		return ErrInsufficientBudget
	}
	return s.gapsStore.Initialize(ctx, id, credits)
}

func (s *heldStore) ReleaseHold(ctx context.Context, id, rid string) error {
	if err := s.gapsStore.ReleaseHold(ctx, id, rid); err != nil || !s.answersGone {
		return err
	}
	return ErrHoldGone
}

// heldStoreOf installs the heldStore over the rig's store, once. A journal
// installed before it is pointed at it too.
func heldStoreOf(g *gapsRig) *heldStore {
	if s, ok := g.proxy.budget.(*heldStore); ok {
		return s
	}
	s := &heldStore{gapsStore: g.store}
	g.proxy.SetBudgetEnforcer(s)
	if g.durable {
		g.proxy.SetPendingStore(p2Store{g.b})
	}
	return s
}

func heldReleaseCases() []heldCase {
	plain := func(g *gapsRig) EvidenceRecorder { return g.p2() }
	noPrepare := func(g *gapsRig) EvidenceRecorder { return &gapsFailPrepare{g.p2()} }
	journal := func(failSave bool, rec func(*gapsRig) EvidenceRecorder) func(*testing.T, *gapsRig) {
		return func(t *testing.T, g *gapsRig) {
			g.b.failSave = failSave
			g.journal(t, rec(g))
		}
	}
	both := func(fs ...func(*testing.T, *gapsRig)) func(*testing.T, *gapsRig) {
		return func(t *testing.T, g *gapsRig) {
			for _, f := range fs {
				if f != nil {
					f(t, g)
				}
			}
		}
	}
	prepareFails := func(t *testing.T, g *gapsRig) { g.store.failPrepare = true }
	childFails := func(t *testing.T, g *gapsRig) { g.store.failChildInit = true }
	childRefused := func(t *testing.T, g *gapsRig) { heldStoreOf(g).initRefused = true }
	rowFails := func(t *testing.T, g *gapsRig) {
		fp := &failingPersister{newLimitStore()}
		fp.setRow(eaBudget, nil) // a stored row decides the handoff, so the child's row is written first
		g.proxy.SetEventPublisher(fp)
	}
	tool, del := gapsTool("paid_read"), gapsDelegate(5)
	var out []heldCase
	add := func(name, mode, reason, method string, setup func(*testing.T, *gapsRig), act func(*gapsRig) *Response, amount int64) {
		out = append(out, heldCase{gapsCase{name: "held/" + name + "/" + mode, opts: eaOpts{observe: mode == "observe"}, reason: reason,
			method: method, setup: setup, act: act}, amount})
	}
	for _, mode := range []string{"control", "observe"} {
		// Observe holds a carve or a priced call only with a journal.
		var journalIfObserve func(*testing.T, *gapsRig)
		if mode == "observe" {
			journalIfObserve = journal(false, plain)
		}
		add("not-stored/tools-call", mode, "pending_capacity_exhausted", MethodToolsCall, journal(true, plain), tool, 1)
		add("not-stored/delegate", mode, "pending_capacity_exhausted", MethodSatGateDelegate, journal(true, plain), del, 5)
		add("prepare/tools-call", mode, "budget_backend_error", MethodToolsCall, both(journal(false, plain), prepareFails), tool, 1)
		add("prepare/delegate", mode, "budget_backend_error", MethodSatGateDelegate, both(journal(false, plain), prepareFails), del, 5)
		add("child-budget/delegate", mode, "budget_backend_error", MethodSatGateDelegate, both(journalIfObserve, childFails), del, 5)
		add("child-refused/delegate", mode, "policy_denied", MethodSatGateDelegate, both(childRefused, journalIfObserve), del, 5)
		add("child-row/delegate", mode, "tool_controls_unavailable", MethodSatGateDelegate, both(journalIfObserve, rowFails), del, 5)
	}
	add("held-gap/tools-call", "observe", "pending_capacity_exhausted", MethodToolsCall, journal(true, noPrepare), tool, 1)
	add("held-gap/delegate", "observe", "pending_capacity_exhausted", MethodSatGateDelegate, journal(true, noPrepare), del, 5)
	return out
}

// failing is c with a store that cannot release (every try fails) and a short
// hold. Its refusal is signed budget_backend_error.
func (c heldCase) failing() heldCase {
	setup := c.setup
	c.name += "/release-fails"
	c.reason = "budget_backend_error"
	c.setup = func(t *testing.T, g *gapsRig) {
		if setup != nil {
			setup(t, g)
		}
		g.store.mu.Lock()
		g.store.failRelease = true
		g.store.mu.Unlock()
		s := g.proxy.every().cfg()
		s.HoldTTL, s.SignTimeout, s.CommitTimeout = heldTTL, 600*time.Millisecond, 500*time.Millisecond
		if err := g.proxy.SetEveryActionSettings(s); err != nil {
			t.Fatal(err)
		}
	}
	return c
}

// ---- the signer's view ----

// heldAtSign is what the budget store held when a refusal was signed.
type heldAtSign struct {
	reason    string
	open      int
	remaining int64
}

type heldProbe struct {
	g   *gapsRig
	mu  sync.Mutex
	saw []heldAtSign
}

func (p *heldProbe) look(d MCPDecision) {
	if d.Decision != "denied" {
		return
	}
	b := p.g.store.InMemoryBudgetEnforcer
	b.mu.Lock()
	open := len(b.holds)
	b.mu.Unlock()
	rem := p.g.remaining()
	p.mu.Lock()
	p.saw = append(p.saw, heldAtSign{d.DecisionReason, open, rem})
	p.mu.Unlock()
}

func (p *heldProbe) seen() []heldAtSign {
	p.mu.Lock()
	defer p.mu.Unlock()
	return append([]heldAtSign(nil), p.saw...)
}

// heldProbeRecorder and heldProbeDurable look at the budget store from inside
// the signer, then sign as the recorder they wrap.
type heldProbeRecorder struct {
	EvidenceRecorder
	WindowRecorder
	p *heldProbe
}

func (r heldProbeRecorder) RecordMCPDecision(ctx context.Context, d MCPDecision) (*MCPEvidence, error) {
	r.p.look(d)
	return r.EvidenceRecorder.RecordMCPDecision(ctx, d)
}

type heldProbeDurable struct {
	gapsDurableRecorder
	p *heldProbe
}

func (r heldProbeDurable) RecordMCPDecision(ctx context.Context, d MCPDecision) (*MCPEvidence, error) {
	r.p.look(d)
	return r.gapsDurableRecorder.RecordMCPDecision(ctx, d)
}

// probed is c with the signer wrapped in a probe.
func (c heldCase) probed(probe **heldProbe) gapsCase {
	setup := c.setup
	g := c.gapsCase
	g.setup = func(t *testing.T, r *gapsRig) {
		if setup != nil {
			setup(t, r)
		}
		p := &heldProbe{g: r}
		*probe = p
		switch rec := r.proxy.evidence.(type) {
		case gapsDurableRecorder:
			r.proxy.SetEvidenceRecorder(heldProbeDurable{rec, p})
		default:
			r.proxy.SetEvidenceRecorder(heldProbeRecorder{rec, rec.(WindowRecorder), p})
		}
		if err := r.proxy.SetSignEveryAction(true); err != nil {
			t.Fatal(err)
		}
	}
	return g
}

// ---- assertions ----

func heldAssertRefusal(t *testing.T, g *gapsRig, c heldCase, r *Response) {
	t.Helper()
	if r == nil || r.Error == nil || g.out(r) != 0 {
		t.Fatalf("stop went through: %s", eaShape(r))
	}
	ds := g.decisions()
	if len(ds) != 1 {
		t.Fatalf("want exactly one signed refusal, got %d: %v", len(ds), g.base.signed())
	}
	d := ds[0]
	if d.Decision != "denied" || d.DecisionReason != c.reason || !d.BudgetNotEvaluated || d.NoVerifiedCapability {
		t.Fatalf("wrong refusal: %s, want denied/%s", eaLabel(d), c.reason)
	}
	if d.CostCredits != 0 || d.RemainingCredits != 0 || d.RemainingBeforeCredits != 0 || d.BudgetLimitCredits != 0 {
		t.Fatal("refusal claimed amounts")
	}
	if d.MCPMethod != c.method || d.TokenID == "" {
		t.Fatalf("refusal identity: method %q", d.MCPMethod)
	}
	if c.reason == "policy_denied" {
		if d.DenialCode != "DELEGATION_REFUSED" {
			t.Fatalf("refusal code %q", d.DenialCode)
		}
	} else if d.DenialCode != "" || !strings.HasPrefix(d.RequestID, callIDPrefix) || d.JSONRPCID != "" {
		t.Fatalf("refusal identity: code %q request %q", d.DenialCode, d.RequestID)
	}
	if outageEvidence(r)["receipt_id"] != "rcpt_1" {
		t.Fatalf("missing refusal handles: %s", eaShape(r))
	}
	if g.misses() != 0 {
		t.Fatal("a stopped call was also counted as a miss")
	}
	g.b.mu.Lock()
	published, rows, gaps := len(g.b.receipts), len(g.b.rows), len(g.b.gaps)
	g.b.mu.Unlock()
	if published != 0 || rows != 0 || gaps != 0 {
		t.Fatalf("something was kept: published %d journal rows %d gaps %d", published, rows, gaps)
	}
	if n := len(g.proxy.PendingRecords()); n != 0 {
		t.Fatalf("pending store holds %d entries", n)
	}
	if _, _, commits, _ := g.store.counts(); commits != 0 {
		t.Fatalf("a hold was committed (%d)", commits)
	}
}

// heldWant is the answer with the refusal switch off, the same faults on.
func heldWant(t *testing.T, c heldCase, road eaRoad) string {
	t.Helper()
	off, done := newGapsRig(t, c.gapsCase, true, false, road)
	defer done()
	return gapsStrip(t, c.act(off))
}

// ---- tests ----

// The release succeeds: the refusal is signed with the stop's own reason, and
// when the signer is called the hold is already gone and the balance whole.
func TestHeldRelease_ConfirmedSignsTheStopsReason(t *testing.T) {
	for _, c := range heldReleaseCases() {
		for _, road := range eaRoads {
			c, road := c, road
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				t.Parallel()
				want := heldWant(t, c, road)
				var probe *heldProbe
				g, done := newGapsRig(t, c.probed(&probe), true, true, road)
				defer done()
				before := g.remaining()
				r := c.act(g)
				heldAssertRefusal(t, g, c, r)
				if got := gapsStrip(t, r); got != want {
					t.Fatalf("answer changed:\n got %s\nwant %s", got, want)
				}
				saw := probe.seen()
				if len(saw) != 1 || saw[0].open != 0 || saw[0].remaining != before {
					t.Fatalf("at signing the hold was not gone: %+v (balance before %d)", saw, before)
				}
				if g.remaining() != before {
					t.Fatalf("the stop changed the balance: %d -> %d", before, g.remaining())
				}
				if n := g.proxy.ReversalFailures(); n != 0 {
					t.Fatalf("%d release failures counted", n)
				}
			})
		}
	}
}

// The release fails on every try: the refusal is signed budget_backend_error
// while the hold is still open, the failure is counted, nothing is issued,
// forwarded, charged or committed, and the held amount comes back by itself
// when the hold lapses.
func TestHeldRelease_UnconfirmedSignsBudgetBackendError(t *testing.T) {
	for _, base := range heldReleaseCases() {
		c := base.failing()
		for _, road := range eaRoads {
			c, road := c, road
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				t.Parallel()
				want := heldWant(t, c, road)
				var probe *heldProbe
				g, done := newGapsRig(t, c.probed(&probe), true, true, road)
				defer done()
				before := g.remaining()
				start := time.Now()
				r := c.act(g)
				if time.Since(start) >= heldTTL {
					t.Fatalf("the request waited for the hold to lapse (%s)", time.Since(start))
				}
				heldAssertRefusal(t, g, c, r)
				if got := gapsStrip(t, r); got != want {
					t.Fatalf("answer changed:\n got %s\nwant %s", got, want)
				}
				saw := probe.seen()
				if len(saw) != 1 || saw[0].open != 1 || saw[0].remaining != before-c.amount {
					t.Fatalf("at signing: %+v (balance before %d, held %d)", saw, before, c.amount)
				}
				if n := g.proxy.ReversalFailures(); n != 1 {
					t.Fatalf("release failures counted %d, want 1", n)
				}
				if _, holds, _, releases := g.store.counts(); holds != 1 || releases == 0 {
					t.Fatalf("holds %d releases tried %d", holds, releases)
				}
				deadline := time.Now().Add(heldTTL + 5*time.Second)
				for g.remaining() != before {
					if time.Now().After(deadline) {
						t.Fatalf("the held amount did not come back after the hold lapsed: %d, want %d", g.remaining(), before)
					}
					time.Sleep(25 * time.Millisecond)
				}
				b := g.store.InMemoryBudgetEnforcer
				b.mu.Lock()
				open := len(b.holds)
				b.mu.Unlock()
				if open != 0 {
					t.Fatalf("%d holds still open after the lapse", open)
				}
				if _, _, commits, _ := g.store.counts(); commits != 0 || g.out(r) != 0 || len(g.decisions()) != 1 {
					t.Fatalf("after the lapse: commits %d out %d signed %v", commits, g.out(r), g.base.signed())
				}
			})
		}
	}
}

// A store that answers a release with "the hold is gone" confirms it: the
// stop keeps its own reason and nothing is held.
func TestHeldRelease_GoneCountsAsConfirmed(t *testing.T) {
	for _, base := range heldReleaseCases() {
		if base.reason == "budget_backend_error" {
			continue
		}
		c := base
		setup := c.setup
		c.name += "/answers-gone"
		c.setup = func(t *testing.T, g *gapsRig) {
			if setup != nil {
				setup(t, g)
			}
			heldStoreOf(g).answersGone = true
		}
		t.Run(c.name, func(t *testing.T) {
			var probe *heldProbe
			g, done := newGapsRig(t, c.probed(&probe), true, true, eaRoad{name: "direct"})
			defer done()
			before := g.remaining()
			r := c.act(g)
			heldAssertRefusal(t, g, c, r)
			saw := probe.seen()
			if len(saw) != 1 || saw[0].open != 0 || saw[0].remaining != before {
				t.Fatalf("at signing: %+v", saw)
			}
		})
	}
}
