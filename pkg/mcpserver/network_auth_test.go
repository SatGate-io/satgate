package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
	"time"

	"gopkg.in/yaml.v3"
)

// Static-token mode ("config"), header mode and tools/list over SSE and
// Streamable HTTP; the startup defaults of LoadConfig and the listener.

// staticTestToken is a fixture value, not a credential.
const staticTestToken = "static-fixture-value-0001"

func newAuthModeProxy(t *testing.T, mode string) (*Proxy, *countingForwardRouter) {
	t.Helper()
	auth := AuthConfig{Mode: mode}
	switch mode {
	case "config":
		auth.Token = staticTestToken
	case "header":
		auth.RootKey = "network-auth-test-root"
	}
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "auth-test", Version: "test"},
		Auth:        auth,
		Budget:      BudgetConfig{Limit: 100, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if mode == "header" {
		p.SetAuthenticator(fixedTokenAuth{good: "good-token"})
	}
	router := &countingForwardRouter{}
	p.SetUpstreamRouter(router)
	return p, router
}

func rpcRequest(method, params string) *Request {
	r := &Request{JSONRPC: "2.0", ID: json.RawMessage(`5`), Method: method}
	if params != "" {
		r.Params = json.RawMessage(params)
	}
	return r
}

func wantAuthRefusal(t *testing.T, label string, resp *Response, err error, wantMsg string) {
	t.Helper()
	if err != nil {
		t.Fatalf("%s: handler error %v", label, err)
	}
	if resp == nil || resp.Error == nil {
		t.Fatalf("%s: not refused: %+v", label, resp)
	}
	if resp.Error.Code != CodePolicyDenied || resp.Error.Message != wantMsg {
		t.Fatalf("%s: refusal code=%d message=%q, want %d %q", label, resp.Error.Code, resp.Error.Message, CodePolicyDenied, wantMsg)
	}
}

// Mode config: a request over the network with no token, or an empty one,
// is refused with exactly the answer header mode gives.
func TestConfigModeRefusesMissingTokenLikeHeaderMode(t *testing.T) {
	header, _ := newAuthModeProxy(t, "header")
	cfgMode, router := newAuthModeProxy(t, "config")
	ctx := context.Background()

	for _, tc := range []struct{ method, params string }{
		{MethodToolsCall, `{"name":"echo","arguments":{}}`},
		{MethodToolsCall, `{"name":"echo","arguments":{},"_meta":{"token":""}}`},
		{MethodSatGateBudget, `{}`},
		{MethodToolsList, `{}`},
		{MethodToolsList, ``},
		{"resources/list", `{}`},
	} {
		label := tc.method + " " + tc.params
		hResp, hErr := header.handleRequest(ctx, rpcRequest(tc.method, tc.params))
		wantAuthRefusal(t, "header "+label, hResp, hErr, "authentication required: no token provided")
		cResp, cErr := cfgMode.handleRequest(ctx, rpcRequest(tc.method, tc.params))
		wantAuthRefusal(t, "config "+label, cResp, cErr, "authentication required: no token provided")
		hb, _ := json.Marshal(hResp)
		cb, _ := json.Marshal(cResp)
		if !bytes.Equal(hb, cb) {
			t.Fatalf("%s: config refusal %s differs from header refusal %s", label, cb, hb)
		}
	}
	if router.calls != 0 {
		t.Fatalf("refused requests reached the upstream %d times", router.calls)
	}
}

// Mode config: the configured token works as before, a wrong one is refused.
func TestConfigModeStaticTokenAcceptedWrongTokenRefused(t *testing.T) {
	p, router := newAuthModeProxy(t, "config")
	ctx := context.Background()

	resp, err := p.handleRequest(ctx, rpcRequest(MethodToolsCall, `{"name":"echo","arguments":{},"_meta":{"token":"`+staticTestToken+`"}}`))
	if err != nil || resp == nil || resp.Error != nil {
		t.Fatalf("correct static token refused: %+v %v", resp, err)
	}
	if router.calls != 1 {
		t.Fatalf("correct token: upstream calls=%d, want 1", router.calls)
	}
	resp, err = p.handleRequest(ctx, rpcRequest(MethodToolsList, `{"_meta":{"token":"`+staticTestToken+`"}}`))
	if err != nil || resp == nil || resp.Error != nil || !strings.Contains(string(resp.Result), `"echo"`) {
		t.Fatalf("correct static token: tools/list %+v %v", resp, err)
	}

	for _, bad := range []string{"x", staticTestToken + "x", staticTestToken[:len(staticTestToken)-1], strings.ToUpper(staticTestToken)} {
		for _, method := range []string{MethodToolsCall, MethodToolsList} {
			resp, err := p.handleRequest(ctx, rpcRequest(method, `{"name":"echo","_meta":{"token":"`+bad+`"}}`))
			wantAuthRefusal(t, method+" wrong token", resp, err, "authentication failed: invalid token")
		}
	}
	if router.calls != 1 {
		t.Fatalf("wrong tokens reached the upstream: calls=%d", router.calls)
	}
}

// The static comparison is over fixed-length digests: a value of any length
// or with any common prefix is refused, and only the exact token passes.
func TestStaticTokenAuthenticatorComparesExactValue(t *testing.T) {
	a := &StaticTokenAuthenticator{Token: staticTestToken, TokenID: hashToken(staticTestToken)}
	ctx := context.Background()
	if info, err := a.Verify(ctx, staticTestToken); err != nil || info.TokenID != hashToken(staticTestToken) || info.Scope != "*" {
		t.Fatalf("exact token: %+v %v", info, err)
	}
	if _, err := a.Verify(ctx, ""); err == nil || err.Error() != "token required" {
		t.Fatalf("empty token: %v", err)
	}
	for _, bad := range []string{"s", staticTestToken[:10], staticTestToken + "\x00", " " + staticTestToken, strings.Repeat("a", 4096)} {
		if _, err := a.Verify(ctx, bad); err == nil || err.Error() != "invalid token" {
			t.Fatalf("%q: %v", bad, err)
		}
	}
	empty := &StaticTokenAuthenticator{}
	if _, err := empty.Verify(ctx, "anything"); err == nil {
		t.Fatal("an authenticator with no token accepted a value")
	}
}

// tools/list needs a token that verifies in header and config modes, and
// stays open in mode none.
func TestToolsListRequiresAuthenticationInHeaderAndConfigModes(t *testing.T) {
	ctx := context.Background()
	for mode, good := range map[string]string{"header": "good-token", "config": staticTestToken} {
		p, _ := newAuthModeProxy(t, mode)
		resp, err := p.handleRequest(ctx, rpcRequest(MethodToolsList, `{}`))
		wantAuthRefusal(t, mode+" no token", resp, err, "authentication required: no token provided")
		resp, err = p.handleRequest(ctx, rpcRequest(MethodToolsList, `{"_meta":{"token":"not-it"}}`))
		if err != nil || resp == nil || resp.Error == nil || resp.Error.Code != CodePolicyDenied || !strings.HasPrefix(resp.Error.Message, "authentication failed:") {
			t.Fatalf("%s bad token: %+v %v", mode, resp, err)
		}
		resp, err = p.handleRequest(ctx, rpcRequest(MethodToolsList, `{"_meta":{"token":"`+good+`"}}`))
		if err != nil || resp == nil || resp.Error != nil || !strings.Contains(string(resp.Result), `"echo"`) {
			t.Fatalf("%s good token: %+v %v", mode, resp, err)
		}
	}
	p, _ := newAuthModeProxy(t, "none")
	resp, err := p.handleRequest(ctx, rpcRequest(MethodToolsList, `{}`))
	if err != nil || resp == nil || resp.Error != nil || !strings.Contains(string(resp.Result), `"echo"`) {
		t.Fatalf("mode none tools/list: %+v %v", resp, err)
	}
}

// A revoked token lists nothing.
func TestToolsListRefusesRevokedToken(t *testing.T) {
	p, _ := newAuthModeProxy(t, "header")
	p.SetRevocationChecker(revokeAll{})
	resp, err := p.handleRequest(context.Background(), rpcRequest(MethodToolsList, `{"_meta":{"token":"good-token"}}`))
	wantAuthRefusal(t, "revoked", resp, err, "token revoked")
}

// stdio is unaffected: a request read by Run in config mode with no token
// runs under the configured identity, and tools/list stays open over stdio
// in header and config modes.
func TestStdioUnaffectedByNetworkTokenRules(t *testing.T) {
	for _, mode := range []string{"config", "header"} {
		p, router := newAuthModeProxy(t, mode)
		client, server := newPipePair()
		ctx, cancel := context.WithCancel(context.Background())
		done := make(chan struct{})
		go func() { defer close(done); _ = p.Run(ctx, server) }()

		call := func(id int, method, params string) *Response {
			t.Helper()
			msg := `{"jsonrpc":"2.0","id":` + itoa(id) + `,"method":"` + method + `","params":` + params + `}`
			if err := client.WriteMessage(ctx, json.RawMessage(msg)); err != nil {
				t.Fatal(err)
			}
			rctx, rcancel := context.WithTimeout(ctx, 10*time.Second)
			defer rcancel()
			raw, err := client.ReadMessage(rctx)
			if err != nil {
				t.Fatalf("%s %s: %v", mode, method, err)
			}
			var resp Response
			if err := json.Unmarshal(raw, &resp); err != nil {
				t.Fatal(err)
			}
			return &resp
		}

		if resp := call(1, MethodToolsList, `{}`); resp.Error != nil || !strings.Contains(string(resp.Result), `"echo"`) {
			t.Fatalf("%s stdio tools/list without token: %+v", mode, resp)
		}
		resp := call(2, MethodToolsCall, `{"name":"echo","arguments":{}}`)
		switch mode {
		case "config":
			if resp.Error != nil || router.calls != 1 {
				t.Fatalf("config stdio tools/call without token: %+v calls=%d", resp, router.calls)
			}
		case "header":
			// Header mode over stdio refuses a tokenless call, as before.
			if resp.Error == nil || resp.Error.Message != "authentication required: no token provided" {
				t.Fatalf("header stdio tools/call without token: %+v", resp)
			}
		}
		cancel()
		_ = server.Close()
		_ = client.Close()
		<-done
	}
}

func itoa(i int) string { b, _ := json.Marshal(i); return string(b) }

// Streamable HTTP in config mode: initialize needs the static token (401
// otherwise, no session); with it, tools/list and tools/call work.
func TestStreamableHTTPConfigModeRequiresToken(t *testing.T) {
	p, router := newAuthModeProxy(t, "config")
	sse := NewSSEServer(p, ":0")
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()

	for tok, want := range map[string]string{"": "authentication required", "wrong-value": "authentication failed"} {
		status, hdr, body := streamablePost(t, srv.URL+"/mcp", "", tok, initializeBody("2025-03-26"))
		if status != http.StatusUnauthorized || hdr.Get("Mcp-Session-Id") != "" || strings.TrimSpace(body) != want {
			t.Fatalf("token %q: initialize status=%d sid=%q body=%q", tok, status, hdr.Get("Mcp-Session-Id"), body)
		}
	}
	if n := streamSessionCount(sse); n != 0 {
		t.Fatalf("refused initialize left %d sessions", n)
	}

	sid := mustInitSession(t, srv.URL, staticTestToken)
	status, _, body := streamablePost(t, srv.URL+"/mcp", sid, staticTestToken, toolsListBody())
	if status != http.StatusOK || !strings.Contains(body, `"echo"`) {
		t.Fatalf("tools/list with token: %d %s", status, body)
	}
	status, _, body = streamablePost(t, srv.URL+"/mcp", sid, staticTestToken, toolCallBody(3, "echo"))
	if status != http.StatusOK || strings.Contains(body, `"error"`) || router.calls != 1 {
		t.Fatalf("tools/call with token: %d %s calls=%d", status, body, router.calls)
	}
}

// SSE in config mode: a tokenless connect behaves as in header mode (the
// stream opens and nothing behind it answers without a token): tools/list,
// tools/call and forwarded methods sent without a token are refused on the
// stream; with the token they are answered.
func TestSSEConfigModeRefusesTokenlessMessages(t *testing.T) {
	p, router := newAuthModeProxy(t, "config")
	srv := httptest.NewServer(NewSSEServer(p, ":0").Handler())
	defer srv.Close()

	resp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	sid, reader := readEndpointEventBuffered(t, resp)

	next := func() Response {
		t.Helper()
		ch := make(chan string, 1)
		go func() {
			for {
				line, err := reader.ReadString('\n')
				if err != nil {
					close(ch)
					return
				}
				if strings.HasPrefix(line, "data: {") {
					ch <- strings.TrimPrefix(strings.TrimSpace(line), "data: ")
					return
				}
			}
		}()
		select {
		case data, ok := <-ch:
			if !ok {
				t.Fatal("stream closed")
			}
			var r Response
			if err := json.Unmarshal([]byte(data), &r); err != nil {
				t.Fatal(err)
			}
			return r
		case <-time.After(10 * time.Second):
			t.Fatal("no message event")
		}
		return Response{}
	}

	for _, msg := range []map[string]any{
		toolsListBody(),
		toolCallBody(8, "echo"),
		{"jsonrpc": "2.0", "id": 9, "method": "resources/list", "params": map[string]any{}},
	} {
		sendMessage(t, srv.URL, sid, "", msg)
		r := next()
		if r.Error == nil || r.Error.Code != CodePolicyDenied || r.Error.Message != "authentication required: no token provided" {
			t.Fatalf("tokenless %v: %+v", msg["method"], r)
		}
	}
	if router.calls != 0 {
		t.Fatalf("tokenless SSE messages reached the upstream %d times", router.calls)
	}

	sendMessage(t, srv.URL, sid, staticTestToken, toolsListBody())
	if r := next(); r.Error != nil || !strings.Contains(string(r.Result), `"echo"`) {
		t.Fatalf("tools/list with token: %+v", r)
	}
	sendMessage(t, srv.URL, sid, staticTestToken, toolCallBody(10, "echo"))
	if r := next(); r.Error != nil || router.calls != 1 {
		t.Fatalf("tools/call with token: %+v calls=%d", r, router.calls)
	}
}

// SSE connect in config mode with the authorization challenge on: refused
// without a token, as in header mode.
func TestSSEConnectConfigModeChallengeLikeHeaderMode(t *testing.T) {
	p, _ := newAuthModeProxy(t, "config")
	sse := NewSSEServer(p, ":0", WithOAuthChallenge(testMetadataURL))
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized || resp.Header.Get("WWW-Authenticate") != wantChallenge {
		t.Fatalf("tokenless sse connect: %d %q", resp.StatusCode, resp.Header.Get("WWW-Authenticate"))
	}
}

func writeConfig(t *testing.T, body string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "satgate-mcp.yaml")
	if err := os.WriteFile(path, []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	return path
}

const testUpstreamYAML = `
upstreams:
  mock:
    transport: stdio
    command: ["true"]
`

// A network transport with no auth.mode refuses to start, naming the
// setting; auth.mode none written out loads; stdio without auth loads as
// before (mode none).
func TestLoadConfigNetworkTransportRequiresAuthMode(t *testing.T) {
	for _, transport := range []string{"sse", "http"} {
		for _, authBlock := range []string{"", "auth:\n  token: some-value\n"} {
			_, err := LoadConfig(writeConfig(t, "server:\n  transport: "+transport+"\n"+authBlock+testUpstreamYAML))
			if err == nil || !strings.Contains(err.Error(), "auth.mode is required") || !strings.Contains(err.Error(), "auth.mode: none") {
				t.Fatalf("%s %q: err=%v", transport, authBlock, err)
			}
		}
		cfg, err := LoadConfig(writeConfig(t, "server:\n  transport: "+transport+"\nauth:\n  mode: none\n"+testUpstreamYAML))
		if err != nil || cfg.Auth.Mode != "none" {
			t.Fatalf("%s explicit none: %v", transport, err)
		}
	}
	for _, server := range []string{"", "server:\n  transport: stdio\n"} {
		cfg, err := LoadConfig(writeConfig(t, server+testUpstreamYAML))
		if err != nil || cfg.Auth.Mode != "none" || cfg.Server.Transport != "stdio" {
			t.Fatalf("stdio without auth: %+v %v", cfg, err)
		}
	}
}

// With a budget and no enforcement.mode, enforcement is control; with no
// budget it is observe; an explicit mode is kept.
func TestEnforcementDefaultsToControlWithBudget(t *testing.T) {
	for _, tc := range []struct{ extra, want string }{
		{"budget:\n  limit: 100\n", "control"},
		{"budget:\n  limit: 0\n", "observe"},
		{"", "observe"},
		{"budget:\n  limit: 100\nenforcement:\n  mode: observe\n", "observe"},
		{"budget:\n  limit: 100\nenforcement:\n  mode: shadow\n", "shadow"},
	} {
		cfg, err := LoadConfig(writeConfig(t, testUpstreamYAML+tc.extra))
		if err != nil {
			t.Fatal(err)
		}
		if cfg.Enforcement.Mode != tc.want {
			t.Fatalf("%q: enforcement %q, want %q", tc.extra, cfg.Enforcement.Mode, tc.want)
		}
	}
}

// Every shipped example, and the quick-start YAML of the guide and of the
// package README, sets auth.mode explicitly and loads.
func TestShippedConfigsSetAuthMode(t *testing.T) {
	root := filepath.Join("..", "..")
	examples, err := filepath.Glob(filepath.Join(root, "examples", "mcp-proxy", "*.yaml"))
	if err != nil || len(examples) == 0 {
		t.Fatalf("no examples: %v", err)
	}
	for _, path := range examples {
		if _, err := LoadConfig(path); err != nil {
			t.Fatalf("%s: %v", path, err)
		}
		raw, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		var parsed Config
		if err := yaml.Unmarshal(raw, &parsed); err != nil || parsed.Auth.Mode == "" {
			t.Fatalf("%s: no auth.mode (%v)", path, err)
		}
	}

	block := regexp.MustCompile("(?s)```yaml\n(.*?)```")
	for _, doc := range []string{
		filepath.Join(root, "docs", "guides", "mcp-gateway.md"),
		filepath.Join(root, "pkg", "mcpserver", "README.md"),
	} {
		raw, err := os.ReadFile(doc)
		if err != nil {
			t.Fatal(err)
		}
		text := string(raw)
		i := strings.Index(text, "## Quick Start")
		if i < 0 {
			t.Fatalf("%s: no Quick Start", doc)
		}
		m := block.FindStringSubmatch(text[i:])
		if m == nil {
			t.Fatalf("%s: no quick-start YAML", doc)
		}
		var parsed Config
		if err := yaml.Unmarshal([]byte(m[1]), &parsed); err != nil {
			t.Fatalf("%s quick start: %v", doc, err)
		}
		if parsed.Auth.Mode == "" {
			t.Fatalf("%s: the quick-start YAML sets no auth.mode", doc)
		}
	}
}
