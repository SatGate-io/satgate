package proxy

import (
	"bufio"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
)

// On-the-wire tests for the request path rule: a client writes the request
// line verbatim over TCP, and a stub upstream records the request target it
// receives.

type pathUpstream struct {
	mu   sync.Mutex
	uris []string
	srv  *httptest.Server
}

func newPathUpstream(t *testing.T) *pathUpstream {
	t.Helper()
	up := &pathUpstream{}
	up.srv = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		up.mu.Lock()
		up.uris = append(up.uris, r.RequestURI)
		up.mu.Unlock()
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(up.srv.Close)
	return up
}

// take returns the request targets received since the last call.
func (up *pathUpstream) take() []string {
	up.mu.Lock()
	defer up.mu.Unlock()
	got := up.uris
	up.uris = nil
	return got
}

// sendTarget writes a request with this request target to the gateway and
// returns the response status.
func sendTarget(t *testing.T, front, method, target string, header ...string) int {
	t.Helper()
	c, err := net.Dial("tcp", strings.TrimPrefix(front, "http://"))
	if err != nil {
		t.Fatal(err)
	}
	defer c.Close()
	_ = c.SetDeadline(time.Now().Add(10 * time.Second))
	var b strings.Builder
	fmt.Fprintf(&b, "%s %s HTTP/1.1\r\nHost: gateway.test\r\n", method, target)
	for _, h := range header {
		b.WriteString(h + "\r\n")
	}
	b.WriteString("Content-Length: 0\r\nConnection: close\r\n\r\n")
	if _, err := io.WriteString(c, b.String()); err != nil {
		t.Fatal(err)
	}
	resp, err := http.ReadResponse(bufio.NewReader(c), nil)
	if err != nil {
		t.Fatal(err)
	}
	_, _ = io.Copy(io.Discard, resp.Body)
	resp.Body.Close()
	return resp.StatusCode
}

// pathGateway serves, in order: /admin/ (capability), /svc/ (public, prefix
// stripped), /bare (public, prefix stripped, no trailing slash) and / (public).
func pathGateway(t *testing.T, up *pathUpstream) string {
	t.Helper()
	gw := newTestGateway(t, &config.Config{
		Admin:     config.AdminConfig{Token: "admin-fixture"},
		Upstreams: map[string]config.Upstream{"backend": {URL: up.srv.URL, Timeout: 10 * time.Second}},
		Routes: []config.Route{
			{Name: "admin", Match: config.RouteMatch{PathPrefix: "/admin/"}, Upstream: "backend", Policy: config.RoutePolicy{Kind: "capability"}},
			{Name: "svc", Match: config.RouteMatch{PathPrefix: "/svc/"}, StripPrefix: true, Upstream: "backend", Policy: config.RoutePolicy{Kind: "public"}},
			{Name: "bare", Match: config.RouteMatch{PathPrefix: "/bare"}, StripPrefix: true, Upstream: "backend", Policy: config.RoutePolicy{Kind: "public"}},
			{Name: "public", Match: config.RouteMatch{PathPrefix: "/"}, Upstream: "backend", Policy: config.RoutePolicy{Kind: "public"}},
		},
	})
	front := httptest.NewServer(gw)
	t.Cleanup(front.Close)
	return front.URL
}

// A path that, as sent or after one percent-decoding, has a "." or ".."
// segment, an encoded slash or backslash, a backslash, a NUL or a "%" not
// followed by two hex digits is refused with 400 before any route is
// matched, and the upstream is not contacted.
func TestRequestPath_RefusedFormsGet400AndNeverReachUpstream(t *testing.T) {
	up := newPathUpstream(t)
	front := pathGateway(t, up)
	for _, target := range []string{
		"/pub/../admin/x",         // dot-dot segment
		"/pub/./admin/x",          // dot segment
		"/pub/..",                 // trailing dot-dot
		"/..",                     // dot-dot at the root
		"/pub/%2e%2e/admin/x",     // encoded dot-dot
		"/pub/%2E%2E/admin/x",     // encoded dot-dot, upper case
		"/pub/.%2e/admin/x",       // half-encoded dot-dot
		"/pub/%2e/admin/x",        // encoded dot
		"/pub%2fadmin/x",          // encoded slash
		"/pub%2Fadmin/x",          // encoded slash, upper case
		"/pub%5cadmin/x",          // encoded backslash
		"/pub%5Cadmin/x",          // encoded backslash, upper case
		`/pub\admin\x`,            // backslash
		"/pub/%00/x",              // encoded NUL
		"/pub/%zz",                // percent sequence that does not decode
		"/pub/%2",                 // truncated percent sequence
		"/pub/%252fadmin",         // decodes once to an encoded slash
		"/pub/%255cadmin",         // decodes once to an encoded backslash
		"/pub/%25zz",              // decodes once to a percent that does not decode
		"/svc/../admin/x",         // dot-dot under a stripped prefix
		"/svc/%2e%2e/admin/x",     // encoded dot-dot under a stripped prefix
		"/svc/a%2fb",              // encoded slash under a stripped prefix
		"/check-payment/../admin", // built-in endpoint prefix
		"/api/agent/pack/../x",    // built-in endpoint prefix
	} {
		if got := sendTarget(t, front, "GET", target); got != http.StatusBadRequest {
			t.Errorf("GET %s: status %d, want 400", target, got)
		}
		if got := up.take(); len(got) != 0 {
			t.Errorf("GET %s: upstream received %q, want no request", target, got)
		}
	}
}

// Paths with encoded characters that are not refused are matched on their
// decoded form and reach the upstream exactly as sent.
func TestRequestPath_EncodedCharactersAllowedAndForwardedAsSent(t *testing.T) {
	up := newPathUpstream(t)
	front := pathGateway(t, up)
	for _, target := range []string{
		"/pub/a%20b",
		"/pub/caf%C3%A9",
		"/pub/a%3Fb",
		"/pub/%41BC",
		"/pub/.well-known/x",
		"/pub/.../x",
		"/pub/a..b/c.",
		"/pub/%252e%252e/x", // decodes once to "%2e%2e", not a dot-dot segment
		"/pub/a?q=../x",     // the query is not part of the path
	} {
		if got := sendTarget(t, front, "GET", target); got != http.StatusOK {
			t.Errorf("GET %s: status %d, want 200", target, got)
			continue
		}
		if got := up.take(); len(got) != 1 || got[0] != target {
			t.Errorf("GET %s: upstream received %q, want exactly [%q]", target, got, target)
		}
	}
}

// The prefix strip works on the checked path: the upstream receives the
// rest of it encoded as the client sent it.
func TestRequestPath_StripPrefixForwardsTheCheckedRemainderAsSent(t *testing.T) {
	up := newPathUpstream(t)
	front := pathGateway(t, up)
	for target, want := range map[string]string{
		"/svc/a%20b/c":     "/a%20b/c",
		"/svc/%41":         "/%41",
		"/svc/caf%C3%A9/x": "/caf%C3%A9/x",
		"/svc/":            "/",
		"/svc/x?q=1":       "/x?q=1",
		"/bare/x":          "/x",
		"/barex":           "/x",
	} {
		if got := sendTarget(t, front, "GET", target); got != http.StatusOK {
			t.Errorf("GET %s: status %d, want 200", target, got)
			continue
		}
		if got := up.take(); len(got) != 1 || got[0] != want {
			t.Errorf("GET %s: upstream received %q, want exactly [%q]", target, got, want)
		}
	}
}

// A strip whose remainder would not pass the path rule is refused rather
// than forwarded.
func TestRequestPath_StripRemainderThatFailsTheRuleIsRefused(t *testing.T) {
	up := newPathUpstream(t)
	front := pathGateway(t, up)
	for _, target := range []string{"/bare./x", "/bare../x"} {
		if got := sendTarget(t, front, "GET", target); got != http.StatusBadRequest {
			t.Errorf("GET %s: status %d, want 400", target, got)
		}
		if got := up.take(); len(got) != 0 {
			t.Errorf("GET %s: upstream received %q, want no request", target, got)
		}
	}
}

// Built-in endpoints are served as before.
func TestRequestPath_BuiltInEndpointsStillServed(t *testing.T) {
	up := newPathUpstream(t)
	front := pathGateway(t, up)
	for _, tc := range []struct {
		method, target string
		header         []string
		want           int
	}{
		{"GET", "/health", nil, http.StatusOK},
		{"GET", "/healthz", nil, http.StatusOK},
		{"GET", "/.well-known/satgate-agent-jwks.json", nil, http.StatusOK},
		{"POST", "/api/capability/mint", []string{"X-Admin-Token: admin-fixture"}, http.StatusBadRequest}, // empty body
		{"GET", "/api/capability/ping", nil, http.StatusUnauthorized},
		{"GET", "/check-payment/" + strings.Repeat("ab", 32), nil, http.StatusOK},
		{"OPTIONS", "/pub/x", nil, http.StatusNoContent},
	} {
		if got := sendTarget(t, front, tc.method, tc.target, tc.header...); got != tc.want {
			t.Errorf("%s %s: status %d, want %d", tc.method, tc.target, got, tc.want)
		}
	}
	if got := up.take(); len(got) != 0 {
		t.Errorf("upstream received %q, want no request", got)
	}
}

// The stripped path is checked when the route is matched, before the
// route's policy runs: the request is refused with 400 rather than asked
// for a credential.
func TestRequestPath_StripRemainderRefusedBeforePolicy(t *testing.T) {
	up := newPathUpstream(t)
	gw := newTestGateway(t, &config.Config{
		Upstreams: map[string]config.Upstream{"backend": {URL: up.srv.URL, Timeout: 10 * time.Second}},
		Routes: []config.Route{
			{Name: "locked", Match: config.RouteMatch{PathPrefix: "/locked"}, StripPrefix: true, Upstream: "backend", Policy: config.RoutePolicy{Kind: "capability"}},
		},
	})
	front := httptest.NewServer(gw)
	t.Cleanup(front.Close)
	if got := sendTarget(t, front.URL, "GET", "/locked./x"); got != http.StatusBadRequest {
		t.Errorf("GET /locked./x: status %d, want 400 before the capability check", got)
	}
	if got := sendTarget(t, front.URL, "GET", "/locked/x"); got != http.StatusUnauthorized {
		t.Errorf("GET /locked/x without a token: status %d, want 401", got)
	}
	if got := up.take(); len(got) != 0 {
		t.Errorf("upstream received %q, want no request", got)
	}
}
