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
// publisher the Delegator publishes to, it answers HandoffLimit from the rows
// it holds, and it writes a row for every child (the child inherits its
// parent's limit, as the gateway's row writer does).
type limitStore struct {
	mu     sync.Mutex
	rows   map[string]*int // row id (budget id) -> stored limit; nil = unlimited
	order  []string        // child rows written, in order
	events int
	err    error // HandoffLimit fails with this when set
}

func newLimitStore() *limitStore { return &limitStore{rows: map[string]*int{}} }

func (s *limitStore) setRow(id string, limit *int) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.rows[id] = limit
}

func (s *limitStore) Publish(ev Event) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.events++
	if ev.Type != EventDelegation {
		return
	}
	child, _ := ev.Data["childBudgetId"].(string)
	if child == "" {
		return
	}
	s.rows[child] = s.rows[ev.BudgetID]
	s.order = append(s.order, child)
}

func (s *limitStore) HandoffLimit(_ context.Context, parent *TokenInfo) (int, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.err != nil {
		return 0, false, s.err
	}
	l, ok := s.rows[parent.BudgetID]
	if !ok || l == nil {
		return 0, false, nil
	}
	return *l, true, nil
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

// A parent with no row at all (a token nobody stored a limit for) is unlimited.
func TestStoredHandoffLimitMissingRowIsUnlimited(t *testing.T) {
	r := newHandoffRig(t)
	parent := r.root(nil)
	delete(r.store.rows, parent.BudgetID)
	r.child(r.handoff(parent, 5))
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
