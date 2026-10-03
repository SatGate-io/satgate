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

// countingBudget fails every balance read so a test can prove the scope
// denial path never reads, checks, or spends budget.
type countingBudget struct {
	BudgetEnforcer
	remainingCalls int
	checkCalls     int
	spendCalls     int
}

func (b *countingBudget) Remaining(context.Context, string) (int64, error) {
	b.remainingCalls++
	return 0, errors.New("budget store unavailable")
}

func (b *countingBudget) Check(context.Context, string, int64) (*BudgetResult, error) {
	b.checkCalls++
	return nil, errors.New("budget store unavailable")
}

func (b *countingBudget) Spend(context.Context, string, string, int64, string) (*BudgetResult, error) {
	b.spendCalls++
	return nil, errors.New("budget store unavailable")
}

func TestMCPToolsCallScopeDenialRecordsSignedDecisionAndLeavesInScopeCallAlone(t *testing.T) {
	ctx := context.Background()
	proxy := newEvidenceTestProxy(t)
	recorder := &fakeEvidenceRecorder{}
	proxy.SetEvidenceRecorder(recorder)
	router := &fakeMCPRouter{}
	proxy.router = router
	if err := proxy.budget.Initialize(ctx, "budget-1", 20); err != nil {
		t.Fatal(err)
	}
	token := &TokenInfo{TokenID: "token-1", BudgetID: "budget-1", BudgetLimit: 20, TenantID: "tenant-1", Scope: "search"}

	allowedReq := &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"search"}`)}
	allowed, err := proxy.handleToolsCall(ctx, allowedReq, token)
	if err != nil {
		t.Fatalf("in-scope handleToolsCall: %v", err)
	}
	if allowed.Error != nil {
		t.Fatalf("in-scope call denied: %+v", allowed.Error)
	}
	if !router.called {
		t.Fatal("in-scope call did not reach upstream")
	}
	if len(recorder.decisions) != 1 || recorder.decisions[0].Decision != "allowed" || recorder.decisions[0].BudgetNotEvaluated {
		t.Fatalf("in-scope decision: %+v", recorder.decisions)
	}

	router.called = false
	deniedReq := &Request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"generate"}`)}
	denied, err := proxy.handleToolsCall(ctx, deniedReq, token)
	if err != nil {
		t.Fatalf("out-of-scope handleToolsCall: %v", err)
	}
	if denied.Error == nil || denied.Error.Code != CodePolicyDenied {
		t.Fatalf("expected policy denied, got %+v", denied)
	}
	if denied.Error.Message != `tool "generate" not in scope "search"` {
		t.Fatalf("scope message changed: %q", denied.Error.Message)
	}
	if router.called {
		t.Fatal("out-of-scope call reached upstream")
	}
	if len(recorder.decisions) != 2 {
		t.Fatalf("expected allowed plus scope denial, got %+v", recorder.decisions)
	}
	decision := recorder.decisions[1]
	if decision.Decision != "denied" || decision.DecisionReason != "policy_denied" || decision.ToolName != "generate" {
		t.Fatalf("scope denial decision: %+v", decision)
	}
	if !decision.BudgetNotEvaluated || decision.NoVerifiedCapability {
		t.Fatalf("scope denial must be a verified capability with budget not evaluated: %+v", decision)
	}
	if decision.TokenID != "token-1" || decision.BudgetID != "budget-1" || decision.TenantID != "tenant-1" {
		t.Fatalf("scope denial missing verified capability identity: %+v", decision)
	}
	if decision.CostCredits != 0 || decision.RemainingCredits != 0 || decision.RemainingBeforeCredits != 0 || decision.BudgetLimitCredits != 0 {
		t.Fatalf("scope denial projected a balance it never read: %+v", decision)
	}
	if remaining, err := proxy.budget.Remaining(ctx, "budget-1"); err != nil || remaining != 10 {
		t.Fatalf("scope denial changed budget: remaining=%d err=%v", remaining, err)
	}
	var data map[string]interface{}
	if err := json.Unmarshal(denied.Error.Data, &data); err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{
		"receipt_id":       "rcpt_test",
		"receipt_hash":     "sha256:test",
		"evidence_pack_id": "ep_test",
		"evidence_url":     "https://issuer.example/v1/evidence/evid_test",
		"verify_url":       "https://issuer.example/verify",
		"jwks_url":         "https://issuer.example/.well-known/jwks.json",
	}
	if len(data) != len(want) {
		t.Fatalf("scope denial data must carry receipt handles only, got %#v", data)
	}
	for k, v := range want {
		if data[k] != v {
			t.Fatalf("scope denial data %s=%#v, want %#v (all: %#v)", k, data[k], v, data)
		}
	}
}

func TestMCPToolsCallScopeDenialNeverReadsBudget(t *testing.T) {
	ctx := context.Background()
	for _, withRecorder := range []bool{false, true} {
		proxy := newEvidenceTestProxy(t)
		budget := &countingBudget{BudgetEnforcer: proxy.budget}
		proxy.budget = budget
		recorder := &fakeEvidenceRecorder{}
		if withRecorder {
			proxy.SetEvidenceRecorder(recorder)
		}
		router := &fakeMCPRouter{}
		proxy.router = router
		req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`4`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"generate"}`)}
		resp, err := proxy.handleToolsCall(ctx, req, &TokenInfo{TokenID: "token-1", BudgetID: "budget-1", BudgetLimit: 25, Scope: "search"})
		if err != nil {
			t.Fatalf("recorder=%v: %v", withRecorder, err)
		}
		if resp.Error == nil || resp.Error.Code != CodePolicyDenied {
			t.Fatalf("recorder=%v: budget outage must not change the scope refusal: %+v", withRecorder, resp)
		}
		if budget.remainingCalls != 0 || budget.checkCalls != 0 || budget.spendCalls != 0 {
			t.Fatalf("recorder=%v: scope denial touched the budget: remaining=%d check=%d spend=%d", withRecorder, budget.remainingCalls, budget.checkCalls, budget.spendCalls)
		}
		if router.called {
			t.Fatalf("recorder=%v: out-of-scope call reached upstream", withRecorder)
		}
		if withRecorder {
			if len(recorder.decisions) != 1 || !recorder.decisions[0].BudgetNotEvaluated || recorder.decisions[0].RemainingCredits != 0 || recorder.decisions[0].BudgetLimitCredits != 0 {
				t.Fatalf("budget outage leaked into signed decision: %+v", recorder.decisions)
			}
		}
	}
}

func TestMCPToolsCallScopeDenialWithoutRecorderMatchesMainBytes(t *testing.T) {
	ctx := context.Background()
	proxy := newEvidenceTestProxy(t)
	router := &fakeMCPRouter{}
	proxy.router = router
	if err := proxy.budget.Initialize(ctx, "budget-1", 20); err != nil {
		t.Fatal(err)
	}
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"generate"}`)}
	resp, err := proxy.handleToolsCall(ctx, req, &TokenInfo{TokenID: "token-1", BudgetID: "budget-1", BudgetLimit: 20, TenantID: "tenant-1", Scope: "search"})
	if err != nil {
		t.Fatal(err)
	}
	got, err := json.Marshal(resp)
	if err != nil {
		t.Fatal(err)
	}
	// Exact bytes main returns for this refusal (no error data at all).
	const mainBytes = `{"jsonrpc":"2.0","id":1,"error":{"code":-32001,"message":"tool \"generate\" not in scope \"search\""}}`
	if string(got) != mainBytes {
		t.Fatalf("no-recorder scope refusal changed on the wire:\n got: %s\nwant: %s", got, mainBytes)
	}
	if router.called {
		t.Fatal("out-of-scope call reached upstream")
	}
}

func TestMCPToolsCallScopeDenialFailsClosedWhenProofUnavailable(t *testing.T) {
	ctx := context.Background()
	cases := map[string]*fakeEvidenceRecorder{
		"record_error":    {recordErr: errors.New("archive down")},
		"preflight_error": {preflightErr: errors.New("signer missing")},
	}
	for name, recorder := range cases {
		proxy := newEvidenceTestProxy(t)
		router := &fakeMCPRouter{}
		proxy.router = router
		proxy.SetEvidenceRecorder(recorder)

		req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`3`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"generate"}`)}
		resp, err := proxy.handleToolsCall(ctx, req, &TokenInfo{TokenID: "token-1", BudgetID: "budget-1", Scope: "search"})
		if err != nil {
			t.Fatalf("%s: handleToolsCall: %v", name, err)
		}
		if resp.Error == nil || resp.Error.Code != CodeInternalError {
			t.Fatalf("%s: expected proof_unavailable refusal, got %+v", name, resp)
		}
		var data map[string]interface{}
		if err := json.Unmarshal(resp.Error.Data, &data); err != nil {
			t.Fatal(err)
		}
		if data["error"] != "proof_unavailable" {
			t.Fatalf("%s: expected proof_unavailable, got %#v", name, data)
		}
		if router.called {
			t.Fatalf("%s: upstream called after scope denial proof failure", name)
		}
	}
}
