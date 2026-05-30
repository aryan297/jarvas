-- Migration: 001_create_users
-- Description: Core users table with RBAC support and OAuth linkage.
-- Multi-tenant awareness: tenant_id allows future sharding by tenant.

CREATE TYPE user_role AS ENUM ('ADMIN', 'USER', 'PREMIUM_USER');
CREATE TYPE user_status AS ENUM ('ACTIVE', 'INACTIVE', 'BANNED');
CREATE TYPE auth_provider AS ENUM ('LOCAL', 'GOOGLE');

CREATE TABLE users (
    id            UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    email         VARCHAR(255) NOT NULL,
    password_hash VARCHAR(255),                    -- NULL for OAuth-only accounts
    full_name     VARCHAR(255) NOT NULL,
    avatar_url    VARCHAR(512),
    role          user_role    NOT NULL DEFAULT 'USER',
    status        user_status  NOT NULL DEFAULT 'ACTIVE',
    provider      auth_provider NOT NULL DEFAULT 'LOCAL',
    provider_id   VARCHAR(255),                   -- Google sub claim
    tenant_id     UUID,                           -- NULL = personal account
    email_verified BOOLEAN NOT NULL DEFAULT FALSE,
    metadata      JSONB NOT NULL DEFAULT '{}',
    last_login_at TIMESTAMPTZ,
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

-- Unique email per tenant (NULL tenant = global constraint)
CREATE UNIQUE INDEX idx_users_email_tenant
    ON users (COALESCE(tenant_id, '00000000-0000-0000-0000-000000000000'::UUID), email);

CREATE UNIQUE INDEX idx_users_provider_id
    ON users (provider, provider_id)
    WHERE provider_id IS NOT NULL;

CREATE INDEX idx_users_tenant_id   ON users (tenant_id);
CREATE INDEX idx_users_role        ON users (role);
CREATE INDEX idx_users_status      ON users (status);
CREATE INDEX idx_users_created_at  ON users (created_at DESC);
