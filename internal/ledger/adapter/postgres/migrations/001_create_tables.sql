CREATE TABLE system_entries (
    id             TEXT PRIMARY KEY,
    event_subject  TEXT NOT NULL,
    event_data     JSONB NOT NULL DEFAULT '{}',
    source_context TEXT NOT NULL,
    created_at     TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_system_entries_source_context ON system_entries (source_context);
CREATE INDEX idx_system_entries_created_at ON system_entries (created_at);

CREATE TABLE domain_entries (
    id            TEXT PRIMARY KEY,
    actor_id      TEXT NOT NULL,
    action        TEXT NOT NULL,
    resource_type TEXT NOT NULL,
    resource_id   TEXT NOT NULL,
    org_id        TEXT,
    workspace_id  TEXT,
    metadata      JSONB NOT NULL DEFAULT '{}',
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_domain_entries_actor_id ON domain_entries (actor_id);
CREATE INDEX idx_domain_entries_resource ON domain_entries (resource_type, resource_id);
CREATE INDEX idx_domain_entries_org_id ON domain_entries (org_id) WHERE org_id IS NOT NULL;
CREATE INDEX idx_domain_entries_workspace_id ON domain_entries (workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX idx_domain_entries_created_at ON domain_entries (created_at);
