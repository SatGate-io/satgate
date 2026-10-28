package governance

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// The count of ended ban store writes made on this gateway (banSeq) orders a
// ban made here after every lookup that started before its write ended. It
// never wraps: a ban made when the count is at its maximum is still recorded
// here and in the store, and from then on every access decision on this
// gateway is refused with errBanSeqSpent (wrapping ErrRevocationUnavailable),
// covered by a ban or not, including a decision already in progress when that
// ban's write ends, until the process restarts; the change is logged once.
// The tests reach the boundary through SetBanSequenceForTest.

// lockedBuffer is a log sink safe for concurrent writes.
type lockedBuffer struct {
	mu sync.Mutex
	b  bytes.Buffer
}

func (l *lockedBuffer) Write(p []byte) (int, error) {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.Write(p)
}

func (l *lockedBuffer) String() string {
	l.mu.Lock()
	defer l.mu.Unlock()
	return l.b.String()
}

// captureLog sends the global logger to a buffer for the rest of the test.
func captureLog(t *testing.T) *lockedBuffer {
	t.Helper()
	buf := &lockedBuffer{}
	prev := log.Logger
	log.Logger = zerolog.New(buf)
	t.Cleanup(func() { log.Logger = prev })
	return buf
}

const seqExhaustedLog = "Local ban sequence exhausted"

// A ban made at the maximum is recorded locally and in the store, the count
// stays at the maximum, and every later decision on this gateway, covered by
// a ban or not, is refused as unavailable, also after Reset and Unban. The
// other gateway on the store honours the ban and is otherwise unaffected.
// The exhaustion is logged once.
func TestSequence_BanAtMaximumFailsClosedWithoutWrapping(t *testing.T) {
	logs := captureLog(t)
	f := newDecisionFixture(t)
	f.svc.now = time.Now
	s := f.svc
	s.SetBanSequenceForTest(math.MaxUint64)

	if err := s.BanContext(context.Background(), "ancestor", "test", "admin"); err != nil {
		t.Fatalf("ban at the maximum: %v", err)
	}
	if seq, spent := s.BanSequenceForTest(); seq != math.MaxUint64 || !spent {
		t.Fatalf("after the ban at the maximum: seq=%d spent=%v, want the maximum and spent", seq, spent)
	}
	if inStore, err := f.store.RedisStore.IsBanned(context.Background(), "ancestor"); err != nil || !inStore {
		t.Fatalf("ban not in the store: %v %v", inStore, err)
	}
	s.mu.RLock()
	_, local := s.banned["ancestor"]
	s.mu.RUnlock()
	if !local {
		t.Fatal("ban not recorded on this gateway")
	}
	if hit, err := f.other.IsAnyBanned(context.Background(), []string{"ancestor", "child"}); !hit || err != nil {
		t.Fatalf("other gateway: banned=%v err=%v, want banned", hit, err)
	}
	if hit, err := f.other.IsAnyBanned(context.Background(), []string{"unrelated"}); hit || err != nil {
		t.Fatalf("other gateway, unrelated token: banned=%v err=%v, want allowed", hit, err)
	}

	refusedUnavailable := func(when string, sigs ...string) {
		t.Helper()
		hit, err := s.IsAnyBanned(context.Background(), sigs)
		if hit || !errors.Is(err, ErrRevocationUnavailable) {
			t.Fatalf("%s, lineage %v: banned=%v err=%v, want refused as unavailable", when, sigs, hit, err)
		}
		if !s.IsBanned(sigs[0]) {
			t.Fatalf("%s: IsBanned(%s) allowed", when, sigs[0])
		}
	}
	refusedUnavailable("later decision", "unrelated-root", "unrelated-child")
	refusedUnavailable("later decision", "ancestor", "child")

	// A second ban at the maximum: still recorded, still no wrap, no
	// second log line.
	if err := s.BanContext(context.Background(), "second", "test", "admin"); err != nil {
		t.Fatalf("second ban at the maximum: %v", err)
	}
	if seq, spent := s.BanSequenceForTest(); seq != math.MaxUint64 || !spent {
		t.Fatalf("after a second ban: seq=%d spent=%v", seq, spent)
	}
	if inStore, _ := f.store.RedisStore.IsBanned(context.Background(), "second"); !inStore {
		t.Fatal("second ban not in the store")
	}
	s.Unban("ancestor")
	refusedUnavailable("after unban", "ancestor", "child")
	s.Reset()
	refusedUnavailable("after reset", "unrelated-root")

	if n := strings.Count(logs.String(), seqExhaustedLog); n != 1 {
		t.Fatalf("exhaustion logged %d times, want once", n)
	}
}

// Below the maximum nothing changes: the ban that takes the count to the
// maximum is ordered like any other, decisions on unrelated tokens are
// allowed, and nothing is logged.
func TestSequence_BanReachingMaximumIsOrderedNormally(t *testing.T) {
	logs := captureLog(t)
	f := newDecisionFixture(t)
	f.svc.now = time.Now
	s := f.svc
	s.SetBanSequenceForTest(math.MaxUint64 - 1)
	if err := s.BanContext(context.Background(), "ancestor", "test", "admin"); err != nil {
		t.Fatal(err)
	}
	if seq, spent := s.BanSequenceForTest(); seq != math.MaxUint64 || spent {
		t.Fatalf("seq=%d spent=%v, want the maximum and not spent", seq, spent)
	}
	if hit, err := s.IsAnyBanned(context.Background(), []string{"unrelated"}); hit || err != nil {
		t.Fatalf("unrelated token at the maximum: banned=%v err=%v, want allowed", hit, err)
	}
	if hit, err := s.IsAnyBanned(context.Background(), []string{"ancestor", "child"}); !hit || err != nil {
		t.Fatalf("banned lineage at the maximum: banned=%v err=%v, want banned", hit, err)
	}
	if strings.Contains(logs.String(), seqExhaustedLog) {
		t.Fatal("exhaustion logged before it happened")
	}
}

// An answer in flight across the boundary: a decision whose store read
// started when the count was at the maximum, with a ban made here (taking the
// count past the maximum) before it settles and the cache turned over. The
// decision is refused as unavailable whether or not the ban covers its
// lineage, never allowed on the answer it read, and no later decision is
// allowed. With the count one below the maximum the same ban is ordered
// normally: the covered decision answers banned and the uncovered one is
// allowed.
func TestSequence_AnswerInFlightAcrossBoundaryNotUsedToAllow(t *testing.T) {
	for _, tc := range []struct {
		name    string
		seed    uint64
		covered bool
	}{
		{"covered_at_maximum", math.MaxUint64, true},
		{"uncovered_at_maximum", math.MaxUint64, false},
		{"covered_below_maximum", math.MaxUint64 - 1, true},
		{"uncovered_below_maximum", math.MaxUint64 - 1, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDecisionFixture(t)
			f.svc.now = time.Now
			s := f.svc
			s.SetBanSequenceForTest(tc.seed)
			sigs := []string{"ancestor", "child"}
			if !tc.covered {
				sigs = []string{"other-root", "other-child"}
			}
			f.store.onRead = func(n int, _ []string) error {
				if n == 1 {
					if err := s.BanContext(context.Background(), "ancestor", "test", "admin"); err != nil {
						t.Error(err)
					}
					s.TurnOverBanCacheForTest()
				}
				return nil
			}
			hit, err := s.IsAnyBanned(context.Background(), sigs)
			atMax := tc.seed == math.MaxUint64
			switch {
			case atMax && (hit || !errors.Is(err, errBanSeqSpent)):
				t.Fatalf("in flight across the boundary, covered=%v: banned=%v err=%v, want refused as unavailable (sequence spent)", tc.covered, hit, err)
			case tc.covered && !atMax && (!hit || err != nil):
				t.Fatalf("in flight below the maximum, covered: banned=%v err=%v, want banned", hit, err)
			case !tc.covered && !atMax && (hit || err != nil):
				t.Fatalf("in flight below the maximum, uncovered: banned=%v err=%v, want allowed", hit, err)
			}
			for _, later := range [][]string{{"ancestor", "child"}, sigs} {
				hit, err := s.IsAnyBanned(context.Background(), later)
				if atMax && (hit || !errors.Is(err, ErrRevocationUnavailable)) {
					t.Fatalf("later decision %v past the maximum: banned=%v err=%v, want refused as unavailable", later, hit, err)
				}
				if !atMax && later[0] == "ancestor" && (!hit || err != nil) {
					t.Fatalf("later decision %v below the maximum: banned=%v err=%v, want banned", later, hit, err)
				}
			}
		})
	}
}

// banFailStore is a scriptedStore whose ban writes fail.
type banFailStore struct{ *scriptedStore }

func (banFailStore) Ban(context.Context, BanRecord) error { return errors.New("ban write failed") }

// A decision in progress when a ban made here spends the count ends with
// errBanSeqSpent, the refusal a decision starting then gets, on every way
// it can settle: whether a ban made here covers its lineage (written to the
// store, or not written and pending) or not, and whether its store answer
// arrived in time or after BanCacheTTL.
func TestSequence_DecisionInProgressWhenSpentReturnsSpentRefusal(t *testing.T) {
	for _, tc := range []struct {
		name      string
		covered   bool
		failWrite bool // the ban's store write fails (pending ban)
		late      bool // the store answer arrives after BanCacheTTL
	}{
		{"covered", true, false, false},
		{"covered_pending", true, true, false},
		{"uncovered", false, false, false},
		{"covered_late_answer", true, false, true},
		{"uncovered_late_answer", false, false, true},
		{"uncovered_pending", false, true, false},
	} {
		t.Run(tc.name, func(t *testing.T) {
			mr := miniredis.RunT(t)
			scripted := &scriptedStore{RedisStore: newMiniRedisStore(t, mr.Addr())}
			var st Store = scripted
			if tc.failWrite {
				st = banFailStore{scripted}
			}
			s := NewService(st)
			clk := newTestClock()
			s.now = clk.Now
			s.SetBanSequenceForTest(math.MaxUint64)
			sigs := []string{"ancestor", "child"}
			if !tc.covered {
				sigs = []string{"other-root", "other-child"}
			}
			scripted.onRead = func(n int, _ []string) error {
				if n == 1 {
					if tc.late {
						clk.set(3 * time.Second)
					}
					err := s.BanContext(context.Background(), "ancestor", "test", "admin")
					if tc.failWrite != (err != nil) {
						t.Errorf("ban: %v", err)
					}
				}
				return nil
			}
			hit, err := s.IsAnyBanned(context.Background(), sigs)
			if hit || !errors.Is(err, errBanSeqSpent) {
				t.Fatalf("decision in progress when the count was spent: banned=%v err=%v, want %v", hit, err, errBanSeqSpent)
			}
			if _, spent := s.BanSequenceForTest(); !spent {
				t.Fatal("fixture: the count is not spent")
			}
			if hit, err := s.IsAnyBanned(context.Background(), sigs); hit || !errors.Is(err, errBanSeqSpent) {
				t.Fatalf("later decision: banned=%v err=%v, want %v", hit, err, errBanSeqSpent)
			}
		})
	}
}

// ---- the reviewers' cases ----

// inFlightRefused runs the decision in flight of the round-4 cases: with the
// ban made at the maximum it is refused as unavailable (sequence spent); with
// an ordinary count it answers banned.
func inFlightRefused(t *testing.T, seed uint64, sigs []string, s *Service) {
	t.Helper()
	hit, err := s.IsAnyBanned(context.Background(), sigs)
	if seed == math.MaxUint64 {
		if hit || !errors.Is(err, errBanSeqSpent) {
			t.Fatalf("in flight across the boundary: banned=%v err=%v, want refused as unavailable (sequence spent)", hit, err)
		}
		return
	}
	if !hit || err != nil {
		t.Fatalf("in flight: banned=%v err=%v, want banned", hit, err)
	}
}

// Astra: with the count at the maximum, a ban made here while a decision read
// the store, then cache turnover; the decision in flight is refused (as
// unavailable at the maximum, banned in the ordinary arm) and a later one,
// 100ms on, is not allowed on the "not banned" answer read before the ban.
// The ordinary arm (count 17) is the control.
func TestSequence_PreBanAnswerNotRevivedAtSequenceMaximum(t *testing.T) {
	for _, tc := range []struct {
		name string
		seed uint64
	}{{"ordinary", 17}, {"maximum", math.MaxUint64}} {
		t.Run(tc.name, func(t *testing.T) {
			f := newDecisionFixture(t)
			s := f.svc
			s.SetBanSequenceForTest(tc.seed)
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
			inFlightRefused(t, tc.seed, sigs, s)
			f.clock.set(100 * time.Millisecond)
			if hit, err := s.IsAnyBanned(context.Background(), sigs); !hit && err == nil {
				t.Fatal("a \"not banned\" answer read before the ban allowed a later decision")
			}
		})
	}
}

// Sol: the count at 0, one below the maximum and the maximum, the real clock;
// a ban made here while a decision read the store, then the cache filled
// past its bound. The decision in flight is refused (as unavailable at the
// maximum, else banned) and the next one is not allowed.
func TestSequence_CompletedLocalBanRejectsPreBanAnswerAtEverySeed(t *testing.T) {
	for _, seed := range []uint64{0, math.MaxUint64 - 1, math.MaxUint64} {
		t.Run(fmt.Sprintf("seed_%d", seed), func(t *testing.T) {
			f := newDecisionFixture(t)
			s := f.svc
			s.now = time.Now
			s.SetBanSequenceForTest(seed)
			f.store.onRead = func(n int, _ []string) error {
				if n == 1 {
					if err := s.BanContext(context.Background(), "ancestor", "test", "admin"); err != nil {
						return err
					}
					s.TurnOverBanCacheForTest()
				}
				return nil
			}
			sigs := []string{"ancestor", "child"}
			inFlightRefused(t, seed, sigs, s)
			if hit, err := s.IsAnyBanned(context.Background(), sigs); !hit && err == nil {
				t.Fatal("a completed local ban did not reject the \"not banned\" answer read before it")
			}
		})
	}
}

// Sol: Reset keeps the count (and whether it is spent).
func TestSequence_ResetKeepsCount(t *testing.T) {
	f := newDecisionFixture(t)
	s := f.svc
	if err := s.BanContext(context.Background(), "ancestor", "test", "admin"); err != nil {
		t.Fatal(err)
	}
	before, _ := s.BanSequenceForTest()
	s.Reset()
	if after, _ := s.BanSequenceForTest(); after != before || after == 0 {
		t.Fatalf("reset changed the count: %d -> %d", before, after)
	}
}
