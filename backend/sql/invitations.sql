-- name: CreateInvitation :one
INSERT INTO invitations (from_user, to_email, property_id, contract_id, role, token, proposed, last_actor, status)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, 'pending')
RETURNING *;

-- name: GetInvitationByToken :one
SELECT * FROM invitations WHERE token = $1;

-- name: ListInvitationsForUser :many
-- Open invitations the user is involved in, as recipient (to_email) or sender (from_user),
-- so both sides can see counters in a free back-and-forth negotiation.
SELECT * FROM invitations
WHERE (to_email = $1 OR from_user = $2) AND status IN ('pending', 'countered')
ORDER BY created_at DESC;

-- name: CounterInvitation :one
UPDATE invitations SET proposed = $2, last_actor = $3, status = 'countered' WHERE id = $1 RETURNING *;

-- name: SetInvitationStatus :one
UPDATE invitations SET status = $2 WHERE id = $1 RETURNING *;
