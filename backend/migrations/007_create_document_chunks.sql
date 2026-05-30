-- Migration: 007_create_document_chunks
-- Description: Chunked text extracted from documents.
-- qdrant_id links each chunk to its vector in Qdrant.
-- Keeping chunks in Postgres allows SQL-level filtering before vector search.

CREATE TABLE document_chunks (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    document_id UUID NOT NULL REFERENCES documents(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    content     TEXT NOT NULL,
    chunk_index INTEGER NOT NULL,
    qdrant_id   UUID,               -- point ID in Qdrant collection
    token_count INTEGER,
    metadata    JSONB NOT NULL DEFAULT '{}',
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_chunks_document_id ON document_chunks (document_id, chunk_index ASC);
CREATE INDEX idx_chunks_user_id     ON document_chunks (user_id);
CREATE INDEX idx_chunks_qdrant_id   ON document_chunks (qdrant_id);
