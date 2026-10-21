package mcpserver

// Sign every action: holds, signing time limits, readiness and the pending
// store.
//
// Three rules live here, each so that the owner's promise ("an allowed call is
// either signed before it goes out or, in Control, not sent and not charged; in
// Observe, sent and counted as exactly one gap") does not depend on a later
// step succeeding.
//
//  1. A priced Control call and a delegation carve are held, not debited. Order:
//     hold, sign, commit, then forward (or issue the child key). A hold lapses
//     by itself after HoldTTL unless it is committed, so a call that never got
//     its receipt ends uncharged even if every later call to the store fails.
//  2. Signing has a time limit (SignTimeout) that is always shorter than the
//     hold's (HoldTTL), and a hold is never signed under if its remaining time
//     is less than SignTimeout's share plus CommitTimeout. The hold cannot lapse
//     before a signed call has been committed.
//  3. What cannot be written down in the moment is kept, by tenant, key, tool
//     and minute, in one bounded pending store, and retried until it is
//     recorded exactly once. Two kinds go there: an Observe call that went
//     through with no receipt (a gap) and a hold whose commit failed after the
//     call was signed (an uncharged amount).

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"fmt"
	"sort"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

// The defaults of EveryActionSettings. Each is a setting; these are the values
// used when a setting is left at zero.
const (
	// DefaultHoldTTL is how long a hold lives unless it is committed.
	DefaultHoldTTL = 60 * time.Second
	// DefaultSignTimeout is the longest a receipt may take to make. It is
	// shorter than DefaultHoldTTL by more than DefaultCommitTimeout.
	DefaultSignTimeout = 10 * time.Second
	// DefaultCommitTimeout bounds the commit of a hold (all its tries).
	DefaultCommitTimeout = 5 * time.Second
	// DefaultMissReportTimeout bounds the first, in-line report of an Observe
	// miss. Observe never waits longer than this for it.
	DefaultMissReportTimeout = 2 * time.Second
	// DefaultPendingBound is how many entries of each kind the pending store
	// keeps. At the bound, Observe answers proof_unavailable and a priced
	// Control call is refused before its hold, until the store drains.
	DefaultPendingBound = 1024
	// DefaultPendingRetry is the pause before the first retry of a pending
	// entry. It doubles after a pass that recorded nothing, up to 32 times.
	DefaultPendingRetry = time.Second
)

// EveryActionSettings are the numbers the switch runs on. A zero field takes
// its default. SetEveryActionSettings checks them.
type EveryActionSettings struct {
	HoldTTL           time.Duration
	SignTimeout       time.Duration
	CommitTimeout     time.Duration
	MissReportTimeout time.Duration
	PendingBound      int
	PendingRetry      time.Duration
}

func (s EveryActionSettings) withDefaults() EveryActionSettings {
	if s.HoldTTL == 0 {
		s.HoldTTL = DefaultHoldTTL
	}
	if s.SignTimeout == 0 {
		s.SignTimeout = DefaultSignTimeout
	}
	if s.CommitTimeout == 0 {
		s.CommitTimeout = DefaultCommitTimeout
	}
	if s.MissReportTimeout == 0 {
		s.MissReportTimeout = DefaultMissReportTimeout
	}
	if s.PendingBound == 0 {
		s.PendingBound = DefaultPendingBound
	}
	if s.PendingRetry == 0 {
		s.PendingRetry = DefaultPendingRetry
	}
	return s
}

// ErrEveryActionSettings is returned by SetEveryActionSettings for numbers that
// cannot keep the promise.
var ErrEveryActionSettings = errors.New("sign every action settings are not usable")

func (s EveryActionSettings) validate() error {
	switch {
	case s.HoldTTL < 0 || s.SignTimeout < 0 || s.CommitTimeout < 0 || s.MissReportTimeout < 0 || s.PendingBound < 0 || s.PendingRetry < 0:
		return fmt.Errorf("%w: no setting may be negative", ErrEveryActionSettings)
	case s.SignTimeout+s.CommitTimeout >= s.HoldTTL:
		return fmt.Errorf("%w: the signing time limit (%s) plus the commit limit (%s) must be shorter than the hold's (%s), or a hold could lapse before a signed call is committed",
			ErrEveryActionSettings, s.SignTimeout, s.CommitTimeout, s.HoldTTL)
	}
	return nil
}

// SetEveryActionSettings sets the hold's time limit, the signing time limit,
// the commit limit, the in-line miss-report limit and the pending store's bound
// and retry pause. A zero field takes its default. It fails, and changes
// nothing, when the signing limit plus the commit limit is not shorter than the
// hold's. Call it before the proxy serves requests.
func (p *Proxy) SetEveryActionSettings(s EveryActionSettings) error {
	s = s.withDefaults()
	if err := s.validate(); err != nil {
		return err
	}
	ea := p.every()
	ea.settings.Store(&s)
	ea.pending.bound.Store(int64(s.PendingBound))
	return nil
}

// EveryActionSettings returns the settings in force, defaults filled in.
func (p *Proxy) EveryActionSettings() EveryActionSettings { return *p.every().settings.Load() }

// everyAction is the state the switch keeps apart from the proxy's own: the
// readiness flag, the settings, the pending store and its retry worker. The
// proxy and its delegator share one.
type everyAction struct {
	ready atomic.Bool
	// on mirrors the switch (SetSignEveryAction): with it on, a receipt that
	// is not given a request id is given one the server makes (newCallID),
	// never one derived from the client's JSON-RPC id.
	on       atomic.Bool
	settings atomic.Pointer[EveryActionSettings]
	pending  *pendingStore

	recMu sync.RWMutex
	rec   EvidenceRecorder

	worker           atomic.Bool
	releaseFailures  atomic.Int64
	uncommittedLapse atomic.Int64

	// carveCharged and carveClosed count the delegations whose carve was
	// settled by the retry worker after the key was issued: charged to the
	// parent and the child funded, or closed because the parent could no
	// longer cover it (the child stays unable to spend).
	carveCharged atomic.Int64
	carveClosed  atomic.Int64
}

func newEveryAction() *everyAction {
	ea := &everyAction{pending: newPendingStore()}
	s := EveryActionSettings{}.withDefaults()
	ea.settings.Store(&s)
	ea.pending.bound.Store(int64(s.PendingBound))
	return ea
}

func (ea *everyAction) cfg() EveryActionSettings { return *ea.settings.Load() }

func (ea *everyAction) setRecorder(rec EvidenceRecorder) {
	ea.recMu.Lock()
	ea.rec = rec
	ea.recMu.Unlock()
}

func (ea *everyAction) recorder() EvidenceRecorder {
	ea.recMu.RLock()
	defer ea.recMu.RUnlock()
	return ea.rec
}

// every returns the proxy's state, made on first use so a Proxy built by hand
// works too.
func (p *Proxy) every() *everyAction {
	p.eaOnce.Do(func() {
		p.ea = newEveryAction()
		p.ea.setRecorder(p.evidence)
	})
	return p.ea
}

// ---- readiness ----

// setupBlocks reports whether the switch is on but has not been set up: the
// call must be answered proof_unavailable before anything is spent or sent.
// Setup is SetSignEveryAction(true) returning nil, which needs a recorder that
// keeps windows. Until it has, no call of a key is let through, free or priced,
// Control or Observe: a gap needs a window recorder to be counted, and without
// one an unsigned call would leave no per-key record at all.
func (p *Proxy) setupBlocks() bool {
	return p.signEveryAction && !p.every().ready.Load()
}

// ---- the signing time limit ----

type holdExpiryKey struct{}

// withHoldExpiry notes on ctx when the hold taken for this request lapses.
func withHoldExpiry(ctx context.Context, expiry time.Time) context.Context {
	return context.WithValue(ctx, holdExpiryKey{}, expiry)
}

func holdExpiryFrom(ctx context.Context) (time.Time, bool) {
	t, ok := ctx.Value(holdExpiryKey{}).(time.Time)
	return t, ok
}

var (
	errHoldTooOld    = errors.New("the budget hold has too little time left to sign under")
	errSignTimedOut  = errors.New("signing did not finish in time")
	errRecorderPanic = errors.New("the evidence recorder panicked")
)

// record asks rec for a receipt under the signing time limit. The limit is
// SignTimeout, and when ctx says a hold was taken for this request, also at
// most what is left of the hold's time minus CommitTimeout: a hold cannot lapse
// while its call is being signed or committed. A recorder that ignores its
// context is not waited for past the limit; if it finishes late, its answer is
// dropped.
func (ea *everyAction) record(ctx context.Context, rec EvidenceRecorder, d MCPDecision) (*MCPEvidence, error) {
	s := ea.cfg()
	limit := s.SignTimeout
	if exp, ok := holdExpiryFrom(ctx); ok {
		room := time.Until(exp) - s.CommitTimeout
		if room <= 0 {
			return nil, errHoldTooOld
		}
		if room < limit {
			limit = room
		}
	}
	sctx, cancel := context.WithTimeout(ctx, limit)
	defer cancel()
	type answer struct {
		ev  *MCPEvidence
		err error
	}
	done := make(chan answer, 1)
	// Keep tracked ownership until the recorder actually returns, even when
	// the select below answers at its deadline before recorder cleanup ends.
	lease := retainOperation(ctx)
	go func() {
		if lease != nil {
			defer lease.release()
		}
		defer func() {
			if recover() != nil {
				done <- answer{nil, errRecorderPanic}
			}
		}()
		ev, err := rec.RecordMCPDecision(sctx, d)
		done <- answer{ev, err}
	}()
	select {
	case a := <-done:
		return a.ev, a.err
	case <-sctx.Done():
		return nil, fmt.Errorf("%w (%s): %v", errSignTimedOut, limit, sctx.Err())
	}
}

// ---- holds ----

// holdFor returns the proxy's budget as a BudgetHolder, or nil when it offers
// no hold.
func holdOf(b BudgetEnforcer) BudgetHolder {
	h, _ := b.(BudgetHolder)
	return h
}

// settlerOf returns the budget as a CarveSettler, or nil when it is not one.
func settlerOf(b BudgetEnforcer) CarveSettler {
	s, _ := b.(CarveSettler)
	return s
}

// ---- the identity of a call ----

// newCallID makes the identity of one admitted call, or of one delegation
// carve: the prefix and 128 random bits from the system's source. It is the
// name of the call's hold, of the amount kept for it if its commit fails, of
// its carve settlement, and the request_id its receipt carries. Nothing a
// client sends goes into it (not the JSON-RPC id, the session, a request id,
// a label), so two calls can never share one, whatever numbering their
// clients use. If the source fails the call is refused before anything is
// taken.
func newCallID(prefix string) (string, error) {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		return "", err
	}
	return prefix + hex.EncodeToString(b[:]), nil
}

// Prefixes of the identities newCallID makes.
const (
	callIDPrefix  = "call-"
	carveIDPrefix = "carve-"
)

// commit turns the hold holdID into a debit on a context of its own, bounded
// by CommitTimeout, trying a few times. It is the step after a signature. The
// store's CommitHold acts on that hold only and is the same the second time
// (BudgetHolder), so a try whose answer was lost and its repeat commit the one
// hold once and touch no other call's.
func (ea *everyAction) commit(ctx context.Context, h BudgetHolder, budgetID, holdID string) error {
	return retryUndo(ctx, ea.cfg().CommitTimeout, func(c context.Context) error {
		return h.CommitHold(c, budgetID, holdID)
	})
}

// release gives the hold holdID back on a context of its own, bounded by
// CommitTimeout. If it still fails the hold lapses by itself; the failure is
// logged and counted (ReversalFailures), and nothing more has to succeed.
func (ea *everyAction) release(ctx context.Context, h BudgetHolder, budgetID, holdID string, amount int64) {
	err := retryUndo(ctx, ea.cfg().CommitTimeout, func(c context.Context) error {
		return h.ReleaseHold(c, budgetID, holdID)
	})
	if err != nil {
		ea.releaseFailures.Add(1)
		log.Error().Err(err).Str("budgetId", budgetID).Str("requestId", holdID).Int64("amount", amount).
			Dur("lapsesWithin", ea.cfg().HoldTTL).Msg("could not release a budget hold; it lapses by itself and the balance comes back")
	}
}

// ---- the pending store ----

// What a pending entry is.
const (
	// PendingGap is an Observe call that went through with no receipt, and
	// whose report to the window recorder has not yet been taken.
	PendingGap = "observe_gap"
	// PendingUncharged is a call that was signed and sent while the commit of
	// its hold failed: the hold lapses, the amount was never charged, and the
	// signed receipt says it was. A delegation's carve is such an entry too
	// (ChildBudgetID is set); the worker settles it itself.
	PendingUncharged = "uncharged_commit"
)

const (
	kindGap = iota
	kindUncharged
	kinds
)

// PendingRecord is one entry of the pending store, as the host sees it: who and
// what it was for, and the minute. It never holds anything the agent sent.
type PendingRecord struct {
	// ID is stable for the life of the entry; AckPending takes it.
	ID   string
	Kind string
	// TenantID and TokenID name the account and the key (the key's id, which is
	// already a hash). BudgetID is the budget debited, when there is one.
	TenantID string
	TokenID  string
	BudgetID string
	Method   string
	Tool     string
	// Minute is the start of the UTC minute of the call.
	Minute time.Time
	// Amount and RequestID are set on PendingUncharged: the credits that were
	// held and never committed, and the identity of the call's hold (made by
	// the server for that one call, see newCallID; also the request_id of its
	// receipt). Two entries never share one.
	Amount    int64
	RequestID string
	// ChildBudgetID is set when the entry is a delegation's carve: the budget
	// of the key that was issued. That budget holds nothing, and the child can
	// spend nothing, until the carve has been charged to the parent. The retry
	// worker settles the entry itself (see CarveSettler); the host need only
	// watch it.
	ChildBudgetID string
	// Attempts is how many times the entry has been tried since it was kept.
	Attempts int
	// Since is when the entry was kept.
	Since time.Time
}

// UnchargedRecorder is an optional interface of an EvidenceRecorder (or any
// recorder the gateway installs) that takes the uncharged amounts of the
// pending store. When the recorder has it, the retry worker hands every
// PendingUncharged entry to it and drops the entry once it returns nil. Without
// it the entries stay until the host takes them with PendingRecords and
// AckPending.
type UnchargedRecorder interface {
	RecordUncharged(ctx context.Context, rec PendingRecord) error
}

type pendingEntry struct {
	PendingRecord
	call MCPWindowedCall // PendingGap
	// settle is set on the uncharged entry of a delegation's carve: it charges
	// the parent and funds the child, exactly once however often it is called
	// (CarveSettler). The retry worker runs it whether or not an
	// UnchargedRecorder is installed.
	settle   func(ctx context.Context) error
	inflight bool
}

type pendingStore struct {
	mu       sync.Mutex
	bound    atomic.Int64
	entries  map[string]*pendingEntry
	reserved [kinds]int
	count    [kinds]int
	seq      uint64
	nonce    string
}

// newPendingStore makes a store. Its entries' ids (which are also the ReportID
// of a gap report, the key the window recorder counts it once by) carry a
// nonce of random bits, so that two gateway processes, or one process before
// and after a restart, never make the same id. If the source of random bits
// fails the nonce falls back to the clock, as before.
func newPendingStore() *pendingStore {
	nonce := fmt.Sprintf("%x", time.Now().UnixNano())
	var b [8]byte
	if _, err := rand.Read(b[:]); err == nil {
		nonce = hex.EncodeToString(b[:])
	}
	return &pendingStore{entries: map[string]*pendingEntry{}, nonce: nonce}
}

func kindName(kind int) string {
	if kind == kindGap {
		return PendingGap
	}
	return PendingUncharged
}

func kindOfName(name string) int {
	if name == PendingGap {
		return kindGap
	}
	return kindUncharged
}

// nextID makes an id no other entry of this process has had.
func (s *pendingStore) nextID(kind int) string {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seq++
	return fmt.Sprintf("%s-%s-%d", kindName(kind), s.nonce, s.seq)
}

// pendingSlot is room in the store set aside before a call goes ahead, so that
// what the call may later have to keep can always be kept. It is released when
// the call is done; put turns it into the entry.
type pendingSlot struct {
	s    *pendingStore
	kind int
	done bool
	mu   sync.Mutex
}

// reserve sets room aside for one entry of the kind, or returns nil when the
// kind is at its bound.
func (s *pendingStore) reserve(kind int) *pendingSlot {
	s.mu.Lock()
	defer s.mu.Unlock()
	if int64(s.count[kind]+s.reserved[kind]) >= s.bound.Load() {
		return nil
	}
	s.reserved[kind]++
	return &pendingSlot{s: s, kind: kind}
}

// release gives the room back. It does nothing after put or a second release.
func (sl *pendingSlot) release() {
	if sl == nil {
		return
	}
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.done {
		return
	}
	sl.done = true
	sl.s.mu.Lock()
	sl.s.reserved[sl.kind]--
	sl.s.mu.Unlock()
}

// put keeps e in the room the slot set aside. It returns false if the slot was
// already used.
func (sl *pendingSlot) put(e *pendingEntry) bool {
	if sl == nil {
		return false
	}
	sl.mu.Lock()
	defer sl.mu.Unlock()
	if sl.done {
		return false
	}
	sl.done = true
	s := sl.s
	s.mu.Lock()
	defer s.mu.Unlock()
	s.reserved[sl.kind]--
	s.insertLocked(sl.kind, e)
	return true
}

func (s *pendingStore) insertLocked(kind int, e *pendingEntry) {
	if _, dup := s.entries[e.ID]; dup {
		return
	}
	e.Kind = kindName(kind)
	if e.Since.IsZero() {
		e.Since = time.Now().UTC()
	}
	s.entries[e.ID] = e
	s.count[kind]++
}

// add keeps e when there is room, and reports whether it did.
func (s *pendingStore) add(kind int, e *pendingEntry) bool {
	return s.reserve(kind).put(e)
}

func (s *pendingStore) remove(id string) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	e, ok := s.entries[id]
	if !ok {
		return false
	}
	delete(s.entries, id)
	s.count[kindOfName(e.Kind)]--
	return true
}

func (s *pendingStore) len(kind int) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	return s.count[kind]
}

func (s *pendingStore) records() []PendingRecord {
	s.mu.Lock()
	defer s.mu.Unlock()
	out := make([]PendingRecord, 0, len(s.entries))
	for _, e := range s.entries {
		out = append(out, e.PendingRecord)
	}
	sort.Slice(out, func(i, j int) bool {
		if !out[i].Since.Equal(out[j].Since) {
			return out[i].Since.Before(out[j].Since)
		}
		return out[i].ID < out[j].ID
	})
	return out
}

// claim takes the entries a pass may try, marking them in flight so that the
// worker and a flush never report one twice.
func (s *pendingStore) claim(withUncharged bool) []*pendingEntry {
	s.mu.Lock()
	defer s.mu.Unlock()
	var out []*pendingEntry
	for _, e := range s.entries {
		if e.inflight {
			continue
		}
		if e.Kind == PendingUncharged && !withUncharged && e.settle == nil {
			continue
		}
		e.inflight = true
		e.Attempts++
		out = append(out, e)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Since.Before(out[j].Since) })
	return out
}

func (s *pendingStore) unclaim(e *pendingEntry) {
	s.mu.Lock()
	e.inflight = false
	s.mu.Unlock()
}

// retryable reports whether a worker pass has something to try.
func (s *pendingStore) retryable(withUncharged bool) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.count[kindGap] > 0 {
		return true
	}
	if withUncharged && s.count[kindUncharged] > 0 {
		return true
	}
	for _, e := range s.entries {
		if e.settle != nil {
			return true
		}
	}
	return false
}

// ---- reporting a miss ----

type missSlotKey struct{}

func withMissSlot(ctx context.Context, sl *pendingSlot) context.Context {
	return context.WithValue(ctx, missSlotKey{}, sl)
}

func missSlotFrom(ctx context.Context) *pendingSlot {
	sl, _ := ctx.Value(missSlotKey{}).(*pendingSlot)
	return sl
}

// reportMiss tells the window recorder, in line, that an Observe call went
// through with no receipt. It runs on a context of its own (a cancelled
// request must still be counted) bounded by MissReportTimeout. If that fails,
// the miss is kept with its tenant, key, tool and minute and retried in the
// background until the recorder takes it, exactly once. It returns true when
// the miss is accounted for (recorded or kept) and false only when the store is
// full and no room was set aside for it: the caller then answers
// proof_unavailable and does not pass the call on.
func (ea *everyAction) reportMiss(ctx context.Context, rec EvidenceRecorder, slot *pendingSlot, call MCPWindowedCall) bool {
	if call.ReportID == "" {
		call.ReportID = ea.pending.nextID(kindGap)
	}
	if wr, ok := rec.(WindowRecorder); ok {
		rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), ea.cfg().MissReportTimeout)
		_, err := wr.RecordWindowedCall(rctx, call)
		cancel()
		if err == nil {
			slot.release()
			return true
		}
		log.Error().Err(err).Str("method", call.Decision.MCPMethod).Str("tool", receiptTool(call.Decision.ToolName)).Msg("observe miss could not be recorded; kept for retry")
	}
	e := &pendingEntry{
		PendingRecord: PendingRecord{
			ID:       call.ReportID,
			TenantID: call.Decision.TenantID,
			TokenID:  call.Decision.TokenID,
			BudgetID: call.Decision.BudgetID,
			Method:   call.Decision.MCPMethod,
			Tool:     receiptTool(call.Decision.ToolName),
			Minute:   call.WindowStart.UTC(),
		},
		call: call,
	}
	kept := false
	if slot != nil {
		kept = slot.put(e)
	}
	if !kept {
		kept = ea.pending.add(kindGap, e)
	}
	if !kept {
		log.Error().Str("method", call.Decision.MCPMethod).Str("tool", receiptTool(call.Decision.ToolName)).Msg("the pending store is full; the call is refused, not passed on")
		return false
	}
	ea.kick()
	return true
}

// keepUncharged keeps the amount of a hold whose commit failed after the call
// was signed, in the slot set aside before the hold was taken.
func (ea *everyAction) keepUncharged(slot *pendingSlot, tokenID, tenantID, budgetID, method, tool, requestID string, amount int64) {
	ea.keepUnchargedEntry(slot, tokenID, tenantID, budgetID, method, tool, requestID, amount, "", nil)
}

// keepUnchargedEntry is keepUncharged for any entry. A delegation's carve
// passes the child's budget and the settle step the retry worker runs.
func (ea *everyAction) keepUnchargedEntry(slot *pendingSlot, tokenID, tenantID, budgetID, method, tool, requestID string, amount int64, childBudgetID string, settle func(context.Context) error) {
	e := &pendingEntry{PendingRecord: PendingRecord{
		ID:            ea.pending.nextID(kindUncharged),
		TenantID:      tenantID,
		TokenID:       tokenID,
		BudgetID:      budgetID,
		Method:        method,
		Tool:          receiptTool(tool),
		Minute:        RateWindowStart(time.Now()),
		Amount:        amount,
		RequestID:     requestID,
		ChildBudgetID: childBudgetID,
	}, settle: settle}
	kept := slot.put(e) || ea.pending.add(kindUncharged, e)
	if !kept {
		// The slot was set aside before the hold, so this is a bug.
		log.Error().Str("budgetId", budgetID).Str("requestId", requestID).Int64("amount", amount).Msg("an uncharged amount could not be kept: the pending store is full")
		return
	}
	if settle != nil {
		log.Error().Str("budgetId", budgetID).Str("childBudgetId", childBudgetID).Str("requestId", requestID).Int64("amount", amount).
			Msg("the delegation was signed and its key issued but the carve could not be settled; the child can spend nothing until the parent is charged, which the retry worker keeps trying")
	} else {
		log.Error().Str("budgetId", budgetID).Str("requestId", requestID).Int64("amount", amount).
			Msg("the call was signed and sent but its hold could not be committed; the amount was never charged and is kept for reconciliation")
	}
	ea.kick()
}

// ---- the retry worker ----

func (ea *everyAction) sink() UnchargedRecorder {
	s, _ := ea.recorder().(UnchargedRecorder)
	return s
}

// kick starts the retry worker if it is not running. The worker runs only
// while the store has something it can retry.
func (ea *everyAction) kick() {
	if !ea.worker.CompareAndSwap(false, true) {
		return
	}
	go func() {
		base := ea.cfg().PendingRetry
		delay := base
		for {
			time.Sleep(delay)
			if ea.flush(context.Background()) {
				delay = base
			} else if delay < base*32 {
				delay *= 2
			}
			if !ea.pending.retryable(ea.sink() != nil) {
				break
			}
		}
		ea.worker.Store(false)
		if ea.pending.retryable(ea.sink() != nil) {
			ea.kick()
		}
	}()
}

// flush tries every retryable entry once. It reports whether any was recorded.
func (ea *everyAction) flush(ctx context.Context) bool {
	rec := ea.recorder()
	sink := ea.sink()
	progressed := false
	for _, e := range ea.pending.claim(sink != nil) {
		ok := false
		rctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), ea.cfg().MissReportTimeout)
		switch e.Kind {
		case PendingGap:
			if wr, isWR := rec.(WindowRecorder); isWR {
				_, err := wr.RecordWindowedCall(rctx, e.call)
				ok = err == nil
			}
		case PendingUncharged:
			switch {
			case e.settle != nil:
				ok = ea.settleCarveEntry(rctx, e)
			case sink != nil:
				ok = sink.RecordUncharged(rctx, e.PendingRecord) == nil
			}
		}
		cancel()
		if ok {
			ea.pending.remove(e.ID)
			progressed = true
		} else {
			ea.pending.unclaim(e)
		}
	}
	return progressed
}

// settleCarveEntry runs the settle step of a delegation's carve and reports
// whether the entry is done with. It is done when the carve was charged to the
// parent and the child funded (counted), and also when the parent could no
// longer cover it: the child's budget stays empty for good, which is logged at
// error level and counted, and the entry closes with that result. Any other
// failure keeps the entry for the next pass.
func (ea *everyAction) settleCarveEntry(ctx context.Context, e *pendingEntry) bool {
	err := e.settle(ctx)
	switch {
	case err == nil:
		ea.carveCharged.Add(1)
		log.Info().Str("budgetId", e.BudgetID).Str("childBudgetId", e.ChildBudgetID).Str("requestId", e.RequestID).Int64("amount", e.Amount).
			Msg("the carve of a delegation was charged to the parent after the fact; the child's budget is now spendable")
		return true
	case errors.Is(err, ErrInsufficientBudget):
		ea.carveClosed.Add(1)
		log.Error().Str("budgetId", e.BudgetID).Str("childBudgetId", e.ChildBudgetID).Str("requestId", e.RequestID).Int64("amount", e.Amount).
			Msg("the parent can no longer cover the carve of a delegation whose key was issued; the child's budget stays empty and the entry is closed")
		return true
	}
	log.Warn().Err(err).Str("budgetId", e.BudgetID).Str("childBudgetId", e.ChildBudgetID).Str("requestId", e.RequestID).
		Msg("the carve of a delegation could not be settled yet; it will be tried again")
	return false
}

// CarveSettlements reports what the retry worker has done with delegations
// whose commit failed after the key was issued: how many carves were charged to
// the parent and funded to the child after the fact, and how many were closed
// because the parent could no longer cover them (the child never became able
// to spend; each is also logged at error level).
func (p *Proxy) CarveSettlements() (charged, closedUnfunded int64) {
	ea := p.every()
	return ea.carveCharged.Load(), ea.carveClosed.Load()
}

// PendingRecords lists what the pending store holds: who and what each entry
// was for, and its minute. An entry leaves when it has been recorded (a gap, by
// the window recorder; an uncharged amount, by an UnchargedRecorder) or when
// the host takes it with AckPending.
func (p *Proxy) PendingRecords() []PendingRecord { return p.every().pending.records() }

// AckPending removes the entry the host has dealt with, and reports whether it
// was there.
func (p *Proxy) AckPending(id string) bool { return p.every().pending.remove(id) }

// FlushPending tries every retryable entry once, now, and returns how many
// entries the store still holds. The retry worker does the same on its own; a
// host can call it before it stops.
func (p *Proxy) FlushPending(ctx context.Context) int {
	ea := p.every()
	ea.flush(ctx)
	return len(ea.pending.records())
}
