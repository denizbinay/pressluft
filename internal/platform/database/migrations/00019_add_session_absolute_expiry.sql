-- +goose Up
ALTER TABLE sessions ADD COLUMN absolute_expires_at TEXT;

UPDATE sessions
SET absolute_expires_at = expires_at
WHERE absolute_expires_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_sessions_absolute_expires_at ON sessions(absolute_expires_at);

-- +goose Down
DROP INDEX IF EXISTS idx_sessions_absolute_expires_at;
