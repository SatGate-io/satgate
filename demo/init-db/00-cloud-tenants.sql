-- Pre-migration: Create cloud_tenants table so gateway migrations can reference it
-- This table is normally created by cloud.EnsureSchema() which runs AFTER migrations,
-- but migrations 003+ have FK references to it.

CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS cloud_tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    slug VARCHAR(63) UNIQUE NOT NULL,
    name VARCHAR(255) NOT NULL,
    email VARCHAR(255) NOT NULL,
    status VARCHAR(20) DEFAULT 'trial',
    plan VARCHAR(20) DEFAULT 'trial',
    stripe_customer_id VARCHAR(255),
    lightning_address VARCHAR(255),
    metadata JSONB DEFAULT '{}',
    trial_ends_at TIMESTAMPTZ,
    capability_root_key VARCHAR(128),
    created_at TIMESTAMPTZ DEFAULT NOW(),
    updated_at TIMESTAMPTZ DEFAULT NOW()
);

CREATE INDEX IF NOT EXISTS idx_cloud_tenants_slug ON cloud_tenants(slug);
CREATE INDEX IF NOT EXISTS idx_cloud_tenants_email ON cloud_tenants(email);
CREATE INDEX IF NOT EXISTS idx_cloud_tenants_status ON cloud_tenants(status);
