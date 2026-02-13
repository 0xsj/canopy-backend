CREATE TABLE signals (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    leaf_id      TEXT NOT NULL,
    user_id      TEXT NOT NULL,
    signal_type  TEXT NOT NULL CHECK (signal_type IN ('upvote', 'pin', 'flag')),
    annotation   TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    UNIQUE (leaf_id, user_id, signal_type)
);

CREATE INDEX idx_signals_leaf_id ON signals (leaf_id);
CREATE INDEX idx_signals_user_id ON signals (user_id);
CREATE INDEX idx_signals_workspace_user ON signals (workspace_id, user_id);

CREATE TABLE checkpoints (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    leaf_ids     TEXT[] NOT NULL,
    status       TEXT NOT NULL CHECK (status IN ('open', 'resolved')),
    signals      JSONB NOT NULL DEFAULT '[]',
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_checkpoints_workspace_id ON checkpoints (workspace_id);
