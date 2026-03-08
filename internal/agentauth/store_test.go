package agentauth

import (
	"database/sql"
	"strings"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestStoreValidateAndLookupServerIDUpdatesLastUsed(t *testing.T) {
	db := openAgentAuthTestDB(t)
	store := NewStore(db)

	token, err := store.Create(7, time.Hour)
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	serverID, err := store.ValidateAndLookupServerID(token)
	if err != nil {
		t.Fatalf("ValidateAndLookupServerID() error = %v", err)
	}
	if serverID != 7 {
		t.Fatalf("serverID = %d, want 7", serverID)
	}

	var lastUsed sql.NullString
	if err := db.QueryRow(`SELECT last_used_at FROM agent_ws_tokens WHERE token_hash = ?`, HashToken(token)).Scan(&lastUsed); err != nil {
		t.Fatalf("query last_used_at: %v", err)
	}
	if !lastUsed.Valid || strings.TrimSpace(lastUsed.String) == "" {
		t.Fatal("expected last_used_at to be populated")
	}
}

func TestStoreValidateAndLookupServerIDRejectsRevokedAndExpiredTokens(t *testing.T) {
	db := openAgentAuthTestDB(t)
	store := NewStore(db)

	revokedToken, err := store.Create(7, time.Hour)
	if err != nil {
		t.Fatalf("Create() revoked token error = %v", err)
	}
	if _, err := db.Exec(`UPDATE agent_ws_tokens SET revoked_at = datetime('now') WHERE token_hash = ?`, HashToken(revokedToken)); err != nil {
		t.Fatalf("revoke token: %v", err)
	}
	if _, err := store.ValidateAndLookupServerID(revokedToken); err == nil {
		t.Fatal("expected revoked token lookup to fail")
	}

	expiredToken, err := store.Create(7, time.Hour)
	if err != nil {
		t.Fatalf("Create() expired token error = %v", err)
	}
	if _, err := db.Exec(`UPDATE agent_ws_tokens SET expires_at = '2000-01-01T00:00:00Z' WHERE token_hash = ?`, HashToken(expiredToken)); err != nil {
		t.Fatalf("expire token: %v", err)
	}
	if _, err := store.ValidateAndLookupServerID(expiredToken); err == nil {
		t.Fatal("expected expired token lookup to fail")
	}
}

func openAgentAuthTestDB(t *testing.T) *sql.DB {
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
	if _, err := db.Exec(`INSERT INTO servers (id) VALUES (7)`); err != nil {
		t.Fatalf("insert server: %v", err)
	}
	if _, err := db.Exec(`
		CREATE TABLE agent_ws_tokens (
			id INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id INTEGER NOT NULL REFERENCES servers(id),
			token_hash TEXT UNIQUE NOT NULL,
			created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')),
			expires_at TEXT NOT NULL,
			revoked_at TEXT,
			last_used_at TEXT
		)
	`); err != nil {
		t.Fatalf("create agent_ws_tokens table: %v", err)
	}
	return db
}
