package proxy

import (
	"crypto/ed25519"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
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
	var pack agentPack
	decode(t, sim, &pack)
	if !pack.UpstreamContacted || pack.UpstreamStatus != http.StatusOK || pack.Decision != "allowed" {
		t.Fatalf("pack did not record the real call: %+v", pack)
	}
	if pack.ProviderPriceStatus != "UNKNOWN" || pack.Settlement || pack.IssuerPinned {
		t.Fatalf("pack overclaimed price, settlement, or issuer: %+v", pack)
	}
	if hits != 1 {
		t.Fatalf("upstream hits %d, want 1", hits)
	}
	pub, err := hex.DecodeString(pack.PublicKey)
	if err != nil {
		t.Fatal(err)
	}
	sig, err := hex.DecodeString(pack.Signature)
	if err != nil {
		t.Fatal(err)
	}
	if !ed25519.Verify(pub, []byte(pack.SignedPayload), sig) {
		t.Fatal("pack signature did not verify")
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
	var pack agentPack
	decode(t, sim, &pack)
	if pack.Decision != "denied" || pack.UpstreamContacted || hits != 0 {
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
	var mcpPack agentPack
	decode(t, mcpSim, &mcpPack)
	if mcpPack.Surface != "mcp" || mcpPack.Tool != "get_portfolio" || !mcpPack.UpstreamContacted {
		t.Fatalf("mcp simulation: %+v", mcpPack)
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
