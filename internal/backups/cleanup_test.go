package backups

import (
	"context"
	"database/sql"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
	"time"

	"pressluft/internal/store"
)

func TestEnqueueExpiredCleanupQueuesOnePerSiteWithDedup(t *testing.T) {
	t.Parallel()

	db := openCleanupTestDB(t)
	seedCleanupTopology(t, db)
	seedExpiredBackups(t, db)
	seedActiveCleanupJob(t, db)

	svc := NewService(db)
	queued, err := svc.EnqueueExpiredCleanup(context.Background())
	if err != nil {
		t.Fatalf("enqueue expired cleanup: %v", err)
	}
	if queued != 1 {
		t.Fatalf("expected one queued cleanup job, got %d", queued)
	}

	var jobType string
	if err := db.QueryRow(`
		SELECT job_type
		FROM jobs
		WHERE job_type = 'backup_cleanup' AND site_id = 'site-1' AND status = 'queued'
		LIMIT 1
	`).Scan(&jobType); err != nil {
		t.Fatalf("query queued backup_cleanup for site-1: %v", err)
	}
	if jobType != "backup_cleanup" {
		t.Fatalf("unexpected queued job type: %s", jobType)
	}

	var payload string
	if err := db.QueryRow(`
		SELECT payload_json
		FROM jobs
		WHERE job_type = 'backup_cleanup' AND site_id = 'site-1' AND status = 'queued'
		LIMIT 1
	`).Scan(&payload); err != nil {
		t.Fatalf("query queued payload: %v", err)
	}
	if !strings.Contains(payload, `"backup_id":"backup-expired-a"`) {
		t.Fatalf("expected oldest expired backup queued for site-1, got payload %s", payload)
	}
}

func TestExecuteQueuedBackupCleanupMarksSuccessAndExpiresBackup(t *testing.T) {
	t.Parallel()

	db := openCleanupTestDB(t)
	seedCleanupTopology(t, db)
	seedBackupForExecution(t, db, "backup-run", "completed", -2)
	seedQueuedCleanupExecutionJob(t, db, "job-cleanup", "backup-run", 0)

	executed, err := ExecuteQueuedBackupCleanup(context.Background(), db, stubCleanupRunner{output: "ok"}, "ansible/playbooks/backup-cleanup.yml")
	if err != nil {
		t.Fatalf("execute backup cleanup: %v", err)
	}
	if !executed {
		t.Fatalf("expected cleanup job execution")
	}

	assertCleanupString(t, db, "SELECT status FROM jobs WHERE id = ?", "succeeded", "job-cleanup")
	assertCleanupString(t, db, "SELECT status FROM backups WHERE id = ?", "expired", "backup-run")
}

func TestExecuteQueuedBackupCleanupRetriesOnRetryableAnsibleFailure(t *testing.T) {
	t.Parallel()

	db := openCleanupTestDB(t)
	seedCleanupTopology(t, db)
	seedBackupForExecution(t, db, "backup-run", "failed", -1)
	seedQueuedCleanupExecutionJob(t, db, "job-cleanup", "backup-run", 0)

	executed, err := ExecuteQueuedBackupCleanup(context.Background(), db, stubCleanupRunner{output: "host failed", err: exitErr(2)}, "ansible/playbooks/backup-cleanup.yml")
	if !executed {
		t.Fatalf("expected cleanup job execution")
	}
	if err == nil {
		t.Fatalf("expected execution error")
	}

	assertCleanupString(t, db, "SELECT status FROM jobs WHERE id = ?", "queued", "job-cleanup")
	assertCleanupString(t, db, "SELECT error_code FROM jobs WHERE id = ?", "ANSIBLE_HOST_FAILED", "job-cleanup")
	assertCleanupString(t, db, "SELECT status FROM backups WHERE id = ?", "failed", "backup-run")

	var runAfter string
	if err := db.QueryRow("SELECT run_after FROM jobs WHERE id = ?", "job-cleanup").Scan(&runAfter); err != nil {
		t.Fatalf("query run_after: %v", err)
	}
	if strings.TrimSpace(runAfter) == "" {
		t.Fatalf("expected run_after to be scheduled")
	}
	if _, parseErr := time.Parse(time.RFC3339, runAfter); parseErr != nil {
		t.Fatalf("expected RFC3339 run_after, got %q: %v", runAfter, parseErr)
	}
}

func TestExecuteQueuedBackupCleanupMarksFailureAfterFinalRetry(t *testing.T) {
	t.Parallel()

	db := openCleanupTestDB(t)
	seedCleanupTopology(t, db)
	seedBackupForExecution(t, db, "backup-run", "completed", -3)
	seedQueuedCleanupExecutionJob(t, db, "job-cleanup", "backup-run", 2)

	executed, err := ExecuteQueuedBackupCleanup(context.Background(), db, stubCleanupRunner{output: "host failed", err: exitErr(2)}, "ansible/playbooks/backup-cleanup.yml")
	if !executed {
		t.Fatalf("expected cleanup job execution")
	}
	if err == nil {
		t.Fatalf("expected execution error")
	}

	assertCleanupString(t, db, "SELECT status FROM jobs WHERE id = ?", "failed", "job-cleanup")
	assertCleanupString(t, db, "SELECT error_code FROM jobs WHERE id = ?", "ANSIBLE_HOST_FAILED", "job-cleanup")
}

func TestBackupCleanupScheduleIntervalIsSixHours(t *testing.T) {
	t.Parallel()

	if BackupCleanupScheduleInterval != 6*time.Hour {
		t.Fatalf("unexpected schedule interval: got %s", BackupCleanupScheduleInterval)
	}
}

type stubCleanupRunner struct {
	output string
	err    error
}

func (s stubCleanupRunner) Run(_ context.Context, _ string, _ ...string) (string, error) {
	return s.output, s.err
}

func openCleanupTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "backup-cleanup-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE environments (
			id TEXT PRIMARY KEY,
			site_id TEXT NOT NULL,
			node_id TEXT NOT NULL
		);
		CREATE TABLE nodes (
			id TEXT PRIMARY KEY,
			hostname TEXT NOT NULL
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

func seedCleanupTopology(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO nodes (id, hostname)
		VALUES ('node-1', 'localhost'), ('node-2', 'localhost');
		INSERT INTO environments (id, site_id, node_id)
		VALUES ('env-1', 'site-1', 'node-1'), ('env-2', 'site-1', 'node-1'), ('env-3', 'site-2', 'node-2');
	`); err != nil {
		t.Fatalf("seed topology: %v", err)
	}
}

func seedExpiredBackups(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO backups (id, environment_id, backup_scope, status, storage_type, storage_path, retention_until, checksum, size_bytes, created_at, completed_at)
		VALUES
		  ('backup-expired-a', 'env-1', 'full', 'completed', 's3', 's3://pressluft/backups/env-1/backup-expired-a.tar.zst', datetime('now', '-2 day'), 'sha256:a', 100, datetime('now', '-3 day'), datetime('now', '-3 day')),
		  ('backup-expired-b', 'env-2', 'full', 'failed', 's3', 's3://pressluft/backups/env-2/backup-expired-b.tar.zst', datetime('now', '-1 day'), 'sha256:b', 100, datetime('now', '-2 day'), datetime('now', '-2 day')),
		  ('backup-site2-expired', 'env-3', 'full', 'completed', 's3', 's3://pressluft/backups/env-3/backup-site2-expired.tar.zst', datetime('now', '-1 day'), 'sha256:c', 100, datetime('now', '-2 day'), datetime('now', '-2 day')),
		  ('backup-future', 'env-3', 'full', 'completed', 's3', 's3://pressluft/backups/env-3/backup-future.tar.zst', datetime('now', '+1 day'), 'sha256:d', 100, datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("seed backups: %v", err)
	}
}

func seedActiveCleanupJob(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (
			'job-active-cleanup', 'backup_cleanup', 'queued', 'site-2', 'env-3', 'node-2', '{"backup_id":"backup-site2-expired","environment_id":"env-3","storage_path":"s3://pressluft/backups/env-3/backup-site2-expired.tar.zst"}',
			0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, datetime('now'), datetime('now')
		)
	`); err != nil {
		t.Fatalf("seed active cleanup job: %v", err)
	}
}

func seedBackupForExecution(t *testing.T, db *sql.DB, backupID, status string, retentionDays int) {
	t.Helper()

	retentionExpr := "datetime('now', '" + strconv.Itoa(retentionDays) + " day')"
	query := `
		INSERT INTO backups (id, environment_id, backup_scope, status, storage_type, storage_path, retention_until, checksum, size_bytes, created_at, completed_at)
		VALUES (?, 'env-1', 'full', ?, 's3', ?, ` + retentionExpr + `, 'sha256:test', 200, datetime('now', '-5 day'), datetime('now', '-5 day'))
	`
	if _, err := db.Exec(query, backupID, status, "s3://pressluft/backups/env-1/"+backupID+".tar.zst"); err != nil {
		t.Fatalf("seed backup for execution: %v", err)
	}
}

func seedQueuedCleanupExecutionJob(t *testing.T, db *sql.DB, jobID, backupID string, attemptCount int) {
	t.Helper()

	payload := `{"backup_id":"` + backupID + `","environment_id":"env-1","storage_path":"s3://pressluft/backups/env-1/` + backupID + `.tar.zst"}`
	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (?, 'backup_cleanup', 'queued', 'site-1', 'env-1', 'node-1', ?, ?, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, jobID, payload, attemptCount); err != nil {
		t.Fatalf("seed queued backup cleanup job: %v", err)
	}
}

func assertCleanupString(t *testing.T, db *sql.DB, query, expected, arg string) {
	t.Helper()

	var got string
	if err := db.QueryRow(query, arg).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if got != expected {
		t.Fatalf("unexpected value for %q: got %q want %q", query, got, expected)
	}
}

func exitErr(code int) error {
	cmd := exec.Command("bash", "-c", "exit "+strconv.Itoa(code))
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
}
