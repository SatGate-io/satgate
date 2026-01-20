package governance

import (
	"context"
	"sync"
	"time"
)

// Store defines the interface for governance data persistence
// Implementations must be safe for concurrent access
type Store interface {
	// Ban list operations
	Ban(ctx context.Context, record BanRecord) error
	Unban(ctx context.Context, signature string) error
	IsBanned(ctx context.Context, signature string) (bool, error)
	GetBanList(ctx context.Context) ([]BanRecord, error)

	// Token registration
	RegisterMint(ctx context.Context, token *MintedToken) error
	GetMintedToken(ctx context.Context, signature string) (*MintedToken, error)
	GetAllMintedTokens(ctx context.Context) ([]*MintedToken, error)

	// Usage tracking
	RecordUsage(ctx context.Context, signature, route string) error
	GetUsage(ctx context.Context, signature string) (*UsageStats, error)
	GetAllUsage(ctx context.Context) (map[string]*UsageStats, error)

	// Metering (for capability token limits)
	IncrementCounter(ctx context.Context, key string, ttl time.Duration) (int64, error)
	GetCounter(ctx context.Context, key string) (int64, error)

	// Admin
	Reset(ctx context.Context) error
	Close() error
}

// MemoryStore is an in-memory implementation of Store (for testing/single instance)
// Thread-safe via internal mutex
type MemoryStore struct {
	mu       sync.RWMutex
	banned   map[string]BanRecord
	minted   map[string]*MintedToken
	usage    map[string]*UsageStats
	counters map[string]int64
}

// NewMemoryStore creates a new in-memory store
func NewMemoryStore() *MemoryStore {
	return &MemoryStore{
		banned:   make(map[string]BanRecord),
		minted:   make(map[string]*MintedToken),
		usage:    make(map[string]*UsageStats),
		counters: make(map[string]int64),
	}
}

func (m *MemoryStore) Ban(ctx context.Context, record BanRecord) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.banned[record.Signature] = record
	return nil
}

func (m *MemoryStore) Unban(ctx context.Context, signature string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.banned, signature)
	return nil
}

func (m *MemoryStore) IsBanned(ctx context.Context, signature string) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, banned := m.banned[signature]
	return banned, nil
}

func (m *MemoryStore) GetBanList(ctx context.Context) ([]BanRecord, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]BanRecord, 0, len(m.banned))
	for _, record := range m.banned {
		result = append(result, record)
	}
	return result, nil
}

func (m *MemoryStore) RegisterMint(ctx context.Context, token *MintedToken) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.minted[token.Signature] = token
	return nil
}

func (m *MemoryStore) GetMintedToken(ctx context.Context, signature string) (*MintedToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.minted[signature], nil
}

func (m *MemoryStore) GetAllMintedTokens(ctx context.Context) ([]*MintedToken, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	result := make([]*MintedToken, 0, len(m.minted))
	for _, token := range m.minted {
		result = append(result, token)
	}
	return result, nil
}

func (m *MemoryStore) RecordUsage(ctx context.Context, signature, route string) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	stats, ok := m.usage[signature]
	if !ok {
		stats = &UsageStats{
			Signature: signature,
			Routes:    make(map[string]int64),
		}
		m.usage[signature] = stats
	}
	stats.TotalRequests++
	stats.LastUsed = time.Now()
	stats.Routes[route]++
	return nil
}

func (m *MemoryStore) GetUsage(ctx context.Context, signature string) (*UsageStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.usage[signature], nil
}

func (m *MemoryStore) GetAllUsage(ctx context.Context) (map[string]*UsageStats, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	// Return a copy to avoid race conditions
	result := make(map[string]*UsageStats, len(m.usage))
	for k, v := range m.usage {
		result[k] = v
	}
	return result, nil
}

func (m *MemoryStore) IncrementCounter(ctx context.Context, key string, ttl time.Duration) (int64, error) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.counters[key]++
	return m.counters[key], nil
}

func (m *MemoryStore) GetCounter(ctx context.Context, key string) (int64, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.counters[key], nil
}

func (m *MemoryStore) Reset(ctx context.Context) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.banned = make(map[string]BanRecord)
	m.minted = make(map[string]*MintedToken)
	m.usage = make(map[string]*UsageStats)
	m.counters = make(map[string]int64)
	return nil
}

func (m *MemoryStore) Close() error {
	return nil
}

