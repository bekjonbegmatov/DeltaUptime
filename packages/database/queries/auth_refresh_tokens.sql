-- name: CreateAuthRefreshToken :one
INSERT INTO auth_refresh_tokens (
    user_id,
    token_hash,
    expires_at
) VALUES (
    $1,
    $2,
    $3
)
RETURNING id, user_id, token_hash, expires_at, created_at, used_at, revoked_at, replaced_by_token_id;

-- name: GetAuthRefreshTokenByTokenHash :one
SELECT id, user_id, token_hash, expires_at, created_at, used_at, revoked_at, replaced_by_token_id
FROM auth_refresh_tokens
WHERE token_hash = $1
LIMIT 1;

-- name: RotateAuthRefreshToken :one
UPDATE auth_refresh_tokens
SET used_at = now(),
    replaced_by_token_id = $2
WHERE id = $1
  AND used_at IS NULL
  AND revoked_at IS NULL
RETURNING id, user_id, token_hash, expires_at, created_at, used_at, revoked_at, replaced_by_token_id;

-- name: RevokeAuthRefreshToken :one
UPDATE auth_refresh_tokens
SET revoked_at = now()
WHERE id = $1
  AND revoked_at IS NULL
RETURNING id, user_id, token_hash, expires_at, created_at, used_at, revoked_at, replaced_by_token_id;
