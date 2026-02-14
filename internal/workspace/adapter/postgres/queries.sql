-- Workspace queries

-- name: CreateWorkspace :exec
INSERT INTO workspaces (id, org_id, team_id, name, description, phase, lore_keeper_mode, lore_keeper_id, configuration, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9, $10, $11);

-- name: FindWorkspaceByID :one
SELECT id, org_id, team_id, name, description, phase, lore_keeper_mode, lore_keeper_id, configuration, created_at, updated_at
FROM workspaces WHERE id = $1;

-- name: FindWorkspacesByOrg :many
SELECT id, org_id, team_id, name, description, phase, lore_keeper_mode, lore_keeper_id, configuration, created_at, updated_at
FROM workspaces WHERE org_id = $1 ORDER BY created_at;

-- name: FindWorkspacesByTeam :many
SELECT id, org_id, team_id, name, description, phase, lore_keeper_mode, lore_keeper_id, configuration, created_at, updated_at
FROM workspaces WHERE team_id = $1 ORDER BY created_at;

-- name: UpdateWorkspace :execresult
UPDATE workspaces
SET name = $2, description = $3, phase = $4, lore_keeper_mode = $5,
    lore_keeper_id = $6, configuration = $7, updated_at = $8
WHERE id = $1;

-- Workspace member queries

-- name: AddWorkspaceMember :exec
INSERT INTO workspace_members (workspace_id, user_id, role, joined_at)
VALUES ($1, $2, $3, $4);

-- name: RemoveWorkspaceMember :execresult
DELETE FROM workspace_members WHERE workspace_id = $1 AND user_id = $2;

-- name: FindWorkspaceMembersByWorkspace :many
SELECT workspace_id, user_id, role, joined_at
FROM workspace_members WHERE workspace_id = $1 ORDER BY joined_at;

-- name: FindWorkspaceMembersByUser :many
SELECT wm.workspace_id, wm.user_id, wm.role, wm.joined_at
FROM workspace_members wm
JOIN workspaces w ON w.id = wm.workspace_id
WHERE w.org_id = $1 AND wm.user_id = $2
ORDER BY wm.joined_at;

-- name: FindWorkspaceMember :one
SELECT workspace_id, user_id, role, joined_at
FROM workspace_members WHERE workspace_id = $1 AND user_id = $2;

-- name: UpdateWorkspaceMemberRole :execresult
UPDATE workspace_members SET role = $3 WHERE workspace_id = $1 AND user_id = $2;

-- LLM config queries

-- name: UpsertLLMConfig :exec
INSERT INTO llm_configs (workspace_id, provider, model, api_key_enc, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (workspace_id)
DO UPDATE SET provider = EXCLUDED.provider, model = EXCLUDED.model,
             api_key_enc = EXCLUDED.api_key_enc, updated_at = EXCLUDED.updated_at;

-- name: FindLLMConfigByWorkspace :one
SELECT workspace_id, provider, model, api_key_enc, created_at, updated_at
FROM llm_configs WHERE workspace_id = $1;

-- name: DeleteLLMConfig :execresult
DELETE FROM llm_configs WHERE workspace_id = $1;
