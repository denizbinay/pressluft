-- +goose Up
ALTER TABLE servers ADD COLUMN setup_state TEXT NOT NULL DEFAULT 'not_started';
ALTER TABLE servers ADD COLUMN setup_last_error TEXT;

-- +goose Down
-- SQLite does not support dropping columns without rebuilding the table.
