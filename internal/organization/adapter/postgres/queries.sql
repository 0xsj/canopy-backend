-- Organization queries

-- name: CreateOrg :exec
INSERT INTO organizations (id, name, slug, personal, owner_id, settings, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8);

-- name: FindOrgByID :one
SELECT id, name, slug, personal, owner_id, settings, created_at, updated_at
FROM organizations WHERE id = $1;

-- name: FindOrgBySlug :one
SELECT id, name, slug, personal, owner_id, settings, created_at, updated_at
FROM organizations WHERE slug = $1;

-- name: FindPersonalOrgByOwner :one
SELECT id, name, slug, personal, owner_id, settings, created_at, updated_at
FROM organizations WHERE owner_id = $1 AND personal = true;

-- name: UpdateOrg :execresult
UPDATE organizations
SET name = $2, slug = $3, settings = $4, updated_at = $5
WHERE id = $1;

-- Org member queries

-- name: AddOrgMember :exec
INSERT INTO org_members (org_id, user_id, role, joined_at)
VALUES ($1, $2, $3, $4);

-- name: RemoveOrgMember :execresult
DELETE FROM org_members WHERE org_id = $1 AND user_id = $2;

-- name: FindOrgMembersByOrg :many
SELECT org_id, user_id, role, joined_at
FROM org_members WHERE org_id = $1 ORDER BY joined_at;

-- name: FindOrgMembersByUser :many
SELECT org_id, user_id, role, joined_at
FROM org_members WHERE user_id = $1 ORDER BY joined_at;

-- name: FindOrgMember :one
SELECT org_id, user_id, role, joined_at
FROM org_members WHERE org_id = $1 AND user_id = $2;

-- name: UpdateOrgMemberRole :execresult
UPDATE org_members SET role = $3 WHERE org_id = $1 AND user_id = $2;

-- name: CountOrgMembersByRole :one
SELECT COUNT(*)::int AS count FROM org_members WHERE org_id = $1 AND role = $2;

-- Team queries

-- name: CreateTeam :exec
INSERT INTO teams (id, org_id, name, description, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6);

-- name: FindTeamByID :one
SELECT id, org_id, name, description, created_at, updated_at
FROM teams WHERE id = $1;

-- name: FindTeamsByOrg :many
SELECT id, org_id, name, description, created_at, updated_at
FROM teams WHERE org_id = $1 ORDER BY name;

-- name: UpdateTeam :execresult
UPDATE teams SET name = $2, description = $3, updated_at = $4
WHERE id = $1;

-- name: DeleteTeam :execresult
DELETE FROM teams WHERE id = $1;

-- name: AddTeamMember :exec
INSERT INTO team_members (team_id, user_id, joined_at)
VALUES ($1, $2, $3);

-- name: RemoveTeamMember :execresult
DELETE FROM team_members WHERE team_id = $1 AND user_id = $2;

-- name: FindTeamMembers :many
SELECT team_id, user_id, joined_at
FROM team_members WHERE team_id = $1 ORDER BY joined_at;

-- name: FindTeamsByUser :many
SELECT t.id, t.org_id, t.name, t.description, t.created_at, t.updated_at
FROM teams t JOIN team_members tm ON t.id = tm.team_id
WHERE t.org_id = $1 AND tm.user_id = $2 ORDER BY t.name;
