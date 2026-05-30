-- Migration: 009_create_workflows
-- Description: Workflow definitions (DAG of steps) and their execution runs.
-- definition stores the full workflow graph as JSONB for schema flexibility.

CREATE TYPE workflow_status AS ENUM ('DRAFT', 'ACTIVE', 'PAUSED', 'ARCHIVED');
CREATE TYPE run_status AS ENUM (
    'PENDING', 'RUNNING', 'COMPLETED', 'FAILED', 'CANCELLED', 'TIMEOUT'
);

CREATE TABLE workflows (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name        VARCHAR(255) NOT NULL,
    description TEXT,
    status      workflow_status NOT NULL DEFAULT 'DRAFT',
    definition  JSONB NOT NULL DEFAULT '{}',  -- nodes, edges, triggers
    trigger_type VARCHAR(50),                 -- MANUAL, SCHEDULE, WEBHOOK, EVENT
    cron_expr   VARCHAR(100),
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE workflow_runs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    workflow_id UUID NOT NULL REFERENCES workflows(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    status      run_status NOT NULL DEFAULT 'PENDING',
    trigger_payload JSONB,
    result      JSONB,
    error_msg   TEXT,
    started_at  TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflows_user_id   ON workflows (user_id);
CREATE INDEX idx_workflows_status    ON workflows (status);
CREATE INDEX idx_runs_workflow_id    ON workflow_runs (workflow_id, created_at DESC);
CREATE INDEX idx_runs_user_id        ON workflow_runs (user_id);
CREATE INDEX idx_runs_status         ON workflow_runs (status);
