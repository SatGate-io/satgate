package proxy

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/satgate-io/satgate/pkg/config"
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
