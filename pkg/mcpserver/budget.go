package mcpserver

import (
	"context"
	"fmt"
	"strings"
	"sync"
)

// BudgetResult contains the outcome of a budget check or spend operation.
type BudgetResult struct {
	Allowed   bool   `json:"allowed"`
	Remaining int64  `json:"remaining"`
	TokenID   string `json:"token_id"`
	Cost      int64  `json:"cost"`
	RequestID string `json:"request_id,omitempty"`
	ErrorCode string `json:"error_code,omitempty"`
}

// BudgetEnforcer is the interface for budget enforcement.
// OSS provides InMemoryBudgetEnforcer; Enterprise provides RedisBudgetEnforcer.
type BudgetEnforcer interface {
	// Check returns whether the token has enough budget for the given cost
	// without actually spending. Used for pre-flight checks.
	Check(ctx context.Context, tokenID string, cost int64) (*BudgetResult, error)

	// Spend atomically decrements the budget. Returns ErrBudgetExhausted
	// if insufficient credits remain.
	Spend(ctx context.Context, tokenID string, toolName string, cost int64, requestID string) (*BudgetResult, error)

	// Remaining returns the current budget for a token.
	Remaining(ctx context.Context, tokenID string) (int64, error)

	// Initialize sets the starting budget for a token.
	Initialize(ctx context.Context, tokenID string, credits int64) error
}

// CostResolver determines the credit cost for a tool call.
type CostResolver interface {
	// Resolve returns the credit cost for the given tool name.
	Resolve(toolName string) int64
}

// TenantCostResolver resolves costs per-tenant. Used in multi-tenant SaaS mode.
// If set on the proxy, it takes priority over the global CostResolver.
type TenantCostResolver interface {
	// ResolveForTenant returns the credit cost for a tool call scoped to a tenant.
	ResolveForTenant(tenantID string, toolName string) int64
}

// --- In-memory budget enforcer (OSS) ---

// InMemoryBudgetEnforcer provides single-instance budget enforcement.
// Safe for concurrent use. Not durable across restarts.
type InMemoryBudgetEnforcer struct {
	mu      sync.Mutex
	budgets map[string]int64 // tokenID -> remaining credits
	spent   []SpendRecord    // append-only receipt-ready spend records
}

// SpendRecord is an in-memory, receipt-ready spend record.
type SpendRecord struct {
	TokenID   string `json:"token_id"`
	ToolName  string `json:"tool_name"`
	Cost      int64  `json:"cost"`
	Remaining int64  `json:"remaining"`
	RequestID string `json:"request_id"`
	TaskID    string `json:"task_id,omitempty"` // MCP task correlation (SEP-1686)
}

// NewInMemoryBudgetEnforcer creates a new in-memory budget enforcer.
func NewInMemoryBudgetEnforcer() *InMemoryBudgetEnforcer {
	return &InMemoryBudgetEnforcer{
		budgets: make(map[string]int64),
	}
}

func (e *InMemoryBudgetEnforcer) Initialize(_ context.Context, tokenID string, credits int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	// Only initialize if not already set (idempotent — don't reset spend)
	if _, exists := e.budgets[tokenID]; !exists {
		e.budgets[tokenID] = credits
	}
	return nil
}

func (e *InMemoryBudgetEnforcer) Check(_ context.Context, tokenID string, cost int64) (*BudgetResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	remaining, ok := e.budgets[tokenID]
	if !ok {
		// No budget configured = unlimited
		return &BudgetResult{Allowed: true, Remaining: -1, TokenID: tokenID, Cost: cost}, nil
	}

	if remaining < cost {
		errorCode := "budget_exhausted"
		if remaining > 0 {
			errorCode = "insufficient_budget"
		}
		return &BudgetResult{
			Allowed:   false,
			Remaining: remaining,
			TokenID:   tokenID,
			Cost:      cost,
			ErrorCode: errorCode,
		}, nil
	}

	return &BudgetResult{Allowed: true, Remaining: remaining, TokenID: tokenID, Cost: cost}, nil
}

func (e *InMemoryBudgetEnforcer) Spend(_ context.Context, tokenID string, toolName string, cost int64, requestID string) (*BudgetResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	remaining, ok := e.budgets[tokenID]
	if !ok {
		// No budget = unlimited
		return &BudgetResult{Allowed: true, Remaining: -1, TokenID: tokenID, Cost: cost}, nil
	}

	if remaining < cost {
		return &BudgetResult{
			Allowed:   false,
			Remaining: remaining,
			TokenID:   tokenID,
			Cost:      cost,
			ErrorCode: "budget_exhausted",
		}, fmt.Errorf("budget exhausted: %d remaining, %d required", remaining, cost)
	}

	remaining -= cost
	e.budgets[tokenID] = remaining

	e.spent = append(e.spent, SpendRecord{
		TokenID:   tokenID,
		ToolName:  toolName,
		Cost:      cost,
		Remaining: remaining,
		RequestID: requestID,
	})

	return &BudgetResult{Allowed: true, Remaining: remaining, TokenID: tokenID, Cost: cost}, nil
}

// SpendWithTask is like Spend but also records the MCP task ID for correlation.
// The budget accounting is identical — the task ID is metadata-only.
func (e *InMemoryBudgetEnforcer) SpendWithTask(ctx context.Context, tokenID string, toolName string, cost int64, requestID string, taskID string) (*BudgetResult, error) {
	result, err := e.Spend(ctx, tokenID, toolName, cost, requestID)
	if err != nil {
		return result, err
	}

	// Patch the last spend record with the task ID
	if taskID != "" {
		e.mu.Lock()
		if len(e.spent) > 0 {
			e.spent[len(e.spent)-1].TaskID = taskID
		}
		e.mu.Unlock()
	}

	return result, nil
}

func (e *InMemoryBudgetEnforcer) Remaining(_ context.Context, tokenID string) (int64, error) {
	e.mu.Lock()
	defer e.mu.Unlock()

	remaining, ok := e.budgets[tokenID]
	if !ok {
		return -1, nil // unlimited
	}
	return remaining, nil
}

// SpendLog returns receipt-ready spend records for debugging/testing, not canonical Evidence Pack export.
func (e *InMemoryBudgetEnforcer) SpendLog() []SpendRecord {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]SpendRecord, len(e.spent))
	copy(out, e.spent)
	return out
}

// --- Static cost resolver (YAML config) ---

// StaticCostResolver resolves tool costs from a static map with wildcard support.
// Matching order: exact → longest wildcard prefix → catch-all "*" → default.
type StaticCostResolver struct {
	costs       map[string]int64
	defaultCost int64
}

// NewStaticCostResolver creates a cost resolver from a map of tool patterns to costs.
func NewStaticCostResolver(costs map[string]int64, defaultCost int64) *StaticCostResolver {
	return &StaticCostResolver{costs: costs, defaultCost: defaultCost}
}

func (r *StaticCostResolver) Resolve(toolName string) int64 {
	if r == nil || r.costs == nil {
		return r.defaultCost
	}

	// Exact match
	if cost, ok := r.costs[toolName]; ok {
		return cost
	}

	// Longest wildcard prefix match
	bestLen := -1
	bestCost := int64(-1)
	catchAll := int64(-1)

	for pattern, cost := range r.costs {
		if !strings.HasSuffix(pattern, "*") {
			continue
		}
		if pattern == "*" {
			catchAll = cost
			continue
		}
		prefix := strings.TrimSuffix(pattern, "*")
		if strings.HasPrefix(toolName, prefix) && len(prefix) > bestLen {
			bestLen = len(prefix)
			bestCost = cost
		}
	}

	if bestCost >= 0 {
		return bestCost
	}
	if catchAll >= 0 {
		return catchAll
	}
	return r.defaultCost
}
