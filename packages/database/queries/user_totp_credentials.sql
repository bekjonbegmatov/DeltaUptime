-- name: UpsertUserTOTPCredential :one
INSERT INTO user_totp_credentials (
    user_id,
    secret_ciphertext,
    secret_nonce,
    enabled_at,
    created_at,
    updated_at
) VALUES (
    $1,
    $2,
    $3,
    NULL,
    now(),
    now()
)
ON CONFLICT (user_id) DO UPDATE
SET secret_ciphertext = EXCLUDED.secret_ciphertext,
    secret_nonce = EXCLUDED.secret_nonce,
    enabled_at = NULL,
    updated_at = now()
RETURNING user_id, secret_ciphertext, secret_nonce, enabled_at, created_at, updated_at;

-- name: GetUserTOTPCredentialByUserID :one
SELECT user_id, secret_ciphertext, secret_nonce, enabled_at, created_at, updated_at
FROM user_totp_credentials
WHERE user_id = $1
LIMIT 1;

-- name: EnableUserTOTPCredential :one
UPDATE user_totp_credentials
SET enabled_at = now(),
    updated_at = now()
WHERE user_id = $1
RETURNING user_id, secret_ciphertext, secret_nonce, enabled_at, created_at, updated_at;

-- name: DeleteUserTOTPCredential :exec
DELETE FROM user_totp_credentials
WHERE user_id = $1;
