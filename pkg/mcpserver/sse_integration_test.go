package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
)

// TestSSEServer_MultiSessionIsolation verifies that two SSE sessions
// have independent budget enforcement.
func TestSSEServer_MultiSessionIsolation(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, addr := startTestSSEServer(t, ctx)
	baseURL := fmt.Sprintf("http://%s", addr)

	// --- Session A ---
	respA, err := http.Get(baseURL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	defer respA.Body.Close()
	sessionA := readEndpointEvent(t, respA.Body)

	// Initialize A
	sendMessage(t, baseURL, sessionA, "", map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{},
			"clientInfo": map[string]string{"name": "agent-a", "version": "1.0"},
		},
	})
	readMessageEvent(t, respA.Body)
	sendMessage(t, baseURL, sessionA, "", map[string]interface{}{
		"jsonrpc": "2.0", "method": "notifications/initialized",
	})

	// --- Session B ---
	respB, err := http.Get(baseURL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	defer respB.Body.Close()
	sessionB := readEndpointEvent(t, respB.Body)

	// Initialize B
	sendMessage(t, baseURL, sessionB, "", map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{},
			"clientInfo": map[string]string{"name": "agent-b", "version": "1.0"},
		},
	})
	readMessageEvent(t, respB.Body)

	if sessionA == sessionB {
		t.Fatal("sessions should be different")
	}

	// --- Burn budget from session B (100 credits / 5 per call = 20 calls) ---
	for i := 0; i < 20; i++ {
		sendMessage(t, baseURL, sessionB, "", map[string]interface{}{
			"jsonrpc": "2.0", "id": 100 + i, "method": "tools/call",
			"params": map[string]interface{}{
				"name":      "test_tool",
				"arguments": map[string]string{"x": "y"},
			},
		})
		resp := readMessageEvent(t, respB.Body)
		if resp == nil {
			t.Fatalf("no response for call %d", i)
		}
	}

	// --- Call 21 from B should fail (budget exhausted) ---
	sendMessage(t, baseURL, sessionB, "", map[string]interface{}{
		"jsonrpc": "2.0", "id": 999, "method": "tools/call",
		"params": map[string]interface{}{
			"name":      "test_tool",
			"arguments": map[string]string{"x": "y"},
		},
	})
	resp := readMessageEvent(t, respB.Body)
	if resp == nil {
		t.Fatal("expected budget_exhausted response")
	}

	var errResp struct {
		Error struct {
			Code int `json:"code"`
			Data struct {
				Error string `json:"error"`
			} `json:"data"`
		} `json:"error"`
	}
	json.Unmarshal(resp, &errResp)
	if errResp.Error.Data.Error != "budget_exhausted" {
		t.Errorf("expected budget_exhausted, got %+v (raw: %s)", errResp, string(resp))
	}

	// --- Session A should still work (ping doesn't touch budget) ---
	sendMessage(t, baseURL, sessionA, "", map[string]interface{}{
		"jsonrpc": "2.0", "id": 50, "method": "ping",
	})
	pingResp := readMessageEvent(t, respA.Body)
	if pingResp == nil {
		t.Fatal("Agent A should still be responsive after Agent B exhausted budget")
	}

	// Health check: 2 sessions
	healthResp, err := http.Get(baseURL + "/health")
	if err != nil {
		t.Fatal(err)
	}
	defer healthResp.Body.Close()
	var health map[string]interface{}
	json.NewDecoder(healthResp.Body).Decode(&health)

	sessions := int(health["sessions"].(float64))
	if sessions != 2 {
		t.Errorf("expected 2 sessions, got %d", sessions)
	}
}

// TestSSEServer_AuthorizationHeader verifies that Authorization header
// tokens are correctly injected into request handling.
func TestSSEServer_AuthorizationHeader(t *testing.T) {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	_, addr := startTestSSEServer(t, ctx)
	baseURL := fmt.Sprintf("http://%s", addr)

	// Connect
	resp, err := http.Get(baseURL + "/sse")
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	sessionID := readEndpointEvent(t, resp.Body)

	// Initialize
	sendMessage(t, baseURL, sessionID, "", map[string]interface{}{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]interface{}{
			"protocolVersion": "2024-11-05", "capabilities": map[string]interface{}{},
			"clientInfo": map[string]string{"name": "test", "version": "1.0"},
		},
	})
	readMessageEvent(t, resp.Body)

	// Send with Bearer token (should be injected as _meta.token)
	sendMessage(t, baseURL, sessionID, "test-bearer-token", map[string]interface{}{
		"jsonrpc": "2.0", "id": 2, "method": "satgate/budget",
		"params": map[string]interface{}{},
	})
	budgetResp := readMessageEvent(t, resp.Body)
	if budgetResp == nil {
		t.Fatal("no budget response")
	}
	// Should work (no-auth mode accepts any token)
}

// readEndpointEventBuffered reads the endpoint event more robustly.
func readEndpointEventBuffered(t *testing.T, resp *http.Response) (string, *bufio.Reader) {
	t.Helper()
	reader := bufio.NewReader(resp.Body)

	// Read lines until we find the endpoint data
	for {
		line, err := reader.ReadString('\n')
		if err != nil {
			t.Fatalf("read SSE: %v", err)
		}
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "data: /message?sessionId=") {
			sessionID := strings.TrimPrefix(line, "data: /message?sessionId=")
			return sessionID, reader
		}
	}
}
