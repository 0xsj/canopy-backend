-- name: CreateUser :exec
INSERT INTO users (id, external_id, display_name, email, avatar_url, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: FindUserByID :one
SELECT id, external_id, display_name, email, avatar_url, created_at, updated_at
FROM users WHERE id = $1;

-- name: FindUserByExternalID :one
SELECT id, external_id, display_name, email, avatar_url, created_at, updated_at
FROM users WHERE external_id = $1;

-- name: FindUsersByIDs :many
SELECT id, external_id, display_name, email, avatar_url, created_at, updated_at
FROM users WHERE id = ANY($1::text[]);

-- name: UpdateUser :execresult
UPDATE users
SET display_name = $2, email = $3, avatar_url = $4, updated_at = $5
WHERE id = $1;
