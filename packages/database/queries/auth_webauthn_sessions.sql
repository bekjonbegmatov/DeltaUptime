-- name: CreateAuthWebAuthnSession :one
INSERT INTO auth_webauthn_sessions (
    user_id,
    flow,
    session_data,
    expires_at
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING id, user_id, flow, session_data, expires_at, created_at;

-- name: GetAuthWebAuthnSessionByID :one
SELECT id, user_id, flow, session_data, expires_at, created_at
FROM auth_webauthn_sessions
WHERE id = $1
LIMIT 1;

-- name: DeleteAuthWebAuthnSession :exec
DELETE FROM auth_webauthn_sessions
WHERE id = $1;
