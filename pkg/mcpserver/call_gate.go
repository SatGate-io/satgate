package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// CallGate is the one hook for checks that need state across calls (a
// spending limit over a day or week, a hold for the owner's approval). The
// proxy asks it about every tools/call, after the scope check and the token's
// argument rules and before cost, budget and the upstream call. A gate never
// sees a call that scope or argument rules already refused.
//
// A gate that admits a call may have taken something to do so (an amount
// reserved against a limit). It hands back Settle, and the proxy calls Settle
// exactly once with how the call ended, whatever happened after the gate:
// a later refusal, an upstream failure, or success.
type CallGate interface {
	// Admit decides about one call. It returns:
	//   (nil, nil)          the gate has no opinion; the call goes on.
	//   (admission, nil)    the call goes on (admission.Refusal == nil) or is
	//                       refused (admission.Refusal != nil).
	//   (nil, err)          the gate could not decide. The proxy refuses the
	//                       call: a control that cannot be checked is not skipped.
	Admit(ctx context.Context, call GateCall) (*GateAdmission, error)
}

// GateCall is what a gate may read about a call. Params is the exact params
// object that will be forwarded.
type GateCall struct {
	Token  *TokenInfo
	Tool   string
	Params json.RawMessage
	// RequestID is a stable name for this call: the same token, tool and
	// JSON-RPC id give the same value, so a retry with the same id is
	// recognisably the same call. Gates that need a narrower identity (for
	// example one per client session) mix their own in.
	RequestID string
	// Now is the decision time. Gates read this, not the clock.
	Now time.Time
}

// GateAdmission is a gate's answer.
type GateAdmission struct {
	// Refusal, when set, stops the call before it is forwarded.
	Refusal *GateRefusal
	// Detail is signed into the allowed receipt when the call is forwarded.
	Detail GateReceiptDetail
	// Settle is called once with the outcome. It may be nil. It is called
	// for a refusal too (so a gate can undo what it took before refusing).
	Settle func(ctx context.Context, outcome GateOutcome)
}

// GateRefusal refuses a call with JSON-RPC -32001 and a signed receipt.
type GateRefusal struct {
	// Code is the stable code in the error data ("error") and in the
	// receipt ("denial_code").
	Code string
	// Message is a sentence for the agent. It must not carry a value the
	// agent sent.
	Message string
	// Data is merged into the error data next to "error", "tool" and the
	// evidence handles. It must not carry a value the agent sent.
	Data map[string]interface{}
	// Detail is signed into the refusal receipt. DenialCode defaults to Code.
	Detail GateReceiptDetail
}

// GateReceiptDetail is what a gate adds to a signed receipt. Every field is
// text a gate chose (a rule's own field name, a limit, a window name, an id).
// None of them may hold a value the agent sent.
type GateReceiptDetail struct {
	DenialCode    string // refusals only
	ArgumentField string // the rule field concerned
	Window        string // "day" or "week"
	Limit         string // the owner's maximum, as written
	WindowTotal   string // total in the window after this call (allowed only)
	ApprovalID    string // the approval concerned
}

// IsZero reports whether the detail adds nothing.
func (d GateReceiptDetail) IsZero() bool { return d == GateReceiptDetail{} }

// GateOutcome says how an admitted call ended.
//
// A gate that took something to admit the call (an amount reserved against a
// limit) gives it back only when it knows the call did not run upstream: a
// zero GateOutcome (a refusal, a call stopped before it was sent), an upstream
// JSON-RPC error, an isError result, or a call that was never sent. When the
// call was sent and its fate is not known, Unknown is true and the gate must
// keep what it took.
type GateOutcome struct {
	// Succeeded is true when the call reached the upstream and the upstream
	// accepted it: no transport failure, no JSON-RPC error, no isError result.
	// It is false for a refusal and for every failure after the gate.
	Succeeded bool
	// Unknown is true when the call was sent to the upstream and nothing says
	// whether it ran: a timeout after the send, a connection reset or closed
	// after the send, a response that could not be read, an HTTP error status
	// with no JSON-RPC error, a context ended after the send. Succeeded is
	// false then too. It is false for every outcome SatGate knows.
	Unknown bool
}

// SetCallGate installs g. Call it before the proxy serves requests. With no
// gate the proxy behaves as before.
func (p *Proxy) SetCallGate(g CallGate) { p.gate = g }

type gateDetailKey struct{}

// withGateDetail carries the admitted call's detail to recordMCPDecision.
func withGateDetail(ctx context.Context, d GateReceiptDetail) context.Context {
	if d.IsZero() {
		return ctx
	}
	return context.WithValue(ctx, gateDetailKey{}, d)
}

func gateDetailFrom(ctx context.Context) (GateReceiptDetail, bool) {
	d, ok := ctx.Value(gateDetailKey{}).(GateReceiptDetail)
	return d, ok
}

// admitCall asks the gate about one call. It returns a refusal response when
// the call must not go on, or an admission to settle when it may.
func (p *Proxy) admitCall(ctx context.Context, req *Request, tokenInfo *TokenInfo, toolName string) (*GateAdmission, *Response) {
	if p.gate == nil {
		// The token carries a control and nothing here can enforce it.
		log.Error().Str("tool", toolName).Msg("token carries a control but no call gate is installed; refusing the call")
		return nil, NewErrorResponseWithData(req.ID, CodeInternalError, "A limit on this token could not be checked, so the call was not sent", map[string]interface{}{
			"error": "tool_controls_unavailable",
			"tool":  toolName,
		})
	}
	adm, err := p.gate.Admit(ctx, GateCall{
		Token:     tokenInfo,
		Tool:      toolName,
		Params:    req.Params,
		RequestID: gateRequestID(tokenInfo, toolName, req),
		Now:       time.Now().UTC(),
	})
	if err != nil {
		log.Error().Err(err).Str("tool", toolName).Msg("call gate could not decide; refusing the call")
		return nil, NewErrorResponseWithData(req.ID, CodeInternalError, "A limit on this token could not be checked, so the call was not sent", map[string]interface{}{
			"error": "tool_controls_unavailable",
			"tool":  toolName,
		})
	}
	if adm == nil {
		return nil, nil
	}
	if adm.Refusal != nil && adm.Refusal.validate() != nil {
		log.Error().Str("tool", toolName).Msg("call gate returned a refusal with no code or message; refusing the call")
		if adm.Settle != nil {
			adm.Settle(ctx, GateOutcome{})
		}
		return nil, NewErrorResponseWithData(req.ID, CodeInternalError, "A limit on this token could not be checked, so the call was not sent", map[string]interface{}{
			"error": "tool_controls_unavailable",
			"tool":  toolName,
		})
	}
	if adm.Refusal != nil {
		resp := p.gateRefusalResponse(ctx, req, tokenInfo, toolName, adm.Refusal)
		if adm.Settle != nil {
			adm.Settle(ctx, GateOutcome{})
		}
		return nil, resp
	}
	return adm, nil
}

// settle reports how the call ended. A reply with a JSON-RPC error or an
// isError result did not succeed and did not run. A call that was never sent
// did not run either. A call that was sent and then failed some other way
// (timeout, closed connection, unreadable reply) is an unknown outcome: the
// gate is told so and keeps what it took.
func (a *GateAdmission) settle(ctx context.Context, resp *Response, err error) {
	if a == nil || a.Settle == nil {
		return
	}
	rec := dispatchFrom(ctx)
	if rec == nil {
		// No record of the call's path (a caller that did not create one):
		// the old, coarse reading. Nothing in the proxy takes this branch.
		a.Settle(ctx, GateOutcome{Succeeded: err == nil && responseSucceeded(resp)})
		return
	}
	switch rec.classify() {
	case endRan:
		a.Settle(ctx, GateOutcome{Succeeded: true})
	case endUnknown:
		a.Settle(ctx, GateOutcome{Unknown: true})
	default:
		a.Settle(ctx, GateOutcome{})
	}
}

func responseSucceeded(resp *Response) bool {
	if resp == nil || resp.Error != nil {
		return false
	}
	if len(resp.Result) == 0 {
		return true
	}
	var r struct {
		IsError json.RawMessage `json:"isError"`
	}
	if json.Unmarshal(resp.Result, &r) != nil {
		return true
	}
	return string(r.IsError) != "true"
}

// ChainGates asks each gate in order. The first refusal stops the chain and
// every gate that already admitted is settled as failed, so a reservation one
// gate took is given back when a later gate refuses. The call goes on only if
// every gate lets it. Order matters: put the gate that must run first (limits)
// before the one that may hold the call (approvals), so a call that would be
// refused is refused, not held.
func ChainGates(gates ...CallGate) CallGate {
	var live []CallGate
	for _, g := range gates {
		if g != nil {
			live = append(live, g)
		}
	}
	switch len(live) {
	case 0:
		return nil
	case 1:
		return live[0]
	}
	return chainGate(live)
}

type chainGate []CallGate

func (c chainGate) Admit(ctx context.Context, call GateCall) (*GateAdmission, error) {
	var admitted []*GateAdmission
	undo := func() {
		for i := len(admitted) - 1; i >= 0; i-- {
			if admitted[i].Settle != nil {
				admitted[i].Settle(ctx, GateOutcome{})
			}
		}
	}
	var detail GateReceiptDetail
	for _, g := range c {
		adm, err := g.Admit(ctx, call)
		if err != nil {
			undo()
			return nil, err
		}
		if adm == nil {
			continue
		}
		if adm.Refusal != nil {
			// Give back what earlier gates took. The refusing gate's own
			// Settle is passed on; the proxy calls it once.
			undo()
			return &GateAdmission{Refusal: adm.Refusal, Settle: adm.Settle}, nil
		}
		admitted = append(admitted, adm)
		detail = mergeGateDetail(detail, adm.Detail)
	}
	if len(admitted) == 0 {
		return nil, nil
	}
	return &GateAdmission{
		Detail: detail,
		Settle: func(ctx context.Context, o GateOutcome) {
			for i := len(admitted) - 1; i >= 0; i-- {
				if admitted[i].Settle != nil {
					admitted[i].Settle(ctx, o)
				}
			}
		},
	}, nil
}

// mergeGateDetail keeps what is set in either; a field set in both keeps the
// earlier gate's value.
func mergeGateDetail(a, b GateReceiptDetail) GateReceiptDetail {
	pick := func(x, y string) string {
		if x != "" {
			return x
		}
		return y
	}
	return GateReceiptDetail{
		DenialCode:    pick(a.DenialCode, b.DenialCode),
		ArgumentField: pick(a.ArgumentField, b.ArgumentField),
		Window:        pick(a.Window, b.Window),
		Limit:         pick(a.Limit, b.Limit),
		WindowTotal:   pick(a.WindowTotal, b.WindowTotal),
		ApprovalID:    pick(a.ApprovalID, b.ApprovalID),
	}
}

func (r *GateRefusal) validate() error {
	if r.Code == "" || r.Message == "" {
		return fmt.Errorf("a gate refusal needs a code and a message")
	}
	return nil
}

// gateRequestID names the call for a gate. A request with no JSON-RPC id (or
// a null one) has no name: it cannot be told apart from another such request,
// so it is never treated as a retry.
func gateRequestID(tokenInfo *TokenInfo, toolName string, req *Request) string {
	if id := strings.TrimSpace(string(req.ID)); id == "" || id == "null" {
		return ""
	}
	return generateRequestID(tokenInfo.BudgetID, toolName, req.ID)
}
