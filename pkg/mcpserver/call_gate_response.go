package mcpserver

import (
	"context"
	"errors"
	"time"

	"github.com/rs/zerolog/log"
	"github.com/satgate-io/satgate/pkg/denial"
)

// gateRefusalResponse refuses a call a gate stopped. It is the scope refusal's
// shape (CodePolicyDenied, a signed denied/policy_denied decision that read no
// balance) with the gate's stable code and receipt detail added. The error
// data holds the gate's own text and the evidence handles; it never holds a
// value the agent sent.
func (p *Proxy) gateRefusalResponse(ctx context.Context, req *Request, tokenInfo *TokenInfo, toolName string, r *GateRefusal) *Response {
	data := map[string]interface{}{}
	for k, v := range r.Data {
		data[k] = v
	}
	data["error"] = r.Code
	data["tool"] = toolName
	if p.evidence == nil {
		return NewErrorResponseWithData(req.ID, CodePolicyDenied, r.Message, data)
	}
	proofUnavailable := func(err error, stage string) *Response {
		log.Error().Err(err).Str("tool", toolName).Str("stage", stage).Msg("MCP gate denial evidence unavailable")
		return NewErrorResponseWithData(req.ID, CodeInternalError, "Evidence proof unavailable", map[string]interface{}{
			"error": "proof_unavailable",
			"tool":  toolName,
		})
	}
	if err := p.evidence.Preflight(ctx); err != nil {
		return proofUnavailable(err, "preflight")
	}
	detail := r.Detail
	if detail.DenialCode == "" {
		detail.DenialCode = r.Code
	}
	recorded, err := denial.RecordSignedDenial(ctx, gateDenialRecorder{proxy: p, req: req, tokenInfo: tokenInfo, detail: detail},
		"policy_denied", tokenInfo.TokenID, toolName, time.Now().UTC())
	if err != nil {
		return proofUnavailable(err, "record")
	}
	evidence, _ := recorded.(*MCPEvidence)
	if evidence == nil || evidence.ReceiptID == "" {
		return proofUnavailable(errors.New("recorder returned no receipt"), "record")
	}
	mergeEvidenceData(data, evidence)
	return NewErrorResponseWithData(req.ID, CodePolicyDenied, r.Message, data)
}

// gateDenialRecorder adapts the MCP evidence recorder to the shared denial
// helper for a gate refusal. It takes no lock.
type gateDenialRecorder struct {
	proxy     *Proxy
	req       *Request
	tokenInfo *TokenInfo
	detail    GateReceiptDetail
}

func (g gateDenialRecorder) RecordSignedDenial(ctx context.Context, in denial.SignedDenial) (any, error) {
	decision := MCPDecision{
		Decision:           "denied",
		DecisionReason:     in.ReasonCode,
		BudgetNotEvaluated: true,
		DenialCode:         g.detail.DenialCode,
		ArgumentField:      g.detail.ArgumentField,
		ControlWindow:      g.detail.Window,
		ControlLimit:       g.detail.Limit,
		ApprovalID:         g.detail.ApprovalID,
	}
	evidence, err := g.proxy.recordMCPDecision(ctx, g.req, g.tokenInfo, in.Target, decision)
	if err != nil {
		return nil, err
	}
	return evidence, nil
}
