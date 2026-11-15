-- name: UpsertUserFromGoogle :one
-- Find-or-create by email; links the Google identity (handles users first created
-- as an invited email, then signing in with Google).
INSERT INTO users (email, name, google_sub, avatar_url)
VALUES ($1, $2, $3, $4)
ON CONFLICT (email) DO UPDATE
SET name       = EXCLUDED.name,
    google_sub = EXCLUDED.google_sub,
    avatar_url = EXCLUDED.avatar_url
RETURNING *;

-- name: GetUser :one
SELECT * FROM users WHERE id = $1;

-- name: GetUserByEmail :one
SELECT * FROM users WHERE email = $1;

-- name: EnsureUserByEmail :one
-- Used when inviting: create a placeholder user for an email that hasn't signed up.
INSERT INTO users (email) VALUES ($1)
ON CONFLICT (email) DO UPDATE SET email = EXCLUDED.email
RETURNING *;
