package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"sync/atomic"
	"testing"
	"time"
)

func TestStreamableRedirect_CrossOriginRefused(t *testing.T) {
	var otherHits atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otherHits.Add(1)
		http.Error(w, "should not be called", http.StatusBadGateway)
	}))
	defer other.Close()

	var sawAuth, sawCustom bool
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		sawAuth = r.Header.Get("Authorization") == "Bearer super-secret-token"
		sawCustom = r.Header.Get("X-Custom-Token") == "custom-secret-value"
		http.Redirect(w, r, other.URL+"/sink?token=leak", http.StatusTemporaryRedirect)
	}))
	defer upstream.Close()

	transport := NewStreamableHTTPTransport(upstream.URL+"/mcp", map[string]string{
		"Authorization":  "Bearer super-secret-token",
		"X-Custom-Token": "custom-secret-value",
	}, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	err := transport.WriteMessage(ctx, redirectBody(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	}))
	assertRedirectRefused(t, err, http.StatusTemporaryRedirect, other.URL)
	if strings.Contains(err.Error(), "/sink") || strings.Contains(err.Error(), "token=leak") {
		t.Fatalf("error leaked target path or query: %v", err)
	}
	if strings.Contains(err.Error(), "super-secret-token") || strings.Contains(err.Error(), "custom-secret-value") {
		t.Fatalf("error leaked stored header: %v", err)
	}
	if !sawAuth || !sawCustom {
		t.Fatalf("configured origin did not receive stored headers auth=%v custom=%v", sawAuth, sawCustom)
	}
	waitNoHits(t, &otherHits)
}

func TestStreamableRedirect_SameOriginWithStoredHeadersRefused(t *testing.T) {
	var nextHits atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/mcp/next?x=1", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/mcp/next", func(w http.ResponseWriter, r *http.Request) {
		nextHits.Add(1)
		writeJSONResult(w)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	transport := NewStreamableHTTPTransport(server.URL+"/mcp", map[string]string{
		"X-Custom-Token": "custom-secret-value",
	}, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	err := transport.WriteMessage(ctx, redirectBody(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	}))
	assertRedirectRefused(t, err, http.StatusTemporaryRedirect, server.URL)
	if strings.Contains(err.Error(), "/mcp/next") || strings.Contains(err.Error(), "x=1") {
		t.Fatalf("error leaked target path or query: %v", err)
	}
	waitNoHits(t, &nextHits)
}

func TestStreamableRedirect_SameOriginWithoutStoredHeadersFollowed(t *testing.T) {
	var nextHits atomic.Int32
	var sawMethod string
	mux := http.NewServeMux()
	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, "/mcp/next", http.StatusTemporaryRedirect)
	})
	mux.HandleFunc("/mcp/next", func(w http.ResponseWriter, r *http.Request) {
		nextHits.Add(1)
		var body struct {
			Method string `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&body)
		sawMethod = body.Method
		writeJSONResult(w)
	})
	server := httptest.NewServer(mux)
	defer server.Close()

	transport := NewStreamableHTTPTransport(server.URL+"/mcp", nil, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := transport.WriteMessage(ctx, redirectBody(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	})); err != nil {
		t.Fatalf("same-origin redirect with no stored headers should be followed: %v", err)
	}
	if nextHits.Load() != 1 {
		t.Fatalf("followed target hits=%d, want 1", nextHits.Load())
	}
	if sawMethod != "ping" {
		t.Fatalf("followed target method=%q", sawMethod)
	}
}

func TestStreamableRedirect_OtherPortIsDifferentOrigin(t *testing.T) {
	var otherHits atomic.Int32
	var otherHost string
	var otherPort string
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otherHits.Add(1)
		http.Error(w, "should not be called", http.StatusBadGateway)
	}))
	defer other.Close()
	otherURL, err := url.Parse(other.URL)
	if err != nil {
		t.Fatal(err)
	}
	otherHost = otherURL.Hostname()
	otherPort = otherURL.Port()

	var upstreamHost string
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		upstreamHost = r.Host
		target := "http://" + otherHost + ":" + otherPort + "/elsewhere?q=secret"
		http.Redirect(w, r, target, http.StatusTemporaryRedirect)
	}))
	defer upstream.Close()
	upURL, err := url.Parse(upstream.URL)
	if err != nil {
		t.Fatal(err)
	}
	if upURL.Hostname() != otherHost {
		t.Fatalf("fixture hosts differ: %s vs %s", upURL.Hostname(), otherHost)
	}
	if upURL.Port() == otherPort {
		t.Fatalf("fixture ports must differ")
	}

	transport := NewStreamableHTTPTransport(upstream.URL+"/mcp", map[string]string{
		"Authorization": "Bearer port-secret-token",
	}, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	err = transport.WriteMessage(ctx, redirectBody(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	}))
	assertRedirectRefused(t, err, http.StatusTemporaryRedirect, other.URL)
	if strings.Contains(err.Error(), "/elsewhere") || strings.Contains(err.Error(), "q=secret") || strings.Contains(err.Error(), "port-secret-token") {
		t.Fatalf("error leaked target or token: %v", err)
	}
	if upstreamHost == "" {
		t.Fatal("configured origin received no request")
	}
	waitNoHits(t, &otherHits)
}

func TestStreamableRedirect_SessionIDRefusesLaterRedirect(t *testing.T) {
	var otherHits atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otherHits.Add(1)
	}))
	defer other.Close()

	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet {
			http.Error(w, "no stream", http.StatusMethodNotAllowed)
			return
		}
		if r.Header.Get("Mcp-Session-Id") == "" {
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Mcp-Session-Id", "sess-1")
			w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
			return
		}
		http.Redirect(w, r, other.URL+"/nope?x=1", http.StatusTemporaryRedirect)
	}))
	defer upstream.Close()

	transport := NewStreamableHTTPTransport(upstream.URL, nil, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := transport.WriteMessage(ctx, redirectBody(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
	})); err != nil {
		t.Fatalf("first post: %v", err)
	}
	err := transport.WriteMessage(ctx, redirectBody(map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "ping",
	}))
	assertRedirectRefused(t, err, http.StatusTemporaryRedirect, other.URL)
	if strings.Contains(err.Error(), "sess-1") || strings.Contains(err.Error(), "/nope") {
		t.Fatalf("error leaked session or path: %v", err)
	}
	waitNoHits(t, &otherHits)
}

func TestStreamableRedirect_ThreeSameOriginRedirectsFollowed(t *testing.T) {
	var landed atomic.Int32
	mux := http.NewServeMux()
	mux.HandleFunc("/ok/3", func(w http.ResponseWriter, r *http.Request) {
		landed.Add(1)
		writeJSONResult(w)
	})
	for i := 0; i < 3; i++ {
		i := i
		mux.HandleFunc(fmt.Sprintf("/ok/%d", i), func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, fmt.Sprintf("/ok/%d", i+1), http.StatusTemporaryRedirect)
		})
	}
	server := httptest.NewServer(mux)
	defer server.Close()

	transport := NewStreamableHTTPTransport(server.URL+"/ok/0", nil, false, true)
	defer transport.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	if err := transport.WriteMessage(ctx, redirectBody(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	})); err != nil {
		t.Fatalf("3 same-origin redirects should be followed: %v", err)
	}
	if landed.Load() != 1 {
		t.Fatalf("final hop hits=%d", landed.Load())
	}
}

func TestStreamableRedirect_FourthSameOriginRedirectRefused(t *testing.T) {
	var fourthHits atomic.Int32
	mux := http.NewServeMux()
	server := httptest.NewServer(mux)
	defer server.Close()
	mux.HandleFunc("/r/4", func(w http.ResponseWriter, r *http.Request) {
		fourthHits.Add(1)
		writeJSONResult(w)
	})
	for i := 0; i < 4; i++ {
		i := i
		mux.HandleFunc(fmt.Sprintf("/r/%d", i), func(w http.ResponseWriter, r *http.Request) {
			http.Redirect(w, r, fmt.Sprintf("/r/%d", i+1), http.StatusTemporaryRedirect)
		})
	}

	transport := NewStreamableHTTPTransport(server.URL+"/r/0", nil, false, true)
	defer transport.Close()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatal(err)
	}
	err := transport.WriteMessage(ctx, redirectBody(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	}))
	assertRedirectRefused(t, err, http.StatusTemporaryRedirect, server.URL)
	waitNoHits(t, &fourthHits)
}

func TestSSEEndpoint_OtherOriginRefused(t *testing.T) {
	var otherHits atomic.Int32
	other := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		otherHits.Add(1)
		http.Error(w, "should not be called", http.StatusBadGateway)
	}))
	defer other.Close()

	sse := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "no flush", http.StatusInternalServerError)
			return
		}
		w.Header().Set("Content-Type", "text/event-stream")
		fmt.Fprintf(w, "event: endpoint\ndata: %s/message?session=secret\n\n", other.URL)
		flusher.Flush()
		<-r.Context().Done()
	}))
	defer sse.Close()

	transport := NewSSETransport(sse.URL+"/sse", map[string]string{
		"Authorization":  "Bearer sse-secret-token",
		"X-Custom-Token": "sse-custom-value",
	}, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := transport.Connect(ctx)
	if err == nil {
		t.Fatal("expected connect to fail")
	}
	origin := originOf(other.URL)
	want := "SSE endpoint origin " + origin
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("missing %q in %v", want, err)
	}
	if strings.Contains(err.Error(), "/message") || strings.Contains(err.Error(), "session=secret") {
		t.Fatalf("error leaked path or query: %v", err)
	}
	if strings.Contains(err.Error(), "sse-secret-token") || strings.Contains(err.Error(), "sse-custom-value") {
		t.Fatalf("error leaked stored header: %v", err)
	}
	if transport.messageEndpoint != "" {
		t.Fatalf("message endpoint set to %q", transport.messageEndpoint)
	}
	waitNoHits(t, &otherHits)
}

func TestSSEEndpoint_RelativeAndAbsoluteSameOrigin(t *testing.T) {
	t.Run("relative", func(t *testing.T) {
		assertSSEEndpointWorks(t, func(serverURL string) string {
			return "/message?session=ok"
		})
	})
	t.Run("relative-no-slash", func(t *testing.T) {
		assertSSEEndpointWorks(t, func(serverURL string) string {
			return "message?session=ok"
		})
	})
	t.Run("absolute", func(t *testing.T) {
		assertSSEEndpointWorks(t, func(serverURL string) string {
			return serverURL + "/message?session=ok"
		})
	})
}

func TestUpstreamOriginDiffersForSubdomainPortAndScheme(t *testing.T) {
	base := mustURL(t, "https://agent.example.com/mcp/trading")
	if sameOrigin(base, mustURL(t, "https://evil.agent.example.com/mcp/trading")) {
		t.Fatal("subdomain must be a different origin")
	}
	if sameOrigin(base, mustURL(t, "https://agent.example.com:8443/mcp/trading")) {
		t.Fatal("other port must be a different origin")
	}
	if sameOrigin(base, mustURL(t, "http://agent.example.com/mcp/trading")) {
		t.Fatal("other scheme must be a different origin")
	}
	if !sameOrigin(base, mustURL(t, "https://Agent.Example.com:443/other")) {
		t.Fatal("host case, default port, and path must not change origin")
	}
	shown := displayOrigin(mustURL(t, "https://evil.example/sink?token=leak"))
	if strings.Contains(shown, "/sink") || strings.Contains(shown, "token") || strings.Contains(shown, "?") {
		t.Fatalf("display origin leaked path or query: %s", shown)
	}
}

func assertSSEEndpointWorks(t *testing.T, endpoint func(serverURL string) string) {
	t.Helper()
	var postHits atomic.Int32
	var server *httptest.Server
	server = httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sse" {
			flusher := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", endpoint(server.URL))
			flusher.Flush()
			<-r.Context().Done()
			return
		}
		if r.URL.Path == "/message" && r.URL.Query().Get("session") == "ok" && r.Method == http.MethodPost {
			postHits.Add(1)
			io.Copy(io.Discard, r.Body)
			w.WriteHeader(http.StatusAccepted)
			return
		}
		http.NotFound(w, r)
	}))
	defer server.Close()

	transport := NewSSETransport(server.URL+"/sse", map[string]string{
		"Authorization": "Bearer same-origin-token",
	}, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect: %v", err)
	}
	wantEndpoint := server.URL + "/message?session=ok"
	if transport.messageEndpoint != wantEndpoint {
		t.Fatalf("endpoint=%q want %q", transport.messageEndpoint, wantEndpoint)
	}
	if err := transport.WriteMessage(ctx, redirectBody(map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	})); err != nil {
		t.Fatalf("WriteMessage: %v", err)
	}
	if postHits.Load() != 1 {
		t.Fatalf("message endpoint hits=%d", postHits.Load())
	}
}

func assertRedirectRefused(t *testing.T, err error, status int, targetRaw string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected redirect error")
	}
	want := fmt.Sprintf("upstream redirect status %d to %s refused", status, originOf(targetRaw))
	if !strings.Contains(err.Error(), want) {
		t.Fatalf("missing %q in %v", want, err)
	}
}

func originOf(raw string) string {
	u, err := url.Parse(raw)
	if err != nil {
		return raw
	}
	return displayOrigin(u)
}

func mustURL(t *testing.T, raw string) *url.URL {
	t.Helper()
	u, err := url.Parse(raw)
	if err != nil {
		t.Fatal(err)
	}
	return u
}

func redirectBody(v any) json.RawMessage {
	b, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return b
}

func writeJSONResult(w http.ResponseWriter) {
	w.Header().Set("Content-Type", "application/json")
	w.Write([]byte(`{"jsonrpc":"2.0","id":1,"result":{}}`))
}

func waitNoHits(t *testing.T, hits *atomic.Int32) {
	t.Helper()
	time.Sleep(150 * time.Millisecond)
	if got := hits.Load(); got != 0 {
		t.Fatalf("other origin received %d requests", got)
	}
}
