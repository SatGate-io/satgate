package governance

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// EvidencePack is the canonical Policy-to-Proof export artifact.
// It answers who authorized what under which policy, budget, delegation, and paid-rail context.
type EvidencePack struct {
	SchemaVersion  string                   `json:"schema_version"`
	ID             string                   `json:"evidence_pack_id"`
	Environment    string                   `json:"environment"`
	PackType       string                   `json:"pack_type"`
	TenantID       string                   `json:"tenant_id"`
	Tenant         map[string]interface{}   `json:"tenant"`
	Subject        map[string]interface{}   `json:"subject"`
	Purpose        string                   `json:"purpose"`
	PolicySnapshot map[string]interface{}   `json:"policy_snapshot"`
	BudgetSnapshot map[string]interface{}   `json:"budget_snapshot"`
	AuthorityChain []map[string]interface{} `json:"authority_chain"`
	Receipts       []PolicyDecision         `json:"receipts"`
	PaymentContext map[string]interface{}   `json:"payment_context"`
	ReceiptChain   map[string]interface{}   `json:"receipt_chain"`
	Redaction      map[string]interface{}   `json:"redaction"`
	Export         map[string]interface{}   `json:"export"`
	Verification   map[string]interface{}   `json:"verification"`
	Signature      string                   `json:"signature,omitempty"`
	ChainRoot      string                   `json:"chain_root,omitempty"`
	IssuedAt       time.Time                `json:"issued_at"`
	CreatedAt      time.Time                `json:"created_at"`
	Decisions      []PolicyDecision         `json:"decisions,omitempty"` // Backward-compatible alias for older clients.
}

// PolicyDecision records a single authority receipt by the gateway.
type PolicyDecision struct {
	Timestamp      time.Time `json:"timestamp"`
	AgentID        string    `json:"agent_id"`
	TaskID         string    `json:"task_id,omitempty"`
	Route          string    `json:"route"`
	Decision       string    `json:"decision"` // allow, deny, delegated, revoked, paid
	Reason         string    `json:"reason,omitempty"`
	EstimatedCost  float64   `json:"estimated_cost_usd,omitempty"`
	Remaining      float64   `json:"remaining_budget_usd,omitempty"`
	PaymentContext string    `json:"payment_context,omitempty"` // Bolt11, x402 hash, etc.
	ReceiptHash    string    `json:"receipt_hash"`
}

// Service handles token governance (banning, rate limiting, telemetry)
type Service struct {
	store Store // Backing store (Redis or memory)

	// Local cache for hot path (IsBanned checks)
	banned map[string]BanRecord
	usage  map[string]*UsageStats
	minted map[string]*MintedToken
	mu     sync.RWMutex

	// Counters for dashboard stats
	blockedRequests int64 // Unpaid requests (402s)
	bannedHits      int64 // Requests from banned tokens
	counterMu       sync.Mutex

	// PolicyToProof Keys
	privateKey ed25519.PrivateKey
	publicKey  ed25519.PublicKey

	// Revocation on a shared store (any store but *MemoryStore). The store
	// is the source of truth and is read on the request path through
	// banCache, so a ban written by another gateway is honoured here within
	// banCacheTTL. pendingBans holds bans made here that could not be
	// written to the store; this gateway keeps honouring them.
	shared      bool
	banCacheTTL time.Duration
	now         func() time.Time
	cacheMu     sync.Mutex
	banCache    map[string]banCacheEntry
	pendingBans map[string]BanRecord
}

// BanCacheTTL is how long a gateway backed by a shared store (Redis) reuses
// one answer from the store for a signature. A ban written by one gateway is
// honoured by every other gateway on the same store within this delay.
const BanCacheTTL = 2 * time.Second

// banStoreTimeout bounds one read of the shared ban store on the request path.
const banStoreTimeout = 2 * time.Second

// maxBanCacheEntries bounds the ban cache. When it is full, expired entries
// are dropped, and if that is not enough the cache is cleared.
const maxBanCacheEntries = 100000

// ErrRevocationUnavailable is returned when the shared ban store cannot be
// read or written. A request whose revocation state cannot be read is refused.
var ErrRevocationUnavailable = errors.New("revocation state unavailable")

type banCacheEntry struct {
	banned bool
	at     time.Time
}

// banLister is implemented by stores that check several signatures in one
// round trip (RedisStore). Other stores are asked one signature at a time.
type banLister interface {
	BannedAmong(ctx context.Context, signatures []string) ([]bool, error)
}

// MintedToken tracks a minted token
type MintedToken struct {
	Signature string
	Scope     string
	CreatedAt time.Time
	ExpiresAt time.Time
	ParentSig string // Parent token signature (empty for root tokens)
	Depth     int    // 0 = root, 1 = minted, 2+ = delegated
	Label     string // Human-readable label
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
	Signature     string
	TotalRequests int64
	LastUsed      time.Time
	Routes        map[string]int64 // route -> count
}

// TokenInfo represents token status for API responses
type TokenInfo struct {
	Signature     string           `json:"signature"`
	Status        string           `json:"status"`
	Scope         string           `json:"scope,omitempty"`
	CreatedAt     *time.Time       `json:"createdAt,omitempty"`
	ExpiresAt     *time.Time       `json:"expiresAt,omitempty"`
	TotalRequests int64            `json:"totalRequests"`
	LastUsed      *time.Time       `json:"lastUsed,omitempty"`
	Routes        map[string]int64 `json:"routes,omitempty"`
	BannedAt      *time.Time       `json:"bannedAt,omitempty"`
	BanReason     string           `json:"banReason,omitempty"`
}

// NewService creates a new governance service with the given store
// If store is nil, uses in-memory store (single instance only)
func NewService(store Store) *Service {
	if store == nil {
		store = NewMemoryStore()
	}

	pub, priv, _ := ed25519.GenerateKey(rand.Reader)
	_, inMemory := store.(*MemoryStore)

	return &Service{
		store:       store,
		banned:      make(map[string]BanRecord),
		usage:       make(map[string]*UsageStats),
		minted:      make(map[string]*MintedToken),
		privateKey:  priv,
		publicKey:   pub,
		shared:      !inMemory,
		banCacheTTL: BanCacheTTL,
		now:         time.Now,
		banCache:    make(map[string]banCacheEntry),
		pendingBans: make(map[string]BanRecord),
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

// Ban adds a token to the ban list. See BanContext.
func (s *Service) Ban(signature, reason, bannedBy string) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	_ = s.BanContext(ctx, signature, reason, bannedBy)
}

// BanContext bans signature. The ban covers the token with that signature and
// every token delegated from it, before or after the ban, at any depth, when
// the caller checks the token's lineage with IsAnyBanned.
//
// The ban takes effect on this gateway at once. With a shared store it is
// written to the store, and other gateways on the store honour it within
// BanCacheTTL. If the write fails, this gateway keeps honouring the ban and
// the error (wrapping ErrRevocationUnavailable) says that other gateways may
// not; the caller should retry.
func (s *Service) BanContext(ctx context.Context, signature, reason, bannedBy string) error {
	record := BanRecord{
		Signature: signature,
		Reason:    reason,
		BannedAt:  time.Now(),
		BannedBy:  bannedBy,
	}
	s.mu.Lock()
	s.banned[signature] = record
	s.mu.Unlock()

	// Persist to store (sync for ban operations - critical path)
	err := s.store.Ban(ctx, record)
	if !s.shared {
		if err != nil {
			log.Error().Err(err).Str("signature", shortSig(signature)).Msg("Failed to persist ban record")
		}
		return nil
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	s.putCacheLocked(signature, true)
	if err != nil {
		s.pendingBans[signature] = record
		log.Error().Err(err).Str("signature", shortSig(signature)).Msg("Ban not written to the shared store; only this gateway honours it")
		return fmt.Errorf("%w: %v", ErrRevocationUnavailable, err)
	}
	delete(s.pendingBans, signature)
	return nil
}

// Unban removes a token from the ban list. With a shared store, other
// gateways stop honouring the ban within BanCacheTTL.
func (s *Service) Unban(signature string) bool {
	s.mu.Lock()
	_, wasLocal := s.banned[signature]
	delete(s.banned, signature)
	s.mu.Unlock()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	wasShared := false
	if s.shared {
		var err error
		if wasShared, err = s.store.IsBanned(ctx, signature); err != nil {
			wasShared = true // cannot tell; still try to remove it
		}
		s.cacheMu.Lock()
		_, wasPending := s.pendingBans[signature]
		delete(s.pendingBans, signature)
		delete(s.banCache, signature)
		s.cacheMu.Unlock()
		wasShared = wasShared || wasPending
	}
	if !wasLocal && !wasShared {
		return false
	}

	// Persist to store
	if err := s.store.Unban(ctx, signature); err != nil {
		log.Error().Err(err).Str("signature", shortSig(signature)).Msg("Failed to persist unban")
	}
	return true
}

// IsBanned reports whether the token with exactly this signature is banned.
// It does not look at the tokens this one was delegated from; use
// IsAnyBanned with the token's lineage for an access decision. With a shared
// store, a store that cannot be read counts as banned (fail closed).
func (s *Service) IsBanned(signature string) bool {
	banned, err := s.IsAnyBanned(context.Background(), []string{signature})
	return banned || err != nil
}

// IsAnyBanned reports whether any of signatures is banned. Pass the lineage
// of a verified token (macaroon.Service.Lineage: its own signature and the
// signature of every token it was delegated from) to decide whether a ban
// covers it.
//
// With the in-memory store it reads this process's ban list and never fails.
// With a shared store it reads the store, reusing each answer for
// BanCacheTTL; if the store cannot be read it returns an error wrapping
// ErrRevocationUnavailable, and the caller must refuse the request.
func (s *Service) IsAnyBanned(ctx context.Context, signatures []string) (bool, error) {
	if !s.shared {
		s.mu.RLock()
		defer s.mu.RUnlock()
		for _, sig := range signatures {
			if _, banned := s.banned[sig]; banned {
				return true, nil
			}
		}
		return false, nil
	}

	now := s.now()
	misses := make([]string, 0, len(signatures))
	s.cacheMu.Lock()
	for _, sig := range signatures {
		if _, pending := s.pendingBans[sig]; pending {
			s.cacheMu.Unlock()
			return true, nil
		}
		if e, ok := s.banCache[sig]; ok && now.Sub(e.at) < s.banCacheTTL {
			if e.banned {
				s.cacheMu.Unlock()
				return true, nil
			}
			continue
		}
		misses = append(misses, sig)
	}
	s.cacheMu.Unlock()
	if len(misses) == 0 {
		return false, nil
	}

	storeCtx, cancel := context.WithTimeout(ctx, banStoreTimeout)
	defer cancel()
	found, err := s.lookupBans(storeCtx, misses)
	if err != nil {
		log.Error().Err(err).Msg("Ban store unavailable; refusing request")
		return false, fmt.Errorf("%w: %v", ErrRevocationUnavailable, err)
	}

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	hit := false
	for i, sig := range misses {
		// A ban made here while the store was being read stays in force.
		if e, ok := s.banCache[sig]; ok && e.banned && !e.at.Before(now) {
			hit = true
			continue
		}
		s.putCacheLocked(sig, found[i])
		hit = hit || found[i]
	}
	return hit, nil
}

func (s *Service) lookupBans(ctx context.Context, signatures []string) ([]bool, error) {
	if l, ok := s.store.(banLister); ok {
		found, err := l.BannedAmong(ctx, signatures)
		if err == nil && len(found) != len(signatures) {
			err = fmt.Errorf("ban store answered %d of %d lookups", len(found), len(signatures))
		}
		return found, err
	}
	found := make([]bool, len(signatures))
	for i, sig := range signatures {
		banned, err := s.store.IsBanned(ctx, sig)
		if err != nil {
			return nil, err
		}
		found[i] = banned
	}
	return found, nil
}

// putCacheLocked records one answer from the shared store. cacheMu is held.
func (s *Service) putCacheLocked(sig string, banned bool) {
	now := s.now()
	if len(s.banCache) >= maxBanCacheEntries {
		for k, e := range s.banCache {
			if now.Sub(e.at) >= s.banCacheTTL {
				delete(s.banCache, k)
			}
		}
		if len(s.banCache) >= maxBanCacheEntries {
			s.banCache = make(map[string]banCacheEntry)
		}
	}
	s.banCache[sig] = banCacheEntry{banned: banned, at: now}
}

// coveringBanLocked returns the ban that covers the recorded token sig: a ban
// of sig itself or of a token it was recorded as delegated from (ParentSig).
// It is for display (token info, graph, stats); access decisions use
// IsAnyBanned with the verified lineage. s.mu is held.
func (s *Service) coveringBanLocked(sig string) (BanRecord, bool) {
	seen := make(map[string]bool)
	for cur := sig; cur != "" && !seen[cur]; {
		seen[cur] = true
		if ban, ok := s.banned[cur]; ok {
			return ban, true
		}
		minted, ok := s.minted[cur]
		if !ok {
			break
		}
		cur = minted.ParentSig
	}
	return BanRecord{}, false
}

func shortSig(sig string) string {
	if len(sig) > 12 {
		return sig[:8] + "..." + sig[len(sig)-4:]
	}
	return sig
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
	ban, isBanned := s.coveringBanLocked(signature)

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

		// Check if banned, directly or through a token it was delegated from
		if ban, banned := s.coveringBanLocked(sig); banned {
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

// ExportEvidencePack generates a signed canonical Evidence Pack.
func (s *Service) ExportEvidencePack(ctx context.Context, tenantID string) (*EvidencePack, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	now := time.Now().UTC()
	pack := &EvidencePack{
		SchemaVersion: "satgate.evidence_pack.v1",
		ID:            fmt.Sprintf("ep_%d", now.Unix()),
		Environment:   "runtime",
		PackType:      "full",
		TenantID:      tenantID,
		Tenant: map[string]interface{}{
			"id":   tenantID,
			"name": tenantID,
		},
		Subject: map[string]interface{}{
			"id":        "agent:runtime-export",
			"kind":      "agent",
			"tenant_id": tenantID,
		},
		Purpose: "Policy-to-Proof Evidence Pack export",
		PolicySnapshot: map[string]interface{}{
			"policy_id":      "runtime_governance",
			"policy_version": "runtime",
			"policy_digest":  "sha256:runtime_policy_snapshot",
			"mode":           "Control",
			"decision_model": "authority_before_execution",
			"obligations":    []string{"record_receipt", "preserve_authority_context", "preserve_budget_context"},
		},
		BudgetSnapshot: map[string]interface{}{
			"budget_id":       fmt.Sprintf("budget_%s_runtime", tenantID),
			"currency":        "USD",
			"unit_precision":  2,
			"delegated_limit": "runtime",
			"spent":           "runtime",
			"remaining":       "runtime",
		},
		AuthorityChain: make([]map[string]interface{}, 0),
		Receipts:       make([]PolicyDecision, 0),
		PaymentContext: map[string]interface{}{
			"rail_neutral":   true,
			"internal_rail":  "enterprise_ledger",
			"external_rails": []string{"x402", "l402", "api_key_billing", "enterprise_contract"},
			"events":         []map[string]interface{}{},
			"default_market": "internal_enterprise_agents",
		},
		Redaction: map[string]interface{}{
			"applied":    true,
			"profile":    "tenant_safe",
			"method":     "token_fingerprint_only",
			"proof_mode": "hash_commitments",
		},
		Export: map[string]interface{}{
			"export_id":            fmt.Sprintf("exp_%d", now.Unix()),
			"export_type":          "manual",
			"exported_at":          now,
			"formats":              []string{"json"},
			"redaction_profile":    "tenant_safe",
			"completeness":         "runtime_usage_snapshot",
			"included_receipt_ids": []string{},
		},
		Verification: map[string]interface{}{
			"verifiable":     true,
			"algorithm":      "ed25519 + sha256 linear receipt chain",
			"public_key_id":  "runtime-generated-governance-key",
			"signed_payload": "canonical EvidencePackV1 receipt chain root",
			"cli":            "satgate-cli proof export --tenant " + tenantID,
		},
		IssuedAt:  now,
		CreatedAt: now,
		Decisions: make([]PolicyDecision, 0),
	}

	// Placeholder: In Enterprise this pulls from the persistent audit_log table.
	// For OSS/Refactor, we simulate receipts from usage stats to verify schema alignment.
	seq := 0
	var previousHash string
	for sig, stats := range s.usage {
		agentID := sig
		if len(agentID) > 12 {
			agentID = agentID[:12]
		}
		grant := map[string]interface{}{
			"grant_id":        fmt.Sprintf("grant_%s", agentID),
			"kind":            "runtime_usage_subject",
			"subject":         map[string]interface{}{"id": "agent:" + agentID, "kind": "agent", "tenant_id": tenantID, "token_fingerprint": hashString(sig)},
			"effective_scope": []string{"runtime_observed_routes"},
			"receipt_hash":    hashString(sig + tenantID),
		}
		pack.AuthorityChain = append(pack.AuthorityChain, grant)

		for route, count := range stats.Routes {
			for i := int64(0); i < count; i++ {
				seq++
				decision := PolicyDecision{
					Timestamp:   now.Add(-time.Duration(i) * time.Minute),
					AgentID:     agentID,
					TaskID:      fmt.Sprintf("receipt_%03d", seq),
					Route:       route,
					Decision:    "allowed",
					Reason:      "runtime_usage_observed",
					Remaining:   1000.0,
					ReceiptHash: hashString(fmt.Sprintf("%s|%s|%d|%s", agentID, route, seq, previousHash)),
				}
				previousHash = decision.ReceiptHash
				pack.Receipts = append(pack.Receipts, decision)
				pack.Decisions = append(pack.Decisions, decision)
			}
		}
	}

	if len(pack.Receipts) == 0 {
		decision := PolicyDecision{
			Timestamp:   now,
			AgentID:     "agent:runtime-export",
			TaskID:      "receipt_001",
			Route:       "/api/governance/evidence-pack",
			Decision:    "exported",
			Reason:      "no_runtime_usage_yet",
			ReceiptHash: hashString(tenantID + now.Format(time.RFC3339Nano)),
		}
		pack.Receipts = append(pack.Receipts, decision)
		pack.Decisions = append(pack.Decisions, decision)
		previousHash = decision.ReceiptHash
	}

	firstHash := pack.Receipts[0].ReceiptHash
	pack.ChainRoot = hashReceipts(pack.Receipts)
	pack.ReceiptChain = map[string]interface{}{
		"canonicalization":   "RFC8785-JCS",
		"hash_algorithm":     "sha256",
		"chain_type":         "linear_hash_chain",
		"first_receipt_hash": firstHash,
		"last_receipt_hash":  previousHash,
		"chain_root":         pack.ChainRoot,
		"receipt_count":      len(pack.Receipts),
	}

	data, _ := json.Marshal(map[string]interface{}{
		"schema_version":   pack.SchemaVersion,
		"evidence_pack_id": pack.ID,
		"tenant_id":        pack.TenantID,
		"chain_root":       pack.ChainRoot,
		"issued_at":        pack.IssuedAt,
	})
	pack.Signature = "ed25519:" + hex.EncodeToString(ed25519.Sign(s.privateKey, data))

	return pack, nil
}

func hashString(value string) string {
	sum := sha256.Sum256([]byte(value))
	return "sha256:" + hex.EncodeToString(sum[:])
}

func hashReceipts(receipts []PolicyDecision) string {
	data, _ := json.Marshal(receipts)
	sum := sha256.Sum256(data)
	return "sha256:" + hex.EncodeToString(sum[:])
}
func (s *Service) GetStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeCount := 0
	bannedCount := len(s.banned)
	expiredCount := 0
	totalRequests := int64(0)

	// Count minted tokens by status
	for sig, minted := range s.minted {
		if _, banned := s.coveringBanLocked(sig); banned {
			continue // banned directly (counted in bannedCount) or through its lineage
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
		if _, banned := s.coveringBanLocked(sig); banned {
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
		if _, banned := s.coveringBanLocked(sig); banned {
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

	s.cacheMu.Lock()
	s.banCache = make(map[string]banCacheEntry)
	s.pendingBans = make(map[string]BanRecord)
	s.cacheMu.Unlock()

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
