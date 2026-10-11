package mcpserver

import (
	"context"
	"errors"
	"net/http"
	"strings"
	"sync"
)

// SendReason says why the transport is about to send an HTTP request. A
// credential source uses it to decide whether it may do slow or costly work
// (renew a sign-in) or must answer from what it already has.
type SendReason int

const (
	// SendAgentRequest: an agent's request is waiting on this send. That
	// covers start-up (initialize and tool discovery) done for an agent call,
	// every message POST, and the first connect of an SSE stream.
	SendAgentRequest SendReason = iota
	// SendBackgroundReconnect: nobody is waiting on this send. The notification
	// GET, an SSE stream's own reconnect, and a replacement connection the
	// manager dials after the old one died, including its handshake.
	SendBackgroundReconnect
	// SendCleanup: ending a session (the Streamable HTTP DELETE on close).
	SendCleanup
)

func (r SendReason) String() string {
	switch r {
	case SendAgentRequest:
		return "agent request"
	case SendBackgroundReconnect:
		return "background reconnect"
	case SendCleanup:
		return "cleanup"
	default:
		return "unknown"
	}
}

// CredentialSource supplies the credential headers of one upstream at the
// moment each HTTP request is sent. It is optional (UpstreamConfig.Credentials,
// never set from YAML). When it is nil the transports send exactly the headers
// of UpstreamConfig.Headers, as before.
//
// When it is set, every HTTP request the transport makes for the upstream
// calls RequestCredentials immediately before sending, after any blocking
// step, and the request carries only the headers it returns (plus the headers
// the transport itself owns: Content-Type, Accept, Cache-Control and
// Mcp-Session-Id). The configured Headers are not sent. An error means the
// request is not sent: an agent request fails with it (wrapped, so errors.As
// and errors.Is still see it), a session DELETE is dropped locally, and a
// background reconnect waits for its normal retry. The source must be safe
// for concurrent use.
type CredentialSource interface {
	RequestCredentials(ctx context.Context, why SendReason) (map[string]string, error)
}

// CredentialObserver is an optional extension of CredentialSource. After the
// response status of a request that carried the source's headers is known,
// ObserveResponse is called with the headers that were sent. It is never
// called for a request the source refused to authorize.
type CredentialObserver interface {
	ObserveResponse(why SendReason, sent map[string]string, status int)
}

type sendReasonKey struct{}

// withSendReason marks ctx so the sends made under it report why.
func withSendReason(ctx context.Context, why SendReason) context.Context {
	return context.WithValue(ctx, sendReasonKey{}, why)
}

// sendReasonOf is the reason a send under ctx reports. The default is an
// agent request.
func sendReasonOf(ctx context.Context) SendReason {
	if v, ok := ctx.Value(sendReasonKey{}).(SendReason); ok {
		return v
	}
	return SendAgentRequest
}

type sentInfoKey struct{}

type sentInfo struct {
	why     SendReason
	headers map[string]string
}

// credentialGate is the one place a transport turns its configuration into
// the headers of a request. With no source it is the configured headers.
type credentialGate struct {
	src CredentialSource

	mu   sync.Mutex
	seen map[string]struct{}
}

func newCredentialGate(src CredentialSource) *credentialGate {
	return &credentialGate{src: src}
}

func (g *credentialGate) active() bool { return g != nil && g.src != nil }

// apply puts the credential headers on req. With no source that is the
// configured headers, exactly as the transports always did. With a source it
// is the source's answer, asked now; on error nothing is set and the caller
// must not send.
func (g *credentialGate) apply(ctx context.Context, why SendReason, req *http.Request, static map[string]string) (*http.Request, error) {
	if !g.active() {
		for k, v := range static {
			req.Header.Set(k, v)
		}
		return req, nil
	}
	hdrs, err := g.src.RequestCredentials(ctx, why)
	if err != nil {
		return req, &credentialSourceError{err: err}
	}
	g.remember(hdrs)
	for k, v := range hdrs {
		req.Header.Set(k, v)
	}
	return req.WithContext(context.WithValue(req.Context(), sentInfoKey{}, &sentInfo{why: why, headers: hdrs})), nil
}

// observe tells an observing source the status of a request it authorized.
func (g *credentialGate) observe(req *http.Request, resp *http.Response, err error) (*http.Response, error) {
	if err != nil || resp == nil || !g.active() || req == nil {
		return resp, err
	}
	obs, ok := g.src.(CredentialObserver)
	if !ok {
		return resp, err
	}
	if info, ok := req.Context().Value(sentInfoKey{}).(*sentInfo); ok {
		obs.ObserveResponse(info.why, info.headers, resp.StatusCode)
	}
	return resp, err
}

func (g *credentialGate) remember(h map[string]string) {
	g.mu.Lock()
	defer g.mu.Unlock()
	if g.seen == nil {
		g.seen = map[string]struct{}{}
	}
	for _, v := range h {
		v = strings.TrimSpace(v)
		if v == "" {
			continue
		}
		g.seen[v] = struct{}{}
		// "Bearer <token>": an upstream echoes the token alone as often as
		// the whole header.
		if i := strings.LastIndexByte(v, ' '); i >= 0 && i+1 < len(v) {
			g.seen[v[i+1:]] = struct{}{}
		}
	}
}

// scrub removes every value the source has returned from err's text. The
// error keeps its type chain (errors.As and errors.Is still work).
func (g *credentialGate) scrub(err error) error {
	if err == nil || !g.active() {
		return err
	}
	g.mu.Lock()
	vals := make([]string, 0, len(g.seen))
	for v := range g.seen {
		vals = append(vals, v)
	}
	g.mu.Unlock()
	msg := err.Error()
	out := msg
	for _, v := range vals {
		out = strings.ReplaceAll(out, v, "[redacted]")
	}
	if out == msg {
		return err
	}
	return &scrubbedError{msg: out, err: err}
}

// scrubText is scrub for a string a transport is about to log or return.
func (g *credentialGate) scrubText(s string) string {
	if !g.active() {
		return s
	}
	g.mu.Lock()
	defer g.mu.Unlock()
	for v := range g.seen {
		s = strings.ReplaceAll(s, v, "[redacted]")
	}
	return s
}

type scrubbedError struct {
	msg string
	err error
}

func (e *scrubbedError) Error() string { return e.msg }
func (e *scrubbedError) Unwrap() error { return e.err }

// credentialSourceError is what a transport returns when the source refused a
// send. The source's own error stays reachable through errors.As / Is.
type credentialSourceError struct{ err error }

func (e *credentialSourceError) Error() string { return "credential source: " + e.err.Error() }
func (e *credentialSourceError) Unwrap() error { return e.err }

// IsCredentialSourceError reports whether err is a send the credential source
// refused (nothing was sent).
func IsCredentialSourceError(err error) bool {
	var ce *credentialSourceError
	return errors.As(err, &ce)
}
