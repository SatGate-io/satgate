package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// ApprovalNotice is what the token's owner is told about a held call. It
// holds the call's own values (the owner is the one approving), so it goes to
// the notifier and nowhere else: never to a receipt, a log or the agent.
type ApprovalNotice struct {
	TenantID   string
	ApprovalID string
	TokenID    string
	Tool       string
	Field      string
	Above      string
	// Value is the rule's value basis (argrules.OrderCostBasis), empty for a
	// rule that reads Field. With it the rule compared the most the order can
	// cost, not Field, and Summary starts with a "Counts as at most" line.
	Value     string
	Summary   []toolcontrols.SummaryItem
	ExpiresAt time.Time
}

// ApprovalNotifier tells the owner a call is waiting. It is called at most
// once per approval that succeeds (see ApprovalStore.MarkNotified). An error
// must not carry any of the notice's values: the gate logs it.
type ApprovalNotifier interface {
	Notify(ctx context.Context, n ApprovalNotice) error
}

// ApprovalGate is the CallGate behind "ask me" rules (toolcontrols.ApprovalRule).
//
// A call to a covered tool whose field is above the rule's threshold is not
// forwarded. The gate stores a pending approval bound to the token and the
// canonical call, tells the owner, and answers the agent APPROVAL_REQUIRED
// with the approval id. When the owner approves, the agent sends the same call
// again; the gate matches it, uses the approval up in one atomic step and lets
// the call through. Anything else (a changed argument, another token, a
// second try) is a different call and is held again.
//
// A rule that carries value "order_max_cost" (toolcontrols.ApprovalRule.Value)
// does not read its field. It compares the most the call can cost
// (argrules.OrderMaxCost, the number the spending limits and the per-order cap
// use) with its threshold: above holds, equal does not. A call it cannot value
// is refused with APPROVAL_CHECK_FAILED, not held, and nobody is emailed: an
// owner cannot approve a number SatGate cannot work out. A rule without the
// key keeps its meaning, including holding a call whose field cannot be read.
//
// Put it after the SpendGate in ChainGates: a call a limit refuses is refused,
// never held, and an approved call still reserves against the spending limit
// at the moment it is forwarded.
type ApprovalGate struct {
	Store    toolcontrols.ApprovalStore
	Notifier ApprovalNotifier
	// TTL is how long a held call waits. Zero means toolcontrols.ApprovalTTL.
	TTL time.Duration
	// Synchronous makes the gate wait for the notifier before it answers the
	// agent. The default tells the owner in the background, so a slow mail
	// service does not slow the agent's reply. Tests set it.
	Synchronous bool

	now func() time.Time
}

// NewApprovalGate returns a gate over store. A nil store refuses every covered
// call that is above a threshold (fail closed).
func NewApprovalGate(store toolcontrols.ApprovalStore, notifier ApprovalNotifier) *ApprovalGate {
	return &ApprovalGate{Store: store, Notifier: notifier}
}

// Admit implements CallGate.
func (g *ApprovalGate) Admit(ctx context.Context, call GateCall) (*GateAdmission, error) {
	all, err := call.Token.AllApprovalRules()
	if err != nil {
		return nil, fmt.Errorf("token controls: %w", err)
	}
	if len(all) == 0 {
		return nil, nil
	}
	// The token carries an "ask me" rule, so the name this gate matches must
	// be the name the upstream reads (the same check SpendGate makes). A name
	// outside the plain set or a call whose top level reads two ways is
	// refused for every tool on the token. A case variant of a plain name is
	// handled by ApprovalRule.Covers, which ignores case.
	if !argrules.ValidToolName(call.Tool) || argrules.CheckCallShape(call.Params, call.Tool) != argrules.NumericFieldOK {
		return &GateAdmission{Refusal: approvalRefusal(toolcontrols.DenialCodeApprovalUnreadable, all[0].Field, "",
			"this token asks the owner before some calls, and this call's tool name or layout cannot be matched exactly to that, so the call was not sent")}, nil
	}
	// What is hashed must identify exactly what is forwarded. A call whose
	// arguments cannot be read without loss (a lone UTF-16 surrogate escape,
	// invalid UTF-8, a repeated key at any depth, a number the canonical form
	// cannot keep exactly) is refused for every tool on the token, whether or
	// not it would be held: it is never forwarded, never held, never emailed.
	if err := toolcontrols.CheckCallLossless(call.Params); err != nil {
		return &GateAdmission{Refusal: approvalRefusal(toolcontrols.DenialCodeApprovalUnreadable, all[0].Field, "",
			"this token asks the owner before some calls, and this call's arguments cannot be matched exactly to an approval (they cannot be read without loss), so the call was not sent")}, nil
	}
	var rules []toolcontrols.ApprovalRule
	for _, r := range all {
		if r.Covers(call.Tool) {
			rules = append(rules, r)
		}
	}
	if len(rules) == 0 {
		return nil, nil
	}
	now := call.Now
	if g.now != nil {
		now = g.now()
	}
	if now.IsZero() {
		now = time.Now().UTC()
	}

	// Which rule holds the call? One whose field is above its threshold, or
	// whose field cannot be read as a number: a call the gate cannot measure
	// is not waved through, it is put in front of the owner.
	//
	// A rule with a value basis is decided on the order's cost, valued once for
	// the call. If one of them cannot value the call the call is refused, even
	// when an earlier rule would have held it.
	var held *toolcontrols.ApprovalRule
	var heldUnits int64 // the cost the holding rule compared, when it has a value basis
	type costResult struct {
		cost argrules.OrderCost
		why  argrules.OrderCostWhy
	}
	costs := map[string]costResult{}
	valueOf := func(r toolcontrols.ApprovalRule) costResult {
		key := r.Value + "\x00" + r.Schema
		if c, ok := costs[key]; ok {
			return c
		}
		var c costResult
		if r.Value == argrules.EventCostBasis {
			// An order that can only close a position counts 0: never held.
			ev, why := argrules.EventOrderMaxCostOfCall(call.Params, call.Tool, r.Schema)
			c = costResult{cost: argrules.OrderCost{Units: ev.Units}, why: why}
		} else {
			c.cost, c.why = argrules.OrderMaxCostOfCall(call.Params, call.Tool, call.Tool)
		}
		costs[key] = c
		return c
	}
	for i := range rules {
		if rules[i].Value != "" {
			v := valueOf(rules[i])
			cost, costWhy := v.cost, v.why
			if !costWhy.Valued() {
				return &GateAdmission{Refusal: approvalRefusal(toolcontrols.DenialCodeApprovalUnreadable, rules[i].Field, "",
					fmt.Sprintf("%s: this token asks the owner before an order that can cost more than %s, counting each order at the most it can cost, and %s, so it was not sent and nobody was asked",
						call.Tool, toolcontrols.FormatAmount(rules[i].Field, rules[i].Above), costWhy))}, nil
			}
			limit, ok := rules[i].ThresholdUnits()
			if !ok {
				return &GateAdmission{Refusal: approvalRefusal(toolcontrols.DenialCodeApprovalUnreadable, rules[i].Field, "",
					fmt.Sprintf("%s: this token's approval rule could not be checked, so the call was not sent", call.Tool))}, nil
			}
			if held == nil && cost.Units > limit {
				held, heldUnits = &rules[i], cost.Units
			}
			continue
		}
		if held != nil {
			continue
		}
		amount, why := argrules.ReadNumericField(call.Params, call.Tool, rules[i].Field)
		// A negative amount is not a size to compare: it is held with the
		// unreadable ones.
		if why != argrules.NumericFieldOK || strings.HasPrefix(amount.String(), "-") || amount.Cmp(rules[i].Threshold()) > 0 {
			held = &rules[i]
		}
	}
	if held == nil {
		return nil, nil // at or below every threshold: not touched
	}
	if g == nil || g.Store == nil {
		return nil, errors.New("no approval store is configured")
	}
	if call.Token == nil || call.Token.TenantID == "" || call.Token.TokenID == "" {
		return nil, errors.New("approvals need a tenant and a token id")
	}

	summary := ownerSummaryFor(call.Params, held, heldUnits)
	hash, err := toolcontrols.CallHash(call.Token.TokenID, call.Tool, call.Params)
	if err != nil {
		return &GateAdmission{Refusal: approvalRefusal(toolcontrols.DenialCodeApprovalUnreadable, held.Field, "",
			fmt.Sprintf("%s: this call could not be read, so it cannot be held for the owner's approval", call.Tool))}, nil
	}
	dec, err := g.Store.Decide(ctx, toolcontrols.ApprovalRequest{
		TenantID: call.Token.TenantID,
		TokenID:  call.Token.TokenID,
		Tool:     call.Tool,
		CallHash: hash,
		Field:    held.Field,
		Above:    held.Above,
		Value:    held.Value,
		Summary:  summary,
		Now:      now,
		TTL:      g.ttl(),
	})
	if err != nil {
		return nil, fmt.Errorf("approval store: %w", err)
	}

	switch dec.Outcome {
	case toolcontrols.ApprovalConsumed:
		// The approval is used up now, whatever happens next: it is single
		// use. A call that a later check stops before it is sent does not get
		// it back; the agent asks again and the owner decides again. A
		// spending limit that runs after this gate reserves and releases by
		// its own rule (SpendGate), not by the approval's.
		return &GateAdmission{Detail: GateReceiptDetail{ApprovalID: dec.ApprovalID}}, nil

	case toolcontrols.ApprovalHeld:
		if dec.NeedsNotice {
			g.tell(ctx, call, dec, held, summary)
		}
		r := approvalRefusal(toolcontrols.DenialCodeApprovalRequired, held.Field, dec.ApprovalID,
			fmt.Sprintf("%s: calls with %s above %s need the owner's approval; the call is waiting for the owner",
				call.Tool, held.Field, toolcontrols.FormatAmount(held.Field, held.Above)))
		if held.Value != "" {
			r.Message = fmt.Sprintf("%s: orders that can cost more than %s need the owner's approval; the call is waiting for the owner",
				call.Tool, toolcontrols.FormatAmount(held.Field, held.Above))
		}
		r.Data["approval_id"] = dec.ApprovalID
		r.Data["expires_at"] = dec.ExpiresAt.UTC().Format(time.RFC3339)
		r.Data["retry"] = "send the same call again after the owner approves"
		r.Data["above"] = held.Above
		return &GateAdmission{Refusal: r}, nil

	case toolcontrols.ApprovalDenied:
		r := approvalRefusal(toolcontrols.DenialCodeApprovalDenied, held.Field, dec.ApprovalID,
			fmt.Sprintf("%s: the owner said no to this call", call.Tool))
		r.Data["approval_id"] = dec.ApprovalID
		return &GateAdmission{Refusal: r}, nil

	case toolcontrols.ApprovalExpired:
		r := approvalRefusal(toolcontrols.DenialCodeApprovalExpired, held.Field, dec.ApprovalID,
			fmt.Sprintf("%s: the owner's approval for this call ran out unused; send the call again to ask again", call.Tool))
		r.Data["approval_id"] = dec.ApprovalID
		return &GateAdmission{Refusal: r}, nil

	case toolcontrols.ApprovalQueueFull:
		return &GateAdmission{Refusal: approvalRefusal(toolcontrols.DenialCodeApprovalTooManyOpen, held.Field, "",
			fmt.Sprintf("%s: too many calls are waiting for the owner (at most %d per token and %d per account); wait for them before sending more",
				call.Tool, toolcontrols.MaxPendingApprovals, toolcontrols.MaxPendingApprovalsPerTenant))}, nil
	}
	return nil, fmt.Errorf("approval store returned an unknown outcome")
}

// ownerSummaryFor is what the owner is shown for a held call: every argument,
// and for a rule with a value basis a first line saying what the order counts
// as (units is the cost the rule compared). It holds the call's values, so it
// goes to the owner only.
func ownerSummaryFor(params json.RawMessage, r *toolcontrols.ApprovalRule, units int64) []toolcontrols.SummaryItem {
	items := toolcontrols.OwnerSummary(params, r.Field)
	if r.Value == "" {
		return items
	}
	return append([]toolcontrols.SummaryItem{toolcontrols.CountsAsItem(r.Field, units)}, items...)
}

func (g *ApprovalGate) ttl() time.Duration {
	if g.TTL > 0 {
		return g.TTL
	}
	return toolcontrols.ApprovalTTL
}

// tell notifies the owner, once per approval. A failed notice is logged and
// tried again the next time the agent repeats the held call.
func (g *ApprovalGate) tell(ctx context.Context, call GateCall, dec toolcontrols.ApprovalDecision, r *toolcontrols.ApprovalRule, summary []toolcontrols.SummaryItem) {
	if g.Notifier == nil {
		log.Error().Str("approval_id", dec.ApprovalID).Msg("a call is waiting for approval but no notifier is configured; the owner can still see it on the approvals page")
		_ = g.Store.NoticeFailed(ctx, call.Token.TenantID, dec.ApprovalID)
		return
	}
	notice := ApprovalNotice{
		TenantID:   call.Token.TenantID,
		ApprovalID: dec.ApprovalID,
		TokenID:    call.Token.TokenID,
		Tool:       call.Tool,
		Field:      r.Field,
		Above:      r.Above,
		Value:      r.Value,
		Summary:    summary,
		ExpiresAt:  dec.ExpiresAt,
	}
	run := func() {
		nctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		defer cancel()
		if err := g.Notifier.Notify(nctx, notice); err != nil {
			log.Error().Err(err).Str("approval_id", notice.ApprovalID).Msg("could not tell the owner about a held call; it is still on the approvals page")
			if ferr := g.Store.NoticeFailed(nctx, notice.TenantID, notice.ApprovalID); ferr != nil {
				log.Error().Err(ferr).Str("approval_id", notice.ApprovalID).Msg("could not record the failed notice")
			}
			return
		}
		if err := g.Store.MarkNotified(nctx, notice.TenantID, notice.ApprovalID, time.Now().UTC()); err != nil {
			log.Error().Err(err).Str("approval_id", notice.ApprovalID).Msg("could not record that the owner was told")
		}
	}
	if g.Synchronous {
		run()
		return
	}
	go run()
}

// approvalRefusal builds a gate refusal whose text and receipt detail come
// from the rule and the approval id alone: no call argument.
func approvalRefusal(code, field, approvalID, message string) *GateRefusal {
	data := map[string]interface{}{"field": field}
	return &GateRefusal{
		Code:    code,
		Message: message,
		Data:    data,
		Detail: GateReceiptDetail{
			DenialCode:    code,
			ArgumentField: field,
			ApprovalID:    approvalID,
		},
	}
}
