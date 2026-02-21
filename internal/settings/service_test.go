package settings

import (
	"context"
	"database/sql"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"pressluft/internal/secrets"
	"pressluft/internal/store"
)

func TestUpdateDomainConfigPersistsEncryptedSecretReference(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	secretsDir := filepath.Join(t.TempDir(), "secrets")
	svc := NewService(db, secrets.NewStore(secretsDir))

	cfg, err := svc.UpdateDomainConfig(context.Background(), UpdateDomainConfigInput{
		ControlPlaneDomain: stringPtr("panel.example.com"),
		PreviewDomain:      stringPtr("wp.example.com"),
		DNS01Provider:      stringPtr("cloudflare"),
		DNS01CredentialsJSON: map[string]string{
			"CF_DNS_API_TOKEN": "top-secret-token",
		},
	})
	if err != nil {
		t.Fatalf("update domain config: %v", err)
	}

	if !cfg.DNS01CredentialsExists {
		t.Fatalf("expected credentials configured")
	}
	if cfg.DNS01CredentialsJSON != nil {
		t.Fatalf("expected redacted dns01_credentials_json")
	}

	var credentialsRef string
	if err := db.QueryRow("SELECT value FROM settings WHERE key = 'dns01_credentials_json'").Scan(&credentialsRef); err != nil {
		t.Fatalf("query dns credentials setting: %v", err)
	}
	if !strings.HasPrefix(credentialsRef, "secret://") {
		t.Fatalf("expected secret reference, got %q", credentialsRef)
	}

	secretPath := filepath.Join(secretsDir, strings.TrimPrefix(credentialsRef, "secret://")+".enc")
	secretData, err := os.ReadFile(secretPath)
	if err != nil {
		t.Fatalf("read secret file: %v", err)
	}
	if strings.Contains(string(secretData), "top-secret-token") {
		t.Fatalf("expected encrypted secret payload")
	}
}

func TestUpdateDomainConfigValidationRules(t *testing.T) {
	t.Parallel()

	db := newTestDB(t)
	svc := NewService(db, secrets.NewStore(filepath.Join(t.TempDir(), "secrets")))

	_, err := svc.UpdateDomainConfig(context.Background(), UpdateDomainConfigInput{
		ControlPlaneDomain: stringPtr("panel.example.com"),
		PreviewDomain:      stringPtr("wp.example.com"),
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if err != ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}

	_, err = svc.UpdateDomainConfig(context.Background(), UpdateDomainConfigInput{
		ControlPlaneDomain: stringPtr("panel.example.com"),
		DNS01Provider:      stringPtr("cloudflare"),
	})
	if err == nil {
		t.Fatalf("expected validation error")
	}
	if err != ErrInvalidInput {
		t.Fatalf("expected ErrInvalidInput, got %v", err)
	}
}

func newTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "settings-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE settings (
			key TEXT PRIMARY KEY,
			value TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("create settings table: %v", err)
	}

	return db
}

func stringPtr(value string) *string {
	return &value
}
