CREATE TABLE deliverables (
    id              TEXT PRIMARY KEY,
    workspace_id    TEXT NOT NULL,
    format          TEXT NOT NULL CHECK (format IN ('markdown', 'pdf', 'html')),
    content         TEXT NOT NULL,
    source_leaf_ids TEXT[] NOT NULL,
    version         INTEGER NOT NULL DEFAULT 1,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_deliverables_workspace_id ON deliverables (workspace_id);
