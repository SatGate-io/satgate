package governance

import "time"

// Hooks for the external tests in this directory (package governance_test),
// which drive the decision through the HTTP gateway.

// HoldCacheLockForTest takes the cache lock and returns its release.
func (s *Service) HoldCacheLockForTest() (release func()) {
	s.cacheMu.Lock()
	return s.cacheMu.Unlock
}

// SetClockForTest replaces the service clock. Call it before the service is
// used concurrently.
func (s *Service) SetClockForTest(now func() time.Time) { s.now = now }

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
