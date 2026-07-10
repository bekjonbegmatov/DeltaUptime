-- name: CreateOrganization :one
INSERT INTO organizations (
    slug,
    name
) VALUES (
    $1,
    $2
)
RETURNING id, slug, name, created_at, updated_at;

-- name: GetOrganizationByID :one
SELECT id, slug, name, created_at, updated_at
FROM organizations
WHERE id = $1
LIMIT 1;

-- name: GetOrganizationBySlug :one
SELECT id, slug, name, created_at, updated_at
FROM organizations
WHERE slug = $1
LIMIT 1;

-- name: ListOrganizationsByUser :many
SELECT
    o.id,
    o.slug,
    o.name,
    o.created_at,
    o.updated_at,
    m.role
FROM organizations AS o
JOIN memberships AS m
    ON m.organization_id = o.id
WHERE m.user_id = $1
ORDER BY o.created_at ASC, o.id ASC;
