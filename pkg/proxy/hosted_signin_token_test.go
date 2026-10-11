package proxy

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/macaroon"
)

// A hosted sign-in token is signed under its own key, never under the key this
// gateway verifies with. Every place the gateway verifies a token or extends one
// refuses it, and no refusal echoes a word of the token's caveats.
//
// Rows: proxy_oss.go capability route (handleCapability), L402 token check
// (verifyL402Token), mock-pay, /api/capability/validate, /delegate, /ping,
// /admin, and the agent loop's simulate for a capability policy.
func TestSignInKeyedTokenIsRefusedWhereverTheGatewayVerifies(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		t.Errorf("the upstream was contacted for a sign-in token: %s", r.URL.Path)
	}))
	defer upstream.Close()
	gw, _ := newTestGatewayWithUpstream(t, upstream, "capability")

	other, err := macaroon.NewService("a-different-server-key-for-sign-in")
	if err != nil {
		t.Fatal(err)
	}
	base, err := other.Mint("api:read", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	mac, err := other.DelegateWithoutVerify(other.Encode(base), []string{
		"resource = https://mcp.example.test/mcp", "tenant_id = secret-tenant", "payment_hash = " + strings.Repeat("a", 64),
	})
	if err != nil {
		t.Fatal(err)
	}
	token := other.Encode(mac)

	do := func(method, path, auth string, body any) *httptest.ResponseRecorder {
		t.Helper()
		var rdr *strings.Reader
		if body != nil {
			raw, _ := json.Marshal(body)
			rdr = strings.NewReader(string(raw))
		} else {
			rdr = strings.NewReader("")
		}
		req := httptest.NewRequest(method, path, rdr)
		req.Header.Set("X-Admin-Token", "admin-secret")
		if auth != "" {
			req.Header.Set("Authorization", auth)
		}
		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, req)
		return rec
	}
	noEcho := func(label string, rec *httptest.ResponseRecorder) {
		t.Helper()
		for _, leak := range []string{"mcp.example.test", "secret-tenant", "resource =", "tenant_id", token} {
			if strings.Contains(rec.Body.String(), leak) {
				t.Fatalf("%s echoes %q: %s", label, leak, rec.Body.String())
			}
		}
	}

	cases := []struct {
		label string
		rec   *httptest.ResponseRecorder
		want  []int
	}{
		{"capability route (proxy_oss.go:372)", do("GET", "/api/anything", "Bearer "+token, nil), []int{401}},
		{"L402 route token check (proxy_oss.go:436)", do("GET", "/api/anything", "L402 "+token+":"+strings.Repeat("b", 64), nil), []int{402, 401}},
		{"mock-pay (proxy_oss.go:610)", do("POST", "/api/l402/mock-pay", "", map[string]string{"payment_hash": strings.Repeat("a", 64), "macaroon": token}), []int{400, 501}},
		{"capability validate (proxy_oss.go:1067)", do("POST", "/api/capability/validate", "", map[string]string{"token": token}), []int{401, 400, 200}},
		{"capability delegate (proxy_oss.go:1102, :1109)", do("POST", "/api/capability/delegate", "", map[string]any{"parentToken": token, "caveats": []string{"scope = api:read"}}), []int{400}},
		{"capability ping (proxy_oss.go:1172)", do("GET", "/api/capability/ping", "Bearer "+token, nil), []int{401}},
		{"capability admin (proxy_oss.go:1236)", do("GET", "/api/capability/admin", "Bearer "+token, nil), []int{401}},
	}
	for _, c := range cases {
		ok := false
		for _, w := range c.want {
			ok = ok || c.rec.Code == w
		}
		if !ok {
			t.Fatalf("%s: HTTP %d %s, want one of %v", c.label, c.rec.Code, c.rec.Body.String(), c.want)
		}
		noEcho(c.label, c.rec)
	}

	// validate answers about the token, and what it says is that it is not valid.
	var v map[string]any
	_ = json.Unmarshal(cases[3].rec.Body.Bytes(), &v)
	if v["valid"] == true {
		t.Fatalf("validate accepted a sign-in token: %v", v)
	}
	// delegate made no child.
	if strings.Contains(cases[4].rec.Body.String(), `"token"`) {
		t.Fatalf("delegate returned a child token: %s", cases[4].rec.Body.String())
	}

	// agent loop (agent_loop.go:226): a capability policy denies it and never
	// contacts the upstream.
	staged := agentPost(t, gw, "/api/agent/policy", map[string]string{"name": "gated", "kind": "capability", "path_prefix": "/gated", "upstream_url": upstream.URL, "scope": "api:read"})
	var policy agentPolicy
	decode(t, staged, &policy)
	sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID, "authorization": "Bearer " + token})
	var pack map[string]any
	decode(t, sim, &pack)
	if pack["decision"] != "denied" || pack["upstream_contacted"] != false {
		t.Fatalf("agent loop admitted a sign-in token: %v", pack)
	}
	noEcho("agent loop (agent_loop.go:226)", sim)
}
