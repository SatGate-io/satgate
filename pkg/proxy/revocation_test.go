package proxy

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
)

// Tests for revocation: a ban covers the banned token and every token
// delegated from it (before or after the ban, at any depth, online or
// offline), on every path the gateway verifies a token, on L402 proofs, and
// across gateways that share Redis.

// countingUpstream counts the requests that reach it.
func countingUpstream(t *testing.T) (*httptest.Server, *atomic.Int64) {
	t.Helper()
	var hits atomic.Int64
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits.Add(1)
		w.WriteHeader(http.StatusOK)
	}))
	t.Cleanup(srv.Close)
	return srv, &hits
}

func revocationConfig(upstreamURL string) *config.Config {
	return &config.Config{
		Server: config.ServerConfig{Listen: ":8080"},
		Admin:  config.AdminConfig{Token: "admin-secret"},
		Cloud:  &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
		Upstreams: map[string]config.Upstream{
			"backend": {URL: upstreamURL, Timeout: 10 * time.Second},
		},
		Routes: []config.Route{
			{
				Name:     "mcp-route",
				Match:    config.RouteMatch{PathPrefix: "/mcp"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "capability"},
				MCP:      &config.MCPConfig{Enabled: true},
			},
			{
				Name:     "api-route",
				Match:    config.RouteMatch{PathPrefix: "/api/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "capability"},
			},
			{
				Name:     "cheap",
				Match:    config.RouteMatch{PathPrefix: "/cheap/"},
				Upstream: "backend",
				Policy:   config.RoutePolicy{Kind: "l402", PriceSats: 10},
			},
		},
	}
}

func newRevocationGateway(t *testing.T, upstreamURL string, gov *governance.Service) *Gateway {
	t.Helper()
	gw, err := New(Options{
		Config:     revocationConfig(upstreamURL),
		Macaroon:   newTestMacaroonService(t),
		Governance: gov,
		Lightning:  lightning.NewMockProvider(),
	})
	if err != nil {
		t.Fatalf("create gateway: %v", err)
	}
	return gw
}

func newRedisGovernance(t *testing.T, addr string) *governance.Service {
	t.Helper()
	gov, err := governance.NewServiceWithRedis(&governance.RedisConfig{Addr: addr})
	if err != nil {
		t.Fatalf("governance on redis: %v", err)
	}
	return gov
}

func postJSON(t *testing.T, gw *Gateway, path string, body any, admin bool) *httptest.ResponseRecorder {
	t.Helper()
	data, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, path, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")
	if admin {
		req.Header.Set("X-Admin-Token", "admin-secret")
	}
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	return rec
}

type delegated struct {
	Token     string `json:"token"`
	Signature string `json:"signature"`
}

// delegateVia delegates parent through the gateway's delegate endpoint.
func delegateVia(t *testing.T, gw *Gateway, parent string, caveats ...string) delegated {
	t.Helper()
	rec := postJSON(t, gw, "/api/capability/delegate", map[string]any{"parentToken": parent, "caveats": caveats}, false)
	if rec.Code != http.StatusOK {
		t.Fatalf("delegate: want 200, got %d", rec.Code)
	}
	var d delegated
	if err := json.NewDecoder(rec.Body).Decode(&d); err != nil || d.Token == "" {
		t.Fatalf("delegate: no token in response (%v)", err)
	}
	return d
}

func banVia(t *testing.T, gw *Gateway, sig string) {
	t.Helper()
	rec := postJSON(t, gw, "/api/governance/ban", map[string]string{"tokenSignature": sig, "reason": "test"}, true)
	if rec.Code != http.StatusOK {
		t.Fatalf("ban: want 200, got %d", rec.Code)
	}
}

func bearerGet(t *testing.T, gw *Gateway, path, token string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	req.Header.Set("Authorization", "Bearer "+token)
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	return rec
}

func mintRoot(t *testing.T, gw *Gateway) (token, sig string) {
	t.Helper()
	mac, err := gw.macaroonSvc.Mint("api:*", time.Now().Add(time.Hour))
	if err != nil {
		t.Fatal(err)
	}
	return gw.macaroonSvc.Encode(mac), mac.Signature
}

func TestRevocation_BanCoversTokensDelegatedBeforeTheBan(t *testing.T) {
	up, hits := countingUpstream(t)
	gw := newRevocationGateway(t, up.URL, governance.NewService(nil))
	root, rootSig := mintRoot(t, gw)
	child := delegateVia(t, gw, root, "scope = api:read")
	grandchild := delegateVia(t, gw, child.Token, "expires = 30m")

	for name, tok := range map[string]string{"root": root, "child": child.Token, "grandchild": grandchild.Token} {
		if rec := bearerGet(t, gw, "/api/x", tok); rec.Code != http.StatusOK {
			t.Fatalf("%s before ban: want 200, got %d", name, rec.Code)
		}
	}
	banVia(t, gw, rootSig)
	before := hits.Load()
	for name, tok := range map[string]string{"root": root, "child": child.Token, "grandchild": grandchild.Token} {
		if rec := bearerGet(t, gw, "/api/x", tok); rec.Code != http.StatusUnauthorized {
			t.Fatalf("%s after ban of root: want 401, got %d", name, rec.Code)
		}
	}
	if hits.Load() != before {
		t.Fatalf("a token covered by the ban reached the upstream")
	}
}

func TestRevocation_BanCoversOfflineAttenuationAtAnyDepth(t *testing.T) {
	up, _ := countingUpstream(t)
	gw := newRevocationGateway(t, up.URL, governance.NewService(nil))
	root, rootSig := mintRoot(t, gw)
	banVia(t, gw, rootSig)

	// A holder chains caveats from the signature without the gateway.
	tok := root
	for depth := 1; depth <= 4; depth++ {
		child, err := gw.macaroonSvc.DelegateWithoutVerify(tok, []string{"scope = api:read"})
		if err != nil {
			t.Fatal(err)
		}
		tok = gw.macaroonSvc.Encode(child)
		if rec := bearerGet(t, gw, "/api/x", tok); rec.Code != http.StatusUnauthorized {
			t.Fatalf("depth %d offline descendant of a banned token: want 401, got %d", depth, rec.Code)
		}
	}
}

func TestRevocation_DelegateRefusesBannedTokenAndItsDescendants(t *testing.T) {
	up, _ := countingUpstream(t)
	gw := newRevocationGateway(t, up.URL, governance.NewService(nil))
	root, rootSig := mintRoot(t, gw)
	child := delegateVia(t, gw, root, "scope = api:read")
	banVia(t, gw, rootSig)
	nodesBefore := len(gw.governance.GetGraph().Nodes)

	for name, parent := range map[string]string{"banned token": root, "descendant of banned token": child.Token} {
		rec := postJSON(t, gw, "/api/capability/delegate", map[string]any{"parentToken": parent, "caveats": []string{"scope = api:read:x"}}, false)
		if rec.Code != http.StatusForbidden {
			t.Fatalf("delegate from %s: want 403, got %d", name, rec.Code)
		}
		var body map[string]any
		_ = json.NewDecoder(rec.Body).Decode(&body)
		if _, minted := body["token"]; minted {
			t.Fatalf("delegate from %s returned a token", name)
		}
		if body["code"] != "TOKEN_BANNED" {
			t.Fatalf("delegate from %s: want code TOKEN_BANNED, got %v", name, body["code"])
		}
	}
	if n := len(gw.governance.GetGraph().Nodes); n != nodesBefore {
		t.Fatalf("refused delegation registered a token: %d nodes, want %d", n, nodesBefore)
	}
}

func TestRevocation_BanningAChildLeavesParentAndSiblings(t *testing.T) {
	up, _ := countingUpstream(t)
	gw := newRevocationGateway(t, up.URL, governance.NewService(nil))
	root, _ := mintRoot(t, gw)
	a := delegateVia(t, gw, root, "scope = api:read")
	// Siblings differ in their caveats: two delegations of one parent with
	// the same caveats are the same token.
	b := delegateVia(t, gw, root, "scope = api:write")
	aChild := delegateVia(t, gw, a.Token, "expires = 30m")
	banVia(t, gw, a.Signature)

	if rec := bearerGet(t, gw, "/api/x", root); rec.Code != http.StatusOK {
		t.Fatalf("parent of banned child: want 200, got %d", rec.Code)
	}
	if rec := bearerGet(t, gw, "/api/x", b.Token); rec.Code != http.StatusOK {
		t.Fatalf("sibling of banned child: want 200, got %d", rec.Code)
	}
	if rec := bearerGet(t, gw, "/api/x", a.Token); rec.Code != http.StatusUnauthorized {
		t.Fatalf("banned child: want 401, got %d", rec.Code)
	}
	if rec := bearerGet(t, gw, "/api/x", aChild.Token); rec.Code != http.StatusUnauthorized {
		t.Fatalf("descendant of banned child: want 401, got %d", rec.Code)
	}
	// Delegating from the parent or the sibling still works.
	delegateVia(t, gw, root, "scope = api:read:y")
	delegateVia(t, gw, b.Token, "scope = api:write:y")
}

func TestRevocation_BuiltInEndpointsRefuseDescendantOfBannedToken(t *testing.T) {
	up, _ := countingUpstream(t)
	gw := newRevocationGateway(t, up.URL, governance.NewService(nil))
	root, rootSig := mintRoot(t, gw)
	child := delegateVia(t, gw, root, "scope = api:*")
	banVia(t, gw, rootSig)

	if rec := bearerGet(t, gw, "/api/capability/ping", child.Token); rec.Code != http.StatusForbidden {
		t.Fatalf("ping with descendant of banned token: want 403, got %d", rec.Code)
	}
	if rec := bearerGet(t, gw, "/api/capability/admin", child.Token); rec.Code != http.StatusForbidden {
		t.Fatalf("admin with descendant of banned token: want 403, got %d", rec.Code)
	}
	rec := postJSON(t, gw, "/api/capability/validate", map[string]string{"token": child.Token}, false)
	var body map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&body)
	if rec.Code != http.StatusUnauthorized || body["valid"] != false {
		t.Fatalf("validate descendant of banned token: want 401 valid=false, got %d %v", rec.Code, body["valid"])
	}
	if info := gw.governance.GetTokenInfoBySignature(child.Signature); info == nil || info.Status != "banned" {
		t.Fatalf("token info of descendant of banned token: want status banned, got %+v", info)
	}
}

func TestRevocation_MCPRouteRefusesDescendantOfBannedToken(t *testing.T) {
	up, hits := countingUpstream(t)
	gw := newRevocationGateway(t, up.URL, governance.NewService(nil))
	root, rootSig := mintRoot(t, gw)
	child := delegateVia(t, gw, root, "scope = api:read")
	call := `{"jsonrpc":"2.0","id":1,"method":"tools/call","params":{"name":"echo","arguments":{}}}`
	send := func() int {
		req := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(call))
		req.Header.Set("Authorization", "Bearer "+child.Token)
		req.Header.Set("Content-Type", "application/json")
		rec := httptest.NewRecorder()
		gw.ServeHTTP(rec, req)
		return rec.Code
	}
	if code := send(); code != http.StatusOK {
		t.Fatalf("MCP call before ban: want 200, got %d", code)
	}
	banVia(t, gw, rootSig)
	before := hits.Load()
	if code := send(); code != http.StatusUnauthorized {
		t.Fatalf("MCP call with descendant of banned token: want 401, got %d", code)
	}
	if hits.Load() != before {
		t.Fatalf("refused MCP call reached the upstream")
	}
}

func TestRevocation_AgentSimulateRefusesDescendantOfBannedToken(t *testing.T) {
	up, hits := countingUpstream(t)
	gw := newTestGateway(t, &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	})
	root, rootSig := mintRoot(t, gw)
	child := delegateVia(t, gw, root, "scope = api:read")
	banVia(t, gw, rootSig)
	staged := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "gated", "kind": "capability", "path_prefix": "/gated", "upstream_url": up.URL, "scope": "api:read",
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID, "authorization": "Bearer " + child.Token})
	var pack map[string]any
	decode(t, sim, &pack)
	if pack["decision"] != "denied" || hits.Load() != 0 {
		t.Fatalf("simulate with descendant of banned token: decision %v, upstream hits %d", pack["decision"], hits.Load())
	}
}

// l402Signature returns the signature of a binary challenge macaroon.
func l402Signature(t *testing.T, gw *Gateway, mac string) string {
	t.Helper()
	m, err := gw.macaroonSvc.Verify(mac)
	if err != nil {
		t.Fatalf("verify challenge macaroon: %v", err)
	}
	return m.Signature
}

func TestRevocation_L402BannedProofRefusedAndNotConsumed(t *testing.T) {
	up, hits := countingUpstream(t)
	gw := newRevocationGateway(t, up.URL, governance.NewService(nil))
	paid := payL402At(t, gw, "/cheap/x")
	sig := l402Signature(t, gw, paid.macaroon)
	banVia(t, gw, sig)

	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusUnauthorized {
		t.Fatalf("banned L402 proof: want 401, got %d", code)
	}
	// A proof the holder narrowed from the banned macaroon is refused too.
	narrowed := appendBinaryCaveats(t, paid.macaroon, "methods = GET")
	if code := useL402(t, gw, "/cheap/x", "L402 "+narrowed+":"+paid.preimage); code != http.StatusUnauthorized {
		t.Fatalf("proof narrowed from a banned macaroon: want 401, got %d", code)
	}
	if hits.Load() != 0 {
		t.Fatalf("a banned proof reached the upstream")
	}

	// The refused proof was not consumed: once unbanned it is accepted once.
	if !gw.governance.Unban(sig) {
		t.Fatalf("unban reported no ban")
	}
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusOK {
		t.Fatalf("proof after unban: want 200 (it was not consumed while banned), got %d", code)
	}
}

func TestRevocation_MockPayRefusesBannedChallenge(t *testing.T) {
	up, _ := countingUpstream(t)
	gw := newRevocationGateway(t, up.URL, governance.NewService(nil))
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/cheap/x", nil))
	mac := extractL402MacaroonFromChallenge(t, rec.Header().Get("WWW-Authenticate"))
	var challenge map[string]any
	_ = json.NewDecoder(rec.Body).Decode(&challenge)
	banVia(t, gw, l402Signature(t, gw, mac))
	pay := postJSON(t, gw, "/api/l402/mock-pay", map[string]any{"payment_hash": challenge["payment_hash"], "macaroon": mac}, true)
	if pay.Code != http.StatusForbidden {
		t.Fatalf("mock pay of banned challenge: want 403, got %d", pay.Code)
	}
}

func TestRevocation_BanOnOneGatewayHonouredOnAnotherSharingRedis(t *testing.T) {
	mr := miniredis.RunT(t)
	up, _ := countingUpstream(t)
	gwA := newRevocationGateway(t, up.URL, newRedisGovernance(t, mr.Addr()))
	gwB := newRevocationGateway(t, up.URL, newRedisGovernance(t, mr.Addr()))

	root, rootSig := mintRoot(t, gwA)
	child := delegateVia(t, gwA, root, "scope = api:read")
	banVia(t, gwA, rootSig)

	// gwB has not looked these tokens up before, so it reads the store now.
	if rec := bearerGet(t, gwB, "/api/x", child.Token); rec.Code != http.StatusUnauthorized {
		t.Fatalf("descendant of a token banned on another gateway: want 401, got %d", rec.Code)
	}
	if rec := bearerGet(t, gwB, "/api/x", root); rec.Code != http.StatusUnauthorized {
		t.Fatalf("token banned on another gateway: want 401, got %d", rec.Code)
	}
	if rec := postJSON(t, gwB, "/api/capability/delegate", map[string]any{"parentToken": child.Token, "caveats": []string{"scope = api:read:z"}}, false); rec.Code != http.StatusForbidden {
		t.Fatalf("delegate on another gateway from a descendant of a banned token: want 403, got %d", rec.Code)
	}

	// L402 proofs too.
	paid := payL402At(t, gwA, "/cheap/x")
	banVia(t, gwA, l402Signature(t, gwA, paid.macaroon))
	if code := useL402(t, gwB, "/cheap/x", paid.authorization()); code != http.StatusUnauthorized {
		t.Fatalf("L402 proof banned on another gateway: want 401, got %d", code)
	}
}

func TestRevocation_StoreUnavailableFailsClosed(t *testing.T) {
	mr := miniredis.RunT(t)
	up, hits := countingUpstream(t)
	gw := newRevocationGateway(t, up.URL, newRedisGovernance(t, mr.Addr()))
	root, _ := mintRoot(t, gw)
	paid := payL402At(t, gw, "/cheap/x")
	nodesBefore := len(gw.governance.GetGraph().Nodes)
	mr.Close()

	if rec := bearerGet(t, gw, "/api/x", root); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("capability route with store down: want 503, got %d", rec.Code)
	}
	if rec := bearerGet(t, gw, "/api/capability/ping", root); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("ping with store down: want 503, got %d", rec.Code)
	}
	if rec := postJSON(t, gw, "/api/capability/validate", map[string]string{"token": root}, false); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("validate with store down: want 503, got %d", rec.Code)
	}
	if rec := postJSON(t, gw, "/api/capability/delegate", map[string]any{"parentToken": root, "caveats": []string{"scope = api:read"}}, false); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("delegate with store down: want 503, got %d", rec.Code)
	}
	if n := len(gw.governance.GetGraph().Nodes); n != nodesBefore {
		t.Fatalf("delegate with store down registered a token")
	}
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusServiceUnavailable {
		t.Fatalf("L402 proof with store down: want 503, got %d", code)
	}
	if rec := postJSON(t, gw, "/api/governance/ban", map[string]string{"tokenSignature": "ab12", "reason": "x"}, true); rec.Code != http.StatusServiceUnavailable {
		t.Fatalf("ban with store down: want 503, got %d", rec.Code)
	}
	if hits.Load() != 0 {
		t.Fatalf("a request reached the upstream while revocation state was unreadable")
	}

	// The L402 proof was not consumed while the store was down.
	if err := mr.Restart(); err != nil {
		t.Fatalf("restart redis: %v", err)
	}
	if code := useL402(t, gw, "/cheap/x", paid.authorization()); code != http.StatusOK {
		t.Fatalf("L402 proof after store recovered: want 200, got %d", code)
	}
}

func TestRevocation_InMemoryStoreKeepsSingleInstanceBehaviour(t *testing.T) {
	up, _ := countingUpstream(t)
	gwA := newRevocationGateway(t, up.URL, governance.NewService(nil))
	gwB := newRevocationGateway(t, up.URL, governance.NewService(nil))
	root, rootSig := mintRoot(t, gwA)
	banVia(t, gwA, rootSig)
	if rec := bearerGet(t, gwA, "/api/x", root); rec.Code != http.StatusUnauthorized {
		t.Fatalf("banned token on the gateway that banned it: want 401, got %d", rec.Code)
	}
	// Separate in-memory stores share nothing.
	if rec := bearerGet(t, gwB, "/api/x", root); rec.Code != http.StatusOK {
		t.Fatalf("in-memory store on another process: want 200, got %d", rec.Code)
	}
	if !gwA.governance.Unban(rootSig) {
		t.Fatal("unban reported no ban")
	}
	if rec := bearerGet(t, gwA, "/api/x", root); rec.Code != http.StatusOK {
		t.Fatalf("after unban: want 200, got %d", rec.Code)
	}
}
