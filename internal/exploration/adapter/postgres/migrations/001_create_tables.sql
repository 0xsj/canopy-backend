CREATE TABLE branches (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    seed_id      TEXT NOT NULL,
    author_id    TEXT NOT NULL,
    root_leaf_id TEXT,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_branches_workspace_id ON branches (workspace_id);
CREATE INDEX idx_branches_seed_id ON branches (seed_id);

CREATE TABLE leaves (
    id             TEXT PRIMARY KEY,
    workspace_id   TEXT NOT NULL,
    seed_id        TEXT NOT NULL,
    branch_id      TEXT NOT NULL REFERENCES branches (id) ON DELETE CASCADE,
    author_id      TEXT NOT NULL,
    parent_leaf_id TEXT REFERENCES leaves (id),
    title          TEXT NOT NULL,
    summary        TEXT NOT NULL,
    key_points     TEXT[],
    open_questions TEXT[],
    tags           TEXT[],
    layer          TEXT NOT NULL CHECK (layer IN ('understory', 'canopy')),
    sources        JSONB,
    metadata       JSONB NOT NULL DEFAULT '{}',
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_leaves_workspace_id ON leaves (workspace_id);
CREATE INDEX idx_leaves_seed_id ON leaves (seed_id);
CREATE INDEX idx_leaves_branch_id ON leaves (branch_id);
CREATE INDEX idx_leaves_author_id ON leaves (author_id);
CREATE INDEX idx_leaves_parent_leaf_id ON leaves (parent_leaf_id) WHERE parent_leaf_id IS NOT NULL;

-- Add FK from branches to leaves after leaves table exists.
ALTER TABLE branches
    ADD CONSTRAINT fk_branches_root_leaf_id
    FOREIGN KEY (root_leaf_id) REFERENCES leaves (id);

CREATE TABLE connections (
    id           TEXT PRIMARY KEY,
    workspace_id TEXT NOT NULL,
    author_id    TEXT NOT NULL,
    leaf_ids     TEXT[] NOT NULL,
    created_at   TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_connections_workspace_id ON connections (workspace_id);
