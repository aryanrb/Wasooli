-- name: CreateContract :one
INSERT INTO contracts (property_id, tenant_user_id, tenant_name, start_date, cycle, anchor)
VALUES ($1, $2, $3, $4, $5, $6)
RETURNING *;

-- name: GetContract :one
SELECT * FROM contracts WHERE id = $1;

-- name: ListContractsForProperty :many
SELECT * FROM contracts WHERE property_id = $1 AND status <> 'archived' ORDER BY created_at DESC;

-- name: ListTenanciesForUser :many
-- Contracts where the user is the tenant side — their standalone records and any
-- contract an owner linked them to. This is the tenant's entry point (not the property).
SELECT c.*, p.name AS property_name, p.currency, p.owner_vacant
FROM contracts c
JOIN properties p ON p.id = c.property_id
WHERE c.tenant_user_id = $1 AND c.status <> 'archived'
ORDER BY c.created_at DESC;

-- name: EndContract :one
UPDATE contracts
SET status = 'ended', ended_at = now()
WHERE id = $1 AND status = 'active'
RETURNING *;

-- name: ArchiveContract :one
UPDATE contracts SET status = 'archived', archived_into = $2 WHERE id = $1 RETURNING *;

-- name: SetContractTenant :one
UPDATE contracts SET tenant_user_id = $2 WHERE id = $1 RETURNING *;

-- name: UpdateContractDetails :one
UPDATE contracts SET tenant_name = $2, cycle = $3, anchor = $4, start_date = $5 WHERE id = $1 RETURNING *;

-- name: CreateTerm :one
INSERT INTO contract_terms (contract_id, effective_from, components, total)
VALUES ($1, $2, $3, $4)
RETURNING *;

-- name: ListTermsForContract :many
SELECT * FROM contract_terms WHERE contract_id = $1 ORDER BY effective_from ASC;
