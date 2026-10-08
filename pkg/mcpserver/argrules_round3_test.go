package mcpserver

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// A rule on a tool that runs scripts cannot be enforced (B1). Mint never makes
// such a token; these tests build it directly, as a holder with a signing key
// could, and show the runtime refuses it on every path with the upstream
// untouched.
const scriptRuleDoc = `{"v":1,"rules":[{"tool":"ExecuteWixAPI","shapes":[{"method":{"one_of":["GET"],"ascii_case_insensitive":true}}]}]}`

const scriptDecoyArgs = `{"method":"GET","code":"async () => wix.request({method: 'DELETE', url: 'https://www.wixapis.com/stores/v1/products/x'})","hasMutations":true}`

func TestScriptToolRuleTokenIsRefusedOnSSEStreamableAndBatch(t *testing.T) {
	for _, scope := range []string{"ExecuteWixAPI", "*", "mcp:*"} {
		t.Run(scope, func(t *testing.T) {
			p, rr, _ := newArgProxy(t)
			_, tok := argToken(t, scope, scriptRuleDoc)
			srv := httptest.NewServer(NewSSEServer(p, ":0").Handler())
			defer srv.Close()

			// Legacy SSE route.
			req, _ := http.NewRequest("GET", srv.URL+"/sse", nil)
			req.Header.Set("Authorization", "Bearer "+tok)
			res, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatal(err)
			}
			defer res.Body.Close()
			sid := readEndpointEvent(t, res.Body)
			var params any
			_ = json.Unmarshal([]byte(`{"name":"ExecuteWixAPI","arguments":`+scriptDecoyArgs+`}`), &params)
			sendMessage(t, srv.URL, sid, tok, map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call", "params": params})
			ev := string(readMessageEvent(t, res.Body))
			if !strings.Contains(ev, argrules.DenialCode) || !strings.Contains(ev, "runs scripts") {
				t.Errorf("SSE: not refused as expected: %s", ev)
			}

			// Streamable HTTP.
			hs := mustInitSession(t, srv.URL, tok)
			status, _, out := streamablePost(t, srv.URL+"/mcp", hs, tok, map[string]any{"jsonrpc": "2.0", "id": 3, "method": "tools/call", "params": params})
			if status != http.StatusOK || !strings.Contains(out, argrules.DenialCode) || !strings.Contains(out, "runs scripts") {
				t.Errorf("streamable: status=%d body=%s", status, out)
			}

			// Batch: refused outright by the route, and each element refused on its own.
			batch := []any{
				map[string]any{"jsonrpc": "2.0", "id": 4, "method": "tools/call", "params": params},
				map[string]any{"jsonrpc": "2.0", "id": 5, "method": "tools/call", "params": params},
			}
			status, _, out = streamablePost(t, srv.URL+"/mcp", hs, tok, batch)
			if status != http.StatusBadRequest {
				t.Errorf("batch: status=%d body=%s", status, out)
			}
			o := runCall(t, p, argCall(6, tok, "ExecuteWixAPI", scriptDecoyArgs))
			if !o.denied || !strings.Contains(o.resp.Error.Message, "runs scripts") {
				t.Errorf("element: %+v", o.resp)
			}
			if rr.calls() != 0 {
				t.Errorf("upstream contacted %d times", rr.calls())
			}
		})
	}
}

// The token is not trusted for anything else either: a rule that cannot be
// enforced next to a rule that can must not leave the second one looking applied.
func TestScriptToolRuleTokenRefusesOtherToolsToo(t *testing.T) {
	p, rr, _ := newArgProxy(t)
	both := `{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET"]}}]},{"tool":"ExecuteWixAPI","shapes":[{"method":{"one_of":["GET"]}}]}]}`
	_, tok := argToken(t, "*", both)
	o := runCall(t, p, argCall(1, tok, "CallWixSiteAPI", `{"method":"GET"}`))
	if !o.denied || rr.calls() != 0 {
		t.Fatalf("denied=%v calls=%d", o.denied, rr.calls())
	}
}

// A key with a rule on CallWixSiteAPI that also allows ExecuteWixAPI with no
// rule is a legal key; the script tool is simply not limited.
func TestRuleOnAnotherToolDoesNotRefuseTheUnruledScriptTool(t *testing.T) {
	p, rr, _ := newArgProxy(t)
	_, tok := argToken(t, "CallWixSiteAPI,ExecuteWixAPI", `{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET"]}}]}]}`)
	if o := runCall(t, p, argCall(1, tok, "ExecuteWixAPI", scriptDecoyArgs)); o.denied || rr.calls() != 1 {
		t.Fatalf("denied=%v calls=%d", o.denied, rr.calls())
	}
}

// B2: the scope check itself refuses a reserved-prefix name, under every scope.
func TestScopeCheckRefusesReservedPrefixNames(t *testing.T) {
	names := []string{"argrules:", "argrules:v1:abc", "ARGRULES:foo", "ArgRules:x"}
	for _, scope := range []string{"*", "api:*", "mcp:*", "argrules:*", "argrules:v1:abc", "mcp:*,tools/call"} {
		for _, n := range names {
			if matchScope(scope, n) {
				t.Errorf("matchScope(%q, %q) = true", scope, n)
			}
			if ScopeValueAllowsTool(scope, n) {
				t.Errorf("ScopeValueAllowsTool(%q, %q) = true", scope, n)
			}
		}
	}
	if !matchScope("*", "CallWixSiteAPI") || !matchScope("CallWixSiteAPI", "CallWixSiteAPI") {
		t.Fatal("ordinary names must still match")
	}
	// A token whose scope is a wildcard, and one that names the exact word.
	for _, scope := range []string{"*", "mcp:*"} {
		svc, tok := argToken(t, scope, `{"v":1,"rules":[{"tool":"CallWixSiteAPI","shapes":[{"method":{"one_of":["GET"]}}]}]}`)
		mac, err := svc.Decode(tok)
		if err != nil {
			t.Fatal(err)
		}
		info := &TokenInfo{Raw: mac, Scope: scope}
		for _, c := range mac.Caveats {
			word := strings.TrimPrefix(c, "scope = ")
			if strings.HasPrefix(word, macaroon.ArgumentRulesScopePrefix) && info.AllowsTool(word) {
				t.Errorf("AllowsTool accepts the rule word under scope %s", scope)
			}
		}
		if info.AllowsTool("argrules:v1:anything") {
			t.Errorf("AllowsTool accepts a reserved-prefix name under scope %s", scope)
		}
	}
	if (&TokenInfo{Scope: "*"}).AllowsTool("argrules:x") {
		t.Fatal("scope-string path accepts a reserved name")
	}
}

// N2: "_meta": null used to panic when the token was written into a nil map.
func TestMetaNullIsTreatedAsAbsentThroughSSEAndStreamable(t *testing.T) {
	raw := json.RawMessage(`{"name":"x","_meta":null}`)
	out := injectMetaToken(raw, "placeholder-token")
	var m struct {
		Meta map[string]string `json:"_meta"`
	}
	if err := json.Unmarshal(out, &m); err != nil || m.Meta["token"] != "placeholder-token" {
		t.Fatalf("out=%s err=%v", out, err)
	}
	for _, in := range []string{`{"name":"x","_meta":{}}`, `{"name":"x","_meta":{"progressToken":7654321e999}}`, `{"name":"x","_meta":"text"}`, `{"name":"x","_meta":[1]}`} {
		injectMetaToken(json.RawMessage(in), "placeholder-token") // must not panic
	}

	p, rr, _ := newArgProxy(t)
	_, tok := argToken(t, "place_equity_order", robinhoodCapDoc)
	srv := httptest.NewServer(NewSSEServer(p, ":0").Handler())
	defer srv.Close()
	req, _ := http.NewRequest("GET", srv.URL+"/sse", nil)
	req.Header.Set("Authorization", "Bearer "+tok)
	res, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatal(err)
	}
	defer res.Body.Close()
	sid := readEndpointEvent(t, res.Body)
	body := map[string]any{"jsonrpc": "2.0", "id": 2, "method": "tools/call"}
	var params any
	_ = json.Unmarshal([]byte(`{"name":"place_equity_order","arguments":{"type":"market","dollar_amount":"1000000"},"_meta":null}`), &params)
	body["params"] = params
	sendMessage(t, srv.URL, sid, tok, body)
	if ev := string(readMessageEvent(t, res.Body)); !strings.Contains(ev, argrules.DenialCode) {
		t.Errorf("SSE with _meta null: %s", ev)
	}
	hs := mustInitSession(t, srv.URL, tok)
	status, _, out2 := streamablePost(t, srv.URL+"/mcp", hs, tok, body)
	if status != http.StatusOK || !strings.Contains(out2, argrules.DenialCode) {
		t.Errorf("streamable with _meta null: status=%d body=%s", status, out2)
	}
	if rr.calls() != 0 {
		t.Errorf("upstream contacted %d times", rr.calls())
	}
}
