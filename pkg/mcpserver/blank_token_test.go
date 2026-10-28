package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

// A static token (auth.mode "config") that is empty after trimming Unicode
// whitespace is no token: it is refused in configuration, and a presented
// one is treated as missing.

// blankTokenValues are values that are empty after trimming Unicode
// whitespace.
var blankTokenValues = []string{" ", "   ", "\t", "\n", " \t\r\n ", "\u00a0", "\u3000", "\u2003\u2028", "\u0085"}

const wantAuthTokenRequired = `auth.token is required for auth.mode "config" and must not be empty or only whitespace`

// LoadConfig refuses auth.mode "config" with no auth.token, an empty one or
// one of only whitespace, on every transport, including a value that comes
// from an environment variable.
func TestLoadConfigConfigModeRefusesBlankToken(t *testing.T) {
	t.Setenv("SATGATE_BLANK_TOKEN_TEST", " \t ")
	tokenLines := []string{
		"",
		"  token: \"\"\n",
		"  token: \"   \"\n",
		"  token: \"\\t\\n\"\n",
		"  token: \"\\u00a0\\u3000\"\n",
		"  token: \"${SATGATE_BLANK_TOKEN_TEST}\"\n",
		"  token: ${SATGATE_UNSET_BLANK_TOKEN_TEST}\n",
	}
	for _, transport := range []string{"stdio", "sse", "http"} {
		for _, line := range tokenLines {
			body := "server:\n  transport: " + transport + "\nauth:\n  mode: config\n" + line + testUpstreamYAML
			_, err := LoadConfig(writeConfig(t, body))
			if err == nil || !strings.Contains(err.Error(), wantAuthTokenRequired) || !strings.HasPrefix(err.Error(), "invalid config: ") {
				t.Fatalf("%s %q: err=%v, want the auth.token configuration error", transport, line, err)
			}
		}
		cfg, err := LoadConfig(writeConfig(t, "server:\n  transport: "+transport+"\nauth:\n  mode: config\n  token: \""+staticTestToken+"\"\n"+testUpstreamYAML))
		if err != nil || cfg.Auth.Token != staticTestToken {
			t.Fatalf("%s with a token: %v", transport, err)
		}
	}
}

// Header and none modes do not read auth.token, so a blank one changes
// nothing for them.
func TestLoadConfigBlankTokenIgnoredInHeaderAndNoneModes(t *testing.T) {
	for _, auth := range []string{
		"auth:\n  mode: header\n  rootKey: blank-token-test-root\n  token: \"   \"\n",
		"auth:\n  mode: none\n  token: \"   \"\n",
		"auth:\n  mode: none\n",
	} {
		for _, transport := range []string{"stdio", "sse"} {
			if _, err := LoadConfig(writeConfig(t, "server:\n  transport: "+transport+"\n"+auth+testUpstreamYAML)); err != nil {
				t.Fatalf("%s %q: %v", transport, auth, err)
			}
		}
	}
	for _, c := range []*Config{
		{Server: ServerConfig{Transport: "sse"}, Auth: AuthConfig{Mode: "header", Token: "  "}},
		{Server: ServerConfig{Transport: "sse"}, Auth: AuthConfig{Mode: "none", Token: "  "}},
	} {
		if err := c.networkAuthError(); err != nil {
			t.Fatalf("mode %s: %v", c.Auth.Mode, err)
		}
	}
}

// NewAuthenticator (and so New) refuses a blank static token with the same
// configuration error.
func TestNewAuthenticatorConfigModeRefusesBlankToken(t *testing.T) {
	for _, tok := range append([]string{""}, blankTokenValues...) {
		if _, err := NewAuthenticator(AuthConfig{Mode: "config", Token: tok}); err == nil || !strings.Contains(err.Error(), wantAuthTokenRequired) {
			t.Fatalf("NewAuthenticator %q: %v", tok, err)
		}
		cfg := &Config{Server: ServerConfig{Transport: "sse"}, Auth: AuthConfig{Mode: "config", Token: tok}}
		if _, err := New(cfg); err == nil || !strings.Contains(err.Error(), wantAuthTokenRequired) {
			t.Fatalf("New %q: %v", tok, err)
		}
	}
	if _, err := NewAuthenticator(AuthConfig{Mode: "config", Token: staticTestToken}); err != nil {
		t.Fatal(err)
	}
}

// blankConfigTokenServer is a config-mode server whose auth.token was set to
// blank in code after New: the listener and the handler checks see it.
func blankConfigTokenServer(t *testing.T, blank string) *embeddedProbe {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "blank-token-test", Version: "test"},
		Auth:        AuthConfig{Mode: "config", Token: staticTestToken},
		Budget:      BudgetConfig{Limit: 100, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	p.config.Auth.Token = blank
	e := &embeddedProbe{router: &countingForwardRouter{}}
	p.SetUpstreamRouter(e.router)
	p.SetToolsListEnricher(func(_ context.Context, _ string, tools []json.RawMessage) []json.RawMessage {
		e.enriched.Add(1)
		return tools
	})
	e.server = NewSSEServer(p, "127.0.0.1:0")
	e.server.HandleFunc("/custom", func(w http.ResponseWriter, _ *http.Request) {
		e.custom.Add(1)
		w.WriteHeader(http.StatusOK)
	})
	e.server.Handle("/custom-handler", http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		e.custom.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	return e
}

// The embedded handler refuses every route with 503 and the auth.token
// error when auth.mode is "config" and auth.token is blank, with or without
// a bearer, and logs the refusal once.
func TestEmbeddedHandlerRefusesBlankConfigToken(t *testing.T) {
	for _, blank := range []string{"", "   ", "\t\n", "\u00a0"} {
		logs := captureLogs(t)
		e := blankConfigTokenServer(t, blank)
		srv := httptest.NewServer(e.server.Handler())
		for _, bearer := range []string{"", staticTestToken, "other-value"} {
			for _, r := range embeddedRequests("") {
				status, body := doEmbedded(t, srv.URL, r, bearer)
				if status != http.StatusServiceUnavailable || !strings.Contains(body, wantAuthTokenRequired) {
					t.Fatalf("blank %q %s (bearer=%v): status %d body %q, want 503 auth.token refusal", blank, r.name, bearer != "", status, body)
				}
			}
		}
		srv.Close()
		sse, stream := e.sessionCounts()
		if e.router.calls != 0 || e.enriched.Load() != 0 || e.custom.Load() != 0 || sse != 0 || stream != 0 {
			t.Fatalf("blank %q: refused requests reached the server: upstream=%d tools/list=%d caller routes=%d sse=%d stream=%d",
				blank, e.router.calls, e.enriched.Load(), e.custom.Load(), sse, stream)
		}
		if n := countLines(logs, `"level":"error"`, "auth.token is required"); n != 1 {
			t.Fatalf("blank %q: %d refusal log lines, want 1", blank, n)
		}
	}
}

// ListenAndServe refuses to start with the same error the handler answers.
func TestListenAndServeRefusesBlankConfigToken(t *testing.T) {
	captureLogs(t)
	e := blankConfigTokenServer(t, "   ")
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	err := e.server.ListenAndServe(ctx)
	if err == nil || !strings.Contains(err.Error(), wantAuthTokenRequired) {
		t.Fatalf("ListenAndServe: %v, want the auth.token error", err)
	}
	rec := httptest.NewRecorder()
	e.server.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/health", nil))
	if rec.Code != http.StatusServiceUnavailable || strings.TrimSpace(rec.Body.String()) != err.Error() {
		t.Fatalf("handler refusal %d %q differs from listener error %q", rec.Code, rec.Body.String(), err.Error())
	}
}

// The static authenticator treats a blank presented token as missing, and
// a blank configured token matches nothing, before any compare. The exact
// token still verifies.
func TestStaticTokenAuthenticatorRefusesBlankTokens(t *testing.T) {
	ctx := context.Background()
	a := &StaticTokenAuthenticator{Token: staticTestToken, TokenID: hashToken(staticTestToken)}
	for _, blank := range blankTokenValues {
		if _, err := a.Verify(ctx, blank); err == nil || err.Error() != "token required" {
			t.Fatalf("presented %q: %v, want token required", blank, err)
		}
		cfgBlank := &StaticTokenAuthenticator{Token: blank, TokenID: hashToken(blank)}
		for _, presented := range []string{blank, staticTestToken, "x"} {
			if info, err := cfgBlank.Verify(ctx, presented); err == nil || info != nil {
				t.Fatalf("configured %q, presented %q: accepted", blank, presented)
			}
		}
	}
	if info, err := a.Verify(ctx, staticTestToken); err != nil || info.TokenID != hashToken(staticTestToken) {
		t.Fatalf("exact token: %+v %v", info, err)
	}
}

// In config and header modes a params._meta.token that is blank is a
// missing token: the request gets the missing-token refusal and reaches no
// upstream. The correct token is still accepted.
func TestBlankMetaTokenIsMissingToken(t *testing.T) {
	ctx := context.Background()
	for _, mode := range []string{"config", "header"} {
		p, router := newAuthModeProxy(t, mode)
		for _, blank := range blankTokenValues {
			tokJSON, _ := json.Marshal(blank)
			for _, tc := range []struct{ method, params string }{
				{MethodToolsCall, `{"name":"echo","arguments":{},"_meta":{"token":` + string(tokJSON) + `}}`},
				{MethodToolsList, `{"_meta":{"token":` + string(tokJSON) + `}}`},
				{MethodSatGateBudget, `{"_meta":{"token":` + string(tokJSON) + `}}`},
				{"resources/list", `{"_meta":{"token":` + string(tokJSON) + `}}`},
			} {
				resp, err := p.handleRequest(ctx, rpcRequest(tc.method, tc.params))
				wantAuthRefusal(t, mode+" "+tc.method+" "+strings.ReplaceAll(tc.params, staticTestToken, "<t>"), resp, err, "authentication required: no token provided")
			}
		}
		if router.calls != 0 {
			t.Fatalf("%s: blank tokens reached the upstream %d times", mode, router.calls)
		}
	}
	p, router := newAuthModeProxy(t, "config")
	resp, err := p.handleRequest(ctx, rpcRequest(MethodToolsCall, `{"name":"echo","arguments":{},"_meta":{"token":"`+staticTestToken+`"}}`))
	if err != nil || resp == nil || resp.Error != nil || router.calls != 1 {
		t.Fatalf("correct token: %+v %v calls=%d", resp, err, router.calls)
	}
}

// blankAuthorizationValues are Authorization header values that carry no
// credential once whitespace is trimmed. Over a real connection the server
// drops leading and trailing spaces and tabs, so "Bearer   " arrives as
// "Bearer"; other Unicode whitespace arrives as sent.
var blankAuthorizationValues = []string{"Bearer ", "Bearer    ", "Bearer \t", "Bearer", "Bearer \u00a0", "Bearer \u3000\u2003", "\u00a0"}

func postWithAuthorization(t *testing.T, url, sid, authorization, body string) (int, http.Header, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	if sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}
	if authorization != "" {
		req.Header.Set("Authorization", authorization)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var b strings.Builder
	_, _ = bufio.NewReader(resp.Body).WriteTo(&b)
	return resp.StatusCode, resp.Header, b.String()
}

const blankInitBody = `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`

// Streamable HTTP in config mode: initialize with a blank Authorization
// header (over a connection, and given to the handler directly so the value
// is not trimmed on the way) or a blank ?token= value gets the
// missing-token 401 and no session. The correct token still opens a
// session, and a request with the correct header is served even when its
// params carry a blank _meta.token.
func TestStreamableHTTPBlankTokenIsMissingToken(t *testing.T) {
	p, router := newAuthModeProxy(t, "config")
	p.config.Server.AllowQueryToken = true
	sse := NewSSEServer(p, ":0")
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()

	for _, authz := range blankAuthorizationValues {
		status, hdr, body := postWithAuthorization(t, srv.URL+"/mcp", "", authz, blankInitBody)
		if status != http.StatusUnauthorized || hdr.Get("Mcp-Session-Id") != "" || strings.TrimSpace(body) != "authentication required" {
			t.Fatalf("connection, Authorization %q: initialize %d sid=%q body=%q", authz, status, hdr.Get("Mcp-Session-Id"), body)
		}
	}
	for _, blank := range blankTokenValues {
		for _, authz := range []string{"Bearer " + blank, blank} {
			req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(blankInitBody))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Accept", "application/json, text/event-stream")
			req.Header.Set("MCP-Protocol-Version", "2025-03-26")
			req.Header["Authorization"] = []string{authz}
			rec := httptest.NewRecorder()
			sse.ServeHTTP(rec, req)
			if rec.Code != http.StatusUnauthorized || rec.Header().Get("Mcp-Session-Id") != "" || strings.TrimSpace(rec.Body.String()) != "authentication required" {
				t.Fatalf("handler, Authorization %q: initialize %d body=%q", authz, rec.Code, rec.Body.String())
			}
		}
	}
	for _, q := range []string{"%20", "%20%20%20", "%09", "%C2%A0", "%E3%80%80"} {
		status, _, body := postWithAuthorization(t, srv.URL+"/mcp?token="+q, "", "", blankInitBody)
		if status != http.StatusUnauthorized || strings.TrimSpace(body) != "authentication required" {
			t.Fatalf("?token=%s: initialize %d body=%q", q, status, body)
		}
	}
	if n := streamSessionCount(sse); n != 0 {
		t.Fatalf("blank tokens left %d sessions", n)
	}

	sid := mustInitSession(t, srv.URL, staticTestToken)
	for _, blank := range blankTokenValues {
		tokJSON, _ := json.Marshal(blank)
		resp := streamableCall(t, srv.URL, sid, staticTestToken, rpcBody(MethodToolsCall, `{"name":"echo","arguments":{},"_meta":{"token":`+string(tokJSON)+`}}`))
		if resp.Error != nil {
			t.Fatalf("correct header, blank _meta.token %q: %+v", blank, resp.Error)
		}
	}
	if router.calls != len(blankTokenValues) {
		t.Fatalf("upstream calls=%d, want %d", router.calls, len(blankTokenValues))
	}
}

// SSE in config mode: messages with a blank Authorization header, or with
// no header and a blank params._meta.token, get the missing-token refusal
// and reach no upstream; a connect with a blank header gets the
// missing-token challenge when the challenge is on. The correct header is
// still served, also next to a blank _meta.token.
func TestSSEBlankTokenIsMissingToken(t *testing.T) {
	p, router := newAuthModeProxy(t, "config")
	srv := httptest.NewServer(NewSSEServer(p, ":0").Handler())
	defer srv.Close()
	conn := eaOpenSSE(t, srv.URL, "")
	defer conn.body.Close()

	post := func(authorization, body string) {
		t.Helper()
		req, _ := http.NewRequest(http.MethodPost, srv.URL+"/message?sessionId="+conn.session, strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		if authorization != "" {
			req.Header.Set("Authorization", authorization)
		}
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		resp.Body.Close()
		if resp.StatusCode != http.StatusAccepted {
			t.Fatalf("POST /message: %d", resp.StatusCode)
		}
	}
	wantMissing := func(label string) {
		t.Helper()
		if r := conn.next(); r.Error == nil || r.Error.Code != CodePolicyDenied || r.Error.Message != "authentication required: no token provided" {
			t.Fatalf("%s: %+v", label, r.Error)
		}
	}

	for _, authz := range blankAuthorizationValues {
		for _, body := range []string{rpcBody(MethodToolsList, ""), rpcBody(MethodToolsCall, `{"name":"echo","arguments":{}}`), rpcBody("resources/list", "")} {
			post(authz, body)
			wantMissing("Authorization " + authz)
		}
	}
	for _, blank := range blankTokenValues {
		tokJSON, _ := json.Marshal(blank)
		post("", rpcBody(MethodToolsCall, `{"name":"echo","arguments":{},"_meta":{"token":`+string(tokJSON)+`}}`))
		wantMissing("_meta.token " + string(tokJSON))
	}
	if router.calls != 0 {
		t.Fatalf("blank tokens reached the upstream %d times", router.calls)
	}

	post("Bearer "+staticTestToken, rpcBody(MethodToolsCall, `{"name":"echo","arguments":{},"_meta":{"token":"   "}}`))
	if r := conn.next(); r.Error != nil || router.calls != 1 {
		t.Fatalf("correct header, blank _meta.token: %+v calls=%d", r.Error, router.calls)
	}
	post("Bearer "+staticTestToken, rpcBody(MethodToolsList, ""))
	if r := conn.next(); r.Error != nil || !strings.Contains(string(r.Result), `"echo"`) {
		t.Fatalf("correct header tools/list: %+v", r.Error)
	}

	chal := httptest.NewServer(NewSSEServer(p, ":0", WithOAuthChallenge(testMetadataURL)).Handler())
	defer chal.Close()
	for _, authz := range blankAuthorizationValues {
		req, _ := http.NewRequest(http.MethodGet, chal.URL+"/sse", nil)
		req.Header.Set("Authorization", authz)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatal(err)
		}
		var b strings.Builder
		_, _ = bufio.NewReader(resp.Body).WriteTo(&b)
		resp.Body.Close()
		if resp.StatusCode != http.StatusUnauthorized || strings.TrimSpace(b.String()) != "authentication required" {
			t.Fatalf("connect with Authorization %q: %d %q", authz, resp.StatusCode, b.String())
		}
	}
}
