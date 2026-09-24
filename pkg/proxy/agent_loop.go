package proxy

import (
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"

	"github.com/satgate-io/satgate/pkg/config"
)

// Agent loop: stage a policy, run it against a real upstream, sign that
// decision, then promote only when the agent asks. Staging does not go live.
type agentPolicy struct {
	ID          string `json:"policy_id"`
	Name        string `json:"name"`
	Kind        string `json:"kind"`
	PathPrefix  string `json:"path_prefix"`
	UpstreamURL string `json:"upstream_url"`
	Surface     string `json:"surface"`
	PackID      string `json:"pack_id,omitempty"`
	Promoted    bool   `json:"promoted"`
}

type agentPack struct {
	Schema              string `json:"schema"`
	ID                  string `json:"pack_id"`
	PolicyID            string `json:"policy_id"`
	Decision            string `json:"decision"`
	Surface             string `json:"surface"`
	Tool                string `json:"tool,omitempty"`
	UpstreamContacted   bool   `json:"upstream_contacted"`
	UpstreamStatus      int    `json:"upstream_status,omitempty"`
	ProviderPriceStatus string `json:"provider_price_status"`
	Settlement          bool   `json:"settlement"`
	IssuerPinned        bool   `json:"trusted_issuer_valid"`
	PublicKey           string `json:"public_key"`
	Signature           string `json:"signature"`
	SignedPayload       string `json:"signed_payload"`
}

type agentLoop struct {
	mu         sync.Mutex
	policies   map[string]*agentPolicy
	packs      map[string]*agentPack
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
}

func (g *Gateway) loop() *agentLoop {
	g.agentOnce.Do(func() {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			panic(err)
		}
		g.agent = &agentLoop{
			policies:   map[string]*agentPolicy{},
			packs:      map[string]*agentPack{},
			publicKey:  pub,
			privateKey: priv,
		}
	})
	return g.agent
}

func (g *Gateway) handleAgentLoop(w http.ResponseWriter, r *http.Request) {
	if !g.adminOK(r) {
		writeJSON(w, http.StatusUnauthorized, map[string]string{"error": "admin_required"})
		return
	}
	switch {
	case r.URL.Path == "/api/agent/policy" && r.Method == http.MethodPost:
		g.stageAgentPolicy(w, r)
	case r.URL.Path == "/api/agent/simulate" && r.Method == http.MethodPost:
		g.simulateAgentPolicy(w, r)
	case r.URL.Path == "/api/agent/promote" && r.Method == http.MethodPost:
		g.promoteAgentPolicy(w, r)
	case strings.HasPrefix(r.URL.Path, "/api/agent/pack/") && r.Method == http.MethodGet:
		g.getAgentPack(w, r)
	default:
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "not_found"})
	}
}

func (g *Gateway) adminOK(r *http.Request) bool {
	return g.validAdminToken(r.Header.Get("X-Admin-Token"))
}

func (g *Gateway) stageAgentPolicy(w http.ResponseWriter, r *http.Request) {
	var req agentPolicy
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_policy"})
		return
	}
	req.Kind = strings.ToLower(strings.TrimSpace(req.Kind))
	req.Surface = strings.ToLower(strings.TrimSpace(req.Surface))
	if req.Surface == "" {
		req.Surface = "http"
	}
	if req.Name == "" || !strings.HasPrefix(req.PathPrefix, "/") || strings.HasPrefix(req.PathPrefix, "/api/agent") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_policy"})
		return
	}
	switch req.Kind {
	case "public", "observe", "deny", "capability", "charge":
	default:
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_kind"})
		return
	}
	if req.Surface != "http" && req.Surface != "mcp" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "unsupported_surface"})
		return
	}
	if req.Kind != "deny" && req.UpstreamURL == "" {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "upstream_required"})
		return
	}
	req.ID = "pol_" + randHex(8)
	req.Promoted = false
	loop := g.loop()
	loop.mu.Lock()
	loop.policies[req.ID] = &req
	loop.mu.Unlock()
	writeJSON(w, http.StatusCreated, req)
}

func (g *Gateway) simulateAgentPolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PolicyID      string `json:"policy_id"`
		Path          string `json:"path"`
		Method        string `json:"method"`
		Body          string `json:"body"`
		Authorization string `json:"authorization"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_simulation"})
		return
	}
	loop := g.loop()
	loop.mu.Lock()
	policy := loop.policies[req.PolicyID]
	loop.mu.Unlock()
	if policy == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "policy_not_found"})
		return
	}
	if req.Path == "" {
		req.Path = policy.PathPrefix
	}
	if req.Method == "" {
		req.Method = http.MethodGet
	}
	decision := "allowed"
	contacted := false
	status := 0
	tool := ""
	if policy.Surface == "mcp" {
		tool = mcpTool(req.Body)
	}
	switch policy.Kind {
	case "deny":
		decision = "denied"
	case "charge":
		decision = "payment_required"
	case "capability":
		token := strings.TrimPrefix(req.Authorization, "Bearer ")
		if token == "" || token == req.Authorization {
			decision = "denied"
			break
		}
		if _, err := g.macaroonSvc.Verify(token); err != nil {
			decision = "denied"
			break
		}
		status, contacted = g.callUpstream(policy.UpstreamURL, req.Method, req.Path, req.Body)
		if !contacted {
			decision = "upstream_failed"
		}
	default:
		status, contacted = g.callUpstream(policy.UpstreamURL, req.Method, req.Path, req.Body)
		if !contacted {
			decision = "upstream_failed"
		}
	}
	pack := g.signPack(policy, decision, tool, contacted, status)
	loop.mu.Lock()
	loop.packs[pack.ID] = pack
	if stored := loop.policies[policy.ID]; stored != nil {
		stored.PackID = pack.ID
	}
	loop.mu.Unlock()
	writeJSON(w, http.StatusOK, pack)
}

func (g *Gateway) promoteAgentPolicy(w http.ResponseWriter, r *http.Request) {
	var req struct {
		PolicyID string `json:"policy_id"`
	}
	if err := json.NewDecoder(io.LimitReader(r.Body, 1<<20)).Decode(&req); err != nil {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_promote"})
		return
	}
	loop := g.loop()
	loop.mu.Lock()
	policy := loop.policies[req.PolicyID]
	var pack *agentPack
	if policy != nil {
		pack = loop.packs[policy.PackID]
	}
	loop.mu.Unlock()
	if policy == nil || pack == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "simulate_before_promote"})
		return
	}
	if policy.Promoted {
		writeJSON(w, http.StatusOK, map[string]any{"promoted": true, "policy_id": policy.ID, "pack_id": pack.ID})
		return
	}
	kind := policy.Kind
	if kind == "observe" {
		kind = "chargeback"
	}
	if kind == "charge" {
		kind = "l402"
	}
	name := "agent-" + policy.ID
	if policy.UpstreamURL != "" {
		proxy, err := g.createProxy(policy.UpstreamURL)
		if err != nil {
			writeJSON(w, http.StatusBadGateway, map[string]string{"error": "upstream_invalid"})
			return
		}
		g.proxyMu.Lock()
		g.proxies[name] = proxy
		g.proxyMu.Unlock()
		if g.config.Upstreams == nil {
			g.config.Upstreams = map[string]config.Upstream{}
		}
		g.config.Upstreams[name] = config.Upstream{URL: policy.UpstreamURL}
	}
	g.config.Routes = append(g.config.Routes, config.Route{
		Name:     policy.Name,
		Match:    config.RouteMatch{PathPrefix: policy.PathPrefix},
		Upstream: name,
		Policy:   config.RoutePolicy{Kind: kind},
	})
	loop.mu.Lock()
	policy.Promoted = true
	loop.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"promoted": true, "policy_id": policy.ID, "pack_id": pack.ID, "live": true})
}

func (g *Gateway) getAgentPack(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/agent/pack/")
	loop := g.loop()
	loop.mu.Lock()
	pack := loop.packs[id]
	loop.mu.Unlock()
	if pack == nil {
		writeJSON(w, http.StatusNotFound, map[string]string{"error": "pack_not_found"})
		return
	}
	writeJSON(w, http.StatusOK, pack)
}

func (g *Gateway) callUpstream(upstream, method, path, body string) (int, bool) {
	proxy, err := g.createProxy(upstream)
	if err != nil {
		return 0, false
	}
	req := httptest.NewRequest(method, path, strings.NewReader(body))
	rec := httptest.NewRecorder()
	proxy.ServeHTTP(rec, req)
	if rec.Code == 0 {
		return 0, false
	}
	return rec.Code, true
}

func (g *Gateway) signPack(policy *agentPolicy, decision, tool string, contacted bool, status int) *agentPack {
	payload := fmt.Sprintf("%s|%s|%s|%s|%t|%d|%s", policy.ID, policy.Kind, policy.Surface, decision, contacted, status, tool)
	sum := sha256.Sum256([]byte(payload))
	signed := hex.EncodeToString(sum[:])
	sig := ed25519.Sign(g.loop().privateKey, []byte(signed))
	return &agentPack{
		Schema:              "satgate.agent-loop.v1",
		ID:                  "ep_" + randHex(8),
		PolicyID:            policy.ID,
		Decision:            decision,
		Surface:             policy.Surface,
		Tool:                tool,
		UpstreamContacted:   contacted,
		UpstreamStatus:      status,
		ProviderPriceStatus: "UNKNOWN",
		Settlement:          false,
		IssuerPinned:        false,
		PublicKey:           hex.EncodeToString(g.loop().publicKey),
		Signature:           hex.EncodeToString(sig),
		SignedPayload:       signed,
	}
}

func mcpTool(body string) string {
	var doc struct {
		Params struct {
			Name string `json:"name"`
			Tool string `json:"tool"`
		} `json:"params"`
	}
	_ = json.Unmarshal([]byte(body), &doc)
	if doc.Params.Name != "" {
		return doc.Params.Name
	}
	return doc.Params.Tool
}

func randHex(n int) string {
	buf := make([]byte, n)
	_, _ = rand.Read(buf)
	return hex.EncodeToString(buf)
}

func writeJSON(w http.ResponseWriter, status int, body any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(body)
}
