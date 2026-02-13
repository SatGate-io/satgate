package mcpserver

import (
	"fmt"
	"os"
	"time"

	"gopkg.in/yaml.v3"
)

// Config is the top-level configuration for the MCP proxy.
type Config struct {
	// Server configures the client-facing transport.
	Server ServerConfig `yaml:"server"`

	// Auth configures token verification.
	Auth AuthConfig `yaml:"auth"`

	// Upstreams defines one or more upstream MCP servers.
	Upstreams map[string]UpstreamConfig `yaml:"upstreams"`

	// DefaultUpstream is used when no routing rule matches (or when only one upstream exists).
	DefaultUpstream string `yaml:"defaultUpstream"`

	// Routing maps tool patterns to upstreams (optional).
	Routing []RoutingRule `yaml:"routing,omitempty"`

	// Budget configures budget enforcement.
	Budget BudgetConfig `yaml:"budget"`

	// Tools configures per-tool costs.
	Tools ToolsConfig `yaml:"tools"`

	// Enforcement controls the enforcement mode.
	Enforcement EnforcementConfig `yaml:"enforcement"`

	// Logging controls log output.
	Logging LoggingConfig `yaml:"logging"`
}

// ServerConfig configures the client-facing MCP server.
type ServerConfig struct {
	// Transport: "stdio" (default) or "sse"
	Transport string `yaml:"transport"`

	// Port for SSE/HTTP transport (ignored for stdio).
	Port int `yaml:"port,omitempty"`

	// Name reported in initialize response.
	Name string `yaml:"name"`

	// Version reported in initialize response.
	Version string `yaml:"version"`
}

// AuthConfig configures token verification for incoming requests.
type AuthConfig struct {
	// Mode: "none" (default for local), "config" (token in config), "header" (per-request)
	Mode string `yaml:"mode"`

	// Token is the static bearer token (for mode=config).
	Token string `yaml:"token,omitempty"`

	// RootKey for macaroon verification (for mode=header).
	RootKey string `yaml:"rootKey,omitempty"`

	// AutoMintRoot: if true (and mode=header), auto-mint a root macaroon on startup
	// and print it to stderr. Useful for demos and development.
	AutoMintRoot bool `yaml:"autoMintRoot,omitempty"`
}

// UpstreamConfig defines an upstream MCP server.
type UpstreamConfig struct {
	// Transport: "stdio" or "http"
	Transport string `yaml:"transport"`

	// Command to start the upstream (for stdio transport).
	Command []string `yaml:"command,omitempty"`

	// URL for HTTP/SSE upstream.
	URL string `yaml:"url,omitempty"`

	// Timeout for upstream requests.
	Timeout time.Duration `yaml:"timeout,omitempty"`

	// Env additional environment variables for subprocess.
	Env map[string]string `yaml:"env,omitempty"`
}

// RoutingRule maps tool name patterns to an upstream.
type RoutingRule struct {
	// Tools is a list of tool name patterns (supports trailing "*" wildcards).
	Tools []string `yaml:"tools"`

	// Upstream is the upstream name to route matching tools to.
	Upstream string `yaml:"upstream"`
}

// BudgetConfig configures the budget enforcer.
type BudgetConfig struct {
	// Backend: "memory" (default, OSS) or "redis" (enterprise)
	Backend string `yaml:"backend"`

	// Limit is the total budget in credits (for memory backend).
	Limit int64 `yaml:"limit,omitempty"`

	// FailMode: "closed" (deny on backend failure, default) or "open" (allow + log)
	FailMode string `yaml:"failMode,omitempty"`

	// Redis connection (enterprise only).
	RedisURL string `yaml:"redisUrl,omitempty"`
}

// ToolsConfig configures per-tool costs.
type ToolsConfig struct {
	// DefaultCost applied when no specific tool cost is configured.
	DefaultCost int64 `yaml:"defaultCost"`

	// Costs maps tool name patterns to credit costs.
	Costs map[string]int64 `yaml:"costs,omitempty"`
}

// EnforcementConfig controls the enforcement mode.
type EnforcementConfig struct {
	// Mode: "hard" (deny on exhaustion, default), "soft" (warn only), "shadow" (observe only)
	Mode string `yaml:"mode"`
}

// LoggingConfig controls log output.
type LoggingConfig struct {
	// Level: "debug", "info" (default), "warn", "error"
	Level string `yaml:"level"`

	// JSON output format.
	JSON bool `yaml:"json"`
}

// LoadConfig reads and parses a YAML config file.
func LoadConfig(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read config: %w", err)
	}

	// Expand environment variables
	expanded := os.ExpandEnv(string(data))

	cfg := &Config{}
	if err := yaml.Unmarshal([]byte(expanded), cfg); err != nil {
		return nil, fmt.Errorf("parse config: %w", err)
	}

	// Apply defaults
	cfg.applyDefaults()

	// Validate
	if err := cfg.validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return cfg, nil
}

func (c *Config) applyDefaults() {
	if c.Server.Transport == "" {
		c.Server.Transport = "stdio"
	}
	if c.Server.Name == "" {
		c.Server.Name = "satgate-mcp-proxy"
	}
	if c.Server.Version == "" {
		c.Server.Version = "0.1.0"
	}
	if c.Auth.Mode == "" {
		c.Auth.Mode = "none"
	}
	if c.Budget.Backend == "" {
		c.Budget.Backend = "memory"
	}
	if c.Budget.FailMode == "" {
		c.Budget.FailMode = "closed"
	}
	if c.Enforcement.Mode == "" {
		c.Enforcement.Mode = "hard"
	}
	if c.Logging.Level == "" {
		c.Logging.Level = "info"
	}
	if c.Tools.DefaultCost == 0 {
		c.Tools.DefaultCost = 1
	}

	// If only one upstream and no default set, use it
	if c.DefaultUpstream == "" && len(c.Upstreams) == 1 {
		for name := range c.Upstreams {
			c.DefaultUpstream = name
		}
	}

	// Set default timeouts
	for name, u := range c.Upstreams {
		if u.Timeout == 0 {
			u.Timeout = 30 * time.Second
			c.Upstreams[name] = u
		}
	}
}

func (c *Config) validate() error {
	if len(c.Upstreams) == 0 {
		return fmt.Errorf("at least one upstream is required")
	}

	for name, u := range c.Upstreams {
		switch u.Transport {
		case "stdio":
			if len(u.Command) == 0 {
				return fmt.Errorf("upstream %q: stdio transport requires command", name)
			}
		case "http", "sse":
			return fmt.Errorf("upstream %q: http/sse upstream transport is not yet implemented (use stdio)", name)
		default:
			return fmt.Errorf("upstream %q: unknown transport %q (want stdio, http, or sse)", name, u.Transport)
		}
	}

	if c.DefaultUpstream != "" {
		if _, ok := c.Upstreams[c.DefaultUpstream]; !ok {
			return fmt.Errorf("defaultUpstream %q not found in upstreams", c.DefaultUpstream)
		}
	}

	for i, rule := range c.Routing {
		if _, ok := c.Upstreams[rule.Upstream]; !ok {
			return fmt.Errorf("routing rule %d: upstream %q not found", i, rule.Upstream)
		}
	}

	switch c.Enforcement.Mode {
	case "hard", "soft", "shadow":
		// ok
	default:
		return fmt.Errorf("enforcement.mode must be hard, soft, or shadow (got %q)", c.Enforcement.Mode)
	}

	switch c.Budget.FailMode {
	case "closed", "open":
		// ok
	default:
		return fmt.Errorf("budget.failMode must be closed or open (got %q)", c.Budget.FailMode)
	}

	return nil
}
