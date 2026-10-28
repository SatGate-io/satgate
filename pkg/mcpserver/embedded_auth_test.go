package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// The handler returned by NewSSEServer (SSEServer.Handler, or the SSEServer
// itself as an http.Handler) follows the listener's rule: with no auth.mode
// written, every request is refused with the auth.mode error.

// countLines counts log lines that contain every one of parts.
func countLines(buf interface{ String() string }, parts ...string) int {
	n := 0
	for _, line := range strings.Split(buf.String(), "\n") {
		ok := line != ""
		for _, p := range parts {
			ok = ok && strings.Contains(line, p)
		}
		if ok {
			n++
		}
	}
	return n
}

// unsetModeConfigs are configurations that do not set auth.mode: built in
// code with an empty Auth.Mode, or loaded from a file without auth.mode
// (stdio defaults).
func unsetModeConfigs(t *testing.T) map[string]*Config {
	t.Helper()
	load := func(body string) *Config {
		cfg, err := LoadConfig(writeConfig(t, body))
		if err != nil {
			t.Fatalf("load: %v", err)
		}
		return cfg
	}
	loadedThenNone := load(testUpstreamYAML)
	loadedThenNone.Auth.Mode = "none"
	return map[string]*Config{
		"code/transport-empty":                {},
		"code/transport-sse":                  {Server: ServerConfig{Transport: "sse"}},
		"code/transport-http":                 {Server: ServerConfig{Transport: "http"}},
		"code/transport-stdio":                {Server: ServerConfig{Transport: "stdio"}},
		"loaded/stdio-defaults-no-auth-block": load(testUpstreamYAML),
		"loaded/auth-block-without-mode":      load("auth:\n  rootKey: embedded-test-root\n" + testUpstreamYAML),
		"loaded/explicit-transport-stdio":     load("server:\n  transport: stdio\n" + testUpstreamYAML),
		// LoadConfig fills in "none"; writing the same value back in code
		// does not count as choosing a mode.
		"loaded/none-filled-by-loader": loadedThenNone,
	}
}

// embeddedProbe counts everything a refused request must not reach.
type embeddedProbe struct {
	server   *SSEServer
	router   *countingForwardRouter
	enriched atomic.Int32
	custom   atomic.Int32
}

func newEmbeddedProbe(t *testing.T, cfg *Config, opts ...SSEOption) *embeddedProbe {
	t.Helper()
	p, err := New(cfg)
	if err != nil {
		t.Fatalf("New: %v", err)
	}
	e := &embeddedProbe{router: &countingForwardRouter{}}
	p.SetUpstreamRouter(e.router)
	p.SetToolsListEnricher(func(_ context.Context, _ string, tools []json.RawMessage) []json.RawMessage {
		e.enriched.Add(1)
		return tools
	})
	e.server = NewSSEServer(p, "127.0.0.1:0", opts...)
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

func (e *embeddedProbe) sessionCounts() (sse, stream int) {
	e.server.mu.Lock()
	sse = len(e.server.sessions)
	e.server.mu.Unlock()
	e.server.streamMu.Lock()
	stream = len(e.server.streamSessions)
	e.server.streamMu.Unlock()
	return sse, stream
}

type embeddedRequest struct {
	name, method, path, body, sid string
}

func rpcBody(method string, params string) string {
	if params == "" {
		params = "{}"
	}
	return `{"jsonrpc":"2.0","id":7,"method":"` + method + `","params":` + params + `}`
}

// embeddedRequests covers every route the handler serves (Streamable HTTP,
// legacy SSE, health, routes added by the caller) and every kind of MCP
// method (initialize, tools/list, tools/call, a passed-on method).
func embeddedRequests(prefix string) []embeddedRequest {
	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
	call := rpcBody(MethodToolsCall, `{"name":"echo","arguments":{}}`)
	sid := streamSessionIDFor(staticTestToken)
	return []embeddedRequest{
		{"streamable/initialize", http.MethodPost, prefix + "/mcp", init, ""},
		{"streamable/tools-list", http.MethodPost, prefix + "/mcp", rpcBody(MethodToolsList, ""), sid},
		{"streamable/tools-call", http.MethodPost, prefix + "/mcp", call, sid},
		{"streamable/resources-list", http.MethodPost, prefix + "/mcp", rpcBody("resources/list", ""), sid},
		{"streamable/notification", http.MethodPost, prefix + "/mcp", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, sid},
		{"streamable/get", http.MethodGet, prefix + "/mcp", "", sid},
		{"streamable/delete", http.MethodDelete, prefix + "/mcp", "", sid},
		{"sse/connect", http.MethodGet, prefix + "/sse", "", ""},
		{"sse/message-tools-list", http.MethodPost, prefix + "/message?sessionId=abc", rpcBody(MethodToolsList, ""), ""},
		{"sse/message-tools-call", http.MethodPost, prefix + "/message?sessionId=abc", call, ""},
		{"sse/message-resources-list", http.MethodPost, prefix + "/message?sessionId=abc", rpcBody("resources/list", ""), ""},
		{"health", http.MethodGet, prefix + "/health", "", ""},
		{"caller-route/HandleFunc", http.MethodGet, prefix + "/custom", "", ""},
		{"caller-route/Handle", http.MethodGet, prefix + "/custom-handler", "", ""},
	}
}

func doEmbedded(t *testing.T, base string, r embeddedRequest, bearer string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(r.method, base+r.path, strings.NewReader(r.body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	if r.sid != "" {
		req.Header.Set("Mcp-Session-Id", r.sid)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("%s: %v", r.name, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(b)
}

// wantAllRefused sends every request to base and checks each gets the
// auth.mode refusal and that nothing behind the handler ran.
func wantAllRefused(t *testing.T, label string, e *embeddedProbe, base, prefix string) {
	t.Helper()
	for _, bearer := range []string{"", staticTestToken} {
		for _, r := range embeddedRequests(prefix) {
			status, body := doEmbedded(t, base, r, bearer)
			if status != http.StatusServiceUnavailable || !strings.Contains(body, "auth.mode is required") {
				t.Fatalf("%s %s (bearer=%v): status %d body %q, want 503 auth.mode refusal",
					label, r.name, bearer != "", status, body)
			}
		}
	}
	sse, stream := e.sessionCounts()
	if e.router.calls != 0 || e.enriched.Load() != 0 || e.custom.Load() != 0 || sse != 0 || stream != 0 {
		t.Fatalf("%s: refused requests reached the server: upstream calls=%d tools/list=%d caller routes=%d sse sessions=%d stream sessions=%d",
			label, e.router.calls, e.enriched.Load(), e.custom.Load(), sse, stream)
	}
}

// Every exported way to get the network handler refuses every request when
// auth.mode was not written: SSEServer.Handler(), the SSEServer itself as an
// http.Handler (ServeHTTP), with WithBasePath and WithOAuthChallenge, and
// with or without a bearer credential.
func TestEmbeddedHandlerRefusesEveryRouteWithoutAuthMode(t *testing.T) {
	captureLogs(t)
	for name, cfg := range unsetModeConfigs(t) {
		t.Run(name, func(t *testing.T) {
			for _, way := range []struct {
				name   string
				prefix string
				opts   []SSEOption
				get    func(*SSEServer) http.Handler
			}{
				{"Handler", "", nil, func(s *SSEServer) http.Handler { return s.Handler() }},
				{"ServeHTTP", "", nil, func(s *SSEServer) http.Handler { return s }},
				{"Handler+WithBasePath", "/mcp", []SSEOption{WithBasePath("/mcp")}, func(s *SSEServer) http.Handler { return http.StripPrefix("/mcp", s.Handler()) }},
				{"Handler+WithOAuthChallenge", "", []SSEOption{WithOAuthChallenge("https://mcp.example.test/.well-known/oauth-protected-resource")}, func(s *SSEServer) http.Handler { return s.Handler() }},
			} {
				e := newEmbeddedProbe(t, cfg, way.opts...)
				srv := httptest.NewServer(way.get(e.server))
				wantAllRefused(t, way.name, e, srv.URL, way.prefix)
				srv.Close()
			}
		})
	}
}

// The refusal is logged once, when the server is created, not per request.
func TestEmbeddedHandlerWithoutAuthModeLogsOnceAtConstruction(t *testing.T) {
	for name, cfg := range unsetModeConfigs(t) {
		t.Run(name, func(t *testing.T) {
			logs := captureLogs(t)
			e := newEmbeddedProbe(t, cfg)
			if n := countLines(logs, `"level":"error"`, "auth.mode is required"); n != 1 {
				t.Fatalf("after construction: %d refusal log lines, want 1: %s", n, logs.String())
			}
			srv := httptest.NewServer(e.server.Handler())
			defer srv.Close()
			wantAllRefused(t, "Handler", e, srv.URL, "")
			if n := countLines(logs, "auth.mode is required"); n != 1 {
				t.Fatalf("after requests: %d refusal log lines, want 1", n)
			}
		})
	}
}

// ListenAndServe on the same server returns the same refusal and does not
// log it a second time.
func TestListenAndServeRefusalMatchesHandlerRefusal(t *testing.T) {
	logs := captureLogs(t)
	e := newEmbeddedProbe(t, &Config{Server: ServerConfig{Transport: "sse"}})
	srv := httptest.NewServer(e.server.Handler())
	defer srv.Close()
	_, body := doEmbedded(t, srv.URL, embeddedRequests("")[1], "")
	err := e.server.ListenAndServe(context.Background())
	if err == nil || strings.TrimSpace(body) != err.Error() {
		t.Fatalf("listener error %v, handler body %q: want the same message", err, body)
	}
	if n := countLines(logs, "auth.mode is required"); n != 1 {
		t.Fatalf("%d refusal log lines, want 1", n)
	}
}

// Reviewer case: a Config built in code with Auth.Mode unset, and a config
// loaded with stdio defaults, embedded through Handler(): Streamable HTTP
// initialize and tools/list, and SSE connect and tools/list, are refused.
func TestEmbeddedHandlerRefusesToolsListForUnsetModeInCodeAndLoadedStdioDefaults(t *testing.T) {
	captureLogs(t)
	for _, loaded := range []bool{false, true} {
		var cfg *Config
		if loaded {
			var err error
			if cfg, err = LoadConfig(writeConfig(t, testUpstreamYAML)); err != nil {
				t.Fatal(err)
			}
		} else {
			cfg = &Config{Server: ServerConfig{Transport: "sse"}}
		}
		p, err := New(cfg)
		if err != nil {
			t.Fatal(err)
		}
		router := &countingForwardRouter{}
		p.SetUpstreamRouter(router)
		srv := httptest.NewServer(NewSSEServer(p, "127.0.0.1:0").Handler())
		init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
		for _, r := range []embeddedRequest{
			{"streamable/initialize", http.MethodPost, "/mcp", init, ""},
			{"streamable/tools-list", http.MethodPost, "/mcp", rpcBody(MethodToolsList, ""), ""},
			{"sse/connect", http.MethodGet, "/sse", "", ""},
			{"sse/tools-list", http.MethodPost, "/message?sessionId=x", rpcBody(MethodToolsList, ""), ""},
		} {
			status, body := doEmbedded(t, srv.URL, r, "")
			if status != http.StatusServiceUnavailable || !strings.Contains(body, "auth.mode is required") || strings.Contains(body, "echo") {
				t.Fatalf("loaded=%v %s: status %d body %q", loaded, r.name, status, body)
			}
		}
		if router.calls != 0 {
			t.Fatalf("loaded=%v: upstream reached", loaded)
		}
		srv.Close()
	}
}

// Reviewer case: with Auth.Mode unset and a real upstream started, the
// embedded Streamable HTTP endpoint refuses initialize and the governed
// methods tools/list, tools/call and resources/list (a passed-on method).
func TestEmbeddedStreamableRefusesGovernedMethodsWithUpstreamRunning(t *testing.T) {
	captureLogs(t)
	cfg := &Config{
		Server:          ServerConfig{Transport: "http"},
		Upstreams:       map[string]UpstreamConfig{"probe": helperUpstream(nil, nil)},
		DefaultUpstream: "probe",
		Enforcement:     EnforcementConfig{Mode: "observe"},
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	if err := p.upstream.Start(ctx); err != nil {
		t.Fatal(err)
	}
	defer p.upstream.Close()
	srv := httptest.NewServer(NewSSEServer(p, "").Handler())
	defer srv.Close()
	init := `{"jsonrpc":"2.0","id":1,"method":"initialize","params":{"protocolVersion":"2025-03-26","capabilities":{},"clientInfo":{"name":"t","version":"1"}}}`
	if status, _ := doEmbedded(t, srv.URL, embeddedRequest{"initialize", http.MethodPost, "/mcp", init, ""}, ""); status != http.StatusServiceUnavailable {
		t.Fatalf("initialize: status %d, want 503", status)
	}
	sid := streamSessionIDFor("any")
	for _, method := range []string{MethodToolsList, MethodToolsCall, "resources/list"} {
		params := `{"name":"report","arguments":{}}`
		status, body := doEmbedded(t, srv.URL, embeddedRequest{method, http.MethodPost, "/mcp", rpcBody(method, params), sid}, "")
		if status != http.StatusServiceUnavailable || !strings.Contains(body, "auth.mode is required") {
			t.Fatalf("%s: status %d body %q, want 503 auth.mode refusal", method, status, body)
		}
	}
}

// An explicit auth.mode none serves as before, with the startup warning
// logged once when the server is created (and not again by ListenAndServe
// or per request).
func TestEmbeddedHandlerExplicitNoneServesWithOneWarning(t *testing.T) {
	loaded, err := LoadConfig(writeConfig(t, "auth:\n  mode: none\n"+testUpstreamYAML))
	if err != nil {
		t.Fatal(err)
	}
	for name, cfg := range map[string]*Config{
		"code":   {Server: ServerConfig{Transport: "sse"}, Auth: AuthConfig{Mode: "none"}},
		"loaded": loaded,
	} {
		t.Run(name, func(t *testing.T) {
			logs := captureLogs(t)
			e := newEmbeddedProbe(t, cfg)
			if n := countLines(logs, `"level":"warn"`, "auth.mode is none"); n != 1 {
				t.Fatalf("after construction: %d warnings, want 1: %s", n, logs.String())
			}
			srv := httptest.NewServer(e.server.Handler())
			defer srv.Close()
			sid := mustInitSession(t, srv.URL, "")
			resp := streamableCall(t, srv.URL, sid, "", rpcBody(MethodToolsList, ""))
			if resp.Error != nil || !strings.Contains(string(resp.Result), "echo") {
				t.Fatalf("tools/list under none: %+v", resp.Error)
			}
			resp = streamableCall(t, srv.URL, sid, "", rpcBody(MethodToolsCall, `{"name":"echo","arguments":{}}`))
			if resp.Error != nil || e.router.calls != 1 {
				t.Fatalf("tools/call under none: %+v calls=%d", resp.Error, e.router.calls)
			}
			conn := eaOpenSSE(t, srv.URL, "")
			if conn.post([]byte(rpcBody(MethodToolsList, ""))) != http.StatusAccepted || conn.next().Error != nil {
				t.Fatal("SSE tools/list under none refused")
			}
			conn.body.Close()
			if status, _ := doEmbedded(t, srv.URL, embeddedRequest{"health", http.MethodGet, "/health", "", ""}, ""); status != http.StatusOK {
				t.Fatalf("health: %d", status)
			}
			if n := countLines(logs, "auth.mode is none"); n != 1 {
				t.Fatalf("after requests: %d warnings, want 1", n)
			}
			if countLines(logs, "auth.mode is required") != 0 {
				t.Fatal("explicit none logged the unset-mode refusal")
			}
		})
	}
}

// "header" and "config" serve as before: a valid token gets tools/list and
// tools/call, no token gets the -32001 refusal, and neither logs the
// auth.mode refusal or the none warning.
func TestEmbeddedHandlerHeaderAndConfigModesServe(t *testing.T) {
	for _, mode := range []string{"header", "config"} {
		t.Run(mode, func(t *testing.T) {
			logs := captureLogs(t)
			p, router := newAuthModeProxy(t, mode)
			good := staticTestToken
			if mode == "header" {
				good = "good-token"
			}
			srv := httptest.NewServer(NewSSEServer(p, "127.0.0.1:0").Handler())
			defer srv.Close()
			sid := mustInitSession(t, srv.URL, good)
			resp := streamableCall(t, srv.URL, sid, good, rpcBody(MethodToolsList, ""))
			if resp.Error != nil {
				t.Fatalf("tools/list with token: %+v", resp.Error)
			}
			resp = streamableCall(t, srv.URL, sid, good, rpcBody(MethodToolsCall, `{"name":"echo","arguments":{}}`))
			if resp.Error != nil || router.calls != 1 {
				t.Fatalf("tools/call with token: %+v calls=%d", resp.Error, router.calls)
			}
			conn := eaOpenSSE(t, srv.URL, "")
			if conn.post([]byte(rpcBody(MethodToolsList, ""))) != http.StatusAccepted {
				t.Fatal("SSE post status")
			}
			if r := conn.next(); r.Error == nil || r.Error.Code != CodePolicyDenied {
				t.Fatalf("tokenless SSE tools/list: %+v", r.Error)
			}
			conn.body.Close()
			if countLines(logs, "auth.mode is required") != 0 || countLines(logs, "auth.mode is none") != 0 {
				t.Fatalf("unexpected auth.mode log: %s", logs.String())
			}
		})
	}
}

// A file without auth.mode, followed by code that sets a mode before New,
// serves under that mode.
func TestEmbeddedHandlerServesModeSetInCodeAfterLoad(t *testing.T) {
	captureLogs(t)
	cfg, err := LoadConfig(writeConfig(t, testUpstreamYAML))
	if err != nil {
		t.Fatal(err)
	}
	cfg.Auth = AuthConfig{Mode: "config", Token: staticTestToken}
	e := newEmbeddedProbe(t, cfg)
	srv := httptest.NewServer(e.server.Handler())
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, staticTestToken)
	if resp := streamableCall(t, srv.URL, sid, staticTestToken, rpcBody(MethodToolsList, "")); resp.Error != nil {
		t.Fatalf("tools/list: %+v", resp.Error)
	}
	if status, _ := doEmbedded(t, srv.URL, embeddedRequest{"no token", http.MethodPost, "/mcp", rpcBody(MethodToolsList, ""), sid}, ""); status == http.StatusOK || status == http.StatusServiceUnavailable {
		t.Fatalf("tools/list without token under config mode: %d, want an authentication refusal", status)
	}
}

// Clearing Auth.Mode after the server was created is refused at the next
// request (the check runs per request, not only at construction).
func TestEmbeddedHandlerRefusesModeClearedAfterConstruction(t *testing.T) {
	logs := captureLogs(t)
	cfg := &Config{Server: ServerConfig{Transport: "sse"}, Auth: AuthConfig{Mode: "none"}}
	e := newEmbeddedProbe(t, cfg)
	srv := httptest.NewServer(e.server.Handler())
	defer srv.Close()
	cfg.Auth.Mode = ""
	wantAllRefused(t, "cleared", e, srv.URL, "")
	if n := countLines(logs, "auth.mode is required"); n != 1 {
		t.Fatalf("%d refusal log lines, want 1", n)
	}
}

// The standalone listener with an explicit auth.mode none starts, serves
// and logs the none warning once (at construction, not again at start).
func TestListenAndServeNoneModeWarnsOnce(t *testing.T) {
	logs := captureLogs(t)
	p, err := New(&Config{Server: ServerConfig{Transport: "sse"}, Auth: AuthConfig{Mode: "none"}})
	if err != nil {
		t.Fatal(err)
	}
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan error, 1)
	go func() { done <- NewSSEServer(p, "127.0.0.1:0").ListenAndServe(ctx) }()
	time.Sleep(200 * time.Millisecond)
	cancel()
	select {
	case err := <-done:
		if err != nil {
			t.Fatalf("ListenAndServe: %v", err)
		}
	case <-time.After(10 * time.Second):
		t.Fatal("ListenAndServe did not stop")
	}
	if n := countLines(logs, `"level":"warn"`, "auth.mode is none"); n != 1 {
		t.Fatalf("%d none warnings, want 1: %s", n, logs.String())
	}
}
