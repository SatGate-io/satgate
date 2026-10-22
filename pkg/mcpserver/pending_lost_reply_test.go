package mcpserver

import (
	"context"
	"errors"
	"strings"
	"testing"
)

// Save really commits the encoded work before losing its answer. Resolve can
// independently be unavailable. This regression also compiles on the step-A API.
type p2LostReply struct {
	p2Store
	unreachable bool
}

func (s p2LostReply) Save(c context.Context, w PendingWork) error {
	if err := s.p2Store.Save(c, w); err != nil {
		return err
	}
	return errors.New("fixture: commit answer lost")
}
func (s p2LostReply) ResolveSave(c context.Context, w PendingWork) (bool, error) {
	if s.unreachable {
		return false, errors.New("fixture: resolve unreachable")
	}
	if resolver, ok := any(s.p2Store).(interface {
		ResolveSave(context.Context, PendingWork) (bool, error)
	}); ok {
		return resolver.ResolveSave(c, w)
	}
	// Step A's contract double predates resolution. Keep this probe runnable
	// there unchanged; step A never invokes this fallback.
	ws, err := s.p2Store.List(c)
	if err != nil {
		return false, err
	}
	for _, v := range ws {
		if v.Record.ID == w.Record.ID {
			return true, nil
		}
	}
	return false, errors.New("fixture: committed work missing")
}
func TestP2LostSaveReply(t *testing.T) {
	for _, kind := range []string{"gap", "uncharged", "carve"} {
		for _, unreachable := range []bool{false, true} {
			name := kind + "/resolved"
			if unreachable {
				name = kind + "/unreachable"
			}
			t.Run(name, func(t *testing.T) {
				b := newP2Backend()
				p := p2Proxy(b)
				p.SetPendingStore(p2LostReply{p2Store{b}, unreachable})
				w := p2Work(kind)
				if kind == "gap" {
					// Step A returned bool; the corrected helper returns error. The
					// same behavioral probe must compile and fail on that old head.
					answer := any(p.every().durableGap(context.Background(), nil, *w.Window))
					var err error
					switch v := answer.(type) {
					case bool:
						if !v {
							err = errors.New("fixture: step-A gap refused")
						}
					case error:
						err = v
					}
					if err != nil && !unreachable {
						t.Error("committed gap was treated as refused")
					}
					if unreachable && (err == nil || !strings.Contains(err.Error(), "unknown")) {
						t.Error("gap must be unknown, not refused")
					}
				} else {
					stored, err := p.every().keepPrepared(context.Background(), &pendingPreparation{work: w})
					if !stored && !unreachable {
						t.Error("committed signed work was treated as refused")
					}
					if unreachable && (err == nil || !strings.Contains(err.Error(), "unknown")) {
						t.Error("unreachable resolution must report unknown, not refusal")
					}
				}
				fresh := p2Proxy(b)
				fresh.FlushPending(context.Background())
				if len(b.completed) != 1 {
					t.Fatal("committed work lost on restart")
				}
			})
		}
	}
}
func TestP2LostSaveReplyAnswers(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		for _, unreachable := range []bool{false, true} {
			name := "resolved"
			if unreachable {
				name = "unreachable"
			}
			t.Run(name, func(t *testing.T) {
				e, inner, _, done := eaHold(t, eaOpts{}, road.via)
				defer done()
				b := newP2Backend()
				e.proxy.SetBudgetEnforcer(&p2Budget{eaHoldStore: inner, b: b})
				e.proxy.SetEvidenceRecorder(&p2Recorder{eaWindowRecorder: e.rec.(*eaWindowRecorder), b: b})
				e.proxy.SetPendingStore(p2LostReply{p2Store{b}, unreachable})
				resp, out := eaAct(e, kind, 1)
				if unreachable {
					raw := eaShape(resp)
					if !strings.Contains(raw, "outcome_unknown") || !strings.Contains(raw, "check before retrying") || out != 0 {
						t.Error("lost answer must be unknown, with no dispatch or credential")
					}
					_, _, _, releases := inner.counts()
					if releases != 0 {
						t.Error("unknown released the hold")
					}
					if len(b.receipts) != 0 {
						t.Error("unknown published a receipt before recovery")
					}
				} else if resp.Error != nil || out != 1 {
					t.Error("resolved committed action was refused")
				}
				fresh := p2Proxy(b)
				fresh.FlushPending(context.Background())
				fresh.FlushPending(context.Background())
				if len(b.receipts) != 1 || len(b.completed) != 1 {
					t.Error("restart lost/doubled receipt or completion")
				}
			})
		}
	})
}
