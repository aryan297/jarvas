-- Migration: 004_create_conversations
-- Description: Chat session container. One conversation = one thread of messages.
-- Conversations are scoped to a user and optionally to a specific agent.

CREATE TYPE conversation_status AS ENUM ('ACTIVE', 'ARCHIVED', 'DELETED');

CREATE TABLE conversations (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_id   UUID REFERENCES agents(id) ON DELETE SET NULL,
    title      VARCHAR(500),
    status     conversation_status NOT NULL DEFAULT 'ACTIVE',
    metadata   JSONB NOT NULL DEFAULT '{}',
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_conversations_user_id    ON conversations (user_id);
CREATE INDEX idx_conversations_agent_id   ON conversations (agent_id);
CREATE INDEX idx_conversations_status     ON conversations (status);
CREATE INDEX idx_conversations_updated_at ON conversations (updated_at DESC);
