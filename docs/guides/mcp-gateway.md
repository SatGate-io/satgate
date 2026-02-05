# Gating MCP Servers with SatGate

Put SatGate in front of any [MCP](https://modelcontextprotocol.io/) (Model Context Protocol) server to get cost attribution, budget enforcement, and access control — with zero code changes.

## What is MCP?

MCP is [Anthropic's open standard](https://modelcontextprotocol.io/specification) for connecting AI agents to tools and data sources. An MCP server exposes "tools" (functions) that agents can discover and call. When served over HTTP (via the Streamable HTTP transport), an MCP server is just another HTTP upstream — which means SatGate can gate it today.

## Why MCP Servers Need Economic Controls

MCP makes it trivially easy for agents to discover and call tools. That's the point — but it creates a new problem:

- **No cost visibility.** An agent calls 10 tools in a chain. What did that cost? Which team pays?
- **No budget enforcement.** A misconfigured agent can burn through expensive tool calls with no guardrails.
- **No access scoping.** Every agent with the MCP server URL can call every tool.

SatGate solves all three by sitting between agents and MCP servers as a reverse proxy.

## Quick Start: SatGate in Front of an MCP Server

### 1. Configure Your Gateway

Add the MCP server as an upstream in your `gateway.yaml`:

```yaml
# gateway.yaml
upstreams:
  - name: mcp-tools
    url: http://localhost:3001    # Your MCP server
    paths:
      - /mcp                      # MCP Streamable HTTP endpoint
    mode: observe                  # Start in observe mode

  - name: internal-api
    url: http://localhost:8080
    paths:
      - /api
    mode: observe
```

### 2. Start SatGate

```bash
satgate --config gateway.yaml
```

### 3. Point Agents at SatGate

Instead of connecting agents directly to the MCP server:

```
# Before: agent → MCP server
MCP_SERVER_URL=http://localhost:3001/mcp

# After: agent → SatGate → MCP server
MCP_SERVER_URL=http://localhost:9090/mcp
```

That's it. SatGate proxies all MCP traffic and starts logging.

## Observe Mode: See What Agents Are Doing

In `observe` mode, SatGate is read-only. It logs every request without blocking anything:

- **Request volume** per MCP endpoint
- **Latency** per upstream call
- **Agent identification** via token or header
- **Cost attribution** to teams and cost centers

Check the dashboard at `cloud.satgate.io` or your self-hosted instance to see traffic flowing.

This is free and takes about 5 minutes to set up. No code changes on the MCP server side.

## Control Mode: Set Budgets and Enforce Limits

When you're ready, switch from `observe` to `control`:

```yaml
upstreams:
  - name: mcp-tools
    url: http://localhost:3001
    paths:
      - /mcp
    mode: control                  # Now enforcing
    budgets:
      default: 100                 # $100/month default per token
    rate_limit:
      requests_per_minute: 60
```

With control mode:

- **Budget enforcement** — agents are stopped when their budget is spent
- **Rate limiting** — prevent runaway tool-call loops
- **Token scoping** — issue macaroon tokens scoped to specific paths (e.g., only `/mcp/tools/search`, not `/mcp/tools/delete`)
- **Time-limited access** — tokens that auto-expire after a session

### Issue Scoped Tokens

```bash
# Create a token that can only access the MCP tools endpoint
# with a $50 budget and 24-hour expiry
satgate token create \
  --scope "/mcp/*" \
  --budget 50 \
  --ttl 24h \
  --label "research-agent-prod"
```

Agents present this token on every request. SatGate verifies it locally (sub-millisecond, no external auth calls) and enforces the budget.

## What Works Today vs. What's Coming

### Works Today (Reverse Proxy)
- ✅ Gate any MCP server exposed over HTTP
- ✅ Request-level logging and cost attribution
- ✅ Budget enforcement per token
- ✅ Rate limiting
- ✅ Token scoping by URL path
- ✅ Full audit trail

### MCP-Aware (SatGate Enterprise)
- ✅ **Per-tool cost attribution** — parses MCP tool calls to attribute costs at the tool level, not just the endpoint level. Configure per-tool cost profiles with wildcard matching.
- ✅ **Tool-level budget enforcement** — set budgets per tool, per agent, per team. Hard stops when budgets are spent.
- 🔜 **Tool-level access control** — scope tokens to specific MCP tools (e.g., "this agent can call `search` but not `delete`")
- 🔜 **Tool marketplace** — publish MCP tools with per-call pricing, monetized via L402 micropayments

## Architecture

```
┌──────────────┐     ┌──────────────┐     ┌──────────────┐
│   AI Agent   │────▶│   SatGate    │────▶│  MCP Server  │
│ (Claude, etc)│     │   Gateway    │     │  (tools)     │
└──────────────┘     └──────┬───────┘     └──────────────┘
                            │
                     ┌──────▼───────┐
                     │  Dashboard   │
                     │  (cost,      │
                     │   usage,     │
                     │   governance)│
                     └──────────────┘
```

SatGate is protocol-agnostic. It works the same way whether your upstream is an MCP server, REST API, GraphQL endpoint, or anything else served over HTTP.

## Further Reading

- [MCP Specification](https://modelcontextprotocol.io/specification)
- [SatGate Quick Start](https://github.com/SatGate-io/satgate#-quick-start)
- [Token Delegation Guide](https://github.com/SatGate-io/satgate#token-delegation)
