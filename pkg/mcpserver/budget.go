package mcpserver

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"
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

// ErrInsufficientBudget is wrapped by a store that finds out, when it creates
// the child's budget, that the parent could not afford the carve. A store whose
// Spend only reserves (the enterprise Redis store moves money when the child
// budget is created) reports a real refusal this way, so the delegator can tell
// "the budget said no" from "the store broke".
var ErrInsufficientBudget = errors.New("insufficient budget")

// budgetRefusal is the error InMemoryBudgetEnforcer.Spend returns with a
// refused result. It matches ErrInsufficientBudget, so a caller can tell it
// from a store failure by type.
type budgetRefusal struct{ msg string }

func (e *budgetRefusal) Error() string        { return e.msg }
func (e *budgetRefusal) Is(target error) bool { return target == ErrInsufficientBudget }

// BudgetRefunder is implemented by budget stores that can give back what one
// request's Spend debited. It is a separate interface so a store that cannot
// give back still satisfies BudgetEnforcer.
//
// Contract:
//   - It adds back what the store recorded that request ID debiting from that
//     budget, as an increment. It never writes an absolute balance, so a spend
//     made after the request's debit stays spent.
//   - At most once per request ID: a second call changes nothing.
//   - Only a debit the store recorded counts. A budget with no record
//     (unlimited), a refused spend, an unknown request ID, or a deferred
//     reservation that moved no money is never credited, and is not an error.
//   - amount is the most the caller expects back; the store never credits more
//     than it recorded.
type BudgetRefunder interface {
	Refund(ctx context.Context, budgetID, requestID string, amount int64) error
}

// BudgetHolder is implemented by budget stores that can set an amount aside for
// one request and keep it apart until the request is known to be one that may
// go ahead. With "sign every action" on, a priced Control call and a delegation
// carve are taken as a hold, never as a debit: the call is signed, and only
// then is the hold committed. If the signature fails, the hold is released; if
// even that fails, the hold lapses by itself after its time limit and the
// balance comes back with no further call. That is what makes "not sent and
// not charged" hold when no later step succeeds.
//
// Contract:
//   - Hold takes amount from budgetID exactly as Spend does (the same result,
//     the same refusal and the same error for an exhausted budget; a budget with
//     no record is unlimited and holds nothing), but the amount is only set
//     aside. Other requests see the balance without it.
//   - The hold lapses on its own at ttl after it was taken: the store gives the
//     amount back to the balance with no call from the proxy. A lapse needs no
//     clock or process of the proxy's; a store that keeps holds in a database
//     gives them an expiry the database enforces.
//   - CommitHold turns the hold into a debit, the same debit Spend records (so
//     Refund can give it back). It fails if the hold is not there any more
//     (ErrHoldGone: it lapsed, or was released).
//   - ReleaseHold gives the hold back now. Releasing a hold that is gone (a
//     second release, a lapse, a commit) is not an error.
//   - Holds are named by budgetID and requestID. If two holds of one budget
//     carry the same requestID at once, each CommitHold or ReleaseHold ends the
//     earliest of them.
//   - Every call honours its context, and none depends on the request's: the
//     proxy runs them on a context of its own with a bound.
type BudgetHolder interface {
	Hold(ctx context.Context, budgetID, toolName string, cost int64, requestID string, ttl time.Duration) (*BudgetResult, error)
	CommitHold(ctx context.Context, budgetID, requestID string) error
	ReleaseHold(ctx context.Context, budgetID, requestID string) error
}

// ErrHoldGone is what CommitHold returns when the hold it names is not there:
// it lapsed, or it was released.
var ErrHoldGone = errors.New("the budget hold is gone")

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

// CheckedTenantCostResolver is an optional addition to TenantCostResolver for a
// resolver that can fail to know a price (it reads it from a store that can be
// down). The proxy uses it, when the tenant resolver has it, instead of
// ResolveForTenant.
//
// ResolveForTenantChecked returns the price and a nil error when the price is
// known: read now, cached, or the last good one the resolver holds. It returns
// a non-nil error when it could not read the price and holds no usable one. The
// price returned with an error is the one ResolveForTenant would have given (a
// guess); the proxy never charges it in a mode that charges or blocks. It uses
// it only in observe mode, which never blocks (see handleToolsCall).
//
// A tool that is free whatever the profile says is (0, nil) with no read.
// ctx is the request's context.
type CheckedTenantCostResolver interface {
	ResolveForTenantChecked(ctx context.Context, tenantID string, toolName string) (int64, error)
}

// ToolPriceUnavailableMessage is the JSON-RPC error message for a tools/call
// refused because its price could not be read (see CheckedTenantCostResolver).
// It is fixed text: it carries no store error, key or tenant. The error data
// has "error":"tool_price_unavailable" and "retryable":true. It differs from
// "Budget service unavailable" on purpose: a gateway in front of the proxy may
// treat that one specially.
const ToolPriceUnavailableMessage = "Tool price temporarily unavailable; the call was not sent and nothing was charged. Retry."

// --- In-memory budget enforcer (OSS) ---

// InMemoryBudgetEnforcer provides single-instance budget enforcement.
// Safe for concurrent use. Not durable across restarts.
type InMemoryBudgetEnforcer struct {
	mu      sync.Mutex
	budgets map[string]int64 // tokenID -> remaining credits
	spent   []SpendRecord    // append-only receipt-ready spend records
	// debits is what each admitted Spend took, by budget and request ID, until
	// Refund gives it back. A spend that took nothing records nothing.
	debits map[string]int64
	// holds are the amounts set aside by Hold and not yet committed or
	// released, by budget and request id, oldest first (see BudgetHolder).
	holds map[string][]memHold
	// nextLapse is the earliest expiry among holds; zero when there are none.
	nextLapse time.Time
	// now is the clock; nil means time.Now. Tests set it.
	now func() time.Time
}

type memHold struct {
	budgetID  string
	tool      string
	amount    int64
	remaining int64
	expires   time.Time
}

var (
	_ BudgetRefunder = (*InMemoryBudgetEnforcer)(nil)
	_ BudgetHolder   = (*InMemoryBudgetEnforcer)(nil)
)

func debitKey(budgetID, requestID string) string {
	return budgetID + "\x00" + requestID
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
		debits:  make(map[string]int64),
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
	e.lapseLocked()

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
	e.lapseLocked()

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
		}, &budgetRefusal{msg: fmt.Sprintf("budget exhausted: %d remaining, %d required", remaining, cost)}
	}

	remaining -= cost
	e.budgets[tokenID] = remaining
	if cost > 0 {
		if e.debits == nil {
			e.debits = make(map[string]int64)
		}
		e.debits[debitKey(tokenID, requestID)] += cost
	}

	e.spent = append(e.spent, SpendRecord{
		TokenID:   tokenID,
		ToolName:  toolName,
		Cost:      cost,
		Remaining: remaining,
		RequestID: requestID,
	})

	return &BudgetResult{Allowed: true, Remaining: remaining, TokenID: tokenID, Cost: cost}, nil
}

// Refund gives back what Spend recorded for this request ID on this budget, as
// an increment under the store's lock, and forgets the record so a second call
// changes nothing.
func (e *InMemoryBudgetEnforcer) Refund(_ context.Context, budgetID, requestID string, amount int64) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lapseLocked()
	if amount <= 0 {
		return nil
	}
	key := debitKey(budgetID, requestID)
	recorded, ok := e.debits[key]
	if !ok {
		return nil
	}
	delete(e.debits, key)
	if _, exists := e.budgets[budgetID]; !exists {
		return nil
	}
	if amount < recorded {
		recorded = amount
	}
	e.budgets[budgetID] += recorded
	return nil
}

func (e *InMemoryBudgetEnforcer) clock() time.Time {
	if e.now != nil {
		return e.now()
	}
	return time.Now()
}

// lapseLocked gives back every hold whose time has run out. It is called by
// every operation that reads or changes a balance, under the lock, so a lapsed
// hold is never seen as spent. With no holds it does nothing.
func (e *InMemoryBudgetEnforcer) lapseLocked() {
	if len(e.holds) == 0 {
		return
	}
	now := e.clock()
	if now.Before(e.nextLapse) {
		return
	}
	e.nextLapse = time.Time{}
	for key, list := range e.holds {
		kept := list[:0]
		for _, h := range list {
			if !now.Before(h.expires) {
				if _, ok := e.budgets[h.budgetID]; ok {
					e.budgets[h.budgetID] += h.amount
				}
				continue
			}
			if e.nextLapse.IsZero() || h.expires.Before(e.nextLapse) {
				e.nextLapse = h.expires
			}
			kept = append(kept, h)
		}
		if len(kept) == 0 {
			delete(e.holds, key)
		} else {
			e.holds[key] = kept
		}
	}
}

// Hold sets cost aside from budgetID for requestID (see BudgetHolder). The
// answer is Spend's: the same result and the same refusal.
func (e *InMemoryBudgetEnforcer) Hold(_ context.Context, budgetID string, toolName string, cost int64, requestID string, ttl time.Duration) (*BudgetResult, error) {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lapseLocked()

	remaining, ok := e.budgets[budgetID]
	if !ok {
		// No budget = unlimited: nothing to set aside.
		return &BudgetResult{Allowed: true, Remaining: -1, TokenID: budgetID, Cost: cost}, nil
	}
	if remaining < cost {
		return &BudgetResult{
			Allowed:   false,
			Remaining: remaining,
			TokenID:   budgetID,
			Cost:      cost,
			ErrorCode: "budget_exhausted",
		}, &budgetRefusal{msg: fmt.Sprintf("budget exhausted: %d remaining, %d required", remaining, cost)}
	}
	if ttl <= 0 {
		return nil, errors.New("a budget hold needs a time limit")
	}
	remaining -= cost
	e.budgets[budgetID] = remaining
	if cost > 0 {
		if e.holds == nil {
			e.holds = make(map[string][]memHold)
		}
		h := memHold{budgetID: budgetID, tool: toolName, amount: cost, remaining: remaining, expires: e.clock().Add(ttl)}
		key := debitKey(budgetID, requestID)
		e.holds[key] = append(e.holds[key], h)
		if e.nextLapse.IsZero() || h.expires.Before(e.nextLapse) {
			e.nextLapse = h.expires
		}
	}
	return &BudgetResult{Allowed: true, Remaining: remaining, TokenID: budgetID, Cost: cost}, nil
}

// takeHoldLocked removes and returns the earliest open hold of a request.
func (e *InMemoryBudgetEnforcer) takeHoldLocked(budgetID, requestID string) (memHold, bool) {
	key := debitKey(budgetID, requestID)
	list := e.holds[key]
	if len(list) == 0 {
		return memHold{}, false
	}
	h := list[0]
	if len(list) == 1 {
		delete(e.holds, key)
	} else {
		e.holds[key] = list[1:]
	}
	return h, true
}

// CommitHold turns the request's hold into the debit Spend would have taken.
func (e *InMemoryBudgetEnforcer) CommitHold(_ context.Context, budgetID, requestID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lapseLocked()
	h, ok := e.takeHoldLocked(budgetID, requestID)
	if !ok {
		if _, exists := e.budgets[budgetID]; !exists {
			return nil // an unlimited budget holds nothing and commits nothing
		}
		return ErrHoldGone
	}
	if e.debits == nil {
		e.debits = make(map[string]int64)
	}
	e.debits[debitKey(budgetID, requestID)] += h.amount
	e.spent = append(e.spent, SpendRecord{
		TokenID:   budgetID,
		ToolName:  h.tool,
		Cost:      h.amount,
		Remaining: h.remaining,
		RequestID: requestID,
	})
	return nil
}

// ReleaseHold gives the request's hold back now. A hold that is gone is not an
// error.
func (e *InMemoryBudgetEnforcer) ReleaseHold(_ context.Context, budgetID, requestID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.lapseLocked()
	h, ok := e.takeHoldLocked(budgetID, requestID)
	if !ok {
		return nil
	}
	if _, exists := e.budgets[budgetID]; exists {
		e.budgets[budgetID] += h.amount
	}
	return nil
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
	e.lapseLocked()

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
