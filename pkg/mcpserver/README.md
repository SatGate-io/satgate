# SatGate MCP Proxy

A transparent MCP (Model Context Protocol) proxy that sits between AI agents and upstream MCP servers, enforcing per-tool budgets with cryptographic token delegation.

## Architecture

```
Agent(s) → [stdio/SSE] → SatGate MCP Proxy → [stdio] → Upstream MCP Server(s)
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
  transport: stdio  # or sse (SSE + Streamable HTTP)
  # host: 127.0.0.1 # for sse: listen address (default 127.0.0.1, this machine only)
  # port: 9100      # for sse: listen port (default 9100)

# Required for sse. "none" is for stdio or a listener only you can reach.
auth:
  mode: none        # none | config (static token) | header (macaroons)

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
  mode: hard  # hard | soft | shadow (left out: control when budget.limit > 0, else observe)
```

### 2. Run

```bash
# stdio mode (single agent, local)
satgate-mcp --config satgate-mcp.yaml

# SSE mode (multiple agents, remote)
# Config: server.transport: sse, auth.mode: header (or config), server.host: 0.0.0.0 to accept other machines
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

| Mode | Use Case | SSE / Streamable HTTP request without a token |
|------|----------|-----------------------------------------------|
| `none` | stdio, or a listener only you can reach | answered under the default identity |
| `config` | Single static token in config file (`auth.token`) | refused |
| `header` | Per-request macaroon verification (production) | refused |

With `config` or `header`, every method over SSE and Streamable HTTP except
`initialize`, `ping` and notifications needs a token that verifies:
`tools/call`, `tools/list`, the `satgate/*` methods and every method passed on
to an upstream. A missing or empty token, or one of only whitespace, gets
JSON-RPC error `-32001`
`authentication required: no token provided` (the same answer in both modes),
and Streamable HTTP `initialize` without one gets `401`. Over stdio the caller
is the local process that started the proxy: `config` mode runs a request
without a token under the configured identity, and `tools/list` needs no
token. The static token is compared in constant time. In `config` mode
`auth.token` must be non-empty: a value that is empty or only whitespace is a
configuration error (`LoadConfig`, `New`, and the listener and handler checks).

Over SSE and Streamable HTTP the token goes in the `Authorization` header
(`Bearer` scheme). By default any request to `GET /sse`, `POST /message` or the
Streamable HTTP endpoint whose URL has a `token` query parameter is refused
(401), whatever its value (empty, repeated, or next to a header), and the value
is not repeated in the response or the log. `server.allowQueryToken: true`
restores the URL fallback for a client that cannot send a header.

## Defaults

- `server.transport` `sse` / `http` without `auth.mode` does not start; write
  `auth.mode: none` to serve without authentication (a warning is logged at
  startup).
  An embedded handler (`SSEServer.Handler`) with no `auth.mode` refuses every
  request with `503` and the same message.
- The listener binds `server.host`, default `127.0.0.1`, on `server.port`,
  default `9100` (`Config.ListenAddr`). Set `host: 0.0.0.0` for every
  interface.
- `enforcement.mode` left out: `control` when `budget.limit` is greater than
  0, otherwise `observe` (applied by `LoadConfig`).
- stdio-only configurations start as before.

## Upgrading

- `auth.mode: config` clients must send the token (`Authorization: Bearer <token>`)
  over SSE and Streamable HTTP; requests without it are now refused.
- `tools/list` and passed-on methods need a token in `config` and `header`
  modes over SSE and Streamable HTTP.
- A config-mode token must be non-empty. `auth.mode: config` with an
  `auth.token` that is empty or only whitespace (an environment variable that
  expands to nothing, for example) no longer loads, on any transport; an
  embedded handler or listener whose `Config` has such a value answers `503`
  with the `auth.token is required ...` message. A presented token of only
  whitespace counts as no token.
- Add `auth.mode` to every `sse` / `http` configuration, `server.host: 0.0.0.0`
  where other machines (or a container's port mapping) must connect, and
  `enforcement.mode: observe` where a budget should only be watched.
- Code that calls `SSEServer.ListenAndServe` must set `Auth.Mode`.
  `NewSSEServer` still listens on the address it is given.
- Embedded handlers follow the same rule. When `Auth.Mode` is not set, a
  handler from `NewSSEServer(...).Handler()`, or the `SSEServer` used as an
  `http.Handler`, answers every request with `503` and the
  `auth.mode is required ...` message. That covers every route, including
  `/health` and routes added with `Handle` / `HandleFunc`. "Not set" means
  `Auth.Mode` is empty, or the configuration came from a file loaded without
  `auth.mode`, stdio-only files included. The error is logged once, by
  `NewSSEServer`. Set `Auth.Mode` to `header` or `config`, or to `none` to
  serve without authentication (logged once as a warning).

See [docs/guides/mcp-gateway.md](../../docs/guides/mcp-gateway.md#upgrading).

## What Upstreams Receive

- **Environment (stdio):** a fixed base set (`PATH`, `HOME`, locale, temp dir,
  `TZ` and the like; see `stdioBaseEnv` in `upstream_hygiene.go`), plus the
  variables named in the upstream's `envPassthrough` list, plus its `env` map,
  which wins (also where names compare without regard to case, as on Windows).
  Nothing else from the proxy's environment.
- **Caller token:** never, by default. `_meta.token` is removed from every
  forwarded request. `forwardCallerToken: true` on an upstream passes that
  upstream the credential the proxy verified for the request (the token
  authentication accepted, or for a method passed on outside the `tools/call` path
  the connection's own credential once it verifies and is not revoked), never a value from params that it did not verify; with none
  verified, none is passed.
  That upstream can then act as the caller.

```yaml
upstreams:
  tools:
    transport: stdio
    command: ["node", "server.js"]
    env:
      API_KEY: "${TOOLS_API_KEY}"
    envPassthrough: [HTTPS_PROXY]
    # forwardCallerToken: true   # only if this upstream verifies SatGate tokens itself
```

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
