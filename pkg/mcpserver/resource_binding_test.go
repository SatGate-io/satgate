package mcpserver

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

const resourceTestRootKey = "resource-binding-test-root-key"

func resourceTestToken(t *testing.T, caveats ...string) string {
	t.Helper()
	svc, err := macaroon.NewService(resourceTestRootKey)
	if err != nil {
		t.Fatal(err)
	}
	mac, err := svc.Mint("*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	tok := svc.Encode(mac)
	if len(caveats) > 0 {
		child, err := svc.Delegate(tok, caveats)
		if err != nil {
			t.Fatal(err)
		}
		tok = svc.Encode(child)
	}
	return tok
}

// newResourceServer builds a header-mode server with the real macaroon
// verifier and the given resource ("" = none configured).
func newResourceServer(t *testing.T, resource string, opts ...SSEOption) *SSEServer {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "resource-test", Version: "test"},
		Auth:        AuthConfig{Mode: "header", RootKey: resourceTestRootKey},
		Budget:      BudgetConfig{Limit: 100, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetUpstreamRouter(&countingForwardRouter{})
	if resource != "" {
		opts = append(opts, WithResourceBinding(resource))
	}
	return NewSSEServer(proxy, ":0", opts...)
}

func TestCheckResourceCaveats(t *testing.T) {
	const res = "https://mcp.example.test/mcp"
	cases := []struct {
		name     string
		caveats  []string
		resource string
		wantErr  bool
	}{
		{"no caveat, no resource", []string{"scope = *"}, "", false},
		{"no caveat, resource set", []string{"scope = *"}, res, false},
		{"exact", []string{"resource = " + res}, res, false},
		{"exact repeated", []string{"resource = " + res, "resource = " + res}, res, false},
		{"fail closed without resource", []string{"resource = " + res}, "", true},
		{"two different", []string{"resource = " + res, "resource = https://other.test/mcp"}, res, true},
		{"trailing slash", []string{"resource = " + res + "/"}, res, true},
		{"upper-case host", []string{"resource = https://MCP.example.test/mcp"}, res, true},
		{"explicit :443", []string{"resource = https://mcp.example.test:443/mcp"}, res, true},
		{"query", []string{"resource = " + res + "?a=1"}, res, true},
		{"fragment", []string{"resource = " + res + "#f"}, res, true},
		{"http", []string{"resource = http://mcp.example.test/mcp"}, res, true},
		{"punycode vs idn", []string{"resource = https://xn--bcher-kva.example/mcp"}, "https://bücher.example/mcp", true},
		{"idn vs punycode", []string{"resource = https://bücher.example/mcp"}, "https://xn--bcher-kva.example/mcp", true},
		{"idn exact", []string{"resource = https://bücher.example/mcp"}, "https://bücher.example/mcp", false},
		{"extra path", []string{"resource = " + res + "/extra"}, res, true},
		{"empty value", []string{"resource = "}, res, true},
		{"other key with prefix", []string{"resources = x"}, res, false},
	}
	for _, c := range cases {
		err := CheckResourceCaveats(c.caveats, c.resource)
		if (err != nil) != c.wantErr {
			t.Errorf("%s: err=%v wantErr=%v", c.name, err, c.wantErr)
		}
	}
}

// A token for resource A is refused by server B on initialize and on
// tools/call, on Streamable HTTP and on legacy SSE, with the challenge.
func TestResourceBindingRefusesOtherResourceOnBothTransports(t *testing.T) {
	const a = "https://a.example.test/mcp"
	const b = "https://b.example.test/mcp"
	tokA := resourceTestToken(t, "resource = "+a)
	tokPlain := resourceTestToken(t)

	sse := newResourceServer(t, b, WithOAuthChallenge(testMetadataURL))
	srv := httptest.NewServer(sse.Handler())
	defer srv.Close()

	// Streamable: initialize with A's token on B.
	status, hdr, _ := streamablePost(t, srv.URL+"/mcp", "", tokA, initializeBody("2025-03-26"))
	if status != http.StatusUnauthorized || hdr.Get("WWW-Authenticate") != wantChallenge {
		t.Fatalf("initialize: status=%d challenge=%q", status, hdr.Get("WWW-Authenticate"))
	}
	if n := streamSessionCount(sse); n != 0 {
		t.Fatalf("refused initialize left %d sessions", n)
	}
	// An unrelated plain token opens a session; A's token then fails on it
	// for every method, not only tools/call.
	sid := mustInitSession(t, srv.URL, tokPlain)
	for _, msg := range []map[string]any{
		toolsListBody(),
		{"jsonrpc": "2.0", "id": 8, "method": "ping"},
		{"jsonrpc": "2.0", "id": 9, "method": "tools/call", "params": map[string]any{"name": "read_file", "arguments": map[string]any{}}},
	} {
		status, hdr, _ = streamablePost(t, srv.URL+"/mcp", sid, tokA, msg)
		if status != http.StatusUnauthorized || hdr.Get("WWW-Authenticate") != wantChallenge {
			t.Fatalf("method %v: status=%d challenge=%q", msg["method"], status, hdr.Get("WWW-Authenticate"))
		}
	}
	// DELETE too.
	req, _ := http.NewRequest(http.MethodDelete, srv.URL+"/mcp", nil)
	req.Header.Set("Mcp-Session-Id", sid)
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	req.Header.Set("Authorization", "Bearer "+tokA)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("delete: status=%d", resp.StatusCode)
	}

	// Legacy SSE: connect and message.
	sreq, _ := http.NewRequest(http.MethodGet, srv.URL+"/sse", nil)
	sreq.Header.Set("Authorization", "Bearer "+tokA)
	sresp, err := http.DefaultClient.Do(sreq)
	if err != nil {
		t.Fatal(err)
	}
	sresp.Body.Close()
	if sresp.StatusCode != http.StatusUnauthorized || sresp.Header.Get("WWW-Authenticate") != wantChallenge {
		t.Fatalf("sse connect: status=%d challenge=%q", sresp.StatusCode, sresp.Header.Get("WWW-Authenticate"))
	}
	// SSE message endpoint with a session opened by a plain token.
	sseSessionID := "sess-resource-test"
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	sse.mu.Lock()
	sse.sessions[sseSessionID] = &sseSession{id: sseSessionID, messages: make(chan json.RawMessage, 8), ctx: ctx, cancel: cancel}
	sse.mu.Unlock()
	for _, method := range []string{"initialize", "tools/call", "tools/list"} {
		body := `{"jsonrpc":"2.0","id":3,"method":"` + method + `","params":{"name":"read_file"}}`
		mreq, _ := http.NewRequest(http.MethodPost, srv.URL+"/message?sessionId="+sseSessionID, strings.NewReader(body))
		mreq.Header.Set("Authorization", "Bearer "+tokA)
		mresp, err := http.DefaultClient.Do(mreq)
		if err != nil {
			t.Fatal(err)
		}
		mresp.Body.Close()
		if mresp.StatusCode != http.StatusUnauthorized {
			t.Fatalf("sse message %s: status=%d", method, mresp.StatusCode)
		}
	}
}

// The matching resource passes; a token with no resource caveat behaves as
// before on a server with and without a resource.
func TestResourceBindingAcceptsOwnResourceAndPlainTokens(t *testing.T) {
	const b = "https://b.example.test/mcp"
	tokB := resourceTestToken(t, "resource = "+b)
	tokPlain := resourceTestToken(t)

	with := httptest.NewServer(newResourceServer(t, b).Handler())
	defer with.Close()
	without := httptest.NewServer(newResourceServer(t, "").Handler())
	defer without.Close()

	for name, tc := range map[string]struct {
		url, tok string
		want     int
	}{
		"own resource":              {with.URL, tokB, http.StatusOK},
		"plain token, resource set": {with.URL, tokPlain, http.StatusOK},
		"plain token, none set":     {without.URL, tokPlain, http.StatusOK},
		"resource token, none set":  {without.URL, tokB, http.StatusUnauthorized},
	} {
		status, _, body := streamablePost(t, tc.url+"/mcp", "", tc.tok, initializeBody("2025-03-26"))
		if status != tc.want {
			t.Errorf("%s: status=%d want %d body=%q", name, status, tc.want, body)
		}
	}
}

// Challenge off: the refusal is the historical bare 401, no header.
func TestResourceBindingRefusalWithoutChallengeIsBare401(t *testing.T) {
	tok := resourceTestToken(t, "resource = https://a.example.test/mcp")
	srv := httptest.NewServer(newResourceServer(t, "https://b.example.test/mcp").Handler())
	defer srv.Close()
	status, hdr, body := streamablePost(t, srv.URL+"/mcp", "", tok, initializeBody("2025-03-26"))
	if status != http.StatusUnauthorized || body != "authentication failed\n" {
		t.Fatalf("status=%d body=%q", status, body)
	}
	if _, present := hdr["Www-Authenticate"]; present {
		t.Fatalf("unexpected challenge %q", hdr.Get("WWW-Authenticate"))
	}
}

// A child token made with Delegate keeps the resource caveat, and cannot
// swap it for another: a second, different caveat fails the check.
func TestResourceBindingChildKeepsAndCannotSwap(t *testing.T) {
	const a = "https://a.example.test/mcp"
	const b = "https://b.example.test/mcp"
	svc, _ := macaroon.NewService(resourceTestRootKey)
	parent := resourceTestToken(t, "resource = "+a)
	child, err := svc.Delegate(parent, []string{"scope = read_file"})
	if err != nil {
		t.Fatal(err)
	}
	if err := CheckResourceCaveats(child.Caveats, a); err != nil {
		t.Fatalf("child lost the caveat: %v", err)
	}
	if err := CheckResourceCaveats(child.Caveats, b); err == nil {
		t.Fatal("child accepted by another resource")
	}
	swapped, err := svc.Delegate(parent, []string{"resource = " + b})
	if err == nil {
		if CheckResourceCaveats(swapped.Caveats, b) == nil || CheckResourceCaveats(swapped.Caveats, a) == nil {
			t.Fatal("a swapped resource caveat was accepted")
		}
	}
}
