package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type p2MissingReply struct {
	p2Store
	unreachable bool
	deadline    time.Time
}

func (s *p2MissingReply) Save(c context.Context, _ PendingWork) error {
	s.deadline, _ = c.Deadline()
	return errors.New("fixture: write not sent, answer unknown")
}
func (s *p2MissingReply) ResolveSave(c context.Context, w PendingWork) (bool, error) {
	d, _ := c.Deadline()
	if !d.Equal(s.deadline) {
		return false, errors.New("fixture: deadline renewed")
	}
	if s.unreachable {
		return false, errors.New("fixture: unavailable")
	}
	return s.p2Store.ResolveSave(c, w)
}
func TestP2MissingSaveReplyCancellation(t *testing.T) {
	for _, kind := range []string{"gap", "uncharged", "carve"} {
		t.Run(kind, func(t *testing.T) {
			b := newP2Backend()
			p := p2Proxy(b)
			s := &p2MissingReply{p2Store: p2Store{b}}
			p.SetPendingStore(s)
			w := p2Work(kind)
			ctx, cancel := context.WithTimeout(context.Background(), time.Second)
			defer cancel()
			if err := p.every().saveDurable(ctx, w); !errors.Is(err, ErrPendingNotStored) {
				t.Fatalf("missing work must cancel: %v", err)
			}
			if !b.cancelled[w.Record.ID] {
				t.Fatal("absence was not durably cancelled")
			}
			if err := (p2Store{b}).Save(ctx, w); !errors.Is(err, ErrPendingNotStored) {
				t.Fatal("late Save resurrected refusal")
			}
			p2Proxy(b).FlushPending(ctx)
			if len(b.receipts)+len(b.gaps)+len(b.reconciled) != 0 || b.parent != 100 || b.child != 0 {
				t.Fatal("cancelled work ran")
			}
		})
	}
}
func TestP2ObserveMoneyGapSaveAnswers(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		for _, mode := range []string{"stored", "absent", "unknown"} {
			t.Run(mode, func(t *testing.T) {
				e, inner, _, done := eaHold(t, eaOpts{observe: true}, road.via)
				defer done()
				b := newP2Backend()
				e.base.setDown(true)
				e.proxy.SetBudgetEnforcer(&p2Budget{eaHoldStore: inner, b: b})
				e.proxy.SetEvidenceRecorder(&p2FailPrepare{p2Recorder: &p2Recorder{eaWindowRecorder: e.rec.(*eaWindowRecorder), b: b}})
				switch mode {
				case "stored":
					e.proxy.SetPendingStore(p2LostReply{p2Store{b}, false})
				case "absent":
					e.proxy.SetPendingStore(&p2MissingReply{p2Store: p2Store{b}})
				default:
					e.proxy.SetPendingStore(p2LostReply{p2Store{b}, true})
				}
				resp, out := eaAct(e, kind, 1)
				if mode == "stored" {
					if resp.Error != nil || out != 1 {
						t.Fatal("stored Observe gap refused")
					}
				} else {
					if out != 0 {
						t.Fatal("unconfirmed gap forwarded or returned credential")
					}
					if mode == "absent" && !eaIsProofUnavailable(resp) {
						t.Fatal("confirmed absence must refuse")
					}
					if mode == "unknown" && (resp.Error == nil || resp.Error.Message != "Action outcome unknown: the action may or may not complete; check before retrying") {
						t.Fatal("unknown was refused")
					}
				}
				fresh := p2Proxy(b)
				fresh.FlushPending(context.Background())
				if mode == "absent" && (b.parent != 100 || b.child != 0 || len(b.gaps) != 0) {
					t.Fatal("refused gap later settled")
				}
				if mode != "absent" && len(b.gaps) != 1 {
					t.Fatal("gap lost/doubled")
				}
				if len(b.receipts) != 0 {
					t.Fatal("Observe gap invented allowed receipt")
				}
			})
		}
	})
}

type p2FailPrepare struct{ *p2Recorder }

func (*p2FailPrepare) PreparePendingDecision(context.Context, MCPDecision) (*MCPEvidence, json.RawMessage, error) {
	return nil, nil, errors.New("fixture: signer down")
}
