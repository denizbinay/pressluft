package releases

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"pressluft/internal/store"
)

func TestTriggerPostMutationHealthCheckQueuesJobForSupportedMutations(t *testing.T) {
	t.Parallel()

	for _, mutationType := range []string{"env_deploy", "env_restore", "env_promote"} {
		mutationType := mutationType
		t.Run(mutationType, func(t *testing.T) {
			t.Parallel()

			db := openReleasesTestDB(t)
			seedReleaseGraph(t, db)
			svc := NewService(db)

			result, err := svc.TriggerPostMutationHealthCheck(context.Background(), HealthCheckTriggerInput{
				SiteID:          "site-1",
				EnvironmentID:   "env-1",
				NodeID:          "node-1",
				MutationJobType: mutationType,
			})
			if err != nil {
				t.Fatalf("trigger health check: %v", err)
			}
			if result.JobID == "" || result.ReleaseID != "rel-new" {
				t.Fatalf("unexpected trigger result: %+v", result)
			}

			assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "health_check", result.JobID)
			assertString(t, db, "SELECT status FROM jobs WHERE id = ?", "queued", result.JobID)
			assertString(t, db, "SELECT environment_id FROM jobs WHERE id = ?", "env-1", result.JobID)
		})
	}
}

func TestHandleHealthCheckFailureMarksReleaseAndQueuesRollback(t *testing.T) {
	t.Parallel()

	db := openReleasesTestDB(t)
	seedReleaseGraph(t, db)
	seedHealthCheckJob(t, db, "job-health")
	svc := NewService(db)

	result, err := svc.HandleHealthCheckFailure(context.Background(), HealthCheckFailureInput{
		SiteID:           "site-1",
		EnvironmentID:    "env-1",
		NodeID:           "node-1",
		HealthCheckJobID: "job-health",
		Err:              errors.New("http check did not return 200"),
	})
	if err != nil {
		t.Fatalf("handle health check failure: %v", err)
	}

	if result.RollbackJobID == "" || result.FailedReleaseID != "rel-new" || result.RestoredReleaseID != "rel-old" {
		t.Fatalf("unexpected rollback result: %+v", result)
	}

	assertString(t, db, "SELECT health_status FROM releases WHERE id = 'rel-new'", "unhealthy")
	assertString(t, db, "SELECT status FROM sites WHERE id = 'site-1'", "restoring")
	assertString(t, db, "SELECT status FROM environments WHERE id = 'env-1'", "restoring")
	assertString(t, db, "SELECT status FROM jobs WHERE id = 'job-health'", "failed")
	assertString(t, db, "SELECT error_code FROM jobs WHERE id = 'job-health'", HealthCheckErrorCodeFailed)
	assertString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "release_rollback", result.RollbackJobID)
}

func TestApplyRollbackSuccessRestoresPreviousReleaseAndActiveStates(t *testing.T) {
	t.Parallel()

	db := openReleasesTestDB(t)
	seedReleaseGraph(t, db)
	seedRollbackJob(t, db, "job-rollback")
	svc := NewService(db)

	if err := svc.ApplyRollbackSuccess(context.Background(), RollbackSuccessInput{
		SiteID:            "site-1",
		EnvironmentID:     "env-1",
		RollbackJobID:     "job-rollback",
		RestoredReleaseID: "rel-old",
	}); err != nil {
		t.Fatalf("apply rollback success: %v", err)
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = 'site-1'", "active")
	assertString(t, db, "SELECT status FROM environments WHERE id = 'env-1'", "active")
	assertString(t, db, "SELECT current_release_id FROM environments WHERE id = 'env-1'", "rel-old")
	assertString(t, db, "SELECT health_status FROM releases WHERE id = 'rel-old'", "healthy")
	assertString(t, db, "SELECT status FROM jobs WHERE id = 'job-rollback'", "succeeded")
}

func TestApplyRollbackFailureStoresStableStructuredError(t *testing.T) {
	t.Parallel()

	db := openReleasesTestDB(t)
	seedReleaseGraph(t, db)
	seedRollbackJob(t, db, "job-rollback")
	svc := NewService(db)

	longError := strings.Repeat("x", 800)
	if err := svc.ApplyRollbackFailure(context.Background(), RollbackFailureInput{
		SiteID:        "site-1",
		EnvironmentID: "env-1",
		RollbackJobID: "job-rollback",
		Err:           errors.New(longError),
	}); err != nil {
		t.Fatalf("apply rollback failure: %v", err)
	}

	assertString(t, db, "SELECT status FROM sites WHERE id = 'site-1'", "failed")
	assertString(t, db, "SELECT status FROM environments WHERE id = 'env-1'", "failed")
	assertString(t, db, "SELECT status FROM jobs WHERE id = 'job-rollback'", "failed")
	assertString(t, db, "SELECT error_code FROM jobs WHERE id = 'job-rollback'", ReleaseRollbackErrorCodeFailed)

	var msg string
	if err := db.QueryRow("SELECT error_message FROM jobs WHERE id = 'job-rollback'").Scan(&msg); err != nil {
		t.Fatalf("query rollback error message: %v", err)
	}
	if len(msg) != 512 {
		t.Fatalf("expected truncated rollback error length 512, got %d", len(msg))
	}
}

func openReleasesTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "releases-test.db")
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
			status TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL
		);
		CREATE TABLE environments (
			id TEXT PRIMARY KEY,
			site_id TEXT NOT NULL,
			node_id TEXT NOT NULL,
			status TEXT NOT NULL,
			current_release_id TEXT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL
		);
		CREATE TABLE releases (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			health_status TEXT NOT NULL,
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

func seedReleaseGraph(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO sites (id, status, updated_at, state_version)
		VALUES ('site-1', 'active', datetime('now'), 1)
	`); err != nil {
		t.Fatalf("seed site: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO environments (id, site_id, node_id, status, current_release_id, updated_at, state_version)
		VALUES ('env-1', 'site-1', 'node-1', 'active', 'rel-new', datetime('now'), 1)
	`); err != nil {
		t.Fatalf("seed environment: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO releases (id, environment_id, health_status, created_at)
		VALUES
		  ('rel-old', 'env-1', 'healthy', datetime('now', '-2 minute')),
		  ('rel-new', 'env-1', 'unknown', datetime('now', '-1 minute'))
	`); err != nil {
		t.Fatalf("seed releases: %v", err)
	}
}

func seedHealthCheckJob(t *testing.T, db *sql.DB, jobID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (?, 'health_check', 'running', 'site-1', 'env-1', 'node-1', '{}', 0, 3, NULL, NULL, NULL, datetime('now'), NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, jobID); err != nil {
		t.Fatalf("seed health_check job: %v", err)
	}
}

func seedRollbackJob(t *testing.T, db *sql.DB, jobID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (?, 'release_rollback', 'running', 'site-1', 'env-1', 'node-1', '{}', 0, 3, NULL, NULL, NULL, datetime('now'), NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, jobID); err != nil {
		t.Fatalf("seed release_rollback job: %v", err)
	}
}

func assertString(t *testing.T, db *sql.DB, query, expected string, args ...any) {
	t.Helper()

	var got string
	if err := db.QueryRow(query, args...).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if got != expected {
		t.Fatalf("unexpected value for %q: got %q want %q", query, got, expected)
	}
}
