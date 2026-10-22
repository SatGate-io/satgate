package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
)

// PendingHorizon is the final-no-op horizon, measured from the original minute,
// never from a retry. Hosts count PendingAgedOut only on committed completion.
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
		return ctx, nil, err
	}
	prep := &pendingPreparation{work: PendingWork{Version: 1, Record: r, BudgetData: data}}
	return context.WithValue(ctx, pendingPreparationKey{}, prep), prep, nil
}

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
		return err
	default:
		stored, resolveErr := ea.durable.ResolveSave(ctx, w)
		if resolveErr != nil {
			return ErrPendingOutcomeUnknown
		}
		if !stored {
			return ErrPendingNotStored
		}
		return nil
	}
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
	if time.Since(w.Record.Minute) > PendingHorizon {
		log.Warn().Str("kind", w.Record.Kind).Msg("durable pending entry passed the 48-hour horizon; final uncounted no-op")
		return PendingAgedOut, nil
	}
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
		if _, err := ea.durable.Process(ctx, w.Record.ID, func(c context.Context, work PendingWork) (PendingDisposition, error) {
			work.Recovery = true
			return ea.resolveDurable(c, work)
		}); err != nil {
			log.Warn().Str("kind", w.Record.Kind).Msg("durable pending entry remains open")
		}
	}
}
func (ea *everyAction) durableGap(ctx context.Context, slot *pendingSlot, call MCPWindowedCall) error {
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
		return err
	}
	slot.release()
	_, _ = ea.durable.Process(c, r.ID, ea.resolveDurable)
	return nil
}
