-- name: CreateSession :exec
INSERT INTO sessions (id, workspace_id, user_id, seed_id, parent_leaf_id,
    session_type, status, messages, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10);

-- name: FindSessionByID :one
SELECT id, workspace_id, user_id, seed_id, parent_leaf_id,
    session_type, status, messages, created_at, updated_at
FROM sessions WHERE id = $1;

-- name: FindActiveSessionsByUser :many
SELECT id, workspace_id, user_id, seed_id, parent_leaf_id,
    session_type, status, messages, created_at, updated_at
FROM sessions
WHERE workspace_id = $1 AND user_id = $2 AND status NOT IN ('completed', 'abandoned')
ORDER BY created_at DESC;

-- name: UpdateSession :execresult
UPDATE sessions
SET status = $2, messages = $3, updated_at = $4
WHERE id = $1;
