-- name: CreateGrant :one
INSERT INTO access_grants (user_id, property_id, role, granted_by)
VALUES ($1, $2, $3, $4)
ON CONFLICT (user_id, property_id) DO UPDATE SET role = EXCLUDED.role
RETURNING *;

-- name: ListGrantsForProperty :many
SELECT g.*, u.email AS user_email, u.name AS user_name
FROM access_grants g
JOIN users u ON u.id = g.user_id
WHERE g.property_id = $1
ORDER BY g.created_at DESC;

-- name: DeleteGrant :exec
DELETE FROM access_grants WHERE property_id = $1 AND user_id = $2;
