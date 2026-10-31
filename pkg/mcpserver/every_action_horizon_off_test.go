package mcpserver

// Rows and fixtures shared by the switches-off snapshot (TestNotStored_
// SwitchOffBytes) and the tests of nothing aging out, the not-written failure and the kept call (every_action_horizon_test.go). This
// file uses no production symbol that the change adds (the sentinel comes from
// fixtureNotWritten, every_action_not_written_fixture_test.go), so the rows
// can be generated on main, where they are the baseline.

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// notWrittenJournal is a journal whose Save fails before it writes anything,
// as the enterprise store's does for a closed pool or a transaction that
// cannot start. kind is what Save returns: "marked" (the store's new
// not-written failure, fixtureNotWritten), "plain" (the same failure as the
// store returned it before it was marked: an unmarked error) or "notstored"
// (the definite ErrPendingNotStored a store already returned for a written
// cancellation or a reached bound). While down is set ResolveSave, List and
// Process fail too, as on a closed pool; resolveUp makes ResolveSave answer
// (the journal holds nothing, so its cancellation is written and it says
// absent) even then.
type notWrittenJournal struct {
	p2Store
	down      *atomic.Bool
	kind      string
	resolveUp bool
}

func (s notWrittenJournal) Save(context.Context, PendingWork) error {
	err := errors.New("fixture: journal pool closed")
	switch s.kind {
	case "marked":
		return fixtureNotWritten(err)
	case "notstored":
		return fmt.Errorf("fixture: %w", ErrPendingNotStored)
	}
	return err
}

func (s notWrittenJournal) ResolveSave(c context.Context, w PendingWork) (bool, error) {
	if s.down.Load() && !s.resolveUp {
		return false, errors.New("fixture: journal unreachable")
	}
	return s.p2Store.ResolveSave(c, w)
}

func (s notWrittenJournal) List(c context.Context) ([]PendingWork, error) {
	if s.down.Load() {
		return nil, errors.New("fixture: journal unreachable")
	}
	return s.p2Store.List(c)
}

func (s notWrittenJournal) Process(c context.Context, id string, fn func(context.Context, PendingWork) (PendingDisposition, error)) (bool, error) {
	if s.down.Load() {
		return false, errors.New("fixture: journal unreachable")
	}
	return s.p2Store.Process(c, id, fn)
}

func newNotWrittenRig(t *testing.T, c unknownCase, kind string, resolveUp, sign, refusals bool, road eaRoad) (*unknownRig, func()) {
	t.Helper()
	return newJournalRig(t, c, func(g *gapsRig, down *atomic.Bool) PendingStore {
		return notWrittenJournal{p2Store{g.b}, down, kind, resolveUp}
	}, sign, refusals, road, nil)
}

// ageJournalRows moves the original minute of every open journal row back by d,
// as the passage of time would. Nothing else of the row changes.
func ageJournalRows(t *testing.T, u *unknownRig, d time.Duration) {
	t.Helper()
	u.b.mu.Lock()
	defer u.b.mu.Unlock()
	for id, raw := range u.b.rows {
		var w PendingWork
		if err := json.Unmarshal(raw, &w); err != nil {
			t.Fatal(err)
		}
		w.Record.Minute = w.Record.Minute.Add(-d)
		if w.Window != nil {
			w.Window.WindowStart = w.Window.WindowStart.Add(-d)
		}
		aged, err := json.Marshal(w)
		if err != nil {
			t.Fatal(err)
		}
		u.b.rows[id] = aged
	}
}

// dispositions is how the journal completed its rows, for a snapshot name.
func (u *unknownRig) dispositions() string {
	u.b.mu.Lock()
	defer u.b.mu.Unlock()
	var out []string
	for _, d := range u.b.completed {
		out = append(out, string(d))
	}
	if len(out) == 0 {
		return "open"
	}
	return strings.Join(out, ",")
}

var offSwitches = []struct {
	name           string
	sign, refusals bool
}{{"both-off", false, false}, {"sign-only", true, false}, {"refusals-only", false, true}}

// notWrittenOffRows: with either switch off, a journal whose Save fails with
// the store's not-written failure is main's: the save is ambiguous (ResolveSave
// first), so the answer, signing, release, count and journaling are exactly
// those the unmarked failure had on main. ResolveSave down: outcome_unknown;
// ResolveSave answering: absent, as main's.
func notWrittenOffRows(t *testing.T) []gapsSnapshot {
	var got []gapsSnapshot
	for _, c := range unknownCases() {
		for _, resolveUp := range []bool{false, true} {
			for _, sw := range offSwitches {
				u, done := newNotWrittenRig(t, c, "marked", resolveUp, sw.sign, sw.refusals, eaRoad{name: "direct"})
				before := u.remaining()
				r := c.act(u.gapsRig)
				done()
				kept := u.kept()
				u.up()
				u.proxy.FlushPending(context.Background())
				u.b.mu.Lock()
				rows := len(u.b.rows)
				u.b.mu.Unlock()
				got = append(got, gapsSnapshot{Case: fmt.Sprintf("%s/not-written/resolve-up=%v/kept=%d", c.name, resolveUp, kept), Switches: sw.name, Response: eaShape(r),
					Signed: strings.Join(u.base.signed(), ","), Out: u.out(r), Debit: before - u.remaining(),
					Misses: u.misses(), Journaled: rows})
			}
		}
	}
	return got
}

// horizonOffRows: with either switch off, a journal row that was stored, never
// published or counted, and is 49 hours old when the journal answers again is
// aged out exactly as on main: nothing is published, counted or settled.
func horizonOffRows(t *testing.T) []gapsSnapshot {
	var got []gapsSnapshot
	for _, c := range unknownCases() {
		for _, sw := range offSwitches {
			u, done := newUnknownRig(t, c, true, sw.sign, sw.refusals, eaRoad{name: "direct"}, nil)
			before := u.remaining()
			r := c.act(u.gapsRig)
			done()
			kept := u.kept()
			ageJournalRows(t, u, 49*time.Hour)
			u.up()
			u.proxy.FlushPending(context.Background())
			u.proxy.FlushPending(context.Background())
			u.b.mu.Lock()
			rows, receipts := len(u.b.rows), len(u.b.receipts)
			u.b.mu.Unlock()
			got = append(got, gapsSnapshot{Case: fmt.Sprintf("%s/aged-49h/kept=%d/receipts=%d/disposition=%s", c.name, kept, receipts, u.dispositions()), Switches: sw.name, Response: eaShape(r),
				Signed: strings.Join(u.base.signed(), ","), Out: u.out(r), Debit: before - u.remaining(),
				Misses: u.misses(), Journaled: rows})
		}
	}
	return got
}
