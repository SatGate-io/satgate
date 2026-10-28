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

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
)

// The count of ended ban store writes made on this gateway (banSeq) orders a
// ban made here after every lookup that started before its write ended. It
// never wraps: a ban made when the count is at its maximum is still recorded
// here and in the store, and from then on every access decision on this
// gateway is refused with ErrRevocationUnavailable (a decision covered by a
// ban made here still answers banned) until the process restarts; the change
// is logged once. The tests reach the boundary through SetBanSequenceForTest.

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
// count past the maximum) before it settles and the cache turned over. A
// decision whose lineage the ban covers answers banned; one it does not
// cover is refused as unavailable, not allowed on the answer it read; and
// neither answer is used to allow a later decision. With the count one below
// the maximum the same ban is ordered normally: the covered decision answers
// banned and the uncovered one is allowed.
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
			case tc.covered && (!hit || err != nil):
				t.Fatalf("in flight, covered: banned=%v err=%v, want banned", hit, err)
			case !tc.covered && atMax && (hit || !errors.Is(err, ErrRevocationUnavailable)):
				t.Fatalf("in flight across the boundary, uncovered: banned=%v err=%v, want refused as unavailable", hit, err)
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

// ---- the reviewers' cases ----

// Astra: with the count at the maximum, a ban made here while a decision read
// the store, then cache turnover; the decision in flight refuses (banned) and
// a later one, 100ms on, is not allowed on the "not banned" answer read
// before the ban. The ordinary arm (count 17) is the control.
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
			if hit, err := s.IsAnyBanned(context.Background(), sigs); !hit || err != nil {
				t.Fatalf("in flight: banned=%v err=%v, want banned", hit, err)
			}
			f.clock.set(100 * time.Millisecond)
			if hit, err := s.IsAnyBanned(context.Background(), sigs); !hit && err == nil {
				t.Fatal("a \"not banned\" answer read before the ban allowed a later decision")
			}
		})
	}
}

// Sol: the count at 0, one below the maximum and the maximum, the real clock;
// a ban made here while a decision read the store, then the cache filled
// past its bound. The decision in flight refuses (banned) and the next one
// is not allowed.
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
			if hit, err := s.IsAnyBanned(context.Background(), sigs); !hit || err != nil {
				t.Fatalf("in flight: banned=%v err=%v, want banned", hit, err)
			}
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
