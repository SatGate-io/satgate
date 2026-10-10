package mcpserver

import (
	"context"
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
	Summary    []toolcontrols.SummaryItem
	ExpiresAt  time.Time
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
	var held *toolcontrols.ApprovalRule
	for i := range rules {
		amount, why := argrules.ReadNumericField(call.Params, call.Tool, rules[i].Field)
		// A negative amount is not a size to compare: it is held with the
		// unreadable ones.
		if why != argrules.NumericFieldOK || strings.HasPrefix(amount.String(), "-") || amount.Cmp(rules[i].Threshold()) > 0 {
			held = &rules[i]
			break
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
		Summary:  toolcontrols.OwnerSummary(call.Params, held.Field),
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
			g.tell(ctx, call, dec, held)
		}
		r := approvalRefusal(toolcontrols.DenialCodeApprovalRequired, held.Field, dec.ApprovalID,
			fmt.Sprintf("%s: calls with %s above %s need the owner's approval; the owner has been asked",
				call.Tool, held.Field, toolcontrols.FormatAmount(held.Field, held.Above)))
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
			fmt.Sprintf("%s: this token already has %d calls waiting for the owner; wait for them before sending more",
				call.Tool, toolcontrols.MaxPendingApprovals))}, nil
	}
	return nil, fmt.Errorf("approval store returned an unknown outcome")
}

func (g *ApprovalGate) ttl() time.Duration {
	if g.TTL > 0 {
		return g.TTL
	}
	return toolcontrols.ApprovalTTL
}

// tell notifies the owner, once per approval. A failed notice is logged and
// tried again the next time the agent repeats the held call.
func (g *ApprovalGate) tell(ctx context.Context, call GateCall, dec toolcontrols.ApprovalDecision, r *toolcontrols.ApprovalRule) {
	if g.Notifier == nil {
		log.Error().Str("approval_id", dec.ApprovalID).Msg("a call is waiting for approval but no notifier is configured; the owner can still see it on the approvals page")
		return
	}
	notice := ApprovalNotice{
		TenantID:   call.Token.TenantID,
		ApprovalID: dec.ApprovalID,
		TokenID:    call.Token.TokenID,
		Tool:       call.Tool,
		Field:      r.Field,
		Above:      r.Above,
		Summary:    toolcontrols.OwnerSummary(call.Params, r.Field),
		ExpiresAt:  dec.ExpiresAt,
	}
	run := func() {
		nctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 20*time.Second)
		defer cancel()
		if err := g.Notifier.Notify(nctx, notice); err != nil {
			log.Error().Err(err).Str("approval_id", notice.ApprovalID).Msg("could not tell the owner about a held call; it is still on the approvals page")
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
