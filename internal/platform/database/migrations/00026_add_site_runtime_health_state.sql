-- +goose Up
ALTER TABLE sites ADD COLUMN runtime_health_state TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE sites ADD COLUMN runtime_health_status_message TEXT;
ALTER TABLE sites ADD COLUMN last_health_check_at TEXT;

CREATE INDEX IF NOT EXISTS idx_sites_runtime_health_state ON sites(runtime_health_state);

-- +goose Down
DROP INDEX IF EXISTS idx_sites_runtime_health_state;

ALTER TABLE sites DROP COLUMN last_health_check_at;
ALTER TABLE sites DROP COLUMN runtime_health_status_message;
ALTER TABLE sites DROP COLUMN runtime_health_state;
