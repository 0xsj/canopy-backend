-- Leaf queries

-- name: CreateLeaf :exec
INSERT INTO leaves (id, workspace_id, seed_id, branch_id, author_id, parent_leaf_id,
    title, summary, key_points, open_questions, tags, layer, sources, metadata, created_at, position_x, position_y)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, $13, $14, $15, $16, $17);

-- name: FindLeafByID :one
SELECT id, workspace_id, seed_id, branch_id, author_id, parent_leaf_id,
    title, summary, key_points, open_questions, tags, layer, sources, metadata, created_at, position_x, position_y
FROM leaves WHERE id = $1;

-- name: FindLeavesByIDs :many
SELECT id, workspace_id, seed_id, branch_id, author_id, parent_leaf_id,
    title, summary, key_points, open_questions, tags, layer, sources, metadata, created_at, position_x, position_y
FROM leaves WHERE id = ANY($1::text[]);

-- name: FindLeavesByWorkspace :many
SELECT id, workspace_id, seed_id, branch_id, author_id, parent_leaf_id,
    title, summary, key_points, open_questions, tags, layer, sources, metadata, created_at, position_x, position_y
FROM leaves
WHERE workspace_id = $1
  AND (sqlc.narg('author_id')::text IS NULL OR author_id = sqlc.narg('author_id'))
  AND (sqlc.narg('seed_id')::text IS NULL OR seed_id = sqlc.narg('seed_id'))
  AND (sqlc.narg('layer')::text IS NULL OR layer = sqlc.narg('layer'))
  AND (sqlc.narg('tags')::text[] IS NULL OR tags && sqlc.narg('tags'))
ORDER BY created_at;

-- name: FindLeavesByBranch :many
SELECT id, workspace_id, seed_id, branch_id, author_id, parent_leaf_id,
    title, summary, key_points, open_questions, tags, layer, sources, metadata, created_at, position_x, position_y
FROM leaves WHERE branch_id = $1 ORDER BY created_at;

-- name: FindLeavesBySeed :many
SELECT id, workspace_id, seed_id, branch_id, author_id, parent_leaf_id,
    title, summary, key_points, open_questions, tags, layer, sources, metadata, created_at, position_x, position_y
FROM leaves WHERE seed_id = $1 ORDER BY created_at;

-- name: UpdateLeafLayer :execresult
UPDATE leaves SET layer = $2 WHERE id = $1;

-- name: UpdateLeafPosition :execresult
UPDATE leaves SET position_x = $2, position_y = $3 WHERE id = $1;

-- Branch queries

-- name: CreateBranch :exec
INSERT INTO branches (id, workspace_id, seed_id, author_id, root_leaf_id, created_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: FindBranchByID :one
SELECT id, workspace_id, seed_id, author_id, root_leaf_id, created_at
FROM branches WHERE id = $1;

-- name: FindBranchesByWorkspace :many
SELECT id, workspace_id, seed_id, author_id, root_leaf_id, created_at
FROM branches WHERE workspace_id = $1 ORDER BY created_at;

-- name: FindBranchesBySeed :many
SELECT id, workspace_id, seed_id, author_id, root_leaf_id, created_at
FROM branches WHERE seed_id = $1 ORDER BY created_at;

-- name: FindBranchesByAuthor :many
SELECT id, workspace_id, seed_id, author_id, root_leaf_id, created_at
FROM branches WHERE workspace_id = $1 AND author_id = $2 ORDER BY created_at;

-- name: UpdateBranch :execresult
UPDATE branches SET root_leaf_id = $2 WHERE id = $1;

-- Connection queries

-- name: CreateConnection :exec
INSERT INTO connections (id, workspace_id, author_id, leaf_ids, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: FindConnectionByID :one
SELECT id, workspace_id, author_id, leaf_ids, created_at
FROM connections WHERE id = $1;

-- name: FindConnectionsByWorkspace :many
SELECT id, workspace_id, author_id, leaf_ids, created_at
FROM connections WHERE workspace_id = $1 ORDER BY created_at;

-- name: FindConnectionsByLeaf :many
SELECT id, workspace_id, author_id, leaf_ids, created_at
FROM connections WHERE sqlc.arg('leaf_id')::text = ANY(leaf_ids) ORDER BY created_at;
