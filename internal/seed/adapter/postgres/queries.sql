-- name: CreateSeed :exec
INSERT INTO seeds (id, workspace_id, author_id, title, description, constraints, tags, created_at, updated_at, position_x, position_y)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: FindSeedByID :one
SELECT id, workspace_id, author_id, title, description, constraints, tags, created_at, updated_at, position_x, position_y
FROM seeds WHERE id = $1;

-- name: FindSeedsByWorkspace :many
SELECT id, workspace_id, author_id, title, description, constraints, tags, created_at, updated_at, position_x, position_y
FROM seeds WHERE workspace_id = $1 ORDER BY created_at;

-- name: UpdateSeed :execresult
UPDATE seeds
SET constraints = $2, tags = $3, updated_at = $4
WHERE id = $1;

-- name: UpdateSeedPosition :execresult
UPDATE seeds SET position_x = $2, position_y = $3, updated_at = NOW() WHERE id = $1;
