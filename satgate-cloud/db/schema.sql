-- SatGate Cloud Database Schema
-- PostgreSQL 15+

-- Enable extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- =============================================================================
-- TENANTS (Users)
-- =============================================================================

CREATE TABLE tenants (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  email TEXT UNIQUE NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_tenants_email ON tenants(email);

-- =============================================================================
-- AUTH CODES (Magic Link)
-- =============================================================================

CREATE TABLE auth_codes (
  id BIGSERIAL PRIMARY KEY,
  email TEXT NOT NULL,
  code_hash TEXT NOT NULL,  -- SHA256 of the code
  expires_at TIMESTAMPTZ NOT NULL,
  used BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_auth_codes_email ON auth_codes(email);
CREATE INDEX idx_auth_codes_expires ON auth_codes(expires_at) WHERE NOT used;

-- =============================================================================
-- SESSIONS
-- =============================================================================

CREATE TABLE sessions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  token_hash TEXT NOT NULL,  -- SHA256 of session token
  expires_at TIMESTAMPTZ NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_sessions_tenant ON sessions(tenant_id);
CREATE INDEX idx_sessions_token ON sessions(token_hash);

-- =============================================================================
-- PROJECTS
-- =============================================================================

CREATE TABLE projects (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  slug TEXT UNIQUE NOT NULL,  -- becomes <slug>.satgate.cloud
  name TEXT NOT NULL,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_projects_tenant ON projects(tenant_id);
CREATE INDEX idx_projects_slug ON projects(slug);

-- Enforce slug format
ALTER TABLE projects ADD CONSTRAINT chk_slug_format 
  CHECK (slug ~ '^[a-z0-9][a-z0-9-]{1,62}[a-z0-9]$');

-- =============================================================================
-- CONFIG VERSIONS (Immutable, append-only)
-- =============================================================================

CREATE TABLE config_versions (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  yaml_content TEXT NOT NULL,
  route_summary JSONB,  -- cached: [{name, path, tier, price}]
  is_active BOOLEAN NOT NULL DEFAULT false,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  created_by UUID REFERENCES tenants(id)
);

CREATE INDEX idx_config_versions_project ON config_versions(project_id);
CREATE INDEX idx_config_versions_active ON config_versions(project_id) WHERE is_active;

-- Ensure only one active config per project
CREATE UNIQUE INDEX idx_one_active_config 
  ON config_versions(project_id) 
  WHERE is_active;

-- =============================================================================
-- PROJECT SECRETS (Encrypted)
-- =============================================================================

CREATE TABLE project_secrets (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  key TEXT NOT NULL,  -- e.g., "API_KEY"
  value_encrypted BYTEA NOT NULL,  -- encrypted with pgcrypto or app-level encryption
  created_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  updated_at TIMESTAMPTZ NOT NULL DEFAULT now(),
  UNIQUE(project_id, key)
);

CREATE INDEX idx_project_secrets_project ON project_secrets(project_id);

-- =============================================================================
-- API KEYS (For automation)
-- =============================================================================

CREATE TABLE api_keys (
  id UUID PRIMARY KEY DEFAULT uuid_generate_v4(),
  tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
  key_hash TEXT NOT NULL,  -- SHA256 of the key
  name TEXT,
  last_used_at TIMESTAMPTZ,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_keys_tenant ON api_keys(tenant_id);
CREATE INDEX idx_api_keys_hash ON api_keys(key_hash);

-- =============================================================================
-- USAGE EVENTS (Time-series)
-- =============================================================================

CREATE TABLE usage_events (
  id BIGSERIAL PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  route_name TEXT,
  event_type TEXT NOT NULL,  -- 'challenge' | 'paid' | 'allowed' | 'denied'
  price_sats INTEGER,
  request_id TEXT,
  created_at TIMESTAMPTZ NOT NULL DEFAULT now()
);

CREATE INDEX idx_usage_events_project_time ON usage_events(project_id, created_at DESC);
CREATE INDEX idx_usage_events_project_type ON usage_events(project_id, event_type);

-- =============================================================================
-- USAGE AGGREGATES (Daily rollups)
-- =============================================================================

CREATE TABLE usage_daily (
  id BIGSERIAL PRIMARY KEY,
  project_id UUID NOT NULL REFERENCES projects(id) ON DELETE CASCADE,
  date DATE NOT NULL,
  route_name TEXT,
  challenges INTEGER NOT NULL DEFAULT 0,
  paid INTEGER NOT NULL DEFAULT 0,
  allowed INTEGER NOT NULL DEFAULT 0,
  denied INTEGER NOT NULL DEFAULT 0,
  sats_invoiced BIGINT NOT NULL DEFAULT 0,
  sats_paid BIGINT NOT NULL DEFAULT 0,
  UNIQUE(project_id, date, route_name)
);

CREATE INDEX idx_usage_daily_project_date ON usage_daily(project_id, date DESC);

-- =============================================================================
-- FUNCTIONS
-- =============================================================================

-- Update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at()
RETURNS TRIGGER AS $$
BEGIN
  NEW.updated_at = now();
  RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Apply to tables
CREATE TRIGGER trg_tenants_updated_at
  BEFORE UPDATE ON tenants
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_projects_updated_at
  BEFORE UPDATE ON projects
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

CREATE TRIGGER trg_project_secrets_updated_at
  BEFORE UPDATE ON project_secrets
  FOR EACH ROW EXECUTE FUNCTION update_updated_at();

-- =============================================================================
-- CLEANUP (Scheduled job or app-level)
-- =============================================================================

-- Delete expired auth codes (run periodically)
-- DELETE FROM auth_codes WHERE expires_at < now() - interval '1 day';

-- Delete expired sessions (run periodically)
-- DELETE FROM sessions WHERE expires_at < now();

