package proxy

import (
	"net"
	"net/http"
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

// extractIP gets the client IP, respecting X-Forwarded-For for proxied setups
func extractIP(r *http.Request) string {
	// Check X-Forwarded-For first (leftmost = original client)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		// Take the first IP (original client)
		for i := 0; i < len(xff); i++ {
			if xff[i] == ',' {
				ip := xff[:i]
				// Trim whitespace
				for len(ip) > 0 && ip[0] == ' ' {
					ip = ip[1:]
				}
				return ip
			}
		}
		return xff
	}

	// Fall back to RemoteAddr
	host, _, err := net.SplitHostPort(r.RemoteAddr)
	if err != nil {
		return r.RemoteAddr
	}
	return host
}
