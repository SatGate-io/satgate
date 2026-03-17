# SDKs & Client Libraries

SatGate provides official SDKs for Python and Node.js/TypeScript.

## Available SDKs

| SDK | Package | Status |
|-----|---------|--------|
| [Python](python.md) | `pip install satgate` | ✅ v2.0.0 |
| [Node.js/TypeScript](nodejs.md) | `npm install @satgate/sdk` | ✅ |
| [Go](go.md) | Import `pkg/` directly | ✅ Native |

## Agent SDKs

Built on top of the base SDKs, these provide automatic token management and L402 payment handling for AI agents:

| SDK | Included In | Description |
|-----|------------|-------------|
| [Python Agent Client](agent-python.md) | `satgate` package | `SatGateAgentClient` — auto-mint, cache, L402 |
| [Node.js Agent Client](agent-nodejs.md) | `@satgate/sdk` | `SatGateAgentClient` — auto-mint, cache, L402 |

## Integrations

| Integration | Included In | Description |
|-------------|------------|-------------|
| [LangChain](../guides/langchain-integration.md) | `satgate` Python package | `SatGateTool`, `SatGateToolkit` for LangChain agents |

## Architecture

Each SDK provides two clients:

- **`SatGateClient`** — Admin operations (mint tokens, ban, governance). Uses `X-Admin-Token` header.
- **`SatGateAgentClient`** — Agent operations (make API calls with automatic token & payment handling). Uses Bearer tokens.

Both clients talk to the same gateway endpoints:
- `/api/capability/mint` — Create tokens
- `/api/capability/validate` — Validate tokens
- `/api/capability/delegate` — Delegate tokens
- `/api/governance/ban` — Ban tokens
- `/api/governance/graph` — Token lineage
