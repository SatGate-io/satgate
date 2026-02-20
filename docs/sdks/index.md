# SDK Documentation

Client libraries for integrating with SatGate Gateway.

## Available SDKs

### Admin/Operator SDKs

For managing the gateway (tokens, governance, configuration):

| Language | Package | Status |
|----------|---------|--------|
| [Go](go.md) | `github.com/satgate-io/satgate-go` | ✅ Stable |
| [Python](python.md) | `satgate` | ✅ Stable |
| [Node.js](nodejs.md) | `satgate-sdk` | ✅ Stable |

### Agent SDKs

For AI agents calling SatGate-protected APIs (auto-402 handling, badge-in, payments):

| Language | Package | Status |
|----------|---------|--------|
| [Python Agent](agent-python.md) | `satgate` (`SatGateAgentClient`) | ✅ Stable |
| [Node.js Agent](agent-nodejs.md) | `@satgate/sdk` (`SatGateAgentClient`) | ✅ Stable |

**Agent SDKs provide:**
- Automatic 402 Payment Required handling (L402 + Fiat402)
- Identity-based token minting via Mint (K8s, AWS, OIDC)
- Token delegation for sub-agents
- LangChain/LangGraph integration

## Quick Comparison

### Authentication

All SDKs support the same authentication methods:

```go
// Go
client := satgate.NewClient("https://api.example.com",
    satgate.WithAdminToken("your-token"))

// Python
client = SatGateClient("https://api.example.com",
    admin_token="your-token")

// Node.js
const client = new SatGateClient("https://api.example.com", {
    adminToken: "your-token"
});
```

### Token Minting

```go
// Go
token, err := client.Tokens.Mint(ctx, &satgate.MintRequest{
    Scope:     "api:read",
    ExpiresIn: 3600,
})

// Python
token = client.tokens.mint(scope="api:read", expires_in=3600)

// Node.js
const token = await client.tokens.mint({
    scope: "api:read",
    expiresIn: 3600
});
```

### Making Protected Requests

```go
// Go
resp, err := client.Request(ctx, "GET", "/api/users",
    satgate.WithToken(token))

// Python
resp = client.request("GET", "/api/users", token=token)

// Node.js
const resp = await client.request("GET", "/api/users", { token });
```

## Common Features

All SDKs provide:

- **Token Management** — Mint, delegate, revoke tokens
- **Governance** — Ban/unban tokens, view lineage
- **Configuration** — Get/update gateway config
- **Statistics** — Query metrics and usage
- **WebSocket Telemetry** — Real-time event streaming

## Error Handling

All SDKs use consistent error types:

| Error | Description |
|-------|-------------|
| `AuthenticationError` | Invalid or missing credentials |
| `AuthorizationError` | Insufficient permissions |
| `NotFoundError` | Resource not found |
| `RateLimitError` | Rate limit exceeded |
| `ValidationError` | Invalid request parameters |
| `ServerError` | Gateway internal error |

## Best Practices

1. **Reuse clients** — Create one client instance and reuse it
2. **Handle rate limits** — Implement exponential backoff
3. **Use timeouts** — Set appropriate request timeouts
4. **Secure tokens** — Never log or expose tokens
5. **Delegate tokens** — Use delegated tokens for downstream services



