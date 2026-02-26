-- +goose Up
CREATE TABLE IF NOT EXISTS server_keys (
    server_id             INTEGER PRIMARY KEY,
    public_key            TEXT    NOT NULL,
    private_key_encrypted TEXT    NOT NULL,
    encryption_key_id     TEXT    NOT NULL,
    created_at            TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    rotated_at            TEXT,
    FOREIGN KEY (server_id) REFERENCES servers(id) ON DELETE CASCADE
);

-- +goose Down
DROP TABLE IF EXISTS server_keys;
