package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// recordedReq is what the fake upstream saw of one request.
type recordedReq struct {
	Method string
	Path   string
	Auth   string
	RPC    string // JSON-RPC method of a POST body, "" otherwise
	Static string // X-Static header, a stand-in for any other configured header
}

// recUpstream is a fake MCP upstream that speaks both Streamable HTTP (/mcp)
// and legacy SSE (/sse + /message) and records every request it receives.
type recUpstream struct {
	srv *httptest.Server

	mu     sync.Mutex
	reqs   []recordedReq
	nextSS int
	// sseStreams are the open legacy SSE streams, by connection number.
	sseStreams map[int]chan string
	// dropSSE ends the newest legacy SSE stream from the server side.
	dropNotif chan struct{}

	// echo makes tools/call put the Authorization it received in the result.
	echo bool
}

func newRecUpstream(t *testing.T) *recUpstream {
	u := &recUpstream{sseStreams: map[int]chan string{}, dropNotif: make(chan struct{}, 4)}
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", u.streamable)
	mux.HandleFunc("/sse", u.sse)
	mux.HandleFunc("/message", u.message)
	u.srv = httptest.NewServer(mux)
	t.Cleanup(u.srv.Close)
	return u
}

func (u *recUpstream) record(r *http.Request, rpc string) {
	u.mu.Lock()
	u.reqs = append(u.reqs, recordedReq{Method: r.Method, Path: r.URL.Path, Auth: r.Header.Get("Authorization"), RPC: rpc, Static: r.Header.Get("X-Static")})
	u.mu.Unlock()
}

func (u *recUpstream) snapshot() []recordedReq {
	u.mu.Lock()
	defer u.mu.Unlock()
	return append([]recordedReq(nil), u.reqs...)
}

func (u *recUpstream) reply(id json.RawMessage, method string, auth string) map[string]any {
	switch method {
	case "initialize":
		return map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]any{"tools": map[string]any{}},
			"serverInfo":      map[string]any{"name": "rec", "version": "1"},
		}}
	case "tools/list":
		return map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{"tools": []map[string]any{
			{"name": "echo_tool", "description": "d", "inputSchema": map[string]any{"type": "object"}},
		}}}
	case "tools/call":
		text := "ok"
		if u.echo {
			text = "you sent " + auth
		}
		return map[string]any{"jsonrpc": "2.0", "id": id, "result": map[string]any{
			"content": []map[string]any{{"type": "text", "text": text}},
		}}
	}
	return map[string]any{"jsonrpc": "2.0", "id": id, "error": map[string]any{"code": -32601, "message": "no"}}
}

func readRPC(r *http.Request) (id json.RawMessage, method string) {
	var req struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
	}
	b, _ := io.ReadAll(r.Body)
	_ = json.Unmarshal(b, &req)
	return req.ID, req.Method
}

func (u *recUpstream) streamable(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case "POST":
		id, method := readRPC(r)
		u.record(r, method)
		if method == "initialize" {
			w.Header().Set("Mcp-Session-Id", "sess-1")
		}
		if len(id) == 0 {
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(u.reply(id, method, r.Header.Get("Authorization")))
	case "GET":
		u.record(r, "")
		w.Header().Set("Content-Type", "text/event-stream")
		w.WriteHeader(200)
		w.(http.Flusher).Flush()
		select {
		case <-r.Context().Done():
		case <-u.dropNotif:
		}
	case "DELETE":
		u.record(r, "")
		w.WriteHeader(200)
	}
}

func (u *recUpstream) sse(w http.ResponseWriter, r *http.Request) {
	u.record(r, "")
	u.mu.Lock()
	u.nextSS++
	n := u.nextSS
	ch := make(chan string, 16)
	u.sseStreams[n] = ch
	u.mu.Unlock()
	w.Header().Set("Content-Type", "text/event-stream")
	w.WriteHeader(200)
	fmt.Fprintf(w, "event: endpoint\ndata: %s/message\n\n", u.srv.URL)
	w.(http.Flusher).Flush()
	for {
		select {
		case <-r.Context().Done():
			return
		case <-u.dropNotif:
			return
		case ev := <-ch:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", ev)
			w.(http.Flusher).Flush()
		}
	}
}

func (u *recUpstream) message(w http.ResponseWriter, r *http.Request) {
	id, method := readRPC(r)
	u.record(r, method)
	w.WriteHeader(http.StatusAccepted)
	if len(id) == 0 {
		return
	}
	b, _ := json.Marshal(u.reply(id, method, r.Header.Get("Authorization")))
	u.mu.Lock()
	defer u.mu.Unlock()
	if ch, ok := u.sseStreams[u.nextSS]; ok {
		ch <- string(b)
	}
}

// testSource is a CredentialSource the tests steer.
type testSource struct {
	mu    sync.Mutex
	token string
	// failFor makes the source refuse these reasons.
	failFor map[SendReason]bool
	calls   []SendReason
	seen    []int // statuses observed
}

var errNoCredential = errors.New("no usable credential")

func (s *testSource) RequestCredentials(_ context.Context, why SendReason) (map[string]string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.calls = append(s.calls, why)
	if s.failFor[why] {
		return nil, errNoCredential
	}
	return map[string]string{"Authorization": "Bearer " + s.token}, nil
}

func (s *testSource) ObserveResponse(_ SendReason, _ map[string]string, status int) {
	s.mu.Lock()
	s.seen = append(s.seen, status)
	s.mu.Unlock()
}

func (s *testSource) set(tok string) { s.mu.Lock(); s.token = tok; s.mu.Unlock() }
func (s *testSource) count() int     { s.mu.Lock(); defer s.mu.Unlock(); return len(s.calls) }
func (s *testSource) countWhy(w SendReason) int {
	s.mu.Lock()
	defer s.mu.Unlock()
	n := 0
	for _, c := range s.calls {
		if c == w {
			n++
		}
	}
	return n
}

func waitFor(t *testing.T, what string, cond func() bool) {
	t.Helper()
	deadline := time.Now().Add(8 * time.Second)
	for time.Now().Before(deadline) {
		if cond() {
			return
		}
		time.Sleep(10 * time.Millisecond)
	}
	t.Fatalf("timed out waiting for %s", what)
}

func countReqs(rs []recordedReq, method, path string) int {
	n := 0
	for _, r := range rs {
		if r.Method == method && (path == "" || r.Path == path) {
			n++
		}
	}
	return n
}

func startRec(t *testing.T, transport string, cfg UpstreamConfig) (*UpstreamManager, *recUpstream, *testSource) {
	t.Helper()
	u := newRecUpstream(t)
	src := &testSource{token: "A", failFor: map[SendReason]bool{}}
	cfg.Transport = transport
	cfg.URL = u.srv.URL + "/mcp"
	if transport == "sse" {
		cfg.URL = u.srv.URL + "/sse"
	}
	cfg.Headers = map[string]string{"Authorization": "Bearer STATIC", "X-Static": "yes"}
	cfg.Credentials = src
	m := NewUpstreamManager(map[string]UpstreamConfig{"up": cfg}, nil, "up", true)
	t.Cleanup(func() { m.Close() })
	return m, u, src
}

// Every request the transport makes asks the source, once, and carries only
// what it returned. Both transports, every sender.
func TestCredentialSourceEveryRequestAsksTheSource(t *testing.T) {
	for _, transport := range []string{"streamable", "sse"} {
		t.Run(transport, func(t *testing.T) {
			m, u, src := startRec(t, transport, UpstreamConfig{})
			if err := m.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			if _, err := m.ForwardToolCall(context.Background(), "echo_tool", json.RawMessage(`{"name":"echo_tool","arguments":{}}`), 5*time.Second); err != nil {
				t.Fatal(err)
			}
			if transport == "streamable" {
				// the notification GET is a background send
				waitFor(t, "notification GET", func() bool { return countReqs(u.snapshot(), "GET", "/mcp") == 1 })
			}
			// Server drops the stream; the transport reconnects on its own.
			if transport == "sse" {
				u.dropNotif <- struct{}{}
				waitFor(t, "sse reconnect", func() bool { return countReqs(u.snapshot(), "GET", "/sse") == 2 })
			}
			src.set("B")
			if _, err := m.ForwardToolCall(context.Background(), "echo_tool", json.RawMessage(`{"name":"echo_tool","arguments":{}}`), 5*time.Second); err != nil {
				t.Fatal(err)
			}
			m.Close()
			rs := u.snapshot()
			if transport == "streamable" {
				waitFor(t, "DELETE", func() bool { return countReqs(u.snapshot(), "DELETE", "/mcp") == 1 })
				rs = u.snapshot()
			}
			if len(rs) == 0 {
				t.Fatal("no requests")
			}
			if got, want := src.count(), len(rs); got != want {
				t.Fatalf("source asked %d times for %d requests: %+v", got, want, rs)
			}
			for _, r := range rs {
				if r.Auth != "Bearer A" && r.Auth != "Bearer B" {
					t.Errorf("request %s %s %s carried %q, not a source-issued header", r.Method, r.Path, r.RPC, r.Auth)
				}
				if r.Static != "" {
					t.Errorf("request %s %s carried the configured header X-Static", r.Method, r.Path)
				}
			}
			// the last tools/call went out after src.set("B")
			var last recordedReq
			for _, r := range rs {
				if r.RPC == "tools/call" {
					last = r
				}
			}
			if last.Auth != "Bearer B" {
				t.Errorf("tools/call after the source changed carried %q, want Bearer B", last.Auth)
			}
			if transport == "streamable" {
				if src.countWhy(SendCleanup) != 1 {
					t.Errorf("cleanup reasons = %d, want 1", src.countWhy(SendCleanup))
				}
				if src.countWhy(SendBackgroundReconnect) < 1 {
					t.Errorf("the notification GET did not report a background send")
				}
			} else if src.countWhy(SendBackgroundReconnect) < 1 {
				t.Errorf("the SSE reconnect did not report a background send")
			}
		})
	}
}

// An error from the source means nothing is sent.
func TestCredentialSourceErrorSendsNothing(t *testing.T) {
	t.Run("agent request", func(t *testing.T) {
		for _, transport := range []string{"streamable", "sse"} {
			m, u, src := startRec(t, transport, UpstreamConfig{})
			if err := m.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			before := len(u.snapshot())
			src.mu.Lock()
			src.failFor[SendAgentRequest] = true
			src.mu.Unlock()
			_, err := m.ForwardToolCall(context.Background(), "echo_tool", json.RawMessage(`{"name":"echo_tool","arguments":{}}`), 2*time.Second)
			if err == nil || !errors.Is(err, errNoCredential) || !IsCredentialSourceError(err) {
				t.Fatalf("%s: want the source's error, got %v", transport, err)
			}
			if after := len(u.snapshot()); after != before {
				t.Errorf("%s: %d request(s) left after the source refused", transport, after-before)
			}
			m.Close()
		}
	})
	t.Run("cleanup drops the session locally", func(t *testing.T) {
		m, u, src := startRec(t, "streamable", UpstreamConfig{})
		if err := m.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		waitFor(t, "notification GET", func() bool { return countReqs(u.snapshot(), "GET", "/mcp") == 1 })
		src.mu.Lock()
		src.failFor[SendCleanup] = true
		src.mu.Unlock()
		done := make(chan struct{})
		go func() { m.Close(); close(done) }()
		select {
		case <-done:
		case <-time.After(5 * time.Second):
			t.Fatal("Close hung with no usable credential")
		}
		if n := countReqs(u.snapshot(), "DELETE", ""); n != 0 {
			t.Errorf("%d DELETE(s) sent with no usable credential", n)
		}
		if src.countWhy(SendCleanup) != 1 {
			t.Errorf("source not asked for cleanup")
		}
	})
	t.Run("sse reconnect waits without sending", func(t *testing.T) {
		m, u, src := startRec(t, "sse", UpstreamConfig{})
		if err := m.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		src.mu.Lock()
		src.failFor[SendBackgroundReconnect] = true
		src.mu.Unlock()
		u.dropNotif <- struct{}{}
		waitFor(t, "two refused reconnect attempts", func() bool { return src.countWhy(SendBackgroundReconnect) >= 2 })
		if n := countReqs(u.snapshot(), "GET", "/sse"); n != 1 {
			t.Errorf("%d stream GETs, want only the first", n)
		}
		src.mu.Lock()
		src.failFor[SendBackgroundReconnect] = false
		src.mu.Unlock()
		waitFor(t, "reconnect once a credential exists", func() bool { return countReqs(u.snapshot(), "GET", "/sse") == 2 })
	})
	t.Run("notification GET is not sent", func(t *testing.T) {
		u := newRecUpstream(t)
		src := &testSource{token: "A", failFor: map[SendReason]bool{SendBackgroundReconnect: true}}
		m := NewUpstreamManager(map[string]UpstreamConfig{"up": {Transport: "streamable", URL: u.srv.URL + "/mcp", Credentials: src}}, nil, "up", true)
		defer m.Close()
		if err := m.Start(context.Background()); err != nil {
			t.Fatal(err)
		}
		waitFor(t, "source asked for the notification GET", func() bool { return src.countWhy(SendBackgroundReconnect) == 1 })
		time.Sleep(200 * time.Millisecond)
		if n := countReqs(u.snapshot(), "GET", ""); n != 0 {
			t.Errorf("notification GET sent with no usable credential (%d)", n)
		}
	})
}

// A replacement connection the manager dials after the old one died asks the
// source with the background reason, for the connect and for the handshake.
func TestCredentialSourceManagerReconnectIsBackground(t *testing.T) {
	for _, transport := range []string{"streamable", "sse"} {
		t.Run(transport, func(t *testing.T) {
			m, u, src := startRec(t, transport, UpstreamConfig{})
			if err := m.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			src.mu.Lock()
			src.calls = nil
			src.mu.Unlock()
			base := len(u.snapshot())
			m.mu.Lock()
			old := m.clients["up"]
			m.mu.Unlock()
			// The transport dies without the manager closing it on purpose.
			old.transport.Close()
			waitFor(t, "replacement handshake", func() bool {
				rs := u.snapshot()[base:]
				return countReqs(rs, "POST", "") >= 2 || (transport == "sse" && countReqs(rs, "POST", "/message") >= 2)
			})
			src.mu.Lock()
			defer src.mu.Unlock()
			if len(src.calls) == 0 {
				t.Fatal("source never asked")
			}
			for _, c := range src.calls {
				if c == SendAgentRequest {
					t.Errorf("a background reconnect asked the source as an agent request: %v", src.calls)
				}
			}
		})
	}
}

// With no source the transports send exactly the configured headers.
func TestCredentialSourceNilSendsConfiguredHeaders(t *testing.T) {
	for _, transport := range []string{"streamable", "sse"} {
		t.Run(transport, func(t *testing.T) {
			u := newRecUpstream(t)
			url := u.srv.URL + "/mcp"
			if transport == "sse" {
				url = u.srv.URL + "/sse"
			}
			m := NewUpstreamManager(map[string]UpstreamConfig{"up": {Transport: transport, URL: url,
				Headers: map[string]string{"Authorization": "Bearer STATIC", "X-Static": "yes"}}}, nil, "up", true)
			if err := m.Start(context.Background()); err != nil {
				t.Fatal(err)
			}
			if _, err := m.ForwardToolCall(context.Background(), "echo_tool", json.RawMessage(`{"name":"echo_tool","arguments":{}}`), 5*time.Second); err != nil {
				t.Fatal(err)
			}
			m.Close()
			if transport == "streamable" {
				waitFor(t, "DELETE", func() bool { return countReqs(u.snapshot(), "DELETE", "") == 1 })
			}
			for _, r := range u.snapshot() {
				if r.Auth != "Bearer STATIC" || r.Static != "yes" {
					t.Errorf("%s %s carried auth=%q static=%q", r.Method, r.Path, r.Auth, r.Static)
				}
			}
		})
	}
}

// A value the source returned never appears in an error the transport returns
// or logs, even when the upstream echoes it.
func TestCredentialSourceValuesAreRedactedFromErrors(t *testing.T) {
	mux := http.NewServeMux()
	var secretSeen atomic.Value
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		secretSeen.Store(r.Header.Get("Authorization"))
		http.Error(w, "bad credential "+r.Header.Get("Authorization"), 500)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	src := &testSource{token: "SECRET-TOKEN-123", failFor: map[SendReason]bool{}}
	tr := NewStreamableHTTPTransport(srv.URL+"/mcp", nil, false, true)
	tr.SetCredentialSource(src)
	err := tr.WriteMessage(context.Background(), json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"initialize"}`))
	if err == nil {
		t.Fatal("want an error")
	}
	if strings.Contains(err.Error(), "SECRET-TOKEN-123") {
		t.Fatalf("error carries the source's token: %v", err)
	}
	if !strings.Contains(err.Error(), "[redacted]") {
		t.Fatalf("expected a redaction marker: %v", err)
	}
	var se *UpstreamStatusError
	if !errors.As(err, &se) || se.Status != 500 {
		t.Fatalf("status error lost: %v", err)
	}
}
