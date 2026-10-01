package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestStreamableHTTPInitializeSessionAndToolsList(t *testing.T) {
	srv, proxy := newStreamableTestServer(t, 5)
	defer srv.Close()

	status, hdr, body := streamablePost(t, srv.URL+"/mcp", "", "", initializeBody("2025-06-18"))
	if status != http.StatusOK {
		t.Fatalf("initialize status=%d body=%s", status, body)
	}
	sid := hdr.Get("Mcp-Session-Id")
	if sid == "" || !strings.HasPrefix(sid, "mcp-") {
		t.Fatalf("session id=%q", sid)
	}
	if !strings.Contains(body, `"protocolVersion":"2025-06-18"`) {
		t.Fatalf("initialize body=%s", body)
	}

	status, _, body = streamablePost(t, srv.URL+"/mcp", sid, "", map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/list",
	})
	if status != http.StatusOK || !strings.Contains(body, `"echo"`) {
		t.Fatalf("tools/list status=%d body=%s", status, body)
	}
	if proxy == nil {
		t.Fatal("proxy missing")
	}
}

func TestStreamableHTTPMountRootAndEmptyPath(t *testing.T) {
	srv, _ := newStreamableTestServer(t, 5)
	defer srv.Close()

	for _, path := range []string{"/", "/mcp", "/mcp/"} {
		status, hdr, body := streamablePost(t, srv.URL+path, "", "", initializeBody("2025-03-26"))
		if status != http.StatusOK || hdr.Get("Mcp-Session-Id") == "" {
			t.Fatalf("path %s status=%d sid=%q body=%s", path, status, hdr.Get("Mcp-Session-Id"), body)
		}
	}

	// StripPrefix("/mcp") on POST /mcp leaves an empty path. That must still
	// be the public .../mcp endpoint, not a 307 back to "/".
	req := httptest.NewRequest(http.MethodPost, srv.URL+"/mcp", bytes.NewReader(mustJSON(t, initializeBody("2025-03-26"))))
	req.URL.Path = ""
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	rr := httptest.NewRecorder()
	srv.Config.Handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK || rr.Header().Get("Mcp-Session-Id") == "" {
		t.Fatalf("empty path status=%d sid=%q body=%s", rr.Code, rr.Header().Get("Mcp-Session-Id"), rr.Body.String())
	}
}

func TestStreamableHTTPToolsCallDebitsAndExhausts(t *testing.T) {
	srv, proxy := newStreamableTestServer(t, 1)
	defer srv.Close()

	sid := mustInitSession(t, srv.URL, "")
	status, _, body := streamablePost(t, srv.URL+"/mcp", sid, "", toolCallBody(2, "echo"))
	if status != http.StatusOK || !strings.Contains(body, `"result"`) {
		t.Fatalf("call status=%d body=%s", status, body)
	}
	remaining, err := proxy.budget.Remaining(context.Background(), "default")
	if err != nil || remaining != 0 {
		t.Fatalf("remaining=%d err=%v", remaining, err)
	}

	status, _, body = streamablePost(t, srv.URL+"/mcp", sid, "", toolCallBody(3, "echo"))
	if status != http.StatusOK || !strings.Contains(body, "Budget exhausted") {
		t.Fatalf("exhausted status=%d body=%s", status, body)
	}
	remaining, err = proxy.budget.Remaining(context.Background(), "default")
	if err != nil || remaining != 0 {
		t.Fatalf("remaining after refusal=%d err=%v", remaining, err)
	}
}

func TestStreamableHTTPWrongTokenRefused(t *testing.T) {
	srv, proxy := newStreamableTestServer(t, 5)
	defer srv.Close()
	before := streamableForwardCount(proxy)

	sid := mustInitSession(t, srv.URL, "token-a")
	status, _, body := streamablePost(t, srv.URL+"/mcp", sid, "token-b", toolCallBody(2, "echo"))
	if status != http.StatusForbidden {
		t.Fatalf("wrong token status=%d body=%s", status, body)
	}
	status, _, body = streamablePost(t, srv.URL+"/mcp", sid, "", toolCallBody(3, "echo"))
	if status != http.StatusForbidden {
		t.Fatalf("missing token status=%d body=%s", status, body)
	}
	if got := streamableForwardCount(proxy); got != before {
		t.Fatalf("forwarded %d calls after token refusal", got-before)
	}

	status, _, body = streamablePost(t, srv.URL+"/mcp", sid, "token-a", toolCallBody(4, "echo"))
	if status != http.StatusOK || !strings.Contains(body, `"result"`) {
		t.Fatalf("owner call status=%d body=%s", status, body)
	}
}

func TestStreamableHTTPMissingAndUnknownSession(t *testing.T) {
	srv, _ := newStreamableTestServer(t, 5)
	defer srv.Close()

	status, _, body := streamablePost(t, srv.URL+"/mcp", "", "", toolCallBody(1, "echo"))
	if status != http.StatusBadRequest {
		t.Fatalf("missing session status=%d body=%s", status, body)
	}
	status, _, body = streamablePost(t, srv.URL+"/mcp", "mcp-does-not-exist", "", toolCallBody(1, "echo"))
	if status != http.StatusNotFound {
		t.Fatalf("unknown session status=%d body=%s", status, body)
	}
}

func TestStreamableHTTPExpiredSession(t *testing.T) {
	handler, proxy := newStreamableHandler(t, 5)
	sid := mustInitSessionHandler(t, handler, "")
	proxyServer, ok := handler.(*SSEServer)
	if !ok {
		t.Fatalf("handler type %T", handler)
	}
	_ = proxy
	proxyServer.expireStreamableSession(sid)

	status, _, body := streamablePostHandler(t, handler, "/mcp", sid, "", toolCallBody(2, "echo"))
	if status != http.StatusNotFound {
		t.Fatalf("expired session status=%d body=%s", status, body)
	}
}

func TestStreamableHTTPDeleteTerminatesSession(t *testing.T) {
	srv, _ := newStreamableTestServer(t, 5)
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, "token-a")

	req, err := http.NewRequest(http.MethodDelete, srv.URL+"/mcp", nil)
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Mcp-Session-Id", sid)
	req.Header.Set("Authorization", "Bearer token-a")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("DELETE status=%d", resp.StatusCode)
	}

	status, _, body := streamablePost(t, srv.URL+"/mcp", sid, "token-a", toolCallBody(2, "echo"))
	if status != http.StatusNotFound {
		t.Fatalf("after DELETE status=%d body=%s", status, body)
	}
}

func TestStreamableHTTPGetIsMethodNotAllowed(t *testing.T) {
	srv, _ := newStreamableTestServer(t, 5)
	defer srv.Close()
	resp, err := http.Get(srv.URL + "/mcp")
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusMethodNotAllowed {
		t.Fatalf("GET status=%d", resp.StatusCode)
	}
}

func TestStreamableHTTPNotificationAccepted(t *testing.T) {
	srv, _ := newStreamableTestServer(t, 5)
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, "")
	status, _, body := streamablePost(t, srv.URL+"/mcp", sid, "", map[string]any{
		"jsonrpc": "2.0", "method": "notifications/initialized",
	})
	if status != http.StatusAccepted || body != "" {
		t.Fatalf("notification status=%d body=%q", status, body)
	}
}

func TestStreamableHTTPRejectsForeignOrigin(t *testing.T) {
	srv, _ := newStreamableTestServer(t, 5)
	defer srv.Close()

	req, err := http.NewRequest(http.MethodPost, srv.URL+"/mcp", bytes.NewReader(mustJSON(t, initializeBody("2025-03-26"))))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Origin", "http://evil.example")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusForbidden {
		t.Fatalf("foreign origin status=%d", resp.StatusCode)
	}

	req, err = http.NewRequest(http.MethodPost, srv.URL+"/mcp", bytes.NewReader(mustJSON(t, initializeBody("2025-03-26"))))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Origin", "http://127.0.0.1:9")
	resp, err = http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK || resp.Header.Get("Mcp-Session-Id") == "" {
		t.Fatalf("loopback origin status=%d", resp.StatusCode)
	}
}

func TestStreamableHTTPRejectsUnsupportedProtocolVersion(t *testing.T) {
	srv, _ := newStreamableTestServer(t, 5)
	defer srv.Close()
	status, _, _ := streamablePost(t, srv.URL+"/mcp", "", "", initializeBody("1999-01-01"))
	if status != http.StatusBadRequest {
		t.Fatalf("bad init version status=%d", status)
	}
	sid := mustInitSession(t, srv.URL, "")
	req, err := http.NewRequest(http.MethodPost, srv.URL+"/mcp", bytes.NewReader(mustJSON(t, toolCallBody(2, "echo"))))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Session-Id", sid)
	req.Header.Set("MCP-Protocol-Version", "1999-01-01")
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusBadRequest {
		t.Fatalf("bad header version status=%d", resp.StatusCode)
	}
}

func TestStreamableHTTPDoesNotStealSSE(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	_, addr := startTestSSEServer(t, ctx)
	resp, err := http.Get("http://" + addr + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("health status=%d", resp.StatusCode)
	}
	sse, err := http.Get("http://" + addr + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	defer sse.Body.Close()
	if sse.StatusCode != http.StatusOK || !strings.Contains(sse.Header.Get("Content-Type"), "text/event-stream") {
		t.Fatalf("sse status=%d ct=%q", sse.StatusCode, sse.Header.Get("Content-Type"))
	}
}

type countingForwardRouter struct {
	calls int
}

func (r *countingForwardRouter) AllToolsForTenant(context.Context, string) []json.RawMessage {
	return []json.RawMessage{json.RawMessage(`{"name":"echo","description":"echo","inputSchema":{"type":"object"}}`)}
}

func (r *countingForwardRouter) ForwardToolCallForTenant(context.Context, string, string, json.RawMessage, time.Duration) (*Response, error) {
	r.calls++
	return &Response{JSONRPC: "2.0", Result: json.RawMessage(`{"content":[{"type":"text","text":"ok"}],"isError":false}`)}, nil
}

func newStreamableTestServer(t *testing.T, budget int64) (*httptest.Server, *Proxy) {
	t.Helper()
	handler, proxy := newStreamableHandler(t, budget)
	return httptest.NewServer(handler), proxy
}

func newStreamableHandler(t *testing.T, budget int64) (http.Handler, *Proxy) {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "streamable-test", Version: "test"},
		Auth:        AuthConfig{Mode: "none"},
		Budget:      BudgetConfig{Limit: budget, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetUpstreamRouter(&countingForwardRouter{})
	return NewSSEServer(proxy, ":0").Handler(), proxy
}

func streamableForwardCount(proxy *Proxy) int {
	router, _ := proxy.router.(*countingForwardRouter)
	if router == nil {
		return 0
	}
	return router.calls
}

func mustInitSession(t *testing.T, baseURL, token string) string {
	t.Helper()
	status, hdr, body := streamablePost(t, baseURL+"/mcp", "", token, initializeBody("2025-03-26"))
	if status != http.StatusOK {
		t.Fatalf("initialize status=%d body=%s", status, body)
	}
	sid := hdr.Get("Mcp-Session-Id")
	if sid == "" {
		t.Fatal("missing session id")
	}
	return sid
}

func mustInitSessionHandler(t *testing.T, handler http.Handler, token string) string {
	t.Helper()
	status, hdr, body := streamablePostHandler(t, handler, "/mcp", "", token, initializeBody("2025-03-26"))
	if status != http.StatusOK {
		t.Fatalf("initialize status=%d body=%s", status, body)
	}
	sid := hdr.Get("Mcp-Session-Id")
	if sid == "" {
		t.Fatal("missing session id")
	}
	return sid
}

func initializeBody(version string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"protocolVersion": version,
			"capabilities":    map[string]any{},
			"clientInfo":      map[string]string{"name": "test", "version": "1"},
		},
	}
}

func toolCallBody(id int, name string) map[string]any {
	return map[string]any{
		"jsonrpc": "2.0",
		"id":      id,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": map[string]any{},
		},
	}
}

func streamablePost(t *testing.T, url, sessionID, token string, msg any) (int, http.Header, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, bytes.NewReader(mustJSON(t, msg)))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
		req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, string(raw)
}

func streamablePostHandler(t *testing.T, handler http.Handler, path, sessionID, token string, msg any) (int, http.Header, string) {
	t.Helper()
	req := httptest.NewRequest(http.MethodPost, "http://example.test"+path, bytes.NewReader(mustJSON(t, msg)))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	if sessionID != "" {
		req.Header.Set("Mcp-Session-Id", sessionID)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	return rr.Code, rr.Header(), rr.Body.String()
}

func mustJSON(t *testing.T, v any) []byte {
	t.Helper()
	b, err := json.Marshal(v)
	if err != nil {
		t.Fatal(err)
	}
	return b
}

// In header auth mode (production), a Streamable HTTP session opened without
// a token must not be able to reach an upstream: tools/call fails closed in
// Proxy.authenticate, exactly as it does on the SSE /message path.
func TestStreamableHTTPHeaderModeNoTokenCannotCallTools(t *testing.T) {
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "streamable-test", Version: "test"},
		Auth:        AuthConfig{Mode: "header", RootKey: "streamable-header-test-root"},
		Budget:      BudgetConfig{Limit: 5, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetUpstreamRouter(&countingForwardRouter{})
	srv := httptest.NewServer(NewSSEServer(proxy, ":0").Handler())
	defer srv.Close()

	// Header mode refuses a tokenless initialize outright (no session).
	status, hdr, body := streamablePost(t, srv.URL+"/mcp", "", "", initializeBody("2025-03-26"))
	if status != http.StatusUnauthorized || hdr.Get("Mcp-Session-Id") != "" {
		t.Fatalf("tokenless initialize status=%d sid=%q body=%s", status, hdr.Get("Mcp-Session-Id"), body)
	}
	// Belt and braces: even if a tokenless request reached the proxy, the
	// header-mode check in Proxy.authenticate refuses the tool call.
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`9`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"echo"}`)}
	resp, err := proxy.handleRequest(context.Background(), req)
	if err == nil && resp != nil && resp.Error == nil {
		t.Fatalf("tokenless tools/call succeeded in header mode: %+v", resp)
	}
	if got := streamableForwardCount(proxy); got != 0 {
		t.Fatalf("tokenless session reached upstream %d times", got)
	}
}

// fixedTokenAuth verifies exactly one token value.
type fixedTokenAuth struct{ good string }

func (a fixedTokenAuth) Verify(_ context.Context, token string) (*TokenInfo, error) {
	if token != "" && token == a.good {
		return &TokenInfo{TokenID: "tok-good", BudgetID: "tok-good", Scope: "*"}, nil
	}
	return nil, fmt.Errorf("invalid token")
}

func newHeaderModeStreamable(t *testing.T, opts ...SSEOption) (*SSEServer, *Proxy) {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "streamable-test", Version: "test"},
		Auth:        AuthConfig{Mode: "header", RootKey: "streamable-header-test-root"},
		Budget:      BudgetConfig{Limit: 5, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetAuthenticator(fixedTokenAuth{good: "good-token"})
	proxy.SetUpstreamRouter(&countingForwardRouter{})
	return NewSSEServer(proxy, ":0", opts...), proxy
}

func streamSessionCount(s *SSEServer) int {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	return len(s.streamSessions)
}

// In header mode, initialize without a token or with a token that does not
// verify is refused with 401 and creates no session.
func TestStreamableHTTPHeaderModeInitializeNeedsVerifiedToken(t *testing.T) {
	sse, _ := newHeaderModeStreamable(t)
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()

	for _, tok := range []string{"", "bad-token"} {
		status, hdr, body := streamablePost(t, srv.URL+"/mcp", "", tok, initializeBody("2025-03-26"))
		if status != http.StatusUnauthorized || hdr.Get("Mcp-Session-Id") != "" {
			t.Fatalf("token %q: initialize status=%d sid=%q body=%s", tok, status, hdr.Get("Mcp-Session-Id"), body)
		}
	}
	if n := streamSessionCount(sse); n != 0 {
		t.Fatalf("refused initialize left %d sessions", n)
	}
	status, hdr, body := streamablePost(t, srv.URL+"/mcp", "", "good-token", initializeBody("2025-03-26"))
	if status != http.StatusOK || hdr.Get("Mcp-Session-Id") == "" {
		t.Fatalf("verified initialize status=%d body=%s", status, body)
	}
}

// The session map is capped. A full map is swept for expired sessions first;
// if it is still full, initialize gets 503 and the map does not grow.
func TestStreamableHTTPSessionCapAndSweep(t *testing.T) {
	sse, _ := newHeaderModeStreamable(t, WithMaxStreamableSessions(2))
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()

	first := mustInitSession(t, srv.URL, "good-token")
	_ = mustInitSession(t, srv.URL, "good-token")
	status, _, body := streamablePost(t, srv.URL+"/mcp", "", "good-token", initializeBody("2025-03-26"))
	if status != http.StatusServiceUnavailable {
		t.Fatalf("over cap status=%d body=%s", status, body)
	}
	if n := streamSessionCount(sse); n != 2 {
		t.Fatalf("sessions=%d after refused create, want 2", n)
	}

	// An expired session is swept to make room even though nobody looks it up.
	sse.expireStreamableSession(first)
	status, _, body = streamablePost(t, srv.URL+"/mcp", "", "good-token", initializeBody("2025-03-26"))
	if status != http.StatusOK {
		t.Fatalf("after expiry status=%d body=%s", status, body)
	}
	if n := streamSessionCount(sse); n != 2 {
		t.Fatalf("sessions=%d after sweep, want 2", n)
	}
	sse.streamMu.Lock()
	_, stillThere := sse.streamSessions[first]
	sse.streamMu.Unlock()
	if stillThere {
		t.Fatal("expired session was not swept")
	}
}
