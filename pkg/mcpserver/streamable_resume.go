package mcpserver

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"net/http"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
)

// Resuming a Streamable HTTP session after a restart.
//
// Sessions live in process memory, so a deploy, a machine restart or a
// request that lands on another machine loses them. The MCP spec says a
// client that gets 404 for its Mcp-Session-Id starts a new session, but
// clients in use today (Claude Code 2.1.92) show the error and stop. So in
// header auth mode a request that presents an unknown session id is served
// under that same id when, and only when:
//
//   - the bearer token on the request verifies, exactly as an initialize
//     must (a revoked token gets the revoked answer, a missing or failing
//     token gets 401);
//   - the session id was issued to that token: every id carries a tag that
//     is an HMAC, keyed by the token's SHA-256, over the id's random part
//     (streamSessionIDFor). Another token, another account, or an id this
//     server did not issue cannot take the id over, even before the owner
//     comes back. A tag mismatch is the 404 it was before;
//   - the id was not ended with DELETE on this process (a tombstone is kept
//     for the idle TTL).
//
// The id grants nothing by itself: the resumed session is bound to the token
// hash and verified identity of the request that resumed it, the same state
// initialize would build, and every later request is checked against it as
// before. What a token may call, its budget, rate and holds are keyed by the
// token and budget, never by the session, so a new or resumed session changes
// none of them.

// streamSessionIDTagLabel separates this HMAC use from any other.
const streamSessionIDTagLabel = "satgate/mcp-session-id/v1\x00"

// streamSessionIDFor returns a new session id bound to token:
// "mcp-" + 32 hex (random) + "-" + 32 hex (tag).
func streamSessionIDFor(token string) string {
	nonce := strings.TrimPrefix(generateSessionID(), "mcp-")
	return "mcp-" + nonce + "-" + streamSessionIDTag(nonce, token)
}

func streamSessionIDTag(nonce, token string) string {
	key := sha256.Sum256([]byte(token))
	mac := hmac.New(sha256.New, key[:])
	_, _ = mac.Write([]byte(streamSessionIDTagLabel))
	_, _ = mac.Write([]byte(nonce))
	return hex.EncodeToString(mac.Sum(nil)[:16])
}

// streamSessionIDBoundTo reports whether id was issued by streamSessionIDFor
// for token. Ids in any other shape (issued before this change, or made up)
// are not bound to anything and cannot be resumed.
func streamSessionIDBoundTo(id, token string) bool {
	rest, ok := strings.CutPrefix(id, "mcp-")
	if !ok {
		return false
	}
	nonce, tag, ok := strings.Cut(rest, "-")
	if !ok || len(nonce) != 32 || len(tag) != 32 || !isLowerHex(nonce) || !isLowerHex(tag) {
		return false
	}
	return hmac.Equal([]byte(tag), []byte(streamSessionIDTag(nonce, token)))
}

func isLowerHex(s string) bool {
	for i := 0; i < len(s); i++ {
		c := s[i]
		if (c < '0' || c > '9') && (c < 'a' || c > 'f') {
			return false
		}
	}
	return true
}

// resumeStreamSession serves a request whose session id this process does not
// hold. It either returns the session to use (resumed, or one a concurrent
// request resumed first) or writes the refusal and returns false.
func (s *SSEServer) resumeStreamSession(w http.ResponseWriter, r *http.Request, sid string) (*streamSession, bool) {
	if !s.requiresVerifiedStreamToken() || s.streamSessionEnded(sid) {
		http.Error(w, "session not found", http.StatusNotFound)
		return nil, false
	}
	token := extractAuthToken(r)
	// The same token checks an initialize gets, in the same order.
	if s.rejectHardAuthFailure(w, r, token) {
		return nil, false
	}
	if token == "" {
		s.writeAuthChallenge(w, "authentication required")
		return nil, false
	}
	info, err := s.proxy.auth.Verify(r.Context(), token)
	if err != nil || info == nil {
		s.writeAuthChallenge(w, "authentication failed")
		return nil, false
	}
	if s.proxy.revocation != nil && info.BudgetID != "" && s.proxy.revocation.IsRevoked(r.Context(), info.BudgetID) {
		s.writePreConnectDenial(w, r, "token revoked")
		return nil, false
	}
	if !streamSessionIDBoundTo(sid, token) {
		// Not this token's session (or an id from before session ids were
		// bound): the client starts a new session, as before.
		http.Error(w, "session not found", http.StatusNotFound)
		return nil, false
	}
	protocol := protocolVersion20250326 // spec: a missing header means 2025-03-26
	if header := strings.TrimSpace(r.Header.Get(headerMCPProtocolVersion)); header != "" {
		v, ok := supportedProtocolVersion(header)
		if !ok {
			http.Error(w, "unsupported MCP-Protocol-Version", http.StatusBadRequest)
			return nil, false
		}
		protocol = v
	}
	sess, created := s.addStreamSession(r.Context(), sid, token, protocol, true)
	if !created {
		w.Header().Set("Retry-After", "60")
		http.Error(w, "too many sessions", http.StatusServiceUnavailable)
		return nil, false
	}
	// A concurrent request may have resumed the id first; the binding check
	// applies to whichever session is in the map.
	if s.rejectStreamSessionToken(w, r, sess) {
		return nil, false
	}
	return sess, true
}

// markStreamSessionEnded remembers an id ended with DELETE for the idle TTL,
// so this process does not bring it back. Caller holds streamMu.
func (s *SSEServer) markStreamSessionEndedLocked(id string, now time.Time) {
	if s.streamEnded == nil {
		s.streamEnded = make(map[string]time.Time)
	}
	if len(s.streamEnded) >= s.streamMaxSessions() {
		for k, until := range s.streamEnded {
			if now.After(until) {
				delete(s.streamEnded, k)
			}
		}
		for k := range s.streamEnded {
			if len(s.streamEnded) < s.streamMaxSessions() {
				break
			}
			delete(s.streamEnded, k)
		}
	}
	s.streamEnded[id] = now.Add(s.streamIdleTTL())
}

func (s *SSEServer) streamSessionEnded(id string) bool {
	s.streamMu.Lock()
	defer s.streamMu.Unlock()
	until, ok := s.streamEnded[id]
	if !ok {
		return false
	}
	if time.Now().After(until) {
		delete(s.streamEnded, id)
		return false
	}
	return true
}

func (s *SSEServer) publishStreamConnect(sess *streamSession, resumed bool) {
	if s.proxy == nil || s.proxy.events == nil || sess == nil {
		return
	}
	ev := Event{
		Type:      EventSessionConnect,
		Timestamp: time.Now(),
		SessionID: sess.id,
		TokenID:   sess.identity.tokenID,
		TenantID:  sess.identity.tenantID,
		BudgetID:  sess.identity.budgetID,
	}
	if resumed {
		ev.Data = map[string]interface{}{"resumed": true}
	}
	s.proxy.events.Publish(ev)
}

func logStreamSession(sess *streamSession, resumed bool) {
	msg := "streamable HTTP session established"
	if resumed {
		msg = "streamable HTTP session resumed"
	}
	log.Info().Str("session", sess.id).Str("protocol", sess.protocol).Msg(msg)
}
