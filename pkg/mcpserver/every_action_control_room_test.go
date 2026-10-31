package mcpserver

// A held call's unknown outcome is kept in the room the call already set
// aside (the pending slot it reserved before any money was held or anything
// was sent), so keeping it never depends on free room elsewhere.
//
// A Control priced tools/call and a Control credit-bearing satgate/delegate
// have no Observe gap slot. Each reserves a slot of the uncharged kind before
// it goes ahead (a full store already ends in a signed
// pending_capacity_exhausted stop). When the journal's answer to the save of
// the call's prepared receipt is unknown, that slot now holds the unknown
// outcome, even while the gap kind of the store is at its bound. Once the
// journal answers:
//
//	row absent  the hold is given back once and one budget_backend_error
//	            stop is signed;
//	row found   the prepared receipt is published once and the budget is
//	            settled once.
//
// Either way the store ends with no entry and no reservation of the call.
// Answers are unchanged (outcome_unknown). With either switch off nothing is
// kept (TestNotStored_SwitchOffBytes holds the bytes).

import (
	"context"
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// controlRoomCases are the held Control calls of the unknown inventory.
func controlRoomCases() []unknownCase {
	var out []unknownCase
	for _, c := range unknownCases() {
		if !c.observe && c.held {
			out = append(out, c)
		}
	}
	return out
}

// roomState is what the in-memory pending store counts: entries and
// reservations, per kind.
func (u *unknownRig) roomState() (count, reserved [kinds]int) {
	s := u.proxy.every().pending
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count, s.reserved
}

// roomSink is a journal recorder that is also an UnchargedRecorder: it counts
// what the retry worker hands it as an uncharged amount.
type roomSink struct {
	*unknownWindows
	smu  sync.Mutex
	sunk []PendingRecord
}

func (s *roomSink) RecordUncharged(_ context.Context, rec PendingRecord) error {
	s.smu.Lock()
	defer s.smu.Unlock()
	s.sunk = append(s.sunk, rec)
	return nil
}

func (s *roomSink) sunkCount() int {
	s.smu.Lock()
	defer s.smu.Unlock()
	return len(s.sunk)
}

// roomFill bounds the store at one entry per kind and takes the only gap
// slot, so the gap kind is full and the uncharged kind has room for one call.
// It returns the filler's release.
func roomFill(t *testing.T, g *gapsRig) func() {
	t.Helper()
	if err := g.proxy.SetEveryActionSettings(EveryActionSettings{PendingBound: 1}); err != nil {
		t.Fatal(err)
	}
	slot := g.proxy.every().pending.reserve(kindGap)
	if slot == nil {
		t.Fatal("could not fill the gap kind of the pending store")
	}
	return slot.release
}

// roomRig is an unknown-outcome rig whose gap kind is full. With sink the
// journal's recorder is an UnchargedRecorder as well.
func roomRig(t *testing.T, c unknownCase, landed bool, road eaRoad, sink bool) (u *unknownRig, rs *roomSink, releaseFiller, done func()) {
	t.Helper()
	u, done = newUnknownRig(t, c, landed, true, true, road, func(t *testing.T, g *gapsRig) {
		releaseFiller = roomFill(t, g)
		if err := g.proxy.SetEveryActionSettings(EveryActionSettings{PendingBound: 1, PendingRetry: 5 * time.Millisecond}); err != nil {
			t.Fatal(err)
		}
		if sink {
			rs = &roomSink{unknownWindows: g.proxy.evidence.(*unknownWindows)}
			g.proxy.SetEvidenceRecorder(rs)
			if err := g.proxy.SetSignEveryAction(true); err != nil {
				t.Fatal(err)
			}
		}
	})
	return u, rs, releaseFiller, done
}

// assertKeptInOwnSlot: the unknown call is one entry of the uncharged kind,
// counted once; its reservation is gone (put turned it into the entry); the
// gap kind is untouched (only the filler's reservation); and the bound
// holds: the kind has no room left for another call.
func assertKeptInOwnSlot(t *testing.T, u *unknownRig, c unknownCase) {
	t.Helper()
	count, reserved := u.roomState()
	if count[kindUncharged] != 1 || count[kindGap] != 0 || reserved[kindUncharged] != 0 || reserved[kindGap] != 1 {
		t.Fatalf("the unknown call is not one entry in its own slot: count %v reserved %v (gap, uncharged)", count, reserved)
	}
	recs := u.proxy.every().pending.records()
	if len(recs) != 1 || recs[0].Method != c.method || recs[0].Kind != PendingUncharged || recs[0].TokenID == "" {
		t.Fatalf("kept entries %+v", recs)
	}
	if extra := u.proxy.every().pending.reserve(kindUncharged); extra != nil {
		extra.release()
		t.Fatal("the bound does not count the kept call: another slot was handed out")
	}
}

// assertRoomBack: nothing of the call is left in the store, and its room can
// be reserved again.
func assertRoomBack(t *testing.T, u *unknownRig, releaseFiller func()) {
	t.Helper()
	count, reserved := u.roomState()
	if count != [kinds]int{} || reserved[kindUncharged] != 0 || reserved[kindGap] != 1 {
		t.Fatalf("the store still holds the call: count %v reserved %v (gap, uncharged)", count, reserved)
	}
	slot := u.proxy.every().pending.reserve(kindUncharged)
	if slot == nil {
		t.Fatal("the call's room was not given back")
	}
	slot.release()
	releaseFiller()
	if count, reserved = u.roomState(); count != [kinds]int{} || reserved != [kinds]int{} {
		t.Fatalf("after the filler: count %v reserved %v", count, reserved)
	}
}

// Row absent, gap kind full: the call is kept in its own slot; once the
// journal answers, its hold is given back once and one budget_backend_error
// stop is signed; nothing is published, sent or issued.
func TestControlRoom_AbsentEndsRecorded(t *testing.T) {
	for _, c := range controlRoomCases() {
		for _, road := range eaRoads {
			for _, sink := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/sink=%v", c.name, road.name, sink), func(t *testing.T) {
					u, rs, releaseFiller, done := roomRig(t, c, false, road, sink)
					defer done()
					r := c.act(u.gapsRig)
					assertUnknownAnswer(t, u, r)
					assertKeptInOwnSlot(t, u, c)
					u.up()
					u.proxy.FlushPending(context.Background())
					unknownEventually(t, "the stop is signed", func() bool { return len(u.decisions()) == 1 })
					unknownEventually(t, "the entry is gone", func() bool { return u.kept() == 0 })
					u.proxy.FlushPending(context.Background())
					u.proxy.FlushPending(context.Background())
					ds := u.decisions()
					if len(ds) != 1 || u.misses() != 0 {
						t.Fatalf("signed %v misses %d, want one stop", u.base.signed(), u.misses())
					}
					d := ds[0]
					if d.Decision != "denied" || d.DecisionReason != "budget_backend_error" || !d.BudgetNotEvaluated || d.MCPMethod != c.method || d.TokenID == "" || !strings.HasPrefix(d.RequestID, callIDPrefix) || d.JSONRPCID != "" {
						t.Fatalf("stop: %s method %q request %q", eaLabel(d), d.MCPMethod, d.RequestID)
					}
					receipts, completed, cancelled := u.published()
					if receipts != 0 || completed != 0 || cancelled != 1 || u.out(r) != 0 {
						t.Fatalf("published %d completed %d cancelled %d out %d", receipts, completed, cancelled, u.out(r))
					}
					if _, holds, commits, releases := u.store.counts(); holds != 1 || commits != 0 || releases != 1 {
						t.Fatalf("hold: holds %d commits %d releases %d, want given back once", holds, commits, releases)
					}
					if rs != nil && rs.sunkCount() != 0 {
						t.Fatalf("the kept unknown was handed to the uncharged recorder %d times", rs.sunkCount())
					}
					assertRoomBack(t, u, releaseFiller)
				})
			}
		}
	}
}

// Row found, gap kind full: the save landed and only its answer was lost. The
// retry worker alone (no host flush) finds the row in the call's own slot's
// entry and recovers it: the prepared receipt is published once, the budget
// settled once, and nothing else is signed, sent or issued.
func TestControlRoom_FoundIsRecovered(t *testing.T) {
	for _, c := range controlRoomCases() {
		for _, road := range eaRoads {
			for _, sink := range []bool{false, true} {
				t.Run(fmt.Sprintf("%s/%s/sink=%v", c.name, road.name, sink), func(t *testing.T) {
					u, rs, releaseFiller, done := roomRig(t, c, true, road, sink)
					defer done()
					r := c.act(u.gapsRig)
					assertUnknownAnswer(t, u, r)
					assertKeptInOwnSlot(t, u, c)
					u.up()
					unknownEventually(t, "the row is recovered by the retry worker", func() bool {
						receipts, completed, _ := u.published()
						return receipts == 1 && completed == 1
					})
					unknownEventually(t, "the entry is gone", func() bool { return u.kept() == 0 })
					u.proxy.FlushPending(context.Background())
					u.proxy.FlushPending(context.Background())
					receipts, completed, cancelled := u.published()
					if receipts != 1 || completed != 1 || cancelled != 0 || len(u.decisions()) != 0 || u.misses() != 0 || u.out(r) != 0 {
						t.Fatalf("published %d completed %d cancelled %d signed %v misses %d out %d", receipts, completed, cancelled, u.base.signed(), u.misses(), u.out(r))
					}
					// Recovery settles through the journal's budget, not the
					// call's own commit: once, and the hold is not given back.
					u.b.mu.Lock()
					settled, parent, child := len(u.b.reconciled), u.b.parent, u.b.child
					u.b.mu.Unlock()
					if _, holds, commits, releases := u.store.counts(); holds != 1 || commits != 0 || releases != 0 || settled != 1 {
						t.Fatalf("budget: holds %d commits %d releases %d settled %d, want settled once", holds, commits, releases, settled)
					}
					if c.method == MethodSatGateDelegate && (parent != 95 || child != 5) {
						t.Fatalf("the carve was not charged and funded once: parent %d child %d", parent, child)
					}
					if rs != nil && rs.sunkCount() != 0 {
						t.Fatalf("the kept unknown was handed to the uncharged recorder %d times", rs.sunkCount())
					}
					assertRoomBack(t, u, releaseFiller)
				})
			}
		}
	}
}

// A kept unknown stays kept, unrecorded and unreleased, for as long as the
// journal cannot answer, in the uncharged kind as in the gap kind, and the
// store never counts it twice.
func TestControlRoom_StaysKeptWhileTheJournalIsDown(t *testing.T) {
	for _, c := range controlRoomCases() {
		for _, sink := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/sink=%v", c.name, sink), func(t *testing.T) {
				u, _, releaseFiller, done := roomRig(t, c, false, eaRoad{name: "direct"}, sink)
				defer done()
				r := c.act(u.gapsRig)
				assertUnknownAnswer(t, u, r)
				for i := 0; i < 3; i++ {
					u.proxy.FlushPending(context.Background())
					assertKeptInOwnSlot(t, u, c)
				}
				if len(u.decisions()) != 0 || u.misses() != 0 {
					t.Fatalf("recorded while the journal could not answer: %v", u.base.signed())
				}
				if _, _, _, releases := u.store.counts(); releases != 0 {
					t.Fatal("released the hold while the journal could not answer")
				}
				u.up()
				u.proxy.FlushPending(context.Background())
				unknownEventually(t, "the stop is signed", func() bool { return len(u.decisions()) == 1 })
				unknownEventually(t, "the entry is gone", func() bool { return u.kept() == 0 })
				assertRoomBack(t, u, releaseFiller)
			})
		}
	}
}

// controlRoomOffRows are the rows TestNotStored_SwitchOffBytes appends for a
// Control unknown outcome with the gap kind of the pending store full: with
// either switch off nothing is kept and every answer and effect is main's.
// Rows appended after the earlier ones; earlier rows unchanged.
func controlRoomOffRows(t *testing.T) []gapsSnapshot {
	var got []gapsSnapshot
	for _, c := range controlRoomCases() {
		for _, landed := range []bool{false, true} {
			for _, sw := range []struct {
				name           string
				sign, refusals bool
			}{{"both-off", false, false}, {"sign-only", true, false}, {"refusals-only", false, true}} {
				u, done := newUnknownRig(t, c, landed, sw.sign, sw.refusals, eaRoad{name: "direct"}, func(t *testing.T, g *gapsRig) { g.fill(t, kindGap) })
				before := u.remaining()
				r := c.act(u.gapsRig)
				done()
				kept := u.kept()
				u.up()
				u.proxy.FlushPending(context.Background())
				u.b.mu.Lock()
				rows := len(u.b.rows)
				u.b.mu.Unlock()
				got = append(got, gapsSnapshot{Case: fmt.Sprintf("full-gap/%s/landed=%v/kept=%d", c.name, landed, kept), Switches: sw.name, Response: eaShape(r),
					Signed: strings.Join(u.base.signed(), ","), Out: u.out(r), Debit: before - u.remaining(),
					Misses: u.misses(), Journaled: rows})
			}
		}
	}
	return got
}

// The released checkers accept the stops of kept Control calls whose gap kind
// was full, pack by pack and as one whole record.
func TestControlRoom_PublicCheckersAcceptThem(t *testing.T) {
	unknownCheckersAccept(t, true)
}
