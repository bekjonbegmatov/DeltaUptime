-- +goose Up
-- Foundational multi-tenant schema: organizations, users, memberships.
-- Roles are permission-driven; the `role` column is a named preset (owner/admin/
-- operator/viewer/billing) — fine-grained permissions arrive in a later migration.

CREATE EXTENSION IF NOT EXISTS "pgcrypto"; -- provides gen_random_uuid()

CREATE TABLE organizations (
    id         uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    slug       text        NOT NULL UNIQUE,
    name       text        NOT NULL,
    created_at timestamptz NOT NULL DEFAULT now(),
    updated_at timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE users (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    email           text        NOT NULL UNIQUE,
    password_hash   text        NOT NULL,               -- Argon2id
    display_name    text        NOT NULL DEFAULT '',
    is_system_admin boolean     NOT NULL DEFAULT false,
    created_at      timestamptz NOT NULL DEFAULT now(),
    updated_at      timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE memberships (
    id              uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    user_id         uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    role            text        NOT NULL CHECK (role IN ('owner', 'admin', 'operator', 'viewer', 'billing')),
    created_at      timestamptz NOT NULL DEFAULT now(),
    UNIQUE (organization_id, user_id)
);

CREATE INDEX idx_memberships_user ON memberships (user_id);
CREATE INDEX idx_memberships_org ON memberships (organization_id);

-- +goose Down
DROP TABLE IF EXISTS memberships;
DROP TABLE IF EXISTS users;
DROP TABLE IF EXISTS organizations;
