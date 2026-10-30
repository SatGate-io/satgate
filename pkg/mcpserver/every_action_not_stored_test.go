package mcpserver

// The last definite stops of a verified key's action that ended with nothing
// signed, counted or kept (see the PR "MCP: sign the stop when a delegation's
// prepared receipt is not stored").
//
// With both signing switches on and a signer that works, each case below now
// ends in a signed refusal, in Control and in Observe, over direct dispatch,
// Streamable HTTP and SSE:
//
//	not-stored  a priced tools/call or a credit-bearing satgate/delegate whose
//	            money is held and whose allowed receipt was only prepared,
//	            when the pending journal definitely did not store it
//	            (pending_capacity_exhausted);
//	slot        no pending slot for the held amount: a priced Observe
//	            tools/call, a credit-bearing Control delegation
//	            (pending_capacity_exhausted);
//	prepare     the budget store could not prepare the held amount's pending
//	            entry: a priced tools/call in either mode, a credit-bearing
//	            Control delegation (budget_backend_error).
//
// A save whose result is unknown keeps its outcome_unknown answer. With either
// switch off every answer and effect is the one main gives
// (TestNotStored_SwitchOffBytes, against a snapshot this test wrote on main
// 12b79cfc). This file uses no symbol the change adds, so it compiles and runs
// on main, where the signed-stop tests fail.

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

// notStoredCases are the stops this change signs.
func notStoredCases() []gapsCase {
	plain := func(g *gapsRig) EvidenceRecorder { return g.p2() }
	journal := func(failSave bool) func(t *testing.T, g *gapsRig) {
		return func(t *testing.T, g *gapsRig) {
			g.b.failSave = failSave
			g.journal(t, plain(g))
		}
	}
	both := func(fs ...func(*testing.T, *gapsRig)) func(*testing.T, *gapsRig) {
		return func(t *testing.T, g *gapsRig) {
			for _, f := range fs {
				f(t, g)
			}
		}
	}
	slotFull := func(t *testing.T, g *gapsRig) { g.fill(t, kindUncharged) }
	prepareFails := func(t *testing.T, g *gapsRig) { g.store.failPrepare = true }
	var out []gapsCase
	for _, observe := range []bool{false, true} {
		mode := map[bool]string{false: "control", true: "observe"}[observe]
		o := eaOpts{observe: observe}
		out = append(out,
			gapsCase{name: "not-stored/delegate/" + mode, opts: o, reason: "pending_capacity_exhausted", method: MethodSatGateDelegate,
				setup: journal(true), act: gapsDelegate(5)},
			gapsCase{name: "not-stored/tools-call/" + mode, opts: o, reason: "pending_capacity_exhausted", method: MethodToolsCall,
				setup: journal(true), act: gapsTool("paid_read")},
			gapsCase{name: "prepare/tools-call/" + mode, opts: o, reason: "budget_backend_error", method: MethodToolsCall,
				setup: both(journal(false), prepareFails), act: gapsTool("paid_read")},
		)
	}
	out = append(out,
		gapsCase{name: "slot/tools-call/observe", opts: eaOpts{observe: true}, reason: "pending_capacity_exhausted", method: MethodToolsCall,
			setup: both(journal(false), slotFull), act: gapsTool("paid_read")},
		gapsCase{name: "slot/delegate/control", opts: eaOpts{}, reason: "pending_capacity_exhausted", method: MethodSatGateDelegate,
			setup: slotFull, act: gapsDelegate(5)},
		gapsCase{name: "slot/delegate-journal/control", opts: eaOpts{}, reason: "pending_capacity_exhausted", method: MethodSatGateDelegate,
			setup: both(journal(false), slotFull), act: gapsDelegate(5)},
		gapsCase{name: "prepare/delegate/control", opts: eaOpts{}, reason: "budget_backend_error", method: MethodSatGateDelegate,
			setup: both(journal(false), prepareFails), act: gapsDelegate(5)},
	)
	return out
}

// notStoredUnknownCases are saves whose result is unknown: the journal lost
// the answer and could not resolve it. They keep their outcome_unknown answer.
func notStoredUnknownCases() []gapsCase {
	unknown := func(t *testing.T, g *gapsRig) {
		g.journal(t, g.p2())
		g.proxy.SetPendingStore(p2LostReply{p2Store{g.b}, true})
	}
	var out []gapsCase
	for _, observe := range []bool{false, true} {
		mode := map[bool]string{false: "control", true: "observe"}[observe]
		o := eaOpts{observe: observe}
		out = append(out,
			gapsCase{name: "unknown/delegate/" + mode, opts: o, method: MethodSatGateDelegate, setup: unknown, act: gapsDelegate(5)},
			gapsCase{name: "unknown/tools-call/" + mode, opts: o, method: MethodToolsCall, setup: unknown, act: gapsTool("paid_read")},
		)
	}
	return out
}

// Each stop is signed with its reason; the answer is the one it gets with the
// refusal switch off plus the receipt's handles; no key is issued, nothing is
// forwarded, the parent's balance is unchanged, the hold is released, nothing
// is published, journaled or counted.
func TestNotStored_StopsAreSigned(t *testing.T) {
	for _, c := range notStoredCases() {
		for _, road := range eaRoads {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				off, doneOff := newGapsRig(t, c, true, false, road)
				want := gapsStrip(t, c.act(off))
				doneOff()
				if !strings.Contains(want, "proof_unavailable") {
					t.Fatalf("with the refusal switch off the answer is not proof_unavailable: %s", want)
				}
				g, done := newGapsRig(t, c, true, true, road)
				defer done()
				before := g.remaining()
				r := c.act(g)
				gapsAssertRefusal(t, g, c, r, before)
				if got := gapsStrip(t, r); got != want {
					t.Fatalf("answer changed:\n got %s\nwant %s", got, want)
				}
				_, holds, commits, releases := g.store.counts()
				if holds != 0 && (commits != 0 || holds != releases) {
					t.Fatalf("a hold was kept or committed: holds %d commits %d releases %d", holds, commits, releases)
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
				if rec, ok := g.proxy.evidence.(*p2Recorder); ok && strings.HasPrefix(c.name, "not-stored/") && rec.prepared != 1 {
					t.Fatalf("the allowed receipt was prepared %d times, want 1", rec.prepared)
				}
			})
		}
	}
}

// A signer that is down signs nothing: these stops stay unsigned, and nothing
// is sent, issued or charged.
func TestNotStored_SignerDownStaysUnsigned(t *testing.T) {
	for _, c := range notStoredCases() {
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
				if !eaIsProofUnavailable(r) {
					t.Fatalf("signer down must answer proof_unavailable: %s", eaShape(r))
				}
			})
		}
	}
}

// A save whose result is unknown is never turned into a refusal: the answer is
// outcome_unknown, nothing is signed, issued or forwarded, and the answer is the
// same with the refusal switch on and off.
func TestNotStored_OutcomeUnknownUnchanged(t *testing.T) {
	for _, c := range notStoredUnknownCases() {
		for _, road := range eaRoads {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				off, doneOff := newGapsRig(t, c, true, false, road)
				want := eaShape(c.act(off))
				doneOff()
				g, done := newGapsRig(t, c, true, true, road)
				defer done()
				r := c.act(g)
				if outageEvidence(r)["error"] != "outcome_unknown" || g.out(r) != 0 {
					t.Fatalf("want outcome_unknown and nothing out: %s", eaShape(r))
				}
				if len(g.decisions()) != 0 || outageEvidence(r)["receipt_id"] != nil || g.misses() != 0 {
					t.Fatalf("an unknown outcome was signed or counted: %v", g.base.signed())
				}
				if got := eaShape(r); got != want {
					t.Fatalf("answer differs with the refusal switch on:\n got %s\nwant %s", got, want)
				}
			})
		}
	}
}

// With either switch off every answer and effect is the one main gives, and an
// unknown outcome is the one main gives with both switches on too. The baseline
// was written by this test on main 12b79cfc (NOT_STORED_OFF_SNAPSHOT).
func TestNotStored_SwitchOffBytes(t *testing.T) {
	type run struct {
		name           string
		sign, refusals bool
	}
	off := []run{{"both-off", false, false}, {"sign-only", true, false}, {"refusals-only", false, true}}
	var got []gapsSnapshot
	add := func(c gapsCase, runs []run) {
		for _, sw := range runs {
			g, done := newGapsRig(t, c, sw.sign, sw.refusals, eaRoad{name: "direct"})
			before := g.remaining()
			r := c.act(g)
			done()
			g.b.mu.Lock()
			rows := len(g.b.rows)
			g.b.mu.Unlock()
			got = append(got, gapsSnapshot{Case: c.name, Switches: sw.name, Response: eaShape(r),
				Signed: strings.Join(g.base.signed(), ","), Out: g.out(r), Debit: before - g.remaining(),
				Misses: g.misses(), Journaled: rows})
		}
	}
	for _, c := range notStoredCases() {
		add(c, off)
	}
	for _, c := range notStoredUnknownCases() {
		add(c, append(off, run{"both-on", true, true}))
	}
	if p := os.Getenv("NOT_STORED_OFF_SNAPSHOT"); p != "" {
		b, _ := json.MarshalIndent(got, "", "  ")
		if err := os.WriteFile(p, append(b, '\n'), 0o600); err != nil {
			t.Fatal(err)
		}
		return
	}
	raw, err := os.ReadFile(filepath.Join("testdata", "every-action-not-stored-off-12b79cfc.json"))
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
		t.Fatalf("switch-off behaviour differs from main (%d rows, want %d)", len(got), len(want))
	}
}

// Every stop this change signs, made by the real proxy over Streamable HTTP,
// signed with real Ed25519 keys by the test-only fixture, and checked by the
// public checkers in tools/, demo/ and the frozen unavailable-check-v1 bundle,
// pack by pack and as one whole record (NOT_STORED_PACK_DIR keeps the packs).
func TestNotStored_PublicCheckersAcceptThem(t *testing.T) {
	root, err := filepath.Abs("../..")
	if err != nil {
		t.Fatal(err)
	}
	dir := os.Getenv("NOT_STORED_PACK_DIR")
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
	refusals := map[string]int{}
	for _, c := range notStoredCases() {
		c := c
		t.Run(c.name, func(t *testing.T) {
			setup := c.setup
			c.setup = func(t *testing.T, g *gapsRig) {
				rec := &gapsPackRecorder{eaWindowRecorder: newEAWindowRecorder(g.base), pack: pack}
				g.rec = rec.eaWindowRecorder
				g.proxy.SetEvidenceRecorder(rec)
				if err := g.proxy.SetSignEveryAction(true); err != nil {
					t.Fatal(err)
				}
				if setup != nil {
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
			before := g.remaining()
			r := c.act(g)
			if outageEvidence(r)["receipt_id"] == nil || g.out(r) != 0 || g.remaining() != before {
				t.Fatalf("no signed refusal: %s", eaShape(r))
			}
			refusals[c.reason]++
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
	if !reflect.DeepEqual(result.Counts, refusals) || result.Unrecorded != 0 || result.Packs != len(notStoredCases()) {
		t.Fatalf("record counts %v (want %v), unrecorded %d, packs %d", result.Counts, refusals, result.Unrecorded, result.Packs)
	}
	t.Logf("%d packs valid and trusted in tools, demo and the unavailable-check-v1 bundle; whole record valid, trusted and complete; counts %v", result.Packs, result.Counts)
}
