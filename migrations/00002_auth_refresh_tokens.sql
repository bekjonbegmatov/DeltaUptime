-- +goose Up
-- Refresh-token storage for rotating auth sessions.

CREATE TABLE auth_refresh_tokens (
    id                   uuid PRIMARY KEY DEFAULT gen_random_uuid(),
    user_id              uuid        NOT NULL REFERENCES users (id) ON DELETE CASCADE,
    token_hash           text        NOT NULL UNIQUE,
    expires_at           timestamptz NOT NULL,
    created_at           timestamptz NOT NULL DEFAULT now(),
    used_at              timestamptz,
    revoked_at           timestamptz,
    replaced_by_token_id uuid        REFERENCES auth_refresh_tokens (id) ON DELETE SET NULL
);

CREATE INDEX idx_auth_refresh_tokens_user ON auth_refresh_tokens (user_id);
CREATE INDEX idx_auth_refresh_tokens_expires_at ON auth_refresh_tokens (expires_at);

-- +goose Down
DROP TABLE IF EXISTS auth_refresh_tokens;
