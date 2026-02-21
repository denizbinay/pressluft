package metrics

import (
	"context"
	"database/sql"
	"path/filepath"
	"testing"

	"pressluft/internal/store"
)

func TestSnapshotAggregatesCounters(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	svc := NewService(db)

	if _, err := db.Exec(`
		INSERT INTO jobs (id, status) VALUES ('job-1', 'running'), ('job-2', 'queued'), ('job-3', 'succeeded');
		INSERT INTO nodes (id, status) VALUES ('node-1', 'active'), ('node-2', 'unreachable');
		INSERT INTO sites (id) VALUES ('site-1'), ('site-2'), ('site-3');
	`); err != nil {
		t.Fatalf("seed counters: %v", err)
	}

	snapshot, err := svc.Snapshot(context.Background())
	if err != nil {
		t.Fatalf("snapshot: %v", err)
	}

	if snapshot.JobsRunning != 1 || snapshot.JobsQueued != 1 || snapshot.NodesActive != 1 || snapshot.SitesTotal != 3 {
		t.Fatalf("unexpected snapshot: %+v", snapshot)
	}
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "metrics-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE jobs (id TEXT PRIMARY KEY, status TEXT NOT NULL);
		CREATE TABLE nodes (id TEXT PRIMARY KEY, status TEXT NOT NULL);
		CREATE TABLE sites (id TEXT PRIMARY KEY);
	`); err != nil {
		t.Fatalf("create tables: %v", err)
	}

	return db
}
