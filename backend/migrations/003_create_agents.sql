-- Migration: 003_create_agents
-- Description: AI agent definitions. A user can create multiple named agents
-- with different system prompts, tool configs, and model settings.

CREATE TYPE agent_type AS ENUM (
    'SUPERVISOR', 'RESEARCH', 'CODING', 'PLANNING', 'WORKFLOW', 'CUSTOM'
);

CREATE TABLE agents (
    id             UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id        UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name           VARCHAR(255) NOT NULL,
    description    TEXT,
    type           agent_type NOT NULL DEFAULT 'CUSTOM',
    system_prompt  TEXT,
    model          VARCHAR(100) NOT NULL DEFAULT 'gpt-4o',
    temperature    NUMERIC(3,2) NOT NULL DEFAULT 0.7,
    max_tokens     INTEGER NOT NULL DEFAULT 4096,
    tools_enabled  JSONB NOT NULL DEFAULT '[]',   -- list of tool IDs
    memory_enabled BOOLEAN NOT NULL DEFAULT TRUE,
    rag_enabled    BOOLEAN NOT NULL DEFAULT FALSE,
    is_active      BOOLEAN NOT NULL DEFAULT TRUE,
    metadata       JSONB NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_agents_user_id ON agents (user_id);
CREATE INDEX idx_agents_type    ON agents (type);
