package registration

import (
	"database/sql"
	"errors"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestStoreConsumeReplayAndExpiry(t *testing.T) {
	db := openRegistrationTestDB(t)
	store := NewStore(db)

	token, err := store.Create(1, time.Hour)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if err := store.Validate(token, 1); err != nil {
		t.Fatalf("Validate() error = %v", err)
	}
	if err := store.Consume(token, 1); err != nil {
		t.Fatalf("Consume() error = %v", err)
	}
	if err := store.Consume(token, 1); !errors.Is(err, ErrConsumedToken) {
		t.Fatalf("Consume() replay error = %v, want %v", err, ErrConsumedToken)
	}

	expired, err := store.Create(1, time.Hour)
	if err != nil {
		t.Fatalf("Create() expired token error = %v", err)
	}
	if _, err := db.Exec(`UPDATE registration_tokens SET expires_at = '2000-01-01T00:00:00Z' WHERE token_hash = ?`, HashToken(expired)); err != nil {
		t.Fatalf("expire token: %v", err)
	}
	if err := store.Validate(expired, 1); !errors.Is(err, ErrExpiredToken) {
		t.Fatalf("Validate() expired error = %v, want %v", err, ErrExpiredToken)
	}
}

func openRegistrationTestDB(t *testing.T) *sql.DB {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}
	if _, err := db.Exec(`CREATE TABLE servers (id INTEGER PRIMARY KEY);`); err != nil {
		t.Fatalf("create servers table: %v", err)
	}
	if _, err := db.Exec(`INSERT INTO servers (id) VALUES (1)`); err != nil {
		t.Fatalf("insert server: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE registration_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id INTEGER NOT NULL REFERENCES servers(id),
			token_hash TEXT UNIQUE NOT NULL,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			expires_at TEXT NOT NULL,
			consumed_at TEXT
		)
	`); err != nil {
		t.Fatalf("create registration_tokens table: %v", err)
	}
	return db
}
