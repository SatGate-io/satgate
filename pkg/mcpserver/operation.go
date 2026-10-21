package mcpserver

// OperationTracker lets a host account for accepted HTTP MCP operations until
// their dispatch, including settlement and recording defers, has finished.
// It does not track the lifetime of an SSE GET connection.
//
// BeginOperation is called synchronously before reading or dispatching an SSE
// POST /message or a Streamable HTTP request. It must atomically either register
// the operation and return a non-nil end function with ok=true, or refuse it with
// ok=false. Refused requests receive HTTP 503 without dispatch. The server calls
// end exactly once, including on early returns and panic unwinding; for SSE the
// lease transfers to the worker before HTTP 202 is written. Neither method nor
// end may panic. Implementations must be safe for concurrent use.
//
// A shutdown-capable host should close admission under the same lock used to
// register operations, then wait with a deadline for the registered leases
// before its final flush. End is independent of the POST context and SSE GET
// lifetime. A drain timeout does not cancel or complete outstanding operations;
// the host owns the deadline and any per-lease identity or timeout reporting.
// This hook does not change ResponseGuard or provide crash durability.
type OperationTracker interface {
	BeginOperation() (end func(), ok bool)
}

// WithOperationTracker installs a host's acceptance-to-completion tracker on
// both HTTP transports. Install it at construction, before serving requests.
// A nil tracker (the default) leaves transport behavior unchanged.
func WithOperationTracker(tracker OperationTracker) SSEOption {
	return func(s *SSEServer) {
		s.operationTracker = tracker
	}
}
