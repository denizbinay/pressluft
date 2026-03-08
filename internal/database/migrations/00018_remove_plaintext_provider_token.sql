-- +goose Up
-- Remove plaintext api_token column, keep only encrypted fields.
-- SQLite does not support DROP COLUMN with NOT NULL constraints,
-- so we rebuild the table.

CREATE TABLE providers_new (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    type       TEXT    NOT NULL,
    name       TEXT    NOT NULL,
    api_token_encrypted TEXT NOT NULL,
    api_token_key_id TEXT NOT NULL,
    api_token_version INTEGER NOT NULL DEFAULT 0,
    status     TEXT    NOT NULL DEFAULT 'active',
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
);

INSERT INTO providers_new (id, type, name, api_token_encrypted, api_token_key_id, api_token_version, status, created_at, updated_at)
SELECT id, type, name, api_token_encrypted, api_token_key_id, api_token_version, status, created_at, updated_at
FROM providers;

DROP TABLE providers;
ALTER TABLE providers_new RENAME TO providers;
CREATE UNIQUE INDEX IF NOT EXISTS idx_providers_type_name ON providers(type, name);

-- +goose Down
-- Cannot reverse: data loss (plaintext tokens were discarded).
-- This is acceptable for development-only database.
