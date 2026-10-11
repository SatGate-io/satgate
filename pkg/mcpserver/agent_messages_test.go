package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"
)

// plainRouter has neither optional seam: it is every router written before
// agent_messages.go, and the default router's shape.
type plainRouter struct {
	tools   []json.RawMessage
	callErr error
}

func (r *plainRouter) AllToolsForTenant(context.Context, string) []json.RawMessage { return r.tools }
func (r *plainRouter) ForwardToolCallForTenant(context.Context, string, string, json.RawMessage, time.Duration) (*Response, error) {
	return nil, r.callErr
}

func newMessagesProxy(t *testing.T, router UpstreamRouter) *Proxy {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "msg-test", Version: "test"},
		Auth:        AuthConfig{Mode: "none"},
		Budget:      BudgetConfig{Limit: 10, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "observe"},
		Logging:     LoggingConfig{Level: "error"},
	}
	p, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	if router != nil {
		p.SetUpstreamRouter(router)
	}
	return p
}

func proxyReplyJSON(t *testing.T, p *Proxy, method string) string {
	t.Helper()
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`7`), Method: method, Params: json.RawMessage(`{"name":"echo","arguments":{}}`)}
	resp, err := p.handleRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("%s: %v", method, err)
	}
	b, _ := json.Marshal(resp)
	return string(b)
}

// A router with neither seam is answered with exactly the bytes the proxy has
// always written. These strings are what origin/main produces; the same test
// body passes there (it uses nothing this change adds).
func TestPlainRouterBytesAreUnchanged(t *testing.T) {
	boom := &plainRouter{
		tools:   []json.RawMessage{json.RawMessage(`{"name":"echo","description":"d","inputSchema":{"type":"object"}}`)},
		callErr: errors.New("boom"),
	}
	p := newMessagesProxy(t, boom)
	if got, want := proxyReplyJSON(t, p, MethodToolsCall), `{"jsonrpc":"2.0","id":7,"error":{"code":-32002,"message":"upstream error: boom"}}`; got != want {
		t.Fatalf("tools/call:\n got %s\nwant %s", got, want)
	}
	if got, want := proxyReplyJSON(t, p, MethodToolsList), `{"jsonrpc":"2.0","id":7,"result":{"tools":[{"name":"echo","description":"d","inputSchema":{"type":"object"}}]}}`; got != want {
		t.Fatalf("tools/list:\n got %s\nwant %s", got, want)
	}
	empty := newMessagesProxy(t, &plainRouter{})
	if got, want := proxyReplyJSON(t, empty, MethodToolsList), `{"jsonrpc":"2.0","id":7,"result":{"tools":[]}}`; got != want {
		t.Fatalf("empty tools/list:\n got %s\nwant %s", got, want)
	}
}

// The default router (no SetUpstreamRouter) answers as before.
func TestDefaultRouterBytesAreUnchanged(t *testing.T) {
	p := newMessagesProxy(t, nil)
	if got, want := proxyReplyJSON(t, p, MethodToolsList), `{"jsonrpc":"2.0","id":7,"result":{"tools":[]}}`; got != want {
		t.Fatalf("default tools/list:\n got %s\nwant %s", got, want)
	}
}

// seamRouter implements both optional interfaces.
type seamRouter struct {
	plainRouter
	listErr error
	words   map[error]string
}

func (r *seamRouter) ToolsForTenant(context.Context, string) ([]json.RawMessage, error) {
	if r.listErr != nil {
		return nil, r.listErr
	}
	return r.tools, nil
}

func (r *seamRouter) AgentMessage(err error) (string, bool) {
	m, ok := r.words[err]
	return m, ok
}

func TestAgentMessengerReplacesTheMessageOnly(t *testing.T) {
	known := errors.New("internal detail")
	r := &seamRouter{plainRouter: plainRouter{callErr: known}, words: map[error]string{known: "Exactly this."}}
	p := newMessagesProxy(t, r)
	got := proxyReplyJSON(t, p, MethodToolsCall)
	if want := `{"jsonrpc":"2.0","id":7,"error":{"code":-32002,"message":"Exactly this."}}`; got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	// An error the router does not recognise keeps today's bytes.
	r.callErr = errors.New("other")
	got = proxyReplyJSON(t, p, MethodToolsCall)
	if want := `{"jsonrpc":"2.0","id":7,"error":{"code":-32002,"message":"upstream error: other"}}`; got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}

func TestTenantToolsListerCanFailToolsList(t *testing.T) {
	known := errors.New("internal detail")
	r := &seamRouter{listErr: known, words: map[error]string{known: "Exactly this."}}
	p := newMessagesProxy(t, r)
	got := proxyReplyJSON(t, p, MethodToolsList)
	if want := `{"jsonrpc":"2.0","id":7,"error":{"code":-32002,"message":"Exactly this."}}`; got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	// Failing without a message for the error: today's prefix.
	r.listErr = errors.New("other")
	got = proxyReplyJSON(t, p, MethodToolsList)
	if want := `{"jsonrpc":"2.0","id":7,"error":{"code":-32002,"message":"upstream error: other"}}`; got != want {
		t.Fatalf("got %s want %s", got, want)
	}
	// No error: the list, through the same path.
	r.listErr = nil
	r.tools = []json.RawMessage{json.RawMessage(`{"name":"echo"}`)}
	got = proxyReplyJSON(t, p, MethodToolsList)
	if want := `{"jsonrpc":"2.0","id":7,"result":{"tools":[{"name":"echo"}]}}`; got != want {
		t.Fatalf("got %s want %s", got, want)
	}
}
