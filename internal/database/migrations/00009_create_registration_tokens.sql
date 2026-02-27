-- +goose Up
CREATE TABLE IF NOT EXISTS registration_tokens (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id INTEGER NOT NULL REFERENCES servers(id),
    token_hash TEXT UNIQUE NOT NULL,
    created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at TEXT NOT NULL,
    consumed_at TEXT
);

CREATE INDEX idx_registration_tokens_server_id ON registration_tokens(server_id);
CREATE INDEX idx_registration_tokens_token_hash ON registration_tokens(token_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_registration_tokens_token_hash;
DROP INDEX IF EXISTS idx_registration_tokens_server_id;
DROP TABLE IF EXISTS registration_tokens;
