package backups

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

func TestCreateQueuesBackupCreateJobWithPendingBackup(t *testing.T) {
	t.Parallel()

	db := openBackupsTestDB(t)
	seedEnvironment(t, db)
	svc := NewService(db)

	result, err := svc.Create(context.Background(), CreateInput{EnvironmentID: "env-1", BackupScope: "full"})
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}

	if result.BackupID == "" || result.JobID == "" {
		t.Fatalf("expected non-empty create result ids")
	}

	assertString(t, db, "SELECT status FROM backups WHERE id = ?", "pending", result.BackupID)
	assertString(t, db, "SELECT backup_scope FROM backups WHERE id = ?", "full", result.BackupID)
	assertString(t, db, "SELECT storage_type FROM backups WHERE id = ?", "s3", result.BackupID)
	assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "backup_create", result.JobID)
	assertString(t, db, "SELECT status FROM jobs WHERE id = ?", "queued", result.JobID)
	assertString(t, db, "SELECT environment_id FROM jobs WHERE id = ?", "env-1", result.JobID)
}

func TestCreateReturnsConcurrencyConflictWhenSiteBusy(t *testing.T) {
	t.Parallel()

	db := openBackupsTestDB(t)
	seedEnvironment(t, db)
	seedQueuedMutation(t, db)
	svc := NewService(db)

	_, err := svc.Create(context.Background(), CreateInput{EnvironmentID: "env-1", BackupScope: "db"})
	if !errors.Is(err, jobs.ErrConcurrencyConflict) {
		t.Fatalf("expected concurrency conflict, got %v", err)
	}
}

func TestListByEnvironmentReturnsRetentionMetadata(t *testing.T) {
	t.Parallel()

	db := openBackupsTestDB(t)
	seedEnvironment(t, db)
	seedBackupRows(t, db)
	svc := NewService(db)

	list, err := svc.ListByEnvironment(context.Background(), "env-1")
	if err != nil {
		t.Fatalf("list backups: %v", err)
	}
	if len(list) != 2 {
		t.Fatalf("expected two backups, got %d", len(list))
	}
	if list[0].RetentionUntil == "" || list[0].StoragePath == "" {
		t.Fatalf("expected retention metadata fields in list response")
	}
}

func TestBackupLifecycleTransitions(t *testing.T) {
	t.Parallel()

	db := openBackupsTestDB(t)
	seedEnvironment(t, db)
	svc := NewService(db)

	result, err := svc.Create(context.Background(), CreateInput{EnvironmentID: "env-1", BackupScope: "files"})
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}

	if err := svc.MarkRunning(context.Background(), result.BackupID); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	assertString(t, db, "SELECT status FROM backups WHERE id = ?", "running", result.BackupID)

	if err := svc.MarkCompleted(context.Background(), result.BackupID, "sha256:abc", 42); err != nil {
		t.Fatalf("mark completed: %v", err)
	}
	assertString(t, db, "SELECT status FROM backups WHERE id = ?", "completed", result.BackupID)

	if err := svc.MarkExpired(context.Background(), result.BackupID); err != nil {
		t.Fatalf("mark expired: %v", err)
	}
	assertString(t, db, "SELECT status FROM backups WHERE id = ?", "expired", result.BackupID)
}

func TestBackupMarkFailedTransitionsFromRunning(t *testing.T) {
	t.Parallel()

	db := openBackupsTestDB(t)
	seedEnvironment(t, db)
	svc := NewService(db)

	result, err := svc.Create(context.Background(), CreateInput{EnvironmentID: "env-1", BackupScope: "db"})
	if err != nil {
		t.Fatalf("create backup: %v", err)
	}
	if err := svc.MarkRunning(context.Background(), result.BackupID); err != nil {
		t.Fatalf("mark running: %v", err)
	}
	if err := svc.MarkFailed(context.Background(), result.BackupID); err != nil {
		t.Fatalf("mark failed: %v", err)
	}
	assertString(t, db, "SELECT status FROM backups WHERE id = ?", "failed", result.BackupID)
}

func openBackupsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "backups-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE environments (
			id TEXT PRIMARY KEY,
			site_id TEXT NOT NULL,
			node_id TEXT NOT NULL
		);
		CREATE TABLE backups (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			backup_scope TEXT NOT NULL,
			status TEXT NOT NULL,
			storage_type TEXT NOT NULL,
			storage_path TEXT NOT NULL,
			retention_until TEXT NOT NULL,
			checksum TEXT NULL,
			size_bytes INTEGER NULL,
			created_at TEXT NOT NULL,
			completed_at TEXT NULL
		);
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
		t.Fatalf("create schema: %v", err)
	}

	return db
}

func seedEnvironment(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO environments (id, site_id, node_id)
		VALUES ('env-1', 'site-1', 'node-1')
	`); err != nil {
		t.Fatalf("seed environment: %v", err)
	}
}

func seedQueuedMutation(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES ('job-existing', 'env_create', 'queued', 'site-1', 'env-1', 'node-1', '{}', 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("seed queued mutation: %v", err)
	}
}

func seedBackupRows(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO backups (id, environment_id, backup_scope, status, storage_type, storage_path, retention_until, checksum, size_bytes, created_at, completed_at)
		VALUES
		  ('backup-2', 'env-1', 'full', 'completed', 's3', 's3://pressluft/backups/env-1/backup-2.tar.zst', datetime('now', '+29 day'), 'sha256:abc', 200, datetime('now'), datetime('now')),
		  ('backup-1', 'env-1', 'db', 'pending', 's3', 's3://pressluft/backups/env-1/backup-1.tar.zst', datetime('now', '+30 day'), NULL, NULL, datetime('now', '-1 day'), NULL)
	`); err != nil {
		t.Fatalf("seed backups: %v", err)
	}
}

func assertString(t *testing.T, db *sql.DB, query, expected, arg string) {
	t.Helper()

	var got string
	if err := db.QueryRow(query, arg).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if got != expected {
		t.Fatalf("unexpected value for %q: got %q want %q", query, got, expected)
	}
}
