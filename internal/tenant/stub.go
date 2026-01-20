// Package tenant provides stub implementations for OSS builds.
// In enterprise builds, this package contains full tenant isolation.
package tenant

import "context"

// Tenant represents a tenant in the system.
type Tenant struct {
	ID   string
	Slug string
	Plan string
}

// IsolationEnforcer enforces tenant isolation policies.
// In OSS, this is a no-op stub. In enterprise, it validates
// tenant boundaries and prevents cross-tenant access.
type IsolationEnforcer struct{}

// NewIsolationEnforcer creates a new isolation enforcer.
// Returns nil in OSS builds (disabled).
func NewIsolationEnforcer() *IsolationEnforcer {
	return nil
}

// GetTenant returns the tenant for a given ID.
// Returns nil in OSS (single-tenant mode).
func (e *IsolationEnforcer) GetTenant(ctx context.Context, id string) *Tenant {
	return nil
}

// EnforceQuotas checks tenant quota limits.
// Returns nil in OSS (no quotas).
func (e *IsolationEnforcer) EnforceQuotas(ctx context.Context, tenantID string) error {
	return nil
}

// RecordUsage records usage for a tenant.
// No-op in OSS.
func (e *IsolationEnforcer) RecordUsage(ctx context.Context, tenantID, route string) {
}
