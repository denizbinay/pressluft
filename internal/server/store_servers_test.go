package server

import (
	"context"
	"database/sql"
	"strings"
	"testing"

	_ "modernc.org/sqlite"
)

func TestServerStoreCreateAndList(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)

	providerID := mustInsertProvider(t, db, "hetzner", "main")

	_, err := store.Create(context.Background(), CreateServerNodeInput{
		ProviderID:   providerID,
		ProviderType: "hetzner",
		Name:         "agency-prod-01",
		Location:     "fsn1",
		ServerType:   "cx22",
		Image:        "ubuntu-24.04",
		ProfileKey:   "nginx-stack",
		Status:       "provisioning",
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	servers, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list servers: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("server count = %d, want %d", len(servers), 1)
	}

	if servers[0].Name != "agency-prod-01" {
		t.Fatalf("server name = %q, want %q", servers[0].Name, "agency-prod-01")
	}
	if servers[0].ProfileKey != "nginx-stack" {
		t.Fatalf("profile key = %q, want %q", servers[0].ProfileKey, "nginx-stack")
	}
}

func TestServerStoreCreateValidatesInput(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)

	_, err := store.Create(context.Background(), CreateServerNodeInput{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}
	if !strings.Contains(err.Error(), "provider_id") {
		t.Fatalf("error = %q, want provider_id validation", err.Error())
	}
}

func TestServerStoreUpdateProvisioning(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)

	providerID := mustInsertProvider(t, db, "hetzner", "main")

	serverID, err := store.Create(context.Background(), CreateServerNodeInput{
		ProviderID:   providerID,
		ProviderType: "hetzner",
		Name:         "agency-stage-01",
		Location:     "nbg1",
		ServerType:   "cx22",
		Image:        "ubuntu-24.04",
		ProfileKey:   "woocommerce-optimized",
		Status:       "provisioning",
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	err = store.UpdateProvisioning(context.Background(), serverID, "123456", "9876", "running", "provisioning")
	if err != nil {
		t.Fatalf("update provisioning: %v", err)
	}

	servers, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("list servers: %v", err)
	}
	if servers[0].ProviderServerID != "123456" {
		t.Fatalf("provider_server_id = %q, want %q", servers[0].ProviderServerID, "123456")
	}
	if servers[0].ActionID != "9876" {
		t.Fatalf("action_id = %q, want %q", servers[0].ActionID, "9876")
	}
}

func mustOpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE providers (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			type       TEXT    NOT NULL,
			name       TEXT    NOT NULL,
			api_token  TEXT    NOT NULL,
			status     TEXT    NOT NULL DEFAULT 'active',
			created_at TEXT    NOT NULL,
			updated_at TEXT    NOT NULL
		);
	`); err != nil {
		t.Fatalf("create providers table: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE servers (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			provider_id        INTEGER NOT NULL,
			provider_type      TEXT    NOT NULL,
			provider_server_id TEXT,
			name               TEXT    NOT NULL,
			location           TEXT    NOT NULL,
			server_type        TEXT    NOT NULL,
			image              TEXT    NOT NULL,
			profile_key        TEXT    NOT NULL,
			status             TEXT    NOT NULL,
			action_id          TEXT,
			action_status      TEXT,
			created_at         TEXT    NOT NULL,
			updated_at         TEXT    NOT NULL,
			FOREIGN KEY (provider_id) REFERENCES providers(id)
		);
	`); err != nil {
		t.Fatalf("create servers table: %v", err)
	}

	return db
}

func mustInsertProvider(t *testing.T, db *sql.DB, providerType, name string) int64 {
	t.Helper()

	res, err := db.Exec(
		`INSERT INTO providers (type, name, api_token, status, created_at, updated_at)
		 VALUES (?, ?, ?, 'active', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		providerType,
		name,
		"secret-token",
	)
	if err != nil {
		t.Fatalf("insert provider: %v", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		t.Fatalf("provider insert id: %v", err)
	}

	return id
}
