package mcpserver

import (
	"context"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/json"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Streamable HTTP (MCP 2025-03-26 / 2025-06-18) shares this server with the
// legacy SSE transport. SSE stays at /sse and /message. The Streamable HTTP
// endpoint is the mount root, which is the conventional public agent URL
// .../mcp:
//
//   - Dedicated gateways mount this handler behind StripPrefix("/mcp").
//     POST https://host/mcp arrives as path "" (no trailing slash) or "/".
//   - Unstripped mounts and the standalone listener also accept /mcp, so the
//     same conventional URL works when nothing strips the prefix.
//   - WithBasePath extras are accepted the same way. The prefix is only an
//     outbound SSE endpoint hint; inbound SSE is still /sse and /message.
//
// Server-initiated GET streams are not offered (405). Tool calls go through
// Proxy.handleRequest, the same budget, evidence, and event path as POST /message.

const (
	headerMCPSessionID       = "Mcp-Session-Id"
	headerMCPProtocolVersion = "MCP-Protocol-Version"

	protocolVersion20250326 = "2025-03-26"
	protocolVersion20250618 = "2025-06-18"

	defaultStreamableSessionTTL = 30 * time.Minute

	// defaultMaxStreamableSessions bounds the in-memory session map. A full
	// map is swept for expired entries first; if it is still full, initialize
	// is refused with 503 rather than growing without limit.
	defaultMaxStreamableSessions = 10000
	streamableSweepInterval      = time.Minute
)

// IsStreamableHTTPPath reports whether path is the agent-facing Streamable
// HTTP endpoint. extra lists additional mount prefixes (WithBasePath).
// Empty path is what http.StripPrefix("/mcp") leaves on POST /mcp.
func IsStreamableHTTPPath(path string, extra ...string) bool {
	if path == "" || path == "/" {
		return true
	}
	trimmed := strings.TrimSuffix(path, "/")
	if trimmed == "" {
		return true
	}
	candidates := append([]string{"/mcp"}, extra...)
	for _, c := range candidates {
		c = strings.TrimSuffix(strings.TrimSpace(c), "/")
		if c == "" || c == "/" {
			continue
		}
		if path == c || path == c+"/" || trimmed == c {
			return true
		}
	}
	return false
}

// WithAllowedOrigins adds origins that may be sent in the Origin header.
// Loopback origins are always allowed. A missing Origin is allowed so
// non-browser agents keep working. A present, non-allowed Origin is rejected
// (DNS-rebinding guidance).
func WithAllowedOrigins(origins ...string) SSEOption {
	return func(s *SSEServer) {
		if s.allowedOrigins == nil {
			s.allowedOrigins = make(map[string]struct{})
		}
		for _, origin := range origins {
			if key := normalizeOrigin(origin); key != "" {
				s.allowedOrigins[key] = struct{}{}
			}
		}
	}
}

// WithStreamableSessionTTL overrides the idle lifetime of a Streamable HTTP
// session. Expired sessions are answered with 404.
func WithStreamableSessionTTL(ttl time.Duration) SSEOption {
	return func(s *SSEServer) {
		if ttl > 0 {
			s.streamTTL = ttl
		}
	}
}

// WithMaxStreamableSessions overrides the cap on concurrent Streamable HTTP
// sessions held in memory.
func WithMaxStreamableSessions(n int) SSEOption {
	return func(s *SSEServer) {
		if n > 0 {
			s.streamMax = n
		}
	}
}

// streamSession is one Streamable HTTP client session.
type streamSession struct {
	id        string
	tokenSum  [32]byte
	protocol  string
	expiresAt time.Time
	identity  sseSessionIdentity
	verified  *TokenInfo
}

// ServeHTTP dispatches legacy SSE routes and the Streamable HTTP endpoint.
func (s *SSEServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if s == nil || s.mux == nil {
		http.Error(w, "mcp server unavailable", http.StatusServiceUnavailable)
		return
	}
	if IsStreamableHTTPPath(r.URL.Path, s.basePath) {
		s.handleStreamable(w, r)
		return
	}
	s.mux.ServeHTTP(w, r)
}

func (s *SSEServer) handleStreamable(w http.ResponseWriter, r *http.Request) {
	if !originAllowed(r.Header.Get("Origin"), s.allowedOrigins) {
		http.Error(w, "origin not allowed", http.StatusForbidden)
		return
	}

	switch r.Method {
	case http.MethodGet:
		// No server-initiated stream. 405 is the spec's explicit alternative.
		w.Header().Set("Allow", "POST, DELETE")
		http.Error(w, "server-initiated stream not supported", http.StatusMethodNotAllowed)
		return
	case http.MethodDelete:
		s.handleStreamableDelete(w, r)
		return
	case http.MethodPost:
		s.handleStreamablePost(w, r)
		return
	default:
		w.Header().Set("Allow", "POST, DELETE, GET")
		http.Error(w, "method not allowed", http.StatusMethodNotAllowed)
	}
}

func (s *SSEServer) handleStreamableDelete(w http.ResponseWriter, r *http.Request) {
	sid := strings.TrimSpace(r.Header.Get(headerMCPSessionID))
	if sid == "" {
		http.Error(w, "Mcp-Session-Id required", http.StatusBadRequest)
		return
	}
	if !validSessionID(sid) {
		http.Error(w, "invalid Mcp-Session-Id", http.StatusBadRequest)
		return
	}
	sess, ok := s.lookupStreamSession(sid)
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}
	if !s.streamSessionAllowsToken(sess, extractAuthToken(r)) {
		http.Error(w, "session_token_mismatch", http.StatusForbidden)
		return
	}
	if !s.protocolVersionAllowed(r) {
		http.Error(w, "unsupported MCP-Protocol-Version", http.StatusBadRequest)
		return
	}
	s.deleteStreamSession(sid)
	w.Header().Set(headerMCPSessionID, sid)
	w.WriteHeader(http.StatusOK)
}

func (s *SSEServer) handleStreamablePost(w http.ResponseWriter, r *http.Request) {
	if !acceptsStreamableResponse(r.Header.Get("Accept")) {
		http.Error(w, "Accept must include application/json and text/event-stream", http.StatusNotAcceptable)
		return
	}
	media := strings.TrimSpace(strings.Split(r.Header.Get("Content-Type"), ";")[0])
	if !strings.EqualFold(media, "application/json") {
		http.Error(w, "Content-Type must be application/json", http.StatusUnsupportedMediaType)
		return
	}

	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024))
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}
	trimmed := strings.TrimSpace(string(body))
	if trimmed == "" {
		http.Error(w, "empty body", http.StatusBadRequest)
		return
	}
	if trimmed[0] == '[' {
		// JSON-RPC batching was removed in MCP 2025-06-18.
		http.Error(w, "JSON-RPC batching is not supported", http.StatusBadRequest)
		return
	}

	req, respMsg, parseErr := ParseMessage(json.RawMessage(trimmed))
	if parseErr != nil {
		http.Error(w, "invalid JSON-RPC", http.StatusBadRequest)
		return
	}

	sid := strings.TrimSpace(r.Header.Get(headerMCPSessionID))
	if sid != "" && !validSessionID(sid) {
		http.Error(w, "invalid Mcp-Session-Id", http.StatusBadRequest)
		return
	}

	var sess *streamSession
	switch {
	case sid == "":
		if req == nil || req.Method != MethodInitialize {
			http.Error(w, "Mcp-Session-Id required", http.StatusBadRequest)
			return
		}
		negotiated, ok := negotiateProtocolVersion(protocolVersionFromInit(req), r.Header.Get(headerMCPProtocolVersion))
		if !ok {
			http.Error(w, "unsupported protocol version", http.StatusBadRequest)
			return
		}
		authToken := extractAuthToken(r)
		if denied := s.rejectHardAuthFailure(w, r, authToken); denied {
			return
		}
		// In header auth mode a session is only created for a token that
		// verifies. A tokenless or unverifiable initialize gets 401 and leaves
		// nothing in the session map.
		if s.requiresVerifiedStreamToken() {
			if authToken == "" {
				http.Error(w, "authentication required", http.StatusUnauthorized)
				return
			}
			if _, err := s.proxy.auth.Verify(r.Context(), authToken); err != nil {
				http.Error(w, "authentication failed", http.StatusUnauthorized)
				return
			}
		}
		var created bool
		sess, created = s.createStreamSession(r.Context(), authToken, negotiated)
		if !created {
			w.Header().Set("Retry-After", "60")
			http.Error(w, "too many sessions", http.StatusServiceUnavailable)
			return
		}
	default:
		var found bool
		sess, found = s.lookupStreamSession(sid)
		if !found {
			http.Error(w, "session not found", http.StatusNotFound)
			return
		}
		if !s.streamSessionAllowsToken(sess, extractAuthToken(r)) {
			http.Error(w, "session_token_mismatch", http.StatusForbidden)
			return
		}
		if !s.protocolVersionAllowed(r) {
			http.Error(w, "unsupported MCP-Protocol-Version", http.StatusBadRequest)
			return
		}
	}

	// Client responses and notifications are accepted and not answered.
	if req == nil || isStreamableNotification(req) {
		if respMsg != nil || isStreamableNotification(req) {
			s.touchStreamSession(sess)
			w.Header().Set(headerMCPSessionID, sess.id)
			w.WriteHeader(http.StatusAccepted)
			return
		}
	}
	if req == nil {
		http.Error(w, "invalid JSON-RPC", http.StatusBadRequest)
		return
	}

	authToken := extractAuthToken(r)
	if authToken != "" && len(req.Params) > 0 {
		req.Params = injectMetaToken(req.Params, authToken)
	} else if authToken != "" {
		metaJSON, _ := json.Marshal(map[string]string{"token": authToken})
		paramsJSON, _ := json.Marshal(map[string]json.RawMessage{"_meta": metaJSON})
		req.Params = paramsJSON
	}

	// r.Context() carries the enterprise spend session. Do not replace it.
	reqCtx := sess.contextOnto(r.Context())
	rpcResp, handleErr := s.proxy.handleRequest(reqCtx, req)
	if handleErr != nil {
		rpcResp = NewErrorResponse(req.ID, CodeInternalError, handleErr.Error())
	}
	if rpcResp == nil {
		s.touchStreamSession(sess)
		w.Header().Set(headerMCPSessionID, sess.id)
		w.WriteHeader(http.StatusAccepted)
		return
	}
	if req.Method == MethodInitialize {
		rpcResp = withNegotiatedProtocol(rpcResp, sess.protocol)
	}

	s.touchStreamSession(sess)
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set(headerMCPSessionID, sess.id)
	w.Header().Set(headerMCPProtocolVersion, sess.protocol)
	w.WriteHeader(http.StatusOK)
	_ = json.NewEncoder(w).Encode(rpcResp)
}

func (s *SSEServer) rejectHardAuthFailure(w http.ResponseWriter, r *http.Request, authToken string) bool {
	if authToken == "" || s.proxy == nil || s.proxy.auth == nil {
		return false
	}
	if _, err := s.proxy.auth.Verify(r.Context(), authToken); err != nil {
		errMsg := err.Error()
		if strings.Contains(errMsg, "enterprise deployment") || strings.Contains(errMsg, "token revoked") {
			s.writePreConnectDenial(w, r, errMsg)
			return true
		}
	}
	return false
}

func isStreamableNotification(req *Request) bool {
	if req == nil {
		return false
	}
	if IsNotification(req.Method) {
		return true
	}
	return len(req.ID) == 0
}

// requiresVerifiedStreamToken reports whether initialize must carry a token
// that verifies before a session is created (header auth mode).
func (s *SSEServer) requiresVerifiedStreamToken() bool {
	return s.proxy != nil && s.proxy.config != nil && s.proxy.auth != nil &&
		s.proxy.config.Auth.Mode == "header"
}

func (s *SSEServer) streamMaxSessions() int {
	if s.streamMax > 0 {
		return s.streamMax
	}
	return defaultMaxStreamableSessions
}

// sweepExpiredStreamSessionsLocked drops expired sessions. Caller holds
// streamMu. It runs at most once per streamableSweepInterval unless force is
// set (the map is full).
func (s *SSEServer) sweepExpiredStreamSessionsLocked(now time.Time, force bool) []*streamSession {
	if !force && !s.streamLastSweep.IsZero() && now.Sub(s.streamLastSweep) < streamableSweepInterval {
		return nil
	}
	s.streamLastSweep = now
	var closed []*streamSession
	for id, sess := range s.streamSessions {
		if !sess.expiresAt.IsZero() && now.After(sess.expiresAt) {
			delete(s.streamSessions, id)
			closed = append(closed, sess)
		}
	}
	return closed
}

func (s *SSEServer) createStreamSession(ctx context.Context, token, protocol string) (*streamSession, bool) {
	now := time.Now()
	sess := &streamSession{
		id:        generateSessionID(),
		tokenSum:  sha256.Sum256([]byte(token)),
		protocol:  protocol,
		expiresAt: now.Add(s.streamIdleTTL()),
	}
	if token != "" && s.proxy != nil && s.proxy.auth != nil {
		if info, err := s.proxy.auth.Verify(ctx, token); err == nil && info != nil {
			copyInfo := *info
			sess.verified = &copyInfo
			sess.identity = sseSessionIdentity{
				tokenID:           info.TokenID,
				tenantID:          info.TenantID,
				budgetID:          info.BudgetID,
				verifiedTokenInfo: &copyInfo,
			}
		}
	}
	s.streamMu.Lock()
	if s.streamSessions == nil {
		s.streamSessions = make(map[string]*streamSession)
	}
	full := len(s.streamSessions) >= s.streamMaxSessions()
	closed := s.sweepExpiredStreamSessionsLocked(now, full)
	if len(s.streamSessions) >= s.streamMaxSessions() {
		s.streamMu.Unlock()
		for _, c := range closed {
			s.publishStreamClose(c)
		}
		log.Warn().Int("sessions", s.streamMaxSessions()).Msg("streamable HTTP session cap reached")
		return nil, false
	}
	s.streamSessions[sess.id] = sess
	s.streamMu.Unlock()
	for _, c := range closed {
		s.publishStreamClose(c)
	}

	if s.proxy != nil && s.proxy.events != nil {
		s.proxy.events.Publish(Event{
			Type:      EventSessionConnect,
			Timestamp: now,
			SessionID: sess.id,
			TokenID:   sess.identity.tokenID,
			TenantID:  sess.identity.tenantID,
			BudgetID:  sess.identity.budgetID,
		})
	}
	log.Info().Str("session", sess.id).Str("protocol", protocol).Msg("streamable HTTP session established")
	return sess, true
}

func (s *SSEServer) lookupStreamSession(id string) (*streamSession, bool) {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	sess, ok := s.streamSessions[id]
	if !ok {
		return nil, false
	}
	if !sess.expiresAt.IsZero() && time.Now().After(sess.expiresAt) {
		delete(s.streamSessions, id)
		go s.publishStreamClose(sess)
		return nil, false
	}
	return sess, true
}

func (s *SSEServer) touchStreamSession(sess *streamSession) {
	if sess == nil {
		return
	}
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	current, ok := s.streamSessions[sess.id]
	if !ok {
		return
	}
	current.expiresAt = time.Now().Add(s.streamIdleTTL())
}

func (s *SSEServer) deleteStreamSession(id string) {
	s.streamMu.Lock()
	sess, ok := s.streamSessions[id]
	if ok {
		delete(s.streamSessions, id)
	}
	s.streamMu.Unlock()
	if ok {
		s.publishStreamClose(sess)
	}
}

func (s *SSEServer) publishStreamClose(sess *streamSession) {
	if sess == nil || s.proxy == nil || s.proxy.events == nil {
		return
	}
	s.proxy.events.Publish(Event{
		Type:      EventSessionClose,
		Timestamp: time.Now(),
		SessionID: sess.id,
		TokenID:   sess.identity.tokenID,
		TenantID:  sess.identity.tenantID,
		BudgetID:  sess.identity.budgetID,
	})
	log.Info().Str("session", sess.id).Msg("streamable HTTP session closed")
}

func (s *SSEServer) streamIdleTTL() time.Duration {
	if s.streamTTL > 0 {
		return s.streamTTL
	}
	return defaultStreamableSessionTTL
}

// expireStreamableSession marks a session expired. Tests use it to prove a
// 404 on an expired id without waiting out the idle TTL.
func (s *SSEServer) expireStreamableSession(id string) {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	if sess, ok := s.streamSessions[id]; ok {
		sess.expiresAt = time.Now().Add(-time.Second)
	}
}

// streamSessionAllowsToken is the OSS identity binding for a Streamable HTTP
// session. A later request must present the same credential that initialized
// the session. Comparison is over a hash so the raw token is not retained.
func (s *SSEServer) streamSessionAllowsToken(sess *streamSession, token string) bool {
	if sess == nil {
		return false
	}
	got := sha256.Sum256([]byte(token))
	return subtle.ConstantTimeCompare(sess.tokenSum[:], got[:]) == 1
}

func (s *SSEServer) protocolVersionAllowed(r *http.Request) bool {
	if r == nil {
		return false
	}
	header := strings.TrimSpace(r.Header.Get(headerMCPProtocolVersion))
	if header == "" {
		// Spec: a missing header assumes 2025-03-26. The session also stored
		// the version negotiated at initialize; either is acceptable.
		return true
	}
	_, ok := supportedProtocolVersion(header)
	return ok
}

func (sess *streamSession) contextOnto(parent context.Context) context.Context {
	if sess == nil {
		return parent
	}
	if parent == nil {
		parent = context.Background()
	}
	if sess.identity.tenantID != "" {
		parent = context.WithValue(parent, CtxTenantID, sess.identity.tenantID)
	}
	if sess.verified != nil {
		parent = context.WithValue(parent, CtxTokenInfo, sess.verified)
	}
	return parent
}

func protocolVersionFromInit(req *Request) string {
	if req == nil || len(req.Params) == 0 {
		return ""
	}
	var params struct {
		ProtocolVersion string `json:"protocolVersion"`
	}
	if err := json.Unmarshal(req.Params, &params); err != nil {
		return ""
	}
	return params.ProtocolVersion
}

func negotiateProtocolVersion(bodyVersion, headerVersion string) (string, bool) {
	bodyVersion = strings.TrimSpace(bodyVersion)
	headerVersion = strings.TrimSpace(headerVersion)
	if headerVersion != "" {
		if _, ok := supportedProtocolVersion(headerVersion); !ok {
			return "", false
		}
	}
	switch bodyVersion {
	case "":
		if headerVersion != "" {
			return headerVersion, true
		}
		return protocolVersion20250326, true
	case protocolVersion20250326, protocolVersion20250618:
		if headerVersion != "" && headerVersion != bodyVersion {
			return "", false
		}
		return bodyVersion, true
	case "2024-11-05":
		// Legacy SSE clients may POST initialize here before falling back.
		// Answer with the oldest Streamable HTTP version; do not claim 2024-11-05.
		if headerVersion != "" && headerVersion != protocolVersion20250326 {
			return "", false
		}
		return protocolVersion20250326, true
	default:
		return "", false
	}
}

func supportedProtocolVersion(v string) (string, bool) {
	switch strings.TrimSpace(v) {
	case protocolVersion20250326, protocolVersion20250618:
		return strings.TrimSpace(v), true
	default:
		return "", false
	}
}

func withNegotiatedProtocol(resp *Response, protocol string) *Response {
	if resp == nil || protocol == "" || len(resp.Result) == 0 {
		return resp
	}
	var result map[string]interface{}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		return resp
	}
	result["protocolVersion"] = protocol
	encoded, err := json.Marshal(result)
	if err != nil {
		return resp
	}
	copied := *resp
	copied.Result = encoded
	return &copied
}

func acceptsStreamableResponse(accept string) bool {
	return acceptIncludes(accept, "application/json") && acceptIncludes(accept, "text/event-stream")
}

func acceptIncludes(header, media string) bool {
	for _, part := range strings.Split(header, ",") {
		typ := strings.ToLower(strings.TrimSpace(strings.Split(part, ";")[0]))
		if typ == strings.ToLower(media) || typ == "*/*" {
			return true
		}
	}
	return false
}

func validSessionID(id string) bool {
	if id == "" || len(id) > 256 {
		return false
	}
	for i := 0; i < len(id); i++ {
		if id[i] < 0x21 || id[i] > 0x7E {
			return false
		}
	}
	return true
}

// originAllowed enforces the spec's DNS-rebinding check when Origin is present.
// A missing Origin is allowed: MCP agents are not browsers. Loopback origins
// are allowed so a local client can talk to a local gateway. Any other present
// Origin must be on the operator allowlist.
func originAllowed(origin string, allow map[string]struct{}) bool {
	origin = strings.TrimSpace(origin)
	if origin == "" {
		return true
	}
	if strings.EqualFold(origin, "null") {
		return false
	}
	u, err := url.Parse(origin)
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") || u.User != nil {
		return false
	}
	host := strings.ToLower(u.Hostname())
	if host == "localhost" || host == "127.0.0.1" || host == "::1" {
		return true
	}
	if allow == nil {
		return false
	}
	_, ok := allow[normalizeOrigin(origin)]
	return ok
}

func normalizeOrigin(origin string) string {
	u, err := url.Parse(strings.TrimSpace(origin))
	if err != nil || u.Host == "" || (u.Scheme != "http" && u.Scheme != "https") {
		return ""
	}
	return strings.ToLower(u.Scheme) + "://" + strings.ToLower(u.Host)
}
