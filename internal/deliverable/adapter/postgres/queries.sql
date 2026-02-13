-- name: CreateDeliverable :exec
INSERT INTO deliverables (id, workspace_id, format, content, source_leaf_ids, version, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: FindDeliverableByID :one
SELECT id, workspace_id, format, content, source_leaf_ids, version, created_at, updated_at
FROM deliverables WHERE id = $1;

-- name: FindDeliverablesByWorkspace :many
SELECT id, workspace_id, format, content, source_leaf_ids, version, created_at, updated_at
FROM deliverables WHERE workspace_id = $1 ORDER BY created_at;

-- name: UpdateDeliverable :execresult
UPDATE deliverables
SET format = $2, content = $3, source_leaf_ids = $4, version = $5, updated_at = $6
WHERE id = $1;
