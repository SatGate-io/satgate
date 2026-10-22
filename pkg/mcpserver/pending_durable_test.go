package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"testing"
	"time"
)

// This is a journal contract double, not a claim of database durability. Clients
// share only encoded rows and idempotent effects; no pendingEntry/closure survives.
type p2Backend struct {
	mu                        sync.Mutex
	rows                      map[string][]byte
	locks                     map[string]*sync.Mutex
	completed                 map[string]PendingDisposition
	cancelled                 map[string]bool
	receipts                  map[string]bool
	gaps                      map[string]bool
	debit, funded, reconciled map[string]bool
	parent, child             int64
	stop                      string
	failSave                  bool
	calls                     int
}

func newP2Backend() *p2Backend {
	return &p2Backend{rows: map[string][]byte{}, locks: map[string]*sync.Mutex{}, completed: map[string]PendingDisposition{}, cancelled: map[string]bool{}, receipts: map[string]bool{}, gaps: map[string]bool{}, debit: map[string]bool{}, funded: map[string]bool{}, reconciled: map[string]bool{}, parent: 100}
}

type p2Store struct{ b *p2Backend }

func (s p2Store) Save(_ context.Context, w PendingWork) error {
	raw, err := json.Marshal(w)
	if err != nil {
		return err
	}
	s.b.mu.Lock()
	defer s.b.mu.Unlock()
	s.b.calls++
	if s.b.failSave {
		return ErrPendingNotStored
	}
	if s.b.cancelled[w.Record.ID] {
		return ErrPendingNotStored
	}
	if old, ok := s.b.rows[w.Record.ID]; ok && string(old) != string(raw) {
		return errors.New("immutable identity collision")
	}
	s.b.rows[w.Record.ID] = raw
	if s.b.locks[w.Record.ID] == nil {
		s.b.locks[w.Record.ID] = &sync.Mutex{}
	}
	return nil
}
func (s p2Store) ResolveSave(_ context.Context, w PendingWork) (bool, error) {
	s.b.mu.Lock()
	defer s.b.mu.Unlock()
	s.b.calls++
	if s.b.cancelled[w.Record.ID] {
		return false, nil
	}
	if raw, ok := s.b.rows[w.Record.ID]; ok {
		want, _ := json.Marshal(w)
		if string(raw) != string(want) {
			return false, errors.New("immutable identity collision")
		}
		return true, nil
	}
	s.b.cancelled[w.Record.ID] = true
	return false, nil
}
func (s p2Store) List(context.Context) ([]PendingWork, error) {
	s.b.mu.Lock()
	defer s.b.mu.Unlock()
	s.b.calls++
	var out []PendingWork
	for id, raw := range s.b.rows {
		if _, done := s.b.completed[id]; !done {
			var w PendingWork
			if err := json.Unmarshal(raw, &w); err != nil {
				return nil, err
			}
			out = append(out, w)
		}
	}
	return out, nil
}
func (s p2Store) Process(ctx context.Context, id string, fn func(context.Context, PendingWork) (PendingDisposition, error)) (bool, error) {
	s.b.mu.Lock()
	lock := s.b.locks[id]
	s.b.calls++
	s.b.mu.Unlock()
	if lock == nil || !lock.TryLock() {
		return false, nil
	}
	defer lock.Unlock()
	s.b.mu.Lock()
	_, done := s.b.completed[id]
	raw := append([]byte(nil), s.b.rows[id]...)
	s.b.mu.Unlock()
	if done {
		return false, nil
	}
	var w PendingWork
	if err := json.Unmarshal(raw, &w); err != nil {
		return false, err
	}
	result, err := fn(ctx, w)
	if err != nil {
		return false, err
	}
	s.b.mu.Lock()
	defer s.b.mu.Unlock()
	if s.b.stop == "completion" {
		s.b.stop = ""
		return false, errors.New("fixture crash before completion")
	}
	s.b.completed[id] = result
	return true, nil
}

type p2Recorder struct {
	*eaWindowRecorder
	b        *p2Backend
	prepared int
}

func (r *p2Recorder) PreparePendingDecision(_ context.Context, d MCPDecision) (*MCPEvidence, json.RawMessage, error) {
	r.prepared++
	raw, err := json.Marshal(map[string]string{"fixture_receipt": d.RequestID})
	return &MCPEvidence{ReceiptID: "fixture-" + d.RequestID}, raw, err
}
func (r *p2Recorder) PublishPendingDecision(_ context.Context, raw json.RawMessage) error {
	var v map[string]string
	if err := json.Unmarshal(raw, &v); err != nil {
		return err
	}
	r.b.mu.Lock()
	defer r.b.mu.Unlock()
	r.b.receipts[v["fixture_receipt"]] = true
	if r.b.stop == "published" {
		r.b.stop = ""
		return errors.New("fixture crash after publication")
	}
	return nil
}
func (r *p2Recorder) RecordWindowedCall(_ context.Context, c MCPWindowedCall) (*MCPEvidence, error) {
	r.b.mu.Lock()
	defer r.b.mu.Unlock()
	r.b.gaps[c.ReportID] = true
	if r.b.stop == "counted" {
		r.b.stop = ""
		return nil, errors.New("fixture crash after count")
	}
	return nil, nil
}

type p2Budget struct {
	*eaHoldStore
	b *p2Backend
}

func (b *p2Budget) PreparePendingBudget(_ context.Context, r PendingRecord) (json.RawMessage, error) {
	return json.Marshal(r)
}
func (b *p2Budget) SettlePendingBudget(_ context.Context, w PendingWork) error {
	b.b.mu.Lock()
	defer b.b.mu.Unlock()
	id := w.Record.RequestID
	if w.Record.ChildBudgetID != "" {
		if !b.b.debit[id] {
			b.b.parent -= w.Record.Amount
			b.b.debit[id] = true
		}
		if b.b.stop == "parent_debited" {
			b.b.stop = ""
			return errors.New("fixture crash after debit")
		}
		if !b.b.funded[id] {
			b.b.child += w.Record.Amount
			b.b.funded[id] = true
		}
		if b.b.stop == "child_funded" {
			b.b.stop = ""
			return errors.New("fixture crash after funding")
		}
	}
	b.b.reconciled[id] = true
	if b.b.stop == "reconciled" {
		b.b.stop = ""
		return errors.New("fixture crash after reconciliation")
	}
	return nil
}
func p2Proxy(b *p2Backend) *Proxy {
	p := &Proxy{budget: &p2Budget{b: b}}
	rec := &p2Recorder{b: b}
	p.SetEvidenceRecorder(rec)
	p.SetPendingStore(p2Store{b})
	p.every().on.Store(true)
	return p
}
func p2Work(kind string) PendingWork {
	r := PendingRecord{ID: "fixture-work", Kind: PendingUncharged, RequestID: "fixture-call", TenantID: "fixture-account", TokenID: "fixture-principal", BudgetID: "fixture-parent", Method: MethodToolsCall, Tool: "fixture-tool", Amount: 10, Minute: RateWindowStart(time.Now()), Since: time.Now().UTC()}
	w := PendingWork{Version: 1, Record: r, Receipt: json.RawMessage(`{"fixture_receipt":"fixture-call"}`), BudgetData: json.RawMessage(`{}`)}
	switch kind {
	case "gap":
		w.Record.Kind = PendingGap
		w.Window = &MCPWindowedCall{Kind: WindowKindUnrecorded, ReportID: r.ID, WindowStart: r.Minute, WindowSeconds: RateWindowSeconds}
		w.Receipt = nil
	case "carve":
		w.Record.ChildBudgetID = "fixture-child"
		w.Record.Method = MethodSatGateDelegate
	}
	return w
}
func TestP2CrashTable(t *testing.T) {
	for _, kind := range []string{"gap", "uncharged", "carve"} {
		steps := []string{"saved", "completion"}
		if kind == "gap" {
			steps = append(steps, "counted")
		} else {
			steps = append(steps, "published", "reconciled")
			if kind == "carve" {
				steps = append(steps, "parent_debited", "child_funded")
			}
		}
		for _, step := range steps {
			t.Run(kind+"/"+step, func(t *testing.T) {
				b := newP2Backend()
				s := p2Store{b}
				w := p2Work(kind)
				if err := s.Save(context.Background(), w); err != nil {
					t.Fatal(err)
				}
				old := p2Proxy(b)
				if step != "saved" {
					b.stop = step
					old.FlushPending(context.Background())
				}
				// No graceful flush. A new proxy and store client re-read only encoded work.
				old = nil
				fresh := p2Proxy(b)
				if n := fresh.FlushPending(context.Background()); n != 0 {
					t.Fatalf("leftovers: %d", n)
				}
				fresh.FlushPending(context.Background())
				if len(b.completed) != 1 {
					t.Fatal("completion missing or doubled")
				}
				if kind == "gap" {
					if len(b.gaps) != 1 || len(b.receipts) != 0 {
						t.Fatal("gap lost/doubled or receipt invented")
					}
				} else {
					if len(b.receipts) != 1 || len(b.reconciled) != 1 {
						t.Fatal("receipt/reconciliation lost or doubled")
					}
					wantParent, wantChild := int64(100), int64(0)
					if kind == "carve" {
						wantParent, wantChild = 90, 10
					}
					if b.parent != wantParent || b.child != wantChild {
						t.Fatalf("balances %d/%d", b.parent, b.child)
					}
				}
				t.Logf("PASS recovery at %s: parent=%d child=%d receipts=%d reconciliation=%d", step, b.parent, b.child, len(b.receipts), len(b.reconciled))
			})
		}
	}
}
func TestP2TwoCompleters(t *testing.T) {
	b := newP2Backend()
	s := p2Store{b}
	for i := 0; i < 20; i++ {
		w := p2Work("carve")
		w.Record.ID = fmt.Sprintf("fixture-work-%d", i)
		w.Record.RequestID = w.Record.ID
		w.Record.Amount = 1
		if err := s.Save(context.Background(), w); err != nil {
			t.Fatal(err)
		}
	}
	a, c := p2Proxy(b), p2Proxy(b)
	var wg sync.WaitGroup
	for _, p := range []*Proxy{a, c} {
		wg.Go(func() { p.FlushPending(context.Background()) })
	}
	wg.Wait()
	if len(b.completed) != 20 || b.parent != 80 || b.child != 20 || len(b.reconciled) != 20 {
		t.Fatalf("completion=%d balance=%d/%d rows=%d", len(b.completed), b.parent, b.child, len(b.reconciled))
	}
}
func TestP2HorizonIsFinalUncountedNoop(t *testing.T) {
	for _, kind := range []string{"gap", "uncharged", "carve"} {
		t.Run(kind, func(t *testing.T) {
			b := newP2Backend()
			w := p2Work(kind)
			w.Record.Minute = time.Now().Add(-PendingHorizon - time.Minute)
			if err := (p2Store{b}).Save(context.Background(), w); err != nil {
				t.Fatal(err)
			}
			p := p2Proxy(b)
			p.FlushPending(context.Background())
			p.FlushPending(context.Background())
			if b.completed[w.Record.ID] != PendingAgedOut || len(b.gaps)+len(b.receipts)+len(b.reconciled) != 0 || b.parent != 100 || b.child != 0 {
				t.Fatal("old entry was counted/settled or not aged out")
			}
		})
	}
}
func TestP2SwitchOffNeverTouchesInstalledStore(t *testing.T) {
	b := newP2Backend()
	p := p2Proxy(b)
	p.every().on.Store(false)
	p.PendingRecords()
	p.AckPending("fixture-absent")
	p.FlushPending(context.Background())
	if b.calls != 0 {
		t.Fatalf("off touched store %d times", b.calls)
	}
}
func TestP2StoreFailurePrecedesAnyEffect(t *testing.T) {
	b := newP2Backend()
	b.failSave = true
	p := p2Proxy(b)
	prep := &pendingPreparation{work: p2Work("carve")}
	stored, err := p.every().keepPrepared(context.Background(), prep)
	if stored || err == nil || b.parent != 100 || b.child != 0 || len(b.receipts) != 0 {
		t.Fatal("effect preceded durable write")
	}
}
func TestP2ProxyUsesJournalBeforeEffects(t *testing.T) {
	eaBoth(t, func(t *testing.T, road eaRoad, kind string) {
		for _, fail := range []bool{false, true} {
			t.Run(fmt.Sprint("save_failure=", fail), func(t *testing.T) {
				e, inner, _, done := eaHold(t, eaOpts{}, road.via)
				defer done()
				b := newP2Backend()
				b.failSave = fail
				e.proxy.SetBudgetEnforcer(&p2Budget{eaHoldStore: inner, b: b})
				rec := &p2Recorder{eaWindowRecorder: e.rec.(*eaWindowRecorder), b: b}
				e.proxy.SetEvidenceRecorder(rec)
				e.proxy.SetPendingStore(p2Store{b})
				resp, out := eaAct(e, kind, 1)
				if fail {
					if !eaIsProofUnavailable(resp) || out != 0 || len(b.receipts) != 0 || len(b.rows) != 0 {
						t.Fatal("failed journal write let an effect through")
					}
				} else {
					if resp.Error != nil || out != 1 || len(b.rows) != 1 || len(b.completed) != 1 || len(b.receipts) != 1 {
						t.Fatalf("journal path not used: response=%s out=%d rows=%d completed=%d receipts=%d", eaShape(resp), out, len(b.rows), len(b.completed), len(b.receipts))
					}
				}
				_, _, commits, _ := inner.counts()
				if commits != 0 {
					t.Fatal("legacy commit bypassed journal budget resolver")
				}
			})
		}
	})
}

func TestP2PreparingReceiptHasNoExternalEffect(t *testing.T) {
	b := newP2Backend()
	p := p2Proxy(b)
	ctx, prep, err := p.every().preparePending(context.Background(), p2Work("carve").Record)
	if err != nil {
		t.Fatal(err)
	}
	ev, err := p.every().record(ctx, p.evidence, MCPDecision{RequestID: "fixture-call"})
	if err != nil || ev == nil || len(prep.work.Receipt) == 0 {
		t.Fatal("prepare failed")
	}
	if len(b.receipts) != 0 || len(b.rows) != 0 {
		t.Fatal("prepare published before journal")
	}
	stored, err := p.every().keepPrepared(ctx, prep)
	if !stored || err != nil {
		t.Fatalf("keep: %v %v", stored, err)
	}
	if len(b.receipts) != 1 || len(b.completed) != 1 {
		t.Fatal("stored receipt was not published exactly once")
	}
}
