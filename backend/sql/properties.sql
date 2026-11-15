-- name: CreateProperty :one
INSERT INTO properties (owner_id, name, address, currency, owner_vacant)
VALUES ($1, $2, $3, $4, $5)
RETURNING *;

-- name: GetProperty :one
SELECT * FROM properties WHERE id = $1;

-- name: ListPropertiesForUser :many
-- Properties the user manages as the real owner, or can view. Tenants reach their
-- tenancies via ListTenanciesForUser instead — a tenant is scoped to one contract,
-- not the whole property.
SELECT DISTINCT p.*
FROM properties p
LEFT JOIN access_grants g ON g.property_id = p.id AND g.user_id = $1
WHERE (p.owner_id = $1 AND p.owner_vacant = false) OR g.user_id IS NOT NULL
ORDER BY p.created_at DESC;

-- name: GetPropertyAccess :one
-- Authorization flags: $1 = property id, $2 = user id. is_owner = holds the owner seat
-- (a real owner OR a tenant's placeholder); owner_vacant distinguishes the two.
SELECT
    (p.owner_id = $2) AS is_owner,
    p.owner_vacant AS owner_vacant,
    EXISTS (SELECT 1 FROM access_grants g WHERE g.property_id = p.id AND g.user_id = $2) AS is_viewer
FROM properties p
WHERE p.id = $1;

-- name: ClaimPropertyOwner :one
-- A real owner takes over a tenant's standalone property on accepting an owner invite.
UPDATE properties SET owner_id = $2, owner_vacant = false WHERE id = $1 RETURNING *;
