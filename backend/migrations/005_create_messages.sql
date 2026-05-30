-- Migration: 005_create_messages
-- Description: Individual messages within a conversation.
-- token_count is populated post-generation for cost tracking.

CREATE TYPE message_role AS ENUM ('USER', 'ASSISTANT', 'SYSTEM', 'TOOL');

CREATE TABLE messages (
    id              UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    conversation_id UUID NOT NULL REFERENCES conversations(id) ON DELETE CASCADE,
    role            message_role NOT NULL,
    content         TEXT NOT NULL,
    token_count     INTEGER,
    model           VARCHAR(100),
    tool_calls      JSONB,          -- structured tool call payload when role=TOOL
    metadata        JSONB NOT NULL DEFAULT '{}',
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_messages_conversation_id ON messages (conversation_id, created_at ASC);
CREATE INDEX idx_messages_role           ON messages (role);
