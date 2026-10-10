package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"
)

const maxSameOriginRedirects = 3

const (
	headerAuthorization = "Authorization"
	headerSessionID     = "Mcp-Session-Id"
)

// newUpstreamHTTPClient builds the client used by upstream transports.
// Redirects are not followed by the client. doUpstream applies the origin rule.
func newUpstreamHTTPClient(tlsSkipVerify, allowPrivate bool) *http.Client {
	return &http.Client{
		Timeout:   0,
		Transport: SSRFSafeTransport(tlsSkipVerify, allowPrivate),
		CheckRedirect: func(req *http.Request, via []*http.Request) error {
			return http.ErrUseLastResponse
		},
	}
}

// upstreamRefusal is a redirect or endpoint refusal. Its text names only a
// status and an origin, so callers can show it without further filtering.
type upstreamRefusal struct{ msg string }

func (e *upstreamRefusal) Error() string { return e.msg }

func refusalf(format string, args ...any) error {
	return &upstreamRefusal{msg: fmt.Sprintf(format, args...)}
}

// wrapUpstreamError prefixes err with context naming rawURL. When err is a
// refusal, the context carries only the origin of rawURL (no userinfo, path,
// or query). Other errors keep the full URL as before.
func wrapUpstreamError(prefix, rawURL string, err error) error {
	var refusal *upstreamRefusal
	if errors.As(err, &refusal) {
		origin := "unknown-origin"
		if u, perr := url.Parse(rawURL); perr == nil {
			origin = displayOrigin(u)
		}
		return fmt.Errorf("%s %s: %w", prefix, origin, err)
	}
	return fmt.Errorf("%s %s: %w", prefix, rawURL, err)
}

// wrapUpstreamErrorRedacted is wrapUpstreamError. With redact set the context
// names the origin of rawURL only: rawURL is configuration, but it can carry a
// credential in its userinfo or query, and the error may be shown to a caller.
func wrapUpstreamErrorRedacted(prefix, rawURL string, err error, redact bool) error {
	if !redact {
		return wrapUpstreamError(prefix, rawURL, err)
	}
	origin := "unknown-origin"
	if u, perr := url.Parse(rawURL); perr == nil {
		origin = displayOrigin(u)
	}
	return fmt.Errorf("%s %s: %w", prefix, origin, err)
}

// A tools/call request can carry out an order. Once its bytes may have left,
// a redirect answer says nothing about whether the first hop ran it, and
// sending the body again could run it twice (or run it once and hide that
// behind a later reply). So a request marked by markNoReplay is never sent a
// second time by this package: a redirect is returned as an error after the
// send (the dispatch record already says "sent", so a gate keeps whatever it
// took), net/http gets no GetBody to replay the body with (an HTTP/2 stream
// refused by the server is retried only when GetBody is set), and the body is
// not rewound by redirectRequest.
type noReplayKey struct{}

// markNoReplay returns req marked as not to be replayed when msg, the JSON-RPC
// message it carries, is a tools/call, or is not a single JSON-RPC object that
// can be read (a batch may contain a tools/call; a body that cannot be read
// cannot be shown safe). Other methods (initialize, tools/list, ping,
// notifications) keep the redirect rule below unchanged.
func markNoReplay(req *http.Request, msg []byte) *http.Request {
	if !mayHaveSideEffects(msg) {
		return req
	}
	req.GetBody = nil
	return req.WithContext(context.WithValue(req.Context(), noReplayKey{}, true))
}

func mayHaveSideEffects(msg []byte) bool {
	var m struct {
		Method string `json:"method"`
	}
	if err := json.Unmarshal(msg, &m); err != nil {
		return true
	}
	return m.Method == MethodToolsCall
}

func noReplay(req *http.Request) bool {
	v, _ := req.Context().Value(noReplayKey{}).(bool)
	return v
}

// doUpstream performs req and applies the upstream redirect rule.
//
// A stored header is a tenant-configured header (including Authorization) or
// Mcp-Session-Id. When any stored header is present, no redirect is followed.
// Otherwise only same-origin redirects are followed, at most 3. A refused
// redirect returns an error naming the status code and the target origin
// (scheme, host, and port only). A request marked by markNoReplay is not
// redirected at all.
func doUpstream(client *http.Client, req *http.Request, hasStoredHeaders bool) (*http.Response, error) {
	return doUpstreamRedacted(client, req, hasStoredHeaders, false)
}

// doUpstreamRedacted is doUpstream. With redact set, a refusal names only the
// status code (the target origin is chosen by the upstream, so it is not
// repeated), and a failure of the HTTP exchange itself is reduced to a fixed
// sentence by kind: its own text can quote the bytes the upstream sent.
func doUpstreamRedacted(client *http.Client, req *http.Request, hasStoredHeaders, redact bool) (*http.Response, error) {
	if client == nil || client.Transport == nil {
		return nil, fmt.Errorf("upstream HTTP client is not configured")
	}
	if req == nil || req.URL == nil {
		return nil, fmt.Errorf("upstream request has no URL")
	}
	upstream := req.URL
	for followed := 0; ; followed++ {
		resp, err := client.Transport.RoundTrip(req)
		if err != nil {
			if redact {
				// net/http parser errors quote the bytes the upstream sent.
				return nil, redactTransportError(err)
			}
			return nil, err
		}
		if !isRedirectStatus(resp.StatusCode) {
			return resp, nil
		}
		status := resp.StatusCode
		loc, ok := redirectLocation(resp)
		discardBody(resp.Body)
		if noReplay(req) {
			// The request may already have run. Do not follow, do not send
			// the body again, and do not take the redirect's answer (or the
			// next hop's) as the call's outcome.
			return nil, refusalf("upstream redirect status %d refused: the call was already sent and is not sent again", status)
		}
		if !ok {
			return nil, refusalf("upstream redirect status %d has an invalid location", status)
		}
		refused := func() error {
			if redact {
				return refusalf("upstream redirect status %d refused", status)
			}
			return refusalf("upstream redirect status %d to %s refused", status, displayOrigin(loc))
		}
		if hasStoredHeaders || requestHasStoredHeader(req) || !sameOrigin(upstream, loc) || followed >= maxSameOriginRedirects {
			return nil, refused()
		}
		next, err := redirectRequest(req, loc, status)
		if err != nil {
			return nil, refused()
		}
		req = next
	}
}

func isRedirectStatus(status int) bool {
	switch status {
	case http.StatusMovedPermanently, http.StatusFound, http.StatusSeeOther,
		http.StatusTemporaryRedirect, http.StatusPermanentRedirect:
		return true
	default:
		return false
	}
}

// redirectLocation resolves Location against the response request and returns
// an absolute URL with userinfo and fragment removed. The raw Location value
// is never returned.
func redirectLocation(resp *http.Response) (*url.URL, bool) {
	if resp == nil || resp.Header == nil {
		return nil, false
	}
	raw := resp.Header.Get("Location")
	if raw == "" {
		return nil, false
	}
	loc, err := url.Parse(raw)
	if err != nil {
		return nil, false
	}
	if resp.Request != nil && resp.Request.URL != nil {
		loc = resp.Request.URL.ResolveReference(loc)
	}
	if loc.Scheme == "" || loc.Host == "" {
		return nil, false
	}
	loc.User = nil
	loc.Fragment = ""
	return loc, true
}

func redirectRequest(prev *http.Request, loc *url.URL, status int) (*http.Request, error) {
	method := prev.Method
	includeBody := status == http.StatusTemporaryRedirect || status == http.StatusPermanentRedirect
	if status == http.StatusMovedPermanently || status == http.StatusFound || status == http.StatusSeeOther {
		includeBody = false
		if method != http.MethodGet && method != http.MethodHead {
			method = http.MethodGet
		}
	}
	if includeBody && prev.Body != nil && prev.GetBody == nil && prev.ContentLength != 0 {
		return nil, fmt.Errorf("redirect body cannot be replayed")
	}
	var body io.Reader
	if includeBody && prev.GetBody != nil {
		b, err := prev.GetBody()
		if err != nil {
			return nil, err
		}
		body = b
	}
	next, err := http.NewRequestWithContext(prev.Context(), method, loc.String(), body)
	if err != nil {
		return nil, err
	}
	if includeBody {
		next.GetBody = prev.GetBody
		next.ContentLength = prev.ContentLength
	}
	for k, vv := range prev.Header {
		if isStoredHeaderName(k) {
			continue
		}
		if !includeBody && isBodyHeaderName(k) {
			continue
		}
		next.Header[k] = append([]string(nil), vv...)
	}
	return next, nil
}

func isStoredHeaderName(name string) bool {
	switch http.CanonicalHeaderKey(name) {
	case headerAuthorization, headerSessionID, "Proxy-Authorization", "Cookie", "Cookie2":
		return true
	default:
		return false
	}
}

func isBodyHeaderName(name string) bool {
	switch http.CanonicalHeaderKey(name) {
	case "Content-Type", "Content-Length", "Content-Encoding", "Content-Language", "Content-Location":
		return true
	default:
		return false
	}
}

func requestHasStoredHeader(req *http.Request) bool {
	if req == nil {
		return false
	}
	return req.Header.Get(headerAuthorization) != "" || req.Header.Get(headerSessionID) != ""
}

func discardBody(body io.ReadCloser) {
	if body == nil {
		return
	}
	io.Copy(io.Discard, io.LimitReader(body, 4096))
	body.Close()
}

// resolveUpstreamReference resolves ref against base, the URL of the stream
// that delivered it. A relative ref follows RFC 3986. Userinfo and fragment
// are dropped.
func resolveUpstreamReference(base *url.URL, refRaw string) (*url.URL, error) {
	if base == nil || base.Scheme == "" || base.Host == "" {
		return nil, refusalf("invalid upstream URL")
	}
	ref, err := url.Parse(strings.TrimSpace(refRaw))
	if err != nil {
		return nil, refusalf("invalid endpoint URL")
	}
	resolved := base.ResolveReference(ref)
	resolved.User = nil
	resolved.Fragment = ""
	if resolved.Scheme == "" || resolved.Hostname() == "" {
		return nil, refusalf("endpoint URL has no origin")
	}
	return resolved, nil
}

// endpointOriginError refuses an endpoint whose origin differs from the
// configured upstream's origin.
func endpointOriginError(upstream, resolved *url.URL) error {
	return endpointOriginErrorRedacted(upstream, resolved, false)
}

// endpointOriginErrorRedacted is endpointOriginError. With redact set it does
// not name the endpoint's origin, which the upstream chose.
func endpointOriginErrorRedacted(upstream, resolved *url.URL, redact bool) error {
	if !sameOrigin(upstream, resolved) {
		if redact {
			return refusalf("SSE endpoint origin does not match upstream origin")
		}
		return refusalf("SSE endpoint origin %s does not match upstream origin %s", displayOrigin(resolved), displayOrigin(upstream))
	}
	return nil
}

func sameOrigin(a, b *url.URL) bool {
	if a == nil || b == nil {
		return false
	}
	return originKey(a) == originKey(b)
}

func originKey(u *url.URL) string {
	scheme := strings.ToLower(u.Scheme)
	host := strings.ToLower(u.Hostname())
	port := u.Port()
	if port == "" {
		switch scheme {
		case "http":
			port = "80"
		case "https":
			port = "443"
		}
	}
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	return scheme + "://" + host + ":" + port
}

// displayOrigin is scheme://host[:port] with no userinfo, path, query, or fragment.
func displayOrigin(u *url.URL) string {
	if u == nil || u.Scheme == "" || u.Hostname() == "" {
		return "unknown-origin"
	}
	host := u.Hostname()
	if strings.Contains(host, ":") {
		host = "[" + host + "]"
	}
	origin := strings.ToLower(u.Scheme) + "://" + host
	if port := u.Port(); port != "" {
		origin += ":" + port
	}
	return origin
}

// displayOriginOf is displayOrigin for a raw URL string.
func displayOriginOf(rawURL string) string {
	u, err := url.Parse(rawURL)
	if err != nil {
		return "unknown-origin"
	}
	return displayOrigin(u)
}
