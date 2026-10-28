package config

import (
	"strings"
	"testing"
)

func routeYAML(match, extra string) []byte {
	return []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: route-under-test
    match:
` + match + `
    upstream: backend
    policy:
      kind: public
` + extra)
}

// A route that sets rateLimit is refused at load with an error naming the
// route: the gateway does not enforce a per-route limit.
func TestParseYAML_RateLimitRefusedNamingRoute(t *testing.T) {
	_, err := ParseYAML(routeYAML("      pathPrefix: /api/", `    rateLimit:
      requestsPerMinute: 100
      burstSize: 10
      key: ip
`))
	if err == nil || !strings.Contains(err.Error(), "route-under-test") || !strings.Contains(err.Error(), "rateLimit") {
		t.Fatalf("got %v, want a rateLimit refusal naming route-under-test", err)
	}
	// An empty rateLimit block is still a rateLimit the route sets.
	if _, err := ParseYAML(routeYAML("      pathPrefix: /api/", "    rateLimit: {}\n")); err == nil {
		t.Error("rateLimit: {} loaded, want refusal")
	}
}

// Matchers that could never work as written are refused at load with an
// error naming the route.
func TestParseYAML_InvalidMatchersRefusedNamingRoute(t *testing.T) {
	for name, match := range map[string]string{
		"invalid pathRegex":      "      pathRegex: \"(\"",
		"invalid header pattern": "      pathPrefix: /\n      headers:\n        X-A: \"~(\"",
		"invalid header name":    "      pathPrefix: /\n      headers:\n        \"X A\": v",
		"Host header":            "      pathPrefix: /\n      headers:\n        host: example.com",
		"lower-case method":      "      pathPrefix: /\n      methods: [get]",
		"mixed-case method":      "      pathPrefix: /\n      methods: [Post]",
		"empty method":           "      pathPrefix: /\n      methods: [\"\"]",
		"method with space":      "      pathPrefix: /\n      methods: [\"GET POST\"]",
	} {
		_, err := ParseYAML(routeYAML(match, ""))
		if err == nil || !strings.Contains(err.Error(), "route-under-test") {
			t.Errorf("%s: got %v, want a refusal naming route-under-test", name, err)
		}
	}
}
