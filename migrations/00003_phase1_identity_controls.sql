-- +goose Up
-- Phase 1 identity controls: TOTP, API keys, audit log.

CREATE TABLE user_totp_credentials (
    user_id            uuid PRIMARY KEY REFERENCES users (id) ON DELETE CASCADE,
    secret_ciphertext  bytea       NOT NULL,
    secret_nonce       bytea       NOT NULL,
    enabled_at         timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now(),
    updated_at         timestamptz NOT NULL DEFAULT now()
);

CREATE TABLE api_keys (
    id                 uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id    uuid        NOT NULL REFERENCES organizations (id) ON DELETE CASCADE,
    created_by_user_id uuid        REFERENCES users (id) ON DELETE SET NULL,
    name               text        NOT NULL,
    key_prefix         text        NOT NULL UNIQUE,
    key_hash           text        NOT NULL UNIQUE,
    scopes             text[]      NOT NULL DEFAULT '{}',
    last_used_at       timestamptz,
    revoked_at         timestamptz,
    created_at         timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_api_keys_org ON api_keys (organization_id, created_at DESC);

CREATE TABLE audit_logs (
    id               uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    organization_id  uuid        REFERENCES organizations (id) ON DELETE CASCADE,
    actor_type       text        NOT NULL CHECK (actor_type IN ('user', 'api_key', 'system')),
    actor_user_id    uuid        REFERENCES users (id) ON DELETE SET NULL,
    actor_api_key_id uuid        REFERENCES api_keys (id) ON DELETE SET NULL,
    action           text        NOT NULL,
    target_type      text        NOT NULL,
    target_id        text        NOT NULL DEFAULT '',
    metadata         jsonb       NOT NULL DEFAULT '{}'::jsonb,
    occurred_at      timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_audit_logs_org_occurred ON audit_logs (organization_id, occurred_at DESC);
CREATE INDEX idx_audit_logs_actor_user ON audit_logs (actor_user_id, occurred_at DESC);

-- +goose Down
DROP TABLE IF EXISTS audit_logs;
DROP TABLE IF EXISTS api_keys;
DROP TABLE IF EXISTS user_totp_credentials;
