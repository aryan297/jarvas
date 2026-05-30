-- Migration: 011_create_audit_logs
-- Description: Immutable audit trail. Rows are append-only; no UPDATE/DELETE
-- via application. Partitioned by month for retention management.
-- Future: extract to a dedicated write-optimised store (ClickHouse, TimescaleDB).

CREATE TABLE audit_logs (
    id          BIGSERIAL,
    user_id     UUID REFERENCES users(id) ON DELETE SET NULL,
    action      VARCHAR(100) NOT NULL,   -- e.g. "auth.login", "document.upload"
    resource    VARCHAR(100),            -- e.g. "document"
    resource_id UUID,
    ip_address  INET,
    user_agent  TEXT,
    payload     JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (id, created_at)
) PARTITION BY RANGE (created_at);

-- Create initial partitions (one per quarter; extend via cron).
CREATE TABLE audit_logs_2025_q2 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-04-01') TO ('2025-07-01');
CREATE TABLE audit_logs_2025_q3 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-07-01') TO ('2025-10-01');
CREATE TABLE audit_logs_2025_q4 PARTITION OF audit_logs
    FOR VALUES FROM ('2025-10-01') TO ('2026-01-01');
CREATE TABLE audit_logs_2026_q1 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-01-01') TO ('2026-04-01');
CREATE TABLE audit_logs_2026_q2 PARTITION OF audit_logs
    FOR VALUES FROM ('2026-04-01') TO ('2026-07-01');

CREATE INDEX idx_audit_user_id  ON audit_logs (user_id, created_at DESC);
CREATE INDEX idx_audit_action   ON audit_logs (action, created_at DESC);
CREATE INDEX idx_audit_resource ON audit_logs (resource, resource_id);
