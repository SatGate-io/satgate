package mcpserver

import (
	"encoding/json"
	"fmt"
)

// JSON-RPC 2.0 types for MCP protocol handling.
// These extend the types in pkg/mcp with response handling needed for the proxy.

// Request is a JSON-RPC 2.0 request.
type Request struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"` // number, string, or null
	Method  string          `json:"method"`
	Params  json.RawMessage `json:"params,omitempty"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	JSONRPC string          `json:"jsonrpc"`
	ID      json.RawMessage `json:"id,omitempty"`
	Result  json.RawMessage `json:"result,omitempty"`
	Error   *RPCError       `json:"error,omitempty"`
}

// RPCError is a JSON-RPC 2.0 error object.
type RPCError struct {
	Code    int             `json:"code"`
	Message string          `json:"message"`
	Data    json.RawMessage `json:"data,omitempty"`
}

// Standard JSON-RPC error codes.
const (
	CodeParseError     = -32700
	CodeInvalidRequest = -32600
	CodeMethodNotFound = -32601
	CodeInvalidParams  = -32602
	CodeInternalError  = -32603
)

// SatGate custom error codes (in the JSON-RPC server error range -32000 to -32099).
const (
	CodeBudgetExhausted = -32000
	CodePolicyDenied    = -32001
	CodeUpstreamError   = -32002
	CodeUpstreamTimeout = -32003
)

// ToolCallParams extracts tool name and arguments from a tools/call request.
type ToolCallParams struct {
	Name      string                 `json:"name"`
	Arguments map[string]interface{} `json:"arguments,omitempty"`
}

// ParseToolCall extracts ToolCallParams from a request's params field.
func ParseToolCall(params json.RawMessage) (*ToolCallParams, error) {
	if len(params) == 0 {
		return nil, fmt.Errorf("empty params")
	}
	var tc ToolCallParams
	if err := json.Unmarshal(params, &tc); err != nil {
		return nil, fmt.Errorf("invalid tool call params: %w", err)
	}
	if tc.Name == "" {
		return nil, fmt.Errorf("tool name is required")
	}
	return &tc, nil
}

// NewErrorResponse creates a JSON-RPC error response.
func NewErrorResponse(id json.RawMessage, code int, message string) *Response {
	return &Response{
		JSONRPC: "2.0",
		ID:      id,
		Error: &RPCError{
			Code:    code,
			Message: message,
		},
	}
}

// NewErrorResponseWithData creates a JSON-RPC error response with additional data.
func NewErrorResponseWithData(id json.RawMessage, code int, message string, data interface{}) *Response {
	resp := NewErrorResponse(id, code, message)
	if data != nil {
		if d, err := json.Marshal(data); err == nil {
			resp.Error.Data = d
		}
	}
	return resp
}

// MCP method constants (re-exported from pkg/mcp for convenience).
const (
	MethodInitialize   = "initialize"
	MethodPing         = "ping"
	MethodToolsList    = "tools/list"
	MethodToolsCall    = "tools/call"
	MethodResourceRead = "resources/read"
	MethodResourceList = "resources/list"
	MethodPromptsList  = "prompts/list"
	MethodPromptsGet   = "prompts/get"

	// Notifications (no response expected)
	MethodInitialized          = "notifications/initialized"
	MethodCancelled            = "notifications/cancelled"
	MethodProgress             = "notifications/progress"
	MethodRootsListChanged     = "notifications/roots/list_changed"
	MethodToolsListChanged     = "notifications/tools/list_changed"
	MethodResourcesListChanged = "notifications/resources/list_changed"
)

// IsNotification returns true if the method is a JSON-RPC notification
// (no ID, no response expected).
func IsNotification(method string) bool {
	return len(method) > 14 && method[:14] == "notifications/"
}

// IsToolCall returns true if this is a tools/call method.
func IsToolCall(method string) bool {
	return method == MethodToolsCall
}
