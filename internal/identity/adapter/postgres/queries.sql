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

-- name: UpsertUserLLMConfig :exec
INSERT INTO user_llm_configs (user_id, provider, model, api_key_enc, created_at, updated_at)
VALUES ($1, $2, $3, $4, $5, $6)
ON CONFLICT (user_id) DO UPDATE
SET provider = EXCLUDED.provider, model = EXCLUDED.model,
    api_key_enc = EXCLUDED.api_key_enc, updated_at = EXCLUDED.updated_at;

-- name: FindUserLLMConfigByUser :one
SELECT user_id, provider, model, api_key_enc, created_at, updated_at
FROM user_llm_configs WHERE user_id = $1;

-- name: DeleteUserLLMConfig :execresult
DELETE FROM user_llm_configs WHERE user_id = $1;
