-- +goose Up
ALTER TABLE sites ADD COLUMN deployment_state TEXT NOT NULL DEFAULT 'pending';
ALTER TABLE sites ADD COLUMN deployment_status_message TEXT;
ALTER TABLE sites ADD COLUMN last_deploy_job_id TEXT;
ALTER TABLE sites ADD COLUMN last_deployed_at TEXT;

CREATE INDEX IF NOT EXISTS idx_sites_deployment_state ON sites(deployment_state);

-- +goose Down
DROP INDEX IF EXISTS idx_sites_deployment_state;

ALTER TABLE sites DROP COLUMN last_deployed_at;
ALTER TABLE sites DROP COLUMN last_deploy_job_id;
ALTER TABLE sites DROP COLUMN deployment_status_message;
ALTER TABLE sites DROP COLUMN deployment_state;
