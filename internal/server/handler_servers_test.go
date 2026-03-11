package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	"pressluft/internal/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/security"
	"pressluft/internal/server/profiles"

	_ "modernc.org/sqlite"

	"pressluft/internal/provider"
)

var registerServerProviderOnce sync.Once

func TestServersCatalogEndpoint(t *testing.T) {
	registerTestServerProvider()

	db := mustOpenServerHandlerDB(t)
	providerID, _ := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")

	handler := NewHandler(db)
	path := "/api/servers/catalog?provider_id=" + providerID
	req := httptest.NewRequest(http.MethodGet, path, nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusOK)
	}

	var payload struct {
		Catalog  provider.ServerCatalog `json:"catalog"`
		Profiles []any                  `json:"profiles"`
	}
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if len(payload.Catalog.Locations) == 0 {
		t.Fatal("expected locations in catalog")
	}
	if len(payload.Profiles) == 0 {
		t.Fatal("expected profiles in response")
	}
}

func TestServersCreateEndpoint(t *testing.T) {
	registerTestServerProvider()

	db := mustOpenServerHandlerDB(t)
	providerID, _ := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")

	handler := NewHandler(db)
	body := map[string]any{
		"provider_id": providerID,
		"name":        "agency-prod-01",
		"location":    "fsn1",
		"server_type": "cx22",
		"profile_key": "nginx-stack",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/servers", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusAccepted, res.Body.String())
	}

	// Verify response contains server_id and job_id
	var respBody map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &respBody); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if _, ok := respBody["server_id"]; !ok {
		t.Fatal("response missing server_id")
	}
	if _, ok := respBody["job_id"]; !ok {
		t.Fatal("response missing job_id")
	}

	// Server should be in pending state (job will transition it)
	servers, err := NewServerStore(db).List(context.Background())
	if err != nil {
		t.Fatalf("list servers: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("server count = %d, want %d", len(servers), 1)
	}
	if servers[0].Status != platform.ServerStatusPending {
		t.Fatalf("server status = %q, want %q", servers[0].Status, platform.ServerStatusPending)
	}
}

func TestServersCreateEndpointValidationFailure(t *testing.T) {
	registerTestServerProvider()

	db := mustOpenServerHandlerDB(t)
	providerID, _ := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")

	handler := NewHandler(db)
	body := map[string]any{
		"provider_id": providerID,
		"name":        "agency-prod-01",
		"location":    "fsn1",
		"server_type": "cx22",
		"profile_key": "unknown-profile",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/servers", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
}

func TestServersCreateEndpointRejectsUnavailableProfile(t *testing.T) {
	registerTestServerProvider()

	db := mustOpenServerHandlerDB(t)
	providerID, _ := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")

	handler := NewHandler(db)
	body := map[string]any{
		"provider_id": providerID,
		"name":        "agency-prod-01",
		"location":    "fsn1",
		"server_type": "cx22",
		"profile_key": "openlitespeed-stack",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/servers", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusBadRequest {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusBadRequest)
	}
	profile, ok := profiles.Get("openlitespeed-stack")
	if !ok {
		t.Fatalf("expected openlitespeed-stack profile to exist")
	}
	if profile.SupportLevel != platform.SupportLevelUnavailable {
		t.Fatalf("SupportLevel = %q, want %q", profile.SupportLevel, platform.SupportLevelUnavailable)
	}
	if !bytes.Contains(res.Body.Bytes(), []byte("automated setup and verification")) {
		t.Fatalf("response body = %q, want support-matrix reason", res.Body.String())
	}
}

func TestServersDeleteEndpointQueuesAsyncDeletion(t *testing.T) {
	registerTestServerProvider()

	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, string(platform.ServerStatusReady))

	handler := NewHandler(db)
	req := httptest.NewRequest(http.MethodDelete, "/api/servers/"+serverID, nil)
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusAccepted, res.Body.String())
	}

	var payload map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	if payload["status"] != string(platform.ServerStatusDeleting) {
		t.Fatalf("status payload = %v, want %q", payload["status"], platform.ServerStatusDeleting)
	}

	server, err := NewServerStore(db).GetByID(context.Background(), serverID)
	if err != nil {
		t.Fatalf("get server: %v", err)
	}
	if server.Status != platform.ServerStatusDeleting {
		t.Fatalf("server status = %q, want %q", server.Status, platform.ServerStatusDeleting)
	}

	jobs, err := orchestrator.NewStore(db).ListJobsByServer(context.Background(), serverID)
	if err != nil {
		t.Fatalf("list jobs: %v", err)
	}
	if len(jobs) != 1 || jobs[0].Kind != "delete_server" {
		t.Fatalf("jobs = %+v, want one delete_server job", jobs)
	}
	if jobs[0].Status != orchestrator.JobStatusQueued {
		t.Fatalf("job status = %q, want %q", jobs[0].Status, orchestrator.JobStatusQueued)
	}
}

func TestServersDeleteEndpointRejectsDuplicateDeletion(t *testing.T) {
	registerTestServerProvider()

	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, string(platform.ServerStatusReady))

	handler := NewHandler(db)
	firstReq := httptest.NewRequest(http.MethodDelete, "/api/servers/"+serverID, nil)
	firstRes := httptest.NewRecorder()
	handler.ServeHTTP(firstRes, firstReq)
	if firstRes.Code != http.StatusAccepted {
		t.Fatalf("first delete status = %d, want %d", firstRes.Code, http.StatusAccepted)
	}

	secondReq := httptest.NewRequest(http.MethodDelete, "/api/servers/"+serverID, nil)
	secondRes := httptest.NewRecorder()
	handler.ServeHTTP(secondRes, secondReq)

	if secondRes.Code != http.StatusConflict {
		t.Fatalf("status = %d, want %d; body = %s", secondRes.Code, http.StatusConflict, secondRes.Body.String())
	}
}

func TestAllAgentStatusIncludesStoredOfflineNodes(t *testing.T) {
	registerTestServerProvider()

	db := mustOpenServerHandlerDB(t)
	_, providerDBID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")
	serverID := mustInsertServerRecord(t, db, providerDBID, string(platform.ServerStatusReady))
	store := NewServerStore(db)
	if err := store.UpdateNodeStatus(context.Background(), serverID, platform.NodeStatusOffline, "2026-01-01T00:00:00Z", "1.0.0"); err != nil {
		t.Fatalf("update node status: %v", err)
	}

	handler := NewHandler(db)
	req := httptest.NewRequest(http.MethodGet, "/api/servers/agents", nil)
	res := httptest.NewRecorder()
	handler.ServeHTTP(res, req)

	if res.Code != http.StatusOK {
		t.Fatalf("status = %d, want %d; body = %s", res.Code, http.StatusOK, res.Body.String())
	}

	var payload map[string]map[string]any
	if err := json.Unmarshal(res.Body.Bytes(), &payload); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	entry, ok := payload[serverID]
	if !ok {
		t.Fatalf("payload missing server %s: %+v", serverID, payload)
	}
	if entry["status"] != "offline" {
		t.Fatalf("status = %v, want offline", entry["status"])
	}
}

func registerTestServerProvider() {
	registerServerProviderOnce.Do(func() {
		provider.Register(&testServerProvider{})
	})
}

type testServerProvider struct{}

func (t *testServerProvider) Info() provider.Info {
	return provider.Info{Type: "test-server-provider", Name: "Test Server Provider"}
}

func (t *testServerProvider) Validate(context.Context, string) (*provider.ValidationResult, error) {
	return &provider.ValidationResult{Valid: true, ReadWrite: true, Message: "ok"}, nil
}

func (t *testServerProvider) ListServerCatalog(context.Context, string) (*provider.ServerCatalog, error) {
	return &provider.ServerCatalog{
		Locations: []provider.ServerLocation{{Name: "fsn1", Description: "Falkenstein"}},
		ServerTypes: []provider.ServerTypeOption{{
			Name: "cx22", Description: "Standard", Cores: 2, MemoryGB: 4, DiskGB: 40,
			AvailableAt: []string{"fsn1"},
		}},
	}, nil
}

func mustOpenServerHandlerDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	db.SetMaxOpenConns(1)
	t.Cleanup(func() { _ = db.Close() })

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
			status       TEXT    NOT NULL DEFAULT 'queued',
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
	`); err != nil {
		t.Fatalf("create jobs table: %v", err)
	}

	if _, err := db.Exec(`
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
		t.Fatalf("create job_events table: %v", err)
	}

	if _, err := db.Exec(`
		CREATE TABLE activity (
			id                   TEXT PRIMARY KEY,
			event_type           TEXT    NOT NULL,
			category             TEXT    NOT NULL,
			level                TEXT    NOT NULL,
			resource_type        TEXT,
			resource_id          TEXT,
			parent_resource_type TEXT,
			parent_resource_id   TEXT,
			actor_type           TEXT    NOT NULL,
			actor_id             TEXT,
			title                TEXT    NOT NULL,
			message              TEXT,
			payload              TEXT,
			requires_attention   INTEGER NOT NULL DEFAULT 0,
			read_at              TEXT,
			created_at           TEXT    NOT NULL
		);
	`); err != nil {
		t.Fatalf("create activity table: %v", err)
	}

	return db
}

func mustInsertProviderRecord(t *testing.T, db *sql.DB, providerType, name, token string) (string, string) {
	t.Helper()

	publicID := testPublicID(1)
	encrypted, keyID, version, err := security.EncryptProviderToken(token)
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

func intToString(v int64) string {
	return strconv.FormatInt(v, 10)
}

func mustInsertServerRecord(t *testing.T, db *sql.DB, providerID string, status string) string {
	t.Helper()
	publicID := testPublicID(2)
	_, err := db.Exec(
		`INSERT INTO servers (id, provider_id, provider_type, name, location, server_type, image, profile_key, status, setup_state, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, 'ready', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		publicID,
		providerID,
		"test-server-provider",
		"agency-prod-01",
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

func testPublicID(seq int) string {
	return fmt.Sprintf("00000000-0000-7000-8000-%012d", seq)
}
