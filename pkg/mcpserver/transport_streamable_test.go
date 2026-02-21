package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"
)

// mockStreamableServer simulates an MCP Streamable HTTP server.
type mockStreamableServer struct {
	server    *httptest.Server
	sessionID string
	tools     []map[string]interface{}
	mu        sync.Mutex
	requests  []string // track received methods
}

func newMockStreamableServer() *mockStreamableServer {
	m := &mockStreamableServer{
		sessionID: "test-session-abc123",
		tools: []map[string]interface{}{
			{"name": "database_query", "description": "Query the database", "inputSchema": map[string]interface{}{"type": "object"}},
			{"name": "web_search", "description": "Search the web", "inputSchema": map[string]interface{}{"type": "object"}},
			{"name": "code_execute", "description": "Execute code", "inputSchema": map[string]interface{}{"type": "object"}},
		},
	}

	mux := http.NewServeMux()

	mux.HandleFunc("/mcp", func(w http.ResponseWriter, r *http.Request) {
		switch r.Method {
		case "GET":
			// Notification stream — send keep-alive then hold
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Mcp-Session-Id", m.sessionID)
			flusher := w.(http.Flusher)
			fmt.Fprintf(w, ": keep-alive\n\n")
			flusher.Flush()
			<-r.Context().Done()

		case "POST":
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

			m.mu.Lock()
			m.requests = append(m.requests, req.Method)
			m.mu.Unlock()

			// Notifications (no ID) get 204
			if req.ID == nil || string(req.ID) == "null" {
				w.WriteHeader(http.StatusNoContent)
				return
			}

			var result interface{}
			switch req.Method {
			case "initialize":
				result = map[string]interface{}{
					"protocolVersion": "2024-11-05",
					"capabilities":    map[string]interface{}{"tools": map[string]interface{}{}},
					"serverInfo":      map[string]interface{}{"name": "test-streamable-server", "version": "1.0.0"},
				}
			case "tools/list":
				result = map[string]interface{}{"tools": m.tools}
			case "tools/call":
				var params struct {
					Name string `json:"name"`
				}
				json.Unmarshal(req.Params, &params)
				result = map[string]interface{}{
					"content": []map[string]interface{}{
						{"type": "text", "text": fmt.Sprintf("Result from %s", params.Name)},
					},
				}
			case "ping":
				result = map[string]interface{}{}
			default:
				w.Header().Set("Content-Type", "application/json")
				w.Header().Set("Mcp-Session-Id", m.sessionID)
				json.NewEncoder(w).Encode(map[string]interface{}{
					"jsonrpc": "2.0",
					"id":      req.ID,
					"error":   map[string]interface{}{"code": -32601, "message": "method not found"},
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Mcp-Session-Id", m.sessionID)
			json.NewEncoder(w).Encode(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      req.ID,
				"result":  result,
			})

		case "DELETE":
			// Session termination
			w.WriteHeader(http.StatusNoContent)

		default:
			http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
		}
	})

	m.server = httptest.NewServer(mux)
	return m
}

func (m *mockStreamableServer) Close() {
	m.server.Close()
}

func TestStreamableHTTPTransport_Connect(t *testing.T) {
	mock := newMockStreamableServer()
	defer mock.Close()

	transport := NewStreamableHTTPTransport(mock.server.URL+"/mcp", nil, false)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}
}

func TestStreamableHTTPTransport_WriteAndRead(t *testing.T) {
	mock := newMockStreamableServer()
	defer mock.Close()

	transport := NewStreamableHTTPTransport(mock.server.URL+"/mcp", nil, false)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := transport.Connect(ctx); err != nil {
		t.Fatalf("Connect failed: %v", err)
	}

	// Send initialize
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
		t.Fatalf("WriteMessage (initialize) failed: %v", err)
	}

	// Read response
	msg, err := transport.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	var resp struct {
		Result struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	if err := json.Unmarshal(msg, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	if resp.Result.ServerInfo.Name != "test-streamable-server" {
		t.Errorf("expected server name 'test-streamable-server', got %q", resp.Result.ServerInfo.Name)
	}

	// Session ID should be captured
	transport.sessionMu.RLock()
	sid := transport.sessionID
	transport.sessionMu.RUnlock()
	if sid != "test-session-abc123" {
		t.Errorf("expected session ID 'test-session-abc123', got %q", sid)
	}
}

func TestStreamableHTTPTransport_ToolsList(t *testing.T) {
	mock := newMockStreamableServer()
	defer mock.Close()

	transport := NewStreamableHTTPTransport(mock.server.URL+"/mcp", nil, false)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport.Connect(ctx)

	// tools/list
	req, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
	})

	if err := transport.WriteMessage(ctx, req); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	msg, err := transport.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	var resp struct {
		Result struct {
			Tools []struct {
				Name string `json:"name"`
			} `json:"tools"`
		} `json:"result"`
	}
	if err := json.Unmarshal(msg, &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if len(resp.Result.Tools) != 3 {
		t.Errorf("expected 3 tools, got %d", len(resp.Result.Tools))
	}
}

func TestStreamableHTTPTransport_ToolCall(t *testing.T) {
	mock := newMockStreamableServer()
	defer mock.Close()

	transport := NewStreamableHTTPTransport(mock.server.URL+"/mcp", nil, false)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport.Connect(ctx)

	req, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      3,
		"method":  "tools/call",
		"params": map[string]interface{}{
			"name":      "database_query",
			"arguments": map[string]string{"sql": "SELECT 1"},
		},
	})

	if err := transport.WriteMessage(ctx, req); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	msg, err := transport.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	var resp struct {
		Result struct {
			Content []struct {
				Text string `json:"text"`
			} `json:"content"`
		} `json:"result"`
	}
	json.Unmarshal(msg, &resp)
	if len(resp.Result.Content) == 0 || !strings.Contains(resp.Result.Content[0].Text, "database_query") {
		t.Errorf("unexpected tool call response: %s", string(msg))
	}
}

func TestStreamableHTTPTransport_Notification(t *testing.T) {
	mock := newMockStreamableServer()
	defer mock.Close()

	transport := NewStreamableHTTPTransport(mock.server.URL+"/mcp", nil, false)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport.Connect(ctx)

	// Send notification (no ID) — should get 204, no response
	notif, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"method":  "notifications/initialized",
	})

	if err := transport.WriteMessage(ctx, notif); err != nil {
		t.Fatalf("WriteMessage (notification) failed: %v", err)
	}

	// Verify server received it
	time.Sleep(50 * time.Millisecond)
	mock.mu.Lock()
	found := false
	for _, m := range mock.requests {
		if m == "notifications/initialized" {
			found = true
		}
	}
	mock.mu.Unlock()
	if !found {
		t.Error("server did not receive notifications/initialized")
	}
}

func TestStreamableHTTPTransport_CustomHeaders(t *testing.T) {
	var receivedAuth string
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAuth = r.Header.Get("Authorization")
		if r.Method == "GET" {
			w.Header().Set("Content-Type", "text/event-stream")
			w.(http.Flusher).Flush()
			<-r.Context().Done()
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0", "id": 1, "result": map[string]interface{}{},
		})
	}))
	defer server.Close()

	headers := map[string]string{
		"Authorization": "Bearer secret-token",
	}
	transport := NewStreamableHTTPTransport(server.URL, headers, false)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport.Connect(ctx)

	req, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	})
	transport.WriteMessage(ctx, req)

	time.Sleep(50 * time.Millisecond)
	if receivedAuth != "Bearer secret-token" {
		t.Errorf("expected auth header 'Bearer secret-token', got %q", receivedAuth)
	}
}

func TestStreamableHTTPTransport_SSEResponse(t *testing.T) {
	// Test server that returns SSE stream for tool calls
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			// Return 405 — this server doesn't support notification stream
			http.Error(w, "not supported", http.StatusMethodNotAllowed)
			return
		}

		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
		}
		json.NewDecoder(r.Body).Decode(&req)

		if req.Method == "tools/call" {
			// Return SSE stream
			flusher := w.(http.Flusher)
			w.Header().Set("Content-Type", "text/event-stream")
			w.Header().Set("Mcp-Session-Id", "sse-resp-session")

			// Send progress notification
			fmt.Fprintf(w, "event: message\ndata: %s\n\n",
				`{"jsonrpc":"2.0","method":"notifications/progress","params":{"progressToken":"p1","progress":50,"total":100}}`)
			flusher.Flush()

			// Send final result
			fmt.Fprintf(w, "event: message\ndata: %s\n\n",
				fmt.Sprintf(`{"jsonrpc":"2.0","id":%s,"result":{"content":[{"type":"text","text":"streamed result"}]}}`, string(req.ID)))
			flusher.Flush()
			return
		}

		// Default: JSON response
		w.Header().Set("Content-Type", "application/json")
		w.Header().Set("Mcp-Session-Id", "sse-resp-session")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"jsonrpc": "2.0", "id": req.ID, "result": map[string]interface{}{},
		})
	}))
	defer server.Close()

	transport := NewStreamableHTTPTransport(server.URL, nil, false)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport.Connect(ctx)

	// Send tool call
	req, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0",
		"id":      42,
		"method":  "tools/call",
		"params":  map[string]interface{}{"name": "slow_tool"},
	})

	if err := transport.WriteMessage(ctx, req); err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Should receive progress notification first
	msg1, err := transport.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("ReadMessage (progress) failed: %v", err)
	}
	if !strings.Contains(string(msg1), "progress") {
		t.Errorf("expected progress notification, got: %s", string(msg1))
	}

	// Then the final result
	msg2, err := transport.ReadMessage(ctx)
	if err != nil {
		t.Fatalf("ReadMessage (result) failed: %v", err)
	}
	if !strings.Contains(string(msg2), "streamed result") {
		t.Errorf("expected 'streamed result', got: %s", string(msg2))
	}
}

func TestStreamableHTTPTransport_ConcurrentCalls(t *testing.T) {
	mock := newMockStreamableServer()
	defer mock.Close()

	transport := NewStreamableHTTPTransport(mock.server.URL+"/mcp", nil, false)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	transport.Connect(ctx)

	// Send 10 concurrent requests
	var wg sync.WaitGroup
	errors := make(chan error, 10)

	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req, _ := json.Marshal(map[string]interface{}{
				"jsonrpc": "2.0",
				"id":      id + 100,
				"method":  "ping",
			})
			if err := transport.WriteMessage(ctx, req); err != nil {
				errors <- fmt.Errorf("write %d: %w", id, err)
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	for err := range errors {
		t.Errorf("concurrent call error: %v", err)
	}

	// Read all responses
	for i := 0; i < 10; i++ {
		readCtx, readCancel := context.WithTimeout(ctx, 2*time.Second)
		_, err := transport.ReadMessage(readCtx)
		readCancel()
		if err != nil {
			t.Errorf("ReadMessage %d failed: %v", i, err)
		}
	}
}

func TestStreamableHTTPTransport_Close(t *testing.T) {
	mock := newMockStreamableServer()
	defer mock.Close()

	transport := NewStreamableHTTPTransport(mock.server.URL+"/mcp", nil, false)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	transport.Connect(ctx)

	done := make(chan struct{})
	go func() {
		transport.Close()
		close(done)
	}()

	select {
	case <-done:
	case <-time.After(5 * time.Second):
		t.Fatal("Close blocked for too long")
	}

	// Double close
	transport.Close()
}

func TestStreamableHTTPTransport_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == "GET" {
			http.Error(w, "not found", http.StatusNotFound)
			return
		}
		http.Error(w, `{"error": "internal error"}`, http.StatusInternalServerError)
	}))
	defer server.Close()

	transport := NewStreamableHTTPTransport(server.URL, nil, false)
	defer transport.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Connect should succeed (GET failure is not fatal)
	transport.Connect(ctx)

	// POST should return error
	req, _ := json.Marshal(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "ping",
	})
	err := transport.WriteMessage(ctx, req)
	if err == nil {
		t.Error("expected error for 500 response")
	}
	if !strings.Contains(err.Error(), "500") {
		t.Errorf("expected 500 in error, got: %v", err)
	}
}
