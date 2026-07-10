-- +goose Up
-- Phase 1 WebAuthn / passkeys.

ALTER TABLE users
ADD COLUMN webauthn_user_handle bytea;

CREATE UNIQUE INDEX idx_users_webauthn_user_handle
    ON users (webauthn_user_handle)
    WHERE webauthn_user_handle IS NOT NULL;

CREATE TABLE auth_webauthn_credentials (
    id                  uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id             uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    credential_id       bytea       NOT NULL UNIQUE,
    credential_ciphertext bytea     NOT NULL,
    credential_nonce    bytea       NOT NULL,
    created_at          timestamptz NOT NULL DEFAULT now(),
    last_used_at        timestamptz
);

CREATE INDEX idx_auth_webauthn_credentials_user ON auth_webauthn_credentials (user_id, created_at DESC);

CREATE TABLE auth_webauthn_sessions (
    id           uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id      uuid        REFERENCES users (id) ON DELETE CASCADE,
    flow         text        NOT NULL CHECK (flow IN ('registration', 'login')),
    session_data jsonb       NOT NULL,
    expires_at   timestamptz NOT NULL,
    created_at   timestamptz NOT NULL DEFAULT now()
);

CREATE INDEX idx_auth_webauthn_sessions_user ON auth_webauthn_sessions (user_id, created_at DESC);
CREATE INDEX idx_auth_webauthn_sessions_expires ON auth_webauthn_sessions (expires_at);

-- +goose Down
DROP TABLE IF EXISTS auth_webauthn_sessions;
DROP TABLE IF EXISTS auth_webauthn_credentials;
DROP INDEX IF EXISTS idx_users_webauthn_user_handle;
ALTER TABLE users DROP COLUMN IF EXISTS webauthn_user_handle;
