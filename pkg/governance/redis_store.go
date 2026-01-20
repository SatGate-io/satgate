package governance

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/redis/go-redis/v9"
)

const (
	// Redis key prefixes
	keyBanList   = "satgate:banlist"      // SET of banned signatures
	keyBanRecord = "satgate:ban:"         // HASH for ban record details
	keyMinted    = "satgate:minted:"      // HASH for minted token details
	keyMintedSet = "satgate:minted:all"   // SET of all minted signatures
	keyUsage     = "satgate:usage:"       // HASH for usage stats
	keyUsageSet  = "satgate:usage:all"    // SET of all usage signatures
	keyCounter   = "satgate:counter:"     // STRING for counters (rate limiting)
)

// RedisStore implements Store using Redis for HA persistence
type RedisStore struct {
	client *redis.Client
}

// RedisConfig holds Redis connection settings
type RedisConfig struct {
	Addr     string `yaml:"addr"`     // e.g., "localhost:6379"
	Password string `yaml:"password"` // empty for no auth
	DB       int    `yaml:"db"`       // database number
}

// NewRedisStore creates a new Redis-backed store
func NewRedisStore(cfg *RedisConfig) (*RedisStore, error) {
	client := redis.NewClient(&redis.Options{
		Addr:     cfg.Addr,
		Password: cfg.Password,
		DB:       cfg.DB,
	})

	// Test connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := client.Ping(ctx).Err(); err != nil {
		return nil, fmt.Errorf("failed to connect to Redis: %w", err)
	}

	return &RedisStore{client: client}, nil
}

// Ban adds a token to the ban list
func (r *RedisStore) Ban(ctx context.Context, record BanRecord) error {
	// Add to ban set
	if err := r.client.SAdd(ctx, keyBanList, record.Signature).Err(); err != nil {
		return fmt.Errorf("failed to add to ban set: %w", err)
	}

	// Store ban details
	data, err := json.Marshal(record)
	if err != nil {
		return fmt.Errorf("failed to marshal ban record: %w", err)
	}

	if err := r.client.Set(ctx, keyBanRecord+record.Signature, data, 0).Err(); err != nil {
		return fmt.Errorf("failed to store ban record: %w", err)
	}

	return nil
}

// Unban removes a token from the ban list
func (r *RedisStore) Unban(ctx context.Context, signature string) error {
	pipe := r.client.Pipeline()
	pipe.SRem(ctx, keyBanList, signature)
	pipe.Del(ctx, keyBanRecord+signature)
	_, err := pipe.Exec(ctx)
	return err
}

// IsBanned checks if a token is banned
func (r *RedisStore) IsBanned(ctx context.Context, signature string) (bool, error) {
	return r.client.SIsMember(ctx, keyBanList, signature).Result()
}

// GetBanList returns all banned tokens
func (r *RedisStore) GetBanList(ctx context.Context) ([]BanRecord, error) {
	signatures, err := r.client.SMembers(ctx, keyBanList).Result()
	if err != nil {
		return nil, err
	}

	if len(signatures) == 0 {
		return []BanRecord{}, nil
	}

	// Fetch all ban records
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(signatures))
	for i, sig := range signatures {
		cmds[i] = pipe.Get(ctx, keyBanRecord+sig)
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	result := make([]BanRecord, 0, len(signatures))
	for i, cmd := range cmds {
		data, err := cmd.Result()
		if err == redis.Nil {
			// Record missing, just add signature
			result = append(result, BanRecord{Signature: signatures[i]})
			continue
		}
		if err != nil {
			continue
		}

		var record BanRecord
		if err := json.Unmarshal([]byte(data), &record); err != nil {
			continue
		}
		result = append(result, record)
	}

	return result, nil
}

// RegisterMint records a newly minted token
func (r *RedisStore) RegisterMint(ctx context.Context, token *MintedToken) error {
	data, err := json.Marshal(token)
	if err != nil {
		return fmt.Errorf("failed to marshal token: %w", err)
	}

	// Calculate TTL based on expiration
	ttl := time.Until(token.ExpiresAt)
	if ttl < 0 {
		ttl = 24 * time.Hour // Keep expired tokens for audit
	} else {
		ttl += 24 * time.Hour // Keep beyond expiry for audit
	}

	pipe := r.client.Pipeline()
	pipe.Set(ctx, keyMinted+token.Signature, data, ttl)
	pipe.SAdd(ctx, keyMintedSet, token.Signature)
	_, err = pipe.Exec(ctx)
	return err
}

// GetMintedToken retrieves a minted token by signature
func (r *RedisStore) GetMintedToken(ctx context.Context, signature string) (*MintedToken, error) {
	data, err := r.client.Get(ctx, keyMinted+signature).Result()
	if err == redis.Nil {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}

	var token MintedToken
	if err := json.Unmarshal([]byte(data), &token); err != nil {
		return nil, err
	}
	return &token, nil
}

// GetAllMintedTokens returns all minted tokens
func (r *RedisStore) GetAllMintedTokens(ctx context.Context) ([]*MintedToken, error) {
	signatures, err := r.client.SMembers(ctx, keyMintedSet).Result()
	if err != nil {
		return nil, err
	}

	if len(signatures) == 0 {
		return []*MintedToken{}, nil
	}

	// Fetch all tokens in a pipeline
	pipe := r.client.Pipeline()
	cmds := make([]*redis.StringCmd, len(signatures))
	for i, sig := range signatures {
		cmds[i] = pipe.Get(ctx, keyMinted+sig)
	}
	_, err = pipe.Exec(ctx)
	if err != nil && err != redis.Nil {
		return nil, err
	}

	result := make([]*MintedToken, 0, len(signatures))
	for _, cmd := range cmds {
		data, err := cmd.Result()
		if err != nil {
			continue
		}

		var token MintedToken
		if err := json.Unmarshal([]byte(data), &token); err != nil {
			continue
		}
		result = append(result, &token)
	}

	return result, nil
}

// RecordUsage tracks token usage
func (r *RedisStore) RecordUsage(ctx context.Context, signature, route string) error {
	now := time.Now()
	pipe := r.client.Pipeline()

	// Increment total requests
	pipe.HIncrBy(ctx, keyUsage+signature, "total", 1)
	// Update last used
	pipe.HSet(ctx, keyUsage+signature, "lastUsed", now.Unix())
	// Increment route counter
	pipe.HIncrBy(ctx, keyUsage+signature, "route:"+route, 1)
	// Add to usage set
	pipe.SAdd(ctx, keyUsageSet, signature)

	_, err := pipe.Exec(ctx)
	return err
}

// GetUsage returns usage stats for a token
func (r *RedisStore) GetUsage(ctx context.Context, signature string) (*UsageStats, error) {
	data, err := r.client.HGetAll(ctx, keyUsage+signature).Result()
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return nil, nil
	}

	stats := &UsageStats{
		Signature: signature,
		Routes:    make(map[string]int64),
	}

	for key, val := range data {
		switch {
		case key == "total":
			fmt.Sscanf(val, "%d", &stats.TotalRequests)
		case key == "lastUsed":
			var ts int64
			fmt.Sscanf(val, "%d", &ts)
			stats.LastUsed = time.Unix(ts, 0)
		case len(key) > 6 && key[:6] == "route:":
			var count int64
			fmt.Sscanf(val, "%d", &count)
			stats.Routes[key[6:]] = count
		}
	}

	return stats, nil
}

// GetAllUsage returns usage stats for all tokens
func (r *RedisStore) GetAllUsage(ctx context.Context) (map[string]*UsageStats, error) {
	signatures, err := r.client.SMembers(ctx, keyUsageSet).Result()
	if err != nil {
		return nil, err
	}

	result := make(map[string]*UsageStats)
	for _, sig := range signatures {
		stats, err := r.GetUsage(ctx, sig)
		if err != nil {
			continue
		}
		if stats != nil {
			result[sig] = stats
		}
	}

	return result, nil
}

// IncrementCounter atomically increments a counter with TTL (for rate limiting)
func (r *RedisStore) IncrementCounter(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	fullKey := keyCounter + key

	// Use INCR + EXPIRE in a transaction
	pipe := r.client.Pipeline()
	incr := pipe.Incr(ctx, fullKey)
	pipe.Expire(ctx, fullKey, ttl)
	_, err := pipe.Exec(ctx)
	if err != nil {
		return 0, err
	}

	return incr.Val(), nil
}

// GetCounter returns the current value of a counter
func (r *RedisStore) GetCounter(ctx context.Context, key string) (int64, error) {
	val, err := r.client.Get(ctx, keyCounter+key).Int64()
	if err == redis.Nil {
		return 0, nil
	}
	return val, err
}

// Reset clears all governance data (use with caution!)
func (r *RedisStore) Reset(ctx context.Context) error {
	// Get all keys with our prefix
	keys, err := r.client.Keys(ctx, "satgate:*").Result()
	if err != nil {
		return err
	}

	if len(keys) == 0 {
		return nil
	}

	return r.client.Del(ctx, keys...).Err()
}

// Close closes the Redis connection
func (r *RedisStore) Close() error {
	return r.client.Close()
}

// IncrWithTTL is an alias for IncrementCounter to match the rate limiter interface
func (r *RedisStore) IncrWithTTL(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	return r.IncrementCounter(ctx, key, ttl)
}

