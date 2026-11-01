package mcpserver

// The remaining "every action" stops (see the PR "Cloud MCP: sign the
// remaining server-side stops and count aged-out misses").
//
// With both signing switches on (SetSignEveryAction and
// SetSignUnavailableCheckRefusals) and a signer that works, each outcome below
// ends in a signed refusal (the call is stopped) or in a signed count (the call
// went through):
//
//	G   satgate/delegate stopped on SatGate's side: the stored handoff limit
//	    cannot be read, the child key cannot be made, the budget store gives no
//	    usable answer to the carve, the child's budget cannot be created, or the
//	    child's row cannot be written. Signed, with the response unchanged.
//	D   Observe: no server identity for the call (tools/call or a carve): the
//	    call goes through and is counted. A credit-bearing Observe delegation
//	    stopped because the pending store is full or the carve's pending entry
//	    cannot be prepared: signed.
//	H   Observe: a governed method stopped because the gap slot is full: signed.
//	    A gap that the pending journal definitely did not store: counted in
//	    memory and the call goes through, or, when money is held for it (the
//	    receipt was only prepared, never published), signed. A forwarded miss
//	    older than PendingHorizon: counted once, not dropped.
//
// With either switch off every answer and effect is the one main gives
// (TestGaps_SwitchOffBytes, against a snapshot taken on main). A signer that is
// down still stops a Control call unsigned. This file uses no symbol the
// change adds, so it compiles and runs on main, where these tests fail.

import (
	"bytes"
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
	"testing"
	"time"
)

// ---- doubles ----

// gapsStore is the budget store of these tests: holds, carve settlement and the
// durable pending budget (p2Budget), with the faults the stops need.
type gapsStore struct {
	*p2Budget
	badCarve      bool // the carve gets no usable answer (nil result, no error)
	failChildInit bool // the child's budget cannot be created
	failPrepare   bool // the carve's pending entry cannot be prepared
}

func (s *gapsStore) Hold(ctx context.Context, id, tool string, cost int64, rid string, ttl time.Duration) (*BudgetResult, error) {
	if s.badCarve && tool == "_delegation" {
		return nil, nil
	}
	return s.p2Budget.Hold(ctx, id, tool, cost, rid, ttl)
}

func (s *gapsStore) Spend(ctx context.Context, id, tool string, cost int64, rid string) (*BudgetResult, error) {
	if s.badCarve && tool == "_delegation" {
		return nil, nil
	}
	return s.p2Budget.Spend(ctx, id, tool, cost, rid)
}

func (s *gapsStore) Initialize(ctx context.Context, id string, credits int64) error {
	if s.failChildInit && strings.HasPrefix(id, "del-") {
		return errors.New("budget store down")
	}
	return s.p2Budget.Initialize(ctx, id, credits)
}

func (s *gapsStore) PreparePendingBudget(ctx context.Context, r PendingRecord) (json.RawMessage, error) {
	if s.failPrepare {
		return nil, errors.New("budget store down")
	}
	return s.p2Budget.PreparePendingBudget(ctx, r)
}

// gapsMangleAuth verifies the key as the proxy does, then hands the delegator a
// parent token it cannot read, so the child key cannot be made.
type gapsMangleAuth struct{ inner Authenticator }

func (a gapsMangleAuth) Verify(ctx context.Context, token string) (*TokenInfo, error) {
	info, err := a.inner.Verify(ctx, token)
	if err != nil || info == nil {
		return info, err
	}
	out := *info
	out.RawToken = "not-a-token"
	return &out, nil
}

// gapsFlakyWindows is the durable recorder whose window store can be made to
// refuse, counting every count it took by report id.
type gapsFlakyWindows struct {
	*p2Recorder
	mu     sync.Mutex
	refuse bool
	taken  map[string]int
}

func (r *gapsFlakyWindows) RecordWindowedCall(ctx context.Context, c MCPWindowedCall) (*MCPEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if r.refuse {
		return nil, errors.New("window store unavailable")
	}
	if r.taken == nil {
		r.taken = map[string]int{}
	}
	r.taken[c.ReportID]++
	return r.p2Recorder.RecordWindowedCall(ctx, c)
}

// gapsFailPrepare is the durable recorder whose receipt cannot be prepared.
type gapsFailPrepare struct{ *p2Recorder }

// gapsJournalWindows is the durable recorder whose counts go to the in-memory
// window recorder, so a test can sign them.
type gapsJournalWindows struct{ *p2Recorder }

func (r gapsJournalWindows) RecordWindowedCall(ctx context.Context, c MCPWindowedCall) (*MCPEvidence, error) {
	return r.p2Recorder.eaWindowRecorder.RecordWindowedCall(ctx, c)
}

func (*gapsFailPrepare) PreparePendingDecision(context.Context, MCPDecision) (*MCPEvidence, json.RawMessage, error) {
	return nil, nil, errors.New("signer gave up")
}

// ---- the rig ----

type gapsRig struct {
	*eaEnv
	store *gapsStore
	b     *p2Backend
	// durable is set when a pending journal is installed.
	durable bool
}

type gapsCase struct {
	name string
	opts eaOpts
	// reason is the unavailable-check reason the stop is signed with; "" means
	// the call goes through and is counted.
	reason string
	method string
	setup  func(t *testing.T, g *gapsRig)
	act    func(g *gapsRig) *Response
}

func newGapsRig(t *testing.T, c gapsCase, sign, refusals bool, road eaRoad) (*gapsRig, func()) {
	t.Helper()
	e := newEAEnv(t, c.opts)
	b := newP2Backend()
	hs := &eaHoldStore{InMemoryBudgetEnforcer: e.proxy.budget.(*InMemoryBudgetEnforcer)}
	gs := &gapsStore{p2Budget: &p2Budget{eaHoldStore: hs, b: b}}
	e.proxy.SetBudgetEnforcer(gs)
	if sign {
		eaSetSwitch(e, true)
	}
	outageSwitch(t, e.proxy, refusals)
	g := &gapsRig{eaEnv: e, store: gs, b: b}
	if c.setup != nil {
		c.setup(t, g)
	}
	done := func() {}
	if road.via != nil {
		done = road.via(t, e)
	}
	return g, done
}

// journal installs the durable pending journal with rec as the recorder.
func (g *gapsRig) journal(t *testing.T, rec EvidenceRecorder) {
	t.Helper()
	g.proxy.SetEvidenceRecorder(rec)
	g.proxy.SetPendingStore(p2Store{g.b})
	g.durable = true
	if g.proxy.SignEveryAction() {
		if err := g.proxy.SetSignEveryAction(true); err != nil {
			t.Fatal(err)
		}
	}
}

func (g *gapsRig) p2() *p2Recorder {
	w, _ := g.rec.(*eaWindowRecorder)
	if w == nil {
		w = newEAWindowRecorder(g.base)
	}
	return &p2Recorder{eaWindowRecorder: w, b: g.b}
}

// fill takes every slot of the kind, so the next reservation fails.
func (g *gapsRig) fill(t *testing.T, kind int) {
	t.Helper()
	if err := g.proxy.SetEveryActionSettings(EveryActionSettings{PendingBound: 1}); err != nil {
		t.Fatal(err)
	}
	slot := g.proxy.every().pending.reserve(kind)
	if slot == nil {
		t.Fatal("could not fill the pending store")
	}
	t.Cleanup(slot.release)
}

// misses is how many Observe misses the window recorder (or the journal's
// recorder) took.
func (g *gapsRig) misses() int {
	if w, ok := g.rec.(*eaWindowRecorder); ok && !g.durable {
		w.mu.Lock()
		defer w.mu.Unlock()
		return len(w.misses)
	}
	g.b.mu.Lock()
	defer g.b.mu.Unlock()
	n := len(g.b.gaps)
	if w, ok := g.rec.(*eaWindowRecorder); ok {
		w.mu.Lock()
		n += len(w.misses)
		w.mu.Unlock()
	}
	return n
}

func (g *gapsRig) decisions() []MCPDecision {
	g.base.mu.Lock()
	defer g.base.mu.Unlock()
	return append([]MCPDecision(nil), g.base.decisions...)
}

// out reports whether the call went through: forwarded, or a key issued.
func (g *gapsRig) out(r *Response) int {
	n := g.rr.forwarded()
	g.proxy.upstream.mu.Lock()
	if c := g.proxy.upstream.clients["default"]; c != nil {
		if tr, ok := c.transport.(*eaUpstreamTransport); ok {
			n += tr.count()
		}
	}
	g.proxy.upstream.mu.Unlock()
	if r != nil && r.Error == nil && r.Result != nil && strings.Contains(string(r.Result), `"token"`) {
		n++
	}
	return n
}

func gapsDelegate(budget int) func(g *gapsRig) *Response {
	return func(g *gapsRig) *Response {
		return g.send(g.raw(1, MethodSatGateDelegate, fmt.Sprintf(`"budget":%d,"scope":"free_read","label":"helper"`, budget)))
	}
}

func gapsTool(tool string) func(g *gapsRig) *Response {
	return func(g *gapsRig) *Response { return g.call(1, tool, "") }
}

func gapsMethod(method string) func(g *gapsRig) *Response {
	return func(g *gapsRig) *Response { return g.send(g.raw(1, method, "")) }
}

func gapsNoIdentity(t *testing.T, g *gapsRig) {
	g.proxy.every().callID = func(string) (string, error) { return "", errors.New("entropy unavailable") }
}

// ---- the cases ----

// gapsDelegateServerCases are the G stops, in Control and in Observe.
func gapsDelegateServerCases() []gapsCase {
	var out []gapsCase
	for _, observe := range []bool{false, true} {
		mode := map[bool]string{false: "control", true: "observe"}[observe]
		o := eaOpts{observe: observe}
		out = append(out,
			gapsCase{name: "G/handoff-unreadable/" + mode, opts: o, reason: "tool_controls_unavailable", method: MethodSatGateDelegate,
				setup: func(t *testing.T, g *gapsRig) {
					s := newLimitStore()
					s.err = errors.New("row store down")
					g.proxy.SetEventPublisher(s)
				}, act: gapsDelegate(5)},
			gapsCase{name: "G/child-key/" + mode, opts: o, reason: "tool_controls_unavailable", method: MethodSatGateDelegate,
				setup: func(t *testing.T, g *gapsRig) { g.proxy.SetAuthenticator(gapsMangleAuth{g.proxy.auth}) },
				act:   gapsDelegate(5)},
			gapsCase{name: "G/carve-answer/" + mode, opts: o, reason: "budget_backend_error", method: MethodSatGateDelegate,
				setup: func(t *testing.T, g *gapsRig) { g.store.badCarve = true }, act: gapsDelegate(5)},
			gapsCase{name: "G/child-budget/" + mode, opts: o, reason: "budget_backend_error", method: MethodSatGateDelegate,
				setup: func(t *testing.T, g *gapsRig) { g.store.failChildInit = true }, act: gapsDelegate(5)},
			gapsCase{name: "G/child-row/" + mode, opts: o, reason: "tool_controls_unavailable", method: MethodSatGateDelegate,
				setup: func(t *testing.T, g *gapsRig) { g.proxy.SetEventPublisher(&failingPersister{newLimitStore()}) },
				act:   gapsDelegate(0)},
		)
	}
	return out
}

// gapsSlotCases are the H stops of a governed method whose gap slot is full.
func gapsSlotCases() []gapsCase {
	full := func(t *testing.T, g *gapsRig) { g.fill(t, kindGap) }
	o := eaOpts{observe: true}
	return []gapsCase{
		{name: "H/slot/tools-call-priced", opts: o, reason: "pending_capacity_exhausted", method: MethodToolsCall, setup: full, act: gapsTool("paid_read")},
		{name: "H/slot/tools-call-free", opts: o, reason: "pending_capacity_exhausted", method: MethodToolsCall, setup: full, act: gapsTool("free_read")},
		{name: "H/slot/delegate", opts: o, reason: "pending_capacity_exhausted", method: MethodSatGateDelegate, setup: full, act: gapsDelegate(5)},
		{name: "H/slot/budget", opts: o, reason: "pending_capacity_exhausted", method: MethodSatGateBudget, setup: full, act: gapsMethod(MethodSatGateBudget)},
		{name: "H/slot/budget-check", opts: o, reason: "pending_capacity_exhausted", method: "budget/check", setup: full, act: gapsMethod("budget/check")},
		{name: "H/slot/forwarded-method", opts: eaOpts{observe: true, defaultUp: true}, reason: "pending_capacity_exhausted", method: "resources/list", setup: full, act: gapsMethod("resources/list")},
	}
}

// gapsObserveCases are the D and H Observe outcomes past the slot.
func gapsObserveCases() []gapsCase {
	o := eaOpts{observe: true}
	journal := func(failSave bool, rec func(g *gapsRig) EvidenceRecorder) func(t *testing.T, g *gapsRig) {
		return func(t *testing.T, g *gapsRig) {
			g.b.failSave = failSave
			g.journal(t, rec(g))
		}
	}
	plain := func(g *gapsRig) EvidenceRecorder { return g.p2() }
	noPrepare := func(g *gapsRig) EvidenceRecorder { return &gapsFailPrepare{g.p2()} }
	signerFails := func(t *testing.T, g *gapsRig) {
		g.base.mu.Lock()
		g.base.failRecord = true // Preflight passes; the allowed receipt fails
		g.base.mu.Unlock()
	}
	both := func(fs ...func(*testing.T, *gapsRig)) func(*testing.T, *gapsRig) {
		return func(t *testing.T, g *gapsRig) {
			for _, f := range fs {
				f(t, g)
			}
		}
	}
	return []gapsCase{
		// D: no server identity. Observe passes the call on and counts it.
		{name: "D/identity/tools-call-priced", opts: o, method: MethodToolsCall, setup: gapsNoIdentity, act: gapsTool("paid_read")},
		{name: "D/identity/tools-call-free", opts: o, method: MethodToolsCall, setup: gapsNoIdentity, act: gapsTool("free_read")},
		{name: "D/identity/delegate", opts: o, method: MethodSatGateDelegate, setup: gapsNoIdentity, act: gapsDelegate(5)},
		{name: "D/identity/delegate-journal", opts: o, method: MethodSatGateDelegate, setup: both(journal(false, plain), gapsNoIdentity), act: gapsDelegate(5)},
		// D: a credit-bearing Observe delegation under the journal.
		{name: "D/delegate-slot", opts: o, reason: "pending_capacity_exhausted", method: MethodSatGateDelegate,
			setup: both(journal(false, plain), func(t *testing.T, g *gapsRig) { g.fill(t, kindUncharged) }), act: gapsDelegate(5)},
		{name: "D/delegate-prepare", opts: o, reason: "budget_backend_error", method: MethodSatGateDelegate,
			setup: both(journal(false, plain), func(t *testing.T, g *gapsRig) { g.store.failPrepare = true }), act: gapsDelegate(5)},
		// H: the journal definitely did not store the gap, no money held: the
		// call goes through and the miss is counted.
		{name: "H/not-stored/tools-call-free", opts: o, method: MethodToolsCall, setup: both(journal(true, plain), signerFails), act: gapsTool("free_read")},
		{name: "H/not-stored/budget", opts: o, method: MethodSatGateBudget, setup: both(journal(true, plain), signerFails), act: gapsMethod(MethodSatGateBudget)},
		{name: "H/not-stored/forwarded-method", opts: eaOpts{observe: true, defaultUp: true}, method: "resources/list", setup: both(journal(true, plain), signerFails), act: gapsMethod("resources/list")},
		{name: "H/not-stored/helper", opts: o, method: MethodSatGateDelegate, setup: both(journal(true, plain), signerFails), act: gapsDelegate(0)},
		// H: money is held and the receipt was only prepared: signed refusal.
		{name: "H/not-stored-held/tools-call", opts: o, reason: "pending_capacity_exhausted", method: MethodToolsCall, setup: journal(true, noPrepare), act: gapsTool("paid_read")},
		{name: "H/not-stored-held/delegate", opts: o, reason: "pending_capacity_exhausted", method: MethodSatGateDelegate, setup: journal(true, noPrepare), act: gapsDelegate(5)},
	}
}

func gapsAllCases() []gapsCase {
	return append(append(gapsDelegateServerCases(), gapsSlotCases()...), gapsObserveCases()...)
}

// ---- assertions ----

func gapsAssertRefusal(t *testing.T, g *gapsRig, c gapsCase, r *Response, before int64) {
	t.Helper()
	if r == nil || r.Error == nil || g.out(r) != 0 {
		t.Fatalf("stop went through: %s", eaShape(r))
	}
	if g.remaining() != before {
		t.Fatalf("stop changed the balance: %d -> %d", before, g.remaining())
	}
	ds := g.decisions()
	if len(ds) != 1 {
		t.Fatalf("want exactly one signed refusal, got %d: %v", len(ds), g.base.signed())
	}
	d := ds[0]
	if d.Decision != "denied" || d.DecisionReason != c.reason || !d.BudgetNotEvaluated || d.NoVerifiedCapability {
		t.Fatalf("wrong refusal: %s", eaLabel(d))
	}
	if d.CostCredits != 0 || d.RemainingCredits != 0 || d.RemainingBeforeCredits != 0 || d.BudgetLimitCredits != 0 || d.DenialCode != "" {
		t.Fatal("refusal claimed amounts or a code")
	}
	if d.MCPMethod != c.method || !strings.HasPrefix(d.RequestID, callIDPrefix) || d.TokenID == "" || d.JSONRPCID != "" {
		t.Fatalf("refusal identity: method %q request %q", d.MCPMethod, d.RequestID)
	}
	if outageEvidence(r)["receipt_id"] != "rcpt_1" {
		t.Fatalf("missing refusal handles: %s", eaShape(r))
	}
	if g.misses() != 0 {
		t.Fatal("a stopped call was also counted as a miss")
	}
}

// wentThrough reports whether an Observe call that was not stopped went
// through: forwarded, a key issued, or (satgate/budget) answered by SatGate.
func (g *gapsRig) wentThrough(method string, r *Response) bool {
	if r == nil || r.Error != nil {
		return false
	}
	return g.out(r) == 1 || (method == MethodSatGateBudget && g.out(r) == 0)
}

func gapsAssertCounted(t *testing.T, g *gapsRig, method string, r *Response) {
	t.Helper()
	if !g.wentThrough(method, r) {
		t.Fatalf("Observe call did not go through: %s", eaShape(r))
	}
	for _, s := range g.base.signed() {
		if strings.HasPrefix(s, "denied/") {
			t.Fatalf("Observe call that went through got a refusal: %s", s)
		}
	}
	if n := g.misses(); n != 1 {
		t.Fatalf("want the miss counted exactly once, got %d", n)
	}
}

// strip removes the receipt's handles from an error response, so a signed
// refusal can be compared with the answer main gives.
func gapsStrip(t *testing.T, r *Response) string {
	t.Helper()
	if r == nil || r.Error == nil {
		return eaShape(r)
	}
	data := map[string]any{}
	_ = json.Unmarshal(r.Error.Data, &data)
	for _, k := range []string{"receipt_id", "receipt_hash", "evidence_pack_id", "evidence_url"} {
		delete(data, k)
	}
	out := *r
	e := *r.Error
	e.Data, _ = json.Marshal(data)
	out.Error = &e
	return eaShape(&out)
}

// ---- tests ----

// G, D and H stops: each is signed with its reason, the answer is the one main
// gives plus the receipt's handles, nothing is sent, issued or charged.
func TestGaps_StopsAreSigned(t *testing.T) {
	for _, c := range gapsAllCases() {
		if c.reason == "" {
			continue
		}
		for _, road := range eaRoads {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				off, doneOff := newGapsRig(t, c, true, false, road)
				want := gapsStrip(t, c.act(off))
				doneOff()
				g, done := newGapsRig(t, c, true, true, road)
				defer done()
				before := g.remaining()
				r := c.act(g)
				gapsAssertRefusal(t, g, c, r, before)
				if got := gapsStrip(t, r); got != want {
					t.Fatalf("answer changed:\n got %s\nwant %s", got, want)
				}
				_, holds, commits, releases := g.store.counts()
				if commits != 0 || holds != releases {
					t.Fatalf("a hold was kept or committed: holds %d commits %d releases %d", holds, commits, releases)
				}
			})
		}
	}
}

// D and H Observe outcomes that go through: no receipt, one counted miss.
func TestGaps_ObserveCallsAreCounted(t *testing.T) {
	for _, c := range gapsObserveCases() {
		if c.reason != "" {
			continue
		}
		for _, road := range eaRoads {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				g, done := newGapsRig(t, c, true, true, road)
				defer done()
				r := c.act(g)
				gapsAssertCounted(t, g, c.method, r)
				if strings.HasPrefix(c.name, "H/not-stored") && len(g.b.rows) != 0 {
					t.Fatal("a gap the journal refused is in the journal")
				}
			})
		}
	}
}

// A signer that is down signs nothing: the stop stays unsigned, in Control
// and in Observe, and nothing is sent, issued or charged.
func TestGaps_SignerDownStaysUnsigned(t *testing.T) {
	for _, c := range gapsAllCases() {
		if c.reason == "" {
			continue
		}
		for _, down := range []string{"preflight", "record"} {
			t.Run(c.name+"/"+down, func(t *testing.T) {
				g, done := newGapsRig(t, c, true, true, eaRoad{name: "direct"})
				defer done()
				g.base.mu.Lock()
				g.base.failPreflight = down == "preflight"
				g.base.failRecord = true
				g.base.mu.Unlock()
				before := g.remaining()
				r := c.act(g)
				if r == nil || r.Error == nil || g.out(r) != 0 || g.remaining() != before {
					t.Fatalf("signer down: the stop went through: %s", eaShape(r))
				}
				if len(g.decisions()) != 0 || outageEvidence(r)["receipt_id"] != nil {
					t.Fatal("signer down: something was signed")
				}
				if !c.opts.observe && !eaIsProofUnavailable(r) {
					t.Fatalf("Control with the signer down must answer proof_unavailable: %s", eaShape(r))
				}
			})
		}
	}
}

// Control with no server identity still stops unsigned (an identity failure
// is not signed), in tools/call and in a carve.
func TestGaps_ControlIdentityFailureStaysUnsigned(t *testing.T) {
	for _, act := range []func(*gapsRig) *Response{gapsTool("paid_read"), gapsTool("free_read"), gapsDelegate(5)} {
		g, done := newGapsRig(t, gapsCase{setup: gapsNoIdentity}, true, true, eaRoad{name: "direct"})
		r := act(g)
		done()
		if !eaIsProofUnavailable(r) || g.out(r) != 0 || len(g.base.signed()) != 0 || g.misses() != 0 {
			t.Fatalf("Control identity failure: %s signed %v", eaShape(r), g.base.signed())
		}
	}
}

// With the unavailable-check switch off, an Observe call with no server
// identity is answered as main answers it at that step: proof_unavailable,
// nothing signed, counted, sent or charged.
func TestGaps_IdentityFailureSwitchOffIsMains(t *testing.T) {
	for _, c := range gapsObserveCases() {
		if !strings.HasPrefix(c.name, "D/identity/") {
			continue
		}
		t.Run(c.name, func(t *testing.T) {
			g, done := newGapsRig(t, c, true, false, eaRoad{name: "direct"})
			defer done()
			before := g.remaining()
			r := c.act(g)
			if !eaIsProofUnavailable(r) || g.out(r) != 0 || len(g.base.signed()) != 0 || g.misses() != 0 || g.remaining() != before {
				t.Fatalf("switch off: %s signed %v", eaShape(r), g.base.signed())
			}
		})
	}
}

// H: a forwarded Observe miss the journal kept, whose count kept failing until
// it was older than PendingHorizon, is counted once when the recorder is back,
// and never again.
func TestGaps_AgedOutMissIsCountedOnce(t *testing.T) {
	for _, refusals := range []bool{true, false} {
		for _, road := range eaRoads {
			t.Run(fmt.Sprintf("refusals=%v/%s", refusals, road.name), func(t *testing.T) {
				var rec *gapsFlakyWindows
				c := gapsCase{opts: eaOpts{observe: true}, setup: func(t *testing.T, g *gapsRig) {
					rec = &gapsFlakyWindows{p2Recorder: g.p2(), refuse: true}
					g.journal(t, rec)
					g.base.mu.Lock()
					g.base.failRecord = true
					g.base.mu.Unlock()
				}}
				g, done := newGapsRig(t, c, true, refusals, road)
				defer done()
				r := g.call(1, "free_read", "")
				if r.Error != nil || g.out(r) != 1 {
					t.Fatalf("Observe call did not go through: %s", eaShape(r))
				}
				g.b.mu.Lock()
				if len(g.b.rows) != 1 || len(g.b.completed) != 0 {
					g.b.mu.Unlock()
					t.Fatal("the miss was not kept open in the journal")
				}
				var id string
				for k, raw := range g.b.rows {
					var w PendingWork
					if err := json.Unmarshal(raw, &w); err != nil {
						g.b.mu.Unlock()
						t.Fatal(err)
					}
					old := RateWindowStart(time.Now().Add(-PendingHorizon - time.Hour))
					w.Record.Minute, w.Window.WindowStart = old, old
					g.b.rows[k], _ = json.Marshal(w)
					id = k
				}
				g.b.mu.Unlock()
				rec.mu.Lock()
				rec.refuse = false
				rec.mu.Unlock()
				g.proxy.FlushPending(context.Background())
				g.proxy.FlushPending(context.Background())
				rec.mu.Lock()
				taken := rec.taken[id]
				rec.mu.Unlock()
				g.b.mu.Lock()
				disposition := g.b.completed[id]
				g.b.mu.Unlock()
				if !refusals {
					// Main's behaviour: the final no-op.
					if taken != 0 || disposition != PendingAgedOut {
						t.Fatalf("switch off changed the horizon: taken %d disposition %q", taken, disposition)
					}
					return
				}
				if taken != 1 || disposition != PendingCompleted {
					t.Fatalf("aged-out miss: counted %d times, disposition %q", taken, disposition)
				}
			})
		}
	}
}

// ---- switches off ----

type gapsSnapshot struct {
	Case      string `json:"case"`
	Switches  string `json:"switches"`
	Response  string `json:"response"`
	Signed    string `json:"signed"`
	Out       int    `json:"out"`
	Debit     int64  `json:"debit"`
	Misses    int    `json:"misses"`
	Journaled int    `json:"journaled"`
}

// With either switch off every answer and effect is the one main gives. The
// baseline was written by this test on main (GAPS_OFF_SNAPSHOT).
func TestGaps_SwitchOffBytes(t *testing.T) {
	var got []gapsSnapshot
	for _, c := range gapsAllCases() {
		if strings.HasPrefix(c.name, "D/identity/") {
			// The identity fault is injected through a hook main does not
			// consult; see TestGaps_IdentityFailureSwitchOffIsMains.
			continue
		}
		for _, sw := range []struct {
			name           string
			sign, refusals bool
		}{{"both-off", false, false}, {"sign-only", true, false}, {"refusals-only", false, true}} {
			g, done := newGapsRig(t, c, sw.sign, sw.refusals, eaRoad{name: "direct"})
			before := g.remaining()
			r := c.act(g)
			done()
			got = append(got, gapsSnapshot{Case: c.name, Switches: sw.name, Response: eaShape(r),
				Signed: strings.Join(g.base.signed(), ","), Out: g.out(r), Debit: before - g.remaining(),
				Misses: g.misses(), Journaled: len(g.b.rows)})
		}
	}
	if p := os.Getenv("GAPS_OFF_SNAPSHOT"); p != "" {
		b, _ := json.MarshalIndent(got, "", "  ")
		if err := os.WriteFile(p, append(b, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "every-action-gaps-off-0a2e8e23.json"))
	if err != nil {
		t.Fatal(err)
	}
	var want []gapsSnapshot
	if err := json.Unmarshal(raw, &want); err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(want, got) {
		for i := range want {
			if i < len(got) && !reflect.DeepEqual(want[i], got[i]) {
				t.Errorf("%s/%s differs from main:\n got %+v\nwant %+v", want[i].Case, want[i].Switches, got[i], want[i])
			}
		}
		t.Fatal("switch-off behaviour differs from main")
	}
}

// ---- the public checkers ----

// gapsPackRecorder signs what the proxy asks for through the test-only fixture
// (tools/mcp_every_action_gaps_fixture.py): a refusal when it is asked for one,
// and, at the end, one gap receipt per key, route and minute of the misses the
// window recorder took. Only a projection with hashed identities crosses over.
type gapsPackRecorder struct {
	*eaWindowRecorder
	pack *outagePackRecorder
}

// gapsDurableRecorder is what a journal's recorder offers.
type gapsDurableRecorder interface {
	EvidenceRecorder
	PendingEvidenceRecorder
	WindowRecorder
}

// gapsSignVia is a journal's recorder whose ordinary receipts are signed by sign.
type gapsSignVia struct {
	gapsDurableRecorder
	sign EvidenceRecorder
}

func (r gapsSignVia) RecordMCPDecision(ctx context.Context, d MCPDecision) (*MCPEvidence, error) {
	return r.sign.RecordMCPDecision(ctx, d)
}

func (r *gapsPackRecorder) RecordMCPDecision(ctx context.Context, d MCPDecision) (*MCPEvidence, error) {
	ev, err := r.eaWindowRecorder.RecordMCPDecision(ctx, d)
	if err != nil || d.Decision != "denied" {
		// Only the refusals under test go to the fixture.
		return ev, err
	}
	return r.pack.RecordMCPDecision(ctx, d)
}

func (r *gapsPackRecorder) signGaps(t *testing.T) int {
	t.Helper()
	r.mu.Lock()
	misses := append([]MCPWindowedCall(nil), r.misses...)
	r.mu.Unlock()
	type window struct {
		call  MCPWindowedCall
		count int
	}
	var order []string
	windows := map[string]*window{}
	for _, m := range misses {
		k := strings.Join([]string{m.Decision.TokenID, m.Decision.RouteOrTool, m.WindowStart.UTC().Format(time.RFC3339)}, "|")
		if windows[k] == nil {
			windows[k] = &window{call: m}
			order = append(order, k)
		}
		windows[k].count++
	}
	for _, k := range order {
		w := windows[k]
		d := w.call.Decision
		projection := map[string]any{"decision": d.Decision, "reason": d.DecisionReason, "budget_not_evaluated": d.BudgetNotEvaluated,
			"request_id": w.call.ReportID, "route_or_tool": d.RouteOrTool, "policy_mode": d.PolicyMode,
			"capability_hash": outageHash(d.TokenID), "key_hash": outageHash(d.BudgetID)}
		if _, err := r.pack.exchange(map[string]any{"action": "gap", "decision": projection,
			"window_start": w.call.WindowStart.UTC().Format("2006-01-02T15:04:05Z"), "count": w.count}); err != nil {
			t.Fatal(err)
		}
	}
	return len(misses)
}

// Every new refusal and every new count, made by the real proxy, signed with
// real Ed25519 keys and checked by the public checkers in tools/, demo/ and the
// frozen unavailable-check-v1 bundle, pack by pack and as one whole record.
func TestGaps_PublicCheckersAcceptThem(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("GAPS_PACK_DIR")
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
	var stderr bytes.Buffer
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
	pick := func(name string) gapsCase {
		for _, c := range gapsAllCases() {
			if c.name == name {
				return c
			}
		}
		t.Fatalf("no case %s", name)
		return gapsCase{}
	}
	refusals := map[string]int{}
	misses := 0
	for _, name := range []string{
		"G/handoff-unreadable/control", "G/child-key/observe", "G/carve-answer/control", "G/child-budget/observe", "G/child-row/control",
		"H/slot/tools-call-priced", "H/slot/delegate", "H/slot/budget", "H/slot/forwarded-method",
		"D/delegate-slot", "D/delegate-prepare", "H/not-stored-held/delegate",
		"D/identity/tools-call-priced", "D/identity/delegate", "H/not-stored/budget",
	} {
		c := pick(name)
		t.Run(name, func(t *testing.T) {
			var rec *gapsPackRecorder
			setup := c.setup
			c.setup = func(t *testing.T, g *gapsRig) {
				rec = &gapsPackRecorder{eaWindowRecorder: newEAWindowRecorder(g.base), pack: pack}
				g.rec = rec.eaWindowRecorder
				g.proxy.SetEvidenceRecorder(rec)
				if err := g.proxy.SetSignEveryAction(true); err != nil {
					t.Fatal(err)
				}
				if strings.HasPrefix(name, "H/not-stored/") {
					// The journal's own recorder counts through the same windows.
					g.b.failSave = true
					g.journal(t, gapsJournalWindows{&p2Recorder{eaWindowRecorder: rec.eaWindowRecorder, b: g.b}})
					g.base.mu.Lock()
					g.base.failRecord = true
					g.base.mu.Unlock()
				} else if setup != nil {
					setup(t, g)
				}
				if g.durable {
					// A journal's recorder signs its ordinary receipts here too.
					g.proxy.SetEvidenceRecorder(gapsSignVia{g.proxy.evidence.(gapsDurableRecorder), rec})
					if err := g.proxy.SetSignEveryAction(true); err != nil {
						t.Fatal(err)
					}
				}
			}
			g, done := newGapsRig(t, c, true, true, eaRoads[1])
			defer done()
			r := c.act(g)
			if c.reason != "" {
				data := outageEvidence(r)
				if data["receipt_id"] == nil || g.out(r) != 0 {
					t.Fatalf("no signed refusal: %s", eaShape(r))
				}
				refusals[c.reason]++
				return
			}
			if !g.wentThrough(c.method, r) {
				t.Fatalf("Observe call did not go through: %s", eaShape(r))
			}
			misses += rec.signGaps(t)
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
	want := map[string]int{"observe_unrecorded": 3}
	for k, v := range refusals {
		want[k] = v
	}
	if !reflect.DeepEqual(result.Counts, want) || misses != 3 || result.Unrecorded != 3 {
		t.Fatalf("record counts %v (want %v), unrecorded %d, misses %d", result.Counts, want, result.Unrecorded, misses)
	}
	t.Logf("%d packs valid and trusted in tools, demo and the unavailable-check-v1 bundle; whole record valid, trusted and complete; counts %v", result.Packs, result.Counts)
}
