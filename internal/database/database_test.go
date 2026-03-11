package database

import (
	"database/sql"
	"io"
	"log/slog"
	"path/filepath"
	"testing"

	_ "modernc.org/sqlite"
)

func TestOpenCreatesParentDirAndRunsLatestMigrations(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dbPath := filepath.Join(t.TempDir(), "nested", "pressluft.db")

	db, err := Open(dbPath, logger)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	requireTable(t, db.DB, "goose_db_version")
	requireTable(t, db.DB, "jobs")
	requireTable(t, db.DB, "job_events")
	requireTable(t, db.DB, "registration_tokens")
	requireTable(t, db.DB, "agent_ws_tokens")
	requireTable(t, db.DB, "ca_certificates")
	requireTable(t, db.DB, "node_certificates")
	requireTable(t, db.DB, "sites")
	requireTable(t, db.DB, "domains")

	requireColumn(t, db.DB, "jobs", "payload")
	requireColumn(t, db.DB, "jobs", "started_at")
	requireColumn(t, db.DB, "jobs", "finished_at")
	requireColumn(t, db.DB, "jobs", "timeout_at")
	requireColumn(t, db.DB, "jobs", "command_id")

	requireTableMissing(t, db.DB, "job_steps")
	requireTableMissing(t, db.DB, "job_checkpoints")
}

func TestOpenAppliesSQLitePragmas(t *testing.T) {
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))
	dbPath := filepath.Join(t.TempDir(), "pressluft.db")

	db, err := Open(dbPath, logger)
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if got := queryPragma(t, db.DB, "foreign_keys"); got != "1" {
		t.Fatalf("PRAGMA foreign_keys = %q, want 1", got)
	}
	if got := queryPragma(t, db.DB, "busy_timeout"); got != "5000" {
		t.Fatalf("PRAGMA busy_timeout = %q, want 5000", got)
	}
	if got := queryPragma(t, db.DB, "synchronous"); got != "1" {
		t.Fatalf("PRAGMA synchronous = %q, want 1 (NORMAL)", got)
	}
	if got := queryPragma(t, db.DB, "journal_mode"); got != "wal" {
		t.Fatalf("PRAGMA journal_mode = %q, want wal", got)
	}
}

func TestOpenBackfillsLegacyJobRuntimeColumns(t *testing.T) {
	dbPath := filepath.Join(t.TempDir(), "pressluft.db")

	legacyDB, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("open legacy database: %v", err)
	}

	legacyStatements := []string{
		`CREATE TABLE jobs (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id INTEGER,
			kind TEXT NOT NULL,
			status TEXT NOT NULL,
			current_step TEXT NOT NULL DEFAULT '',
			retry_count INTEGER NOT NULL DEFAULT 0,
			last_error TEXT,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			payload TEXT,
			command_id TEXT
		)`,
	}
	for _, stmt := range legacyStatements {
		if _, err := legacyDB.Exec(stmt); err != nil {
			_ = legacyDB.Close()
			t.Fatalf("seed legacy schema: %v", err)
		}
	}
	if err := legacyDB.Close(); err != nil {
		t.Fatalf("close legacy database: %v", err)
	}

	db, err := sql.Open("sqlite", dbPath)
	if err != nil {
		t.Fatalf("reopen legacy database: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if err := reconcileLegacySchema(db); err != nil {
		t.Fatalf("reconcileLegacySchema() error = %v", err)
	}

	requireColumn(t, db, "jobs", "started_at")
	requireColumn(t, db, "jobs", "finished_at")
	requireColumn(t, db, "jobs", "timeout_at")
	requireColumn(t, db, "jobs", "payload")
	requireColumn(t, db, "jobs", "command_id")
}

func requireTable(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var name string
	if err := db.QueryRow(`SELECT name FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&name); err != nil {
		t.Fatalf("table %q missing: %v", table, err)
	}
}

func requireTableMissing(t *testing.T, db *sql.DB, table string) {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = ?`, table).Scan(&count); err != nil {
		t.Fatalf("query missing table %q: %v", table, err)
	}
	if count != 0 {
		t.Fatalf("expected table %q to be absent", table)
	}
}

func requireColumn(t *testing.T, db *sql.DB, table, column string) {
	t.Helper()
	rows, err := db.Query(`PRAGMA table_info(` + table + `)`)
	if err != nil {
		t.Fatalf("PRAGMA table_info(%s): %v", table, err)
	}
	defer rows.Close()

	for rows.Next() {
		var cid int
		var name, dataType string
		var notNull, pk int
		var defaultValue any
		if err := rows.Scan(&cid, &name, &dataType, &notNull, &defaultValue, &pk); err != nil {
			t.Fatalf("scan pragma row: %v", err)
		}
		if name == column {
			return
		}
	}
	if err := rows.Err(); err != nil {
		t.Fatalf("iterate pragma rows: %v", err)
	}
	t.Fatalf("column %q missing from table %q", column, table)
}

func queryPragma(t *testing.T, db *sql.DB, pragma string) string {
	t.Helper()
	var value string
	if err := db.QueryRow(`PRAGMA ` + pragma).Scan(&value); err != nil {
		t.Fatalf("PRAGMA %s: %v", pragma, err)
	}
	return value
}
