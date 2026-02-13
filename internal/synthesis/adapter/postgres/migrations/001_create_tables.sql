CREATE TABLE workflows (
    id              TEXT PRIMARY KEY,
    workspace_id    TEXT NOT NULL,
    initiator_id    TEXT NOT NULL,
    source_leaf_ids TEXT[] NOT NULL,
    result_leaf_id  TEXT,
    status          TEXT NOT NULL CHECK (status IN ('pending', 'processing', 'completed', 'failed')),
    failure_reason  TEXT,
    created_at      TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at      TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workflows_workspace_id ON workflows (workspace_id);
CREATE INDEX idx_workflows_initiator_id ON workflows (initiator_id);
