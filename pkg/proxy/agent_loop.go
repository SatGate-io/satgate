package proxy

import (
	"bytes"
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/satgate-io/satgate/pkg/config"
	"github.com/satgate-io/satgate/pkg/lightning"
)

// Agent loop: stage a policy, run it against a real upstream, sign that
// decision, then promote only when the agent asks. Staging does not go live.
type agentPolicy struct {
	ID             string `json:"policy_id"`
	Name           string `json:"name"`
	Kind           string `json:"kind"`
	PathPrefix     string `json:"path_prefix"`
	UpstreamURL    string `json:"upstream_url"`
	Surface        string `json:"surface"`
	Scope          string `json:"scope,omitempty"`
	Issuer         string `json:"issuer,omitempty"`
	PriceSats      int64  `json:"price_sats,omitempty"`
	IdentityIssuer string `json:"identity_issuer,omitempty"`
	IdentityJWKS   string `json:"identity_jwks,omitempty"`
	PackID         string `json:"pack_id,omitempty"`
	Promoted       bool   `json:"promoted"`
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
	live       map[string]string
	publicKey  ed25519.PublicKey
	privateKey ed25519.PrivateKey
	issuer     string
	kid        string
	packDir    string
}

func (g *Gateway) restoreAgentRoutes() {
	if os.Getenv("SATGATE_AGENT_PACK_DIR") == "" {
		return
	}
	g.loop()
}

func (g *Gateway) loop() *agentLoop {
	g.agentOnce.Do(func() {
		dir := os.Getenv("SATGATE_AGENT_PACK_DIR")
		pub, priv := loadAgentKey(dir)
		g.agent = &agentLoop{
			policies:   map[string]*agentPolicy{},
			packs:      map[string]map[string]any{},
			live:       map[string]string{},
			publicKey:  pub,
			privateKey: priv,
			kid:        agentKid(pub),
			packDir:    dir,
		}
		g.restoreAgentPolicies()
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
	case "public", "observe", "deny", "capability", "charge", "identity":
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
	if req.PriceSats < 0 {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_price"})
		return
	}
	if req.Kind == "identity" && (!validIssuerOrigin(req.IdentityIssuer) || req.IdentityJWKS == "") {
		writeJSON(w, http.StatusBadRequest, map[string]string{"error": "invalid_identity"})
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
	_ = g.saveAgentPolicy(&req)
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
	case "identity":
		fp, err := g.verifyAgentIdentity(policy, req.Authorization)
		if err != nil {
			decision = "denied"
			break
		}
		identity = fp
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
		if policy.PriceSats <= 0 {
			pack["promote_block"] = "price_unknown"
		} else if !g.realPaymentRail() {
			pack["promote_block"] = "payment_rail_unavailable"
		}
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
	if stored := g.loop().policies[policy.ID]; stored != nil {
		_ = g.saveAgentPolicy(stored)
	}
	_ = g.saveAgentPack(pack)
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
	if policy != nil && pack == nil && policy.PackID != "" {
		pack = g.loadAgentPack(policy.PackID)
	}
	if policy == nil || pack == nil {
		writeJSON(w, http.StatusConflict, map[string]string{"error": "simulate_before_promote"})
		return
	}
	packID, _ := pack["evidence_pack_id"].(string)
	if policy.Promoted {
		writeJSON(w, http.StatusOK, map[string]any{"promoted": true, "policy_id": policy.ID, "pack_id": packID})
		return
	}
	if err := g.installPromotedPolicy(policy); err != nil {
		status := http.StatusBadGateway
		code := "upstream_invalid"
		if err.Error() == "price_unknown" || err.Error() == "payment_rail_unavailable" {
			status = http.StatusConflict
			code = err.Error()
		}
		writeJSON(w, status, map[string]any{
			"error":      code,
			"promoted":   false,
			"policy_id":  policy.ID,
			"pack_id":    packID,
			"settlement": false,
		})
		return
	}
	loop.mu.Lock()
	policy.Promoted = true
	loop.live[policy.PathPrefix] = policy.ID
	loop.mu.Unlock()
	_ = g.saveAgentPolicy(policy)
	writeJSON(w, http.StatusOK, map[string]any{"promoted": true, "policy_id": policy.ID, "pack_id": packID, "live": true})
}

func (g *Gateway) installPromotedPolicy(policy *agentPolicy) error {
	kind := policy.Kind
	switch kind {
	case "charge":
		if policy.PriceSats <= 0 {
			return errors.New("price_unknown")
		}
		if !g.realPaymentRail() {
			return errors.New("payment_rail_unavailable")
		}
		kind = "l402"
	case "observe":
		kind = "chargeback"
	}
	name := "agent-" + policy.ID
	if policy.UpstreamURL != "" {
		proxy, err := g.createProxy(policy.UpstreamURL)
		if err != nil {
			return err
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
		Policy:   config.RoutePolicy{Kind: kind, Scope: policy.Scope, PriceSats: policy.PriceSats},
	})
	return nil
}

func (g *Gateway) getAgentPack(w http.ResponseWriter, r *http.Request) {
	id := strings.TrimPrefix(r.URL.Path, "/api/agent/pack/")
	loop := g.loop()
	loop.mu.Lock()
	pack := loop.packs[id]
	loop.mu.Unlock()
	if pack == nil {
		pack = g.loadAgentPack(id)
	}
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
		"schema_version":        "satgate.receipt.v1",
		"schema_url":            "https://satgate.io/.well-known/satgate-receipt.schema.json",
		"receipt_id":            receiptID,
		"evidence_pack_id":      packID,
		"issuer":                g.loop().issuerOrigin(),
		"issuer_kid":            g.loop().kid,
		"decision":              decision,
		"decision_reason":       reason,
		"policy_version":        policy.ID,
		"timestamp":             now,
		"issued_at":             now,
		"canonicalization":      "jcs-rfc8785",
		"hash_algorithm":        "sha256",
		"signature_algorithm":   "ed25519",
		"capability_hash":       "sha256:" + hex.EncodeToString(sum[:]),
		"route_or_tool":         routeOrTool,
		"upstream_contacted":    contacted,
		"upstream_status":       status,
		"provider_price_status": "UNKNOWN",
		"settlement":            false,
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

func (g *Gateway) withLivePack(w http.ResponseWriter, r *http.Request, route *config.Route) http.ResponseWriter {
	policy := g.agentPolicyForRoute(route)
	if policy == nil {
		return w
	}
	tool := ""
	if policy.Surface == "mcp" && r.Body != nil {
		raw, err := io.ReadAll(io.LimitReader(r.Body, 1<<20))
		if err == nil {
			r.Body = io.NopCloser(bytes.NewReader(raw))
			tool = mcpTool(string(raw))
		}
	}
	return &livePackWriter{ResponseWriter: w, gw: g, route: route, req: r, tool: tool}
}

func (g *Gateway) agentPolicyForRoute(route *config.Route) *agentPolicy {
	if g.agent == nil || route == nil {
		return nil
	}
	loop := g.agent
	loop.mu.Lock()
	defer loop.mu.Unlock()
	return loop.policies[loop.live[route.Match.PathPrefix]]
}

func (g *Gateway) recordLivePack(route *config.Route, r *http.Request, code int, tool string) string {
	if g.agent == nil || route == nil {
		return ""
	}
	loop := g.agent
	loop.mu.Lock()
	policy := loop.policies[loop.live[route.Match.PathPrefix]]
	loop.mu.Unlock()
	if policy == nil {
		return ""
	}
	decision, reason := "allowed", "policy_allowed"
	contacted := true
	status := code
	if policy.Kind == "deny" || policy.Kind == "charge" || code == http.StatusUnauthorized || code == http.StatusForbidden || code == http.StatusPaymentRequired {
		decision = "denied"
		reason = "policy_denied"
		if policy.Kind == "capability" || policy.Kind == "identity" {
			reason = "capability_invalid"
		}
		if code == http.StatusPaymentRequired {
			reason = "payment_required"
		}
		contacted = false
		status = 0
	}
	routeOrTool := r.URL.Path
	if tool != "" {
		routeOrTool = tool
	}
	pack := g.evidencePack(policy, decision, reason, routeOrTool, contacted, status)
	if fp, _ := r.Context().Value(identityFingerprintKey).(string); fp != "" {
		pack["identity_fingerprint"] = fp
	}
	loop.mu.Lock()
	id, _ := pack["evidence_pack_id"].(string)
	loop.packs[id] = pack
	loop.mu.Unlock()
	_ = g.saveAgentPack(pack)
	return id
}

type livePackWriter struct {
	http.ResponseWriter
	gw    *Gateway
	route *config.Route
	req   *http.Request
	tool  string
	done  bool
}

func (p *livePackWriter) WriteHeader(code int) {
	if !p.done {
		p.done = true
		if id := p.gw.recordLivePack(p.route, p.req, code, p.tool); id != "" {
			p.Header().Set("SatGate-Evidence-Pack", id)
		}
	}
	p.ResponseWriter.WriteHeader(code)
}

func (p *livePackWriter) Write(b []byte) (int, error) {
	if !p.done {
		p.WriteHeader(http.StatusOK)
	}
	return p.ResponseWriter.Write(b)
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

func loadAgentKey(dir string) (ed25519.PublicKey, ed25519.PrivateKey) {
	if dir == "" {
		pub, priv, err := ed25519.GenerateKey(rand.Reader)
		if err != nil {
			panic(err)
		}
		return pub, priv
	}
	if err := os.MkdirAll(filepath.Join(dir, "packs"), 0o700); err != nil {
		panic(err)
	}
	if err := os.MkdirAll(filepath.Join(dir, "policies"), 0o700); err != nil {
		panic(err)
	}
	path := filepath.Join(dir, "signing-key.json")
	raw, err := os.ReadFile(path)
	if err == nil {
		var doc struct {
			Public  string `json:"public"`
			Private string `json:"private"`
		}
		if json.Unmarshal(raw, &doc) == nil {
			priv, privErr := base64.RawURLEncoding.DecodeString(doc.Private)
			pub, pubErr := base64.RawURLEncoding.DecodeString(doc.Public)
			if privErr == nil && pubErr == nil && len(priv) == ed25519.PrivateKeySize && len(pub) == ed25519.PublicKeySize {
				return ed25519.PublicKey(pub), ed25519.PrivateKey(priv)
			}
		}
	}
	pub, priv, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		panic(err)
	}
	doc := map[string]string{
		"public":  base64.RawURLEncoding.EncodeToString(pub),
		"private": base64.RawURLEncoding.EncodeToString(priv),
	}
	body, _ := json.Marshal(doc)
	if err := writeAtomic(path, body); err != nil {
		panic(err)
	}
	return pub, priv
}

func (g *Gateway) saveAgentPolicy(policy *agentPolicy) error {
	if g.agent == nil || g.agent.packDir == "" || policy == nil || !safePolicyID(policy.ID) {
		return nil
	}
	raw, err := json.Marshal(policy)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(g.agent.packDir, "policies", policy.ID+".json"), raw)
}

func (g *Gateway) restoreAgentPolicies() {
	if g.agent == nil || g.agent.packDir == "" {
		return
	}
	dir := filepath.Join(g.agent.packDir, "policies")
	entries, err := os.ReadDir(dir)
	if err != nil {
		return
	}
	for _, entry := range entries {
		if entry.IsDir() || !strings.HasSuffix(entry.Name(), ".json") {
			continue
		}
		raw, err := os.ReadFile(filepath.Join(dir, entry.Name()))
		if err != nil {
			continue
		}
		var policy agentPolicy
		if json.Unmarshal(raw, &policy) != nil || !safePolicyID(policy.ID) {
			continue
		}
		g.agent.policies[policy.ID] = &policy
		if policy.Issuer != "" && g.agent.issuer == "" {
			g.agent.issuer = policy.Issuer
		}
		if !policy.Promoted || policy.Kind == "charge" {
			continue
		}
		if err := g.installPromotedPolicy(&policy); err != nil {
			policy.Promoted = false
			continue
		}
		g.agent.live[policy.PathPrefix] = policy.ID
	}
}

func safePolicyID(id string) bool {
	return strings.HasPrefix(id, "pol_") && !strings.ContainsAny(id, "/.\\")
}

func (g *Gateway) saveAgentPack(pack map[string]any) error {
	if g.agent == nil || g.agent.packDir == "" {
		return nil
	}
	id, _ := pack["evidence_pack_id"].(string)
	if !safePackID(id) {
		return errors.New("bad pack id")
	}
	raw, err := json.Marshal(pack)
	if err != nil {
		return err
	}
	return writeAtomic(filepath.Join(g.agent.packDir, "packs", id+".json"), raw)
}

func (g *Gateway) loadAgentPack(id string) map[string]any {
	if g.agent == nil || g.agent.packDir == "" || !safePackID(id) {
		return nil
	}
	raw, err := os.ReadFile(filepath.Join(g.agent.packDir, "packs", id+".json"))
	if err != nil {
		return nil
	}
	var pack map[string]any
	if json.Unmarshal(raw, &pack) != nil {
		return nil
	}
	g.agent.mu.Lock()
	g.agent.packs[id] = pack
	g.agent.mu.Unlock()
	return pack
}

func safePackID(id string) bool {
	return strings.HasPrefix(id, "ep_") && !strings.ContainsAny(id, "/.\\")
}

func writeAtomic(path string, data []byte) error {
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
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

type identityCtxKey struct{}

var identityFingerprintKey = identityCtxKey{}

func (g *Gateway) realPaymentRail() bool {
	if g.lightning == nil {
		return false
	}
	_, mock := g.lightning.(*lightning.MockProvider)
	return !mock
}

func (g *Gateway) handleIdentity(w http.ResponseWriter, r *http.Request, route *config.Route) {
	policy := g.agentPolicyForRoute(route)
	if policy == nil {
		http.Error(w, "identity_not_configured", http.StatusForbidden)
		return
	}
	fp, err := g.verifyAgentIdentity(policy, r.Header.Get("Authorization"))
	if err != nil {
		http.Error(w, "Invalid token", http.StatusUnauthorized)
		return
	}
	r = r.WithContext(contextWithIdentity(r.Context(), fp))
	g.proxyRequest(w, r, route)
}

func (g *Gateway) verifyAgentIdentity(policy *agentPolicy, authorization string) (string, error) {
	token := strings.TrimPrefix(authorization, "Bearer ")
	if token == "" || token == authorization || policy == nil {
		return "", errors.New("identity_invalid")
	}
	keys, err := g.fetchIdentityKeys(policy.IdentityJWKS)
	if err != nil {
		return "", err
	}
	return verifyEdJWT(token, policy.IdentityIssuer, keys)
}

func (g *Gateway) fetchIdentityKeys(rawURL string) (map[string]ed25519.PublicKey, error) {
	if err := g.identityURLAllowed(rawURL); err != nil {
		return nil, err
	}
	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Get(rawURL)
	if err != nil {
		return nil, errors.New("identity_provider_unavailable")
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("identity_provider_unavailable")
	}
	var doc struct {
		Keys []struct {
			Kty string `json:"kty"`
			Crv string `json:"crv"`
			Kid string `json:"kid"`
			X   string `json:"x"`
		} `json:"keys"`
	}
	if err := json.NewDecoder(io.LimitReader(resp.Body, 1<<20)).Decode(&doc); err != nil {
		return nil, errors.New("identity_provider_unavailable")
	}
	keys := map[string]ed25519.PublicKey{}
	for _, key := range doc.Keys {
		if key.Kty != "OKP" || key.Crv != "Ed25519" || key.Kid == "" {
			continue
		}
		raw, err := base64.RawURLEncoding.DecodeString(key.X)
		if err != nil || len(raw) != ed25519.PublicKeySize {
			continue
		}
		keys[key.Kid] = ed25519.PublicKey(raw)
	}
	if len(keys) == 0 {
		return nil, errors.New("identity_provider_unavailable")
	}
	return keys, nil
}

func (g *Gateway) identityURLAllowed(raw string) error {
	parsed, err := url.Parse(raw)
	if err != nil || parsed.Host == "" || parsed.User != nil {
		return errors.New("identity_jwks_invalid")
	}
	allowPrivate := g.config != nil && g.config.Cloud != nil && g.config.Cloud.SSRF != nil && g.config.Cloud.SSRF.AllowPrivateIPs
	if parsed.Scheme != "https" && !(allowPrivate && parsed.Scheme == "http") {
		return errors.New("identity_jwks_invalid")
	}
	if ip := net.ParseIP(parsed.Hostname()); ip != nil && !allowPrivate && (ip.IsLoopback() || ip.IsPrivate() || ip.IsLinkLocalUnicast() || ip.IsUnspecified()) {
		return errors.New("identity_jwks_invalid")
	}
	return nil
}

func verifyEdJWT(token, issuer string, keys map[string]ed25519.PublicKey) (string, error) {
	parts := strings.Split(token, ".")
	if len(parts) != 3 {
		return "", errors.New("identity_invalid")
	}
	headerRaw, err := base64.RawURLEncoding.DecodeString(parts[0])
	if err != nil {
		return "", errors.New("identity_invalid")
	}
	var header struct {
		Alg string `json:"alg"`
		Kid string `json:"kid"`
	}
	if json.Unmarshal(headerRaw, &header) != nil || header.Alg != "EdDSA" {
		return "", errors.New("identity_invalid")
	}
	pub := keys[header.Kid]
	sig, err := base64.RawURLEncoding.DecodeString(parts[2])
	if err != nil || len(pub) == 0 || !ed25519.Verify(pub, []byte(parts[0]+"."+parts[1]), sig) {
		return "", errors.New("identity_invalid")
	}
	payloadRaw, err := base64.RawURLEncoding.DecodeString(parts[1])
	if err != nil {
		return "", errors.New("identity_invalid")
	}
	var payload struct {
		Iss string  `json:"iss"`
		Exp float64 `json:"exp"`
	}
	if json.Unmarshal(payloadRaw, &payload) != nil || payload.Iss != issuer || payload.Exp <= float64(time.Now().Unix()) {
		return "", errors.New("identity_invalid")
	}
	return "sha256:" + hex.EncodeToString(mustSHA256(parts[2])), nil
}

func contextWithIdentity(ctx context.Context, fingerprint string) context.Context {
	return context.WithValue(ctx, identityFingerprintKey, fingerprint)
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
