-- +goose Up
ALTER TABLE jobs ADD COLUMN command_id TEXT;
CREATE UNIQUE INDEX idx_jobs_command_id ON jobs(command_id) WHERE command_id IS NOT NULL;

-- +goose Down
DROP INDEX IF EXISTS idx_jobs_command_id;
-- IRREVERSIBLE: SQLite does not support DROP COLUMN. To reverse, recreate table without column.
-- Manual steps required:
-- 1. CREATE TABLE jobs_backup AS SELECT id, server_id, type, status, params, output, created_at, updated_at FROM jobs;
-- 2. DROP TABLE jobs;
-- 3. ALTER TABLE jobs_backup RENAME TO jobs;
-- Recreate indexes and foreign keys as needed.
