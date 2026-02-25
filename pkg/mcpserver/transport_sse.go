package mcpserver

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// SSETransport implements MCP over the legacy SSE protocol.
//
// Protocol:
//   - Server→Client: SSE stream (text/event-stream) on GET to base URL
//   - Client→Server: HTTP POST to the message endpoint (provided by server via "endpoint" SSE event)
//
// The SSE connection is established first. The server sends an "endpoint" event
// with the URL to POST messages to. All JSON-RPC requests are sent via POST;
// all JSON-RPC responses arrive via the SSE stream.
type SSETransport struct {
	baseURL    string
	headers    map[string]string
	httpClient *http.Client

	// messageEndpoint is set when server sends the "endpoint" SSE event.
	messageEndpoint   string
	endpointReady     chan struct{}
	endpointReadyOnce sync.Once

	// inbound messages from SSE stream
	inbox  chan json.RawMessage
	errors chan error

	// lifecycle
	ctx    context.Context
	cancel context.CancelFunc
	wg     sync.WaitGroup
	closed bool
	mu     sync.Mutex
}

// NewSSETransport creates a transport for the legacy MCP SSE protocol.
// allowPrivate bypasses SSRF protection (for local dev/testing).
func NewSSETransport(baseURL string, headers map[string]string, tlsSkipVerify bool, allowPrivate ...bool) *SSETransport {
	ctx, cancel := context.WithCancel(context.Background())

	ap := len(allowPrivate) > 0 && allowPrivate[0]
	httpClient := &http.Client{
		Timeout:   0, // no global timeout — SSE streams are long-lived
		Transport: SSRFSafeTransport(tlsSkipVerify, ap),
	}

	return &SSETransport{
		baseURL:       strings.TrimRight(baseURL, "/"),
		headers:       headers,
		httpClient:    httpClient,
		endpointReady: make(chan struct{}),
		inbox:         make(chan json.RawMessage, 64),
		errors:        make(chan error, 8),
		ctx:           ctx,
		cancel:        cancel,
	}
}

// Connect establishes the SSE stream and waits for the endpoint event.
func (t *SSETransport) Connect(ctx context.Context) error {
	t.wg.Add(1)
	go t.sseLoop()

	// Wait for endpoint event or timeout
	select {
	case <-t.endpointReady:
		log.Debug().Str("endpoint", t.messageEndpoint).Msg("SSE transport: endpoint received")
		return nil
	case err := <-t.errors:
		return fmt.Errorf("SSE connect failed: %w", err)
	case <-ctx.Done():
		return fmt.Errorf("SSE connect timeout: %w", ctx.Err())
	case <-time.After(30 * time.Second):
		return fmt.Errorf("SSE connect timeout: no endpoint event received within 30s")
	}
}

// sseLoop maintains the SSE connection with reconnection.
func (t *SSETransport) sseLoop() {
	defer t.wg.Done()

	backoff := time.Second
	maxBackoff := 30 * time.Second

	for {
		if t.ctx.Err() != nil {
			return
		}

		err := t.runSSEStream()
		if t.ctx.Err() != nil {
			return // clean shutdown
		}

		log.Warn().Err(err).Str("url", t.baseURL).Dur("backoff", backoff).
			Msg("SSE stream disconnected, reconnecting")

		select {
		case <-time.After(backoff):
			backoff = backoff * 2
			if backoff > maxBackoff {
				backoff = maxBackoff
			}
		case <-t.ctx.Done():
			return
		}
	}
}

// runSSEStream connects to the SSE endpoint and reads events until error or close.
func (t *SSETransport) runSSEStream() error {
	req, err := http.NewRequestWithContext(t.ctx, "GET", t.baseURL, nil)
	if err != nil {
		return fmt.Errorf("create SSE request: %w", err)
	}
	req.Header.Set("Accept", "text/event-stream")
	req.Header.Set("Cache-Control", "no-cache")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		t.sendError(fmt.Errorf("SSE GET %s: %w", t.baseURL, err))
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(io.LimitReader(resp.Body, 1024))
		err := fmt.Errorf("SSE GET %s: status %d: %s", t.baseURL, resp.StatusCode, string(body))
		t.sendError(err)
		return err
	}

	ct := resp.Header.Get("Content-Type")
	if !strings.HasPrefix(ct, "text/event-stream") {
		err := fmt.Errorf("SSE GET %s: unexpected content-type %q (expected text/event-stream)", t.baseURL, ct)
		t.sendError(err)
		return err
	}

	// Parse SSE stream
	scanner := bufio.NewScanner(resp.Body)
	scanner.Buffer(make([]byte, 0, 64*1024), 10*1024*1024) // 10MB max line

	var eventType string
	var dataLines []string

	for scanner.Scan() {
		line := scanner.Text()

		if line == "" {
			// Empty line = end of event
			if len(dataLines) > 0 {
				data := strings.Join(dataLines, "\n")
				t.handleSSEEvent(eventType, data)
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
			// Comment / keep-alive, ignore
			continue
		}
	}

	if err := scanner.Err(); err != nil {
		return fmt.Errorf("SSE scanner: %w", err)
	}
	return io.EOF
}

func (t *SSETransport) handleSSEEvent(eventType, data string) {
	switch eventType {
	case "endpoint":
		// Server provides the POST endpoint URL
		endpoint := strings.TrimSpace(data)
		if endpoint == "" {
			return
		}
		// Handle relative URLs
		if strings.HasPrefix(endpoint, "/") {
			// Extract scheme + host from baseURL
			parts := strings.SplitN(t.baseURL, "://", 2)
			if len(parts) == 2 {
				hostEnd := strings.Index(parts[1], "/")
				if hostEnd == -1 {
					endpoint = parts[0] + "://" + parts[1] + endpoint
				} else {
					endpoint = parts[0] + "://" + parts[1][:hostEnd] + endpoint
				}
			}
		}
		t.mu.Lock()
		t.messageEndpoint = endpoint
		t.mu.Unlock()
		t.endpointReadyOnce.Do(func() {
			close(t.endpointReady)
		})
		log.Debug().Str("endpoint", endpoint).Msg("SSE: received message endpoint")

	case "message", "":
		// JSON-RPC message from server
		if data == "" {
			return
		}
		select {
		case t.inbox <- json.RawMessage(data):
		default:
			log.Warn().Msg("SSE: inbox full, dropping message")
		}

	default:
		log.Debug().Str("event", eventType).Str("data", data[:min(len(data), 200)]).
			Msg("SSE: unknown event type")
	}
}

func (t *SSETransport) sendError(err error) {
	select {
	case t.errors <- err:
	default:
	}
}

// ReadMessage blocks until a message arrives from the SSE stream.
func (t *SSETransport) ReadMessage(ctx context.Context) (json.RawMessage, error) {
	select {
	case msg := <-t.inbox:
		return msg, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	case <-t.ctx.Done():
		return nil, fmt.Errorf("SSE transport closed")
	}
}

// WriteMessage sends a JSON-RPC message via HTTP POST to the message endpoint.
func (t *SSETransport) WriteMessage(ctx context.Context, msg json.RawMessage) error {
	t.mu.Lock()
	endpoint := t.messageEndpoint
	t.mu.Unlock()

	if endpoint == "" {
		return fmt.Errorf("SSE transport: no message endpoint available (not connected?)")
	}

	req, err := http.NewRequestWithContext(ctx, "POST", endpoint, bytes.NewReader(msg))
	if err != nil {
		return fmt.Errorf("SSE POST create: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	for k, v := range t.headers {
		req.Header.Set(k, v)
	}

	resp, err := t.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("SSE POST %s: %w", endpoint, err)
	}
	defer resp.Body.Close()

	// Drain body to allow connection reuse
	io.Copy(io.Discard, io.LimitReader(resp.Body, 64*1024))

	if resp.StatusCode >= 400 {
		return fmt.Errorf("SSE POST %s: status %d", endpoint, resp.StatusCode)
	}

	return nil
}

// Close shuts down the SSE transport.
func (t *SSETransport) Close() error {
	t.mu.Lock()
	if t.closed {
		t.mu.Unlock()
		return nil
	}
	t.closed = true
	t.mu.Unlock()

	t.cancel()
	t.wg.Wait()
	return nil
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
