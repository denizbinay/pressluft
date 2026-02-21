package promotion

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

func TestDriftCheckQueuesJobAndPersistsDriftRecord(t *testing.T) {
	t.Parallel()

	db := openPromotionTestDB(t)
	seedPromotionSiteAndEnvironments(t, db)
	svc := NewService(db)

	result, err := svc.DriftCheck(context.Background(), DriftCheckInput{EnvironmentID: "env-staging"})
	if err != nil {
		t.Fatalf("drift check: %v", err)
	}
	if result.JobID == "" || result.DriftCheckID == "" {
		t.Fatalf("expected drift check ids")
	}

	assertPromotionString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "drift_check", result.JobID)
	assertPromotionString(t, db, "SELECT drift_status FROM environments WHERE id = ?", "clean", "env-staging")
	assertPromotionString(t, db, "SELECT status FROM drift_checks WHERE id = ?", "clean", result.DriftCheckID)
}

func TestPromoteRequiresDriftAndFreshBackupGates(t *testing.T) {
	t.Parallel()

	db := openPromotionTestDB(t)
	seedPromotionSiteAndEnvironments(t, db)
	svc := NewService(db)

	_, err := svc.Promote(context.Background(), PromoteInput{
		EnvironmentID:       "env-staging",
		TargetEnvironmentID: "env-production",
	})
	if !errors.Is(err, ErrDriftGateNotMet) {
		t.Fatalf("expected drift gate error, got %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO drift_checks (id, environment_id, promotion_preset, status, db_checksums_json, file_checksums_json, checked_at)
		VALUES ('drift-1', 'env-staging', 'content-protect', 'clean', '{}', '{}', strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	`); err != nil {
		t.Fatalf("seed drift check: %v", err)
	}
	if _, err := db.Exec(`
		UPDATE environments
		SET drift_status = 'clean', last_drift_check_id = 'drift-1'
		WHERE id = 'env-staging'
	`); err != nil {
		t.Fatalf("seed clean drift status: %v", err)
	}

	_, err = svc.Promote(context.Background(), PromoteInput{
		EnvironmentID:       "env-staging",
		TargetEnvironmentID: "env-production",
	})
	if !errors.Is(err, ErrBackupGateNotMet) {
		t.Fatalf("expected backup gate error, got %v", err)
	}
}

func TestPromoteQueuesJobWhenGatesPass(t *testing.T) {
	t.Parallel()

	db := openPromotionTestDB(t)
	seedPromotionSiteAndEnvironments(t, db)
	seedCleanDriftState(t, db)
	seedFreshFullBackup(t, db, "env-production", "backup-prod")
	svc := NewService(db)

	result, err := svc.Promote(context.Background(), PromoteInput{
		EnvironmentID:       "env-staging",
		TargetEnvironmentID: "env-production",
	})
	if err != nil {
		t.Fatalf("promote: %v", err)
	}
	if result.JobID == "" {
		t.Fatalf("expected promote job id")
	}

	assertPromotionString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_promote", result.JobID)
	assertPromotionString(t, db, "SELECT status FROM environments WHERE id = ?", "deploying", "env-production")
	assertPromotionString(t, db, "SELECT status FROM sites WHERE id = ?", "deploying", "site-1")
}

func TestMarkPromoteFailedSetsFailedState(t *testing.T) {
	t.Parallel()

	db := openPromotionTestDB(t)
	seedPromotionSiteAndEnvironments(t, db)
	seedRunningPromoteJob(t, db, "job-promote")
	svc := NewService(db)

	if err := svc.MarkPromoteFailed(context.Background(), "site-1", "env-production", "job-promote", "ENV_PROMOTE_FAILED", "promotion failed"); err != nil {
		t.Fatalf("mark promote failed: %v", err)
	}

	assertPromotionString(t, db, "SELECT status FROM environments WHERE id = ?", "failed", "env-production")
	assertPromotionString(t, db, "SELECT status FROM sites WHERE id = ?", "failed", "site-1")
	assertPromotionString(t, db, "SELECT status FROM jobs WHERE id = ?", "failed", "job-promote")
	assertPromotionString(t, db, "SELECT error_code FROM jobs WHERE id = ?", "ENV_PROMOTE_FAILED", "job-promote")
}

func openPromotionTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "promotion-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

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
		CREATE TABLE drift_checks (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			promotion_preset TEXT NOT NULL,
			status TEXT NOT NULL,
			db_checksums_json TEXT NULL,
			file_checksums_json TEXT NULL,
			checked_at TEXT NOT NULL
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
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db
}

func seedPromotionSiteAndEnvironments(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO sites (id, name, slug, status, primary_environment_id, created_at, updated_at, state_version)
		VALUES ('site-1', 'Acme', 'acme', 'active', 'env-production', datetime('now'), datetime('now'), 1)
	`); err != nil {
		t.Fatalf("seed site: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO environments (
			id, site_id, name, slug, environment_type, status, node_id, source_environment_id,
			promotion_preset, preview_url, primary_domain_id, current_release_id, drift_status,
			drift_checked_at, last_drift_check_id, fastcgi_cache_enabled, redis_cache_enabled,
			created_at, updated_at, state_version
		)
		VALUES
		('env-staging', 'site-1', 'Staging', 'staging', 'staging', 'active', 'node-1', 'env-production', 'content-protect', 'http://staging', NULL, NULL, 'unknown', NULL, NULL, 1, 1, datetime('now'), datetime('now'), 1),
		('env-production', 'site-1', 'Production', 'production', 'production', 'active', 'node-1', NULL, 'content-protect', 'http://production', NULL, NULL, 'unknown', NULL, NULL, 1, 1, datetime('now'), datetime('now'), 1)
	`); err != nil {
		t.Fatalf("seed environments: %v", err)
	}
}

func seedCleanDriftState(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO drift_checks (id, environment_id, promotion_preset, status, db_checksums_json, file_checksums_json, checked_at)
		VALUES ('drift-clean', 'env-staging', 'content-protect', 'clean', '{}', '{}', strftime('%Y-%m-%dT%H:%M:%SZ', 'now'))
	`); err != nil {
		t.Fatalf("seed drift check: %v", err)
	}

	if _, err := db.Exec(`
		UPDATE environments
		SET drift_status = 'clean', last_drift_check_id = 'drift-clean'
		WHERE id = 'env-staging'
	`); err != nil {
		t.Fatalf("seed clean drift status: %v", err)
	}
}

func seedFreshFullBackup(t *testing.T, db *sql.DB, environmentID, backupID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO backups (
			id, environment_id, backup_scope, status, storage_type, storage_path,
			retention_until, checksum, size_bytes, created_at, completed_at
		)
		VALUES (?, ?, 'full', 'completed', 's3', 's3://pressluft/backups/test.tar.zst', strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+30 day'), 'sha256:ok', 123, strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-5 minute'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-5 minute'))
	`, backupID, environmentID); err != nil {
		t.Fatalf("seed fresh full backup: %v", err)
	}
}

func seedRunningPromoteJob(t *testing.T, db *sql.DB, jobID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (?, 'env_promote', 'running', 'site-1', 'env-production', 'node-1', '{}', 1, 3, NULL, datetime('now'), 'worker', datetime('now'), NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, jobID); err != nil {
		t.Fatalf("seed running promote job: %v", err)
	}
}

func assertPromotionString(t *testing.T, db *sql.DB, query, expected, arg string) {
	t.Helper()

	var got string
	if err := db.QueryRow(query, arg).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if got != expected {
		t.Fatalf("unexpected value for %q: got %q want %q", query, got, expected)
	}
}

func TestPromoteReturnsConcurrencyConflictWhenTargetSiteBusy(t *testing.T) {
	t.Parallel()

	db := openPromotionTestDB(t)
	seedPromotionSiteAndEnvironments(t, db)
	seedCleanDriftState(t, db)
	seedFreshFullBackup(t, db, "env-production", "backup-prod")
	seedQueuedSiteMutationJob(t, db)
	svc := NewService(db)

	_, err := svc.Promote(context.Background(), PromoteInput{
		EnvironmentID:       "env-staging",
		TargetEnvironmentID: "env-production",
	})
	if !errors.Is(err, jobs.ErrConcurrencyConflict) {
		t.Fatalf("expected concurrency conflict, got %v", err)
	}
}

func seedQueuedSiteMutationJob(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES ('job-existing', 'env_deploy', 'queued', 'site-1', 'env-production', 'node-1', '{}', 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("seed queued job: %v", err)
	}
}
