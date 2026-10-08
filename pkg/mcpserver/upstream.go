package mcpserver

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/rs/zerolog/log"
)

// UpstreamManager manages connections to upstream MCP servers.
type UpstreamManager struct {
	config                map[string]UpstreamConfig
	routing               []RoutingRule
	defaultUp             string
	allowPrivateUpstreams bool

	mu        sync.Mutex
	clients   map[string]*UpstreamClient
	idCounter atomic.Int64

	// redact keeps upstream-supplied text out of errors and logs. See
	// SetRedactUpstreamText.
	redactOn atomic.Bool

	// closed is set by Close. A closed manager starts and adds nothing.
	closed atomic.Bool

	// inflight holds clients that a read loop has dialed to replace a dead
	// transport and not yet published, so Close can end them.
	inflight map[*UpstreamClient]struct{}
}

// track registers a freshly dialed replacement client. It reports false when
// the old client or the manager was shut down already.
func (m *UpstreamManager) track(old, fresh *UpstreamClient, ctx context.Context) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	if m.closed.Load() || old.stopped(ctx) {
		return false
	}
	if m.inflight == nil {
		m.inflight = map[*UpstreamClient]struct{}{}
	}
	m.inflight[fresh] = struct{}{}
	return true
}

func (m *UpstreamManager) untrack(c *UpstreamClient) {
	m.mu.Lock()
	delete(m.inflight, c)
	m.mu.Unlock()
}

// SetRedactUpstreamText makes the manager and the transports it builds leave
// upstream-supplied text out of the errors they return and the logs they
// write: JSON-RPC error messages (the code is kept), response bodies, content
// types, event data, session ids, tool names and redirect or endpoint origins.
// Failures keep their kind and their HTTP status. Call it before Start. Off by
// default; when unset nothing changes. Safe to call at any time; connections
// already made keep the setting they were made with.
func (m *UpstreamManager) SetRedactUpstreamText(on bool) { m.redactOn.Store(on) }

// redacting reports whether upstream-supplied text is kept out of errors and
// logs.
func (m *UpstreamManager) redacting() bool { return m.redactOn.Load() }

// UpstreamClient wraps a transport to an upstream MCP server
// with request/response correlation.
type UpstreamClient struct {
	name      string
	transport Transport
	pending   sync.Map // id (string) -> chan *Response
	tools     []json.RawMessage
	toolNames []string
	ready     bool

	// closed is set before the manager closes the transport on purpose, so
	// readLoop treats the read error that follows as the end of the client
	// and does not reconnect.
	closed atomic.Bool

	// doneCh is closed by shutdown so a read loop waiting out a reconnect
	// backoff wakes at once instead of sleeping on and dialing again.
	doneInit  sync.Once
	doneClose sync.Once
	doneCh    chan struct{}
}

// done returns a channel that is closed when the client is shut down.
func (c *UpstreamClient) done() <-chan struct{} {
	c.doneInit.Do(func() { c.doneCh = make(chan struct{}) })
	return c.doneCh
}

// shutdown closes the client's transport on purpose.
func (c *UpstreamClient) shutdown() error {
	c.closed.Store(true)
	c.done()
	c.doneClose.Do(func() { close(c.doneCh) })
	return c.transport.Close()
}

// waitBackoff waits d and reports whether the caller may go on: false when the
// context ended or the client was shut down during the wait (or before it).
func (c *UpstreamClient) waitBackoff(ctx context.Context, d time.Duration) bool {
	if ctx.Err() != nil || c.closed.Load() {
		return false
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-t.C:
	case <-ctx.Done():
		return false
	case <-c.done():
		return false
	}
	return ctx.Err() == nil && !c.closed.Load()
}

// stopped reports whether the client must not dial, publish or log any more.
func (c *UpstreamClient) stopped(ctx context.Context) bool {
	return ctx.Err() != nil || c.closed.Load()
}

// NewUpstreamManager creates a new upstream manager.
func NewUpstreamManager(config map[string]UpstreamConfig, routing []RoutingRule, defaultUpstream string, allowPrivateUpstreams ...bool) *UpstreamManager {
	allowPrivate := len(allowPrivateUpstreams) > 0 && allowPrivateUpstreams[0]
	return &UpstreamManager{
		config:                config,
		routing:               routing,
		defaultUp:             defaultUpstream,
		allowPrivateUpstreams: allowPrivate,
		clients:               make(map[string]*UpstreamClient),
	}
}

// Start launches all upstream connections and discovers tools.
func (m *UpstreamManager) Start(ctx context.Context) error {
	if m.closed.Load() {
		return fmt.Errorf("upstream manager is closed")
	}
	if len(m.config) <= 1 {
		// Single upstream — sequential (no goroutine overhead)
		for name, cfg := range m.config {
			if err := m.startOne(ctx, name, cfg); err != nil {
				return err
			}
		}
		return nil
	}

	// Multiple upstreams — connect in parallel for faster tool discovery
	type result struct {
		name   string
		client *UpstreamClient
		err    error
	}
	results := make(chan result, len(m.config))

	for name, cfg := range m.config {
		go func(n string, c UpstreamConfig) {
			client, err := m.connect(ctx, n, c)
			if err != nil {
				results <- result{n, nil, fmt.Errorf("upstream %q: %w", n, err)}
				return
			}
			go m.readLoop(ctx, client)
			if err := m.initializeUpstream(ctx, client); err != nil {
				client.shutdown()
				results <- result{n, nil, fmt.Errorf("upstream %q initialize: %w", n, err)}
				return
			}
			if err := m.discoverTools(ctx, client); err != nil {
				client.shutdown()
				results <- result{n, nil, fmt.Errorf("upstream %q tools/list: %w", n, err)}
				return
			}
			results <- result{n, client, nil}
		}(name, cfg)
	}

	var firstErr error
	for i := 0; i < len(m.config); i++ {
		r := <-results
		if r.err != nil {
			if firstErr == nil {
				firstErr = r.err
			}
			log.Error().Err(r.err).Str("upstream", r.name).Msg("upstream connect failed (parallel)")
			continue
		}
		m.mu.Lock()
		if m.closed.Load() {
			m.mu.Unlock()
			r.client.shutdown()
			continue
		}
		m.clients[r.name] = r.client
		m.mu.Unlock()
		m.logConnected(r.name, r.client)
	}

	// Fail only if ALL upstreams failed; partial success is OK
	if len(m.clients) == 0 && firstErr != nil {
		return firstErr
	}
	return nil
}

// startOne connects a single upstream sequentially (connect → init → discover).
func (m *UpstreamManager) startOne(ctx context.Context, name string, cfg UpstreamConfig) error {
	client, err := m.connect(ctx, name, cfg)
	if err != nil {
		return fmt.Errorf("upstream %q: %w", name, err)
	}
	m.mu.Lock()
	m.clients[name] = client
	m.mu.Unlock()

	go m.readLoop(ctx, client)

	// A start that fails leaves nothing behind: the transport is closed on
	// purpose, so its read loop ends instead of reconnecting with the
	// upstream's credentials, and the client is not kept.
	fail := func(err error) error {
		m.mu.Lock()
		if m.clients[name] == client {
			delete(m.clients, name)
		}
		m.mu.Unlock()
		client.shutdown()
		return err
	}
	if err := m.initializeUpstream(ctx, client); err != nil {
		return fail(fmt.Errorf("upstream %q initialize: %w", name, err))
	}
	if err := m.discoverTools(ctx, client); err != nil {
		return fail(fmt.Errorf("upstream %q tools/list: %w", name, err))
	}

	m.logConnected(name, client)
	return nil
}

// logConnected logs a successful connect. Tool names come from the upstream,
// so they are left out when text is redacted.
func (m *UpstreamManager) logConnected(name string, client *UpstreamClient) {
	if m.redacting() {
		log.Info().Str("upstream", name).Int("tools", len(client.toolNames)).Msg("upstream connected")
		return
	}
	log.Info().Str("upstream", name).Int("tools", len(client.toolNames)).
		Strs("tools", client.toolNames).Msg("upstream connected")
}

func (m *UpstreamManager) connect(ctx context.Context, name string, cfg UpstreamConfig) (*UpstreamClient, error) {
	switch cfg.Transport {
	case "stdio":
		return m.connectStdio(name, cfg)
	case "sse":
		return m.connectSSE(ctx, name, cfg)
	case "http", "streamable":
		return m.connectStreamable(ctx, name, cfg)
	default:
		return nil, fmt.Errorf("unknown transport %q", cfg.Transport)
	}
}

func (m *UpstreamManager) connectSSE(ctx context.Context, name string, cfg UpstreamConfig) (*UpstreamClient, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("url is required for SSE transport")
	}

	transport := NewSSETransport(cfg.URL, cfg.Headers, cfg.TLSSkipVerify, m.allowPrivateUpstreams)
	transport.SetRedactUpstreamText(m.redacting())

	connectCtx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	if err := transport.Connect(connectCtx); err != nil {
		transport.Close()
		return nil, wrapUpstreamError("SSE connect to", cfg.URL, err)
	}

	return &UpstreamClient{
		name:      name,
		transport: transport,
	}, nil
}

func (m *UpstreamManager) connectStreamable(ctx context.Context, name string, cfg UpstreamConfig) (*UpstreamClient, error) {
	if cfg.URL == "" {
		return nil, fmt.Errorf("url is required for streamable HTTP transport")
	}

	transport := NewStreamableHTTPTransport(cfg.URL, cfg.Headers, cfg.TLSSkipVerify, m.allowPrivateUpstreams)
	transport.SetRedactUpstreamText(m.redacting())

	connectCtx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	if err := transport.Connect(connectCtx); err != nil {
		transport.Close()
		return nil, wrapUpstreamError("streamable HTTP connect to", cfg.URL, err)
	}

	return &UpstreamClient{
		name:      name,
		transport: transport,
	}, nil
}

func (m *UpstreamManager) connectStdio(name string, cfg UpstreamConfig) (*UpstreamClient, error) {
	if len(cfg.Command) == 0 {
		return nil, fmt.Errorf("command is required for stdio transport")
	}

	cmd := exec.Command(cfg.Command[0], cfg.Command[1:]...)

	// Set environment
	cmd.Env = os.Environ()
	for k, v := range cfg.Env {
		cmd.Env = append(cmd.Env, k+"="+v)
	}

	// Stderr goes to our stderr for debugging
	cmd.Stderr = os.Stderr

	stdin, err := cmd.StdinPipe()
	if err != nil {
		return nil, fmt.Errorf("stdin pipe: %w", err)
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return nil, fmt.Errorf("stdout pipe: %w", err)
	}

	if err := cmd.Start(); err != nil {
		return nil, fmt.Errorf("start command %v: %w", cfg.Command, err)
	}

	transport := NewProcessTransport(stdin, stdout, stdin, cmd)

	return &UpstreamClient{
		name:      name,
		transport: transport,
	}, nil
}

// readLoop continuously reads responses from an upstream and dispatches them.
// If the upstream process dies, it attempts to respawn (stdio only).
func (m *UpstreamManager) readLoop(ctx context.Context, client *UpstreamClient) {
	for {
		msg, err := client.transport.ReadMessage(ctx)
		if err != nil {
			if ctx.Err() != nil || client.closed.Load() {
				return // context cancelled or client closed on purpose: clean shutdown
			}
			log.Error().Err(err).Str("upstream", client.name).Msg("upstream read error")

			// Attempt respawn for stdio upstreams
			published := false
			cfg, hasCfg := m.config[client.name]
			if hasCfg && cfg.Transport == "stdio" {
				published = m.reconnect(ctx, client, "respawning upstream", "upstream respawned successfully", "respawn",
					func() (*UpstreamClient, error) { return m.connectStdio(client.name, cfg) })
			} else if hasCfg && (cfg.Transport == "sse" || cfg.Transport == "http" || cfg.Transport == "streamable") {
				// Reconnect SSE/streamable upstreams with backoff
				published = m.reconnect(ctx, client, "reconnecting upstream", "upstream reconnected successfully", "reconnect",
					func() (*UpstreamClient, error) { return m.connect(ctx, client.name, cfg) })
			} else {
				return
			}
			// Published: the replacement's own read loop owns the upstream
			// now. Not published: nothing more to do for a client that was
			// closed; otherwise keep reading what is left of this one.
			if published || client.stopped(ctx) {
				return
			}
			continue
		}

		_, resp, err := ParseMessage(msg)
		if err != nil {
			if m.redacting() {
				log.Debug().Str("upstream", client.name).Msg("unparseable upstream message")
			} else {
				log.Debug().Err(err).Str("upstream", client.name).Msg("unparseable upstream message")
			}
			continue
		}

		if resp == nil {
			// Upstream sent a request/notification to us (e.g., tools/list_changed)
			// For now, log and ignore. Future: propagate to client.
			if m.redacting() {
				log.Debug().Str("upstream", client.name).Msg("upstream notification")
			} else {
				log.Debug().RawJSON("msg", msg).Str("upstream", client.name).Msg("upstream notification")
			}
			continue
		}

		// Dispatch response to waiting caller
		idStr := string(resp.ID)
		if ch, ok := client.pending.LoadAndDelete(idStr); ok {
			ch.(chan *Response) <- resp
		} else {
			if m.redacting() {
				log.Debug().Str("upstream", client.name).Msg("orphan response")
			} else {
				log.Debug().RawJSON("id", resp.ID).Str("upstream", client.name).Msg("orphan response")
			}
		}
	}
}

// reconnect replaces client's dead transport with a new client, trying up to
// five times with a growing backoff, and reports whether a replacement was
// published. It never dials, initializes or publishes once the client was shut
// down or ctx ended: the closed flag is read after every backoff wait,
// immediately before each dial, and again, under the manager's lock,
// immediately before the new client is published. The wait itself ends as soon
// as the client is shut down. A client made after that point is closed, not
// kept.
//
// The replacement has a read loop of its own from the moment it is dialed
// (nothing else reads its transport, and initialization waits for a reply);
// once it is published the calling loop ends and the replacement's loop owns
// the upstream.
func (m *UpstreamManager) reconnect(ctx context.Context, client *UpstreamClient, startMsg, okMsg, failPrefix string, dial func() (*UpstreamClient, error)) bool {
	for attempt := 1; attempt <= 5; attempt++ {
		if client.stopped(ctx) {
			return false
		}
		backoff := time.Duration(attempt) * 2 * time.Second
		log.Info().Str("upstream", client.name).Int("attempt", attempt).
			Dur("backoff", backoff).Msg(startMsg)
		if !client.waitBackoff(ctx, backoff) {
			return false
		}

		newClient, err := dial()
		if err != nil {
			log.Error().Err(err).Str("upstream", client.name).Msg(failPrefix + " connect failed")
			continue
		}
		// Track the new client so Close can end it while it initializes, and
		// refuse it if the manager or the client was closed in the meantime.
		if !m.track(client, newClient, ctx) {
			newClient.shutdown()
			return false
		}
		go m.readLoop(ctx, newClient)

		// The handshake ends as soon as either client is shut down (Close,
		// RemoveUpstream), not when its HTTP round trip happens to finish.
		hctx, endHandshake := context.WithCancel(ctx)
		stopWatch := make(chan struct{})
		go func() {
			select {
			case <-newClient.done():
			case <-client.done():
			case <-stopWatch:
			}
			endHandshake()
		}()
		finish := func() { close(stopWatch); endHandshake() }

		if err := m.initializeUpstream(hctx, newClient); err != nil {
			finish()
			if client.stopped(ctx) || newClient.closed.Load() {
				m.untrack(newClient)
				newClient.shutdown()
				return false
			}
			log.Error().Err(err).Str("upstream", client.name).Msg(failPrefix + " initialize failed")
			m.untrack(newClient)
			newClient.shutdown()
			continue
		}
		if err := m.discoverTools(hctx, newClient); err != nil {
			finish()
			if client.stopped(ctx) || newClient.closed.Load() {
				m.untrack(newClient)
				newClient.shutdown()
				return false
			}
			log.Error().Err(err).Str("upstream", client.name).Msg(failPrefix + " discover failed")
			m.untrack(newClient)
			newClient.shutdown()
			continue
		}
		finish()

		m.mu.Lock()
		delete(m.inflight, newClient)
		cur, registered := m.clients[client.name]
		if client.stopped(ctx) || m.closed.Load() || (registered && cur != client) {
			m.mu.Unlock()
			newClient.shutdown()
			return false
		}
		m.clients[client.name] = newClient
		m.mu.Unlock()
		client.shutdown() // the dead transport

		log.Info().Str("upstream", client.name).Int("tools", len(newClient.toolNames)).
			Msg(okMsg)
		return true
	}
	return false
}

// sendRequest sends a JSON-RPC request to an upstream and waits for the response.
func (m *UpstreamManager) sendRequest(ctx context.Context, client *UpstreamClient, method string, params json.RawMessage, timeout time.Duration) (*Response, error) {
	id := m.idCounter.Add(1)
	idJSON, _ := json.Marshal(id)

	req := Request{
		JSONRPC: "2.0",
		ID:      idJSON,
		Method:  method,
		Params:  params,
	}

	data, err := json.Marshal(req)
	if err != nil {
		return nil, fmt.Errorf("marshal request: %w", err)
	}

	// Register response channel
	ch := make(chan *Response, 1)
	client.pending.Store(string(idJSON), ch)
	defer client.pending.Delete(string(idJSON))

	// Send
	if err := client.transport.WriteMessage(ctx, data); err != nil {
		return nil, fmt.Errorf("write to upstream: %w", err)
	}

	// Wait for response
	if timeout == 0 {
		timeout = 30 * time.Second
	}
	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case resp := <-ch:
		return resp, nil
	case <-timer.C:
		return nil, fmt.Errorf("upstream timeout after %v", timeout)
	case <-ctx.Done():
		return nil, ctx.Err()
	}
}

// sendNotification sends a JSON-RPC notification (no ID, no response expected).
func (m *UpstreamManager) sendNotification(ctx context.Context, client *UpstreamClient, method string, params json.RawMessage) error {
	req := Request{
		JSONRPC: "2.0",
		Method:  method,
		Params:  params,
	}
	data, err := json.Marshal(req)
	if err != nil {
		return err
	}
	return client.transport.WriteMessage(ctx, data)
}

func (m *UpstreamManager) initializeUpstream(ctx context.Context, client *UpstreamClient) error {
	params, _ := json.Marshal(map[string]interface{}{
		"protocolVersion": "2024-11-05",
		"capabilities":    map[string]interface{}{},
		"clientInfo": map[string]string{
			"name":    "satgate-mcp-proxy",
			"version": "0.1.0",
		},
	})

	resp, err := m.sendRequest(ctx, client, MethodInitialize, params, 30*time.Second)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		if m.redacting() {
			return fmt.Errorf("initialize error: server returned a JSON-RPC error (code %d)", resp.Error.Code)
		}
		return fmt.Errorf("initialize error: %s", resp.Error.Message)
	}

	// Send initialized notification
	if err := m.sendNotification(ctx, client, MethodInitialized, nil); err != nil {
		return fmt.Errorf("initialized notification: %w", err)
	}

	client.ready = true
	return nil
}

func (m *UpstreamManager) discoverTools(ctx context.Context, client *UpstreamClient) error {
	resp, err := m.sendRequest(ctx, client, MethodToolsList, nil, 30*time.Second)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		if m.redacting() {
			return fmt.Errorf("tools/list error: server returned a JSON-RPC error (code %d)", resp.Error.Code)
		}
		return fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	// Parse tools list result
	var result struct {
		Tools []json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
		if m.redacting() {
			return fmt.Errorf("parse tools/list result: reply could not be parsed")
		}
		return fmt.Errorf("parse tools/list result: %w", err)
	}

	client.tools = result.Tools
	client.toolNames = make([]string, 0, len(result.Tools))

	for _, t := range result.Tools {
		var tool struct {
			Name string `json:"name"`
		}
		if err := json.Unmarshal(t, &tool); err == nil && tool.Name != "" {
			client.toolNames = append(client.toolNames, tool.Name)
		}
	}

	return nil
}

// ResolveUpstream returns the upstream client for a given tool name.
func (m *UpstreamManager) ResolveUpstream(toolName string) (*UpstreamClient, error) {
	// Check routing rules first
	for _, rule := range m.routing {
		for _, pattern := range rule.Tools {
			if matchToolPattern(pattern, toolName) {
				if client, ok := m.clients[rule.Upstream]; ok {
					return client, nil
				}
			}
		}
	}

	// Auto-resolve: find the upstream that discovered this tool
	m.mu.Lock()
	defer m.mu.Unlock()
	for _, client := range m.clients {
		for _, tn := range client.toolNames {
			if tn == toolName {
				return client, nil
			}
		}
	}

	// Default upstream
	if client, ok := m.clients[m.defaultUp]; ok {
		return client, nil
	}

	return nil, fmt.Errorf("no upstream found for tool %q", toolName)
}

// AllTools returns the aggregated tools list from all upstreams.
func (m *UpstreamManager) AllTools() []json.RawMessage {
	var all []json.RawMessage
	for _, client := range m.clients {
		all = append(all, client.tools...)
	}
	return all
}

// ForwardToolCall sends a tools/call to the appropriate upstream.
func (m *UpstreamManager) ForwardToolCall(ctx context.Context, toolName string, params json.RawMessage, timeout time.Duration) (*Response, error) {
	client, err := m.ResolveUpstream(toolName)
	if err != nil {
		return nil, err
	}
	return m.sendRequest(ctx, client, MethodToolsCall, params, timeout)
}

// ForwardRequest sends an arbitrary request to the default upstream.
func (m *UpstreamManager) ForwardRequest(ctx context.Context, method string, params json.RawMessage, timeout time.Duration) (*Response, error) {
	client, ok := m.clients[m.defaultUp]
	if !ok {
		return nil, fmt.Errorf("no default upstream configured")
	}
	return m.sendRequest(ctx, client, method, params, timeout)
}

// AddUpstream connects a new upstream at runtime (live reload).
// If an upstream with the same name already exists, it is replaced.
func (m *UpstreamManager) AddUpstream(ctx context.Context, name string, cfg UpstreamConfig) error {
	if m.closed.Load() {
		return fmt.Errorf("upstream manager is closed")
	}
	client, err := m.connect(ctx, name, cfg)
	if err != nil {
		return fmt.Errorf("connect upstream %q: %w", name, err)
	}

	// Start read loop
	go m.readLoop(ctx, client)

	// Initialize
	if err := m.initializeUpstream(ctx, client); err != nil {
		client.shutdown()
		return fmt.Errorf("initialize upstream %q: %w", name, err)
	}

	// Discover tools
	if err := m.discoverTools(ctx, client); err != nil {
		client.shutdown()
		return fmt.Errorf("discover tools for %q: %w", name, err)
	}

	// Swap into clients map (close old if replacing). A manager closed while
	// this upstream was connecting keeps nothing.
	m.mu.Lock()
	if m.closed.Load() {
		m.mu.Unlock()
		client.shutdown()
		return fmt.Errorf("upstream manager is closed")
	}
	if old, exists := m.clients[name]; exists {
		old.shutdown()
	}
	m.clients[name] = client
	m.config[name] = cfg
	m.mu.Unlock()

	m.logAdded(name, client)
	return nil
}

// logAdded logs a live-added upstream; tool names are upstream text and are
// left out when text is redacted.
func (m *UpstreamManager) logAdded(name string, client *UpstreamClient) {
	if m.redacting() {
		log.Info().Str("upstream", name).Int("tools", len(client.toolNames)).Msg("upstream added (live)")
		return
	}
	log.Info().Str("upstream", name).Int("tools", len(client.toolNames)).
		Strs("tools", client.toolNames).Msg("upstream added (live)")
}

// RemoveUpstream disconnects and removes an upstream at runtime.
func (m *UpstreamManager) RemoveUpstream(name string) error {
	m.mu.Lock()
	client, exists := m.clients[name]
	if !exists {
		m.mu.Unlock()
		return fmt.Errorf("upstream %q not found", name)
	}
	delete(m.clients, name)
	delete(m.config, name)
	m.mu.Unlock()

	if err := client.shutdown(); err != nil {
		log.Warn().Err(err).Str("upstream", name).Msg("error closing removed upstream")
	}

	log.Info().Str("upstream", name).Msg("upstream removed (live)")
	return nil
}

// UpstreamNames returns the names of all connected upstreams.
func (m *UpstreamManager) UpstreamNames() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	names := make([]string, 0, len(m.clients))
	for name := range m.clients {
		names = append(names, name)
	}
	return names
}

// Close shuts down all upstream connections. After it returns the manager
// dials nothing more: read loops waiting out a reconnect backoff wake and end,
// and a connection a loop was in the middle of making is closed.
func (m *UpstreamManager) Close() error {
	m.closed.Store(true)
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for name, client := range m.clients {
		if err := client.shutdown(); err != nil {
			log.Error().Err(err).Str("upstream", name).Msg("close upstream")
			if firstErr == nil {
				firstErr = err
			}
		}
	}
	for c := range m.inflight {
		_ = c.shutdown()
		delete(m.inflight, c)
	}
	return firstErr
}

// matchToolPattern matches a tool name against a pattern (supports trailing "*").
func matchToolPattern(pattern, toolName string) bool {
	if pattern == "*" {
		return true
	}
	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(toolName, prefix)
	}
	return pattern == toolName
}
