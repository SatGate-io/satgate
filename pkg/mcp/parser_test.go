package mcp

import (
	"bytes"
	"context"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

// ---------------------------------------------------------------------------
// Parser tests
// ---------------------------------------------------------------------------

func TestParseSingle_ToolsCall(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"database_query","arguments":{"sql":"SELECT 1"}}}`
	r := newJSONRequest(body)

	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Method != MethodToolsCall {
		t.Errorf("method = %q, want %q", info.Method, MethodToolsCall)
	}
	if info.ToolName != "database_query" {
		t.Errorf("tool = %q, want %q", info.ToolName, "database_query")
	}
	if info.IsBatch {
		t.Error("expected IsBatch = false")
	}

	// Body must be replayable.
	assertBodyReplay(t, r, body)
}

func TestParseSingle_ToolsList(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":2,"method":"tools/list"}`
	r := newJSONRequest(body)

	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Method != MethodToolsList {
		t.Errorf("method = %q, want %q", info.Method, MethodToolsList)
	}
	if info.ToolName != "" {
		t.Errorf("tool = %q, want empty", info.ToolName)
	}
}

func TestParseSingle_ResourcesRead(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":3,"method":"resources/read","params":{"uri":"file:///tmp/data.csv"}}`
	r := newJSONRequest(body)

	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Method != MethodResourceRead {
		t.Errorf("method = %q, want %q", info.Method, MethodResourceRead)
	}
	if info.ResourceURI != "file:///tmp/data.csv" {
		t.Errorf("uri = %q, want %q", info.ResourceURI, "file:///tmp/data.csv")
	}
}

func TestParseSingle_Initialize(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":0,"method":"initialize","params":{}}`
	r := newJSONRequest(body)

	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Method != MethodInitialize {
		t.Errorf("method = %q, want %q", info.Method, MethodInitialize)
	}
}

func TestParseSingle_Ping(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":99,"method":"ping"}`
	r := newJSONRequest(body)
	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Method != MethodPing {
		t.Errorf("method = %q, want %q", info.Method, MethodPing)
	}
}

func TestParseSingle_PromptsGet(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":4,"method":"prompts/get","params":{"name":"summarize"}}`
	r := newJSONRequest(body)
	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Method != MethodPromptsGet {
		t.Errorf("method = %q, want %q", info.Method, MethodPromptsGet)
	}
}

func TestParseSingle_PromptsList(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":5,"method":"prompts/list"}`
	r := newJSONRequest(body)
	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Method != MethodPromptsList {
		t.Errorf("method = %q, want %q", info.Method, MethodPromptsList)
	}
}

func TestParseSingle_ResourcesList(t *testing.T) {
	body := `{"jsonrpc":"2.0","id":6,"method":"resources/list"}`
	r := newJSONRequest(body)
	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Method != MethodResourceList {
		t.Errorf("method = %q, want %q", info.Method, MethodResourceList)
	}
}

// ---------------------------------------------------------------------------
// Batch tests
// ---------------------------------------------------------------------------

func TestParseBatch(t *testing.T) {
	body := `[
		{"jsonrpc":"2.0","id":1,"method":"tools/list"},
		{"jsonrpc":"2.0","id":2,"method":"tools/call","params":{"name":"web_search","arguments":{"q":"hello"}}}
	]`
	r := newJSONRequest(body)

	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if !info.IsBatch {
		t.Error("expected IsBatch = true")
	}
	if info.BatchSize != 2 {
		t.Errorf("BatchSize = %d, want 2", info.BatchSize)
	}
	// Should prefer tools/call.
	if info.Method != MethodToolsCall {
		t.Errorf("method = %q, want %q", info.Method, MethodToolsCall)
	}
	if info.ToolName != "web_search" {
		t.Errorf("tool = %q, want %q", info.ToolName, "web_search")
	}

	assertBodyReplay(t, r, body)
}

func TestParseBatch_NoToolsCall(t *testing.T) {
	body := `[{"jsonrpc":"2.0","id":1,"method":"tools/list"},{"jsonrpc":"2.0","id":2,"method":"ping"}]`
	r := newJSONRequest(body)
	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info")
	}
	if info.Method != MethodToolsList {
		t.Errorf("method = %q, want first method %q", info.Method, MethodToolsList)
	}
}

func TestParseBatch_Empty(t *testing.T) {
	body := `[]`
	r := newJSONRequest(body)
	info := Parse(r, DefaultMaxBodySize)
	if info != nil {
		t.Errorf("expected nil for empty batch, got %+v", info)
	}
}

// ---------------------------------------------------------------------------
// Non-MCP / malformed / edge cases
// ---------------------------------------------------------------------------

func TestParse_NonJSON(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader("hello world"))
	r.Header.Set("Content-Type", "text/plain")
	info := Parse(r, DefaultMaxBodySize)
	if info != nil {
		t.Errorf("expected nil for non-JSON body, got %+v", info)
	}
}

func TestParse_MalformedJSON(t *testing.T) {
	r := newJSONRequest(`{invalid json!!!}`)
	info := Parse(r, DefaultMaxBodySize)
	if info != nil {
		t.Errorf("expected nil for malformed JSON, got %+v", info)
	}
	// Body should still be replayable.
	assertBodyReplay(t, r, `{invalid json!!!}`)
}

func TestParse_NotJSONRPC(t *testing.T) {
	r := newJSONRequest(`{"foo":"bar"}`)
	info := Parse(r, DefaultMaxBodySize)
	if info != nil {
		t.Errorf("expected nil for non-JSONRPC, got %+v", info)
	}
}

func TestParse_MissingMethod(t *testing.T) {
	r := newJSONRequest(`{"jsonrpc":"2.0","id":1}`)
	info := Parse(r, DefaultMaxBodySize)
	if info != nil {
		t.Errorf("expected nil for missing method, got %+v", info)
	}
}

func TestParse_WrongVersion(t *testing.T) {
	r := newJSONRequest(`{"jsonrpc":"1.0","method":"tools/call"}`)
	info := Parse(r, DefaultMaxBodySize)
	if info != nil {
		t.Errorf("expected nil for wrong version, got %+v", info)
	}
}

func TestParse_NilBody(t *testing.T) {
	r := httptest.NewRequest(http.MethodPost, "/", nil)
	r.Body = nil
	info := Parse(r, DefaultMaxBodySize)
	if info != nil {
		t.Errorf("expected nil for nil body, got %+v", info)
	}
}

func TestParse_EmptyBody(t *testing.T) {
	r := newJSONRequest("")
	info := Parse(r, DefaultMaxBodySize)
	if info != nil {
		t.Errorf("expected nil for empty body, got %+v", info)
	}
}

func TestParse_OversizedBody(t *testing.T) {
	big := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"x","arguments":{"data":"` +
		strings.Repeat("A", 200) + `"}}}`
	r := newJSONRequest(big)

	info := Parse(r, 50)
	if info != nil {
		t.Errorf("expected nil for oversized body, got %+v", info)
	}

	// Body should still be replayable even when oversized.
	replayed, _ := io.ReadAll(r.Body)
	if !strings.HasPrefix(string(replayed), `{"jsonrpc"`) {
		t.Error("body not replayed after oversized skip")
	}
}

func TestParse_LeadingWhitespace(t *testing.T) {
	body := `   {"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test"}}`
	r := newJSONRequest(body)
	info := Parse(r, DefaultMaxBodySize)
	if info == nil {
		t.Fatal("expected non-nil info with leading whitespace")
	}
	if info.ToolName != "test" {
		t.Errorf("tool = %q, want %q", info.ToolName, "test")
	}
}

// ---------------------------------------------------------------------------
// Context round-trip tests
// ---------------------------------------------------------------------------

func TestContext_RoundTrip(t *testing.T) {
	info := &RequestInfo{Method: MethodToolsCall, ToolName: "test_tool"}
	ctx := WithInfo(context.Background(), info)
	got := GetInfo(ctx)
	if got == nil {
		t.Fatal("expected non-nil info from context")
	}
	if got.ToolName != "test_tool" {
		t.Errorf("ToolName = %q, want %q", got.ToolName, "test_tool")
	}
}

func TestContext_Missing(t *testing.T) {
	got := GetInfo(context.Background())
	if got != nil {
		t.Errorf("expected nil from empty context, got %+v", got)
	}
}

// ---------------------------------------------------------------------------
// Middleware tests
// ---------------------------------------------------------------------------

func TestMiddleware_MCPRequest(t *testing.T) {
	var captured *RequestInfo
	handler := Middleware(DefaultMaxBodySize)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		captured = GetInfo(r.Context())
		// Verify body is still readable.
		body, _ := io.ReadAll(r.Body)
		if len(body) == 0 {
			t.Error("body was empty in downstream handler")
		}
		w.WriteHeader(http.StatusOK)
	}))

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"web_search","arguments":{"q":"hello"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if captured == nil {
		t.Fatal("middleware did not set context info")
	}
	if captured.ToolName != "web_search" {
		t.Errorf("tool = %q, want web_search", captured.ToolName)
	}
}

func TestMiddleware_NonMCPPassthrough(t *testing.T) {
	handler := Middleware(DefaultMaxBodySize)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := GetInfo(r.Context())
		if info != nil {
			t.Errorf("expected nil info for non-MCP request, got %+v", info)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodGet, "/api/data", nil)
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestMiddleware_NonJSONContentType(t *testing.T) {
	handler := Middleware(DefaultMaxBodySize)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := GetInfo(r.Context())
		if info != nil {
			t.Errorf("expected nil info for non-JSON content, got %+v", info)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{"jsonrpc":"2.0","method":"ping"}`))
	req.Header.Set("Content-Type", "text/plain")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

func TestMiddleware_MalformedJSONPassthrough(t *testing.T) {
	called := false
	handler := Middleware(DefaultMaxBodySize)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		called = true
		info := GetInfo(r.Context())
		if info != nil {
			t.Error("expected nil info for malformed JSON")
		}
		body, _ := io.ReadAll(r.Body)
		if string(body) != `{bad json` {
			t.Errorf("body = %q, want {bad json", string(body))
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(`{bad json`))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)

	if !called {
		t.Error("downstream handler was not called")
	}
}

func TestMiddleware_BodyReplayAfterParsing(t *testing.T) {
	original := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"test","arguments":{"key":"value"}}}`

	handler := Middleware(DefaultMaxBodySize)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, err := io.ReadAll(r.Body)
		if err != nil {
			t.Fatalf("failed to read replayed body: %v", err)
		}
		if string(body) != original {
			t.Errorf("replayed body = %q, want %q", string(body), original)
		}
		w.WriteHeader(http.StatusOK)
	}))

	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(original))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
}

func TestMiddleware_OversizedSkipsParsingButPassesThrough(t *testing.T) {
	handler := Middleware(50)(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		info := GetInfo(r.Context())
		if info != nil {
			t.Error("expected nil info for oversized body")
		}
		w.WriteHeader(http.StatusOK)
	}))

	body := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"x","arguments":{"data":"` +
		strings.Repeat("A", 200) + `"}}}`
	req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	rr := httptest.NewRecorder()
	handler.ServeHTTP(rr, req)
	if rr.Code != http.StatusOK {
		t.Errorf("status = %d, want 200", rr.Code)
	}
}

// ---------------------------------------------------------------------------
// Helpers
// ---------------------------------------------------------------------------

func newJSONRequest(body string) *http.Request {
	r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(body))
	r.Header.Set("Content-Type", "application/json")
	return r
}

func assertBodyReplay(t *testing.T, r *http.Request, expected string) {
	t.Helper()
	replayed, err := io.ReadAll(r.Body)
	if err != nil {
		t.Fatalf("failed to read replayed body: %v", err)
	}
	if !bytes.Equal(replayed, []byte(expected)) {
		t.Errorf("replayed body mismatch:\ngot:  %q\nwant: %q", string(replayed), expected)
	}
}
