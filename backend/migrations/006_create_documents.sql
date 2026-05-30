-- Migration: 006_create_documents
-- Description: Uploaded documents for RAG. The raw file lives in MinIO;
-- metadata + processing state live here.

CREATE TYPE document_status AS ENUM (
    'UPLOADED', 'PROCESSING', 'INDEXED', 'FAILED'
);
CREATE TYPE document_type AS ENUM (
    'PDF', 'DOCX', 'TXT', 'MD', 'HTML', 'CSV', 'XLSX', 'OTHER'
);

CREATE TABLE documents (
    id           UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    name         VARCHAR(500) NOT NULL,
    type         document_type NOT NULL DEFAULT 'OTHER',
    mime_type    VARCHAR(100),
    size_bytes   BIGINT NOT NULL DEFAULT 0,
    storage_key  VARCHAR(1000) NOT NULL,  -- MinIO object key
    status       document_status NOT NULL DEFAULT 'UPLOADED',
    chunk_count  INTEGER NOT NULL DEFAULT 0,
    error_msg    TEXT,
    metadata     JSONB NOT NULL DEFAULT '{}',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_documents_user_id   ON documents (user_id);
CREATE INDEX idx_documents_status    ON documents (status);
CREATE INDEX idx_documents_type      ON documents (type);
CREATE INDEX idx_documents_created   ON documents (created_at DESC);
