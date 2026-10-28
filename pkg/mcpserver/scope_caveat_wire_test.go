package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/argrules"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/macaroon"
)

// wireScopeToken mints a token on svc's key with an optional scope, adds
// the raw caveats as written and encodes it as JSON or binary.
func wireScopeToken(t *testing.T, svc *macaroon.Service, scope string, binary bool, caveats ...string) string {
	t.Helper()
	mac, err := svc.Mint(scope, time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac.Caveats = append(mac.Caveats, caveats...)
	mac.Signature = svc.RecalculateSignature(mac)
	if binary {
		tok, err := svc.EncodeBinary(mac)
		if err != nil {
			t.Fatal(err)
		}
		return tok
	}
	return svc.Encode(mac)
}

// wireCallsEcho runs tools/call "echo" over Streamable HTTP and reports
// whether it returned a result and whether it was forwarded upstream; the
// two must agree.
func wireCallsEcho(t *testing.T, r *lineageRig, tok string, args map[string]any) bool {
	t.Helper()
	r.fund(tok, 10)
	sid := mustInitSession(t, r.srv.URL, tok)
	before := streamableForwardCount(r.proxy)
	body := toolCallBody(2, "echo")
	if args != nil {
		body["params"].(map[string]any)["arguments"] = args
	}
	_, _, resp := streamablePost(t, r.srv.URL+"/mcp", sid, tok, body)
	allowed := strings.Contains(resp, `"result"`)
	forwarded := streamableForwardCount(r.proxy) > before
	if allowed != forwarded {
		t.Fatalf("tools/call result=%v but forwarded=%v (body %s)", allowed, forwarded, resp)
	}
	return allowed
}

// D1: tools/call enforces every caveat Verify reads as a scope caveat. Verify
// trims the key, so " scope = api:*" is a scope caveat and narrows the
// token; MCP finds scope caveats with the same reading
// (macaroon.ScopeCaveatValue).
func TestToolsCall_EnforcesEveryScopeCaveatVerifyReads(t *testing.T) {
	r := newLineageRig(t, nil)
	for _, tc := range []struct {
		caveat string
		want   bool
	}{
		{" scope = api:*", false},
		{"scope  = api:*", false},
		{"\tscope = api:*", false},
		{"scope\t = api:*", false},
		{" scope = other", false},
		{"  scope  = mcp:other", false},
		// The same key reading allows what the caveat grants.
		{" scope = echo", true},
		{"\tscope = mcp:echo", true},
		// Not a scope caveat for Verify either (another key): ignored by both.
		{"scopes = api:*", true},
		{"scope_x = api:*", true},
	} {
		for _, binary := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/binary=%v", tc.caveat, binary), func(t *testing.T) {
				tok := wireScopeToken(t, r.svc, "mcp:*", binary, tc.caveat)
				if _, err := r.svc.Verify(tok); err != nil {
					t.Fatalf("fixture does not verify: %v", err)
				}
				if got := wireCallsEcho(t, r, tok, nil); got != tc.want {
					t.Errorf("tools/call echo allowed=%v, want %v", got, tc.want)
				}
			})
		}
	}
}

// D1: an argument-rule caveat is read by argrules.Collect only when written
// exactly "scope = <rule>". A rule word under any other spelling of the
// scope key (which Verify still reads as a scope caveat) would not be
// enforced, so it allows no tool.
func TestToolsCall_RuleCaveatInOtherSpellingAllowsNoTool(t *testing.T) {
	rule, err := argrules.Caveat([]argrules.Rule{{Tool: "echo", Shapes: []argrules.Shape{{Fields: []argrules.Condition{{Field: "mode", OneOf: []string{"read"}}}}}}})
	if err != nil {
		t.Fatal(err)
	}
	r := newLineageRig(t, nil)
	for _, tc := range []struct {
		name, caveat string
		want         bool
	}{
		{"canonical", rule, true},
		{"leading space", " " + rule, false},
		{"tab", "\t" + rule, false},
		{"double space", strings.Replace(rule, "scope = ", "scope  = ", 1), false},
	} {
		for _, binary := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/binary=%v", tc.name, binary), func(t *testing.T) {
				tok := wireScopeToken(t, r.svc, "mcp:*", binary, tc.caveat)
				info, err := (&MacaroonAuthenticator{Service: r.svc}).Verify(context.Background(), tok)
				if err != nil {
					t.Fatalf("fixture does not verify: %v", err)
				}
				if got := info.AllowsTool("echo"); got != tc.want {
					t.Errorf("AllowsTool(echo) = %v, want %v", got, tc.want)
				}
				if got := wireCallsEcho(t, r, tok, map[string]any{"mode": "read"}); got != tc.want {
					t.Errorf("tools/call echo allowed=%v, want %v", got, tc.want)
				}
			})
		}
	}
}

// D2: a word's prefix is removed as a tenant only when it is a complete
// canonical UUID followed by ":". Any other namespace stays as written, so a
// wildcard of another namespace never becomes a tool grant.
func TestToolsCall_TenantPrefixOnlyForAUUID(t *testing.T) {
	const tenant = "0f2b7c4e-1d3a-4b5c-8e9f-0a1b2c3d4e5f"
	// 36 bytes with "-" at 8 and 13 but not a UUID: the shape the old check
	// accepted.
	fake := func(ns string) string {
		b := []byte(ns + strings.Repeat("a", 36-len(ns)))
		b[8], b[13] = '-', '-'
		return string(b)
	}
	r := newLineageRig(t, nil)
	for _, tc := range []struct {
		scope string
		want  bool
	}{
		{tenant + ":mcp:*", true},
		{tenant + ":*", true},
		{tenant + ":echo", true},
		{tenant + ":mcp:echo", true},
		{strings.ToUpper(tenant) + ":mcp:*", true},
		{"12345678-1234-1234-1234-123456789abc:mcp:*", true},
		{"01234567-89ab-cdef-0123-456789abcdef:mcp:*", true},
		{tenant + ":api:*", false},
		{tenant + ": mcp:*", false},
		{tenant + ":mcp:* ", false},
		{tenant + "::mcp:*", false},
		{fake("api:") + ":*", false},
		{fake("mcp:") + ":*", false},
		{fake("api:") + ":mcp:*", false},
		{"api:xxxx-xxxx-" + strings.Repeat("x", 22) + ":mcp:*", false},
		{"0f2b7c4g-1d3a-4b5c-8e9f-0a1b2c3d4e5f:mcp:*", false},
		{"0f2b7c4e-1d3a-4b5c-8e9f_0a1b2c3d4e5f:mcp:*", false},
		{"0f2b7c4e-1d3a-4b5cx8e9f-0a1b2c3d4e5f:mcp:*", false},
		{"0f2b7c4e1d3a4b5c8e9f0a1b2c3d4e5f0000:mcp:*", false},
		{"{0f2b7c4e-1d3a-4b5c-8e9f-0a1b2c3d4e5}:mcp:*", false},
	} {
		for _, binary := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/binary=%v", tc.scope, binary), func(t *testing.T) {
				tok := wireScopeToken(t, r.svc, tc.scope, binary)
				if got := wireCallsEcho(t, r, tok, nil); got != tc.want {
					t.Errorf("tools/call echo allowed=%v, want %v", got, tc.want)
				}
			})
		}
	}
}

// D3: a scope word is not trimmed. A word with surrounding or inner
// whitespace, or any other form macaroon.ScopeGrants calls malformed, grants
// nothing, as on the HTTP gateway. Other words of the same list still count.
func TestToolsCall_MalformedScopeWordsGrantNothing(t *testing.T) {
	r := newLineageRig(t, nil)
	for _, tc := range []struct {
		scope string
		want  bool
	}{
		{"mcp:* ", false},
		{" mcp:*", false},
		{"mcp:*\t", false},
		{"* ", false},
		{"echo ", false},
		{" echo", false},
		{"mcp: echo", false},
		{"mcp:\techo", false},
		{"mc p:*", false},
		{"mcp:echo\n", false},
		{"mcp:ec ho", false},
		{"other, echo", false},
		{"other ,echo", true},
		{"other,echo", true},
		{"mcp:echo,mcp:other", true},
		{"mcp:echo,", true},
		{",", false},
		{"mcp::echo", false},
		{"mcp:ec*", false},
		{"mcp:*:bad", false},
		{"MCP:*", false},
		{"mcp:ECHO", false},
		{"mcp:echo:*", false},
		{"api:*", false},
		{"api:echo", false},
	} {
		for _, binary := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/binary=%v", tc.scope, binary), func(t *testing.T) {
				tok := wireScopeToken(t, r.svc, "", binary, "scope = "+tc.scope)
				if got := wireCallsEcho(t, r, tok, nil); got != tc.want {
					t.Errorf("tools/call echo allowed=%v, want %v", got, tc.want)
				}
			})
		}
	}
}

// D4 on MCP: the expiry spellings of macaroon.ParseExpires, through the
// authenticator, Streamable HTTP, SSE and stdio, JSON and binary tokens.
func TestMCP_ExpiresRFC3339SpellingEveryTransport(t *testing.T) {
	stdio := newStdioRig(t, governance.NewService(nil), nil)
	for _, tc := range mcpExpirySpellings() {
		for _, binary := range []bool{false, true} {
			t.Run(fmt.Sprintf("%q/binary=%v", tc.value, binary), func(t *testing.T) {
				r := newLineageRig(t, nil)
				tok := wireScopeToken(t, r.svc, "mcp:*", binary, "expires = "+tc.value)
				if _, err := (&MacaroonAuthenticator{Service: r.svc}).Verify(context.Background(), tok); (err == nil) != tc.ok {
					t.Errorf("authenticator accepted=%v, want %v (%v)", err == nil, tc.ok, err)
				}
				if status, _, _ := streamablePost(t, r.srv.URL+"/mcp", "", tok, initializeBody("2025-03-26")); (status == http.StatusOK) != tc.ok {
					t.Errorf("Streamable HTTP initialize status %d, want accepted=%v", status, tc.ok)
				}
				conn := eaOpenSSE(t, r.srv.URL, tok)
				defer conn.body.Close()
				raw, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": 1, "method": "tools/list"})
				if status := conn.post(raw); status != http.StatusAccepted {
					t.Fatalf("SSE message status %d", status)
				}
				if resp := conn.next(); (resp.Error == nil) != tc.ok {
					t.Errorf("SSE tools/list accepted=%v, want %v", resp.Error == nil, tc.ok)
				}
				stok := wireScopeToken(t, stdio.svc, "mcp:*", binary, "expires = "+tc.value)
				if resp := stdio.call(MethodToolsList, stok); (resp.Error == nil) != tc.ok {
					t.Errorf("stdio tools/list accepted=%v, want %v", resp.Error == nil, tc.ok)
				}
			})
		}
	}
}

type mcpExpirySpelling struct {
	value string
	ok    bool
}

// mcpExpirySpellings is the date table of pkg/macaroon TestParseExpires_RFC3339Spelling
// (all accepted spellings are in the future).
func mcpExpirySpellings() []mcpExpirySpelling {
	return []mcpExpirySpelling{
		{"2099-01-01T00:00:00Z", true},
		{"2099-01-01T00:00:00.5Z", true},
		{"2099-01-01T00:00:00.123456789Z", true},
		{"2099-01-01T01:30:00+01:30", true},
		{"2099-01-01T00:00:00-05:00", true},
		{"2099-12-31T23:59:59+23:59", true},
		{"2096-02-29T00:00:00Z", true},
		{" 2099-01-01T00:00:00Z\t", true},
		{"4102444800000", true},
		{"2099-01-01T0:00:00Z", false},
		{"2099-01-01T00:0:00Z", false},
		{"2099-01-01T00:00:0Z", false},
		{"2099-1-01T00:00:00Z", false},
		{"2099-01-1T00:00:00Z", false},
		{"20990-01-01T00:00:00Z", false},
		{"+2099-01-01T00:00:00Z", false},
		{"2099-01-01T00:00:00,5Z", false},
		{"2099-01-01T00:00:00.Z", false},
		{"2099-01-01T00:00:00.5.5Z", false},
		{"2099-01-01T00:00:00+24:00", false},
		{"2099-01-01T00:00:00+00:60", false},
		{"2099-01-01T00:00:00+0100", false},
		{"2099-01-01T00:00:00+01", false},
		{"2099-01-01T00:00:00", false},
		{"2099-01-01t00:00:00Z", false},
		{"2099-01-01T00:00:00z", false},
		{"2099-01-01 00:00:00Z", false},
		{"2099-01-01T24:00:00Z", false},
		{"2099-01-01T00:60:00Z", false},
		{"2099-01-01T00:00:60Z", false},
		{"2099-13-01T00:00:00Z", false},
		{"2099-00-01T00:00:00Z", false},
		{"2099-01-00T00:00:00Z", false},
		{"2099-01-32T00:00:00Z", false},
		{"2099-02-30T00:00:00Z", false},
		{"2097-02-29T00:00:00Z", false},
		{"2099-01-01T00:00:00ZZ", false},
		{"2099-01-01T00:00:00Z junk", false},
		{"2099-01-01T00:00Z", false},
		{"2099-01-01", false},
		{"Thu, 01 Jan 2099 00:00:00 GMT", false},
		{"24h", false},
	}
}
