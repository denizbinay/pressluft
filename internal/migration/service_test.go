package migration

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

func TestImportSiteQueuesJobAndSetsRestoringState(t *testing.T) {
	t.Parallel()

	db := openMigrationTestDB(t)
	seedMigrationSite(t, db)
	svc := NewService(db)

	result, err := svc.ImportSite(context.Background(), ImportInput{
		SiteID:     "site-1",
		ArchiveURL: "https://example.com/archive.tar.gz",
	})
	if err != nil {
		t.Fatalf("import site: %v", err)
	}

	if result.JobID == "" || result.EnvironmentID == "" {
		t.Fatalf("expected import identifiers")
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "restoring", "site-1")
	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "restoring", "env-1")
	assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "site_import", result.JobID)

	var payload string
	if err := db.QueryRow("SELECT payload_json FROM jobs WHERE id = ?", result.JobID).Scan(&payload); err != nil {
		t.Fatalf("query payload_json: %v", err)
	}
	if !strings.Contains(payload, `"target_url":"http://preview.test"`) {
		t.Fatalf("expected payload target_url to use preview_url, got %s", payload)
	}
}

func TestExecuteQueuedSiteImportRetriesOnRetryableExitCode(t *testing.T) {
	t.Parallel()

	db := openMigrationTestDB(t)
	seedMigrationSite(t, db)
	seedQueuedImportJob(t, db, "job-import", "release-1", 0)

	executed, err := ExecuteQueuedSiteImport(context.Background(), db, stubRunner{output: "host failed", err: exitErr(2)}, "ansible/playbooks/site-import.yml")
	if !executed {
		t.Fatalf("expected job to be executed")
	}
	if err == nil {
		t.Fatalf("expected execution error")
	}

	assertString(t, db, "SELECT status FROM jobs WHERE id = ?", "queued", "job-import")
	assertString(t, db, "SELECT error_code FROM jobs WHERE id = ?", "ANSIBLE_HOST_FAILED", "job-import")

	var runAfter string
	if err := db.QueryRow("SELECT run_after FROM jobs WHERE id = ?", "job-import").Scan(&runAfter); err != nil {
		t.Fatalf("query run_after: %v", err)
	}
	if strings.TrimSpace(runAfter) == "" {
		t.Fatalf("expected run_after to be scheduled")
	}
	if _, parseErr := time.Parse(time.RFC3339, runAfter); parseErr != nil {
		t.Fatalf("expected RFC3339 run_after, got %q: %v", runAfter, parseErr)
	}
}

func TestExecuteQueuedSiteImportMarksSuccessAndActivatesRelease(t *testing.T) {
	t.Parallel()

	db := openMigrationTestDB(t)
	seedMigrationSite(t, db)
	seedQueuedImportJob(t, db, "job-import", "release-1", 0)

	executed, err := ExecuteQueuedSiteImport(context.Background(), db, stubRunner{output: "ok", err: nil}, "ansible/playbooks/site-import.yml")
	if err != nil {
		t.Fatalf("execute import: %v", err)
	}
	if !executed {
		t.Fatalf("expected job to be executed")
	}

	assertString(t, db, "SELECT status FROM jobs WHERE id = ?", "succeeded", "job-import")
	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "active", "site-1")
	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "active", "env-1")
	assertString(t, db, "SELECT current_release_id FROM environments WHERE id = ?", "release-1", "env-1")
}

type stubRunner struct {
	output string
	err    error
}

func (s stubRunner) Run(_ context.Context, _ string, _ ...string) (string, error) {
	return s.output, s.err
}

func openMigrationTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "migration-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE sites (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL,
			primary_environment_id TEXT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL
		);
		CREATE TABLE environments (
			id TEXT PRIMARY KEY,
			site_id TEXT NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			environment_type TEXT NOT NULL,
			status TEXT NOT NULL,
			node_id TEXT NOT NULL,
			source_environment_id TEXT NULL,
			promotion_preset TEXT NOT NULL,
			preview_url TEXT NOT NULL,
			primary_domain_id TEXT NULL,
			current_release_id TEXT NULL,
			drift_status TEXT NOT NULL,
			drift_checked_at TEXT NULL,
			last_drift_check_id TEXT NULL,
			fastcgi_cache_enabled INTEGER NOT NULL,
			redis_cache_enabled INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL,
			UNIQUE(site_id, slug)
		);
		CREATE TABLE nodes (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			hostname TEXT NOT NULL,
			public_ip TEXT NULL,
			ssh_port INTEGER NOT NULL,
			ssh_user TEXT NOT NULL,
			status TEXT NOT NULL,
			is_local INTEGER NOT NULL,
			last_seen_at TEXT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL
		);
		CREATE TABLE domains (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			hostname TEXT NOT NULL UNIQUE,
			tls_status TEXT NOT NULL,
			tls_issuer TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE releases (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			source_type TEXT NOT NULL,
			source_ref TEXT NOT NULL,
			path TEXT NOT NULL,
			health_status TEXT NOT NULL,
			notes TEXT NULL,
			created_at TEXT NOT NULL
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

func seedMigrationSite(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO nodes (id, name, hostname, public_ip, ssh_port, ssh_user, status, is_local, last_seen_at, created_at, updated_at, state_version)
		VALUES ('node-1', 'local', 'localhost', '203.0.113.10', 22, 'root', 'active', 1, NULL, datetime('now'), datetime('now'), 1);
		INSERT INTO sites (id, name, slug, status, primary_environment_id, created_at, updated_at, state_version)
		VALUES ('site-1', 'Acme', 'acme', 'active', 'env-1', datetime('now'), datetime('now'), 1);
		INSERT INTO environments (
			id, site_id, name, slug, environment_type, status, node_id, source_environment_id,
			promotion_preset, preview_url, primary_domain_id, current_release_id, drift_status,
			drift_checked_at, last_drift_check_id, fastcgi_cache_enabled, redis_cache_enabled,
			created_at, updated_at, state_version
		)
		VALUES (
			'env-1', 'site-1', 'Production', 'production', 'production', 'active', 'node-1', NULL,
			'content-protect', 'http://preview.test', NULL, NULL, 'unknown',
			NULL, NULL, 1, 1, datetime('now'), datetime('now'), 1
		);
	`); err != nil {
		t.Fatalf("seed migration site: %v", err)
	}
}

func seedQueuedImportJob(t *testing.T, db *sql.DB, jobID, releaseID string, attemptCount int) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (?, 'site_import', 'queued', 'site-1', 'env-1', 'node-1', ?, ?, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, jobID, `{"site_id":"site-1","environment_id":"env-1","node_id":"node-1","archive_url":"https://example.com/archive.tar.gz","release_id":"`+releaseID+`","target_url":"http://preview.test"}`, attemptCount); err != nil {
		t.Fatalf("seed queued site_import job: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO releases (id, environment_id, source_type, source_ref, path, health_status, notes, created_at)
		VALUES (?, 'env-1', 'upload', 'https://example.com/archive.tar.gz', '/var/www/sites/releases/pending', 'unknown', 'site import', datetime('now'))
	`, releaseID); err != nil {
		t.Fatalf("seed release: %v", err)
	}
}

func exitErr(code int) error {
	cmd := exec.Command("bash", "-c", "exit "+strconv.Itoa(code))
	if err := cmd.Run(); err != nil {
		return err
	}
	return nil
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
