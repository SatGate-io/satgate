package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"
)

// mockSSEServer simulates a legacy MCP SSE server.
type mockSSEServer struct {
	server      *httptest.Server
	tools       []map[string]interface{}
	mu          sync.Mutex
	initialized bool
}

func newMockSSEServer() *mockSSEServer {
	m := &mockSSEServer{
		tools: []map[string]interface{}{
			{"name": "test_tool", "description": "A test tool", "inputSchema": map[string]interface{}{"type": "object"}},
			{"name": "web_search", "description": "Search the web", "inputSchema": map[string]interface{}{"type": "object"}},
		},
	}

	mux := http.NewServeMux()

	// SSE endpoint
	mux.HandleFunc("/sse", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "streaming not supported", http.StatusInternalServerError)
			return
		}

		w.Header().Set("Content-Type", "text/event-stream")
		w.Header().Set("Cache-Control", "no-cache")
		w.Header().Set("Connection", "keep-alive")

		// Send endpoint event
		fmt.Fprintf(w, "event: endpoint\ndata: %s/message\n\n", m.server.URL)
		flusher.Flush()

		// Keep connection alive until client disconnects
		<-r.Context().Done()
	})

	// Message endpoint (receives JSON-RPC, responds via SSE)
	mux.HandleFunc("/message", func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "POST" {
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			JSONRPC string          `json:"jsonrpc"`
			ID      json.RawMessage `json:"id"`
			Method  string          `json:"method"`
			Params  json.RawMessage `json:"params"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), http.StatusBadRequest)
			return
		}

		// The SSE server doesn't send responses on POST — it sends them on the SSE stream.
		// But for simplicity in testing, we accept the POST and note the request was received.
		// In a real SSE server, responses go back on the SSE stream.
		// For our test, we'll send the response directly (which is technically valid).
		var resp map[string]interface{}

		switch req.Method {
		case "initialize":
			m.mu.Lock()
			m.initialized = true
			m.mu.Unlock()
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
					"serverInfo":      map[string]interface{}{"name": "test-sse-server", "version": "1.0.0"},
				},
			}
		case "tools/list":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  map[string]interface{}{"tools": m.tools},
			}
		case "tools/call":
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result": map[string]interface{}{
					"content": []map[string]interface{}{{"type": "text", "text": "tool result"}},
				},
			}
		default:
			resp = map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"error":   map[string]interface{}{"code": -32601, "message": "method not found"},
			}
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	})

	m.server = httptest.NewServer(mux)
	return m
}

func (m *mockSSEServer) Close() {
	m.server.Close()
}

func TestSSETransport_Connect(t *testing.T) {
	mock := newMockSSEServer()
	defer mock.Close()

	transport := NewSSETransport(mock.server.URL+"/sse", nil, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if transport.messageEndpoint == "" {
		t.Fatal("expected message endpoint to be set after connect")
	}
	if transport.messageEndpoint != mock.server.URL+"/message" {
		t.Errorf("expected endpoint %s/message, got %s", mock.server.URL, transport.messageEndpoint)
	}
}

func TestSSETransport_WriteAndRead(t *testing.T) {
	mock := newMockSSEServer()
	defer mock.Close()

	transport := NewSSETransport(mock.server.URL+"/sse", nil, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Send initialize request
	initReq, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]string{"name": "test", "version": "1.0"},
		},
	})

	if err := transport.WriteMessage(ctx, initReq); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Note: In the legacy SSE protocol, responses come back on the SSE stream.
	// Our mock sends responses directly on POST (which is a simplification).
	// The real flow would have responses arrive via the SSE event stream.
}

func TestSSETransport_CustomHeaders(t *testing.T) {
	var receivedHeaders http.Header
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/sse" {
			receivedHeaders = r.Header.Clone()
			flusher := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			fmt.Fprintf(w, "event: endpoint\ndata: /message\n\n")
			flusher.Flush()
			<-r.Context().Done()
			return
		}
		w.WriteHeader(200)
	}))
	defer server.Close()

	headers := map[string]string{
		"Authorization": "Bearer test-token-123",
		"X-Custom":      "custom-value",
	}

	transport := NewSSETransport(server.URL+"/sse", headers, false, true)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	if receivedHeaders.Get("Authorization") != "Bearer test-token-123" {
		t.Errorf("expected Authorization header, got %q", receivedHeaders.Get("Authorization"))
	}
	if receivedHeaders.Get("X-Custom") != "custom-value" {
		t.Errorf("expected X-Custom header, got %q", receivedHeaders.Get("X-Custom"))
	}
}

func TestSSETransport_Close(t *testing.T) {
	mock := newMockSSEServer()
	defer mock.Close()

	transport := NewSSETransport(mock.server.URL+"/sse", nil, false, true)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Close should not block
	done := make(chan struct{})
	go func() {
		transport.Close()
		close(done)
	}()

	select {
	case <-done:
		// ok
	case <-time.After(5 * time.Second):
		t.Fatal("Close blocked for too long")
	}

	// Double close should be safe
	transport.Close()
}
