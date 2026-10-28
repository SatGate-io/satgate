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

// Tests for the final settlement of an access decision with a shared store:
// the clock is read under the cache lock after every cache update, and the
// decision allows only when every lineage answer is fresh at that moment and
// the decision deadline has not passed. A wait for the cache lock never
// extends an answer's freshness or the deadline. And a ban made on this
// gateway is never outlived by a "not banned" answer from a lookup that
// started before the ban's store write ended, even after the cache turned
// over. They use only API that existed before these rules (IsAnyBanned,
// BanContext, cacheMu, now, putCacheLocked), so they also run against the
// earlier code.

type decisionResult struct {
	banned bool
	err    error
}

func (r decisionResult) allowed() bool { return !r.banned && r.err == nil }

// decideAsync runs IsAnyBanned on another goroutine.
func decideAsync(s *Service, sigs []string) <-chan decisionResult {
	done := make(chan decisionResult, 1)
	go func() {
		b, err := s.IsAnyBanned(context.Background(), sigs)
		done <- decisionResult{b, err}
	}()
	return done
}

// armStart makes the fixture clock report, by closing the returned channel,
// the first time the service reads it (the start of the next decision). The
// value read is taken before the channel is closed.
func (f *decisionFixture) armStart() <-chan struct{} {
	started := make(chan struct{})
	var once sync.Once
	f.svc.now = func() time.Time {
		v := f.clock.Now()
		once.Do(func() { close(started) })
		return v
	}
	return started
}

// armAfterRead makes the first store read take the cache lock and keep it,
// and reports, by closing the returned channel, the first clock read after
// that (the decision has its answer and is about to settle). The caller
// releases the lock with f.svc.cacheMu.Unlock.
func (f *decisionFixture) armAfterRead() <-chan struct{} {
	settling := make(chan struct{})
	var phase atomic.Int32
	f.svc.now = func() time.Time {
		v := f.clock.Now()
		if phase.CompareAndSwap(1, 2) {
			close(settling)
		}
		return v
	}
	f.store.onRead = func(n int, _ []string) error {
		if n == 1 {
			f.svc.cacheMu.Lock()
			phase.Store(1)
		}
		return nil
	}
	return settling
}

// B1, initial settlement: answers cached at 0 and 1.9s old when the decision
// starts, the decision waits for the cache lock until 2.1s. At every depth
// a ban made elsewhere meanwhile is honoured: the answers, expired at the
// moment the decision holds the lock, are read again.
func TestSettlement_CachedAnswersExpiredDuringCacheLockWaitAreReadAgain(t *testing.T) {
	sigs := lineageOf(lineageLen)
	run := func(t *testing.T, ban int) (decisionResult, [][]string) {
		f := newDecisionFixture(t)
		for _, sig := range sigs {
			f.cache(sig, 0)
		}
		if ban >= 0 {
			f.banElsewhere(t, sigs[ban])
		}
		f.clock.set(1900 * time.Millisecond)
		started := f.armStart()
		f.svc.cacheMu.Lock()
		done := decideAsync(f.svc, sigs)
		<-started
		f.clock.set(2100 * time.Millisecond)
		f.svc.cacheMu.Unlock()
		return <-done, f.store.readLog()
	}
	for k := range sigs {
		t.Run(fmt.Sprintf("banned_depth_%d", k), func(t *testing.T) {
			if r, reads := run(t, k); r.allowed() {
				t.Fatalf("allowed on answers 2.1s old after a cache lock wait (ban at depth %d); reads=%v", k, reads)
			}
		})
	}
	t.Run("nothing_banned", func(t *testing.T) {
		r, reads := run(t, -1)
		if !r.allowed() {
			t.Fatalf("nothing banned: banned=%v err=%v", r.banned, r.err)
		}
		if len(reads) != 1 || len(reads[0]) != lineageLen {
			t.Fatalf("want one read of the whole lineage: %v", reads)
		}
	})
}

// B1, settlement after a store read: the decision reads the missing entry,
// then waits for the cache lock to settle; the ancestor answer cached at 0
// is 1.9s old when the read ends and 2.1s old when the decision holds the
// lock. It is read again and the ancestor ban made elsewhere refuses.
func TestSettlement_AnswerExpiredWhileWaitingToSettleAfterReadIsReadAgain(t *testing.T) {
	sigs := lineageOf(lineageLen)
	for k := 0; k < lineageLen-1; k++ {
		t.Run(fmt.Sprintf("banned_depth_%d", k), func(t *testing.T) {
			f := newDecisionFixture(t)
			for _, sig := range sigs[:lineageLen-1] {
				f.cache(sig, 0)
			}
			f.banElsewhere(t, sigs[k])
			f.clock.set(1900 * time.Millisecond)
			settling := f.armAfterRead()
			done := decideAsync(f.svc, sigs)
			<-settling
			f.clock.set(2100 * time.Millisecond)
			f.svc.cacheMu.Unlock()
			if r := <-done; r.allowed() {
				t.Fatalf("allowed on an answer 2.1s old at settlement (ban at depth %d); reads=%v", k, f.store.readLog())
			}
		})
	}
}

// B1, deadline, initial settlement: the decision starts at 0.1s and holds the
// cache lock at 2.2s, past its deadline (0.1s + BanStoreReadTimeout), with
// fresh answers for every entry cached meanwhile. It is refused with
// ErrRevocationUnavailable.
func TestSettlement_DeadlinePassedWhileWaitingForCacheLockRefusesUnavailable(t *testing.T) {
	f := newDecisionFixture(t)
	sigs := lineageOf(lineageLen)
	f.clock.set(100 * time.Millisecond)
	started := f.armStart()
	f.svc.cacheMu.Lock()
	done := decideAsync(f.svc, sigs)
	<-started
	f.clock.set(2200 * time.Millisecond)
	for _, sig := range sigs {
		f.svc.putCacheLocked(sig, false, f.clock.at(2150*time.Millisecond))
	}
	f.svc.cacheMu.Unlock()
	r := <-done
	if r.banned || !errors.Is(r.err, ErrRevocationUnavailable) {
		t.Fatalf("want ErrRevocationUnavailable after the deadline, got banned=%v err=%v", r.banned, r.err)
	}
}

// B1, deadline, settlement after a read: the decision starts at 0.5s, its read
// ends, and it holds the cache lock at 2.6s, past its deadline, with fresh
// answers cached meanwhile for the entries it did not read. It is refused
// with ErrRevocationUnavailable.
func TestSettlement_DeadlinePassedWhileWaitingToSettleAfterReadRefusesUnavailable(t *testing.T) {
	f := newDecisionFixture(t)
	sigs := lineageOf(lineageLen)
	f.clock.set(500 * time.Millisecond)
	settling := f.armAfterRead()
	done := decideAsync(f.svc, sigs)
	<-settling
	f.clock.set(2600 * time.Millisecond)
	for _, sig := range sigs[:lineageLen-1] {
		f.svc.putCacheLocked(sig, false, f.clock.at(2550*time.Millisecond))
	}
	f.svc.cacheMu.Unlock()
	r := <-done
	if r.banned || !errors.Is(r.err, ErrRevocationUnavailable) {
		t.Fatalf("want ErrRevocationUnavailable after the deadline, got banned=%v err=%v", r.banned, r.err)
	}
}

// Control: a short wait for the cache lock, answers still fresh and the
// deadline not passed at settlement: allowed with no read.
func TestSettlement_ShortCacheLockWaitWithFreshAnswersAllows(t *testing.T) {
	f := newDecisionFixture(t)
	sigs := lineageOf(lineageLen)
	for _, sig := range sigs {
		f.cache(sig, time.Second)
	}
	f.clock.set(2 * time.Second)
	started := f.armStart()
	f.svc.cacheMu.Lock()
	done := decideAsync(f.svc, sigs)
	<-started
	f.clock.set(2900 * time.Millisecond)
	f.svc.cacheMu.Unlock()
	if r := <-done; !r.allowed() {
		t.Fatalf("fresh answers within the deadline: banned=%v err=%v", r.banned, r.err)
	}
	if reads := f.store.readLog(); len(reads) != 0 {
		t.Fatalf("no read expected: %v", reads)
	}
}

// turnOver fills the cache past maxBanCacheEntries with fresh answers for
// other signatures, so the bounded cache drops what it held (the real
// insertion path). cacheMu must not be held.
func turnOver(s *Service) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	at := s.now()
	for i := 0; i <= maxBanCacheEntries; i++ {
		s.putCacheLocked(fmt.Sprintf("filler-%d", i), false, at)
	}
}

// B2: a ban made here while a decision read the store for an ancestor (so the
// read missed it), then cache turnover before that decision settles. The
// decision in flight refuses; so does every later one, at every depth: the
// "not banned" answer read before the ban is never cached or used.
func TestSettlement_CompletedLocalBanNotOutlivedByOlderAnswerAfterCacheTurnover(t *testing.T) {
	sigs := lineageOf(lineageLen)
	for k := range sigs {
		for _, turnover := range []string{"bounded_cache_overflow", "cache_cleared"} {
			t.Run(fmt.Sprintf("depth_%d_%s", k, turnover), func(t *testing.T) {
				f := newDecisionFixture(t)
				f.store.onRead = func(n int, _ []string) error {
					if n == 1 {
						if err := f.svc.BanContext(context.Background(), sigs[k], "test", "admin"); err != nil {
							t.Error(err)
						}
						if turnover == "cache_cleared" {
							f.svc.cacheMu.Lock()
							f.svc.banCache = make(map[string]banCacheEntry)
							f.svc.cacheMu.Unlock()
						} else {
							turnOver(f.svc)
						}
					}
					return nil
				}
				if hit, err := f.svc.IsAnyBanned(context.Background(), sigs); !hit || err != nil {
					t.Fatalf("decision in flight: banned=%v err=%v", hit, err)
				}
				f.clock.set(100 * time.Millisecond)
				for i := 0; i < 2; i++ {
					if hit, err := f.svc.IsAnyBanned(context.Background(), sigs); !hit && err == nil {
						t.Fatalf("later decision %d allowed despite the completed local ban at depth %d", i+1, k)
					}
				}
			})
		}
	}
}

// Control: an answer from a lookup that started after a local ban's write
// ended is used as usual, and a ban covering a sibling does not touch it.
func TestSettlement_AnswerReadAfterLocalBanIsUsed(t *testing.T) {
	f := newDecisionFixture(t)
	if err := f.svc.BanContext(context.Background(), "sibling", "test", "admin"); err != nil {
		t.Fatal(err)
	}
	f.clock.set(100 * time.Millisecond)
	sigs := []string{"parent", "child"}
	if hit, err := f.svc.IsAnyBanned(context.Background(), sigs); hit || err != nil {
		t.Fatalf("first: banned=%v err=%v", hit, err)
	}
	f.clock.set(200 * time.Millisecond)
	if hit, err := f.svc.IsAnyBanned(context.Background(), sigs); hit || err != nil {
		t.Fatalf("second: banned=%v err=%v", hit, err)
	}
	if reads := f.store.readLog(); len(reads) != 1 {
		t.Fatalf("want the cached answers used by the second decision: %v", reads)
	}
}

// ---- the reviewers' cases, real clock ----

// Astra: real monotonic clock; the ancestor answer is 1.85s old when the
// decision starts and 2.1s old when it gets the cache lock.
func TestSettlement_RealClock_CacheLockWaitDoesNotExtendFreshness(t *testing.T) {
	f := newDecisionFixture(t)
	s := f.svc
	s.now = time.Now
	if hit, err := s.IsAnyBanned(context.Background(), []string{"ancestor"}); hit || err != nil {
		t.Fatal("healthy control")
	}
	f.banElsewhere(t, "ancestor")
	s.cacheMu.Lock()
	at := s.banCache["ancestor"].at
	s.cacheMu.Unlock()
	time.Sleep(time.Until(at.Add(1850 * time.Millisecond)))
	started := make(chan struct{})
	var once sync.Once
	s.now = func() time.Time { v := time.Now(); once.Do(func() { close(started) }); return v }
	s.cacheMu.Lock()
	done := decideAsync(s, []string{"ancestor"})
	<-started
	time.Sleep(time.Until(at.Add(2100 * time.Millisecond)))
	s.cacheMu.Unlock()
	if r := <-done; r.allowed() {
		t.Fatalf("allowed on a cached ancestor answer older than the TTL after a cache lock wait; age=%s", time.Since(at))
	}
}

// Sol: two services on one Redis; the ancestor is banned on the other one
// after this one cached its answer; the decision waits for the cache lock
// past the TTL.
func TestSettlement_RealClock_CacheLockWaitRevalidatesAtAllowAcrossGateways(t *testing.T) {
	mr := miniredis.RunT(t)
	s := newRedisService(t, mr.Addr())
	other := newRedisService(t, mr.Addr())
	sigs := []string{"review-ancestor", "review-child"}
	if hit, err := s.IsAnyBanned(context.Background(), sigs); hit || err != nil {
		t.Fatal("warm control")
	}
	if err := other.BanContext(context.Background(), sigs[0], "test", "admin"); err != nil {
		t.Fatal(err)
	}
	s.cacheMu.Lock()
	at := s.banCache[sigs[0]].at
	s.cacheMu.Unlock()
	time.Sleep(time.Until(at.Add(1850 * time.Millisecond)))
	started := make(chan struct{})
	var once atomic.Bool
	s.now = func() time.Time {
		v := time.Now()
		if once.CompareAndSwap(false, true) {
			close(started)
		}
		return v
	}
	s.cacheMu.Lock()
	done := decideAsync(s, sigs)
	<-started
	time.Sleep(time.Until(at.Add(2100 * time.Millisecond)))
	s.cacheMu.Unlock()
	if r := <-done; r.allowed() {
		t.Fatal("allowed using a cached ancestor answer older than the TTL after waiting for the cache lock")
	}
}

// Sol: the decision reads the child, then waits for the cache lock to settle
// until the cached ancestor answer is 2.1s old.
func TestSettlement_RealClock_PostReadLockWaitRevalidatesAtAllow(t *testing.T) {
	f := newDecisionFixture(t)
	f.svc.now = time.Now
	sigs := []string{"review-ancestor", "review-child"}
	at := time.Now()
	f.svc.cacheMu.Lock()
	f.svc.putCacheLocked(sigs[0], false, at)
	f.svc.cacheMu.Unlock()
	f.banElsewhere(t, sigs[0])
	time.Sleep(time.Until(at.Add(1850 * time.Millisecond)))
	locked := make(chan struct{})
	settling := make(chan struct{})
	var phase atomic.Int64
	f.svc.now = func() time.Time {
		v := time.Now()
		if phase.CompareAndSwap(1, 2) {
			close(settling)
		}
		return v
	}
	f.store.onRead = func(n int, _ []string) error {
		if n == 1 {
			f.svc.cacheMu.Lock()
			close(locked)
			phase.Store(1)
		}
		return nil
	}
	done := decideAsync(f.svc, sigs)
	<-locked
	<-settling
	time.Sleep(time.Until(at.Add(2100 * time.Millisecond)))
	f.svc.cacheMu.Unlock()
	if r := <-done; r.allowed() {
		t.Fatal("allowed using a freshness time taken before the settlement lock wait")
	}
}

// Sol: the decision waits for the cache lock longer than BanStoreReadTimeout
// with fresh cached answers.
func TestSettlement_RealClock_DecisionDeadlineCheckedAfterCacheLockWait(t *testing.T) {
	mr := miniredis.RunT(t)
	s := newRedisService(t, mr.Addr())
	sigs := []string{"review-deadline-ancestor", "review-deadline-child"}
	if hit, err := s.IsAnyBanned(context.Background(), sigs); hit || err != nil {
		t.Fatal("healthy control")
	}
	started := make(chan struct{})
	var once atomic.Bool
	s.now = func() time.Time {
		v := time.Now()
		if once.CompareAndSwap(false, true) {
			close(started)
		}
		return v
	}
	s.cacheMu.Lock()
	done := decideAsync(s, sigs)
	<-started
	time.Sleep(BanStoreReadTimeout + 100*time.Millisecond)
	s.cacheMu.Unlock()
	if r := <-done; r.allowed() {
		t.Fatal("allowed after the whole-decision deadline")
	}
}

// Astra: real clock; a ban made here during the read, then the bounded cache
// overflows; the next decision must not allow.
func TestSettlement_RealClock_CompletedLocalBanSurvivesOldAnswerAfterEviction(t *testing.T) {
	f := newDecisionFixture(t)
	s := f.svc
	s.now = time.Now
	f.store.onRead = func(n int, _ []string) error {
		if n == 1 {
			if err := s.BanContext(context.Background(), "ancestor", "test", "admin"); err != nil {
				return err
			}
			turnOver(s)
		}
		return nil
	}
	sigs := []string{"ancestor", "child"}
	if hit, err := s.IsAnyBanned(context.Background(), sigs); !hit || err != nil {
		t.Fatalf("in-flight decision did not refuse: %v %v", hit, err)
	}
	if hit, err := s.IsAnyBanned(context.Background(), sigs); !hit && err == nil {
		t.Fatal("next decision allowed despite a completed local ancestor ban")
	}
}
