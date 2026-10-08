package mcpserver

import (
	"bytes"
	"encoding/json"
	"errors"
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
	// CodeRateLimited: the token made too many requests. data.error is
	// "rate_limited" and data.retry_after_seconds says when to retry.
	CodeRateLimited = -32004
)

// ToolCallParams extracts tool name and arguments from a tools/call request.
//
// Arguments stays the raw JSON the client sent. It is never decoded into Go
// values here: a decode into float64 fails on a number such as 1e999, and the
// Go error text carries the number back to the caller before any argument rule
// has run. The rule checker (pkg/argrules) is the one reader of argument
// values, and it reads exact text.
type ToolCallParams struct {
	Name      string          `json:"name"`
	Arguments json.RawMessage `json:"arguments,omitempty"`
}

// Fixed messages for a tools/call whose params cannot be read. None of them
// carries a byte of the input: the text goes back to the agent, and the input
// may hold a value the token's rules were written to keep private.
var (
	errToolCallEmptyParams = errors.New("empty params")
	errToolCallParams      = errors.New("invalid tool call params")
	errToolCallName        = errors.New("tool name is required")
	errToolCallArguments   = errors.New("invalid tool call params: arguments must be an object")
)

// ParseToolCall extracts ToolCallParams from a request's params field. It
// reads the name and keeps the arguments as raw JSON. Every error is one of a
// few fixed messages with no input bytes.
func ParseToolCall(params json.RawMessage) (*ToolCallParams, error) {
	if len(params) == 0 {
		return nil, errToolCallEmptyParams
	}
	var raw struct {
		Name      json.RawMessage `json:"name"`
		Arguments json.RawMessage `json:"arguments"`
	}
	if err := json.Unmarshal(params, &raw); err != nil {
		// The Go error text can quote the offending value. Say nothing of it.
		return nil, errToolCallParams
	}
	var tc ToolCallParams
	if len(raw.Name) > 0 {
		if err := json.Unmarshal(raw.Name, &tc.Name); err != nil {
			// A name that is a number, array or object. Do not echo it.
			return nil, errToolCallParams
		}
	}
	if tc.Name == "" {
		return nil, errToolCallName
	}
	if args := bytes.TrimSpace(raw.Arguments); len(args) > 0 && !bytes.Equal(args, []byte("null")) {
		if args[0] != '{' {
			return nil, errToolCallArguments
		}
		tc.Arguments = append(json.RawMessage(nil), args...)
	}
	return &tc, nil
}

// toolCallParseMessage is the only text about a bad tools/call that goes back
// to the agent. It passes one of the fixed ParseToolCall messages through and
// turns anything else into a generic one, so a future error that quotes input
// cannot reach the client.
func toolCallParseMessage(err error) string {
	switch {
	case errors.Is(err, errToolCallEmptyParams),
		errors.Is(err, errToolCallName),
		errors.Is(err, errToolCallArguments),
		errors.Is(err, errToolCallParams):
		return err.Error()
	}
	return errToolCallParams.Error()
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
