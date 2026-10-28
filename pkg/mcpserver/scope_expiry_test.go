package mcpserver

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"testing"
	"time"
)

// A scope word allows a tool when macaroon.ScopeGrants grants the word, read
// in the "mcp" namespace, the tool's scope "mcp:<tool>".
func TestMatchScope_WildcardStaysInNamespace(t *testing.T) {
	const tenant = "0f2b7c4e-1d3a-4b5c-8e9f-0a1b2c3d4e5f"
	for _, tc := range []struct {
		scope, tool string
		want        bool
	}{
		// Exact.
		{"search", "search", true},
		{"mcp:search", "search", true},
		{"search", "fetch", false},
		{"search,fetch", "fetch", true},
		{"db:query", "db:query", true},

		// Same-namespace wildcard.
		{"mcp:*", "search", true},
		{"mcp:*", "db:query", true},
		{"db:*", "db:query", true},
		{"mcp:db:*", "db:query", true},
		{"db:*", "db", false},
		{"db:*", "dbx:query", false},
		{"db:*", "db_query", false},

		// The HTTP API namespace never reaches a tool, and the reverse is
		// covered by TestScopeGrants in pkg/macaroon.
		{"api:*", "search", false},
		{"api:*", "api:read", false},
		{"api:read", "api:read", false},
		{"api:*,search", "search", true},

		// Bare wildcard and tenant prefixes.
		{"*", "search", true},
		{tenant + ":mcp:*", "search", true},
		{tenant + ":*", "search", true},
		{tenant + ":api:*", "search", false},

		// Empty and malformed.
		{"", "search", false},
		{",", "search", false},
		{"mcp:*", "", false},
		{"*", "", false},
		{"sea*", "search", false},
		{"mcp:sea*", "search", false},
		{"*:*", "search", false},
		{"mcp:*", "bad name", false},
		{"mcp:*", "a::b", false},
		{"mcp:*", "trailing:", false},
		{"mcp:*", "star*", false},
	} {
		if got := matchScope(tc.scope, tc.tool); got != tc.want {
			t.Errorf("matchScope(%q, %q) = %v, want %v", tc.scope, tc.tool, got, tc.want)
		}
	}
}

// A TokenInfo without caveats (a hand-built or non-macaroon identity) is read
// with the same rule: "api:*" allows no tool.
func TestAllowsTool_ScopeFieldWildcardStaysInNamespace(t *testing.T) {
	for _, tc := range []struct {
		scope string
		want  bool
	}{
		{"", true},
		{"*", true},
		{"mcp:*", true},
		{"search", true},
		{"api:*", false},
		{"api:read", false},
	} {
		info := &TokenInfo{Scope: tc.scope}
		if got := info.AllowsTool("search"); got != tc.want {
			t.Errorf("TokenInfo{Scope:%q}.AllowsTool(search) = %v, want %v", tc.scope, got, tc.want)
		}
	}
}

// The MCP server: an "api:*" token is admitted but calls no tool; an
// "mcp:*" token calls it.
func TestMCPServer_APIWildcardCallsNoTool(t *testing.T) {
	r := newLineageRig(t, nil)
	for _, tc := range []struct {
		scope string
		want  bool
	}{
		{"mcp:*", true},
		{"*", true},
		{"echo", true},
		{"api:*", false},
	} {
		mac, err := r.svc.Mint(tc.scope, time.Now().Add(time.Hour))
		if err != nil {
			t.Fatal(err)
		}
		tok := r.svc.Encode(mac)
		r.fund(tok, 5)
		before := streamableForwardCount(r.proxy)
		sid := mustInitSession(t, r.srv.URL, tok)
		_, _, body := streamablePost(t, r.srv.URL+"/mcp", sid, tok, toolCallBody(2, "echo"))
		forwarded := streamableForwardCount(r.proxy) > before
		if got := strings.Contains(body, `"result"`); got != tc.want || forwarded != tc.want {
			t.Errorf("scope %q: tool call allowed=%v forwarded=%v, want %v (body %s)", tc.scope, got, forwarded, tc.want, body)
		}
	}
}

// The MCP server verifies every expires caveat as an absolute time: a
// duration or any other non-absolute value refuses the token.
func TestMCPServer_ExpiresMustBeAbsolute(t *testing.T) {
	r := newLineageRig(t, nil)
	root, _ := r.mint()
	auth := &MacaroonAuthenticator{Service: r.svc}
	now := time.Now()
	for _, tc := range []struct {
		value string
		ok    bool
	}{
		{now.Add(time.Hour).UTC().Format(time.RFC3339), true},
		{strconv.FormatInt(now.Add(time.Hour).UnixMilli(), 10), true},
		{now.Add(-time.Hour).UTC().Format(time.RFC3339), false},
		{strconv.FormatInt(now.Add(-time.Hour).UnixMilli(), 10), false},
		{"24h", false},
		{"5m", false},
		{"soon", false},
		{"12abc", false},
		{"2026-11-01", false},
	} {
		tok := r.attenuate(root, "expires = "+tc.value)
		_, err := auth.Verify(context.Background(), tok)
		if (err == nil) != tc.ok {
			t.Errorf("expires = %q: Verify err = %v, want ok=%v", tc.value, err, tc.ok)
		}
		status, _, _ := streamablePost(t, r.srv.URL+"/mcp", "", tok, initializeBody("2025-03-26"))
		if (status == http.StatusOK) != tc.ok {
			t.Errorf("expires = %q: initialize status %d, want ok=%v", tc.value, status, tc.ok)
		}
	}
}

// The root token the proxy mints for itself is scoped to MCP tools.
func TestProxyAutoMintRootIsScopedToTools(t *testing.T) {
	cfg := &Config{
		Server: ServerConfig{Transport: "stdio", Name: "test", Version: "1.0"},
		Auth: AuthConfig{
			Mode:         "header",
			RootKey:      "test-root-key",
			AutoMintRoot: true,
		},
		Upstreams: map[string]UpstreamConfig{
			"mock": {Transport: "stdio", Command: []string{"python3", "-c", "import sys; sys.exit(0)"}},
		},
		Budget:      BudgetConfig{Backend: "memory", Limit: 500, FailMode: "closed"},
		Enforcement: EnforcementConfig{Mode: "hard"},
		Logging:     LoggingConfig{Level: "error"},
	}
	cfg.applyDefaults()
	var proxy *Proxy
	captureStderr(t, func() {
		var err error
		if proxy, err = New(cfg); err != nil {
			t.Fatalf("create proxy: %v", err)
		}
	})
	info, err := proxy.auth.Verify(context.Background(), proxy.RootToken())
	if err != nil {
		t.Fatal(err)
	}
	if info.Scope != "mcp:*" || !info.AllowsTool("search") {
		t.Fatalf("root token scope %q, allows search %v; want mcp:* allowing every tool", info.Scope, info.AllowsTool("search"))
	}
}
