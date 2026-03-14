package integration

import (
	"encoding/json"
	"testing"

	"github.com/satgate-io/satgate/pkg/mcpserver"
)

// TestAPI_TaskTrackerQueryByToken verifies the in-memory TaskTracker's
// GetTasksByToken and GetTaskSpend methods — the data source that feeds
// the enterprise /api/v1/tasks endpoint.
func TestAPI_TaskTrackerQueryByToken(t *testing.T) {
	tracker := mcpserver.NewTaskTracker()

	// Simulate events from two different tokens
	tracker.RecordSpend("task-1", "token-a", "budget-a", "tenant-1", "search", 10)
	tracker.RecordSpend("task-1", "token-a", "budget-a", "tenant-1", "search", 10)
	tracker.RecordSpend("task-2", "token-b", "budget-b", "tenant-1", "generate", 50)

	// Query by token-a
	tasksA := tracker.GetTasksByToken("token-a")
	if len(tasksA) != 1 {
		t.Fatalf("expected 1 task for token-a, got %d", len(tasksA))
	}
	if tasksA[0].TaskID != "task-1" {
		t.Errorf("expected task-1, got %s", tasksA[0].TaskID)
	}
	if tasksA[0].TotalCost != 20 {
		t.Errorf("expected total cost 20 for task-1, got %d", tasksA[0].TotalCost)
	}
	if tasksA[0].Attempts != 2 {
		t.Errorf("expected 2 attempts, got %d", tasksA[0].Attempts)
	}

	// Query by token-b
	tasksB := tracker.GetTasksByToken("token-b")
	if len(tasksB) != 1 {
		t.Fatalf("expected 1 task for token-b, got %d", len(tasksB))
	}
	if tasksB[0].TotalCost != 50 {
		t.Errorf("expected total cost 50 for task-2, got %d", tasksB[0].TotalCost)
	}

	// Non-existent token
	tasksC := tracker.GetTasksByToken("token-c")
	if len(tasksC) != 0 {
		t.Errorf("expected 0 tasks for token-c, got %d", len(tasksC))
	}
}

// TestAPI_TaskTrackerStatusFilter simulates status-based filtering
// that the enterprise API performs.
func TestAPI_TaskTrackerStatusFilter(t *testing.T) {
	tracker := mcpserver.NewTaskTracker()

	tracker.RecordSpend("task-running", "token-a", "budget-a", "", "search", 10)
	tracker.RecordSpend("task-done", "token-a", "budget-a", "", "search", 10)
	tracker.UpdateStatus("task-done", "completed")

	// GetTasksByToken returns all tasks for token
	all := tracker.GetTasksByToken("token-a")
	if len(all) != 2 {
		t.Fatalf("expected 2 tasks, got %d", len(all))
	}

	// Manual status filter (mirrors what enterprise API does)
	var running, completed int
	for _, task := range all {
		switch task.Status {
		case "running":
			running++
		case "completed":
			completed++
		}
	}
	if running != 1 {
		t.Errorf("expected 1 running task, got %d", running)
	}
	if completed != 1 {
		t.Errorf("expected 1 completed task, got %d", completed)
	}
}

// TestAPI_TaskSpendSerialization verifies TaskSpend serializes to JSON
// in the format expected by the enterprise /api/v1/tasks endpoint.
func TestAPI_TaskSpendSerialization(t *testing.T) {
	tracker := mcpserver.NewTaskTracker()
	tracker.RecordSpend("task-ser", "tok-1", "budget-1", "tenant-x", "search", 42)
	tracker.UpdateStatus("task-ser", "completed")

	spend := tracker.GetTaskSpend("task-ser")
	if spend == nil {
		t.Fatal("expected task spend")
	}

	data, err := json.Marshal(spend)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	// Verify it round-trips correctly
	var parsed map[string]interface{}
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	// Check required fields present
	requiredFields := []string{"task_id", "token_id", "budget_id", "tool_name", "total_cost", "attempts", "first_seen", "last_seen", "status"}
	for _, field := range requiredFields {
		if _, ok := parsed[field]; !ok {
			t.Errorf("missing required field %q in serialized TaskSpend", field)
		}
	}

	if parsed["task_id"] != "task-ser" {
		t.Errorf("expected task_id 'task-ser', got %v", parsed["task_id"])
	}
	if parsed["status"] != "completed" {
		t.Errorf("expected status 'completed', got %v", parsed["status"])
	}
	if int64(parsed["total_cost"].(float64)) != 42 {
		t.Errorf("expected total_cost 42, got %v", parsed["total_cost"])
	}
	if parsed["tenant_id"] != "tenant-x" {
		t.Errorf("expected tenant_id 'tenant-x', got %v", parsed["tenant_id"])
	}
}

// TestAPI_EventTaskSpendFormat verifies the task_spend event data matches
// the contract expected by the enterprise Redis event handler.
func TestAPI_EventTaskSpendFormat(t *testing.T) {
	taskResult := TaskResponse("task-evt-fmt", "result")

	env := NewTestEnv(t,
		WithBudget(1000),
		WithToolResponseJSON(map[string]json.RawMessage{
			"search": taskResult,
		}),
	)
	defer env.Close()

	env.ToolCall(t, 1, "search", env.RootToken)

	events := env.Events.ByType(mcpserver.EventTaskSpend)
	if len(events) == 0 {
		t.Fatal("expected task_spend event")
	}

	evt := events[0]

	// Verify the event has the fields the enterprise handler expects
	if evt.Type != mcpserver.EventTaskSpend {
		t.Errorf("expected type %q, got %q", mcpserver.EventTaskSpend, evt.Type)
	}
	if evt.TokenID == "" {
		t.Error("event missing TokenID")
	}
	if evt.BudgetID == "" {
		t.Error("event missing BudgetID")
	}

	// Data fields the enterprise handler reads
	data := evt.Data
	if _, ok := data["task_id"]; !ok {
		t.Error("event data missing task_id")
	}
	if _, ok := data["tool"]; !ok {
		t.Error("event data missing tool")
	}
	if _, ok := data["cost"]; !ok {
		t.Error("event data missing cost")
	}

	// Verify serialization works (enterprise handler marshals to Redis)
	serialized, err := mcpserver.MarshalEvent(evt)
	if err != nil {
		t.Fatalf("MarshalEvent: %v", err)
	}

	roundtrip, err := mcpserver.UnmarshalEvent(serialized)
	if err != nil {
		t.Fatalf("UnmarshalEvent: %v", err)
	}

	if roundtrip.Type != mcpserver.EventTaskSpend {
		t.Errorf("roundtrip type mismatch: %q vs %q", roundtrip.Type, evt.Type)
	}
	if roundtrip.Data["task_id"] != evt.Data["task_id"] {
		t.Errorf("roundtrip task_id mismatch")
	}
}

// TestAPI_PaginationContract verifies pagination math works the same way
// the enterprise API does it (offset + limit against a slice).
func TestAPI_PaginationContract(t *testing.T) {
	tracker := mcpserver.NewTaskTracker()

	// Create 15 tasks
	for i := 0; i < 15; i++ {
		tid := "task-" + string(rune('a'+i))
		tracker.RecordSpend(tid, "token-a", "budget-a", "", "search", 10)
	}

	all := tracker.GetTasksByToken("token-a")
	total := len(all)
	if total != 15 {
		t.Fatalf("expected 15 tasks, got %d", total)
	}

	// Page 1: offset=0, limit=5
	start, limit := 0, 5
	end := start + limit
	if end > total {
		end = total
	}
	page1 := all[start:end]
	if len(page1) != 5 {
		t.Errorf("page 1 expected 5 items, got %d", len(page1))
	}

	// Page 3: offset=10, limit=5
	start = 10
	end = start + limit
	if end > total {
		end = total
	}
	page3 := all[start:end]
	if len(page3) != 5 {
		t.Errorf("page 3 expected 5 items, got %d", len(page3))
	}

	// Beyond: offset=20, limit=5
	start = 20
	if start > total {
		start = total
	}
	end = start + limit
	if end > total {
		end = total
	}
	beyond := all[start:end]
	if len(beyond) != 0 {
		t.Errorf("beyond expected 0 items, got %d", len(beyond))
	}
}
