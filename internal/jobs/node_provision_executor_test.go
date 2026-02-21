package jobs

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"strings"
	"testing"

	"pressluft/internal/store"
)

func TestExecuteQueuedNodeProvisionSuccess(t *testing.T) {
	t.Parallel()

	db := openJobsTestDB(t)
	seedNodeProvisionJob(t, db, "node-1")

	runner := stubRunner{output: "ok"}
	executed, err := ExecuteQueuedNodeProvision(context.Background(), db, runner, "ansible/playbooks/node-provision.yml")
	if err != nil {
		t.Fatalf("execute queued node provision: %v", err)
	}
	if !executed {
		t.Fatalf("expected a job to execute")
	}

	assertString(t, db, "SELECT status FROM jobs WHERE id = 'job-1'", "succeeded")
	assertString(t, db, "SELECT status FROM nodes WHERE id = 'node-1'", "active")
}

func TestExecuteQueuedNodeProvisionFailureSetsErrorCodeAndTruncates(t *testing.T) {
	t.Parallel()

	db := openJobsTestDB(t)
	seedNodeProvisionJob(t, db, "node-1")

	longOutput := strings.Repeat("x", 600)
	runner := stubRunner{output: longOutput, err: errors.New("ansible failed")}
	_, err := ExecuteQueuedNodeProvision(context.Background(), db, runner, "ansible/playbooks/node-provision.yml")
	if err == nil {
		t.Fatalf("expected executor to return error")
	}

	assertString(t, db, "SELECT status FROM jobs WHERE id = 'job-1'", "failed")
	assertString(t, db, "SELECT status FROM nodes WHERE id = 'node-1'", "unreachable")
	assertString(t, db, "SELECT error_code FROM jobs WHERE id = 'job-1'", nodeProvisionErrorCodeFailed)

	var msg string
	if err := db.QueryRow("SELECT error_message FROM jobs WHERE id = 'job-1'").Scan(&msg); err != nil {
		t.Fatalf("query error message: %v", err)
	}
	if len(msg) != 512 {
		t.Fatalf("expected truncated error length 512, got %d", len(msg))
	}
}

func TestExecuteQueuedNodeProvisionTimeoutSetsTimeoutCode(t *testing.T) {
	t.Parallel()

	db := openJobsTestDB(t)
	seedNodeProvisionJob(t, db, "node-1")

	runner := stubRunner{err: context.DeadlineExceeded}
	_, err := ExecuteQueuedNodeProvision(context.Background(), db, runner, "ansible/playbooks/node-provision.yml")
	if err == nil {
		t.Fatalf("expected executor to return error")
	}

	assertString(t, db, "SELECT error_code FROM jobs WHERE id = 'job-1'", nodeProvisionErrorCodeTimeout)
}

func TestEnqueueMutationJobRejectsConcurrentNodeAndSiteMutations(t *testing.T) {
	t.Parallel()

	db := openJobsTestDB(t)

	err := store.WithTx(context.Background(), db, func(tx *sql.Tx) error {
		first := MutationJobInput{
			JobID:       "job-node-1",
			JobType:     "node_provision",
			NodeID:      sql.NullString{String: "node-1", Valid: true},
			PayloadJSON: `{}`,
		}
		if err := EnqueueMutationJob(context.Background(), tx, first); err != nil {
			return err
		}

		second := MutationJobInput{
			JobID:       "job-node-2",
			JobType:     "node_provision",
			NodeID:      sql.NullString{String: "node-1", Valid: true},
			PayloadJSON: `{}`,
		}
		if err := EnqueueMutationJob(context.Background(), tx, second); !errors.Is(err, ErrConcurrencyConflict) {
			return errors.New("expected node concurrency conflict")
		}

		third := MutationJobInput{
			JobID:       "job-site-1",
			JobType:     "site_create",
			SiteID:      sql.NullString{String: "site-1", Valid: true},
			PayloadJSON: `{}`,
		}
		if err := EnqueueMutationJob(context.Background(), tx, third); err != nil {
			return err
		}

		fourth := MutationJobInput{
			JobID:       "job-site-2",
			JobType:     "site_import",
			SiteID:      sql.NullString{String: "site-1", Valid: true},
			PayloadJSON: `{}`,
		}
		if err := EnqueueMutationJob(context.Background(), tx, fourth); !errors.Is(err, ErrConcurrencyConflict) {
			return errors.New("expected site concurrency conflict")
		}

		return nil
	})
	if err != nil {
		t.Fatalf("assert enqueue concurrency: %v", err)
	}
}

type stubRunner struct {
	output string
	err    error
}

func (s stubRunner) Run(_ context.Context, _ string, _ ...string) (string, error) {
	return s.output, s.err
}

func openJobsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "jobs-test.db")
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
		t.Fatalf("create tables: %v", err)
	}

	if _, err := db.Exec(`
		INSERT INTO nodes (id, name, hostname, public_ip, ssh_port, ssh_user, status, is_local, last_seen_at, created_at, updated_at, state_version)
		VALUES ('node-1', 'local-node', 'localhost', NULL, 22, 'root', 'provisioning', 1, NULL, datetime('now'), datetime('now'), 1)
	`); err != nil {
		t.Fatalf("seed node: %v", err)
	}

	return db
}

func seedNodeProvisionJob(t *testing.T, db *sql.DB, nodeID string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		) VALUES ('job-1', 'node_provision', 'queued', NULL, NULL, ?, '{}', 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, nodeID); err != nil {
		t.Fatalf("seed node_provision job: %v", err)
	}
}

func assertString(t *testing.T, db *sql.DB, query, expected string) {
	t.Helper()

	var got string
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if got != expected {
		t.Fatalf("unexpected value for %q: got %q want %q", query, got, expected)
	}
}
