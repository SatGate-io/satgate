package mcpserver

import (
	"context"
	"errors"

	"strings"
	"testing"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

func sellTestHook(t *testing.T, g *SpendGate, exempt bool, err error) {
	t.Helper()
	g.DollarBudgetExempt = func(context.Context, GateCall, toolcontrols.DollarBudgetOrder) (bool, error) {
		return exempt, err
	}
}

type sellStore struct {
	*toolcontrols.MemoryStore
	reserves, releases, reads int
}

func (s *sellStore) Reserve(ctx context.Context, r toolcontrols.Reservation) (toolcontrols.ReserveResult, error) {
	s.reserves++
	return s.MemoryStore.Reserve(ctx, r)
}
func (s *sellStore) Release(ctx context.Context, r toolcontrols.Reservation) error {
	s.releases++
	return s.MemoryStore.Release(ctx, r)
}
func (s *sellStore) Total(ctx context.Context, key string) (int64, error) {
	s.reads++
	return s.MemoryStore.Total(ctx, key)
}
func sellCall(tok *TokenInfo, tool, args string) GateCall {
	return GateCall{Token: tok, Tool: tool, Params: ocParams(tool, args), RequestID: "sell-test", Now: ocNow,
		Origin: ToolOrigin{Listed: true, Service: "robinhood", Original: tool}}
}
func sellGate(t *testing.T) (*SpendGate, *sellStore) {
	t.Helper()
	s := &sellStore{MemoryStore: toolcontrols.NewMemoryStore(ocClock)}
	g := newTestSpendGate(s)
	sellTestHook(t, g, true, nil)
	return g, s
}
func sellAdmit(t *testing.T, g CallGate, c GateCall) *GateAdmission {
	t.Helper()
	a, e := g.Admit(context.Background(), c)
	if e != nil {
		t.Fatal(e)
	}
	return a
}

func TestSellDollarBudgetFullWindows(t *testing.T) {
	cases := []struct{ tool, args string }{
		{ocEquity, `{"symbol":"AAPL","side":"sell","dollar_amount":"20"}`},
		{ocCrypto, `{"symbol":"BTC","side":"sell","dollar_amount":"20"}`},
		{ocEquity, `{"symbol":"AAPL","side":"SeLl","quantity":"2","limit_price":"10","type":"limit","time_in_force":"gfd"}`},
		{ocCrypto, `{"symbol":"BTC","side":"SELL","quantity":"0.01"}`},
	}
	for _, tc := range cases {
		for _, kind := range ocLimitKinds {
			for _, legacy := range []bool{false, true} {
				name := tc.tool + "/" + kind.name + "/" + tc.args
				if legacy {
					name += "/legacy"
				}
				t.Run(name, func(t *testing.T) {
					entry := kind.entry(tc.tool, "100")
					if legacy {
						entry = strings.ReplaceAll(entry, `,"value":"order_max_cost"`, "")
					}
					tok := ocToken(t, `{"v":1,"spend_limits":[`+entry+`]}`)
					g, s := sellGate(t)
					for _, l := range tok.SpendLimits {
						key := counterKey(tok.TenantID, counterName(l), windowIDOf(l, ocNow))
						_, err := s.MemoryStore.Reserve(context.Background(), toolcontrols.Reservation{Key: key, CallID: "seed", Units: l.MaxUnits(), Max: l.MaxUnits(), ExpiresAt: ocNow.Add(counterGrace)})
						if err != nil {
							t.Fatal(err)
						}
					}
					c := sellCall(tok, tc.tool, tc.args)
					a := sellAdmit(t, g, c)
					if ocRefusalCode(a) != "" {
						t.Errorf("sell refused at full dollars: %+v", a.Refusal)
					}
					if a != nil && a.Settle != nil {
						a.Settle(context.Background(), GateOutcome{})
					}
					if s.reserves != 0 || s.releases != 0 || s.reads != 0 {
						t.Errorf("dollar store calls: Reserve=%d Release=%d Total=%d", s.reserves, s.releases, s.reads)
					}
					if !ocAllEqual(ocTotals(s.MemoryStore), 100*units) {
						t.Errorf("balances changed: %v", ocTotals(s.MemoryStore))
					}
				})
			}
		}
	}
}

func TestSellDollarBudgetNoReserveOrRelease(t *testing.T) {
	for _, outcome := range []GateOutcome{{Succeeded: true}, {}, {Unknown: true}} {
		t.Run(strings.ReplaceAll(strings.TrimSpace(strings.Join([]string{boolName(outcome.Succeeded), boolName(outcome.Unknown)}, "-")), " ", ""), func(t *testing.T) {
			g, s := sellGate(t)
			c := sellCall(ocToken(t, ocAllLimitsDoc(ocEquity, "100")), ocEquity, `{"symbol":"AAPL","side":"sell","dollar_amount":"20"}`)
			a := sellAdmit(t, g, c)
			if ocRefusalCode(a) != "" {
				t.Fatal(a.Refusal)
			}
			if a != nil && a.Settle != nil {
				a.Settle(context.Background(), outcome)
			}
			if s.reserves != 0 || s.releases != 0 || s.reads != 0 || len(s.Keys()) != 0 {
				t.Fatalf("store touched: reserves=%d releases=%d reads=%d keys=%v", s.reserves, s.releases, s.reads, s.Keys())
			}
		})
	}
}
func boolName(v bool) string {
	if v {
		return "true"
	}
	return "false"
}

func TestSellDollarBudgetFallbackAndOtherServices(t *testing.T) {
	for _, mode := range []string{"nil", "false", "error", "buy", "alpaca", "kalshi", "unknown", "origin-error", "unlisted", "other-tool", "prefix"} {
		t.Run(mode, func(t *testing.T) {
			g, s := sellGate(t)
			c := sellCall(ocToken(t, ocLimitDoc(0, ocEquity, "100")), ocEquity, `{"symbol":"AAPL","side":"sell","dollar_amount":"20"}`)
			switch mode {
			case "nil":
				g = newTestSpendGate(s)
			case "false":
				sellTestHook(t, g, false, nil)
			case "error":
				sellTestHook(t, g, true, errors.New("unavailable"))
			case "buy":
				c.Params = ocParams(c.Tool, `{"symbol":"AAPL","side":"buy","dollar_amount":"20"}`)
			case "alpaca", "kalshi":
				c.Origin.Service = mode
			case "unknown":
				c.Origin = ToolOrigin{}
			case "origin-error":
				c.Origin.Err = ErrToolTableUnavailable
			case "unlisted":
				c.Origin.Listed = false
			case "other-tool":
				c.Tool = "another_order"
				c.Origin.Original = c.Tool
				c.Params = ocParams(c.Tool, `{"side":"sell","dollar_amount":"20"}`)
				c.Token = ocToken(t, strings.ReplaceAll(strings.ReplaceAll(ocLimitDoc(0, ocEquity, "100"), `,"value":"order_max_cost"`, ""), ocEquity, c.Tool))
			case "prefix":
				c.Origin.Service = "robinhood-other"
			}
			a := sellAdmit(t, g, c)
			if ocRefusalCode(a) != "" {
				t.Fatal(a.Refusal)
			}
			if s.reserves != 1 || !ocAllEqual(ocTotals(s.MemoryStore), 20*units) {
				t.Fatalf("not counted as on base: reserves=%d totals=%v", s.reserves, ocTotals(s.MemoryStore))
			}
			if a != nil && a.Settle != nil {
				a.Settle(context.Background(), GateOutcome{})
			}
			if s.releases != 1 || !ocAllEqual(ocTotals(s.MemoryStore), 0) {
				t.Fatalf("not released as on base: %d %v", s.releases, ocTotals(s.MemoryStore))
			}
		})
	}
}

func TestSellDollarBudgetStrictSide(t *testing.T) {
	for _, fields := range []string{
		`"side":"sell","side":"sell"`, `"side":"sell","Side":"sell"`, `"Side":"sell"`,
		`"side":"ſell"`, `"side":"sеll"`, `"side":null`, `"side":" sell"`, `"side":"sell "`,
		`"side":42`, `"side":true`, `"side":{}`, `"side":""`, `"other":"sell"`,
	} {
		t.Run(fields, func(t *testing.T) {
			g, s := sellGate(t)
			c := sellCall(ocToken(t, ocLimitDoc(0, ocEquity, "1")), ocEquity, `{"symbol":"AAPL","dollar_amount":"20",`+fields+`}`)
			a := sellAdmit(t, g, c)
			if ocRefusalCode(a) == "" {
				t.Fatal("malformed side exempted")
			}
			for _, v := range ocTotals(s.MemoryStore) {
				if v != 0 {
					t.Fatalf("refused call charged: %v", ocTotals(s.MemoryStore))
				}
			}
		})
	}
}

func TestSellDollarBudgetAliasesAndNonDollar(t *testing.T) {
	for _, field := range []string{"usd", "cost_usd", "DOLLARS", "dollar_amount"} {
		t.Run(field, func(t *testing.T) {
			g, s := sellGate(t)
			doc := `{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"` + field + `","max":"1","window":"day"}]}`
			c := sellCall(ocToken(t, doc), ocEquity, `{"symbol":"AAPL","side":"sell","dollar_amount":"20"}`)
			if a := sellAdmit(t, g, c); ocRefusalCode(a) != "" {
				t.Fatal(a.Refusal)
			}
			if s.reserves != 0 {
				t.Fatalf("dollar alias reserved: %d", s.reserves)
			}
		})
	}
	t.Run("quantity still counts", func(t *testing.T) {
		g, s := sellGate(t)
		doc := `{"v":1,"spend_limits":[{"tools":["place_equity_order"],"field":"quantity","max":"3","window":"day"}]}`
		c := sellCall(ocToken(t, doc, ocLimitDoc(0, ocEquity, "1")), ocEquity, `{"symbol":"AAPL","side":"sell","quantity":"2","limit_price":"10","type":"limit","time_in_force":"gfd"}`)
		a := sellAdmit(t, g, c)
		if ocRefusalCode(a) != "" {
			t.Fatal(a.Refusal)
		}
		if s.reserves != 1 || !ocAllEqual(ocTotals(s.MemoryStore), 2*units) {
			t.Fatalf("non-dollar count changed: %d %v", s.reserves, ocTotals(s.MemoryStore))
		}
		a.Settle(context.Background(), GateOutcome{})
		if s.releases != 1 {
			t.Fatal("non-dollar release missing")
		}
	})
}

func TestSellDollarBudgetValidationStillRuns(t *testing.T) {
	for _, mode := range []string{"shape", "unsealed", "saved-budget", "coverage", "control-shape", "missing-id"} {
		t.Run(mode, func(t *testing.T) {
			g, s := sellGate(t)
			c := sellCall(ocToken(t, ocLimitDoc(2, ocEquity, "100")), ocEquity, `{"symbol":"AAPL","side":"sell","dollar_amount":"20"}`)
			switch mode {
			case "shape":
				c.Params = []byte(`{"name":"place_equity_order","name":"place_equity_order","arguments":{"side":"sell","dollar_amount":"20"}}`)
			case "unsealed":
				c.Token.SpendLimits[0].Sealed = false
			case "saved-budget":
				g.SharedBudgetValidator = func(context.Context, string, string, []toolcontrols.SpendLimit, func(string) bool, string) error {
					return ErrSharedBudgetUnavailable
				}
			case "coverage":
				c.Token = ocToken(t, ocLimitDoc(2, ocCrypto, "100"))
			case "control-shape":
				c.Token.Raw = &macaroon.Macaroon{Caveats: []string{"scope = argrules:ctl:v1:bad"}}
			case "missing-id":
				c.Token.SpendLimits[0].ID = ""
			}
			a, err := g.Admit(context.Background(), c)
			if err == nil && ocRefusalCode(a) == "" {
				t.Fatal("invalid control admitted")
			}
			if s.reserves != 0 || s.releases != 0 {
				t.Fatalf("validation touched counters: %d %d", s.reserves, s.releases)
			}
		})
	}
}

func TestSellDollarBudgetPerOrderAndApprovalUnchanged(t *testing.T) {
	for _, args := range []string{
		`{"symbol":"AAPL","side":"sell","dollar_amount":"20"}`,
		`{"symbol":"AAPL","side":"sell","quantity":"2","limit_price":"10","type":"limit","time_in_force":"gfd"}`,
	} {
		t.Run(args, func(t *testing.T) {
			g, s := sellGate(t)
			tok := ocToken(t, ocLimitDoc(0, ocEquity, "1"), ocApprovalDoc(ocEquity, "10"))
			c := sellCall(tok, ocEquity, args)
			cost, why := argrules.OrderMaxCostOfCall(c.Params, c.Tool, c.Tool)
			if !why.Valued() || cost.Units != 20*units {
				t.Fatalf("valuation changed: %+v %s", cost, why)
			}
			tok.ArgumentRules = ocCapRules(t, ocEquity, "10")
			if tok.CheckToolArguments(c.Tool, c.Params) == nil {
				t.Fatal("per-order cap bypassed")
			}
			ag := NewApprovalGate(toolcontrols.NewMemoryApprovalStore(), nil)
			a := sellAdmit(t, ChainGates(g, ag), c)
			if ocRefusalCode(a) != toolcontrols.DenialCodeApprovalRequired {
				t.Fatalf("approval not held at real value: %s", ocRefusalCode(a))
			}
			if s.reserves != 0 || s.releases != 0 {
				t.Fatalf("approval touched dollar store: %d %d", s.reserves, s.releases)
			}
		})
	}
}
