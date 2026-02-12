package mcpserver

import (
	"encoding/json"
	"testing"
)

func TestParseToolCall(t *testing.T) {
	params := json.RawMessage(`{"name":"db_query","arguments":{"sql":"SELECT 1"}}`)
	tc, err := ParseToolCall(params)
	if err != nil {
		t.Fatal(err)
	}
	if tc.Name != "db_query" {
		t.Errorf("expected db_query, got %s", tc.Name)
	}
	if tc.Arguments["sql"] != "SELECT 1" {
		t.Errorf("unexpected arguments: %v", tc.Arguments)
	}
}

func TestParseToolCall_EmptyName(t *testing.T) {
	params := json.RawMessage(`{"arguments":{"x":1}}`)
	_, err := ParseToolCall(params)
	if err == nil {
		t.Fatal("expected error for empty tool name")
	}
}

func TestParseMessage_Request(t *testing.T) {
	msg := json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test"}}`)
	req, resp, err := ParseMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	if req == nil {
		t.Fatal("expected request")
	}
	if resp != nil {
		t.Fatal("expected nil response")
	}
	if req.Method != "tools/call" {
		t.Errorf("expected tools/call, got %s", req.Method)
	}
}

func TestParseMessage_Response(t *testing.T) {
	msg := json.RawMessage(`{"jsonrpc":"2.0","id":1,"result":{"tools":[]}}`)
	req, resp, err := ParseMessage(msg)
	if err != nil {
		t.Fatal(err)
	}
	if req != nil {
		t.Fatal("expected nil request")
	}
	if resp == nil {
		t.Fatal("expected response")
	}
}

func TestIsNotification(t *testing.T) {
	if !IsNotification("notifications/initialized") {
		t.Error("expected true for notifications/initialized")
	}
	if IsNotification("tools/call") {
		t.Error("expected false for tools/call")
	}
}
