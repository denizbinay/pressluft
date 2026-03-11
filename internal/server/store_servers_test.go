package server

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"testing"
	"time"

	"pressluft/internal/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/security"

	_ "modernc.org/sqlite"
)

func TestServerStoreCreateAndList(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)

	providerID, _ := mustInsertProvider(t, db, "hetzner", "main")

	_, err := store.Create(context.Background(), CreateServerNodeInput{
		ProviderID:   providerID,
		ProviderType: "hetzner",
		Name:         "agency-prod-01",
		Location:     "fsn1",
		ServerType:   "cx22",
		Image:        "ubuntu-24.04",
		ProfileKey:   "nginx-stack",
		Status:       platform.ServerStatusProvisioning,
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

	providerID, _ := mustInsertProvider(t, db, "hetzner", "main")

	serverID, err := store.Create(context.Background(), CreateServerNodeInput{
		ProviderID:   providerID,
		ProviderType: "hetzner",
		Name:         "agency-stage-01",
		Location:     "nbg1",
		ServerType:   "cx22",
		Image:        "ubuntu-24.04",
		ProfileKey:   "woocommerce-optimized",
		Status:       platform.ServerStatusProvisioning,
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	err = store.UpdateProvisioning(context.Background(), serverID, "123456", "9876", "running", platform.ServerStatusProvisioning, "203.0.113.10", "2001:db8::10")
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
	if servers[0].IPv4 != "203.0.113.10" {
		t.Fatalf("ipv4 = %q, want %q", servers[0].IPv4, "203.0.113.10")
	}
	if servers[0].IPv6 != "2001:db8::10" {
		t.Fatalf("ipv6 = %q, want %q", servers[0].IPv6, "2001:db8::10")
	}
}

func TestServerStoreQueueServerJobUpdatesLifecycleStatus(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)
	providerID, _ := mustInsertProvider(t, db, "hetzner", "main")

	serverID, err := store.Create(context.Background(), CreateServerNodeInput{
		ProviderID:   providerID,
		ProviderType: "hetzner",
		Name:         "agency-delete-01",
		Location:     "fsn1",
		ServerType:   "cx22",
		Image:        "ubuntu-24.04",
		ProfileKey:   "nginx-stack",
		Status:       platform.ServerStatusReady,
	})
	if err != nil {
		t.Fatalf("create server: %v", err)
	}

	server, job, err := store.QueueServerJob(context.Background(), QueueServerJobInput{
		ServerID: serverID,
		Kind:     string(orchestrator.JobKindDeleteServer),
	})
	if err != nil {
		t.Fatalf("queue server job: %v", err)
	}
	if server.Status != platform.ServerStatusDeleting {
		t.Fatalf("server status = %q, want %q", server.Status, platform.ServerStatusDeleting)
	}
	if job.Status != orchestrator.JobStatusQueued {
		t.Fatalf("job status = %q, want %q", job.Status, orchestrator.JobStatusQueued)
	}
}

func TestServerStoreQueueServerJobBlocksDuplicateDestructiveActions(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)
	serverID := mustInsertServerWithStatus(t, db, string(platform.ServerStatusReady))

	if _, _, err := store.QueueServerJob(context.Background(), QueueServerJobInput{
		ServerID: serverID,
		Kind:     string(orchestrator.JobKindRebuildServer),
	}); err != nil {
		t.Fatalf("queue first destructive job: %v", err)
	}

	_, _, err := store.QueueServerJob(context.Background(), QueueServerJobInput{
		ServerID: serverID,
		Kind:     string(orchestrator.JobKindResizeServer),
	})
	if !errors.Is(err, ErrServerActionConflict) {
		t.Fatalf("err = %v, want ErrServerActionConflict", err)
	}
}

func TestServerStoreQueueServerJobRejectsDeletedServers(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)
	serverID := mustInsertServerWithStatus(t, db, string(platform.ServerStatusDeleted))

	_, _, err := store.QueueServerJob(context.Background(), QueueServerJobInput{
		ServerID: serverID,
		Kind:     string(orchestrator.JobKindRestartService),
		Payload:  `{"service_name":"nginx"}`,
	})
	if !errors.Is(err, ErrServerDeleted) {
		t.Fatalf("err = %v, want ErrServerDeleted", err)
	}
}

func TestServerStoreUpdateNodeStatusPersistsOnlineHeartbeat(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)
	serverID := mustInsertServerWithStatus(t, db, string(platform.ServerStatusReady))
	lastSeen := time.Now().UTC().Format(time.RFC3339)

	if err := store.UpdateNodeStatus(context.Background(), serverID, platform.NodeStatusOnline, lastSeen, "1.2.3"); err != nil {
		t.Fatalf("update node status: %v", err)
	}

	server, err := store.GetByID(context.Background(), serverID)
	if err != nil {
		t.Fatalf("get server: %v", err)
	}
	if server.NodeStatus != platform.NodeStatusOnline || server.NodeLastSeen != lastSeen || server.NodeVersion != "1.2.3" {
		t.Fatalf("server node state = %+v, want persisted online heartbeat", server)
	}
}

func TestServerStoreMarkNodesOfflineBeforeOnlyUpdatesStaleNodes(t *testing.T) {
	db := mustOpenTestDB(t)
	store := NewServerStore(db)
	staleID := mustInsertServerWithStatus(t, db, string(platform.ServerStatusReady))
	freshID := mustInsertServerWithStatus(t, db, string(platform.ServerStatusReady))

	staleLastSeen := time.Now().Add(-4 * time.Minute).UTC().Format(time.RFC3339)
	freshLastSeen := time.Now().Add(-10 * time.Second).UTC().Format(time.RFC3339)
	if err := store.UpdateNodeStatus(context.Background(), staleID, platform.NodeStatusUnhealthy, staleLastSeen, "v1"); err != nil {
		t.Fatalf("seed stale node: %v", err)
	}
	if err := store.UpdateNodeStatus(context.Background(), freshID, platform.NodeStatusOnline, freshLastSeen, "v2"); err != nil {
		t.Fatalf("seed fresh node: %v", err)
	}

	rows, err := store.MarkNodesOfflineBefore(context.Background(), time.Now().Add(-150*time.Second))
	if err != nil {
		t.Fatalf("mark nodes offline: %v", err)
	}
	if rows != 1 {
		t.Fatalf("rows = %d, want 1", rows)
	}

	staleServer, err := store.GetByID(context.Background(), staleID)
	if err != nil {
		t.Fatalf("get stale server: %v", err)
	}
	freshServer, err := store.GetByID(context.Background(), freshID)
	if err != nil {
		t.Fatalf("get fresh server: %v", err)
	}
	if staleServer.NodeStatus != platform.NodeStatusOffline {
		t.Fatalf("stale status = %q, want offline", staleServer.NodeStatus)
	}
	if freshServer.NodeStatus != platform.NodeStatusOnline {
		t.Fatalf("fresh status = %q, want online", freshServer.NodeStatus)
	}
}

func mustOpenTestDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`PRAGMA foreign_keys = ON`); err != nil {
		t.Fatalf("enable foreign keys: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE providers (
			id         TEXT PRIMARY KEY,
			type       TEXT    NOT NULL,
			name       TEXT    NOT NULL,
			api_token_encrypted TEXT NOT NULL,
			api_token_key_id TEXT NOT NULL,
			api_token_version INTEGER NOT NULL DEFAULT 0,
			status     TEXT    NOT NULL DEFAULT 'active',
			created_at TEXT    NOT NULL,
			updated_at TEXT    NOT NULL
		);
	`); err != nil {
		t.Fatalf("create providers table: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE servers (
			id                 TEXT PRIMARY KEY,
			provider_id        TEXT    NOT NULL,
			provider_type      TEXT    NOT NULL,
			provider_server_id TEXT,
			ipv4               TEXT,
			ipv6               TEXT,
			name               TEXT    NOT NULL,
			location           TEXT    NOT NULL,
			server_type        TEXT    NOT NULL,
			image              TEXT    NOT NULL,
			profile_key        TEXT    NOT NULL,
			status             TEXT    NOT NULL,
			setup_state        TEXT    NOT NULL DEFAULT 'not_started',
			setup_last_error   TEXT,
			action_id          TEXT,
			action_status      TEXT,
			node_status        TEXT DEFAULT 'unknown',
			node_last_seen     TEXT,
			node_version       TEXT,
			created_at         TEXT    NOT NULL,
			updated_at         TEXT    NOT NULL,
			FOREIGN KEY (provider_id) REFERENCES providers(id)
		);
	`); err != nil {
		t.Fatalf("create servers table: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE sites (
			id                TEXT PRIMARY KEY,
			server_id         TEXT    NOT NULL,
			name              TEXT    NOT NULL,
			primary_domain    TEXT,
			status            TEXT    NOT NULL DEFAULT 'draft',
			wordpress_path    TEXT,
			php_version       TEXT,
			wordpress_version TEXT,
			created_at        TEXT    NOT NULL,
			updated_at        TEXT    NOT NULL,
			FOREIGN KEY (server_id) REFERENCES servers(id)
		);
	`); err != nil {
		t.Fatalf("create sites table: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE domains (
			id               TEXT PRIMARY KEY,
			hostname         TEXT    NOT NULL,
			kind             TEXT    NOT NULL,
			ownership        TEXT    NOT NULL,
			status           TEXT    NOT NULL DEFAULT 'active',
			site_id          TEXT,
			parent_domain_id TEXT,
			is_primary       INTEGER NOT NULL DEFAULT 0,
			created_at       TEXT    NOT NULL,
			updated_at       TEXT    NOT NULL,
			FOREIGN KEY (site_id) REFERENCES sites(id),
			FOREIGN KEY (parent_domain_id) REFERENCES domains(id)
		);
	`); err != nil {
		t.Fatalf("create domains table: %v", err)
	}
	if _, err := db.Exec(`
		CREATE UNIQUE INDEX idx_domains_hostname_unique ON domains(hostname);
		CREATE INDEX idx_domains_site_id ON domains(site_id);
		CREATE INDEX idx_domains_parent_domain_id ON domains(parent_domain_id);
		CREATE INDEX idx_domains_status ON domains(status);
		CREATE INDEX idx_domains_kind ON domains(kind);
		CREATE UNIQUE INDEX idx_domains_primary_site_unique ON domains(site_id) WHERE site_id IS NOT NULL AND is_primary = 1;
	`); err != nil {
		t.Fatalf("create domain indexes: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE server_keys (
			server_id             TEXT PRIMARY KEY,
			public_key            TEXT    NOT NULL,
			private_key_encrypted TEXT    NOT NULL,
			encryption_key_id     TEXT    NOT NULL,
			created_at            TEXT    NOT NULL,
			rotated_at            TEXT,
			FOREIGN KEY (server_id) REFERENCES servers(id)
		);
	`); err != nil {
		t.Fatalf("create server_keys table: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE jobs (
			id           TEXT PRIMARY KEY,
			server_id    TEXT,
			kind         TEXT    NOT NULL,
			status       TEXT    NOT NULL,
			current_step TEXT    NOT NULL DEFAULT '',
			retry_count  INTEGER NOT NULL DEFAULT 0,
			last_error   TEXT,
			payload      TEXT,
			started_at   TEXT,
			finished_at  TEXT,
			timeout_at   TEXT,
			command_id   TEXT,
			created_at   TEXT    NOT NULL,
			updated_at   TEXT    NOT NULL,
			FOREIGN KEY (server_id) REFERENCES servers(id)
		);
		CREATE TABLE job_events (
			id         TEXT PRIMARY KEY,
			job_id     TEXT    NOT NULL,
			seq        INTEGER NOT NULL,
			event_type TEXT    NOT NULL,
			level      TEXT    NOT NULL,
			step_key   TEXT,
			status     TEXT,
			message    TEXT    NOT NULL,
			payload    TEXT,
			created_at TEXT    NOT NULL,
			FOREIGN KEY (job_id) REFERENCES jobs(id)
		);
	`); err != nil {
		t.Fatalf("create jobs tables: %v", err)
	}

	return db
}

func mustInsertProvider(t *testing.T, db *sql.DB, providerType, name string) (string, string) {
	t.Helper()

	publicID := nextTestPublicID(t, db, "providers")
	encrypted, keyID, version, err := security.EncryptProviderToken("secret-token")
	if err != nil {
		t.Fatalf("encrypt provider token: %v", err)
	}

	_, err = db.Exec(
		`INSERT INTO providers (id, type, name, api_token_encrypted, api_token_key_id, api_token_version, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, 'active', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		publicID,
		providerType,
		name,
		encrypted,
		keyID,
		version,
	)
	if err != nil {
		t.Fatalf("insert provider: %v", err)
	}

	return publicID, publicID
}

func mustInsertServerWithStatus(t *testing.T, db *sql.DB, status string) string {
	t.Helper()
	providerID, _ := mustInsertProvider(t, db, "hetzner", "secondary")
	publicID := nextTestPublicID(t, db, "servers")
	_, err := db.Exec(
		`INSERT INTO servers (id, provider_id, provider_type, name, location, server_type, image, profile_key, status, setup_state, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'ready', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		publicID,
		providerID,
		"hetzner",
		"server-under-test",
		"fsn1",
		"cx22",
		"ubuntu-24.04",
		"nginx-stack",
		status,
	)
	if err != nil {
		t.Fatalf("insert server: %v", err)
	}
	return publicID
}

func nextTestPublicID(t *testing.T, db *sql.DB, table string) string {
	t.Helper()
	var count int
	if err := db.QueryRow(`SELECT COUNT(*) FROM ` + table).Scan(&count); err != nil {
		t.Fatalf("count %s rows: %v", table, err)
	}
	return fmt.Sprintf("00000000-0000-7000-8000-%012d", count+1)
}
