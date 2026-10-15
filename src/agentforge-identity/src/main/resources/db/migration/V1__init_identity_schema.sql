-- V1__init_identity_schema.sql
-- AgentForge Identity & Tenant Schema
-- Conforms to docs/03-domains/IDENTITY.md and docs/05-data/DATABASE.md

-- 1. Organizations (Tenant Boundary)
CREATE TABLE organizations (
    id UUID PRIMARY KEY,
    name VARCHAR(255) NOT NULL,
    slug VARCHAR(100) NOT NULL UNIQUE,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) NOT NULL,
    updated_at TIMESTAMPTZ,
    updated_by VARCHAR(255)
);

CREATE INDEX idx_organizations_slug ON organizations(slug);
CREATE INDEX idx_organizations_status ON organizations(status);

-- 2. Users (Authenticated Principals)
CREATE TABLE users (
    id UUID PRIMARY KEY,
    email VARCHAR(255) NOT NULL UNIQUE,
    password_hash VARCHAR(255) NOT NULL,
    full_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'ACTIVE',
    version BIGINT NOT NULL DEFAULT 0,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) NOT NULL,
    updated_at TIMESTAMPTZ,
    updated_by VARCHAR(255)
);

CREATE INDEX idx_users_email ON users(LOWER(email));
CREATE INDEX idx_users_status ON users(status);

-- 3. Memberships (User <-> Organization Association)
CREATE TABLE memberships (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    user_id UUID NOT NULL REFERENCES users(id) ON DELETE CASCADE,
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) NOT NULL,
    updated_at TIMESTAMPTZ,
    updated_by VARCHAR(255),
    CONSTRAINT uq_membership_org_user UNIQUE (organization_id, user_id)
);

CREATE INDEX idx_memberships_org_user ON memberships(organization_id, user_id);
CREATE INDEX idx_memberships_user ON memberships(user_id);

-- 4. Membership Roles
CREATE TABLE membership_roles (
    membership_id UUID NOT NULL REFERENCES memberships(id) ON DELETE CASCADE,
    role VARCHAR(50) NOT NULL,
    PRIMARY KEY (membership_id, role)
);

CREATE INDEX idx_membership_roles_role ON membership_roles(role);

-- 5. API Keys (Programmatic Credentials)
CREATE TABLE api_keys (
    id UUID PRIMARY KEY,
    organization_id UUID NOT NULL REFERENCES organizations(id) ON DELETE CASCADE,
    name VARCHAR(255) NOT NULL,
    key_prefix VARCHAR(32) NOT NULL,
    hashed_secret VARCHAR(255) NOT NULL,
    permissions TEXT NOT NULL,
    expires_at TIMESTAMPTZ,
    revoked_at TIMESTAMPTZ,
    revoked_by VARCHAR(255),
    created_at TIMESTAMPTZ NOT NULL DEFAULT CURRENT_TIMESTAMP,
    created_by VARCHAR(255) NOT NULL
);

CREATE INDEX idx_api_keys_org ON api_keys(organization_id);
CREATE INDEX idx_api_keys_prefix ON api_keys(key_prefix);
CREATE INDEX idx_api_keys_hash ON api_keys(hashed_secret);
