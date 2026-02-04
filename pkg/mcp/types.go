// Package mcp provides MCP (Model Context Protocol) request parsing and
// cost attribution for the SatGate gateway proxy.
//
// MCP uses JSON-RPC 2.0 over HTTP. This package detects MCP traffic,
// parses the JSON-RPC envelope, and extracts tool-level metadata so
// the gateway can attribute costs per-tool instead of per-route.
package mcp

import "context"

// JSON-RPC 2.0 request envelope used by MCP.
type JSONRPCRequest struct {
	JSONRPC string      `json:"jsonrpc"`
	ID      interface{} `json:"id,omitempty"` // string | number | null
	Method  string      `json:"method"`
	Params  Params      `json:"params,omitempty"`
}

// Params holds the parameters for an MCP method call.
// For tools/call, Name is the tool name and Arguments contains the tool input.
type Params struct {
	Name      string                 `json:"name,omitempty"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`

	// For resources/read
	URI string `json:"uri,omitempty"`
}

// RequestInfo is the parsed metadata extracted from an MCP JSON-RPC request.
// It is stored in the request context by the middleware.
type RequestInfo struct {
	// Method is the JSON-RPC method (e.g. "tools/call", "tools/list").
	Method string

	// ToolName is the tool being invoked (only set for tools/call).
	ToolName string

	// ResourceURI is the resource URI (only set for resources/read).
	ResourceURI string

	// IsBatch is true when the request body is a JSON-RPC batch (array).
	IsBatch bool

	// BatchSize is the number of requests in a batch (0 for single requests).
	BatchSize int

	// CostCredits is the resolved cost for this request after tool-cost lookup.
	CostCredits int
}

// Known MCP methods.
const (
	MethodToolsCall    = "tools/call"
	MethodToolsList    = "tools/list"
	MethodResourceRead = "resources/read"
	MethodResourceList = "resources/list"
	MethodPromptsGet   = "prompts/get"
	MethodPromptsList  = "prompts/list"
	MethodInitialize   = "initialize"
	MethodPing         = "ping"
)

// context key type (unexported to prevent collisions).
type ctxKey int

const (
	ctxKeyInfo ctxKey = iota
)

// WithInfo stores a RequestInfo in the context.
func WithInfo(ctx context.Context, info *RequestInfo) context.Context {
	return context.WithValue(ctx, ctxKeyInfo, info)
}

// GetInfo retrieves the RequestInfo from the context, or nil if not present.
func GetInfo(ctx context.Context) *RequestInfo {
	v, _ := ctx.Value(ctxKeyInfo).(*RequestInfo)
	return v
}
