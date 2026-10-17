package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// The handoff limit an owner stored for a token must hold on satgate/delegate.
// limitStore stands in for the gateway's delegation tree: it is the event
// publisher the Delegator publishes to, it answers HandoffRecord from the rows
// it holds, and it writes a row for every child from the decision the event
// carries (depth + 1, the same limit), before the key is returned
// (PersistChild) and again when the event is published.
type limitStore struct {
	mu        sync.Mutex
	rows      map[string]*storedRow // row id (budget id) -> stored row
	missing   map[string]bool       // ids that should have a row and have none
	order     []string              // child rows written, in order
	persisted []string              // child rows written by PersistChild
	events    int
	err       error // HandoffRecord fails with this when set
}

type storedRow struct {
	limit *int
	depth int
}

type missingRow struct{}

func (missingRow) Error() string              { return "no stored row" }
func (missingRow) HandoffRecordMissing() bool { return true }

func newLimitStore() *limitStore {
	return &limitStore{rows: map[string]*storedRow{}, missing: map[string]bool{}}
}

func (s *limitStore) setRow(id string, limit *int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[id] = &storedRow{limit: limit}
}

func (s *limitStore) record(ev Event, persist bool) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if ev.Type != EventDelegation {
		return nil
	}
	child, _ := ev.Data["childBudgetId"].(string)
	if child == "" {
		child, _ = ev.Data["childTokenId"].(string)
	}
	parent, _ := ev.Data[EventParentRowID].(string)
	if child == "" || parent == "" {
		return nil
	}
	if persist {
		s.persisted = append(s.persisted, child)
	}
	if _, done := s.rows[child]; done {
		return nil
	}
	row := &storedRow{depth: ev.Data[EventParentDepth].(int) + 1}
	if l, ok := ev.Data[EventParentLimit].(int); ok {
		row.limit = &l
	}
	s.rows[child] = row
	s.order = append(s.order, child)
	return nil
}

func (s *limitStore) Publish(ev Event) {
	s.mu.Lock()
	s.events++
	s.mu.Unlock()
	_ = s.record(ev, false)
}

func (s *limitStore) PersistChild(_ context.Context, ev Event) error { return s.record(ev, true) }

func (s *limitStore) HandoffRecord(_ context.Context, parent *TokenInfo) (string, int, int, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return "", 0, 0, false, s.err
	}
	if s.missing[parent.BudgetID] {
		return "", 0, 0, false, missingRow{}
	}
	r, ok := s.rows[parent.BudgetID]
	if !ok {
		return "", 0, 0, false, nil
	}
	if r.limit == nil {
		return parent.BudgetID, r.depth, 0, false, nil
	}
	return parent.BudgetID, r.depth, *r.limit, true, nil
}

func (s *limitStore) children() []string {
	s.mu.Lock()
	defer s.mu.Unlock()
	return append([]string(nil), s.order...)
}

type handoffRig struct {
	t      *testing.T
	svc    *macaroon.Service
	d      *Delegator
	budget *recordingBudget
	store  *limitStore
}

func newHandoffRig(t *testing.T) *handoffRig {
	t.Helper()
	svc, err := macaroon.NewService("handoff-limit-root")
	if err != nil {
		t.Fatal(err)
	}
	rb := &recordingBudget{InMemoryBudgetEnforcer: NewInMemoryBudgetEnforcer()}
	store := newLimitStore()
	d := NewDelegator(svc, rb)
	d.SetEventPublisher(store)
	return &handoffRig{t: t, svc: svc, d: d, budget: rb, store: store}
}

// root mints a root token with a 1000 credit budget and stores limit for it.
func (r *handoffRig) root(limit *int) *TokenInfo {
	r.t.Helper()
	mac, err := r.svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		r.t.Fatal(err)
	}
	tok := r.svc.Encode(mac)
	id := hashToken(mac.Identifier + mac.Signature)
	info := &TokenInfo{TokenID: id, BudgetID: id, Scope: "api:*", RawToken: tok, BudgetLimit: 1000}
	if err := r.budget.Initialize(context.Background(), id, 1000); err != nil {
		r.t.Fatal(err)
	}
	r.store.setRow(id, limit)
	return info
}

// handoff calls the tool's real handler and returns the JSON-RPC response.
func (r *handoffRig) handoff(parent *TokenInfo, budget int64) *Response {
	r.t.Helper()
	raw, _ := json.Marshal(DelegateParams{Budget: budget, Label: "helper"})
	resp, err := r.d.HandleDelegate(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodSatGateDelegate, Params: raw}, parent)
	if err != nil || resp == nil {
		r.t.Fatalf("HandleDelegate: %+v %v", resp, err)
	}
	return resp
}

// child reads the token a successful handoff returned as the next caller.
func (r *handoffRig) child(resp *Response) *TokenInfo {
	r.t.Helper()
	if resp.Error != nil {
		r.t.Fatalf("handoff refused: %d %q", resp.Error.Code, resp.Error.Message)
	}
	var res DelegateResult
	if err := json.Unmarshal(resp.Result, &res); err != nil {
		r.t.Fatal(err)
	}
	mac, err := r.svc.Decode(res.Token)
	if err != nil {
		r.t.Fatal(err)
	}
	info, err := FillTokenInfo(r.svc, mac, res.Token)
	if err != nil {
		r.t.Fatal(err)
	}
	return info
}

func (r *handoffRig) wantRefused(resp *Response, msg string) {
	r.t.Helper()
	if resp.Error == nil {
		r.t.Fatalf("want a refusal, got a result: %s", resp.Result)
	}
	// The same code and data as the depth-limit refusal beside it.
	if resp.Error.Code != CodeBudgetExhausted || resp.Error.Message != msg {
		r.t.Fatalf("code %d message %q, want code %d message %q", resp.Error.Code, resp.Error.Message, CodeBudgetExhausted, msg)
	}
	data, _ := json.Marshal(resp.Error.Data)
	if string(data) != `{"error":"delegation_failed"}` {
		r.t.Fatalf("data %s", data)
	}
}

func intp(n int) *int { return &n }

const wantNoHandoff = "this token can't hand off to a helper"

// A parent whose stored limit is 0 is refused: no child row, no carve, no
// child budget, no event.
func TestStoredHandoffLimitZeroRefusesAndCreatesNothing(t *testing.T) {
	for _, budget := range []int64{0, 25} {
		r := newHandoffRig(t)
		parent := r.root(intp(0))
		resp := r.handoff(parent, budget)
		r.wantRefused(resp, wantNoHandoff)
		if rows := r.store.children(); len(rows) != 0 {
			t.Fatalf("budget %d: child rows written: %v", budget, rows)
		}
		if len(r.budget.spends) != 0 || len(r.budget.initialized) != 1 {
			t.Fatalf("budget %d: the budget was touched: spends %v initializes %v", budget, r.budget.spends, r.budget.initialized)
		}
		if n, _ := r.budget.Remaining(context.Background(), parent.BudgetID); n != 1000 {
			t.Fatalf("budget %d: parent remaining %d, want 1000", budget, n)
		}
		if r.store.events != 0 {
			t.Fatalf("budget %d: %d events published for a refused handoff", budget, r.store.events)
		}
	}
}

// Limit 1: the parent gets a child, and that child's own handoff is refused.
func TestStoredHandoffLimitOneAllowsOneLevel(t *testing.T) {
	r := newHandoffRig(t)
	parent := r.root(intp(1))
	child := r.child(r.handoff(parent, 100))
	if child.Depth != 1 {
		t.Fatalf("child depth %d, want 1", child.Depth)
	}
	if rows := r.store.children(); len(rows) != 1 {
		t.Fatalf("child rows %v, want one", rows)
	}
	child.TenantID = parent.TenantID
	r.wantRefused(r.handoff(child, 10), "this token has reached its handoff limit (1 levels)")
	if rows := r.store.children(); len(rows) != 1 {
		t.Fatalf("a refused handoff wrote a row: %v", rows)
	}
	if n, _ := r.budget.Remaining(context.Background(), child.BudgetID); n != 100 {
		t.Fatalf("the refused handoff changed the child's budget to %d", n)
	}
}

// No stored limit: a child and a grandchild, as before.
func TestStoredHandoffLimitNilIsUnlimited(t *testing.T) {
	r := newHandoffRig(t)
	parent := r.root(nil)
	child := r.child(r.handoff(parent, 100))
	grandchild := r.child(r.handoff(child, 10))
	if grandchild.Depth != 2 {
		t.Fatalf("grandchild depth %d, want 2", grandchild.Depth)
	}
	if n, _ := r.budget.Remaining(context.Background(), parent.BudgetID); n != 900 {
		t.Fatalf("parent remaining %d, want 900 (budgets are carved from the parent as before)", n)
	}
}

// A parent with no stored limit at all (the store keeps nothing for it, such
// as a token that names no tenant) is unlimited, as before.
func TestStoredHandoffLimitMissingRowIsUnlimited(t *testing.T) {
	r := newHandoffRig(t)
	parent := r.root(nil)
	delete(r.store.rows, parent.BudgetID)
	r.child(r.handoff(parent, 5))
}

// Round 2: a parent the store says should have a row, and has none, is
// refused with the plain words, and nothing is created or carved.
func TestStoredHandoffLimitRowThatShouldExistAndDoesNotRefuses(t *testing.T) {
	for _, budget := range []int64{0, 25} {
		r := newHandoffRig(t)
		parent := r.root(nil)
		delete(r.store.rows, parent.BudgetID)
		r.store.missing[parent.BudgetID] = true
		r.wantRefused(r.handoff(parent, budget), wantNoHandoff)
		if len(r.store.children()) != 0 || len(r.budget.spends) != 0 || r.store.events != 0 {
			t.Fatalf("budget %d: something was created: rows %v spends %v events %d", budget, r.store.children(), r.budget.spends, r.store.events)
		}
	}
}

// Round 2: the child's row is written from the row the decision read (depth
// + 1, the same limit), before the key is returned, for a child with credits
// as well as a helper.
func TestStoredHandoffLimitChildRowIsWrittenFromTheDecisionBeforeTheKey(t *testing.T) {
	for _, budget := range []int64{0, 40} {
		r := newHandoffRig(t)
		parent := r.root(intp(2))
		resp := r.handoff(parent, budget)
		child := r.child(resp)
		var res DelegateResult
		_ = json.Unmarshal(resp.Result, &res)
		id := res.BudgetID
		if id == "" {
			id = res.TokenID
		}
		if len(r.store.persisted) == 0 || r.store.persisted[0] != id {
			t.Fatalf("budget %d: the child row was not written before the key: %v (child %s)", budget, r.store.persisted, id)
		}
		row := r.store.rows[id]
		if row == nil || row.depth != 1 || row.limit == nil || *row.limit != 2 {
			t.Fatalf("budget %d: child row %+v, want depth 1 limit 2", budget, row)
		}
		_ = child
	}
}

// Round 2: a child with credits whose row cannot be written gets no key, and
// its carve goes back to the parent.
func TestStoredHandoffLimitChildRowNotWrittenGivesNoKeyAndGivesTheCarveBack(t *testing.T) {
	r := newHandoffRig(t)
	parent := r.root(intp(2))
	r.d.SetEventPublisher(&failingPersister{r.store})
	r.wantRefused(r.handoff(parent, 40), ErrChildTokenFailed.Error())
	if n, _ := r.budget.Remaining(context.Background(), parent.BudgetID); n != 1000 {
		t.Fatalf("parent remaining %d, want 1000 (the carve given back)", n)
	}
}

type failingPersister struct{ *limitStore }

func (f *failingPersister) PersistChild(context.Context, Event) error {
	return errors.New("PRIVATE_DB_CAUSE")
}

// A limit that cannot be read refuses with the server-side sentence; nothing
// is created or carved.
func TestStoredHandoffLimitUnreadableRefusesWithoutCreatingAnything(t *testing.T) {
	r := newHandoffRig(t)
	parent := r.root(nil)
	r.store.err = errors.New("PRIVATE_DB_CAUSE")
	resp := r.handoff(parent, 5)
	r.wantRefused(resp, ErrChildTokenFailed.Error())
	if len(r.store.children()) != 0 || len(r.budget.spends) != 0 {
		t.Fatalf("something was created: rows %v spends %v", r.store.children(), r.budget.spends)
	}
}

// A publisher that stores no limit (the default) changes nothing.
func TestPublisherWithoutHandoffLimiterKeepsTodaysBehaviour(t *testing.T) {
	r := newHandoffRig(t)
	r.d.SetEventPublisher(&NoOpPublisher{})
	parent := r.root(intp(0)) // the row exists, but nobody reads it
	r.child(r.handoff(parent, 5))
}
