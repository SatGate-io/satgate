package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
)

// PendingHorizon is the final-no-op horizon, measured from the original minute,
// never from a retry. Hosts must count PendingAgedOut only on committed completion.
const PendingHorizon = 48 * time.Hour

// PendingWork is a versioned, self-contained recovery instruction. Receipt is an
// already signed envelope, NOT instructions to sign again. BudgetData is opaque
// to the proxy; its budget adapter must encode all facts needed after restart.
// No closure, request body, credential, or context is persisted.
type PendingWork struct {
	Version    int
	Record     PendingRecord
	Window     *MCPWindowedCall `json:",omitempty"`
	Receipt    json.RawMessage  `json:",omitempty"`
	BudgetData json.RawMessage  `json:",omitempty"`
}

type PendingDisposition string

const (
	PendingCompleted PendingDisposition = "completed"
	PendingAgedOut   PendingDisposition = "aged_out"
)

// PendingStore is an optional write-ahead journal. Save must durably insert an
// immutable entry before returning; duplicate IDs with different contents must
// fail. Process exclusively owns an OPEN entry across all clients until its
// callback returns. On error it remains open; on success completion and metrics
// must commit atomically. Process must skip a busy/completed/missing entry, and
// release ownership on process death. A lease with an expiring timer alone is
// insufficient: a paused former owner must not overlap a replacement.
//
// External callback effects must themselves be idempotent by the stored identity:
// a crash after an effect and before completion WILL invoke the callback again.
// List returns open entries only. Implementations must honor cancellation.
type PendingStore interface {
	Save(context.Context, PendingWork) error
	Process(context.Context, string, func(context.Context, PendingWork) (PendingDisposition, error)) (bool, error)
	List(context.Context) ([]PendingWork, error)
}

// PendingEvidenceRecorder separates pure signing from publication. Prepare must
// have NO external effect: the signed envelope goes into PendingStore before
// Publish is called. Publish archives those exact bytes idempotently; recovery
// never calls Prepare. The envelope must contain no credential or request body.
type PendingEvidenceRecorder interface {
	PreparePendingDecision(context.Context, MCPDecision) (*MCPEvidence, json.RawMessage, error)
	PublishPendingDecision(context.Context, json.RawMessage) error
}

// PendingBudget is required for durable priced calls and carves. Prepare captures
// the hold's immutable billing facts before they can expire. Settle must resolve
// ambiguous commits from an authoritative outcome, not infer a debit from an
// admission tombstone, nor replay a fresh Spend. Its money effects and ledger /
// reconciliation rows must be idempotent beyond PendingHorizon. For carves it
// must preserve the parent-debited outcome even if an expiring guard is lost.
type PendingBudget interface {
	PreparePendingBudget(context.Context, PendingRecord) (json.RawMessage, error)
	SettlePendingBudget(context.Context, PendingWork) error
}

// SetPendingStore installs the host journal before serving. Nil preserves the
// original in-memory behavior. Installing a store does not enable the switch,
// access the store, start a worker, or change switch-off calls. Hosts call
// FlushPending at startup and on a timer while the switch is on.
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

// keepPrepared stores the signed envelope before its publication or any debit.
// stored=true means failures may safely leave recovery to another process.
func (ea *everyAction) keepPrepared(ctx context.Context, prep *pendingPreparation) (stored bool, err error) {
	if prep == nil || len(prep.work.Receipt) == 0 {
		return false, errors.New("no prepared receipt")
	}
	c, cancel := context.WithTimeout(context.WithoutCancel(ctx), ea.cfg().CommitTimeout)
	defer cancel()
	if err := ea.durable.Save(c, prep.work); err != nil {
		return false, err
	}
	_, err = ea.durable.Process(c, prep.work.Record.ID, ea.resolveDurable)
	return true, err
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
		if _, err := ea.durable.Process(ctx, w.Record.ID, ea.resolveDurable); err != nil {
			log.Warn().Str("kind", w.Record.Kind).Msg("durable pending entry remains open")
		}
	}
}

func (ea *everyAction) durableGap(ctx context.Context, slot *pendingSlot, call MCPWindowedCall) bool {
	r := PendingRecord{ID: call.ReportID, Kind: PendingGap, TenantID: call.Decision.TenantID,
		TokenID: call.Decision.TokenID, BudgetID: call.Decision.BudgetID, Method: call.Decision.MCPMethod,
		Tool: receiptTool(call.Decision.ToolName), Minute: call.WindowStart, Since: time.Now().UTC()}
	// Whitelist recovery fields; do not persist scope text or client identifiers.
	call.Decision = MCPDecision{TenantID: r.TenantID, TokenID: r.TokenID, BudgetID: r.BudgetID,
		MCPMethod: r.Method, ToolName: r.Tool, Decision: "allowed", DecisionReason: DecisionReasonObserveUnrecorded,
		PolicyMode: call.Decision.PolicyMode, BudgetNotEvaluated: true, RouteOrTool: methodRoute(r.Method, r.Tool)}
	c, cancel := context.WithTimeout(context.WithoutCancel(ctx), ea.cfg().MissReportTimeout)
	defer cancel()
	if err := ea.durable.Save(c, PendingWork{Version: 1, Record: r, Window: &call}); err != nil {
		return false
	}
	slot.release()
	// Once stored, even a failed report cannot lose the gap. The host timer retries.
	_, _ = ea.durable.Process(c, r.ID, ea.resolveDurable)
	return true
}
