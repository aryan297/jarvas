-- Migration: 010_create_tools
-- Description: Tool registry and per-user configuration.
-- Tools are registered globally; users bind credentials/settings per-tool.

CREATE TYPE tool_category AS ENUM (
    'DATABASE', 'HTTP', 'PRODUCTIVITY', 'COMMUNICATION', 'DEVELOPMENT', 'CUSTOM'
);

CREATE TABLE tools (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(100) NOT NULL UNIQUE,
    display_name VARCHAR(255) NOT NULL,
    description TEXT,
    category    tool_category NOT NULL DEFAULT 'CUSTOM',
    schema      JSONB NOT NULL DEFAULT '{}',  -- OpenAPI-style input schema
    is_builtin  BOOLEAN NOT NULL DEFAULT FALSE,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Per-user tool credentials and settings.
CREATE TABLE user_tool_configs (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    tool_id     UUID NOT NULL REFERENCES tools(id) ON DELETE CASCADE,
    config      JSONB NOT NULL DEFAULT '{}',  -- encrypted at application layer
    is_enabled  BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (user_id, tool_id)
);

CREATE INDEX idx_user_tool_configs_user ON user_tool_configs (user_id);
CREATE INDEX idx_tools_category         ON tools (category);

-- Seed built-in tools.
INSERT INTO tools (name, display_name, description, category, is_builtin, schema) VALUES
    ('postgres_query',    'PostgreSQL Query',   'Execute read-only SQL queries',  'DATABASE',     TRUE, '{"type":"object","properties":{"query":{"type":"string"}}}'),
    ('http_request',      'HTTP Request',       'Make HTTP requests to any URL',  'HTTP',         TRUE, '{"type":"object","properties":{"url":{"type":"string"},"method":{"type":"string"}}}'),
    ('github',            'GitHub',             'Interact with GitHub repos',     'DEVELOPMENT',  TRUE, '{"type":"object","properties":{"owner":{"type":"string"},"repo":{"type":"string"}}}'),
    ('google_calendar',   'Google Calendar',    'Read and create calendar events', 'PRODUCTIVITY', TRUE, '{"type":"object","properties":{"action":{"type":"string"}}}'),
    ('email',             'Email',              'Send emails via SMTP',           'COMMUNICATION',TRUE, '{"type":"object","properties":{"to":{"type":"string"},"subject":{"type":"string"},"body":{"type":"string"}}}');
