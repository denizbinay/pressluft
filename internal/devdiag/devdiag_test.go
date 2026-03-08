package devdiag

import (
	"log/slog"
	"os"
	"path/filepath"
	"testing"

	"pressluft/internal/database"
	"pressluft/internal/envconfig"
	"pressluft/internal/pki"
	"pressluft/internal/security"
)

func TestInspectReportsStoredCAMismatch(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	dbPath := filepath.Join(dataDir, "pressluft.db")
	agePath := filepath.Join(dataDir, "age.key")
	caPath := filepath.Join(dataDir, "ca.key")
	sessionPath := filepath.Join(dataDir, "session.key")
	if err := os.MkdirAll(dataDir, 0o755); err != nil {
		t.Fatalf("MkdirAll() error = %v", err)
	}

	if _, err := security.EnsureAgeKey(agePath, true); err != nil {
		t.Fatalf("EnsureAgeKey() error = %v", err)
	}
	db, err := database.Open(dbPath, slog.New(slog.NewTextHandler(os.Stdout, nil)))
	if err != nil {
		t.Fatalf("Open() error = %v", err)
	}
	if _, err := pki.LoadOrCreateCA(db.DB, agePath, caPath); err != nil {
		t.Fatalf("LoadOrCreateCA() error = %v", err)
	}
	if err := db.Close(); err != nil {
		t.Fatalf("Close() error = %v", err)
	}
	if err := os.WriteFile(sessionPath, []byte("secret"), 0o600); err != nil {
		t.Fatalf("WriteFile(session) error = %v", err)
	}

	if err := os.Remove(agePath); err != nil {
		t.Fatalf("Remove(age) error = %v", err)
	}
	if _, err := security.EnsureAgeKey(agePath, true); err != nil {
		t.Fatalf("EnsureAgeKey(new) error = %v", err)
	}

	report := Inspect(envconfig.ControlPlaneRuntime{
		DBPath:            dbPath,
		DataDir:           dataDir,
		AgeKeyPath:        agePath,
		CAKeyPath:         caPath,
		SessionSecretPath: sessionPath,
	})
	if report.HealthyFor(WorkflowDev) {
		t.Fatal("HealthyFor(dev) = true, want false")
	}
	found := false
	for _, check := range report.Checks {
		if check.Name == "ca_state" && check.Status == CheckStatusError {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected ca_state error, got %#v", report.Checks)
	}
}

func TestInspectWarnsWhenStateIsMissing(t *testing.T) {
	root := t.TempDir()
	dataDir := filepath.Join(root, "data")
	report := Inspect(envconfig.ControlPlaneRuntime{
		DBPath:            filepath.Join(dataDir, "pressluft.db"),
		DataDir:           dataDir,
		AgeKeyPath:        filepath.Join(dataDir, "age.key"),
		CAKeyPath:         filepath.Join(dataDir, "ca.key"),
		SessionSecretPath: filepath.Join(dataDir, "session.key"),
	})
	if !report.HealthyFor(WorkflowDev) {
		t.Fatal("HealthyFor(dev) = false, want true when only startup-generated state is missing")
	}
}
