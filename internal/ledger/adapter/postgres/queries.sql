-- name: AppendSystemEntry :exec
INSERT INTO system_entries (id, event_subject, event_data, source_context, created_at)
VALUES ($1, $2, $3, $4, $5);

-- name: AppendDomainEntry :exec
INSERT INTO domain_entries (id, actor_id, action, resource_type, resource_id, org_id, workspace_id, metadata, created_at)
VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9);

-- name: FindSystemEntries :many
SELECT id, event_subject, event_data, source_context, created_at
FROM system_entries
WHERE (sqlc.narg('source_context')::text IS NULL OR source_context = sqlc.narg('source_context'))
  AND (sqlc.narg('event_subject')::text IS NULL OR event_subject = sqlc.narg('event_subject'))
  AND (sqlc.narg('after')::timestamptz IS NULL OR created_at > sqlc.narg('after'))
  AND (sqlc.narg('before')::timestamptz IS NULL OR created_at < sqlc.narg('before'))
ORDER BY created_at DESC
LIMIT sqlc.arg('query_limit');

-- name: FindDomainEntries :many
SELECT id, actor_id, action, resource_type, resource_id, org_id, workspace_id, metadata, created_at
FROM domain_entries
WHERE (sqlc.narg('actor_id')::text IS NULL OR actor_id = sqlc.narg('actor_id'))
  AND (sqlc.narg('action')::text IS NULL OR action = sqlc.narg('action'))
  AND (sqlc.narg('resource_type')::text IS NULL OR resource_type = sqlc.narg('resource_type'))
  AND (sqlc.narg('resource_id')::text IS NULL OR resource_id = sqlc.narg('resource_id'))
  AND (sqlc.narg('org_id')::text IS NULL OR org_id = sqlc.narg('org_id'))
  AND (sqlc.narg('workspace_id')::text IS NULL OR workspace_id = sqlc.narg('workspace_id'))
  AND (sqlc.narg('after')::timestamptz IS NULL OR created_at > sqlc.narg('after'))
  AND (sqlc.narg('before')::timestamptz IS NULL OR created_at < sqlc.narg('before'))
ORDER BY created_at DESC
LIMIT sqlc.arg('query_limit');
