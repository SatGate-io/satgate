package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"net/http/httptest"
	"strings"
	"testing"
)

// storeCause stands for text inside a budget store's error that must never
// reach a client.
const storeCause = "PRIVATE_STORE_CAUSE"

// secretPieces are parts of storeCause; none may appear in a client response.
var secretPieces = []string{storeCause, "PRIVATE", "STORE_CAUSE"}

// A budget store's error text never reaches the app, whatever shape the answer
// has. Through the real /mcp path: HTTP 200, the same JSON-RPC code, data
// "delegation_failed", no child token, no part of the store's text anywhere in
// the response, the cause in the server log, and the parent's balance whole.
func TestBudgetStoreErrorNeverReachesTheApp(t *testing.T) {
	type answer struct {
		name        string
		carve       func(*BudgetResult, error) (*BudgetResult, error) // the Spend answer
		childInit   func(context.Context, string, int64) error        // the child-budget creation answer
		want        string                                            // exact client message
		wantLogged  string                                            // level the cause is logged at
		giveBacks   int
		wantSpends  int
		wantInitial int64
	}
	refusalWrapped := fmt.Errorf("%s: %w", storeCause, ErrInsufficientBudget)
	cases := []answer{
		{name: "denied, empty code, error", carve: func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false}, errors.New(storeCause)
		}, want: wantServerSide, wantLogged: "error"},
		{name: "denied, known code, error", carve: func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false, ErrorCode: "budget_exhausted", Remaining: 3, Cost: 40}, errors.New(storeCause)
		}, want: wantServerSide, wantLogged: "error"},
		{name: "denied, known code, wrapped error", carve: func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false, ErrorCode: "insufficient_budget"}, fmt.Errorf("spend: %w", errors.New(storeCause))
		}, want: wantServerSide, wantLogged: "error"},
		{name: "nil result, error", carve: func(*BudgetResult, error) (*BudgetResult, error) { return nil, errors.New(storeCause) },
			want: wantServerSide, wantLogged: "error"},
		{name: "allowed, error", carve: func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: true, Remaining: 60}, errors.New(storeCause)
		}, want: wantServerSide, wantLogged: "error"},
		{name: "error wrapping ErrInsufficientBudget, nil result", carve: func(*BudgetResult, error) (*BudgetResult, error) { return nil, refusalWrapped },
			want: "insufficient parent budget", wantLogged: "warn"},
		{name: "error wrapping ErrInsufficientBudget, allowed true", carve: func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: true}, refusalWrapped
		}, want: "insufficient parent budget", wantLogged: "warn"},
		{name: "child budget creation: error", childInit: func(context.Context, string, int64) error { return errors.New(storeCause) },
			want: wantServerSide, wantLogged: "error", giveBacks: 1},
		{name: "child budget creation: wrapped error", childInit: func(context.Context, string, int64) error {
			return fmt.Errorf("initialize: %w", errors.New(storeCause))
		}, want: wantServerSide, wantLogged: "error", giveBacks: 1},
		{name: "child budget creation: error wrapping ErrInsufficientBudget", childInit: func(context.Context, string, int64) error { return refusalWrapped },
			want: "insufficient parent budget", wantLogged: "warn", giveBacks: 1},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logs := gbLogs(t)
			s, p := newHeaderModeStreamable(t)
			svc := gbService(t)
			parent := gbParent(t, svc)
			p.SetAuthenticator(gbAuth{parent})
			b := newGBBudget()
			_ = b.InMemoryBudgetEnforcer.Initialize(context.Background(), parent.BudgetID, 100)
			b.spendResult = tc.carve
			b.initChild = tc.childInit
			p.delegator = NewDelegator(svc, b)
			p.SetBudgetEnforcer(b)
			srv := httptest.NewServer(s.Handler())
			defer srv.Close()
			sid := mustInitSession(t, srv.URL, "good-token")
			status, _, body := streamablePost(t, srv.URL+"/mcp", sid, "good-token", map[string]any{"jsonrpc": "2.0", "id": 2, "method": "satgate/delegate", "params": map[string]any{"budget": 40}})

			if status != 200 {
				t.Fatalf("HTTP %d: %s", status, body)
			}
			for _, want := range []string{`"code":-32000`, `"delegation_failed"`, `"message":"` + tc.want + `"`} {
				if !strings.Contains(body, want) {
					t.Fatalf("response lacks %s: %s", want, body)
				}
			}
			if strings.Contains(body, `"token":`) || strings.Contains(body, `"result"`) {
				t.Fatalf("a child token was issued: %s", body)
			}
			for _, piece := range secretPieces {
				if strings.Contains(body, piece) {
					t.Fatalf("response contains %q: %s", piece, body)
				}
			}
			if b.refundCalls != tc.giveBacks {
				t.Fatalf("give-backs %d, want %d", b.refundCalls, tc.giveBacks)
			}
			if got := gbRemaining(t, b, parent.BudgetID); got != 100 {
				t.Fatalf("parent balance %d, want 100", got)
			}
			out := logs.String()
			if !strings.Contains(out, storeCause) {
				t.Fatalf("the cause was not logged: %s", out)
			}
			if !strings.Contains(out, `"level":"`+tc.wantLogged+`"`) || !strings.Contains(out, `"parent":"parent-token"`) || !strings.Contains(out, `"budgetId":"parent"`) {
				t.Fatalf("log lacks level %s, parent token ID or budget ID: %s", tc.wantLogged, out)
			}
			if strings.Contains(out, "eyJ") {
				t.Fatalf("log contains token text: %s", out)
			}
		})
	}
}

// The real refusal keeps its text, over /mcp: built from the store's code and
// numbers, not from the store's error text.
func TestRealBudgetRefusalKeepsItsTextOverMCP(t *testing.T) {
	for _, tc := range []struct {
		name  string
		carve func(*BudgetResult, error) (*BudgetResult, error)
		want  string
	}{
		{"code, no error", func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false, ErrorCode: "budget_exhausted"}, nil
		}, "insufficient parent budget: budget_exhausted"},
		{"no code, no error", func(*BudgetResult, error) (*BudgetResult, error) { return &BudgetResult{Allowed: false}, nil },
			"insufficient parent budget: not allowed"},
		{"refusal result with ErrInsufficientBudget", func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false, ErrorCode: "budget_exhausted", Remaining: 10, Cost: 40}, fmt.Errorf("%s: %w", storeCause, ErrInsufficientBudget)
		}, "insufficient parent budget: budget exhausted: 10 remaining, 40 required"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			s, p := newHeaderModeStreamable(t)
			svc := gbService(t)
			parent := gbParent(t, svc)
			p.SetAuthenticator(gbAuth{parent})
			b := newGBBudget()
			_ = b.InMemoryBudgetEnforcer.Initialize(context.Background(), parent.BudgetID, 100)
			b.spendResult = tc.carve
			p.delegator = NewDelegator(svc, b)
			p.SetBudgetEnforcer(b)
			srv := httptest.NewServer(s.Handler())
			defer srv.Close()
			sid := mustInitSession(t, srv.URL, "good-token")
			status, _, body := streamablePost(t, srv.URL+"/mcp", sid, "good-token", map[string]any{"jsonrpc": "2.0", "id": 2, "method": "satgate/delegate", "params": map[string]any{"budget": 40}})
			if status != 200 || !strings.Contains(body, `"message":"`+tc.want+`"`) || !strings.Contains(body, `"delegation_failed"`) || strings.Contains(body, storeCause) {
				t.Fatalf("status %d body %s", status, body)
			}
			if b.refundCalls != 0 || gbRemaining(t, b, parent.BudgetID) != 100 {
				t.Fatalf("give-backs %d, parent %d", b.refundCalls, gbRemaining(t, b, parent.BudgetID))
			}
		})
	}
}

// HandleDelegate sends no error's text to the client: a Delegate error that is
// not one of this package's own refusals gets the server-side sentence, even if
// it wraps ErrChildTokenFailed around a cause.
func TestHandleDelegateNeverSendsAnErrorValuesText(t *testing.T) {
	d := &Delegator{}
	for _, err := range []error{
		errors.New(storeCause),
		fmt.Errorf("%w: %s", ErrChildTokenFailed, storeCause),
		fmt.Errorf("%s: %w", storeCause, ErrChildTokenFailed),
		fmt.Errorf("%w: %s", ErrParentCannotDelegate, storeCause),
		fmt.Errorf("%s: %w", storeCause, ErrInsufficientBudget),
	} {
		got := delegationClientMessage(err)
		if got != wantServerSide && got != wantCannotDelegate {
			t.Fatalf("%v -> %q", err, got)
		}
		for _, piece := range secretPieces {
			if strings.Contains(got, piece) {
				t.Fatalf("%v -> %q", err, got)
			}
		}
	}
	_ = d
}
