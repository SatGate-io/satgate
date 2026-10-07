package mcpserver

import (
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

// doUpstream performs req and applies the upstream redirect rule.
//
// A stored header is a tenant-configured header (including Authorization) or
// Mcp-Session-Id. When any stored header is present, no redirect is followed.
// Otherwise only same-origin redirects are followed, at most 3. A refused
// redirect returns an error naming the status code and the target origin
// (scheme, host, and port only).
func doUpstream(client *http.Client, req *http.Request, hasStoredHeaders bool) (*http.Response, error) {
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
			return nil, err
		}
		if !isRedirectStatus(resp.StatusCode) {
			return resp, nil
		}
		status := resp.StatusCode
		loc, ok := redirectLocation(resp)
		discardBody(resp.Body)
		if !ok {
			return nil, refusalf("upstream redirect status %d has an invalid location", status)
		}
		origin := displayOrigin(loc)
		if hasStoredHeaders || requestHasStoredHeader(req) || !sameOrigin(upstream, loc) || followed >= maxSameOriginRedirects {
			return nil, refusalf("upstream redirect status %d to %s refused", status, origin)
		}
		next, err := redirectRequest(req, loc, status)
		if err != nil {
			return nil, refusalf("upstream redirect status %d to %s refused", status, origin)
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
	if !sameOrigin(upstream, resolved) {
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
