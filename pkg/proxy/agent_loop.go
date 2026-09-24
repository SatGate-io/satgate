package proxy

import (
	"bytes"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

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
	Scope       string `json:"scope,omitempty"`
	Issuer      string `json:"issuer,omitempty"`
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
	packs      map[string]map[string]any
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	issuer     string
	kid        string
}

func (g *Gateway) loop() *agentLoop {
	g.agentOnce.Do(func() {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			panic(err)
		}
		g.agent = &agentLoop{
			policies:   map[string]*agentPolicy{},
			packs:      map[string]map[string]any{},
			publicKey:  pub,
			privateKey: priv,
			kid:        agentKid(pub),
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
	if req.Issuer != "" && !validIssuerOrigin(req.Issuer) {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_issuer"})
		return
	}
	req.ID = "pol_" + randHex(8)
	req.Promoted = false
	loop := g.loop()
	loop.mu.Lock()
	if req.Issuer != "" {
		loop.issuer = req.Issuer
	}
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
	identity := ""
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
		mac, err := g.macaroonSvc.Verify(token)
		if err != nil || (policy.Scope != "" && !mac.HasScope(policy.Scope)) {
			decision = "denied"
			break
		}
		identity = "sha256:" + hex.EncodeToString(mustSHA256(mac.Signature))
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
	protocolDecision, reason := protocolDecision(policy.Kind, decision)
	routeOrTool := req.Path
	if tool != "" {
		routeOrTool = tool
	}
	pack := g.evidencePack(policy, protocolDecision, reason, routeOrTool, contacted, status)
	if policy.Kind == "charge" {
		pack["promote_block"] = "price_unknown"
	}
	if identity != "" {
		pack["identity_fingerprint"] = identity
	}
	loop.mu.Lock()
	id, _ := pack["evidence_pack_id"].(string)
	loop.packs[id] = pack
	if stored := loop.policies[policy.ID]; stored != nil {
		stored.PackID = id
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
	var pack map[string]any
	if policy != nil {
		pack = loop.packs[policy.PackID]
	}
	loop.mu.Unlock()
	if policy == nil || pack == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "simulate_before_promote"})
		return
	}
	packID, _ := pack["evidence_pack_id"].(string)
	if policy.Promoted {
		writeJSON(w, http.StatusOK, map[string]any{"promoted": true, "policy_id": policy.ID, "pack_id": packID})
		return
	}
	kind := policy.Kind
	if kind == "observe" {
		kind = "chargeback"
	}
	if kind == "charge" {
		writeJSON(w, http.StatusConflict, map[string]any{
			"error":      "price_unknown",
			"promoted":   false,
			"policy_id":  policy.ID,
			"pack_id":    packID,
			"settlement": false,
		})
		return
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
		Policy:   config.RoutePolicy{Kind: kind, Scope: policy.Scope},
	})
	loop.mu.Lock()
	policy.Promoted = true
	loop.mu.Unlock()
	writeJSON(w, http.StatusOK, map[string]any{"promoted": true, "policy_id": policy.ID, "pack_id": packID, "live": true})
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

func protocolDecision(kind, raw string) (string, string) {
	switch raw {
	case "payment_required", "upstream_failed":
		return "denied", "policy_denied"
	case "denied":
		if kind == "capability" {
			return "denied", "capability_invalid"
		}
		return "denied", "policy_denied"
	default:
		return "allowed", "policy_allowed"
	}
}

func (g *Gateway) evidencePack(policy *agentPolicy, decision, reason, routeOrTool string, contacted bool, status int) map[string]any {
	now := time.Now().UTC().Format(time.RFC3339)
	packID := "ep_" + randHex(8)
	receiptID := "rcpt_" + randHex(8)
	sum := sha256.Sum256([]byte(policy.ID))
	pub := base64.RawURLEncoding.EncodeToString(g.loop().publicKey)
	receipt := map[string]any{
		"schema_version":       "satgate.receipt.v1",
		"schema_url":           "https://satgate.io/.well-known/satgate-receipt.schema.json",
		"receipt_id":           receiptID,
		"evidence_pack_id":     packID,
		"issuer":               g.loop().issuerOrigin(),
		"issuer_kid":           g.loop().kid,
		"decision":             decision,
		"decision_reason":      reason,
		"policy_version":       policy.ID,
		"timestamp":            now,
		"issued_at":            now,
		"canonicalization":     "jcs-rfc8785",
		"hash_algorithm":       "sha256",
		"signature_algorithm":  "ed25519",
		"capability_hash":      "sha256:" + hex.EncodeToString(sum[:]),
		"route_or_tool":        routeOrTool,
		"upstream_contacted":   contacted,
		"upstream_status":      status,
		"provider_price_status": "UNKNOWN",
		"settlement":           false,
		"metadata": map[string]any{
			"public_key_ed25519_b64": pub,
		},
	}
	payload := jcs(receipt)
	hash := sha256.Sum256(payload)
	receiptHash := "sha256:" + base64.RawURLEncoding.EncodeToString(hash[:])
	sig := ed25519.Sign(g.loop().privateKey, payload)
	receipt["receipt_hash"] = receiptHash
	receipt["signature"] = "ed25519:" + base64.RawURLEncoding.EncodeToString(sig)
	return map[string]any{
		"schema_version":        "satgate.evidence_pack.v1",
		"evidence_pack_id":      packID,
		"receipt_id":            receiptID,
		"issuer":                g.loop().issuerOrigin(),
		"decision":              decision,
		"decision_reason":       reason,
		"route_or_tool":         routeOrTool,
		"capability_hash":       receipt["capability_hash"],
		"receipt_hash":          receiptHash,
		"budget_state":          map[string]any{},
		"environment":           "runtime",
		"settlement":            false,
		"provider_price_status": "UNKNOWN",
		"upstream_contacted":    contacted,
		"upstream_status":       status,
		"receipts":              []any{receipt},
	}
}

func (g *Gateway) setAgentIssuer(issuer string) {
	g.loop().issuer = issuer
}

func (loop *agentLoop) issuerOrigin() string {
	if loop.issuer != "" {
		return loop.issuer
	}
	return "https://gateway.invalid"
}

func validIssuerOrigin(issuer string) bool {
	rest, ok := strings.CutPrefix(issuer, "https://")
	return ok && rest != "" && !strings.ContainsAny(rest, "/?#")
}

func agentKid(pub ed25519.PublicKey) string {
	sum := sha256.Sum256(pub)
	return "ed25519-" + hex.EncodeToString(sum[:8])
}

func (g *Gateway) agentJWKS() map[string]any {
	loop := g.loop()
	return map[string]any{
		"keys": []any{map[string]any{
			"kty": "OKP",
			"crv": "Ed25519",
			"alg": "EdDSA",
			"use": "sig",
			"kid": loop.kid,
			"x":   base64.RawURLEncoding.EncodeToString(loop.publicKey),
		}},
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

func mustSHA256(value string) []byte {
	sum := sha256.Sum256([]byte(value))
	return sum[:]
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

func jcs(v any) []byte {
	var buf bytes.Buffer
	writeJCS(&buf, v)
	return buf.Bytes()
}

func writeJCS(buf *bytes.Buffer, v any) {
	switch x := v.(type) {
	case map[string]any:
		keys := make([]string, 0, len(x))
		for key := range x {
			keys = append(keys, key)
		}
		sort.Strings(keys)
		buf.WriteByte('{')
		for i, key := range keys {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeJCS(buf, key)
			buf.WriteByte(':')
			writeJCS(buf, x[key])
		}
		buf.WriteByte('}')
	case []any:
		buf.WriteByte('[')
		for i, item := range x {
			if i > 0 {
				buf.WriteByte(',')
			}
			writeJCS(buf, item)
		}
		buf.WriteByte(']')
	case string:
		var raw bytes.Buffer
		enc := json.NewEncoder(&raw)
		enc.SetEscapeHTML(false)
		_ = enc.Encode(x)
		buf.Write(bytes.TrimRight(raw.Bytes(), "\n"))
	case bool:
		buf.WriteString(strconv.FormatBool(x))
	case int:
		buf.WriteString(strconv.Itoa(x))
	case nil:
		buf.WriteString("null")
	default:
		buf.WriteString("null")
	}
}
