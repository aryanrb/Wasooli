-- name: CreateTransaction :one
INSERT INTO transactions
    (property_id, contract_id, period_key, recorded_by, direction, amount, components, note, paid_at, approval_status, approved_by)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11)
RETURNING *;

-- name: GetTransaction :one
SELECT * FROM transactions WHERE id = $1;

-- name: ListTransactionsForProperty :many
SELECT * FROM transactions WHERE property_id = $1 ORDER BY paid_at DESC, created_at DESC;

-- name: ListTransactionsForContract :many
SELECT * FROM transactions WHERE contract_id = $1 ORDER BY paid_at DESC, created_at DESC;

-- name: ListApprovedIncomeForPeriod :many
SELECT * FROM transactions
WHERE contract_id = $1 AND period_key = $2
  AND approval_status = 'approved' AND direction = 'income';

-- name: ListPendingForUser :many
-- Pending transactions on the user's contracts that they did NOT record (their inbox).
SELECT t.*
FROM transactions t
JOIN contracts c ON c.id = t.contract_id
JOIN properties p ON p.id = t.property_id
WHERE t.approval_status = 'pending'
  AND t.recorded_by <> $1
  AND (p.owner_id = $1 OR c.tenant_user_id = $1)
ORDER BY t.created_at DESC;

-- name: SetTransactionApproval :one
UPDATE transactions
SET approval_status = $2, approved_by = $3
WHERE id = $1
RETURNING *;

-- name: SumApprovedByYear :many
-- Analytics: approved income/expense totals per year across the given properties.
SELECT
    EXTRACT(YEAR FROM paid_at)::int AS year,
    direction,
    SUM(amount)::bigint AS total
FROM transactions
WHERE property_id = ANY($1::uuid[]) AND approval_status = 'approved'
GROUP BY year, direction
ORDER BY year DESC;
