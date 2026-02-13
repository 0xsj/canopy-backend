CREATE TABLE sessions (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL,
    user_id        TEXT NOT NULL,
    seed_id        TEXT NOT NULL,
    parent_leaf_id TEXT,
    session_type   TEXT NOT NULL CHECK (session_type IN ('exploration', 'shaping', 'synthesis', 'digest')),
    status         TEXT NOT NULL CHECK (status IN ('active', 'checkpoint', 'completed', 'abandoned')),
    messages       JSONB NOT NULL DEFAULT '[]',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sessions_workspace_id ON sessions (workspace_id);
CREATE INDEX idx_sessions_user_id ON sessions (user_id);
CREATE INDEX idx_sessions_seed_id ON sessions (seed_id);
