# SatGate MCP Proxy

A transparent MCP (Model Context Protocol) proxy that sits between AI agents and upstream MCP servers, enforcing per-tool budgets with cryptographic token delegation.

## Architecture

```
Agent(s) → [stdio/SSE] → SatGate MCP Proxy → [stdio/HTTP] → Upstream MCP Server(s)
                               │
                        Budget Enforcement
                        (in-memory / Redis)
                               │
                        Cost Attribution
                        (per-tool pricing)
```

## Features

| Feature | OSS | Enterprise |
|---------|-----|-----------|
| stdio transport | ✓ | ✓ |
| SSE/HTTP transport (multi-agent) | ✓ | ✓ |
| Tool discovery + relay | ✓ | ✓ |
| Per-tool cost profiles (wildcards) | ✓ | ✓ |
| In-memory budget enforcement | ✓ | ✓ |
| Redis-backed budget (HA) | | ✓ |
| Token delegation (parent→child) | ✓ | ✓ |
| Macaroon auth (HMAC chain) | ✓ | ✓ |
| Atomic budget transfer (Lua) | | ✓ |
| Spend ledger (Postgres) | | ✓ |
| Dashboard integration | | ✓ |
| Multi-tenant isolation | | ✓ |

## Quick Start

### 1. Create config

```yaml
# satgate-mcp.yaml
server:
  transport: stdio  # or sse
  port: 9100        # for sse

upstreams:
  my-tools:
    transport: stdio
    command: ["python3", "my_mcp_server.py"]

budget:
  limit: 1000       # total credits

tools:
  defaultCost: 5
  costs:
    web_search: 5
    gpt4_summarize: 25
    dalle_generate: 50

enforcement:
  mode: hard  # hard | soft | shadow
```

### 2. Run

```bash
# stdio mode (single agent, local)
satgate-mcp --config satgate-mcp.yaml

# SSE mode (multiple agents, remote)
# Config: server.transport: sse, server.port: 9100
satgate-mcp --config satgate-mcp-sse.yaml
```

### 3. Connect your agent

The proxy speaks standard MCP — any MCP client works:

```python
# Agent sends tools/call, proxy intercepts and enforces budget
{"jsonrpc":"2.0","id":1,"method":"tools/call",
 "params":{"name":"web_search","arguments":{"query":"test"}}}

# When budget exhausted → JSON-RPC error with budget_exhausted
{"jsonrpc":"2.0","id":42,"error":{
  "code":-32000,
  "message":"Budget exhausted",
  "data":{"error":"budget_exhausted","remaining_credits":0,"cost_credits":5}
}}
```

## Delegation

Parent agents can delegate budgets to sub-agents using SatGate extension methods:

```python
# Parent delegates 200 credits to a child agent
{"jsonrpc":"2.0","id":1,"method":"satgate/delegate",
 "params":{"budget":200,"label":"research-agent","_meta":{"token":"<parent-token>"}}}

# Response includes child token
{"jsonrpc":"2.0","id":1,"result":{
  "token":"<child-macaroon>","tokenId":"abc123",
  "budget":200,"parentRemaining":800
}}

# Child uses their token for tool calls
{"jsonrpc":"2.0","id":1,"method":"tools/call",
 "params":{"name":"search","arguments":{"q":"test"},
            "_meta":{"token":"<child-macaroon>"}}}
```

Budget isolation: when the child exhausts their budget, siblings and parent are unaffected.

## Enforcement Modes

| Mode | Behavior |
|------|----------|
| `hard` | Deny tool call with JSON-RPC error (402 equivalent) |
| `soft` | Allow but log warning (budget advisory) |
| `shadow` | Observe only, no enforcement (dry run) |

## Auth Modes

| Mode | Use Case |
|------|----------|
| `none` | Local dev, single agent |
| `config` | Single static token in config file |
| `header` | Per-request macaroon verification (production) |

## Per-Tool Routing

Route different tools to different upstream servers:

```yaml
upstreams:
  db-server:
    transport: stdio
    command: ["db-mcp-server"]
  ai-server:
    transport: http
    url: "http://ai-service:9090"

routing:
  - tools: ["db_*", "sql_*"]
    upstream: db-server
  - tools: ["gpt4_*", "dalle_*"]
    upstream: ai-server
```

## Testing

```bash
# Unit tests (28 tests)
go test ./pkg/mcpserver/ -v

# E2E: Simple budget enforcement
cd examples/mcp-proxy && python3 test-e2e.py

# E2E: Delegation with budget isolation
cd examples/mcp-proxy && python3 test-delegation.py
```
