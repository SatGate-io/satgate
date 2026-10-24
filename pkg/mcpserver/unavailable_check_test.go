package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"
	"time"
)

// The original table still pins the default vocabulary. In the separately
// approved mode, only E12/E13/E14 change classification; real spending/approval
// policy refusals, Observe and rate windows keep exactly their former meaning.
func TestUnavailableCheck_MasterMatrix(t *testing.T) {
	for _, row := range eaTable() {
		t.Run(row.id, func(t *testing.T) {
			o := row.opts
			o.window = row.window
			e := newEAEnv(t, o)
			eaSetSwitch(e, true)
			outageSwitch(t, e.proxy, true)
			row.run(e)
			want := row.want
			if row.unavailableReason != "" {
				want = []string{"denied/" + row.unavailableReason}
			}
			if !reflect.DeepEqual(e.base.signed(), want) {
				t.Fatal("approved-mode classification differs from table")
			}
		})
	}
}

func TestUnavailableCheck_UnknownAllowedOutcomeIsNotRefused(t *testing.T) {
	for _, unreachable := range []bool{false, true} {
		t.Run(map[bool]string{false: "persisted", true: "unknown"}[unreachable], func(t *testing.T) {
			e, inner, _, done := eaHold(t, eaOpts{}, nil)
			defer done()
			outageSwitch(t, e.proxy, true)
			b := newP2Backend()
			e.proxy.SetBudgetEnforcer(&p2Budget{eaHoldStore: inner, b: b})
			e.proxy.SetEvidenceRecorder(&p2Recorder{eaWindowRecorder: e.rec.(*eaWindowRecorder), b: b})
			e.proxy.SetPendingStore(p2LostReply{p2Store{b}, unreachable})
			r := e.call(1, "paid_read", "")
			if unreachable {
				if outageEvidence(r)["error"] != "outcome_unknown" || e.rr.forwarded() != 0 {
					t.Fatal("unknown allowed outcome became refusal")
				}
			} else if r.Error != nil || e.rr.forwarded() != 1 {
				t.Fatal("persisted allowed outcome became refusal")
			}
			if len(e.base.attempts) != 0 {
				t.Fatal("second receipt attempted for prepared allow")
			}
			_, _, _, releases := inner.counts()
			if releases != 0 {
				t.Fatal("persisted or unknown allow released its hold")
			}
		})
	}
}

func outageEvidence(r *Response) map[string]any {
	m := map[string]any{}
	if r != nil && r.Error != nil {
		_ = json.Unmarshal(r.Error.Data, &m)
	}
	return m
}
func outageAssertDecision(t *testing.T, d MCPDecision, reason string) {
	t.Helper()
	if d.Decision != "denied" || d.DecisionReason != reason || !d.BudgetNotEvaluated || d.NoVerifiedCapability {
		t.Fatal("wrong refusal profile")
	}
	if d.CostCredits != 0 || d.RemainingCredits != 0 || d.RemainingBeforeCredits != 0 || d.BudgetLimitCredits != 0 {
		t.Fatal("receipt claimed backend amounts")
	}
	if d.DenialCode != "" || d.ControlLimit != "" || d.ControlWindow != "" || d.ControlWindowTotal != "" || d.ApprovalID != "" || d.ControlShared || d.ControlSymbol != "" {
		t.Fatal("receipt leaked gate detail")
	}
	if !strings.HasPrefix(d.RequestID, callIDPrefix) || d.RequestID == "1" || d.TokenID == "" || d.MCPMethod != MethodToolsCall {
		t.Fatal("receipt identity missing")
	}
}

func TestUnavailableCheck_Stops(t *testing.T) {
	roads := append([]struct {
		name string
		via  func(*testing.T, *eaEnv) func()
	}{{name: "direct"}}, eaTransports...)
	for _, c := range outageCases() {
		if c.reason == "" {
			continue
		}
		for _, road := range roads {
			for _, failure := range []string{"", "preflight", "record"} {
				t.Run(c.name+"/"+road.name+"/"+failure, func(t *testing.T) {
					e, s := outageEnv(t, c, false)
					outageSwitch(t, e.proxy, true)
					if road.via != nil {
						defer road.via(t, e)()
					}
					e.base.failPreflight = failure == "preflight"
					e.base.failRecord = failure == "record"
					before := e.remaining()
					r := outageCall(e, c)
					if r == nil || r.Error == nil || e.rr.forwarded() != 0 || e.remaining() != before {
						t.Fatal("stopped call forwarded or debited")
					}
					spends, holds, commits, releases := s.counts()
					if spends != 0 || holds != 0 || commits != 0 || releases != 0 {
						t.Fatal("pre-hold stop touched budget")
					}
					e.base.mu.Lock()
					defer e.base.mu.Unlock()
					if failure != "" {
						if !eaIsProofUnavailable(r) || len(e.base.decisions) != 0 || outageEvidence(r)["receipt_id"] != nil {
							t.Fatal("proof failure was signed")
						}
						return
					}
					if len(e.base.decisions) != 1 || len(e.base.attempts) != 1 {
						t.Fatal("not exactly one refusal")
					}
					outageAssertDecision(t, e.base.decisions[0], c.reason)
					if outageEvidence(r)["receipt_id"] != "rcpt_1" {
						t.Fatal("missing refusal handles")
					}
					if len(e.proxy.PendingRecords()) != 0 {
						t.Fatal("refusal reserved another pending slot")
					}
				})
			}
		}
	}
}

func TestUnavailableCheck_RequiresBothSettingsAndRollback(t *testing.T) {
	for _, on := range []bool{false, true} {
		e := newEAEnv(t, eaOpts{badPrice: true})
		outageSwitch(t, e.proxy, on)
		if e.proxy.unavailableCheckRefusalsEnabled() {
			t.Fatal("new setting implicitly enables sign every action")
		}
		r := e.call(1, "paid_read", "")
		if len(e.base.signed()) != 0 || outageEvidence(r)["receipt_id"] != nil {
			t.Fatal("signed with only one switch")
		}
	}
	e, _ := outageEnv(t, outageCases()[0], false)
	outageSwitch(t, e.proxy, true)
	outageCall(e, outageCases()[0])
	outageSwitch(t, e.proxy, false)
	r := outageCall(e, outageCases()[0])
	if len(e.base.signed()) != 1 || outageEvidence(r)["receipt_id"] != nil {
		t.Fatal("rollback rewrote history or still emits")
	}
}

type outageInspectRecorder struct {
	*eaWindowRecorder
	before func(MCPDecision)
}

func (r *outageInspectRecorder) RecordMCPDecision(ctx context.Context, d MCPDecision) (*MCPEvidence, error) {
	if r.before != nil {
		r.before(d)
	}
	return r.eaWindowRecorder.RecordMCPDecision(ctx, d)
}

func TestUnavailableCheck_GateRollbackBeforeSigning(t *testing.T) {
	for _, invalid := range []bool{false, true} {
		t.Run(map[bool]string{false: "error", true: "invalid"}[invalid], func(t *testing.T) {
			e, s := outageEnv(t, outageCase{}, false)
			outageSwitch(t, e.proxy, true)
			var order []string
			admitted := outageGateFunc(func(context.Context, GateCall) (*GateAdmission, error) {
				return &GateAdmission{Detail: GateReceiptDetail{Limit: "987", ApprovalID: "not-a-claim"}, Settle: func(_ context.Context, o GateOutcome) {
					if o != (GateOutcome{}) {
						t.Error("rollback outcome")
					}
					order = append(order, "earlier")
				}}, nil
			})
			fails := outageGateFunc(func(context.Context, GateCall) (*GateAdmission, error) {
				if !invalid {
					return nil, errors.New("controls unreadable")
				}
				return &GateAdmission{Refusal: &GateRefusal{}, Settle: func(_ context.Context, o GateOutcome) { order = append(order, "invalid") }}, nil
			})
			e.proxy.SetCallGate(ChainGates(admitted, fails))
			rec := &outageInspectRecorder{eaWindowRecorder: newEAWindowRecorder(e.base), before: func(d MCPDecision) {
				want := []string{"earlier"}
				if invalid {
					want = append(want, "invalid")
				}
				if !reflect.DeepEqual(order, want) {
					t.Error("signing preceded rollback")
				}
				outageAssertDecision(t, d, "tool_controls_unavailable")
				order = append(order, "signed")
			}}
			e.proxy.SetEvidenceRecorder(rec)
			if err := e.proxy.SetSignEveryAction(true); err != nil {
				t.Fatal(err)
			}
			if r := e.call(1, "paid_read", ""); outageEvidence(r)["receipt_id"] == nil {
				t.Fatal("missing handles")
			}
			if a, b, c, d := s.counts(); a+b+c+d != 0 {
				t.Fatal("gate stop touched budget")
			}
		})
	}
}

func TestUnavailableCheck_IdentityFailureAndInvalidReason(t *testing.T) {
	e, _ := outageEnv(t, outageCases()[0], false)
	outageSwitch(t, e.proxy, true)
	e.proxy.every().callID = func(string) (string, error) { return "", errors.New("entropy unavailable") }
	if r := outageCall(e, outageCases()[0]); !eaIsProofUnavailable(r) || len(e.base.signed()) != 0 || len(e.base.attempts) != 0 {
		t.Fatal("identity failure was signed")
	}
	info, err := e.proxy.auth.Verify(e.ctx, e.token)
	if err != nil {
		t.Fatal("verify test key")
	}
	r := e.proxy.refuseUnavailableCheck(e.ctx, info, "paid_read", unavailableCheckReason(255), NewErrorResponse(nil, CodeInternalError, "unused"))
	if !eaIsProofUnavailable(r) || len(e.base.attempts) != 0 {
		t.Fatal("unknown reason signed")
	}
}

func TestUnavailableCheck_ExcludedPostHoldAndMissingSupport(t *testing.T) {
	for _, prepare := range []bool{false, true} {
		t.Run(map[bool]string{false: "missing-holder", true: "prepare-failure"}[prepare], func(t *testing.T) {
			e, s := outageEnv(t, outageCase{}, false)
			outageSwitch(t, e.proxy, true)
			if prepare {
				e.proxy.SetPendingStore(p2Store{newP2Backend()})
			} else {
				e.proxy.SetBudgetEnforcer(&eaNoHoldStore{inner: s.InMemoryBudgetEnforcer})
			}
			before := e.remaining()
			r := e.call(1, "paid_read", "")
			if !eaIsProofUnavailable(r) || len(e.base.signed()) != 0 || e.rr.forwarded() != 0 || e.remaining() != before {
				t.Fatal("excluded failure signed, sent or charged")
			}
			_, h, _, rel := s.counts()
			if prepare && (h != 1 || rel != 1) {
				t.Fatal("prepare failure did not release its hold")
			}
		})
	}
}

func TestUnavailableCheck_TransportHasOnlyAdmission(t *testing.T) {
	for _, tool := range []string{"paid_read", "free_read"} {
		for _, road := range eaTransports {
			t.Run(tool+"/"+road.name, func(t *testing.T) {
				e, _ := outageEnv(t, outageCase{}, false)
				outageSwitch(t, e.proxy, true)
				e.rr.replies = []func() (*Response, error){forwardFails}
				e.proxy.SetCallGate(outageGateFunc(func(context.Context, GateCall) (*GateAdmission, error) {
					return &GateAdmission{Settle: func(_ context.Context, o GateOutcome) {
						if !o.Unknown || o.Succeeded {
							t.Error("transport outcome changed")
						}
					}}, nil
				}))
				defer road.via(t, e)()
				r := e.call(1, tool, "")
				if r.Error == nil || r.Error.Code != CodeUpstreamError || !strings.HasPrefix(r.Error.Message, "upstream error: ") {
					t.Fatal("transport error changed")
				}
				if outageEvidence(r)["receipt_id"] != "rcpt_1" || len(e.base.signed()) != 1 || !strings.HasPrefix(e.base.signed()[0], "allowed/") || e.rr.forwarded() != 1 {
					t.Fatal("transport created denial or lost admission")
				}
			})
		}
	}
}

func TestUnavailableCheck_ObserveAndFailOpenUnchanged(t *testing.T) {
	for _, o := range []eaOpts{{observe: true, badPrice: true}, {observe: true, brokenBud: true}, {brokenBud: true, failOpen: true}} {
		for _, down := range []bool{false, true} {
			results := []string{}
			for _, on := range []bool{false, true} {
				e := newEAEnv(t, o)
				eaSetSwitch(e, true)
				outageSwitch(t, e.proxy, on)
				e.base.setDown(down)
				before := e.remaining()
				r := e.call(1, "paid_read", "")
				results = append(results, outageBytes(t, r))
				for _, s := range e.base.signed() {
					if strings.HasPrefix(s, "denied/") {
						t.Fatal("fail-open/Observe made outage refusal")
					}
				}
				if o.observe && (e.rr.forwarded() != 1 || r.Error != nil) {
					t.Fatal("Observe stopped")
				}
				if !o.observe && down {
					if !eaIsProofUnavailable(r) || e.remaining() != before {
						t.Fatal("fail-open Control bypassed proof failure")
					}
				}
			}
			// Successful responses may contain server-selected request ids in result
			// metadata; fixed recorder handles and transport replies remain identical.
			if results[0] != results[1] {
				t.Fatal("Observe/fail-open output changed")
			}
		}
	}
}

// The helper's recorder deadline remains the every-action signing deadline.
func TestUnavailableCheck_SigningTimeout(t *testing.T) {
	e, _ := outageEnv(t, outageCases()[0], false)
	outageSwitch(t, e.proxy, true)
	if err := e.proxy.SetEveryActionSettings(EveryActionSettings{SignTimeout: 20 * time.Millisecond}); err != nil {
		t.Fatal(err)
	}
	r := newEAScriptRecorder(newEAWindowRecorder(e.base), eaScriptStep{gate: make(chan struct{})})
	e.proxy.SetEvidenceRecorder(r)
	if err := e.proxy.SetSignEveryAction(true); err != nil {
		t.Fatal(err)
	}
	if resp := outageCall(e, outageCases()[0]); !eaIsProofUnavailable(resp) || len(e.base.signed()) != 0 {
		t.Fatal("timed-out signing became outage proof")
	}
}
