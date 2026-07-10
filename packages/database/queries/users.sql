-- name: CreateUser :one
INSERT INTO users (
    email,
    password_hash,
    display_name,
    is_system_admin
) VALUES (
    $1,
    $2,
    $3,
    $4
)
RETURNING id, email, password_hash, display_name, is_system_admin, created_at, updated_at, webauthn_user_handle;

-- name: GetUserByID :one
SELECT id, email, password_hash, display_name, is_system_admin, created_at, updated_at, webauthn_user_handle
FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, display_name, is_system_admin, created_at, updated_at, webauthn_user_handle
FROM users
WHERE email = $1
LIMIT 1;

-- name: SetUserWebAuthnHandle :one
UPDATE users
SET webauthn_user_handle = $2,
    updated_at = now()
WHERE id = $1
RETURNING id, email, password_hash, display_name, is_system_admin, created_at, updated_at, webauthn_user_handle;

-- name: ListUsersByOrganization :many
SELECT
    u.id,
    u.email,
    u.password_hash,
    u.display_name,
    u.is_system_admin,
    u.created_at,
    u.updated_at,
    u.webauthn_user_handle,
    m.role
FROM users AS u
JOIN memberships AS m
    ON m.user_id = u.id
WHERE m.organization_id = $1
ORDER BY u.created_at ASC, u.id ASC;
