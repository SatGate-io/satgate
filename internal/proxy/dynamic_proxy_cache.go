package proxy

import (
	"net/http/httputil"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// DynamicProxyTTL is the time-to-live for cached tenant proxies.
// Unused proxies are evicted after this duration.
const DynamicProxyTTL = 5 * time.Minute

// DynamicProxyEvictionInterval is how often we check for stale proxies.
const DynamicProxyEvictionInterval = 1 * time.Minute

// DynamicProxyCache manages a cache of reverse proxies with TTL-based eviction.
// This is used for tenant-defined upstream URLs in Cloud mode.
type DynamicProxyCache struct {
	mu      sync.RWMutex
	entries map[string]*cachedProxyEntry
	ttl     time.Duration
	stop    chan struct{}
}

// cachedProxyEntry wraps a reverse proxy with metadata for TTL eviction.
type cachedProxyEntry struct {
	proxy    *httputil.ReverseProxy
	lastUsed time.Time
	key      string
}

// NewDynamicProxyCache creates a new cache with TTL eviction.
func NewDynamicProxyCache(ttl time.Duration) *DynamicProxyCache {
	if ttl <= 0 {
		ttl = DynamicProxyTTL
	}
	c := &DynamicProxyCache{
		entries: make(map[string]*cachedProxyEntry),
		ttl:     ttl,
		stop:    make(chan struct{}),
	}
	go c.runEviction()
	return c
}

// Get retrieves a proxy from the cache, updating its last-used time.
func (c *DynamicProxyCache) Get(key string) (*httputil.ReverseProxy, bool) {
	c.mu.RLock()
	entry, ok := c.entries[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	// Update last used (write lock)
	c.mu.Lock()
	if entry, ok = c.entries[key]; ok {
		entry.lastUsed = time.Now()
	}
	c.mu.Unlock()

	return entry.proxy, ok
}

// Set stores a proxy in the cache.
func (c *DynamicProxyCache) Set(key string, proxy *httputil.ReverseProxy) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries[key] = &cachedProxyEntry{
		proxy:    proxy,
		lastUsed: time.Now(),
		key:      key,
	}
}

// GetOrCreate retrieves a cached proxy or creates a new one using the factory function.
func (c *DynamicProxyCache) GetOrCreate(key string, factory func() (*httputil.ReverseProxy, error)) (*httputil.ReverseProxy, error) {
	// Fast path: check cache
	if proxy, ok := c.Get(key); ok {
		return proxy, nil
	}

	// Slow path: create new proxy
	c.mu.Lock()
	defer c.mu.Unlock()

	// Double-check after acquiring write lock
	if entry, ok := c.entries[key]; ok {
		entry.lastUsed = time.Now()
		return entry.proxy, nil
	}

	proxy, err := factory()
	if err != nil {
		return nil, err
	}

	c.entries[key] = &cachedProxyEntry{
		proxy:    proxy,
		lastUsed: time.Now(),
		key:      key,
	}

	return proxy, nil
}

// Size returns the number of cached proxies.
func (c *DynamicProxyCache) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.entries)
}

// Stop stops the eviction goroutine.
func (c *DynamicProxyCache) Stop() {
	close(c.stop)
}

// runEviction periodically evicts stale entries.
func (c *DynamicProxyCache) runEviction() {
	ticker := time.NewTicker(DynamicProxyEvictionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stop:
			return
		case <-ticker.C:
			c.evictStale()
		}
	}
}

// evictStale removes entries that haven't been used within the TTL.
func (c *DynamicProxyCache) evictStale() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	evicted := 0

	for key, entry := range c.entries {
		if now.Sub(entry.lastUsed) > c.ttl {
			delete(c.entries, key)
			evicted++
		}
	}

	if evicted > 0 {
		log.Debug().
			Int("evicted", evicted).
			Int("remaining", len(c.entries)).
			Dur("ttl", c.ttl).
			Msg("Evicted stale dynamic proxy entries")
	}
}


