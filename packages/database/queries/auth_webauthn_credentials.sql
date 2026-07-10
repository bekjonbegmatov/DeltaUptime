-- name: CreateAuthWebAuthnCredential :one
INSERT INTO auth_webauthn_credentials (
    user_id,
    credential_id,
    credential_ciphertext,
    credential_nonce
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING id, user_id, credential_id, credential_ciphertext, credential_nonce, created_at, last_used_at;

-- name: ListAuthWebAuthnCredentialsByUserID :many
SELECT id, user_id, credential_id, credential_ciphertext, credential_nonce, created_at, last_used_at
FROM auth_webauthn_credentials
WHERE user_id = $1
ORDER BY created_at ASC, id ASC;

-- name: UpdateAuthWebAuthnCredential :one
UPDATE auth_webauthn_credentials
SET credential_ciphertext = $2,
    credential_nonce = $3,
    last_used_at = now()
WHERE id = $1
RETURNING id, user_id, credential_id, credential_ciphertext, credential_nonce, created_at, last_used_at;
