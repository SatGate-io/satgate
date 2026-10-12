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

// A refusal that comes with no error keeps its text, over /mcp: built from the
// store's code, not from the store's error text. (An error that matches
// ErrInsufficientBudget gets the one fixed sentence instead; see
// TestTypedBudgetRefusalAlwaysGetsTheFixedSentence.)
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

// Delegate itself returns only text this package wrote: for every store answer
// the error's text is a fixed sentence or built from the store's code and
// numbers, never the store's own error text, and the error does not wrap it.
func TestDelegateErrorsCarryNoStoreText(t *testing.T) {
	ctx := context.Background()
	refusalWrapped := fmt.Errorf("%s: %w", storeCause, ErrInsufficientBudget)
	for _, tc := range []struct {
		name      string
		carve     func(*BudgetResult, error) (*BudgetResult, error)
		childInit func(context.Context, string, int64) error
		want      string
	}{
		{"denied, empty code, error", func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false}, errors.New(storeCause)
		}, nil, wantServerSide},
		{"denied, known code, error", func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false, ErrorCode: "insufficient_budget"}, errors.New(storeCause)
		}, nil, wantServerSide},
		{"nil result, error", func(*BudgetResult, error) (*BudgetResult, error) { return nil, errors.New(storeCause) }, nil, wantServerSide},
		{"allowed, error", func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: true}, errors.New(storeCause)
		}, nil, wantServerSide},
		{"ErrInsufficientBudget, nil result", func(*BudgetResult, error) (*BudgetResult, error) { return nil, refusalWrapped }, nil, "insufficient parent budget"},
		{"ErrInsufficientBudget, refusal result", func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false, ErrorCode: "insufficient_budget"}, refusalWrapped
		}, nil, "insufficient parent budget"},
		{"no error, refusal result with a code", func(*BudgetResult, error) (*BudgetResult, error) {
			return &BudgetResult{Allowed: false, ErrorCode: "insufficient_budget"}, nil
		}, nil, "insufficient parent budget: insufficient_budget"},
		{"child budget creation: error", nil, func(context.Context, string, int64) error { return errors.New(storeCause) }, wantServerSide},
		{"child budget creation: ErrInsufficientBudget", nil, func(context.Context, string, int64) error { return refusalWrapped }, "insufficient parent budget"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			svc := gbService(t)
			parent := gbParent(t, svc)
			b := newGBBudget()
			_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
			b.spendResult, b.initChild = tc.carve, tc.childInit
			_, err := NewDelegator(svc, b).Delegate(ctx, parent, &DelegateParams{Budget: 40})
			if err == nil || err.Error() != tc.want {
				t.Fatalf("error %v, want text %q", err, tc.want)
			}
			if errors.Unwrap(err) != nil || errors.Is(err, errors.New(storeCause)) {
				t.Fatalf("the returned error wraps another: %v", errors.Unwrap(err))
			}
		})
	}
}

// A parent token that cannot be read is an unexpected failure: the app gets the
// server-side sentence, the decode error goes to the log, nothing is carved.
func TestUnreadableParentGetsTheServerSentence(t *testing.T) {
	logs := gbLogs(t)
	ctx := context.Background()
	svc := gbService(t)
	parent := gbParent(t, svc)
	parent.RawToken = "not-a-token-" + storeCause
	b := newGBBudget()
	_ = b.InMemoryBudgetEnforcer.Initialize(ctx, parent.BudgetID, 100)
	_, err := NewDelegator(svc, b).Delegate(ctx, parent, &DelegateParams{Budget: 40})
	if err == nil || err.Error() != wantServerSide {
		t.Fatalf("error %v", err)
	}
	if b.parentSpends != 0 || b.refundCalls != 0 || gbRemaining(t, b, parent.BudgetID) != 100 {
		t.Fatalf("spends %d give-backs %d", b.parentSpends, b.refundCalls)
	}
	if !strings.Contains(logs.String(), `"level":"error"`) {
		t.Fatalf("not logged: %s", logs.String())
	}
}

// Whenever the budget store's error matches ErrInsufficientBudget, the app gets
// the one fixed sentence "insufficient parent budget": whatever else the result
// holds (any recognised refusal code, numbers, Allowed true), and however the
// error is wrapped. Through the real /mcp path: HTTP 200, the same JSON-RPC
// code, data "delegation_failed", no digits and no code text, no child token,
// the parent's balance whole, and the cause in the server log at warn level.
func TestTypedBudgetRefusalAlwaysGetsTheFixedSentence(t *testing.T) {
	typed := fmt.Errorf("%s: %w", storeCause, ErrInsufficientBudget)
	type answer struct {
		name      string
		carve     func(*BudgetResult, error) (*BudgetResult, error)
		childInit func(context.Context, string, int64) error
		giveBacks int
	}
	var cases []answer
	add := func(name string, res *BudgetResult, err error) {
		cases = append(cases, answer{name: name, carve: func(*BudgetResult, error) (*BudgetResult, error) { return res, err }})
	}
	add("typed error alone", nil, typed)
	add("typed error, result allowed", &BudgetResult{Allowed: true, Remaining: 60}, typed)
	add("typed error, denied with no code", &BudgetResult{Allowed: false, Remaining: 17, Cost: 40}, typed)
	for _, code := range []string{"budget_exhausted", "insufficient_budget", "budget_not_initialized", "token_not_found"} {
		add("typed error with code "+code, &BudgetResult{Allowed: false, ErrorCode: code, Remaining: 17, Cost: 40}, typed)
	}
	add("typed error with an unrecognised code", &BudgetResult{Allowed: false, ErrorCode: "weird_code", Remaining: 17, Cost: 40}, typed)
	add("typed error wrapped once", &BudgetResult{Allowed: false, ErrorCode: "budget_exhausted", Remaining: 17, Cost: 40}, fmt.Errorf("spend: %w", typed))
	add("typed error wrapped twice, nil result", nil, fmt.Errorf("carve: %w", fmt.Errorf("spend: %w", typed)))
	add("typed error joined with another", &BudgetResult{Allowed: false, ErrorCode: "budget_exhausted", Remaining: 17, Cost: 40}, errors.Join(errors.New("other"), typed))
	add("the in-memory store's own refusal", &BudgetResult{Allowed: false, ErrorCode: "budget_exhausted", Remaining: 17, Cost: 40}, &budgetRefusal{msg: "budget exhausted: 17 remaining, 40 required: " + storeCause})
	cases = append(cases,
		answer{name: "child budget creation: typed error", childInit: func(context.Context, string, int64) error { return typed }, giveBacks: 1},
		answer{name: "child budget creation: typed error wrapped", childInit: func(context.Context, string, int64) error {
			return fmt.Errorf("transfer: %w", fmt.Errorf("initialize: %w", typed))
		}, giveBacks: 1},
	)
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			logs := gbLogs(t)
			s, p := newHeaderModeStreamable(t)
			svc := gbService(t)
			parent := gbParent(t, svc)
			p.SetAuthenticator(gbAuth{parent})
			b := newGBBudget()
			_ = b.InMemoryBudgetEnforcer.Initialize(context.Background(), parent.BudgetID, 100)
			b.spendResult, b.initChild = tc.carve, tc.childInit
			p.delegator = NewDelegator(svc, b)
			p.SetBudgetEnforcer(b)
			srv := httptest.NewServer(s.Handler())
			defer srv.Close()
			sid := mustInitSession(t, srv.URL, "good-token")
			status, _, body := streamablePost(t, srv.URL+"/mcp", sid, "good-token", map[string]any{"jsonrpc": "2.0", "id": 2, "method": "satgate/delegate", "params": map[string]any{"budget": 40}})
			if status != 200 {
				t.Fatalf("HTTP %d: %s", status, body)
			}
			const want = `{"jsonrpc":"2.0","id":2,"error":{"code":-32000,"message":"insufficient parent budget","data":{"error":"delegation_failed"}}}`
			if strings.TrimSpace(body) != want && !strings.Contains(body, want) {
				t.Fatalf("response is not exactly the fixed refusal: %s", body)
			}
			for _, bad := range []string{"17", "40", "remaining", "required", "exhausted", "not allowed", "insufficient_budget", "not_initialized", "token_not_found", "weird_code", `"token":`, `"result"`} {
				if strings.Contains(body, bad) {
					t.Fatalf("response contains %q: %s", bad, body)
				}
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
			if !strings.Contains(out, `"level":"warn"`) || !strings.Contains(out, `"parent":"parent-token"`) || !strings.Contains(out, `"budgetId":"parent"`) {
				t.Fatalf("no warning with the parent token ID and budget ID: %s", out)
			}
			if !strings.Contains(out, "PRIVATE_STORE_CAUSE") && tc.name != "the in-memory store's own refusal" {
				t.Fatalf("the cause was not logged: %s", out)
			}
			if tc.name == "the in-memory store's own refusal" && !strings.Contains(out, "17 remaining, 40 required") {
				t.Fatalf("the cause was not logged: %s", out)
			}
			if strings.Contains(out, "eyJ") {
				t.Fatalf("log contains token text: %s", out)
			}
		})
	}
}
