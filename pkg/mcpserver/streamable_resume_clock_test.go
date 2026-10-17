package mcpserver

import "time"

// advanceStreamTombstones moves the DELETE tombstone clock of s forward by d:
// every stored tombstone deadline is brought d closer, which is the same as
// d passing for them. Live sessions are not touched.
func advanceStreamTombstones(s *SSEServer, d time.Duration) {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	for id, until := range s.streamEnded {
		s.streamEnded[id] = until.Add(-d)
	}
	if !s.streamEndedFullUntil.IsZero() {
		s.streamEndedFullUntil = s.streamEndedFullUntil.Add(-d)
	}
	if !s.streamEndedNextExpiry.IsZero() {
		s.streamEndedNextExpiry = s.streamEndedNextExpiry.Add(-d)
	}
}

func streamTombstones(s *SSEServer) (ids map[string]time.Time, fullUntil time.Time) {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	ids = make(map[string]time.Time, len(s.streamEnded))
	for id, until := range s.streamEnded {
		ids[id] = until
	}
	return ids, s.streamEndedFullUntil
}
