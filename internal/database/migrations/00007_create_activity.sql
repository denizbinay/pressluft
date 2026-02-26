-- +goose Up
CREATE TABLE IF NOT EXISTS activity (
    id                   INTEGER PRIMARY KEY AUTOINCREMENT,

    -- Event classification
    event_type           TEXT NOT NULL,
    category             TEXT NOT NULL,
    level                TEXT NOT NULL,

    -- Polymorphic resource reference
    resource_type        TEXT,
    resource_id          INTEGER,

    -- Secondary resource (e.g., job belongs to server)
    parent_resource_type TEXT,
    parent_resource_id   INTEGER,

    -- Actor tracking
    actor_type           TEXT NOT NULL,
    actor_id             TEXT,

    -- Content
    title                TEXT NOT NULL,
    message              TEXT,
    payload              TEXT,

    -- Notification projection
    requires_attention   INTEGER NOT NULL DEFAULT 0,
    read_at              TEXT,

    created_at           TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE INDEX idx_activity_created ON activity(created_at DESC);
CREATE INDEX idx_activity_category ON activity(category, created_at DESC);
CREATE INDEX idx_activity_resource ON activity(resource_type, resource_id, created_at DESC);
CREATE INDEX idx_activity_parent ON activity(parent_resource_type, parent_resource_id, created_at DESC);
CREATE INDEX idx_activity_attention ON activity(requires_attention, read_at, created_at DESC);
CREATE INDEX idx_activity_event_type ON activity(event_type);

-- +goose Down
DROP INDEX IF EXISTS idx_activity_event_type;
DROP INDEX IF EXISTS idx_activity_attention;
DROP INDEX IF EXISTS idx_activity_parent;
DROP INDEX IF EXISTS idx_activity_resource;
DROP INDEX IF EXISTS idx_activity_category;
DROP INDEX IF EXISTS idx_activity_created;
DROP TABLE IF EXISTS activity;
