-- +goose Up
ALTER TABLE jobs ADD COLUMN payload TEXT;

-- +goose Down
-- SQLite does not support DROP COLUMN for payload.
