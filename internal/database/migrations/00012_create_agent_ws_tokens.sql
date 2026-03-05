-- +goose Up
CREATE TABLE IF NOT EXISTS agent_ws_tokens (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id    INTEGER NOT NULL,
    token_hash   TEXT    NOT NULL UNIQUE,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    expires_at   TEXT    NOT NULL,
    revoked_at   TEXT,
    last_used_at TEXT,
    FOREIGN KEY (server_id) REFERENCES servers(id)
);

CREATE INDEX IF NOT EXISTS idx_agent_ws_tokens_server_id ON agent_ws_tokens(server_id);
CREATE INDEX IF NOT EXISTS idx_agent_ws_tokens_token_hash ON agent_ws_tokens(token_hash);

-- +goose Down
DROP INDEX IF EXISTS idx_agent_ws_tokens_token_hash;
DROP INDEX IF EXISTS idx_agent_ws_tokens_server_id;
DROP TABLE IF EXISTS agent_ws_tokens;
