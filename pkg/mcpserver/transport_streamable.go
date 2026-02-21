package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// StreamableHTTPTransport implements MCP over the Streamable HTTP protocol (2025-03-26 spec).
//
// Protocol:
//   - Single endpoint: all communication via HTTP POST to one URL
//   - Request: POST with Content-Type: application/json
//   - Response: either application/json (single response) or text/event-stream (streaming)
//   - Session: server may provide Mcp-Session-Id in response headers; client echoes it back
//   - Notifications: GET to same endpoint opens SSE stream for server-initiated messages
//   - Termination: DELETE to same endpoint closes the session
type StreamableHTTPTransport struct {
	url        string
	headers    map[string]string
	httpClient *http.Client

	// Session management
	sessionID string
	sessionMu sync.RWMutex

	// Inbound messages (from SSE responses and notification stream)
	inbox  chan json.RawMessage
	errors chan error

	// Notification SSE stream
	notifCtx    context.Context
	notifCancel context.CancelFunc
	notifWg     sync.WaitGroup

	// Lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	closed bool
	mu     sync.Mutex
}

// NewStreamableHTTPTransport creates a transport for the MCP Streamable HTTP protocol.
func NewStreamableHTTPTransport(url string, headers map[string]string, tlsSkipVerify bool) *StreamableHTTPTransport {
	ctx, cancel := context.WithCancel(context.Background())
	notifCtx, notifCancel := context.WithCancel(ctx)

	transport := &http.Transport{
		MaxIdleConns:        10,
		MaxIdleConnsPerHost: 10,
		IdleConnTimeout:     90 * time.Second,
	}
	if tlsSkipVerify {
		transport.TLSClientConfig = &tls.Config{InsecureSkipVerify: true}
	}

	httpClient := &http.Client{
		Transport: transport,
		Timeout:   0, // per-request timeouts via context
	}

	return &StreamableHTTPTransport{
		url:         strings.TrimRight(url, "/"),
		headers:     headers,
		httpClient:  httpClient,
		inbox:       make(chan json.RawMessage, 64),
		errors:      make(chan error, 8),
		ctx:         ctx,
		cancel:      cancel,
		notifCtx:    notifCtx,
		notifCancel: notifCancel,
	}
}

// Connect validates the transport URL and optionally starts the notification listener.
// For Streamable HTTP, the actual MCP session is established with the first request
// (initialize), not during connect.
func (t *StreamableHTTPTransport) Connect(ctx context.Context) error {
	// Validate the URL is well-formed — don't probe the server with GET here
	// because the notification stream (SSE) would block. The notification listener
	// is started lazily after the first successful POST that returns a session ID.
	_, err := http.NewRequest("GET", t.url, nil)
	if err != nil {
		return fmt.Errorf("streamable HTTP: invalid URL %q: %w", t.url, err)
	}

	log.Debug().Str("url", t.url).Msg("streamable HTTP transport: ready")
	return nil
}

// maybeStartNotificationListener starts the notification SSE stream after we
// have a valid session ID. Called once after the first successful POST.
func (t *StreamableHTTPTransport) maybeStartNotificationListener() {
	t.notifWg.Add(1)
	go func() {
		defer t.notifWg.Done()

		// Small delay to let the initialize handshake complete
		select {
		case <-time.After(100 * time.Millisecond):
		case <-t.notifCtx.Done():
			return
		}

		// Try a GET — if server supports it, keep streaming
		req, err := http.NewRequestWithContext(t.notifCtx, "GET", t.url, nil)
		if err != nil {
			return
		}
		req.Header.Set("Accept", "text/event-stream")
		for k, v := range t.headers {
			req.Header.Set(k, v)
		}
		t.sessionMu.RLock()
		if t.sessionID != "" {
			req.Header.Set("Mcp-Session-Id", t.sessionID)
		}
		t.sessionMu.RUnlock()

		resp, err := t.httpClient.Do(req)
		if err != nil {
			log.Debug().Err(err).Msg("streamable HTTP: notification stream not available")
			return
		}

		if resp.StatusCode != http.StatusOK ||
			!strings.HasPrefix(resp.Header.Get("Content-Type"), "text/event-stream") {
			io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
			resp.Body.Close()
			log.Debug().Int("status", resp.StatusCode).Msg("streamable HTTP: server doesn't support notification stream")
			return
		}

		defer resp.Body.Close()
		t.readSSEStream(resp.Body)
	}()
}

// (notification listener is started lazily via maybeStartNotificationListener)

// ReadMessage blocks until a message arrives from either a POST response or the notification stream.
func (t *StreamableHTTPTransport) ReadMessage(ctx context.Context) (json.RawMessage, error) {
	select {
	case msg := <-t.inbox:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.ctx.Done():
		return nil, fmt.Errorf("streamable HTTP transport closed")
	}
}

// WriteMessage sends a JSON-RPC message via HTTP POST and processes the response.
// The response may be a single JSON object or an SSE stream of multiple messages.
func (t *StreamableHTTPTransport) WriteMessage(ctx context.Context, msg json.RawMessage) error {
	req, err := http.NewRequestWithContext(ctx, "POST", t.url, bytes.NewReader(msg))
	if err != nil {
		return fmt.Errorf("streamable HTTP POST create: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}
	t.sessionMu.RLock()
	if t.sessionID != "" {
		req.Header.Set("Mcp-Session-Id", t.sessionID)
	}
	t.sessionMu.RUnlock()

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("streamable HTTP POST %s: %w", t.url, err)
	}

	// Capture session ID from response and start notification listener once
	if sid := resp.Header.Get("Mcp-Session-Id"); sid != "" {
		t.sessionMu.Lock()
		isNew := t.sessionID == ""
		t.sessionID = sid
		t.sessionMu.Unlock()
		if isNew {
			log.Debug().Str("sessionId", sid).Msg("streamable HTTP: session established")
			t.maybeStartNotificationListener()
		}
	}

	if resp.StatusCode >= 400 {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 4096))
		resp.Body.Close()
		return fmt.Errorf("streamable HTTP POST %s: status %d: %s", t.url, resp.StatusCode, string(body))
	}

	ct := resp.Header.Get("Content-Type")

	switch {
	case resp.StatusCode == http.StatusNoContent || resp.ContentLength == 0:
		// Notification accepted, no response body (e.g., notifications/initialized)
		resp.Body.Close()
		return nil

	case strings.HasPrefix(ct, "text/event-stream"):
		// Streaming response — read SSE events in background
		go func() {
			defer resp.Body.Close()
			if err := t.readSSEStream(resp.Body); err != nil && t.ctx.Err() == nil {
				log.Debug().Err(err).Msg("streamable HTTP: SSE response stream ended")
			}
		}()
		return nil

	case strings.HasPrefix(ct, "application/json"):
		// Single JSON response
		defer resp.Body.Close()
		body, err := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024)) // 10MB max
		if err != nil {
			return fmt.Errorf("streamable HTTP: read response body: %w", err)
		}

		// Check if it's a JSON-RPC batch (array)
		trimmed := bytes.TrimSpace(body)
		if len(trimmed) > 0 && trimmed[0] == '[' {
			var batch []json.RawMessage
			if err := json.Unmarshal(trimmed, &batch); err != nil {
				// Not a valid batch — deliver as-is
				t.deliverMessage(json.RawMessage(trimmed))
				return nil
			}
			for _, item := range batch {
				t.deliverMessage(item)
			}
			return nil
		}

		t.deliverMessage(json.RawMessage(trimmed))
		return nil

	default:
		// Unknown content type — try to read as JSON
		defer resp.Body.Close()
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 10*1024*1024))
		if len(body) > 0 {
			t.deliverMessage(json.RawMessage(body))
		}
		return nil
	}
}

// readSSEStream parses an SSE stream body and delivers messages to the inbox.
func (t *StreamableHTTPTransport) readSSEStream(body io.Reader) error {
	scanner := bufio.NewScanner(body)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024)

	var eventType string
	var dataLines []string

	for scanner.Scan() {
		if t.ctx.Err() != nil {
			return t.ctx.Err()
		}

		line := scanner.Text()

		if line == "" {
			// End of event
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				t.handleStreamEvent(eventType, data)
				eventType = ""
				dataLines = nil
			}
			continue
		}

		if strings.HasPrefix(line, "event:") {
			eventType = strings.TrimSpace(strings.TrimPrefix(line, "event:"))
		} else if strings.HasPrefix(line, "data:") {
			dataLines = append(dataLines, strings.TrimSpace(strings.TrimPrefix(line, "data:")))
		} else if strings.HasPrefix(line, ":") {
			// Comment / keep-alive
			continue
		}
	}

	// Flush any remaining event
	if len(dataLines) > 0 {
		data := strings.Join(dataLines, "\n")
		t.handleStreamEvent(eventType, data)
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("SSE scanner: %w", err)
	}
	return io.EOF
}

func (t *StreamableHTTPTransport) handleStreamEvent(eventType, data string) {
	if data == "" {
		return
	}

	switch eventType {
	case "message", "":
		t.deliverMessage(json.RawMessage(data))
	default:
		log.Debug().Str("event", eventType).Str("data", data[:minInt(len(data), 200)]).
			Msg("streamable HTTP: unknown SSE event")
	}
}

func (t *StreamableHTTPTransport) deliverMessage(msg json.RawMessage) {
	select {
	case t.inbox <- msg:
	default:
		log.Warn().Msg("streamable HTTP: inbox full, dropping message")
	}
}

// Close shuts down the Streamable HTTP transport and terminates the server session.
func (t *StreamableHTTPTransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()

	// Stop notification listener
	t.notifCancel()
	t.notifWg.Wait()

	// Send DELETE to terminate server session
	t.sessionMu.RLock()
	sid := t.sessionID
	t.sessionMu.RUnlock()

	if sid != "" {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		req, err := http.NewRequestWithContext(ctx, "DELETE", t.url, nil)
		if err == nil {
			req.Header.Set("Mcp-Session-Id", sid)
			for k, v := range t.headers {
				req.Header.Set(k, v)
			}
			resp, err := t.httpClient.Do(req)
			if err != nil {
				log.Debug().Err(err).Msg("streamable HTTP: session DELETE failed")
			} else {
				io.Copy(io.Discard, io.LimitReader(resp.Body, 1024))
				resp.Body.Close()
				log.Debug().Int("status", resp.StatusCode).Msg("streamable HTTP: session terminated")
			}
		}
	}

	t.cancel()
	return nil
}

func minInt(a, b int) int {
	if a < b {
		return a
	}
	return b
}
