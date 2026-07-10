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
RETURNING id, email, password_hash, display_name, is_system_admin, created_at, updated_at;

-- name: GetUserByID :one
SELECT id, email, password_hash, display_name, is_system_admin, created_at, updated_at
FROM users
WHERE id = $1
LIMIT 1;

-- name: GetUserByEmail :one
SELECT id, email, password_hash, display_name, is_system_admin, created_at, updated_at
FROM users
WHERE email = $1
LIMIT 1;

-- name: ListUsersByOrganization :many
SELECT
    u.id,
    u.email,
    u.password_hash,
    u.display_name,
    u.is_system_admin,
    u.created_at,
    u.updated_at,
    m.role
FROM users AS u
JOIN memberships AS m
    ON m.user_id = u.id
WHERE m.organization_id = $1
ORDER BY u.created_at ASC, u.id ASC;
