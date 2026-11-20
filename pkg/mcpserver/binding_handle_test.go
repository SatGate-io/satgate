package mcpserver

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"os"
	"strings"
	"testing"
	"time"
)

const bindingTestSignal = `{"io.satgate/receipt-binding":{"version":1}}`
const bindingTestMeta = `{"_meta":` + bindingTestSignal + `}`
const bindingTestInit = `{"capabilities":{"experimental":` + bindingTestSignal + `}}`
const bindingTestTool = ` {"name":"echo", "inputSchema":{ "type":"object" },"title":"\u0065cho","description":"echo","annotations":{"readOnlyHint":true},"_meta":{"n":1.00e2},"x-satgate":{ "n":9007199254740993 }} `
const bindingTestSchema = `{ "type":"object", "properties": {"n":{"maximum":9007199254740993}} }`
const bindingTestExisting = `{"name":"existing","outputSchema":` + bindingTestSchema + `,"inputSchema":{"type":"object"}}`
const bindingTestCollision = `{"name":"collision","outputSchema":{"properties":{"broker_result":{}}}}`

var bindingTestRef = HandleRef{ID: "decision-local", Hash: "test-hash", URL: "https://example.invalid/test"}

type bindingTestRouter struct {
	tools  []json.RawMessage
	result json.RawMessage
}

func (r bindingTestRouter) AllToolsForTenant(context.Context, string) []json.RawMessage {
	return r.tools
}
func (r bindingTestRouter) ForwardToolCallForTenant(context.Context, string, string, json.RawMessage, time.Duration) (*Response, error) {
	return &Response{JSONRPC: "2.0", ID: json.RawMessage(`999`), Result: r.result}, nil
}

type bindingTestProvider func(context.Context, *Request) (*BindingHandle, error)

func (f bindingTestProvider) HandleFor(ctx context.Context, req *Request) (*BindingHandle, error) {
	return f(ctx, req)
}

func bindingTestRequest(t *testing.T, p *Proxy, ctx context.Context, method, params string) *Response {
	t.Helper()
	req := &Request{JSONRPC: "2.0", ID: json.RawMessage(`"dummy"`), Method: method, Params: json.RawMessage(params)}
	resp, err := p.handleRequest(ctx, req)
	if err != nil || resp == nil || resp.Error != nil {
		t.Fatalf("%s: %+v, %v", method, resp, err)
	}
	return resp
}

func bindingTestFields(t *testing.T, raw json.RawMessage) map[string]json.RawMessage {
	t.Helper()
	var fields map[string]json.RawMessage
	if err := json.Unmarshal(raw, &fields); err != nil || fields == nil {
		t.Fatalf("object: %s (%v)", raw, err)
	}
	return fields
}

func bindingTestNew(t *testing.T) *BindingHandle {
	t.Helper()
	h, err := NewBindingHandle(bindingTestRef, ObservationPending, nil)
	if err != nil || h == nil || !h.IsOwned() {
		t.Fatalf("handle: %+v, %v", h, err)
	}
	return h
}

func TestBindingHandle_OldSchemaUnchanged(t *testing.T) {
	httpHandler, p := newStreamableHandler(t, 100)
	upstream := json.RawMessage(`{"content":[], "_meta":{"satgate_evidence":{"binding":"dummy"}}, "structuredContent":{ "satgate_receipt":"dummy" }}`)
	tools := []json.RawMessage{json.RawMessage(bindingTestTool), json.RawMessage(bindingTestExisting)}
	p.SetUpstreamRouter(bindingTestRouter{tools: tools, result: upstream})
	ctx := context.Background()
	wantList := mustJSON(t, map[string]any{"tools": tools})
	for _, params := range []string{"", bindingTestMeta} {
		if got := bindingTestRequest(t, p, ctx, MethodToolsList, params).Result; !bytes.Equal(got, wantList) {
			t.Fatalf("tools/list = %s, want %s", got, wantList)
		}
	}
	wantInit := `{"capabilities":{"tools":{"listChanged":true}},"protocolVersion":"2024-11-05","serverInfo":{"name":"streamable-test","version":"test"}}`
	for _, params := range []string{"", bindingTestInit} {
		if got := bindingTestRequest(t, p, ctx, MethodInitialize, params).Result; string(got) != wantInit {
			t.Fatalf("initialize = %s, want %s", got, wantInit)
		}
	}
	sid := bindingTestHTTPSession(t, httpHandler, bindingTestInit)
	if got := bindingTestHTTPList(t, httpHandler, sid, ""); !bytes.Equal(got, wantList) {
		t.Fatalf("session with nil provider: %s", got)
	}
	if got := bindingTestRequest(t, p, ctx, MethodToolsCall, `{"name":"echo","_meta":`+bindingTestSignal+`}`).Result; !bytes.Equal(got, upstream) {
		t.Fatalf("call with nil provider: %s", got)
	}
	p.SetBindingProvider(bindingTestProvider(func(context.Context, *Request) (*BindingHandle, error) {
		t.Fatal("unexpected provider call")
		return nil, nil
	}))
	if got := bindingTestRequest(t, p, ctx, MethodToolsCall, `{"name":"echo"}`).Result; !bytes.Equal(got, upstream) {
		t.Fatalf("call without opt-in: %s", got)
	}
	for _, params := range []string{
		`{"_meta":{"token":"dummy"}}`,
		`{"_meta":{"io.satgate/receipt-binding":{"version":"1"}}}`,
		`{"_meta":{"io.satgate/receipt-binding":{"version":2}}}`,
		`{"_meta":{"io.satgate/receipt-binding":{}}}`,
		`{"_meta":{"io.satgate/receipt-binding":1}}`,
		`{"_meta":{"io.satgate/receipt-binding":{"version":1.0000000000000000001}}}`,
	} {
		if got := bindingTestRequest(t, p, ctx, MethodToolsList, params).Result; !bytes.Equal(got, wantList) {
			t.Fatalf("invalid opt-in %s: %s", params, got)
		}
	}
}

func TestBindingHandle_UpstreamHandleNotOwned(t *testing.T) {
	data, err := os.ReadFile("testdata/untrusted-metadata-fixtures.json")
	if err != nil {
		t.Fatal(err)
	}
	var fixtures []json.RawMessage
	if err := json.Unmarshal(data, &fixtures); err != nil || len(fixtures) < 2 {
		t.Fatalf("fixtures: %d, %v", len(fixtures), err)
	}
	_, p := newStreamableHandler(t, 100)
	owned := bindingTestNew(t)
	p.SetBindingProvider(bindingTestProvider(func(context.Context, *Request) (*BindingHandle, error) { return owned, nil }))
	for _, raw := range fixtures {
		fields := bindingTestFields(t, raw)
		var handleRaw json.RawMessage
		if meta, ok := fields["_meta"]; ok {
			handleRaw = bindingTestFields(t, meta)["satgate_evidence"]
		} else {
			handleRaw = bindingTestFields(t, fields["structuredContent"])["satgate_receipt"]
		}
		var decoded BindingHandle
		if err := json.Unmarshal(handleRaw, &decoded); err != nil || decoded.IsOwned() {
			t.Fatalf("untrusted handle: %+v, %v", decoded, err)
		}
		if got, ok := ApplyBindingHandle(raw, nil, &decoded); ok || !bytes.Equal(got, raw) {
			t.Fatalf("untrusted result: %s, %v", got, ok)
		}
		local, err := NewBindingHandle(decoded.Decision, decoded.ObservationState, decoded.Observation)
		if err != nil || !local.IsOwned() {
			t.Fatalf("constructed handle: %+v, %v", local, err)
		}
		copied := *local
		if !copied.IsOwned() {
			t.Fatal("in-process copy lost ownership")
		}
		if err := json.Unmarshal(mustJSON(t, local), local); err != nil || local.IsOwned() {
			t.Fatalf("round trip: %+v, %v", local, err)
		}
		p.SetUpstreamRouter(bindingTestRouter{result: raw})
		resp := bindingTestRequest(t, p, context.Background(), MethodToolsCall, `{"name":"echo","_meta":`+bindingTestSignal+`}`)
		if string(resp.ID) != `"dummy"` {
			t.Fatalf("id: %s", resp.ID)
		}
		wrappedFields := bindingTestFields(t, resp.Result)
		wrapped := bindingTestFields(t, wrappedFields["structuredContent"])
		if got := wrapped["satgate_receipt"]; !bytes.Equal(got, mustJSON(t, owned)) {
			t.Fatalf("provider handle: %s", got)
		}
		if got := wrapped["broker_result"]; !bytes.Equal(got, fields["structuredContent"]) {
			t.Fatalf("broker result: %s, want %s", got, fields["structuredContent"])
		}
		meta := bindingTestFields(t, wrappedFields["_meta"])
		if evidence := bindingTestFields(t, meta["satgate_evidence"]); len(evidence) != 1 || !bytes.Equal(evidence["binding"], wrapped["satgate_receipt"]) {
			t.Fatalf("evidence without recorder: %s", meta["satgate_evidence"])
		}
	}
}

func TestBindingHandle_OptInWrap(t *testing.T) {
	t.Run("schemas", func(t *testing.T) {
		plain := json.RawMessage(bindingTestTool)
		wrapped, ok := ToolWithBindingSchema(plain)
		if !ok {
			t.Fatal("missing wrapper")
		}
		fields := bindingTestFields(t, wrapped)
		schema := fields["outputSchema"]
		if len(schema) == 0 {
			t.Fatal("missing outputSchema")
		}
		if got := bytes.Replace(wrapped, append([]byte(`,"outputSchema":`), schema...), nil, 1); !bytes.Equal(got, plain) {
			t.Fatalf("other tool bytes changed: %s", got)
		}
		ref := `{"type":"object","additionalProperties":false,"required":["id","hash","url"],"properties":{"id":{"type":"string"},"hash":{"type":"string"},"url":{"type":"string"}}}`
		want := `{"type":"object","additionalProperties":false,"required":["broker_result","satgate_receipt"],"properties":{"broker_result":{"type":"object"},"satgate_receipt":{"type":"object","additionalProperties":false,"required":["schema","decision","observation_state"],"properties":{"schema":{"const":"satgate.binding_handle.v1"},"decision":` + ref + `,"observation":` + ref + `,"observation_state":{"enum":["available","pending","unknown","not_dispatched"]}}}}}`
		var gotObject, wantObject any
		if json.Unmarshal(schema, &gotObject) != nil || json.Unmarshal([]byte(want), &wantObject) != nil || !bytes.Equal(mustJSON(t, gotObject), mustJSON(t, wantObject)) {
			t.Fatalf("outputSchema: %s", schema)
		}
		existing, ok := ToolWithBindingSchema(json.RawMessage(bindingTestExisting))
		if !ok {
			t.Fatal("existing schema was not wrapped")
		}
		existingSchema := bindingTestFields(t, bindingTestFields(t, existing)["outputSchema"])
		if got := bindingTestFields(t, existingSchema["properties"])["broker_result"]; string(got) != bindingTestSchema {
			t.Fatalf("original outputSchema: %s", got)
		}
		for _, raw := range []string{bindingTestCollision, strings.ReplaceAll(bindingTestCollision, "broker_result", "satgate_receipt"), `null`, `{"name":`} {
			if got, ok := ToolWithBindingSchema(json.RawMessage(raw)); ok || string(got) != raw {
				t.Fatalf("schema collision or invalid: %s, %v", got, ok)
			}
		}
	})
	t.Run("result", func(t *testing.T) {
		h := bindingTestNew(t)
		raw := json.RawMessage(`{"content":[ {"type":"text","text":"\u0061"} ],"isError":false,"other":{ "n":9007199254740993 },"_meta":{"extra": { "n":1.0e3 },"satgate_evidence":{"upstream_only":true}}}`)
		e := &MCPEvidence{ReceiptID: "receipt-local", ReceiptHash: "hash-local", EvidencePackID: "pack-local", EvidenceURL: "https://example.invalid/evidence", VerifyURL: "https://example.invalid/check", JWKSURL: "https://example.invalid/jwks"}
		wrapped, ok := ApplyBindingHandle(raw, e, h)
		if !ok {
			t.Fatal("result was not wrapped")
		}
		bindingTestCheckResult(t, raw, wrapped, e, h)
		if got, ok := ApplyBindingHandle(raw, e, nil); ok || !bytes.Equal(got, raw) {
			t.Fatalf("nil handle: %s, %v", got, ok)
		}
		if got, ok := ApplyBindingHandle(json.RawMessage(`{}`), nil, h); !ok || string(bindingTestFields(t, bindingTestFields(t, got)["structuredContent"])["broker_result"]) != "null" {
			t.Fatalf("missing result fields: %s, %v", got, ok)
		}
		if got, ok := ApplyBindingHandle(json.RawMessage(`{"structuredContent":null,"content":[]}`), nil, h); !ok || string(bindingTestFields(t, bindingTestFields(t, got)["structuredContent"])["broker_result"]) != "null" {
			t.Fatalf("present null structuredContent: %s, %v", got, ok)
		}
		_, p := newStreamableHandler(t, 100)
		p.SetUpstreamRouter(bindingTestRouter{result: raw})
		recorder := &fakeEvidenceRecorder{}
		p.SetEvidenceRecorder(recorder)
		wantEvidence, err := (&fakeEvidenceRecorder{}).RecordMCPDecision(context.Background(), MCPDecision{})
		if err != nil {
			t.Fatal(err)
		}
		params := `{"name":"echo","_meta":` + bindingTestSignal + `}`
		p.SetBindingProvider(bindingTestProvider(func(context.Context, *Request) (*BindingHandle, error) { return h, nil }))
		bindingTestCheckResult(t, raw, bindingTestRequest(t, p, context.Background(), MethodToolsCall, params).Result, wantEvidence, h)
	})
	t.Run("provider-error", func(t *testing.T) {
		_, p := newStreamableHandler(t, 100)
		raw := json.RawMessage(`{"content":[], "structuredContent":{"n":1.00}}`)
		p.SetUpstreamRouter(bindingTestRouter{result: raw})
		p.SetBindingProvider(bindingTestProvider(func(context.Context, *Request) (*BindingHandle, error) { return nil, errors.New("test") }))
		params := `{"name":"echo","_meta":` + bindingTestSignal + `}`
		if got := bindingTestRequest(t, p, context.Background(), MethodToolsCall, params).Result; !bytes.Equal(got, raw) {
			t.Fatalf("provider error: %s", got)
		}
	})
	t.Run("session-isolation", func(t *testing.T) {
		handler, p := newStreamableHandler(t, 100)
		tools := []json.RawMessage{json.RawMessage(bindingTestTool), json.RawMessage(bindingTestExisting), json.RawMessage(bindingTestCollision)}
		p.SetUpstreamRouter(bindingTestRouter{tools: tools})
		p.SetBindingProvider(bindingTestProvider(func(context.Context, *Request) (*BindingHandle, error) {
			t.Fatal("provider called for tools/list")
			return nil, nil
		}))
		first := bindingTestHTTPSession(t, handler, bindingTestInit)
		second := bindingTestHTTPSession(t, handler, `{"capabilities":{"tools":`+bindingTestSignal+`}}`)
		if first == second {
			t.Fatal("sessions were not distinct")
		}
		wantOld := mustJSON(t, map[string]any{"tools": tools})
		checkWrapped := func(got json.RawMessage, exact bool) {
			t.Helper()
			var listed []json.RawMessage
			if err := json.Unmarshal(bindingTestFields(t, got)["tools"], &listed); err != nil || len(listed) != len(tools) {
				t.Fatalf("tools: %s, %v", got, err)
			}
			for i := range tools {
				want := tools[i]
				if transformed, ok := ToolWithBindingSchema(want); ok {
					want = transformed
				}
				if !exact {
					want = mustJSON(t, want)
				}
				if !bytes.Equal(listed[i], bytes.TrimSpace(want)) {
					t.Fatalf("tool %d: %s, want %s", i, listed[i], want)
				}
			}
		}
		checkWrapped(bindingTestRequest(t, p, withBindingOptIn(context.Background()), MethodToolsList, "").Result, true)
		checkWrapped(bindingTestHTTPList(t, handler, first, ""), false)
		checkWrapped(bindingTestHTTPList(t, handler, second, bindingTestMeta), false)
		if got := bindingTestHTTPList(t, handler, second, ""); !bytes.Equal(got, wantOld) {
			t.Fatalf("opt-in leaked between HTTP requests or sessions: %s", got)
		}
		if string(tools[0]) != bindingTestTool || string(tools[1]) != bindingTestExisting || string(tools[2]) != bindingTestCollision {
			t.Fatalf("cached tools changed: %s", tools)
		}
	})
}

func bindingTestCheckResult(t *testing.T, raw, wrapped json.RawMessage, e *MCPEvidence, h *BindingHandle) {
	t.Helper()
	original, got := bindingTestFields(t, raw), bindingTestFields(t, wrapped)
	for _, key := range []string{"content", "isError", "other"} {
		if !bytes.Equal(got[key], original[key]) {
			t.Fatalf("%s: %s, want %s", key, got[key], original[key])
		}
	}
	meta := bindingTestFields(t, got["_meta"])
	if want := bindingTestFields(t, original["_meta"])["extra"]; !bytes.Equal(meta["extra"], want) {
		t.Fatalf("other metadata: %s, want %s", meta["extra"], want)
	}
	wrappedContent := bindingTestFields(t, got["structuredContent"])
	if !bytes.Equal(wrappedContent["broker_result"], original["content"]) {
		t.Fatalf("content fallback: %s", wrappedContent["broker_result"])
	}
	evidence := bindingTestFields(t, meta["satgate_evidence"])
	if len(evidence) != 7 || !bytes.Equal(evidence["binding"], wrappedContent["satgate_receipt"]) || !bytes.Equal(evidence["binding"], mustJSON(t, h)) {
		t.Fatalf("satgate_evidence: %s", meta["satgate_evidence"])
	}
	for key, value := range map[string]string{"receipt_id": e.ReceiptID, "receipt_hash": e.ReceiptHash, "evidence_pack_id": e.EvidencePackID, "evidence_url": e.EvidenceURL, "verify_url": e.VerifyURL, "jwks_url": e.JWKSURL} {
		if !bytes.Equal(evidence[key], mustJSON(t, value)) {
			t.Fatalf("%s: %s", key, evidence[key])
		}
	}
}

func bindingTestHTTPSession(t *testing.T, handler http.Handler, params string) string {
	t.Helper()
	status, hdr, body := streamablePostHandler(t, handler, "/mcp", "", "", json.RawMessage(`{"jsonrpc":"2.0","id":1,"method":"initialize","params":`+params+`}`))
	if sid := hdr.Get("Mcp-Session-Id"); status == http.StatusOK && sid != "" {
		return sid
	}
	t.Fatalf("initialize: %d %s", status, body)
	return ""
}

func bindingTestHTTPList(t *testing.T, handler http.Handler, sid, params string) json.RawMessage {
	t.Helper()
	req := Request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: MethodToolsList, Params: json.RawMessage(params)}
	status, _, body := streamablePostHandler(t, handler, "/mcp", sid, "", req)
	var resp Response
	if err := json.Unmarshal([]byte(body), &resp); status != http.StatusOK || err != nil || resp.Error != nil {
		t.Fatalf("tools/list: %d %s (%v)", status, body, err)
	}
	return resp.Result
}

func TestBindingHandle_Stage1States(t *testing.T) {
	for _, state := range []ObservationState{ObservationPending, ObservationUnknown, ObservationNotDispatched} {
		h, err := NewBindingHandle(bindingTestRef, state, nil)
		if err != nil || !h.IsOwned() {
			t.Fatalf("%s: %+v, %v", state, h, err)
		}
		fields := bindingTestFields(t, mustJSON(t, h))
		if len(fields) != 3 || string(fields["observation_state"]) != `"`+string(state)+`"` || string(fields["schema"]) != `"`+BindingHandleSchema+`"` || !bytes.Equal(fields["decision"], mustJSON(t, bindingTestRef)) {
			t.Fatalf("%s: %s", state, mustJSON(t, h))
		}
		if h, err := NewBindingHandle(bindingTestRef, state, &bindingTestRef); err == nil || h != nil {
			t.Fatalf("%s with observation: %+v, %v", state, h, err)
		}
	}
	if h, err := NewBindingHandle(bindingTestRef, ObservationAvailable, nil); err == nil || h != nil {
		t.Fatalf("available without observation: %+v, %v", h, err)
	}
	ref := bindingTestRef
	h, err := NewBindingHandle(bindingTestRef, ObservationAvailable, &ref)
	if err != nil || !h.IsOwned() || !bytes.Equal(bindingTestFields(t, mustJSON(t, h))["observation"], mustJSON(t, ref)) {
		t.Fatalf("available: %+v, %v", h, err)
	}
	ref.ID = "changed"
	if h.Observation.ID != bindingTestRef.ID {
		t.Fatalf("observation ref aliased: %+v", h.Observation)
	}
	if h, err := NewBindingHandle(bindingTestRef, ObservationState("other"), nil); err == nil || h != nil {
		t.Fatalf("unrecognized state: %+v, %v", h, err)
	}
	for _, invalid := range []HandleRef{
		{Hash: "h", URL: bindingTestRef.URL},
		{ID: "i", URL: bindingTestRef.URL},
		{ID: "tab\t", Hash: "h", URL: bindingTestRef.URL},
		{ID: "i", Hash: "space\u00a0", URL: bindingTestRef.URL},
		{ID: strings.Repeat("x", 257), Hash: "h", URL: bindingTestRef.URL},
		{ID: "i", Hash: "h", URL: ""},
		{ID: "i", Hash: "h", URL: "http://example.invalid"},
		{ID: "i", Hash: "h", URL: "https:/test"},
		{ID: "i", Hash: "h", URL: "https://:443/test"},
		{ID: "i", Hash: "h", URL: "https://dummy@example.invalid"},
		{ID: "i", Hash: "h", URL: bindingTestRef.URL + strings.Repeat("x", 2048)},
	} {
		if h, err := NewBindingHandle(invalid, ObservationPending, nil); err == nil || h != nil {
			t.Fatalf("invalid decision: %+v, %v", h, err)
		}
		if h, err := NewBindingHandle(bindingTestRef, ObservationAvailable, &invalid); err == nil || h != nil {
			t.Fatalf("invalid observation: %+v, %v", h, err)
		}
	}
}
