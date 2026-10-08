package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// brokenTransport fails every read, as a transport does after its connection
// dropped.
type brokenTransport struct{}

func (*brokenTransport) ReadMessage(context.Context) (json.RawMessage, error) { return nil, io.EOF }
func (*brokenTransport) WriteMessage(context.Context, json.RawMessage) error  { return nil }
func (*brokenTransport) Close() error                                         { return nil }

// countingUpstream is a streamable upstream that counts the requests it gets
// and answers initialize and tools/list.
func countingUpstream(t *testing.T, hits *atomic.Int32, failInit bool) *httptest.Server {
	t.Helper()
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		var env struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&env)
		w.Header().Set("Content-Type", "application/json")
		switch env.Method {
		case "initialize":
			if failInit {
				fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"error":{"code":-32000,"message":"no"}}`, env.ID)
				return
			}
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, env.ID)
		case "tools/list":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"tools":[{"name":"t"}]}}`, env.ID)
		default:
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	t.Cleanup(srv.Close)
	return srv
}

// N1: Close while a read loop waits out its reconnect backoff means no further
// request to the upstream, and the loop ends at once (not when the backoff
// would have run out).
func TestCloseDuringBackoffMakesNoRequestAndEndsTheLoop(t *testing.T) {
	buf := captureLogs(t)
	var hits atomic.Int32
	srv := countingUpstream(t, &hits, false)
	m := NewUpstreamManager(map[string]UpstreamConfig{"up": {Transport: "streamable", URL: srv.URL}}, nil, "", true)
	c := &UpstreamClient{name: "up", transport: &brokenTransport{}}
	m.clients["up"] = c
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { m.readLoop(ctx, c); close(done) }()

	deadline := time.Now().Add(2 * time.Second)
	for !strings.Contains(buf.String(), "reconnecting upstream") && time.Now().Before(deadline) {
		time.Sleep(time.Millisecond)
	}
	start := time.Now()
	m.Close()
	select {
	case <-done:
	case <-time.After(1500 * time.Millisecond):
		t.Fatal("read loop still waiting out the backoff after Close (ctx not cancelled)")
	}
	if d := time.Since(start); d > time.Second {
		t.Fatalf("loop took %v to end after Close", d)
	}
	time.Sleep(2500 * time.Millisecond) // longer than the first backoff
	if n := hits.Load(); n != 0 {
		t.Fatalf("closed manager made %d upstream requests", n)
	}
}

// N1: an open manager still reconnects, and a reconnect replaces the client's
// dead transport (the upstream sees a new initialize and the client is ready).
func TestCloseDuringBackoffOpenManagerStillReconnects(t *testing.T) {
	captureLogs(t)
	var hits atomic.Int32
	srv := countingUpstream(t, &hits, false)
	m := NewUpstreamManager(map[string]UpstreamConfig{"up": {Transport: "streamable", URL: srv.URL, Timeout: 2 * time.Second}}, nil, "", true)
	c := &UpstreamClient{name: "up", transport: &brokenTransport{}}
	m.clients["up"] = c
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	loopDone := make(chan struct{})
	go func() { m.readLoop(ctx, c); close(loopDone) }()
	deadline := time.Now().Add(6 * time.Second)
	for hits.Load() < 3 && time.Now().Before(deadline) { // initialize, initialized, tools/list
		time.Sleep(20 * time.Millisecond)
	}
	if hits.Load() < 3 {
		t.Fatalf("open manager did not reconnect: %d requests", hits.Load())
	}
	select {
	case <-loopDone: // the loop that reconnected ends once it published
	case <-time.After(3 * time.Second):
		t.Fatal("the reconnecting loop did not end after publishing")
	}
	m.mu.Lock()
	cur := m.clients["up"]
	m.mu.Unlock()
	if cur == nil || cur == c || !cur.ready || len(cur.toolNames) != 1 {
		t.Fatalf("reconnect did not publish a ready replacement: %+v", cur)
	}
	m.Close()
	after := hits.Load()
	time.Sleep(300 * time.Millisecond)
	if hits.Load() != after {
		t.Fatal("requests after Close")
	}
}

// N1: a Close that lands while a replacement is being initialized closes the
// replacement; nothing is published and nothing else is dialed.
func TestCloseDuringBackoffCloseWhileReplacementInitializes(t *testing.T) {
	captureLogs(t)
	var hits atomic.Int32
	entered, release := make(chan struct{}, 1), make(chan struct{})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		select {
		case entered <- struct{}{}:
		default:
		}
		select {
		case <-release:
		case <-r.Context().Done():
		}
	}))
	defer srv.Close()
	defer close(release)
	m := NewUpstreamManager(map[string]UpstreamConfig{"up": {Transport: "streamable", URL: srv.URL, Timeout: 20 * time.Second}}, nil, "", true)
	c := &UpstreamClient{name: "up", transport: &brokenTransport{}}
	m.clients["up"] = c
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	done := make(chan struct{})
	go func() { m.readLoop(ctx, c); close(done) }()
	select {
	case <-entered:
	case <-time.After(6 * time.Second):
		t.Fatal("no reconnect attempt")
	}
	m.Close()
	select {
	case <-done:
	case <-time.After(3 * time.Second):
		t.Fatal("read loop did not end after Close during initialize")
	}
	m.mu.Lock()
	cur := m.clients["up"]
	n := len(m.inflight)
	m.mu.Unlock()
	if cur != c || n != 0 {
		t.Fatalf("a replacement was published or left tracked: same=%v inflight=%d", cur == c, n)
	}
}

// N1: a manager that is closed refuses to start or add anything and dials
// nothing.
func TestCloseDuringBackoffClosedManagerDialsNothing(t *testing.T) {
	captureLogs(t)
	var hits atomic.Int32
	srv := countingUpstream(t, &hits, false)
	cfg := UpstreamConfig{Transport: "streamable", URL: srv.URL}
	m := NewUpstreamManager(map[string]UpstreamConfig{"up": cfg}, nil, "", true)
	m.Close()
	if err := m.Start(context.Background()); err == nil {
		t.Error("Start on a closed manager succeeded")
	}
	if err := m.AddUpstream(context.Background(), "two", cfg); err == nil {
		t.Error("AddUpstream on a closed manager succeeded")
	}
	if hits.Load() != 0 {
		t.Fatalf("closed manager dialed: %d requests", hits.Load())
	}
}

// N1: a Close that lands while AddUpstream is still connecting keeps nothing.
func TestCloseDuringBackoffAddUpstreamRacingClose(t *testing.T) {
	captureLogs(t)
	entered, release := make(chan struct{}, 1), make(chan struct{})
	var hits atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		var env struct {
			Method string          `json:"method"`
			ID     json.RawMessage `json:"id"`
		}
		_ = json.NewDecoder(r.Body).Decode(&env)
		if env.Method == "initialize" {
			select {
			case entered <- struct{}{}:
			default:
			}
			<-release
		}
		w.Header().Set("Content-Type", "application/json")
		switch env.Method {
		case "initialize":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{}}`, env.ID)
		case "tools/list":
			fmt.Fprintf(w, `{"jsonrpc":"2.0","id":%s,"result":{"tools":[]}}`, env.ID)
		default:
			w.WriteHeader(http.StatusAccepted)
		}
	}))
	defer srv.Close()
	m := NewUpstreamManager(map[string]UpstreamConfig{}, nil, "", true)
	errc := make(chan error, 1)
	go func() {
		errc <- m.AddUpstream(context.Background(), "up", UpstreamConfig{Transport: "streamable", URL: srv.URL, Timeout: 5 * time.Second})
	}()
	<-entered
	m.Close()
	close(release)
	if err := <-errc; err == nil {
		t.Fatal("AddUpstream succeeded on a manager closed meanwhile")
	}
	if names := m.UpstreamNames(); len(names) != 0 {
		t.Fatalf("closed manager kept %v", names)
	}
}

// N1: a failed start (single upstream: startOne; several: parallel) and a
// failed AddUpstream close their transport and keep no client, and nothing
// reconnects afterwards.
func TestCloseDuringBackoffFailedStartLeavesNothingRunning(t *testing.T) {
	for name, run := range map[string]func(m *UpstreamManager, cfg UpstreamConfig) error{
		"startOne": func(m *UpstreamManager, _ UpstreamConfig) error { return m.Start(context.Background()) },
		"add": func(m *UpstreamManager, cfg UpstreamConfig) error {
			return m.AddUpstream(context.Background(), "up", cfg)
		},
	} {
		t.Run(name, func(t *testing.T) {
			buf := captureLogs(t)
			var hits atomic.Int32
			srv := countingUpstream(t, &hits, true)
			cfg := UpstreamConfig{Transport: "streamable", URL: srv.URL, Timeout: 2 * time.Second}
			conf := map[string]UpstreamConfig{}
			if name == "startOne" {
				conf["up"] = cfg
			}
			m := NewUpstreamManager(conf, nil, "", true)
			if err := run(m, cfg); err == nil {
				t.Fatal("start with a refusing initialize succeeded")
			}
			if names := m.UpstreamNames(); len(names) != 0 {
				t.Fatalf("failed start kept %v", names)
			}
			before := hits.Load()
			time.Sleep(2500 * time.Millisecond)
			if hits.Load() != before {
				t.Fatalf("failed start kept talking to the upstream: %d -> %d", before, hits.Load())
			}
			if out := buf.String(); strings.Contains(out, "reconnecting upstream") {
				t.Fatalf("failed start's read loop reconnected:\n%s", out)
			}
		})
	}
}
