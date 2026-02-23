-- +goose Up
CREATE TABLE IF NOT EXISTS providers (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    type       TEXT    NOT NULL,
    name       TEXT    NOT NULL,
    api_token  TEXT    NOT NULL,
    status     TEXT    NOT NULL DEFAULT 'active',
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_providers_type_name ON providers(type, name);

-- +goose Down
DROP INDEX IF EXISTS idx_providers_type_name;
DROP TABLE IF EXISTS providers;
