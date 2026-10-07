package mcpserver

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"net/url"
	"regexp"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

// refusalPieces are parts of the configured URL that must not appear in a
// refusal error: userinfo, path, and query.
var refusalPieces = []string{"alice", "hunter2", "private-path", "key=abc123", "post-path", "sid=zzz"}

var urlInText = regexp.MustCompile(`[a-zA-Z][a-zA-Z0-9+.-]*://[^\s]+`)

func assertRefusalText(t *testing.T, label string, err error, status int, target, upstream string) {
	t.Helper()
	if err == nil {
		t.Fatalf("%s: expected an error", label)
	}
	text := err.Error()
	for _, piece := range refusalPieces {
		if strings.Contains(text, piece) {
			t.Fatalf("%s: error contains %q: %v", label, piece, err)
		}
	}
	if status != 0 {
		want := fmt.Sprintf("upstream redirect status %d to %s refused", status, originOf(target))
		if !strings.Contains(text, want) {
			t.Fatalf("%s: missing %q: %v", label, want, err)
		}
	} else if !strings.Contains(text, "SSE endpoint origin "+originOf(target)) {
		t.Fatalf("%s: missing endpoint origin: %v", label, err)
	}
	// Every URL in the text is an origin: the target's or the upstream's.
	allowed := map[string]bool{originOf(target): true, originOf(upstream): true}
	for _, found := range urlInText.FindAllString(text, -1) {
		found = strings.TrimRight(found, ":,.")
		if !allowed[found] {
			t.Fatalf("%s: error contains %q, which is not an allowed origin: %v", label, found, err)
		}
	}
}

// configuredWithUserinfo returns serverURL with userinfo, a path, and a query.
func configuredWithUserinfo(t *testing.T, serverURL, path string) string {
	t.Helper()
	u := mustURL(t, serverURL)
	u.User = url.UserPassword("alice", "hunter2")
	u.Path = path
	u.RawQuery = "key=abc123"
	return u.String()
}

func TestRefusalError_StreamableWriteMessage(t *testing.T) {
	var otherHits atomic.Int32
	other := countingServer(&otherHits)
	defer other.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/elsewhere?x=1", http.StatusTemporaryRedirect)
	}))
	defer upstream.Close()

	configured := configuredWithUserinfo(t, upstream.URL, "/private-path")
	transport := NewStreamableHTTPTransport(configured, map[string]string{"X-Key": "v"}, false, true)
	defer transport.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := transport.WriteMessage(ctx, redirectBody(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"}))
	assertRefusalText(t, "streamable POST", err, http.StatusTemporaryRedirect, other.URL, upstream.URL)
	waitNoHits(t, &otherHits)
}

func TestRefusalError_SSEGetAndManager(t *testing.T) {
	var otherHits atomic.Int32
	other := countingServer(&otherHits)
	defer other.Close()
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, other.URL+"/elsewhere?x=1", http.StatusFound)
	}))
	defer upstream.Close()

	configured := configuredWithUserinfo(t, upstream.URL, "/private-path")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport := NewSSETransport(configured, nil, false, true)
	err := transport.Connect(ctx)
	transport.Close()
	assertRefusalText(t, "SSE GET", err, http.StatusFound, other.URL, upstream.URL)

	m := NewUpstreamManager(nil, nil, "", true)
	_, err = m.connect(ctx, "u", UpstreamConfig{Transport: "sse", URL: configured})
	assertRefusalText(t, "manager SSE GET", err, http.StatusFound, other.URL, upstream.URL)
	waitNoHits(t, &otherHits)
}

func TestRefusalError_SSEPost(t *testing.T) {
	var otherHits atomic.Int32
	other := countingServer(&otherHits)
	defer other.Close()
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodPost {
			http.Redirect(w, r, other.URL+"/elsewhere?x=1", http.StatusTemporaryRedirect)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		w.Write([]byte("event: endpoint\ndata: /post-path?sid=zzz\n\n"))
		w.(http.Flusher).Flush()
		<-r.Context().Done()
	}))
	defer server.Close()

	configured := configuredWithUserinfo(t, server.URL, "/private-path")
	transport := NewSSETransport(configured, map[string]string{"X-Key": "v"}, false, true)
	defer transport.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	err := transport.WriteMessage(ctx, redirectBody(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "ping"}))
	assertRefusalText(t, "SSE POST", err, http.StatusTemporaryRedirect, other.URL, server.URL)
	waitNoHits(t, &otherHits)
}

func TestRefusalError_SSEEndpointAndManager(t *testing.T) {
	var otherHits, posts atomic.Int32
	other := countingServer(&otherHits)
	defer other.Close()
	server := sseEventServer(func(string) []string { return []string{other.URL + "/message?sid=zzz"} }, true, &posts)
	defer server.Close()

	configured := configuredWithUserinfo(t, server.URL, "/private-path")
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport := NewSSETransport(configured, nil, false, true)
	err := transport.Connect(ctx)
	transport.Close()
	assertRefusalText(t, "SSE endpoint", err, 0, other.URL, server.URL)

	m := NewUpstreamManager(nil, nil, "", true)
	_, err = m.connect(ctx, "u", UpstreamConfig{Transport: "sse", URL: configured})
	assertRefusalText(t, "manager SSE endpoint", err, 0, other.URL, server.URL)

	_, err = m.connect(ctx, "u", UpstreamConfig{Transport: "streamable", URL: configured})
	if err != nil {
		t.Fatalf("streamable connect does not probe: %v", err)
	}
	waitNoHits(t, &otherHits)
}
