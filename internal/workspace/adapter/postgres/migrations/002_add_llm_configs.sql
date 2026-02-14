CREATE TABLE llm_configs (
    workspace_id TEXT PRIMARY KEY REFERENCES workspaces(id) ON DELETE CASCADE,
    provider     TEXT NOT NULL CHECK (provider IN ('anthropic', 'openai', 'gemini')),
    model        TEXT NOT NULL,
    api_key_enc  BYTEA NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);
