package mcpserver

import (
	"context"
	"testing"
)

func TestInMemoryBudgetEnforcer_SpendAndExhaust(t *testing.T) {
	ctx := context.Background()
	e := NewInMemoryBudgetEnforcer()

	if err := e.Initialize(ctx, "tok1", 100); err != nil {
		t.Fatal(err)
	}

	// Spend 30
	r, err := e.Spend(ctx, "tok1", "tool_a", 30, "req1")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Allowed || r.Remaining != 70 {
		t.Errorf("expected allowed=true remaining=70, got allowed=%v remaining=%d", r.Allowed, r.Remaining)
	}

	// Spend 70 (exact exhaust)
	r, err = e.Spend(ctx, "tok1", "tool_b", 70, "req2")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Allowed || r.Remaining != 0 {
		t.Errorf("expected allowed=true remaining=0, got allowed=%v remaining=%d", r.Allowed, r.Remaining)
	}

	// Spend 1 more — should fail
	r, err = e.Spend(ctx, "tok1", "tool_a", 1, "req3")
	if err == nil {
		t.Fatal("expected error on exhausted budget")
	}
	if r.Allowed {
		t.Error("expected allowed=false")
	}
	if r.Remaining != 0 {
		t.Errorf("expected remaining=0, got %d", r.Remaining)
	}

	// Check spend log
	log := e.SpendLog()
	if len(log) != 2 {
		t.Errorf("expected 2 spend records, got %d", len(log))
	}
}

func TestInMemoryBudgetEnforcer_UnlimitedWhenNotInitialized(t *testing.T) {
	ctx := context.Background()
	e := NewInMemoryBudgetEnforcer()

	// No budget set — should allow everything
	r, err := e.Spend(ctx, "unknown", "tool", 999, "req1")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Allowed {
		t.Error("expected unlimited when no budget set")
	}
	if r.Remaining != -1 {
		t.Errorf("expected remaining=-1 (unlimited), got %d", r.Remaining)
	}
}

func TestInMemoryBudgetEnforcer_SpendWithTask(t *testing.T) {
	ctx := context.Background()
	e := NewInMemoryBudgetEnforcer()

	if err := e.Initialize(ctx, "tok1", 100); err != nil {
		t.Fatal(err)
	}

	// SpendWithTask should record the task ID
	r, err := e.SpendWithTask(ctx, "tok1", "tool_a", 20, "req1", "task-abc-123")
	if err != nil {
		t.Fatal(err)
	}
	if !r.Allowed || r.Remaining != 80 {
		t.Errorf("expected allowed=true remaining=80, got allowed=%v remaining=%d", r.Allowed, r.Remaining)
	}

	log := e.SpendLog()
	if len(log) != 1 {
		t.Fatalf("expected 1 spend record, got %d", len(log))
	}
	if log[0].TaskID != "task-abc-123" {
		t.Errorf("TaskID = %q, want %q", log[0].TaskID, "task-abc-123")
	}

	// SpendWithTask with empty taskID should still work (backward compat)
	_, err = e.SpendWithTask(ctx, "tok1", "tool_b", 10, "req2", "")
	if err != nil {
		t.Fatal(err)
	}

	log = e.SpendLog()
	if len(log) != 2 {
		t.Fatalf("expected 2 spend records, got %d", len(log))
	}
	if log[1].TaskID != "" {
		t.Errorf("TaskID = %q, want empty", log[1].TaskID)
	}
}

func TestStaticCostResolver(t *testing.T) {
	r := NewStaticCostResolver(map[string]int64{
		"db_query":       1,
		"db_write*":      5,
		"gpt4*":          10,
		"dalle_generate": 50,
		"*":              3,
	}, 1)

	tests := []struct {
		tool string
		want int64
	}{
		{"db_query", 1},        // exact match
		{"dalle_generate", 50}, // exact match
		{"db_write_row", 5},    // wildcard prefix
		{"db_write", 5},        // wildcard prefix (exact prefix)
		{"gpt4_summarize", 10}, // wildcard prefix
		{"gpt4", 10},           // wildcard prefix
		{"unknown_tool", 3},    // catch-all "*"
	}

	for _, tt := range tests {
		got := r.Resolve(tt.tool)
		if got != tt.want {
			t.Errorf("Resolve(%q) = %d, want %d", tt.tool, got, tt.want)
		}
	}
}

func TestStaticCostResolver_NilCosts(t *testing.T) {
	r := NewStaticCostResolver(nil, 5)
	if got := r.Resolve("anything"); got != 5 {
		t.Errorf("expected default 5, got %d", got)
	}
}
