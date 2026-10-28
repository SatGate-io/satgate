package config

import (
	"net/http"
	"testing"
)

// Valid matchers load and are compiled once, at load.
func TestParseYAML_MatchersCompiledAtLoad(t *testing.T) {
	cfg, err := ParseYAML(routeYAML(`      pathPrefix: /api/
      pathRegex: "^/api/v[0-9]+/"
      methods: [GET, POST, CUSTOM-METHOD]
      headers:
        X-Version: "2"
        X-Env: "~^prod-"`, ""))
	if err != nil {
		t.Fatal(err)
	}
	m := &cfg.Routes[0].Match
	if m.compiledPathRegex == nil || len(m.compiledHeaders) != 2 {
		t.Fatalf("matchers not compiled at load: regex %v, headers %d", m.compiledPathRegex, len(m.compiledHeaders))
	}
	h := http.Header{"X-Version": {"2"}, "X-Env": {"prod-eu"}}
	if !cfg.Routes[0].MatchRequest("/api/v1/x", "POST", h) {
		t.Error("request meeting every matcher did not match")
	}
	if cfg.Routes[0].MatchRequest("/api/v1/x", "PUT", h) {
		t.Error("PUT matched a route limited to GET, POST, CUSTOM-METHOD")
	}
}

// MatchRequest: every configured matcher must hold; a header must be sent
// exactly once; header names compare case-insensitively.
func TestRouteMatchRequest_EveryMatcher(t *testing.T) {
	routes := []Route{{Name: "r", Match: RouteMatch{
		PathPrefix: "/api/*",
		PathRegex:  `^/api/[a-z]+$`,
		Methods:    []string{"GET"},
		Headers:    map[string]string{"X-Client": "cli"},
	}}}
	if err := CheckRouteMatchers(routes); err != nil {
		t.Fatal(err)
	}
	r := &routes[0]
	ok := http.Header{"x-client": {"cli"}}
	for _, tc := range []struct {
		name, path, method string
		header             http.Header
		want               bool
	}{
		{"all hold", "/api/users", "GET", ok, true},
		{"prefix fails", "/apx/users", "GET", ok, false},
		{"regex fails", "/api/users/1", "GET", ok, false},
		{"method fails", "/api/users", "POST", ok, false},
		{"method case differs", "/api/users", "get", ok, false},
		{"header missing", "/api/users", "GET", nil, false},
		{"header twice", "/api/users", "GET", http.Header{"X-Client": {"cli", "cli"}}, false},
		{"header value differs", "/api/users", "GET", http.Header{"X-Client": {"CLI"}}, false},
	} {
		if got := r.MatchRequest(tc.path, tc.method, tc.header); got != tc.want {
			t.Errorf("%s: MatchRequest = %v, want %v", tc.name, got, tc.want)
		}
	}
}

// A route with no path matcher matches nothing; a route built in code with
// an invalid pattern matches nothing.
func TestRouteMatchRequest_FailsClosed(t *testing.T) {
	if (&Route{Match: RouteMatch{Methods: []string{"GET"}}}).MatchRequest("/", "GET", nil) {
		t.Error("route with no path matcher matched")
	}
	if (&Route{Match: RouteMatch{PathRegex: "("}}).MatchRequest("/", "GET", nil) {
		t.Error("route with an invalid pathRegex matched")
	}
	if (&Route{Match: RouteMatch{PathPrefix: "/", Headers: map[string]string{"X-A": "~("}}}).MatchRequest("/", "GET", http.Header{"X-A": {"("}}) {
		t.Error("route with an invalid header pattern matched")
	}
}
