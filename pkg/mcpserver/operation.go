package mcpserver

import (
	"context"
	"sync/atomic"
)

// OperationTracker lets a host account for accepted HTTP MCP operations until
// their dispatch, including settlement and recording defers and recording
// children that outlive their signing deadline, has finished. It also covers
// finite SSE connect-time checks/denial recording, not a healthy GET stream's
// lifetime.
//
// BeginOperation is called synchronously before SSE connect-time work, reading
// or dispatching an SSE POST /message, or a Streamable HTTP request. It must
// atomically either register the operation and return a non-nil end function
// with ok=true, or refuse it with ok=false. A refusal (including ok=true with a
// nil end) receives HTTP 503 without dispatch. The server calls a valid end
// exactly once, including on early returns and panic unwinding; for SSE POST
// the lease transfers to the worker before HTTP 202 is written. Neither method
// nor end may panic. Implementations must be safe for concurrent use.
//
// A shutdown-capable host should close admission under the same lock used to
// register operations, then wait with its own deadline for the registered
// leases before its final flush. End is independent of the POST context and
// SSE GET lifetime. A drain timeout does not cancel or complete outstanding
// operations; the host owns the deadline and any per-lease timeout reporting.
// Request body reads and response writes have no deadline unless the host sets
// one. SSE response queue delivery may wait up to five additional seconds; it
// does not guarantee client consumption. A recorder's context deadline bounds
// the answer's wait, not the recorder's cleanup: its lease remains until return.
// This hook does not change ResponseGuard or provide crash durability.
type OperationTracker interface {
	BeginOperation() (end func(), ok bool)
}

// WithOperationTracker installs a host's acceptance-to-completion tracker on
// both HTTP transports. Install it at construction, before serving requests.
// A nil tracker (the default) leaves transport behavior unchanged.
func WithOperationTracker(tracker OperationTracker) SSEOption {
	return func(s *SSEServer) { s.operationTracker = tracker }
}

// Each owner releases one share. A recording child takes its share before
// launch, while the dispatch owner is still live. No goroutine or wait is
// needed to delay host completion, and the request keeps its original timing.
type operationLease struct {
	shares atomic.Int64
	end    func()
}

func (s *SSEServer) beginOperation() (*operationLease, bool) {
	end, ok := s.operationTracker.BeginOperation()
	if !ok || end == nil {
		return nil, false
	}
	lease := &operationLease{end: end}
	lease.shares.Store(1)
	return lease, true
}

func (l *operationLease) release() {
	if l.shares.Add(-1) == 0 {
		l.end()
	}
}

type operationLeaseKey struct{}

func withOperationLease(ctx context.Context, lease *operationLease) context.Context {
	if lease == nil {
		return ctx
	}
	return context.WithValue(ctx, operationLeaseKey{}, lease)
}

func retainOperation(ctx context.Context) *operationLease {
	lease, _ := ctx.Value(operationLeaseKey{}).(*operationLease)
	if lease != nil {
		lease.shares.Add(1)
	}
	return lease
}
