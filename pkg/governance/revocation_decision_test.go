package governance

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
)

// These tests check that an allow decision with a shared store rests on one
// fresh answer for every lineage entry at the moment it allows, whether the
// answer was cached or just read, and that a ban made on this gateway while
// the store was being read refuses. They use only API that existed before
// that rule (IsAnyBanned, BanContext, the cache fields), so they also run
// against the earlier code.

// testClock is an injected clock.
type testClock struct {
	mu   sync.Mutex
	base time.Time
	now  time.Time
}

func newTestClock() *testClock {
	t := time.Now()
	return &testClock{base: t, now: t}
}

func (c *testClock) Now() time.Time { c.mu.Lock(); defer c.mu.Unlock(); return c.now }

// set moves the clock to base+d.
func (c *testClock) set(d time.Duration) { c.mu.Lock(); c.now = c.base.Add(d); c.mu.Unlock() }

func (c *testClock) at(d time.Duration) time.Time { return c.base.Add(d) }

// scriptedStore answers BannedAmong from the real store, then runs onRead
// (if set) with the read's number (1, 2, ...) and the signatures read; an
// error from onRead is returned instead of the answer.
type scriptedStore struct {
	*RedisStore
	mu     sync.Mutex
	reads  [][]string
	onRead func(n int, sigs []string) error
}

func (s *scriptedStore) BannedAmong(ctx context.Context, sigs []string) ([]bool, error) {
	out := make([]bool, len(sigs))
	for i, sig := range sigs {
		b, err := s.RedisStore.IsBanned(ctx, sig)
		if err != nil {
			return nil, err
		}
		out[i] = b
	}
	s.mu.Lock()
	s.reads = append(s.reads, append([]string(nil), sigs...))
	n := len(s.reads)
	onRead := s.onRead
	s.mu.Unlock()
	if onRead != nil {
		if err := onRead(n, sigs); err != nil {
			return nil, err
		}
	}
	return out, nil
}

func (s *scriptedStore) readLog() [][]string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([][]string(nil), s.reads...)
}

// decisionFixture is a gateway (svc, injected clock, scripted store) and
// another gateway (other) on the same Redis.
type decisionFixture struct {
	svc   *Service
	other *Service
	store *scriptedStore
	clock *testClock
}

func newDecisionFixture(t *testing.T) *decisionFixture {
	t.Helper()
	mr := miniredis.RunT(t)
	st := &scriptedStore{RedisStore: newMiniRedisStore(t, mr.Addr())}
	svc := NewService(st)
	clk := newTestClock()
	svc.now = clk.Now
	return &decisionFixture{svc: svc, other: newRedisService(t, mr.Addr()), store: st, clock: clk}
}

// cache records a "not banned" answer for sig from a lookup that started at
// base+d.
func (f *decisionFixture) cache(sig string, d time.Duration) {
	f.svc.cacheMu.Lock()
	f.svc.putCacheLocked(sig, false, f.clock.at(d))
	f.svc.cacheMu.Unlock()
}

func (f *decisionFixture) banElsewhere(t *testing.T, sig string) {
	t.Helper()
	if err := f.other.BanContext(context.Background(), sig, "test", "admin"); err != nil {
		t.Fatal(err)
	}
}

func lineageOf(n int) []string {
	out := make([]string, n)
	for i := range out {
		out[i] = fmt.Sprintf("sig-depth-%d", i)
	}
	return out
}

const lineageLen = 4 // root, two delegated tokens, the presented token

// The reviewers' first case: an ancestor answer that was fresh when the
// decision started (1.9s old), a read of another entry that took until the
// ancestor answer was 3.9s old, and a ban of the ancestor made elsewhere
// meanwhile. The decision must not allow on the 3.9s-old answer.
func TestDecision_CachedAncestorThatAgedDuringTheReadIsNotUsedToAllow(t *testing.T) {
	f := newDecisionFixture(t)
	f.cache("ancestor", 0)
	f.clock.set(1900 * time.Millisecond)
	f.store.onRead = func(n int, _ []string) error {
		if n == 1 {
			f.banElsewhere(t, "ancestor")
			f.clock.set(3900*time.Millisecond - time.Nanosecond)
		}
		return nil
	}
	hit, err := f.svc.IsAnyBanned(context.Background(), []string{"ancestor", "child"})
	if err != nil || !hit {
		t.Fatalf("allowed on an ancestor answer %s old: banned=%v err=%v", 3900*time.Millisecond, hit, err)
	}
}

// The reviewers' second case: a ban of an ancestor made on this gateway while
// the store was read for another entry refuses the request in flight.
func TestDecision_LocalAncestorBanDuringReadOfAnotherEntryRefuses(t *testing.T) {
	f := newDecisionFixture(t)
	f.svc.now = time.Now
	f.svc.cacheMu.Lock()
	f.svc.putCacheLocked("ancestor", false, time.Now())
	f.svc.cacheMu.Unlock()
	f.store.onRead = func(n int, _ []string) error {
		if n == 1 {
			if err := f.svc.BanContext(context.Background(), "ancestor", "test", "admin"); err != nil {
				t.Error(err)
			}
		}
		return nil
	}
	hit, err := f.svc.IsAnyBanned(context.Background(), []string{"ancestor", "child"})
	if err != nil || !hit {
		t.Fatalf("local ancestor ban made during the read not honoured: banned=%v err=%v", hit, err)
	}
}

// At every depth of the lineage: an entry whose cached answer was fresh at
// the start of the decision but aged past the TTL while another entry was
// read is read again, and a ban made elsewhere meanwhile is honoured.
func TestDecision_EachDepth_CachedEntryAgedDuringReadIsReadAgain(t *testing.T) {
	sigs := lineageOf(lineageLen)
	for k := range sigs {
		t.Run(fmt.Sprintf("banned_depth_%d", k), func(t *testing.T) {
			f := newDecisionFixture(t)
			miss := (k + 1) % lineageLen
			for i, sig := range sigs {
				if i != miss {
					f.cache(sig, 0)
				}
			}
			f.clock.set(1900 * time.Millisecond)
			f.store.onRead = func(n int, _ []string) error {
				if n == 1 {
					f.banElsewhere(t, sigs[k])
					f.clock.set(2500 * time.Millisecond) // the read took 0.6s
				}
				return nil
			}
			hit, err := f.svc.IsAnyBanned(context.Background(), sigs)
			if err != nil || !hit {
				t.Fatalf("depth %d banned elsewhere, cached answer 2.5s old at the decision: banned=%v err=%v reads=%v", k, hit, err, f.store.readLog())
			}
		})
	}
	// Control: nothing banned. Cached answers that aged are read again (in
	// one lookup of the whole lineage) and the request is allowed.
	t.Run("nothing_banned", func(t *testing.T) {
		f := newDecisionFixture(t)
		for _, sig := range sigs[:lineageLen-1] {
			f.cache(sig, 0)
		}
		f.clock.set(1900 * time.Millisecond)
		f.store.onRead = func(n int, _ []string) error {
			if n == 1 {
				f.clock.set(2500 * time.Millisecond)
			}
			return nil
		}
		hit, err := f.svc.IsAnyBanned(context.Background(), sigs)
		if err != nil || hit {
			t.Fatalf("nothing banned: banned=%v err=%v", hit, err)
		}
		reads := f.store.readLog()
		if len(reads) != 2 || len(reads[0]) != 1 || len(reads[1]) != lineageLen {
			t.Fatalf("want the missing entry read, then the whole lineage: %v", reads)
		}
	})
	// Control: cached answers still fresh when the read ends are used as
	// they are (no second read).
	t.Run("cached_still_fresh", func(t *testing.T) {
		f := newDecisionFixture(t)
		for _, sig := range sigs[:lineageLen-1] {
			f.cache(sig, 0)
		}
		f.clock.set(1500 * time.Millisecond)
		f.store.onRead = func(n int, _ []string) error {
			f.clock.set(1900 * time.Millisecond)
			return nil
		}
		hit, err := f.svc.IsAnyBanned(context.Background(), sigs)
		if err != nil || hit {
			t.Fatalf("nothing banned: banned=%v err=%v", hit, err)
		}
		if reads := f.store.readLog(); len(reads) != 1 || len(reads[0]) != 1 {
			t.Fatalf("want one read of the missing entry: %v", reads)
		}
	})
}

// At every depth: an entry whose cached answer had already expired when the
// decision started is read; the other entries, cached and fresh, are not.
func TestDecision_EachDepth_ExpiredEntryIsReadFreshEntriesAreNot(t *testing.T) {
	sigs := lineageOf(lineageLen)
	for k := range sigs {
		t.Run(fmt.Sprintf("expired_depth_%d", k), func(t *testing.T) {
			f := newDecisionFixture(t)
			for i, sig := range sigs {
				if i == k {
					f.cache(sig, 0) // 2.5s old at the decision
				} else {
					f.cache(sig, time.Second) // 1.5s old
				}
			}
			f.banElsewhere(t, sigs[k])
			f.clock.set(2500 * time.Millisecond)
			hit, err := f.svc.IsAnyBanned(context.Background(), sigs)
			if err != nil || !hit {
				t.Fatalf("expired entry at depth %d not read: banned=%v err=%v", k, hit, err)
			}
			if reads := f.store.readLog(); len(reads) != 1 || len(reads[0]) != 1 || reads[0][0] != sigs[k] {
				t.Fatalf("want one read of the expired entry only: %v", reads)
			}
		})
	}
}

// At every depth: with every answer cached and fresh, no read is made, and a
// ban made elsewhere is honoured as soon as the cached answer for that entry
// is BanCacheTTL old.
func TestDecision_EachDepth_FreshCacheUsedUntilTTLThenBanHonoured(t *testing.T) {
	sigs := lineageOf(lineageLen)
	for k := range sigs {
		t.Run(fmt.Sprintf("depth_%d", k), func(t *testing.T) {
			f := newDecisionFixture(t)
			for _, sig := range sigs {
				f.cache(sig, 0)
			}
			f.banElsewhere(t, sigs[k])
			f.clock.set(BanCacheTTL - time.Millisecond)
			if hit, err := f.svc.IsAnyBanned(context.Background(), sigs); err != nil || hit {
				t.Fatalf("within the TTL the cached answers are used: banned=%v err=%v", hit, err)
			}
			if reads := f.store.readLog(); len(reads) != 0 {
				t.Fatalf("no read expected with every answer fresh: %v", reads)
			}
			f.clock.set(BanCacheTTL)
			if hit, err := f.svc.IsAnyBanned(context.Background(), sigs); err != nil || !hit {
				t.Fatalf("ban at depth %d not honoured once the cached answer is %s old: banned=%v err=%v", k, BanCacheTTL, hit, err)
			}
		})
	}
}

// At every depth: a ban made on this gateway while the store is read for
// another entry refuses, even when the cache no longer holds it (evicted):
// the decision reads this gateway's ban set again before it allows.
func TestDecision_EachDepth_LocalBanDuringReadRefusesEvenIfEvicted(t *testing.T) {
	sigs := lineageOf(lineageLen)
	for k := range sigs {
		t.Run(fmt.Sprintf("depth_%d", k), func(t *testing.T) {
			f := newDecisionFixture(t)
			miss := (k + 1) % lineageLen
			for i, sig := range sigs {
				if i != miss {
					f.cache(sig, 0)
				}
			}
			f.clock.set(500 * time.Millisecond)
			f.store.onRead = func(n int, _ []string) error {
				if n == 1 {
					if err := f.svc.BanContext(context.Background(), sigs[k], "test", "admin"); err != nil {
						t.Error(err)
					}
					f.svc.cacheMu.Lock()
					f.svc.banCache = make(map[string]banCacheEntry)
					f.svc.cacheMu.Unlock()
				}
				return nil
			}
			hit, err := f.svc.IsAnyBanned(context.Background(), sigs)
			if err != nil || !hit {
				t.Fatalf("local ban at depth %d during the read not honoured: banned=%v err=%v", k, hit, err)
			}
		})
	}
}

// blockingBanStore holds Ban until release is closed.
type blockingBanStore struct {
	*scriptedStore
	entered, release chan struct{}
}

func (s *blockingBanStore) Ban(ctx context.Context, record BanRecord) error {
	close(s.entered)
	<-s.release
	return s.scriptedStore.Ban(ctx, record)
}

// A ban made on this gateway counts from the moment it is made: while its
// write to the shared store is in progress (the store does not have it yet),
// a decision on this gateway refuses.
func TestDecision_LocalBanBeingWrittenRefuses(t *testing.T) {
	mr := miniredis.RunT(t)
	st := &blockingBanStore{
		scriptedStore: &scriptedStore{RedisStore: newMiniRedisStore(t, mr.Addr())},
		entered:       make(chan struct{}),
		release:       make(chan struct{}),
	}
	svc := NewService(st)
	banned := make(chan error, 1)
	go func() { banned <- svc.BanContext(context.Background(), "sig-writing", "test", "admin") }()
	<-st.entered
	hit, err := svc.IsAnyBanned(context.Background(), []string{"sig-writing", "child"})
	close(st.release)
	if e := <-banned; e != nil {
		t.Fatal(e)
	}
	if err != nil || !hit {
		t.Fatalf("ban being written here not honoured: banned=%v err=%v", hit, err)
	}
}

// When cached answers aged during the read and the second read (of the whole
// lineage) fails or answers BanCacheTTL late, the request is refused with
// ErrRevocationUnavailable, never allowed.
func TestDecision_RefreshThatFailsOrAnswersLateRefuses(t *testing.T) {
	for _, tc := range []struct {
		name   string
		second func(f *decisionFixture) error
	}{
		{"store_error", func(*decisionFixture) error { return errors.New("store down") }},
		{"late_answer", func(f *decisionFixture) error { f.clock.set(2500*time.Millisecond + BanCacheTTL); return nil }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDecisionFixture(t)
			f.cache("ancestor", 0)
			f.clock.set(1900 * time.Millisecond)
			f.store.onRead = func(n int, _ []string) error {
				switch n {
				case 1:
					f.clock.set(2500 * time.Millisecond)
				case 2:
					return tc.second(f)
				}
				return nil
			}
			hit, err := f.svc.IsAnyBanned(context.Background(), []string{"ancestor", "child"})
			if hit || !errors.Is(err, ErrRevocationUnavailable) {
				t.Fatalf("want ErrRevocationUnavailable, got banned=%v err=%v", hit, err)
			}
		})
	}
}

// The whole decision, both reads included, ends within BanStoreReadTimeout.
func TestDecision_TwoReadsShareOneDeadline(t *testing.T) {
	f := newDecisionFixture(t)
	f.svc.now = time.Now
	f.svc.cacheMu.Lock()
	f.svc.putCacheLocked("ancestor", false, time.Now().Add(-1500*time.Millisecond))
	f.svc.cacheMu.Unlock()
	var second atomic.Bool
	f.store.onRead = func(n int, _ []string) error {
		if n == 1 {
			time.Sleep(700 * time.Millisecond) // the ancestor answer is now 2.2s old
			return nil
		}
		second.Store(true)
		time.Sleep(3 * time.Second) // never answers in time
		return nil
	}
	start := time.Now()
	hit, err := f.svc.IsAnyBanned(context.Background(), []string{"ancestor", "child"})
	elapsed := time.Since(start)
	if hit || !errors.Is(err, ErrRevocationUnavailable) {
		t.Fatalf("want ErrRevocationUnavailable, got banned=%v err=%v", hit, err)
	}
	if !second.Load() {
		t.Fatal("the aged ancestor answer was not read again")
	}
	if elapsed > BanStoreReadTimeout+300*time.Millisecond {
		t.Fatalf("decision took %s; bound is %s", elapsed, BanStoreReadTimeout)
	}
}

// A ban made on this gateway while the views read the shared ban list is
// shown by that read.
func TestViewsContext_LocalBanDuringListReadShown(t *testing.T) {
	mr := miniredis.RunT(t)
	st := &listHeldStore{RedisStore: newMiniRedisStore(t, mr.Addr()), read: make(chan struct{}), release: make(chan struct{})}
	svc := NewService(st)
	type res struct {
		bans []BanRecord
		err  error
	}
	done := make(chan res, 1)
	go func() {
		bans, err := svc.GetBanListContext(context.Background())
		done <- res{bans, err}
	}()
	<-st.read
	if err := svc.BanContext(context.Background(), "sig-view-local", "test", "admin"); err != nil {
		t.Fatal(err)
	}
	close(st.release)
	r := <-done
	if r.err != nil {
		t.Fatal(r.err)
	}
	for _, b := range r.bans {
		if b.Signature == "sig-view-local" {
			return
		}
	}
	t.Fatalf("ban made during the list read not shown: %v", r.bans)
}

// listHeldStore reads the ban list, then waits for release before answering.
type listHeldStore struct {
	*RedisStore
	once          sync.Once
	read, release chan struct{}
}

func (s *listHeldStore) GetBanList(ctx context.Context) ([]BanRecord, error) {
	list, err := s.RedisStore.GetBanList(ctx)
	s.once.Do(func() { close(s.read); <-s.release })
	return list, err
}
