-- +goose Up
CREATE TABLE IF NOT EXISTS servers (
    id                 INTEGER PRIMARY KEY AUTOINCREMENT,
    provider_id        INTEGER NOT NULL,
    provider_type      TEXT    NOT NULL,
    provider_server_id TEXT,
    name               TEXT    NOT NULL,
    location           TEXT    NOT NULL,
    server_type        TEXT    NOT NULL,
    image              TEXT    NOT NULL,
    profile_key        TEXT    NOT NULL,
    status             TEXT    NOT NULL DEFAULT 'provisioning',
    action_id          TEXT,
    action_status      TEXT,
    created_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at         TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (provider_id) REFERENCES providers(id)
);

CREATE INDEX IF NOT EXISTS idx_servers_provider_id ON servers(provider_id);
CREATE INDEX IF NOT EXISTS idx_servers_status ON servers(status);
CREATE INDEX IF NOT EXISTS idx_servers_created_at ON servers(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_servers_created_at;
DROP INDEX IF EXISTS idx_servers_status;
DROP INDEX IF EXISTS idx_servers_provider_id;
DROP TABLE IF EXISTS servers;
