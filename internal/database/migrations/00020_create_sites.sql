-- +goose Up
CREATE TABLE IF NOT EXISTS sites (
    id                TEXT PRIMARY KEY,
    server_id         TEXT    NOT NULL,
    name              TEXT    NOT NULL,
    primary_domain    TEXT,
    status            TEXT    NOT NULL DEFAULT 'draft',
    wordpress_path    TEXT,
    php_version       TEXT,
    wordpress_version TEXT,
    created_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at        TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (server_id) REFERENCES servers(id)
);

CREATE INDEX IF NOT EXISTS idx_sites_server_id ON sites(server_id);
CREATE INDEX IF NOT EXISTS idx_sites_status ON sites(status);
CREATE INDEX IF NOT EXISTS idx_sites_created_at ON sites(created_at DESC);

-- +goose Down
DROP INDEX IF EXISTS idx_sites_created_at;
DROP INDEX IF EXISTS idx_sites_status;
DROP INDEX IF EXISTS idx_sites_server_id;
DROP TABLE IF EXISTS sites;
