# Architecture

## Overview

SatGate is a single Go binary that acts as a reverse proxy with economic governance.

```
                    ┌─────────────────────────────────────────┐
                    │              SatGate Gateway             │
Requests ──────────▶│                                         │──────────▶ Upstreams
                    │  Route Matching → Policy Enforcement    │
                    │  → Macaroon Verification                │
                    │  → Budget Checking (fiat402)            │
                    │  → Payment Verification (L402)          │
                    │  → MCP Parsing (optional)               │
                    │  → Rate Limiting                        │
                    │  → Proxy to Upstream                    │
                    │                                         │
                    │  Management APIs:                       │
                    │  /api/capability/* (token ops)           │
                    │  /api/governance/* (admin ops)           │
                    └─────────────────────────────────────────┘
```

## Request Flow

1. **Route matching** — find the first route that matches path, method, headers
2. **Policy enforcement** — apply the route's policy:
   - `public` → pass through
   - `deny` → reject
   - `capability` → verify macaroon token
   - `chargeback` → verify → allow → meter
   - `fiat402` → verify → check budget → allow/deny
   - `l402` → verify → check payment proof → allow/deny
3. **Transform** — optional path rewrite, header injection
4. **Proxy** — forward to upstream, return response

## Components

### Gateway (`pkg/proxy/`)
HTTP reverse proxy with route matching, policy enforcement, and upstream management. Includes circuit breaker and health checking.

### Macaroon Engine (`pkg/macaroon/`)
Cryptographic token creation, verification, and delegation. Supports caveats for scope, budget, expiry, and IP binding.

### MCP Parser (`pkg/mcp/`)
JSON-RPC 2.0 parser for Model Context Protocol payloads. Extracts tool names for per-tool cost attribution.

### MCP Proxy (`pkg/mcpserver/`, `cmd/satgate-mcp/`)
Separate binary (`satgate-mcp`) that acts as an MCP-aware proxy. Supports stdio transport with upstream MCP servers.

### Billing (`pkg/billing/`)
Enterprise billing engine supporting chargeback (metering), fiat402 (budget enforcement), and L402 (Lightning payments).

### Lightning (`pkg/lightning/`)
Lightning payment backends: NWC, Alby, Phoenixd, LND, LNbits, mock.

### Config (`pkg/config/`)
YAML configuration loading, validation, and environment variable expansion.

## Ports

- **8080** — Main proxy + API (default, configurable via `server.listen`)
- **9090** — Optional separate admin listener (if `admin.separateListener` is set)

## Dependencies

- **Required:** None (single binary, in-memory state)
- **Optional:** Redis (distributed budgets), PostgreSQL (persistent storage), Lightning node (L402 payments)
