package sites

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

func TestCreatePersistsSiteEnvironmentAndQueuedJob(t *testing.T) {
	t.Parallel()

	db := openSitesTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5", "active")
	svc := NewService(db)

	result, err := svc.Create(context.Background(), CreateInput{Name: "Acme", Slug: "acme"})
	if err != nil {
		t.Fatalf("create site: %v", err)
	}

	if result.JobID == "" || result.SiteID == "" {
		t.Fatalf("expected non-empty create result ids")
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "active", result.SiteID)

	var envID, envStatus, envType string
	if err := db.QueryRow(`
		SELECT sites.primary_environment_id, environments.status, environments.environment_type
		FROM environments
		JOIN sites ON sites.primary_environment_id = environments.id
		WHERE sites.id = ?
	`, result.SiteID).Scan(&envID, &envStatus, &envType); err != nil {
		t.Fatalf("query initial environment: %v", err)
	}
	if envStatus != "active" || envType != "production" {
		t.Fatalf("unexpected environment values status=%s type=%s", envStatus, envType)
	}

	assertString(t, db, "SELECT status FROM jobs WHERE id = ?", "queued", result.JobID)
	assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "site_create", result.JobID)
	assertString(t, db, "SELECT site_id FROM jobs WHERE id = ?", result.SiteID, result.JobID)
	assertString(t, db, "SELECT node_id FROM jobs WHERE id = ?", "node-1", result.JobID)

	_ = envID
}

func TestCreateRejectsDuplicateSlug(t *testing.T) {
	t.Parallel()

	db := openSitesTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5", "active")
	svc := NewService(db)

	if _, err := svc.Create(context.Background(), CreateInput{Name: "Acme", Slug: "acme"}); err != nil {
		t.Fatalf("first create site: %v", err)
	}

	if _, err := svc.Create(context.Background(), CreateInput{Name: "Acme 2", Slug: "acme"}); !errors.Is(err, ErrSlugConflict) {
		t.Fatalf("expected ErrSlugConflict, got %v", err)
	}
}

func TestCreateReturnsConcurrencyConflictWhenNodeBusy(t *testing.T) {
	t.Parallel()

	db := openSitesTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5", "active")
	seedQueuedNodeMutationJob(t, db, "node-1")
	svc := NewService(db)

	_, err := svc.Create(context.Background(), CreateInput{Name: "Acme", Slug: "acme"})
	if !errors.Is(err, jobs.ErrConcurrencyConflict) {
		t.Fatalf("expected concurrency conflict, got %v", err)
	}
}

func TestResetFailedSetsSiteToActive(t *testing.T) {
	t.Parallel()

	db := openSitesTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5", "active")
	seedFailedSite(t, db, "site-1")
	svc := NewService(db)

	site, err := svc.ResetFailed(context.Background(), "site-1")
	if err != nil {
		t.Fatalf("reset failed site: %v", err)
	}
	if site.Status != "active" {
		t.Fatalf("expected active status, got %s", site.Status)
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "active", "site-1")
}

func TestResetFailedRejectsNonFailedSite(t *testing.T) {
	t.Parallel()

	db := openSitesTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5", "active")
	seedFailedSite(t, db, "site-1")
	if _, err := db.Exec(`UPDATE sites SET status = 'active' WHERE id = 'site-1'`); err != nil {
		t.Fatalf("set active status: %v", err)
	}
	svc := NewService(db)

	_, err := svc.ResetFailed(context.Background(), "site-1")
	if !errors.Is(err, ErrResourceNotFailed) {
		t.Fatalf("expected ErrResourceNotFailed, got %v", err)
	}
}

func TestResetFailedRejectsWhenActiveMutationExists(t *testing.T) {
	t.Parallel()

	db := openSitesTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5", "active")
	seedFailedSite(t, db, "site-1")
	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES ('job-running', 'env_deploy', 'running', 'site-1', NULL, 'node-1', '{}', 1, 3, NULL, datetime('now'), 'worker', datetime('now'), NULL, NULL, NULL, datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("seed running job: %v", err)
	}
	svc := NewService(db)

	_, err := svc.ResetFailed(context.Background(), "site-1")
	if !errors.Is(err, ErrResetValidationFailed) {
		t.Fatalf("expected ErrResetValidationFailed, got %v", err)
	}
}

func openSitesTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "sites-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
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
		CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
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

func seedNode(t *testing.T, db *sql.DB, nodeID, publicIP, status string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO nodes (id, name, hostname, public_ip, ssh_port, ssh_user, status, is_local, last_seen_at, created_at, updated_at, state_version)
		VALUES (?, 'local-node', 'localhost', ?, 22, 'root', ?, 1, NULL, datetime('now'), datetime('now'), 1)
	`, nodeID, publicIP, status); err != nil {
		t.Fatalf("seed node: %v", err)
	}
}

func seedQueuedNodeMutationJob(t *testing.T, db *sql.DB, nodeID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES ('job-existing', 'node_provision', 'queued', NULL, NULL, ?, '{}', 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, nodeID); err != nil {
		t.Fatalf("seed queued mutation job: %v", err)
	}
}

func seedFailedSite(t *testing.T, db *sql.DB, siteID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO sites (id, name, slug, status, primary_environment_id, created_at, updated_at, state_version)
		VALUES (?, 'Acme', ?, 'failed', NULL, datetime('now'), datetime('now'), 1)
	`, siteID, siteID); err != nil {
		t.Fatalf("seed failed site: %v", err)
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
