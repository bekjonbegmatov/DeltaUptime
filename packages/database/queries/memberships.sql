-- name: CreateMembership :one
INSERT INTO memberships (
    organization_id,
    user_id,
    role
) VALUES (
    $1,
    $2,
    $3
)
RETURNING id, organization_id, user_id, role, created_at;

-- name: GetMembership :one
SELECT id, organization_id, user_id, role, created_at
FROM memberships
WHERE organization_id = $1
  AND user_id = $2
LIMIT 1;

-- name: ListMembershipsByOrganization :many
SELECT id, organization_id, user_id, role, created_at
FROM memberships
WHERE organization_id = $1
ORDER BY created_at ASC, id ASC;

-- name: ListMembershipsByUser :many
SELECT id, organization_id, user_id, role, created_at
FROM memberships
WHERE user_id = $1
ORDER BY created_at ASC, id ASC;

-- name: UpdateMembershipRole :one
UPDATE memberships
SET role = $3
WHERE organization_id = $1
  AND user_id = $2
RETURNING id, organization_id, user_id, role, created_at;
