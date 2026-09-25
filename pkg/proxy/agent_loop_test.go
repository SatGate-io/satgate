package proxy

import (
	"context"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/governance"
	"github.com/satgate-io/satgate/pkg/lightning"
)

func TestAgentLoopStagesSimulatesAndPromotesARealHTTPCall(t *testing.T) {
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()

	gw := newTestGateway(t, &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	})

	gw.setAgentIssuer("https://issuer.example")

	stage := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name":         "orders",
		"kind":         "observe",
		"path_prefix":  "/orders",
		"upstream_url": upstream.URL,
		"surface":      "http",
	})
	if stage.Code != http.StatusCreated {
		t.Fatalf("stage %d %s", stage.Code, stage.Body.String())
	}
	var staged agentPolicy
	decode(t, stage, &staged)

	before := agentGet(t, gw, "/orders")
	if before.Code != http.StatusNotFound {
		t.Fatalf("staged policy went live early: %d", before.Code)
	}

	promotedEarly := agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": staged.ID})
	if promotedEarly.Code != http.StatusConflict {
		t.Fatalf("promote before simulate: %d", promotedEarly.Code)
	}

	sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": staged.ID, "path": "/orders"})
	if sim.Code != http.StatusOK {
		t.Fatalf("simulate %d %s", sim.Code, sim.Body.String())
	}
	var pack map[string]any
	decode(t, sim, &pack)
	if pack["decision"] != "allowed" || pack["upstream_contacted"] != true || intFrom(pack["upstream_status"]) != http.StatusOK {
		t.Fatalf("pack did not record the real call: %+v", pack)
	}
	if pack["provider_price_status"] != "UNKNOWN" || pack["settlement"] != false || pack["trusted_issuer_valid"] == true || pack["issuer"] == "https://satgate.io" {
		t.Fatalf("pack overclaimed price, settlement, or issuer: %+v", pack)
	}
	if hits != 1 {
		t.Fatalf("upstream hits %d, want 1", hits)
	}
	if !verifierAccepts(t, pack) {
		t.Fatal("public verifier rejected the pack")
	}
	jwksRec := httptest.NewRecorder()
	gw.ServeHTTP(jwksRec, httptest.NewRequest(http.MethodGet, "/.well-known/satgate-agent-jwks.json", nil))
	if jwksRec.Code != http.StatusOK {
		t.Fatalf("jwks %d %s", jwksRec.Code, jwksRec.Body.String())
	}
	pinned := verifierResult(t, pack, jwksRec.Body.Bytes(), true)
	if pinned["valid"] != true || pinned["trusted_issuer_valid"] != true {
		t.Fatalf("pinned issuer was not trusted: %s", jwksRec.Body.String())
	}
	var jwks map[string]any
	if err := json.Unmarshal(jwksRec.Body.Bytes(), &jwks); err != nil {
		t.Fatal(err)
	}
	keys := jwks["keys"].([]any)
	keys[0].(map[string]any)["kid"] = "not-this-key"
	wrong, _ := json.Marshal(jwks)
	rejected := verifierResult(t, pack, wrong, true)
	if rejected["valid"] == true || rejected["trusted_issuer_valid"] == true {
		t.Fatalf("wrong pin was trusted: %+v", rejected)
	}

	promoted := agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": staged.ID})
	if promoted.Code != http.StatusOK {
		t.Fatalf("promote %d %s", promoted.Code, promoted.Body.String())
	}
	live := agentGet(t, gw, "/orders")
	if live.Code != http.StatusOK || !strings.Contains(live.Body.String(), `"ok":true`) {
		t.Fatalf("live request after promote: %d %s", live.Code, live.Body.String())
	}
	if hits != 2 {
		t.Fatalf("live request did not reach upstream, hits %d", hits)
	}
	packID := live.Header().Get("SatGate-Evidence-Pack")
	if packID == "" {
		t.Fatal("live request did not return a pack")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/agent/pack/"+packID, nil)
	req.Header.Set("X-Admin-Token", "admin-secret")
	got := httptest.NewRecorder()
	gw.ServeHTTP(got, req)
	var livePack map[string]any
	decode(t, got, &livePack)
	if livePack["decision"] != "allowed" || livePack["upstream_contacted"] != true || livePack["settlement"] != false || livePack["provider_price_status"] != "UNKNOWN" {
		t.Fatalf("live pack: %+v", livePack)
	}
	if !verifierAccepts(t, livePack) {
		t.Fatal("public verifier rejected the live pack")
	}
}

func TestAgentLoopDenyDoesNotContactUpstreamAndMCPRecordsTheTool(t *testing.T) {
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = io.Copy(io.Discard, r.Body)
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	gw := newTestGateway(t, &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	})

	denied := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "blocked", "kind": "deny", "path_prefix": "/blocked", "surface": "http",
	})
	var denyPolicy agentPolicy
	decode(t, denied, &denyPolicy)
	sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": denyPolicy.ID})
	var pack map[string]any
	decode(t, sim, &pack)
	if pack["decision"] != "denied" || pack["upstream_contacted"] != false || hits != 0 {
		t.Fatalf("deny contacted upstream or was allowed: %+v hits %d", pack, hits)
	}
	promoted := agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": denyPolicy.ID})
	if promoted.Code != http.StatusOK {
		t.Fatalf("deny promote: %d %s", promoted.Code, promoted.Body.String())
	}
	blocked := agentGet(t, gw, "/blocked")
	if blocked.Code != http.StatusForbidden || hits != 0 {
		t.Fatalf("live deny: %d hits %d", blocked.Code, hits)
	}
	if blocked.Header().Get("SatGate-Evidence-Pack") == "" {
		t.Fatal("live deny did not return a pack")
	}

	mcp := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "tools", "kind": "observe", "path_prefix": "/mcp", "upstream_url": upstream.URL, "surface": "mcp",
	})
	var mcpPolicy agentPolicy
	decode(t, mcp, &mcpPolicy)
	body := `{"jsonrpc":"2.0","method":"tools/call","params":{"name":"get_portfolio"}}`
	mcpSim := agentPost(t, gw, "/api/agent/simulate", map[string]string{
		"policy_id": mcpPolicy.ID, "path": "/mcp", "method": "POST", "body": body,
	})
	var mcpPack map[string]any
	decode(t, mcpSim, &mcpPack)
	if mcpPack["route_or_tool"] != "get_portfolio" || mcpPack["upstream_contacted"] != true {
		t.Fatalf("mcp simulation: %+v", mcpPack)
	}
	if !verifierAccepts(t, mcpPack) {
		t.Fatal("public verifier rejected the mcp pack")
	}
	promotedMCP := agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": mcpPolicy.ID})
	if promotedMCP.Code != http.StatusOK {
		t.Fatalf("mcp promote: %d %s", promotedMCP.Code, promotedMCP.Body.String())
	}
	liveReq := httptest.NewRequest(http.MethodPost, "/mcp", strings.NewReader(body))
	liveRec := httptest.NewRecorder()
	gw.ServeHTTP(liveRec, liveReq)
	if liveRec.Code != http.StatusOK || hits != 2 {
		t.Fatalf("live mcp: %d hits %d", liveRec.Code, hits)
	}
	packID := liveRec.Header().Get("SatGate-Evidence-Pack")
	got := httptest.NewRecorder()
	fetch := httptest.NewRequest(http.MethodGet, "/api/agent/pack/"+packID, nil)
	fetch.Header.Set("X-Admin-Token", "admin-secret")
	gw.ServeHTTP(got, fetch)
	var livePack map[string]any
	decode(t, got, &livePack)
	if livePack["route_or_tool"] != "get_portfolio" || livePack["upstream_contacted"] != true {
		t.Fatalf("live mcp pack: %+v", livePack)
	}
}

func TestAgentLoopChargeDoesNotInventAPrice(t *testing.T) {
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	gw := newTestGateway(t, &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	})
	staged := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "paid", "kind": "charge", "path_prefix": "/paid", "upstream_url": upstream.URL, "surface": "http",
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID, "path": "/paid"})
	var pack map[string]any
	decode(t, sim, &pack)
	body := sim.Body.String()
	if pack["decision"] != "denied" || pack["upstream_contacted"] != false || pack["promote_block"] != "price_unknown" || pack["settlement"] != false || hits != 0 {
		t.Fatalf("charge simulation invented a call or a price: %+v hits %d", pack, hits)
	}
	if strings.Contains(body, "amount_sats") || strings.Contains(body, "invoice") {
		t.Fatalf("charge simulation contained a price: %s", body)
	}
	promoted := agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": policy.ID})
	if promoted.Code != http.StatusConflict || !strings.Contains(promoted.Body.String(), "price_unknown") {
		t.Fatalf("charge promote: %d %s", promoted.Code, promoted.Body.String())
	}
	if strings.Contains(promoted.Body.String(), "amount_sats") || strings.Contains(promoted.Body.String(), "invoice") {
		t.Fatalf("promote invented a price: %s", promoted.Body.String())
	}
	live := agentGet(t, gw, "/paid")
	if live.Code == http.StatusPaymentRequired || strings.Contains(live.Body.String(), "amount_sats") || hits != 0 {
		t.Fatalf("charge went live: %d %s hits %d", live.Code, live.Body.String(), hits)
	}
}

func TestAgentLoopCapabilityUsesARealTokenAndLeavesItOutOfThePack(t *testing.T) {
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	gw := newTestGateway(t, &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	})
	mint := agentPost(t, gw, "/api/capability/mint", map[string]string{"scope": "api:read", "duration": "1h"})
	if mint.Code != http.StatusOK {
		t.Fatalf("mint %d %s", mint.Code, mint.Body.String())
	}
	var minted struct {
		Token string `json:"token"`
	}
	decode(t, mint, &minted)
	if minted.Token == "" {
		t.Fatal("mint returned no token")
	}
	staged := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "gated", "kind": "capability", "path_prefix": "/gated", "upstream_url": upstream.URL, "scope": "api:read",
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	missing := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID})
	var missingPack map[string]any
	decode(t, missing, &missingPack)
	if missingPack["decision"] != "denied" || missingPack["upstream_contacted"] != false || hits != 0 {
		t.Fatalf("missing token contacted upstream: %+v", missingPack)
	}
	bad := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID, "authorization": "Bearer not-a-token"})
	var badPack map[string]any
	decode(t, bad, &badPack)
	if badPack["decision"] != "denied" || hits != 0 {
		t.Fatalf("bad token was admitted: %+v hits %d", badPack, hits)
	}
	sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID, "authorization": "Bearer " + minted.Token})
	var pack map[string]any
	decode(t, sim, &pack)
	if pack["decision"] != "allowed" || pack["upstream_contacted"] != true || hits != 1 {
		t.Fatalf("valid token did not reach upstream: %+v hits %d", pack, hits)
	}
	if strings.Contains(sim.Body.String(), minted.Token) {
		t.Fatal("pack contained the raw token")
	}
	if pack["identity_fingerprint"] == "" || pack["identity_fingerprint"] == minted.Token {
		t.Fatalf("identity fingerprint missing or raw: %+v", pack["identity_fingerprint"])
	}
	if !verifierAccepts(t, pack) {
		t.Fatal("public verifier rejected the capability pack")
	}
	promoted := agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": policy.ID})
	if promoted.Code != http.StatusOK {
		t.Fatalf("promote %d %s", promoted.Code, promoted.Body.String())
	}
	open := agentGet(t, gw, "/gated")
	if open.Code != http.StatusUnauthorized {
		t.Fatalf("live route allowed a missing token: %d %s", open.Code, open.Body.String())
	}
	req := httptest.NewRequest(http.MethodGet, "/gated", nil)
	req.Header.Set("Authorization", "Bearer "+minted.Token)
	live := httptest.NewRecorder()
	gw.ServeHTTP(live, req)
	if live.Code != http.StatusOK || hits != 2 {
		t.Fatalf("live token call: %d %s hits %d", live.Code, live.Body.String(), hits)
	}
}

func TestAgentLoopIssuerComesFromThePolicyNotSatGate(t *testing.T) {
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	gw := newTestGateway(t, &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	})
	bad := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "bad", "kind": "observe", "path_prefix": "/bad", "upstream_url": upstream.URL, "issuer": "http://issuer.example",
	})
	if bad.Code != http.StatusBadRequest {
		t.Fatalf("http issuer accepted: %d %s", bad.Code, bad.Body.String())
	}
	staged := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "orders", "kind": "observe", "path_prefix": "/orders", "upstream_url": upstream.URL, "issuer": "https://issuer.example",
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID})
	var pack map[string]any
	decode(t, sim, &pack)
	if pack["issuer"] != "https://issuer.example" || pack["issuer"] == "https://satgate.io" {
		t.Fatalf("issuer was not the one the agent set: %+v", pack["issuer"])
	}
}

func TestAgentLoopPackAndKeySurviveANewProcess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SATGATE_AGENT_PACK_DIR", dir)
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer upstream.Close()
	cfg := &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	}
	gw := newTestGateway(t, cfg)
	gw.setAgentIssuer("https://issuer.example")
	staged := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "orders", "kind": "observe", "path_prefix": "/orders", "upstream_url": upstream.URL,
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID})
	var pack map[string]any
	decode(t, sim, &pack)
	id, _ := pack["evidence_pack_id"].(string)
	first := agentJWKSKid(t, gw)
	gw.agent.mu.Lock()
	delete(gw.agent.packs, id)
	gw.agent.mu.Unlock()
	next := newTestGateway(t, cfg)
	if agentJWKSKid(t, next) != first {
		t.Fatal("restart minted a new signing key")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/agent/pack/"+id, nil)
	req.Header.Set("X-Admin-Token", "admin-secret")
	got := httptest.NewRecorder()
	next.ServeHTTP(got, req)
	if got.Code != http.StatusOK {
		t.Fatalf("pack did not survive: %d %s", got.Code, got.Body.String())
	}
	var loaded map[string]any
	decode(t, got, &loaded)
	jwks := httptest.NewRecorder()
	next.ServeHTTP(jwks, httptest.NewRequest(http.MethodGet, "/.well-known/satgate-agent-jwks.json", nil))
	pinned := verifierResult(t, loaded, jwks.Body.Bytes(), true)
	if pinned["valid"] != true || pinned["trusted_issuer_valid"] != true {
		t.Fatalf("restarted pin was not trusted: %+v", pinned)
	}
	keyRaw, err := os.ReadFile(filepath.Join(dir, "signing-key.json"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got.Body.Bytes()), string(keyRaw)) || strings.Contains(got.Body.String(), "private") {
		t.Fatal("pack contained the signing key")
	}
}

func TestAgentLoopPromotedRouteSurvivesANewProcess(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SATGATE_AGENT_PACK_DIR", dir)
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()
	cfg := &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	}
	gw := newTestGateway(t, cfg)
	staged := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "orders", "kind": "observe", "path_prefix": "/orders", "upstream_url": upstream.URL, "issuer": "https://issuer.example",
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID})
	if sim.Code != http.StatusOK {
		t.Fatalf("simulate %d %s", sim.Code, sim.Body.String())
	}
	promoted := agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": policy.ID})
	if promoted.Code != http.StatusOK {
		t.Fatalf("promote %d %s", promoted.Code, promoted.Body.String())
	}
	next := newTestGateway(t, &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	})
	live := agentGet(t, next, "/orders")
	if live.Code != http.StatusOK || !strings.Contains(live.Body.String(), `"ok":true`) || live.Header().Get("SatGate-Evidence-Pack") == "" {
		t.Fatalf("promoted route did not survive: %d %s", live.Code, live.Body.String())
	}
	if hits < 2 {
		t.Fatalf("restarted route did not reach upstream, hits %d", hits)
	}
}

func TestAgentLoopPromoteAfterRestartUsesTheSavedPack(t *testing.T) {
	dir := t.TempDir()
	t.Setenv("SATGATE_AGENT_PACK_DIR", dir)
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()
	cfg := func() *config.Config {
		return &config.Config{
			Admin: config.AdminConfig{Token: "admin-secret"},
			Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
		}
	}
	gw := newTestGateway(t, cfg())
	staged := agentPost(t, gw, "/api/agent/policy", map[string]string{
		"name": "orders", "kind": "observe", "path_prefix": "/orders", "upstream_url": upstream.URL,
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	if agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID}).Code != http.StatusOK {
		t.Fatal("simulate failed")
	}
	next := newTestGateway(t, cfg())
	promoted := agentPost(t, next, "/api/agent/promote", map[string]string{"policy_id": policy.ID})
	if promoted.Code != http.StatusOK {
		t.Fatalf("promote after restart: %d %s", promoted.Code, promoted.Body.String())
	}
	live := agentGet(t, next, "/orders")
	if live.Code != http.StatusOK || hits < 2 {
		t.Fatalf("live after restarted promote: %d hits %d", live.Code, hits)
	}
}

func agentJWKSKid(t *testing.T, gw *Gateway) string {
	t.Helper()
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, httptest.NewRequest(http.MethodGet, "/.well-known/satgate-agent-jwks.json", nil))
	var doc struct {
		Keys []struct {
			Kid string `json:"kid"`
		} `json:"keys"`
	}
	decode(t, rec, &doc)
	if len(doc.Keys) != 1 || doc.Keys[0].Kid == "" {
		t.Fatalf("jwks: %s", rec.Body.String())
	}
	return doc.Keys[0].Kid
}

func agentPost(t *testing.T, gw *Gateway, path string, body any) *httptest.ResponseRecorder {
	t.Helper()
	raw, err := json.Marshal(body)
	if err != nil {
		t.Fatal(err)
	}
	req := httptest.NewRequest(http.MethodPost, path, strings.NewReader(string(raw)))
	req.Header.Set("X-Admin-Token", "admin-secret")
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	return rec
}

func agentGet(t *testing.T, gw *Gateway, path string) *httptest.ResponseRecorder {
	t.Helper()
	req := httptest.NewRequest(http.MethodGet, path, nil)
	rec := httptest.NewRecorder()
	gw.ServeHTTP(rec, req)
	return rec
}

func decode(t *testing.T, rec *httptest.ResponseRecorder, dest any) {
	t.Helper()
	if err := json.Unmarshal(rec.Body.Bytes(), dest); err != nil {
		t.Fatalf("decode %s: %v", rec.Body.String(), err)
	}
}

func intFrom(v any) int {
	switch n := v.(type) {
	case float64:
		return int(n)
	case int:
		return n
	default:
		return 0
	}
}

func verifierAccepts(t *testing.T, pack map[string]any) bool {
	t.Helper()
	result := verifierResult(t, pack, nil, false)
	return result["valid"] == true && result["trusted_issuer_valid"] == false
}

func verifierResult(t *testing.T, pack map[string]any, jwks []byte, require bool) map[string]any {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "pack.json")
	raw, err := json.Marshal(pack)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, raw, 0o600); err != nil {
		t.Fatal(err)
	}
	args := []string{filepath.Join("..", "..", "tools", "verify_evidence_pack.py"), path}
	if jwks != nil {
		jwksPath := filepath.Join(dir, "jwks.json")
		if err := os.WriteFile(jwksPath, jwks, 0o600); err != nil {
			t.Fatal(err)
		}
		args = append(args, "--jwks-file", jwksPath)
	}
	if require {
		args = append(args, "--require-trusted-issuer")
	}
	cmd := exec.Command("python3", args...)
	out, err := cmd.CombinedOutput()
	var result map[string]any
	if jsonErr := json.Unmarshal(out, &result); jsonErr != nil {
		t.Fatalf("verifier output %s: %v", out, jsonErr)
	}
	if require && err != nil && result["valid"] == true {
		t.Fatalf("verifier failed: %v\n%s", err, out)
	}
	if !require && err != nil {
		t.Fatalf("verifier failed: %v\n%s", err, out)
	}
	return result
}

func TestAgentLoopIdentityChecksASignedToken(t *testing.T) {
	pub, priv, err := ed25519.GenerateKey(nil)
	if err != nil {
		t.Fatal(err)
	}
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_ = json.NewEncoder(w).Encode(map[string]any{"keys": []any{map[string]any{
			"kty": "OKP", "crv": "Ed25519", "kid": "idp-1", "x": base64.RawURLEncoding.EncodeToString(pub),
		}}})
	}))
	defer jwks.Close()
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		hits++
		_, _ = w.Write([]byte(`{"ok":true}`))
	}))
	defer upstream.Close()
	gw := newTestGateway(t, &config.Config{Admin: config.AdminConfig{Token: "admin-secret"}})
	staged := agentPost(t, gw, "/api/agent/policy", map[string]any{
		"name": "who", "kind": "identity", "path_prefix": "/who", "upstream_url": upstream.URL,
		"identity_issuer": "https://idp.example", "identity_jwks": jwks.URL, "issuer": "https://issuer.example",
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	bad := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID, "authorization": "Bearer not-a-jwt"})
	if bad.Code != http.StatusOK || hits != 0 || strings.Contains(bad.Body.String(), "not-a-jwt") {
		t.Fatalf("bad identity reached upstream: %d %s hits %d", bad.Code, bad.Body.String(), hits)
	}
	token := signTestJWT(t, priv, "idp-1", "https://idp.example")
	good := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID, "authorization": "Bearer " + token})
	var pack map[string]any
	decode(t, good, &pack)
	if pack["identity_fingerprint"] == "" || pack["upstream_contacted"] != true || strings.Contains(good.Body.String(), token) || hits != 1 {
		t.Fatalf("identity simulation: %+v hits %d", pack, hits)
	}
	if agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": policy.ID}).Code != http.StatusOK {
		t.Fatal("identity promote failed")
	}
	missing := agentGet(t, gw, "/who")
	live := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/who", nil)
	req.Header.Set("Authorization", "Bearer "+token)
	gw.ServeHTTP(live, req)
	if missing.Code == http.StatusOK || live.Code != http.StatusOK || hits != 2 || strings.Contains(live.Body.String(), token) {
		t.Fatalf("live identity: missing %d live %d hits %d", missing.Code, live.Code, hits)
	}
}

func TestAgentLoopIdentityJWKSDoesNotFollowRedirectsOrPrivateDials(t *testing.T) {
	var followed int
	target := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		followed++
		w.WriteHeader(http.StatusOK)
	}))
	defer target.Close()
	jwks := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		http.Redirect(w, r, target.URL, http.StatusFound)
	}))
	defer jwks.Close()
	gw := newTestGateway(t, &config.Config{Admin: config.AdminConfig{Token: "admin-secret"}})
	if _, err := gw.fetchIdentityKeys(jwks.URL); err == nil || followed != 0 {
		t.Fatalf("redirect was followed: err %v hits %d", err, followed)
	}
	blocked := &Gateway{config: &config.Config{}}
	if _, err := blocked.identityHTTPClient().Transport.(*http.Transport).DialContext(context.Background(), "tcp", "127.0.0.1:1"); err == nil {
		t.Fatal("private dial was allowed")
	}
}

func TestAgentLoopExplicitPriceDoesNotUseTheMockRail(t *testing.T) {
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer upstream.Close()
	gw := newTestGateway(t, &config.Config{Admin: config.AdminConfig{Token: "admin-secret"}})
	staged := agentPost(t, gw, "/api/agent/policy", map[string]any{
		"name": "paid", "kind": "charge", "path_prefix": "/paid", "upstream_url": upstream.URL, "price_sats": 42,
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	sim := agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID})
	promoted := agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": policy.ID})
	if !strings.Contains(sim.Body.String(), "payment_rail_unavailable") || promoted.Code != http.StatusConflict || hits != 0 {
		t.Fatalf("mock rail was treated as payment: sim %s promote %d %s", sim.Body.String(), promoted.Code, promoted.Body.String())
	}
	if strings.Contains(sim.Body.String(), "amount_sats") || strings.Contains(promoted.Body.String(), "invoice") || strings.Contains(promoted.Body.String(), "100") {
		t.Fatalf("explicit price invented an invoice: %s %s", sim.Body.String(), promoted.Body.String())
	}
}

func TestAgentLoopExplicitPriceIsTheInvoiceAmount(t *testing.T) {
	var hits int
	upstream := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) { hits++ }))
	defer upstream.Close()
	cfg := &config.Config{
		Admin: config.AdminConfig{Token: "admin-secret"},
		Cloud: &config.CloudConfig{SSRF: &config.CloudSSRFConfig{AllowPrivateIPs: true}},
	}
	gw, err := New(Options{Config: cfg, Macaroon: newTestMacaroonService(t), Governance: governance.NewService(nil), Lightning: quotedRail{}})
	if err != nil {
		t.Fatal(err)
	}
	staged := agentPost(t, gw, "/api/agent/policy", map[string]any{
		"name": "paid", "kind": "charge", "path_prefix": "/paid", "upstream_url": upstream.URL, "price_sats": 42,
	})
	var policy agentPolicy
	decode(t, staged, &policy)
	if agentPost(t, gw, "/api/agent/simulate", map[string]string{"policy_id": policy.ID}).Code != http.StatusOK || hits != 0 {
		t.Fatal("priced charge contacted upstream during simulation")
	}
	if agentPost(t, gw, "/api/agent/promote", map[string]string{"policy_id": policy.ID}).Code != http.StatusOK {
		t.Fatal("priced charge did not promote onto the quoted rail")
	}
	live := agentGet(t, gw, "/paid")
	var body map[string]any
	decode(t, live, &body)
	if live.Code != http.StatusPaymentRequired || body["amount_sats"] != float64(42) || hits != 0 || body["settlement"] == true {
		t.Fatalf("invoice was not the agent price: %d %+v hits %d", live.Code, body, hits)
	}
	packID := live.Header().Get("SatGate-Evidence-Pack")
	if packID == "" {
		t.Fatal("live charge did not return a pack")
	}
	req := httptest.NewRequest(http.MethodGet, "/api/agent/pack/"+packID, nil)
	req.Header.Set("X-Admin-Token", "admin-secret")
	got := httptest.NewRecorder()
	gw.ServeHTTP(got, req)
	var pack map[string]any
	decode(t, got, &pack)
	if pack["decision"] != "denied" || pack["decision_reason"] != "payment_required" || pack["settlement"] != false {
		t.Fatalf("live charge pack: %+v", pack)
	}
	if !verifierAccepts(t, pack) {
		t.Fatal("public verifier rejected the live charge pack")
	}
}

type quotedRail struct{}

func (quotedRail) CreateInvoice(amount int64, memo string) (*lightning.Invoice, error) {
	return &lightning.Invoice{Bolt11: "lnbc42", PaymentHash: "00112233445566778899aabbccddeeff", Amount: amount, Memo: memo}, nil
}
func (quotedRail) CheckPayment(string) (bool, error) { return false, nil }
func (quotedRail) GetBalance() (int64, error)        { return 0, nil }
func (quotedRail) GetInfo() (*lightning.NodeInfo, error) {
	return &lightning.NodeInfo{Alias: "quoted"}, nil
}

func signTestJWT(t *testing.T, priv ed25519.PrivateKey, kid, issuer string) string {
	t.Helper()
	header, _ := json.Marshal(map[string]string{"alg": "EdDSA", "kid": kid, "typ": "JWT"})
	payload, _ := json.Marshal(map[string]any{"iss": issuer, "sub": "agent-1", "exp": time.Now().Add(time.Hour).Unix()})
	h := base64.RawURLEncoding.EncodeToString(header)
	p := base64.RawURLEncoding.EncodeToString(payload)
	sig := ed25519.Sign(priv, []byte(h+"."+p))
	return h + "." + p + "." + base64.RawURLEncoding.EncodeToString(sig)
}
