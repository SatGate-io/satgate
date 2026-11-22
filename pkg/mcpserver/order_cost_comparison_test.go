package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math/big"
	"slices"
	"testing"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

const comparisonService = "test-route"

func comparisonRouter(p *Proxy, service string) *originRouter {
	r := &originRouter{origins: map[string]ToolOrigin{
		ocCrypto: {Listed: true, Service: service, Original: ocCrypto},
	}}
	p.SetUpstreamRouter(r)
	return r
}

func comparisonBump(t *testing.T) OrderCostComparison {
	t.Helper()
	return func(_ context.Context, origin ToolOrigin, tool string, args map[string]json.RawMessage, base argrules.OrderCost) (*big.Rat, bool, error) {
		if !origin.Listed || origin.Err != nil || origin.Service != comparisonService {
			return nil, false, nil
		}
		if origin.Original != ocCrypto || tool != ocCrypto || string(args["type"]) != `"market"` {
			t.Fatalf("wrong call for comparison: %v %q %s", origin, tool, args["type"])
		}
		var amount string
		if err := json.Unmarshal(args["dollar_amount"], &amount); err != nil {
			t.Fatal(err)
		}
		d, err := argrules.ParseDecimal(amount)
		if err != nil {
			t.Fatal(err)
		}
		u, ok := argrules.UnitsCeil(d.Rat())
		if !ok || u != base.Units {
			t.Fatalf("base units = %d, want %d", base.Units, u)
		}
		// ceil(amount * 101) gives the adjusted number of cents.
		scaled := new(big.Rat).Mul(d.Rat(), big.NewRat(101, 1))
		cents := new(big.Int).Div(new(big.Int).Neg(scaled.Num()), scaled.Denom())
		// Even if a hook edits the buffer it receives, it cannot rewrite the call.
		args["dollar_amount"][1] = '0'
		delete(args, "type")
		return new(big.Rat).SetFrac(cents.Neg(cents), big.NewInt(100)), true, nil
	}
}

func TestOrderCostComparisonPerOrder(t *testing.T) {
	const capDoc = `{"v":1,"rules":[{"tool":"place_crypto_order","shapes":[{"order":{"max_cost":"100"}}]}]}`
	for _, c := range []struct {
		name, service string
		enabled       bool
		deny100       bool
	}{
		{"matching-route", comparisonService, true, true},
		{"same-tool-different-route", "other-route", true, false},
		{"no-hook", comparisonService, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			p, _, _ := newArgProxy(t)
			rr := comparisonRouter(p, c.service)
			if c.enabled {
				p.SetOrderCostComparison(comparisonBump(t))
			}
			_, token := argToken(t, ocCrypto, capDoc)
			for i, amount := range []string{"100.00", "99.00"} {
				args := order(amount)
				out := runCall(t, p, argCall(i+1, token, ocCrypto, args))
				if amount == "100.00" && c.deny100 {
					assertArgDenied(t, out, ocCrypto, "order")
					if rr.forwarded() != 0 {
						t.Fatal("refused order was forwarded")
					}
					continue
				}
				if out.denied {
					t.Fatalf("%s: %v", amount, out.resp.Error)
				}
				if got, want := rr.seen[len(rr.seen)-1], string(ocParams(ocCrypto, args)); got != want {
					t.Fatalf("forwarded params = %s, want %s", got, want)
				}
			}
		})
	}
}

func TestOrderCostComparisonApproval(t *testing.T) {
	for _, c := range []struct {
		name, service string
		enabled       bool
		hold          bool
	}{
		{"matching-route", comparisonService, true, true},
		{"same-tool-different-route", "other-route", true, false},
		{"no-hook", comparisonService, false, false},
	} {
		t.Run(c.name, func(t *testing.T) {
			approvalDoc := ocApprovalDoc(ocCrypto, "50")
			a := newApprovalHarness(t, approvalDoc)
			_, a.token = sealedBudgetToken(t, "test-tenant", "test-budget", ocCrypto, approvalDoc, ocAllLimitsDoc(ocCrypto, "100"))
			rr := comparisonRouter(a.proxy, c.service)
			if c.enabled {
				a.gate.OrderCostComparison = comparisonBump(t)
			}
			if r := a.call(t, 1, ocCrypto, order("49.50")); r.Error != nil {
				t.Fatalf("equal to the adjusted threshold: %v", r.Error)
			}
			args := order("49.51")
			r := a.call(t, 2, ocCrypto, args)
			if !c.hold {
				if r.Error != nil || rr.forwarded() != 2 || len(a.approvals.Rows()) != 0 {
					t.Fatalf("unexpected hold: %v", r.Error)
				}
				assertComparisonSpend(t, a.store, 9901*(units/100))
				return
			}
			if got := code(t, r); got != toolcontrols.DenialCodeApprovalRequired || rr.forwarded() != 1 {
				t.Fatalf("49.51: code = %q, forwarded = %d", got, rr.forwarded())
			}
			assertComparisonSpend(t, a.store, 4950*(units/100))
			row := a.onlyApproval(t)
			params := ocParams(ocCrypto, args)
			wantHash, err := toolcontrols.CallHash(row.TokenID, ocCrypto, params)
			if err != nil || row.CallHash != wantHash {
				t.Fatalf("hold did not hash the original call: %v", err)
			}
			wantSummary := append([]toolcontrols.SummaryItem{{Name: "Counts as at most", Value: "$49.51"}}, toolcontrols.OwnerSummary(params, "dollar_amount")...)
			if !slices.Equal(row.Summary, wantSummary) || a.notifier.count() != 1 || !slices.Equal(a.notifier.notices[0].Summary, wantSummary) {
				t.Fatalf("hold did not summarize the original call: %v", row.Summary)
			}
			if err := a.approvals.Approve(row.TenantID, row.ID, "test-reviewer", a.clock); err != nil {
				t.Fatal(err)
			}
			if r := a.call(t, 3, ocCrypto, args); r.Error != nil || rr.forwarded() != 2 || rr.seen[1] != string(params) {
				t.Fatalf("original retry not forwarded intact: %v, %v", r.Error, rr.seen)
			}
			assertComparisonSpend(t, a.store, 9901*(units/100))
		})
	}
}

func TestOrderCostComparisonFallback(t *testing.T) {
	for _, c := range []struct {
		name string
		cost *big.Rat
		ok   bool
		err  error
	}{
		{"lower", big.NewRat(1, 1), true, nil},
		{"error", big.NewRat(1000, 1), true, errors.New("test adjustment unavailable")},
		{"not-applied", big.NewRat(1000, 1), false, nil},
		{"overflow", new(big.Rat).SetInt(new(big.Int).Lsh(big.NewInt(1), 64)), true, nil},
		{"nil-amount", nil, true, nil},
	} {
		t.Run(c.name, func(t *testing.T) {
			const capDoc = `{"v":1,"rules":[{"tool":"place_crypto_order","shapes":[{"order":{"max_cost":"100"}}]}]}`
			p, _, _ := newArgProxy(t)
			comparisonRouter(p, comparisonService)
			calls := 0
			hook := func(context.Context, ToolOrigin, string, map[string]json.RawMessage, argrules.OrderCost) (*big.Rat, bool, error) {
				calls++
				return c.cost, c.ok, c.err
			}
			p.SetOrderCostComparison(hook)
			_, token := argToken(t, ocCrypto, capDoc)
			if r := runCall(t, p, argCall(1, token, ocCrypto, order("100"))); r.denied {
				t.Fatalf("fallback tightened cap: %v", r.resp.Error)
			}
			assertArgDenied(t, runCall(t, p, argCall(2, token, ocCrypto, order("100.01"))), ocCrypto, "order")

			a := newApprovalHarness(t, ocApprovalDoc(ocCrypto, "50"))
			comparisonRouter(a.proxy, comparisonService)
			a.gate.OrderCostComparison = hook
			if r := a.call(t, 1, ocCrypto, order("49.51")); r.Error != nil {
				t.Fatalf("fallback tightened approval: %v", r.Error)
			}
			if got := code(t, a.call(t, 2, ocCrypto, order("50.01"))); got != toolcontrols.DenialCodeApprovalRequired {
				t.Fatalf("fallback widened approval: %q", got)
			}
			if calls != 4 {
				t.Fatalf("hook called %d times, want 4", calls)
			}
		})
	}
}

func TestOrderCostComparisonNotUsedForSpendOrEventCost(t *testing.T) {
	unexpected := func(t *testing.T) OrderCostComparison {
		return func(context.Context, ToolOrigin, string, map[string]json.RawMessage, argrules.OrderCost) (*big.Rat, bool, error) {
			t.Fatal("comparison hook called outside a dollar-order comparison")
			return nil, false, nil
		}
	}
	t.Run("dollar-spend", func(t *testing.T) {
		h := newSpendHarness(t, ocLimitDoc(0, ocCrypto, "99"))
		_, h.token = sealedBudgetToken(t, "test-tenant", "test-budget", ocCrypto, ocAllLimitsDoc(ocCrypto, "99"))
		hook := unexpected(t)
		comparisonRouter(h.proxy, comparisonService)
		h.proxy.SetOrderCostComparison(hook)
		ag := NewApprovalGate(toolcontrols.NewMemoryApprovalStore(), nil)
		ag.OrderCostComparison = hook
		h.proxy.SetCallGate(ChainGates(newTestSpendGate(h.store), ag))
		if r := h.call(t, 1, ocCrypto, ocArgs(ocCrypto, `"side":"buy","type":"market","dollar_amount":"99.00"`)); r.Error != nil {
			t.Fatalf("spend did not fit at original amount: %v", r.Error)
		}
		assertComparisonSpend(t, h.store, 99*units)
	})
	t.Run("event-cost", func(t *testing.T) {
		hook := unexpected(t)
		doc := fmt.Sprintf(approveDoc, "1")
		h := newEvHarness(t, doc, fmt.Sprintf(capRuleDoc, `,"max_cost":"2"`), &evOpts{
			gate: func(store *toolcontrols.MemoryStore, approvals *toolcontrols.MemoryApprovalStore, schema string) CallGate {
				ag := NewApprovalGate(approvals, &fakeNotifier{})
				ag.Synchronous, ag.OrderCostComparison = true, hook
				return ChainGates(NewEventGate(store, map[string]string{schema: "venue"}), ag)
			},
		})
		h.proxy.SetOrderCostComparison(hook)
		if got := code(t, h.call(t, evBuy("MKT-A1", "2", "0.5"))); got != "" {
			t.Fatalf("event at approval threshold: %q", got)
		}
		if got := code(t, h.call(t, evBuy("MKT-A1", "3", "0.5"))); got != toolcontrols.DenialCodeApprovalRequired {
			t.Fatalf("event above approval threshold: %q", got)
		}
		if got := code(t, h.call(t, evBuy("MKT-A1", "5", "0.5"))); got != argrules.DenialCode {
			t.Fatalf("event above per-order cap: %q", got)
		}
	})
}

func assertComparisonSpend(t *testing.T, store *toolcontrols.MemoryStore, want int64) {
	t.Helper()
	totals := ocTotals(store)
	if len(totals) != len(ocLimitKinds) || !ocAllEqual(totals, want) {
		t.Fatalf("spend totals = %v, want %d units each", totals, want)
	}
}
