package proxy

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// A route matches only requests that meet every matcher it sets. A request
// that fails one does not match that route, and matching continues with
// the next route in order.

// twoUpstreams returns a gateway whose first route (with the given match)
// goes to upstream "a" and, unless onlyFirst, whose second route (prefix
// "/") goes to upstream "b"; and a function that sends a request and
// reports which upstream received it ("a", "b", or "" with the status).
func twoUpstreams(t *testing.T, match config.RouteMatch, onlyFirst bool) func(method, path string, header http.Header) (string, int) {
	t.Helper()
	var mu sync.Mutex
	hit := ""
	stub := func(name string) *httptest.Server {
		s := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			mu.Lock()
			hit = name
			mu.Unlock()
			w.WriteHeader(http.StatusOK)
		}))
		t.Cleanup(s.Close)
		return s
	}
	a, b := stub("a"), stub("b")
	routes := []config.Route{{Name: "first", Match: match, Upstream: "a", Policy: config.RoutePolicy{Kind: "public"}}}
	if !onlyFirst {
		routes = append(routes, config.Route{Name: "default", Match: config.RouteMatch{PathPrefix: "/"}, Upstream: "b", Policy: config.RoutePolicy{Kind: "public"}})
	}
	gw := newTestGateway(t, &config.Config{
		Upstreams: map[string]config.Upstream{"a": {URL: a.URL, Timeout: 10 * time.Second}, "b": {URL: b.URL, Timeout: 10 * time.Second}},
		Routes:    routes,
	})
	return func(method, path string, header http.Header) (string, int) {
		mu.Lock()
		hit = ""
		mu.Unlock()
		req := httptest.NewRequest(method, path, nil)
		for k, vs := range header {
			req.Header[k] = vs
		}
		w := httptest.NewRecorder()
		gw.ServeHTTP(w, req)
		mu.Lock()
		defer mu.Unlock()
		return hit, w.Code
	}
}

func TestRouteMatch_MethodsComparedExactly(t *testing.T) {
	send := twoUpstreams(t, config.RouteMatch{PathPrefix: "/x", Methods: []string{"GET", "PUT"}}, false)
	for _, tc := range []struct{ method, want string }{
		{"GET", "a"}, {"PUT", "a"},
		{"POST", "b"}, {"DELETE", "b"}, {"HEAD", "b"}, {"PATCH", "b"},
		{"get", "b"}, {"Put", "b"}, // HTTP methods are case-sensitive
	} {
		if got, code := send(tc.method, "/x/1", nil); got != tc.want {
			t.Errorf("%s /x/1: reached %q (status %d), want %q", tc.method, got, code, tc.want)
		}
	}
}

func TestRouteMatch_MethodMismatchWithNoOtherRouteIs404(t *testing.T) {
	send := twoUpstreams(t, config.RouteMatch{PathPrefix: "/x", Methods: []string{"GET"}}, true)
	if got, code := send("GET", "/x", nil); got != "a" || code != http.StatusOK {
		t.Fatalf("GET /x: reached %q (status %d), want a/200", got, code)
	}
	if got, code := send("POST", "/x", nil); got != "" || code != http.StatusNotFound {
		t.Errorf("POST /x: reached %q (status %d), want no upstream and 404", got, code)
	}
}

// A route limited to GET does not serve other methods under its policy: a
// POST falls through to the next route and gets that route's policy.
func TestRouteMatch_MethodLimitedPublicRouteDoesNotServeOtherMethods(t *testing.T) {
	up := newPathUpstream(t)
	gw := newTestGateway(t, &config.Config{
		Upstreams: map[string]config.Upstream{"backend": {URL: up.srv.URL, Timeout: 10 * time.Second}},
		Routes: []config.Route{
			{Name: "read", Match: config.RouteMatch{PathPrefix: "/data", Methods: []string{"GET"}}, Upstream: "backend", Policy: config.RoutePolicy{Kind: "public"}},
			{Name: "write", Match: config.RouteMatch{PathPrefix: "/data"}, Upstream: "backend", Policy: config.RoutePolicy{Kind: "capability"}},
		},
	})
	front := httptest.NewServer(gw)
	t.Cleanup(front.Close)
	if got := sendTarget(t, front.URL, "GET", "/data/1"); got != http.StatusOK {
		t.Errorf("GET /data/1: status %d, want 200", got)
	}
	up.take()
	for _, m := range []string{"POST", "PUT", "DELETE"} {
		if got := sendTarget(t, front.URL, m, "/data/1"); got != http.StatusUnauthorized {
			t.Errorf("%s /data/1 without a token: status %d, want 401 (capability route)", m, got)
		}
	}
	if got := up.take(); len(got) != 0 {
		t.Errorf("upstream received %q, want no request", got)
	}
}

func TestRouteMatch_HeadersExactValueNameCaseInsensitive(t *testing.T) {
	send := twoUpstreams(t, config.RouteMatch{PathPrefix: "/", Headers: map[string]string{"X-Tier": "gold"}}, false)
	for _, tc := range []struct {
		name   string
		header http.Header
		want   string
	}{
		{"exact", http.Header{"X-Tier": {"gold"}}, "a"},
		{"name in lower case", http.Header{"x-tier": {"gold"}}, "a"},
		{"value differs in case", http.Header{"X-Tier": {"Gold"}}, "b"},
		{"value with extra text", http.Header{"X-Tier": {"gold-plus"}}, "b"},
		{"missing", nil, "b"},
		{"sent twice", http.Header{"X-Tier": {"gold", "gold"}}, "b"},
		{"sent twice under different case", http.Header{"X-Tier": {"gold"}, "x-tier": {"gold"}}, "b"},
	} {
		if got, code := send("GET", "/p", tc.header); got != tc.want {
			t.Errorf("%s: reached %q (status %d), want %q", tc.name, got, code, tc.want)
		}
	}
}

func TestRouteMatch_HeadersRegexValue(t *testing.T) {
	send := twoUpstreams(t, config.RouteMatch{PathPrefix: "/", Headers: map[string]string{"X-Version": "~^v[0-9]+$"}}, false)
	for _, tc := range []struct{ value, want string }{
		{"v2", "a"}, {"v10", "a"}, {"v2x", "b"}, {"V2", "b"}, {"", "b"},
	} {
		if got, code := send("GET", "/p", http.Header{"X-Version": {tc.value}}); got != tc.want {
			t.Errorf("X-Version %q: reached %q (status %d), want %q", tc.value, got, code, tc.want)
		}
	}
	// Unanchored patterns match anywhere in the value.
	send = twoUpstreams(t, config.RouteMatch{PathPrefix: "/", Headers: map[string]string{"X-Env": "~prod"}}, false)
	if got, _ := send("GET", "/p", http.Header{"X-Env": {"preprod-1"}}); got != "a" {
		t.Errorf("unanchored ~prod against preprod-1: reached %q, want a", got)
	}
}

func TestRouteMatch_PathRegex(t *testing.T) {
	// pathRegex as the only path matcher.
	send := twoUpstreams(t, config.RouteMatch{PathRegex: `^/items/[0-9]+$`}, false)
	for _, tc := range []struct{ path, want string }{
		{"/items/12", "a"}, {"/items/ab", "b"}, {"/items/12/x", "b"}, {"/v1/items/12", "b"},
	} {
		if got, code := send("GET", tc.path, nil); got != tc.want {
			t.Errorf("GET %s: reached %q (status %d), want %q", tc.path, got, code, tc.want)
		}
	}
	// pathRegex together with pathPrefix: both must hold.
	send = twoUpstreams(t, config.RouteMatch{PathPrefix: "/api/", PathRegex: `/v[0-9]+/`}, false)
	for _, tc := range []struct{ path, want string }{
		{"/api/v1/x", "a"}, {"/api/x", "b"}, {"/other/v1/x", "b"},
	} {
		if got, code := send("GET", tc.path, nil); got != tc.want {
			t.Errorf("GET %s: reached %q (status %d), want %q", tc.path, got, code, tc.want)
		}
	}
}

// pathRegex is matched against the checked, once-decoded path.
func TestRouteMatch_PathRegexSeesDecodedPath(t *testing.T) {
	send := twoUpstreams(t, config.RouteMatch{PathRegex: `^/a b$`}, false)
	if got, _ := send("GET", "/a%20b", nil); got != "a" {
		t.Errorf("GET /a%%20b: reached %q, want a", got)
	}
}

func TestRouteMatch_ExactAndPrefixBothApply(t *testing.T) {
	send := twoUpstreams(t, config.RouteMatch{PathExact: "/a/b", PathPrefix: "/z/"}, false)
	if got, _ := send("GET", "/a/b", nil); got != "b" {
		t.Errorf("GET /a/b with pathExact /a/b and pathPrefix /z/: reached %q, want b", got)
	}
}

func TestRouteMatch_EveryMatcherMustHold(t *testing.T) {
	send := twoUpstreams(t, config.RouteMatch{
		PathPrefix: "/api/",
		PathRegex:  `^/api/[a-z]+$`,
		Methods:    []string{"POST"},
		Headers:    map[string]string{"X-Client": "cli"},
	}, false)
	ok := http.Header{"X-Client": {"cli"}}
	for _, tc := range []struct {
		name, method, path string
		header             http.Header
		want               string
	}{
		{"all hold", "POST", "/api/orders", ok, "a"},
		{"method fails", "GET", "/api/orders", ok, "b"},
		{"regex fails", "POST", "/api/orders/1", ok, "b"},
		{"header fails", "POST", "/api/orders", http.Header{"X-Client": {"web"}}, "b"},
		{"prefix fails", "POST", "/apx/orders", ok, "b"},
	} {
		if got, code := send(tc.method, tc.path, tc.header); got != tc.want {
			t.Errorf("%s: reached %q (status %d), want %q", tc.name, got, code, tc.want)
		}
	}
}

func newGatewayErr(cfg *config.Config) error {
	cfg.Cloud = &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}}
	svc, err := macaroon.NewService("test-secret-key-for-tests")
	if err != nil {
		return err
	}
	_, err = New(Options{Config: cfg, Macaroon: svc, Governance: governance.NewService(nil), Lightning: lightning.NewMockProvider()})
	return err
}

// A configuration built in code is checked when the gateway is built: an
// invalid pathRegex or header pattern, a lower-case method and a rateLimit
// are refused with an error naming the route.
func TestRouteMatch_GatewayRefusesInvalidMatchersAndRateLimit(t *testing.T) {
	for name, route := range map[string]config.Route{
		"bad pathRegex":      {Match: config.RouteMatch{PathRegex: "("}},
		"bad header pattern": {Match: config.RouteMatch{PathPrefix: "/", Headers: map[string]string{"X-A": "~("}}},
		"bad header name":    {Match: config.RouteMatch{PathPrefix: "/", Headers: map[string]string{"X A": "v"}}},
		"lower-case method":  {Match: config.RouteMatch{PathPrefix: "/", Methods: []string{"get"}}},
		"empty method":       {Match: config.RouteMatch{PathPrefix: "/", Methods: []string{""}}},
		"rateLimit":          {Match: config.RouteMatch{PathPrefix: "/"}, RateLimit: &config.RateLimit{RequestsPerMinute: 10}},
	} {
		route.Name = "route-under-test"
		route.Policy = config.RoutePolicy{Kind: "public"}
		err := newGatewayErr(&config.Config{Routes: []config.Route{route}})
		if err == nil {
			t.Errorf("%s: gateway built, want refusal", name)
			continue
		}
		if !strings.Contains(err.Error(), "route-under-test") {
			t.Errorf("%s: error %q does not name the route", name, err)
		}
	}
}
