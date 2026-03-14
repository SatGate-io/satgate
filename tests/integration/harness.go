// Package integration provides shared test infrastructure for SatGate MCP proxy
// end-to-end and integration tests.
package integration

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
	"github.com/satgate-io/satgate/pkg/mcpserver"
)

// ---------- Event collector ----------

// EventCollector captures events for test assertions.
// Implements mcpserver.EventPublisher.
type EventCollector struct {
	mu     sync.Mutex
	events []mcpserver.Event
}

func NewEventCollector() *EventCollector {
	return &EventCollector{}
}

func (c *EventCollector) Publish(event mcpserver.Event) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = append(c.events, event)
}

// All returns a copy of all collected events.
func (c *EventCollector) All() []mcpserver.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	out := make([]mcpserver.Event, len(c.events))
	copy(out, c.events)
	return out
}

// ByType returns events filtered by type.
func (c *EventCollector) ByType(t mcpserver.EventType) []mcpserver.Event {
	c.mu.Lock()
	defer c.mu.Unlock()
	var out []mcpserver.Event
	for _, e := range c.events {
		if e.Type == t {
			out = append(out, e)
		}
	}
	return out
}

// Count returns the total number of collected events.
func (c *EventCollector) Count() int {
	c.mu.Lock()
	defer c.mu.Unlock()
	return len(c.events)
}

// Reset clears all collected events.
func (c *EventCollector) Reset() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.events = nil
}

// ---------- Task response helpers ----------

// TaskResponse builds a tool call result JSON that includes a task ID in _meta.
func TaskResponse(taskID string, text string) json.RawMessage {
	result, _ := json.Marshal(map[string]interface{}{
		"_meta": map[string]string{"taskId": taskID},
		"content": []map[string]string{
			{"type": "text", "text": text},
		},
	})
	return result
}

// TaskResponseWithStatus builds a tool call result with task ID and status.
func TaskResponseWithStatus(taskID, status, text string) json.RawMessage {
	result, _ := json.Marshal(map[string]interface{}{
		"_meta": map[string]interface{}{
			"taskId": taskID,
			"status": status,
		},
		"content": []map[string]string{
			{"type": "text", "text": text},
		},
	})
	return result
}

// ---------- Mock tool definition ----------

// MockTool describes a tool offered by the mock upstream.
type MockTool struct {
	Name        string
	Description string
}

// ---------- Test proxy builder ----------

// TestEnv holds all the components for a proxy integration test.
type TestEnv struct {
	Proxy           *mcpserver.Proxy
	Budget          *mcpserver.InMemoryBudgetEnforcer
	Events          *EventCollector
	MacService      *macaroon.Service
	RootToken       string
	RootTokenID     string
	ClientTransport mcpserver.Transport
	Cancel          context.CancelFunc

	ctx context.Context
}

// TestEnvOption configures the test environment.
type TestEnvOption func(*testEnvConfig)

type testEnvConfig struct {
	budgetLimit     int64
	enforcementMode string
	tools           []MockTool
	toolCosts       map[string]int64
	defaultCost     int64
	// toolResponseJSON maps tool name -> static JSON result body.
	// This is embedded directly into the Python mock upstream.
	toolResponseJSON map[string]json.RawMessage
	authMode         string
}

func WithBudget(limit int64) TestEnvOption {
	return func(c *testEnvConfig) { c.budgetLimit = limit }
}

func WithEnforcement(mode string) TestEnvOption {
	return func(c *testEnvConfig) { c.enforcementMode = mode }
}

func WithTools(tools ...MockTool) TestEnvOption {
	return func(c *testEnvConfig) { c.tools = tools }
}

func WithToolCosts(costs map[string]int64) TestEnvOption {
	return func(c *testEnvConfig) { c.toolCosts = costs }
}

func WithDefaultCost(cost int64) TestEnvOption {
	return func(c *testEnvConfig) { c.defaultCost = cost }
}

// WithToolResponseJSON sets static JSON results for specific tools in the mock upstream.
func WithToolResponseJSON(responses map[string]json.RawMessage) TestEnvOption {
	return func(c *testEnvConfig) { c.toolResponseJSON = responses }
}

func WithAuthMode(mode string) TestEnvOption {
	return func(c *testEnvConfig) { c.authMode = mode }
}

// NewTestEnv creates a fully wired test environment with proxy, mock upstream,
// budget enforcer, event collector, and macaroon auth.
func NewTestEnv(t *testing.T, opts ...TestEnvOption) *TestEnv {
	t.Helper()

	cfg := &testEnvConfig{
		budgetLimit:     1000,
		enforcementMode: "hard",
		tools: []MockTool{
			{Name: "search", Description: "search tool"},
			{Name: "generate", Description: "generate tool"},
		},
		toolCosts:   map[string]int64{"search": 10, "generate": 50},
		defaultCost: 1,
		authMode:    "header",
	}
	for _, opt := range opts {
		opt(cfg)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)

	// Macaroon service
	macService, err := macaroon.NewService("integration-test-root-key")
	if err != nil {
		cancel()
		t.Fatalf("macaroon service: %v", err)
	}

	// Mint root token
	rootMac, err := macService.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		cancel()
		t.Fatalf("mint root: %v", err)
	}
	rootToken := macService.Encode(rootMac)
	rootTokenID := HashToken(rootMac.Identifier + rootMac.Signature)

	// Build Python mock upstream script
	pythonScript := buildPythonMock(cfg.tools, cfg.toolResponseJSON)

	realCfg := &mcpserver.Config{
		Server: mcpserver.ServerConfig{Transport: "stdio", Name: "test-proxy", Version: "0.1.0"},
		Auth:   mcpserver.AuthConfig{Mode: cfg.authMode, RootKey: "integration-test-root-key"},
		Upstreams: map[string]mcpserver.UpstreamConfig{
			"mock": {Transport: "stdio", Command: []string{"python3", "-c", pythonScript}, Timeout: 10 * time.Second},
		},
		DefaultUpstream: "mock",
		Budget:          mcpserver.BudgetConfig{Backend: "memory", Limit: cfg.budgetLimit, FailMode: "closed"},
		Tools:           mcpserver.ToolsConfig{DefaultCost: cfg.defaultCost, Costs: cfg.toolCosts},
		Enforcement:     mcpserver.EnforcementConfig{Mode: cfg.enforcementMode},
		Logging:         mcpserver.LoggingConfig{Level: "error"},
	}

	proxy, err := mcpserver.New(realCfg)
	if err != nil {
		cancel()
		t.Fatalf("create proxy: %v", err)
	}

	// Replace budget and events with our test instances
	budget := mcpserver.NewInMemoryBudgetEnforcer()
	events := NewEventCollector()
	proxy.SetBudgetEnforcer(budget)
	proxy.SetEventPublisher(events)

	// Initialize budget for root token
	if err := budget.Initialize(ctx, rootTokenID, cfg.budgetLimit); err != nil {
		cancel()
		t.Fatalf("init budget: %v", err)
	}

	// Pipe for client <-> proxy
	clientRead, proxyWrite := io.Pipe()
	proxyRead, clientWrite := io.Pipe()
	clientTransport := mcpserver.NewStdioTransport(clientRead, clientWrite, nil)
	proxyTransport := mcpserver.NewStdioTransport(proxyRead, proxyWrite, nil)

	// Start proxy
	go func() {
		_ = proxy.Run(ctx, proxyTransport)
	}()

	env := &TestEnv{
		Proxy:           proxy,
		Budget:          budget,
		Events:          events,
		MacService:      macService,
		RootToken:       rootToken,
		RootTokenID:     rootTokenID,
		ClientTransport: clientTransport,
		Cancel:          cancel,
		ctx:             ctx,
	}

	// Do MCP handshake
	env.Initialize(t)

	return env
}

// Initialize performs the MCP initialize handshake.
func (e *TestEnv) Initialize(t *testing.T) {
	t.Helper()
	e.Send(t, map[string]interface{}{
		"jsonrpc": "2.0", "id": 0, "method": "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05",
			"capabilities":    map[string]interface{}{},
			"clientInfo":      map[string]string{"name": "integration-test", "version": "1.0"},
		},
	})
	resp := e.Recv(t)
	if _, ok := resp["result"]; !ok {
		t.Fatalf("initialize failed: %v", resp)
	}
	e.Send(t, map[string]interface{}{"jsonrpc": "2.0", "method": "notifications/initialized"})
}

// Send marshals and writes a JSON-RPC message.
func (e *TestEnv) Send(t *testing.T, msg map[string]interface{}) {
	t.Helper()
	data, err := json.Marshal(msg)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if err := e.ClientTransport.WriteMessage(e.ctx, data); err != nil {
		t.Fatalf("send: %v", err)
	}
}

// Recv reads and unmarshals a JSON-RPC response.
func (e *TestEnv) Recv(t *testing.T) map[string]interface{} {
	t.Helper()
	msg, err := e.ClientTransport.ReadMessage(e.ctx)
	if err != nil {
		t.Fatalf("recv: %v", err)
	}
	var result map[string]interface{}
	if err := json.Unmarshal(msg, &result); err != nil {
		t.Fatalf("unmarshal recv: %v", err)
	}
	return result
}

// ToolCall sends a tools/call request with the given token and returns the response.
func (e *TestEnv) ToolCall(t *testing.T, id int, toolName string, token string) map[string]interface{} {
	t.Helper()
	params := map[string]interface{}{
		"name":      toolName,
		"arguments": map[string]interface{}{},
	}
	if token != "" {
		params["_meta"] = map[string]string{"token": token}
	}
	e.Send(t, map[string]interface{}{
		"jsonrpc": "2.0", "id": id, "method": "tools/call",
		"params": params,
	})
	return e.Recv(t)
}

// MintToken creates a macaroon with the given scope and expiry.
func (e *TestEnv) MintToken(t *testing.T, scope string, expiry time.Duration) string {
	t.Helper()
	mac, err := e.MacService.Mint(scope, time.Now().Add(expiry))
	if err != nil {
		t.Fatalf("mint: %v", err)
	}
	return e.MacService.Encode(mac)
}

// MintExpiredToken creates a macaroon that has already expired.
func (e *TestEnv) MintExpiredToken(t *testing.T) string {
	t.Helper()
	mac, err := e.MacService.Mint("api:*", time.Now().Add(-time.Hour))
	if err != nil {
		t.Fatalf("mint expired: %v", err)
	}
	return e.MacService.Encode(mac)
}

// Close cleans up the test environment.
func (e *TestEnv) Close() {
	e.Cancel()
}

// ---------- Helpers ----------

// HashToken matches mcpserver.hashToken — SHA-256 first 12 hex chars.
func HashToken(token string) string {
	h := sha256.New()
	h.Write([]byte(token))
	return hex.EncodeToString(h.Sum(nil))[:12]
}

// buildPythonMock generates a Python script that acts as an MCP upstream.
func buildPythonMock(tools []MockTool, toolResponseJSON map[string]json.RawMessage) string {
	// Build tool definitions JSON
	toolDefs := "["
	for i, tool := range tools {
		if i > 0 {
			toolDefs += ","
		}
		toolDefs += fmt.Sprintf(`{"name":"%s","description":"%s","inputSchema":{"type":"object"}}`, tool.Name, tool.Description)
	}
	toolDefs += "]"

	// Build custom response map
	customResponses := "{"
	if toolResponseJSON != nil {
		first := true
		for name, result := range toolResponseJSON {
			if !first {
				customResponses += ","
			}
			first = false
			customResponses += fmt.Sprintf(`"%s":%s`, name, string(result))
		}
	}
	customResponses += "}"

	return fmt.Sprintf(`
import json, sys
TOOLS = %s
CUSTOM = %s
for line in sys.stdin:
    line = line.strip()
    if not line:
        continue
    try:
        req = json.loads(line)
    except:
        continue
    m = req.get("method","")
    rid = req.get("id")
    if m == "initialize":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"protocolVersion":"2024-11-05","capabilities":{"tools":{"listChanged":True}},"serverInfo":{"name":"mock","version":"1.0"}}}), flush=True)
    elif m == "notifications/initialized":
        pass
    elif m == "tools/list":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"tools":TOOLS}}), flush=True)
    elif m == "tools/call":
        params = req.get("params",{})
        tool_name = params.get("name","")
        if tool_name in CUSTOM:
            print(json.dumps({"jsonrpc":"2.0","id":rid,"result":CUSTOM[tool_name]}), flush=True)
        else:
            print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{"content":[{"type":"text","text":"ok"}]}}), flush=True)
    elif m == "ping":
        print(json.dumps({"jsonrpc":"2.0","id":rid,"result":{}}), flush=True)
    else:
        if not m.startswith("notifications/"):
            print(json.dumps({"jsonrpc":"2.0","id":rid,"error":{"code":-32601,"message":"method not found"}}), flush=True)
`, toolDefs, customResponses)
}
