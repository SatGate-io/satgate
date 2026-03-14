package integration

import (
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/mcpserver"
)

// TestProxy_ToolCallSuccess verifies a basic tool call flows through the proxy,
// deducts budget, and emits the correct events.
func TestProxy_ToolCallSuccess(t *testing.T) {
	env := NewTestEnv(t, WithBudget(1000))
	defer env.Close()

	resp := env.ToolCall(t, 1, "search", env.RootToken)
	if _, ok := resp["result"]; !ok {
		t.Fatalf("expected result, got: %v", resp)
	}

	// Budget should be deducted (search costs 10)
	remaining, err := env.Budget.Remaining(env.ctx, env.RootTokenID)
	if err != nil {
		t.Fatalf("remaining: %v", err)
	}
	if remaining != 990 {
		t.Errorf("expected 990 remaining, got %d", remaining)
	}

	// Should have budget_spend and tool_call events
	spendEvents := env.Events.ByType(mcpserver.EventBudgetSpend)
	if len(spendEvents) != 1 {
		t.Errorf("expected 1 budget_spend event, got %d", len(spendEvents))
	}
	toolEvents := env.Events.ByType(mcpserver.EventToolCall)
	if len(toolEvents) != 1 {
		t.Errorf("expected 1 tool_call event, got %d", len(toolEvents))
	}
}

// TestProxy_BudgetExhausted verifies that a 402-equivalent error is returned
// when budget runs out in hard enforcement mode.
func TestProxy_BudgetExhausted(t *testing.T) {
	env := NewTestEnv(t,
		WithBudget(25), // Only enough for 2 search calls (10 each) + partial
		WithEnforcement("hard"),
	)
	defer env.Close()

	// First two calls should succeed
	for i := 1; i <= 2; i++ {
		resp := env.ToolCall(t, i, "search", env.RootToken)
		if _, ok := resp["result"]; !ok {
			t.Fatalf("call %d should succeed, got: %v", i, resp)
		}
	}

	// Third call should fail (5 remaining < 10 cost)
	resp := env.ToolCall(t, 3, "search", env.RootToken)
	errObj, ok := resp["error"]
	if !ok {
		t.Fatalf("expected error, got result: %v", resp)
	}

	errMap := errObj.(map[string]interface{})
	code := int(errMap["code"].(float64))
	if code != mcpserver.CodeBudgetExhausted {
		t.Errorf("expected code %d, got %d", mcpserver.CodeBudgetExhausted, code)
	}

	data := errMap["data"].(map[string]interface{})
	if data["error"] != "budget_exhausted" {
		t.Errorf("expected budget_exhausted error code, got %v", data["error"])
	}

	// Verify exhaustion event was emitted
	exhaustEvents := env.Events.ByType(mcpserver.EventBudgetExhaust)
	if len(exhaustEvents) == 0 {
		t.Error("expected budget_exhausted event")
	}
}

// TestProxy_ExpiredMacaroon verifies that an expired token is rejected.
func TestProxy_ExpiredMacaroon(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Close()

	expired := env.MintExpiredToken(t)
	resp := env.ToolCall(t, 1, "search", expired)

	errObj, ok := resp["error"]
	if !ok {
		t.Fatalf("expected error for expired token, got result: %v", resp)
	}
	errMap := errObj.(map[string]interface{})
	code := int(errMap["code"].(float64))
	if code != mcpserver.CodePolicyDenied {
		t.Errorf("expected CodePolicyDenied (%d), got %d", mcpserver.CodePolicyDenied, code)
	}
}

// TestProxy_InvalidToken verifies that a garbage token is rejected.
func TestProxy_InvalidToken(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Close()

	resp := env.ToolCall(t, 1, "search", "not-a-valid-macaroon")

	errObj, ok := resp["error"]
	if !ok {
		t.Fatalf("expected error for invalid token, got result: %v", resp)
	}
	errMap := errObj.(map[string]interface{})
	code := int(errMap["code"].(float64))
	if code != mcpserver.CodePolicyDenied {
		t.Errorf("expected CodePolicyDenied (%d), got %d", mcpserver.CodePolicyDenied, code)
	}
}

// TestProxy_ToolsList verifies the tools/list method returns all upstream tools.
func TestProxy_ToolsList(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Close()

	env.Send(t, map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "tools/list",
	})
	resp := env.Recv(t)

	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected result, got: %v", resp)
	}
	tools, ok := result["tools"].([]interface{})
	if !ok {
		t.Fatalf("expected tools array, got: %v", result)
	}
	if len(tools) != 2 {
		t.Errorf("expected 2 tools, got %d", len(tools))
	}
}

// TestProxy_SequentialBudgetConsistency verifies budget stays consistent
// across many sequential tool calls (race condition regression test).
func TestProxy_SequentialBudgetConsistency(t *testing.T) {
	env := NewTestEnv(t, WithBudget(10000))
	defer env.Close()

	const numCalls = 20

	// Serial send+recv to avoid pipe deadlock, then verify budget is exact
	for i := 0; i < numCalls; i++ {
		resp := env.ToolCall(t, 1000+i, "search", env.RootToken)
		if _, ok := resp["result"]; !ok {
			t.Fatalf("call %d failed: %v", i, resp)
		}
	}

	// Budget should be consistent: 10000 - (20 * 10) = 9800
	remaining, err := env.Budget.Remaining(env.ctx, env.RootTokenID)
	if err != nil {
		t.Fatalf("remaining: %v", err)
	}
	expected := int64(10000 - numCalls*10)
	if remaining != expected {
		t.Errorf("expected %d remaining, got %d (race condition?)", expected, remaining)
	}

	// Verify spend log has exactly numCalls entries
	log := env.Budget.SpendLog()
	if len(log) != numCalls {
		t.Errorf("expected %d spend records, got %d", numCalls, len(log))
	}
}

// TestProxy_SoftEnforcement verifies soft mode allows calls even when budget is exhausted.
func TestProxy_SoftEnforcement(t *testing.T) {
	env := NewTestEnv(t,
		WithBudget(5), // Less than one search call (10)
		WithEnforcement("soft"),
	)
	defer env.Close()

	// Should succeed despite insufficient budget (soft mode)
	resp := env.ToolCall(t, 1, "search", env.RootToken)
	if _, ok := resp["result"]; !ok {
		t.Fatalf("soft mode should allow call, got: %v", resp)
	}
}

// TestProxy_Ping verifies the ping method works.
func TestProxy_Ping(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Close()

	env.Send(t, map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	})
	resp := env.Recv(t)
	if _, ok := resp["result"]; !ok {
		t.Fatalf("ping should return result, got: %v", resp)
	}
}

// TestProxy_MultipleDifferentTools verifies different tool costs are applied correctly.
func TestProxy_MultipleDifferentTools(t *testing.T) {
	env := NewTestEnv(t, WithBudget(1000))
	defer env.Close()

	// search costs 10
	resp := env.ToolCall(t, 1, "search", env.RootToken)
	if _, ok := resp["result"]; !ok {
		t.Fatalf("search should succeed: %v", resp)
	}

	// generate costs 50
	resp = env.ToolCall(t, 2, "generate", env.RootToken)
	if _, ok := resp["result"]; !ok {
		t.Fatalf("generate should succeed: %v", resp)
	}

	remaining, _ := env.Budget.Remaining(env.ctx, env.RootTokenID)
	expected := int64(1000 - 10 - 50) // 940
	if remaining != expected {
		t.Errorf("expected %d remaining, got %d", expected, remaining)
	}
}

// TestProxy_NoAuthMode verifies the proxy works without authentication.
func TestProxy_NoAuthMode(t *testing.T) {
	env := NewTestEnv(t,
		WithAuthMode("none"),
		WithBudget(100),
	)
	defer env.Close()

	// Tool call without a token should work in none mode
	resp := env.ToolCall(t, 1, "search", "")
	if _, ok := resp["result"]; !ok {
		t.Fatalf("no-auth call should succeed, got: %v", resp)
	}
}

// TestProxy_DelegationAndExhaustion tests delegating budget to a child token.
func TestProxy_DelegationAndExhaustion(t *testing.T) {
	env := NewTestEnv(t, WithBudget(500))
	defer env.Close()

	// Delegate 100 credits
	env.Send(t, map[string]interface{}{
		"jsonrpc": "2.0", "id": 10, "method": "satgate/delegate",
		"params": map[string]interface{}{
			"budget": 100,
			"label":  "child-agent",
			"_meta":  map[string]string{"token": env.RootToken},
		},
	})
	resp := env.Recv(t)
	result, ok := resp["result"].(map[string]interface{})
	if !ok {
		t.Fatalf("delegation should succeed, got: %v", resp)
	}

	childToken := result["token"].(string)
	childBudget := int64(result["budget"].(float64))
	if childBudget != 100 {
		t.Errorf("expected child budget 100, got %d", childBudget)
	}

	parentRemaining := int64(result["parentRemaining"].(float64))
	if parentRemaining != 400 {
		t.Errorf("expected parent remaining 400, got %d", parentRemaining)
	}

	// Child makes 10 search calls (10 * 10 = 100, exactly exhausts budget)
	for i := 1; i <= 10; i++ {
		resp = env.ToolCall(t, 100+i, "search", childToken)
		if _, ok := resp["result"]; !ok {
			t.Fatalf("child call %d should succeed, got: %v", i, resp)
		}
	}

	// Child's 11th call should fail
	resp = env.ToolCall(t, 200, "search", childToken)
	if _, ok := resp["error"]; !ok {
		t.Fatalf("child call 11 should fail, got result: %v", resp)
	}

	// Parent should still have 400
	env.Send(t, map[string]interface{}{
		"jsonrpc": "2.0", "id": 300, "method": "satgate/budget",
		"params": map[string]interface{}{
			"_meta": map[string]string{"token": env.RootToken},
		},
	})
	resp = env.Recv(t)
	budgetResult := resp["result"].(map[string]interface{})
	parentLeft := int64(budgetResult["remaining"].(float64))
	if parentLeft != 400 {
		t.Errorf("expected parent remaining 400, got %d", parentLeft)
	}
}

// TestProxy_EventTimestamps verifies events have valid timestamps.
func TestProxy_EventTimestamps(t *testing.T) {
	env := NewTestEnv(t)
	defer env.Close()

	before := time.Now()
	env.ToolCall(t, 1, "search", env.RootToken)
	after := time.Now()

	events := env.Events.All()
	for _, e := range events {
		if e.Timestamp.Before(before) || e.Timestamp.After(after) {
			t.Errorf("event timestamp %v outside expected range [%v, %v]", e.Timestamp, before, after)
		}
	}
}
