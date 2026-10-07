package mcpserver

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strconv"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// ---- F2: no float64 and no input bytes on the tools/call parse path ---------

var (
	longInt = strings.Repeat("9", 400)
	// Each case is one argument position the parser or the rule checker reads.
	oddNumberParams = []struct {
		name   string
		params string
		secret string // text that must never come back to the caller
	}{
		{"huge exponent in a rule field", `{"name":"place_equity_order","arguments":{"type":"limit","quantity":123456789123456789e999,"dollar_amount":"1"}}`, "123456789123456789e999"},
		{"400 digit integer in a rule field", `{"name":"place_equity_order","arguments":{"type":"limit","quantity":` + longInt + `,"dollar_amount":"1"}}`, longInt},
		{"1e999", `{"name":"place_equity_order","arguments":{"type":"limit","quantity":1e999,"dollar_amount":"1"}}`, "1e999"},
		{"-1e999", `{"name":"place_equity_order","arguments":{"type":"limit","quantity":-1e999,"dollar_amount":"1"}}`, "-1e999"},
		{"number in a nested object", `{"name":"place_equity_order","arguments":{"type":"limit","quantity":{"a":{"b":7777777e999}},"dollar_amount":"1"}}`, "7777777e999"},
		{"number in an array", `{"name":"place_equity_order","arguments":{"type":"limit","quantity":[8888888e999],"dollar_amount":"1"}}`, "8888888e999"},
		{"number in an unruled field", `{"name":"place_equity_order","arguments":{"type":"limit","quantity":"1","dollar_amount":"1","note":6666666e999}}`, "6666666e999"},
		{"JSON number for params.name", `{"name":5555555e999,"arguments":{"quantity":"1"}}`, "5555555e999"},
		{"huge integer for params.name", `{"name":` + longInt + `,"arguments":{}}`, longInt},
		{"arguments is a number", `{"name":"place_equity_order","arguments":4444444e999}`, "4444444e999"},
		{"arguments is an array", `{"name":"place_equity_order","arguments":[3333333e999]}`, "3333333e999"},
		{"name is an object holding a number", `{"name":{"x":2222222e999}}`, "2222222e999"},
		{"params is an array", `[1111111e999]`, "1111111e999"},
		{"params is a bare number", `9876543210123e999`, "9876543210123e999"},
		{"truncated after a huge number", `{"name":"place_equity_order","arguments":{"quantity":1212121212e999`, "1212121212e999"},
		{"invalid number text", `{"name":"place_equity_order","arguments":{"quantity":0x7e999}}`, "0x7e999"},
	}
)

// ParseToolCall must give back raw arguments and only fixed error text.
func TestParseToolCallKeepsNumbersRawAndErrorsFixed(t *testing.T) {
	tc, err := ParseToolCall(json.RawMessage(`{"name":"t","arguments":{"q":123456789123456789e999,"n":[1e999,{"x":-1e999}]}}`))
	if err != nil {
		t.Fatalf("a valid JSON number out of float64 range must parse: %v", err)
	}
	if string(tc.Arguments) != `{"q":123456789123456789e999,"n":[1e999,{"x":-1e999}]}` {
		t.Fatalf("arguments were not kept as sent: %s", tc.Arguments)
	}
	for _, c := range oddNumberParams {
		t.Run(c.name, func(t *testing.T) {
			_, err := ParseToolCall(json.RawMessage(c.params))
			if err == nil {
				return
			}
			if strings.Contains(err.Error(), c.secret) {
				t.Fatalf("parse error carries input: %q", err.Error())
			}
			switch err {
			case errToolCallEmptyParams, errToolCallParams, errToolCallName, errToolCallArguments:
			default:
				t.Fatalf("parse error is not one of the fixed messages: %q", err.Error())
			}
			if m := toolCallParseMessage(err); strings.Contains(m, c.secret) {
				t.Fatalf("message to the agent carries input: %q", m)
			}
		})
	}
	// A foreign error can never reach the agent through the message helper.
	if m := toolCallParseMessage(context.DeadlineExceeded); m != errToolCallParams.Error() {
		t.Fatalf("unknown error leaked: %q", m)
	}
}

// Every position, through handleRequest: the answer never carries the value,
// and the upstream is never contacted for a refused call.
func TestOddNumbersNeverEchoedAtProxy(t *testing.T) {
	for _, c := range oddNumberParams {
		t.Run(c.name, func(t *testing.T) {
			if !json.Valid([]byte(c.params)) || c.params[0] != '{' {
				t.Skip("not a JSON object: cannot carry _meta.token; covered by ParseToolCall and the HTTP test")
			}
			p, rr, events := newArgProxy(t)
			_, tok := argToken(t, "place_equity_order", robinhoodCapDoc)
			_ = drainEvents(events)
			params := injectMetaToken(json.RawMessage(c.params), tok)
			o := runCall(t, p, &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodToolsCall, Params: params})
			b, _ := json.Marshal(o.resp)
			if strings.Contains(string(b), c.secret) {
				t.Fatalf("response carries the input value: %.300s", b)
			}
			if !o.denied {
				// Only the unruled-field case may pass, because the rule does not read it.
				if c.name != "number in an unruled field" {
					t.Fatalf("call was not refused: %.300s", b)
				}
				return
			}
			if rr.calls() != 0 {
				t.Fatalf("a refused call reached the upstream")
			}
			code := o.resp.Error.Code
			if code != CodePolicyDenied && code != CodeInvalidParams {
				t.Fatalf("unexpected code %d", code)
			}
			if code == CodePolicyDenied {
				var data map[string]any
				_ = json.Unmarshal(o.resp.Error.Data, &data)
				if data["error"] != argrules.DenialCode {
					t.Fatalf("policy refusal without the argument-denial shape: %s", o.resp.Error.Data)
				}
			}
		})
	}
}

// The same positions over the real HTTP boundary (Streamable HTTP route).
func TestOddNumbersNeverEchoedOverHTTP(t *testing.T) {
	proxy, rr, _ := newArgProxy(t)
	_, tok := argToken(t, "place_equity_order", robinhoodCapDoc)
	srv := httptest.NewServer(NewSSEServer(proxy, ":0").Handler())
	defer srv.Close()
	sid := mustInitSession(t, srv.URL, tok)
	for i, c := range oddNumberParams {
		if c.name == "number in an unruled field" {
			continue
		}
		raw := `{"jsonrpc":"2.0","id":` + strconv.Itoa(i+2) + `,"method":"tools/call","params":` + c.params + `}`
		before := rr.calls()
		status, out := rawStreamablePost(t, srv.URL+"/mcp", sid, tok, raw)
		if strings.Contains(out, c.secret) {
			t.Errorf("%s: HTTP answer (status %d) carries the input: %.300s", c.name, status, out)
		}
		if rr.calls() != before {
			t.Errorf("%s: upstream contacted", c.name)
		}
		// 200 with a JSON-RPC error, or a 4xx with fixed text. Never a pass.
		if status == http.StatusOK && !strings.Contains(out, `"error"`) {
			t.Errorf("%s: call not refused: %.200s", c.name, out)
		}
		if status >= 500 {
			t.Errorf("%s: server error %d", c.name, status)
		}
	}
	if rr.calls() != 0 {
		t.Errorf("upstream call count = %d, want 0", rr.calls())
	}
}

// rawStreamablePost sends the bytes as written, so a body that is not valid
// JSON reaches the server the way a hostile client would send it.
func rawStreamablePost(t *testing.T, url, sessionID, token, body string) (int, string) {
	t.Helper()
	req, err := http.NewRequest(http.MethodPost, url, strings.NewReader(body))
	if err != nil {
		t.Fatal(err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	req.Header.Set("Mcp-Session-Id", sessionID)
	req.Header.Set("MCP-Protocol-Version", "2025-03-26")
	req.Header.Set("Authorization", "Bearer "+token)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer resp.Body.Close()
	b, _ := io.ReadAll(resp.Body)
	return resp.StatusCode, string(b)
}

// _meta numbers are carried as written when the token is injected.
func TestInjectMetaTokenKeepsNumbersAsWritten(t *testing.T) {
	out := injectMetaToken(json.RawMessage(`{"name":"t","_meta":{"progressToken":123456789123456789e999,"n":1.50}}`), "tok")
	s := string(out)
	if !strings.Contains(s, "123456789123456789e999") || !strings.Contains(s, "1.50") || !strings.Contains(s, `"token":"tok"`) {
		t.Fatalf("meta was re-encoded: %s", s)
	}
}

// ---- F5: nonce in the word, reserved prefix refused by new runtimes ---------

func TestSameRulesMintDifferentWords(t *testing.T) {
	a := ruleScopeWord(t, robinhoodCapDoc)
	b := ruleScopeWord(t, robinhoodCapDoc)
	if a == b {
		t.Fatalf("two mints of the same rules gave the same word")
	}
	pa := strings.TrimPrefix(a, "scope = ")
	pb := strings.TrimPrefix(b, "scope = ")
	parts := func(w string) []string { return strings.SplitN(w, ":", 4) }
	if len(parts(pa)) != 4 || len(parts(pa)[2]) != 32 || parts(pa)[2] == parts(pb)[2] {
		t.Fatalf("word shape: %q / %q", pa, pb)
	}
	// Both words carry the same rules.
	ra, err := argrules.ParseScopeValue(pa)
	if err != nil {
		t.Fatal(err)
	}
	rb, err := argrules.ParseScopeValue(pb)
	if err != nil {
		t.Fatal(err)
	}
	da, _ := argrules.Marshal(ra)
	db, _ := argrules.Marshal(rb)
	if string(da) != string(db) {
		t.Fatalf("same rules did not round trip to the same document")
	}
}

func TestRuleWordWithoutNonceIsRefused(t *testing.T) {
	word := strings.TrimPrefix(ruleScopeWord(t, robinhoodCapDoc), "scope = ")
	payload := word[strings.LastIndex(word, ":")+1:]
	for name, w := range map[string]string{
		"no nonce":          "argrules:v1:" + payload,
		"short nonce":       "argrules:v1:abcd:" + payload,
		"upper-case nonce":  "argrules:v1:" + strings.Repeat("A", 32) + ":" + payload,
		"non-hex nonce":     "argrules:v1:" + strings.Repeat("z", 32) + ":" + payload,
		"long nonce":        "argrules:v1:" + strings.Repeat("a", 33) + ":" + payload,
		"nonce but bad doc": "argrules:v1:" + strings.Repeat("a", 32) + ":AAAA",
	} {
		if _, err := argrules.ParseScopeValue(w); err == nil {
			t.Errorf("%s: parsed", name)
		}
	}
}

// A freshly minted word cannot equal a name an upstream chose without seeing
// the token: the best guess (no nonce, or any fixed nonce) never matches.
func TestUpstreamToolNamedLikeAGuessedWordDoesNotMatchAMintedWord(t *testing.T) {
	doc := `{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET"]}}]}]}`
	svc, tok := argToken(t, "*", doc)
	mac, err := svc.Decode(tok)
	if err != nil {
		t.Fatal(err)
	}
	old := &oldTokenInfo{Raw: mac}
	payload := ""
	for _, c := range mac.Caveats {
		if w := strings.TrimPrefix(c, "scope = "); macaroon.IsArgumentRulesScope(w) {
			payload = w[strings.LastIndex(w, ":")+1:]
		}
	}
	if payload == "" {
		t.Fatal("no rule caveat on the token")
	}
	guesses := []string{
		"argrules:v1:" + payload,                                 // the round-1 word
		"argrules:v1:" + strings.Repeat("0", 32) + ":" + payload, // a guessed nonce
		ruleScopeWord(t, doc),                                    // a second mint of the same rules
	}
	for _, g := range guesses {
		g = strings.TrimPrefix(g, "scope = ")
		if old.AllowsTool(g) {
			t.Errorf("old verifier accepts a tool named like a guessed word %q", g[:24])
		}
	}
}

type toolsRouter struct {
	recordingRouter
	tools []json.RawMessage
}

func (r *toolsRouter) AllToolsForTenant(context.Context, string) []json.RawMessage {
	return r.tools
}

func TestReservedPrefixToolsAreDroppedAndRefused(t *testing.T) {
	p, _, _ := newArgProxy(t)
	rt := &toolsRouter{tools: []json.RawMessage{
		json.RawMessage(`{"name":"CallWixSiteAPI"}`),
		json.RawMessage(`{"name":"argrules:v1:whatever"}`),
		json.RawMessage(`{"name":"ARGRULES:"}`),
		json.RawMessage(`{"name":"argrules:v1:` + strings.Repeat("a", 32) + `:AAAA"}`),
		json.RawMessage(`{"name":"argrulesX"}`), // does not start with the prefix "argrules:"
		json.RawMessage(`{"name":7}`),           // unreadable: dropped
		json.RawMessage(`not json`),             // unreadable: dropped
	}}
	p.router = rt
	resp, err := p.handleToolsListWithCtx(context.Background(), &Request{JSONRPC: "2.0", ID: json.RawMessage(`1`), Method: MethodToolsList})
	if err != nil {
		t.Fatal(err)
	}
	var got struct {
		Tools []struct {
			Name string `json:"name"`
		} `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &got); err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, x := range got.Tools {
		names = append(names, x.Name)
	}
	if strings.Join(names, ",") != "CallWixSiteAPI,argrulesX" {
		t.Fatalf("tools/list = %v", names)
	}
}

func TestReservedPrefixToolCallIsRefusedBeforeAnyUpstreamCall(t *testing.T) {
	p, rr, _ := newArgProxy(t)
	// A token that allows everything, with no rules at all: the name alone is refused.
	_, tok := argToken(t, "*")
	for _, name := range []string{"argrules:v1:abc", "argrules:", "ARGRULES:v1:x"} {
		o := runCall(t, p, argCall(1, tok, name, `{}`))
		if !o.denied || o.resp.Error.Code != CodePolicyDenied {
			t.Errorf("%s: not refused: %+v", name, o.resp)
		}
	}
	if rr.calls() != 0 {
		t.Fatalf("a reserved-prefix tool call reached the upstream")
	}
	// An ordinary name still passes on the same token.
	if o := runCall(t, p, argCall(2, tok, "CallWixSiteAPI", `{}`)); o.denied || rr.calls() != 1 {
		t.Fatalf("control call failed: %+v calls=%d", o.resp, rr.calls())
	}
	_ = time.Second
}

// ---- F1 (runtime side): the Wix read-only token leaves ExecuteWixAPI out -----

// The document the dashboard writes after round 2: GET also needs the Wix host.
const wixReadOnlyDocR2 = `{"v":1,"rules":[
 {"tool":"CallWixSiteAPI","shapes":[
  {"method":{"one_of":["GET"],"ascii_case_insensitive":true},"url":{"url":{"hosts":["www.wixapis.com"]}}},
  {"method":{"one_of":["POST"],"ascii_case_insensitive":true},"url":{"url":{"hosts":["www.wixapis.com"],"path_suffixes":["/query","/search","/count"]}}}]},
 {"tool":"ManageWixSite","shapes":[
  {"method":{"one_of":["GET"],"ascii_case_insensitive":true}}]}]}`

// The token the picker mints for "Wix read only": ExecuteWixAPI is not in the
// tool list. A script call with a decoy method=GET is refused by tool scope
// (CodePolicyDenied, not TOOL_ARGUMENT_DENIED), and the upstream never hears of it.
func TestWixReadOnlyTokenRefusesExecuteWixAPIByToolScope(t *testing.T) {
	p, rr, events := newArgProxy(t)
	_, tok := argToken(t, "CallWixSiteAPI,ManageWixSite", wixReadOnlyDocR2)
	_ = drainEvents(events)
	decoy := `{"method":"GET","code":"async () => wix.request({method: 'DELETE', url: 'https://www.wixapis.com/stores/v1/products/x'})","hasMutations":true}`
	o := runCall(t, p, argCall(1, tok, "ExecuteWixAPI", decoy))
	if !o.denied || o.resp.Error.Code != CodePolicyDenied {
		t.Fatalf("ExecuteWixAPI was not refused: %+v", o.resp)
	}
	var data map[string]any
	_ = json.Unmarshal(o.resp.Error.Data, &data)
	if data["error"] == argrules.DenialCode {
		t.Fatalf("refused by an argument rule, want tool scope: %s", o.resp.Error.Data)
	}
	if rr.calls() != 0 {
		t.Fatalf("upstream contacted")
	}
}

func TestWixReadOnlyGETNeedsTheWixHost(t *testing.T) {
	runArgCases(t, "CallWixSiteAPI,ManageWixSite", wixReadOnlyDocR2, []argCase{
		{"GET on the Wix host", "CallWixSiteAPI", `{"method":"GET","url":"https://www.wixapis.com/stores/v1/products"}`, true, ""},
		{"get lower case on the Wix host", "CallWixSiteAPI", `{"method":"get","url":"https://www.wixapis.com/x"}`, true, ""},
		{"GET on another host", "CallWixSiteAPI", `{"method":"GET","url":"https://evil.example/x"}`, false, "url"},
		{"GET on a lookalike host", "CallWixSiteAPI", `{"method":"GET","url":"https://www.wixapis.com.evil.example/x"}`, false, "url"},
		{"GET with no url", "CallWixSiteAPI", `{"method":"GET"}`, false, "url"},
		{"GET on plain http", "CallWixSiteAPI", `{"method":"GET","url":"http://www.wixapis.com/x"}`, false, "url"},
		{"POST query on the Wix host", "CallWixSiteAPI", `{"method":"POST","url":"https://www.wixapis.com/stores/v1/products/query"}`, true, ""},
		{"DELETE on the Wix host", "CallWixSiteAPI", `{"method":"DELETE","url":"https://www.wixapis.com/stores/v1/products/1"}`, false, "method"},
		{"ManageWixSite GET", "ManageWixSite", `{"method":"GET"}`, true, ""},
		{"ManageWixSite POST", "ManageWixSite", `{"method":"POST"}`, false, "method"},
	})
}
