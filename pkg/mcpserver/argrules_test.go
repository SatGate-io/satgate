package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// recordingRouter is an upstream stand-in that records every tools/call it is
// handed, with the exact params it received.
type recordingRouter struct {
	mu     sync.Mutex
	params []string
}

func (r *recordingRouter) AllToolsForTenant(context.Context, string) []json.RawMessage {
	return nil
}

func (r *recordingRouter) ForwardToolCallForTenant(_ context.Context, _ string, _ string, params json.RawMessage, _ time.Duration) (*Response, error) {
	r.mu.Lock()
	r.params = append(r.params, string(params))
	r.mu.Unlock()
	return &Response{JSONRPC: "2.0", Result: json.RawMessage(`{"content":[{"type":"text","text":"ok"}],"isError":false}`)}, nil
}

func (r *recordingRouter) calls() int {
	r.mu.Lock()
	defer r.mu.Unlock()
	return len(r.params)
}

const argTestRoot = "argrules-test-root-key"

// Preset documents as the dashboard writes them. REPORT.md cites the public
// documentation each one was checked against.
const (
	wixReadOnlyDoc = `{"v":1,"rules":[
	 {"tool":"CallWixSiteAPI","shapes":[
	  {"method":{"one_of":["GET"],"ascii_case_insensitive":true}},
	  {"method":{"one_of":["POST"],"ascii_case_insensitive":true},"url":{"url":{"hosts":["www.wixapis.com"],"path_suffixes":["/query","/search"]}}}]},
	 {"tool":"ManageWixSite","shapes":[
	  {"method":{"one_of":["GET"],"ascii_case_insensitive":true}}]}]}`
	squareReadOnlyDoc = `{"v":1,"rules":[{"tool":"make_api_request","shapes":[
	  {"method":{"one_of":["list","get","search","info"]}}]}]}`
	robinhoodCapDoc = `{"v":1,"rules":[{"tool":"place_equity_order","shapes":[
	  {"type":{"one_of":["limit"]},"quantity":{"max":"10"},"dollar_amount":{"max":"500"}}]}]}`
)

func ruleScopeWord(t *testing.T, doc string) string {
	t.Helper()
	rules, err := argrules.ParseJSON([]byte(doc))
	if err != nil {
		t.Fatal(err)
	}
	v, err := argrules.ScopeValue(rules)
	if err != nil {
		t.Fatal(err)
	}
	return "scope = " + v
}

func newArgProxy(t *testing.T) (*Proxy, *recordingRouter, *ChannelPublisher) {
	t.Helper()
	cfg := &Config{
		Server:      ServerConfig{Transport: "sse", Name: "argrules-test", Version: "test"},
		Auth:        AuthConfig{Mode: "header", RootKey: argTestRoot},
		Budget:      BudgetConfig{Limit: 100, FailMode: "closed"},
		Tools:       ToolsConfig{DefaultCost: 1},
		Enforcement: EnforcementConfig{Mode: "control"},
		Logging:     LoggingConfig{Level: "error"},
	}
	proxy, err := New(cfg)
	if err != nil {
		t.Fatal(err)
	}
	rr := &recordingRouter{}
	proxy.SetUpstreamRouter(rr)
	events := NewChannelPublisher(256)
	proxy.SetEventPublisher(events)
	return proxy, rr, events
}

// argToken mints a token for the given tools and appends the rule documents
// the way a holder narrowing the token would.
func argToken(t *testing.T, scope string, docs ...string) (*macaroon.Service, string) {
	t.Helper()
	svc, tok := mintBudgetToken(t, argTestRoot, scope, "", "budget-arg", "50")
	for _, d := range docs {
		tok = appendCaveat(t, svc, tok, ruleScopeWord(t, d))
	}
	return svc, tok
}

func argCall(id int, token, tool, args string) *Request {
	params := `{"name":"` + tool + `"`
	if args != "" {
		params += `,"arguments":` + args
	}
	params += `,"_meta":{"token":"` + token + `"}}`
	return &Request{JSONRPC: "2.0", ID: json.RawMessage(mustJSONString(id)), Method: MethodToolsCall, Params: json.RawMessage(params)}
}

type argOutcome struct {
	denied bool
	resp   *Response
}

func runCall(t *testing.T, p *Proxy, req *Request) argOutcome {
	t.Helper()
	resp, err := p.handleRequest(context.Background(), req)
	if err != nil || resp == nil {
		t.Fatalf("handleRequest: resp=%v err=%v", resp, err)
	}
	return argOutcome{denied: resp.Error != nil, resp: resp}
}

func assertArgDenied(t *testing.T, o argOutcome, tool, field string) {
	t.Helper()
	if o.resp.Error == nil {
		t.Fatalf("call was not refused")
	}
	if o.resp.Error.Code != CodePolicyDenied {
		t.Fatalf("code = %d, want %d", o.resp.Error.Code, CodePolicyDenied)
	}
	var data map[string]any
	if err := json.Unmarshal(o.resp.Error.Data, &data); err != nil {
		t.Fatalf("error data: %v (%s)", err, o.resp.Error.Data)
	}
	if data["error"] != argrules.DenialCode || data["tool"] != tool {
		t.Fatalf("error data = %v", data)
	}
	if field != "" && data["field"] != field {
		t.Fatalf("field = %v, want %s", data["field"], field)
	}
	if !strings.Contains(o.resp.Error.Message, tool) {
		t.Fatalf("message %q does not name the tool", o.resp.Error.Message)
	}
}

type argCase struct {
	name  string
	tool  string
	args  string
	allow bool
	field string
}

func runArgCases(t *testing.T, scope, doc string, cases []argCase) {
	t.Helper()
	proxy, rr, events := newArgProxy(t)
	_, tok := argToken(t, scope, doc)
	for i, c := range cases {
		before := rr.calls()
		remBefore := remainingFor2(t, proxy, tok)
		_ = drainEvents(events)
		o := runCall(t, proxy, argCall(i+1, tok, c.tool, c.args))
		after := rr.calls()
		if c.allow {
			if o.denied || after != before+1 {
				t.Errorf("%s: want pass; denied=%v upstream calls %d->%d resp=%+v", c.name, o.denied, before, after, o.resp.Error)
			}
			continue
		}
		if !o.denied {
			t.Errorf("%s: want refusal, call passed", c.name)
			continue
		}
		if after != before {
			t.Errorf("%s: refused but upstream was contacted", c.name)
		}
		if got := remainingFor2(t, proxy, tok); got != remBefore {
			t.Errorf("%s: refused call changed budget %d -> %d", c.name, remBefore, got)
		}
		for _, e := range drainEvents(events) {
			if e.Type == EventToolCall || e.Type == EventBudgetSpend {
				t.Errorf("%s: refused call published %s", c.name, e.Type)
			}
		}
		func() {
			defer func() {
				if r := recover(); r != nil {
					t.Errorf("%s: %v", c.name, r)
				}
			}()
			assertArgDenied(t, o, c.tool, c.field)
		}()
	}
}

func remainingFor2(t *testing.T, p *Proxy, token string) int64 {
	t.Helper()
	info, err := p.auth.Verify(context.Background(), token)
	if err != nil {
		t.Fatal(err)
	}
	rem, err := p.budget.Remaining(context.Background(), info.BudgetID)
	if err != nil {
		return -1
	}
	return rem
}

func TestWixReadOnlyPresetEndToEnd(t *testing.T) {
	// ExecuteWixAPI runs scripts and cannot carry a rule (B1); it is not here.
	const scope = "CallWixSiteAPI,ManageWixSite,ListWixSites"
	doc := wixReadOnlyDoc
	site := "https://www.wixapis.com/stores/v1/products"
	cases := []argCase{
		{"GET reads", "CallWixSiteAPI", `{"method":"GET","url":"` + site + `/abc"}`, true, ""},
		{"get lower case", "CallWixSiteAPI", `{"method":"get","url":"` + site + `/abc"}`, true, ""},
		{"POST query reads", "CallWixSiteAPI", `{"method":"POST","url":"` + site + `/query","body":"{}"}`, true, ""},
		{"ListWixSites has no rule", "ListWixSites", `{}`, true, ""},
		{"POST write path", "CallWixSiteAPI", `{"method":"POST","url":"` + site + `","body":"{}"}`, false, "url"},
		{"POST delete path", "CallWixSiteAPI", `{"method":"POST","url":"` + site + `/abc/delete"}`, false, "url"},
		{"POST query lookalike", "CallWixSiteAPI", `{"method":"POST","url":"` + site + `/myquery"}`, false, "url"},
		{"POST query in query string", "CallWixSiteAPI", `{"method":"POST","url":"` + site + `/create?x=/query"}`, false, "url"},
		{"POST dotdot to query", "CallWixSiteAPI", `{"method":"POST","url":"` + site + `/../delete/../query"}`, false, "url"},
		{"POST encoded slash", "CallWixSiteAPI", `{"method":"POST","url":"` + site + `/x%2Fquery"}`, false, "url"},
		{"POST other host", "CallWixSiteAPI", `{"method":"POST","url":"https://evil.example/stores/v1/products/query"}`, false, "url"},
		{"POST relative", "CallWixSiteAPI", `{"method":"POST","url":"/stores/v1/products/query"}`, false, "url"},
		{"DELETE", "CallWixSiteAPI", `{"method":"DELETE","url":"` + site + `/abc"}`, false, "method"},
		{"PATCH", "CallWixSiteAPI", `{"method":"PATCH","url":"` + site + `/abc","body":"{}"}`, false, "method"},
		{"method missing", "CallWixSiteAPI", `{"url":"` + site + `/abc"}`, false, "method"},
		{"method not a string", "CallWixSiteAPI", `{"method":["GET"],"url":"` + site + `/abc"}`, false, "method"},
		{"no arguments", "CallWixSiteAPI", ``, false, "method"},
		{"ManageWixSite publish", "ManageWixSite", `{"method":"POST","url":"https://www.wixapis.com/site-actions/v1/actions/publish","body":"{}"}`, false, "method"},
		{"ManageWixSite create", "ManageWixSite", `{"method":"POST","url":"https://www.wixapis.com/site-list/v2/sites","body":"{}"}`, false, "method"},
		{"ManageWixSite GET", "ManageWixSite", `{"method":"GET","url":"https://www.wixapis.com/site-list/v2/sites"}`, true, ""},
	}
	runArgCases(t, scope, doc, cases)
}

func TestSquareReadOnlyPresetEndToEnd(t *testing.T) {
	runArgCases(t, "make_api_request", squareReadOnlyDoc, []argCase{
		{"list", "make_api_request", `{"service":"customers","method":"list","request":{}}`, true, ""},
		{"search", "make_api_request", `{"service":"orders","method":"search","request":{}}`, true, ""},
		{"get", "make_api_request", `{"service":"payments","method":"get","request":{"payment_id":"x"}}`, true, ""},
		{"create", "make_api_request", `{"service":"payments","method":"create","request":{}}`, false, "method"},
		{"update", "make_api_request", `{"service":"customers","method":"update","request":{}}`, false, "method"},
		{"delete", "make_api_request", `{"service":"customers","method":"delete","request":{}}`, false, "method"},
		{"List capitalised is not list", "make_api_request", `{"service":"customers","method":"List","request":{}}`, false, "method"},
		{"missing method", "make_api_request", `{"service":"customers"}`, false, "method"},
	})
}

func TestRobinhoodOrderCapPresetEndToEnd(t *testing.T) {
	runArgCases(t, "place_equity_order", robinhoodCapDoc, []argCase{
		{"within caps", "place_equity_order", `{"symbol":"X","type":"limit","quantity":"1.5","dollar_amount":"100.25"}`, true, ""},
		{"at caps", "place_equity_order", `{"type":"limit","quantity":"10","dollar_amount":"500"}`, true, ""},
		{"quantity just over", "place_equity_order", `{"type":"limit","quantity":"10.000001","dollar_amount":"1"}`, false, "quantity"},
		{"dollar just over", "place_equity_order", `{"type":"limit","quantity":"1","dollar_amount":"500.01"}`, false, "dollar_amount"},
		{"market order", "place_equity_order", `{"type":"market","quantity":"1","dollar_amount":"1"}`, false, "type"},
		{"exponent", "place_equity_order", `{"type":"limit","quantity":"1e1","dollar_amount":"1"}`, false, "quantity"},
		{"leading plus", "place_equity_order", `{"type":"limit","quantity":"+1","dollar_amount":"1"}`, false, "quantity"},
		{"quantity missing", "place_equity_order", `{"type":"limit","dollar_amount":"1"}`, false, "quantity"},
		{"dollar missing", "place_equity_order", `{"type":"limit","quantity":"1"}`, false, "dollar_amount"},
		{"non numeric", "place_equity_order", `{"type":"limit","quantity":"lots","dollar_amount":"1"}`, false, "quantity"},
	})
}

// A denial never echoes an argument value, however it is named.
func TestArgumentDenialDoesNotEchoValues(t *testing.T) {
	proxy, _, _ := newArgProxy(t)
	_, tok := argToken(t, "CallWixSiteAPI,ManageWixSite", wixReadOnlyDoc)
	o := runCall(t, proxy, argCall(1, tok, "CallWixSiteAPI", `{"method":"DELETE","url":"https://www.wixapis.com/contacts/v4/contacts/jane.doe@example.com","body":"{\"secret\":\"CUSTOMER-DATA-123\"}"}`))
	raw, _ := json.Marshal(o.resp)
	for _, leak := range []string{"DELETE", "jane.doe", "CUSTOMER-DATA", "contacts/v4"} {
		if strings.Contains(string(raw), leak) {
			t.Fatalf("denial leaks %q: %s", leak, raw)
		}
	}
}

// Duplicate keys: the rule check reads the same arguments the upstream gets,
// so an object whose keys repeat is refused in either order.
func TestDuplicateKeysRefusedBothOrdersAtProxy(t *testing.T) {
	const scope = "CallWixSiteAPI,ManageWixSite"
	site := "https://www.wixapis.com/stores/v1/products"
	runArgCases(t, scope, wixReadOnlyDoc, []argCase{
		{"GET then DELETE", "CallWixSiteAPI", `{"method":"GET","method":"DELETE","url":"` + site + `"}`, false, ""},
		{"DELETE then GET", "CallWixSiteAPI", `{"method":"DELETE","method":"GET","url":"` + site + `"}`, false, ""},
		{"GET then GET", "CallWixSiteAPI", `{"method":"GET","method":"GET","url":"` + site + `"}`, false, ""},
		{"escaped duplicate", "CallWixSiteAPI", `{"method":"GET","m\u0065thod":"DELETE","url":"` + site + `"}`, false, ""},
		{"nested duplicate", "CallWixSiteAPI", `{"method":"GET","url":"` + site + `","body":{"a":1,"a":2}}`, false, ""},
	})
	// Unchanged bytes are what the upstream receives on a pass.
	proxy, rr, _ := newArgProxy(t)
	_, tok := argToken(t, scope, wixReadOnlyDoc)
	req := argCall(1, tok, "CallWixSiteAPI", `{"method":"GET","url":"`+site+`"}`)
	want := string(req.Params)
	if o := runCall(t, proxy, req); o.denied {
		t.Fatalf("allowed call refused: %+v", o.resp.Error)
	}
	if len(rr.params) != 1 || rr.params[0] != want {
		t.Fatalf("upstream got %v, want the checked bytes %s", rr.params, want)
	}
}

// Two rules for one tool are ANDed; a rule cannot be widened by appending one.
func TestAttenuationAndsRulesAndChildCannotWiden(t *testing.T) {
	parentDoc := `{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET","POST"]}}]}]}`
	childDoc := `{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET","DELETE"]}}]}]}`
	proxy, rr, _ := newArgProxy(t)
	svc, tok := argToken(t, "CallWixSiteAPI", parentDoc)

	try := func(token, method string) bool {
		before := rr.calls()
		o := runCall(t, proxy, argCall(1, token, "CallWixSiteAPI", `{"method":"`+method+`"}`))
		if o.denied && rr.calls() != before {
			t.Fatalf("refused call reached the upstream")
		}
		return !o.denied
	}
	if !try(tok, "GET") || !try(tok, "POST") || try(tok, "DELETE") {
		t.Fatal("parent behaves wrongly")
	}
	child := appendCaveat(t, svc, tok, ruleScopeWord(t, childDoc))
	if !try(child, "GET") {
		t.Error("child lost GET, which both rules allow")
	}
	if try(child, "POST") {
		t.Error("child still allows POST that its own rule removed")
	}
	if try(child, "DELETE") {
		t.Error("child gained DELETE the parent never allowed")
	}
	// A wider child rule (every method) does not loosen the parent's.
	wide := `{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET","POST","DELETE","PATCH","PUT"]}}]}]}`
	widened := appendCaveat(t, svc, tok, ruleScopeWord(t, wide))
	if try(widened, "DELETE") || try(widened, "PATCH") {
		t.Error("a child rule widened the parent")
	}
	if !try(widened, "POST") {
		t.Error("child rule narrowed more than it should")
	}
	// A child cannot drop the rule by a later scope caveat.
	rewider := appendCaveat(t, svc, tok, "scope = CallWixSiteAPI")
	if try(rewider, "DELETE") {
		t.Error("repeating the scope caveat dropped the rule")
	}
}

// A rule for a tool outside the scope, or one that does not parse, makes the
// token unusable at verify.
func TestTokenWithBadRulesRefusedAtVerify(t *testing.T) {
	proxy, rr, _ := newArgProxy(t)
	good := `{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET"]}}]}]}`
	svc, tok := argToken(t, "CallWixSiteAPI", good)
	auth := &MacaroonAuthenticator{Service: svc}
	if _, err := auth.Verify(context.Background(), tok); err != nil {
		t.Fatalf("good token refused: %v", err)
	}
	other := `{"v":1,"rules":[{"tool":"ManageWixSite","shapes":[{"method":{"one_of":["GET"]}}]}]}`
	cases := map[string]string{
		"rule for tool outside scope": appendCaveat(t, svc, tok, ruleScopeWord(t, other)),
		"garbage payload":             appendCaveat(t, svc, tok, "scope = argrules:v1:!!!"),
		"unknown version":             appendCaveat(t, svc, tok, "scope = argrules:v9:e30"),
		"unknown condition": appendCaveat(t, svc, tok, "scope = argrules:v1:"+
			b64(`{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"regex":"."}}]}]}`)),
		"rule word in a tool list": appendCaveat(t, svc, tok, "scope = CallWixSiteAPI,"+strings.TrimPrefix(ruleScopeWord(t, good), "scope = ")),
	}
	for name, bad := range cases {
		if _, err := auth.Verify(context.Background(), bad); err == nil {
			t.Errorf("%s: verified", name)
		}
		o := runCall(t, proxy, argCall(1, bad, "CallWixSiteAPI", `{"method":"GET"}`))
		if !o.denied || rr.calls() != 0 {
			t.Errorf("%s: call went through (denied=%v calls=%d)", name, o.denied, rr.calls())
		}
	}
}

func b64(s string) string {
	return strings.TrimRight(base64URL(s), "=")
}

// The old verifier. oldMatchScope and oldAllowsTool are copied unchanged from
// pkg/mcpserver/proxy.go (matchScope) and pkg/mcpserver/caveat_policy.go
// (TokenInfo.AllowsTool) as they stand at the module version enterprise pins
// (a2333137) and at public main (fdfc80fc); the two files are byte-identical at
// those commits. Only the names and the receiver changed.
func oldMatchScope(scope, toolName string) bool {
	for _, s := range strings.Split(scope, ",") {
		s = strings.TrimSpace(s)

		// Strip tenant UUID prefix if present (format: "uuid:scope")
		// UUIDs are 36 chars with hyphens (8-4-4-4-12)
		if len(s) > 37 && s[36] == ':' && s[8] == '-' && s[13] == '-' {
			s = s[37:]
		}

		if s == "*" || s == "api:*" || s == "mcp:*" {
			return true
		}
		if strings.HasSuffix(s, ":*") {
			prefix := strings.TrimSuffix(s, ":*")
			if strings.HasPrefix(toolName, prefix+":") || strings.HasPrefix(toolName, prefix+"_") {
				return true
			}
		}
		if s == toolName {
			return true
		}
	}
	return false
}

type oldTokenInfo struct {
	Scope string
	Raw   *macaroon.Macaroon
}

func (t *oldTokenInfo) AllowsTool(toolName string) bool {
	if t == nil {
		return false
	}
	if t.Raw != nil {
		saw := false
		for _, caveat := range t.Raw.Caveats {
			if !strings.HasPrefix(caveat, "scope = ") {
				continue
			}
			saw = true
			if !oldMatchScope(strings.TrimPrefix(caveat, "scope = "), toolName) {
				return false
			}
		}
		if saw {
			return true
		}
	}
	if t.Scope == "" || t.Scope == "*" || t.Scope == "api:*" || t.Scope == "mcp:*" {
		return true
	}
	return oldMatchScope(t.Scope, toolName)
}

func TestOldVerifierDeniesRuleCarryingToken(t *testing.T) {
	const scope = "CallWixSiteAPI,ManageWixSite"
	svc, plainTok := argToken(t, scope)
	_, ruledTok := argToken(t, scope, wixReadOnlyDoc)
	plain, err := svc.Decode(plainTok)
	if err != nil {
		t.Fatal(err)
	}
	ruled, err := svc.Decode(ruledTok)
	if err != nil {
		t.Fatal(err)
	}
	// Control: the old code allows the tools on the token without rules.
	oldPlain := &oldTokenInfo{Raw: plain}
	if !oldPlain.AllowsTool("CallWixSiteAPI") || !oldPlain.AllowsTool("ManageWixSite") {
		t.Fatal("control failed: old AllowsTool should allow the token without rules")
	}
	// A rule-carrying token: the old code denies every tool, including the
	// ones the first scope caveat names. Only a tool named exactly like the whole
	// word (which holds a fresh nonce) could match; see the nonce tests.
	old := &oldTokenInfo{Scope: "CallWixSiteAPI,ManageWixSite", Raw: ruled}
	for _, tool := range []string{"CallWixSiteAPI", "ManageWixSite", "ExecuteWixAPI", "anything", "*", "mcp:*", ""} {
		if old.AllowsTool(tool) {
			t.Errorf("old AllowsTool allowed %q for a rule-carrying token", tool)
		}
	}
	// The old code ignores unknown caveat NAMES, which is why the rule is not
	// carried in one: prove that a rule in a new caveat name would have passed.
	viaNewName := appendCaveat(t, svc, plainTok, "arg_rules = "+strings.TrimPrefix(ruleScopeWord(t, wixReadOnlyDoc), "scope = "))
	m, err := svc.Decode(viaNewName)
	if err != nil {
		t.Fatal(err)
	}
	if !(&oldTokenInfo{Raw: m}).AllowsTool("CallWixSiteAPI") {
		t.Fatal("expected the old code to ignore an unknown caveat name (the reason a scope word is used)")
	}
	// Old Service.Verify (the macaroon package at the old version) accepts the
	// ruled token's signature; the denial comes from AllowsTool. Both layers are
	// shown: the old verify step does not save the day, the scope word does.
	// And the old matchScope applied to the bare rule word:
	for _, c := range ruled.Caveats {
		if strings.HasPrefix(c, "scope = argrules:") && oldMatchScope(strings.TrimPrefix(c, "scope = "), "CallWixSiteAPI") {
			t.Fatal("old matchScope matched the rule word")
		}
	}
}

// A verifier that did not opt in (HTTP routes, any other macaroon.Service
// user) refuses a rule-carrying token instead of ignoring the rule.
func TestDefaultServiceRefusesRuleCarryingToken(t *testing.T) {
	svc, tok := argToken(t, "CallWixSiteAPI,ManageWixSite", wixReadOnlyDoc)
	if _, err := svc.Verify(tok); err == nil || !strings.Contains(err.Error(), "argument rules") {
		t.Fatalf("default Verify = %v, want a refusal naming argument rules", err)
	}
	if _, err := svc.AcceptingArgumentRules().Verify(tok); err != nil {
		t.Fatalf("enforcing Verify refused: %v", err)
	}
	// Delegate (verifying) also refuses unless the caller enforces.
	if _, err := svc.Delegate(tok, []string{"expires = " + time.Now().Add(time.Minute).UTC().Format(time.RFC3339)}); err == nil {
		t.Fatal("default Delegate accepted a rule-carrying parent")
	}
	if !macaroon.IsArgumentRulesScope("a, argrules:v1:xx") || macaroon.IsArgumentRulesScope("a,b") {
		t.Fatal("IsArgumentRulesScope wrong")
	}
}

// Over HTTP: Streamable HTTP, the legacy /message route and the tool path all
// reach handleRequest; a refused call must not reach the upstream on any.
func TestStreamableHTTPRefusesAndPassesByArguments(t *testing.T) {
	proxy, rr, _ := newArgProxy(t)
	_, tok := argToken(t, "CallWixSiteAPI,ManageWixSite", wixReadOnlyDoc)
	srv := httptest.NewServer(NewSSEServer(proxy, ":0").Handler())
	defer srv.Close()

	sid := mustInitSession(t, srv.URL, tok)
	body := func(id int, args string) map[string]any {
		var a any
		if err := json.Unmarshal([]byte(args), &a); err != nil {
			t.Fatal(err)
		}
		return map[string]any{"jsonrpc": "2.0", "id": id, "method": "tools/call",
			"params": map[string]any{"name": "CallWixSiteAPI", "arguments": a}}
	}
	status, _, out := streamablePost(t, srv.URL+"/mcp", sid, tok, body(2, `{"method":"GET","url":"https://www.wixapis.com/stores/v1/products"}`))
	if status != http.StatusOK || strings.Contains(out, `"error"`) || rr.calls() != 1 {
		t.Fatalf("read: status=%d calls=%d body=%s", status, rr.calls(), out)
	}
	status, _, out = streamablePost(t, srv.URL+"/mcp", sid, tok, body(3, `{"method":"DELETE","url":"https://www.wixapis.com/stores/v1/products/1"}`))
	if status != http.StatusOK || !strings.Contains(out, argrules.DenialCode) || rr.calls() != 1 {
		t.Fatalf("delete: status=%d calls=%d body=%s", status, rr.calls(), out)
	}
	// A body with the duplicate key, sent as raw bytes the way a client could.
	raw := `{"jsonrpc":"2.0","id":4,"method":"tools/call","params":{"name":"CallWixSiteAPI","arguments":{"method":"DELETE","method":"GET","url":"https://www.wixapis.com/x"}}}`
	status, _, out = streamablePost(t, srv.URL+"/mcp", sid, tok, json.RawMessage(raw))
	if status != http.StatusOK || !strings.Contains(out, argrules.DenialCode) || rr.calls() != 1 {
		t.Fatalf("duplicate: status=%d calls=%d body=%s", status, rr.calls(), out)
	}
}

// Batches are refused by the Streamable HTTP route outright, so a refused
// call cannot ride in with an allowed one. A caller that feeds a batch to
// the legacy route gets each element judged on its own.
func TestBatchWithAllowedAndRefusedCall(t *testing.T) {
	proxy, rr, _ := newArgProxy(t)
	_, tok := argToken(t, "CallWixSiteAPI,ManageWixSite", wixReadOnlyDoc)
	srv := httptest.NewServer(NewSSEServer(proxy, ":0").Handler())
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, tok)

	batch := []any{
		map[string]any{"jsonrpc": "2.0", "id": 10, "method": "tools/call", "params": map[string]any{"name": "CallWixSiteAPI", "arguments": map[string]any{"method": "GET", "url": "https://www.wixapis.com/a"}}},
		map[string]any{"jsonrpc": "2.0", "id": 11, "method": "tools/call", "params": map[string]any{"name": "CallWixSiteAPI", "arguments": map[string]any{"method": "DELETE", "url": "https://www.wixapis.com/a"}}},
	}
	status, _, out := streamablePost(t, srv.URL+"/mcp", sid, tok, batch)
	if status != http.StatusBadRequest {
		t.Fatalf("batch status = %d body=%s", status, out)
	}
	if rr.calls() != 0 {
		t.Fatalf("batch reached the upstream %d times", rr.calls())
	}
	// Each element on its own, through the dispatcher the batch would use.
	allowed := runCall(t, proxy, argCall(10, tok, "CallWixSiteAPI", `{"method":"GET","url":"https://www.wixapis.com/a"}`))
	refused := runCall(t, proxy, argCall(11, tok, "CallWixSiteAPI", `{"method":"DELETE","url":"https://www.wixapis.com/a"}`))
	if allowed.denied || !refused.denied || rr.calls() != 1 {
		t.Fatalf("allowed.denied=%v refused.denied=%v calls=%d", allowed.denied, refused.denied, rr.calls())
	}
}

// A notification, an unknown method and forwardToDefault never carry a
// tools/call to an upstream.
func TestNonToolCallPathsCannotForwardToolCalls(t *testing.T) {
	proxy, rr, _ := newArgProxy(t)
	_, tok := argToken(t, "CallWixSiteAPI,ManageWixSite", wixReadOnlyDoc)
	// A tools/call arriving as a notification (no id) still goes through the
	// governed path: it is judged and the answer is simply not sent.
	n := argCall(0, tok, "CallWixSiteAPI", `{"method":"DELETE","url":"https://www.wixapis.com/a"}`)
	n.ID = nil
	if resp, _ := proxy.handleRequest(context.Background(), n); resp == nil || resp.Error == nil || rr.calls() != 0 {
		t.Fatalf("notification-style DELETE: resp=%+v calls=%d", resp, rr.calls())
	}
	// forwardToDefault refuses a tools/call even if a later edit routed one there.
	resp, err := proxy.forwardToDefault(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodToolsCall, Params: json.RawMessage(`{"name":"CallWixSiteAPI"}`)})
	if err != nil || resp == nil || resp.Error == nil || rr.calls() != 0 {
		t.Fatalf("forwardToDefault: resp=%+v err=%v calls=%d", resp, err, rr.calls())
	}
	// Notifications that are not tools/call are dropped without a reply.
	resp, err = proxy.handleRequest(context.Background(), &Request{JSONRPC: "2.0", Method: "notifications/whatever"})
	if err != nil || resp != nil {
		t.Fatalf("notification: resp=%+v err=%v", resp, err)
	}
}

// No rules, no change: a token without rules and a tool without a rule pass
// exactly as before, even with odd arguments.
func TestToolWithoutRuleBehavesAsBefore(t *testing.T) {
	proxy, rr, _ := newArgProxy(t)
	_, plain := argToken(t, "CallWixSiteAPI,other_tool")
	_, ruled := argToken(t, "CallWixSiteAPI,ManageWixSite,other_tool", wixReadOnlyDoc)
	for _, tc := range []struct{ tok, tool, args string }{
		{plain, "CallWixSiteAPI", `{"method":"DELETE","method":"GET"}`},
		{plain, "CallWixSiteAPI", `{"method":"DELETE"}`},
		{ruled, "other_tool", `{"method":"DELETE","method":"GET"}`},
	} {
		before := rr.calls()
		if o := runCall(t, proxy, argCall(1, tc.tok, tc.tool, tc.args)); o.denied || rr.calls() != before+1 {
			t.Errorf("%s %s: denied=%v", tc.tool, tc.args, o.denied)
		}
	}
}

// The delegate method keeps rules: a child minted from a ruled parent through
// the proxy still carries them.
func TestDelegatedChildKeepsParentRules(t *testing.T) {
	proxy, rr, _ := newArgProxy(t)
	svc, tok := argToken(t, "CallWixSiteAPI,ManageWixSite", wixReadOnlyDoc)
	info, err := proxy.auth.Verify(context.Background(), tok)
	if err != nil {
		t.Fatal(err)
	}
	if len(info.ArgumentRules) == 0 {
		t.Fatal("TokenInfo.ArgumentRules empty for a ruled token")
	}
	d := NewDelegator(svc, proxy.budget)
	if d.macaroonSvc == svc {
		t.Fatal("delegator must use the rule-accepting service")
	}
	child, err := d.macaroonSvc.Delegate(tok, []string{"scope = CallWixSiteAPI"})
	if err != nil {
		t.Fatal(err)
	}
	ctok := svc.Encode(child)
	o := runCall(t, proxy, argCall(1, ctok, "CallWixSiteAPI", `{"method":"DELETE","url":"https://www.wixapis.com/a"}`))
	if !o.denied || rr.calls() != 0 {
		t.Fatalf("child dropped parent rule: denied=%v calls=%d", o.denied, rr.calls())
	}
}

func base64URL(s string) string {
	return base64.RawURLEncoding.EncodeToString([]byte(s))
}
