package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"
)

func TestBindingOptedIn(t *testing.T) {
	t.Run("table", func(t *testing.T) {
		bg := context.Background()
		session := withBindingOptIn(bg)
		request := func(params string) *Request {
			return &Request{Method: MethodToolsCall, Params: json.RawMessage(params)}
		}
		for _, tt := range []struct {
			name string
			ctx  context.Context
			req  *Request
			want bool
		}{
			{"nil request", bg, nil, false},
			{"nil request with session flag", session, nil, false},
			{"nil context and request", nil, nil, false},
			{"nil context with signal", nil, request(bindingTestMeta), true},
			{"nil context without signal", nil, request(`{"name":"echo"}`), false},
			{"session flag without metadata", session, request(`{"name":"echo"}`), true},
			{"no params", bg, request(""), false},
			{"no metadata", bg, request(`{"name":"echo"}`), false},
			{"signal", bg, request(bindingTestMeta), true},
			{"initialize capability is not request metadata", bg, &Request{Method: MethodInitialize, Params: json.RawMessage(bindingTestInit)}, false},
			{"token only", bg, request(`{"_meta":{"token":"dummy"}}`), false},
			{"string version", bg, request(`{"_meta":{"io.satgate/receipt-binding":{"version":"1"}}}`), false},
			{"other version", bg, request(`{"_meta":{"io.satgate/receipt-binding":{"version":2}}}`), false},
			{"missing version", bg, request(`{"_meta":{"io.satgate/receipt-binding":{}}}`), false},
			{"non-object capability", bg, request(`{"_meta":{"io.satgate/receipt-binding":1}}`), false},
			{"nearby decimal version", bg, request(`{"_meta":{"io.satgate/receipt-binding":{"version":1.0000000000000000001}}}`), false},
		} {
			t.Run(tt.name, func(t *testing.T) {
				got := BindingOptedIn(tt.ctx, tt.req)
				if got != tt.want {
					t.Errorf("BindingOptedIn = %v, want %v", got, tt.want)
				}
				if tt.req != nil {
					ctx := tt.ctx
					if ctx == nil {
						ctx = bg
					}
					if proxy := bindingRequestOptIn(ctx, tt.req); proxy != got {
						t.Errorf("BindingOptedIn = %v, proxy check = %v", got, proxy)
					}
				}
			})
		}
	})

	t.Run("sessions", func(t *testing.T) {
		handler, p := newStreamableHandler(t, 100)
		tools := []json.RawMessage{json.RawMessage(bindingTestTool), json.RawMessage(bindingTestExisting)}
		result := json.RawMessage(`{"content":[], "structuredContent":{ "n":9007199254740993 }}`)
		seen := make(chan bool, 1)
		p.SetUpstreamRouter(bindingOptInTestRouter{
			bindingTestRouter: bindingTestRouter{tools: tools, result: result},
			observe: func(ctx context.Context, params json.RawMessage) {
				// The router sees the context on every call and the params with any caller token removed.
				req := &Request{Method: MethodToolsCall, Params: params}
				select {
				case seen <- BindingOptedIn(ctx, req):
				default:
					t.Error("unexpected additional router call")
				}
			},
		})
		handle := bindingTestNew(t)
		p.SetBindingProvider(bindingTestProvider(func(context.Context, *Request) (*BindingHandle, error) { return handle, nil }))

		const plain = `{"name":"echo"}`
		const signal = `{"name":"echo","_meta":` + bindingTestSignal + `}`
		checkResponse := func(raw []byte, req Request) Response {
			t.Helper()
			var resp Response
			if err := json.Unmarshal(raw, &resp); err != nil || resp.Error != nil || !bytes.Equal(resp.ID, req.ID) || resp.Result == nil {
				t.Fatalf("%s id %s: %s (%v)", req.Method, req.ID, raw, err)
			}
			return resp
		}
		checkCall := func(raw []byte, req Request, want bool) {
			t.Helper()
			resp := checkResponse(raw, req)
			select {
			case got := <-seen:
				if got != want {
					t.Errorf("call id %s: BindingOptedIn = %v, want %v", req.ID, got, want)
				}
			default:
				t.Fatalf("call id %s did not reach the router", req.ID)
			}
			if want {
				structured := bindingTestFields(t, bindingTestFields(t, resp.Result)["structuredContent"])
				if got := structured["satgate_receipt"]; !bytes.Equal(got, mustJSON(t, handle)) {
					t.Errorf("call id %s: receipt = %s", req.ID, got)
				}
			}
		}
		checkBytes := func(body string, req Request, wantResult json.RawMessage) {
			t.Helper()
			want := string(mustJSON(t, Response{JSONRPC: "2.0", ID: req.ID, Result: wantResult})) + "\n"
			if body != want {
				t.Errorf("%s id %s: body = %q, want %q", req.Method, req.ID, body, want)
			}
		}

		opted := bindingTestHTTPSession(t, handler, bindingTestInit)
		legacy := bindingTestHTTPSession(t, handler, `{}`)
		if opted == legacy {
			t.Fatal("HTTP sessions were not distinct")
		}
		httpPost := func(sid string, req Request) string {
			t.Helper()
			status, _, body := streamablePostHandler(t, handler, "/mcp", sid, "", req)
			if status != http.StatusOK {
				t.Fatalf("%s id %s: HTTP %d: %s", req.Method, req.ID, status, body)
			}
			return body
		}
		list := Request{JSONRPC: "2.0", ID: json.RawMessage(`2`), Method: MethodToolsList}
		wantList := mustJSON(t, map[string]any{"tools": tools})
		checkBytes(httpPost(legacy, list), list, wantList)
		for _, call := range []struct {
			sid, id, params string
			want            bool
		}{
			{opted, "3", plain, true},
			{legacy, "4", plain, false},
			{legacy, "5", signal, true},
			{legacy, "6", plain, false},
			{opted, "7", plain, true},
		} {
			req := Request{JSONRPC: "2.0", ID: json.RawMessage(call.id), Method: MethodToolsCall, Params: json.RawMessage(call.params)}
			body := httpPost(call.sid, req)
			checkCall([]byte(body), req, call.want)
			if !call.want {
				checkBytes(body, req, result)
			}
		}
		checkBytes(httpPost(legacy, list), list, wantList)

		srv := httptest.NewServer(handler)
		defer srv.Close()
		client := srv.Client()
		client.Timeout = 5 * time.Second
		stream, err := client.Get(srv.URL + "/sse")
		if err != nil {
			t.Fatal(err)
		}
		defer stream.Body.Close()
		if stream.StatusCode != http.StatusOK {
			t.Fatalf("SSE connect: HTTP %d", stream.StatusCode)
		}
		scanner := bufio.NewScanner(stream.Body)
		nextData := func() string {
			t.Helper()
			for scanner.Scan() {
				if data, ok := strings.CutPrefix(scanner.Text(), "data: "); ok {
					return data
				}
			}
			t.Fatalf("missing SSE data: %v", scanner.Err())
			return ""
		}
		endpoint := nextData()
		if sid, ok := strings.CutPrefix(endpoint, "/message?sessionId="); !ok || sid == "" {
			t.Fatalf("invalid SSE endpoint: %q", endpoint)
		}
		ssePost := func(req Request) []byte {
			t.Helper()
			resp, err := client.Post(srv.URL+endpoint, "application/json", bytes.NewReader(mustJSON(t, req)))
			if err != nil {
				t.Fatal(err)
			}
			resp.Body.Close()
			if resp.StatusCode != http.StatusAccepted {
				t.Fatalf("SSE %s: HTTP %d", req.Method, resp.StatusCode)
			}
			return []byte(nextData())
		}
		init := Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodInitialize, Params: json.RawMessage(bindingTestInit)}
		checkResponse(ssePost(init), init)
		for _, id := range []string{"2", "3"} {
			req := Request{JSONRPC: "2.0", ID: json.RawMessage(id), Method: MethodToolsCall, Params: json.RawMessage(plain)}
			checkCall(ssePost(req), req, true)
		}
	})
}

type bindingOptInTestRouter struct {
	bindingTestRouter
	observe func(context.Context, json.RawMessage)
}

func (r bindingOptInTestRouter) ForwardToolCallForTenant(ctx context.Context, tenant, tool string, params json.RawMessage, timeout time.Duration) (*Response, error) {
	r.observe(ctx, params)
	return r.bindingTestRouter.ForwardToolCallForTenant(ctx, tenant, tool, params, timeout)
}
