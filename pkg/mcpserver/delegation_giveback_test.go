package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/rs/zerolog"
	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// gbBudget wraps the in-memory store with hooks, so a test can fail or
// interrupt exactly one step of a delegation and count what the parent saw.
type gbBudget struct {
	*InMemoryBudgetEnforcer

	spendResult func(*BudgetResult, error) (*BudgetResult, error) // replaces the carve: the store takes nothing
	initChild   func(ctx context.Context, id string, n int64) error
	refundErr   error

	parentSpends  int
	parentInits   int
	refundCalls   int
	refundAmounts []int64
	refundCtxErrs []error
}

func newGBBudget() *gbBudget { return &gbBudget{InMemoryBudgetEnforcer: NewInMemoryBudgetEnforcer()} }

func (b *gbBudget) Spend(ctx context.Context, id, tool string, cost int64, req string) (*BudgetResult, error) {
	b.parentSpends++
	if b.spendResult != nil {
		return b.spendResult(nil, nil)
	}
	return b.InMemoryBudgetEnforcer.Spend(ctx, id, tool, cost, req)
}

func (b *gbBudget) Initialize(ctx context.Context, id string, n int64) error {
	if strings.HasPrefix(id, "del-") {
		if b.initChild != nil {
			return b.initChild(ctx, id, n)
		}
	} else {
		b.parentInits++
	}
	return b.InMemoryBudgetEnforcer.Initialize(ctx, id, n)
}

// Refund behaves like a networked store: it fails on a cancelled context.
func (b *gbBudget) Refund(ctx context.Context, budgetID, requestID string, amount int64) error {
	b.refundCalls++
	b.refundAmounts = append(b.refundAmounts, amount)
	b.refundCtxErrs = append(b.refundCtxErrs, ctx.Err())
	if err := ctx.Err(); err != nil {
		return err
	}
	if b.refundErr != nil {
		return b.refundErr
	}
	return b.InMemoryBudgetEnforcer.Refund(ctx, budgetID, requestID, amount)
}

type gbEvents struct {
	mu sync.Mutex
	n  int
}

func (e *gbEvents) Publish(Event) { e.mu.Lock(); e.n++; e.mu.Unlock() }

func gbParent(t *testing.T, svc *macaroon.Service) *TokenInfo {
	t.Helper()
	m, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return &TokenInfo{RawToken: svc.Encode(m), TokenID: "parent-token", BudgetID: "parent"}
}

func gbService(t *testing.T) *macaroon.Service {
	t.Helper()
	svc, err := macaroon.NewService("test-root-key")
	if err != nil {
		t.Fatal(err)
	}
	return svc
}

// gbLogs captures the server log for one test.
func gbLogs(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	old := log.Logger
	log.Logger = zerolog.New(&buf)
	t.Cleanup(func() { log.Logger = old })
	return &buf
}

func gbRemaining(t *testing.T, b BudgetEnforcer, id string) int64 {
	t.Helper()
	n, err := b.Remaining(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

// Parent 100, carve 40, then each failure after the carve ends at 100. A
// failed Delegate and a failed seal happen before the carve in this order, so
// the parent is never charged at all.
func TestFailedChildTokenEndsAtTheParentsFullBalance(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name       string
		svc        func(t *testing.T) (*macaroon.Service, *TokenInfo)
		initChild  func(ctx context.Context, id string, n int64) error
		wantMsg    string
		wantSpends int
		wantRefund int
	}{
		{"macaroon Delegate fails (foreign key)", func(t *testing.T) (*macaroon.Service, *TokenInfo) { return unextendableParent(t) },
			nil, wantCannotDelegate, 0, 0},
		{"macaroon Delegate fails (cannot decode)", func(t *testing.T) (*macaroon.Service, *TokenInfo) {
			svc := gbService(t)
			p := gbParent(t, svc)
			p.RawToken = "not-a-token"
			return svc, p
		}, nil, wantServerSide, 0, 0},
		{"seal fails (no root key)", func(t *testing.T) (*macaroon.Service, *TokenInfo) {
			svc := &macaroon.Service{}
			return svc, gbParent(t, svc)
		}, nil, wantServerSide, 0, 0},
		{"child budget creation fails", func(t *testing.T) (*macaroon.Service, *TokenInfo) {
			svc := gbService(t)
			return svc, gbParent(t, svc)
		}, func(context.Context, string, int64) error { return errors.New("redis: connection refused") }, wantServerSide, 1, 1},
	} {
		t.Run(tc.name, func(t *testing.T) {
			logs := gbLogs(t)
			svc, parent := tc.svc(t)
			b := newGBBudget()
			b.initChild = tc.initChild
			_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
			ev := &gbEvents{}
			d := NewDelegator(svc, b)
			d.SetEventPublisher(ev)

			res, err := d.Delegate(ctx, parent, &DelegateParams{Budget: 40, Label: "kid"})
			if err == nil || res != nil {
				t.Fatalf("want a refusal, got %v %v", res, err)
			}
			if err.Error() != tc.wantMsg {
				t.Fatalf("message %q, want %q", err.Error(), tc.wantMsg)
			}
			if got := gbRemaining(t, b, parent.BudgetID); got != 100 {
				t.Fatalf("parent = %d, want 100", got)
			}
			if b.parentSpends != tc.wantSpends || b.refundCalls != tc.wantRefund {
				t.Fatalf("spends %d refunds %d, want %d and %d", b.parentSpends, b.refundCalls, tc.wantSpends, tc.wantRefund)
			}
			if tc.wantRefund == 1 && b.refundAmounts[0] != 40 {
				t.Fatalf("gave back %d, want 40", b.refundAmounts[0])
			}
			if ev.n != 0 {
				t.Fatalf("a failed request published %d events", ev.n)
			}
			if strings.Contains(logs.String(), "eyJ") {
				t.Fatalf("token text in the log: %s", logs.String())
			}
		})
	}
}

// Whatever the owner can't fix is logged with its cause and never shown.
func TestServerSideCauseIsLoggedAndNeverSent(t *testing.T) {
	logs := gbLogs(t)
	ctx := context.Background()
	svc := gbService(t)
	parent := gbParent(t, svc)
	b := newGBBudget()
	_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
	b.initChild = func(context.Context, string, int64) error { return errors.New("dial tcp 10.1.2.3:6379: refused") }
	raw, _ := json.Marshal(DelegateParams{Budget: 5})
	resp, err := NewDelegator(svc, b).HandleDelegate(ctx, &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Params: raw}, parent)
	if err != nil || resp == nil || resp.Error == nil {
		t.Fatalf("want a JSON-RPC error, got %+v %v", resp, err)
	}
	body, _ := json.Marshal(resp)
	if strings.Contains(string(body), "10.1.2.3") || strings.Contains(string(body), "dial tcp") {
		t.Fatalf("raw cause reached the client: %s", body)
	}
	if resp.Error.Message != wantServerSide || resp.Error.Code != CodeBudgetExhausted {
		t.Fatalf("code %d message %q", resp.Error.Code, resp.Error.Message)
	}
	if d, _ := json.Marshal(resp.Error.Data); string(d) != `{"error":"delegation_failed"}` {
		t.Fatalf("data %s", d)
	}
	if !strings.Contains(logs.String(), "10.1.2.3") || !strings.Contains(logs.String(), `"level":"error"`) {
		t.Fatalf("cause not in the server log: %s", logs.String())
	}
}

// The give-back is an increment: a spend made between the carve and the give-back stays spent.
func TestGiveBackKeepsASpendMadeInBetween(t *testing.T) {
	ctx := context.Background()
	svc := gbService(t)
	parent := gbParent(t, svc)
	b := newGBBudget()
	_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
	b.initChild = func(ctx context.Context, id string, n int64) error {
		if _, err := b.InMemoryBudgetEnforcer.Spend(ctx, parent.BudgetID, "tool", 10, "someone-else"); err != nil {
			t.Error(err)
		}
		return errors.New("child budget store down")
	}
	if _, err := NewDelegator(svc, b).Delegate(ctx, parent, &DelegateParams{Budget: 40}); err == nil {
		t.Fatal("want a refusal")
	}
	if got := gbRemaining(t, b, parent.BudgetID); got != 90 {
		t.Fatalf("parent = %d, want 90 (not 100, not 60)", got)
	}
}

// A request cancelled while the child budget is being created still gets its carve back.
func TestGiveBackSurvivesACancelledRequest(t *testing.T) {
	svc := gbService(t)
	parent := gbParent(t, svc)
	b := newGBBudget()
	rctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_ = b.InMemoryBudgetEnforcer.Initialize(context.Background(), parent.BudgetID, 100)
	b.initChild = func(ctx context.Context, id string, n int64) error {
		cancel()
		return ctx.Err()
	}
	if _, err := NewDelegator(svc, b).Delegate(rctx, parent, &DelegateParams{Budget: 1}); err == nil {
		t.Fatal("want a refusal")
	}
	if got := gbRemaining(t, b, parent.BudgetID); got != 100 {
		t.Fatalf("parent = %d, want 100", got)
	}
	if b.refundCalls != 1 || b.refundCtxErrs[0] != nil {
		t.Fatalf("refund calls %d, context error at the refund %v: must run once on a live context", b.refundCalls, b.refundCtxErrs)
	}
}

// A failed give-back is logged at error level with the budget ID, request ID
// and amount, and the client still gets the refusal.
func TestFailedGiveBackIsLoggedAndStillRefuses(t *testing.T) {
	logs := gbLogs(t)
	ctx := context.Background()
	svc := gbService(t)
	parent := gbParent(t, svc)
	b := newGBBudget()
	_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
	b.initChild = func(context.Context, string, int64) error { return errors.New("child down") }
	b.refundErr = errors.New("redis: timeout")
	_, err := NewDelegator(svc, b).Delegate(ctx, parent, &DelegateParams{Budget: 40})
	if err == nil || err.Error() != wantServerSide {
		t.Fatalf("err %v", err)
	}
	out := logs.String()
	for _, want := range []string{`"level":"error"`, `"budgetId":"parent"`, `"requestId":"delegate-parent-`, `"amount":40`, "give-back failed"} {
		if !strings.Contains(out, want) {
			t.Fatalf("log lacks %s: %s", want, out)
		}
	}
	if strings.Contains(out, "eyJ") {
		t.Fatalf("token text in the log: %s", out)
	}
}

// Refund: at most once, increment only, and only a debit the store recorded.
func TestRefundContract(t *testing.T) {
	ctx := context.Background()

	t.Run("second give-back changes nothing", func(t *testing.T) {
		b := NewInMemoryBudgetEnforcer()
		_ = b.Initialize(ctx, "p", 100)
		if _, err := b.Spend(ctx, "p", "_delegation", 40, "r1"); err != nil {
			t.Fatal(err)
		}
		for i := 0; i < 3; i++ {
			if err := b.Refund(ctx, "p", "r1", 40); err != nil {
				t.Fatal(err)
			}
			if got := gbRemaining(t, b, "p"); got != 100 {
				t.Fatalf("after give-back %d: %d, want 100", i+1, got)
			}
		}
	})
	t.Run("a spend in between stays spent", func(t *testing.T) {
		b := NewInMemoryBudgetEnforcer()
		_ = b.Initialize(ctx, "p", 100)
		_, _ = b.Spend(ctx, "p", "_delegation", 40, "r1")
		_, _ = b.Spend(ctx, "p", "tool", 10, "r2")
		_ = b.Refund(ctx, "p", "r1", 40)
		if got := gbRemaining(t, b, "p"); got != 90 {
			t.Fatalf("got %d, want 90", got)
		}
	})
	t.Run("never more than the store recorded", func(t *testing.T) {
		b := NewInMemoryBudgetEnforcer()
		_ = b.Initialize(ctx, "p", 100)
		_, _ = b.Spend(ctx, "p", "_delegation", 40, "r1")
		_ = b.Refund(ctx, "p", "r1", 1000)
		if got := gbRemaining(t, b, "p"); got != 100 {
			t.Fatalf("got %d, want 100", got)
		}
	})
	t.Run("an unlimited parent gets no budget record", func(t *testing.T) {
		b := NewInMemoryBudgetEnforcer()
		r, err := b.Spend(ctx, "nobudget", "_delegation", 40, "r1")
		if err != nil || r == nil || !r.Allowed {
			t.Fatalf("unlimited spend: %v %v", r, err)
		}
		_ = b.Refund(ctx, "nobudget", "r1", 40)
		if got := gbRemaining(t, b, "nobudget"); got != -1 {
			t.Fatalf("remaining %d, want -1 (unlimited)", got)
		}
		b.mu.Lock()
		_, has := b.budgets["nobudget"]
		b.mu.Unlock()
		if has {
			t.Fatal("a give-back created a budget record for an unlimited parent")
		}
	})
	t.Run("a refused spend and an unknown request are never credited", func(t *testing.T) {
		b := NewInMemoryBudgetEnforcer()
		_ = b.Initialize(ctx, "p", 10)
		if r, err := b.Spend(ctx, "p", "_delegation", 40, "refused"); err == nil || r == nil || r.Allowed {
			t.Fatalf("expected a refusal, got %v %v", r, err)
		}
		_ = b.Refund(ctx, "p", "refused", 40)
		_ = b.Refund(ctx, "p", "never-spent", 40)
		_ = b.Refund(ctx, "p", "refused", 0)
		_ = b.Refund(ctx, "p", "refused", -5)
		if got := gbRemaining(t, b, "p"); got != 10 {
			t.Fatalf("got %d, want 10", got)
		}
	})
	t.Run("another budget's request ID is not this budget's debit", func(t *testing.T) {
		b := NewInMemoryBudgetEnforcer()
		_ = b.Initialize(ctx, "p", 100)
		_ = b.Initialize(ctx, "q", 100)
		_, _ = b.Spend(ctx, "p", "_delegation", 40, "r1")
		_ = b.Refund(ctx, "q", "r1", 40)
		if got := gbRemaining(t, b, "q"); got != 100 {
			t.Fatalf("q = %d, want 100", got)
		}
		if got := gbRemaining(t, b, "p"); got != 60 {
			t.Fatalf("p = %d, want 60", got)
		}
	})
	t.Run("concurrent give-backs credit once", func(t *testing.T) {
		b := NewInMemoryBudgetEnforcer()
		_ = b.Initialize(ctx, "p", 100)
		_, _ = b.Spend(ctx, "p", "_delegation", 40, "r1")
		var wg sync.WaitGroup
		for i := 0; i < 32; i++ {
			wg.Add(1)
			go func() { defer wg.Done(); _ = b.Refund(ctx, "p", "r1", 40) }()
		}
		wg.Wait()
		if got := gbRemaining(t, b, "p"); got != 100 {
			t.Fatalf("got %d, want 100", got)
		}
	})
}

// Budget 0 makes zero Spend and zero Initialize calls on the parent, even
// when the parent carries a budget limit.
func TestZeroBudgetNeverTouchesTheParentBudget(t *testing.T) {
	ctx := context.Background()
	svc := gbService(t)
	parent := gbParent(t, svc)
	parent.BudgetLimit = 100
	b := newGBBudget()
	res, err := NewDelegator(svc, b).Delegate(ctx, parent, &DelegateParams{Budget: 0, Label: "observer"})
	if err != nil || res == nil || res.Token == "" {
		t.Fatalf("budget 0 should give a child: %v %v", res, err)
	}
	if b.parentSpends != 0 || b.parentInits != 0 {
		t.Fatalf("parent saw %d spends and %d initializes, want 0 and 0", b.parentSpends, b.parentInits)
	}
	// and when it fails
	usvc, uparent := unextendableParent(t)
	uparent.BudgetLimit = 100
	b2 := newGBBudget()
	if _, err := NewDelegator(usvc, b2).Delegate(ctx, uparent, &DelegateParams{Budget: 0}); err == nil {
		t.Fatal("want a refusal")
	}
	if b2.parentSpends != 0 || b2.parentInits != 0 {
		t.Fatalf("failed budget-0 request: %d spends %d initializes", b2.parentSpends, b2.parentInits)
	}
}

// A carve is approved only by a non-nil result with Allowed true and no error.
func TestOnlyAnApprovedAnswerCarves(t *testing.T) {
	ctx := context.Background()
	for _, tc := range []struct {
		name    string
		answer  func(*BudgetResult, error) (*BudgetResult, error)
		wantMsg func(string) bool
	}{
		{"nil result and nil error", func(*BudgetResult, error) (*BudgetResult, error) { return nil, nil },
			func(m string) bool { return m == wantServerSide }},
		{"nil result and an error", func(*BudgetResult, error) (*BudgetResult, error) { return nil, errors.New("redis down") },
			func(m string) bool { return m == wantServerSide }},
		{"allowed false, no error, no code", func(*BudgetResult, error) (*BudgetResult, error) { return &BudgetResult{Allowed: false}, nil },
			func(m string) bool { return strings.HasPrefix(m, "insufficient parent budget") }},
		{"allowed false, budget code", func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false, ErrorCode: "budget_exhausted"}, nil
		}, func(m string) bool { return strings.HasPrefix(m, "insufficient parent budget") }},
		{"allowed false, outage code", func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false, ErrorCode: "policy_engine_unavailable"}, errors.New("policy engine unavailable")
		}, func(m string) bool { return m == wantServerSide }},
		{"allowed true with an error", func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: true, Remaining: 60}, errors.New("half an answer")
		}, func(m string) bool { return m == wantServerSide }},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := gbService(t)
			parent := gbParent(t, svc)
			b := newGBBudget()
			_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
			b.spendResult = tc.answer
			childInits := 0
			b.initChild = func(context.Context, string, int64) error { childInits++; return nil }
			ev := &gbEvents{}
			d := NewDelegator(svc, b)
			d.SetEventPublisher(ev)

			raw, _ := json.Marshal(DelegateParams{Budget: 40})
			resp, err := d.HandleDelegate(ctx, &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Params: raw}, parent)
			if err != nil || resp == nil || resp.Error == nil || resp.Result != nil {
				t.Fatalf("want a JSON-RPC refusal, got %+v %v", resp, err)
			}
			if resp.Error.Code != CodeBudgetExhausted {
				t.Fatalf("code %d", resp.Error.Code)
			}
			if d, _ := json.Marshal(resp.Error.Data); string(d) != `{"error":"delegation_failed"}` {
				t.Fatalf("data %s", d)
			}
			if !tc.wantMsg(resp.Error.Message) {
				t.Fatalf("message %q", resp.Error.Message)
			}
			if childInits != 0 || ev.n != 0 || b.refundCalls != 0 {
				t.Fatalf("child budget created %d, events %d, give-backs %d: nothing should follow a carve that was not approved", childInits, ev.n, b.refundCalls)
			}
		})
	}
}

// A real budget refusal keeps its text and takes nothing.
func TestInsufficientParentBudgetKeepsItsText(t *testing.T) {
	ctx := context.Background()
	svc := gbService(t)
	parent := gbParent(t, svc)
	b := newGBBudget()
	_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 10)
	_, err := NewDelegator(svc, b).Delegate(ctx, parent, &DelegateParams{Budget: 40})
	if err == nil || !strings.HasPrefix(err.Error(), "insufficient parent budget: budget exhausted: 10 remaining, 40 required") {
		t.Fatalf("err %v", err)
	}
	if b.refundCalls != 0 || gbRemaining(t, b, parent.BudgetID) != 10 {
		t.Fatalf("a refused carve gave back %d times, parent %d", b.refundCalls, gbRemaining(t, b, parent.BudgetID))
	}
}

// A store that only finds out at child-budget creation that the parent cannot
// afford the carve reports it with ErrInsufficientBudget; the client sees the
// budget text, not "a problem on our side". The carve is still given back.
func TestChildBudgetCreationRefusalIsABudgetRefusal(t *testing.T) {
	ctx := context.Background()
	svc := gbService(t)
	parent := gbParent(t, svc)
	b := newGBBudget()
	_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
	b.initChild = func(context.Context, string, int64) error {
		return fmt.Errorf("budget transfer: %w", ErrInsufficientBudget)
	}
	_, err := NewDelegator(svc, b).Delegate(ctx, parent, &DelegateParams{Budget: 40})
	if err == nil || err.Error() != "insufficient parent budget" {
		t.Fatalf("err %v", err)
	}
	if b.refundCalls != 1 || gbRemaining(t, b, parent.BudgetID) != 100 {
		t.Fatalf("give-backs %d parent %d", b.refundCalls, gbRemaining(t, b, parent.BudgetID))
	}
}

// Every message rule, with the exact text.
func TestRefusalMessageRules(t *testing.T) {
	ctx := context.Background()
	t.Run("only a signature that fails the ordinary key gets the Tokens-page sentence", func(t *testing.T) {
		svc, parent := unextendableParent(t)
		_, err := NewDelegator(svc, NewInMemoryBudgetEnforcer()).Delegate(ctx, parent, &DelegateParams{Budget: 1})
		if !errors.Is(err, ErrParentCannotDelegate) || err.Error() != wantCannotDelegate {
			t.Fatalf("err %v", err)
		}
		if _, vErr := svc.Verify(parent.RawToken); !errors.Is(vErr, macaroon.ErrInvalidSignature) {
			t.Fatalf("Verify should wrap ErrInvalidSignature, got %v", vErr)
		}
	})
	for _, tc := range []struct {
		name  string
		build func(t *testing.T) (*macaroon.Service, *TokenInfo)
	}{
		{"decode", func(t *testing.T) (*macaroon.Service, *TokenInfo) {
			svc := gbService(t)
			p := gbParent(t, svc)
			p.RawToken = "%%% not base64 %%%"
			return svc, p
		}},
		{"caveat check (expired parent)", func(t *testing.T) (*macaroon.Service, *TokenInfo) {
			svc := gbService(t)
			m, err := svc.Mint("api:*", time.Now().Add(-time.Hour))
			if err != nil {
				t.Fatal(err)
			}
			return svc, &TokenInfo{RawToken: svc.Encode(m), TokenID: "p", BudgetID: "p"}
		}},
		{"seal with no root key", func(t *testing.T) (*macaroon.Service, *TokenInfo) {
			svc := &macaroon.Service{}
			return svc, gbParent(t, svc)
		}},
	} {
		t.Run(tc.name+" gets the server-side sentence", func(t *testing.T) {
			svc, parent := tc.build(t)
			_, err := NewDelegator(svc, NewInMemoryBudgetEnforcer()).Delegate(ctx, parent, &DelegateParams{Budget: 1})
			if !errors.Is(err, ErrChildTokenFailed) || err.Error() != wantServerSide {
				t.Fatalf("err %v", err)
			}
		})
	}
	t.Run("depth and per-delegation cap are unchanged", func(t *testing.T) {
		TestDelegateOtherErrorsKeepTheirMessages(t)
	})
}

// P5. The seal step reached with no carve must not touch the parent's budget:
// no Spend, no Initialize, no give-back, and the parent keeps its balance. The
// store here answers "nothing" to a carve, the case where the old order
// reached the seal with no carve recorded.
func TestSealStepWithNoCarveDoesNotTouchTheParentBudget(t *testing.T) {
	ctx := context.Background()
	svc := &macaroon.Service{} // no root key: sealing fails
	parent := gbParent(t, svc)
	parent.BudgetLimit = 100
	b := newGBBudget()
	_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
	b.spendResult = func(*BudgetResult, error) (*BudgetResult, error) { return nil, nil }
	res, err := NewDelegator(svc, b).Delegate(ctx, parent, &DelegateParams{Budget: 1})
	if err == nil || res != nil {
		t.Fatalf("want a refusal, got %v %v", res, err)
	}
	if b.parentSpends != 0 || b.parentInits != 0 || b.refundCalls != 0 {
		t.Fatalf("seal failure touched the parent: %d spends, %d initializes, %d give-backs", b.parentSpends, b.parentInits, b.refundCalls)
	}
	if got := gbRemaining(t, b, parent.BudgetID); got != 100 {
		t.Fatalf("parent = %d, want 100", got)
	}
}

// A success still moves exactly the carve.
func TestSuccessfulDelegationMovesExactlyTheCarve(t *testing.T) {
	ctx := context.Background()
	svc := gbService(t)
	parent := gbParent(t, svc)
	b := newGBBudget()
	_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
	res, err := NewDelegator(svc, b).Delegate(ctx, parent, &DelegateParams{Budget: 40, Label: "kid"})
	if err != nil || res == nil {
		t.Fatalf("%v", err)
	}
	if res.ParentRemaining != 60 || gbRemaining(t, b, parent.BudgetID) != 60 || gbRemaining(t, b, res.BudgetID) != 40 {
		t.Fatalf("parent %d child %d result %+v", gbRemaining(t, b, parent.BudgetID), gbRemaining(t, b, res.BudgetID), res)
	}
	if b.refundCalls != 0 {
		t.Fatalf("a success gave back %d times", b.refundCalls)
	}
	if _, err := svc.Verify(res.Token); err != nil {
		t.Fatalf("child token does not verify: %v", err)
	}
}

type gbAuth struct{ parent *TokenInfo }

func (a gbAuth) Verify(context.Context, string) (*TokenInfo, error) { return a.parent, nil }

// Through the real /mcp path: a store that answers nothing to the carve gives
// HTTP 200 and a JSON-RPC error, and the proxy's delegator uses the store the
// proxy was given by SetBudgetEnforcer (the enterprise constructors call it
// after the proxy is built).
func TestDelegateOverMCPUsesTheProxysStoreAndRefusesANilAnswer(t *testing.T) {
	s, p := newHeaderModeStreamable(t)
	svc, err := macaroon.NewService("streamable-header-test-root")
	if err != nil {
		t.Fatal(err)
	}
	parent := gbParent(t, svc)
	p.SetAuthenticator(gbAuth{parent})
	b := newGBBudget()
	_ = b.InMemoryBudgetEnforcer.Initialize(context.Background(), parent.BudgetID, 100)
	p.SetBudgetEnforcer(b) // after construction, as the enterprise entrypoints do
	if p.delegator == nil || p.delegator.budget != BudgetEnforcer(b) {
		t.Fatal("the delegator did not take the store the proxy was given")
	}
	srv := httptest.NewServer(s.Handler())
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, "good-token")
	call := func(id int, budget int64) (int, string) {
		status, _, body := streamablePost(t, srv.URL+"/mcp", sid, "good-token", map[string]any{"jsonrpc": "2.0", "id": id, "method": "satgate/delegate", "params": map[string]any{"budget": budget}})
		return status, body
	}

	// success moves exactly the carve, through the proxy's store
	if status, body := call(2, 30); status != 200 || strings.Contains(body, `"error"`) || b.parentSpends != 1 {
		t.Fatalf("success: status %d spends %d body %s", status, b.parentSpends, body)
	}
	if got := gbRemaining(t, b, parent.BudgetID); got != 70 {
		t.Fatalf("parent = %d, want 70", got)
	}

	// a nil answer is a refusal and moves nothing
	b.spendResult = func(*BudgetResult, error) (*BudgetResult, error) { return nil, nil }
	status, body := call(3, 30)
	if status != 200 || !strings.Contains(body, `"error"`) || !strings.Contains(body, `delegation_failed`) || !strings.Contains(body, wantServerSide) {
		t.Fatalf("nil answer: status %d body %s", status, body)
	}
	if got := gbRemaining(t, b, parent.BudgetID); got != 70 {
		t.Fatalf("parent = %d, want 70", got)
	}
}
