# Task: Implement HTTP/SSE and Streamable HTTP Upstream Transport

## Context
SatGate's MCP proxy aggregates upstream MCP servers and adds budget enforcement, cost attribution, and access control. Currently only stdio transport is implemented for upstreams. We need HTTP/SSE and Streamable HTTP transports so the proxy can connect to remote MCP servers over the network — this is required for any real enterprise deployment.

## Architecture

The Transport interface is in `pkg/mcpserver/transport.go`:
```go
type Transport interface {
    ReadMessage(ctx context.Context) (json.RawMessage, error)
    WriteMessage(ctx context.Context, msg json.RawMessage) error
    Close() error
}
```

The UpstreamManager in `pkg/mcpserver/upstream.go` calls `connect()` which currently only supports stdio. The `UpstreamConfig` struct already has `URL` and `Transport` fields.

## Requirements

### 1. SSE Transport (`transport_sse.go`)
For MCP servers using the older SSE protocol (pre-2025-03-26 spec):
- **Client→Server**: HTTP POST to the upstream's message endpoint
- **Server→Client**: SSE stream (text/event-stream) for responses and notifications
- The SSE endpoint URL is obtained during initialization (server sends an `endpoint` event)
- Must handle SSE reconnection with exponential backoff
- Must handle `event: message` (JSON-RPC response) and `event: endpoint` (session URL)
- Content-Type for POST: `application/json`
- Must pass session ID from SSE connection to POST requests

### 2. Streamable HTTP Transport (`transport_streamable.go`)
For MCP servers using the new Streamable HTTP protocol (2025-03-26 spec):
- **Single endpoint**: All communication via HTTP POST to one URL
- **Request**: POST with `Content-Type: application/json`, body is JSON-RPC message
- **Response**: Can be either:
  - `Content-Type: application/json` — single JSON-RPC response
  - `Content-Type: text/event-stream` — SSE stream of multiple responses (for long-running tools)
- Must include `Mcp-Session-Id` header after initialization (server provides it in init response)
- GET request to same endpoint opens an SSE stream for server-initiated notifications
- DELETE request to same endpoint terminates the session
- Must handle both response types transparently

### 3. Update `upstream.go`
- Implement `connectHTTP(name string, cfg UpstreamConfig) (*UpstreamClient, error)` for SSE
- Implement `connectStreamable(name string, cfg UpstreamConfig) (*UpstreamClient, error)` for Streamable HTTP
- Update `connect()` switch to route `"sse"` → `connectHTTP`, `"http"` or `"streamable"` → `connectStreamable`
- Add reconnection logic in `readLoop` for HTTP transports (similar to stdio respawn)

### 4. Config
- `transport: "sse"` → SSE transport (legacy)
- `transport: "http"` or `transport: "streamable"` → Streamable HTTP transport (preferred)
- `url` field is required for both
- Optional `headers` map for auth headers (e.g., `Authorization: Bearer <token>`)
- Optional `tls_skip_verify` for self-signed certs (development only)

Add to UpstreamConfig in `config.go`:
```go
Headers       map[string]string `yaml:"headers,omitempty"`
TLSSkipVerify bool              `yaml:"tlsSkipVerify,omitempty"`
```

### 5. Tests
- Unit tests for both transports using httptest.Server
- Test SSE reconnection behavior
- Test Streamable HTTP with both JSON and SSE response modes
- Test session management (Mcp-Session-Id lifecycle)
- Test timeout handling
- Test concurrent tool calls through HTTP transport
- Integration test: start mock MCP server over HTTP, connect, discover tools, call a tool

## Quality Bar
- No external dependencies beyond stdlib + zerolog (already in go.mod)
- Connection pooling via http.Client (reuse connections)
- Graceful shutdown: Close() must drain in-flight requests
- All errors must be wrapped with context (upstream name, URL, method)
- Debug logging for all HTTP requests/responses (zerolog)
- Must handle upstream returning 4xx/5xx gracefully (not crash)
- Race-free: safe for concurrent tool calls from multiple agents

## Files to Create/Modify
- CREATE: `pkg/mcpserver/transport_sse.go`
- CREATE: `pkg/mcpserver/transport_sse_test.go`
- CREATE: `pkg/mcpserver/transport_streamable.go`
- CREATE: `pkg/mcpserver/transport_streamable_test.go`
- MODIFY: `pkg/mcpserver/upstream.go` (add connectHTTP, connectStreamable)
- MODIFY: `pkg/mcpserver/config.go` (add Headers, TLSSkipVerify to UpstreamConfig)
- MODIFY: `pkg/mcpserver/upstream.go` connect() switch

## DO NOT
- Do not modify transport.go's Transport interface
- Do not change the stdio transport
- Do not add external dependencies
- Do not break existing tests (run `go test ./pkg/mcpserver/...`)
