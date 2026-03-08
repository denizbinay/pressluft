-- +goose Up
-- no-op: legacy jobs-table compatibility is reconciled in application startup so
-- older developer databases are repaired safely without relying on
-- ALTER TABLE ... ADD COLUMN IF NOT EXISTS support across SQLite variants.

-- +goose Down
-- no-op.
