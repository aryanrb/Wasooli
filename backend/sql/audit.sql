-- name: CreateAudit :one
INSERT INTO audit_log (entity_type, entity_id, action, actor, details)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: ListAuditForEntity :many
SELECT a.*, u.name AS actor_name
FROM audit_log a
JOIN users u ON u.id = a.actor
WHERE a.entity_type = $1 AND a.entity_id = $2
ORDER BY a.created_at DESC;
