package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"
	"time"
)

// This transcript was captured by this exact script against 7b67e685 using
// a read-only Go overlay of the baseline production files. Session ids are
// fixed in the fixture, not redacted or normalized from responses. HTTP header
// bytes, bodies and SSE JSON-RPC message bytes are compared without rewriting.
// Date/Transfer-Encoding are net/http wire framing, outside the handler.
func TestOperationNoTrackerBaselineTranscript(t *testing.T) {
	var transcript strings.Builder
	for _, transport := range []string{"sse", "streamable"} {
		_, p := newStreamableHandler(t, 2)
		s := NewSSEServer(p, ":0") // deliberately no tracker option
		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()
		session := &sseSession{id: "fixture", ctx: ctx, cancel: cancel, messages: make(chan json.RawMessage, 64)}
		s.sessions[session.id] = session

		// tokenSum for the empty token, exactly as a tokenless initialize stores it.
		sess, ok := s.createStreamSession(context.Background(), "", protocolVersion20250326)
		if !ok {
			t.Fatal("create session")
		}
		delete(s.streamSessions, sess.id)
		sess.id = "fixture"
		s.streamSessions[sess.id] = sess
		steps := []struct {
			name, body  string
			method      string
			wantMessage bool
		}{
			{"ping", `{"jsonrpc":"2.0","id":1,"method":"ping"}`, "POST", true},
			{"tools-list", `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`, "POST", true},
			{"tool-call", `{"jsonrpc":"2.0","id":3,"method":"tools/call","params":{"name":"echo","arguments":{}}}`, "POST", true},
			{"unknown", `{"jsonrpc":"2.0","id":4,"method":"not/a/method"}`, "POST", true},
			{"malformed", `{`, "POST", false},
			{"response", `{"jsonrpc":"2.0","id":5,"result":{}}`, "POST", false},
			{"notification", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, "POST", false},
			{"method", ``, "GET", false},
		}
		for _, step := range steps {
			// Join a notification too: it has no SSE response to wait for.
			notified := make(chan struct{})
			if step.name == "notification" {
				p.SetResponseGuard(func(ctx context.Context, _ *Request, handle func(context.Context) (*Response, error)) (*Response, error) {
					defer close(notified)
					return handle(ctx)
				})
			}
			path := "/message?sessionId=fixture"
			if transport == "streamable" {
				path = "/mcp"
			}
			req := lifecycleRequest(path, "fixture", step.body)
			req.Method = step.method
			rr := httptest.NewRecorder()
			s.ServeHTTP(rr, req)
			if step.name == "notification" && transport == "sse" {
				lifecycleWait(t, notified)
			}
			fmt.Fprintf(&transcript, "%s/%s\n%d\n", transport, step.name, rr.Code)
			if err := rr.Header().Write(&transcript); err != nil {
				t.Fatal(err)
			}
			transcript.WriteString("\n")
			transcript.Write(rr.Body.Bytes())
			transcript.WriteString("\n")
			if transport == "sse" && step.wantMessage {
				select {
				case msg := <-session.messages:
					transcript.Write(msg)
					transcript.WriteString("\n")
				case <-time.After(5 * time.Second):
					t.Fatal("missing SSE response")
				}
			}
		}
	}
	raw := transcript.String()
	const golden = "testdata/operation_no_tracker.json"
	if path := os.Getenv("MCP_OPERATION_CAPTURE_TRANSCRIPT"); path != "" {
		// Maintainer-only baseline capture; never enabled by the normal test
		// suite. JSON escapes CRLF without discarding any header bytes.
		encoded, err := json.MarshalIndent(strings.Split(raw, "\n"), "", "  ")
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, append(encoded, '\n'), 0600); err != nil {
			t.Fatal(err)
		}
	}
	encoded, err := os.ReadFile(golden)
	if err != nil {
		t.Fatal(err)
	}
	var lines []string
	if err := json.Unmarshal(encoded, &lines); err != nil {
		t.Fatal(err)
	}
	if raw != strings.Join(lines, "\n") {
		t.Fatalf("no-tracker transcript differs from 7b67e685\n%s", raw)
	}
}

// Preserve the causal timing contract: SSE acknowledges while dispatch is
// held, Streamable HTTP does not. This is also run on the baseline overlay.
func TestOperationNoTrackerDispatchTiming(t *testing.T) {
	for _, transport := range []string{"sse", "streamable"} {
		t.Run(transport, func(t *testing.T) {
			s, p, sseID, streamID := lifecycleServer(t, nil)
			entered, release := make(chan struct{}), make(chan struct{})
			var once sync.Once
			defer once.Do(func() { close(release) })
			p.SetResponseGuard(func(ctx context.Context, r *Request, h func(context.Context) (*Response, error)) (*Response, error) {
				close(entered)
				<-release
				return h(ctx)
			})
			path := "/message?sessionId=" + sseID
			if transport == "streamable" {
				path = "/mcp"
			}
			done := make(chan struct{})
			go func() {
				defer close(done)
				s.ServeHTTP(httptest.NewRecorder(), lifecycleRequest(path, streamID, `{"jsonrpc":"2.0","id":1,"method":"ping"}`))
			}()
			lifecycleWait(t, entered)
			if transport == "sse" {
				lifecycleWait(t, done)
			} else {
				lifecycleBlocked(t, done)
			}
			once.Do(func() { close(release) })
			lifecycleWait(t, done)
			if transport == "sse" {
				select {
				case <-s.sessions[sseID].messages:
				case <-time.After(5 * time.Second):
					t.Fatal("worker did not send its response")
				}
			}
		})
	}
}
