package mcpserver

import (
	"context"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// Round 3 of the kept-tools fix (1012c): no list that is served outlives
// KeptToolsMaxAge, and no list is built from a read that a sign-in write made
// stale.

// failNames is a name store whose Load can be made to fail.
type failNames struct {
	ToolNameStore
	fail atomic.Bool
}

func (s *failNames) Load(ctx context.Context) ([]ToolNameAssignment, error) {
	if s.fail.Load() {
		return nil, errString("synthetic names store unavailable")
	}
	return s.ToolNameStore.Load(ctx)
}

func r3Setup(t *testing.T, clk *clockAt) (*UpstreamManager, *hostRead, *failNames, *fakeService, *fakeService) {
	t.Helper()
	rh, alp := twoBrokers(t)
	h := &hostRead{}
	h.set(nil, claimSvc("robinhood", "Robinhood", rh, false), claimSvc("alpaca", "Alpaca", alp, false))
	store := &failNames{ToolNameStore: newSeenStore()}
	m, _ := startCfg(t, func(m *UpstreamManager) {
		m.SetToolNameStore(store)
		if clk != nil {
			m.SetClock(clk.now)
		}
		m.SetClaimOrderSource(h.get)
	}, nil, map[string]UpstreamConfig{"robinhood": rh.cfg("Robinhood"), "alpaca": alp.cfg("Alpaca")})
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("setup: %d of 9", n)
	}
	return m, h, store, rh, alp
}

// R2-B1 (a), at the manager: a view built from a complete read is served for
// KeptToolsMaxAge and no longer, even when nothing tells the manager to build
// again (the host's read is blocked before it can say it failed).
func TestChecked_ACompleteViewIsNotServedPastTheLimitEvenWhenNothingRebuildsIt(t *testing.T) {
	clk := newClock()
	m, h, _, _, _ := r3Setup(t, clk)
	h.down() // no Recheck, no SetCredentialsUnreadable: the view is simply old
	clk.advance(KeptToolsMaxAge - time.Second)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("at the limit minus 1 s: %d of 9", n)
	}
	clk.advance(2 * time.Second)
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("at the limit plus 1 s the cached complete view still lists %d tools", n)
	}
	if o := m.ToolOrigin("get_account"); o.Listed {
		t.Fatal("an expired view still routes a tool")
	}
	h.set(nil)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("the first list after a complete read: %d of 9", n)
	}
}

// The same view with the host working: past the limit the next list reads
// again and shows the tools (rebuilt, not just dropped).
func TestChecked_ACompleteViewPastTheLimitIsRebuiltFromANewRead(t *testing.T) {
	clk := newClock()
	m, _, _, _, _ := r3Setup(t, clk)
	clk.advance(KeptToolsMaxAge + time.Minute)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("with a working read past the limit: %d of 9", n)
	}
}

// R2-B1 (b): the fallback when the stored names cannot be read.
func TestChecked_NameStoreFailureCannotOutliveTheKeptLimit(t *testing.T) {
	clk := newClock()
	m, h, store, _, _ := r3Setup(t, clk)
	h.down()
	store.fail.Store(true)
	m.Recheck()
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("inside the limit the last list is kept: %d of 9", n)
	}
	clk.advance(KeptToolsMaxAge + time.Second)
	m.Recheck()
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("expired list survived a names-store failure: %d tools, want 0", n)
	}
	// A failed read never extends it, and the store coming back does not
	// bring an expired list back without a complete read.
	store.fail.Store(false)
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("store back, read still failing, past the limit: %d tools, want 0", n)
	}
	h.set(nil)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("after a complete read: %d of 9", n)
	}
}

func TestChecked_NameStoreFailureCannotUndoAForget(t *testing.T) {
	m, h, store, _, _ := r3Setup(t, nil)
	h.down()
	store.fail.Store(true)
	m.checked.Forget()
	m.Recheck()
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("a forgotten list survived a names-store failure: %d tools, want 0", n)
	}
}

// A cached complete view ends when the record is forgotten, with nothing else
// telling the manager to build again.
func TestChecked_ACachedViewEndsWhenTheRecordIsForgotten(t *testing.T) {
	m, h, _, _, _ := r3Setup(t, nil)
	h.down()
	m.checked.Forget() // InvalidateTenant: no Recheck here
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("after Forget with the read failing the old view still lists %d tools", n)
	}
	h.set(nil)
	if n := len(m.AllTools()); n != 9 {
		t.Fatalf("after a complete read: %d of 9", n)
	}
	// ForgetSignIns (a sign-in written on this machine) ends it the same way.
	h.down()
	m.checked.ForgetSignIns()
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("after ForgetSignIns with the read failing the old view still lists %d tools", n)
	}
}

// R2-B2: a read made stale while the view is built is never published. A
// forget that lands after the host's read, on every attempt, holds everything.
func TestChecked_AReadInvalidatedOnEveryAttemptPublishesNothing(t *testing.T) {
	m, h, _, _, _ := r3Setup(t, nil)
	rec := m.checked
	var reads atomic.Int32
	m.SetClaimOrderSource(func(ctx context.Context) (ClaimOrder, error) {
		epoch := rec.Epoch()
		co, err := h.get(ctx)
		co.ReadEpoch, co.ReadAt = epoch, time.Now()
		reads.Add(1)
		rec.Forget() // a sign-in committed right after the read
		return co, err
	})
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("a view built from a read made stale %d times published %d tools", reads.Load(), n)
	}
	if got := reads.Load(); got < 2 || got > viewBuildAttempts {
		t.Fatalf("the build was tried %d times, want 2 to %d", got, viewBuildAttempts)
	}
}

// A sign-in committed after the read and before the view is returned: the
// first build is dropped, the second uses the new values, and the tool that
// holds the new value is not listed (the rest are).
func TestChecked_AWriteDuringTheViewsReadIsRebuiltWithTheNewValues(t *testing.T) {
	m, h, _, _, _ := r3Setup(t, nil)
	rec := m.checked
	var reads atomic.Int32
	const newValue = "Get the trading account."
	m.SetClaimOrderSource(func(ctx context.Context) (ClaimOrder, error) {
		epoch := rec.Epoch()
		co, err := h.get(ctx) // the values as they were when read
		co.ReadEpoch, co.ReadAt = epoch, time.Now()
		if reads.Add(1) == 1 {
			// The write lands now: the host stores the new value and tells
			// the record. This read still carries the old values.
			h.withSecrets("robinhood", newValue)
			rec.ForgetSignIns()
		}
		return co, err
	})
	m.Recheck()
	got := listTools(t, m)
	for name, tl := range got {
		if strings.Contains(tl.Description, "trading account") {
			t.Fatalf("a view built from the read before the write listed %q, which holds the new value", name)
		}
	}
	if len(got) != 8 {
		t.Fatalf("want the other 8 tools from the rebuilt view: %v", names(got))
	}
	if reads.Load() != 2 {
		t.Fatalf("the view read %d times, want 2 (one stale, one rebuilt)", reads.Load())
	}
}

// The ignored-result guard: a build whose generation moved is not stored as
// the current view, so a later list does not see it either.
func TestChecked_AStaleBuildIsNotKeptAsTheCurrentView(t *testing.T) {
	m, h, _, _, _ := r3Setup(t, nil)
	rec := m.checked
	m.SetClaimOrderSource(func(ctx context.Context) (ClaimOrder, error) {
		epoch := rec.Epoch()
		co, err := h.get(ctx)
		co.ReadEpoch, co.ReadAt = epoch, time.Now()
		rec.Forget()
		return co, err
	})
	_ = m.AllTools()
	if v := m.viewPtr.Load(); v == nil || len(v.tools) != 0 || v.stale {
		t.Fatalf("the stored view must be the held view, got %+v", v)
	}
}
