package jobs

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"pressluft/internal/store"
)

func TestServiceListAndGet(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	svc := NewService(db)

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES
			('job-1', 'site_create', 'queued', 'site-1', NULL, 'node-1', '{}', 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '2026-02-21T10:00:00Z', '2026-02-21T10:00:00Z'),
			('job-2', 'env_deploy', 'failed', NULL, 'env-1', 'node-1', '{}', 3, 3, NULL, NULL, NULL, '2026-02-21T10:01:00Z', '2026-02-21T10:02:00Z', 'ENV_DEPLOY_FAILED', 'failed', '2026-02-21T10:01:00Z', '2026-02-21T10:02:00Z')
	`); err != nil {
		t.Fatalf("seed jobs: %v", err)
	}

	jobsList, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobsList) != 2 {
		t.Fatalf("expected 2 jobs, got %d", len(jobsList))
	}

	job, err := svc.Get(context.Background(), "job-2")
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if job.ErrorCode == nil || *job.ErrorCode != "ENV_DEPLOY_FAILED" {
		t.Fatalf("unexpected error code: %v", job.ErrorCode)
	}
}

func TestServiceGetReturnsNotFound(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	svc := NewService(db)

	_, err := svc.Get(context.Background(), "missing")
	if !errors.Is(err, ErrNotFound) {
		t.Fatalf("expected ErrNotFound, got %v", err)
	}
}

func TestServiceCancelQueuedAndRunningJobs(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	svc := NewService(db)

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES
			('job-queued', 'site_create', 'queued', 'site-1', NULL, 'node-1', '{}', 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, '2026-02-21T10:00:00Z', '2026-02-21T10:00:00Z'),
			('job-running', 'env_deploy', 'running', 'site-1', 'env-1', 'node-1', '{}', 1, 3, NULL, '2026-02-21T10:01:00Z', 'worker-1', '2026-02-21T10:01:00Z', NULL, NULL, NULL, '2026-02-21T10:01:00Z', '2026-02-21T10:01:00Z')
	`); err != nil {
		t.Fatalf("seed jobs: %v", err)
	}

	queued, err := svc.Cancel(context.Background(), "job-queued")
	if err != nil {
		t.Fatalf("cancel queued job: %v", err)
	}
	if queued.Status != "cancelled" {
		t.Fatalf("expected queued job cancelled, got %s", queued.Status)
	}

	running, err := svc.Cancel(context.Background(), "job-running")
	if err != nil {
		t.Fatalf("cancel running job: %v", err)
	}
	if running.Status != "cancelled" {
		t.Fatalf("expected running job cancelled, got %s", running.Status)
	}
}

func TestServiceCancelRejectsNonCancellableState(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	svc := NewService(db)

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES ('job-done', 'env_deploy', 'succeeded', 'site-1', 'env-1', 'node-1', '{}', 1, 3, NULL, NULL, NULL, '2026-02-21T10:01:00Z', '2026-02-21T10:02:00Z', NULL, NULL, '2026-02-21T10:01:00Z', '2026-02-21T10:02:00Z')
	`); err != nil {
		t.Fatalf("seed completed job: %v", err)
	}

	_, err := svc.Cancel(context.Background(), "job-done")
	if !errors.Is(err, ErrNotCancellable) {
		t.Fatalf("expected ErrNotCancellable, got %v", err)
	}
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "jobs-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE jobs (
			id TEXT PRIMARY KEY,
			job_type TEXT NOT NULL,
			status TEXT NOT NULL,
			site_id TEXT NULL,
			environment_id TEXT NULL,
			node_id TEXT NULL,
			payload_json TEXT NOT NULL,
			attempt_count INTEGER NOT NULL,
			max_attempts INTEGER NOT NULL,
			run_after TEXT NULL,
			locked_at TEXT NULL,
			locked_by TEXT NULL,
			started_at TEXT NULL,
			finished_at TEXT NULL,
			error_code TEXT NULL,
			error_message TEXT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("create jobs table: %v", err)
	}

	return db
}
