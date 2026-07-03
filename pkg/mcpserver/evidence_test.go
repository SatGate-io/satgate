package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

type fakeMCPRouter struct{ called bool }

func (r *fakeMCPRouter) AllToolsForTenant(context.Context, string) []json.RawMessage { return nil }
func (r *fakeMCPRouter) ForwardToolCallForTenant(ctx context.Context, tenantID, toolName string, params json.RawMessage, timeout time.Duration) (*Response, error) {
	r.called = true
	result, _ := json.Marshal(map[string]interface{}{
		"content": []map[string]interface{}{{"type": "text", "text": "ok"}},
	})
	return &Response{JSONRPC: "2.0", Result: result}, nil
}

type fakeEvidenceRecorder struct {
	preflightCalls int
	decisions      []MCPDecision
	preflightErr   error
	recordErr      error
}

func (r *fakeEvidenceRecorder) Preflight(context.Context) error {
	r.preflightCalls++
	return r.preflightErr
}

func (r *fakeEvidenceRecorder) RecordMCPDecision(_ context.Context, decision MCPDecision) (*MCPEvidence, error) {
	r.decisions = append(r.decisions, decision)
	if r.recordErr != nil {
		return nil, r.recordErr
	}
	return &MCPEvidence{
		ReceiptID:      "rcpt_test",
		ReceiptHash:    "sha256:test",
		EvidencePackID: "ep_test",
		EvidenceURL:    "https://issuer.example/v1/evidence/evid_test",
		VerifyURL:      "https://issuer.example/verify",
		JWKSURL:        "https://issuer.example/.well-known/jwks.json",
	}, nil
}

type compensatingBudget struct {
	remaining   int64
	spent       bool
	compensated bool
}

func (b *compensatingBudget) Check(context.Context, string, int64) (*BudgetResult, error) {
	return &BudgetResult{Allowed: true, Remaining: b.remaining}, nil
}
func (b *compensatingBudget) Spend(_ context.Context, tokenID string, toolName string, cost int64, requestID string) (*BudgetResult, error) {
	b.spent = true
	b.remaining -= cost
	return &BudgetResult{Allowed: true, Remaining: b.remaining, TokenID: tokenID, Cost: cost}, nil
}
func (b *compensatingBudget) Remaining(context.Context, string) (int64, error) {
	return b.remaining, nil
}
func (b *compensatingBudget) Initialize(context.Context, string, int64) error { return nil }
func (b *compensatingBudget) Compensate(_ context.Context, tokenID string, requestID string, cost int64) error {
	b.compensated = true
	b.remaining += cost
	return nil
}

func newEvidenceTestProxy(t *testing.T) *Proxy {
	t.Helper()
	cfg := &Config{
		Server: ServerConfig{Transport: "stdio", Name: "test", Version: "1.0"},
		Auth:   AuthConfig{Mode: "none"},
		Upstreams: map[string]UpstreamConfig{
			"mock": {Transport: "stdio", Command: []string{"python3", "-c", "import sys; sys.exit(0)"}},
		},
		Budget:      BudgetConfig{Backend: "memory", Limit: 0, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 10, Costs: map[string]int64{"search": 10}},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	cfg.applyDefaults()
	proxy, err := New(cfg)
	if err != nil {
		t.Fatalf("create proxy: %v", err)
	}
	proxy.router = &fakeMCPRouter{}
	return proxy
}

func TestMCPToolsCallRecordsEvidenceOnAllowedDecision(t *testing.T) {
	ctx := context.Background()
	proxy := newEvidenceTestProxy(t)
	recorder := &fakeEvidenceRecorder{}
	proxy.SetEvidenceRecorder(recorder)
	if err := proxy.budget.Initialize(ctx, "budget-1", 20); err != nil {
		t.Fatal(err)
	}

	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"search","arguments":{"q":"satgate"}}`)}
	resp, err := proxy.handleToolsCall(ctx, req, &TokenInfo{TokenID: "token-1", BudgetID: "budget-1", BudgetLimit: 20, TenantID: "tenant-1", Scope: "mcp:*", DelegationDepth: 2, DelegationBudget: 7, ParentTokenID: "parent-1"})
	if err != nil {
		t.Fatalf("handleToolsCall: %v", err)
	}
	if resp.Error != nil {
		t.Fatalf("unexpected error response: %+v", resp.Error)
	}
	if recorder.preflightCalls != 1 {
		t.Fatalf("expected evidence preflight once, got %d", recorder.preflightCalls)
	}
	if len(recorder.decisions) != 1 {
		t.Fatalf("expected one decision, got %d", len(recorder.decisions))
	}
	decision := recorder.decisions[0]
	if decision.Decision != "allowed" || decision.DecisionReason != "budget_authorized" || decision.ToolName != "search" || decision.MCPMethod != MethodToolsCall {
		t.Fatalf("unexpected decision: %+v", decision)
	}
	if decision.TenantID != "tenant-1" || decision.TokenID != "token-1" || decision.BudgetID != "budget-1" || decision.ParentTokenID != "parent-1" {
		t.Fatalf("decision missing authority fields: %+v", decision)
	}
	if decision.CostCredits != 10 || decision.RemainingCredits != 10 || decision.RemainingBeforeCredits != 20 {
		t.Fatalf("decision missing budget fields: %+v", decision)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		t.Fatal(err)
	}
	meta, _ := result["_meta"].(map[string]interface{})
	evidence, _ := meta["satgate_evidence"].(map[string]interface{})
	if evidence["evidence_url"] == "" || evidence["receipt_hash"] == "" {
		t.Fatalf("response missing evidence metadata: %#v", result)
	}
}

func TestMCPToolsCallRecordsEvidenceOnBudgetDenial(t *testing.T) {
	ctx := context.Background()
	proxy := newEvidenceTestProxy(t)
	recorder := &fakeEvidenceRecorder{}
	proxy.SetEvidenceRecorder(recorder)
	if err := proxy.budget.Initialize(ctx, "budget-1", 5); err != nil {
		t.Fatal(err)
	}

	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"search"}`)}
	resp, err := proxy.handleToolsCall(ctx, req, &TokenInfo{TokenID: "token-1", BudgetID: "budget-1", BudgetLimit: 5, TenantID: "tenant-1", Scope: "mcp:*"})
	if err != nil {
		t.Fatalf("handleToolsCall: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != CodeBudgetExhausted {
		t.Fatalf("expected budget exhausted response, got %+v", resp)
	}
	if len(recorder.decisions) != 1 || recorder.decisions[0].Decision != "denied" {
		t.Fatalf("expected denied evidence decision, got %+v", recorder.decisions)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(resp.Error.Data, &data); err != nil {
		t.Fatal(err)
	}
	if data["evidence_url"] == "" || data["receipt_hash"] == "" {
		t.Fatalf("denial data missing evidence metadata: %#v", data)
	}
}

func TestMCPToolsCallEvidenceFailureCompensatesDebitAndSkipsUpstream(t *testing.T) {
	ctx := context.Background()
	proxy := newEvidenceTestProxy(t)
	budget := &compensatingBudget{remaining: 20}
	proxy.budget = budget
	router := &fakeMCPRouter{}
	proxy.router = router
	proxy.SetEvidenceRecorder(&fakeEvidenceRecorder{recordErr: errors.New("archive down")})

	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`3`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"search"}`)}
	resp, err := proxy.handleToolsCall(ctx, req, &TokenInfo{TokenID: "token-1", BudgetID: "budget-1", BudgetLimit: 20, TenantID: "tenant-1", Scope: "mcp:*"})
	if err != nil {
		t.Fatalf("handleToolsCall: %v", err)
	}
	if resp.Error == nil || resp.Error.Code != CodeInternalError {
		t.Fatalf("expected proof_unavailable error, got %+v", resp)
	}
	if !budget.spent || !budget.compensated || budget.remaining != 20 {
		t.Fatalf("expected compensated debit, got spent=%v compensated=%v remaining=%d", budget.spent, budget.compensated, budget.remaining)
	}
	if router.called {
		t.Fatal("upstream should not be called when evidence generation fails")
	}
}
