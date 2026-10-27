package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/rs/zerolog"
	zlog "github.com/rs/zerolog/log"
)

// More tests for upstream_hygiene.go: an upstream configured with
// forwardCallerToken receives only the credential the proxy verified for the
// request, never a value from params; the token query parameter is refused by
// its presence; and what the proxy forwards keeps every byte it does not need
// to remove.

const (
	// callerWrittenValue is a value a caller puts into params._meta.token
	// that the test authenticator does not accept.
	callerWrittenValue = "caller-written-unverified-value"
	// otherConnectionCredential is a connection credential the test
	// authenticator does not accept.
	otherConnectionCredential = "connection-credential-not-verifiable"
)

// rawPost sends body as it is to base+path with the given HTTP method,
// optional session id and optional bearer credential.
func rawPost(t *testing.T, base, sid, bearer, method, path, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(method, base+path, strings.NewReader(body))
	if err != nil {
		t.Fatalf("build request: %v", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	if sid != "" {
		req.Header.Set("Mcp-Session-Id", sid)
	}
	if bearer != "" {
		req.Header.Set("Authorization", "Bearer "+bearer)
	}
	resp, err := (&http.Client{Timeout: 5 * time.Second}).Do(req)
	if err != nil {
		t.Fatalf("%s %s: %v", method, path, err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// streamableCall posts raw on a Streamable HTTP session and returns the
// JSON-RPC response.
func streamableCall(t *testing.T, base, sid, bearer, raw string) *Response {
	t.Helper()
	status, body := rawPost(t, base, sid, bearer, http.MethodPost, "/mcp", raw)
	var resp Response
	if status != http.StatusOK || json.Unmarshal([]byte(body), &resp) != nil {
		t.Fatalf("streamable status %d: %s", status, redactToken(json.RawMessage(body)))
	}
	return &resp
}

func wantForwarded(t *testing.T, what string, params json.RawMessage, want string) {
	t.Helper()
	got, has := metaTokenOf(t, params)
	switch {
	case want == "" && has:
		t.Errorf("%s: upstream received a credential, want none: %s", what, redactToken(params))
	case want != "" && (!has || got != want):
		t.Errorf("%s: upstream did not receive the verified credential: %s", what, redactToken(params))
	}
	if strings.Contains(string(params), callerWrittenValue) {
		t.Errorf("%s: upstream received the value the caller wrote into _meta", what)
	}
}

// With forwardCallerToken on, the upstream receives the credential the proxy
// verified for the request, not the value a caller wrote into _meta.token,
// for tools/call and for every method passed on to the default upstream, over
// both HTTP transports. The rest of _meta is passed on. Off, it receives none.
func TestForwardedCredentialIsTheVerifiedOne(t *testing.T) {
	methods := []string{"resources/list", "resources/read", "prompts/list", "prompts/get", "completion/complete", "logging/setLevel", "custom/method"}
	for _, forward := range []bool{false, true} {
		want := ""
		if forward {
			want = hygieneToken
		}
		p := newTokenProbe(t, forward)
		sid := mustInitSession(t, p.srv.URL, hygieneToken)
		conn := eaOpenSSE(t, p.srv.URL, hygieneToken)
		for _, method := range methods {
			raw := `{"jsonrpc":"2.0","id":500,"method":"` + method + `","params":{"_meta":{"token":"` + callerWrittenValue + `","progressToken":7}}}`
			what := fmt.Sprintf("forward=%v streamable %s", forward, method)
			params := receivedParams(t, method, streamableCall(t, p.srv.URL, sid, hygieneToken, raw))
			wantForwarded(t, what, params, want)
			if !strings.Contains(string(params), `"progressToken":7`) {
				t.Errorf("%s: the rest of _meta was not passed on: %s", what, redactToken(params))
			}
			if status := conn.post([]byte(raw)); status != http.StatusAccepted {
				t.Fatalf("POST /message status %d", status)
			}
			what = fmt.Sprintf("forward=%v sse %s", forward, method)
			wantForwarded(t, what, receivedParams(t, method, conn.next()), want)
		}
		// tools/call with no _meta: the transport's credential, verified by
		// authenticate.
		raw := `{"jsonrpc":"2.0","id":501,"method":"tools/call","params":{"name":"report","arguments":{}}}`
		wantForwarded(t, fmt.Sprintf("forward=%v streamable tools/call", forward),
			receivedParams(t, MethodToolsCall, streamableCall(t, p.srv.URL, sid, hygieneToken, raw)), want)
		if status := conn.post([]byte(raw)); status != http.StatusAccepted {
			t.Fatalf("POST /message status %d", status)
		}
		wantForwarded(t, fmt.Sprintf("forward=%v sse tools/call", forward), receivedParams(t, MethodToolsCall, conn.next()), want)
		_ = conn.body.Close()
	}
}

// A Streamable HTTP session resumed on another proxy (and the same call
// repeated) forwards the verified credential, not the caller's value.
func TestForwardedCredentialResumedStream(t *testing.T) {
	before := newTokenProbe(t, true)
	sid := mustInitSession(t, before.srv.URL, hygieneToken)
	after := newTokenProbe(t, true)
	for i := 0; i < 2; i++ {
		for _, method := range []string{"resources/list", MethodToolsCall} {
			raw := `{"jsonrpc":"2.0","id":502,"method":"` + method + `","params":{"name":"report","arguments":{},"_meta":{"token":"` + callerWrittenValue + `","progressToken":9}}}`
			if method == MethodToolsCall {
				raw = `{"jsonrpc":"2.0","id":502,"method":"tools/call","params":{"name":"report","arguments":{},"_meta":{"progressToken":9}}}`
			}
			params := receivedParams(t, method, streamableCall(t, after.srv.URL, sid, hygieneToken, raw))
			wantForwarded(t, fmt.Sprintf("resumed %s #%d", method, i), params, hygieneToken)
		}
	}
}

// A request whose connection credential does not verify, or that has none,
// forwards no credential, even when params carry one that would verify:
// params are never where the forwarded credential comes from.
func TestNoCredentialForwardedWithoutVerification(t *testing.T) {
	p := newTokenProbe(t, true)
	conn := eaOpenSSE(t, p.srv.URL, hygieneToken)
	defer conn.body.Close()
	for _, bearer := range []string{otherConnectionCredential, ""} {
		conn.token = bearer
		raw := `{"jsonrpc":"2.0","id":503,"method":"resources/list","params":{"_meta":{"token":"` + hygieneToken + `"}}}`
		if status := conn.post([]byte(raw)); status != http.StatusAccepted {
			t.Fatalf("POST /message status %d", status)
		}
		wantForwarded(t, fmt.Sprintf("sse bearer=%v", bearer != ""), receivedParams(t, "resources/list", conn.next()), "")
	}

	// Direct dispatch (stdio transport): no connection credential, so a
	// method passed on unauthenticated forwards none; tools/call forwards
	// the credential authenticate verified.
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`504`), Method: "resources/list",
		Params: json.RawMessage(`{"_meta":{"token":"` + hygieneToken + `"}}`)}
	resp, err := p.proxy.handleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	wantForwarded(t, "direct resources/list", receivedParams(t, "resources/list", resp), "")
	req = &Request{JSONRPC: "2.0", ID: json.RawMessage(`505`), Method: MethodToolsCall,
		Params: json.RawMessage(`{"name":"report","arguments":{},"_meta":{"token":"` + hygieneToken + `"}}`)}
	if resp, err = p.proxy.handleRequest(context.Background(), req); err != nil {
		t.Fatal(err)
	}
	wantForwarded(t, "direct tools/call", receivedParams(t, MethodToolsCall, resp), hygieneToken)
	// The connection credential on the context, a caller's value in params.
	req = &Request{JSONRPC: "2.0", ID: json.RawMessage(`506`), Method: "resources/list",
		Params: json.RawMessage(`{"_meta":{"token":"` + callerWrittenValue + `"}}`)}
	if resp, err = p.proxy.handleRequest(withTransportToken(context.Background(), hygieneToken), req); err != nil {
		t.Fatal(err)
	}
	wantForwarded(t, "direct resources/list with connection credential", receivedParams(t, "resources/list", resp), hygieneToken)
}

type revokeEveryKey struct{}

func (revokeEveryKey) IsRevoked(context.Context, string) bool { return true }

// A connection credential that verifies but is revoked is not forwarded.
func TestNoCredentialForwardedWhenRevoked(t *testing.T) {
	p := newTokenProbe(t, true)
	sid := mustInitSession(t, p.srv.URL, hygieneToken)
	p.proxy.SetRevocationChecker(revokeEveryKey{})
	raw := `{"jsonrpc":"2.0","id":507,"method":"resources/list","params":{}}`
	wantForwarded(t, "revoked", receivedParams(t, "resources/list", streamableCall(t, p.srv.URL, sid, hygieneToken, raw)), "")
}

// Network upstreams (SSE and Streamable HTTP) get the verified credential
// from the context only, never the one in params, and every other value as
// written.
func TestNetworkUpstreamsGetOnlyVerifiedCredential(t *testing.T) {
	for _, kind := range []string{"sse", "streamable"} {
		for _, forward := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/forward=%v", kind, forward), func(t *testing.T) {
				events := make(chan []byte, 32)
				handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodGet {
						w.Header().Set("Content-Type", "text/event-stream")
						if kind == "sse" {
							fmt.Fprint(w, "event: endpoint\ndata: /message\n\n")
						} else {
							fmt.Fprint(w, ": keepalive\n\n")
						}
						w.(http.Flusher).Flush()
						for {
							select {
							case <-r.Context().Done():
								return
							case ev := <-events:
								fmt.Fprintf(w, "event: message\ndata: %s\n\n", ev)
								w.(http.Flusher).Flush()
							}
						}
					}
					if r.Method == http.MethodDelete {
						w.WriteHeader(http.StatusNoContent)
						return
					}
					var req Request
					if json.NewDecoder(r.Body).Decode(&req) != nil {
						w.WriteHeader(http.StatusBadRequest)
						return
					}
					if len(req.ID) == 0 {
						w.WriteHeader(http.StatusNoContent)
						return
					}
					var result any
					switch req.Method {
					case MethodInitialize:
						result = map[string]any{"protocolVersion": "2024-11-05", "capabilities": map[string]any{"tools": map[string]any{}},
							"serverInfo": map[string]string{"name": "echo", "version": "1"}}
					case MethodToolsList:
						result = map[string]any{"tools": []any{map[string]any{"name": "report", "inputSchema": map[string]string{"type": "object"}}}}
					default:
						result = map[string]any{"echo": req.Params}
					}
					envelope := map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result}
					if kind == "sse" {
						b, _ := json.Marshal(envelope)
						events <- b
						w.WriteHeader(http.StatusAccepted)
						return
					}
					w.Header().Set("Content-Type", "application/json")
					w.Header().Set("Mcp-Session-Id", "test-session")
					_ = json.NewEncoder(w).Encode(envelope)
				})
				srv := httptest.NewServer(handler)
				defer srv.Close()
				path := "/mcp"
				if kind == "sse" {
					path = "/sse"
				}
				cfg := UpstreamConfig{Transport: kind, URL: srv.URL + path, ForwardCallerToken: forward}
				m := NewUpstreamManager(map[string]UpstreamConfig{"probe": cfg}, nil, "probe", true)
				ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
				defer cancel()
				if err := m.Start(ctx); err != nil {
					t.Fatalf("network upstream start: %v", err)
				}
				defer m.Close()
				want := ""
				if forward {
					want = hygieneToken
				}
				for i := 0; i < 2; i++ {
					resp, err := m.ForwardRequest(withVerifiedCredential(ctx, hygieneToken), "resources/list",
						json.RawMessage(`{"_meta":{"TOKEN":"`+callerWrittenValue+`","token":"`+callerWrittenValue+`","progressToken":9007199254740993},"n":1.50}`), 3*time.Second)
					if err != nil {
						t.Fatalf("network forward: %v", err)
					}
					params := receivedParams(t, "resources/list", resp)
					wantForwarded(t, "network", params, want)
					if !bytes.Contains(params, []byte(`"progressToken":9007199254740993`)) || !bytes.Contains(params, []byte(`"n":1.50`)) {
						t.Errorf("other values changed: %s", redactToken(params))
					}
				}
			})
		}
	}
}

// By default nothing a caller puts in _meta.token, in any spelling, reaches
// the upstream, on any method, transport, resumed session or repeated call;
// numbers keep their bytes.
func TestDefaultUpstreamWritesHaveNoCredential(t *testing.T) {
	p := newTokenProbe(t, false)
	sid := mustInitSession(t, p.srv.URL, hygieneToken)
	variants := []string{
		`{"token":"` + hygieneToken + `","progressToken":9007199254740993}`,
		`{"ToKeN":"` + hygieneToken + `","progressToken":"p"}`,
		`{"t\u006fken":"` + hygieneToken + `","token":"` + hygieneToken + `","progressToken":7}`,
		`{"to\u212aen":"` + hygieneToken + `"}`,
	}
	methods := []string{MethodToolsCall, "resources/list", "resources/read", "prompts/list", "prompts/get", "completion/complete", "logging/setLevel", "custom/method"}
	conn := eaOpenSSE(t, p.srv.URL, hygieneToken)
	defer conn.body.Close()
	for _, method := range methods {
		for _, v := range variants {
			raw := `{"jsonrpc":"2.0","id":400,"method":"` + method + `","params":{"name":"report","arguments":{"n":1.50,"s":"exact"},"_meta":` + v + `}}`
			received := receivedParams(t, method, streamableCall(t, p.srv.URL, sid, hygieneToken, raw))
			wantForwarded(t, "streamable "+method, received, "")
			if !bytes.Contains(received, []byte(`"n":1.50`)) {
				t.Errorf("number bytes changed: %s", redactToken(received))
			}
			if status := conn.post([]byte(raw)); status != http.StatusAccepted {
				t.Fatalf("SSE post status %d", status)
			}
			wantForwarded(t, "sse "+method, receivedParams(t, method, conn.next()), "")
		}
	}
	// Resumed on a new server, then the same call again.
	srv := httptest.NewServer(NewSSEServer(p.proxy, "").Handler())
	defer srv.Close()
	for i := 0; i < 2; i++ {
		raw := `{"jsonrpc":"2.0","id":401,"method":"tools/call","params":{"name":"report","arguments":{},"_meta":{"token":"` + hygieneToken + `","progressToken":9}}}`
		wantForwarded(t, "resumed", receivedParams(t, MethodToolsCall, streamableCall(t, srv.URL, sid, hygieneToken, raw)), "")
	}
	// Batches are refused and notifications accepted without forwarding.
	if status, _ := rawPost(t, p.srv.URL, sid, hygieneToken, http.MethodPost, "/mcp", `[{"jsonrpc":"2.0","method":"custom/method","params":{"_meta":{"token":"x"}}}]`); status != http.StatusBadRequest {
		t.Errorf("batch status %d, want 400", status)
	}
	if status, _ := rawPost(t, p.srv.URL, sid, hygieneToken, http.MethodPost, "/mcp", `{"jsonrpc":"2.0","method":"notifications/progress","params":{"_meta":{"token":"x"}}}`); status != http.StatusAccepted {
		t.Errorf("notification status %d, want 202", status)
	}
	// Direct dispatch, two spellings of _meta.
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`402`), Method: MethodToolsCall,
		Params: json.RawMessage(`{"name":"report","arguments":{},"_META":{"TOKEN":"` + hygieneToken + `"},"_meta":{"token":"` + hygieneToken + `"}}`)}
	resp, err := p.proxy.handleRequest(context.Background(), req)
	if err != nil {
		t.Fatal(err)
	}
	wantForwarded(t, "direct", receivedParams(t, MethodToolsCall, resp), "")
}

// --- the token query parameter ----------------------------------------------

func TestHasQueryToken(t *testing.T) {
	for q, want := range map[string]bool{
		"token=x":            true,
		"token=":             true,
		"token":              true,
		"token=&token=x":     true,
		"a=1&token=x":        true,
		"a=1;token=x":        true,
		"to%6Ben=x":          true,
		"%74oken=":           true,
		"token=%zz":          true,
		"":                   false,
		"sessionId=abc":      false,
		"tokens=x":           false,
		"Token=x":            false,
		"x=token":            false,
		"sessionId=token%3D": false,
	} {
		if got := hasQueryToken(q); got != want {
			t.Errorf("hasQueryToken(%q) = %v, want %v", q, got, want)
		}
	}
}

// With the default config, every request whose URL names the token parameter
// is refused with 401 on every endpoint that read it (Streamable HTTP at any
// of its paths, GET /sse, POST /message), whatever the value and whether or
// not a header is sent too. Neither the response nor the log carries the
// value.
func TestQueryTokenPresenceRefused(t *testing.T) {
	var logs bytes.Buffer
	old := zlog.Logger
	zlog.Logger = zerolog.New(&logs).Level(zerolog.DebugLevel)
	defer func() { zlog.Logger = old }()

	queries := []string{
		"?token=" + hygieneToken,
		"?token=",
		"?token",
		"?token=&token=" + hygieneToken,
		"?to%6ben=" + hygieneToken,
		"?sessionId=x&token=" + hygieneToken,
		"?a=1;token=" + hygieneToken,
		"?token=%zz" + hygieneToken,
	}
	cfg := &Config{Auth: AuthConfig{Mode: "header", RootKey: "query-presence-root"}, Logging: LoggingConfig{Level: "error"}}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetAuthenticator(fixedTokenAuth{good: hygieneToken})
	zlog.Logger = zerolog.New(&logs).Level(zerolog.DebugLevel)
	handler := NewSSEServer(proxy, "").Handler()
	type route struct{ method, path string }
	routes := []route{
		{http.MethodPost, "/"}, {http.MethodPost, "/mcp"}, {http.MethodPost, "/mcp/"},
		{http.MethodGet, "/mcp"}, {http.MethodDelete, "/mcp"}, {http.MethodOptions, "/mcp"},
		{http.MethodPost, "/message"}, {http.MethodGet, "/message"},
		{http.MethodGet, "/sse"}, {http.MethodPost, "/sse"},
	}
	for _, q := range queries {
		for _, header := range []string{"", hygieneToken} {
			for _, rt := range routes {
				req := httptest.NewRequest(rt.method, rt.path+q, bytes.NewReader(mustJSON(t, initializeBody("2025-03-26"))))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Accept", "application/json, text/event-stream")
				if header != "" {
					req.Header.Set("Authorization", "Bearer "+header)
				}
				ctx, cancel := context.WithCancel(req.Context())
				cancel() // bounds a stream that a regression would open
				w := httptest.NewRecorder()
				handler.ServeHTTP(w, req.WithContext(ctx))
				if w.Code != http.StatusUnauthorized {
					t.Errorf("%s %s (query #%d, header=%v): status %d, want 401", rt.method, rt.path, indexOf(queries, q), header != "", w.Code)
				}
				if strings.Contains(w.Body.String(), hygieneToken) {
					t.Errorf("%s %s: the response carries the query value", rt.method, rt.path)
				}
			}
		}
	}
	if strings.Contains(logs.String(), hygieneToken) {
		t.Error("the log carries the query value")
	}
}

func indexOf(list []string, s string) int {
	for i, v := range list {
		if v == s {
			return i
		}
	}
	return -1
}

// On a live server: POST /message and Streamable GET/DELETE on an existing
// session with the token parameter are refused by default (also when the
// first value is empty), and handled as before with allowQueryToken.
func TestQueryTokenRefusedOnLiveSessions(t *testing.T) {
	for _, allow := range []bool{false, true} {
		srv := newQueryTokenServer(t, allow)
		conn := eaOpenSSE(t, srv.URL, hygieneToken)
		sid := mustInitSession(t, srv.URL, hygieneToken)
		for _, q := range []string{"token=" + hygieneToken, "token=&token=" + hygieneToken, "token="} {
			for _, header := range []string{"", hygieneToken} {
				if q == "token=" && header == "" && allow {
					continue // no credential at all: not what this test is about
				}
				raw := string(mustJSON(t, toolCallBody(800, "echo")))
				status, _ := rawPost(t, srv.URL, "", header, http.MethodPost, "/message?sessionId="+conn.session+"&"+q, raw)
				want := http.StatusUnauthorized
				if allow {
					want = http.StatusAccepted
				}
				if status != want {
					t.Errorf("allow=%v POST /message?%s header=%v: status %d, want %d", allow, redactToken(json.RawMessage(q)), header != "", status, want)
				}
				if status == http.StatusAccepted {
					_ = conn.next()
				}
			}
			for _, method := range []string{http.MethodGet, http.MethodDelete} {
				status, _ := rawPost(t, srv.URL, sid, hygieneToken, method, "/mcp?"+q, "")
				want := http.StatusUnauthorized
				if allow {
					want = http.StatusMethodNotAllowed
					if method == http.MethodDelete {
						want = http.StatusOK
					}
				}
				if status != want {
					t.Errorf("allow=%v %s /mcp?%s: status %d, want %d", allow, method, redactToken(json.RawMessage(q)), status, want)
				}
			}
			sid = mustInitSession(t, srv.URL, hygieneToken)
		}
		_ = conn.body.Close()
	}
}

// With allowQueryToken the old behaviour returns: the query value is used
// when there is no header, the header wins when both are sent, and an empty
// value next to a header is ignored.
func TestQueryTokenFlagKeepsOldBehaviour(t *testing.T) {
	srv := newQueryTokenServer(t, true)
	status, hdr, _ := streamablePost(t, srv.URL+"/mcp?token="+hygieneToken, "", "", initializeBody("2025-03-26"))
	if status != http.StatusOK {
		t.Fatalf("query initialize: status %d", status)
	}
	status, _, body := streamablePost(t, srv.URL+"/mcp?token="+hygieneToken, hdr.Get("Mcp-Session-Id"), "", toolCallBody(602, "echo"))
	var resp Response
	if status != http.StatusOK || json.Unmarshal([]byte(body), &resp) != nil || resp.Error != nil {
		t.Errorf("query tools/call: status %d", status)
	}
	if status, _, _ = streamablePost(t, srv.URL+"/mcp?token="+otherConnectionCredential, "", hygieneToken, initializeBody("2025-03-26")); status != http.StatusOK {
		t.Errorf("header next to a query value: status %d, want 200", status)
	}
	if status, _, _ = streamablePost(t, srv.URL+"/mcp?token=", "", hygieneToken, initializeBody("2025-03-26")); status != http.StatusOK {
		t.Errorf("header next to an empty query value: status %d, want 200", status)
	}
}

// --- bytes and environment ---------------------------------------------------

// Params with no token are passed on exactly as written; when a token is
// removed, every other byte stays as it was.
func TestWithoutMetaTokenKeepsBytes(t *testing.T) {
	for _, in := range []string{
		`{ "name" : "report", "arguments" : {"n":1.50}, "_meta" : {"progressToken":7} }`,
		`{ "n\u0061me" : "report", "arguments" : {"v":1.50}, "_meta" : { "progressToken" : 9 } }`,
		"{\n  \"_Meta\": {\"x\": 1},\n  \"name\": \"a\"\n}",
	} {
		if got := withoutMetaToken(json.RawMessage(in)); string(got) != in {
			t.Errorf("params without a token changed: %s -> %s", in, got)
		}
	}
	cases := []struct{ in, want string }{
		{`{ "n\u0061me" : "report", "_meta" : { "progressToken" : 9, "token" : "t" }, "x" : 1.50 }`,
			`{ "n\u0061me" : "report", "_meta" : { "progressToken" : 9 }, "x" : 1.50 }`},
		{`{ "_meta" : { "token" : "t" } , "name" : "a" }`, `{  "name" : "a" }`},
		{"{\n  \"name\": \"a\",\n  \"_meta\": {\"token\": \"t\", \"p\": 1}\n}", "{\n  \"name\": \"a\",\n  \"_meta\": { \"p\": 1}\n}"},
	}
	for _, c := range cases {
		got := withoutMetaToken(json.RawMessage(c.in))
		if string(got) != c.want {
			t.Errorf("withoutMetaToken(%s) = %s, want %s", c.in, got, c.want)
		}
		if !json.Valid(got) {
			t.Errorf("withoutMetaToken(%s) is not valid JSON", c.in)
		}
	}
	for _, in := range []string{
		`{"_meta":{"token":"a","token":"b","progressToken":1},"arguments":{"v":1.50}}`,
		`{"_meta":{"to\u212aen":"a"},"_META":{"TOKEN":"b"}}`,
		`{"\u005fmeta":{"t\u006fken":"a"}}`,
	} {
		if _, has := metaTokenOf(t, withoutMetaToken(json.RawMessage(in))); has {
			t.Errorf("token remained in %s", in)
		}
	}
}

// Where environment names are compared without regard to case (Windows),
// the upstream's env map still wins over the base list and passthrough.
func TestStdioEnvConfiguredWinsCaseInsensitively(t *testing.T) {
	lookup := func(n string) (string, bool) {
		if strings.EqualFold(n, "SystemRoot") || n == "API_KEY" {
			return "from-proxy", true
		}
		return "", false
	}
	env := stdioEnv(UpstreamConfig{
		Env:            map[string]string{"SYSTEMROOT": "from-config", "api_key": "from-config"},
		EnvPassthrough: []string{"API_KEY"},
	}, lookup)
	// os/exec on Windows keeps the last of two names equal but for case.
	got := map[string]string{}
	for _, kv := range env {
		k, v, _ := strings.Cut(kv, "=")
		got[strings.ToLower(k)] = v
	}
	if got["systemroot"] != "from-config" || got["api_key"] != "from-config" {
		t.Errorf("configured value lost to the proxy's environment: %q", env)
	}
}

// Every base name keeps an empty value; nothing missing is invented; config
// ${VAR} expansion is unchanged.
func TestStdioEnvBaseNamesAndExpansion(t *testing.T) {
	for _, name := range stdioBaseEnv {
		e := stdioEnv(UpstreamConfig{}, func(k string) (string, bool) { return "", k == name })
		if len(e) != 1 || e[0] != name+"=" {
			t.Errorf("empty value of %s not kept: %q", name, e)
		}
	}
	t.Setenv("HYGIENE_EXPANSION", "expanded")
	t.Setenv("HYGIENE_EMPTY", "")
	file := filepath.Join(t.TempDir(), "config.yaml")
	raw := "auth:\n  mode: none\nupstreams:\n  test:\n    transport: stdio\n    command: [echo]\n    env:\n      FILLED: '${HYGIENE_EXPANSION}'\n      EMPTY: '${HYGIENE_EMPTY}'\n      MISSING: '${HYGIENE_MISSING_VAR}'\n"
	if err := os.WriteFile(file, []byte(raw), 0o600); err != nil {
		t.Fatal(err)
	}
	cfg, err := LoadConfig(file)
	if err != nil {
		t.Fatal(err)
	}
	env := cfg.Upstreams["test"].Env
	if env["FILLED"] != "expanded" || env["EMPTY"] != "" || env["MISSING"] != "" {
		t.Errorf("expansion changed: %q", env)
	}
}
