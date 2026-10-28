package governance

import (
	"fmt"
	"time"
)

// Hooks for the tests in this directory; the external tests (package
// governance_test) drive the decision through the HTTP gateway.

// HoldCacheLockForTest takes the cache lock and returns its release.
func (s *Service) HoldCacheLockForTest() (release func()) {
	s.cacheMu.Lock()
	return s.cacheMu.Unlock
}

// SetClockForTest replaces the service clock. Call it before the service is
// used concurrently.
func (s *Service) SetClockForTest(now func() time.Time) { s.now = now }

// SetBanSequenceForTest sets the count of ended ban store writes (banSeq),
// to put the service at the boundary of that count.
func (s *Service) SetBanSequenceForTest(v uint64) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.banSeq = v
}

// BanSequenceForTest returns banSeq and whether it is spent.
func (s *Service) BanSequenceForTest() (seq uint64, spent bool) {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	return s.banSeq, s.banSeqSpent
}

// TurnOverBanCacheForTest fills the ban cache past its bound with "not
// banned" answers, so that it is cleared and every earlier entry is gone.
func (s *Service) TurnOverBanCacheForTest() {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	at := s.now()
	for i := 0; i <= maxBanCacheEntries; i++ {
		s.putCacheLocked(fmt.Sprintf("filler-%d", i), false, at)
	}
}

// OldestCachedAnswerForTest returns the start of the oldest cached lookup
// among sigs.
func (s *Service) OldestCachedAnswerForTest(sigs []string) time.Time {
	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	var oldest time.Time
	for _, sig := range sigs {
		if at := s.banCache[sig].at; oldest.IsZero() || at.Before(oldest) {
			oldest = at
		}
	}
	return oldest
}
