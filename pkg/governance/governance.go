package governance

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// Service handles token governance (banning, rate limiting, telemetry)
type Service struct {
	store Store // Backing store (Redis or memory)

	// Local cache for hot path (IsBanned checks)
	banned  map[string]BanRecord
	usage   map[string]*UsageStats
	minted  map[string]*MintedToken
	mu      sync.RWMutex

	// Counters for dashboard stats
	blockedRequests int64 // Unpaid requests (402s)
	bannedHits      int64 // Requests from banned tokens
	counterMu       sync.Mutex
}

// MintedToken tracks a minted token
type MintedToken struct {
	Signature  string
	Scope      string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	ParentSig  string // Parent token signature (empty for root tokens)
	Depth      int    // 0 = root, 1 = minted, 2+ = delegated
	Label      string // Human-readable label
}

// BanRecord represents a banned token
type BanRecord struct {
	Signature string
	Reason    string
	BannedAt  time.Time
	BannedBy  string
}

// UsageStats tracks token usage
type UsageStats struct {
	Signature    string
	TotalRequests int64
	LastUsed     time.Time
	Routes       map[string]int64 // route -> count
}

// TokenInfo represents token status for API responses
type TokenInfo struct {
	Signature     string            `json:"signature"`
	Status        string            `json:"status"`
	Scope         string            `json:"scope,omitempty"`
	CreatedAt     *time.Time        `json:"createdAt,omitempty"`
	ExpiresAt     *time.Time        `json:"expiresAt,omitempty"`
	TotalRequests int64             `json:"totalRequests"`
	LastUsed      *time.Time        `json:"lastUsed,omitempty"`
	Routes        map[string]int64  `json:"routes,omitempty"`
	BannedAt      *time.Time        `json:"bannedAt,omitempty"`
	BanReason     string            `json:"banReason,omitempty"`
}

// NewService creates a new governance service with the given store
// If store is nil, uses in-memory store (single instance only)
func NewService(store Store) *Service {
	if store == nil {
		store = NewMemoryStore()
	}
	return &Service{
		store:  store,
		banned: make(map[string]BanRecord),
		usage:  make(map[string]*UsageStats),
		minted: make(map[string]*MintedToken),
	}
}

// NewServiceWithRedis creates a governance service backed by Redis
func NewServiceWithRedis(cfg *RedisConfig) (*Service, error) {
	store, err := NewRedisStore(cfg)
	if err != nil {
		return nil, err
	}

	svc := NewService(store)

	// Load existing data from Redis into local cache
	ctx := context.Background()
	if err := svc.loadFromStore(ctx); err != nil {
		log.Warn().Err(err).Msg("Failed to load existing governance data from Redis")
	}

	return svc, nil
}

// loadFromStore populates local cache from backing store
func (s *Service) loadFromStore(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	// Load ban list
	bans, err := s.store.GetBanList(ctx)
	if err != nil {
		return err
	}
	for _, ban := range bans {
		s.banned[ban.Signature] = ban
	}

	// Load minted tokens
	tokens, err := s.store.GetAllMintedTokens(ctx)
	if err != nil {
		return err
	}
	for _, token := range tokens {
		s.minted[token.Signature] = token
	}

	// Load usage stats
	usage, err := s.store.GetAllUsage(ctx)
	if err != nil {
		return err
	}
	s.usage = usage

	log.Info().
		Int("banned", len(s.banned)).
		Int("minted", len(s.minted)).
		Int("usage", len(s.usage)).
		Msg("Loaded governance data from store")

	return nil
}

// RegisterMint records a newly minted token (depth 1, no parent)
func (s *Service) RegisterMint(signature, scope string, expiresAt time.Time) {
	s.RegisterMintWithLineage(signature, scope, expiresAt, "", 1, "Agent Token")
}

// RegisterMintWithLineage records a token with full lineage info
func (s *Service) RegisterMintWithLineage(signature, scope string, expiresAt time.Time, parentSig string, depth int, label string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	token := &MintedToken{
		Signature: signature,
		Scope:     scope,
		CreatedAt: time.Now(),
		ExpiresAt: expiresAt,
		ParentSig: parentSig,
		Depth:     depth,
		Label:     label,
	}
	s.minted[signature] = token

	// Persist to store (async, don't block)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.store.RegisterMint(ctx, token); err != nil {
			log.Error().Err(err).Str("signature", signature).Msg("Failed to persist minted token")
		}
	}()
}

// RegisterDelegation records a delegated child token
func (s *Service) RegisterDelegation(signature, scope string, expiresAt time.Time, parentSig string) {
	// Determine depth based on parent
	depth := 2 // Default for delegated tokens
	s.mu.RLock()
	if parent, ok := s.minted[parentSig]; ok {
		depth = parent.Depth + 1
	}
	s.mu.RUnlock()

	s.RegisterMintWithLineage(signature, scope, expiresAt, parentSig, depth, "Worker Token")
}

// RecordBlockedRequest increments the blocked requests counter
func (s *Service) RecordBlockedRequest() {
	s.counterMu.Lock()
	s.blockedRequests++
	s.counterMu.Unlock()
}

// RecordBannedHit increments the banned hits counter
func (s *Service) RecordBannedHit() {
	s.counterMu.Lock()
	s.bannedHits++
	s.counterMu.Unlock()
}

// Ban adds a token to the ban list
func (s *Service) Ban(signature, reason, bannedBy string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	record := BanRecord{
		Signature: signature,
		Reason:    reason,
		BannedAt:  time.Now(),
		BannedBy:  bannedBy,
	}
	s.banned[signature] = record

	// Persist to store (sync for ban operations - critical path)
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := s.store.Ban(ctx, record); err != nil {
		log.Error().Err(err).Str("signature", signature).Msg("Failed to persist ban record")
	}
}

// Unban removes a token from the ban list
func (s *Service) Unban(signature string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, ok := s.banned[signature]; ok {
		delete(s.banned, signature)

		// Persist to store
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.store.Unban(ctx, signature); err != nil {
			log.Error().Err(err).Str("signature", signature).Msg("Failed to persist unban")
		}
		return true
	}
	return false
}

// IsBanned checks if a token is banned
func (s *Service) IsBanned(signature string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()

	_, banned := s.banned[signature]
	return banned
}

// GetBanList returns all banned tokens
func (s *Service) GetBanList() []BanRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]BanRecord, 0, len(s.banned))
	for _, record := range s.banned {
		result = append(result, record)
	}
	return result
}

// RecordUsage tracks token usage
func (s *Service) RecordUsage(signature, route string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	stats, ok := s.usage[signature]
	if !ok {
		stats = &UsageStats{
			Signature: signature,
			Routes:    make(map[string]int64),
		}
		s.usage[signature] = stats
	}

	stats.TotalRequests++
	stats.LastUsed = time.Now()
	stats.Routes[route]++

	// Persist to store (async, don't block hot path)
	go func() {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		if err := s.store.RecordUsage(ctx, signature, route); err != nil {
			log.Debug().Err(err).Str("signature", signature).Msg("Failed to persist usage")
		}
	}()
}

// CheckUsageLimit checks if a token has exceeded its usage limit
// Returns (allowed, currentCount, error)
func (s *Service) CheckUsageLimit(signature string, maxCalls int64, windowSeconds int64) (bool, int64, error) {
	if maxCalls <= 0 {
		return true, 0, nil // No limit
	}

	key := signature + ":calls"
	ttl := time.Duration(windowSeconds) * time.Second

	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()

	count, err := s.store.IncrementCounter(ctx, key, ttl)
	if err != nil {
		// On error, allow (fail open for availability)
		log.Warn().Err(err).Str("signature", signature).Msg("Usage limit check failed, allowing request")
		return true, 0, err
	}

	return count <= maxCalls, count, nil
}

// GetUsage returns usage stats for a token
func (s *Service) GetUsage(signature string) *UsageStats {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if stats, ok := s.usage[signature]; ok {
		return stats
	}
	return nil
}

// GetTokenInfoBySignature returns full info about a specific token
func (s *Service) GetTokenInfoBySignature(signature string) *TokenInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if we know about this token from minting
	minted, hasMinted := s.minted[signature]
	stats, hasStats := s.usage[signature]
	ban, isBanned := s.banned[signature]

	// If we don't have any info about this token, return nil
	if !hasMinted && !hasStats && !isBanned {
		return nil
	}

	info := &TokenInfo{
		Signature: signature,
		Status:    "active",
	}

	// Add minted info if available
	if hasMinted {
		info.Scope = minted.Scope
		info.CreatedAt = &minted.CreatedAt
		info.ExpiresAt = &minted.ExpiresAt
		if time.Now().After(minted.ExpiresAt) {
			info.Status = "expired"
		}
	}

	// Add usage stats if available
	if hasStats {
		info.TotalRequests = stats.TotalRequests
		info.LastUsed = &stats.LastUsed
		info.Routes = stats.Routes
	}

	// Check if banned
	if isBanned {
		info.Status = "banned"
		info.BannedAt = &ban.BannedAt
		info.BanReason = ban.Reason
	}

	return info
}

// GetAllTokenInfo returns info about all tracked tokens
func (s *Service) GetAllTokenInfo() []TokenInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]TokenInfo, 0)
	seen := make(map[string]bool)

	// Start with minted tokens (all tokens we know about)
	for sig, minted := range s.minted {
		info := TokenInfo{
			Signature: sig,
			Status:    "active",
			Scope:     minted.Scope,
			CreatedAt: &minted.CreatedAt,
			ExpiresAt: &minted.ExpiresAt,
		}

		// Check if expired
		if time.Now().After(minted.ExpiresAt) {
			info.Status = "expired"
		}

		// Add usage stats if available
		if stats, ok := s.usage[sig]; ok {
			info.TotalRequests = stats.TotalRequests
			info.LastUsed = &stats.LastUsed
			info.Routes = stats.Routes
		}

		// Check if banned
		if ban, banned := s.banned[sig]; banned {
			info.Status = "banned"
			info.BannedAt = &ban.BannedAt
			info.BanReason = ban.Reason
		}

		result = append(result, info)
		seen[sig] = true
	}

	// Add any used tokens that weren't minted through our API (legacy)
	for sig, stats := range s.usage {
		if seen[sig] {
			continue
		}
		info := TokenInfo{
			Signature:     sig,
			Status:        "active",
			TotalRequests: stats.TotalRequests,
			LastUsed:      &stats.LastUsed,
			Routes:        stats.Routes,
		}

		if ban, banned := s.banned[sig]; banned {
			info.Status = "banned"
			info.BannedAt = &ban.BannedAt
			info.BanReason = ban.Reason
		}

		result = append(result, info)
		seen[sig] = true
	}

	// Add banned tokens that weren't in usage or minted
	for sig, ban := range s.banned {
		if seen[sig] {
			continue
		}
		result = append(result, TokenInfo{
			Signature: sig,
			Status:    "banned",
			BannedAt:  &ban.BannedAt,
			BanReason: ban.Reason,
		})
	}

	return result
}

// GetStats returns aggregate statistics
func (s *Service) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeCount := 0
	bannedCount := len(s.banned)
	expiredCount := 0
	totalRequests := int64(0)

	// Count minted tokens by status
	for sig, minted := range s.minted {
		if _, banned := s.banned[sig]; banned {
			continue // counted in bannedCount
		}
		if time.Now().After(minted.ExpiresAt) {
			expiredCount++
		} else {
			activeCount++
		}
	}

	// Sum usage
	for _, stats := range s.usage {
		totalRequests += stats.TotalRequests
	}

	return map[string]interface{}{
		"active":        activeCount,
		"banned":        bannedCount,
		"expired":       expiredCount,
		"total":         len(s.minted),
		"totalRequests": totalRequests,
	}
}

// GraphNode represents a node in the token lineage graph
type GraphNode struct {
	Data GraphNodeData `json:"data"`
}

// GraphNodeData contains the node details
type GraphNodeData struct {
	ID          string   `json:"id"`
	Label       string   `json:"label"`
	Constraints []string `json:"constraints"`
	LastSeen    string   `json:"lastSeen"`
	Depth       int      `json:"depth"`
	Status      string   `json:"status"`
}

// GraphEdge represents an edge in the token lineage graph
type GraphEdge struct {
	Data GraphEdgeData `json:"data"`
}

// GraphEdgeData contains the edge details
type GraphEdgeData struct {
	Source string `json:"source"`
	Target string `json:"target"`
}

// GraphStats contains aggregate statistics
type GraphStats struct {
	Active     int   `json:"active"`
	Blocked    int64 `json:"blocked"`
	Banned     int   `json:"banned"`
	BannedHits int64 `json:"bannedHits"`
}

// GraphResponse is the full response for /api/governance/graph
type GraphResponse struct {
	Nodes []GraphNode `json:"nodes"`
	Edges []GraphEdge `json:"edges"`
	Stats GraphStats  `json:"stats"`
}

// GetGraph returns the token lineage graph for the dashboard
func (s *Service) GetGraph() GraphResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]GraphNode, 0)
	edges := make([]GraphEdge, 0)

	for sig, token := range s.minted {
		// Determine status
		status := "ACTIVE"
		if _, banned := s.banned[sig]; banned {
			status = "BANNED"
		} else if time.Now().After(token.ExpiresAt) {
			status = "EXPIRED"
		}

		// Build constraints list
		constraints := []string{
			fmt.Sprintf("scope = %s", token.Scope),
		}
		
		// Add expiry as relative time
		expiresIn := time.Until(token.ExpiresAt)
		if expiresIn > 0 {
			if expiresIn < time.Minute {
				constraints = append(constraints, fmt.Sprintf("expires = %ds", int(expiresIn.Seconds())))
			} else if expiresIn < time.Hour {
				constraints = append(constraints, fmt.Sprintf("expires = %dm", int(expiresIn.Minutes())))
			} else {
				constraints = append(constraints, fmt.Sprintf("expires = %dh", int(expiresIn.Hours())))
			}
		} else {
			constraints = append(constraints, "expires = expired")
		}

		// Determine last seen from usage
		lastSeen := "never"
		if usage, ok := s.usage[sig]; ok {
			ago := time.Since(usage.LastUsed)
			if ago < time.Second {
				lastSeen = "just now"
			} else if ago < time.Minute {
				lastSeen = fmt.Sprintf("%ds ago", int(ago.Seconds()))
			} else if ago < time.Hour {
				lastSeen = fmt.Sprintf("%dm ago", int(ago.Minutes()))
			} else {
				lastSeen = fmt.Sprintf("%dh ago", int(ago.Hours()))
			}
		}

		// Determine label
		label := token.Label
		if label == "" {
			switch token.Depth {
			case 0:
				label = "Root Token"
			case 1:
				label = "Agent Token"
			default:
				label = "Worker Token"
			}
		}

		nodes = append(nodes, GraphNode{
			Data: GraphNodeData{
				ID:          sig,
				Label:       label,
				Constraints: constraints,
				LastSeen:    lastSeen,
				Depth:       token.Depth,
				Status:      status,
			},
		})

		// Add edge from parent if exists
		if token.ParentSig != "" {
			edges = append(edges, GraphEdge{
				Data: GraphEdgeData{
					Source: token.ParentSig,
					Target: sig,
				},
			})
		}
	}

	// Get counters
	s.counterMu.Lock()
	blocked := s.blockedRequests
	bannedHits := s.bannedHits
	s.counterMu.Unlock()

	// Count active and banned tokens
	activeCount := 0
	bannedCount := 0
	for sig := range s.minted {
		if _, banned := s.banned[sig]; banned {
			bannedCount++
		} else {
			activeCount++
		}
	}

	return GraphResponse{
		Nodes: nodes,
		Edges: edges,
		Stats: GraphStats{
			Active:     activeCount,
			Blocked:    blocked,
			Banned:     bannedCount,
			BannedHits: bannedHits,
		},
	}
}

// Reset clears all governance data
func (s *Service) Reset() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.banned = make(map[string]BanRecord)
	s.usage = make(map[string]*UsageStats)
	s.minted = make(map[string]*MintedToken)

	// Reset counters
	s.counterMu.Lock()
	s.blockedRequests = 0
	s.bannedHits = 0
	s.counterMu.Unlock()

	// Reset store
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := s.store.Reset(ctx); err != nil {
		log.Error().Err(err).Msg("Failed to reset store")
	}
}

// Close closes the backing store
func (s *Service) Close() error {
	return s.store.Close()
}

// GetStore returns the underlying store for advanced operations (e.g., rate limiting)
func (s *Service) GetStore() Store {
	return s.store
}

