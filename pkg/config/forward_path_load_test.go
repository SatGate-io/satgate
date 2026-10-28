package config

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// loadEvery loads data through ParseYAML, Load and Validate and returns
// the error of each, keyed by the load path.
func loadEvery(t *testing.T, data []byte) map[string]error {
	t.Helper()
	errs := map[string]error{}
	_, errs["ParseYAML"] = ParseYAML(data)
	path := filepath.Join(t.TempDir(), "gateway.yaml")
	if err := os.WriteFile(path, data, 0o600); err != nil {
		t.Fatal(err)
	}
	_, errs["Load"] = Load(path)
	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		t.Fatal(err)
	}
	errs["Validate"] = cfg.Validate()
	return errs
}

func forwardYAML(upstreamURL, rewrite string) []byte {
	s := "upstreams:\n  backend:\n    url: \"" + upstreamURL + "\"\nroutes:\n  - name: route-under-test\n    match:\n      pathPrefix: /api/\n    upstream: backend\n"
	if rewrite != "" {
		s += "    rewrite: '" + rewrite + "'\n"
	}
	return []byte(s + "    policy:\n      kind: public\n")
}

// A static rewrite is the path sent upstream; one that fails the request
// path rule on its own is refused on every load path, naming the route.
func TestLoad_RewriteThatFailsThePathRuleRefusedNamingRoute(t *testing.T) {
	for _, rewrite := range []string{
		"/other/../changed", // dot-dot segment
		"/a/./b",            // dot segment
		"/a/..",             // trailing dot-dot
		"relative",          // does not begin with "/"
		"/a%2Fb",            // encoded slash
		"/a%5cb",            // encoded backslash
		`/a\b`,              // backslash
		"/a%zz",             // "%" not followed by two hex digits
	} {
		for api, err := range loadEvery(t, forwardYAML("https://api.example.com", rewrite)) {
			if err == nil || !strings.Contains(err.Error(), "route route-under-test") || !strings.Contains(err.Error(), "rewrite") {
				t.Errorf("%s rewrite %q: got %v, want a rewrite refusal naming route-under-test", api, rewrite, err)
			}
		}
	}
}

// Rewrites that pass the rule still load. A rewrite is the decoded path
// as written, so "%" in it is sent as "%25": "/a%2e%2e/b" is sent as
// "/a%252e%252e/b" and has no dot segment.
func TestLoad_RewriteThatPassesThePathRuleLoads(t *testing.T) {
	for _, rewrite := range []string{"/v2/new-endpoint", "/", "/a b", "/a%20b", "/.well-known/x", "/a..b", "/a%2e%2e/b"} {
		for api, err := range loadEvery(t, forwardYAML("https://api.example.com", rewrite)) {
			if err != nil {
				t.Errorf("%s rewrite %q: %v, want it to load", api, rewrite, err)
			}
		}
	}
}

// An upstream URL's path is joined in front of every forwarded path; one
// that fails the request path rule on its own is refused on every load
// path, naming the upstream.
func TestLoad_UpstreamBasePathThatFailsThePathRuleRefusedNamingUpstream(t *testing.T) {
	for _, u := range []string{
		"https://api.example.com/a/../b",
		"https://api.example.com/a/./",
		"https://api.example.com/a/%2e%2e/b",
		"https://api.example.com/a%2Fb",
		"https://api.example.com/a%5Cb",
		"https://api.example.com/a%zz", // does not parse as a URL; its path fails the rule
		"https://api.example.com/a/..?q=1",
	} {
		for api, err := range loadEvery(t, forwardYAML(u, "")) {
			if err == nil || !strings.Contains(err.Error(), "upstream backend") || !strings.Contains(err.Error(), "URL path") {
				t.Errorf("%s upstream %q: got %v, want a URL path refusal naming upstream backend", api, u, err)
			}
		}
	}
}

// Upstream URLs whose path is empty or passes the rule still load, as does
// one that fails to parse for a reason other than its path (as on main).
func TestLoad_UpstreamBasePathThatPassesThePathRuleLoads(t *testing.T) {
	for _, u := range []string{
		"https://api.example.com",
		"https://api.example.com/",
		"https://api.example.com/v1",
		"https://api.example.com/v1/",
		"https://api.example.com/a%20b/",
		"https://api.example.com/v1?key=x",
		"https://api.example.com:port/v1",
	} {
		for api, err := range loadEvery(t, forwardYAML(u, "")) {
			if err != nil {
				t.Errorf("%s upstream %q: %v, want it to load", api, u, err)
			}
		}
	}
}

// A header matcher on Host or on a framing or hop-by-hop header is refused
// on every load path, naming the route: the HTTP server handles these
// before routing, so how many times one was sent cannot be judged.
func TestLoad_FramingAndHostHeaderMatchersRefusedNamingRoute(t *testing.T) {
	for _, name := range []string{
		"Host", "host",
		"Content-Length", "content-length",
		"Transfer-Encoding", "Connection", "Keep-Alive", "Proxy-Connection",
		"TE", "te", "Trailer", "Upgrade",
	} {
		data := routeYAML("      pathPrefix: /\n      headers:\n        "+name+": \"0\"", "")
		for api, err := range loadEvery(t, data) {
			if err == nil || !strings.Contains(err.Error(), "route route-under-test") || !strings.Contains(err.Error(), "cannot be matched") {
				t.Errorf("%s headers name %q: got %v, want a refusal naming route-under-test", api, name, err)
			}
		}
	}
	// Other header names, including ones that merely start like these,
	// still load.
	for _, name := range []string{"X-Content-Length", "Content-Type", "Tenant", "Hosts"} {
		data := routeYAML("      pathPrefix: /\n      headers:\n        "+name+": v", "")
		for api, err := range loadEvery(t, data) {
			if err != nil {
				t.Errorf("%s headers name %q: %v, want it to load", api, name, err)
			}
		}
	}
}

// Every matcher refusal, on every load path, names the route.
func TestLoad_EveryMatcherRefusalOnEveryLoadPathNamesRoute(t *testing.T) {
	for name, match := range map[string]string{
		"invalid pathRegex":      "      pathRegex: \"(\"",
		"invalid header pattern": "      pathPrefix: /\n      headers:\n        X-A: \"~(\"",
		"invalid header name":    "      pathPrefix: /\n      headers:\n        \"X A\": v",
		"lower-case method":      "      pathPrefix: /\n      methods: [get]",
		"mixed-case method":      "      pathPrefix: /\n      methods: [Post]",
		"empty method":           "      pathPrefix: /\n      methods: [\"\"]",
		"non-token method":       "      pathPrefix: /\n      methods: [\"GET POST\"]",
		"rateLimit":              "      pathPrefix: /\n    rateLimit: {}",
	} {
		for api, err := range loadEvery(t, routeYAML(match, "")) {
			if err == nil || !strings.Contains(err.Error(), "route-under-test") {
				t.Errorf("%s %s: got %v, want a refusal naming route-under-test", api, name, err)
			}
		}
	}
}
