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
	config    map[string]UpstreamConfig
	routing   []RoutingRule
	defaultUp string

	mu        sync.Mutex
	clients   map[string]*UpstreamClient
	idCounter atomic.Int64
}

// UpstreamClient wraps a transport to an upstream MCP server
// with request/response correlation.
type UpstreamClient struct {
	name      string
	transport Transport
	pending   sync.Map // id (string) -> chan *Response
	tools     []json.RawMessage
	toolNames []string
	ready     bool
}

// NewUpstreamManager creates a new upstream manager.
func NewUpstreamManager(config map[string]UpstreamConfig, routing []RoutingRule, defaultUpstream string) *UpstreamManager {
	return &UpstreamManager{
		config:    config,
		routing:   routing,
		defaultUp: defaultUpstream,
		clients:   make(map[string]*UpstreamClient),
	}
}

// Start launches all upstream connections and discovers tools.
func (m *UpstreamManager) Start(ctx context.Context) error {
	for name, cfg := range m.config {
		client, err := m.connect(ctx, name, cfg)
		if err != nil {
			return fmt.Errorf("upstream %q: %w", name, err)
		}
		m.clients[name] = client

		// Start reading responses in background
		go m.readLoop(ctx, client)

		// Initialize MCP session
		if err := m.initializeUpstream(ctx, client); err != nil {
			return fmt.Errorf("upstream %q initialize: %w", name, err)
		}

		// Discover tools
		if err := m.discoverTools(ctx, client); err != nil {
			return fmt.Errorf("upstream %q tools/list: %w", name, err)
		}

		log.Info().Str("upstream", name).Int("tools", len(client.toolNames)).
			Strs("tools", client.toolNames).Msg("upstream connected")
	}
	return nil
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

	transport := NewSSETransport(cfg.URL, cfg.Headers, cfg.TLSSkipVerify)

	connectCtx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	if err := transport.Connect(connectCtx); err != nil {
		transport.Close()
		return nil, fmt.Errorf("SSE connect to %s: %w", cfg.URL, err)
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

	transport := NewStreamableHTTPTransport(cfg.URL, cfg.Headers, cfg.TLSSkipVerify)

	connectCtx := ctx
	if cfg.Timeout > 0 {
		var cancel context.CancelFunc
		connectCtx, cancel = context.WithTimeout(ctx, cfg.Timeout)
		defer cancel()
	}

	if err := transport.Connect(connectCtx); err != nil {
		transport.Close()
		return nil, fmt.Errorf("streamable HTTP connect to %s: %w", cfg.URL, err)
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
			if ctx.Err() != nil {
				return // context cancelled, clean shutdown
			}
			log.Error().Err(err).Str("upstream", client.name).Msg("upstream read error")

			// Attempt respawn for stdio upstreams
			cfg, hasCfg := m.config[client.name]
			if hasCfg && cfg.Transport == "stdio" {
				for attempt := 1; attempt <= 5; attempt++ {
					if ctx.Err() != nil {
						return
					}
					backoff := time.Duration(attempt) * 2 * time.Second
					log.Info().Str("upstream", client.name).Int("attempt", attempt).
						Dur("backoff", backoff).Msg("respawning upstream")
					time.Sleep(backoff)

					newClient, err := m.connectStdio(client.name, cfg)
					if err != nil {
						log.Error().Err(err).Str("upstream", client.name).Msg("respawn connect failed")
						continue
					}

					// Re-initialize and discover tools
					if err := m.initializeUpstream(ctx, newClient); err != nil {
						log.Error().Err(err).Str("upstream", client.name).Msg("respawn initialize failed")
						continue
					}
					if err := m.discoverTools(ctx, newClient); err != nil {
						log.Error().Err(err).Str("upstream", client.name).Msg("respawn discover failed")
						continue
					}

					// Swap client in place
					m.mu.Lock()
					client.transport = newClient.transport
					client.tools = newClient.tools
					client.toolNames = newClient.toolNames
					client.ready = true
					m.mu.Unlock()

					log.Info().Str("upstream", client.name).Int("tools", len(newClient.toolNames)).
						Msg("upstream respawned successfully")
					break
				}
			} else {
				return
			}
			continue
		}

		_, resp, err := ParseMessage(msg)
		if err != nil {
			log.Debug().Err(err).Str("upstream", client.name).Msg("unparseable upstream message")
			continue
		}

		if resp == nil {
			// Upstream sent a request/notification to us (e.g., tools/list_changed)
			// For now, log and ignore. Future: propagate to client.
			log.Debug().RawJSON("msg", msg).Str("upstream", client.name).Msg("upstream notification")
			continue
		}

		// Dispatch response to waiting caller
		idStr := string(resp.ID)
		if ch, ok := client.pending.LoadAndDelete(idStr); ok {
			ch.(chan *Response) <- resp
		} else {
			log.Debug().RawJSON("id", resp.ID).Str("upstream", client.name).Msg("orphan response")
		}
	}
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

	resp, err := m.sendRequest(ctx, client, MethodInitialize, params, 10*time.Second)
	if err != nil {
		return err
	}
	if resp.Error != nil {
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
	resp, err := m.sendRequest(ctx, client, MethodToolsList, nil, 10*time.Second)
	if err != nil {
		return err
	}
	if resp.Error != nil {
		return fmt.Errorf("tools/list error: %s", resp.Error.Message)
	}

	// Parse tools list result
	var result struct {
		Tools []json.RawMessage `json:"tools"`
	}
	if err := json.Unmarshal(resp.Result, &result); err != nil {
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

// Close shuts down all upstream connections.
func (m *UpstreamManager) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	var firstErr error
	for name, client := range m.clients {
		if err := client.transport.Close(); err != nil {
			log.Error().Err(err).Str("upstream", name).Msg("close upstream")
			if firstErr == nil {
				firstErr = err
			}
		}
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
