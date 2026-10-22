package mcpserver

import (
	"context"
	"sync/atomic"
	"testing"
	"time"
)

type p2ReadOnlyObserve struct {
	*p2Budget
	holds atomic.Int64
}

func (*p2ReadOnlyObserve) ObserveSpendIsReadOnly(context.Context) bool { return true }
func (s *p2ReadOnlyObserve) Hold(context.Context, string, string, int64, string, time.Duration) (*BudgetResult, error) {
	s.holds.Add(1)
	return nil, ErrInsufficientBudget
}
func (*p2ReadOnlyObserve) Spend(_ context.Context, id, _ string, cost int64, request string) (*BudgetResult, error) {
	return &BudgetResult{Allowed: true, TokenID: id, Cost: cost, RequestID: request}, nil
}
func TestP2ReadOnlyObserveRetainsNonBillablePath(t *testing.T) {
	for _, down := range []bool{false, true} {
		name := "signed"
		if down {
			name = "gap"
		}
		t.Run(name, func(t *testing.T) {
			e, inner, _, done := eaHold(t, eaOpts{observe: true}, nil)
			defer done()
			b := newP2Backend()
			s := &p2ReadOnlyObserve{p2Budget: &p2Budget{eaHoldStore: inner, b: b}}
			e.base.setDown(down)
			e.proxy.SetBudgetEnforcer(s)
			e.proxy.SetEvidenceRecorder(&p2Recorder{eaWindowRecorder: e.rec.(*eaWindowRecorder), b: b})
			e.proxy.SetPendingStore(p2Store{b})
			resp, out := eaAct(e, "tool call", 1)
			if resp.Error != nil || out != 1 || s.holds.Load() != 0 {
				t.Fatal("non-billable Observe was held/refused")
			}
			if b.parent != 100 || b.child != 0 || len(b.reconciled) != 0 {
				t.Fatal("read-only Observe created money effects")
			}
			if down && len(b.gaps) != 1 {
				t.Fatal("read-only Observe lost its durable gap")
			}
		})
	}
}
