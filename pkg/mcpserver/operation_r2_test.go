package mcpserver

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// Cancellation begins cleanup; it is not evidence that the recorder returned.
type operationHeldRecorder struct {
	EvidenceRecorder
	WindowRecorder
	entered, cancelled, release, finished chan struct{}
	waitCancel                            bool
}

func (r *operationHeldRecorder) RecordMCPDecision(ctx context.Context, _ MCPDecision) (*MCPEvidence, error) {
	defer close(r.finished)
	close(r.entered)
	if r.waitCancel {
		<-ctx.Done()
		close(r.cancelled)
	}
	<-r.release
	if ctx.Err() != nil {
		return nil, ctx.Err()
	}
	return nil, errors.New("synthetic recorder unavailable")
}
func newOperationHeldRecorder() *operationHeldRecorder {
	return &operationHeldRecorder{entered: make(chan struct{}), cancelled: make(chan struct{}), release: make(chan struct{}), finished: make(chan struct{})}
}

func TestOperationSSEConnectRecording(t *testing.T) {
	for _, stopped := range []bool{false, true} {
		t.Run(fmt.Sprint(stopped), func(t *testing.T) {
			p := newEvidenceTestProxy(t)
			p.SetAuthenticator(rejectingAuthenticator{err: errors.New("token revoked")})
			rec := newOperationHeldRecorder()
			rec.EvidenceRecorder = &fakeEvidenceRecorder{}
			p.SetEvidenceRecorder(rec)
			tracker := newLifecycleTracker()
			if stopped {
				tracker.stop()
			}
			s := NewSSEServer(p, ":0", WithOperationTracker(tracker))
			req := httptest.NewRequest(http.MethodGet, "/sse", nil)
			req.Header.Set("Authorization", "Bearer synthetic-revoked")
			req = req.WithContext(context.WithValue(req.Context(), CtxTenantID, eaTenant))
			w := httptest.NewRecorder()
			done := make(chan struct{})
			go func() { defer close(done); s.ServeHTTP(w, req) }()
			var once sync.Once
			defer func() { once.Do(func() { close(rec.release) }); lifecycleWait(t, done) }()
			select {
			case <-rec.entered:
				if stopped {
					t.Error("closed admission entered connect-time recorder")
				} else {
					if a, b, c := tracker.counts(); a != 1 || b != 1 || c != 0 {
						t.Errorf("untracked connect: %d/%d/%d", a, b, c)
					}
					drained := tracker.stop()
					select {
					case <-drained:
						t.Error("drain overtook connect-time recorder")
					default:
					}
				}
			case <-done:
				if !stopped || w.Code != http.StatusServiceUnavailable {
					t.Errorf("unexpected status %d", w.Code)
				}
			case <-time.After(5 * time.Second):
				t.Fatal("connect made no progress")
			}
			once.Do(func() { close(rec.release) })
			lifecycleWait(t, done)
			if stopped {
				select {
				case <-rec.entered:
					t.Error("refused connect recorded")
				default:
				}
				if a, b, c := tracker.counts(); a+b+c != 0 {
					t.Errorf("refused counts %d/%d/%d", a, b, c)
				}
			} else {
				lifecycleWait(t, rec.finished)
				lifecycleWait(t, tracker.drained)
				if w.Code != http.StatusForbidden {
					t.Errorf("denial status %d", w.Code)
				}
				if a, b, c := tracker.counts(); a != 0 || b != 1 || c != 1 {
					t.Errorf("connect counts %d/%d/%d", a, b, c)
				}
			}
		})
	}
}

func TestOperationSSEHealthyConnectEndsBeforeStream(t *testing.T) {
	tracker := newLifecycleTracker()
	s, _, _, _ := lifecycleServer(t, tracker)
	srv := httptest.NewServer(s)
	defer srv.Close()
	conn := eaOpenSSE(t, srv.URL, "")
	defer conn.body.Close()
	lifecycleWait(t, tracker.stop())
	if a, b, c := tracker.counts(); a != 0 || b != 1 || c != 1 {
		t.Fatalf("finite connect counts %d/%d/%d", a, b, c)
	}
	// The GET is still open after the finite connect lease drained.
	s.mu.Lock()
	_, open := s.sessions[conn.session]
	s.mu.Unlock()
	if !open {
		t.Fatal("healthy stream closed with its connect lease")
	}
}

// This causal timing control also runs against 7b67e685: SSE acknowledges
// before the signing deadline, Streamable answers at that deadline, and neither
// answer waits for held recorder cleanup. Only tracked drain waits for cleanup.
func TestOperationRecorderCleanupDeadline(t *testing.T) {
	for _, transport := range []string{"sse", "streamable"} {
		t.Run(transport, func(t *testing.T) {
			for _, tracked := range []bool{false, true} {
				t.Run(fmt.Sprint(tracked), func(t *testing.T) {
					e, store, _, cleanup := eaHold(t, eaOpts{}, nil)
					defer cleanup()
					const limit = 150 * time.Millisecond
					if err := e.proxy.SetEveryActionSettings(EveryActionSettings{SignTimeout: limit, CommitTimeout: 50 * time.Millisecond, HoldTTL: time.Second}); err != nil {
						t.Fatal(err)
					}
					rec := newOperationHeldRecorder()
					rec.EvidenceRecorder = e.base
					rec.WindowRecorder = e.proxy.evidence.(WindowRecorder)
					rec.waitCancel = true
					e.proxy.SetEvidenceRecorder(rec)
					tracker := newLifecycleTracker()
					var option OperationTracker
					if tracked {
						option = tracker
					}
					s := NewSSEServer(e.proxy, ":0", WithOperationTracker(option))
					// Fixed sessions avoid random response headers in the parity check.
					ctx, cancel := context.WithCancel(context.Background())
					defer cancel()
					session := &sseSession{id: "fixture", ctx: ctx, cancel: cancel, messages: make(chan json.RawMessage, 64)}
					s.sessions[session.id] = session
					sess, ok := s.createStreamSession(context.Background(), e.token, protocolVersion20250326)
					if !ok {
						t.Fatal("session")
					}
					delete(s.streamSessions, sess.id)
					sess.id = "fixture"
					s.streamSessions[sess.id] = sess
					path := "/message?sessionId=fixture"
					if transport == "streamable" {
						path = "/mcp"
					}
					req := lifecycleRequest(path, "fixture", string(mustJSON(t, toolCallBody(71, "paid_read"))))
					req.Header.Set("Authorization", "Bearer "+e.token)
					w := httptest.NewRecorder()
					done := make(chan struct{})
					started := time.Now()
					go func() { defer close(done); s.ServeHTTP(w, req) }()
					var once sync.Once
					defer func() { once.Do(func() { close(rec.release) }); lifecycleWait(t, rec.finished); lifecycleWait(t, done) }()
					lifecycleWait(t, rec.entered)
					if transport == "sse" {
						lifecycleWait(t, done)
						select {
						case <-rec.cancelled:
							t.Error("202 waited for signing deadline")
						default:
						}
					}
					lifecycleWait(t, rec.cancelled)
					var body string
					if transport == "sse" {
						select {
						case msg := <-session.messages:
							body = string(msg)
						case <-time.After(time.Second):
							t.Fatal("SSE answer waited for cleanup")
						}
						if w.Code != http.StatusAccepted {
							t.Errorf("status %d", w.Code)
						}
					} else {
						select {
						case <-done:
						case <-time.After(time.Second):
							t.Fatal("HTTP answer waited for cleanup")
						}
						body = strings.TrimSpace(w.Body.String())
						if w.Code != http.StatusOK {
							t.Errorf("status %d", w.Code)
						}
					}
					elapsed := time.Since(started)
					if elapsed < limit || elapsed > limit+time.Second {
						t.Errorf("deadline answer took %s", elapsed)
					}
					// Exact baseline failure answer: no receipt, signature or random identity.
					const want = `{"jsonrpc":"2.0","id":71,"error":{"code":-32603,"message":"Evidence proof unavailable","data":{"error":"proof_unavailable","tool":"paid_read"}}}`
					if body != want {
						t.Errorf("deadline answer differs: %s", body)
					}
					if streamableForwardCount(e.proxy) != 0 {
						t.Error("unsigned call forwarded")
					}
					if spent, _, committed, _ := store.counts(); spent != 0 || committed != 0 {
						t.Error("unsigned call charged")
					}
					var drained <-chan struct{}
					if tracked {
						drained = tracker.stop()
						lifecycleBlocked(t, drained)
					}
					once.Do(func() { close(rec.release) })
					lifecycleWait(t, rec.finished)
					if tracked {
						lifecycleWait(t, drained)
						if a, b, c := tracker.counts(); a != 0 || b != 1 || c != 1 {
							t.Errorf("cleanup counts %d/%d/%d", a, b, c)
						}
					}
					t.Logf("tracked=%t answer at %s while recorder cleanup held", tracked, elapsed)
				})
			}
		})
	}
}

type operationNilEndTracker struct{}

func (operationNilEndTracker) BeginOperation() (func(), bool) { return nil, true }
func TestOperationNilEndRefused(t *testing.T) {
	for _, transport := range []string{"sse", "streamable", "sse-connect"} {
		t.Run(transport, func(t *testing.T) {
			s, p, sseID, streamID := lifecycleServer(t, operationNilEndTracker{})
			gate := &countingGate{}
			p.SetCallGate(gate)
			rec := &lockedRecorder{}
			p.SetEvidenceRecorder(rec)
			path := "/message?sessionId=" + sseID
			if transport == "streamable" {
				path = "/mcp"
			}
			req := lifecycleRequest(path, streamID, `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo"}}`)
			if transport == "sse-connect" {
				req.Method = http.MethodGet
				req.URL.Path = "/sse"
				p.SetAuthenticator(rejectingAuthenticator{err: errors.New("token revoked")})
				req.Header.Set("Authorization", "Bearer synthetic-revoked")
				req = req.WithContext(context.WithValue(req.Context(), CtxTenantID, eaTenant))
			}
			body := &lifecycleReadSpy{Reader: req.Body}
			req.Body = body
			w := httptest.NewRecorder()
			func() {
				defer func() {
					if v := recover(); v != nil {
						t.Errorf("nil end panicked: %v", v)
					}
				}()
				s.ServeHTTP(w, req)
			}()
			if transport == "sse" && w.Code == http.StatusAccepted {
				// Join an incorrectly dispatched worker on the fail-before head.
				select {
				case <-s.sessions[sseID].messages:
				case <-time.After(5 * time.Second):
					t.Fatal("unexpected dispatch did not finish")
				}
			}
			if w.Code != http.StatusServiceUnavailable || body.reads != 0 {
				t.Errorf("status=%d reads=%d", w.Code, body.reads)
			}
			if streamableForwardCount(p) != 0 || gate.admits.Load() != 0 || len(rec.decisions) != 0 || len(p.PendingRecords()) != 0 {
				t.Error("nil-end request dispatched")
			}
		})
	}
}
