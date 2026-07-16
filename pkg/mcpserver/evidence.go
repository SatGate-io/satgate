package mcpserver

import "context"

// MCPDecision is the receipt-ready decision envelope emitted for governed MCP
// tools/call budget decisions. Enterprise implementations can turn this into a
// signed, issuer-anchored Evidence Pack without depending on proxy internals.
type MCPDecision struct {
	Decision               string `json:"decision"`
	DecisionReason         string `json:"decision_reason"`
	PolicyMode             string `json:"policy_mode"`
	TenantID               string `json:"tenant_id,omitempty"`
	TokenID                string `json:"token_id,omitempty"`
	BudgetID               string `json:"budget_id,omitempty"`
	BudgetSubjectID        string `json:"budget_subject_id,omitempty"`
	BudgetLimitCredits     int64  `json:"budget_limit_credits,omitempty"`
	DelegationDepth        int64  `json:"delegation_depth,omitempty"`
	DelegationBudget       int64  `json:"delegation_budget,omitempty"`
	ParentTokenID          string `json:"parent_token_id,omitempty"`
	Scope                  string `json:"scope,omitempty"`
	MCPMethod              string `json:"mcp_method"`
	ToolName               string `json:"tool_name"`
	RouteOrTool            string `json:"route_or_tool"`
	RequestID              string `json:"request_id"`
	JSONRPCID              string `json:"jsonrpc_id,omitempty"`
	CostCredits            int64  `json:"cost_credits"`
	RemainingCredits       int64  `json:"remaining_credits"`
	RemainingBeforeCredits int64  `json:"remaining_before_credits"`
	// NoVerifiedCapability marks terminal denials produced before credential
	// verification succeeds. Recorders must not project token, capability, or
	// evaluated-budget authority for this profile.
	NoVerifiedCapability bool `json:"no_verified_capability,omitempty"`
}

// MCPEvidence is the verifier-facing handle returned after recording an MCP
// decision as a signed artifact.
type MCPEvidence struct {
	ReceiptID      string `json:"receipt_id,omitempty"`
	ReceiptHash    string `json:"receipt_hash,omitempty"`
	EvidencePackID string `json:"evidence_pack_id,omitempty"`
	EvidenceURL    string `json:"evidence_url,omitempty"`
	VerifyURL      string `json:"verify_url,omitempty"`
	JWKSURL        string `json:"jwks_url,omitempty"`
}

// EvidenceRecorder records governed MCP decisions into a durable, independently
// verifiable artifact. Enterprise implementations should fail closed from
// Preflight/RecordMCPDecision when required signing/archive infrastructure is
// unavailable.
type EvidenceRecorder interface {
	Preflight(ctx context.Context) error
	RecordMCPDecision(ctx context.Context, decision MCPDecision) (*MCPEvidence, error)
}

// BudgetCompensator is optionally implemented by budget enforcers that can undo
// a debit when proof generation fails before an upstream tool call is made.
type BudgetCompensator interface {
	Compensate(ctx context.Context, tokenID string, requestID string, cost int64) error
}
