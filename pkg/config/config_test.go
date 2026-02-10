package config

import (
	"os"
	"testing"
	"time"
)

// --- ParseYAML / Load tests ---

func TestParseYAML_MinimalValid(t *testing.T) {
	yaml := []byte(`
version: 1
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: test-route
    match:
      pathPrefix: /api/
    upstream: backend
    policy:
      kind: public
`)
	cfg, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Version != 1 {
		t.Errorf("expected version 1, got %d", cfg.Version)
	}
	if len(cfg.Routes) != 1 {
		t.Fatalf("expected 1 route, got %d", len(cfg.Routes))
	}
	if cfg.Routes[0].Name != "test-route" {
		t.Errorf("expected route name 'test-route', got %q", cfg.Routes[0].Name)
	}
}

func TestParseYAML_Defaults(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: r
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`)
	cfg, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Listen != ":8080" {
		t.Errorf("expected default listen ':8080', got %q", cfg.Server.Listen)
	}
	if cfg.Server.ReadTimeout != 30*time.Second {
		t.Errorf("expected default readTimeout 30s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 30*time.Second {
		t.Errorf("expected default writeTimeout 30s, got %v", cfg.Server.WriteTimeout)
	}
}

func TestParseYAML_CustomServerSettings(t *testing.T) {
	yaml := []byte(`
server:
  listen: ":9090"
  readTimeout: 10s
  writeTimeout: 15s
  maxRequestBody: 1048576
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: r
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`)
	cfg, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Server.Listen != ":9090" {
		t.Errorf("expected listen ':9090', got %q", cfg.Server.Listen)
	}
	if cfg.Server.ReadTimeout != 10*time.Second {
		t.Errorf("expected readTimeout 10s, got %v", cfg.Server.ReadTimeout)
	}
	if cfg.Server.WriteTimeout != 15*time.Second {
		t.Errorf("expected writeTimeout 15s, got %v", cfg.Server.WriteTimeout)
	}
	if cfg.Server.MaxRequestBody != 1048576 {
		t.Errorf("expected maxRequestBody 1048576, got %d", cfg.Server.MaxRequestBody)
	}
}

func TestParseYAML_EnvVarExpansion(t *testing.T) {
	os.Setenv("TEST_UPSTREAM_URL", "https://env-api.example.com")
	defer os.Unsetenv("TEST_UPSTREAM_URL")

	yaml := []byte(`
upstreams:
  backend:
    url: $TEST_UPSTREAM_URL
routes:
  - name: r
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`)
	cfg, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	upstream, ok := cfg.Upstreams["backend"]
	if !ok {
		t.Fatal("expected 'backend' upstream")
	}
	if upstream.URL != "https://env-api.example.com" {
		t.Errorf("expected env-expanded URL, got %q", upstream.URL)
	}
}

func TestParseYAML_InvalidYAML(t *testing.T) {
	yaml := []byte(`{{{invalid yaml`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestLoad_MissingFile(t *testing.T) {
	_, err := Load("/nonexistent/path/config.yaml")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestLoad_ValidFile(t *testing.T) {
	content := `
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: r
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`
	tmpFile, err := os.CreateTemp("", "satgate-config-*.yaml")
	if err != nil {
		t.Fatal(err)
	}
	defer os.Remove(tmpFile.Name())

	if _, err := tmpFile.WriteString(content); err != nil {
		t.Fatal(err)
	}
	tmpFile.Close()

	cfg, err := Load(tmpFile.Name())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Routes) != 1 {
		t.Errorf("expected 1 route, got %d", len(cfg.Routes))
	}
}

// --- Validation tests ---

func TestValidate_NoRoutes(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes: []
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for no routes")
	}
}

func TestValidate_RouteNoName(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for route with no name")
	}
}

func TestValidate_RouteNoPathMatcher(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: bad-route
    upstream: backend
    policy:
      kind: public
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for route with no path matcher")
	}
}

func TestValidate_RouteNoPolicyKind(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: no-policy
    match:
      pathPrefix: /
    upstream: backend
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for route with no policy kind")
	}
}

func TestValidate_RouteUnknownUpstream(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: bad-upstream
    match:
      pathPrefix: /
    upstream: nonexistent
    policy:
      kind: public
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for unknown upstream reference")
	}
}

func TestValidate_UpstreamNoURL(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    timeout: 10s
routes:
  - name: r
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for upstream with no URL")
	}
}

func TestValidate_UpstreamBadScheme(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: ftp://api.example.com
routes:
  - name: r
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for non-http/https scheme")
	}
}

func TestValidate_UpstreamCredentialsInURL(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://user:pass@api.example.com
routes:
  - name: r
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for credentials in upstream URL")
	}
}

func TestValidate_UpstreamPrivateIP(t *testing.T) {
	tests := []struct {
		name string
		url  string
	}{
		{"loopback", "http://127.0.0.1/api"},
		{"private_10", "http://10.0.0.1/api"},
		{"private_172", "http://172.16.0.1/api"},
		{"private_192", "http://192.168.1.1/api"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			yaml := []byte(`
upstreams:
  backend:
    url: ` + tt.url + `
routes:
  - name: r
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`)
			_, err := ParseYAML(yaml)
			if err == nil {
				t.Errorf("expected error for private IP %s", tt.url)
			}
		})
	}
}

func TestValidate_UpstreamPublicDNS_OK(t *testing.T) {
	// DNS hostnames (not IPs) should pass the SSRF check
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: r
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`)
	_, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error for public DNS upstream: %v", err)
	}
}

func TestValidate_L402NoPriceError(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: paid
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: l402
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for l402 route with no price")
	}
}

func TestValidate_L402WithPriceSats(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: paid
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: l402
      priceSats: 100
`)
	cfg, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Routes[0].Policy.Pay == nil {
		t.Error("expected synthesized Pay policy for l402")
	}
	if cfg.Routes[0].Policy.Pay.Price != 100 {
		t.Errorf("expected price 100, got %f", cfg.Routes[0].Policy.Pay.Price)
	}
}

func TestValidate_Fiat402NoPay(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: budget
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: fiat402
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for fiat402 route with no pay config")
	}
}

func TestValidate_InvalidPathRegex(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: regex-route
    match:
      pathRegex: "[invalid"
    upstream: backend
    policy:
      kind: public
`)
	_, err := ParseYAML(yaml)
	if err == nil {
		t.Error("expected error for invalid path regex")
	}
}

func TestValidate_ValidPathRegex(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: regex-route
    match:
      pathRegex: "^/api/v[0-9]+/"
    upstream: backend
    policy:
      kind: public
`)
	_, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error for valid regex: %v", err)
	}
}

// --- NormalizePolicyKind tests ---

func TestNormalizePolicyKind(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"observe", "chargeback"},
		{"audit", "chargeback"},
		{"control", "fiat402"},
		{"budget", "fiat402"},
		{"charge", "l402"},
		{"monetize", "l402"},
		{"protected", "capability"},
		{"protect", "capability"},
		// Pass-through
		{"public", "public"},
		{"l402", "l402"},
		{"capability", "capability"},
		{"chargeback", "chargeback"},
		{"fiat402", "fiat402"},
		{"deny", "deny"},
		{"unknown", "unknown"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizePolicyKind(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizePolicyKind(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestNormalizePayMode(t *testing.T) {
	tests := []struct {
		input    string
		expected string
	}{
		{"observe", "chargeback"},
		{"audit", "chargeback"},
		{"control", "fiat402"},
		{"budget", "fiat402"},
		{"charge", "l402"},
		{"monetize", "l402"},
		{"l402", "l402"},
		{"fiat402", "fiat402"},
		{"chargeback", "chargeback"},
	}
	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := NormalizePayMode(tt.input)
			if got != tt.expected {
				t.Errorf("NormalizePayMode(%q) = %q, want %q", tt.input, got, tt.expected)
			}
		})
	}
}

func TestIsBillingMode(t *testing.T) {
	tests := []struct {
		kind     string
		expected bool
	}{
		{"chargeback", true},
		{"fiat402", true},
		{"l402", true},
		{"observe", true},  // normalized to chargeback
		{"audit", true},    // normalized to chargeback
		{"control", true},  // normalized to fiat402
		{"budget", true},   // normalized to fiat402
		{"charge", true},   // normalized to l402
		{"monetize", true}, // normalized to l402
		{"public", false},
		{"capability", false},
		{"deny", false},
		{"protected", false}, // normalized to capability
	}
	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			got := IsBillingMode(tt.kind)
			if got != tt.expected {
				t.Errorf("IsBillingMode(%q) = %v, want %v", tt.kind, got, tt.expected)
			}
		})
	}
}

// --- Route matching tests ---

func TestRouteMatches_PathPrefix(t *testing.T) {
	route := Route{
		Name:  "prefix",
		Match: RouteMatch{PathPrefix: "/api/"},
	}
	tests := []struct {
		path   string
		method string
		want   bool
	}{
		{"/api/users", "GET", true},
		{"/api/", "POST", true},
		{"/api/v1/data", "GET", true},
		{"/other", "GET", false},
		{"/", "GET", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := route.Matches(tt.path, tt.method)
			if got != tt.want {
				t.Errorf("route.Matches(%q, %q) = %v, want %v", tt.path, tt.method, got, tt.want)
			}
		})
	}
}

func TestRouteMatches_PathExact(t *testing.T) {
	route := Route{
		Name:  "exact",
		Match: RouteMatch{PathExact: "/health"},
	}
	tests := []struct {
		path string
		want bool
	}{
		{"/health", true},
		{"/health/", false},
		{"/healthz", false},
		{"/", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := route.Matches(tt.path, "GET")
			if got != tt.want {
				t.Errorf("route.Matches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRouteMatches_PathRegex(t *testing.T) {
	route := Route{
		Name: "regex",
		Match: RouteMatch{
			PathRegex: `^/api/v\d+/`,
		},
	}
	if err := route.CompileRegex(); err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		path string
		want bool
	}{
		{"/api/v1/users", true},
		{"/api/v2/data", true},
		{"/api/v10/stuff", true},
		{"/api/vX/nope", false},
		{"/other", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := route.Matches(tt.path, "GET")
			if got != tt.want {
				t.Errorf("route.Matches(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}

func TestRouteMatches_Methods(t *testing.T) {
	route := Route{
		Name: "methods",
		Match: RouteMatch{
			PathPrefix: "/api/",
			Methods:    []string{"GET", "POST"},
		},
	}
	tests := []struct {
		method string
		want   bool
	}{
		{"GET", true},
		{"POST", true},
		{"get", true},  // case insensitive
		{"post", true}, // case insensitive
		{"PUT", false},
		{"DELETE", false},
	}
	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			got := route.Matches("/api/test", tt.method)
			if got != tt.want {
				t.Errorf("route.Matches with method %q = %v, want %v", tt.method, got, tt.want)
			}
		})
	}
}

func TestRouteMatchesWithHeaders_ExactMatch(t *testing.T) {
	route := Route{
		Name: "headers",
		Match: RouteMatch{
			PathPrefix: "/",
			Headers: map[string]string{
				"Content-Type": "application/json",
			},
		},
	}
	tests := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{"exact match", map[string]string{"Content-Type": "application/json"}, true},
		{"wrong value", map[string]string{"Content-Type": "text/html"}, false},
		{"missing header", map[string]string{}, false},
		{"nil headers (skips check)", nil, true},
		{"case insensitive key", map[string]string{"content-type": "application/json"}, true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := route.MatchesWithHeaders("/test", "GET", tt.headers)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestRouteMatchesWithHeaders_RegexPattern(t *testing.T) {
	route := Route{
		Name: "regex-header",
		Match: RouteMatch{
			PathPrefix: "/",
			Headers: map[string]string{
				"Content-Type": "~application/(json|xml)",
			},
		},
	}
	tests := []struct {
		name    string
		headers map[string]string
		want    bool
	}{
		{"json", map[string]string{"Content-Type": "application/json"}, true},
		{"xml", map[string]string{"Content-Type": "application/xml"}, true},
		{"text", map[string]string{"Content-Type": "text/plain"}, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := route.MatchesWithHeaders("/test", "GET", tt.headers)
			if got != tt.want {
				t.Errorf("got %v, want %v", got, tt.want)
			}
		})
	}
}

func TestGetRoute(t *testing.T) {
	cfg := &Config{
		Routes: []Route{
			{Name: "exact", Match: RouteMatch{PathExact: "/health"}, Policy: RoutePolicy{Kind: "public"}},
			{Name: "api", Match: RouteMatch{PathPrefix: "/api/"}, Policy: RoutePolicy{Kind: "capability"}},
			{Name: "catch-all", Match: RouteMatch{PathPrefix: "/"}, Policy: RoutePolicy{Kind: "public"}},
		},
	}
	tests := []struct {
		path     string
		method   string
		wantName string
		wantNil  bool
	}{
		{"/health", "GET", "exact", false},
		{"/api/users", "GET", "api", false},
		{"/other", "GET", "catch-all", false},
	}
	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			route := cfg.GetRoute(tt.path, tt.method)
			if tt.wantNil {
				if route != nil {
					t.Errorf("expected nil route, got %v", route.Name)
				}
				return
			}
			if route == nil {
				t.Fatal("expected route, got nil")
			}
			if route.Name != tt.wantName {
				t.Errorf("expected route %q, got %q", tt.wantName, route.Name)
			}
		})
	}
}

func TestGetRouteWithHeaders(t *testing.T) {
	cfg := &Config{
		Routes: []Route{
			{
				Name: "json-api",
				Match: RouteMatch{
					PathPrefix: "/api/",
					Headers:    map[string]string{"Accept": "application/json"},
				},
				Policy: RoutePolicy{Kind: "public"},
			},
			{
				Name:   "api-fallback",
				Match:  RouteMatch{PathPrefix: "/api/"},
				Policy: RoutePolicy{Kind: "public"},
			},
		},
	}

	// With matching header → first route
	route := cfg.GetRouteWithHeaders("/api/test", "GET", map[string]string{"Accept": "application/json"})
	if route == nil || route.Name != "json-api" {
		t.Errorf("expected 'json-api', got %v", route)
	}

	// Without header → fallback
	route = cfg.GetRouteWithHeaders("/api/test", "GET", map[string]string{})
	if route == nil || route.Name != "api-fallback" {
		t.Errorf("expected 'api-fallback', got %v", route)
	}
}

// --- Policy normalization during validation ---

func TestValidate_PolicyNormalization(t *testing.T) {
	tests := []struct {
		inputKind    string
		expectedKind string
	}{
		{"protect", "capability"},
		{"protected", "capability"},
		{"audit", "chargeback"},
		{"observe", "chargeback"},
	}
	for _, tt := range tests {
		t.Run(tt.inputKind, func(t *testing.T) {
			yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: normalized
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: ` + tt.inputKind + `
`)
			cfg, err := ParseYAML(yaml)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if cfg.Routes[0].Policy.Kind != tt.expectedKind {
				t.Errorf("expected normalized kind %q, got %q", tt.expectedKind, cfg.Routes[0].Policy.Kind)
			}
		})
	}
}

func TestValidate_ChargebackSynthesizesPay(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: audit-route
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: audit
`)
	cfg, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if cfg.Routes[0].Policy.Pay == nil {
		t.Error("expected synthesized Pay policy for chargeback")
	}
	if cfg.Routes[0].Policy.Pay.Mode != "chargeback" {
		t.Errorf("expected mode 'chargeback', got %q", cfg.Routes[0].Policy.Pay.Mode)
	}
}

// --- isPrivateIP tests ---

func TestIsPrivateIP(t *testing.T) {
	tests := []struct {
		host    string
		private bool
	}{
		{"127.0.0.1", true},
		{"10.0.0.1", true},
		{"172.16.0.1", true},
		{"192.168.1.1", true},
		{"169.254.1.1", true},  // link-local
		{"8.8.8.8", false},     // public
		{"1.1.1.1", false},     // public
		{"example.com", false}, // DNS name, not IP
	}
	for _, tt := range tests {
		t.Run(tt.host, func(t *testing.T) {
			got := isPrivateIP(tt.host)
			if got != tt.private {
				t.Errorf("isPrivateIP(%q) = %v, want %v", tt.host, got, tt.private)
			}
		})
	}
}

// --- parseURL tests ---

func TestParseURL(t *testing.T) {
	tests := []struct {
		url     string
		scheme  string
		host    string
		hasUser bool
		wantErr bool
	}{
		{"https://api.example.com", "https", "api.example.com", false, false},
		{"http://api.example.com:8080/path", "http", "api.example.com:8080", false, false},
		{"https://user:pass@api.example.com", "https", "api.example.com", true, false},
		{"noscheme", "", "", false, true},
	}
	for _, tt := range tests {
		t.Run(tt.url, func(t *testing.T) {
			parsed, err := parseURL(tt.url)
			if tt.wantErr {
				if err == nil {
					t.Error("expected error")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if parsed.Scheme != tt.scheme {
				t.Errorf("expected scheme %q, got %q", tt.scheme, parsed.Scheme)
			}
			if parsed.Host != tt.host {
				t.Errorf("expected host %q, got %q", tt.host, parsed.Host)
			}
			if tt.hasUser && parsed.User == nil {
				t.Error("expected user info")
			}
			if !tt.hasUser && parsed.User != nil {
				t.Error("did not expect user info")
			}
		})
	}
}

// --- Multiple upstreams and routes ---

func TestParseYAML_MultipleUpstreamsAndRoutes(t *testing.T) {
	yaml := []byte(`
upstreams:
  openai:
    url: https://api.openai.com
    timeout: 30s
    headers:
      Authorization: "Bearer sk-test"
  anthropic:
    url: https://api.anthropic.com
    timeout: 60s
routes:
  - name: openai-route
    match:
      pathPrefix: /openai/
    upstream: openai
    stripPrefix: true
    policy:
      kind: capability
  - name: anthropic-route
    match:
      pathPrefix: /anthropic/
    upstream: anthropic
    policy:
      kind: l402
      priceSats: 50
`)
	cfg, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(cfg.Upstreams) != 2 {
		t.Errorf("expected 2 upstreams, got %d", len(cfg.Upstreams))
	}
	if len(cfg.Routes) != 2 {
		t.Errorf("expected 2 routes, got %d", len(cfg.Routes))
	}
	if cfg.Routes[0].StripPrefix != true {
		t.Error("expected stripPrefix true on first route")
	}
	openai := cfg.Upstreams["openai"]
	if openai.Timeout != 30*time.Second {
		t.Errorf("expected openai timeout 30s, got %v", openai.Timeout)
	}
	if openai.Headers["Authorization"] != "Bearer sk-test" {
		t.Errorf("expected Authorization header, got %q", openai.Headers["Authorization"])
	}
}

// --- CompileRegex ---

func TestCompileRegex_EmptyIsNoop(t *testing.T) {
	route := &Route{Name: "no-regex", Match: RouteMatch{PathPrefix: "/"}}
	if err := route.CompileRegex(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCompileRegex_Invalid(t *testing.T) {
	route := &Route{Name: "bad-regex", Match: RouteMatch{PathRegex: "[invalid"}}
	if err := route.CompileRegex(); err == nil {
		t.Error("expected error for invalid regex")
	}
}

// --- Upstream with optional fields ---

func TestParseYAML_UpstreamWithHealthCheck(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
    healthCheck:
      path: /health
      interval: 10s
      timeout: 5s
    circuitBreaker:
      maxFailures: 5
      resetTimeout: 30s
      halfOpenMaxReqs: 2
routes:
  - name: r
    match:
      pathPrefix: /
    upstream: backend
    policy:
      kind: public
`)
	cfg, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	backend := cfg.Upstreams["backend"]
	if backend.HealthCheck == nil {
		t.Fatal("expected health check config")
	}
	if backend.HealthCheck.Path != "/health" {
		t.Errorf("expected health check path '/health', got %q", backend.HealthCheck.Path)
	}
	if backend.CircuitBreaker == nil {
		t.Fatal("expected circuit breaker config")
	}
	if backend.CircuitBreaker.MaxFailures != 5 {
		t.Errorf("expected maxFailures 5, got %d", backend.CircuitBreaker.MaxFailures)
	}
}

// --- Route with transform and rate limit ---

func TestParseYAML_RouteTransformAndRateLimit(t *testing.T) {
	yaml := []byte(`
upstreams:
  backend:
    url: https://api.example.com
routes:
  - name: transformed
    match:
      pathPrefix: /api/
    upstream: backend
    policy:
      kind: public
    transform:
      stripPrefix: /api
      addHeaders:
        X-Gateway: satgate
    rateLimit:
      requestsPerMinute: 100
      burstSize: 10
      key: ip
`)
	cfg, err := ParseYAML(yaml)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	route := cfg.Routes[0]
	if route.Transform == nil {
		t.Fatal("expected transform config")
	}
	if route.Transform.StripPrefix != "/api" {
		t.Errorf("expected stripPrefix '/api', got %q", route.Transform.StripPrefix)
	}
	if route.Transform.AddHeaders["X-Gateway"] != "satgate" {
		t.Errorf("expected X-Gateway header 'satgate'")
	}
	if route.RateLimit == nil {
		t.Fatal("expected rate limit config")
	}
	if route.RateLimit.RequestsPerMinute != 100 {
		t.Errorf("expected 100 rpm, got %d", route.RateLimit.RequestsPerMinute)
	}
}
