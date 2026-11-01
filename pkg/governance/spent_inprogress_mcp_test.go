package governance_test

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/mcpserver"
)

// The MCP runtime: a decision in progress when the count is spent ends with
// the revocation-unavailable answer on every transport: HTTP 503
// REVOCATION_UNAVAILABLE for SSE connect, Streamable HTTP initialize, requests
// and resume; the JSON-RPC error with data.error "revocation_unavailable" for
// SSE messages and stdio, with sign-every-action off and on. Nothing is
// forwarded, spent or minted.

const spentMCPRoot = "spent-inprogress-mcp-root"

// countingRouter answers the tool list and counts forwarded tool calls.
type countingRouter struct{ calls atomic.Int64 }

func (r *countingRouter) AllToolsForTenant(context.Context, string) []json.RawMessage {
	return []json.RawMessage{json.RawMessage(`{"name":"echo","description":"echo","inputSchema":{"type":"object"}}`)}
}

func (r *countingRouter) ForwardToolCallForTenant(context.Context, string, string, json.RawMessage, time.Duration) (*mcpserver.Response, error) {
	r.calls.Add(1)
	return &mcpserver.Response{JSONRPC: "2.0", Result: json.RawMessage(`{"content":[{"type":"text","text":"ok"}],"isError":false}`)}, nil
}

// newMCPUpstream is a Streamable HTTP MCP server with one tool (echo). It
// counts every request other than the handshake and the tool list (work).
func newMCPUpstream(t *testing.T) (string, *atomic.Int64) {
	t.Helper()
	var work atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			w.WriteHeader(http.StatusMethodNotAllowed)
			return
		}
		var msg struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&msg); err != nil {
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		if len(msg.ID) == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		var result any
		switch msg.Method {
		case "initialize":
			result = map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{"tools": map[string]any{}},
				"serverInfo": map[string]string{"name": "fixture", "version": "1"}}
		case "tools/list":
			result = map[string]any{"tools": []any{map[string]any{"name": "echo", "description": "echo", "inputSchema": map[string]any{"type": "object"}}}}
		case "tools/call":
			work.Add(1)
			result = map[string]any{"content": []any{map[string]string{"type": "text", "text": "ok"}}, "isError": false}
		default:
			work.Add(1)
			result = map[string]any{"echo": msg.Params}
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(map[string]any{"jsonrpc": "2.0", "id": msg.ID, "result": result})
	}))
	t.Cleanup(srv.Close)
	return srv.URL, &work
}

// spentMCPKeys mints tokens under the MCP rigs' root key.
type spentMCPKeys struct {
	svc           *macaroon.Service
	root, rootSig string
	child         string // delegated from root
	// tenantChild is delegated from root with a tenant: an SSE session
	// opened with it has its identity from the connect, so the decision
	// of a message is the one its dispatch makes.
	tenantChild string
}

func newSpentMCPKeys(t *testing.T) spentMCPKeys {
	t.Helper()
	svc, err := macaroon.NewService(spentMCPRoot)
	if err != nil {
		t.Fatal(err)
	}
	m, err := svc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	root := svc.Encode(m)
	c, err := svc.Delegate(root, []string{"label = spent-in-progress"})
	if err != nil {
		t.Fatal(err)
	}
	tc, err := svc.Delegate(root, []string{"tenant_id = spent-tenant"})
	if err != nil {
		t.Fatal(err)
	}
	return spentMCPKeys{svc: svc, root: root, rootSig: m.Signature, child: svc.Encode(c), tenantChild: svc.Encode(tc)}
}

// fund gives tok's budget credits on budget and returns the budget id.
func (k spentMCPKeys) fund(t *testing.T, budget *mcpserver.InMemoryBudgetEnforcer, tok string, credits int64) string {
	t.Helper()
	info, err := (&mcpserver.MacaroonAuthenticator{Service: k.svc}).Verify(context.Background(), tok)
	if err != nil {
		t.Fatal(err)
	}
	if err := budget.Initialize(context.Background(), info.BudgetID, credits); err != nil {
		t.Fatal(err)
	}
	return info.BudgetID
}

func remaining(t *testing.T, budget *mcpserver.InMemoryBudgetEnforcer, id string) int64 {
	t.Helper()
	n, err := budget.Remaining(context.Background(), id)
	if err != nil {
		t.Fatal(err)
	}
	return n
}

func spentMCPConfig() *mcpserver.Config {
	return &mcpserver.Config{
		Server:      mcpserver.ServerConfig{Transport: "sse", Name: "spent-test", Version: "test"},
		Auth:        mcpserver.AuthConfig{Mode: "header", RootKey: spentMCPRoot},
		Budget:      mcpserver.BudgetConfig{Limit: 100, FailMode: "closed"},
		Tools:       mcpserver.ToolsConfig{DefaultCost: 1},
		Enforcement: mcpserver.EnforcementConfig{Mode: "control"},
		Logging:     mcpserver.LoggingConfig{Level: "error"},
	}
}

// spentNet is an MCP proxy on SSE and Streamable HTTP whose lineage checks use
// a spentBoundary's service.
type spentNet struct {
	proxy    *mcpserver.Proxy
	router   *countingRouter
	budget   *mcpserver.InMemoryBudgetEnforcer
	url      string
	sessions *sseConnects
}

// sseConnects is the proxy's event publisher: it records the SSE sessions
// whose connect has ended. The server sends the endpoint event before it
// checks the connect's token, and publishes session_connect after that check.
type sseConnects struct {
	mu   sync.Mutex
	done map[string]bool
}

func (c *sseConnects) Publish(e mcpserver.Event) {
	if e.Type != mcpserver.EventSessionConnect {
		return
	}
	c.mu.Lock()
	defer c.mu.Unlock()
	c.done[e.SessionID] = true
}

func (c *sseConnects) ended(session string) bool {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.done[session]
}

func newSpentNet(t *testing.T, b *spentBoundary, budget *mcpserver.InMemoryBudgetEnforcer) *spentNet {
	t.Helper()
	p, err := mcpserver.New(spentMCPConfig())
	if err != nil {
		t.Fatal(err)
	}
	router := &countingRouter{}
	p.SetUpstreamRouter(router)
	p.SetBudgetEnforcer(budget)
	p.SetLineageRevocationChecker(b.g)
	sessions := &sseConnects{done: map[string]bool{}}
	p.SetEventPublisher(sessions)
	srv := httptest.NewServer(mcpserver.NewSSEServer(p, ":0").Handler())
	t.Cleanup(srv.Close)
	return &spentNet{proxy: p, router: router, budget: budget, url: srv.URL, sessions: sessions}
}

func rpc(id int, method string, params map[string]any) map[string]any {
	return map[string]any{"jsonrpc": "2.0", "id": id, "method": method, "params": params}
}

func initializeMsg() map[string]any {
	return rpc(1, "initialize", map[string]any{"protocolVersion": "2025-03-26", "capabilities": map[string]any{},
		"clientInfo": map[string]string{"name": "test", "version": "1"}})
}

func toolCallMsg(id int) map[string]any {
	return rpc(id, "tools/call", map[string]any{"name": "echo", "arguments": map[string]any{}})
}

func delegateMsg(id int) map[string]any {
	return rpc(id, mcpserver.MethodSatGateDelegate, map[string]any{"budget": 0, "label": "helper", "scope": "api:read"})
}

// mcpPost posts msg to /mcp and returns the status, headers and body.
func mcpPost(t *testing.T, url, session, token string, msg any) (int, http.Header, string) {
	t.Helper()
	raw, _ := json.Marshal(msg)
	req, _ := http.NewRequest(http.MethodPost, url+"/mcp", bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	if session != "" {
		req.Header.Set("Mcp-Session-Id", session)
		req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	}
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return 0, nil, ""
	}
	defer resp.Body.Close()
	body, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, resp.Header, string(body)
}

func initSession(t *testing.T, url, token string) string {
	t.Helper()
	code, hdr, _ := mcpPost(t, url, "", token, initializeMsg())
	if code != http.StatusOK || hdr.Get("Mcp-Session-Id") == "" {
		t.Fatalf("initialize: status %d", code)
	}
	return hdr.Get("Mcp-Session-Id")
}

// sseConnect opens GET /sse and returns the status and the start of the body.
func sseConnect(t *testing.T, url, token string) (int, string) {
	t.Helper()
	req, _ := http.NewRequest(http.MethodGet, url+"/sse", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return 0, ""
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		return resp.StatusCode, ""
	}
	body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
	return resp.StatusCode, string(body)
}

// sseConn is an open SSE stream.
type sseConn struct {
	url, token, session string
	messages            chan json.RawMessage
	body                io.Closer
}

// openSSE opens an SSE stream on n and returns once the server has ended the
// connect, so the connect's own token check never reads the ban store during
// a later spentBoundary.cross: the read cross holds is the message's.
func openSSE(t *testing.T, n *spentNet, token string) *sseConn {
	t.Helper()
	url := n.url
	req, _ := http.NewRequest(http.MethodGet, url+"/sse", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil || resp.StatusCode != http.StatusOK {
		t.Fatalf("SSE connect: %v", err)
	}
	t.Cleanup(func() { _ = resp.Body.Close() })
	c := &sseConn{url: url, token: token, messages: make(chan json.RawMessage, 16), body: resp.Body}
	rd := bufio.NewReader(resp.Body)
	for c.session == "" {
		line, err := rd.ReadString('\n')
		if err != nil {
			t.Fatalf("SSE endpoint event: %v", err)
		}
		if line = strings.TrimSpace(line); strings.HasPrefix(line, "data: /message?sessionId=") {
			c.session = strings.TrimPrefix(line, "data: /message?sessionId=")
		}
	}
	for deadline := time.Now().Add(10 * time.Second); !n.sessions.ended(c.session); time.Sleep(time.Millisecond) {
		if time.Now().After(deadline) {
			t.Fatal("SSE connect did not end")
		}
	}
	go func() {
		for {
			line, err := rd.ReadString('\n')
			if err != nil {
				return
			}
			if line = strings.TrimSpace(line); strings.HasPrefix(line, "data: {") {
				c.messages <- json.RawMessage(strings.TrimPrefix(line, "data: "))
			}
		}
	}()
	return c
}

// send posts msg to /message and returns the next message on the stream.
func (c *sseConn) send(t *testing.T, msg any) *mcpserver.Response {
	raw, _ := json.Marshal(msg)
	req, _ := http.NewRequest(http.MethodPost, c.url+"/message?sessionId="+c.session, bytes.NewReader(raw))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Error(err)
		return nil
	}
	_ = resp.Body.Close()
	select {
	case m := <-c.messages:
		var out mcpserver.Response
		if err := json.Unmarshal(m, &out); err != nil {
			t.Error(err)
			return nil
		}
		return &out
	case <-time.After(10 * time.Second):
		t.Error("no answer on the SSE stream")
		return nil
	}
}

// rpcUnavailable reports whether resp is the JSON-RPC revocation-unavailable
// refusal.
func rpcUnavailable(resp *mcpserver.Response) bool {
	if resp == nil || resp.Error == nil || resp.Error.Code != mcpserver.CodeInternalError {
		return false
	}
	var data struct {
		Error string `json:"error"`
	}
	return json.Unmarshal(resp.Error.Data, &data) == nil && data.Error == "revocation_unavailable"
}

func rpcIn(body string) *mcpserver.Response {
	var resp mcpserver.Response
	if json.Unmarshal([]byte(body), &resp) != nil {
		return nil
	}
	return &resp
}

func helperIn(resp *mcpserver.Response) bool {
	return resp != nil && strings.Contains(string(resp.Result), `"token"`)
}

func TestSpentMCP_NetworkDecisionInProgressAnswersUnavailable(t *testing.T) {
	type outcome struct {
		status int    // HTTP status (not for an SSE message)
		body   string // HTTP body
		resp   *mcpserver.Response
	}
	cases := []struct {
		name    string
		wantRPC bool // the answer is (or carries) the JSON-RPC refusal
		sseOnly bool // the answer comes on the SSE stream, not as the HTTP response
		// prepare runs before the boundary; decide is the request in
		// progress across it.
		prepare func(t *testing.T, k spentMCPKeys, b *spentBoundary, a, other *spentNet) func() outcome
	}{
		{"sse_connect", false, false, func(t *testing.T, k spentMCPKeys, _ *spentBoundary, a, _ *spentNet) func() outcome {
			return func() outcome { s, body := sseConnect(t, a.url, k.child); return outcome{status: s, body: body} }
		}},
		{"sse_message", true, true, func(t *testing.T, k spentMCPKeys, _ *spentBoundary, a, _ *spentNet) func() outcome {
			conn := openSSE(t, a, k.tenantChild)
			return func() outcome { return outcome{resp: conn.send(t, toolCallMsg(2))} }
		}},
		{"sse_message_delegate", true, true, func(t *testing.T, k spentMCPKeys, _ *spentBoundary, a, _ *spentNet) func() outcome {
			conn := openSSE(t, a, k.tenantChild)
			return func() outcome { return outcome{resp: conn.send(t, delegateMsg(2))} }
		}},
		{"streamable_initialize", false, false, func(t *testing.T, k spentMCPKeys, _ *spentBoundary, a, _ *spentNet) func() outcome {
			return func() outcome {
				s, _, body := mcpPost(t, a.url, "", k.child, initializeMsg())
				return outcome{status: s, body: body}
			}
		}},
		{"streamable_request", true, false, func(t *testing.T, k spentMCPKeys, _ *spentBoundary, a, _ *spentNet) func() outcome {
			sid := initSession(t, a.url, k.child)
			return func() outcome {
				s, _, body := mcpPost(t, a.url, sid, k.child, toolCallMsg(2))
				return outcome{status: s, body: body, resp: rpcIn(body)}
			}
		}},
		{"streamable_delegate", true, false, func(t *testing.T, k spentMCPKeys, _ *spentBoundary, a, _ *spentNet) func() outcome {
			sid := initSession(t, a.url, k.child)
			return func() outcome {
				s, _, body := mcpPost(t, a.url, sid, k.child, delegateMsg(2))
				return outcome{status: s, body: body, resp: rpcIn(body)}
			}
		}},
		{"streamable_resume", false, false, func(t *testing.T, k spentMCPKeys, _ *spentBoundary, a, other *spentNet) func() outcome {
			sid := initSession(t, a.url, k.child)
			return func() outcome {
				s, _, body := mcpPost(t, other.url, sid, k.child, toolCallMsg(2))
				return outcome{status: s, body: body}
			}
		}},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			b := newSpentBoundaryAtMaximum(t)
			k := newSpentMCPKeys(t)
			budget := mcpserver.NewInMemoryBudgetEnforcer()
			a, other := newSpentNet(t, b, budget), newSpentNet(t, b, budget)
			id := k.fund(t, budget, k.child, 10)
			tenantID := k.fund(t, budget, k.tenantChild, 10)
			decide := tc.prepare(t, k, b, a, other)
			var got outcome
			b.cross(t, k.rootSig, func() { got = decide() })
			if !tc.sseOnly && (got.status != http.StatusServiceUnavailable || !strings.Contains(got.body, "REVOCATION_UNAVAILABLE")) {
				t.Errorf("%s in progress when the count was spent: status %d body %q, want 503 REVOCATION_UNAVAILABLE", tc.name, got.status, got.body)
			}
			if tc.wantRPC && !rpcUnavailable(got.resp) {
				t.Errorf("%s in progress when the count was spent: JSON-RPC answer %+v, want revocation_unavailable", tc.name, got.resp)
			}
			if helperIn(got.resp) {
				t.Errorf("%s: a helper token was minted", tc.name)
			}
			if n := a.router.calls.Load() + other.router.calls.Load(); n != 0 {
				t.Errorf("%s: %d tool calls forwarded", tc.name, n)
			}
			if n, m := remaining(t, budget, id), remaining(t, budget, tenantID); n != 10 || m != 10 {
				t.Errorf("%s: budgets %d and %d after the refusal, want 10", tc.name, n, m)
			}
			// A decision that starts now is refused the same way.
			if s, _, body := mcpPost(t, a.url, "", k.child, initializeMsg()); s != http.StatusServiceUnavailable || !strings.Contains(body, "REVOCATION_UNAVAILABLE") {
				t.Errorf("later initialize: status %d, want 503 REVOCATION_UNAVAILABLE", s)
			}
		})
	}
}

// spentStdio is an MCP proxy on stdio (Proxy.Run) with a default upstream,
// whose lineage checks use a spentBoundary's service.
type spentStdio struct {
	t      *testing.T
	budget *mcpserver.InMemoryBudgetEnforcer
	work   *atomic.Int64
	client *mcpserver.StdioTransport
	ctx    context.Context
	nextID int
}

// windowRecorder is an evidence recorder that keeps windows, so that
// sign-every-action can be on.
type windowRecorder struct {
	mu sync.Mutex
	n  int
}

func (r *windowRecorder) Preflight(context.Context) error { return nil }

func (r *windowRecorder) RecordMCPDecision(context.Context, mcpserver.MCPDecision) (*mcpserver.MCPEvidence, error) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.n++
	return &mcpserver.MCPEvidence{ReceiptID: fmt.Sprintf("rcpt_%d", r.n), ReceiptHash: fmt.Sprintf("sha256:%d", r.n),
		EvidencePackID: fmt.Sprintf("ep_%d", r.n), EvidenceURL: fmt.Sprintf("https://issuer.example/v1/evidence/evid_%d", r.n)}, nil
}

func (r *windowRecorder) RecordWindowedCall(ctx context.Context, call mcpserver.MCPWindowedCall) (*mcpserver.MCPEvidence, error) {
	return r.RecordMCPDecision(ctx, call.Decision)
}

func newSpentStdio(t *testing.T, b *spentBoundary, sign bool) *spentStdio {
	t.Helper()
	upURL, work := newMCPUpstream(t)
	cfg := spentMCPConfig()
	cfg.Server.Transport = "stdio"
	cfg.Upstreams = map[string]mcpserver.UpstreamConfig{"fixture": {Transport: "streamable", URL: upURL, Timeout: 10 * time.Second}}
	cfg.DefaultUpstream = "fixture"
	cfg.AllowPrivateUpstreams = true
	p, err := mcpserver.New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	budget := mcpserver.NewInMemoryBudgetEnforcer()
	p.SetBudgetEnforcer(budget)
	p.SetLineageRevocationChecker(b.g)
	if sign {
		p.SetEvidenceRecorder(&windowRecorder{})
		if err := p.SetSignEveryAction(true); err != nil {
			t.Fatal(err)
		}
	}
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()
	done := make(chan struct{})
	go func() {
		defer close(done)
		_ = p.Run(ctx, mcpserver.NewStdioTransport(serverRead, serverWrite, nil))
	}()
	t.Cleanup(func() {
		cancel()
		_ = clientWrite.Close()
		_ = clientRead.Close()
		_ = serverRead.Close()
		_ = serverWrite.Close()
		select {
		case <-done:
		case <-time.After(10 * time.Second):
			t.Error("stdio loop did not stop")
		}
	})
	return &spentStdio{t: t, budget: budget, work: work, client: mcpserver.NewStdioTransport(clientRead, clientWrite, nil), ctx: ctx}
}

// call sends one request with params._meta.token = token over stdio.
func (s *spentStdio) call(method, token string) *mcpserver.Response {
	s.nextID++
	params := map[string]any{"name": "echo", "arguments": map[string]any{}, "budget": 0, "label": "h", "uri": "file:///x",
		"_meta": map[string]string{"token": token}}
	raw, _ := json.Marshal(rpc(s.nextID, method, params))
	if err := s.client.WriteMessage(s.ctx, raw); err != nil {
		s.t.Error(err)
		return nil
	}
	answer, err := s.client.ReadMessage(s.ctx)
	if err != nil {
		s.t.Error(err)
		return nil
	}
	var resp mcpserver.Response
	if err := json.Unmarshal(answer, &resp); err != nil {
		s.t.Error(err)
		return nil
	}
	return &resp
}

// stdioCheckedMethods are the stdio methods that check a supplied token:
// tools/list, the authenticated methods and methods passed on to the default
// upstream.
var stdioCheckedMethods = []string{"tools/list", "tools/call", mcpserver.MethodSatGateDelegate, mcpserver.MethodSatGateBudget, "budget/check",
	"resources/list", "resources/read", "prompts/list", "prompts/get", "completion/complete", "vendor/custom"}

func signName(sign bool) string {
	if sign {
		return "sign_every_action_on"
	}
	return "sign_every_action_off"
}

func TestSpentMCP_StdioDecisionInProgressAnswersUnavailable(t *testing.T) {
	// The control: with a healthy token the fixture upstream is reached, so
	// "no upstream work" below is observed, not assumed.
	t.Run("control_healthy_token_is_passed_on", func(t *testing.T) {
		b := newSpentBoundaryAtMaximum(t)
		k := newSpentMCPKeys(t)
		s := newSpentStdio(t, b, false)
		if resp := s.call("resources/list", k.child); resp == nil || resp.Error != nil || s.work.Load() != 1 {
			t.Fatalf("healthy token: answer %+v, upstream work %d; want passed on once", resp, s.work.Load())
		}
	})
	for _, sign := range []bool{false, true} {
		for _, method := range stdioCheckedMethods {
			t.Run(signName(sign)+"/"+method, func(t *testing.T) {
				t.Parallel()
				b := newSpentBoundaryAtMaximum(t)
				k := newSpentMCPKeys(t)
				s := newSpentStdio(t, b, sign)
				id := k.fund(t, s.budget, k.child, 10)
				var resp *mcpserver.Response
				b.cross(t, k.rootSig, func() { resp = s.call(method, k.child) })
				if !rpcUnavailable(resp) {
					t.Errorf("%s in progress when the count was spent: answer %+v, want revocation_unavailable", method, resp)
				}
				if helperIn(resp) {
					t.Errorf("%s: a helper token was minted", method)
				}
				if n := s.work.Load(); n != 0 {
					t.Errorf("%s: upstream work %d", method, n)
				}
				if n := remaining(t, s.budget, id); n != 10 {
					t.Errorf("%s: budget %d after the refusal, want 10", method, n)
				}
				if resp := s.call(method, k.child); !rpcUnavailable(resp) {
					t.Errorf("later %s: answer %+v, want revocation_unavailable", method, resp)
				}
			})
		}
	}
}

// Sol's MCP case: a decision crossing into the spent state (ban of the root
// covering the presented token, cache turned over) on Streamable HTTP
// initialize, SSE connect, a Streamable HTTP request, an SSE message and
// stdio tools/list is answered unavailable, not as revoked, and nothing is
// forwarded or charged.
func TestSpentMCP_DecisionCrossingIntoSpentStateProjectsUnavailable(t *testing.T) {
	for _, kind := range []string{"streamable_initialize", "sse_connect", "streamable_call", "sse_message", "stdio"} {
		t.Run(kind, func(t *testing.T) {
			b := newSpentBoundaryAtMaximum(t)
			k := newSpentMCPKeys(t)
			budget := mcpserver.NewInMemoryBudgetEnforcer()
			n := newSpentNet(t, b, budget)
			id := k.fund(t, budget, k.child, 10)
			var decide func() bool
			switch kind {
			case "streamable_initialize":
				decide = func() bool {
					s, _, body := mcpPost(t, n.url, "", k.child, initializeMsg())
					return s == http.StatusServiceUnavailable && strings.Contains(body, "REVOCATION_UNAVAILABLE")
				}
			case "sse_connect":
				decide = func() bool { s, _ := sseConnect(t, n.url, k.child); return s == http.StatusServiceUnavailable }
			case "streamable_call":
				sid := initSession(t, n.url, k.child)
				decide = func() bool {
					s, _, body := mcpPost(t, n.url, sid, k.child, toolCallMsg(2))
					return s == http.StatusServiceUnavailable && rpcUnavailable(rpcIn(body))
				}
			case "sse_message":
				conn := openSSE(t, n, k.child)
				decide = func() bool { return rpcUnavailable(conn.send(t, toolCallMsg(3))) }
			case "stdio":
				s := newSpentStdio(t, b, false)
				decide = func() bool { return rpcUnavailable(s.call("tools/list", k.child)) }
			}
			var unavailable bool
			b.cross(t, k.rootSig, func() { unavailable = decide() })
			if !unavailable {
				t.Error("decision crossing into the spent state was not answered unavailable")
			}
			if n.router.calls.Load() != 0 || remaining(t, budget, id) != 10 {
				t.Error("crossing decision forwarded or charged")
			}
		})
	}
}

// Astra's MCP case: the ban that spends the count is made while the
// decision's store read is in progress (no cache turnover). Over stdio, with
// sign-every-action off and on, every authenticated method and every method
// passed on is answered unavailable, and so is the next call; Streamable HTTP
// initialize and SSE connect are answered 503, with nothing forwarded.
func TestSpentMCP_BanDuringStoreReadStdioAndConnectReportUnavailable(t *testing.T) {
	methods := []string{"tools/call", mcpserver.MethodSatGateDelegate, mcpserver.MethodSatGateBudget, "budget/check",
		"resources/list", "resources/read", "prompts/list", "prompts/get", "completion/complete", "vendor/custom"}
	for _, sign := range []bool{false, true} {
		for _, method := range methods {
			t.Run(signName(sign)+"/"+method, func(t *testing.T) {
				t.Parallel()
				b := newSpentBoundaryAtMaximum(t)
				k := newSpentMCPKeys(t)
				s := newSpentStdio(t, b, sign)
				var resp *mcpserver.Response
				b.crossWith(t, k.rootSig, false, func() { resp = s.call(method, k.child) })
				if !rpcUnavailable(resp) {
					t.Error("stdio decision in progress when the count was spent returned a plain denial rather than unavailable")
				}
				if resp := s.call(method, k.child); !rpcUnavailable(resp) {
					t.Error("later stdio decision did not return unavailable")
				}
				if s.work.Load() != 0 {
					t.Error("refused request reached the upstream")
				}
			})
		}
	}
	for _, transport := range []string{"streamable", "sse"} {
		t.Run(transport, func(t *testing.T) {
			b := newSpentBoundaryAtMaximum(t)
			k := newSpentMCPKeys(t)
			n := newSpentNet(t, b, mcpserver.NewInMemoryBudgetEnforcer())
			var code int
			b.crossWith(t, k.rootSig, false, func() {
				if transport == "streamable" {
					code, _, _ = mcpPost(t, n.url, "", k.child, initializeMsg())
				} else {
					code, _ = sseConnect(t, n.url, k.child)
				}
			})
			if code != http.StatusServiceUnavailable {
				t.Errorf("%s in progress when the count was spent: want HTTP 503, got %d", transport, code)
			}
			if n.router.calls.Load() != 0 {
				t.Error("refused request forwarded")
			}
		})
	}
}
