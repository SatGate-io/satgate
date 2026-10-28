package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
)

// The path rule runs before the preflight answer: OPTIONS on an allowed
// path keeps its 204, and OPTIONS on a refused path gets 400. Neither is
// forwarded.
func TestOptions_AllowedPathGets204(t *testing.T) {
	up := newPathUpstream(t)
	front := pathGateway(t, up)
	for _, target := range []string{"/pub/x", "/", "/svc/a%20b", "/admin/x", "/pub/caf%C3%A9", "/pub/x?q=../y"} {
		if got := sendTarget(t, front, "OPTIONS", target); got != http.StatusNoContent {
			t.Errorf("OPTIONS %s: status %d, want 204", target, got)
		}
	}
	if got := up.take(); len(got) != 0 {
		t.Errorf("upstream received %q, want no request", got)
	}
}

func TestOptions_RefusedPathGets400(t *testing.T) {
	up := newPathUpstream(t)
	front := pathGateway(t, up)
	for _, target := range []string{
		"/x/../a", "/x/./a", "/..", "/x/%2e%2e/a", "/x%2fa", "/x%5Ca", `/x\a`,
		"/x/%00", "/x/%zz", "/x/%252f", "http:opaque",
	} {
		// ("OPTIONS *" is answered by net/http itself and never reaches
		// the gateway.)
		if got := sendTarget(t, front, "OPTIONS", target); got != http.StatusBadRequest {
			t.Errorf("OPTIONS %s: status %d, want 400", target, got)
		}
	}
	if got := up.take(); len(got) != 0 {
		t.Errorf("upstream received %q, want no request", got)
	}
}

// forwardGateway serves one capability route /locked/ and one public route
// /, both to an upstream at up's URL plus basePath, with rewrite on both.
// change, if not nil, runs on the built gateway before it serves.
func forwardGateway(t *testing.T, up *pathUpstream, basePath, rewrite string, change func(*Gateway)) string {
	t.Helper()
	gw := newTestGateway(t, &config.Config{
		Upstreams: map[string]config.Upstream{"backend": {URL: up.srv.URL + basePath, Timeout: 10 * time.Second}},
		Routes: []config.Route{
			{Name: "locked", Match: config.RouteMatch{PathPrefix: "/locked/"}, Rewrite: rewrite, Upstream: "backend", Policy: config.RoutePolicy{Kind: "capability"}},
			{Name: "public", Match: config.RouteMatch{PathPrefix: "/"}, Rewrite: rewrite, Upstream: "backend", Policy: config.RoutePolicy{Kind: "public"}},
		},
	})
	if change != nil {
		change(gw)
	}
	front := httptest.NewServer(gw)
	t.Cleanup(front.Close)
	return front.URL
}

// A static rewrite and an upstream base path that pass the rule are
// forwarded: the upstream receives the base path joined with the rewrite,
// or with the request path as checked.
func TestForwardPath_RewriteAndBasePathThatPassAreForwarded(t *testing.T) {
	for _, tc := range []struct {
		basePath, rewrite, target, want string
	}{
		{"", "/v2/new", "/old/x", "/v2/new"},
		{"", "/v2/new", "/old/x?q=1", "/v2/new?q=1"},
		{"", "/a b", "/old", "/a%20b"},
		{"/base", "", "/checked/%41", "/base/checked/%41"},
		{"/base/", "", "/checked/a%20b", "/base/checked/a%20b"},
		{"/base", "", "/", "/base/"},
		{"/base", "/v2/new", "/old", "/base/v2/new"},
		{"/a%20b", "", "/c", "/a%20b/c"},
	} {
		up := newPathUpstream(t)
		front := forwardGateway(t, up, tc.basePath, tc.rewrite, nil)
		if got := sendTarget(t, front, "GET", tc.target); got != http.StatusOK {
			t.Errorf("base %q rewrite %q GET %s: status %d, want 200", tc.basePath, tc.rewrite, tc.target, got)
			continue
		}
		if got := up.take(); len(got) != 1 || got[0] != tc.want {
			t.Errorf("base %q rewrite %q GET %s: upstream received %q, want exactly [%q]", tc.basePath, tc.rewrite, tc.target, got, tc.want)
		}
	}
}

// A static rewrite or an upstream base path that fails the rule on its own
// is refused when the gateway is built, naming the route or upstream.
func TestForwardPath_RewriteOrBasePathThatFailsRefusedAtNew(t *testing.T) {
	up := newPathUpstream(t)
	for _, rewrite := range []string{"/other/../changed", "/a/.", "x", "/a%2Fb", `/a\b`} {
		err := newGatewayErr(&config.Config{
			Upstreams: map[string]config.Upstream{"backend": {URL: up.srv.URL}},
			Routes:    []config.Route{{Name: "route-under-test", Match: config.RouteMatch{PathPrefix: "/"}, Rewrite: rewrite, Upstream: "backend", Policy: config.RoutePolicy{Kind: "public"}}},
		})
		if err == nil || !strings.Contains(err.Error(), "route route-under-test") {
			t.Errorf("rewrite %q: got %v, want a refusal naming route-under-test", rewrite, err)
		}
	}
	for _, base := range []string{"/a/../b", "/a/./", "/a/%2e%2e", "/a%2fb", "/a%zz"} {
		err := newGatewayErr(&config.Config{
			Upstreams: map[string]config.Upstream{"upstream-under-test": {URL: up.srv.URL + base}},
			Routes:    []config.Route{{Name: "r", Match: config.RouteMatch{PathPrefix: "/"}, Upstream: "upstream-under-test", Policy: config.RoutePolicy{Kind: "public"}}},
		})
		if err == nil || !strings.Contains(err.Error(), "upstream upstream-under-test") {
			t.Errorf("base path %q: got %v, want a refusal naming upstream-under-test", base, err)
		}
	}
}

// At request time the final outgoing path, after the rewrite or strip and
// the join with the upstream URL's base path, is checked again before the
// route's policy runs. Here the rewrite and the upstream are changed after
// the gateway is built (as routes and upstreams added at run time are), so
// only the request-time check can refuse them: the request gets 400, not
// the capability route's 401, and the upstream is not contacted.
func TestForwardPath_FinalPathThatFailsGets400BeforePolicy(t *testing.T) {
	t.Run("rewrite", func(t *testing.T) {
		up := newPathUpstream(t)
		front := forwardGateway(t, up, "", "", func(gw *Gateway) {
			for i := range gw.config.Routes {
				gw.config.Routes[i].Rewrite = "/other/../changed"
			}
		})
		for _, target := range []string{"/locked/x", "/pub/x"} {
			if got := sendTarget(t, front, "GET", target); got != http.StatusBadRequest {
				t.Errorf("GET %s: status %d, want 400", target, got)
			}
		}
		if got := up.take(); len(got) != 0 {
			t.Errorf("upstream received %q, want no request", got)
		}
	})
	t.Run("upstream base path", func(t *testing.T) {
		up := newPathUpstream(t)
		front := forwardGateway(t, up, "", "", func(gw *Gateway) {
			p, err := gw.createProxy(up.srv.URL+"/base/..", nil)
			if err != nil {
				t.Fatal(err)
			}
			gw.proxyMu.Lock()
			gw.proxies["backend"] = p
			gw.proxyMu.Unlock()
		})
		for _, target := range []string{"/locked/x", "/pub/x"} {
			if got := sendTarget(t, front, "GET", target); got != http.StatusBadRequest {
				t.Errorf("GET %s: status %d, want 400", target, got)
			}
		}
		if got := up.take(); len(got) != 0 {
			t.Errorf("upstream received %q, want no request", got)
		}
	})
}

// A header matcher on Content-Length (or another framing or hop-by-hop
// header, or Host) is refused when the gateway is built: the HTTP server
// collapses identical repeated Content-Length fields before routing, so a
// repeated field could otherwise satisfy the matcher.
func TestRouteMatch_FramingHeaderMatcherRefusedAtNew(t *testing.T) {
	for _, name := range []string{"Content-Length", "Transfer-Encoding", "Connection", "Keep-Alive", "Proxy-Connection", "TE", "Trailer", "Upgrade", "Host"} {
		err := newGatewayErr(&config.Config{Routes: []config.Route{{
			Name:   "route-under-test",
			Match:  config.RouteMatch{PathPrefix: "/", Headers: map[string]string{name: "0"}},
			Policy: config.RoutePolicy{Kind: "deny"},
		}}})
		if err == nil || !strings.Contains(err.Error(), "route route-under-test") {
			t.Errorf("headers name %q: got %v, want a refusal naming route-under-test", name, err)
		}
	}
}

// Characters the client sent unescaped that a path must carry escaped are
// sent upstream escaped (the checked path's escaped form); a path the
// client sent already escaped is sent as sent.
func TestRequestPath_ForwardedInItsEscapedForm(t *testing.T) {
	up := newPathUpstream(t)
	front := pathGateway(t, up)
	for target, want := range map[string]string{
		"/pub/café":  "/pub/caf%C3%A9",
		"/pub/a#b":   "/pub/a%23b",
		"/pub/a|b":   "/pub/a%7Cb",
		"/pub/a^b":   "/pub/a%5Eb",
		"/pub/a`b":   "/pub/a%60b",
		"/pub/a{b}":  "/pub/a%7Bb%7D",
		"/pub/a%7cb": "/pub/a%7cb",
		"/pub/[x]":   "/pub/[x]",
		// With one character that must be escaped, the whole path is sent
		// as the standard escaping of its decoded form.
		"/pub/a|%41": "/pub/a%7CA",
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
