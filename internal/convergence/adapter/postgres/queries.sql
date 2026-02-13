-- Signal queries

-- name: CreateSignal :exec
INSERT INTO signals (id, workspace_id, leaf_id, user_id, signal_type, annotation, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: DeleteSignal :execresult
DELETE FROM signals WHERE id = $1;

-- name: FindSignalsByLeaf :many
SELECT id, workspace_id, leaf_id, user_id, signal_type, annotation, created_at
FROM signals WHERE leaf_id = $1 ORDER BY created_at;

-- name: FindSignalsByUser :many
SELECT id, workspace_id, leaf_id, user_id, signal_type, annotation, created_at
FROM signals WHERE workspace_id = $1 AND user_id = $2 ORDER BY created_at;

-- name: FindSignalByLeafAndUser :one
SELECT id, workspace_id, leaf_id, user_id, signal_type, annotation, created_at
FROM signals WHERE leaf_id = $1 AND user_id = $2;

-- name: CountSignalsByLeaf :many
SELECT signal_type, COUNT(*)::int AS count
FROM signals WHERE leaf_id = $1 GROUP BY signal_type;

-- Checkpoint queries

-- name: CreateCheckpoint :exec
INSERT INTO checkpoints (id, workspace_id, leaf_ids, status, signals, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7);

-- name: FindCheckpointByID :one
SELECT id, workspace_id, leaf_ids, status, signals, created_at, updated_at
FROM checkpoints WHERE id = $1;

-- name: FindCheckpointsByWorkspace :many
SELECT id, workspace_id, leaf_ids, status, signals, created_at, updated_at
FROM checkpoints WHERE workspace_id = $1 ORDER BY created_at;

-- name: FindOpenCheckpointsByWorkspace :many
SELECT id, workspace_id, leaf_ids, status, signals, created_at, updated_at
FROM checkpoints WHERE workspace_id = $1 AND status = 'open' ORDER BY created_at;

-- name: UpdateCheckpoint :execresult
UPDATE checkpoints SET status = $2, signals = $3, updated_at = $4
WHERE id = $1;
