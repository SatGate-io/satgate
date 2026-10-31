package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// PendingHorizon is the final-no-op horizon, measured from the original minute,
// never from a retry. Hosts count PendingAgedOut only on committed completion.
// With both signing switches on nothing ages out: a journaled gap is counted,
// and a prepared receipt published and its money settled, however late (see
// everyAction.closesGaps). With either switch off an entry older than the
// horizon is a final no-op, as it always was.
const PendingHorizon = 48 * time.Hour

// PendingWork is a versioned, self-contained recovery instruction. Receipt is an
// already signed envelope, NOT instructions to sign again. BudgetData captures
// immutable billing facts. Never persist closures, request bodies or credentials.
type PendingWork struct {
	Version    int
	Record     PendingRecord
	Window     *MCPWindowedCall `json:",omitempty"`
	Receipt    json.RawMessage  `json:",omitempty"`
	BudgetData json.RawMessage  `json:",omitempty"`
	// Recovery is transient: credential delivery is unconfirmed after a crash.
	Recovery bool `json:"-"`
}

type PendingDisposition string

const (
	PendingCompleted PendingDisposition = "completed"
	PendingAgedOut   PendingDisposition = "aged_out"
)

type PendingSaveOutcome string

const (
	PendingStored      PendingSaveOutcome = "stored"
	PendingNotStored   PendingSaveOutcome = "definitely_not_stored"
	PendingSaveUnknown PendingSaveOutcome = "unknown"
)

// ErrPendingNotStored is permitted only when the store proves no work was stored.
// Every other error, including a lost commit reply, means UNKNOWN.
var ErrPendingNotStored = errors.New("pending work definitely not stored")

// ErrPendingNotWritten is ErrPendingNotStored for a store that proves, from a
// failure before it wrote anything (no connection, a transaction that cannot
// start, a lock or read that fails, work it cannot encode), that this work is
// not stored and cannot be stored later. It wraps ErrPendingNotStored, so
// errors.Is(err, ErrPendingNotStored) holds. A store returns it only for those
// failures; a written cancellation or a reached bound stays the plain
// ErrPendingNotStored, as before. The proxy answers it as a definite
// not-stored save only with both signing switches on; with either off it
// treats the save as ambiguous, as it always did (ResolveSave first).
var ErrPendingNotWritten = fmt.Errorf("%w: nothing was written", ErrPendingNotStored)

var ErrPendingOutcomeUnknown = errors.New("pending action outcome unknown")

func PendingSaveResult(err error) PendingSaveOutcome {
	if err == nil {
		return PendingStored
	}
	if errors.Is(err, ErrPendingNotStored) {
		return PendingNotStored
	}
	return PendingSaveUnknown
}

// PendingStore is an optional write-ahead journal. Save has three outcomes as
// classified by PendingSaveResult. Entries are immutable. ResolveSave atomically
// resolves UNKNOWN by ID: true confirms identical stored work (even completed);
// false confirms a durable cancellation tombstone, written even if no row ever
// existed. An error confirms neither. Save and ResolveSave MUST serialize by ID:
// a delayed Save cannot resurrect cancelled work. Cancellation is permanent.
// Process never invokes a callback for cancelled work. It exclusively owns an
// OPEN entry across all clients until the callback returns. On error work stays
// open; on success completion and metrics commit atomically. Busy/completed/
// missing entries are skipped. Process death releases ownership. An expiring
// lease alone is insufficient: a paused former owner cannot overlap another.
// External callback effects MUST be idempotent by stored identity: a crash after
// an effect and before completion invokes the callback again. List is open-only.
// Implementations must honor cancellation.
type PendingStore interface {
	Save(context.Context, PendingWork) error
	ResolveSave(context.Context, PendingWork) (stored bool, err error)
	Process(context.Context, string, func(context.Context, PendingWork) (PendingDisposition, error)) (bool, error)
	List(context.Context) ([]PendingWork, error)
}

// PendingEvidenceRecorder separates pure signing from publication. Prepare has
// NO external effect. Publish archives the exact stored bytes idempotently;
// recovery never calls Prepare. No credential or request body belongs here.
type PendingEvidenceRecorder interface {
	PreparePendingDecision(context.Context, MCPDecision) (*MCPEvidence, json.RawMessage, error)
	PublishPendingDecision(context.Context, json.RawMessage) error
}

// PendingBudget captures the hold's immutable billing facts. Settle resolves
// ambiguous commits from an authoritative outcome, not an admission tombstone
// or fresh Spend. Money and reconciliation are idempotent beyond PendingHorizon.
// Carves preserve parent-debited outcomes even when expiring guards disappear.
// For Recovery carves, a funded child needs a delivery-unconfirmed reconciliation
// row: its credential may never have reached the agent. Never persist the token.
type PendingBudget interface {
	PreparePendingBudget(context.Context, PendingRecord) (json.RawMessage, error)
	SettlePendingBudget(context.Context, PendingWork) error
}

// PendingObserveBudget may exempt ordinary Observe tools/call from holds ONLY
// when its Spend is guaranteed to create no debit, ledger or idempotency effect
// in this resolved request mode. This never exempts delegation transfers.
// Without this explicit host guarantee durable Observe must hold before acting.
type PendingObserveBudget interface {
	ObserveSpendIsReadOnly(context.Context) bool
}

func (ea *everyAction) holdObserve(ctx context.Context, budget BudgetEnforcer) bool {
	if ea.durable == nil {
		return false
	}
	readOnly, ok := budget.(PendingObserveBudget)
	return !ok || !readOnly.ObserveSpendIsReadOnly(ctx)
}

// SetPendingStore installs a host journal before serving. Nil preserves the RAM
// path. Installation does not enable the switch, access storage or start workers.
// Hosts FlushPending at startup and on a timer only while the switch is on.
func (p *Proxy) SetPendingStore(store PendingStore) {
	ea := p.every()
	ea.durable = store
	ea.pendingBudget, _ = p.budget.(PendingBudget)
}

type pendingPreparationKey struct{}
type pendingPreparation struct{ work PendingWork }

func (ea *everyAction) preparePending(ctx context.Context, r PendingRecord) (context.Context, *pendingPreparation, error) {
	if ea.durable == nil {
		return ctx, nil, nil
	}
	if _, ok := ea.recorder().(PendingEvidenceRecorder); !ok || ea.pendingBudget == nil {
		return ctx, nil, errors.New("durable pending requires a preparing recorder and budget")
	}
	r.ID, r.Kind = r.RequestID, PendingUncharged
	r.Since = time.Now().UTC()
	r.Minute = RateWindowStart(r.Since)
	data, err := ea.pendingBudget.PreparePendingBudget(ctx, r)
	if err != nil {
		return ctx, nil, &pendingBudgetError{err}
	}
	prep := &pendingPreparation{work: PendingWork{Version: 1, Record: r, BudgetData: data}}
	return context.WithValue(ctx, pendingPreparationKey{}, prep), prep, nil
}

// pendingBudgetError is a failure of the budget store's PreparePendingBudget,
// as opposed to a setup without one.
type pendingBudgetError struct{ err error }

func (e *pendingBudgetError) Error() string { return e.err.Error() }
func (e *pendingBudgetError) Unwrap() error { return e.err }

// keepPrepared stores before publication/debit. stored=true means an effect
// failure can safely be recovered. false+UNKNOWN must NOT release or refuse.
func (ea *everyAction) keepPrepared(ctx context.Context, prep *pendingPreparation) (stored bool, err error) {
	if prep == nil || len(prep.work.Receipt) == 0 {
		return false, errors.New("no prepared receipt")
	}
	c, cancel := context.WithTimeout(context.WithoutCancel(ctx), ea.cfg().CommitTimeout)
	defer cancel()
	if err := ea.saveDurable(c, prep.work); err != nil {
		return false, err
	}
	_, err = ea.durable.Process(c, prep.work.Record.ID, ea.resolveDurable)
	return true, err
}

// The SAME deadline covers Save and its one idempotent resolve step.
func (ea *everyAction) saveDurable(ctx context.Context, w PendingWork) error {
	err := ea.durable.Save(ctx, w)
	switch PendingSaveResult(err) {
	case PendingStored:
		return nil
	case PendingNotStored:
		// A store's not-written failure is definite only with both switches
		// on. With either off it takes main's ambiguous path below, so no
		// answer, hold, release, record or count differs from main's. The
		// switch state is read here, once, and nowhere in the store.
		if !errors.Is(err, ErrPendingNotWritten) || ea.closesGaps() {
			return err
		}
	}
	stored, resolveErr := ea.durable.ResolveSave(ctx, w)
	if resolveErr != nil {
		return ErrPendingOutcomeUnknown
	}
	if !stored {
		return ErrPendingNotStored
	}
	return nil
}
func pendingFailureResponse(id json.RawMessage, tool string, err error) *Response {
	if errors.Is(err, ErrPendingOutcomeUnknown) {
		return NewErrorResponseWithData(id, CodeInternalError, "Action outcome unknown: the action may or may not complete; check before retrying", map[string]interface{}{"error": "outcome_unknown", "tool": tool})
	}
	return proofUnavailableResponse(id, tool)
}
func (ea *everyAction) resolveDurable(ctx context.Context, w PendingWork) (PendingDisposition, error) {
	if w.Version != 1 {
		return "", errors.New("unsupported pending version")
	}
	if time.Since(w.Record.Minute) > PendingHorizon && !ea.closesGaps() {
		log.Warn().Str("kind", w.Record.Kind).Msg("durable pending entry passed the 48-hour horizon; final uncounted no-op")
		return PendingAgedOut, nil
	}
	// With both switches on, nothing ages out. A gap is a call that went
	// through: it is counted in its own minute however late. A prepared
	// receipt is published (its exact stored bytes, idempotently) and its
	// money settled however late, by the settlement contract on
	// PendingBudget: idempotent beyond PendingHorizon, never a fresh Spend.
	// Until both have landed the entry stays open.
	switch w.Record.Kind {
	case PendingGap:
		wr, ok := ea.recorder().(WindowRecorder)
		if !ok || w.Window == nil {
			return "", errors.New("pending window recorder unavailable")
		}
		_, err := wr.RecordWindowedCall(ctx, *w.Window)
		if err == nil && len(w.BudgetData) != 0 {
			if ea.pendingBudget == nil {
				return "", errors.New("pending gap budget unavailable")
			}
			err = ea.pendingBudget.SettlePendingBudget(ctx, w)
		}
		return PendingCompleted, err
	case PendingUncharged:
		rec, ok := ea.recorder().(PendingEvidenceRecorder)
		if !ok || ea.pendingBudget == nil || len(w.Receipt) == 0 {
			return "", errors.New("pending settlement dependencies unavailable")
		}
		if err := rec.PublishPendingDecision(ctx, w.Receipt); err != nil {
			return "", err
		}
		return PendingCompleted, ea.pendingBudget.SettlePendingBudget(ctx, w)
	default:
		return "", errors.New("unknown pending kind")
	}
}
func (ea *everyAction) flushDurable(ctx context.Context) {
	ws, err := ea.durable.List(ctx)
	if err != nil {
		log.Error().Msg("durable pending journal could not be read")
		return
	}
	for _, w := range ws {
		if ctx.Err() != nil {
			return
		}
		if _, err := ea.durable.Process(ctx, w.Record.ID, ea.recoverDurable); err != nil {
			log.Warn().Str("kind", w.Record.Kind).Msg("durable pending entry remains open")
		}
	}
}

// recoverDurable is what recovery does with an open journal row: the row is
// resolved as stored, its credential delivery unconfirmed.
func (ea *everyAction) recoverDurable(c context.Context, work PendingWork) (PendingDisposition, error) {
	work.Recovery = true
	return ea.resolveDurable(c, work)
}

func (ea *everyAction) durableGap(ctx context.Context, slot *pendingSlot, call MCPWindowedCall) error {
	_, err := ea.durableGapWork(ctx, slot, call)
	return err
}

// durableGapWork is durableGap that also returns the work it tried to store.
func (ea *everyAction) durableGapWork(ctx context.Context, slot *pendingSlot, call MCPWindowedCall) (PendingWork, error) {
	r := PendingRecord{ID: call.ReportID, Kind: PendingGap, TenantID: call.Decision.TenantID, TokenID: call.Decision.TokenID, BudgetID: call.Decision.BudgetID, Method: call.Decision.MCPMethod, Tool: receiptTool(call.Decision.ToolName), Minute: call.WindowStart, Since: time.Now().UTC()}
	w := PendingWork{Version: 1}
	if prep, _ := ctx.Value(pendingPreparationKey{}).(*pendingPreparation); prep != nil {
		// Durable Observe also holds money until its gap is stored. Otherwise a
		// refused gap could leave an already-funded child whose key never left.
		r = prep.work.Record
		r.Kind = PendingGap
		call.ReportID = r.ID
		w.BudgetData = prep.work.BudgetData
	}
	// Whitelist recovery fields; never scope text or client identifiers.
	call.Decision = MCPDecision{TenantID: r.TenantID, TokenID: r.TokenID, BudgetID: r.BudgetID, MCPMethod: r.Method, ToolName: r.Tool, Decision: "allowed", DecisionReason: DecisionReasonObserveUnrecorded, PolicyMode: call.Decision.PolicyMode, BudgetNotEvaluated: true, RouteOrTool: methodRoute(r.Method, r.Tool)}
	c, cancel := context.WithTimeout(context.WithoutCancel(ctx), ea.cfg().MissReportTimeout)
	defer cancel()
	w.Record, w.Window = r, &call
	if err := ea.saveDurable(c, w); err != nil {
		return w, err
	}
	slot.release()
	_, _ = ea.durable.Process(c, r.ID, ea.resolveDurable)
	return w, nil
}

// ---- an unknown outcome, resolved later ----

// unknownOutcome is a verified key's call whose journal save has an unknown
// result: the journal may or may not hold its work, and asking it again
// failed too (ErrPendingOutcomeUnknown). Its answer stays outcome_unknown and
// nothing is released, refused, sent or issued for it then. With both switches
// on it is kept, in the bounded in-memory pending store a gap uses
// (reportMiss), and the retry worker asks the journal again (ResolveSave)
// until it can tell:
//
//   - the journal holds the work: the row is recovered as any journal row is
//     (recoverDurable: the prepared receipt is published, or the gap counted,
//     then the budget settled);
//   - the journal confirmed the work absent (ResolveSave wrote its
//     cancellation, so no late Save can store it): the call's hold, if any,
//     is given back and the call is recorded by absent: an Observe call is
//     counted as a miss in the call's own minute, a Control call's stop is
//     signed budget_backend_error.
//
// Residual, as for a gap kept in memory: a store at its bound, or a process
// that ends before the journal answers, leaves the call unrecorded (logged).
type unknownOutcome struct {
	work  PendingWork
	store PendingStore
	// release gives back the call's hold once the work is known absent; nil
	// when nothing is held.
	release func(context.Context)
	// absent records the call once the work is known absent. It is given the
	// kept entry: its ID names the count, its Minute is the call's.
	absent func(context.Context, PendingRecord) error
	// known is set once the journal confirmed the work absent.
	known bool
	// stopCtx is the context a Control call's stop is signed under later: built
	// from nothing of the request (detachedCtx), plus the identity the key
	// proved without the key (keptIdentity). Nil for an Observe call, which
	// counts under the retry worker's own context.
	stopCtx context.Context
}

// keepUnknown keeps u in slot, or, with none, in the gap store if it has
// room, and starts the retry worker. rec names the call (tenant, key, budget,
// method, tool, the call's minute); the entry gets an id of its own.
func (ea *everyAction) keepUnknown(slot *pendingSlot, rec PendingRecord, u *unknownOutcome) bool {
	rec.ID, rec.Kind, rec.Since = ea.pending.nextID(kindGap), PendingGap, time.Time{}
	e := &pendingEntry{PendingRecord: rec, unknown: u}
	if !slot.put(e) && !ea.pending.add(kindGap, e) {
		log.Error().Str("method", rec.Method).Str("tool", rec.Tool).Msg("an action whose outcome is unknown could not be kept for retry: the pending store is full; it stays unrecorded")
		return false
	}
	log.Warn().Str("method", rec.Method).Str("tool", rec.Tool).Msg("action outcome unknown; kept until the pending journal can tell whether it holds the action")
	ea.kick()
	return true
}

// resolveUnknown is one retry of a kept unknown outcome. It reports whether
// the entry is done with.
func (ea *everyAction) resolveUnknown(ctx context.Context, e *pendingEntry) bool {
	u := e.unknown
	if !u.known {
		// Failures are not logged here: the entry is retried, silently,
		// as a kept gap is (flush), until the journal answers.
		stored, err := u.store.ResolveSave(ctx, u.work)
		if err != nil {
			return false
		}
		if stored {
			// A row that is busy or already completed is another
			// recovery's to finish (Process returns false, nil).
			if _, err := u.store.Process(ctx, u.work.Record.ID, ea.recoverDurable); err != nil {
				return false
			}
			log.Info().Str("method", e.Method).Str("tool", e.Tool).Msg("an action whose outcome was unknown is in the pending journal; recovered")
			return true
		}
		u.known = true
		if u.release != nil {
			u.release(ctx)
		}
	}
	if err := u.absent(ctx, e.PendingRecord); err != nil {
		return false
	}
	log.Info().Str("method", e.Method).Str("tool", e.Tool).Msg("an action whose outcome was unknown was never in the pending journal; recorded")
	return true
}

// countMiss counts call as an Observe miss with the window recorder.
func (ea *everyAction) countMiss(ctx context.Context, call MCPWindowedCall) error {
	wr, ok := ea.recorder().(WindowRecorder)
	if !ok {
		return errors.New("window recorder unavailable")
	}
	_, err := wr.RecordWindowedCall(ctx, call)
	return err
}

// keepUnknownGap keeps an Observe gap whose journal save is unknown (see
// unknownOutcome). Absent, the gap is counted as recovery would have counted
// it, in the call's minute; money held for it (prep) is given back first.
func (ea *everyAction) keepUnknownGap(slot *pendingSlot, w PendingWork, prep *pendingPreparation) {
	if w.Window == nil {
		return
	}
	call := *w.Window
	u := &unknownOutcome{work: w, store: ea.durable}
	if prep != nil {
		if h, ok := ea.pendingBudget.(BudgetHolder); ok {
			r := prep.work.Record
			u.release = func(c context.Context) { ea.release(c, h, r.BudgetID, r.RequestID, r.Amount) }
		}
	}
	u.absent = func(c context.Context, rec PendingRecord) error {
		miss := call
		miss.WindowStart = rec.Minute
		return ea.countMiss(c, miss)
	}
	rec := w.Record
	rec.Minute = call.WindowStart
	ea.keepUnknown(slot, rec, u)
}

// detachedCtx is the context a kept call carries for a step the retry worker
// runs after the request has been answered. It is built from
// context.Background() and the explicit list of values that step needs, below.
// Nothing of the request's context is kept, directly or as a parent: not its
// cancellation or deadline, and none of its values, so the key, the raw token,
// the connection's credential, the request and its body are not reachable from
// what the kept call holds. The step adds the identity it signs for (see
// keptIdentity).
//
// The list, each value copied (strings.Clone, so no part of a larger string,
// such as a token, is kept alive by a substring):
//   - CtxTenantID: the tenant's id, a plain account identifier (not a
//     credential). Budget and evidence routing of a multi-tenant host read it.
//   - ctxEnforcementMode: the mode name the call was decided under ("observe",
//     "control", ...), read when the stop is signed, so the receipt's policy
//     mode is the call's.
//
// Left out on purpose: the prepared receipt, the hold's expiry, the operation
// lease and the pending slot (the request's own handles), and everything else.
func detachedCtx(ctx context.Context) context.Context {
	c := context.Background()
	if tenant := TenantFromContext(ctx); tenant != "" {
		c = context.WithValue(c, CtxTenantID, strings.Clone(tenant))
	}
	if mode, ok := EnforcementModeFromContext(ctx); ok {
		c = context.WithValue(c, ctxEnforcementMode, strings.Clone(mode))
	}
	return c
}

// keptIdentity is the identity a verified key proved, for a kept call to sign
// or count for later, and nothing of the key: it is built field by field from
// the ids, limits and scope a receipt names (identifyDecision), never copied
// whole. RawToken (the encoded key), Raw (the decoded macaroon and its
// signature) and the rule lists, which a receipt does not carry, are left out.
func keptIdentity(t *TokenInfo) *TokenInfo {
	return &TokenInfo{
		TokenID:          strings.Clone(t.TokenID),
		BudgetID:         strings.Clone(t.BudgetID),
		BudgetLimit:      t.BudgetLimit,
		TenantID:         strings.Clone(t.TenantID),
		Scope:            strings.Clone(t.Scope),
		DelegationDepth:  t.DelegationDepth,
		DelegationBudget: t.DelegationBudget,
		Depth:            t.Depth,
		ParentTokenID:    strings.Clone(t.ParentTokenID),
	}
}
