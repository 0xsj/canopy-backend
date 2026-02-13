CREATE TABLE workspaces (
    id               TEXT PRIMARY KEY,
    org_id           TEXT NOT NULL,
    team_id          TEXT,
    name             TEXT NOT NULL,
    description      TEXT,
    phase            TEXT NOT NULL CHECK (phase IN ('floor', 'understory', 'canopy', 'emergent')),
    lore_keeper_mode TEXT NOT NULL CHECK (lore_keeper_mode IN ('human', 'ai', 'hybrid')),
    lore_keeper_id   TEXT,
    configuration    JSONB NOT NULL DEFAULT '{}',
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_workspaces_org_id ON workspaces (org_id);
CREATE INDEX idx_workspaces_team_id ON workspaces (team_id) WHERE team_id IS NOT NULL;

CREATE TABLE workspace_members (
    workspace_id TEXT NOT NULL REFERENCES workspaces (id) ON DELETE CASCADE,
    user_id      TEXT NOT NULL,
    role         TEXT NOT NULL CHECK (role IN ('participant', 'lore_keeper')),
    joined_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (workspace_id, user_id)
);

CREATE INDEX idx_workspace_members_user_id ON workspace_members (user_id);
