package mcpserver

import (
	"bufio"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"sort"
	"strings"
	"testing"
	"time"
)

// Tests for what the proxy hands to upstreams (upstream_hygiene.go): the
// environment of a stdio upstream, the caller's token in params._meta, and
// the ?token= URL fallback.

const (
	// hygieneHelperEnv makes this test binary act as a stdio MCP upstream
	// (TestHygieneHelperUpstream). It reaches the child through the
	// upstream's Env map, which is also what the env test checks.
	hygieneHelperEnv = "SATGATE_HYGIENE_HELPER_UPSTREAM"
	hygieneToken     = "hygiene-caller-token"
)

// TestHygieneHelperUpstream is not a test: run as a child with
// hygieneHelperEnv=1 it is a stdio MCP server with one tool, "report", whose
// answer is the child's environment and the params it was sent. Every other
// method is answered with the params it was sent.
func TestHygieneHelperUpstream(t *testing.T) {
	if os.Getenv(hygieneHelperEnv) != "1" {
		t.Skip("helper process for the upstream hygiene tests")
	}
	sc := bufio.NewScanner(os.Stdin)
	sc.Buffer(make([]byte, 1<<20), 1<<20)
	out := bufio.NewWriter(os.Stdout)
	for sc.Scan() {
		var req struct {
			ID     json.RawMessage `json:"id"`
			Method string          `json:"method"`
			Params json.RawMessage `json:"params"`
		}
		if json.Unmarshal(sc.Bytes(), &req) != nil || len(req.ID) == 0 {
			continue
		}
		params := req.Params
		if len(params) == 0 {
			params = json.RawMessage("null")
		}
		var result any
		switch req.Method {
		case MethodInitialize:
			result = map[string]any{
				"protocolVersion": "2024-11-05",
				"capabilities":    map[string]any{"tools": map[string]any{}},
				"serverInfo":      map[string]any{"name": "hygiene-helper", "version": "1"},
			}
		case MethodToolsList:
			result = map[string]any{"tools": []any{map[string]any{
				"name": "report", "description": "reports what the upstream received",
				"inputSchema": map[string]any{"type": "object"},
			}}}
		case MethodToolsCall:
			text, _ := json.Marshal(map[string]any{"env": os.Environ(), "params": params})
			result = map[string]any{"content": []any{map[string]any{"type": "text", "text": string(text)}}}
		default:
			result = map[string]any{"echo": params}
		}
		line, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": req.ID, "result": result})
		out.Write(line)
		out.WriteByte('\n')
		out.Flush()
	}
	os.Exit(0)
}

// helperUpstream is the config of a stdio upstream run by this test binary.
func helperUpstream(env map[string]string, passthrough []string, forward bool) UpstreamConfig {
	e := map[string]string{hygieneHelperEnv: "1"}
	for k, v := range env {
		e[k] = v
	}
	return UpstreamConfig{
		Transport:          "stdio",
		Command:            []string{os.Args[0], "-test.run=^TestHygieneHelperUpstream$"},
		Timeout:            20 * time.Second,
		Env:                e,
		EnvPassthrough:     passthrough,
		ForwardCallerToken: forward,
	}
}

// helperReport is what the "report" tool said it received.
type helperReport struct {
	Env    []string        `json:"env"`
	Params json.RawMessage `json:"params"`
}

func parseHelperReport(t *testing.T, result json.RawMessage) helperReport {
	t.Helper()
	var r struct {
		Content []struct {
			Text string `json:"text"`
		} `json:"content"`
	}
	if err := json.Unmarshal(result, &r); err != nil || len(r.Content) == 0 {
		t.Fatalf("tool result %s: %v", result, err)
	}
	var rep helperReport
	if err := json.Unmarshal([]byte(r.Content[0].Text), &rep); err != nil {
		t.Fatalf("report %q: %v", r.Content[0].Text, err)
	}
	return rep
}

// metaTokenOf returns params._meta.token as the upstream received it, and
// whether any spelling of it was there.
func metaTokenOf(t *testing.T, params json.RawMessage) (string, bool) {
	t.Helper()
	var top map[string]json.RawMessage
	if json.Unmarshal(params, &top) != nil {
		return "", false
	}
	for k, raw := range top {
		if !strings.EqualFold(k, "_meta") {
			continue
		}
		var meta map[string]json.RawMessage
		if json.Unmarshal(raw, &meta) != nil {
			continue
		}
		for mk, v := range meta {
			if strings.EqualFold(mk, "token") {
				var s string
				_ = json.Unmarshal(v, &s)
				return s, true
			}
		}
	}
	return "", false
}

// --- 1. Least-privilege environment -----------------------------------------

// A stdio upstream that prints its environment sees only the base list, the
// names its config passes through and its own env map: no other variable of
// the proxy's environment.
func TestStdioUpstreamSeesOnlyAllowedEnv(t *testing.T) {
	t.Setenv("SATGATE_HYGIENE_PROXY_ONLY", "proxy-only-value")
	t.Setenv("SATGATE_HYGIENE_PASSED", "passed-value")
	t.Setenv("PATH", os.Getenv("PATH"))

	cfg := helperUpstream(map[string]string{"SATGATE_HYGIENE_CONFIGURED": "configured-value"},
		[]string{"SATGATE_HYGIENE_PASSED", "SATGATE_HYGIENE_NOT_SET"}, false)
	m := NewUpstreamManager(map[string]UpstreamConfig{"envprobe": cfg}, nil, "envprobe")
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	if err := m.Start(ctx); err != nil {
		t.Fatalf("start helper upstream: %v", err)
	}
	defer m.Close()

	resp, err := m.ForwardToolCall(ctx, "report", json.RawMessage(`{"name":"report","arguments":{}}`), 20*time.Second)
	if err != nil || resp.Error != nil {
		t.Fatalf("report call: %v %+v", err, resp)
	}
	rep := parseHelperReport(t, resp.Result)

	allowed := map[string]bool{}
	for _, n := range stdioBaseEnv {
		allowed[n] = true
	}
	for _, n := range cfg.EnvPassthrough {
		allowed[n] = true
	}
	for n := range cfg.Env {
		allowed[n] = true
	}
	got := map[string]string{}
	for _, kv := range rep.Env {
		k, v, _ := strings.Cut(kv, "=")
		got[k] = v
		if !allowed[k] {
			t.Errorf("upstream received %s, which is neither in the base list nor configured", k)
		}
	}
	if _, ok := got["SATGATE_HYGIENE_PROXY_ONLY"]; ok {
		t.Errorf("a variable of the proxy's environment reached the upstream")
	}
	if got["SATGATE_HYGIENE_PASSED"] != "passed-value" {
		t.Errorf("passthrough variable: got %q", got["SATGATE_HYGIENE_PASSED"])
	}
	if _, ok := got["SATGATE_HYGIENE_NOT_SET"]; ok {
		t.Errorf("an unset passthrough name was given a value")
	}
	if got["SATGATE_HYGIENE_CONFIGURED"] != "configured-value" {
		t.Errorf("env map variable: got %q", got["SATGATE_HYGIENE_CONFIGURED"])
	}
	if got["PATH"] != os.Getenv("PATH") {
		t.Errorf("PATH not passed from the base list")
	}
}

func TestStdioEnvList(t *testing.T) {
	proxyEnv := map[string]string{
		"PATH": "/bin", "HOME": "/home/x", "TZ": "UTC",
		"ROOT_KEY": "not-for-upstreams", "DATABASE_URL": "not-for-upstreams",
		"API_KEY": "passed", "OVERRIDE": "from-proxy",
	}
	lookup := func(k string) (string, bool) { v, ok := proxyEnv[k]; return v, ok }
	cfg := UpstreamConfig{
		Env:            map[string]string{"OVERRIDE": "from-config", "HOME": "/srv/tool"},
		EnvPassthrough: []string{"API_KEY", "OVERRIDE", "MISSING", "", "BAD=NAME"},
	}
	got := stdioEnv(cfg, lookup)
	want := []string{"API_KEY=passed", "HOME=/srv/tool", "OVERRIDE=from-config", "PATH=/bin", "TZ=UTC"}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("stdioEnv = %q, want %q", got, want)
	}
	// Nothing configured and nothing in the base list set: an empty list, not
	// nil (a nil exec.Cmd.Env means the whole environment of the proxy).
	if env := stdioEnv(UpstreamConfig{}, func(string) (string, bool) { return "", false }); env == nil || len(env) != 0 {
		t.Fatalf("empty stdioEnv = %#v, want an empty non-nil list", env)
	}
	if !sort.StringsAreSorted(got) {
		t.Fatalf("stdioEnv not sorted: %q", got)
	}
}

// --- 2. Caller tokens stay with the proxy -----------------------------------

// tokenProbe is a proxy in header mode with one stdio helper upstream,
// served over SSE and Streamable HTTP.
type tokenProbe struct {
	srv   *httptest.Server
	proxy *Proxy
}

func newTokenProbe(t *testing.T, forward bool) *tokenProbe {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "hygiene-test", Version: "test"},
		Auth:        AuthConfig{Mode: "header", RootKey: "hygiene-test-root"},
		Upstreams:   map[string]UpstreamConfig{"probe": helperUpstream(nil, nil, forward)},
		Budget:      BudgetConfig{Limit: 1000, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "observe"},
		Logging:     LoggingConfig{Level: "error"},
	}
	cfg.applyDefaults()
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetAuthenticator(fixedTokenAuth{good: hygieneToken})
	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	if err := proxy.upstream.Start(ctx); err != nil {
		t.Fatalf("start helper upstream: %v", err)
	}
	t.Cleanup(func() { _ = proxy.upstream.Close() })
	srv := httptest.NewServer(NewSSEServer(proxy, "").Handler())
	t.Cleanup(srv.Close)
	return &tokenProbe{srv: srv, proxy: proxy}
}

// The calls each transport makes: a tools/call with no _meta, a tools/call
// whose _meta the caller filled (with the token in two spellings and a
// progress token), and a method passed to the default upstream as it is.
func hygieneCalls() []map[string]any {
	return []map[string]any{
		{"jsonrpc": "2.0", "id": 11, "method": MethodToolsCall,
			"params": map[string]any{"name": "report", "arguments": map[string]any{}}},
		{"jsonrpc": "2.0", "id": 12, "method": MethodToolsCall,
			"params": map[string]any{"name": "report", "arguments": map[string]any{},
				"_meta": map[string]any{"token": hygieneToken, "progressToken": 7}}},
		{"jsonrpc": "2.0", "id": 13, "method": MethodToolsCall,
			"params": map[string]any{"name": "report", "arguments": map[string]any{},
				"_META": map[string]any{"TOKEN": hygieneToken}}},
		{"jsonrpc": "2.0", "id": 14, "method": "resources/list", "params": map[string]any{}},
	}
}

// receivedParams is the params the upstream reports for resp.
func receivedParams(t *testing.T, method string, resp *Response) json.RawMessage {
	t.Helper()
	if resp == nil || resp.Error != nil {
		t.Fatalf("%s: error response %+v", method, resp)
	}
	if method == MethodToolsCall {
		return parseHelperReport(t, resp.Result).Params
	}
	var r struct {
		Echo json.RawMessage `json:"echo"`
	}
	if err := json.Unmarshal(resp.Result, &r); err != nil {
		t.Fatalf("%s result %s: %v", method, resp.Result, err)
	}
	return r.Echo
}

func checkUpstreamToken(t *testing.T, transport string, forward bool, call map[string]any, params json.RawMessage) {
	t.Helper()
	tok, has := metaTokenOf(t, params)
	switch {
	case !forward && has:
		t.Errorf("%s id=%v: upstream received _meta.token by default: %s", transport, call["id"], redactToken(params))
	case forward && (!has || tok != hygieneToken):
		t.Errorf("%s id=%v: opted-in upstream did not receive the caller's token: %s", transport, call["id"], redactToken(params))
	}
	if call["id"] == 12 && !strings.Contains(string(params), `"progressToken":7`) {
		t.Errorf("%s: the rest of _meta was not passed on: %s", transport, redactToken(params))
	}
}

func redactToken(b json.RawMessage) string {
	return strings.ReplaceAll(string(b), hygieneToken, "[caller-token]")
}

func TestUpstreamGetsNoCallerTokenSSE(t *testing.T) {
	for _, forward := range []bool{false, true} {
		p := newTokenProbe(t, forward)
		conn := eaOpenSSE(t, p.srv.URL, hygieneToken)
		for _, call := range hygieneCalls() {
			if status := conn.post(mustJSON(t, call)); status != http.StatusAccepted {
				t.Fatalf("POST /message status %d", status)
			}
			params := receivedParams(t, call["method"].(string), conn.next())
			checkUpstreamToken(t, "sse", forward, call, params)
		}
		_ = conn.body.Close()
	}
}

func TestUpstreamGetsNoCallerTokenStreamable(t *testing.T) {
	for _, forward := range []bool{false, true} {
		p := newTokenProbe(t, forward)
		sid := mustInitSession(t, p.srv.URL, hygieneToken)
		for _, call := range hygieneCalls() {
			status, _, body := streamablePost(t, p.srv.URL+"/mcp", sid, hygieneToken, call)
			if status != http.StatusOK {
				t.Fatalf("streamable status %d: %s", status, redactToken(json.RawMessage(body)))
			}
			var resp Response
			if err := json.Unmarshal([]byte(body), &resp); err != nil {
				t.Fatalf("streamable body %q: %v", redactToken(json.RawMessage(body)), err)
			}
			params := receivedParams(t, call["method"].(string), &resp)
			checkUpstreamToken(t, "streamable", forward, call, params)
		}
	}
}

func TestWithoutMetaToken(t *testing.T) {
	cases := []struct{ name, in, want string }{
		{"no _meta: same bytes", `{"name":"a", "arguments":{"x":1}}`, `{"name":"a", "arguments":{"x":1}}`},
		{"only the token: _meta dropped, order and bytes kept", `{"name":"a","arguments":{"b":2, "a":1.50},"_meta":{"token":"t"}}`, `{"name":"a","arguments":{"b":2, "a":1.50}}`},
		{"other _meta members kept", `{"_meta":{"progressToken":7,"token":"t"},"name":"a"}`, `{"_meta":{"progressToken":7},"name":"a"}`},
		{"any spelling", `{"name":"a","_Meta":{"ToKeN":"t","x":1}}`, `{"name":"a","_Meta":{"x":1}}`},
		{"kelvin sign", `{"_meta":{"to\u212aen":"t"}}`, `{}`},
		{"escaped name", `{"\u005fmeta":{"t\u006fken":"t"},"name":"a"}`, `{"name":"a"}`},
		{"repeated _meta: every copy cleaned", `{"_meta":{"token":"t1"},"name":"a","_meta":{"token":"t2","x":1}}`, `{"name":"a","_meta":{"x":1}}`},
		{"_meta not an object", `{"_meta":"t","name":"a"}`, `{"_meta":"t","name":"a"}`},
		{"not an object", `[{"_meta":{"token":"t"}}]`, `[{"_meta":{"token":"t"}}]`},
		{"null", `null`, `null`},
		{"token nested in arguments is the tool's", `{"arguments":{"_meta":{"token":"x"}}}`, `{"arguments":{"_meta":{"token":"x"}}}`},
	}
	for _, c := range cases {
		if got := string(withoutMetaToken(json.RawMessage(c.in))); got != c.want {
			t.Errorf("%s: got %s, want %s", c.name, got, c.want)
		}
	}
	if got := withoutMetaToken(nil); got != nil {
		t.Errorf("nil params: got %q", got)
	}
}

// --- 3. Token in the URL ----------------------------------------------------

func newQueryTokenServer(t *testing.T, allow bool) *httptest.Server {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "hygiene-query", Version: "test", AllowQueryToken: allow},
		Auth:        AuthConfig{Mode: "header", RootKey: "hygiene-query-root"},
		Budget:      BudgetConfig{Limit: 1000, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "observe"},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	proxy.SetAuthenticator(fixedTokenAuth{good: hygieneToken})
	proxy.SetUpstreamRouter(&countingForwardRouter{})
	srv := httptest.NewServer(NewSSEServer(proxy, "").Handler())
	t.Cleanup(srv.Close)
	return srv
}

func TestQueryTokenRefusedByDefault(t *testing.T) {
	srv := newQueryTokenServer(t, false)
	q := "?token=" + hygieneToken

	// Streamable HTTP: initialize with the token only in the URL.
	if status, hdr, _ := streamablePost(t, srv.URL+"/mcp"+q, "", "", initializeBody("2025-03-26")); status != http.StatusUnauthorized || hdr.Get("Mcp-Session-Id") != "" {
		t.Fatalf("streamable ?token= initialize: status %d, session %q; want 401 and no session", status, hdr.Get("Mcp-Session-Id"))
	}
	// Also refused next to a header: the URL still carries the token.
	if status, _, _ := streamablePost(t, srv.URL+"/mcp"+q, "", hygieneToken, initializeBody("2025-03-26")); status != http.StatusUnauthorized {
		t.Fatalf("streamable ?token= with header: status %d, want 401", status)
	}
	// Legacy SSE connect and message.
	resp, err := http.Get(srv.URL + "/sse" + q)
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("GET /sse?token=: status %d, want 401", resp.StatusCode)
	}
	conn := eaOpenSSE(t, srv.URL, hygieneToken)
	defer conn.body.Close()
	resp, err = http.Post(srv.URL+"/message?sessionId="+conn.session+"&token="+hygieneToken, "application/json",
		strings.NewReader(string(mustJSON(t, toolCallBody(2, "echo")))))
	if err != nil {
		t.Fatal(err)
	}
	resp.Body.Close()
	if resp.StatusCode != http.StatusUnauthorized {
		t.Fatalf("POST /message?token=: status %d, want 401", resp.StatusCode)
	}
	// The header alone still works.
	_ = mustInitSession(t, srv.URL, hygieneToken)
}

func TestQueryTokenAcceptedWithFlag(t *testing.T) {
	srv := newQueryTokenServer(t, true)
	q := "?token=" + hygieneToken

	status, hdr, body := streamablePost(t, srv.URL+"/mcp"+q, "", "", initializeBody("2025-03-26"))
	if status != http.StatusOK || hdr.Get("Mcp-Session-Id") == "" {
		t.Fatalf("streamable ?token= initialize with the flag: status %d body %s", status, redactToken(json.RawMessage(body)))
	}
	status, _, body = streamablePost(t, srv.URL+"/mcp"+q, hdr.Get("Mcp-Session-Id"), "", toolCallBody(2, "echo"))
	var r Response
	if status != http.StatusOK || json.Unmarshal([]byte(body), &r) != nil || r.Error != nil {
		t.Fatalf("streamable ?token= tools/call with the flag: status %d body %s", status, redactToken(json.RawMessage(body)))
	}

	resp, err := http.Get(srv.URL + "/sse" + q)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		t.Fatalf("GET /sse?token= with the flag: status %d", resp.StatusCode)
	}
}
