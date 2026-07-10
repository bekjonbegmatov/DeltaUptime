-- name: CreateAPIKey :one
INSERT INTO api_keys (
    organization_id,
    created_by_user_id,
    name,
    key_prefix,
    key_hash,
    scopes
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6
)
RETURNING id, organization_id, created_by_user_id, name, key_prefix, key_hash, scopes, last_used_at, revoked_at, created_at;

-- name: GetAPIKeyByPrefix :one
SELECT id, organization_id, created_by_user_id, name, key_prefix, key_hash, scopes, last_used_at, revoked_at, created_at
FROM api_keys
WHERE key_prefix = $1
LIMIT 1;

-- name: GetAPIKeyByID :one
SELECT id, organization_id, created_by_user_id, name, key_prefix, key_hash, scopes, last_used_at, revoked_at, created_at
FROM api_keys
WHERE id = $1
LIMIT 1;

-- name: ListAPIKeysByOrganization :many
SELECT id, organization_id, created_by_user_id, name, key_prefix, key_hash, scopes, last_used_at, revoked_at, created_at
FROM api_keys
WHERE organization_id = $1
ORDER BY created_at DESC, id DESC;

-- name: TouchAPIKeyLastUsed :one
UPDATE api_keys
SET last_used_at = now()
WHERE id = $1
RETURNING id, organization_id, created_by_user_id, name, key_prefix, key_hash, scopes, last_used_at, revoked_at, created_at;

-- name: RevokeAPIKey :one
UPDATE api_keys
SET revoked_at = now()
WHERE id = $1
  AND revoked_at IS NULL
RETURNING id, organization_id, created_by_user_id, name, key_prefix, key_hash, scopes, last_used_at, revoked_at, created_at;
