package mcpserver

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"errors"
	"fmt"
	"net"
)

// UpstreamStatusError is returned by the HTTP transports when the upstream
// answered a request with an HTTP error status. Callers that need to act on a
// status (for example to renew a credential after a 401) match it with
// errors.As instead of searching the error text. Only a status line the HTTP
// client parsed produces one: a malformed response, a status mentioned in a
// body, and a JSON-RPC error never do.
type UpstreamStatusError struct {
	// Status is the HTTP status code of the response.
	Status int
	msg    string
}

func (e *UpstreamStatusError) Error() string { return e.msg }

// NewUpstreamStatusError returns an UpstreamStatusError for status with a
// fixed message. Wrappers that must keep the status available to errors.As
// but not the original text use it.
func NewUpstreamStatusError(status int) *UpstreamStatusError {
	return &UpstreamStatusError{Status: status, msg: fmt.Sprintf("upstream returned HTTP status %d", status)}
}

func newUpstreamStatusError(status int, format string, args ...any) error {
	return &UpstreamStatusError{Status: status, msg: fmt.Sprintf(format, args...)}
}

// UpstreamHTTPStatus reports the HTTP status carried by err, if any.
func UpstreamHTTPStatus(err error) (int, bool) {
	var se *UpstreamStatusError
	if errors.As(err, &se) {
		return se.Status, true
	}
	return 0, false
}

// upstreamTransportError is a transport or protocol failure reduced to a fixed
// sentence. It is what a transport returns, with SetRedactUpstreamText on, for
// a failure whose own text could quote bytes the upstream sent. cause is kept
// only for errors the caller may need to match (context errors); it is never
// part of the text.
type upstreamTransportError struct {
	msg   string
	cause error
}

func (e *upstreamTransportError) Error() string { return e.msg }
func (e *upstreamTransportError) Unwrap() error { return e.cause }

// ssrfBlockedError is the dialer's refusal of a private or internal address.
type ssrfBlockedError struct{ host string }

func (e *ssrfBlockedError) Error() string {
	return fmt.Sprintf("SSRF blocked: upstream resolved to private/internal IP %s", e.host)
}

// redactTransportError reduces an error from the HTTP client to a fixed
// sentence chosen by its type. net/http parser errors quote the bytes the
// upstream sent (a status line, a header line), so their text is never kept.
func redactTransportError(err error) error {
	if err == nil {
		return nil
	}
	var ssrf *ssrfBlockedError
	var unknownAuthority x509.UnknownAuthorityError
	var certInvalid x509.CertificateInvalidError
	var hostname x509.HostnameError
	var verify *tls.CertificateVerificationError
	var tlsRecord tls.RecordHeaderError
	var netErr net.Error
	var opErr *net.OpError
	switch {
	case errors.Is(err, context.Canceled):
		return &upstreamTransportError{msg: "upstream request canceled", cause: context.Canceled}
	case errors.Is(err, context.DeadlineExceeded):
		return &upstreamTransportError{msg: "upstream request timed out", cause: context.DeadlineExceeded}
	case errors.As(err, &ssrf):
		return &upstreamTransportError{msg: "upstream address is not allowed (private or internal)"}
	case errors.As(err, &unknownAuthority), errors.As(err, &certInvalid), errors.As(err, &hostname),
		errors.As(err, &verify), errors.As(err, &tlsRecord):
		return &upstreamTransportError{msg: "TLS handshake with upstream failed"}
	case errors.As(err, &netErr) && netErr.Timeout():
		return &upstreamTransportError{msg: "upstream connection timed out"}
	case errors.As(err, &opErr) && opErr.Op == "dial":
		return &upstreamTransportError{msg: "could not connect to upstream"}
	}
	return &upstreamTransportError{msg: "upstream connection failed or sent a malformed response"}
}
