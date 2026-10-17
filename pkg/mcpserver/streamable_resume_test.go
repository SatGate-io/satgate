package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// These tests play a SatGate restart: the client keeps the Mcp-Session-Id it
// got from one server instance and sends its next request to a new instance
// (new Proxy, new SSEServer, empty session map) with the same authenticator.

// resumeTokens verifies a fixed set of tokens. A token in revokedErr fails
// the way the enterprise authenticator fails for a revoked token.
type resumeTokens struct {
	mu         sync.Mutex
	tokens     map[string]TokenInfo
	revokedErr map[string]bool
}

func newResumeTokens() *resumeTokens {
	return &resumeTokens{
		tokens: map[string]TokenInfo{
			"token-alice": {TokenID: "tok-alice", BudgetID: "bud-alice", TenantID: "tenant-1", Scope: "*"},
			"token-bob":   {TokenID: "tok-bob", BudgetID: "bud-bob", TenantID: "tenant-1", Scope: "*"},
			"token-other": {TokenID: "tok-other", BudgetID: "bud-other", TenantID: "tenant-2", Scope: "*"},
		},
		revokedErr: map[string]bool{},
	}
}

func (a *resumeTokens) Verify(_ context.Context, token string) (*TokenInfo, error) {
	a.mu.Lock()
	defer a.mu.Unlock()
	if a.revokedErr[token] {
		return nil, fmt.Errorf("token revoked")
	}
	info, ok := a.tokens[token]
	if !ok {
		return nil, fmt.Errorf("invalid token")
	}
	return &info, nil
}

func (a *resumeTokens) revokeAtVerify(token string) {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.revokedErr[token] = true
}

// revokedBudgets is a RevocationChecker over budget ids.
type revokedBudgets struct {
	mu  sync.Mutex
	ids map[string]bool
}

func (r *revokedBudgets) IsRevoked(_ context.Context, budgetID string) bool {
	r.mu.Lock()
	defer r.mu.Unlock()
	return r.ids[budgetID]
}

type recordingPublisher struct {
	mu     sync.Mutex
	events []Event
}

func (p *recordingPublisher) Publish(e Event) {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.events = append(p.events, e)
}

func (p *recordingPublisher) connects() []Event {
	p.mu.Lock()
	defer p.mu.Unlock()
	var out []Event
	for _, e := range p.events {
		if e.Type == EventSessionConnect {
			out = append(out, e)
		}
	}
	return out
}

type resumeInstance struct {
	sse    *SSEServer
	proxy  *Proxy
	srv    *httptest.Server
	events *recordingPublisher
}

// newResumeInstance is one SatGate process in header auth mode (production).
func newResumeInstance(t *testing.T, auth Authenticator, opts ...SSEOption) *resumeInstance {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "resume-test", Version: "test"},
		Auth:        AuthConfig{Mode: "header", RootKey: "resume-test-root"},
		Budget:      BudgetConfig{Limit: 100, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 0},
		Enforcement: EnforcementConfig{Mode: "observe"},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetAuthenticator(auth)
	proxy.SetUpstreamRouter(&countingForwardRouter{})
	events := &recordingPublisher{}
	proxy.SetEventPublisher(events)
	sse := NewSSEServer(proxy, ":0", opts...)
	srv := httptest.NewServer(sse.Handler())
	t.Cleanup(srv.Close)
	return &resumeInstance{sse: sse, proxy: proxy, srv: srv, events: events}
}

func (in *resumeInstance) url() string { return in.srv.URL + "/mcp" }

// resumePost sends one Streamable HTTP POST the way MCP clients do after
// initialize (MCP-Protocol-Version 2025-06-18).
func resumePost(t *testing.T, url, sid, token string, msg any) (int, http.Header, string) {
	t.Helper()
	return resumeDo(t, http.MethodPost, url, sid, token, "2025-06-18", msg)
}

func resumeDo(t *testing.T, method, url, sid, token, protocol string, msg any) (int, http.Header, string) {
	t.Helper()
	var body *strings.Reader
	if msg != nil {
		body = strings.NewReader(string(mustJSON(t, msg)))
	} else {
		body = strings.NewReader("")
	}
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		t.Fatal(err)
	}
	if msg != nil {
		req.Header.Set("Content-Type", "application/json")
	}
	req.Header.Set("Accept", "application/json, text/event-stream")
	if sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}
	if protocol != "" {
		req.Header.Set("MCP-Protocol-Version", protocol)
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	var sb strings.Builder
	buf := make([]byte, 4096)
	for {
		n, rerr := resp.Body.Read(buf)
		sb.Write(buf[:n])
		if rerr != nil {
			break
		}
	}
	return resp.StatusCode, resp.Header, sb.String()
}

func resumeInit(t *testing.T, in *resumeInstance, token string) string {
	t.Helper()
	status, hdr, body := resumeDo(t, http.MethodPost, in.url(), "", token, "", initializeBody("2025-06-18"))
	if status != http.StatusOK || hdr.Get("Mcp-Session-Id") == "" {
		t.Fatalf("initialize status=%d body=%s", status, body)
	}
	return hdr.Get("Mcp-Session-Id")
}

func requireToolResult(t *testing.T, what string, status int, body string) {
	t.Helper()
	if status != http.StatusOK || !strings.Contains(body, `"result"`) || strings.Contains(body, `"error"`) {
		t.Fatalf("%s: status=%d body=%s", what, status, body)
	}
}

// A restart in the middle of a session: the next tools/call from the same
// token, with the same session id, works on the new instance, under that id.
func TestStreamableResumeAfterRestartSameToken(t *testing.T) {
	auth := newResumeTokens()
	before := newResumeInstance(t, auth)
	sid := resumeInit(t, before, "token-alice")
	status, _, body := resumePost(t, before.url(), sid, "token-alice", toolCallBody(2, "echo"))
	requireToolResult(t, "call before restart", status, body)

	after := newResumeInstance(t, auth) // the restarted process
	status, hdr, body := resumePost(t, after.url(), sid, "token-alice", toolCallBody(3, "echo"))
	requireToolResult(t, "call after restart", status, body)
	if got := hdr.Get("Mcp-Session-Id"); got != sid {
		t.Fatalf("session id after restart = %q, want the client's %q", got, sid)
	}
	if got := hdr.Get("MCP-Protocol-Version"); got != "2025-06-18" {
		t.Fatalf("protocol after restart = %q, want the client's 2025-06-18", got)
	}
	if got := streamableForwardCount(after.proxy); got != 1 {
		t.Fatalf("restarted instance forwarded %d calls, want 1", got)
	}
	// tools/list and a notification work in the resumed session too.
	status, _, body = resumePost(t, after.url(), sid, "token-alice", map[string]any{"jsonrpc": "2.0", "id": 4, "method": "tools/list"})
	if status != http.StatusOK || !strings.Contains(body, `"echo"`) {
		t.Fatalf("tools/list after restart: status=%d body=%s", status, body)
	}
	status, _, _ = resumePost(t, after.url(), sid, "token-alice", map[string]any{"jsonrpc": "2.0", "method": "notifications/initialized"})
	if status != http.StatusAccepted {
		t.Fatalf("notification after restart: status=%d", status)
	}
	if n := streamSessionCount(after.sse); n != 1 {
		t.Fatalf("restarted instance holds %d sessions, want 1", n)
	}
	// The resumed session is bound to the token like any other: another
	// verified token on the now-live id is refused as before (403).
	status, _, body = resumePost(t, after.url(), sid, "token-bob", toolCallBody(5, "echo"))
	if status != http.StatusForbidden {
		t.Fatalf("other token on resumed session: status=%d body=%s", status, body)
	}
	if got := streamableForwardCount(after.proxy); got != 1 {
		t.Fatalf("forwarded %d calls, want 1", got)
	}
	// The dashboard hears about the session again, marked as resumed, with
	// the verified identity of the token that resumed it.
	connects := after.events.connects()
	if len(connects) != 1 || connects[0].SessionID != sid || connects[0].TokenID != "tok-alice" ||
		connects[0].Data == nil || connects[0].Data["resumed"] != true {
		t.Fatalf("connect events after restart = %+v", connects)
	}
}

// A client that never sends MCP-Protocol-Version gets the spec default.
func TestStreamableResumeWithoutProtocolHeaderUsesDefault(t *testing.T) {
	auth := newResumeTokens()
	sid := resumeInit(t, newResumeInstance(t, auth), "token-alice")
	after := newResumeInstance(t, auth)
	status, hdr, body := resumeDo(t, http.MethodPost, after.url(), sid, "token-alice", "", toolCallBody(2, "echo"))
	requireToolResult(t, "call after restart without protocol header", status, body)
	if got := hdr.Get("MCP-Protocol-Version"); got != "2025-03-26" {
		t.Fatalf("protocol = %q, want 2025-03-26", got)
	}
	// An unsupported header is refused and creates nothing.
	other := newResumeInstance(t, auth)
	status, _, body = resumeDo(t, http.MethodPost, other.url(), sid, "token-alice", "1999-01-01", toolCallBody(3, "echo"))
	if status != http.StatusBadRequest || streamSessionCount(other.sse) != 0 || streamableForwardCount(other.proxy) != 0 {
		t.Fatalf("unsupported protocol on resume: status=%d sessions=%d body=%s", status, streamSessionCount(other.sse), body)
	}
}

// After a restart, a different valid token (same account or another one)
// presenting the session id is refused and cannot claim the id: the owner
// still resumes it afterwards.
func TestStreamableResumeRefusesAnotherToken(t *testing.T) {
	auth := newResumeTokens()
	sid := resumeInit(t, newResumeInstance(t, auth), "token-alice")
	after := newResumeInstance(t, auth)

	for _, tok := range []string{"token-bob", "token-other"} {
		status, hdr, body := resumePost(t, after.url(), sid, tok, toolCallBody(2, "echo"))
		if status != http.StatusNotFound || strings.Contains(body, `"result"`) {
			t.Fatalf("%s on alice's session after restart: status=%d body=%s", tok, status, body)
		}
		if hdr.Get("Mcp-Session-Id") != "" {
			t.Fatalf("%s was handed a session id", tok)
		}
	}
	if n := streamSessionCount(after.sse); n != 0 {
		t.Fatalf("refused tokens left %d sessions", n)
	}
	if got := streamableForwardCount(after.proxy); got != 0 {
		t.Fatalf("refused tokens reached the upstream %d times", got)
	}
	status, _, body := resumePost(t, after.url(), sid, "token-alice", toolCallBody(3, "echo"))
	requireToolResult(t, "owner after refused takeover", status, body)

	// A made-up id is not resumed for anyone.
	for _, fake := range []string{"mcp-does-not-exist", "mcp-0123456789abcdef0123456789abcdef"} {
		status, _, body = resumePost(t, after.url(), fake, "token-alice", toolCallBody(4, "echo"))
		if status != http.StatusNotFound {
			t.Fatalf("made-up id %q: status=%d body=%s", fake, status, body)
		}
	}
}

// After a restart a revoked, unknown, expired or missing token is refused
// exactly as at initialize, and nothing is created or forwarded.
func TestStreamableResumeRefusesRevokedAndInvalidTokens(t *testing.T) {
	auth := newResumeTokens()
	sid := resumeInit(t, newResumeInstance(t, auth), "token-alice")

	// Revoked by the revocation checker (budget id).
	after := newResumeInstance(t, auth)
	after.proxy.SetRevocationChecker(&revokedBudgets{ids: map[string]bool{"bud-alice": true}})
	status, _, body := resumePost(t, after.url(), sid, "token-alice", toolCallBody(2, "echo"))
	if status != http.StatusForbidden || !strings.Contains(body, "TOKEN_REVOKED") {
		t.Fatalf("revoked (checker): status=%d body=%s", status, body)
	}
	if streamSessionCount(after.sse) != 0 || streamableForwardCount(after.proxy) != 0 {
		t.Fatalf("revoked token left a session or reached the upstream")
	}

	// Revoked at verification (the enterprise authenticator's answer).
	auth2 := newResumeTokens()
	sid2 := resumeInit(t, newResumeInstance(t, auth2), "token-alice")
	auth2.revokeAtVerify("token-alice")
	after2 := newResumeInstance(t, auth2)
	status, _, body = resumePost(t, after2.url(), sid2, "token-alice", toolCallBody(2, "echo"))
	if status != http.StatusForbidden || !strings.Contains(body, "TOKEN_REVOKED") {
		t.Fatalf("revoked (verify): status=%d body=%s", status, body)
	}

	// No token and a token that does not verify (expired, wrong key) get 401.
	for _, tok := range []string{"", "token-expired-or-forged"} {
		status, _, body = resumePost(t, after2.url(), sid2, tok, toolCallBody(3, "echo"))
		if status != http.StatusUnauthorized {
			t.Fatalf("token %q: status=%d body=%s", tok, status, body)
		}
	}
	if streamSessionCount(after2.sse) != 0 || streamableForwardCount(after2.proxy) != 0 {
		t.Fatalf("refused tokens left a session or reached the upstream")
	}
}

// With a real macaroon: an expired token cannot resume its session.
func TestStreamableResumeRefusesExpiredMacaroon(t *testing.T) {
	cfg := AuthConfig{Mode: "header", RootKey: "resume-macaroon-root"}
	auth, err := NewAuthenticator(cfg)
	if err != nil {
		t.Fatal(err)
	}
	svc := auth.(*MacaroonAuthenticator).Service
	short, err := svc.Mint("api:*", time.Now().Add(1500*time.Millisecond))
	if err != nil {
		t.Fatal(err)
	}
	token := svc.Encode(short)
	sid := resumeInit(t, newResumeInstance(t, auth), token)
	after := newResumeInstance(t, auth)
	status, _, body := resumePost(t, after.url(), sid, token, toolCallBody(2, "echo"))
	requireToolResult(t, "unexpired macaroon after restart", status, body)

	time.Sleep(1600 * time.Millisecond)
	later := newResumeInstance(t, auth)
	status, _, body = resumePost(t, later.url(), sid, token, toolCallBody(3, "echo"))
	if status != http.StatusUnauthorized {
		t.Fatalf("expired macaroon: status=%d body=%s", status, body)
	}
	if streamSessionCount(later.sse) != 0 || streamableForwardCount(later.proxy) != 0 {
		t.Fatalf("expired token left a session or reached the upstream")
	}
}

// DELETE ends a session. The process that ended it does not bring the id
// back, not even for its own token. A process that never saw the DELETE (a
// restart, another machine) serves the id again only for the same valid
// token, which could open an identical session with initialize anyway.
func TestStreamableResumeDeletedSession(t *testing.T) {
	auth := newResumeTokens()
	in := newResumeInstance(t, auth)
	sid := resumeInit(t, in, "token-alice")
	status, _, body := resumeDo(t, http.MethodDelete, in.url(), sid, "token-alice", "2025-06-18", nil)
	if status != http.StatusOK {
		t.Fatalf("DELETE status=%d body=%s", status, body)
	}
	status, _, body = resumePost(t, in.url(), sid, "token-alice", toolCallBody(2, "echo"))
	if status != http.StatusNotFound {
		t.Fatalf("POST after DELETE on the same process: status=%d body=%s", status, body)
	}
	status, _, _ = resumeDo(t, http.MethodDelete, in.url(), sid, "token-alice", "2025-06-18", nil)
	if status != http.StatusNotFound {
		t.Fatalf("second DELETE: status=%d", status)
	}
	if streamSessionCount(in.sse) != 0 || streamableForwardCount(in.proxy) != 0 {
		t.Fatalf("deleted session came back on the same process")
	}

	after := newResumeInstance(t, auth)
	for _, tok := range []string{"", "token-bob", "token-expired-or-forged"} {
		status, _, body = resumePost(t, after.url(), sid, tok, toolCallBody(3, "echo"))
		if status == http.StatusOK {
			t.Fatalf("deleted id revived for token %q: body=%s", tok, body)
		}
	}
	if streamSessionCount(after.sse) != 0 {
		t.Fatalf("deleted id revived for a wrong token")
	}
	status, _, body = resumePost(t, after.url(), sid, "token-alice", toolCallBody(4, "echo"))
	requireToolResult(t, "deleted id on a new process, same valid token", status, body)
}

// The Streamable HTTP GET stream is not offered (405) before or after a
// restart. A GET with the old id creates nothing, and the client's next
// POST resumes the session.
func TestStreamableResumeGETStreamAfterRestart(t *testing.T) {
	auth := newResumeTokens()
	before := newResumeInstance(t, auth)
	sid := resumeInit(t, before, "token-alice")
	status, hdr, _ := resumeDo(t, http.MethodGet, before.url(), sid, "token-alice", "2025-06-18", nil)
	if status != http.StatusMethodNotAllowed || !strings.Contains(hdr.Get("Allow"), "POST") {
		t.Fatalf("GET before restart: status=%d allow=%q", status, hdr.Get("Allow"))
	}

	after := newResumeInstance(t, auth)
	for _, tok := range []string{"token-alice", "token-bob", ""} {
		status, _, body := resumeDo(t, http.MethodGet, after.url(), sid, tok, "2025-06-18", nil)
		if status != http.StatusMethodNotAllowed {
			t.Fatalf("GET after restart (token %q): status=%d body=%s", tok, status, body)
		}
	}
	if n := streamSessionCount(after.sse); n != 0 {
		t.Fatalf("GET after restart created %d sessions", n)
	}
	status, _, body := resumePost(t, after.url(), sid, "token-alice", toolCallBody(2, "echo"))
	requireToolResult(t, "POST after GET after restart", status, body)
}

// Resuming counts against the session cap like initialize does.
func TestStreamableResumeRespectsSessionCap(t *testing.T) {
	auth := newResumeTokens()
	first := newResumeInstance(t, auth)
	sid := resumeInit(t, first, "token-alice")
	after := newResumeInstance(t, auth, WithMaxStreamableSessions(1))
	_ = resumeInit(t, after, "token-bob")
	status, _, body := resumePost(t, after.url(), sid, "token-alice", toolCallBody(2, "echo"))
	if status != http.StatusServiceUnavailable {
		t.Fatalf("resume over cap: status=%d body=%s", status, body)
	}
	if n := streamSessionCount(after.sse); n != 1 || streamableForwardCount(after.proxy) != 0 {
		t.Fatalf("resume over cap: sessions=%d forwards=%d", n, streamableForwardCount(after.proxy))
	}
}

// Auth modes without per-request verification keep answering an unknown id
// with 404: there is no verified token to bind a resumed session to.
func TestStreamableResumeOnlyInHeaderMode(t *testing.T) {
	before, _ := newStreamableTestServer(t, 5)
	defer before.Close()
	sid := mustInitSession(t, before.URL, "")
	after, proxy := newStreamableTestServer(t, 5)
	defer after.Close()
	status, _, body := streamablePost(t, after.URL+"/mcp", sid, "", toolCallBody(2, "echo"))
	if status != http.StatusNotFound || streamableForwardCount(proxy) != 0 {
		t.Fatalf("auth mode none, unknown id: status=%d body=%s", status, body)
	}
}

// Two requests racing to resume the same id end up in one session.
func TestStreamableResumeConcurrentRequestsShareOneSession(t *testing.T) {
	auth := newResumeTokens()
	sid := resumeInit(t, newResumeInstance(t, auth), "token-alice")
	after := newResumeInstance(t, auth)
	var wg sync.WaitGroup
	codes := make([]int, 8)
	for i := range codes {
		wg.Add(1)
		go func(i int) {
			defer wg.Done()
			status, _, _ := resumePost(t, after.url(), sid, "token-alice", map[string]any{"jsonrpc": "2.0", "id": 10 + i, "method": "tools/list"})
			codes[i] = status
		}(i)
	}
	wg.Wait()
	for i, c := range codes {
		if c != http.StatusOK {
			t.Fatalf("request %d: status=%d", i, c)
		}
	}
	if n := streamSessionCount(after.sse); n != 1 {
		t.Fatalf("sessions=%d, want 1", n)
	}
	if n := len(after.events.connects()); n != 1 {
		t.Fatalf("connect events=%d, want 1", n)
	}
}
