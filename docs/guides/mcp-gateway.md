# Gating MCP Servers with SatGate

The `satgate-mcp` proxy sits between AI agents and upstream MCP servers, enforcing per-tool budgets, cost attribution, and access control — with zero code changes to your MCP servers.

> **Note:** `satgate-mcp` is a **separate binary** from the main `satgate` HTTP gateway. It has its own YAML config and speaks the MCP protocol natively (JSON-RPC 2.0 over stdio).

## What is MCP?

MCP is [Anthropic's open standard](https://modelcontextprotocol.io/specification) for connecting AI agents to tools and data sources. An MCP server exposes "tools" (functions) that agents can discover and call via JSON-RPC 2.0.

## Why MCP Servers Need Economic Controls

MCP makes it trivially easy for agents to discover and call tools. That's the point — but it creates a new problem:

- **No cost visibility.** An agent calls 10 tools in a chain. What did that cost? Which team pays?
- **No budget enforcement.** A misconfigured agent can burn through expensive tool calls with no guardrails.
- **No access scoping.** Every agent with the MCP server URL can call every tool.

`satgate-mcp` solves all three by sitting between agents and MCP servers as a protocol-aware proxy.

## Architecture

```
┌──────────────┐     ┌────────────────┐     ┌──────────────┐
│   AI Agent   │────▶│  satgate-mcp   │────▶│  MCP Server  │
│ (Claude Code,│     │    (proxy)     │────▶│  MCP Server  │
│  Agent Zero) │     │                │     │  ...         │
└──────────────┘     └───────┬────────┘     └──────────────┘
                             │
                    budget enforcement
                    cost attribution
                    auth & scoping
                    multi-upstream routing
```

The proxy intercepts every `tools/call`, resolves its cost, checks the caller's budget, and either allows or denies the call — all before forwarding to the upstream MCP server.

## Quick Start

### 1. Create a Config File

```yaml
# satgate-mcp.yaml — minimal config
server:
  transport: stdio
  name: my-mcp-proxy

upstreams:
  filesystem:
    transport: stdio
    command: ["npx", "-y", "@modelcontextprotocol/server-filesystem", "/tmp"]

budget:
  backend: memory
  limit: 1000

tools:
  defaultCost: 1

enforcement:
  mode: shadow    # observe only — no blocking
```

### 2. Run the Proxy

```bash
satgate-mcp --config satgate-mcp.yaml
```

### 3. Point Your Agent at the Proxy

In Claude Code's MCP config (or any MCP client), replace the direct server command with `satgate-mcp`:

```json
{
  "mcpServers": {
    "gated-filesystem": {
      "command": "satgate-mcp",
      "args": ["--config", "/path/to/satgate-mcp.yaml"]
    }
  }
}
```

That's it. The proxy discovers tools from your upstream, presents them to the agent, and intercepts every call.

## Full Config Reference

The config file maps directly to the Go `Config` struct. Every field shown here is a real YAML key that passes validation.

```yaml
# satgate-mcp.yaml — full reference

# --- Client-facing transport ---
server:
  transport: stdio          # "stdio" (default). "sse"/"http" defined but not yet implemented.
  name: satgate-mcp-proxy   # Reported in MCP initialize response
  version: 0.1.0            # Reported in MCP initialize response
  # port: 9100              # For future SSE/HTTP transport

# --- Authentication ---
auth:
  mode: none                # "none" (default), "config", or "header"
  # token: "my-secret"      # Static bearer token (mode: config)
  # rootKey: "hex-key"      # Macaroon root key (mode: header)
  # autoMintRoot: true       # Auto-mint a root macaroon on startup (mode: header)
  # rootToken: "stable-tok"  # Reuse this root token across restarts (with autoMintRoot)

# --- Upstream MCP servers ---
upstreams:
  my-server:
    transport: stdio         # Only "stdio" is currently implemented for upstream validation
    command: ["node", "server.js"]
    env:                     # Additional env vars for the subprocess
      API_KEY: "${MY_API_KEY}"
    timeout: 30s             # Per-request timeout (default: 30s)
    displayName: "My Server" # Name shown to agents when two upstreams share a tool name (default: the key)

  # Example: second upstream for routing
  # db-server:
  #   transport: stdio
  #   command: ["python", "-m", "db_mcp_server"]

defaultUpstream: my-server   # Used when no routing rule matches (auto-detected if only one upstream)

# --- Multi-upstream routing (optional) ---
routing:
  - tools: ["db_*", "sql_*"]        # Tool name patterns (trailing * wildcards)
    upstream: db-server
  - tools: ["fs_read", "fs_write"]
    upstream: my-server

# --- Budget enforcement ---
budget:
  backend: memory            # "memory" (default, in-process) or "redis" (enterprise)
  limit: 1000                # Total budget in credits
  failMode: closed           # "closed" (deny on backend failure, default) or "open" (allow + log)
  # redisUrl: redis://localhost:6379  # Enterprise only

# --- Per-tool costs ---
tools:
  defaultCost: 1             # Cost when no specific pattern matches
  costs:
    "db_query": 10           # Exact tool name
    "db_*": 5                # Wildcard prefix — matches db_insert, db_delete, etc.
    "fs_read": 1
    "fs_write": 3
    "*": 2                   # Catch-all (overrides defaultCost)

# --- Enforcement mode ---
enforcement:
  mode: shadow               # "hard" (deny on exhaustion, default)
                              # "soft" (warn but allow)
                              # "shadow" (observe only, log what would happen)

# --- Logging ---
logging:
  level: info                # "debug", "info" (default), "warn", "error"
  json: false                # JSON output format (default: human-readable to stderr)

# --- SSRF protection ---
# allowPrivateUpstreams: true  # Bypass SSRF protection (only for local dev)
```

### Environment Variable Expansion

Config values support `${ENV_VAR}` expansion. The proxy calls `os.ExpandEnv` on the entire YAML before parsing:

```yaml
auth:
  rootKey: "${SATGATE_ROOT_KEY}"

upstreams:
  my-server:
    transport: stdio
    command: ["node", "server.js"]
    env:
      DATABASE_URL: "${DATABASE_URL}"
```

## Features

### Multi-Upstream Routing

Route different tools to different MCP servers based on name patterns:

```yaml
upstreams:
  code-tools:
    transport: stdio
    command: ["npx", "-y", "@anthropic/mcp-code-tools"]
  db-tools:
    transport: stdio
    command: ["python", "-m", "db_server"]

defaultUpstream: code-tools

routing:
  - tools: ["sql_*", "db_*"]
    upstream: db-tools
  - tools: ["git_*", "file_*"]
    upstream: code-tools
```

Routing rules are evaluated in order. The first matching pattern wins.

Tool resolution also auto-discovers: during startup, the proxy calls `tools/list` on each upstream and tracks which tools came from where. A call reaches a tool only by a name the proxy lists in `tools/list`; a name that is not listed (another spelling, a pattern that matches nothing listed, a tool the upstream never offered) is refused, and `defaultUpstream` does not make it reachable.

### Tool Names When Two Upstreams Share One

Two upstreams often offer a tool with the same name (for example `place_crypto_order` at two brokers, with different arguments). Agents need a distinct name for each, and the name must never change, because tokens carry limits and allow-lists by name.

- Every tool in `tools/list` has a unique name, compared without regard to letter case. `tools/list` and `tools/call` use the same mapping.
- Every page of an upstream's `tools/list` is read (up to 50 pages; a repeated cursor stops the walk) before any name is chosen. If an upstream's list is cut short, its last complete list stays in use. An upstream that has never given a complete list gets no new names, and a call to one of its unlisted tools is answered "this service's tools aren't available yet, try again in a minute". A name that is not listed never reaches an upstream.
- The upstream that comes first keeps the plain name. Another upstream's tool with the same name is listed as `<service>_<tool>`, where `<service>` is a short lowercase form of the upstream's `displayName` (or its key if there is none): `a-z`, `0-9` and `_`, with a number added if two upstreams share one. Calls to `<service>_<tool>` reach that upstream under the tool's own name. With two or more upstreams, every description starts with the service name (`Robinhood: ...`); with one upstream, `tools/list` is exactly what the upstream sent.
- A new plain name that starts with another upstream's slug and an underscore (`robinhood_place_crypto_order` from a service that is not Robinhood) is treated as a clash and listed under its own service's prefix, so one service cannot pass as another. A name already issued is never renamed.
- Which upstream comes first: an explicit `routing` rule that names the upstream for that tool decides; otherwise upstreams are sorted by key.
- **Degraded start.** With static configuration there is no database to remember who answered first. While an upstream that comes earlier in that order has not answered in this process, a later upstream's tools are listed only under their prefixed names, never plain, and one upstream being down never hides another's tools. When the earlier upstream answers it takes its plain names. Within a process a name never changes. After a restart with every upstream up, the same configuration gives the same names. A degraded start can therefore give a later upstream prefixed names that become plain after a healthy restart; it never moves a name to a different upstream. A gateway with a database (`ToolNameStore`, `SetClaimOrderSource`) claims names in connection order and waits for an earlier service that has not answered; see the enterprise documentation.
- Names are fixed for the life of the process. A live reload that adds an upstream adds names and never renames or reuses one. A name stays reserved for the upstream it was given to, even if that upstream is removed; the same address connected again gets it back, and a different upstream never inherits it. (A self-hosted proxy has no database, so names live in memory; a gateway with a database keeps them for good.)
- A tool whose own name an agent app would refuse (a dot, a space, more than 64 characters; the rule is `^[a-zA-Z0-9_-]{1,64}$`) is listed under a valid name made from it, shortened with a short hash when needed.
- Names that start with `satgate_` (any letter case) are never listed from an upstream, whether the upstream offers one as it is or a sanitized or prefixed form of one would become it (`satgate.budget_check` is not listed as `satgate_budget_check`). A service whose slug would start with `satgate` is given another.
- A tool whose name or description holds one of its own service's credentials (a URL query value, a header value, an OAuth token) is not listed, not stored and not shown. The log says the service's display name and how many tools were left out.
- A token with a spending limit or approval rule cannot call a renamed tool unless one of its rules, or its allow-list, names that exact tool; the agent is told "This tool isn't covered by your spending limits. Ask the owner to add it." Tokens with no such rules are not affected. A rule written for `place_crypto_order` never covers `alpaca_place_crypto_order`.
- An upstream connected twice with the same address offers its tools once, unless the host gives each its own `UpstreamConfig.NameKey` (two logins to one broker); then each is a separate service with its own names.

Code that embeds the proxy can supply the place names are kept with `UpstreamManager.SetToolNameStore(ToolNameStore)`, the order with `SetServiceOrder`, and a durable connection order with `SetClaimOrderSource`.

### Per-Tool Cost Attribution

Assign credit costs to tools using exact names or wildcard patterns:

```yaml
tools:
  defaultCost: 1
  costs:
    "expensive_search": 50
    "db_*": 10
    "read_*": 1
    "*": 2
```

**Matching order:** exact match → longest wildcard prefix → catch-all `*` → `defaultCost`.

### Enforcement Modes: Shadow → Soft → Hard

Progress from observation to enforcement without config overhaul:

| Mode | Behavior |
|------|----------|
| `shadow` | Log costs, never block. Use this to understand your spend before enforcing. |
| `soft` | Log warnings when budget is exhausted, but allow calls through. |
| `hard` | Deny calls when budget is exhausted. Returns JSON-RPC error code `-32000`. |

**Recommended rollout:**
1. Deploy with `mode: shadow` — see what agents are doing
2. Set `limit` based on observed spend
3. Switch to `mode: soft` — watch for warnings
4. Switch to `mode: hard` — enforce

### Authentication

Three auth modes:

**`none`** (default) — No authentication. All calls use a default budget identity. Fine for local development.

**`config`** — Static bearer token in the config file. Agents pass it in `params._meta.token` on `tools/call` requests.

```yaml
auth:
  mode: config
  token: "my-secret-token"
```

**`header`** — Macaroon-based auth. Agents pass a macaroon token in `params._meta.token`. Supports delegation (sub-agent tokens with carved budgets).

```yaml
auth:
  mode: header
  rootKey: "64-char-hex-key"
  autoMintRoot: true        # Print ROOT_TOKEN to stderr on startup
```

With `autoMintRoot: true`, the proxy mints a root macaroon on startup and prints it to stderr:

```
ROOT_TOKEN=eyJpZCI6Ii...
TOKEN_ID=a1b2c3d4e5f6
```

Use `rootToken` to keep the token stable across restarts (so Redis budgets survive):

```yaml
auth:
  mode: header
  rootKey: "64-char-hex-key"
  autoMintRoot: true
  rootToken: "eyJpZCI6Ii..."   # Reuse this token instead of minting new ones
```

### Token Delegation

With macaroon auth (`mode: header`), agents can create scoped sub-tokens via the `satgate/delegate` MCP method:

```json
{
  "jsonrpc": "2.0",
  "id": 1,
  "method": "satgate/delegate",
  "params": {
    "_meta": { "token": "parent-macaroon" },
    "budget": 100,
    "scope": "db_*",
    "label": "research-agent",
    "expiresIn": 3600
  }
}
```

This carves 100 credits from the parent's budget and creates a child token that can only call `db_*` tools. Delegation depth and per-delegation budget caps are enforced via macaroon caveats.

Agents can also check their remaining budget:

```json
{
  "jsonrpc": "2.0",
  "id": 2,
  "method": "satgate/budget",
  "params": {
    "_meta": { "token": "my-macaroon" }
  }
}
```

### Budget Fail Modes

Control what happens when the budget backend (Redis) is unreachable:

```yaml
budget:
  failMode: closed    # Deny all calls (safe default)
  # failMode: open    # Allow calls + log warnings (availability over safety)
```

### SSRF Protection

By default, the proxy blocks upstream connections to private/internal IPs (loopback, RFC 1918, link-local). This prevents SSRF via DNS rebinding.

For local development where your MCP server runs on localhost:

```yaml
allowPrivateUpstreams: true
```

> **Warning:** Only enable this for local development. In production, upstream MCP servers should be on public IPs or behind a service mesh.

## Transport Support

### Client-Facing (server.transport)

| Transport | Status |
|-----------|--------|
| `stdio` | ✅ Fully implemented. Default. |
| `sse` / `http` | ⚠️ Defined in code but upstream validation rejects `http`/`sse`. Use `stdio`. |

### Upstream Connections (upstreams.*.transport)

| Transport | Status |
|-----------|--------|
| `stdio` | ✅ Fully implemented. Spawns the upstream as a subprocess. |
| `sse` | ❌ Validation error: "not yet implemented" |
| `http` | ❌ Validation error: "not yet implemented" |

> The codebase has SSE and Streamable HTTP transport implementations (`transport_sse.go`, `transport_streamable.go`) but config validation currently rejects them. Use `stdio` for now.

## Upstream Process Management

For `stdio` upstreams, the proxy manages the subprocess lifecycle:

- **Auto-spawn:** Starts the process on proxy startup
- **Auto-respawn:** If the upstream process dies, retries up to 5 times with exponential backoff (2s, 4s, 6s, 8s, 10s)
- **Tool re-discovery:** After respawn, calls `initialize` and `tools/list` to rebuild the tool map
- **Env passthrough:** The subprocess inherits `os.Environ()` plus any `env` overrides in the upstream config

## Example Configs

### Minimal: Single Server, Observe Only

```yaml
server:
  transport: stdio

upstreams:
  tools:
    transport: stdio
    command: ["npx", "-y", "@modelcontextprotocol/server-filesystem", "/home/user"]

enforcement:
  mode: shadow
```

### Multi-Server with Budget Enforcement

```yaml
server:
  transport: stdio
  name: production-proxy

auth:
  mode: header
  rootKey: "${SATGATE_ROOT_KEY}"
  autoMintRoot: true

upstreams:
  code:
    transport: stdio
    command: ["npx", "-y", "@anthropic/mcp-code-tools"]
  database:
    transport: stdio
    command: ["python", "-m", "db_mcp_server"]
    env:
      DATABASE_URL: "${DATABASE_URL}"

routing:
  - tools: ["sql_*", "db_*"]
    upstream: database

budget:
  backend: memory
  limit: 5000
  failMode: closed

tools:
  defaultCost: 1
  costs:
    "sql_query": 10
    "sql_write": 25
    "db_*": 5
    "git_commit": 3

enforcement:
  mode: hard

logging:
  level: info
  json: false
```

### Development: No Auth, All Logging

```yaml
server:
  transport: stdio

upstreams:
  dev:
    transport: stdio
    command: ["node", "my-mcp-server.js"]

allowPrivateUpstreams: true

budget:
  backend: memory
  limit: 10000

tools:
  defaultCost: 1

enforcement:
  mode: shadow

logging:
  level: debug
```

## SatGate MCP Extensions

The proxy adds two custom JSON-RPC methods (namespaced under `satgate/` to avoid MCP conflicts):

| Method | Auth Required | Description |
|--------|--------------|-------------|
| `satgate/delegate` | Yes (macaroon) | Create a scoped child token with carved budget |
| `satgate/budget` | Yes | Check remaining budget for the authenticated token |

Standard MCP methods (`tools/list`, `tools/call`, `initialize`, `ping`, etc.) are fully supported and proxied transparently.

## Further Reading

- [MCP Specification](https://modelcontextprotocol.io/specification)
- [SatGate Quick Start](https://github.com/SatGate-io/satgate#quick-start)
- [Capability schema](../reference/capability-schema.md)
