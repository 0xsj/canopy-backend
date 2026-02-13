-- name: CreateWorkflow :exec
INSERT INTO workflows (id, workspace_id, initiator_id, source_leaf_ids, result_leaf_id,
    status, failure_reason, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: FindWorkflowByID :one
SELECT id, workspace_id, initiator_id, source_leaf_ids, result_leaf_id,
    status, failure_reason, created_at, updated_at
FROM workflows WHERE id = $1;

-- name: FindWorkflowsByWorkspace :many
SELECT id, workspace_id, initiator_id, source_leaf_ids, result_leaf_id,
    status, failure_reason, created_at, updated_at
FROM workflows WHERE workspace_id = $1 ORDER BY created_at;

-- name: UpdateWorkflow :execresult
UPDATE workflows
SET status = $2, result_leaf_id = $3, failure_reason = $4, updated_at = $5
WHERE id = $1;
