package proxy

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"github.com/satgate-io/satgate/pkg/config"
)

const (
	// l402MacaroonLifetime is how long an L402 challenge macaroon is valid.
	l402MacaroonLifetime = time.Hour

	// l402ReplayTTL is how long a consumed payment hash is remembered. It
	// must be at least l402MacaroonLifetime so that a proof cannot be used
	// again while its macaroon is still valid.
	l402ReplayTTL = 24 * time.Hour

	// l402ReplayKeyPrefix prefixes consumed payment hashes in Redis.
	l402ReplayKeyPrefix = "satgate:l402:consumed:"

	l402ReplayStoreTimeout = 2 * time.Second
)

// errL402ReplayStoreUnavailable is returned when the consumed-proof store
// cannot be reached. The request is refused, never served.
var errL402ReplayStoreUnavailable = errors.New("L402 consumed-proof store unavailable")

// L402ReplayStore records consumed L402 payment hashes so that each payment
// proof is accepted once.
type L402ReplayStore interface {
	// Consume records paymentHash as used for ttl. It returns true the first
	// time a hash is recorded and false when it was already recorded.
	Consume(ctx context.Context, paymentHash string, ttl time.Duration) (bool, error)
}

// memoryReplayStore keeps consumed payment hashes in process memory. It is
// correct for a single gateway instance only: a restart or a second replica
// does not see the hashes it recorded. Configure Redis for anything else.
type memoryReplayStore struct {
	seen sync.Map // payment_hash -> expiry time
}

func newMemoryReplayStore(cleanupInterval time.Duration) *memoryReplayStore {
	rs := &memoryReplayStore{}
	go func() {
		ticker := time.NewTicker(cleanupInterval)
		defer ticker.Stop()
		for range ticker.C {
			now := time.Now()
			rs.seen.Range(func(key, value any) bool {
				if expiry, ok := value.(time.Time); ok && now.After(expiry) {
					rs.seen.Delete(key)
				}
				return true
			})
		}
	}()
	return rs
}

func (rs *memoryReplayStore) Consume(_ context.Context, paymentHash string, ttl time.Duration) (bool, error) {
	_, loaded := rs.seen.LoadOrStore(paymentHash, time.Now().Add(ttl))
	return !loaded, nil
}

// redisReplayStore keeps consumed payment hashes in Redis, shared by every
// gateway instance that uses the same Redis.
type redisReplayStore struct {
	client *redis.Client
}

// NewRedisL402ReplayStore returns a consumed-proof store backed by client.
func NewRedisL402ReplayStore(client *redis.Client) L402ReplayStore {
	return &redisReplayStore{client: client}
}

func (rs *redisReplayStore) Consume(ctx context.Context, paymentHash string, ttl time.Duration) (bool, error) {
	first, err := rs.client.SetNX(ctx, l402ReplayKeyPrefix+paymentHash, "1", ttl).Result()
	if err != nil {
		return false, err
	}
	return first, nil
}

// newConfiguredReplayStore returns the Redis store when Redis is configured
// and enabled, and nil otherwise. A configured Redis that cannot be reached is
// an error, so a gateway never falls back to a per-process store silently.
func newConfiguredReplayStore(cfg *config.RedisConfig) (L402ReplayStore, error) {
	if cfg == nil || !cfg.Enabled {
		return nil, nil
	}
	if cfg.Addr == "" {
		return nil, fmt.Errorf("redis is enabled but redis.addr is empty")
	}
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		_ = client.Close()
		return nil, fmt.Errorf("connect to redis for the L402 consumed-proof store: %w", err)
	}
	return NewRedisL402ReplayStore(client), nil
}
