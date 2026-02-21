package bootstrap

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"pressluft/internal/store"
)

func TestRunRegistersFirstLocalNodeAndQueuesProvisionJob(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)

	result, err := Run(context.Background(), db, "localhost")
	if err != nil {
		t.Fatalf("run bootstrap: %v", err)
	}

	if result.NodeID == "" {
		t.Fatalf("expected node id")
	}
	if !result.NodeCreated {
		t.Fatalf("expected node to be created")
	}
	if !result.NodeProvisionJob {
		t.Fatalf("expected node_provision job to be queued")
	}
	if result.NodeCurrentStatus != "provisioning" {
		t.Fatalf("unexpected node status: %s", result.NodeCurrentStatus)
	}

	assertCount(t, db, "SELECT COUNT(1) FROM nodes WHERE is_local = 1", 1)
	assertCount(t, db, "SELECT COUNT(1) FROM jobs WHERE job_type = 'node_provision' AND status = 'queued'", 1)
}

func TestRunIsIdempotentForNodeAndProvisionJob(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)

	first, err := Run(context.Background(), db, "localhost")
	if err != nil {
		t.Fatalf("first run bootstrap: %v", err)
	}

	second, err := Run(context.Background(), db, "localhost")
	if err != nil {
		t.Fatalf("second run bootstrap: %v", err)
	}

	if first.NodeID != second.NodeID {
		t.Fatalf("expected same node id, got %s and %s", first.NodeID, second.NodeID)
	}
	if second.NodeCreated {
		t.Fatalf("expected second run not to create node")
	}
	if second.NodeProvisionJob {
		t.Fatalf("expected second run not to enqueue duplicate provision job")
	}

	assertCount(t, db, "SELECT COUNT(1) FROM nodes WHERE is_local = 1", 1)
	assertCount(t, db, "SELECT COUNT(1) FROM jobs WHERE job_type = 'node_provision' AND status IN ('queued', 'running')", 1)
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "pressluft-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS nodes (
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
		CREATE TABLE IF NOT EXISTS jobs (
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

func assertCount(t *testing.T, db *sql.DB, query string, expected int) {
	t.Helper()

	var got int
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query count: %v", err)
	}

	if got != expected {
		t.Fatalf("unexpected count for %q: got %d want %d", query, got, expected)
	}
}
