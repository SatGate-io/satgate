package config

import (
	"fmt"
	"net"
	"os"
	"regexp"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// HAConfig holds High Availability configuration (enterprise feature).
type HAConfig struct {
	Enabled        bool   `yaml:"enabled,omitempty"`
	Region         string `yaml:"region,omitempty"`
	CoordinatorURL string `yaml:"coordinatorUrl,omitempty"`
}

// Config represents the gateway configuration
type Config struct {
	Version       int                  `yaml:"version"`
	Server        ServerConfig         `yaml:"server"`
	Admin         AdminConfig          `yaml:"admin"`
	Lightning     LightningConfig      `yaml:"lightning"`
	Billing       *BillingConfig       `yaml:"billing,omitempty"`       // Enterprise billing configuration
	Cloud         *CloudConfig         `yaml:"cloud,omitempty"`         // SatGate Cloud multi-tenant platform
	Mint          *MintConfig          `yaml:"mint,omitempty"`          // SatGate Mint trust broker for agent provisioning
	Redis         *RedisConfig         `yaml:"redis,omitempty"`         // Optional Redis for HA
	HA            *HAConfig            `yaml:"ha,omitempty"`            // Multi-region HA coordination
	Postgres      *PostgresConfig      `yaml:"postgres,omitempty"`      // Optional Postgres for persistence
	GitOps        *GitOpsConfig        `yaml:"gitops,omitempty"`        // Optional GitOps signed config verification
	Notifications *NotificationsConfig `yaml:"notifications,omitempty"` // Alert webhooks (Slack, Discord, HTTP, Email)
	Tracing       *TracingConfig       `yaml:"tracing,omitempty"`       // OpenTelemetry distributed tracing
	Upstreams     map[string]Upstream  `yaml:"upstreams"`
	Routes        []Route              `yaml:"routes"`
}

// MintConfig configures the SatGate Mint trust broker
type MintConfig struct {
	// Enabled activates the Mint service
	Enabled bool `yaml:"enabled"`

	// RootKey is the secret key for signing macaroons (hex or base64 encoded)
	// Set via MINT_ROOT_KEY environment variable
	// MUST be a cryptographically random 32+ byte value
	RootKey string `yaml:"rootKey"`

	// DefaultTTL is the default token lifetime (e.g., "1h", "24h")
	DefaultTTL string `yaml:"defaultTtl"`

	// MaxTTL is the maximum allowed token lifetime
	MaxTTL string `yaml:"maxTtl"`

	// PolicyDirectory is the path to agent policy YAML files
	// Policies define which identities get what permissions
	PolicyDirectory string `yaml:"policyDirectory"`

	// IdentityProviders configures the trust sources
	IdentityProviders []MintIdentityProviderConfig `yaml:"identityProviders"`

	// UseGatewayMacaroons makes Mint use the gateway's macaroon format
	// This ensures tokens work with capability/protect mode
	// Default: true (recommended). If omitted, the gateway will treat it as true.
	UseGatewayMacaroons *bool `yaml:"useGatewayMacaroons,omitempty"`
}

// MintIdentityProviderConfig configures an identity provider for the Mint
type MintIdentityProviderConfig struct {
	// Type is the provider type: "kubernetes", "aws", "oidc"
	Type string `yaml:"type"`

	// Name is a unique identifier for this provider instance
	Name string `yaml:"name"`

	// Enabled allows disabling a provider without removing config
	Enabled bool `yaml:"enabled"`

	// FailClosed fails verification if the provider has errors (default: true)
	// Setting to false is INSECURE and should only be used for development
	FailClosed *bool `yaml:"failClosed,omitempty"`

	// Config is provider-specific configuration
	Config map[string]interface{} `yaml:"config"`
}

// CloudConfig configures SatGate Cloud multi-tenant platform features
type CloudConfig struct {
	Enabled bool `yaml:"enabled"` // Enable cloud platform mode

	// Base URL for the cloud platform (e.g., "https://cloud.satgate.io")
	BaseURL string `yaml:"baseURL"`

	// SECURITY: Separate JWT secret for cloud sessions (use env: CLOUD_JWT_SECRET)
	// This MUST be different from the admin JWT_SECRET to minimize blast radius.
	// If not set, will generate a random secret on startup (not recommended for production).
	JWTSecret string `yaml:"jwtSecret"`

	// Stripe configuration
	StripeSecretKey       string `yaml:"stripeSecretKey"`       // Stripe secret key (use env: STRIPE_SECRET_KEY)
	StripeWebhookSecret   string `yaml:"stripeWebhookSecret"`   // Stripe webhook signing secret
	StripePriceStarter    string `yaml:"stripePriceStarter"`    // Stripe Price ID for Starter plan
	StripePricePro        string `yaml:"stripePricePro"`        // Stripe Price ID for Pro plan
	StripePriceEnterprise string `yaml:"stripePriceEnterprise"` // Stripe Price ID for Enterprise plan

	// Resend email configuration
	ResendAPIKey    string `yaml:"resendAPIKey"`    // Resend API key (use env: RESEND_API_KEY)
	ResendFromEmail string `yaml:"resendFromEmail"` // From email address
	ResendFromName  string `yaml:"resendFromName"`  // From display name

	// Lightning (Alby) configuration for subscription payments
	AlbyAccessToken string `yaml:"albyAccessToken"` // Alby API access token (use env: ALBY_ACCESS_TOKEN)

	// Feature flags
	EnableLifecycleWorker bool `yaml:"enableLifecycleWorker"` // Enable lifecycle automation (trials, onboarding)

	// SECURITY: Trusted proxy CIDRs for cloud rate limiting
	// Only requests from these CIDRs will have X-Forwarded-For/X-Real-IP headers trusted
	// Example: ["10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16"] for RFC1918
	// Set via env: CLOUD_TRUSTED_PROXY_CIDRS (comma-separated)
	TrustedProxyCIDRs []string `yaml:"trustedProxyCidrs"`

	// SSRF protection for tenant-defined upstreams (Cloud mode).
	// REQUIRED for SaaS production: allowedHostSuffixes should be set.
	SSRF *CloudSSRFConfig `yaml:"ssrf,omitempty"`
}

// CloudSSRFConfig configures SSRF protection for tenant-defined upstreams in Cloud mode.
type CloudSSRFConfig struct {
	AllowHTTP            bool     `yaml:"allowHTTP"`
	AllowedPorts         []int    `yaml:"allowedPorts"`
	AllowPrivateIPs      bool     `yaml:"allowPrivateIPs"`
	AllowedHostSuffixes  []string `yaml:"allowedHostSuffixes"`
	FailClosedOnDNSError *bool    `yaml:"failClosedOnDNSError"`
}

// BillingConfig configures enterprise billing modes
type BillingConfig struct {
	Enabled     bool   `yaml:"enabled"`
	DefaultMode string `yaml:"defaultMode"` // chargeback, l402, fiat402
	DefaultUnit string `yaml:"defaultUnit"` // sats, USD, credits

	// FailOpenChargeback allows chargeback mode to fail-open on errors.
	// For fiat402/l402, we always fail-closed to prevent bypass.
	// Default: false (fail-closed for all modes)
	FailOpenChargeback bool `yaml:"failOpenChargeback"`

	Chargeback ChargebackBillingConfig `yaml:"chargeback"`
	L402       L402BillingConfig       `yaml:"l402"`
	Fiat402    Fiat402BillingConfig    `yaml:"fiat402"`
	Alerting   AlertingConfig          `yaml:"alerting"` // Webhook notifications
	Export     ExportConfig            `yaml:"export"`   // Finance system export settings
}

// ChargebackBillingConfig configures chargeback/showback mode
type ChargebackBillingConfig struct {
	Export  BillingExportConfig `yaml:"export"`
	Budgets BillingBudgetConfig `yaml:"budgets"`
}

// BillingExportConfig configures usage exports
type BillingExportConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Format   string `yaml:"format"`   // csv, json, sap
	Schedule string `yaml:"schedule"` // cron expression
}

// BillingBudgetConfig configures budget enforcement
type BillingBudgetConfig struct {
	Enabled        bool    `yaml:"enabled"`
	DefaultMonthly float64 `yaml:"defaultMonthly"`
	AlertThreshold float64 `yaml:"alertThreshold"` // 0.0-1.0
}

// L402BillingConfig configures L402 billing display options
type L402BillingConfig struct {
	DisplayUnit  string  `yaml:"displayUnit"`  // For UI display (USD, credits)
	ExchangeRate float64 `yaml:"exchangeRate"` // sats per USD (for conversion)
}

// Fiat402BillingConfig configures fiat402 receipt tokens
type Fiat402BillingConfig struct {
	ReceiptTTL       string `yaml:"receiptTTL"`       // e.g., "5m"
	SigningKeyRef    string `yaml:"signingKeyRef"`    // Reference to KeyManager
	ReplayProtection string `yaml:"replayProtection"` // expiry, jti, both
}

// AlertingConfig configures webhook notifications for billing events
type AlertingConfig struct {
	Enabled  bool            `yaml:"enabled"`
	Webhooks []WebhookConfig `yaml:"webhooks"`
}

// WebhookConfig configures a webhook endpoint
type WebhookConfig struct {
	URL         string            `yaml:"url"`
	Secret      string            `yaml:"secret,omitempty"`      // HMAC signing secret
	Headers     map[string]string `yaml:"headers,omitempty"`     // Custom headers
	AlertTypes  []string          `yaml:"alertTypes,omitempty"`  // Filter: budget.threshold, budget.exceeded, etc.
	TenantIDs   []string          `yaml:"tenantIds,omitempty"`   // Filter by tenant
	MinSeverity string            `yaml:"minSeverity,omitempty"` // info, warning, critical
	Enabled     bool              `yaml:"enabled"`
	RetryCount  int               `yaml:"retryCount,omitempty"`
	TimeoutSecs int               `yaml:"timeoutSecs,omitempty"`
}

// NotificationsConfig configures alert notification channels (Slack, Discord, HTTP, Email)
type NotificationsConfig struct {
	// Enabled activates notifications
	Enabled bool `yaml:"enabled"`

	// Channels defines notification channels
	Channels []NotificationChannelConfig `yaml:"channels"`

	// DedupeWindowSecs prevents duplicate alerts within this window (default: 300)
	DedupeWindowSecs int `yaml:"dedupeWindowSecs,omitempty"`

	// RetryAttempts is max retries for failed notifications (default: 3)
	RetryAttempts int `yaml:"retryAttempts,omitempty"`

	// RetryBackoffSecs is initial backoff duration (default: 1)
	RetryBackoffSecs int `yaml:"retryBackoffSecs,omitempty"`

	// RateLimitPerMinute limits notifications per channel per minute (default: 60)
	RateLimitPerMinute int `yaml:"rateLimitPerMinute,omitempty"`
}

// NotificationChannelConfig defines a notification channel
type NotificationChannelConfig struct {
	// Name is a unique identifier for this channel
	Name string `yaml:"name"`

	// Type is "slack", "discord", "http", or "email"
	Type string `yaml:"type"`

	// Enabled allows disabling a channel without removing config
	Enabled bool `yaml:"enabled"`

	// URL is the webhook URL (for slack, discord, http)
	URL string `yaml:"url,omitempty"`

	// Headers are custom HTTP headers (for http type)
	Headers map[string]string `yaml:"headers,omitempty"`

	// SMTP configuration (for email type)
	SMTP *SMTPConfig `yaml:"smtp,omitempty"`

	// AlertTypes filters which alert types this channel receives
	// Empty = all types
	// Types: budget_warning, budget_exceeded, quota_warning, quota_exceeded,
	//        token_expiring, token_expired, token_revoked, ha_leader_change, ha_node_down
	AlertTypes []string `yaml:"alertTypes,omitempty"`

	// MinSeverity filters alerts by severity (info, warning, critical)
	MinSeverity string `yaml:"minSeverity,omitempty"`

	// Template is a custom Go template for the message (for http type)
	Template string `yaml:"template,omitempty"`
}

// SMTPConfig holds email configuration for notifications
type SMTPConfig struct {
	Host     string   `yaml:"host"`
	Port     int      `yaml:"port"`
	Username string   `yaml:"username"`
	Password string   `yaml:"password"` // Use env: SMTP_PASSWORD
	From     string   `yaml:"from"`
	To       []string `yaml:"to"`
	UseTLS   bool     `yaml:"useTls"`
}

// ExportConfig configures finance system exports
type ExportConfig struct {
	CompanyCode   string `yaml:"companyCode"`
	CurrencyCode  string `yaml:"currencyCode"`
	GLAccountCode string `yaml:"glAccountCode"`
	CostElement   string `yaml:"costElement"`
	ProfitCenter  string `yaml:"profitCenter"`
}

// GitOpsConfig configures signed configuration verification
type GitOpsConfig struct {
	Enabled          bool          `yaml:"enabled"`          // Enable signed config verification
	RequireSignature bool          `yaml:"requireSignature"` // Reject unsigned configs (strict mode)
	TrustedKeysDir   string        `yaml:"trustedKeysDir"`   // Directory containing trusted public keys (.pub files)
	PollInterval     time.Duration `yaml:"pollInterval"`     // Polling interval for config changes (default: 30s)
}

// RedisConfig holds Redis connection settings for HA persistence
type RedisConfig struct {
	Enabled  bool   `yaml:"enabled"`
	Addr     string `yaml:"addr"`     // e.g., "localhost:6379"
	Password string `yaml:"password"` // empty for no auth
	DB       int    `yaml:"db"`       // database number
}

// TracingConfig configures OpenTelemetry distributed tracing
type TracingConfig struct {
	// Enabled activates tracing
	Enabled bool `yaml:"enabled"`

	// ServiceName is the service name for traces (default: "satgate-gateway")
	ServiceName string `yaml:"serviceName"`

	// ServiceVersion is the service version
	ServiceVersion string `yaml:"serviceVersion"`

	// Environment is the deployment environment (production, staging, development)
	Environment string `yaml:"environment"`

	// Exporter configures the trace exporter
	Exporter TracingExporterConfig `yaml:"exporter"`

	// Sampling configures trace sampling
	Sampling TracingSamplingConfig `yaml:"sampling"`

	// Propagators is a list of propagation formats (tracecontext, baggage)
	Propagators []string `yaml:"propagators"`
}

// TracingExporterConfig configures the trace exporter
type TracingExporterConfig struct {
	// Type is "otlp", "otlp-http", "stdout", or "none"
	Type string `yaml:"type"`

	// Endpoint is the exporter endpoint
	// For OTLP gRPC: "localhost:4317"
	// For OTLP HTTP: "localhost:4318"
	Endpoint string `yaml:"endpoint"`

	// Headers are additional headers for the exporter
	Headers map[string]string `yaml:"headers"`

	// Insecure disables TLS for the exporter
	Insecure bool `yaml:"insecure"`

	// TimeoutSecs is the export timeout in seconds (default: 10)
	TimeoutSecs int `yaml:"timeoutSecs"`

	// Compression enables gzip compression for exports
	Compression bool `yaml:"compression"`
}

// TracingSamplingConfig configures trace sampling
type TracingSamplingConfig struct {
	// Type is "always", "never", "ratio", or "parent" (default: "parent")
	Type string `yaml:"type"`

	// Ratio is the sampling ratio (0.0 to 1.0) for ratio sampling (default: 1.0)
	Ratio float64 `yaml:"ratio"`
}

// PostgresConfig holds PostgreSQL connection settings for persistent storage
type PostgresConfig struct {
	Enabled        bool   `yaml:"enabled"`
	URL            string `yaml:"url"`            // e.g., "postgres://user:pass@localhost:5432/satgate?sslmode=disable"
	MaxConnections int    `yaml:"maxConnections"` // connection pool size
	MigrationsPath string `yaml:"migrationsPath"` // path to SQL migrations
	AutoMigrate    bool   `yaml:"autoMigrate"`    // run migrations on startup
}

// ServerConfig contains server settings
type ServerConfig struct {
	Listen         string        `yaml:"listen"`
	ReadTimeout    time.Duration `yaml:"readTimeout"`
	WriteTimeout   time.Duration `yaml:"writeTimeout"`
	MaxRequestBody int64         `yaml:"maxRequestBody"`
	TrustedProxies []string      `yaml:"trustedProxies"` // CIDRs allowed to set X-Tenant-ID (e.g., "10.0.0.0/8", "172.16.0.0/12")

	// DEV ONLY: Trust X-Tenant-ID header from any source without checking trustedProxies.
	// WARNING: Never enable in production - allows tenant spoofing!
	// Use for local development/testing only.
	TrustTenantHeaderInDev bool `yaml:"trustTenantHeaderInDev"`

	// Single-tenant default: if set, requests without tenant context are attributed to this tenant
	// This enables TENANT_ISOLATION_REQUIRED=true without breaking clients
	DefaultTenantId string `yaml:"defaultTenantId"`

	// Host allowlist: only accept requests to these hosts (empty = allow all)
	// Prevents misrouting attacks; skips /healthz and /readyz for k8s probes
	AllowedHosts []string `yaml:"allowedHosts"`

	// Tenant isolation controls (enterprise hardening)
	TenantIsolationEnabled  bool `yaml:"tenantIsolationEnabled"`  // Enable tenant context extraction + quota enforcement
	TenantIsolationRequired bool `yaml:"tenantIsolationRequired"` // Reject requests without valid tenant context (403)
	TenantQuotasEnabled     bool `yaml:"tenantQuotasEnabled"`     // Enforce per-tenant quotas
}

// AdminConfig contains admin API settings
type AdminConfig struct {
	Token                string     `yaml:"token"`            // Break-glass admin token
	JWTSecret            string     `yaml:"jwtSecret"`        // DEPRECATED: Use KeyManagement instead
	JWTExpiry            string     `yaml:"jwtExpiry"`        // JWT token expiry (e.g., "24h")
	RefreshExpiry        string     `yaml:"refreshExpiry"`    // Refresh token expiry (e.g., "7d")
	KeyManagement        *KeyConfig `yaml:"keyManagement"`    // Key management configuration
	SeparateListener     string     `yaml:"separateListener"` // Separate listener for admin API (e.g., "127.0.0.1:9090")
	AllowedIPs           []string   `yaml:"allowedIps"`
	CORSAllowedOrigins   []string   `yaml:"corsAllowedOrigins"`   // Explicit origin allowlist (no wildcards in production)
	CORSAllowCredentials bool       `yaml:"corsAllowCredentials"` // Only enable if needed for cookie-based auth
	RateLimitPerMinute   int        `yaml:"rateLimitPerMinute"`   // Admin API rate limit (0 = unlimited)
	RateLimitKeyType     string     `yaml:"rateLimitKeyType"`     // "ip", "token", or "global"
	RateLimitBackend     string     `yaml:"rateLimitBackend"`     // "memory" or "redis" (redis for multi-instance)

	// Audit hash chain key (tamper-evident logging)
	// Set via AUDIT_HASH_CHAIN_KEY env var (base64, 32+ bytes recommended)
	AuditHashChainKey        string `yaml:"auditHashChainKey"`
	RequireAuditHashChainKey bool   `yaml:"requireAuditHashChainKey"` // Fail startup if key not set

	// P0 SECURITY: Development-only features (MUST be false in production)
	// EnableDevLogin allows the placeholder /api/v1/auth/login endpoint
	// WARNING: This endpoint issues tokens for ANY email - never enable in production!
	EnableDevLogin bool `yaml:"enableDevLogin"`

	// EnableSwaggerUI enables the /api/docs Swagger UI (disabled by default in production)
	EnableSwaggerUI bool `yaml:"enableSwaggerUI"`

	// Trusted proxy CIDRs for X-Forwarded-For / X-Real-IP headers
	// Only requests from these CIDRs will have forwarded headers honored
	// Empty = trust RemoteAddr only (most secure)
	TrustedProxyCIDRs []string `yaml:"trustedProxyCIDRs"`

	// CapabilityRootKey is the secret key for signing capability macaroons
	// Set via CAPABILITY_ROOT_KEY env var (hex-encoded, 32+ bytes recommended)
	// If not set, falls back to admin.token for backward compatibility
	CapabilityRootKey string `yaml:"capabilityRootKey"`
}

// KeyConfig holds key management configuration
type KeyConfig struct {
	Provider       string            `yaml:"provider"`       // "env", "file", "memory", "aws-kms", "gcp-kms", "vault"
	Algorithm      string            `yaml:"algorithm"`      // "RS256", "RS384", "RS512", "ES256", "ES384"
	KeySize        int               `yaml:"keySize"`        // RSA key size (2048, 4096)
	RotationDays   int               `yaml:"rotationDays"`   // Days before key rotation
	RetainPrevious int               `yaml:"retainPrevious"` // Number of previous keys to retain
	Config         map[string]string `yaml:"config"`         // Provider-specific config (keyFile, vaultPath, etc.)

	// RequireSigningKey fails startup if no signing key is configured (JWT_SIGNING_KEY or keyFile).
	// Prevents accidental use of ephemeral keys in production.
	// Default: false (for backward compatibility), set to true in enterprise deployments.
	RequireSigningKey bool `yaml:"requireSigningKey"`
}

// LightningConfig contains Lightning node settings
type LightningConfig struct {
	Provider string                 `yaml:"provider"` // phoenixd, lnd, cln, alby, lnbits
	Config   map[string]interface{} `yaml:"config"`

	// L402 Security Settings (P0 - CRITICAL)
	// L402RootKey is the secret key for signing L402 macaroons
	// MUST be a cryptographically random 32+ byte value (base64 encoded)
	// Set via L402_ROOT_KEY env var - NEVER derive from provider name
	L402RootKey string `yaml:"l402RootKey"`

	// RequireInvoiceRecord enforces that L402 tokens can only be validated
	// if the invoice exists in the persistent store (fail-closed)
	// Default: true in production, false allows preimage-only validation (INSECURE)
	RequireInvoiceRecord bool `yaml:"requireInvoiceRecord"`

	// VerifyWithNode checks payment status with Lightning node if no local record
	// Only effective when RequireInvoiceRecord=false
	VerifyWithNode bool `yaml:"verifyWithNode"`
}

// Upstream represents a backend API
type Upstream struct {
	URL            string            `yaml:"url"`
	Timeout        time.Duration     `yaml:"timeout"`
	Headers        map[string]string `yaml:"headers"`
	HealthCheck    *HealthCheck      `yaml:"healthCheck"`
	CircuitBreaker *CircuitBreaker   `yaml:"circuitBreaker"`
	TLS            *UpstreamTLS      `yaml:"tls,omitempty"` // mTLS configuration
}

// UpstreamTLS configures TLS/mTLS for upstream connections
type UpstreamTLS struct {
	// Skip certificate verification (NOT recommended for production)
	InsecureSkipVerify bool `yaml:"insecureSkipVerify"`
	// Path to CA certificate file for verifying upstream
	CACertFile string `yaml:"caCertFile"`
	// Path to client certificate file (for mTLS)
	ClientCertFile string `yaml:"clientCertFile"`
	// Path to client private key file (for mTLS)
	ClientKeyFile string `yaml:"clientKeyFile"`
	// Server name for SNI (defaults to hostname from URL)
	ServerName string `yaml:"serverName"`
}

// HealthCheck configuration
type HealthCheck struct {
	Path     string        `yaml:"path"`
	Interval time.Duration `yaml:"interval"`
	Timeout  time.Duration `yaml:"timeout"`
}

// CircuitBreaker configuration
type CircuitBreaker struct {
	MaxFailures     int           `yaml:"maxFailures"`
	ResetTimeout    time.Duration `yaml:"resetTimeout"`
	HalfOpenMaxReqs int           `yaml:"halfOpenMaxReqs"`
}

// MCPConfig configures MCP (Model Context Protocol) request parsing for a route.
// When enabled, the gateway parses JSON-RPC 2.0 request bodies to extract
// tool-level metadata (method, tool name) for logging and telemetry.
//
// Per-tool cost attribution (ToolCosts, DefaultCost) is available in
// SatGate Enterprise.
type MCPConfig struct {
	Enabled     bool  `yaml:"enabled"`
	MaxBodySize int64 `yaml:"maxBodySize"` // max bytes to buffer for parsing (default 1MB)
}

// Route defines how requests are matched and handled
type Route struct {
	Name        string      `yaml:"name"`
	Match       RouteMatch  `yaml:"match"`
	Upstream    string      `yaml:"upstream"`
	Rewrite     string      `yaml:"rewrite,omitempty"`     // Rewrite path before proxying (static)
	StripPrefix bool        `yaml:"stripPrefix,omitempty"` // Strip the matched pathPrefix before proxying
	Policy      RoutePolicy `yaml:"policy"`
	Transform   *Transform  `yaml:"transform,omitempty"`
	RateLimit   *RateLimit  `yaml:"rateLimit,omitempty"`
	MCP         *MCPConfig  `yaml:"mcp,omitempty"` // MCP request parsing for tool-level cost attribution
}

// RouteMatch defines matching criteria
type RouteMatch struct {
	PathPrefix string            `yaml:"pathPrefix"`
	PathExact  string            `yaml:"pathExact"`
	PathRegex  string            `yaml:"pathRegex"`
	Methods    []string          `yaml:"methods"`
	Headers    map[string]string `yaml:"headers"` // header -> value (supports regex with prefix "~")

	// Compiled regex (populated during validation)
	compiledPathRegex *regexp.Regexp
}

// RoutePolicy defines access control
type RoutePolicy struct {
	// Kind specifies the access control policy type:
	//
	// NO VERIFICATION:
	//   - "public": No authentication required
	//   - "deny": Block all requests
	//
	// LAYER 0 - DEFAULT PROTECTION (verify only, no economics):
	//   - "protected" / "protect" / "capability": Macaroon verification
	//
	// LAYER 1 - ECONOMIC POLICIES:
	//   - "observe" / "audit" / "chargeback" / "policy_to_proof": verify → allow → evidence
	//   - "control" / "budget" / "fiat402" / "enforce": verify → enforce budget → allow
	//   - "charge" / "monetize" / "l402": verify → payment proof → allow
	//
	// LEGACY:
	//   - "pay": Use detailed Pay policy below
	Kind      string `yaml:"kind"`
	PriceSats int64  `yaml:"priceSats,omitempty"`
	Scope     string `yaml:"scope,omitempty"`
	Tier      string `yaml:"tier,omitempty"`

	// EvidencePack configures policy-to-proof receipt preservation.
	// When enabled, every policy decision generates a tamper-evident receipt.
	EvidencePack *EvidencePackConfig `yaml:"evidence_pack,omitempty"`

	// CostCredits specifies the cost in credits for this route (budget enforcement).
	// Default is 1 credit per request if not specified.
	// Use this to differentiate expensive endpoints (e.g., GPT-4 = 10) from cheap ones (embeddings = 1).
	// Only applies to control/fiat402 policy mode.
	CostCredits int64 `yaml:"costCredits,omitempty"`

	// Pay policy (when Kind == "pay")
	Pay *PayPolicy `yaml:"pay,omitempty"`
}

// NormalizePolicyKind converts strategic policy kind aliases to canonical forms.
// This allows users to use either legacy or strategic names in configuration.
//
// LAYER 0: Default Protection (always-on cryptographic verification)
//   - "protected" / "protect" → "capability" (verify only, no economics)
//
// LAYER 1: Economic Policies (per-route)
//   - "observe" / "audit"    → "chargeback" (verify → allow → meter/log)
//   - "control" / "budget"   → "fiat402"    (verify → enforce budget → allow)
//   - "charge"  / "monetize" → "l402"       (verify → payment proof → allow)
//
// "Default Protection. Choose your economic policy."
func NormalizePolicyKind(kind string) string {
	switch strings.ToLower(kind) {
	// Economic Policy: Observe (passive metering)
	case "observe", "audit", "policy_to_proof":
		return "chargeback"
	// Economic Policy: Control (budget enforcement)
	case "control", "budget", "enforce":
		return "fiat402"
	// Economic Policy: Charge (Lightning payments)
	case "charge", "monetize":
		return "l402"
	// Layer 0: Protected (verify only, no economics)
	case "protected", "protect":
		return "capability"
	default:
		return kind
	}
}

// NormalizePayMode converts strategic pay mode aliases to canonical forms.
// Used within pay policy configuration.
func NormalizePayMode(mode string) string {
	switch mode {
	case "observe", "audit":
		return "chargeback"
	case "control", "budget":
		return "fiat402"
	case "charge", "monetize":
		return "l402"
	default:
		return mode
	}
}

// IsBillingMode returns true if the policy kind requires billing manager handling.
func IsBillingMode(kind string) bool {
	normalized := NormalizePolicyKind(kind)
	return normalized == "chargeback" || normalized == "fiat402" || normalized == "l402"
}

// EvidencePackConfig configures policy-to-proof receipt preservation.
type EvidencePackConfig struct {
	Required               bool     `yaml:"required"`
	ReceiptID              string   `yaml:"receipt_id,omitempty"` // "generated_per_decision", "tenant_scoped"
	IncludePaymentContext  bool     `yaml:"include_payment_context"`
	IncludePaidRailContext bool     `yaml:"include_paid_rail_context"`
	RequiredFields         []string `yaml:"required_fields,omitempty"`
}

// PayPolicy configures billing for a route
type PayPolicy struct {
	Mode             string  `yaml:"mode"`             // chargeback, l402, fiat402
	Price            float64 `yaml:"price"`            // Price in specified unit
	Unit             string  `yaml:"unit"`             // sats, USD, credits
	Scope            string  `yaml:"scope,omitempty"`  // Required scope for access
	CostCenterHeader string  `yaml:"costCenterHeader"` // Header to extract cost center (e.g., "X-Cost-Center")
	EnforceBudget    bool    `yaml:"enforceBudget"`    // Enforce tenant/cost-center budgets
	CreditsPerSat    int64   `yaml:"creditsPerSat"`    // L402 → Credits conversion rate (default: 1 sat = 1 credit)
}

// Transform defines request/response transformations
type Transform struct {
	StripPrefix string            `yaml:"stripPrefix"`
	AddHeaders  map[string]string `yaml:"addHeaders"`
}

// RateLimit defines rate limiting rules
type RateLimit struct {
	RequestsPerMinute int    `yaml:"requestsPerMinute"`
	BurstSize         int    `yaml:"burstSize"`
	Key               string `yaml:"key"` // ip, token, header:X-Custom
}

// Load reads configuration from a YAML file
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("failed to read config file: %w", err)
	}

	return ParseYAML(data)
}

// ParseYAML parses configuration from YAML bytes.
// This is used by the agent mode to apply config pushed from the control plane.
func ParseYAML(data []byte) (*Config, error) {
	// Expand environment variables
	data = []byte(os.ExpandEnv(string(data)))

	var cfg Config
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return nil, fmt.Errorf("failed to parse config: %w", err)
	}

	// Set defaults
	if cfg.Server.Listen == "" {
		cfg.Server.Listen = ":8080"
	}
	if cfg.Server.ReadTimeout == 0 {
		cfg.Server.ReadTimeout = 30 * time.Second
	}
	if cfg.Server.WriteTimeout == 0 {
		cfg.Server.WriteTimeout = 30 * time.Second
	}

	// Validate configuration
	if err := cfg.Validate(); err != nil {
		return nil, fmt.Errorf("invalid configuration: %w", err)
	}

	return &cfg, nil
}

// Validate checks the configuration for errors
func (c *Config) Validate() error {
	if len(c.Routes) == 0 {
		return fmt.Errorf("no routes defined")
	}

	// Validate each route
	for i := range c.Routes {
		route := &c.Routes[i]
		if route.Name == "" {
			return fmt.Errorf("route %d has no name", i)
		}
		if route.Match.PathPrefix == "" && route.Match.PathExact == "" && route.Match.PathRegex == "" {
			return fmt.Errorf("route %s has no path matcher", route.Name)
		}
		if route.Policy.Kind == "" {
			return fmt.Errorf("route %s has no policy kind", route.Name)
		}

	// Normalize policy.kind aliases to canonical forms
		// This allows users to use strategic names (audit, budget, monetize, protect, policy_to_proof)
		// which get converted to runtime names (chargeback, fiat402, l402, capability)
		route.Policy.Kind = NormalizePolicyKind(route.Policy.Kind)

		// Auto-enable EvidencePack for policy_to_proof or when explicitly requested
		if strings.ToLower(route.Policy.Kind) == "chargeback" && route.Policy.EvidencePack == nil && strings.Contains(strings.ToLower(os.Getenv("SATGATE_ENABLE_V2_AUDIT")), "true") {
			route.Policy.EvidencePack = &EvidencePackConfig{Required: true, ReceiptID: "generated_per_decision"}
		}

		// For billing modes, synthesize Pay policy if not present
		switch route.Policy.Kind {
		case "l402":
			// l402 requires a price - either via priceSats or pay.price
			if route.Policy.PriceSats <= 0 && (route.Policy.Pay == nil || route.Policy.Pay.Price <= 0) {
				return fmt.Errorf("route %s has l402 policy but no price", route.Name)
			}
			// Synthesize Pay policy from legacy priceSats if needed
			if route.Policy.Pay == nil && route.Policy.PriceSats > 0 {
				route.Policy.Pay = &PayPolicy{
					Mode:  "l402",
					Price: float64(route.Policy.PriceSats),
					Unit:  "sats",
				}
			}
		case "fiat402":
			// fiat402 requires pay policy with price
			if route.Policy.Pay == nil {
				return fmt.Errorf("route %s has fiat402 policy but no 'pay' configuration", route.Name)
			}
			if route.Policy.Pay.Mode == "" {
				route.Policy.Pay.Mode = "fiat402"
			}
		case "chargeback":
			// chargeback (audit) mode - synthesize pay policy for metering
			if route.Policy.Pay == nil {
				route.Policy.Pay = &PayPolicy{
					Mode:  "chargeback",
					Price: 0,
					Unit:  "USD",
				}
			}
			if route.Policy.Pay.Mode == "" {
				route.Policy.Pay.Mode = "chargeback"
			}
		}

		// Also normalize pay.mode if present
		if route.Policy.Pay != nil && route.Policy.Pay.Mode != "" {
			route.Policy.Pay.Mode = NormalizePayMode(route.Policy.Pay.Mode)
		}

		// Legacy check - keep for backward compatibility
		if route.Policy.Kind == "l402" && route.Policy.PriceSats <= 0 && route.Policy.Pay == nil {
			return fmt.Errorf("route %s has l402 policy but no price", route.Name)
		}
		if route.Upstream != "" {
			if _, ok := c.Upstreams[route.Upstream]; !ok {
				return fmt.Errorf("route %s references unknown upstream %s", route.Name, route.Upstream)
			}
		}

		// Compile path regex
		if err := route.CompileRegex(); err != nil {
			return err
		}
	}

	// Validate upstreams
	for name, upstream := range c.Upstreams {
		if err := validateUpstreamURL(name, upstream.URL); err != nil {
			return err
		}
	}

	return nil
}

// validateUpstreamURL validates an upstream URL for security
func validateUpstreamURL(name, urlStr string) error {
	if urlStr == "" {
		return fmt.Errorf("upstream %s has no URL", name)
	}

	// Parse URL
	parsed, err := parseURL(urlStr)
	if err != nil {
		return fmt.Errorf("upstream %s has invalid URL: %w", name, err)
	}

	// Only allow http/https schemes
	scheme := strings.ToLower(parsed.Scheme)
	if scheme != "http" && scheme != "https" {
		return fmt.Errorf("upstream %s has disallowed scheme %q (only http/https allowed)", name, parsed.Scheme)
	}

	// Block credentials in URL (security risk - can leak in logs)
	if parsed.User != nil {
		return fmt.Errorf("upstream %s has credentials in URL (security risk - use headers instead)", name)
	}

	// Block private IP ranges to prevent SSRF attacks
	// This is enforced at config load time; runtime SSRF guard provides additional protection
	host := parsed.Host
	// Strip port if present
	if colonIdx := strings.LastIndex(host, ":"); colonIdx != -1 {
		host = host[:colonIdx]
	}
	if isPrivateIP(host) {
		return fmt.Errorf("upstream %s targets private IP %s (blocked by SSRF policy - use allowlist if intentional)", name, host)
	}

	return nil
}

// isPrivateIP checks if a hostname is a private/internal IP address
func isPrivateIP(hostname string) bool {
	ip := net.ParseIP(hostname)
	if ip == nil {
		return false // Not an IP address, assume OK (DNS will be checked at runtime)
	}

	if ip.IsLoopback() || ip.IsLinkLocalUnicast() || ip.IsLinkLocalMulticast() || ip.IsPrivate() {
		return true
	}

	return false
}

// parseURL is a helper to parse URLs
func parseURL(urlStr string) (*urlParsed, error) {
	// Simple URL parsing - in production use net/url
	parts := strings.SplitN(urlStr, "://", 2)
	if len(parts) != 2 {
		return nil, fmt.Errorf("missing scheme")
	}

	result := &urlParsed{Scheme: parts[0]}

	hostPath := parts[1]

	// Check for credentials (user:pass@host)
	if atIdx := strings.Index(hostPath, "@"); atIdx != -1 {
		userInfo := hostPath[:atIdx]
		result.User = &userInfo
		hostPath = hostPath[atIdx+1:]
	}

	// Extract host
	if slashIdx := strings.Index(hostPath, "/"); slashIdx != -1 {
		result.Host = hostPath[:slashIdx]
	} else {
		result.Host = hostPath
	}

	return result, nil
}

type urlParsed struct {
	Scheme string
	User   *string
	Host   string
}

// GetRoute finds the first matching route for a request
func (c *Config) GetRoute(path string, method string) *Route {
	return c.GetRouteWithHeaders(path, method, nil)
}

// GetRouteWithHeaders finds the first matching route including header checks
func (c *Config) GetRouteWithHeaders(path string, method string, headers map[string]string) *Route {
	for i := range c.Routes {
		route := &c.Routes[i]
		if route.MatchesWithHeaders(path, method, headers) {
			return route
		}
	}
	return nil
}

// Matches checks if a request matches this route
func (r *Route) Matches(path string, method string) bool {
	return r.MatchesWithHeaders(path, method, nil)
}

// MatchesWithHeaders checks if a request matches this route including headers
func (r *Route) MatchesWithHeaders(path string, method string, headers map[string]string) bool {
	// Check path prefix
	if r.Match.PathPrefix != "" {
		if !strings.HasPrefix(path, r.Match.PathPrefix) {
			return false
		}
	}

	// Check exact path
	if r.Match.PathExact != "" && path != r.Match.PathExact {
		return false
	}

	// Check path regex
	if r.Match.compiledPathRegex != nil {
		if !r.Match.compiledPathRegex.MatchString(path) {
			return false
		}
	}

	// Check method
	if len(r.Match.Methods) > 0 {
		found := false
		methodUpper := strings.ToUpper(method)
		for _, m := range r.Match.Methods {
			if strings.ToUpper(m) == methodUpper {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	// Check headers
	if len(r.Match.Headers) > 0 && headers != nil {
		for key, pattern := range r.Match.Headers {
			headerValue, exists := headers[key]
			if !exists {
				// Also check case-insensitive
				for k, v := range headers {
					if strings.EqualFold(k, key) {
						headerValue = v
						exists = true
						break
					}
				}
			}
			if !exists {
				return false
			}

			// Check if pattern is regex (prefixed with ~)
			if strings.HasPrefix(pattern, "~") {
				re, err := regexp.Compile(pattern[1:])
				if err != nil {
					return false
				}
				if !re.MatchString(headerValue) {
					return false
				}
			} else {
				// Exact match
				if headerValue != pattern {
					return false
				}
			}
		}
	}

	return true
}

// CompileRegex compiles the path regex if specified
func (r *Route) CompileRegex() error {
	if r.Match.PathRegex != "" {
		re, err := regexp.Compile(r.Match.PathRegex)
		if err != nil {
			return fmt.Errorf("invalid path regex for route %s: %w", r.Name, err)
		}
		r.Match.compiledPathRegex = re
	}
	return nil
}
