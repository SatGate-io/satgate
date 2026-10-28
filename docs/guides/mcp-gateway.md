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

# Over stdio the caller is the local process that starts the proxy.
# For SSE / Streamable HTTP pick "header" or "config" (see Authentication).
auth:
  mode: none

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
  mode: shadow    # observe only — no blocking (without this line a budget is enforced)
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
  transport: stdio          # "stdio" (default), or "sse" / "http" (SSE and Streamable HTTP on one listener)
  name: satgate-mcp-proxy   # Reported in MCP initialize response
  version: 0.1.0            # Reported in MCP initialize response
  # host: 127.0.0.1         # Listen address for sse/http (default 127.0.0.1; 0.0.0.0 or :: for every interface). No port here.
  # port: 9100              # Listen port for sse/http (default 9100)
  # allowQueryToken: false   # Accept the token from a ?token= URL parameter (SSE / Streamable HTTP). Off by default: any ?token= gets 401.

# --- Authentication ---
auth:
  mode: none                # "none", "config" or "header". Required for sse/http; stdio without it means "none".
  # token: "my-secret"      # Static bearer token (mode: config). Required and non-empty for mode: config; a value of only whitespace is refused.
  # rootKey: "hex-key"      # Macaroon root key (mode: header)
  # autoMintRoot: true       # Auto-mint a root macaroon on startup (mode: header)
  # rootToken: "stable-tok"  # Reuse this root token across restarts (with autoMintRoot)

# --- Upstream MCP servers ---
upstreams:
  my-server:
    transport: stdio         # Only "stdio" is currently implemented for upstream validation
    command: ["node", "server.js"]
    env:                     # Env vars set for the subprocess (see "Upstream Environment")
      API_KEY: "${MY_API_KEY}"
    # envPassthrough: [HTTPS_PROXY]  # Names of proxy env vars the subprocess also receives
    # forwardCallerToken: false      # Pass the credential the proxy verified for the request in params._meta.token. Off by default.
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
  mode: shadow               # "hard" / "control" (deny on exhaustion)
                              # "soft" (warn but allow)
                              # "shadow" / "observe" (observe only, log what would happen)
                              # Not set: "control" when budget.limit > 0, else "observe"

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
- **Waiting for an earlier upstream (static configuration).** There is no database to remember who answered first, so the gateway decides from the configuration alone: the order is the routing rules first, then the upstream keys, sorted. Until every upstream earlier in that order has a complete tool list in this process, the tools of later upstreams are **not listed**, and a call to one gets "this service's tools aren't available yet, try again in a minute". No temporary name is ever offered, so the same configuration and the same tool lists always give the same names, whatever was up at start, and no name is offered that a later start takes back. **The cost:** one earlier upstream that is down hides the tools of every later upstream until it answers. The gateway logs one line at start for each such wait: turn the earlier upstream off, or change the order. A limit written for a name that is not listed yet matches nothing, and a call to that name is refused before it reaches any upstream. Known limit: without a database, if an upstream's tool list changes between restarts so that an earlier upstream now offers a tool a later one already had, the earlier upstream takes the plain name at the next start. A gateway with a database (`ToolNameStore`, `SetClaimOrderSource`) claims names in connection order, keeps them across restarts, and waits for an earlier service that has not answered; see the enterprise documentation.
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

**Default.** When `enforcement.mode` is not set, it is `control` (calls are denied when the budget is exhausted) if `budget.limit` is greater than 0, and `observe` otherwise. Write `enforcement.mode: observe` (or `shadow`) to watch a budget without enforcing it.

### Authentication

Three auth modes:

| Mode | Token | SSE / Streamable HTTP without a token |
|------|-------|----------------------------------------|
| `none` | none | answered under the default identity |
| `config` | one static bearer token, `auth.token` | refused |
| `header` | a macaroon signed with `auth.rootKey` | refused |

**`none`** — No authentication. All calls use a default budget identity. For stdio, or a listener only you can reach. Over SSE / Streamable HTTP it must be written out (`auth.mode: none`), and the proxy logs a warning at startup.

**`config`** — One static bearer token in the config file. Agents send it as `Authorization: Bearer <token>` (SSE and Streamable HTTP) or in `params._meta.token`. The proxy hashes the sent and configured values with SHA-256 and compares the fixed-length digests in constant time, so the comparison does not depend on how much of the value matches (hashing time still depends on the length sent).

```yaml
auth:
  mode: config
  token: "${SATGATE_MCP_TOKEN}"
```

`auth.token` must be non-empty. A value that is empty or only whitespace after environment expansion (an unset variable, for example) is a configuration error: `LoadConfig` and `New` refuse it with `auth.token is required for auth.mode "config" and must not be empty or only whitespace ...`, and an embedded handler or listener whose `Config` has such a value refuses every request with that message (see [Listener and startup checks](#listener-and-startup-checks)). A presented token that is empty after trimming whitespace, in the `Authorization` header or in `params._meta.token`, counts as no token.

**`header`** — Macaroon-based auth. Agents send a macaroon the same way. Supports delegation (sub-agent tokens with carved budgets).

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

#### What needs a token

With `mode: config` or `mode: header`, a request over SSE (`POST /message`) or Streamable HTTP must carry a token that verifies before the proxy reveals the tool list or reaches an upstream:

- `tools/call`, `satgate/delegate`, `satgate/budget`, `budget/check`, `tools/list`, and every method the proxy passes on to an upstream (`resources/list`, `prompts/get` and the like).
- A request with no token, an empty one, or one that is only whitespace gets the JSON-RPC error `-32001` with the message `authentication required: no token provided`; a token that does not verify gets `-32001` `authentication failed: ...`; a revoked one `-32001` `token revoked`. The answer is the same in both modes.
- Streamable HTTP refuses `initialize` with `401` (`authentication required` / `authentication failed`) and creates no session.
- `GET /sse` without a token still opens the stream (some clients send the token only on `POST /message`), as in header mode; with the authorization challenge on (`WithOAuthChallenge`) it gets `401`.
- `initialize`, `ping` and notifications need no token.

Over stdio the caller is the local process that started the proxy: in `config` mode a request without a token runs under the configured token's identity, as before, and `tools/list` and passed-on methods need no token in either mode. In `header` mode `tools/call` over stdio still needs a token, as before.

#### Listener and startup checks

- `server.transport: sse` or `http` with no `auth.mode` refuses to start: `auth.mode is required to serve over SSE / Streamable HTTP ...`. Set `auth.mode` to `header` or `config`, or write `auth.mode: none` to serve without authentication (the proxy then logs a warning at startup).
- `auth.mode: config` with no `auth.token`, an empty one, or one of only whitespace does not load, on any transport: `auth.token is required for auth.mode "config" ...`. An embedded handler or listener whose `Config` has such a value (set in code) refuses in the same way as for a missing `auth.mode`: `503` with that message on every route, and `ListenAndServe` returns it. `header` and `none` modes do not read `auth.token`.
- The same rule applies to the handler of an embedded server (`NewSSEServer(...).Handler()`, or the `SSEServer` itself as an `http.Handler`). If the configuration does not set `auth.mode` (`Auth.Mode` empty, or a file loaded without `auth.mode`, including a stdio-only file), every request to that handler gets `503` with the same `auth.mode is required ...` message. This covers every route: `/mcp`, `/sse`, `/message`, `/health` and routes added with `Handle` / `HandleFunc`. The request is refused before any MCP method runs or an upstream is contacted. The error is logged once, when `NewSSEServer` is called. With an explicit `none`, the warning is also logged once, at that point.
- The listener binds `server.host`, default `127.0.0.1` (this machine only), on `server.port`, default `9100`. Set `host: 0.0.0.0` (or `::`) to accept connections from other machines, for example in a container. `host` takes no port.
- A stdio-only configuration starts as before, with or without an `auth` block.

### Tokens and Upstreams

Agents authenticate to the proxy with the `Authorization` header, `Bearer` scheme (SSE and Streamable HTTP), or `params._meta.token` (stdio). The token is for the proxy:

- **Not forwarded by default.** The proxy removes `_meta.token` from every request before it reaches an upstream, including a `_meta.token` the agent put there itself. Other `_meta` members (such as `progressToken`) are passed on.
- **Opt-in per upstream.** An upstream that must check the caller's SatGate token itself can be given it with `forwardCallerToken: true`. That upstream then holds the caller's token and can act as the caller with it until it expires or is revoked; enable it only for an upstream you run and trust.
- **Only the verified credential.** What an opted-in upstream receives in `_meta.token` is the credential the proxy verified for that request, never a value from the request's params that it did not verify: for `tools/call` (and every method, with sign-every-action on) the token authentication accepted; for a method the proxy passes on outside the `tools/call` path (`resources/list`, `prompts/get` and the like), the connection's credential (the `Authorization` header, or the URL token with `allowQueryToken`), which the proxy verifies and checks for revocation first. A request with no credential that verifies (none sent, one that fails verification or is revoked, or, with sign-every-action off, a stdio request other than `tools/call`) forwards none.

```yaml
upstreams:
  my-verifying-server:
    transport: stdio
    command: ["./verifier-mcp"]
    forwardCallerToken: true
```

**Token in the URL.** By default `GET /sse`, `POST /message` and the Streamable HTTP endpoint (every HTTP method) refuse with 401 any request whose URL has a `token` query parameter, whatever its value: also empty, repeated, or next to an `Authorization` header. A URL ends up in access logs, browser history and `Referer` headers. The value is not repeated in the response or the log. Send the token in the `Authorization` header. For a client that cannot set a header, `server.allowQueryToken: true` restores the URL fallback (the header first, else the first `token` value):

```yaml
server:
  transport: sse
  allowQueryToken: true   # only for clients that cannot send Authorization
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

### Revoking Tokens

`satgate-mcp` honours the bans written through a SatGate gateway's
`POST /api/governance/ban` (see [Revocation](../api/overview.md#revocation)).
A ban covers the banned macaroon and every token delegated from it, at any
depth, before or after the ban, through `satgate/delegate`, the gateway, or a
holder adding caveats offline. It never covers the token's parent or siblings.

Point the proxy at the Redis the gateways use (`redis.enabled: true` on the
gateway):

```yaml
revocation:
  redis:
    addr: "redis.internal:6379"
    password: "${SATGATE_REDIS_PASSWORD}"
    db: 0
```

- **With `revocation.redis.addr`**, a ban written by any gateway on that Redis
  is honoured here within **2 seconds**, and an unban likewise. If the address
  cannot be reached at startup, `satgate-mcp` does not start.
- **Without it**, bans are kept in the proxy's own memory and apply to that
  process only. `satgate-mcp` has no ban endpoint, so a ban made on a gateway
  does not reach it; only a program that embeds the proxy and bans through its
  own governance service sees its bans applied.

The check runs wherever the proxy accepts a token, before anything is admitted,
spent, forwarded or minted: SSE connect and every message, Streamable HTTP
`initialize`, every request and every resumed session, stdio, `tools/list` and
the methods passed on to an upstream, the credential passed to an upstream
with `forwardCallerToken` (never passed on for a banned token), and
`satgate/delegate` (checked again before the helper is minted).
A banned token gets `403` with `code: TOKEN_REVOKED` where the refusal is an
HTTP answer (SSE connect, Streamable `initialize` and resume; `401` with the
sign-in challenge when the OAuth challenge is on), and otherwise the JSON-RPC
error `-32001` "token revoked". It applies to macaroons (`mode: header`); a
static token (`mode: config`) is not a ban subject.

If Redis cannot be read within **2 seconds** for a token whose answer is not
cached, the request is refused, never served: HTTP `503` with `code:
REVOCATION_UNAVAILABLE` (SSE connect, Streamable HTTP requests, `initialize` and
resume), or the JSON-RPC error `-32603` with `data.error:
"revocation_unavailable"` (SSE messages, stdio).

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
| `sse` / `http` | ✅ SSE (`GET /sse`, `POST /message`) and Streamable HTTP (`/mcp`) on one listener, `server.host` (default `127.0.0.1`) : `server.port` (default `9100`). Requires `auth.mode`. |

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
- **Environment:** The subprocess does not inherit the proxy's environment. See [Upstream Environment](#upstream-environment).

### Upstream Environment

A `stdio` upstream starts with a small fixed set of variables taken from the proxy's environment (each only if the proxy has it):

`PATH`, `HOME`, `USER`, `LOGNAME`, `SHELL`, `LANG`, `LANGUAGE`, `LC_ALL`, `LC_CTYPE`, `LC_MESSAGES`, `TMPDIR`, `TMP`, `TEMP`, `TZ`, `TERM`, `SSL_CERT_FILE`, `SSL_CERT_DIR`, and on Windows `SystemRoot`, `SystemDrive`, `ComSpec`, `PATHEXT`, `windir`, `USERPROFILE`, `APPDATA`, `LOCALAPPDATA`, `ProgramData`.

Two per-upstream keys add to it:

- `env` — variables set for the subprocess, with the value given (after `${VAR}` expansion of the config file). A value here wins over the base set and over `envPassthrough`. These entries come after the inherited ones, so they also win where variable names are compared without regard to case (Windows).
- `envPassthrough` — names of variables of the proxy's environment the subprocess also receives, with the proxy's value. A name the proxy does not have is skipped.

Nothing else reaches the subprocess: not the proxy's root key, database or cloud credentials, or any other variable. If an upstream read a variable from the inherited environment before, name it in `envPassthrough` or set it in `env`:

```yaml
upstreams:
  github:
    transport: stdio
    command: ["npx", "@modelcontextprotocol/server-github"]
    env:
      GITHUB_PERSONAL_ACCESS_TOKEN: "${GITHUB_TOKEN}"
    envPassthrough: [HTTPS_PROXY, NO_PROXY, NODE_EXTRA_CA_CERTS]
```

## Example Configs

### Minimal: Single Server, Observe Only

```yaml
server:
  transport: stdio

auth:
  mode: none

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

auth:
  mode: none

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

Standard MCP methods (`tools/list`, `tools/call`, `initialize`, `ping`, etc.) are fully supported and proxied transparently. With `auth.mode` `config` or `header`, every method except `initialize`, `ping` and notifications needs a token over SSE and Streamable HTTP (see [What needs a token](#what-needs-a-token)).

## Upgrading

This release changes three defaults of the MCP proxy:

1. **Static-token mode refuses requests without a token.** With `auth.mode: config`, a request over SSE or Streamable HTTP with no token, or an empty one, is refused like in `header` mode; before, it ran under the configured token's identity. Streamable HTTP `initialize` without the token now gets `401`. Clients must send `Authorization: Bearer <token>`. stdio is unchanged.
2. **`tools/list` and passed-on methods need a token in `config` and `header` modes** over SSE and Streamable HTTP. In `none` mode, and over stdio, they stay open.
3. **Safer startup defaults for `satgate-mcp`:**
   - `server.transport: sse` / `http` without `auth.mode` no longer starts. Add an `auth` block; to keep serving without authentication, write `auth.mode: none`.
   - The listener binds `127.0.0.1` instead of every interface. To accept other machines (containers, a LAN), set `server.host: 0.0.0.0`.
   - With `budget.limit` greater than 0 and no `enforcement.mode`, enforcement is `control` instead of `observe`, so an exhausted budget denies calls. To keep the old behaviour, write `enforcement.mode: observe`.

A stdio-only configuration starts as before, with or without an `auth` block; only the enforcement default applies to it. Code that builds a `Config` itself (without `LoadConfig`) keeps the enforcement mode it sets. `NewSSEServer` listens on the address it is given, and `Config.ListenAddr` returns the default one.

**A config-mode token must be non-empty.** `auth.mode: config` with an `auth.token` that is empty or only whitespace (including an environment variable that expands to nothing) no longer loads, on any transport; before, a value of only whitespace was accepted. Set `auth.token` to a non-empty value. A token presented as only whitespace counts as no token and gets `authentication required: no token provided` (before: `authentication failed: invalid token`).

**Embedded handlers follow the same rule as the listener.** Code that serves `NewSSEServer(...).Handler()` (or the `SSEServer` itself) from its own `http.Server` must set `Auth.Mode`, the same as code that calls `SSEServer.ListenAndServe`. Without it, every request to the handler, on every route including `/health`, gets `503` with the `auth.mode is required ...` message. A file loaded with `LoadConfig` that has no `auth.mode` counts as unset even for stdio, unless the code sets `Auth.Mode` to `header` or `config` before `New`. To keep serving without authentication, set `auth.mode: none` in the file, or `Auth.Mode: "none"` in a `Config` built in code. The warning is then logged once when the server is created.

## Further Reading

- [MCP Specification](https://modelcontextprotocol.io/specification)
- [SatGate Quick Start](https://github.com/SatGate-io/satgate#quick-start)
- [Capability schema](../reference/capability-schema.md)
