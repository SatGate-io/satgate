-- Public demo dependency closure for the gateway billing projection outbox.
-- Adapted from gateway/internal/billing/store/postgres.go at source commit
-- 200cdec3a8d820ec6da19ad5848fa9392db3f5b9.

CREATE TABLE IF NOT EXISTS billing_usage (
    id SERIAL PRIMARY KEY,
    created_at TIMESTAMPTZ DEFAULT NOW(),
    event_time TIMESTAMPTZ NOT NULL,
    duration_ms INTEGER,
    tenant_id VARCHAR(255) NOT NULL,
    cost_center VARCHAR(255),
    user_id VARCHAR(255),
    token_id VARCHAR(255),
    route_name VARCHAR(255) NOT NULL,
    method VARCHAR(16),
    path TEXT,
    status_code INTEGER,
    bytes_in BIGINT,
    bytes_out BIGINT,
    unit VARCHAR(32) NOT NULL,
    amount DECIMAL(18,8) NOT NULL,
    payment_mode VARCHAR(32),
    labels JSONB,
    billing_subject_id VARCHAR(255),
    caller_request_id VARCHAR(256),
    request_identity VARCHAR(128),
    surface VARCHAR(16),
    mcp_method VARCHAR(64),
    mcp_tool VARCHAR(255)
);

CREATE INDEX IF NOT EXISTS idx_billing_usage_tenant
    ON billing_usage(tenant_id, event_time);
CREATE INDEX IF NOT EXISTS idx_billing_usage_cost_center
    ON billing_usage(cost_center, event_time);
CREATE INDEX IF NOT EXISTS idx_billing_usage_payment_mode
    ON billing_usage(tenant_id, payment_mode, event_time);
CREATE UNIQUE INDEX IF NOT EXISTS uq_billing_usage_chargeable_identity
    ON billing_usage (tenant_id, billing_subject_id, request_identity)
    WHERE request_identity IS NOT NULL
      AND payment_mode IN ('fiat402', 'control', 'l402', 'charge');
