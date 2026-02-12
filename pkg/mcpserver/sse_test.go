package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"
)

func TestSSEServer_HealthCheck(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	srv, addr := startTestSSEServer(t, ctx)
	_ = srv

	// Health check
	resp, err := http.Get(fmt.Sprintf("http://%s/health", addr))
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()

	var health map[string]interface{}
	json.NewDecoder(resp.Body).Decode(&health)

	if health["status"] != "ok" {
		t.Errorf("expected ok, got %v", health["status"])
	}
}

func TestSSEServer_ConnectAndMessage(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, addr := startTestSSEServer(t, ctx)
	baseURL := fmt.Sprintf("http://%s", addr)

	// Connect SSE
	sseResp, err := http.Get(baseURL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	defer sseResp.Body.Close()

	// Read endpoint event
	sessionID := readEndpointEvent(t, sseResp.Body)
	if sessionID == "" {
		t.Fatal("no session ID from endpoint event")
	}

	// Send initialize
	sendMessage(t, baseURL, sessionID, "", map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]string{"name": "test", "version": "1.0"},
		},
	})

	// Read response
	resp := readMessageEvent(t, sseResp.Body)
	if resp == nil {
		t.Fatal("no initialize response")
	}

	var result struct {
		Result struct {
			ServerInfo struct {
				Name string `json:"name"`
			} `json:"serverInfo"`
		} `json:"result"`
	}
	json.Unmarshal(resp, &result)
	if result.Result.ServerInfo.Name == "" {
		t.Error("expected server name in initialize response")
	}

	// Ping (no upstream needed)
	sendMessage(t, baseURL, sessionID, "", map[string]interface{}{
		"jsonrpc": "2.0", "id": 2, "method": "ping",
	})

	pingResp := readMessageEvent(t, sseResp.Body)
	if pingResp == nil {
		t.Fatal("no ping response")
	}
}

func TestSSEServer_BudgetEnforcement(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, addr := startTestSSEServer(t, ctx)
	baseURL := fmt.Sprintf("http://%s", addr)

	// Connect
	sseResp, err := http.Get(baseURL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	defer sseResp.Body.Close()

	sessionID := readEndpointEvent(t, sseResp.Body)

	// Initialize
	sendMessage(t, baseURL, sessionID, "", map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]string{"name": "test", "version": "1.0"},
		},
	})
	readMessageEvent(t, sseResp.Body) // consume init response

	// Check budget
	sendMessage(t, baseURL, sessionID, "", map[string]interface{}{
		"jsonrpc": "2.0", "id": 2, "method": "satgate/budget", "params": map[string]interface{}{},
	})
	budgetResp := readMessageEvent(t, sseResp.Body)

	var budgetResult struct {
		Result struct {
			Remaining int64 `json:"remaining"`
		} `json:"result"`
	}
	json.Unmarshal(budgetResp, &budgetResult)
	if budgetResult.Result.Remaining != 100 {
		t.Errorf("expected 100 budget, got %d", budgetResult.Result.Remaining)
	}
}

// --- helpers ---

func startTestSSEServer(t *testing.T, ctx context.Context) (*SSEServer, string) {
	t.Helper()

	cfg := &Config{
		Server: ServerConfig{Transport: "sse", Port: 0, Name: "test-proxy", Version: "test"},
		Auth:   AuthConfig{Mode: "none"},
		Upstreams: map[string]UpstreamConfig{
			// We use a simple echo-like mock — just enough for init/ping
			"mock": {Transport: "stdio", Command: []string{"python3", "-c", `
import json, sys
for line in sys.stdin:
    req = json.loads(line.strip())
    m = req.get("method","")
    rid = req.get("id")
    if m == "initialize":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{"listChanged":True}},"serverInfo":{"name":"mock","version":"1.0"}}}), flush=True)
    elif m == "notifications/initialized":
        pass
    elif m == "ping":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{}}), flush=True)
    elif m == "tools/list":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"tools":[{"name":"test_tool","description":"test","inputSchema":{"type":"object"}}]}}), flush=True)
    elif m == "tools/call":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"content":[{"type":"text","text":"ok"}]}}), flush=True)
`}},
		},
		Budget:      BudgetConfig{Backend: "memory", Limit: 100, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 5},
		Enforcement: EnforcementConfig{Mode: "hard"},
		Logging:     LoggingConfig{Level: "error"},
	}
	cfg.applyDefaults()

	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Use port 0 for random available port
	srv := NewSSEServer(proxy, "127.0.0.1:0")

	// Use a fixed test port
	addr := "127.0.0.1:19100"
	srv = NewSSEServer(proxy, addr)

	go func() {
		if err := srv.ListenAndServe(ctx); err != nil && ctx.Err() == nil {
			t.Logf("SSE server error: %v", err)
		}
	}()

	// Wait for server to be ready
	deadline := time.Now().Add(5 * time.Second)
	for time.Now().Before(deadline) {
		resp, err := http.Get("http://" + addr + "/health")
		if err == nil {
			resp.Body.Close()
			break
		}
		time.Sleep(50 * time.Millisecond)
	}

	return srv, addr
}

func sendMessage(t *testing.T, baseURL, sessionID, token string, msg interface{}) {
	t.Helper()
	data, _ := json.Marshal(msg)
	req, _ := http.NewRequest("POST", fmt.Sprintf("%s/message?sessionId=%s", baseURL, sessionID), strings.NewReader(string(data)))
	req.Header.Set("Content-Type", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("send message: %v", err)
	}
	resp.Body.Close()
}

func readEndpointEvent(t *testing.T, body io.Reader) string {
	t.Helper()
	buf := make([]byte, 4096)
	n, err := body.Read(buf)
	if err != nil {
		t.Fatalf("read SSE: %v", err)
	}
	data := string(buf[:n])

	// Parse "event: endpoint\ndata: /message?sessionId=xxx\n\n"
	for _, line := range strings.Split(data, "\n") {
		if strings.HasPrefix(line, "data: /message?sessionId=") {
			return strings.TrimPrefix(line, "data: /message?sessionId=")
		}
	}
	t.Fatalf("no endpoint event in: %s", data)
	return ""
}

func readMessageEvent(t *testing.T, body io.Reader) json.RawMessage {
	t.Helper()
	buf := make([]byte, 65536)

	// Set a deadline via a channel
	ch := make(chan []byte, 1)
	go func() {
		n, err := body.Read(buf)
		if err != nil {
			return
		}
		cpy := make([]byte, n)
		copy(cpy, buf[:n])
		ch <- cpy
	}()

	select {
	case data := <-ch:
		// Parse "event: message\ndata: {...}\n\n"
		for _, line := range strings.Split(string(data), "\n") {
			if strings.HasPrefix(line, "data: {") {
				return json.RawMessage(strings.TrimPrefix(line, "data: "))
			}
		}
		t.Logf("no message event in: %s", string(data))
		return nil
	case <-time.After(5 * time.Second):
		t.Fatal("timeout waiting for SSE message")
		return nil
	}
}
