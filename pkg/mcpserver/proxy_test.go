package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// pipeTransport connects two ends of a pipe as a Transport.
type pipeTransport struct {
	r *io.PipeReader
	w *io.PipeWriter
	*StdioTransport
}

func newPipePair() (client Transport, server Transport) {
	cr, sw := io.Pipe() // server writes → client reads
	sr, cw := io.Pipe() // client writes → server reads

	clientT := NewStdioTransport(cr, cw, nil)
	serverT := NewStdioTransport(sr, sw, nil)

	return clientT, serverT
}

// TestProxy_FullDelegationFlow tests the complete delegation scenario:
// 1. Parent connects with budget
// 2. Parent delegates to child
// 3. Child makes calls until budget exhaustion
// 4. Parent still has budget
func TestProxy_FullDelegationFlow(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	// Setup macaroon service
	svc, _ := macaroon.NewService("test-root-key")

	// Mint root token
	rootMac, _ := svc.Mint("api:*", time.Now().Add(time.Hour))
	rootToken := svc.Encode(rootMac)
	rootTokenID := hashToken(rootMac.Identifier + rootMac.Signature)

	// Create config
	cfg := &Config{
		Server: ServerConfig{Transport: "stdio", Name: "test", Version: "1.0"},
		Auth:   AuthConfig{Mode: "header", RootKey: "test-root-key"},
		Upstreams: map[string]UpstreamConfig{
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
    elif m == "tools/list":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"tools":[{"name":"search","description":"s","inputSchema":{"type":"object"}},{"name":"generate","description":"g","inputSchema":{"type":"object"}}]}}), flush=True)
    elif m == "tools/call":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"content":[{"type":"text","text":"ok"}]}}), flush=True)
    elif m == "ping":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{}}), flush=True)
`}},
		},
		Budget: BudgetConfig{Backend: "memory", Limit: 500, FailMode: "closed"},
		Tools: ToolsConfig{
			DefaultCost: 5,
			Costs:       map[string]int64{"search": 5, "generate": 50},
		},
		Enforcement: EnforcementConfig{Mode: "hard"},
		Logging:     LoggingConfig{Level: "error"},
	}
	cfg.applyDefaults()

	// Create proxy
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}

	// Initialize root token budget
	proxy.budget.Initialize(ctx, rootTokenID, 500)
	proxy.tokenID = rootTokenID

	// Create pipe transport
	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()

	clientTransport := NewStdioTransport(clientRead, clientWrite, nil)
	serverTransport := NewStdioTransport(serverRead, serverWrite, nil)

	// Run proxy in background
	go proxy.Run(ctx, serverTransport)

	send := func(msg interface{}) {
		data, _ := json.Marshal(msg)
		clientTransport.WriteMessage(ctx, data)
	}

	recv := func() map[string]interface{} {
		msg, err := clientTransport.ReadMessage(ctx)
		if err != nil {
			t.Fatalf("recv: %v", err)
		}
		var result map[string]interface{}
		json.Unmarshal(msg, &result)
		return result
	}

	// --- Initialize ---
	send(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]string{"name": "test", "version": "1.0"},
		},
	})
	resp := recv()
	if _, ok := resp["result"]; !ok {
		t.Fatalf("init failed: %v", resp)
	}

	send(map[string]interface{}{"jsonrpc": "2.0", "method": "notifications/initialized"})

	// --- List tools ---
	send(map[string]interface{}{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
	resp = recv()
	result := resp["result"].(map[string]interface{})
	tools := result["tools"].([]interface{})
	if len(tools) != 2 {
		t.Fatalf("expected 2 tools, got %d", len(tools))
	}

	// --- Delegate 200 credits to child ---
	send(map[string]interface{}{
		"jsonrpc": "2.0", "id": 3, "method": "satgate/delegate",
		"params": map[string]interface{}{
			"budget": 200, "label": "research-agent",
			"_meta": map[string]string{"token": rootToken},
		},
	})
	resp = recv()
	delegateResult := resp["result"].(map[string]interface{})
	childToken := delegateResult["token"].(string)
	childBudget := int64(delegateResult["budget"].(float64))
	parentRemaining := int64(delegateResult["parentRemaining"].(float64))

	if childBudget != 200 {
		t.Errorf("expected child budget 200, got %d", childBudget)
	}
	if parentRemaining != 300 {
		t.Errorf("expected parent remaining 300, got %d", parentRemaining)
	}

	// --- Child makes search calls (5 credits each, 200/5 = 40 max) ---
	childCalls := 0
	for i := 0; i < 45; i++ {
		send(map[string]interface{}{
			"jsonrpc": "2.0", "id": 100 + i, "method": "tools/call",
			"params": map[string]interface{}{
				"name": "search", "arguments": map[string]string{"q": "test"},
				"_meta": map[string]string{"token": childToken},
			},
		})
		resp = recv()
		if _, ok := resp["result"]; ok {
			childCalls++
		} else {
			// Should be budget_exhausted
			errObj := resp["error"].(map[string]interface{})
			data := errObj["data"].(map[string]interface{})
			if data["error"] != "budget_exhausted" {
				t.Errorf("expected budget_exhausted, got %v", data["error"])
			}
			break
		}
	}

	if childCalls != 40 {
		t.Errorf("expected 40 calls before exhaustion, got %d", childCalls)
	}

	// --- Parent checks own budget (should still have 300) ---
	send(map[string]interface{}{
		"jsonrpc": "2.0", "id": 999, "method": "satgate/budget",
		"params": map[string]interface{}{
			"_meta": map[string]string{"token": rootToken},
		},
	})
	resp = recv()
	budgetResult := resp["result"].(map[string]interface{})
	remaining := int64(budgetResult["remaining"].(float64))
	if remaining != 300 {
		t.Errorf("expected parent remaining 300, got %d", remaining)
	}

	// --- Parent can still make calls ---
	send(map[string]interface{}{
		"jsonrpc": "2.0", "id": 1000, "method": "tools/call",
		"params": map[string]interface{}{
			"name": "search", "arguments": map[string]string{"q": "parent"},
			"_meta": map[string]string{"token": rootToken},
		},
	})
	resp = recv()
	if _, ok := resp["result"]; !ok {
		t.Errorf("parent call should succeed, got: %v", resp)
	}
}
