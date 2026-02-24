package mcpserver

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// SSEServer implements the MCP SSE transport.
// Each client connects via GET /sse (event stream) and sends messages via POST /message.
// This enables multiple agents to connect to a single MCP proxy over HTTP.
type SSEServer struct {
	proxy  *Proxy
	mux    *http.ServeMux
	server *http.Server

	mu       sync.Mutex
	sessions map[string]*sseSession
}

// sseSession represents one connected MCP client over SSE.
type sseSession struct {
	id       string
	messages chan json.RawMessage // outbound messages to client
	ctx      context.Context
	cancel   context.CancelFunc
	tokenID  string // from auth at connect time (may be empty)
	tenantID string // from auth at connect time (may be empty)
	budgetID string // from auth at connect time (may be empty)
}

// NewSSEServer creates an SSE transport server for the given proxy.
func NewSSEServer(proxy *Proxy, addr string) *SSEServer {
	s := &SSEServer{
		proxy:    proxy,
		sessions: make(map[string]*sseSession),
	}

	mux := http.NewServeMux()
	mux.HandleFunc("/sse", s.handleSSE)
	mux.HandleFunc("/message", s.handleMessage)
	mux.HandleFunc("/health", s.handleHealth)

	s.mux = mux
	s.server = &http.Server{
		Addr:         addr,
		Handler:      mux,
		ReadTimeout:  0, // SSE needs no read timeout
		WriteTimeout: 0, // SSE needs no write timeout
		IdleTimeout:  0,
	}

	return s
}

// Handle registers an additional handler on the SSE server's mux.
func (s *SSEServer) Handle(pattern string, handler http.Handler) {
	s.mux.Handle(pattern, handler)
}

// HandleFunc registers an additional handler function on the SSE server's mux.
func (s *SSEServer) HandleFunc(pattern string, handler http.HandlerFunc) {
	s.mux.HandleFunc(pattern, handler)
}

// ListenAndServe starts the SSE server. Blocks until context cancelled.
func (s *SSEServer) ListenAndServe(ctx context.Context) error {
	// Start upstreams first
	if err := s.proxy.upstream.Start(ctx); err != nil {
		return fmt.Errorf("start upstreams: %w", err)
	}

	log.Info().
		Str("addr", s.server.Addr).
		Str("auth", s.proxy.config.Auth.Mode).
		Str("enforcement", s.proxy.config.Enforcement.Mode).
		Int64("budget", s.proxy.config.Budget.Limit).
		Msg("MCP SSE server listening")

	// Graceful shutdown on context cancel
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		s.server.Shutdown(shutdownCtx)
		s.proxy.upstream.Close()
	}()

	if err := s.server.ListenAndServe(); err != nil && err != http.ErrServerClosed {
		return err
	}
	return nil
}

// handleSSE establishes an SSE connection for a client.
// GET /sse
// Response: text/event-stream with JSON-RPC messages as "message" events.
// The endpoint URL for posting messages is sent as the first event.
func (s *SSEServer) handleSSE(w http.ResponseWriter, r *http.Request) {
	flusher, ok := w.(http.Flusher)
	if !ok {
		http.Error(w, "streaming not supported", http.StatusInternalServerError)
		return
	}

	// Create session
	sessionID := generateSessionID()
	ctx, cancel := context.WithCancel(r.Context())
	session := &sseSession{
		id:       sessionID,
		messages: make(chan json.RawMessage, 64),
		ctx:      ctx,
		cancel:   cancel,
	}

	// Capture auth token from header before writing response.
	// Verification happens after SSE stream is established to avoid blocking.
	authToken := extractAuthToken(r)

	s.mu.Lock()
	s.sessions[sessionID] = session
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		delete(s.sessions, sessionID)
		s.mu.Unlock()
		cancel()
		log.Info().Str("session", sessionID).Msg("SSE session closed")
		s.proxy.events.Publish(Event{
			Type:      EventSessionClose,
			Timestamp: time.Now(),
			SessionID: sessionID,
			TokenID:   session.tokenID,
			TenantID:  session.tenantID,
			BudgetID:  session.budgetID,
		})
	}()

	// Set SSE headers and flush FIRST — the client needs the stream open
	// before we do any potentially slow operations (auth verify, Redis publish).
	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")
	w.Header().Set("X-Accel-Buffering", "no") // nginx
	w.WriteHeader(http.StatusOK)

	// Send endpoint event — tells the client where to POST messages
	messageURL := fmt.Sprintf("/message?sessionId=%s", sessionID)
	fmt.Fprintf(w, "event: endpoint\ndata: %s\n\n", messageURL)
	flusher.Flush()

	// Now extract session identity from auth token (after stream is open).
	// Try full verification first; fall back to caveat extraction if verify fails
	// (e.g., token signed by a different root key in multi-tenant SaaS).
	if authToken != "" {
		var resolved bool
		if s.proxy.auth != nil {
			if info, err := s.proxy.auth.Verify(ctx, authToken); err == nil {
				session.tokenID = info.TokenID
				session.tenantID = info.TenantID
				session.budgetID = info.BudgetID
				resolved = true
			}
		}
		if !resolved {
			// Best-effort: decode macaroon and read caveats for dashboard tracking.
			// This is NOT security enforcement — tool calls are verified on /message.
			if tid, bid := extractTokenCaveats(authToken); tid != "" {
				session.tenantID = tid
				session.budgetID = bid
			}
		}
	}

	log.Info().Str("session", sessionID).Str("tenant", session.tenantID).Msg("SSE session established")
	s.proxy.events.Publish(Event{
		Type:      EventSessionConnect,
		Timestamp: time.Now(),
		SessionID: sessionID,
		TokenID:   session.tokenID,
		TenantID:  session.tenantID,
		BudgetID:  session.budgetID,
	})

	// Stream outbound messages with keepalive
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case msg := <-session.messages:
			fmt.Fprintf(w, "event: message\ndata: %s\n\n", string(msg))
			flusher.Flush()

		case <-keepalive.C:
			// SSE comment line — keeps connection alive through proxies/load balancers
			fmt.Fprintf(w, ": keepalive\n\n")
			flusher.Flush()
			// Publish keepalive event so enterprise Redis TTLs stay fresh
			s.proxy.events.Publish(Event{
				Type:      EventSessionKeepalive,
				Timestamp: time.Now(),
				SessionID: sessionID,
				TokenID:   session.tokenID,
				TenantID:  session.tenantID,
				BudgetID:  session.budgetID,
			})

		case <-ctx.Done():
			return

		case <-r.Context().Done():
			return
		}
	}
}

// handleMessage receives a JSON-RPC message from a client.
// POST /message?sessionId=xxx
// Body: JSON-RPC request
// Response: 202 Accepted (response comes via SSE)
func (s *SSEServer) handleMessage(w http.ResponseWriter, r *http.Request) {
	if r.Method != http.MethodPost {
		http.Error(w, "POST required", http.StatusMethodNotAllowed)
		return
	}

	sessionID := r.URL.Query().Get("sessionId")
	if sessionID == "" {
		http.Error(w, "sessionId required", http.StatusBadRequest)
		return
	}

	s.mu.Lock()
	session, ok := s.sessions[sessionID]
	s.mu.Unlock()
	if !ok {
		http.Error(w, "session not found", http.StatusNotFound)
		return
	}

	// Read request body
	body, err := io.ReadAll(io.LimitReader(r.Body, 10*1024*1024)) // 10MB max
	if err != nil {
		http.Error(w, "read error", http.StatusBadRequest)
		return
	}

	// Also check Authorization header for token (SSE auth)
	authToken := extractAuthToken(r)

	// Late identity resolution: some clients (e.g., Cursor) don't send the
	// Authorization header on the initial SSE GET — only on POST /message.
	// When we see a token here for the first time, extract tenant/budget info
	// and re-publish a session_connect event so the dashboard picks it up.
	if authToken != "" && session.tenantID == "" {
		var resolved bool
		if s.proxy.auth != nil {
			if info, err := s.proxy.auth.Verify(session.ctx, authToken); err == nil {
				session.tokenID = info.TokenID
				session.tenantID = info.TenantID
				session.budgetID = info.BudgetID
				resolved = true
			}
		}
		if !resolved {
			if tid, bid := extractTokenCaveats(authToken); tid != "" {
				session.tenantID = tid
				session.budgetID = bid
			}
		}
		// Re-publish session_connect with correct identity
		if session.tenantID != "" {
			log.Info().Str("session", sessionID).Str("tenant", session.tenantID).Msg("session identity resolved from message auth")
			s.proxy.events.Publish(Event{
				Type:      EventSessionConnect,
				Timestamp: time.Now(),
				SessionID: sessionID,
				TokenID:   session.tokenID,
				TenantID:  session.tenantID,
				BudgetID:  session.budgetID,
			})
		}
	}

	// Parse and handle
	req, _, parseErr := ParseMessage(json.RawMessage(body))
	if parseErr != nil {
		http.Error(w, "invalid JSON-RPC", http.StatusBadRequest)
		return
	}

	if req == nil {
		w.WriteHeader(http.StatusAccepted)
		return
	}

	// Inject auth token into params._meta if from header
	if authToken != "" && req.Params != nil {
		req.Params = injectMetaToken(req.Params, authToken)
	} else if authToken != "" && req.Params == nil {
		req.Params = json.RawMessage(fmt.Sprintf(`{"_meta":{"token":"%s"}}`, authToken))
	}

	// Handle request asynchronously — don't block the HTTP response
	go func() {
		// Inject session's tenant ID into context so tools/list and other
		// methods can access it (for multi-tenant upstream routing).
		reqCtx := session.ctx
		if session.tenantID != "" {
			reqCtx = context.WithValue(reqCtx, CtxTenantID, session.tenantID)
		}
		resp, handleErr := s.proxy.handleRequest(reqCtx, req)
		if handleErr != nil {
			resp = NewErrorResponse(req.ID, CodeInternalError, handleErr.Error())
		}

		if resp != nil {
			data, _ := json.Marshal(resp)
			// Block up to 5s for buffer space; if still full, drop with error log.
			// Clients should handle missing responses via JSON-RPC request timeouts.
			select {
			case session.messages <- data:
			case <-time.After(5 * time.Second):
				log.Error().Str("session", sessionID).Str("reqId", string(req.ID)).
					Msg("SSE message buffer full after 5s, dropping response")
			case <-session.ctx.Done():
				log.Debug().Str("session", sessionID).Msg("session closed during send")
			}
		}
	}()

	w.WriteHeader(http.StatusAccepted)
}

func (s *SSEServer) handleHealth(w http.ResponseWriter, _ *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status":  "ok",
		"server":  s.proxy.config.Server.Name,
		"version": s.proxy.config.Server.Version,
		"sessions": func() int {
			s.mu.Lock()
			defer s.mu.Unlock()
			return len(s.sessions)
		}(),
	})
}

// extractAuthToken pulls Bearer token from Authorization header.
func extractAuthToken(r *http.Request) string {
	// Check Authorization header first
	auth := r.Header.Get("Authorization")
	if auth != "" {
		return strings.TrimPrefix(auth, "Bearer ")
	}
	// Fall back to ?token= query parameter (used by SSE clients like Cursor)
	if token := r.URL.Query().Get("token"); token != "" {
		return token
	}
	return ""
}

// injectMetaToken adds _meta.token to params JSON if not already present.
func injectMetaToken(params json.RawMessage, token string) json.RawMessage {
	var m map[string]json.RawMessage
	if err := json.Unmarshal(params, &m); err != nil {
		return params
	}

	// Check if _meta already has a token
	if metaRaw, ok := m["_meta"]; ok {
		var meta map[string]interface{}
		if err := json.Unmarshal(metaRaw, &meta); err == nil {
			if _, hasToken := meta["token"]; hasToken {
				return params // already has token, don't override
			}
			meta["token"] = token
			metaJSON, _ := json.Marshal(meta)
			m["_meta"] = metaJSON
		}
	} else {
		metaJSON, _ := json.Marshal(map[string]string{"token": token})
		m["_meta"] = metaJSON
	}

	result, _ := json.Marshal(m)
	return result
}

// generateSessionID creates a short unique session ID.
func generateSessionID() string {
	return fmt.Sprintf("mcp-%s", hashToken(fmt.Sprintf("%d", time.Now().UnixNano()))[:8])
}

// extractTokenCaveats does a best-effort decode of a macaroon token to read
// tenant_id and budget_id caveats without cryptographic verification.
// Used for dashboard session tracking when the SSE proxy can't verify the token
// (e.g., token signed by a different per-tenant root key in multi-tenant SaaS).
func extractTokenCaveats(token string) (tenantID, budgetID string) {
	// Macaroon tokens are base64-encoded JSON with a "c" (caveats) array
	decoded, err := base64.RawURLEncoding.DecodeString(token)
	if err != nil {
		// Try standard base64
		decoded, err = base64.StdEncoding.DecodeString(token)
		if err != nil {
			return "", ""
		}
	}

	var raw struct {
		Caveats []string `json:"c"`
	}
	if err := json.Unmarshal(decoded, &raw); err != nil {
		return "", ""
	}

	for _, c := range raw.Caveats {
		if strings.HasPrefix(c, "tenant_id = ") {
			tenantID = strings.TrimPrefix(c, "tenant_id = ")
		} else if strings.HasPrefix(c, "budget_id = ") {
			budgetID = strings.TrimPrefix(c, "budget_id = ")
		}
	}
	return tenantID, budgetID
}
