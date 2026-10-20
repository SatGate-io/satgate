package mcpserver

// Round-3 follow-up tests for "sign every action": a delegation whose commit
// fails after its receipt was signed must not leave a child able to spend
// money its parent was never charged for.
//
// The rule under test: the key is issued (the receipt stands) with a budget
// that holds nothing. The carve reaches the child only once the parent has
// been charged, by the commit or, failing that, by the pending worker's
// SettleCarve, exactly once. If the parent can no longer cover it, the child
// stays unable to spend and the entry closes with that result.

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"testing"
	"time"
)

// SettleCarve makes eaHoldStore a CarveSettler with the faults of the other
// calls: a store that cannot commit cannot settle.
func (b *eaHoldStore) SettleCarve(ctx context.Context, parent, child, rid string, amount int64) (*BudgetResult, error) {
	b.mu.Lock()
	b.settles++
	fail := b.failCommit || b.failSettle
	lose := b.loseSettleAnswers > 0
	if lose {
		b.loseSettleAnswers--
	}
	b.mu.Unlock()
	if fail {
		return nil, errors.New("store busy")
	}
	res, err := b.InMemoryBudgetEnforcer.SettleCarve(ctx, parent, child, rid, amount)
	if err == nil && lose {
		return nil, errors.New("answer lost")
	}
	return res, err
}

func (b *eaHoldStore) settleCount() int {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.settles
}

// eaLostCommit applies the commit and then answers with an error.
type eaLostCommit struct{ *eaHoldStore }

func (b *eaLostCommit) CommitHold(ctx context.Context, id, rid string) error {
	if err := b.eaHoldStore.InMemoryBudgetEnforcer.CommitHold(ctx, id, rid); err != nil {
		return err
	}
	return errors.New("answer lost")
}

// eaNoSettleStore is a budget store with holds and no CarveSettler.
type eaNoSettleStore struct {
	BudgetEnforcer
	BudgetHolder
}

const eaPricedDelegateWire = `"budget":5,"scope":"paid_read","label":"helper"`

// eaDelegatePriced makes the delegation (a carve of 5, scope paid_read) and
// returns the answer and the child's key and budget when one was issued.
func eaDelegatePriced(e *eaEnv, id int) (resp *Response, token, budgetID string) {
	resp = e.send(e.raw(id, MethodSatGateDelegate, eaPricedDelegateWire))
	if resp.Error == nil && resp.Result != nil {
		var r DelegateResult
		if json.Unmarshal(resp.Result, &r) == nil {
			return resp, r.Token, r.BudgetID
		}
	}
	return resp, "", ""
}

// eaChildEnv is an environment for the child: the same proxy, router and
// recorder, the child's key, carried over the same road as the parent's.
func eaChildEnv(t *testing.T, e *eaEnv, road eaRoad, token, budgetID string) (*eaEnv, func()) {
	t.Helper()
	c := *e
	c.token, c.budgetID, c.sender = token, budgetID, nil
	done := func() {}
	if road.via != nil {
		done = road.via(t, &c)
	}
	return &c, done
}

func eaIsBudgetRefusal(r *Response) bool {
	return r != nil && r.Error != nil && r.Error.Code == CodeBudgetExhausted
}

// eaCarveEntries counts the debits the budget store recorded for a request id.
func eaCarveDebits(store *eaHoldStore, budgetID string) (n int, total int64) {
	for _, r := range store.SpendLog() {
		if r.TokenID == budgetID && r.ToolName == "_delegation" {
			n++
			total += r.Cost
		}
	}
	return n, total
}

// The commit fails on every try after the receipt was signed. The key is
// issued and the receipt stands. The child can spend nothing: its priced call
// is refused for budget and nothing is forwarded. The store recovers, the
// pending worker's retry charges the parent, and only then the child's priced
// call is allowed. The parent is charged exactly once.
func TestR3F_CommitFailsAfterSigning_ChildSpendsOnlyOnceTheParentIsCharged(t *testing.T) {
	for _, hold := range []string{"hold lapsed", "hold still open"} {
		for _, via := range []string{"flush", "worker"} {
			for _, road := range eaRoads {
				hold, via, road := hold, via, road
				t.Run(hold+"/"+via+"/"+road.name, func(t *testing.T) {
					e, store, clk, done := eaHold(t, eaOpts{}, road.via)
					defer done()
					if via == "worker" {
						if err := e.proxy.SetEveryActionSettings(EveryActionSettings{PendingRetry: 5 * time.Millisecond}); err != nil {
							t.Fatal(err)
						}
					}
					store.failCommit = true
					start := e.remaining()

					resp, token, childBudget := eaDelegatePriced(e, 1)
					if resp.Error != nil || token == "" || childBudget == "" {
						t.Fatalf("the signed delegation did not issue its key: %s", eaShape(resp))
					}
					if got := e.base.signed(); len(got) != 1 {
						t.Fatalf("receipts = %v, want the one allowed receipt", got)
					}
					if hold == "hold lapsed" {
						clk.advance(e.proxy.EveryActionSettings().HoldTTL + time.Second)
					}
					child, childDone := eaChildEnv(t, e, road, token, childBudget)
					defer childDone()

					// The parent has not been charged, so the child can spend nothing.
					if n, _ := eaCarveDebits(store, e.budgetID); n != 0 {
						t.Fatalf("the parent was charged %d time(s) before the retry", n)
					}
					if via == "worker" {
						// Keep the retry from running yet: the store is still failing.
						time.Sleep(30 * time.Millisecond)
					}
					before := e.rr.forwarded()
					if r := child.call(2, "paid_read", ""); !eaIsBudgetRefusal(r) {
						t.Fatalf("the child's priced call = %s, want a budget refusal", eaShape(r))
					}
					if got := e.rr.forwarded(); got != before {
						t.Fatalf("the child's refused call was forwarded (%d -> %d)", before, got)
					}
					if got := child.remaining(); got != 0 {
						t.Fatalf("the child's budget holds %d before the parent was charged", got)
					}
					recs := e.proxy.PendingRecords()
					if len(recs) != 1 || recs[0].Kind != PendingUncharged || recs[0].Amount != 5 ||
						recs[0].ChildBudgetID != childBudget || recs[0].BudgetID != e.budgetID || recs[0].Method != MethodSatGateDelegate {
						t.Fatalf("pending = %+v, want the one carve", recs)
					}

					// The store recovers; the retry charges the parent and funds the child.
					store.mu.Lock()
					store.failCommit = false
					store.mu.Unlock()
					if via == "flush" {
						if n := e.proxy.FlushPending(context.Background()); n != 0 {
							t.Fatalf("after the retry the store holds %d entries, want 0", n)
						}
					} else {
						eaWait(t, func() bool { return len(e.proxy.PendingRecords()) == 0 })
					}
					if n, total := eaCarveDebits(store, e.budgetID); n != 1 || total != 5 {
						t.Fatalf("the parent was charged %d time(s) for %d, want once for 5", n, total)
					}
					if got := e.remaining(); got != start-5 {
						t.Fatalf("parent remaining %d, want %d", got, start-5)
					}
					if got := child.remaining(); got != 5 {
						t.Fatalf("the child's budget = %d, want 5", got)
					}
					if charged, closed := e.proxy.CarveSettlements(); charged != 1 || closed != 0 {
						t.Fatalf("settlements = %d charged, %d closed; want 1, 0", charged, closed)
					}
					if r := child.call(3, "paid_read", ""); r.Error != nil {
						t.Fatalf("the child's priced call after the charge = %s", eaShape(r))
					}
					if got := e.rr.forwarded(); got != before+1 {
						t.Fatalf("forwarded %d, want %d", got, before+1)
					}
					if got := child.remaining(); got != 4 {
						t.Fatalf("the child's budget = %d after one priced call, want 4", got)
					}
					// Nothing is charged again.
					e.proxy.FlushPending(context.Background())
					time.Sleep(20 * time.Millisecond)
					if n, _ := eaCarveDebits(store, e.budgetID); n != 1 {
						t.Fatalf("the parent was charged %d times", n)
					}
					if got := e.remaining(); got != start-5 {
						t.Fatalf("parent remaining %d at the end, want %d", got, start-5)
					}
				})
			}
		}
	}
}

// The parent spends its balance before the retry. The retry cannot charge it,
// so the child never becomes able to spend, and the entry closes with that
// result: logged, counted, not retried again.
func TestR3F_ParentSpendsItsBalanceBeforeTheRetry_ChildNeverSpends(t *testing.T) {
	for _, road := range eaRoads {
		road := road
		t.Run(road.name, func(t *testing.T) {
			e, store, clk, done := eaHold(t, eaOpts{}, road.via)
			defer done()
			store.failCommit = true
			start := e.remaining()

			resp, token, childBudget := eaDelegatePriced(e, 1)
			if resp.Error != nil || token == "" {
				t.Fatalf("the signed delegation did not issue its key: %s", eaShape(resp))
			}
			child, childDone := eaChildEnv(t, e, road, token, childBudget)
			defer childDone()

			// The hold lapses and the parent spends all but 2 of its balance.
			clk.advance(e.proxy.EveryActionSettings().HoldTTL + time.Second)
			if got := e.remaining(); got != start {
				t.Fatalf("parent remaining %d, want %d (never charged)", got, start)
			}
			if res, err := store.Spend(context.Background(), e.budgetID, "paid_read", start-2, "parent-spends"); err != nil || !res.Allowed {
				t.Fatalf("the parent could not spend: %v", err)
			}

			store.mu.Lock()
			store.failCommit = false
			store.mu.Unlock()
			if n := e.proxy.FlushPending(context.Background()); n != 0 {
				t.Fatalf("the entry did not close: %d left (%+v)", n, e.proxy.PendingRecords())
			}
			if charged, closed := e.proxy.CarveSettlements(); charged != 0 || closed != 1 {
				t.Fatalf("settlements = %d charged, %d closed; want 0, 1", charged, closed)
			}
			if n, _ := eaCarveDebits(store, e.budgetID); n != 0 {
				t.Fatalf("the parent was charged %d time(s) for a carve it cannot cover", n)
			}
			if got := e.remaining(); got != 2 {
				t.Fatalf("parent remaining %d, want 2 (the retry took nothing)", got)
			}
			before := e.rr.forwarded()
			if r := child.call(2, "paid_read", ""); !eaIsBudgetRefusal(r) {
				t.Fatalf("the child's priced call = %s, want a budget refusal", eaShape(r))
			}
			if got := child.remaining(); got != 0 || e.rr.forwarded() != before {
				t.Fatalf("child budget %d, forwarded %d -> %d", got, before, e.rr.forwarded())
			}
			// Closed means closed: no more tries.
			tries := store.settleCount()
			e.proxy.FlushPending(context.Background())
			time.Sleep(20 * time.Millisecond)
			if store.settleCount() != tries {
				t.Fatal("a closed entry was tried again")
			}
		})
	}
}

// A retry whose answer is lost does not charge twice, and neither does a
// commit whose answer was lost before it.
func TestR3F_LostAnswerDoesNotChargeTwice(t *testing.T) {
	for _, lost := range []string{"retry", "commit"} {
		for _, road := range eaRoads {
			lost, road := lost, road
			t.Run(lost+"/"+road.name, func(t *testing.T) {
				e, store, clk, done := eaHold(t, eaOpts{}, road.via)
				defer done()
				start := e.remaining()
				if lost == "retry" {
					store.failCommit = true
				} else {
					e.proxy.SetBudgetEnforcer(&eaLostCommit{store})
				}
				resp, token, childBudget := eaDelegatePriced(e, 1)
				if resp.Error != nil || token == "" {
					t.Fatalf("the signed delegation did not issue its key: %s", eaShape(resp))
				}
				child, childDone := eaChildEnv(t, e, road, token, childBudget)
				defer childDone()
				clk.advance(e.proxy.EveryActionSettings().HoldTTL + time.Second)
				if lost == "retry" {
					store.mu.Lock()
					store.failCommit = false
					store.loseSettleAnswers = 1
					store.mu.Unlock()
					// The first retry takes effect and its answer is lost.
					if n := e.proxy.FlushPending(context.Background()); n != 1 {
						t.Fatalf("after a lost answer the entry must stay: %d entries", n)
					}
				}
				// The retry again; whatever happened, the parent is charged once.
				if n := e.proxy.FlushPending(context.Background()); n != 0 {
					t.Fatalf("the entry did not close: %d left", n)
				}
				if got := e.remaining(); got != start-5 {
					t.Fatalf("parent remaining %d, want %d (charged once)", got, start-5)
				}
				if got := child.remaining(); got != 5 {
					t.Fatalf("the child's budget = %d, want 5 (funded once)", got)
				}
				for i := 0; i < 2; i++ {
					e.proxy.FlushPending(context.Background())
				}
				if got := e.remaining(); got != start-5 || child.remaining() != 5 {
					t.Fatalf("parent %d child %d after more flushes", e.remaining(), child.remaining())
				}
				if r := child.call(2, "paid_read", ""); r.Error != nil {
					t.Fatalf("the child's priced call = %s", eaShape(r))
				}
			})
		}
	}
}

// The commit lands but the child's funding cannot be written: the parent is
// charged, the key is issued, the child can spend nothing, and the retry funds
// it without charging again.
func TestR3F_FundingFailsAfterTheCommit(t *testing.T) {
	for _, road := range eaRoads {
		road := road
		t.Run(road.name, func(t *testing.T) {
			e, store, _, done := eaHold(t, eaOpts{}, road.via)
			defer done()
			store.failSettle = true
			start := e.remaining()
			resp, token, childBudget := eaDelegatePriced(e, 1)
			if resp.Error != nil || token == "" {
				t.Fatalf("the signed delegation did not issue its key: %s", eaShape(resp))
			}
			child, childDone := eaChildEnv(t, e, road, token, childBudget)
			defer childDone()
			if got := e.remaining(); got != start-5 {
				t.Fatalf("parent remaining %d, want %d (committed)", got, start-5)
			}
			before := e.rr.forwarded()
			if r := child.call(2, "paid_read", ""); !eaIsBudgetRefusal(r) || e.rr.forwarded() != before {
				t.Fatalf("the unfunded child's priced call = %s (forwarded %d -> %d)", eaShape(r), before, e.rr.forwarded())
			}
			store.mu.Lock()
			store.failSettle = false
			store.mu.Unlock()
			if n := e.proxy.FlushPending(context.Background()); n != 0 {
				t.Fatalf("the entry did not close: %d left", n)
			}
			if got := e.remaining(); got != start-5 {
				t.Fatalf("parent remaining %d, want %d (charged once)", got, start-5)
			}
			if r := child.call(3, "paid_read", ""); r.Error != nil {
				t.Fatalf("the funded child's priced call = %s", eaShape(r))
			}
		})
	}
}

// With nothing going wrong the child is funded when the key leaves, and the
// parent is charged once. (The round-3 tests cover the rest of the happy path.)
func TestR3F_SuccessFundsTheChildOnceTheParentIsCharged(t *testing.T) {
	for _, road := range eaRoads {
		road := road
		t.Run(road.name, func(t *testing.T) {
			e, store, _, done := eaHold(t, eaOpts{}, road.via)
			defer done()
			start := e.remaining()
			resp, token, childBudget := eaDelegatePriced(e, 1)
			if resp.Error != nil || token == "" {
				t.Fatalf("delegation = %s", eaShape(resp))
			}
			child, childDone := eaChildEnv(t, e, road, token, childBudget)
			defer childDone()
			if n, total := eaCarveDebits(store, e.budgetID); n != 1 || total != 5 {
				t.Fatalf("charged %d time(s) for %d", n, total)
			}
			if e.remaining() != start-5 || child.remaining() != 5 || len(e.proxy.PendingRecords()) != 0 {
				t.Fatalf("parent %d child %d pending %v", e.remaining(), child.remaining(), e.proxy.PendingRecords())
			}
			if r := child.call(2, "paid_read", ""); r.Error != nil {
				t.Fatalf("the child's priced call = %s", eaShape(r))
			}
		})
	}
}

// A store that holds but cannot fund a child once its parent is charged cannot
// keep the rule, so a carve is refused before anything is taken.
func TestR3F_StoreWithoutSettleIsRefusedBeforeTheCarve(t *testing.T) {
	for _, road := range eaRoads {
		road := road
		t.Run(road.name, func(t *testing.T) {
			e, store, _, done := eaHold(t, eaOpts{}, road.via)
			defer done()
			e.proxy.SetBudgetEnforcer(eaNoSettleStore{BudgetEnforcer: store, BudgetHolder: store})
			start := e.remaining()
			resp, token, _ := eaDelegatePriced(e, 1)
			if !eaIsProofUnavailable(resp) || token != "" {
				t.Fatalf("answer = %s", eaShape(resp))
			}
			if _, holds, commits, _ := store.counts(); holds != 0 || commits != 0 || e.remaining() != start {
				t.Fatalf("holds %d commits %d remaining %d: something was taken", holds, commits, e.remaining())
			}
			if len(e.proxy.PendingRecords()) != 0 || len(e.base.signed()) != 0 {
				t.Fatalf("pending %v receipts %v", e.proxy.PendingRecords(), e.base.signed())
			}
		})
	}
}

// Switch off, and Observe, are not changed by any of this: the child's budget
// is created with its carve, as before.
func TestR3F_OffAndObserveCreateTheChildFunded(t *testing.T) {
	for _, name := range []string{"switch off", "observe"} {
		name := name
		t.Run(name, func(t *testing.T) {
			e := newEAEnv(t, eaOpts{observe: name == "observe"})
			if name == "observe" {
				eaSetSwitch(e, true)
			}
			start := e.remaining()
			resp, token, childBudget := eaDelegatePriced(e, 1)
			if resp.Error != nil || token == "" {
				t.Fatalf("delegation = %s", eaShape(resp))
			}
			rem, _ := e.proxy.budget.Remaining(context.Background(), childBudget)
			if rem != 5 || e.remaining() != start-5 {
				t.Fatalf("child %d parent %d, want 5 and %d", rem, e.remaining(), start-5)
			}
		})
	}
}

// ---- the recorder's context (caveat 1 of the round-3 report) ----

// eaHonestRecorder does its work (it takes `work`), and right before it signs
// it looks at its context and signs nothing once the context is done. This is
// what the enterprise recorder must do.
type eaHonestRecorder struct {
	*eaWindowRecorder
	work time.Duration
}

func (r *eaHonestRecorder) RecordMCPDecision(ctx context.Context, d MCPDecision) (*MCPEvidence, error) {
	time.Sleep(r.work) // work that does not watch the context
	if err := ctx.Err(); err != nil {
		return nil, err // the last look before signing
	}
	return r.eaWindowRecorder.RecordMCPDecision(ctx, d)
}

// A recorder that honours its context makes no receipt after the signing
// limit: the call is refused as proof_unavailable, nothing is forwarded or
// issued, and when the recorder's work ends there is still no receipt.
func TestR3F_RecorderHonouringItsContextSignsNothingAfterTheLimit(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		e, store, _, done := eaHold(t, eaOpts{}, road.via)
		defer done()
		rec := &eaHonestRecorder{eaWindowRecorder: newEAWindowRecorder(e.base), work: 300 * time.Millisecond}
		e.proxy.SetEvidenceRecorder(rec)
		if err := e.proxy.SetSignEveryAction(true); err != nil {
			t.Fatal(err)
		}
		if err := e.proxy.SetEveryActionSettings(EveryActionSettings{SignTimeout: 100 * time.Millisecond, CommitTimeout: 100 * time.Millisecond, HoldTTL: time.Minute}); err != nil {
			t.Fatal(err)
		}
		start := e.remaining()
		resp, out := eaAct(e, kind, 1)
		if !eaIsProofUnavailable(resp) || out != 0 || e.rr.forwarded() != 0 {
			t.Fatalf("answer = %s (out %d, forwarded %d)", eaShape(resp), out, e.rr.forwarded())
		}
		// The recorder's work ends well after the limit.
		time.Sleep(500 * time.Millisecond)
		if got := e.base.signed(); len(got) != 0 {
			t.Fatalf("a receipt was made after the signing limit: %v", got)
		}
		if _, _, commits, releases := store.counts(); commits != 0 || releases < 1 {
			t.Fatalf("commits %d releases %d", commits, releases)
		}
		if got := e.remaining(); got != start {
			t.Fatalf("remaining %d, want %d", got, start)
		}
		if strings.Contains(eaShape(resp), "rcpt_") {
			t.Fatalf("the refusal names a receipt: %s", eaShape(resp))
		}
	})
}
