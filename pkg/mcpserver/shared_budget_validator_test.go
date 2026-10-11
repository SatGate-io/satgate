package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// newTestSpendGate is the spend gate with a validator that honours every
// sealed reference: the tests that are about counting, not about the owner's
// saved record, use it. The tests of the validator itself (round 3) build the
// gate with NewSpendGate and set their own.
func newTestSpendGate(store toolcontrols.SpendStore) *SpendGate {
	g := NewSpendGate(store)
	g.SharedBudgetValidator = func(context.Context, string, string, []toolcontrols.SpendLimit, func(string) bool, string) error {
		return nil
	}
	return g
}

// The validator is called for every sealed budget reference before any counter
// is touched. Each outcome below is checked against a store that counts every
// call: a refusal must leave it at zero.
func TestSharedBudgetValidatorRunsBeforeAnyCounter(t *testing.T) {
	type outcome struct {
		name      string
		validator SharedBudgetValidator // nil: none configured
		want      string                // "" means the order is admitted
	}
	var calls atomic.Int64
	cases := []outcome{
		{"a validator that accepts", func(context.Context, string, string, []toolcontrols.SpendLimit, func(string) bool, string) error {
			calls.Add(1)
			return nil
		}, ""},
		{"a validator that says unavailable", func(context.Context, string, string, []toolcontrols.SpendLimit, func(string) bool, string) error {
			calls.Add(1)
			return ErrSharedBudgetUnavailable
		}, MsgSharedBudgetUnavailable},
		{"a wrapped unavailable", func(context.Context, string, string, []toolcontrols.SpendLimit, func(string) bool, string) error {
			calls.Add(1)
			return fmt.Errorf("lookup: %w", ErrSharedBudgetUnavailable)
		}, MsgSharedBudgetUnavailable},
		{"a validator that fails", func(context.Context, string, string, []toolcontrols.SpendLimit, func(string) bool, string) error {
			calls.Add(1)
			return errors.New("database down")
		}, MsgSharedBudgetCheckFailed},
		{"no validator configured", nil, MsgSharedBudgetUnavailable},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			calls.Store(0)
			proxy, _, _ := newArgProxy(t)
			proxy.SetUpstreamRouter(&scriptedRouter{})
			proxy.SetEvidenceRecorder(&lockedRecorder{})
			cs := &countingStore{SpendStore: toolcontrols.NewMemoryStore(nil)}
			g := NewSpendGate(cs)
			g.SharedBudgetValidator = c.validator
			proxy.SetCallGate(g)
			_, tok := sealedBudgetToken(t, "tenant-1", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))

			resp, err := proxy.handleRequest(context.Background(), argCall(1, tok, "place_crypto_order", orderSym("BTC", "10")))
			if err != nil || resp == nil {
				t.Fatalf("handleRequest: %v %v", resp, err)
			}
			if c.want == "" {
				if resp.Error != nil || cs.touched() == 0 {
					t.Fatalf("an accepted reference was not counted: err=%v touched=%d", resp.Error, cs.touched())
				}
				return
			}
			if resp.Error == nil || !strings.Contains(resp.Error.Message, c.want) {
				t.Fatalf("not refused with %q: %+v", c.want, resp.Error)
			}
			if n := cs.touched(); n != 0 {
				t.Fatalf("%d store call(s) before the refusal", n)
			}
			if c.validator != nil && calls.Load() == 0 {
				t.Fatal("the validator was never asked")
			}
		})
	}
}

// The validator is asked once per budget the token names, with the tenant, the
// budget id and the whole group of sealed limits for that id: the limits that
// do not cover the call too, and the token's own tool check.
func TestSharedBudgetValidatorSeesTheWholeGroup(t *testing.T) {
	type ask struct {
		tenant, id, tool string
		n                int
		crypto           bool
		equity           bool
	}
	run := func(t *testing.T, tool, args string, docs ...string) []ask {
		proxy, _, _ := newArgProxy(t)
		proxy.SetUpstreamRouter(&scriptedRouter{})
		proxy.SetEvidenceRecorder(&lockedRecorder{})
		g := NewSpendGate(toolcontrols.NewMemoryStore(nil))
		var asks []ask
		g.SharedBudgetValidator = func(_ context.Context, tenant, id string, group []toolcontrols.SpendLimit, allows func(string) bool, tool string) error {
			asks = append(asks, ask{tenant, id, tool, len(group), allows("place_crypto_order"), allows("place_equity_order")})
			return nil
		}
		proxy.SetCallGate(g)
		scope := bothTools
		_, tok := sealedBudgetToken(t, "tenant-7", "own-1", scope, docs...)
		if resp, err := proxy.handleRequest(context.Background(), argCall(1, tok, tool, args)); err != nil || resp.Error != nil {
			t.Fatalf("call: %v %v", resp, err)
		}
		return asks
	}
	t.Run("a covered order: one ask with the total and the cap", func(t *testing.T) {
		asks := run(t, "place_crypto_order", orderSym("BTC", "10"), budgetDoc(budgetA, "200", "50"))
		if len(asks) != 1 || asks[0] != (ask{"tenant-7", budgetA, "place_crypto_order", 2, true, true}) {
			t.Fatalf("asks = %+v", asks)
		}
	})
	t.Run("a call the budget does not cover still asks, with the group", func(t *testing.T) {
		asks := run(t, "place_equity_order", orderSym("AAPL", "10"), budgetDoc(budgetA, "200", "50"))
		if len(asks) != 1 || asks[0].n != 2 || asks[0].tool != "place_equity_order" {
			t.Fatalf("asks = %+v", asks)
		}
	})
	t.Run("two budgets: one ask each, each with its own limits", func(t *testing.T) {
		asks := run(t, "place_crypto_order", orderSym("BTC", "10"), budgetDoc(budgetA, "200", "50"), budgetDoc(budgetB, "300", ""))
		if len(asks) != 2 || asks[0].id != budgetA || asks[0].n != 2 || asks[1].id != budgetB || asks[1].n != 1 {
			t.Fatalf("asks = %+v", asks)
		}
	})
	t.Run("a token with no budget asks nothing", func(t *testing.T) {
		if asks := run(t, "place_crypto_order", orderSym("BTC", "10")); len(asks) != 0 {
			t.Fatalf("asks = %+v", asks)
		}
	})
}
