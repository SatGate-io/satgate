package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
)

// Optional router seams for a tenant router that has something exact to tell
// an agent.
//
// The proxy asks its UpstreamRouter for two things it cannot express through
// the UpstreamRouter interface: a tools/list that can fail, and a message for
// an error that is shown to the agent as it is. A router opts in by
// implementing either interface below. A router that implements neither (the
// default router, and any router written before these seams existed) is
// handled exactly as before: tools/list lists whatever AllToolsForTenant
// returns, and a failed tools/call is answered "upstream error: <error>".

// TenantToolsLister is implemented by a router whose tools/list can fail.
//
// When the router implements it, the proxy calls ToolsForTenant instead of
// AllToolsForTenant. A non-nil error ends the request: the agent gets a
// JSON-RPC error (code CodeUpstreamError) and never an empty list that looks
// like "this tenant has no tools". The error's message is AgentMessage(err)
// when the router also implements AgentMessenger and recognises err, and
// "upstream error: <err>" otherwise.
type TenantToolsLister interface {
	ToolsForTenant(ctx context.Context, tenantID string) ([]json.RawMessage, error)
}

// AgentMessenger is implemented by a router that owns the exact words for some
// of its errors.
//
// For a tools/call or tools/list that failed with err, the proxy calls
// AgentMessage(err). When ok is true, msg is the whole JSON-RPC error message,
// with no "upstream error: " prefix and nothing added. The error code is
// unchanged (CodeUpstreamError). When ok is false, the message is
// "upstream error: <err>" as it always was.
//
// Only the message changes. The proxy still records the original err for the
// call gate, so a call that may have run upstream stays an unknown outcome
// whatever words the agent is shown.
type AgentMessenger interface {
	AgentMessage(err error) (msg string, ok bool)
}

// upstreamErrorMessage is the message of a failed tools/call or tools/list.
func upstreamErrorMessage(router UpstreamRouter, err error) string {
	if m, ok := router.(AgentMessenger); ok {
		if msg, ok := m.AgentMessage(err); ok {
			return msg
		}
	}
	return fmt.Sprintf("upstream error: %v", err)
}

// routerTools lists a tenant's tools through the router. For a router that
// implements TenantToolsLister a failure becomes the second result, a ready
// JSON-RPC error response; the first result is then nil. For every other
// router it is AllToolsForTenant and a nil response, as before.
func routerTools(ctx context.Context, router UpstreamRouter, id json.RawMessage, tenantID string) ([]json.RawMessage, *Response) {
	l, ok := router.(TenantToolsLister)
	if !ok {
		return router.AllToolsForTenant(ctx, tenantID), nil
	}
	tools, err := l.ToolsForTenant(ctx, tenantID)
	if err != nil {
		return nil, NewErrorResponse(id, CodeUpstreamError, upstreamErrorMessage(router, err))
	}
	return tools, nil
}

// UpstreamNameForTool returns the key of the upstream that would receive a
// call to toolName, as it was configured. A router that must act on one
// service (for example, renew only that service's sign-in after it answered
// 401) uses it to tell which service a tool belongs to. It reports false when
// no upstream would receive the call.
func (m *UpstreamManager) UpstreamNameForTool(toolName string) (string, bool) {
	c, err := m.ResolveUpstream(toolName)
	if err != nil || c == nil {
		return "", false
	}
	return c.name, true
}
