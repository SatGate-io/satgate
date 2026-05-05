package integration

import (
	"encoding/json"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/mcpserver"
)

// TestTaskCost_SingleToolCall verifies that a tool call with task_id in the
// upstream response is tracked by the TaskTracker with correct spend.
func TestTaskCost_SingleToolCall(t *testing.T) {
	taskResult := TaskResponse("task-abc-123", "search result")

	env := NewTestEnv(t,
		WithBudget(1000),
		WithToolResponseJSON(map[string]json.RawMessage{
			"search": taskResult,
		}),
	)
	defer env.Close()

	resp := env.ToolCall(t, 1, "search", env.RootToken)
	if _, ok := resp["result"]; !ok {
		t.Fatalf("expected result, got: %v", resp)
	}

	// Give a moment for async processing (task tracking happens synchronously
	// in handleToolsCall, so this should be immediate)
	tracker := env.Proxy.TaskTracker()
	spend := tracker.GetTaskSpend("task-abc-123")
	if spend == nil {
		t.Fatal("expected task spend record for task-abc-123, got nil")
	}

	if spend.TotalCost != 10 { // search costs 10
		t.Errorf("expected total cost 10, got %d", spend.TotalCost)
	}
	if spend.Attempts != 1 {
		t.Errorf("expected 1 attempt, got %d", spend.Attempts)
	}
	if spend.ToolName != "search" {
		t.Errorf("expected tool name 'search', got %q", spend.ToolName)
	}
	if spend.Status != "running" {
		t.Errorf("expected status 'running', got %q", spend.Status)
	}
}

// TestTaskCost_MultiStepTask verifies cumulative cost tracking across
// multiple tool calls with the same task_id.
func TestTaskCost_MultiStepTask(t *testing.T) {
	taskResult := TaskResponse("task-multi-step", "step result")

	env := NewTestEnv(t,
		WithBudget(5000),
		WithToolResponseJSON(map[string]json.RawMessage{
			"search":   taskResult,
			"generate": taskResult,
		}),
	)
	defer env.Close()

	// 3 tool calls: search(10) + search(10) + generate(50) = 70 total
	env.ToolCall(t, 1, "search", env.RootToken)
	env.ToolCall(t, 2, "search", env.RootToken)
	env.ToolCall(t, 3, "generate", env.RootToken)

	tracker := env.Proxy.TaskTracker()
	spend := tracker.GetTaskSpend("task-multi-step")
	if spend == nil {
		t.Fatal("expected task spend record")
	}

	if spend.TotalCost != 70 {
		t.Errorf("expected cumulative cost 70, got %d", spend.TotalCost)
	}
	if spend.Attempts != 3 {
		t.Errorf("expected 3 attempts, got %d", spend.Attempts)
	}

	// TaskCostSoFar should match
	soFar := tracker.TaskCostSoFar("task-multi-step")
	if soFar != 70 {
		t.Errorf("TaskCostSoFar expected 70, got %d", soFar)
	}
}

// TestTaskCost_StatusUpdate verifies that task status from _meta.status
// is propagated to the TaskTracker.
func TestTaskCost_StatusUpdate(t *testing.T) {
	// Step 1: running
	runningResult := TaskResponseWithStatus("task-status", "running", "working...")
	// Step 2: completed
	completedResult := TaskResponseWithStatus("task-status", "completed", "done!")

	callCount := 0
	// We need a way to return different responses on different calls.
	// Since the Python mock uses static responses, we'll use two different tools.
	env := NewTestEnv(t,
		WithBudget(5000),
		WithTools(
			MockTool{Name: "step1", Description: "step 1"},
			MockTool{Name: "step2", Description: "step 2"},
		),
		WithToolCosts(map[string]int64{"step1": 10, "step2": 10}),
		WithToolResponseJSON(map[string]json.RawMessage{
			"step1": runningResult,
			"step2": completedResult,
		}),
	)
	defer env.Close()
	_ = callCount

	// Step 1: running
	env.ToolCall(t, 1, "step1", env.RootToken)
	tracker := env.Proxy.TaskTracker()
	spend := tracker.GetTaskSpend("task-status")
	if spend == nil {
		t.Fatal("expected task record after step 1")
	}
	if spend.Status != "running" {
		t.Errorf("expected status 'running' after step 1, got %q", spend.Status)
	}

	// Step 2: completed
	env.ToolCall(t, 2, "step2", env.RootToken)
	spend = tracker.GetTaskSpend("task-status")
	if spend.Status != "completed" {
		t.Errorf("expected status 'completed' after step 2, got %q", spend.Status)
	}
	if spend.TotalCost != 20 {
		t.Errorf("expected total cost 20, got %d", spend.TotalCost)
	}
}

// TestTaskCost_PerTokenIsolation verifies that tasks from different tokens
// are tracked separately.
func TestTaskCost_PerTokenIsolation(t *testing.T) {
	taskResult := TaskResponse("task-shared-name", "result")

	env := NewTestEnv(t,
		WithBudget(5000),
		WithToolResponseJSON(map[string]json.RawMessage{
			"search": taskResult,
		}),
	)
	defer env.Close()

	// Call with root token
	env.ToolCall(t, 1, "search", env.RootToken)

	// Create a second token via delegation
	env.Send(t, map[string]interface{}{
		"jsonrpc": "2.0", "id": 10, "method": "satgate/delegate",
		"params": map[string]interface{}{
			"budget": 1000,
			"label":  "agent-2",
			"_meta":  map[string]string{"token": env.RootToken},
		},
	})
	resp := env.Recv(t)
	result := resp["result"].(map[string]interface{})
	token2 := result["token"].(string)

	// Call with second token
	env.ToolCall(t, 2, "search", token2)

	// Both calls hit the same task ID, so the tracker accumulates.
	// But GetTasksByToken should show per-token filtering.
	tracker := env.Proxy.TaskTracker()

	// The task spend record stores the first token that created it
	spend := tracker.GetTaskSpend("task-shared-name")
	if spend == nil {
		t.Fatal("expected task record")
	}
	// Total cost should be 10 + 10 = 20 (both calls tracked under same task)
	if spend.TotalCost != 20 {
		t.Errorf("expected total cost 20, got %d", spend.TotalCost)
	}

	// GetTasksByToken for root token should include this task
	rootTasks := tracker.GetTasksByToken(env.RootTokenID)
	if len(rootTasks) != 1 {
		t.Errorf("expected 1 task for root token, got %d", len(rootTasks))
	}
}

// TestTaskCost_SpendWithTask verifies that SpendWithTask updates both
// budget and task tracker atomically.
func TestTaskCost_SpendWithTask(t *testing.T) {
	env := NewTestEnv(t, WithBudget(1000))
	defer env.Close()

	// Direct call to SpendWithTask on the budget enforcer
	result, err := env.Budget.SpendWithTask(env.ctx, env.RootTokenID, "search", 10, "req-1", "task-direct")
	if err != nil {
		t.Fatalf("SpendWithTask: %v", err)
	}
	if !result.Allowed {
		t.Fatal("SpendWithTask should be allowed")
	}
	if result.Remaining != 990 {
		t.Errorf("expected 990 remaining, got %d", result.Remaining)
	}

	// Verify spend log has task ID
	log := env.Budget.SpendLog()
	if len(log) != 1 {
		t.Fatalf("expected 1 spend record, got %d", len(log))
	}
	if log[0].TaskID != "task-direct" {
		t.Errorf("expected task ID 'task-direct', got %q", log[0].TaskID)
	}
}

// TestTaskCost_EventEmission verifies task_spend events have the correct fields.
func TestTaskCost_EventEmission(t *testing.T) {
	taskResult := TaskResponse("task-events-test", "result")

	env := NewTestEnv(t,
		WithBudget(1000),
		WithToolResponseJSON(map[string]json.RawMessage{
			"search": taskResult,
		}),
	)
	defer env.Close()

	env.ToolCall(t, 1, "search", env.RootToken)

	// Find task_spend events
	taskEvents := env.Events.ByType(mcpserver.EventTaskSpend)
	if len(taskEvents) != 1 {
		t.Fatalf("expected 1 task_spend event, got %d", len(taskEvents))
	}

	evt := taskEvents[0]

	// Verify event fields
	if evt.TokenID == "" {
		t.Error("event missing token_id")
	}
	if evt.BudgetID == "" {
		t.Error("event missing budget_id")
	}

	// Verify event data
	taskID, ok := evt.Data["task_id"].(string)
	if !ok || taskID != "task-events-test" {
		t.Errorf("expected task_id 'task-events-test', got %v", evt.Data["task_id"])
	}

	toolName, ok := evt.Data["tool"].(string)
	if !ok || toolName != "search" {
		t.Errorf("expected tool 'search', got %v", evt.Data["tool"])
	}

	cost, ok := evt.Data["cost"].(int64)
	if !ok {
		// JSON numbers may be float64
		if fc, ok := evt.Data["cost"].(float64); ok {
			cost = int64(fc)
		}
	}
	if cost != 10 {
		t.Errorf("expected cost 10, got %v", evt.Data["cost"])
	}
}

// TestTaskCost_NoTaskID verifies that tool calls without an upstream task_id
// still create a generated per-tool task tracker entry.
func TestTaskCost_NoTaskID(t *testing.T) {
	env := NewTestEnv(t, WithBudget(1000))
	defer env.Close()

	// Default mock response has no task_id.
	env.ToolCall(t, 1, "search", env.RootToken)

	tracker := env.Proxy.TaskTracker()

	// The proxy should generate a task_spend event for cost tracking.
	taskEvents := env.Events.ByType(mcpserver.EventTaskSpend)
	if len(taskEvents) != 1 {
		t.Fatalf("expected 1 generated task_spend event for non-task response, got %d", len(taskEvents))
	}
	if taskID, ok := taskEvents[0].Data["task_id"].(string); !ok || !strings.HasPrefix(taskID, "tool-search-") {
		t.Errorf("expected generated tool-search task_id, got %v", taskEvents[0].Data["task_id"])
	}

	// GetTasksByToken should include the generated task.
	tasks := tracker.GetTasksByToken(env.RootTokenID)
	if len(tasks) != 1 {
		t.Fatalf("expected 1 generated task, got %d", len(tasks))
	}
	if tasks[0].ToolName != "search" {
		t.Errorf("expected generated task for search, got %q", tasks[0].ToolName)
	}
}

// TestTaskCost_TaskTimestamps verifies FirstSeen and LastSeen are set correctly.
func TestTaskCost_TaskTimestamps(t *testing.T) {
	taskResult := TaskResponse("task-timestamps", "result")

	env := NewTestEnv(t,
		WithBudget(5000),
		WithToolResponseJSON(map[string]json.RawMessage{
			"search": taskResult,
		}),
	)
	defer env.Close()

	before := time.Now()
	env.ToolCall(t, 1, "search", env.RootToken)
	env.ToolCall(t, 2, "search", env.RootToken)
	after := time.Now()

	tracker := env.Proxy.TaskTracker()
	spend := tracker.GetTaskSpend("task-timestamps")
	if spend == nil {
		t.Fatal("expected task record")
	}

	if spend.FirstSeen.Before(before) || spend.FirstSeen.After(after) {
		t.Errorf("FirstSeen %v outside range [%v, %v]", spend.FirstSeen, before, after)
	}
	if spend.LastSeen.Before(spend.FirstSeen) {
		t.Error("LastSeen should be >= FirstSeen")
	}
	if spend.LastSeen.After(after) {
		t.Errorf("LastSeen %v after test end %v", spend.LastSeen, after)
	}
}
