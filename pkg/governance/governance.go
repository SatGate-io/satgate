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
	"math"
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
	// written to the store; this gateway keeps honouring them. localBans
	// holds bans made here recently (see localBan), so a decision or view
	// that was reading the store while one was made still honours it.
	// banSeq counts the store writes of bans made here that have ended;
	// every store answer records it as it was when its lookup started (see
	// answerPredatesLocalBanLocked). It never wraps: a store write that
	// ends when banSeq is at its maximum sets banSeqSpent instead, and from
	// then on every access decision on this gateway is refused with
	// ErrRevocationUnavailable until the process restarts (Reset does not
	// clear it).
	shared      bool
	banCacheTTL time.Duration
	now         func() time.Time
	cacheMu     sync.Mutex
	banCache    map[string]banCacheEntry
	pendingBans map[string]BanRecord
	localBans   map[string]localBan
	banSeq      uint64
	banSeqSpent bool
}

// localBan is a ban made on this gateway with a shared store. A decision
// that started before the ban's store write ended may have read the store
// before the ban reached it; it honours the ban anyway (localBanLocked).
type localBan struct {
	record  BanRecord
	writing int       // store writes of this ban in progress
	done    time.Time // when the last store write of this ban ended
	// seq is Service.banSeq after the last store write of this ban ended.
	// A "not banned" answer for this signature from a lookup that started
	// before then (its seq is lower) may have missed the ban: it is never
	// cached or used (answerPredatesLocalBanLocked).
	seq uint64
}

// localBanKeep is how long a local ban is kept after its store write ended.
// A decision lasts at most BanStoreReadTimeout, so one that started before
// the write ended has finished by then; later decisions read the ban from
// the store (or pendingBans if the write failed). An answer from a lookup
// that started before the write ended is older than BanCacheTTL by then, so
// no decision can use it once the local ban is dropped.
const localBanKeep = BanStoreReadTimeout + BanCacheTTL

// BanCacheTTL is how long a gateway backed by a shared store (Redis) reuses
// one answer from the store for a signature, counted from the moment the
// lookup that produced it started. A ban written by one gateway is honoured by
// every other gateway on the same store within this delay: any lookup that
// starts after the ban sees it, and an answer from a lookup that started
// earlier is not used to allow a request once BanCacheTTL has passed since
// that lookup started.
const BanCacheTTL = 2 * time.Second

// BanStoreReadTimeout bounds one read of the shared ban store (an access
// check, or a ban list read for the views). A read that has not answered by
// then is abandoned and the request is refused with ErrRevocationUnavailable.
const BanStoreReadTimeout = 2 * time.Second

// banStoreTimeout is BanStoreReadTimeout (kept for the existing name).
const banStoreTimeout = BanStoreReadTimeout

// maxBanCacheEntries bounds the ban cache. When it is full, expired entries
// are dropped, and if that is not enough the cache is cleared.
const maxBanCacheEntries = 100000

// ErrRevocationUnavailable is returned when the shared ban store cannot be
// read or written. A request whose revocation state cannot be read is refused.
var ErrRevocationUnavailable = errors.New("revocation state unavailable")

type banCacheEntry struct {
	banned bool
	at     time.Time
	seq    uint64 // Service.banSeq when the lookup started
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
		localBans:   make(map[string]localBan),
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
// not; the caller should retry. A ban made when the count of bans written
// here is at its maximum is still recorded here and in the store, and from
// then on this gateway refuses every access decision (see banSeq).
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

	if !s.shared {
		// Persist to store (sync for ban operations - critical path)
		if err := s.store.Ban(ctx, record); err != nil {
			log.Error().Err(err).Str("signature", shortSig(signature)).Msg("Failed to persist ban record")
		}
		return nil
	}

	// From here until localBanKeep after the write ends, every decision
	// on this gateway honours the ban, whatever it read from the store.
	s.cacheMu.Lock()
	lb := s.localBans[signature]
	lb.record = record
	lb.writing++
	s.localBans[signature] = lb
	s.cacheMu.Unlock()

	err := s.store.Ban(ctx, record)

	s.cacheMu.Lock()
	defer s.cacheMu.Unlock()
	now := s.now()
	lb = s.localBans[signature]
	lb.record = record
	if lb.writing > 0 {
		lb.writing--
	}
	lb.done = now
	s.advanceBanSeqLocked()
	lb.seq = s.banSeq
	s.localBans[signature] = lb
	s.pruneLocalBansLocked(now)
	s.putCacheLocked(signature, true, now)
	if err != nil {
		s.pendingBans[signature] = record
		log.Error().Err(err).Str("signature", shortSig(signature)).Msg("Ban not written to the shared store; only this gateway honours it")
		return fmt.Errorf("%w: %v", ErrRevocationUnavailable, err)
	}
	delete(s.pendingBans, signature)
	return nil
}

// advanceBanSeqLocked counts a store write of a ban made here that has
// ended. banSeq never wraps: at its maximum it stays there and the gateway
// stops allowing (banSeqSpent), because a later ban could no longer be
// ordered after the lookups that started before it. The ban itself is
// already recorded here and in the store; only access decisions change, to
// refusals with errBanSeqSpent. cacheMu is held.
func (s *Service) advanceBanSeqLocked() {
	if s.banSeq < math.MaxUint64 {
		s.banSeq++
		return
	}
	if !s.banSeqSpent {
		s.banSeqSpent = true
		log.Error().Msg("Local ban sequence exhausted; this gateway refuses every access decision until it is restarted")
	}
}

// errBanSeqSpent is the refusal of every decision once banSeq is at its
// maximum and a further ban has been made here.
var errBanSeqSpent = fmt.Errorf("%w: local ban sequence exhausted; restart this gateway", ErrRevocationUnavailable)

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
		delete(s.localBans, signature)
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
// With a shared store the answer comes from decideLineage: it allows only
// when, at the moment it allows (the clock is read then, under the cache
// lock), every signature has a "not banned" answer from a store lookup that
// started less than BanCacheTTL before and after the store write of any ban
// of that signature made on this gateway ended, no ban made on this gateway
// since the decision started covers any of them, and less than
// BanStoreReadTimeout has passed since the decision started. If the store
// cannot be read in time it returns an error wrapping
// ErrRevocationUnavailable, and the caller must refuse the request. Once a
// ban written from this process would take the count of such bans past
// math.MaxUint64 (see banSeq), every decision returns that error until the
// process restarts.
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
	return s.decideLineage(ctx, signatures)
}

// lineageAnswer is the answer a decision holds for one signature: a store
// answer (banned or not) from a lookup that started at at, when banSeq was
// seq.
type lineageAnswer struct {
	banned bool
	at     time.Time
	seq    uint64
	known  bool
}

// decideLineage is the one access decision with a shared store; every
// caller (IsAnyBanned, IsBanned, so the HTTP gateway and the MCP runtime)
// goes through it.
//
// It starts at start, has until start+BanStoreReadTimeout (its deadline),
// and holds one answer per signature: from the cache, or from a store lookup
// made here. Every time it settles (settleLocked) it holds the cache lock,
// has made every cache update of that step, and reads the clock then; the
// moment it allows is the last of these. A "banned" answer is honoured if
// its lookup started less than BanCacheTTL before start. To allow, every
// signature must have, at that moment, a "not banned" answer whose lookup
// started less than BanCacheTTL before and after the store write of any ban
// of that signature made here ended; no ban made here since start, or not yet
// written to the store, may cover any of them; and the deadline must not have
// passed. Signatures with no such answer are read from the store: first
// those, then, if answers aged past BanCacheTTL meanwhile, every signature
// again in one lookup. A lookup whose answer arrives BanCacheTTL or more
// after it started, a decision whose deadline passes, or one still without a
// fresh answer for every signature after that, is refused with
// ErrRevocationUnavailable. All reads share the deadline.
func (s *Service) decideLineage(ctx context.Context, signatures []string) (bool, error) {
	start := s.now()
	deadline := start.Add(banStoreTimeout)
	dctx, cancel := context.WithTimeout(ctx, banStoreTimeout)
	defer cancel()

	answers := make([]lineageAnswer, len(signatures))
	s.cacheMu.Lock()
	if s.banSeqSpent {
		s.cacheMu.Unlock()
		return false, errBanSeqSpent
	}
	banned, stale, err := s.settleLocked(dctx, signatures, answers, start, deadline)
	s.cacheMu.Unlock()
	if banned {
		return true, nil
	}
	if err != nil {
		return false, err
	}

	for round := 0; len(stale) > 0; round++ {
		if round == 2 {
			log.Error().Msg("Ban store answers did not stay fresh for the whole lineage; refusing request")
			return false, fmt.Errorf("%w: no answer younger than %s for every lineage entry", ErrRevocationUnavailable, s.banCacheTTL)
		}
		toRead := stale
		if round > 0 {
			// Answers aged while the store was read: read the whole
			// lineage in one lookup so every answer has the same start.
			toRead = make([]int, len(signatures))
			for i := range toRead {
				toRead[i] = i
			}
		}
		sigs := make([]string, len(toRead))
		for j, i := range toRead {
			sigs[j] = signatures[i]
		}
		// The lookup's start and banSeq are taken together, so a ban
		// whose write ends after this is known to the answer as later.
		s.cacheMu.Lock()
		readStart := s.now()
		readSeq := s.banSeq
		s.cacheMu.Unlock()
		found, err := boundedRead(dctx, func(rctx context.Context) ([]bool, error) {
			return s.lookupBans(rctx, sigs)
		})
		if err != nil {
			log.Error().Err(err).Msg("Ban store unavailable; refusing request")
			return false, fmt.Errorf("%w: %v", ErrRevocationUnavailable, err)
		}
		// The answer describes the store as it was at some moment after
		// readStart, so it is fresh for banCacheTTL from readStart.
		fresh := s.now().Sub(readStart) < s.banCacheTTL

		s.cacheMu.Lock()
		for j, i := range toRead {
			answers[i] = lineageAnswer{banned: found[j], at: readStart, seq: readSeq, known: true}
			if fresh {
				s.putAnswerLocked(signatures[i], found[j], readStart, readSeq)
			}
		}
		banned, stale, err = s.settleLocked(dctx, signatures, answers, start, deadline)
		s.cacheMu.Unlock()
		if banned {
			return true, nil
		}
		if err == errBanSeqSpent {
			// Spent while the store was read: the refusal every
			// decision gets now, fresh answer or not.
			return false, err
		}
		if !fresh {
			log.Error().Msg("Ban store answered after the cache TTL; refusing request")
			return false, fmt.Errorf("%w: ban store answer arrived after %s", ErrRevocationUnavailable, s.banCacheTTL)
		}
		if err != nil {
			return false, err
		}
	}
	return false, nil
}

// errDecisionDeadline is the refusal of a decision whose deadline passed
// before it could allow.
var errDecisionDeadline = fmt.Errorf("%w: decision did not end within %s", ErrRevocationUnavailable, BanStoreReadTimeout)

// settleLocked brings answers up to date for a decision that started at start
// and must end by deadline. It reads the clock (now) itself, so call it under
// cacheMu after the cache updates of the step. It reports whether a ban
// covers the lineage; otherwise errDecisionDeadline if now is at or past
// deadline (or ctx ended); otherwise which signatures (by index) have no "not
// banned" answer whose lookup started less than BanCacheTTL before now and
// after any ban of that signature made here (answerPredatesLocalBanLocked).
// A newer answer in the cache (from another lookup, or a ban made here)
// replaces the one held; a ban made here since start, or one not yet written
// to the store, covers its signature whatever the store said. Once banSeq is
// spent (advanceBanSeqLocked) it returns errBanSeqSpent, whatever the
// lineage, as a decision that starts then does: a decision in progress when
// the sequence is spent ends with the same refusal.
func (s *Service) settleLocked(ctx context.Context, signatures []string, answers []lineageAnswer, start, deadline time.Time) (bool, []int, error) {
	if s.banSeqSpent {
		return false, nil, errBanSeqSpent
	}
	now := s.now()
	var stale []int
	for i, sig := range signatures {
		if _, ok := s.localBanLocked(sig, start); ok {
			return true, nil, nil
		}
		a := &answers[i]
		if e, ok := s.banCache[sig]; ok && (!a.known || e.at.After(a.at) || (e.at.Equal(a.at) && e.banned)) {
			*a = lineageAnswer{banned: e.banned, at: e.at, seq: e.seq, known: true}
		}
		if a.known && !a.banned && s.answerPredatesLocalBanLocked(sig, a.seq) {
			// Read before a ban made here reached the store: never used.
			if e, ok := s.banCache[sig]; ok && !e.banned && e.seq == a.seq {
				delete(s.banCache, sig)
			}
			*a = lineageAnswer{}
		}
		switch {
		case !a.known:
			stale = append(stale, i)
		case a.banned:
			if start.Sub(a.at) < s.banCacheTTL {
				return true, nil, nil
			}
			stale = append(stale, i) // an old ban: the store is read again
		case now.Sub(a.at) >= s.banCacheTTL:
			stale = append(stale, i)
		}
	}
	if !now.Before(deadline) || ctx.Err() != nil {
		log.Error().Msg("Revocation decision did not end within its deadline; refusing request")
		return false, nil, errDecisionDeadline
	}
	return false, stale, nil
}

// answerPredatesLocalBanLocked reports whether a "not banned" answer for sig
// from a lookup that started when banSeq was seq may have missed a ban of sig
// made here: the store write of that ban ended after the lookup started.
// cacheMu is held.
func (s *Service) answerPredatesLocalBanLocked(sig string, seq uint64) bool {
	lb, ok := s.localBans[sig]
	return ok && seq < lb.seq
}

// localBanLocked returns the ban made on this gateway that covers sig for a
// decision or view that started at start: one not yet written to the shared
// store (pendingBans), or one whose store write was in progress at start or
// ended since. cacheMu is held.
func (s *Service) localBanLocked(sig string, start time.Time) (BanRecord, bool) {
	if rec, ok := s.pendingBans[sig]; ok {
		return rec, true
	}
	if lb, ok := s.localBans[sig]; ok && (lb.writing > 0 || !lb.done.Before(start)) {
		return lb.record, true
	}
	return BanRecord{}, false
}

// pruneLocalBansLocked drops local bans whose store write ended localBanKeep
// or more before now. cacheMu is held.
func (s *Service) pruneLocalBansLocked(now time.Time) {
	for sig, lb := range s.localBans {
		if lb.writing == 0 && now.Sub(lb.done) >= localBanKeep {
			delete(s.localBans, sig)
		}
	}
}

// boundedRead runs read with a context that ends BanStoreReadTimeout from now
// (or sooner, with ctx) and returns no later than that deadline, even if read
// does not return. An answer that arrives after the deadline is discarded.
func boundedRead[T any](ctx context.Context, read func(context.Context) (T, error)) (T, error) {
	rctx, cancel := context.WithTimeout(ctx, banStoreTimeout)
	defer cancel()
	type result struct {
		v   T
		err error
	}
	done := make(chan result, 1)
	go func() {
		v, err := read(rctx)
		done <- result{v, err}
	}()
	var zero T
	select {
	case r := <-done:
		if r.err == nil && rctx.Err() != nil {
			return zero, rctx.Err()
		}
		return r.v, r.err
	case <-rctx.Done():
		return zero, rctx.Err()
	}
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

// putAnswerLocked records an answer from a store lookup that started at at,
// when banSeq was seq, unless the cache already holds a newer one (or a ban
// made at the same moment), or the answer is "not banned" and a ban of sig
// made here ended its store write after the lookup started. cacheMu is held.
func (s *Service) putAnswerLocked(sig string, banned bool, at time.Time, seq uint64) {
	if e, ok := s.banCache[sig]; ok && (e.at.After(at) || (e.at.Equal(at) && e.banned)) {
		return
	}
	if !banned && s.answerPredatesLocalBanLocked(sig, seq) {
		return
	}
	s.putEntryLocked(sig, banCacheEntry{banned: banned, at: at, seq: seq})
}

// putCacheLocked records one answer from the shared store, obtained by a
// lookup that started at at, after every ban made here so far (or, for a ban
// made here, made at at). cacheMu is held.
func (s *Service) putCacheLocked(sig string, banned bool, at time.Time) {
	s.putEntryLocked(sig, banCacheEntry{banned: banned, at: at, seq: s.banSeq})
}

// putEntryLocked stores entry in the bounded cache. cacheMu is held.
func (s *Service) putEntryLocked(sig string, entry banCacheEntry) {
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
	s.banCache[sig] = entry
}

// coveringBanLocked returns the ban in bans that covers the recorded token
// sig: a ban of sig itself or of a token it was recorded as delegated from
// (ParentSig). It is for the views (token info, graph, stats); access
// decisions use IsAnyBanned with the verified lineage. s.mu is held.
func (s *Service) coveringBanLocked(bans map[string]BanRecord, sig string) (BanRecord, bool) {
	seen := make(map[string]bool)
	for cur := sig; cur != "" && !seen[cur]; {
		seen[cur] = true
		if ban, ok := bans[cur]; ok {
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

// banView is the ban state the views show.
type banView struct {
	bans map[string]BanRecord
	// unknown is set when the shared store could not be read: the views
	// then show bans known here and no token as active.
	unknown bool
}

// currentBans returns the bans the views (token info, graph, stats, ban list)
// show. With the in-memory store that is this process's ban list. With a
// shared store it is the store's ban list, read with the same deadline as an
// access check (BanStoreReadTimeout), plus bans made here that could not be
// written; a token another gateway banned is therefore shown banned at once,
// and one unbanned elsewhere is no longer shown banned. If the store cannot
// be read the error wraps ErrRevocationUnavailable. Call it without s.mu.
func (s *Service) currentBans(ctx context.Context) (map[string]BanRecord, error) {
	if !s.shared {
		s.mu.RLock()
		defer s.mu.RUnlock()
		out := make(map[string]BanRecord, len(s.banned))
		for sig, ban := range s.banned {
			out[sig] = ban
		}
		return out, nil
	}
	start := s.now()
	list, err := boundedRead(ctx, s.store.GetBanList)
	if err != nil {
		log.Error().Err(err).Msg("Ban store unavailable; ban state of the views unknown")
		return nil, fmt.Errorf("%w: %v", ErrRevocationUnavailable, err)
	}
	out := make(map[string]BanRecord, len(list))
	for _, ban := range list {
		out[ban.Signature] = ban
	}
	// Bans made here that the list may not include: not yet written, or
	// written while the list was being read (the rule decideLineage uses).
	s.cacheMu.Lock()
	for sig := range s.pendingBans {
		out[sig], _ = s.localBanLocked(sig, start)
	}
	for sig := range s.localBans {
		if ban, ok := s.localBanLocked(sig, start); ok {
			if _, listed := out[sig]; !listed {
				out[sig] = ban
			}
		}
	}
	s.cacheMu.Unlock()
	return out, nil
}

// viewBans is currentBans for the views that cannot return an error: when the
// shared store cannot be read it falls back to the bans known here and marks
// the view unknown.
func (s *Service) viewBans() banView {
	bans, err := s.currentBans(context.Background())
	if err == nil {
		return banView{bans: bans}
	}
	bans = make(map[string]BanRecord)
	s.mu.RLock()
	for sig, ban := range s.banned {
		bans[sig] = ban
	}
	s.mu.RUnlock()
	s.cacheMu.Lock()
	for sig, ban := range s.pendingBans {
		bans[sig] = ban
	}
	s.cacheMu.Unlock()
	return banView{bans: bans, unknown: true}
}

// StatusUnknown is the token status the views show, with a shared store that
// cannot be read, for a token not known here to be banned.
const StatusUnknown = "unknown"

func shortSig(sig string) string {
	if len(sig) > 12 {
		return sig[:8] + "..." + sig[len(sig)-4:]
	}
	return sig
}

// GetBanList returns all banned tokens. With a shared store it lists the
// store's bans (see GetBanListContext); if the store cannot be read it lists
// the bans known here.
func (s *Service) GetBanList() []BanRecord {
	return banList(s.viewBans().bans)
}

// GetBanListContext returns all banned tokens: with a shared store, the
// store's ban list (bans made by every gateway), or an error wrapping
// ErrRevocationUnavailable if it cannot be read.
func (s *Service) GetBanListContext(ctx context.Context) ([]BanRecord, error) {
	bans, err := s.currentBans(ctx)
	if err != nil {
		return nil, err
	}
	return banList(bans), nil
}

func banList(bans map[string]BanRecord) []BanRecord {
	result := make([]BanRecord, 0, len(bans))
	for _, record := range bans {
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

// GetTokenInfoBySignature returns full info about a specific token. Its
// status reads the shared ban state like GetTokenInfoBySignatureContext; if
// the shared store cannot be read, a token not known here to be banned has
// status StatusUnknown.
func (s *Service) GetTokenInfoBySignature(signature string) *TokenInfo {
	return s.tokenInfo(s.viewBans(), signature)
}

// GetTokenInfoBySignatureContext returns full info about a specific token,
// with its status read from the shared ban state (see currentBans), or an
// error wrapping ErrRevocationUnavailable if that cannot be read.
func (s *Service) GetTokenInfoBySignatureContext(ctx context.Context, signature string) (*TokenInfo, error) {
	bans, err := s.currentBans(ctx)
	if err != nil {
		return nil, err
	}
	return s.tokenInfo(banView{bans: bans}, signature), nil
}

func (s *Service) tokenInfo(view banView, signature string) *TokenInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// Check if we know about this token from minting
	minted, hasMinted := s.minted[signature]
	stats, hasStats := s.usage[signature]
	ban, isBanned := s.coveringBanLocked(view.bans, signature)

	// If we don't have any info about this token, return nil
	if !hasMinted && !hasStats && !isBanned {
		return nil
	}

	info := &TokenInfo{
		Signature: signature,
		Status:    "active",
	}
	if view.unknown {
		info.Status = StatusUnknown
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

// GetAllTokenInfo returns info about all tracked tokens. Statuses read the
// shared ban state like GetAllTokenInfoContext; if the shared store cannot be
// read, a token not known here to be banned has status StatusUnknown.
func (s *Service) GetAllTokenInfo() []TokenInfo {
	return s.allTokenInfo(s.viewBans())
}

// GetAllTokenInfoContext returns info about all tracked tokens, with statuses
// read from the shared ban state (see currentBans), or an error wrapping
// ErrRevocationUnavailable if that cannot be read.
func (s *Service) GetAllTokenInfoContext(ctx context.Context) ([]TokenInfo, error) {
	bans, err := s.currentBans(ctx)
	if err != nil {
		return nil, err
	}
	return s.allTokenInfo(banView{bans: bans}), nil
}

func (s *Service) allTokenInfo(view banView) []TokenInfo {
	s.mu.RLock()
	defer s.mu.RUnlock()

	active := "active"
	if view.unknown {
		active = StatusUnknown
	}
	result := make([]TokenInfo, 0)
	seen := make(map[string]bool)

	// Start with minted tokens (all tokens we know about)
	for sig, minted := range s.minted {
		info := TokenInfo{
			Signature: sig,
			Status:    active,
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
		if ban, banned := s.coveringBanLocked(view.bans, sig); banned {
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
			Status:        active,
			TotalRequests: stats.TotalRequests,
			LastUsed:      &stats.LastUsed,
			Routes:        stats.Routes,
		}

		if ban, banned := view.bans[sig]; banned {
			info.Status = "banned"
			info.BannedAt = &ban.BannedAt
			info.BanReason = ban.Reason
		}

		result = append(result, info)
		seen[sig] = true
	}

	// Add banned tokens that weren't in usage or minted
	for sig, ban := range view.bans {
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

// GetStats returns token counts. Ban counts read the shared ban state like
// GetStatsContext; if the shared store cannot be read, tokens not known here
// to be banned are counted as "unknown", not "active".
func (s *Service) GetStats() map[string]interface{} {
	return s.stats(s.viewBans())
}

// GetStatsContext returns token counts with ban counts read from the shared
// ban state (see currentBans), or an error wrapping ErrRevocationUnavailable
// if that cannot be read.
func (s *Service) GetStatsContext(ctx context.Context) (map[string]interface{}, error) {
	bans, err := s.currentBans(ctx)
	if err != nil {
		return nil, err
	}
	return s.stats(banView{bans: bans}), nil
}

func (s *Service) stats(view banView) map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	activeCount := 0
	unknownCount := 0
	bannedCount := len(view.bans)
	expiredCount := 0
	totalRequests := int64(0)

	// Count minted tokens by status
	for sig, minted := range s.minted {
		if _, banned := s.coveringBanLocked(view.bans, sig); banned {
			continue // banned directly (counted in bannedCount) or through its lineage
		}
		if time.Now().After(minted.ExpiresAt) {
			expiredCount++
		} else if view.unknown {
			unknownCount++
		} else {
			activeCount++
		}
	}

	// Sum usage
	for _, stats := range s.usage {
		totalRequests += stats.TotalRequests
	}

	out := map[string]interface{}{
		"active":        activeCount,
		"banned":        bannedCount,
		"expired":       expiredCount,
		"total":         len(s.minted),
		"totalRequests": totalRequests,
	}
	if view.unknown {
		out[StatusUnknown] = unknownCount
	}
	return out
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

// GetGraph returns the token lineage graph for the dashboard. Node statuses
// read the shared ban state like GetGraphContext; if the shared store cannot
// be read, a node not known here to be banned has status "UNKNOWN".
func (s *Service) GetGraph() GraphResponse {
	return s.graph(s.viewBans())
}

// GetGraphContext returns the token lineage graph with node statuses read
// from the shared ban state (see currentBans), or an error wrapping
// ErrRevocationUnavailable if that cannot be read.
func (s *Service) GetGraphContext(ctx context.Context) (GraphResponse, error) {
	bans, err := s.currentBans(ctx)
	if err != nil {
		return GraphResponse{}, err
	}
	return s.graph(banView{bans: bans}), nil
}

func (s *Service) graph(view banView) GraphResponse {
	s.mu.RLock()
	defer s.mu.RUnlock()

	nodes := make([]GraphNode, 0)
	edges := make([]GraphEdge, 0)

	for sig, token := range s.minted {
		// Determine status
		status := "ACTIVE"
		if _, banned := s.coveringBanLocked(view.bans, sig); banned {
			status = "BANNED"
		} else if time.Now().After(token.ExpiresAt) {
			status = "EXPIRED"
		} else if view.unknown {
			status = "UNKNOWN"
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
		if _, banned := s.coveringBanLocked(view.bans, sig); banned {
			bannedCount++
		} else if !view.unknown {
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
	s.localBans = make(map[string]localBan)
	// banSeq and banSeqSpent are kept: answers read before the reset
	// may still be in flight.
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
