package mcpserver

import (
	"context"
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

func (t *SSETransport) endpointForTest() string {
	t.mu.Lock()
	defer t.mu.Unlock()
	return t.messageEndpoint
}

// countingServer counts every request it receives.
func countingServer(hits *atomic.Int32) *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		http.Error(w, "unexpected request", http.StatusBadGateway)
	}))
}

// sseEventServer streams the given endpoint values as separate events in a
// single flush, then ends the stream (or holds it open when hold is true).
func sseEventServer(endpoints func(serverURL string) []string, hold bool, posts *atomic.Int32) *httptest.Server {
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			posts.Add(1)
			io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusAccepted)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		var b strings.Builder
		for _, e := range endpoints(server.URL) {
			fmt.Fprintf(&b, "event: endpoint\ndata: %s\n\n", e)
		}
		io.WriteString(w, b.String())
		w.(http.Flusher).Flush()
		if hold {
			<-r.Context().Done()
		}
	}))
	return server
}

func TestSSEEndpoint_RefusedThenValidSameFlushFailsConnect(t *testing.T) {
	var otherHits, posts atomic.Int32
	other := countingServer(&otherHits)
	defer other.Close()
	server := sseEventServer(func(u string) []string {
		return []string{other.URL + "/message", u + "/message?session=ok"}
	}, true, &posts)
	defer server.Close()

	transport := NewSSETransport(server.URL+"/sse", nil, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := transport.Connect(ctx)
	if err == nil {
		t.Fatal("expected connect to fail after a refused endpoint")
	}
	if !strings.Contains(err.Error(), "SSE endpoint origin "+originOf(other.URL)) {
		t.Fatalf("unexpected error: %v", err)
	}
	if got := transport.endpointForTest(); got != "" {
		t.Fatalf("endpoint stored after refusal: %q", got)
	}
	if err := transport.WriteMessage(ctx, redirectBody(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"})); err == nil {
		t.Fatal("WriteMessage should fail after a refused endpoint")
	}
	transport.Close()
	if posts.Load() != 0 {
		t.Fatalf("same-origin server received %d POST requests", posts.Load())
	}
	waitNoHits(t, &otherHits)
}

// The reader handles every event in the stream before Connect selects. The
// refusal must still be what Connect returns.
func TestSSEEndpoint_RefusedThenValidReaderAheadFailsConnect(t *testing.T) {
	for i := 0; i < 20; i++ {
		var otherHits, posts atomic.Int32
		other := countingServer(&otherHits)
		server := sseEventServer(func(u string) []string {
			return []string{other.URL + "/message", u + "/message?session=ok"}
		}, false, &posts)

		transport := NewSSETransport(server.URL+"/sse", nil, false, true)
		var once sync.Once
		readerDone := make(chan struct{})
		transport.onStreamEnd = func() { once.Do(func() { close(readerDone) }) }
		transport.beforeConnect = func() {
			select {
			case <-readerDone:
			case <-time.After(5 * time.Second):
			}
		}

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := transport.Connect(ctx)
		cancel()
		if err == nil {
			t.Fatalf("iteration %d: connect succeeded after a refused endpoint", i)
		}
		if !strings.Contains(err.Error(), "SSE endpoint origin") {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}
		if got := transport.endpointForTest(); got != "" {
			t.Fatalf("iteration %d: endpoint stored after refusal: %q", i, got)
		}
		transport.Close()
		if posts.Load() != 0 || otherHits.Load() != 0 {
			t.Fatalf("iteration %d: requests after refusal posts=%d other=%d", i, posts.Load(), otherHits.Load())
		}
		server.Close()
		other.Close()
	}
}

func TestSSEEndpoint_AcceptedThenBadKeepsAccepted(t *testing.T) {
	var otherHits, posts atomic.Int32
	other := countingServer(&otherHits)
	defer other.Close()
	server := sseEventServer(func(u string) []string {
		return []string{u + "/message?session=ok", other.URL + "/message"}
	}, false, &posts)
	defer server.Close()

	transport := NewSSETransport(server.URL+"/sse", nil, false, true)
	var once sync.Once
	readerDone := make(chan struct{})
	transport.onStreamEnd = func() { once.Do(func() { close(readerDone) }) }
	transport.beforeConnect = func() {
		select {
		case <-readerDone:
		case <-time.After(5 * time.Second):
		}
	}
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	want := server.URL + "/message?session=ok"
	if got := transport.endpointForTest(); got != want {
		t.Fatalf("endpoint=%q want %q", got, want)
	}
	if err := transport.WriteMessage(ctx, redirectBody(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"})); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	if posts.Load() != 1 {
		t.Fatalf("posts=%d want 1", posts.Load())
	}
	waitNoHits(t, &otherHits)
}

// sseResolveCase serves an SSE stream at streamPath (optionally reached through
// a same-origin redirect from entryPath) and checks where the POST lands.
func runSSEResolveCase(t *testing.T, entryPath, streamPath, endpoint, wantPostPath string) {
	t.Helper()
	var gotPath atomic.Value
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.Method == http.MethodPost:
			gotPath.Store(r.URL.Path)
			io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusAccepted)
		case r.URL.Path == entryPath && entryPath != streamPath:
			http.Redirect(w, r, streamPath, http.StatusFound)
		case r.URL.Path == streamPath:
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpoint)
			w.(http.Flusher).Flush()
			<-r.Context().Done()
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	configured := server.URL + entryPath
	transport := NewSSETransport(configured, nil, false, true)
	defer transport.Close()
	if transport.baseURL != configured {
		t.Fatalf("configured URL changed: %q", transport.baseURL)
	}
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	if want := server.URL + wantPostPath; transport.endpointForTest() != want {
		t.Fatalf("endpoint=%q want %q", transport.endpointForTest(), want)
	}
	if err := transport.WriteMessage(ctx, redirectBody(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"})); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	if got, _ := gotPath.Load().(string); got != wantPostPath {
		t.Fatalf("POST path=%q want %q", got, wantPostPath)
	}
}

func TestSSEEndpoint_RelativeResolvesAgainstStreamURL(t *testing.T) {
	t.Run("trailing-slash-base", func(t *testing.T) {
		runSSEResolveCase(t, "/nested/sse/", "/nested/sse/", "messages", "/nested/sse/messages")
	})
	t.Run("same-origin-redirect-uses-final-url", func(t *testing.T) {
		runSSEResolveCase(t, "/initial/sse", "/final/sse", "messages", "/final/messages")
	})
	t.Run("absolute-path", func(t *testing.T) {
		runSSEResolveCase(t, "/a/b/sse", "/a/b/sse", "/messages", "/messages")
	})
	t.Run("dot-segment", func(t *testing.T) {
		runSSEResolveCase(t, "/a/b/sse", "/a/b/sse", "./messages", "/a/b/messages")
	})
	t.Run("dot-dot-segment", func(t *testing.T) {
		runSSEResolveCase(t, "/a/b/sse", "/a/b/sse", "../messages", "/a/messages")
	})
}

func TestSSEEndpoint_RelativeCannotChangeOrigin(t *testing.T) {
	var otherHits, posts atomic.Int32
	other := countingServer(&otherHits)
	defer other.Close()
	otherHost := strings.TrimPrefix(other.URL, "http://")
	for _, endpoint := range []string{
		"//" + otherHost + "/messages",
		"http://" + otherHost + "/messages",
	} {
		server := sseEventServer(func(string) []string { return []string{endpoint} }, true, &posts)
		transport := NewSSETransport(server.URL+"/sse", nil, false, true)
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		err := transport.Connect(ctx)
		cancel()
		if err == nil {
			t.Fatalf("%q: connect succeeded", endpoint)
		}
		if got := transport.endpointForTest(); got != "" {
			t.Fatalf("%q: endpoint stored: %q", endpoint, got)
		}
		transport.Close()
		server.Close()
	}
	if posts.Load() != 0 {
		t.Fatalf("posts=%d", posts.Load())
	}
	waitNoHits(t, &otherHits)
}
