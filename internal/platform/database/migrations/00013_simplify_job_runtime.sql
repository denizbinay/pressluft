-- +goose Up
-- started_at, finished_at, and timeout_at are already part of the base jobs
-- schema. Keep this migration focused on removing no-longer-supported runtime
-- tables so fresh SQLite setups and upgrades use the same compatible SQL.

DROP INDEX IF EXISTS idx_job_steps_job_step;
DROP INDEX IF EXISTS idx_job_steps_job_id;
DROP TABLE IF EXISTS job_steps;

DROP INDEX IF EXISTS idx_job_checkpoints_job_key;
DROP TABLE IF EXISTS job_checkpoints;

-- +goose Down
CREATE TABLE IF NOT EXISTS job_steps (
    id          INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id      INTEGER NOT NULL,
    step_key    TEXT    NOT NULL,
    status      TEXT    NOT NULL,
    started_at  TEXT,
    finished_at TEXT,
    details     TEXT,
    created_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at  TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE INDEX IF NOT EXISTS idx_job_steps_job_id ON job_steps(job_id);
CREATE UNIQUE INDEX IF NOT EXISTS idx_job_steps_job_step ON job_steps(job_id, step_key);

CREATE TABLE IF NOT EXISTS job_checkpoints (
    id             INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id         INTEGER NOT NULL,
    checkpoint_key TEXT    NOT NULL,
    resume_token   TEXT,
    data           TEXT,
    created_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at     TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_job_checkpoints_job_key ON job_checkpoints(job_id, checkpoint_key);

-- IRREVERSIBLE: SQLite does not support DROP COLUMN. started_at, finished_at, and timeout_at remain.
