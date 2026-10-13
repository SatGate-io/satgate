package mcpserver

import (
	"context"
	"encoding/json"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Round 4 of the kept-tools fix (1012d): one time budget per list. It covers
// the wait for the view lock and every build, so the waits cannot stack, and
// a list queued behind a build that gave up shares that result.

// setListBudgetForTest shortens the budget of a list and returns what
// restores it. The tests that use it do not run in parallel.
func setListBudgetForTest(d time.Duration) func() {
	prev := listBudget
	listBudget = d
	return func() { listBudget = prev }
}

func descriptions(t *testing.T, m *UpstreamManager) []string {
	t.Helper()
	var out []string
	for _, raw := range m.AllTools() {
		var tl listedTool
		if err := json.Unmarshal(raw, &tl); err != nil {
			t.Fatal(err)
		}
		out = append(out, tl.Description)
	}
	return out
}

// 1a: a list that queues for the view lock gives up at its budget, whoever
// holds the lock and however long it holds it.
func TestBudget_AQueuedListDoesNotWaitForTheViewLockPastItsBudget(t *testing.T) {
	defer setListBudgetForTest(300 * time.Millisecond)()
	const hold = 2500 * time.Millisecond

	t.Run("no usable view: the held result, no tools", func(t *testing.T) {
		m, _, _, _, _ := r3Setup(t, nil)
		m.viewMu.Lock() // a build that takes its time
		timer := time.AfterFunc(hold, m.viewMu.Unlock)
		defer timer.Stop()
		m.Recheck() // the cached view is out of date
		start := time.Now()
		n := len(m.AllTools())
		el := time.Since(start)
		if n != 0 {
			t.Fatalf("the list behind a build listed %d tools; with no usable view it must answer with none", n)
		}
		if el < 250*time.Millisecond || el > 900*time.Millisecond {
			t.Fatalf("the list waited %s for the view lock, want its budget (300ms) and not the %s the lock was held", el, hold)
		}
	})

	t.Run("a complete valid view published while it waited is used", func(t *testing.T) {
		m, _, _, _, _ := r3Setup(t, nil)
		m.viewMu.Lock()
		timer := time.AfterFunc(hold, m.viewMu.Unlock)
		defer timer.Stop()
		old := *m.viewPtr.Load()
		m.Recheck()
		// The build that holds the lock publishes its result at 100ms (the
		// lock itself stays held).
		time.AfterFunc(100*time.Millisecond, func() {
			cur := old
			cur.ver = m.namesVer.Load()
			m.viewPtr.Store(&cur)
		})
		start := time.Now()
		n := len(m.AllTools())
		el := time.Since(start)
		if n != 9 {
			t.Fatalf("a complete valid view was published while the list waited, it listed %d of 9", n)
		}
		if el > 900*time.Millisecond {
			t.Fatalf("the list waited %s, want its budget (300ms)", el)
		}
	})

	t.Run("the lock is not lost: a later list builds as usual", func(t *testing.T) {
		m, _, _, _, _ := r3Setup(t, nil)
		m.viewMu.Lock()
		m.Recheck()
		done := make(chan int, 1)
		go func() { done <- len(m.AllTools()) }()
		if n := <-done; n != 0 {
			t.Fatalf("while held: %d tools", n)
		}
		m.viewMu.Unlock()
		// The helper that waited for the lock for the list that gave up has
		// taken it and let it go; a new list builds.
		deadline := time.Now().Add(time.Second)
		for {
			if n := len(m.AllTools()); n == 9 {
				break
			}
			if time.Now().After(deadline) {
				t.Fatal("after the lock was released no list could build the view")
			}
			time.Sleep(10 * time.Millisecond)
		}
	})
}

// 1c: the builds of one round of stale reads end in a held view stamped with
// the version read before them, although the writes that made them stale
// moved namesVer. Lists queued behind that build share it.
func TestBudget_AQueuedListSharesTheHeldResultAfterNamesVerMoved(t *testing.T) {
	m, h, _, _, _ := r3Setup(t, nil)
	rec := m.checked
	var reads atomic.Int32
	m.SetClaimOrderSource(func(ctx context.Context) (ClaimOrder, error) {
		epoch := rec.Epoch()
		co, err := h.get(ctx)
		co.ReadEpoch, co.ReadAt = epoch, time.Now()
		reads.Add(1)
		time.Sleep(60 * time.Millisecond)
		// A sign-in committed on this machine: the record forgets what was
		// read and the version moves (OnSignInWritten does both).
		rec.ForgetSignIns()
		m.Recheck()
		return co, err
	})
	reads.Store(0)

	const lists = 6
	var wg sync.WaitGroup
	start := make(chan struct{})
	counts := make([]int, lists)
	els := make([]time.Duration, lists)
	for i := 0; i < lists; i++ {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			<-start
			began := time.Now()
			counts[i] = len(m.AllTools())
			els[i] = time.Since(began)
		}(i)
	}
	close(start)
	wg.Wait()
	for i := range counts {
		if counts[i] != 0 {
			t.Fatalf("list %d listed %d tools from reads that were all stale", i, counts[i])
		}
		if els[i] > 2*time.Second {
			t.Fatalf("list %d took %s", i, els[i])
		}
	}
	if got := reads.Load(); got > viewBuildAttempts {
		t.Fatalf("%d lists made %d reads; lists queued behind the build must share its held result (at most %d)", lists, got, viewBuildAttempts)
	}
	v := m.viewPtr.Load()
	if v == nil || !v.held || len(v.tools) != 0 {
		t.Fatalf("the published view is not the held view: %+v", v)
	}

	// A list that arrives after the held result was published builds again.
	before := reads.Load()
	if n := len(m.AllTools()); n != 0 {
		t.Fatalf("the late list listed %d tools", n)
	}
	if reads.Load() <= before {
		t.Fatal("a list that arrived after the held result was published did not start its own build")
	}
}

// 1c: a view with tools and an old version is never handed to a list that
// queued behind it.
func TestBudget_AQueuedListNeverGetsAnOutOfDateNonemptyView(t *testing.T) {
	m, h, _, _, _ := r3Setup(t, nil)
	var reads atomic.Int32
	inRead := make(chan struct{})
	release := make(chan struct{})
	const newValue = "Get the trading account."
	m.SetClaimOrderSource(func(ctx context.Context) (ClaimOrder, error) {
		co, err := h.get(ctx) // the values as they are now
		co.ReadEpoch, co.ReadAt = m.checked.Epoch(), time.Now()
		if reads.Add(1) == 1 {
			close(inRead)
			<-release
			// The account changes after this read, and the manager is told to
			// look again (not a sign-in write: the read stays current).
			h.withSecrets("robinhood", newValue)
			m.Recheck()
		}
		return co, err
	})
	reads.Store(0)
	m.Recheck()

	var a, b []string
	var wg sync.WaitGroup
	wg.Add(2)
	go func() { defer wg.Done(); a = descriptions(t, m) }()
	<-inRead
	go func() { defer wg.Done(); b = descriptions(t, m) }()
	time.Sleep(150 * time.Millisecond) // b is queued behind a's build
	close(release)
	wg.Wait()

	if len(a) != 9 {
		t.Fatalf("the list that built the view listed %d of 9", len(a))
	}
	if reads.Load() != 2 {
		t.Fatalf("the view was read %d times, want 2: the queued list must build again, not share a view whose version is out of date", reads.Load())
	}
	for _, d := range b {
		if strings.Contains(d, "trading account") {
			t.Fatalf("the queued list got a tool that holds the new sign-in value: %q", d)
		}
	}
	if len(b) != 8 {
		t.Fatalf("the queued list listed %d tools, want the 8 of the rebuilt view", len(b))
	}
}

// 1a, 1b: the budget ends the first build; three builds are a cap, not three
// allowances; no build starts once the budget is used up.
func TestBudget_ThreeBuildsAreACapNotThreeAllowances(t *testing.T) {
	defer setListBudgetForTest(400 * time.Millisecond)()

	t.Run("the first build ends with the budget", func(t *testing.T) {
		m, h, _, _, _ := r3Setup(t, nil)
		var reads atomic.Int32
		m.SetClaimOrderSource(func(ctx context.Context) (ClaimOrder, error) {
			reads.Add(1)
			<-ctx.Done() // a read that never answers
			co, _ := h.get(ctx)
			co.AllUnreadable = true
			return co, ctx.Err()
		})
		m.Recheck()
		start := time.Now()
		_ = m.AllTools()
		if el := time.Since(start); el > 900*time.Millisecond {
			t.Fatalf("the first build ran %s on a list with a 400ms budget", el)
		}
	})

	t.Run("builds that are made stale share the budget", func(t *testing.T) {
		m, h, _, _, _ := r3Setup(t, nil)
		rec := m.checked
		var reads atomic.Int32
		m.SetClaimOrderSource(func(ctx context.Context) (ClaimOrder, error) {
			epoch := rec.Epoch()
			co, err := h.get(ctx)
			co.ReadEpoch, co.ReadAt = epoch, time.Now()
			reads.Add(1)
			select {
			case <-time.After(300 * time.Millisecond):
			case <-ctx.Done():
			}
			rec.ForgetSignIns()
			return co, err
		})
		m.Recheck()
		start := time.Now()
		n := len(m.AllTools())
		el := time.Since(start)
		if n != 0 {
			t.Fatalf("listed %d tools from stale reads", n)
		}
		if el > 800*time.Millisecond {
			t.Fatalf("builds ran %s on a list with a 400ms budget: each build got a fresh allowance", el)
		}
	})

	t.Run("no build starts once the budget is used up", func(t *testing.T) {
		m, h, _, _, _ := r3Setup(t, nil)
		rec := m.checked
		var reads atomic.Int32
		m.SetClaimOrderSource(func(ctx context.Context) (ClaimOrder, error) {
			epoch := rec.Epoch()
			co, err := h.get(ctx)
			co.ReadEpoch, co.ReadAt = epoch, time.Now()
			reads.Add(1)
			<-ctx.Done() // the read uses all of the budget
			rec.ForgetSignIns()
			return co, err
		})
		m.Recheck()
		_ = m.AllTools()
		if got := reads.Load(); got != 1 {
			t.Fatalf("%d builds started; the first used the whole budget, so no other may start", got)
		}
	})
}

// Item 4: a stalled name store is bounded by the list's budget too (the
// build's context is the budget's).
func TestBudget_AStalledNameStoreIsBoundedByTheListBudget(t *testing.T) {
	defer setListBudgetForTest(400 * time.Millisecond)()
	m, _, store, _, _ := r3Setup(t, nil)
	store.stall.Store(true)
	m.Recheck()
	start := time.Now()
	n := len(m.AllTools())
	el := time.Since(start)
	if el > 900*time.Millisecond {
		t.Fatalf("a list behind a stalled name store took %s, want about its 400ms budget", el)
	}
	if n != 9 && n != 0 {
		t.Fatalf("listed %d tools; the last valid view (9) or nothing (0)", n)
	}
	// Calls are not given the list's budget (they wait for the store as on
	// main, up to a build's 10 s); only that they are not cut short is
	// checked here, with a store that answers after 700ms.
	store.stall.Store(false)
	store.delay.Store(int64(700 * time.Millisecond))
	m.Recheck()
	start = time.Now()
	if o := m.ToolOrigin("get_account"); !o.Listed {
		t.Fatalf("a call's view was cut short by the list budget: %+v", o)
	}
	if el := time.Since(start); el < 650*time.Millisecond {
		t.Fatalf("the call's build did not wait for the store (%s)", el)
	}
}

// 1e: calls (the toolView paths) keep their behavior: they wait for the lock
// and for the build as long as that takes, and are not cut at the budget.
func TestBudget_CallsAreNotGivenTheListBudget(t *testing.T) {
	defer setListBudgetForTest(100 * time.Millisecond)()
	m, h, _, _, _ := r3Setup(t, nil)

	// The lock: a call waits for it.
	m.viewMu.Lock()
	time.AfterFunc(450*time.Millisecond, m.viewMu.Unlock)
	m.Recheck()
	start := time.Now()
	o := m.ToolOrigin("get_account")
	if el := time.Since(start); el < 400*time.Millisecond {
		t.Fatalf("a call did not wait for the view lock: %s", el)
	}
	if !o.Listed {
		t.Fatalf("a call after the wait: %+v", o)
	}

	// The build: a read that takes 400ms is not cut at 100ms.
	m.SetClaimOrderSource(func(ctx context.Context) (ClaimOrder, error) {
		select {
		case <-time.After(400 * time.Millisecond):
		case <-ctx.Done():
			t.Error("a call's build was cut short")
		}
		return h.get(ctx)
	})
	start = time.Now()
	if o := m.ToolOrigin("get_account"); !o.Listed {
		t.Fatalf("a call whose build takes longer than the list budget: %+v", o)
	}
	if el := time.Since(start); el < 350*time.Millisecond {
		t.Fatalf("the call did not wait for its build: %s", el)
	}
}

// Item 3: the record's own expiry, with nothing else in the way. The view
// checks it a second time (valid), which hid this layer from the other tests.
func TestChecked_TheRecordOwnsItsExpiry(t *testing.T) {
	rec := NewToolCheckRecord()
	t0 := time.Now()
	if !rec.NoteCompleteRead(rec.Epoch(), t0, []string{"synthetic-access-value", "synthetic-refresh-value"}) {
		t.Fatal("a read in the current generation was refused")
	}
	vals, until, ok := rec.kept(t0.Add(KeptToolsMaxAge - time.Second))
	if !ok || len(vals) != 2 || !until.Equal(t0.Add(KeptToolsMaxAge)) {
		t.Fatalf("one second inside the limit: ok=%v vals=%d until=%s", ok, len(vals), until)
	}
	vals, until, ok = rec.kept(t0.Add(KeptToolsMaxAge))
	if ok || len(vals) != 0 {
		t.Fatalf("at the limit the record still keeps %d values (ok=%v)", len(vals), ok)
	}
	if !until.Equal(t0.Add(KeptToolsMaxAge)) {
		t.Fatalf("until=%s", until)
	}
	if _, _, ok = rec.kept(t0.Add(KeptToolsMaxAge + time.Hour)); ok {
		t.Fatal("an hour past the limit the record still keeps values")
	}
	// And a record with no complete read keeps nothing.
	if _, _, ok = NewToolCheckRecord().kept(t0); ok {
		t.Fatal("a record with no complete read keeps values")
	}
}
