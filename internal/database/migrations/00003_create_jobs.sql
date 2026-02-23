-- +goose Up
CREATE TABLE IF NOT EXISTS jobs (
    id           INTEGER PRIMARY KEY AUTOINCREMENT,
    server_id    INTEGER,
    kind         TEXT    NOT NULL,
    status       TEXT    NOT NULL,
    current_step TEXT    NOT NULL DEFAULT '',
    retry_count  INTEGER NOT NULL DEFAULT 0,
    last_error   TEXT,
    created_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    updated_at   TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (server_id) REFERENCES servers(id)
);

CREATE INDEX IF NOT EXISTS idx_jobs_status ON jobs(status);
CREATE INDEX IF NOT EXISTS idx_jobs_server_id ON jobs(server_id);
CREATE INDEX IF NOT EXISTS idx_jobs_created_at ON jobs(created_at DESC);

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

CREATE TABLE IF NOT EXISTS job_events (
    id         INTEGER PRIMARY KEY AUTOINCREMENT,
    job_id     INTEGER NOT NULL,
    seq        INTEGER NOT NULL,
    event_type TEXT    NOT NULL,
    level      TEXT    NOT NULL,
    step_key   TEXT,
    status     TEXT,
    message    TEXT    NOT NULL,
    payload    TEXT,
    created_at TEXT    NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
    FOREIGN KEY (job_id) REFERENCES jobs(id) ON DELETE CASCADE
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_job_events_job_seq ON job_events(job_id, seq);
CREATE INDEX IF NOT EXISTS idx_job_events_job_id ON job_events(job_id);

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

-- +goose Down
DROP INDEX IF EXISTS idx_job_checkpoints_job_key;
DROP TABLE IF EXISTS job_checkpoints;

DROP INDEX IF EXISTS idx_job_events_job_id;
DROP INDEX IF EXISTS idx_job_events_job_seq;
DROP TABLE IF EXISTS job_events;

DROP INDEX IF EXISTS idx_job_steps_job_step;
DROP INDEX IF EXISTS idx_job_steps_job_id;
DROP TABLE IF EXISTS job_steps;

DROP INDEX IF EXISTS idx_jobs_created_at;
DROP INDEX IF EXISTS idx_jobs_server_id;
DROP INDEX IF EXISTS idx_jobs_status;
DROP TABLE IF EXISTS jobs;
