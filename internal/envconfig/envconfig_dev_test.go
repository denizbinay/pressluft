//go:build dev

package envconfig

import (
	"path/filepath"
	"testing"
)

func TestResolveUsesRepoLocalStateInDevMode(t *testing.T) {
	t.Setenv("PRESSLUFT_DB", "")
	t.Setenv("PRESSLUFT_AGE_KEY_PATH", "")
	t.Setenv("PRESSLUFT_CA_KEY_PATH", "")

	paths := Resolve()
	expectedDir := filepath.Join(repoRoot(), ".pressluft")

	if paths.DataDir != expectedDir {
		t.Fatalf("DataDir = %q, want %q", paths.DataDir, expectedDir)
	}
	if paths.DBPath != filepath.Join(expectedDir, "pressluft.db") {
		t.Fatalf("DBPath = %q", paths.DBPath)
	}
	if paths.AgeKeyPath != filepath.Join(expectedDir, "age.key") {
		t.Fatalf("AgeKeyPath = %q", paths.AgeKeyPath)
	}
	if paths.CAKeyPath != filepath.Join(expectedDir, "ca.key") {
		t.Fatalf("CAKeyPath = %q", paths.CAKeyPath)
	}
}

func TestResolveRespectsOverridesInDevMode(t *testing.T) {
	t.Setenv("PRESSLUFT_DB", "custom.db")
	t.Setenv("PRESSLUFT_AGE_KEY_PATH", "keys/age.key")
	t.Setenv("PRESSLUFT_CA_KEY_PATH", "keys/ca.key")

	paths := Resolve()

	if paths.DBPath != "custom.db" {
		t.Fatalf("DBPath = %q", paths.DBPath)
	}
	if paths.AgeKeyPath != filepath.Clean("keys/age.key") {
		t.Fatalf("AgeKeyPath = %q", paths.AgeKeyPath)
	}
	if paths.CAKeyPath != filepath.Clean("keys/ca.key") {
		t.Fatalf("CAKeyPath = %q", paths.CAKeyPath)
	}
}

func TestResolveControlPlaneRuntimeUsesRepoLocalStateInDevMode(t *testing.T) {
	t.Setenv("PRESSLUFT_DB", "")
	t.Setenv("PRESSLUFT_AGE_KEY_PATH", "")
	t.Setenv("PRESSLUFT_CA_KEY_PATH", "")
	t.Setenv("PRESSLUFT_SESSION_KEY_PATH", "")
	t.Setenv("PRESSLUFT_ANSIBLE_DIR", "")
	t.Setenv("PRESSLUFT_ANSIBLE_BIN", "")

	runtime, err := ResolveControlPlaneRuntime(true, "/repo/root")
	if err != nil {
		t.Fatalf("ResolveControlPlaneRuntime() error = %v", err)
	}

	expectedDir := filepath.Join(repoRoot(), ".pressluft")
	if runtime.DataDir != expectedDir {
		t.Fatalf("DataDir = %q, want %q", runtime.DataDir, expectedDir)
	}
	if runtime.DBPath != filepath.Join(expectedDir, "pressluft.db") {
		t.Fatalf("DBPath = %q", runtime.DBPath)
	}
	if runtime.AgeKeyPath != filepath.Join(expectedDir, "age.key") {
		t.Fatalf("AgeKeyPath = %q", runtime.AgeKeyPath)
	}
	if runtime.CAKeyPath != filepath.Join(expectedDir, "ca.key") {
		t.Fatalf("CAKeyPath = %q", runtime.CAKeyPath)
	}
	if runtime.SessionSecretPath != filepath.Join(expectedDir, "session.key") {
		t.Fatalf("SessionSecretPath = %q", runtime.SessionSecretPath)
	}
}
