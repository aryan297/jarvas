-- Migration: 008_create_memories
-- Description: Long-term memory store. Short-term lives in Redis (TTL-based).
-- Semantic memory vectors are stored in Qdrant; this table holds the source text
-- and metadata for hybrid retrieval.

CREATE TYPE memory_type AS ENUM (
    'FACT',        -- "User prefers Python over Java"
    'PREFERENCE',  -- "User likes dark mode"
    'EVENT',       -- "User mentioned birthday is Dec 5"
    'SKILL',       -- "User is proficient in Go"
    'RELATIONSHIP' -- "User's manager is Alice"
);

CREATE TABLE memories (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    agent_id    UUID REFERENCES agents(id) ON DELETE SET NULL,
    type        memory_type NOT NULL DEFAULT 'FACT',
    content     TEXT NOT NULL,
    importance  NUMERIC(3,2) NOT NULL DEFAULT 0.5,  -- 0-1 relevance weight
    qdrant_id   UUID,
    last_accessed_at TIMESTAMPTZ,
    access_count     INTEGER NOT NULL DEFAULT 0,
    expires_at  TIMESTAMPTZ,    -- NULL = permanent
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_memories_user_id   ON memories (user_id);
CREATE INDEX idx_memories_type      ON memories (type);
CREATE INDEX idx_memories_importance ON memories (user_id, importance DESC);
CREATE INDEX idx_memories_expires   ON memories (expires_at)
    WHERE expires_at IS NOT NULL;
