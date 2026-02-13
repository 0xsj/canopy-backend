CREATE TABLE seeds (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    author_id    TEXT NOT NULL,
    title        TEXT NOT NULL,
    description  TEXT,
    constraints  JSONB NOT NULL DEFAULT '{}',
    tags         TEXT[],
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_seeds_workspace_id ON seeds (workspace_id);
CREATE INDEX idx_seeds_author_id ON seeds (author_id);
