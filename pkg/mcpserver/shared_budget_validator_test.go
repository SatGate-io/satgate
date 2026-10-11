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
	g.SharedBudgetValidator = func(context.Context, string, toolcontrols.SpendLimit) error { return nil }
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
		{"a validator that accepts", func(context.Context, string, toolcontrols.SpendLimit) error { calls.Add(1); return nil }, ""},
		{"a validator that says unavailable", func(context.Context, string, toolcontrols.SpendLimit) error {
			calls.Add(1)
			return ErrSharedBudgetUnavailable
		}, MsgSharedBudgetUnavailable},
		{"a wrapped unavailable", func(context.Context, string, toolcontrols.SpendLimit) error {
			calls.Add(1)
			return fmt.Errorf("lookup: %w", ErrSharedBudgetUnavailable)
		}, MsgSharedBudgetUnavailable},
		{"a validator that fails", func(context.Context, string, toolcontrols.SpendLimit) error {
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

// The validator sees the token's tenant and the sealed limit, once per
// reference, and is not asked about a limit with no budget.
func TestSharedBudgetValidatorSeesTenantAndLimit(t *testing.T) {
	proxy, _, _ := newArgProxy(t)
	proxy.SetUpstreamRouter(&scriptedRouter{})
	proxy.SetEvidenceRecorder(&lockedRecorder{})
	g := NewSpendGate(toolcontrols.NewMemoryStore(nil))
	var seen []toolcontrols.SpendLimit
	var tenants []string
	g.SharedBudgetValidator = func(_ context.Context, tenant string, l toolcontrols.SpendLimit) error {
		seen, tenants = append(seen, l), append(tenants, tenant)
		return nil
	}
	proxy.SetCallGate(g)
	_, tok := sealedBudgetToken(t, "tenant-7", "own-1", bothTools, budgetDoc(budgetA, "200", "50"))
	if resp, err := proxy.handleRequest(context.Background(), argCall(1, tok, "place_crypto_order", orderSym("BTC", "10"))); err != nil || resp.Error != nil {
		t.Fatalf("order: %v %v", resp, err)
	}
	if len(seen) != 2 {
		t.Fatalf("validator saw %d limits, want the total and the BTC cap: %+v", len(seen), seen)
	}
	for i, l := range seen {
		if tenants[i] != "tenant-7" || l.Budget != budgetA {
			t.Fatalf("saw tenant %q budget %q", tenants[i], l.Budget)
		}
	}
}
