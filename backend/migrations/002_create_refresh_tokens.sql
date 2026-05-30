-- Migration: 002_create_refresh_tokens
-- Description: Persistent refresh token store for JWT rotation strategy.
-- Security: tokens are stored hashed (SHA-256). Rotating invalidates old tokens.

CREATE TABLE refresh_tokens (
    id         UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id    UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    token_hash VARCHAR(64) NOT NULL,  -- SHA-256 hex of the raw token
    device_id  VARCHAR(255),          -- optional device fingerprint for multi-device support
    ip_address INET,
    user_agent TEXT,
    revoked    BOOLEAN NOT NULL DEFAULT FALSE,
    expires_at TIMESTAMPTZ NOT NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_refresh_tokens_hash    ON refresh_tokens (token_hash);
CREATE INDEX        idx_refresh_tokens_user_id ON refresh_tokens (user_id);
CREATE INDEX        idx_refresh_tokens_expires ON refresh_tokens (expires_at)
    WHERE revoked = FALSE;
