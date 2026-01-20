// Package cloud provides stub implementations for OSS builds.
// In enterprise builds, this package contains the full control plane.
package cloud

import (
	"context"
	"net/http"

	"github.com/satgate-io/satgate/internal/config"
)

// GatewayConfig represents a tenant's gateway configuration.
// In enterprise, this comes from the control plane database.
// In OSS, this is always nil (single-tenant mode).
type GatewayConfig struct {
	Routes    []RouteConfig
	Upstreams map[string]UpstreamConfig
}

// RouteConfig represents a route configuration.
type RouteConfig struct {
	Name     string
	Path     string
	Upstream string
	Policy   PolicyConfig
}

// UpstreamConfig represents an upstream configuration.
type UpstreamConfig struct {
	URL string
}

// PolicyConfig represents a route's policy configuration.
type PolicyConfig struct {
	Kind      string
	Scope     string
	PriceSats int64
}

// GetRouteWithHeaders returns a route matching the request path and headers.
// Returns nil in OSS (use static config).
func (c *GatewayConfig) GetRouteWithHeaders(path string, headers http.Header) *config.Route {
	return nil
}

// TenantRoutingConfig returns nil in OSS builds (no multi-tenancy).
// In enterprise, this returns the resolved tenant's gateway config.
func TenantRoutingConfig(ctx context.Context) *GatewayConfig {
	return nil
}

// ResolvedTenant represents a resolved tenant context.
// Used by enterprise for tenant isolation and config routing.
type ResolvedTenant struct {
	ID      string
	Slug    string
	Version int
}

// TenantContext returns nil in OSS builds.
// In enterprise, this returns the resolved tenant from context.
func TenantContext(ctx context.Context) *ResolvedTenant {
	return nil
}
