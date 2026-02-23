package server

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"strconv"
	"sync"
	"testing"

	_ "modernc.org/sqlite"

	"pressluft/internal/provider"
)

var registerServerProviderOnce sync.Once

func TestServersCatalogEndpoint(t *testing.T) {
	registerTestServerProvider()

	db := mustOpenServerHandlerDB(t)
	providerID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")

	handler := NewHandler(db)
	path := "/api/servers/catalog?provider_id=" + intToString(providerID)
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
	providerID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")

	handler := NewHandler(db)
	body := map[string]any{
		"provider_id": providerID,
		"name":        "agency-prod-01",
		"location":    "fsn1",
		"server_type": "cx22",
		"image":       "ubuntu-24.04",
		"profile_key": "nginx-stack",
	}
	bodyBytes, _ := json.Marshal(body)
	req := httptest.NewRequest(http.MethodPost, "/api/servers", bytes.NewReader(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	res := httptest.NewRecorder()

	handler.ServeHTTP(res, req)

	if res.Code != http.StatusAccepted {
		t.Fatalf("status = %d, want %d", res.Code, http.StatusAccepted)
	}

	servers, err := NewServerStore(db).List(context.Background())
	if err != nil {
		t.Fatalf("list servers: %v", err)
	}
	if len(servers) != 1 {
		t.Fatalf("server count = %d, want %d", len(servers), 1)
	}
	if servers[0].Status != "provisioning" {
		t.Fatalf("server status = %q, want %q", servers[0].Status, "provisioning")
	}
}

func TestServersCreateEndpointValidationFailure(t *testing.T) {
	registerTestServerProvider()

	db := mustOpenServerHandlerDB(t)
	providerID := mustInsertProviderRecord(t, db, "test-server-provider", "agency", "token-ok")

	handler := NewHandler(db)
	body := map[string]any{
		"provider_id": providerID,
		"name":        "agency-prod-01",
		"location":    "fsn1",
		"server_type": "cx22",
		"image":       "ubuntu-24.04",
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
		}},
		Images: []provider.ServerImageOption{{Name: "ubuntu-24.04", Description: "Ubuntu 24.04"}},
	}, nil
}

func (t *testServerProvider) CreateServer(_ context.Context, token string, _ provider.CreateServerRequest) (*provider.CreateServerResult, error) {
	if token == "fail-token" {
		return nil, errors.New("forced create failure")
	}
	return &provider.CreateServerResult{
		ProviderServerID: "srv-test-1",
		ActionID:         "act-test-1",
		Status:           "running",
	}, nil
}

func mustOpenServerHandlerDB(t *testing.T) *sql.DB {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

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

func mustInsertProviderRecord(t *testing.T, db *sql.DB, providerType, name, token string) int64 {
	t.Helper()

	res, err := db.Exec(
		`INSERT INTO providers (type, name, api_token, status, created_at, updated_at)
		 VALUES (?, ?, ?, 'active', '2026-01-01T00:00:00Z', '2026-01-01T00:00:00Z')`,
		providerType,
		name,
		token,
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

func intToString(v int64) string {
	return strconv.FormatInt(v, 10)
}
