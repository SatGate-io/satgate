package mcpserver

import (
	"fmt"
	"sync"
	"testing"
)

func TestTaskTracker_RecordSingleSpend(t *testing.T) {
	tt := NewTaskTracker()
	tt.RecordSpend("task-1", "tok-a", "bud-a", "tenant-1", "gpt4_summarize", 10)

	ts := tt.GetTaskSpend("task-1")
	if ts == nil {
		t.Fatal("expected task spend, got nil")
	}
	if ts.TaskID != "task-1" {
		t.Errorf("TaskID = %q, want %q", ts.TaskID, "task-1")
	}
	if ts.TokenID != "tok-a" {
		t.Errorf("TokenID = %q, want %q", ts.TokenID, "tok-a")
	}
	if ts.BudgetID != "bud-a" {
		t.Errorf("BudgetID = %q, want %q", ts.BudgetID, "bud-a")
	}
	if ts.TenantID != "tenant-1" {
		t.Errorf("TenantID = %q, want %q", ts.TenantID, "tenant-1")
	}
	if ts.ToolName != "gpt4_summarize" {
		t.Errorf("ToolName = %q, want %q", ts.ToolName, "gpt4_summarize")
	}
	if ts.TotalCost != 10 {
		t.Errorf("TotalCost = %d, want 10", ts.TotalCost)
	}
	if ts.Attempts != 1 {
		t.Errorf("Attempts = %d, want 1", ts.Attempts)
	}
	if ts.Status != "running" {
		t.Errorf("Status = %q, want %q", ts.Status, "running")
	}
	if ts.FirstSeen.IsZero() || ts.LastSeen.IsZero() {
		t.Error("expected FirstSeen and LastSeen to be set")
	}
}

func TestTaskTracker_AccumulateSpend(t *testing.T) {
	tt := NewTaskTracker()
	tt.RecordSpend("task-1", "tok-a", "bud-a", "", "tool_x", 10)
	tt.RecordSpend("task-1", "tok-a", "bud-a", "", "tool_x", 25)
	tt.RecordSpend("task-1", "tok-a", "bud-a", "", "tool_x", 5)

	ts := tt.GetTaskSpend("task-1")
	if ts == nil {
		t.Fatal("expected task spend, got nil")
	}
	if ts.TotalCost != 40 {
		t.Errorf("TotalCost = %d, want 40", ts.TotalCost)
	}
	if ts.Attempts != 3 {
		t.Errorf("Attempts = %d, want 3", ts.Attempts)
	}
}

func TestTaskTracker_UpdateStatus(t *testing.T) {
	tt := NewTaskTracker()
	tt.RecordSpend("task-1", "tok-a", "bud-a", "", "tool_x", 10)

	tt.UpdateStatus("task-1", "completed")
	ts := tt.GetTaskSpend("task-1")
	if ts.Status != "completed" {
		t.Errorf("Status = %q, want %q", ts.Status, "completed")
	}

	tt.UpdateStatus("task-1", "failed")
	ts = tt.GetTaskSpend("task-1")
	if ts.Status != "failed" {
		t.Errorf("Status = %q, want %q", ts.Status, "failed")
	}

	// Updating non-existent task is a no-op
	tt.UpdateStatus("nonexistent", "completed")
}

func TestTaskTracker_GetTasksByToken(t *testing.T) {
	tt := NewTaskTracker()
	tt.RecordSpend("task-1", "tok-a", "bud-a", "", "tool_x", 10)
	tt.RecordSpend("task-2", "tok-a", "bud-a", "", "tool_y", 20)
	tt.RecordSpend("task-3", "tok-b", "bud-b", "", "tool_x", 30)

	tasksA := tt.GetTasksByToken("tok-a")
	if len(tasksA) != 2 {
		t.Errorf("expected 2 tasks for tok-a, got %d", len(tasksA))
	}

	tasksB := tt.GetTasksByToken("tok-b")
	if len(tasksB) != 1 {
		t.Errorf("expected 1 task for tok-b, got %d", len(tasksB))
	}

	tasksC := tt.GetTasksByToken("tok-unknown")
	if len(tasksC) != 0 {
		t.Errorf("expected 0 tasks for unknown token, got %d", len(tasksC))
	}
}

func TestTaskTracker_TaskCostSoFar(t *testing.T) {
	tt := NewTaskTracker()

	// Unknown task returns 0
	if got := tt.TaskCostSoFar("nope"); got != 0 {
		t.Errorf("TaskCostSoFar(unknown) = %d, want 0", got)
	}

	tt.RecordSpend("task-1", "tok-a", "bud-a", "", "tool_x", 15)
	tt.RecordSpend("task-1", "tok-a", "bud-a", "", "tool_x", 25)

	if got := tt.TaskCostSoFar("task-1"); got != 40 {
		t.Errorf("TaskCostSoFar = %d, want 40", got)
	}
}

func TestTaskTracker_GetTaskSpend_NotFound(t *testing.T) {
	tt := NewTaskTracker()
	if ts := tt.GetTaskSpend("nonexistent"); ts != nil {
		t.Errorf("expected nil for nonexistent task, got %+v", ts)
	}
}

func TestTaskTracker_GetTaskSpend_ReturnsCopy(t *testing.T) {
	tt := NewTaskTracker()
	tt.RecordSpend("task-1", "tok-a", "bud-a", "", "tool_x", 10)

	ts := tt.GetTaskSpend("task-1")
	ts.TotalCost = 999 // mutate the copy

	original := tt.GetTaskSpend("task-1")
	if original.TotalCost != 10 {
		t.Errorf("GetTaskSpend returned reference, not copy: TotalCost = %d", original.TotalCost)
	}
}

func TestTaskTracker_ConcurrentAccess(t *testing.T) {
	tt := NewTaskTracker()
	var wg sync.WaitGroup
	const goroutines = 50
	const opsPerGoroutine = 100

	wg.Add(goroutines)
	for i := 0; i < goroutines; i++ {
		go func(id int) {
			defer wg.Done()
			taskID := fmt.Sprintf("task-%d", id%5) // 5 tasks shared across goroutines
			for j := 0; j < opsPerGoroutine; j++ {
				tt.RecordSpend(taskID, "tok", "bud", "", "tool", 1)
				tt.GetTaskSpend(taskID)
				tt.TaskCostSoFar(taskID)
				tt.GetTasksByToken("tok")
				if j%10 == 0 {
					tt.UpdateStatus(taskID, "running")
				}
			}
		}(i)
	}

	wg.Wait()

	// Verify no data corruption — total cost across all 5 tasks should equal goroutines * opsPerGoroutine
	var totalCost int64
	for i := 0; i < 5; i++ {
		totalCost += tt.TaskCostSoFar(fmt.Sprintf("task-%d", i))
	}
	if totalCost != goroutines*opsPerGoroutine {
		t.Errorf("total cost = %d, want %d", totalCost, goroutines*opsPerGoroutine)
	}
}
