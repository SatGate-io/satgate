package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

// Registration and admission closure share one lock. Each closure represents
// a distinct lease, not a caller-supplied JSON-RPC id or session id.
type lifecycleTracker struct {
	mu                   sync.Mutex
	stopped              bool
	active, begins, ends int
	drained              chan struct{}
	onEnd                func()
}

func newLifecycleTracker() *lifecycleTracker { return &lifecycleTracker{drained: make(chan struct{})} }
func (x *lifecycleTracker) BeginOperation() (func(), bool) {
	x.mu.Lock()
	defer x.mu.Unlock()
	if x.stopped {
		return nil, false
	}
	x.active++
	x.begins++
	return func() {
		if x.onEnd != nil {
			x.onEnd()
		}
		x.mu.Lock()
		defer x.mu.Unlock()
		x.active--
		x.ends++
		if x.stopped && x.active == 0 {
			close(x.drained)
		}
	}, true
}
func (x *lifecycleTracker) stop() <-chan struct{} {
	x.mu.Lock()
	defer x.mu.Unlock()
	x.stopped = true
	if x.active == 0 {
		close(x.drained)
	}
	return x.drained
}
func (x *lifecycleTracker) counts() (int, int, int) {
	x.mu.Lock()
	defer x.mu.Unlock()
	return x.active, x.begins, x.ends
}
func lifecycleWait(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
	case <-time.After(5 * time.Second):
		t.Fatal("operation did not finish")
	}
}
func lifecycleBlocked(t *testing.T, ch <-chan struct{}) {
	t.Helper()
	select {
	case <-ch:
		t.Fatal("drain overtook accepted work")
	case <-time.After(30 * time.Millisecond):
	}
}

type lifecycleRouter struct {
	UpstreamRouter
	entered, release chan struct{}
}

func (r *lifecycleRouter) ForwardToolCallForTenant(context.Context, string, string, json.RawMessage, time.Duration) (*Response, error) {
	close(r.entered)
	<-r.release
	return &Response{JSONRPC: "2.0", Result: json.RawMessage(`{}`)}, nil
}

// Public-transport version of the async SSE shutdown regression. Real proxy
// hold/receipt/pending code runs, then the upstream and settlement are held in
// turn. Closing the SSE GET must not make either phase disappear from drain.
func TestOperationDrainWaitsForDispatchSettlementAndRecords(t *testing.T) {
	for _, transport := range []string{"sse", "streamable"} {
		t.Run(transport, func(t *testing.T) {
			e, store, _, done := eaHold(t, eaOpts{}, nil)
			defer done()
			store.failCommit = true // creates a real pending reconciliation entry
			tracker := newLifecycleTracker()
			router := &lifecycleRouter{UpstreamRouter: e.proxy.router, entered: make(chan struct{}), release: make(chan struct{})}
			e.proxy.SetUpstreamRouter(router)
			settled, releaseSettle := make(chan struct{}), make(chan struct{})
			var upstreamOnce, settleOnce sync.Once
			defer upstreamOnce.Do(func() { close(router.release) })
			defer settleOnce.Do(func() { close(releaseSettle) })
			var settlementDone atomic.Bool
			e.proxy.SetCallGate(gateFunc(func(context.Context, GateCall) (*GateAdmission, error) {
				return &GateAdmission{Settle: func(context.Context, GateOutcome) {
					close(settled)
					<-releaseSettle
					settlementDone.Store(true)
				}}, nil
			}))
			var sawEnd atomic.Bool
			tracker.onEnd = func() {
				if !settlementDone.Load() {
					t.Error("end ran before settlement finished")
				}
				if len(e.proxy.PendingRecords()) != 1 {
					t.Error("end did not see the pending reconciliation entry")
				}
				if len(e.base.signed()) != 1 {
					t.Error("end did not see the completed receipt record")
				}
				sawEnd.Store(true)
			}
			s := NewSSEServer(e.proxy, ":0", WithOperationTracker(tracker))
			srv := httptest.NewServer(s)
			defer srv.Close()
			// Release blocked work before httptest.Server.Close even on a
			// regression assertion, so fail-before probes terminate cleanly.
			defer upstreamOnce.Do(func() { close(router.release) })
			defer settleOnce.Do(func() { close(releaseSettle) })
			postDone := make(chan struct{})
			if transport == "sse" {
				conn := eaOpenSSE(t, srv.URL, e.token)
				defer conn.body.Close()
				if conn.post(mustJSON(t, toolCallBody(17, "paid_read"))) != http.StatusAccepted {
					t.Fatal("POST not accepted")
				}
				// Registration must already be visible when the POST returns 202.
				if active, begins, ends := tracker.counts(); active != 1 || begins != 1 || ends != 0 {
					t.Errorf("at acceptance: %d %d %d", active, begins, ends)
				}
				lifecycleWait(t, router.entered)
				conn.body.Close()
				// Join GET cleanup, not merely the client's close, before draining.
				eaWait(t, func() bool { s.mu.Lock(); defer s.mu.Unlock(); return len(s.sessions) == 0 })
				close(postDone)
			} else {
				// Establish a session without consuming the operation under test.
				sess, ok := s.createStreamSession(context.Background(), e.token, protocolVersion20250326)
				if !ok {
					t.Fatal("session not created")
				}
				go func() {
					defer close(postDone)
					status, _, _ := streamablePost(t, srv.URL+"/mcp", sess.id, e.token, toolCallBody(17, "paid_read"))
					if status != http.StatusOK {
						t.Errorf("status %d", status)
					}
				}()
				lifecycleWait(t, router.entered)
			}
			drained := tracker.stop()
			lifecycleBlocked(t, drained)
			upstreamOnce.Do(func() { close(router.release) })
			lifecycleWait(t, settled)
			lifecycleBlocked(t, drained)
			settleOnce.Do(func() { close(releaseSettle) })
			lifecycleWait(t, drained)
			lifecycleWait(t, postDone)
			if active, begins, ends := tracker.counts(); active != 0 || begins != 1 || ends != 1 || !sawEnd.Load() {
				t.Fatalf("after drain: %d %d %d", active, begins, ends)
			}
		})
	}
}

func lifecycleServer(t *testing.T, tracker OperationTracker) (*SSEServer, *Proxy, string, string) {
	t.Helper()
	_, p := newStreamableHandler(t, 5)
	s := NewSSEServer(p, ":0", WithOperationTracker(tracker))
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	s.sessions["fixture"] = &sseSession{id: "fixture", ctx: ctx, cancel: cancel, messages: make(chan json.RawMessage, 64)}
	sess, ok := s.createStreamSession(context.Background(), "", protocolVersion20250326)
	if !ok {
		t.Fatal("create session")
	}
	return s, p, "fixture", sess.id
}
func lifecycleRequest(path, sid, body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, path, strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Accept", "application/json, text/event-stream")
	if sid != "" {
		r.Header.Set(headerMCPSessionID, sid)
	}
	return r
}

func TestOperationRefusedBeforeReadHoldOrForward(t *testing.T) {
	for _, transport := range []string{"sse", "streamable"} {
		t.Run(transport, func(t *testing.T) {
			tracker := newLifecycleTracker()
			s, p, sseID, streamID := lifecycleServer(t, tracker)
			store := &eaHoldStore{InMemoryBudgetEnforcer: p.budget.(*InMemoryBudgetEnforcer)}
			p.SetBudgetEnforcer(store)
			gate := &countingGate{}
			p.SetCallGate(gate)
			rec := &lockedRecorder{}
			p.SetEvidenceRecorder(rec)
			lifecycleWait(t, tracker.stop())
			path := "/message?sessionId=" + sseID
			if transport == "streamable" {
				path = "/mcp"
			}
			r := lifecycleRequest(path, streamID, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo"}}`)
			body := &lifecycleReadSpy{Reader: r.Body}
			r.Body = body
			w := httptest.NewRecorder()
			s.ServeHTTP(w, r)
			if w.Code != http.StatusServiceUnavailable || body.reads != 0 {
				t.Fatalf("status=%d reads=%d", w.Code, body.reads)
			}
			if a, b, c, d := store.counts(); a+b+c+d != 0 {
				t.Fatalf("budget work: %d %d %d %d", a, b, c, d)
			}
			if streamableForwardCount(p) != 0 || gate.admits.Load() != 0 || len(rec.decisions) != 0 || len(p.PendingRecords()) != 0 {
				t.Fatal("refused operation did work")
			}
			if a, b, c := tracker.counts(); a+b+c != 0 {
				t.Fatalf("refused lease: %d %d %d", a, b, c)
			}
		})
	}
}

type lifecycleReadError struct{ reads int }

type lifecycleReadSpy struct {
	io.Reader
	reads int
}

func (r *lifecycleReadSpy) Read(p []byte) (int, error) {
	r.reads++
	return r.Reader.Read(p)
}

func (*lifecycleReadSpy) Close() error { return nil }

func (r *lifecycleReadError) Read([]byte) (int, error) {
	r.reads++
	return 0, errors.New("read failed")
}
func (*lifecycleReadError) Close() error { return nil }

func TestOperationEndExactlyOnce(t *testing.T) {
	for _, transport := range []string{"sse", "streamable"} {
		for _, scenario := range []string{"success", "error", "closed-session", "malformed", "read-error", "response", "notification", "missing-session"} {
			t.Run(transport+"/"+scenario, func(t *testing.T) {
				tracker := newLifecycleTracker()
				s, p, sseID, streamID := lifecycleServer(t, tracker)
				body := `{"jsonrpc":"2.0","id":1,"method":"ping"}`
				want := http.StatusOK
				if transport == "sse" {
					want = http.StatusAccepted
				}
				switch scenario {
				case "error":
					p.SetResponseGuard(func(context.Context, *Request, func(context.Context) (*Response, error)) (*Response, error) {
						return nil, errors.New("dispatch error")
					})
				case "closed-session":
					delete(s.sessions, sseID)
					s.deleteStreamSession(streamID)
					want = http.StatusNotFound
				case "malformed":
					body = `{`
					want = http.StatusBadRequest
				case "read-error":
					want = http.StatusBadRequest
				case "response":
					body = `{"jsonrpc":"2.0","id":1,"result":{}}`
					want = http.StatusAccepted
				case "notification":
					body = `{"jsonrpc":"2.0","method":"notifications/initialized"}`
					want = http.StatusAccepted
				case "missing-session":
					sseID = ""
					streamID = ""
					want = http.StatusBadRequest
				}
				path := "/message?sessionId=" + sseID
				if transport == "streamable" {
					path = "/mcp"
				}
				r := lifecycleRequest(path, streamID, body)
				if scenario == "read-error" {
					r.Body = &lifecycleReadError{}
				}
				w := httptest.NewRecorder()
				s.ServeHTTP(w, r)
				if w.Code != want {
					t.Fatalf("status=%d want=%d: %s", w.Code, want, w.Body)
				}
				lifecycleWait(t, tracker.stop())
				if a, b, c := tracker.counts(); a != 0 || b != 1 || c != 1 {
					t.Fatalf("active/begins/ends=%d/%d/%d", a, b, c)
				}
			})
		}
	}
}

func TestOperationStreamableWholeDispatch(t *testing.T) {
	for _, method := range []string{"initialize", "tools/list", "resources/list", "prompts/list", "unknown", "delete", "get", "bad-origin", "bad-content-type"} {
		t.Run(method, func(t *testing.T) {
			tracker := newLifecycleTracker()
			s, _, _, sid := lifecycleServer(t, tracker)
			body := fmt.Sprintf(`{"jsonrpc":"2.0","id":1,"method":%q}`, method)
			if method == "initialize" {
				body = string(mustJSON(t, initializeBody(protocolVersion20250326)))
				sid = ""
			}
			r := lifecycleRequest("/mcp", sid, body)
			switch method {
			case "delete":
				r.Method = http.MethodDelete
			case "get":
				r.Method = http.MethodGet
			case "bad-origin":
				r.Header.Set("Origin", "https://untrusted.example")
			case "bad-content-type":
				r.Header.Set("Content-Type", "text/plain")
			}
			s.ServeHTTP(httptest.NewRecorder(), r)
			lifecycleWait(t, tracker.stop())
			if a, b, c := tracker.counts(); a != 0 || b != 1 || c != 1 {
				t.Fatalf("active/begins/ends=%d/%d/%d", a, b, c)
			}
		})
	}
}

func TestOperationPanicUnwindsLease(t *testing.T) {
	for _, transport := range []string{"sse", "streamable"} {
		t.Run(transport, func(t *testing.T) {
			if transport == "sse" {
				// An unrecovered worker panic must retain its historical process-fatal
				// behavior. A subprocess proves its defers without adding recovery.
				marker := filepath.Join(t.TempDir(), "ends")
				cmd := exec.Command(os.Args[0], "-test.run=^TestOperationSSEPanicChild$", "-test.count=1")
				cmd.Env = append(os.Environ(), "MCP_OPERATION_PANIC_MARKER="+marker)
				output, err := cmd.CombinedOutput()
				if err == nil || !strings.Contains(string(output), "lifecycle panic") {
					t.Fatalf("child did not panic: %v %s", err, output)
				}
				raw, err := os.ReadFile(marker)
				if err != nil || string(raw) != "end\n" {
					t.Fatalf("end marker %q: %v", raw, err)
				}
				return
			}
			tracker := newLifecycleTracker()
			s, p, _, sid := lifecycleServer(t, tracker)
			p.SetResponseGuard(func(context.Context, *Request, func(context.Context) (*Response, error)) (*Response, error) {
				panic("lifecycle panic")
			})
			func() {
				defer func() {
					if recover() != "lifecycle panic" {
						t.Error("panic was swallowed")
					}
				}()
				s.ServeHTTP(httptest.NewRecorder(), lifecycleRequest("/mcp", sid, `{"jsonrpc":"2.0","id":1,"method":"ping"}`))
			}()
			lifecycleWait(t, tracker.stop())
			if a, b, c := tracker.counts(); a != 0 || b != 1 || c != 1 {
				t.Fatalf("active/begins/ends=%d/%d/%d", a, b, c)
			}
		})
	}
}
func TestOperationSSEPanicChild(t *testing.T) {
	marker := os.Getenv("MCP_OPERATION_PANIC_MARKER")
	if marker == "" {
		t.Skip("subprocess only")
	}
	tracker := newLifecycleTracker()
	tracker.onEnd = func() {
		f, err := os.OpenFile(marker, os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0600)
		if err != nil {
			panic(err)
		}
		defer f.Close()
		fmt.Fprintln(f, "end")
	}
	s, p, sid, _ := lifecycleServer(t, tracker)
	p.SetResponseGuard(func(context.Context, *Request, func(context.Context) (*Response, error)) (*Response, error) {
		panic("lifecycle panic")
	})
	s.ServeHTTP(httptest.NewRecorder(), lifecycleRequest("/message?sessionId="+sid, "", `{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	select {} // the worker panic terminates this child
}

// A panic writing the 202 happens after transfer: only the worker may end it.
type lifecyclePanicWriter struct{ *httptest.ResponseRecorder }

type lifecyclePanicBody struct{}

func (lifecyclePanicBody) Read([]byte) (int, error) { panic("body panic") }
func (lifecyclePanicBody) Close() error             { return nil }

func TestOperationPanicBeforeDispatch(t *testing.T) {
	for _, transport := range []string{"sse", "streamable"} {
		t.Run(transport, func(t *testing.T) {
			tracker := newLifecycleTracker()
			s, _, sseID, streamID := lifecycleServer(t, tracker)
			path := "/message?sessionId=" + sseID
			if transport == "streamable" {
				path = "/mcp"
			}
			r := lifecycleRequest(path, streamID, "")
			r.Body = lifecyclePanicBody{}
			func() {
				defer func() {
					if recover() != "body panic" {
						t.Error("expected body panic")
					}
				}()
				s.ServeHTTP(httptest.NewRecorder(), r)
			}()
			lifecycleWait(t, tracker.stop())
			if a, b, c := tracker.counts(); a != 0 || b != 1 || c != 1 {
				t.Fatalf("active/begins/ends=%d/%d/%d", a, b, c)
			}
		})
	}
}

func (w lifecyclePanicWriter) WriteHeader(int) { panic("write panic") }
func TestOperationSSEHandoffSurvivesResponsePanic(t *testing.T) {
	tracker := newLifecycleTracker()
	s, p, sid, _ := lifecycleServer(t, tracker)
	entered, release := make(chan struct{}), make(chan struct{})
	var once sync.Once
	defer once.Do(func() { close(release) })
	p.SetResponseGuard(func(ctx context.Context, r *Request, h func(context.Context) (*Response, error)) (*Response, error) {
		close(entered)
		<-release
		return h(ctx)
	})
	func() {
		defer func() {
			if recover() != "write panic" {
				t.Error("expected writer panic")
			}
		}()
		s.ServeHTTP(lifecyclePanicWriter{httptest.NewRecorder()}, lifecycleRequest("/message?sessionId="+sid, "", `{"jsonrpc":"2.0","id":1,"method":"ping"}`))
	}()
	lifecycleWait(t, entered)
	drained := tracker.stop()
	lifecycleBlocked(t, drained)
	once.Do(func() { close(release) })
	lifecycleWait(t, drained)
	if a, b, c := tracker.counts(); a != 0 || b != 1 || c != 1 {
		t.Fatalf("active/begins/ends=%d/%d/%d", a, b, c)
	}
}

// Acceptance must be registered even while the POST body has not arrived yet.
type lifecycleBlockedBody struct{ entered, release chan struct{} }

func (b *lifecycleBlockedBody) Read([]byte) (int, error) {
	close(b.entered)
	<-b.release
	return 0, io.ErrUnexpectedEOF
}
func (*lifecycleBlockedBody) Close() error { return nil }
func TestOperationRegistersBeforeBodyRead(t *testing.T) {
	for _, transport := range []string{"sse", "streamable"} {
		t.Run(transport, func(t *testing.T) {
			tracker := newLifecycleTracker()
			s, _, sseID, streamID := lifecycleServer(t, tracker)
			path := "/message?sessionId=" + sseID
			if transport == "streamable" {
				path = "/mcp"
			}
			r := lifecycleRequest(path, streamID, "")
			b := &lifecycleBlockedBody{make(chan struct{}), make(chan struct{})}
			r.Body = b
			var once sync.Once
			defer once.Do(func() { close(b.release) })
			done := make(chan struct{})
			go func() { defer close(done); s.ServeHTTP(httptest.NewRecorder(), r) }()
			lifecycleWait(t, b.entered)
			drained := tracker.stop()
			lifecycleBlocked(t, drained)
			once.Do(func() { close(b.release) })
			lifecycleWait(t, done)
			lifecycleWait(t, drained)
			if a, b, c := tracker.counts(); a != 0 || b != 1 || c != 1 {
				t.Fatalf("active/begins/ends=%d/%d/%d", a, b, c)
			}
		})
	}
}
