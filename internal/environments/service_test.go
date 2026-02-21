package environments

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

func TestCreateQueuesEnvCreateAndSetsCloningStates(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, sourceEnvironmentID := seedSiteWithProduction(t, db)
	svc := NewService(db)

	result, err := svc.Create(context.Background(), CreateInput{
		SiteID:              siteID,
		Name:                "Staging",
		Slug:                "staging",
		EnvironmentType:     "staging",
		SourceEnvironmentID: &sourceEnvironmentID,
		PromotionPreset:     "content-protect",
	})
	if err != nil {
		t.Fatalf("create environment: %v", err)
	}

	if result.JobID == "" || result.EnvironmentID == "" {
		t.Fatalf("expected non-empty create result ids")
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "cloning", siteID)
	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "cloning", result.EnvironmentID)
	assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_create", result.JobID)
	assertString(t, db, "SELECT status FROM jobs WHERE id = ?", "queued", result.JobID)
}

func TestCreateReturnsConcurrencyConflictWhenSiteBusy(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, sourceEnvironmentID := seedSiteWithProduction(t, db)
	seedQueuedSiteMutationJob(t, db, siteID, "node-1")
	svc := NewService(db)

	_, err := svc.Create(context.Background(), CreateInput{
		SiteID:              siteID,
		Name:                "Staging",
		Slug:                "staging",
		EnvironmentType:     "staging",
		SourceEnvironmentID: &sourceEnvironmentID,
		PromotionPreset:     "content-protect",
	})
	if !errors.Is(err, jobs.ErrConcurrencyConflict) {
		t.Fatalf("expected concurrency conflict, got %v", err)
	}
}

func TestDeployQueuesJobAndSetsDeployingState(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, environmentID := seedSiteWithProduction(t, db)
	svc := NewService(db)

	result, err := svc.Deploy(context.Background(), DeployInput{
		EnvironmentID: environmentID,
		SourceType:    "git",
		SourceRef:     "git@github.com:acme/site.git#main",
	})
	if err != nil {
		t.Fatalf("deploy environment: %v", err)
	}
	if result.JobID == "" || result.ReleaseID == "" {
		t.Fatalf("expected non-empty deploy result ids")
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "deploying", siteID)
	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "deploying", environmentID)
	assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_deploy", result.JobID)
	assertString(t, db, "SELECT source_type FROM releases WHERE id = ?", "git", result.ReleaseID)
}

func TestUpdatesQueuesJobAndCreatesPreUpdateBackupWhenMissing(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, environmentID := seedSiteWithProduction(t, db)
	svc := NewService(db)

	result, err := svc.Updates(context.Background(), UpdatesInput{
		EnvironmentID: environmentID,
		Scope:         "all",
	})
	if err != nil {
		t.Fatalf("queue updates: %v", err)
	}
	if result.JobID == "" || result.PreUpdateBackup == "" {
		t.Fatalf("expected update result ids")
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "deploying", siteID)
	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "deploying", environmentID)
	assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_update", result.JobID)
	assertString(t, db, "SELECT status FROM backups WHERE id = ?", "pending", result.PreUpdateBackup)
}

func TestUpdatesUsesFreshCompletedBackup(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	_, environmentID := seedSiteWithProduction(t, db)
	seedCompletedBackup(t, db, environmentID, "backup-fresh")
	svc := NewService(db)

	result, err := svc.Updates(context.Background(), UpdatesInput{
		EnvironmentID: environmentID,
		Scope:         "plugins",
	})
	if err != nil {
		t.Fatalf("queue updates: %v", err)
	}
	if result.PreUpdateBackup != "backup-fresh" {
		t.Fatalf("expected fresh backup reuse, got %s", result.PreUpdateBackup)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM backups WHERE environment_id = ?", environmentID).Scan(&count); err != nil {
		t.Fatalf("count backups: %v", err)
	}
	if count != 1 {
		t.Fatalf("expected one backup row, got %d", count)
	}

	var payloadJSON string
	if err := db.QueryRow("SELECT payload_json FROM jobs WHERE id = ?", result.JobID).Scan(&payloadJSON); err != nil {
		t.Fatalf("query update payload: %v", err)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode update payload: %v", err)
	}
	if payload["pre_update_backup_id"] != "backup-fresh" {
		t.Fatalf("expected payload backup id backup-fresh, got %q", payload["pre_update_backup_id"])
	}
}

func TestRestoreQueuesJobAndSetsRestoringState(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, environmentID := seedSiteWithProduction(t, db)
	seedCompletedBackup(t, db, environmentID, "backup-restore")
	svc := NewService(db)

	result, err := svc.Restore(context.Background(), RestoreInput{
		EnvironmentID: environmentID,
		BackupID:      "backup-restore",
	})
	if err != nil {
		t.Fatalf("queue restore: %v", err)
	}
	if result.JobID == "" || result.PreRestoreBackup == "" {
		t.Fatalf("expected restore result ids")
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "restoring", siteID)
	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "restoring", environmentID)
	assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_restore", result.JobID)
	assertString(t, db, "SELECT status FROM backups WHERE id = ?", "pending", result.PreRestoreBackup)

	var payloadJSON string
	if err := db.QueryRow("SELECT payload_json FROM jobs WHERE id = ?", result.JobID).Scan(&payloadJSON); err != nil {
		t.Fatalf("query restore payload: %v", err)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode restore payload: %v", err)
	}
	if payload["backup_id"] != "backup-restore" {
		t.Fatalf("expected restore backup id backup-restore, got %q", payload["backup_id"])
	}
}

func TestRestoreUsesFreshCompletedPreRestoreBackup(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	_, environmentID := seedSiteWithProduction(t, db)
	if _, err := db.Exec(`
		INSERT INTO backups (
			id, environment_id, backup_scope, status, storage_type, storage_path,
			retention_until, checksum, size_bytes, created_at, completed_at
		)
		VALUES (?, ?, 'full', 'completed', 's3', 's3://pressluft/backups/test.tar.zst', strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+30 day'), 'sha256:old', 123, strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-2 day'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-2 day'))
	`, "backup-restore", environmentID); err != nil {
		t.Fatalf("seed restore backup: %v", err)
	}
	seedCompletedBackup(t, db, environmentID, "backup-fresh")
	svc := NewService(db)

	result, err := svc.Restore(context.Background(), RestoreInput{
		EnvironmentID: environmentID,
		BackupID:      "backup-restore",
	})
	if err != nil {
		t.Fatalf("queue restore: %v", err)
	}
	if result.PreRestoreBackup != "backup-fresh" {
		t.Fatalf("expected fresh pre-restore backup reuse, got %s", result.PreRestoreBackup)
	}

	var payloadJSON string
	if err := db.QueryRow("SELECT payload_json FROM jobs WHERE id = ?", result.JobID).Scan(&payloadJSON); err != nil {
		t.Fatalf("query restore payload: %v", err)
	}
	var payload map[string]string
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode restore payload: %v", err)
	}
	if payload["pre_restore_backup_id"] != "backup-fresh" {
		t.Fatalf("expected pre-restore backup id backup-fresh, got %q", payload["pre_restore_backup_id"])
	}
}

func TestRestoreReturnsBackupNotFoundForMismatchedBackup(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	_, environmentID := seedSiteWithProduction(t, db)
	seedCompletedBackup(t, db, "other-env", "backup-other")
	svc := NewService(db)

	_, err := svc.Restore(context.Background(), RestoreInput{
		EnvironmentID: environmentID,
		BackupID:      "backup-other",
	})
	if !errors.Is(err, ErrBackupNotFound) {
		t.Fatalf("expected backup not found, got %v", err)
	}
}

func TestToggleCacheQueuesJobAndUpdatesRequestedFieldsOnly(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	_, environmentID := seedSiteWithProduction(t, db)
	svc := NewService(db)

	fastcgiDisabled := false
	result, err := svc.ToggleCache(context.Background(), CacheToggleInput{
		EnvironmentID:       environmentID,
		FastCGICacheEnabled: &fastcgiDisabled,
	})
	if err != nil {
		t.Fatalf("toggle cache: %v", err)
	}
	if result.JobID == "" {
		t.Fatalf("expected non-empty job id")
	}

	assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "env_cache_toggle", result.JobID)

	var fastcgi, redis int
	if err := db.QueryRow("SELECT fastcgi_cache_enabled, redis_cache_enabled FROM environments WHERE id = ?", environmentID).Scan(&fastcgi, &redis); err != nil {
		t.Fatalf("query cache flags: %v", err)
	}
	if fastcgi != 0 || redis != 1 {
		t.Fatalf("unexpected cache flags fastcgi=%d redis=%d", fastcgi, redis)
	}

	var payloadJSON string
	if err := db.QueryRow("SELECT payload_json FROM jobs WHERE id = ?", result.JobID).Scan(&payloadJSON); err != nil {
		t.Fatalf("query cache toggle payload: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode cache toggle payload: %v", err)
	}
	if payload["fastcgi_cache_enabled"] != false {
		t.Fatalf("expected payload fastcgi_cache_enabled false, got %v", payload["fastcgi_cache_enabled"])
	}
	if _, ok := payload["redis_cache_enabled"]; ok {
		t.Fatalf("expected payload to omit redis_cache_enabled when not provided")
	}
}

func TestToggleCacheRejectsEmptyPayload(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	_, environmentID := seedSiteWithProduction(t, db)
	svc := NewService(db)

	_, err := svc.ToggleCache(context.Background(), CacheToggleInput{EnvironmentID: environmentID})
	if !errors.Is(err, ErrInvalidInput) {
		t.Fatalf("expected invalid input, got %v", err)
	}
}

func TestPurgeCacheQueuesJob(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	_, environmentID := seedSiteWithProduction(t, db)
	svc := NewService(db)

	result, err := svc.PurgeCache(context.Background(), CachePurgeInput{EnvironmentID: environmentID})
	if err != nil {
		t.Fatalf("purge cache: %v", err)
	}
	if result.JobID == "" {
		t.Fatalf("expected non-empty job id")
	}

	assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "cache_purge", result.JobID)

	var payloadJSON string
	if err := db.QueryRow("SELECT payload_json FROM jobs WHERE id = ?", result.JobID).Scan(&payloadJSON); err != nil {
		t.Fatalf("query cache purge payload: %v", err)
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
		t.Fatalf("decode cache purge payload: %v", err)
	}
	if payload["environment_id"] != environmentID {
		t.Fatalf("expected environment_id %s, got %v", environmentID, payload["environment_id"])
	}
}

func TestMarkDeployOrUpdateSucceededReturnsEnvironmentToActive(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, environmentID := seedSiteWithProduction(t, db)
	seedRunningMutationJob(t, db, "job-deploy", siteID, environmentID)
	seedRelease(t, db, "rel-new", environmentID)
	svc := NewService(db)

	if err := svc.MarkDeployOrUpdateSucceeded(context.Background(), siteID, environmentID, "job-deploy", "rel-new"); err != nil {
		t.Fatalf("mark mutation success: %v", err)
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "active", siteID)
	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "active", environmentID)
	assertString(t, db, "SELECT current_release_id FROM environments WHERE id = ?", "rel-new", environmentID)
	assertString(t, db, "SELECT status FROM jobs WHERE id = ?", "succeeded", "job-deploy")
}

func TestMarkDeployOrUpdateFailedSetsFailedState(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, environmentID := seedSiteWithProduction(t, db)
	seedRunningMutationJob(t, db, "job-update", siteID, environmentID)
	svc := NewService(db)

	if err := svc.MarkDeployOrUpdateFailed(context.Background(), siteID, environmentID, "job-update", "ENV_UPDATE_FAILED", "wp-cli update failed"); err != nil {
		t.Fatalf("mark mutation failure: %v", err)
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "failed", siteID)
	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "failed", environmentID)
	assertString(t, db, "SELECT status FROM jobs WHERE id = ?", "failed", "job-update")
	assertString(t, db, "SELECT error_code FROM jobs WHERE id = ?", "ENV_UPDATE_FAILED", "job-update")
}

func TestMarkRestoreSucceededReturnsEnvironmentToActive(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, environmentID := seedSiteWithProduction(t, db)
	seedRunningRestoreJob(t, db, "job-restore", siteID, environmentID)
	svc := NewService(db)

	if err := svc.MarkRestoreSucceeded(context.Background(), siteID, environmentID, "job-restore"); err != nil {
		t.Fatalf("mark restore success: %v", err)
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "active", siteID)
	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "active", environmentID)
	assertString(t, db, "SELECT status FROM jobs WHERE id = ?", "succeeded", "job-restore")
}

func TestMarkRestoreFailedSetsFailedState(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, environmentID := seedSiteWithProduction(t, db)
	seedRunningRestoreJob(t, db, "job-restore", siteID, environmentID)
	svc := NewService(db)

	if err := svc.MarkRestoreFailed(context.Background(), siteID, environmentID, "job-restore", "ENV_RESTORE_FAILED", "restore task failed"); err != nil {
		t.Fatalf("mark restore failure: %v", err)
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "failed", siteID)
	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "failed", environmentID)
	assertString(t, db, "SELECT status FROM jobs WHERE id = ?", "failed", "job-restore")
	assertString(t, db, "SELECT error_code FROM jobs WHERE id = ?", "ENV_RESTORE_FAILED", "job-restore")
}

func TestResetFailedEnvironmentSetsEnvironmentAndSiteActiveWhenSafe(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, environmentID := seedSiteWithProduction(t, db)
	if _, err := db.Exec(`
		UPDATE environments SET status = 'failed' WHERE id = ?;
		UPDATE sites SET status = 'failed' WHERE id = ?;
	`, environmentID, siteID); err != nil {
		t.Fatalf("seed failed environment + site: %v", err)
	}
	svc := NewService(db)

	env, err := svc.ResetFailed(context.Background(), environmentID)
	if err != nil {
		t.Fatalf("reset failed environment: %v", err)
	}
	if env.Status != "active" {
		t.Fatalf("expected environment active, got %s", env.Status)
	}

	assertString(t, db, "SELECT status FROM environments WHERE id = ?", "active", environmentID)
	assertString(t, db, "SELECT status FROM sites WHERE id = ?", "active", siteID)
}

func TestResetFailedEnvironmentRejectsNonFailed(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	_, environmentID := seedSiteWithProduction(t, db)
	svc := NewService(db)

	_, err := svc.ResetFailed(context.Background(), environmentID)
	if !errors.Is(err, ErrResourceNotFailed) {
		t.Fatalf("expected ErrResourceNotFailed, got %v", err)
	}
}

func TestResetFailedEnvironmentRejectsWhenActiveJobExists(t *testing.T) {
	t.Parallel()

	db := openEnvironmentsTestDB(t)
	seedNode(t, db, "node-1", "203.0.113.5")
	siteID, environmentID := seedSiteWithProduction(t, db)
	if _, err := db.Exec(`
		UPDATE environments SET status = 'failed' WHERE id = ?;
		UPDATE sites SET status = 'failed' WHERE id = ?;
	`, environmentID, siteID); err != nil {
		t.Fatalf("seed failed environment + site: %v", err)
	}
	seedRunningMutationJob(t, db, "job-running", siteID, environmentID)
	svc := NewService(db)

	_, err := svc.ResetFailed(context.Background(), environmentID)
	if !errors.Is(err, ErrResetValidationFailed) {
		t.Fatalf("expected ErrResetValidationFailed, got %v", err)
	}
}

func openEnvironmentsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "environments-test.db")
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

func seedNode(t *testing.T, db *sql.DB, nodeID, publicIP string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO nodes (id, name, hostname, public_ip, ssh_port, ssh_user, status, is_local, last_seen_at, created_at, updated_at, state_version)
		VALUES (?, 'local-node', 'localhost', ?, 22, 'root', 'active', 1, NULL, datetime('now'), datetime('now'), 1)
	`, nodeID, publicIP); err != nil {
		t.Fatalf("seed node: %v", err)
	}
}

func seedSiteWithProduction(t *testing.T, db *sql.DB) (string, string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO sites (id, name, slug, status, primary_environment_id, created_at, updated_at, state_version)
		VALUES ('site-1', 'Acme', 'acme', 'active', 'env-1', datetime('now'), datetime('now'), 1)
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
		VALUES ('env-1', 'site-1', 'Production', 'production', 'production', 'active', 'node-1', NULL, 'content-protect', 'http://prod.203-0-113-5.sslip.io', NULL, NULL, 'unknown', NULL, NULL, 1, 1, datetime('now'), datetime('now'), 1)
	`); err != nil {
		t.Fatalf("seed production environment: %v", err)
	}

	return "site-1", "env-1"
}

func seedQueuedSiteMutationJob(t *testing.T, db *sql.DB, siteID, nodeID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES ('job-existing', 'site_create', 'queued', ?, NULL, ?, '{}', 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, siteID, nodeID); err != nil {
		t.Fatalf("seed queued mutation job: %v", err)
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

func seedRunningMutationJob(t *testing.T, db *sql.DB, jobID, siteID, environmentID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (?, 'env_deploy', 'running', ?, ?, 'node-1', '{}', 1, 3, NULL, datetime('now'), 'worker', datetime('now'), NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, jobID, siteID, environmentID); err != nil {
		t.Fatalf("seed running mutation job: %v", err)
	}
}

func seedRunningRestoreJob(t *testing.T, db *sql.DB, jobID, siteID, environmentID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (?, 'env_restore', 'running', ?, ?, 'node-1', '{}', 1, 3, NULL, datetime('now'), 'worker', datetime('now'), NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, jobID, siteID, environmentID); err != nil {
		t.Fatalf("seed running restore job: %v", err)
	}
}

func seedRelease(t *testing.T, db *sql.DB, releaseID, environmentID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO releases (id, environment_id, source_type, source_ref, path, health_status, notes, created_at)
		VALUES (?, ?, 'git', 'git@github.com:acme/site.git#main', '/var/www/sites/releases/new', 'unknown', NULL, datetime('now'))
	`, releaseID, environmentID); err != nil {
		t.Fatalf("seed release: %v", err)
	}
}

func seedCompletedBackup(t *testing.T, db *sql.DB, environmentID, backupID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO backups (
			id, environment_id, backup_scope, status, storage_type, storage_path,
			retention_until, checksum, size_bytes, created_at, completed_at
		)
		VALUES (?, ?, 'full', 'completed', 's3', 's3://pressluft/backups/test.tar.zst', strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '+30 day'), 'sha256:ok', 123, strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-5 minute'), strftime('%Y-%m-%dT%H:%M:%SZ', 'now', '-5 minute'))
	`, backupID, environmentID); err != nil {
		t.Fatalf("seed completed backup: %v", err)
	}
}
