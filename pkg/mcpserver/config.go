package mcpserver

import (
	"errors"
	"fmt"
	"net"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/rs/zerolog/log"
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

	// Revocation says where satgate-mcp reads token bans from.
	Revocation RevocationConfig `yaml:"revocation,omitempty"`

	// Logging controls log output.
	Logging LoggingConfig `yaml:"logging"`

	// AllowPrivateUpstreams bypasses SSRF protection on upstream connections.
	// Only enable for local development or when upstreams are on private networks.
	AllowPrivateUpstreams bool `yaml:"allowPrivateUpstreams,omitempty"`

	// authModeUnset records that the loaded file did not set auth.mode
	// (LoadConfig). A network transport then refuses to start, and an
	// SSEServer handler refuses every request (see networkAuthError).
	authModeUnset bool
}

// Listener defaults for the SSE / Streamable HTTP transport.
const (
	// DefaultListenHost is the interface a network transport listens on when
	// server.host is not set: loopback only.
	DefaultListenHost = "127.0.0.1"
	// DefaultListenPort is the port used when server.port is not set.
	DefaultListenPort = 9100
)

// IsNetworkTransport reports whether a server.transport value serves clients
// over the network (SSE and Streamable HTTP) rather than over stdio.
func IsNetworkTransport(transport string) bool {
	return transport == "sse" || transport == "http"
}

// ListenAddr is the address a network transport listens on: server.host
// (default 127.0.0.1) and server.port (default 9100).
func (c *Config) ListenAddr() string {
	host := strings.TrimSuffix(strings.TrimPrefix(strings.TrimSpace(c.Server.Host), "["), "]")
	if host == "" {
		host = DefaultListenHost
	}
	port := c.Server.Port
	if port == 0 {
		port = DefaultListenPort
	}
	return net.JoinHostPort(host, strconv.Itoa(port))
}

// errAuthModeRequired is the refusal for a network transport (the listener
// or an embedded handler) whose configuration does not say how clients
// authenticate.
func errAuthModeRequired(transport string) error {
	where := ""
	if IsNetworkTransport(transport) {
		where = fmt.Sprintf(" (server.transport %q)", transport)
	}
	return fmt.Errorf("auth.mode is required to serve over SSE / Streamable HTTP%s: set auth.mode to \"header\" (macaroon tokens) or \"config\" (one static token in auth.token); to serve without authentication, write auth.mode: none", where)
}

// errAuthTokenRequired is the refusal of auth.mode "config" whose auth.token
// is empty, or only whitespace. Such a value is no token: it is never
// compared with what a client presents.
func errAuthTokenRequired() error {
	return fmt.Errorf("auth.token is required for auth.mode \"config\" and must not be empty or only whitespace: set auth.token to the static token clients send (Authorization: Bearer <token>), or set auth.mode to \"header\" (macaroon tokens)")
}

// authTokenMissing reports that auth.mode is "config" and auth.token is
// empty after trimming Unicode whitespace. Other modes do not read auth.token.
func (c *Config) authTokenMissing() bool {
	return c.Auth.Mode == "config" && blankToken(c.Auth.Token)
}

// blankToken reports whether a token, configured or presented, is empty
// after trimming Unicode whitespace: it counts as no token.
func blankToken(token string) bool {
	return strings.TrimSpace(token) == ""
}

// noneModeWarning is logged once per network server when auth.mode is none.
const noneModeWarning = "auth.mode is none: any client that can reach this address can list and call tools without a token"

// authModeMissing reports that no auth mode was written: Auth.Mode is empty,
// or the configuration came from a file without auth.mode (LoadConfig fills
// in "none" for stdio) and code has not set another mode since.
func (c *Config) authModeMissing() bool {
	return c.Auth.Mode == "" || (c.authModeUnset && c.Auth.Mode == "none")
}

// networkAuthError is the refusal shared by SSEServer.ListenAndServe and the
// handler (SSEServer.ServeHTTP): non-nil when the configuration does not set
// auth.mode, or sets auth.mode "config" with no auth.token (empty or only
// whitespace). It logs nothing.
func (c *Config) networkAuthError() error {
	if c == nil {
		return fmt.Errorf("no configuration: %w", errAuthModeRequired(""))
	}
	if c.authModeMissing() {
		return errAuthModeRequired(c.Server.Transport)
	}
	if c.authTokenMissing() {
		return errAuthTokenRequired()
	}
	return nil
}

// checkNetworkAuth is the startup check of a network transport: it refuses a
// configuration that does not set auth.mode, and logs a warning for an
// explicit auth.mode none.
func (c *Config) checkNetworkAuth(addr string) error {
	if err := c.networkAuthError(); err != nil {
		return err
	}
	if c.Auth.Mode == "none" {
		log.Warn().Str("addr", addr).Msg(noneModeWarning)
	}
	return nil
}

// ServerConfig configures the client-facing MCP server.
type ServerConfig struct {
	// Transport: "stdio" (default), "sse" or "http" (SSE and Streamable HTTP
	// on one listener).
	Transport string `yaml:"transport"`

	// Host is the interface the SSE / Streamable HTTP listener binds to (an
	// IP address or host name, without a port). Empty means 127.0.0.1, so a
	// listener is reachable from this machine only until a host is set
	// (0.0.0.0 or :: for every interface). Ignored for stdio.
	Host string `yaml:"host,omitempty"`

	// Port for SSE/HTTP transport (ignored for stdio). Empty means 9100.
	Port int `yaml:"port,omitempty"`

	// Name reported in initialize response.
	Name string `yaml:"name"`

	// Version reported in initialize response.
	Version string `yaml:"version"`

	// AllowQueryToken accepts the caller's token from a "token" URL query
	// parameter (?token=...) on the SSE and Streamable HTTP endpoints, for
	// clients that cannot send an Authorization header. Off by default: a
	// request that carries ?token= is refused with 401, and the token must be
	// sent as "Authorization: Bearer <token>". A token in a URL is kept in
	// access logs, browser history and Referer headers; turn this on only
	// for a client that has no other way to authenticate.
	AllowQueryToken bool `yaml:"allowQueryToken,omitempty"`
}

// AuthConfig configures token verification for incoming requests.
type AuthConfig struct {
	// Mode: "none" (no authentication), "config" (one static bearer token,
	// Token) or "header" (per-request macaroon, RootKey). A file loaded with
	// LoadConfig that leaves it out gets "none" for stdio, and is refused
	// for the SSE / Streamable HTTP transports: write "none" to serve them
	// without authentication.
	Mode string `yaml:"mode"`

	// Token is the static bearer token (for mode=config).
	Token string `yaml:"token,omitempty"`

	// RootKey for macaroon verification (for mode=header).
	RootKey string `yaml:"rootKey,omitempty"`

	// AutoMintRoot: if true (and mode=header), auto-mint a root macaroon on startup.
	// The raw root token is retained programmatically via Proxy.RootToken but is
	// never printed to stderr/logs; only TOKEN_ID is emitted for operator wiring.
	AutoMintRoot bool `yaml:"autoMintRoot,omitempty"`

	// RootToken: if set (with autoMintRoot), reuse this token instead of minting a
	// new one each startup. Keeps the token ID stable across proxy restarts so
	// clients don't need reconfiguration and Redis budgets survive.
	RootToken string `yaml:"rootToken,omitempty"`
}

// UpstreamConfig defines an upstream MCP server.
type UpstreamConfig struct {
	// Transport: "stdio", "sse", "http", or "streamable"
	//   - "stdio": subprocess (command required)
	//   - "sse": legacy SSE protocol (url required)
	//   - "http" or "streamable": Streamable HTTP protocol (url required, preferred)
	Transport string `yaml:"transport"`

	// Command to start the upstream (for stdio transport).
	Command []string `yaml:"command,omitempty"`

	// URL for HTTP/SSE/Streamable upstream.
	URL string `yaml:"url,omitempty"`

	// Timeout for upstream requests.
	Timeout time.Duration `yaml:"timeout,omitempty"`

	// Env sets environment variables of a stdio upstream's process. A stdio
	// upstream does not inherit the proxy's environment: it starts with the
	// short fixed list in stdioBaseEnv (PATH, HOME, locale, temp dir, time
	// zone and the like, each only if the proxy has it), then the variables
	// named in EnvPassthrough, then Env. A value in Env wins over both.
	Env map[string]string `yaml:"env,omitempty"`

	// EnvPassthrough names variables of the proxy's own environment that a
	// stdio upstream also receives, with the proxy's value (e.g. an API key
	// the upstream reads, or HTTPS_PROXY). A name the proxy does not have is
	// skipped. Nothing else from the proxy's environment reaches the upstream.
	EnvPassthrough []string `yaml:"envPassthrough,omitempty"`

	// ForwardCallerToken is no longer supported, and a configuration or
	// upstream that sets it is refused (see errForwardCallerToken). The
	// proxy removes _meta.token from every request it forwards, including
	// one the caller put there, and passes no SatGate credential to any
	// upstream. An upstream that needs a credential is given its own in
	// Headers (network transports) or Env (stdio).
	ForwardCallerToken bool `yaml:"forwardCallerToken,omitempty"`

	// Headers are additional HTTP headers sent with every request (e.g., Authorization).
	// Only used for HTTP/SSE/Streamable transports.
	Headers map[string]string `yaml:"headers,omitempty"`

	// TLSSkipVerify disables TLS certificate verification.
	// WARNING: Only use for development with self-signed certificates.
	TLSSkipVerify bool `yaml:"tlsSkipVerify,omitempty"`

	// DisplayName is how the service is named to people and in tool
	// descriptions and prefixes ("Robinhood: ..."). Empty means the upstream's
	// key in the configuration.
	DisplayName string `yaml:"displayName,omitempty"`

	// NameKey, when set, is the key the service's tool names are stored under
	// instead of the digest of its address. A host that keeps several
	// services at one address (two logins to one broker) gives each its own.
	// It is a digest or an opaque id, never a URL or a secret.
	NameKey string `yaml:"-"`

	// ExtraSecrets are credential values of this service the host knows and
	// the configuration does not carry (an OAuth refresh token). A tool whose
	// name or description holds one is not listed.
	ExtraSecrets []string `yaml:"-"`

	// WithholdTools says the host could not read this service's complete
	// credential set (stored sign-in tokens, header values, URL query values),
	// so a tool name or description cannot be checked against it. The service
	// is connected, but none of its tools is listed or named until the host
	// sets this off again.
	WithholdTools bool `yaml:"-"`
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
	// Mode: "observe" (deduct, never block) or "control" (deduct and enforce
	// limits). Left out of a file loaded with LoadConfig, it is "control"
	// when budget.limit is greater than 0 and "observe" otherwise.
	Mode string `yaml:"mode"`
}

// LoggingConfig controls log output.
type LoggingConfig struct {
	// Level: "debug", "info" (default), "warn", "error"
	Level string `yaml:"level"`

	// JSON output format.
	JSON bool `yaml:"json"`
}

// errForwardCallerToken refuses an upstream configured with
// forwardCallerToken. The option passed the caller's whole SatGate token on,
// which the upstream could then use as the caller anywhere the token is
// accepted; it was removed rather than kept that way.
var errForwardCallerToken = errors.New("forwardCallerToken is no longer supported: the caller's SatGate token is never passed to an upstream; give the upstream its own credential in headers or env")

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
		// Kept for stdio; a network transport is refused in validate.
		c.authModeUnset = true
		c.Auth.Mode = "none"
	}
	if c.Budget.Backend == "" {
		c.Budget.Backend = "memory"
	}
	if c.Budget.FailMode == "" {
		c.Budget.FailMode = "closed"
	}
	if c.Enforcement.Mode == "" {
		// A configured budget is enforced unless the file says otherwise.
		if c.Budget.Limit > 0 {
			c.Enforcement.Mode = "control"
		} else {
			c.Enforcement.Mode = "observe"
		}
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
	if IsNetworkTransport(c.Server.Transport) && c.authModeUnset {
		return errAuthModeRequired(c.Server.Transport)
	}
	if c.authTokenMissing() {
		return errAuthTokenRequired()
	}
	if h := strings.TrimSpace(c.Server.Host); h != "" {
		if _, _, err := net.SplitHostPort(h); err == nil {
			return fmt.Errorf("server.host %q must not include a port: set the port in server.port", h)
		}
	}

	if len(c.Upstreams) == 0 {
		return fmt.Errorf("at least one upstream is required")
	}

	for name, u := range c.Upstreams {
		if u.ForwardCallerToken {
			return fmt.Errorf("upstream %q: %w", name, errForwardCallerToken)
		}
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
	case "observe", "control":
		// Canonical mode names
	case "hard", "soft", "shadow":
		// Legacy aliases accepted for backward compatibility
	default:
		return fmt.Errorf("enforcement.mode must be observe or control (got %q)", c.Enforcement.Mode)
	}

	switch c.Budget.FailMode {
	case "closed", "open":
		// ok
	default:
		return fmt.Errorf("budget.failMode must be closed or open (got %q)", c.Budget.FailMode)
	}

	return nil
}

// copyUpstreamConfig returns cfg with its own copy of every map and slice, so
// the copy shares no memory with the caller's value.
func copyUpstreamConfig(cfg UpstreamConfig) UpstreamConfig {
	out := cfg
	out.Command = append([]string(nil), cfg.Command...)
	if cfg.Command == nil {
		out.Command = nil
	}
	out.ExtraSecrets = append([]string(nil), cfg.ExtraSecrets...)
	if cfg.ExtraSecrets == nil {
		out.ExtraSecrets = nil
	}
	out.EnvPassthrough = append([]string(nil), cfg.EnvPassthrough...)
	if cfg.EnvPassthrough == nil {
		out.EnvPassthrough = nil
	}
	out.Env = copyStringMap(cfg.Env)
	out.Headers = copyStringMap(cfg.Headers)
	return out
}

func copyStringMap(m map[string]string) map[string]string {
	if m == nil {
		return nil
	}
	out := make(map[string]string, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

// copyUpstreamConfigs copies a whole config map, deeply.
func copyUpstreamConfigs(in map[string]UpstreamConfig) map[string]UpstreamConfig {
	out := make(map[string]UpstreamConfig, len(in))
	for k, v := range in {
		out[k] = copyUpstreamConfig(v)
	}
	return out
}

// copyRoutingRules copies routing rules, deeply.
func copyRoutingRules(in []RoutingRule) []RoutingRule {
	if in == nil {
		return nil
	}
	out := make([]RoutingRule, len(in))
	for i, r := range in {
		out[i] = RoutingRule{Tools: append([]string(nil), r.Tools...), Upstream: r.Upstream}
	}
	return out
}
