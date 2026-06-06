-- Migration: 012_create_tenants
-- Description: Org-level multi-tenancy. Users belong to one or more tenants.
-- Data isolation is enforced at the middleware layer via X-Tenant-ID header.
-- Existing tables remain user-scoped; tenant_id enriches context for collaboration.

CREATE TYPE tenant_role AS ENUM ('OWNER', 'ADMIN', 'MEMBER');

CREATE TABLE tenants (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name        VARCHAR(255) NOT NULL,
    slug        VARCHAR(100) NOT NULL UNIQUE,
    plan        VARCHAR(50) NOT NULL DEFAULT 'free',  -- free | pro | enterprise
    owner_id    UUID NOT NULL REFERENCES users(id) ON DELETE RESTRICT,
    is_active   BOOLEAN NOT NULL DEFAULT TRUE,
    created_at  TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at  TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE TABLE tenant_members (
    id          UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id   UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    user_id     UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    role        tenant_role NOT NULL DEFAULT 'MEMBER',
    invited_by  UUID REFERENCES users(id) ON DELETE SET NULL,
    joined_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (tenant_id, user_id)
);

CREATE INDEX idx_tenants_owner_id         ON tenants (owner_id);
CREATE INDEX idx_tenant_members_user_id   ON tenant_members (user_id);
CREATE INDEX idx_tenant_members_tenant_id ON tenant_members (tenant_id);
