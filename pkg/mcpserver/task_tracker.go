package mcpserver

import (
	"sync"
	"time"
)

// TaskTracker correlates budget spend with MCP task IDs.
// Phase 1: in-memory tracking for task-level cost aggregation.
// When an upstream MCP server returns a task response (SEP-1686), the proxy
// records spend against the task ID so callers can query cumulative cost.
type TaskTracker struct {
	mu    sync.RWMutex
	tasks map[string]*TaskSpend // taskID -> aggregated spend
}

// TaskSpend holds the aggregated spend for a single MCP task.
type TaskSpend struct {
	TaskID    string    `json:"task_id"`
	TokenID   string    `json:"token_id"`
	BudgetID  string    `json:"budget_id"`
	TenantID  string    `json:"tenant_id,omitempty"`
	ToolName  string    `json:"tool_name"`
	TotalCost int64     `json:"total_cost"`
	Attempts  int       `json:"attempts"`
	FirstSeen time.Time `json:"first_seen"`
	LastSeen  time.Time `json:"last_seen"`
	Status    string    `json:"status"` // "running", "completed", "failed"
}

// NewTaskTracker creates a new in-memory task tracker.
func NewTaskTracker() *TaskTracker {
	return &TaskTracker{
		tasks: make(map[string]*TaskSpend),
	}
}

// RecordSpend creates or updates the spend record for a task.
// If the task is seen for the first time, a new TaskSpend is created.
// Subsequent calls accumulate TotalCost and increment Attempts.
func (t *TaskTracker) RecordSpend(taskID, tokenID, budgetID, tenantID, toolName string, cost int64) {
	t.mu.Lock()
	defer t.mu.Unlock()

	now := time.Now()
	ts, ok := t.tasks[taskID]
	if !ok {
		t.tasks[taskID] = &TaskSpend{
			TaskID:    taskID,
			TokenID:   tokenID,
			BudgetID:  budgetID,
			TenantID:  tenantID,
			ToolName:  toolName,
			TotalCost: cost,
			Attempts:  1,
			FirstSeen: now,
			LastSeen:  now,
			Status:    "running",
		}
		return
	}

	ts.TotalCost += cost
	ts.Attempts++
	ts.LastSeen = now
}

// UpdateStatus sets the status of a tracked task (e.g., "running", "completed", "failed").
func (t *TaskTracker) UpdateStatus(taskID, status string) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if ts, ok := t.tasks[taskID]; ok {
		ts.Status = status
		ts.LastSeen = time.Now()
	}
}

// GetTaskSpend returns a copy of the spend record for a task, or nil if not found.
func (t *TaskTracker) GetTaskSpend(taskID string) *TaskSpend {
	t.mu.RLock()
	defer t.mu.RUnlock()

	ts, ok := t.tasks[taskID]
	if !ok {
		return nil
	}
	cp := *ts
	return &cp
}

// GetTasksByToken returns all task spend records for a given token ID.
func (t *TaskTracker) GetTasksByToken(tokenID string) []*TaskSpend {
	t.mu.RLock()
	defer t.mu.RUnlock()

	var result []*TaskSpend
	for _, ts := range t.tasks {
		if ts.TokenID == tokenID {
			cp := *ts
			result = append(result, &cp)
		}
	}
	return result
}

// TaskCostSoFar returns the cumulative cost for a task, or 0 if not tracked.
func (t *TaskTracker) TaskCostSoFar(taskID string) int64 {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if ts, ok := t.tasks[taskID]; ok {
		return ts.TotalCost
	}
	return 0
}
