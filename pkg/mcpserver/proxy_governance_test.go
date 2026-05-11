package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// TestProxy_AgentGovernanceProofPath proves the buyer-facing MCP governance path:
// an MCP-capable agent can initialize, list tools, call an allowed tool, receive
// a policy denial before upstream execution, exhaust a scoped budget, and emit
// receipt-ready events without exposing raw bearer tokens.
func TestProxy_AgentGovernanceProofPath(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	svc, err := macaroon.NewService("governance-proof-root-key")
	if err != nil {
		t.Fatal(err)
	}
	wrapper := macaroon.NewMintWrapper(svc)
	tokenBuilder, err := wrapper.Mint("search", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	tokenBuilder.AddCaveat("tenant_id", "ten_mcp_governance_test")
	tokenBuilder.AddCaveat("budget_id", "bud_mcp_governance_test")
	tokenBuilder.AddCaveat("budget_limit", "60")
	tokenBuilder.AddCaveat("delegation_depth", "2")
	token := wrapper.Encode(tokenBuilder)

	cfg := &Config{
		Server: ServerConfig{Transport: "stdio", Name: "satgate-mcp-proof", Version: "1.0"},
		Auth:   AuthConfig{Mode: "header", RootKey: "governance-proof-root-key"},
		Upstreams: map[string]UpstreamConfig{
			"mock": {Transport: "stdio", Command: []string{"python3", "-c", `
import json, sys
for line in sys.stdin:
    req = json.loads(line.strip())
    method = req.get("method", "")
    rid = req.get("id")
    if method == "initialize":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{"listChanged":True}},"serverInfo":{"name":"mock","version":"1.0"}}}), flush=True)
    elif method == "notifications/initialized":
        pass
    elif method == "tools/list":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"tools":[{"name":"search","description":"allowed search","inputSchema":{"type":"object"}},{"name":"generate","description":"blocked by scope","inputSchema":{"type":"object"}}]}}), flush=True)
    elif method == "tools/call":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"content":[{"type":"text","text":"ok"}]}}), flush=True)
`}},
		},
		Budget:      BudgetConfig{Backend: "memory", Limit: 60, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 5, Costs: map[string]int64{"search": 5, "generate": 50}},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	cfg.applyDefaults()

	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	events := NewChannelPublisher(64)
	proxy.SetEventPublisher(events)

	clientRead, serverWrite := io.Pipe()
	serverRead, clientWrite := io.Pipe()
	clientTransport := NewStdioTransport(clientRead, clientWrite, nil)
	serverTransport := NewStdioTransport(serverRead, serverWrite, nil)
	go proxy.Run(ctx, serverTransport)

	send := func(msg interface{}) {
		data, _ := json.Marshal(msg)
		if err := clientTransport.WriteMessage(ctx, data); err != nil {
			t.Fatalf("send: %v", err)
		}
	}
	recv := func() map[string]interface{} {
		msg, err := clientTransport.ReadMessage(ctx)
		if err != nil {
			t.Fatalf("recv: %v", err)
		}
		var result map[string]interface{}
		if err := json.Unmarshal(msg, &result); err != nil {
			t.Fatalf("decode: %v", err)
		}
		return result
	}

	send(map[string]interface{}{"jsonrpc": "2.0", "id": 1, "method": "initialize", "params": map[string]interface{}{"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{}, "clientInfo": map[string]string{"name": "hermes-agent", "version": "mcp-client"}}})
	if resp := recv(); resp["result"] == nil {
		t.Fatalf("initialize failed: %v", resp)
	}
	send(map[string]interface{}{"jsonrpc": "2.0", "method": "notifications/initialized"})

	send(map[string]interface{}{"jsonrpc": "2.0", "id": 2, "method": "tools/list"})
	if resp := recv(); resp["result"] == nil {
		t.Fatalf("tools/list failed: %v", resp)
	}

	callTool := func(id int, tool string) map[string]interface{} {
		send(map[string]interface{}{"jsonrpc": "2.0", "id": id, "method": "tools/call", "params": map[string]interface{}{"name": tool, "arguments": map[string]string{"q": "test"}, "_meta": map[string]string{"token": token}}})
		return recv()
	}

	if resp := callTool(3, "search"); resp["result"] == nil {
		t.Fatalf("allowed search call failed: %v", resp)
	}

	if resp := callTool(4, "generate"); resp["error"] == nil || !strings.Contains(resp["error"].(map[string]interface{})["message"].(string), "not in scope") {
		t.Fatalf("expected policy scope denial before upstream execution, got: %v", resp)
	}

	var exhausted map[string]interface{}
	for i := 0; i < 20; i++ {
		resp := callTool(100+i, "search")
		if resp["error"] != nil {
			exhausted = resp
			break
		}
	}
	if exhausted == nil {
		t.Fatal("expected budget exhaustion denial")
	}
	data := exhausted["error"].(map[string]interface{})["data"].(map[string]interface{})
	if data["error"] != "budget_exhausted" {
		t.Fatalf("expected budget_exhausted, got: %v", data)
	}

	seenSpend := false
	seenToolCall := false
	seenBudgetExhaust := false
	for {
		select {
		case event := <-events.Events():
			if strings.Contains(event.TokenID, token) {
				t.Fatalf("raw token leaked in event token id")
			}
			switch event.Type {
			case EventBudgetSpend:
				seenSpend = true
			case EventToolCall:
				seenToolCall = true
			case EventBudgetExhaust:
				seenBudgetExhaust = true
			}
		default:
			if !seenSpend || !seenToolCall || !seenBudgetExhaust {
				t.Fatalf("missing receipt-ready events: spend=%v tool_call=%v budget_exhaust=%v", seenSpend, seenToolCall, seenBudgetExhaust)
			}
			return
		}
	}
}
