package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/toolcontrols"
)

// orderUpstream is a disposable localhost MCP server (Streamable HTTP) that
// stands in for a brokerage. It never talks to a real one. For each
// place_crypto_order call it records the order as executed (the side effect),
// then answers the way mode says.
type orderUpstream struct {
	srv      *httptest.Server
	mode     atomic.Value // string
	executed atomic.Int64 // dollars "placed"
	calls    atomic.Int64
	mu       sync.Mutex
	release  chan struct{}
}

func newOrderUpstream(t *testing.T) *orderUpstream {
	t.Helper()
	u := &orderUpstream{release: make(chan struct{})}
	u.mode.Store("ok")
	u.srv = httptest.NewServer(http.HandlerFunc(u.handle))
	t.Cleanup(func() {
		close(u.release)
		u.srv.Close()
	})
	return u
}

func (u *orderUpstream) handle(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		w.WriteHeader(http.StatusMethodNotAllowed)
		return
	}
	body, _ := io.ReadAll(r.Body)
	var req struct {
		ID     json.RawMessage `json:"id"`
		Method string          `json:"method"`
		Params struct {
			Name      string `json:"name"`
			Arguments struct {
				Amount json.RawMessage `json:"dollar_amount"`
			} `json:"arguments"`
		} `json:"params"`
	}
	_ = json.Unmarshal(body, &req)
	reply := func(result string) {
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"result":` + result + `}`))
	}
	switch req.Method {
	case MethodInitialize:
		reply(`{"protocolVersion":"2024-11-05","capabilities":{},"serverInfo":{"name":"fake","version":"0"}}`)
	case MethodInitialized:
		w.WriteHeader(http.StatusAccepted)
	case MethodToolsList:
		reply(`{"tools":[{"name":"place_crypto_order","inputSchema":{"type":"object"}}]}`)
	case MethodToolsCall:
		u.calls.Add(1)
		mode := u.mode.Load().(string)
		if mode != "reject" && mode != "rpcerror" {
			// The order is carried out before the answer is decided.
			var amt json.Number
			_ = json.NewDecoder(bytes.NewReader(req.Params.Arguments.Amount)).Decode(&amt)
			if amt == "" {
				var s string
				_ = json.Unmarshal(req.Params.Arguments.Amount, &s)
				amt = json.Number(s)
			}
			if n, err := amt.Int64(); err == nil {
				u.executed.Add(n)
			}
		}
		switch mode {
		case "lost": // executed, then close the connection without a reply
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
		case "slow": // executed, then holds the request open
			select {
			case <-u.release:
			case <-r.Context().Done():
			}
		case "silent": // executed, then an event stream that never carries the reply
			w.Header().Set("Content-Type", "text/event-stream")
			w.WriteHeader(http.StatusOK)
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			select {
			case <-u.release:
			case <-r.Context().Done():
			}
		case "http500": // executed, then a bare 500
			w.WriteHeader(http.StatusInternalServerError)
		case "partial": // executed, then a body that stops half way
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Content-Length", "500")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":`))
			if f, ok := w.(http.Flusher); ok {
				f.Flush()
			}
			conn, _, err := w.(http.Hijacker).Hijack()
			if err == nil {
				conn.Close()
			}
		case "rpcerror":
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"jsonrpc":"2.0","id":` + string(req.ID) + `,"error":{"code":-32000,"message":"no"}}`))
		case "reject":
			reply(`{"content":[{"type":"text","text":"no"}],"isError":true}`)
		default:
			reply(`{"content":[{"type":"text","text":"ok"}],"isError":false}`)
		}
	default:
		w.WriteHeader(http.StatusBadRequest)
	}
}

// httpSpendHarness is a proxy with the real UpstreamManager and the real
// Streamable HTTP transport talking to an orderUpstream.
func httpSpendHarness(t *testing.T, up *orderUpstream, timeout time.Duration) *spendHarness {
	t.Helper()
	h := newSpendHarness(t, spendDoc200Day)
	cfg := h.proxy.config
	cfg.AllowPrivateUpstreams = true
	cfg.DefaultUpstream = "u"
	cfg.Upstreams = map[string]UpstreamConfig{"u": {Transport: "streamable", URL: up.srv.URL, Timeout: timeout}}
	mgr := NewUpstreamManager(cfg.Upstreams, nil, "u", true)
	ctx, cancel := context.WithCancel(context.Background())
	if err := mgr.Start(ctx); err != nil {
		cancel()
		t.Fatalf("start upstream: %v", err)
	}
	t.Cleanup(func() { cancel(); mgr.Close() })
	h.proxy.upstream = mgr
	h.proxy.SetUpstreamRouter(&defaultRouter{mgr: mgr})
	return h
}

// Astra's scenario (TestReviewExecutedOrderLostResponse): the upstream records
// a $200 order and closes the connection without replying. Two calls under a
// $200/day limit must not execute $400.
func TestReviewExecutedOrderLostResponse(t *testing.T) {
	up := newOrderUpstream(t)
	up.mode.Store("lost")
	h := httpSpendHarness(t, up, 2*time.Second)

	h.call(t, 1, "place_crypto_order", order("200"))
	if h.counter(t) != 200*units {
		t.Fatalf("counter after an executed order with a lost reply = %d, want 200", h.counter(t)/units)
	}
	r := h.call(t, 2, "place_crypto_order", order("200"))
	if got := errData(t, r)["error"]; got != toolcontrols.DenialCodeSpendLimit {
		t.Fatalf("second order: error = %v, want a spend-limit refusal", got)
	}
	if up.executed.Load() != 200 || up.calls.Load() != 1 {
		t.Fatalf("upstream executed %d over %d calls, want 200 over 1", up.executed.Load(), up.calls.Load())
	}
}

// Every way of ending after the send, other than an answer from the upstream
// that says it did not run, keeps the reservation.
func TestSpendKeptAfterSendWhateverTheFailure(t *testing.T) {
	for _, mode := range []string{"lost", "silent", "http500", "partial"} {
		t.Run(mode, func(t *testing.T) {
			up := newOrderUpstream(t)
			up.mode.Store(mode)
			h := httpSpendHarness(t, up, 400*time.Millisecond)
			h.call(t, 1, "place_crypto_order", order("200"))
			if h.counter(t) != 200*units {
				t.Fatalf("%s: counter = %d, want 200 kept", mode, h.counter(t)/units)
			}
			if r := h.call(t, 2, "place_crypto_order", order("200")); r.Error == nil {
				t.Fatalf("%s: a second order went through", mode)
			}
			if up.executed.Load() != 200 {
				t.Fatalf("%s: upstream executed %d", mode, up.executed.Load())
			}
		})
	}
}

// A context cancelled after the request was dispatched keeps the reservation.
func TestSpendKeptWhenContextEndsAfterDispatch(t *testing.T) {
	up := newOrderUpstream(t)
	up.mode.Store("slow")
	h := httpSpendHarness(t, up, 10*time.Second)
	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		for up.calls.Load() == 0 {
			time.Sleep(5 * time.Millisecond)
		}
		cancel()
	}()
	if _, err := h.proxy.handleRequest(ctx, argCall(1, h.token, "place_crypto_order", order("200"))); err != nil {
		t.Fatal(err)
	}
	if h.counter(t) != 200*units {
		t.Fatalf("counter = %d, want 200 kept", h.counter(t)/units)
	}
}

// The upstream answered "I did not do this": released.
func TestSpendReleasedWhenUpstreamSaysItDidNotRun(t *testing.T) {
	for _, mode := range []string{"rpcerror", "reject"} {
		t.Run(mode, func(t *testing.T) {
			up := newOrderUpstream(t)
			up.mode.Store(mode)
			h := httpSpendHarness(t, up, 2*time.Second)
			h.call(t, 1, "place_crypto_order", order("200"))
			if h.counter(t) != 0 {
				t.Fatalf("%s: counter = %d, want released", mode, h.counter(t)/units)
			}
		})
	}
}

// A dial failure before any byte is written: released, and a retry once the
// upstream is back may use the whole allowance.
func TestSpendReleasedWhenDialFailsBeforeSend(t *testing.T) {
	up := newOrderUpstream(t)
	h := httpSpendHarness(t, up, 2*time.Second)
	up.srv.Close() // connection refused from here on
	r := h.call(t, 1, "place_crypto_order", order("200"))
	if r.Error == nil {
		t.Fatalf("call to a dead upstream succeeded: %s", r.Result)
	}
	if h.counter(t) != 0 {
		t.Fatalf("counter after a dial failure = %d, want released", h.counter(t)/units)
	}
	if up.executed.Load() != 0 {
		t.Fatalf("executed %d", up.executed.Load())
	}
}

func TestDispatchRecordClassification(t *testing.T) {
	ok := &Response{Result: json.RawMessage(`{"isError":false}`)}
	isErr := &Response{Result: json.RawMessage(`{"isError":true}`)}
	rpcErr := &Response{Error: &RPCError{Code: -1, Message: "x"}}
	cases := []struct {
		name      string
		forwarded bool
		attempted bool
		sent      bool
		resp      *Response
		err       error
		want      callEnd
	}{
		{"never handed to the router", false, false, false, nil, nil, endNotRun},
		{"answered ok", true, true, true, ok, nil, endRan},
		{"answered isError", true, true, true, isErr, nil, endNotRun},
		{"answered JSON-RPC error", true, true, true, rpcErr, nil, endNotRun},
		{"failed, attempted, not sent", true, true, false, nil, errors.New("dial"), endNotRun},
		{"failed after send", true, true, true, nil, errors.New("eof"), endUnknown},
		{"failed, router never said", true, false, false, nil, errors.New("?"), endUnknown},
		{"no reply and no error", true, true, true, nil, nil, endUnknown},
	}
	for _, c := range cases {
		r := &dispatchRecord{forwarded: c.forwarded, resp: c.resp, err: c.err}
		r.attempted.Store(c.attempted)
		r.sent.Store(c.sent)
		if got := r.classify(); got != c.want {
			t.Errorf("%s: classify = %d, want %d", c.name, got, c.want)
		}
	}
}

func TestStdioWriteRecordsDispatch(t *testing.T) {
	ctx, rec := withDispatchRecord(context.Background())
	pr, pw := io.Pipe()
	pr.Close() // the reader is gone: nothing can be written
	tr := NewStdioTransport(bytes.NewReader(nil), pw, nil)
	if err := tr.WriteMessage(ctx, json.RawMessage(`{"a":1}`)); err == nil {
		t.Fatal("write to a closed pipe succeeded")
	}
	if rec.sent.Load() {
		t.Fatal("a write that delivered nothing counted as sent")
	}

	ctx2, rec2 := withDispatchRecord(context.Background())
	var buf bytes.Buffer
	if err := NewStdioTransport(bytes.NewReader(nil), &buf, nil).WriteMessage(ctx2, json.RawMessage(`{"a":1}`)); err != nil {
		t.Fatal(err)
	}
	if !rec2.sent.Load() {
		t.Fatal("a write that went through did not count as sent")
	}
	_ = os.ErrClosed
}
