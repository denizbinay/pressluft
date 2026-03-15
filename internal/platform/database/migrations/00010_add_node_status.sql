-- +goose Up
ALTER TABLE servers ADD COLUMN node_status TEXT DEFAULT 'unknown';
ALTER TABLE servers ADD COLUMN node_last_seen TEXT;
ALTER TABLE servers ADD COLUMN node_version TEXT;

-- +goose Down
-- IRREVERSIBLE: SQLite does not support DROP COLUMN. To reverse, recreate table without columns.
-- Manual steps required:
-- 1. CREATE TABLE servers_backup AS SELECT id, name, ... (all columns except new ones) FROM servers;
-- 2. DROP TABLE servers;
-- 3. ALTER TABLE servers_backup RENAME TO servers;
-- Recreate indexes and foreign keys as needed.
