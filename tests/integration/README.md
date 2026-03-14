# Integration Tests

End-to-end tests for the SatGate MCP proxy, covering budget enforcement, macaroon auth, delegation, and task-level cost tracking (SEP-1686).

## Running

```bash
# From repo root
make test-integration

# Or directly
go test -v -count=1 -timeout 120s ./tests/integration/...
```

## Prerequisites

- Go 1.25+
- Python 3 (used for mock MCP upstream subprocesses)
- No external services required (Redis, etc.)

## Test Files

| File | What it tests |
|---|---|
| `harness.go` | Shared test infrastructure: `TestEnv` builder, event collector, mock upstream helpers, task response builders |
| `proxy_test.go` | MCP proxy E2E: tool calls, budget exhaustion (402), expired/invalid tokens, soft enforcement, delegation, tools/list, ping |
| `task_cost_test.go` | Task-level cost tracking: single/multi-step tasks, status propagation, per-token isolation, `SpendWithTask`, event emission |
| `api_test.go` | API contract tests: TaskTracker queries, status filtering, JSON serialization, event format, pagination math |

## Architecture

Each test creates an isolated `TestEnv` that wires up:

1. **Mock MCP upstream** — A Python subprocess acting as an MCP server (configurable tool responses, including task IDs in `_meta`)
2. **Proxy** — Real `mcpserver.Proxy` instance with in-memory budget enforcer
3. **Event collector** — Captures all events for assertion (implements `EventPublisher`)
4. **Macaroon auth** — Real macaroon service for token minting/verification

Communication uses Go pipe pairs (client ↔ proxy ↔ upstream), so tests are fully in-process with no network I/O.

## Adding Tests

Use the `TestEnv` builder with functional options:

```go
env := NewTestEnv(t,
    WithBudget(1000),
    WithEnforcement("hard"),
    WithTools(MockTool{Name: "my-tool", Description: "test"}),
    WithToolCosts(map[string]int64{"my-tool": 25}),
    WithToolResponseJSON(map[string]json.RawMessage{
        "my-tool": TaskResponse("task-123", "result text"),
    }),
)
defer env.Close()

resp := env.ToolCall(t, 1, "my-tool", env.RootToken)
// assert on resp, env.Budget, env.Events, env.Proxy.TaskTracker()
```
