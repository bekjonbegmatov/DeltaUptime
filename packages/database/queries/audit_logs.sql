-- name: CreateAuditLog :one
INSERT INTO audit_logs (
    organization_id,
    actor_type,
    actor_user_id,
    actor_api_key_id,
    action,
    target_type,
    target_id,
    metadata
) VALUES (
    $1,
    $2,
    $3,
    $4,
    $5,
    $6,
    $7,
    $8
)
RETURNING id, organization_id, actor_type, actor_user_id, actor_api_key_id, action, target_type, target_id, metadata, occurred_at;

-- name: ListAuditLogsByOrganization :many
SELECT id, organization_id, actor_type, actor_user_id, actor_api_key_id, action, target_type, target_id, metadata, occurred_at
FROM audit_logs
WHERE organization_id = $1
ORDER BY occurred_at DESC, id DESC
LIMIT $2
OFFSET $3;
