package proxy

import (
	"net"
	"net/http"
	"strings"
	"sync"
	"time"
)

// rateLimiter implements a simple in-memory token bucket rate limiter
// keyed by IP address. Designed for admin API protection.
type rateLimiter struct {
	mu        sync.Mutex
	buckets   map[string]*bucket
	rate      int           // requests per window
	window    time.Duration // sliding window size
	cleanup   time.Duration // how often to purge expired entries
	lastPurge time.Time
}

type bucket struct {
	tokens []time.Time // timestamps of recent requests
}

// newRateLimiter creates a rate limiter. rate=0 means unlimited.
func newRateLimiter(requestsPerMinute int) *rateLimiter {
	return &rateLimiter{
		buckets:   make(map[string]*bucket),
		rate:      requestsPerMinute,
		window:    time.Minute,
		cleanup:   5 * time.Minute,
		lastPurge: time.Now(),
	}
}

// allow checks if a request from the given key should be allowed.
// Returns true if allowed, false if rate limited.
func (rl *rateLimiter) allow(key string) bool {
	if rl.rate <= 0 {
		return true // unlimited
	}

	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()
	cutoff := now.Add(-rl.window)

	// Periodic cleanup of stale entries
	if now.Sub(rl.lastPurge) > rl.cleanup {
		for k, b := range rl.buckets {
			b.prune(cutoff)
			if len(b.tokens) == 0 {
				delete(rl.buckets, k)
			}
		}
		rl.lastPurge = now
	}

	b, ok := rl.buckets[key]
	if !ok {
		b = &bucket{}
		rl.buckets[key] = b
	}

	// Prune old tokens
	b.prune(cutoff)

	if len(b.tokens) >= rl.rate {
		return false
	}

	b.tokens = append(b.tokens, now)
	return true
}

// prune removes timestamps older than cutoff
func (b *bucket) prune(cutoff time.Time) {
	i := 0
	for i < len(b.tokens) && b.tokens[i].Before(cutoff) {
		i++
	}
	if i > 0 {
		b.tokens = b.tokens[i:]
	}
}

// extractIP gets the client IP from trusted sources only.
// X-Forwarded-For is user-spoofable without trusted proxy configuration.
// Fly.io sets Fly-Client-IP at its edge (not spoofable by clients).
// Falls back to RemoteAddr (TCP peer) which is always reliable.
func extractIP(r *http.Request) string {
	// Fly.io edge header — set by the platform, not the client
	if flyIP := r.Header.Get("Fly-Client-IP"); flyIP != "" {
		return strings.TrimSpace(flyIP)
	}

	// Fall back to RemoteAddr (TCP-level, always trustworthy)
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
