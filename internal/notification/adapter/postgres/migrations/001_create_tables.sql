CREATE TABLE notifications (
    id            TEXT PRIMARY KEY,
    user_id       TEXT NOT NULL,
    channel       TEXT NOT NULL CHECK (channel IN ('in_app', 'push', 'email')),
    title         TEXT NOT NULL,
    body          TEXT,
    resource_type TEXT,
    resource_id   TEXT,
    workspace_id  TEXT,
    status        TEXT NOT NULL CHECK (status IN ('pending', 'delivered', 'read')),
    created_at    TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at    TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_notifications_user_id ON notifications (user_id);
CREATE INDEX idx_notifications_workspace_id ON notifications (workspace_id) WHERE workspace_id IS NOT NULL;
CREATE INDEX idx_notifications_status ON notifications (status);

CREATE TABLE subscriptions (
    user_id          TEXT NOT NULL,
    workspace_id     TEXT NOT NULL,
    channels         TEXT[] NOT NULL,
    digest_frequency TEXT NOT NULL CHECK (digest_frequency IN ('daily', 'per_session', 'weekly')),
    created_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at       TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    PRIMARY KEY (user_id, workspace_id)
);
