package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

// This file also runs unchanged against 40b22c58. The optional setting is
// discovered by interface so the same byte probe compiles on the released module.
func outageSwitch(t *testing.T, p *Proxy, on bool) {
	t.Helper()
	if s, ok := any(p).(interface{ SetSignUnavailableCheckRefusals(bool) }); ok {
		s.SetSignUnavailableCheckRefusals(on)
	} else if on {
		t.Fatal("unavailable-check setting absent")
	}
}

type outageGateFunc func(context.Context, GateCall) (*GateAdmission, error)

func (f outageGateFunc) Admit(ctx context.Context, c GateCall) (*GateAdmission, error) {
	return f(ctx, c)
}

type outageCase struct {
	name, reason string
	tool         string
	opts         eaOpts
	setup        func(*testing.T, *eaEnv, *eaHoldStore)
}

func outageCases() []outageCase {
	return []outageCase{
		{name: "price", reason: "tool_price_unavailable", opts: eaOpts{badPrice: true}, setup: func(t *testing.T, e *eaEnv, s *eaHoldStore) {
			// A stale fallback must never reach the receipt or controls.
			e.proxy.SetTenantCostResolver(&checkedResolver{plainResolver: plainResolver{price: 987}, err: errors.New("unreadable price")})
			e.proxy.SetCallGate(outageGateFunc(func(context.Context, GateCall) (*GateAdmission, error) {
				t.Error("price failure asked controls")
				return nil, nil
			}))
		}},
		{name: "controls-missing", reason: "tool_controls_unavailable", tool: "place_crypto_order", opts: eaOpts{caveats: []func(*testing.T) string{eaControl(spendDoc200Day)}}},
		{name: "controls-error", reason: "tool_controls_unavailable", setup: func(t *testing.T, e *eaEnv, s *eaHoldStore) {
			e.proxy.SetCallGate(outageGateFunc(func(context.Context, GateCall) (*GateAdmission, error) {
				return nil, errors.New("unavailable controls")
			}))
		}},
		{name: "controls-invalid", reason: "tool_controls_unavailable", setup: func(t *testing.T, e *eaEnv, s *eaHoldStore) {
			e.proxy.SetCallGate(outageGateFunc(func(context.Context, GateCall) (*GateAdmission, error) {
				return &GateAdmission{Refusal: &GateRefusal{Detail: GateReceiptDetail{Limit: "987"}}, Settle: func(_ context.Context, o GateOutcome) {
					if o != (GateOutcome{}) {
						t.Error("invalid refusal settlement")
					}
				}}, nil
			}))
		}},
		{name: "budget", reason: "budget_backend_error", setup: func(t *testing.T, e *eaEnv, s *eaHoldStore) {
			e.proxy.SetBudgetEnforcer(failingBudget{s.InMemoryBudgetEnforcer})
		}},
		{name: "capacity", reason: "pending_capacity_exhausted", setup: func(t *testing.T, e *eaEnv, s *eaHoldStore) {
			if err := e.proxy.SetEveryActionSettings(EveryActionSettings{PendingBound: 1}); err != nil {
				t.Fatal(err)
			}
			slot := e.proxy.every().pending.reserve(kindUncharged)
			if slot == nil {
				t.Fatal("could not fill pending store")
			}
			t.Cleanup(slot.release)
		}},
		{name: "transport", reason: "", setup: func(t *testing.T, e *eaEnv, s *eaHoldStore) { e.rr.replies = []func() (*Response, error){forwardFails} }},
	}
}
func outageEnv(t *testing.T, c outageCase, explicitOff bool) (*eaEnv, *eaHoldStore) {
	t.Helper()
	e := newEAEnv(t, c.opts)
	s := &eaHoldStore{InMemoryBudgetEnforcer: e.proxy.budget.(*InMemoryBudgetEnforcer)}
	e.proxy.SetBudgetEnforcer(s)
	eaSetSwitch(e, true)
	if explicitOff {
		outageSwitch(t, e.proxy, false)
	}
	if c.setup != nil {
		c.setup(t, e, s)
	}
	return e, s
}
func outageBytes(t *testing.T, r *Response) string {
	t.Helper()
	b, err := json.Marshal(r)
	if err != nil {
		t.Fatal(err)
	}
	return string(b)
}
func outageCall(e *eaEnv, c outageCase) *Response {
	if c.tool != "" {
		return e.call(1, c.tool, order("10"))
	}
	return e.call(1, "paid_read", "")
}

func TestUnavailableCheck_SwitchOffBytes(t *testing.T) {
	var snapshots []string
	roads := append([]struct {
		name string
		via  func(*testing.T, *eaEnv) func()
	}{{name: "direct"}}, eaTransports...)
	for _, c := range outageCases() {
		for _, road := range roads {
			t.Run(c.name+"/"+road.name, func(t *testing.T) {
				e, _ := outageEnv(t, c, false)
				if road.via != nil {
					defer road.via(t, e)()
				}
				before := e.remaining()
				r := outageCall(e, c)
				// Exact JSON bytes, not normalized ids/data. The transport case uses
				// the same deterministic recorder handles on main and this branch.
				got := outageBytes(t, r)
				other, _ := outageEnv(t, c, true)
				if road.via != nil {
					defer road.via(t, other)()
				}
				if got != outageBytes(t, outageCall(other, c)) {
					t.Fatal("default and explicit false differ")
				}
				if c.reason != "" && (len(e.base.signed()) != 0 || e.rr.forwarded() != 0 || e.remaining() != before) {
					t.Fatal("off stop changed effects")
				}
				snapshots = append(snapshots, c.name+"/"+road.name+" "+got)
			})
		}
	}
	if p := os.Getenv("OUTAGE_BYTE_SNAPSHOT"); p != "" {
		b, _ := json.MarshalIndent(snapshots, "", "  ")
		if err := os.WriteFile(p, b, 0600); err != nil {
			t.Fatal(err)
		}
	}
	baseline := os.Getenv("OUTAGE_BYTE_BASELINE")
	if baseline == "" && os.Getenv("OUTAGE_BYTE_SNAPSHOT") == "" {
		baseline = "testdata/unavailable-check-off-40b22c58.json"
	}
	if p := baseline; p != "" {
		b, err := os.ReadFile(filepath.Clean(p))
		if err != nil {
			t.Fatal(err)
		}
		var want []string
		if err = json.Unmarshal(b, &want); err != nil {
			t.Fatal(err)
		}
		if !reflect.DeepEqual(want, snapshots) {
			t.Fatal("bytes differ from released module")
		}
	}
}

// Compile-time checks keep the compatibility harness on the old public surface.
var _ BudgetHolder = (*eaHoldStore)(nil)
