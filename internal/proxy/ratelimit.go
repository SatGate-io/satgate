package proxy

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/internal/config"
)

// RateLimitEvent represents a rate limit event for analytics
type RateLimitEvent struct {
	Route       string
	Key         string
	KeyType     string
	Count       int64
	Limit       int
	Window      string
	Blocked     bool
	ClientIP    string
	UserAgent   string
	RequestPath string
}

// RateLimitEventRecorder is called when rate limit events occur
type RateLimitEventRecorder func(event RateLimitEvent)

// RouteRateLimiter handles per-route rate limiting
type RouteRateLimiter struct {
	store         RateLimitStore
	config        *config.Config
	eventRecorder RateLimitEventRecorder
}

// RateLimitStore interface for rate limit state storage
type RateLimitStore interface {
	Increment(ctx context.Context, key string, window time.Duration) (int64, error)
}

// MemoryRateLimitStore is an in-memory implementation (not HA-safe)
type MemoryRateLimitStore struct {
	counts map[string]*rateLimitCounter
	mu     sync.RWMutex
}

type rateLimitCounter struct {
	count     int64
	expiresAt time.Time
}

// NewMemoryRateLimitStore creates a new in-memory rate limit store
func NewMemoryRateLimitStore() *MemoryRateLimitStore {
	store := &MemoryRateLimitStore{
		counts: make(map[string]*rateLimitCounter),
	}
	go store.cleanup()
	return store
}

func (m *MemoryRateLimitStore) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	c, exists := m.counts[key]

	if !exists || now.After(c.expiresAt) {
		m.counts[key] = &rateLimitCounter{
			count:     1,
			expiresAt: now.Add(window),
		}
		return 1, nil
	}

	c.count++
	return c.count, nil
}

func (m *MemoryRateLimitStore) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		m.mu.Lock()
		now := time.Now()
		for key, c := range m.counts {
			if now.After(c.expiresAt) {
				delete(m.counts, key)
			}
		}
		m.mu.Unlock()
	}
}

// RedisRateLimitStore is a Redis-backed implementation (HA-safe)
type RedisRateLimitStore struct {
	client *redis.Client
}

// NewRedisRateLimitStore creates a Redis-backed rate limit store
func NewRedisRateLimitStore(addr, password string, db int) (*RedisRateLimitStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     addr,
		Password: password,
		DB:       db,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, err
	}

	log.Info().Str("addr", addr).Msg("Redis rate limiter connected")
	return &RedisRateLimitStore{client: client}, nil
}

func (r *RedisRateLimitStore) Increment(ctx context.Context, key string, window time.Duration) (int64, error) {
	fullKey := "ratelimit:dataplane:" + key

	// Use INCR with EXPIRE for atomic increment
	pipe := r.client.TxPipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, window)

	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

// NewRouteRateLimiter creates a route rate limiter with the appropriate backend
func NewRouteRateLimiter(cfg *config.Config) (*RouteRateLimiter, error) {
	var store RateLimitStore

	if cfg.Redis != nil && cfg.Redis.Enabled {
		redisStore, err := NewRedisRateLimitStore(cfg.Redis.Addr, cfg.Redis.Password, cfg.Redis.DB)
		if err != nil {
			log.Warn().Err(err).Msg("Failed to connect to Redis for rate limiting, falling back to memory")
			store = NewMemoryRateLimitStore()
		} else {
			store = redisStore
			log.Info().Msg("Route rate limiting using Redis backend (multi-instance consistent)")
		}
	} else {
		store = NewMemoryRateLimitStore()
		log.Warn().Msg("Route rate limiting using memory backend (not consistent across instances)")
	}

	return &RouteRateLimiter{
		store:  store,
		config: cfg,
	}, nil
}

// SetEventRecorder sets the callback for rate limit events
func (rl *RouteRateLimiter) SetEventRecorder(recorder RateLimitEventRecorder) {
	if rl != nil {
		rl.eventRecorder = recorder
	}
}

// Check verifies rate limits for a route and returns true if allowed
func (rl *RouteRateLimiter) Check(r *http.Request, route *config.Route) (allowed bool, headers map[string]string) {
	headers = make(map[string]string)

	// No rate limit configured for this route
	if route.RateLimit == nil || route.RateLimit.RequestsPerMinute <= 0 {
		return true, headers
	}

	// Extract rate limit key based on configuration
	key := rl.extractKey(r, route)
	keyType := route.RateLimit.Key
	if keyType == "" {
		keyType = "ip"
	}

	ctx, cancel := context.WithTimeout(r.Context(), 2*time.Second)
	defer cancel()

	count, err := rl.store.Increment(ctx, key, time.Minute)
	if err != nil {
		// Fail open for availability
		log.Warn().Err(err).Str("key", key).Msg("Rate limit check failed, allowing request")
		return true, headers
	}

	limit := route.RateLimit.RequestsPerMinute
	remaining := limit - int(count)
	if remaining < 0 {
		remaining = 0
	}

	// Set rate limit headers
	headers["X-RateLimit-Limit"] = strconv.Itoa(limit)
	headers["X-RateLimit-Remaining"] = strconv.Itoa(remaining)
	headers["X-RateLimit-Reset"] = strconv.FormatInt(time.Now().Add(time.Minute).Unix(), 10)

	blocked := int(count) > limit
	
	// Record event if we're at threshold or blocked
	if rl.eventRecorder != nil {
		// Record when blocked or approaching limit (80%+)
		shouldRecord := blocked || float64(count) >= float64(limit)*0.8
		if shouldRecord {
			rl.eventRecorder(RateLimitEvent{
				Route:       route.Name,
				Key:         key,
				KeyType:     keyType,
				Count:       count,
				Limit:       limit,
				Window:      "minute",
				Blocked:     blocked,
				ClientIP:    extractClientIP(r),
				UserAgent:   r.UserAgent(),
				RequestPath: r.URL.Path,
			})
		}
	}

	if blocked {
		log.Warn().
			Str("route", route.Name).
			Str("key", key).
			Int64("count", count).
			Int("limit", limit).
			Msg("Route rate limit exceeded")
		headers["Retry-After"] = "60"
		return false, headers
	}

	return true, headers
}

// extractKey gets the rate limit key based on route configuration
func (rl *RouteRateLimiter) extractKey(r *http.Request, route *config.Route) string {
	keyType := route.RateLimit.Key
	if keyType == "" {
		keyType = "ip" // Default to IP-based
	}

	routeName := route.Name

	switch keyType {
	case "token":
		// Use token (or its prefix) as key
		token := r.Header.Get("Authorization")
		if token == "" {
			token = r.Header.Get("X-API-Key")
		}
		if len(token) > 32 {
			token = token[:32] // Use prefix for privacy
		}
		return routeName + ":token:" + token

	case "global":
		// All requests share the same limit
		return routeName + ":global"

	case "tenant":
		// Per-tenant rate limiting
		tenantID := r.Header.Get("X-Tenant-ID")
		if tenantID == "" {
			tenantID = "default"
		}
		return routeName + ":tenant:" + tenantID

	default:
		// Check for custom header: "header:X-Custom"
		if strings.HasPrefix(keyType, "header:") {
			headerName := strings.TrimPrefix(keyType, "header:")
			headerValue := r.Header.Get(headerName)
			if headerValue == "" {
				headerValue = "unknown"
			}
			return routeName + ":header:" + headerValue
		}

		// Default to IP-based
		ip := extractClientIP(r)
		return routeName + ":ip:" + ip
	}
}

// extractClientIP gets the client IP from the request
func extractClientIP(r *http.Request) string {
	// Check X-Forwarded-For first (for proxied requests)
	if xff := r.Header.Get("X-Forwarded-For"); xff != "" {
		ips := strings.Split(xff, ",")
		if len(ips) > 0 {
			return strings.TrimSpace(ips[0])
		}
	}

	// Check X-Real-IP
	if xri := r.Header.Get("X-Real-IP"); xri != "" {
		return xri
	}

	// Fall back to RemoteAddr
	ip := r.RemoteAddr
	if colonIdx := strings.LastIndex(ip, ":"); colonIdx != -1 {
		ip = ip[:colonIdx]
	}
	return ip
}

