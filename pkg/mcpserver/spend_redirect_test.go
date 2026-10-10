package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// redirectOrderUpstream is a disposable headerless Streamable HTTP upstream
// (it never issues a session id) that stands in for a brokerage. A tools/call
// to /mcp carries out the order, then answers 307 to /next. /next carries out
// the order again if it is ever asked to (which is the replay), then answers
// with nextReply: a JSON-RPC error or a success.
type redirectOrderUpstream struct {
	srv       *httptest.Server
	executed  atomic.Int64 // dollars "placed", by either hop
	firstHops atomic.Int64
	replays   atomic.Int64
	nextReply string // "error" or "ok"
}

func newRedirectOrderUpstream(t *testing.T, nextReply string) *redirectOrderUpstream {
	t.Helper()
	u := &redirectOrderUpstream{nextReply: nextReply}
	mux := http.NewServeMux()
	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		req := decodeOrderRequest(body)
		if req.Method != MethodToolsCall {
			answerOrderRequest(w, req)
			return
		}
		u.firstHops.Add(1)
		u.executed.Add(req.amount())
		http.Redirect(w, r, "/next", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/next", func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		req := decodeOrderRequest(body)
		u.replays.Add(1)
		u.executed.Add(req.amount())
		w.Header().Set("Content-Type", "application/json")
		if u.nextReply == "error" {
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32602,"message":"invalid params"}}`))
			return
		}
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":{"content":[{"type":"text","text":"ok"}],"isError":false}}`))
	})
	u.srv = httptest.NewServer(mux)
	t.Cleanup(u.srv.Close)
	return u
}

type orderRequest struct {
	ID     json.RawMessage `json:"id"`
	Method string          `json:"method"`
	Params struct {
		Arguments struct {
			Amount json.Number `json:"dollar_amount"`
		} `json:"arguments"`
	} `json:"params"`
}

func decodeOrderRequest(body []byte) orderRequest {
	var r orderRequest
	_ = json.Unmarshal(body, &r)
	return r
}

func (r orderRequest) amount() int64 {
	n, _ := r.Params.Arguments.Amount.Int64()
	return n
}

func answerOrderRequest(w http.ResponseWriter, req orderRequest) {
	w.Header().Set("Content-Type", "application/json")
	switch req.Method {
	case MethodInitialize:
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"fake","version":"0"}}}`))
	case MethodToolsList:
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":{"tools":[{"name":"place_crypto_order","inputSchema":{"type":"object"}}]}}`))
	default:
		w.WriteHeader(http.StatusAccepted)
	}
}

// Grok's round-2 scenarios against the real UpstreamManager and the real
// Streamable HTTP transport. The first hop executes $200 and answers 307.
// Before the fix the body was sent again to /next: with a JSON-RPC error there
// the reservation was released and the next $200 ran too (executed 400,
// counter 0); with a success there one reservation paid for two executions.
func TestSpendRedirectAfterSendIsNotFollowed(t *testing.T) {
	for _, next := range []string{"error", "ok"} {
		t.Run("307 then "+next, func(t *testing.T) {
			up := newRedirectOrderUpstream(t, next)
			h := httpSpendHarness(t, &orderUpstream{srv: up.srv}, 2*time.Second)

			first := h.call(t, 1, "place_crypto_order", order("200"))
			if first.Error == nil {
				t.Fatalf("a redirected tools/call was answered as a success: %s", first.Result)
			}
			if up.replays.Load() != 0 {
				t.Fatalf("the call body was sent again to the redirect target %d time(s)", up.replays.Load())
			}
			if h.counter(t) != 200*units {
				t.Fatalf("counter = %d, want the 200 reservation kept", h.counter(t)/units)
			}
			second := h.call(t, 2, "place_crypto_order", order("200"))
			if got := errData(t, second)["error"]; got != toolcontrols.DenialCodeSpendLimit {
				t.Fatalf("second order: error = %v, want a spend-limit refusal", got)
			}
			if up.executed.Load() != 200 || up.firstHops.Load() != 1 {
				t.Fatalf("upstream executed %d over %d first hops, want 200 over 1", up.executed.Load(), up.firstHops.Load())
			}
		})
	}
}

// Redirects for methods without a side effect behave as before.
func TestRedirectStillFollowedForNonCallMethods(t *testing.T) {
	var hits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/next", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/next", func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		writeJSONResult(w)
	})
	srv := httptest.NewServer(mux)
	defer srv.Close()
	for _, method := range []string{MethodToolsList, MethodInitialize, "ping"} {
		before := hits.Load()
		tr := NewStreamableHTTPTransport(srv.URL+"/mcp", nil, false, true)
		ctx, cancel := contextWithTimeout(t)
		if err := tr.Connect(ctx); err != nil {
			t.Fatal(err)
		}
		if err := tr.WriteMessage(ctx, redirectBody(map[string]any{"jsonrpc": "2.0", "id": 1, "method": method})); err != nil {
			t.Fatalf("%s: same-origin redirect should still be followed: %v", method, err)
		}
		if hits.Load() != before+1 {
			t.Fatalf("%s: target hits = %d, want %d", method, hits.Load(), before+1)
		}
		tr.Close()
		cancel()
	}
}

// A request that carries a tools/call (or cannot be read, or is a batch that
// might hold one) gets no GetBody, so neither redirectRequest nor net/http's
// own retry (HTTP/2 REFUSED_STREAM, a reused connection that died) can send
// the body again. Other methods keep GetBody.
func TestMarkNoReplay(t *testing.T) {
	cases := []struct {
		name   string
		msg    string
		replay bool // false: marked, GetBody removed
	}{
		{"tools/call", `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{}}`, false},
		{"batch", `[{"jsonrpc":"2.0","id":1,"method":"tools/call"}]`, false},
		{"unreadable", `not json`, false},
		{"tools/list", `{"jsonrpc":"2.0","id":1,"method":"tools/list"}`, true},
		{"initialize", `{"jsonrpc":"2.0","id":1,"method":"initialize"}`, true},
		{"notification", `{"jsonrpc":"2.0","method":"notifications/initialized"}`, true},
	}
	for _, c := range cases {
		req, err := http.NewRequest("POST", "http://example.invalid/mcp", stringsReader(c.msg))
		if err != nil {
			t.Fatal(err)
		}
		if req.GetBody == nil {
			t.Fatal("test setup: no GetBody")
		}
		got := markNoReplay(req, []byte(c.msg))
		if (got.GetBody != nil) != c.replay {
			t.Errorf("%s: GetBody set = %v, want %v", c.name, got.GetBody != nil, c.replay)
		}
		if noReplay(got) == c.replay {
			t.Errorf("%s: noReplay = %v, want %v", c.name, noReplay(got), !c.replay)
		}
	}
}

func stringsReader(s string) io.Reader { return strings.NewReader(s) }

func contextWithTimeout(t *testing.T) (context.Context, context.CancelFunc) {
	t.Helper()
	return context.WithTimeout(context.Background(), 5*time.Second)
}

// The legacy SSE transport POSTs to the message endpoint. With no stored
// headers a same-origin 307 used to be followed with the body; for a
// tools/call it is now refused after the send, and nothing is sent twice.
func TestSSEPostRedirectOfToolsCallIsNotFollowed(t *testing.T) {
	var firstHits, replayHits atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/sse":
			w.Header().Set("Content-Type", "text/event-stream")
			_, _ = io.WriteString(w, "event: endpoint\ndata: /message\n\n")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		case "/message":
			firstHits.Add(1)
			_, _ = io.Copy(io.Discard, r.Body)
			http.Redirect(w, r, "/next", http.StatusTemporaryRedirect)
		case "/next":
			replayHits.Add(1)
			_, _ = io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusAccepted)
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	tr := NewSSETransport(server.URL+"/sse", nil, false, true)
	defer tr.Close()
	ctx, cancel := contextWithTimeout(t)
	defer cancel()
	if err := tr.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	callCtx, rec := withDispatchRecord(ctx)
	err := tr.WriteMessage(callCtx, redirectBody(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/call", "params": map[string]any{"name": "place_crypto_order"}}))
	if err == nil {
		t.Fatal("a redirected tools/call POST was reported as sent successfully")
	}
	if firstHits.Load() != 1 || replayHits.Load() != 0 {
		t.Fatalf("first hop hits=%d replay hits=%d, want 1 and 0", firstHits.Load(), replayHits.Load())
	}
	if !rec.sent.Load() {
		t.Fatal("the dispatch record does not say the call was sent")
	}
}
