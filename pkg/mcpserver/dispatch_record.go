package mcpserver

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/http/httptrace"
	"os"
	"sync/atomic"
	"syscall"
)

// A call gate that takes something to admit a call (an amount reserved against
// a spending limit) needs to know, when the call ends, whether the call may
// have run upstream. An error from the upstream path does not say: a request
// whose response was lost may have been carried out. So the transports record
// what they actually did, in a dispatchRecord carried by the context of the
// call:
//
//	attempted  the upstream path was entered (a transport or the upstream
//	           manager was asked to send this call)
//	sent       a byte of the request may have left: a connection to the
//	           upstream was obtained (after dial and TLS), a request header
//	           was handed to it, or a write to a child process's stdin was
//	           made
//
// "attempted and not sent" is the only evidence of a failure that happened
// before dispatch (a dial or TLS failure, a request that could not be built,
// a context ended before the write, a closed session). Everything else that
// goes wrong after "sent" is an unknown outcome.
//
// The record is created by the proxy only when a gate admitted the call. With
// no record in the context every helper here does nothing.
type dispatchRecord struct {
	attempted atomic.Bool
	sent      atomic.Bool

	// Set by the proxy, on the request goroutine, once the router returned.
	forwarded bool
	resp      *Response
	err       error
}

type dispatchKey struct{}

func withDispatchRecord(ctx context.Context) (context.Context, *dispatchRecord) {
	if r := dispatchFrom(ctx); r != nil {
		return ctx, r
	}
	r := &dispatchRecord{}
	return context.WithValue(ctx, dispatchKey{}, r), r
}

func dispatchFrom(ctx context.Context) *dispatchRecord {
	r, _ := ctx.Value(dispatchKey{}).(*dispatchRecord)
	return r
}

// NoteDispatchAttempt records that the upstream path was entered for the call
// carried by ctx. UpstreamManager does it; a router that sends a call by some
// other means must call it first, and NoteDispatched before the first byte of
// the request can leave. A failure with an attempt and no NoteDispatched is
// treated as "never sent"; a failure with no attempt at all is an unknown
// outcome.
func NoteDispatchAttempt(ctx context.Context) {
	if r := dispatchFrom(ctx); r != nil {
		r.attempted.Store(true)
	}
}

// NoteDispatched records that a byte of the request may have left. Call it
// before the write, not after: a write that fails half way has still sent.
func NoteDispatched(ctx context.Context) {
	if r := dispatchFrom(ctx); r != nil {
		r.attempted.Store(true)
		r.sent.Store(true)
	}
}

// traceDispatch makes req report to the call's record the moment a connection
// is obtained or a request header is handed to it. That is before a byte can
// reach the network, so a dial failure, a TLS failure or a connection that
// was never obtained leaves "sent" unset, and anything later sets it. net/http may
// retry a request on a fresh connection; the flag stays set once any attempt
// reached the header write.
func traceDispatch(ctx context.Context, req *http.Request) *http.Request {
	rec := dispatchFrom(ctx)
	if rec == nil {
		return req
	}
	rec.attempted.Store(true)
	mark := func() { rec.sent.Store(true) }
	trace := &httptrace.ClientTrace{
		// GotConn counts too: once a connection exists, a write can be on its
		// way even if the first header callback has not run yet when the
		// caller gives up. A dial or TLS failure never reaches GotConn.
		GotConn:          func(httptrace.GotConnInfo) { mark() },
		WroteHeaderField: func(string, []string) { mark() },
		WroteHeaders:     mark,
		WroteRequest:     func(httptrace.WroteRequestInfo) { mark() },
	}
	return req.WithContext(httptrace.WithClientTrace(req.Context(), trace))
}

// noteWrite records a write to a process's stdin. A write that delivered no
// byte because the pipe was already closed (or the reader is gone) wrote
// nothing; any other write may have.
func noteWrite(ctx context.Context, n int, err error) {
	if n == 0 && err != nil && (errors.Is(err, os.ErrClosed) || errors.Is(err, io.ErrClosedPipe) || errors.Is(err, syscall.EPIPE)) {
		NoteDispatchAttempt(ctx)
		return
	}
	NoteDispatched(ctx)
}

// recordForward is called by the proxy with what the router returned.
func (r *dispatchRecord) recordForward(resp *Response, err error) {
	if r == nil {
		return
	}
	r.forwarded, r.resp, r.err = true, resp, err
}

// callEnd is how a forwarded call ended, for a gate that must decide whether
// to give back what it took.
type callEnd int

const (
	// endRan: the upstream accepted the call. Keep what was taken.
	endRan callEnd = iota
	// endNotRun: SatGate knows the call did not run: it was never sent, or
	// the upstream answered with a JSON-RPC protocol error that by the
	// JSON-RPC 2.0 specification means the request was not processed
	// (-32700, -32600, -32601, -32602; see rpcErrorMeansNotRun).
	endNotRun
	// endUnknown: the call was sent and nothing says it did not run: a
	// timeout, a connection closed after the send, a response that could not
	// be read, an HTTP error status, a redirect, any other JSON-RPC error
	// (-32603 Internal error and -32000..-32099 server errors can follow an
	// execution), or a result with isError=true (a tool-level error can
	// follow a partial or full execution). Keep what was taken.
	endUnknown
)

// classify says how the call ended. forwarded is false when the call was never
// handed to the router, which is "never sent".
func (r *dispatchRecord) classify() callEnd {
	if r == nil || !r.forwarded {
		return endNotRun
	}
	if r.err != nil {
		if r.attempted.Load() && !r.sent.Load() {
			return endNotRun
		}
		return endUnknown
	}
	if r.resp == nil {
		return endUnknown
	}
	if r.resp.Error != nil {
		if rpcErrorMeansNotRun(r.resp.Error.Code) {
			return endNotRun
		}
		return endUnknown
	}
	if !responseSucceeded(r.resp) {
		// isError=true: ambiguous, a tool can fail after it has acted.
		return endUnknown
	}
	return endRan
}

// rpcErrorMeansNotRun is true for the JSON-RPC 2.0 error codes that say the
// request was not processed: parse error, invalid request, method not found,
// invalid params. Every other code (-32603 Internal error, the -32000..-32099
// range the spec leaves to the server, anything else) can be sent by a server
// after it carried the call out, so it proves nothing.
func rpcErrorMeansNotRun(code int) bool {
	switch code {
	case CodeParseError, CodeInvalidRequest, CodeMethodNotFound, CodeInvalidParams:
		return true
	}
	return false
}
