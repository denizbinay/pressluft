//go:build !dev

package envconfig

import (
	"path/filepath"
	"testing"
)

func TestResolveUsesXDGDataHomeByDefault(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(string(filepath.Separator), "tmp", "xdg"))
	t.Setenv("PRESSLUFT_DB", "")
	t.Setenv("PRESSLUFT_AGE_KEY_PATH", "")
	t.Setenv("PRESSLUFT_CA_KEY_PATH", "")

	paths := Resolve()

	if paths.DataDir != filepath.Join(string(filepath.Separator), "tmp", "xdg", "pressluft") {
		t.Fatalf("DataDir = %q", paths.DataDir)
	}
	if paths.DBPath != filepath.Join(string(filepath.Separator), "tmp", "xdg", "pressluft", "pressluft.db") {
		t.Fatalf("DBPath = %q", paths.DBPath)
	}
	if paths.CAKeyPath != filepath.Join(string(filepath.Separator), "tmp", "xdg", "pressluft", "ca.key") {
		t.Fatalf("CAKeyPath = %q", paths.CAKeyPath)
	}
	if paths.AgeKeyPath != filepath.Join(string(filepath.Separator), "tmp", "xdg", "pressluft", "age.key") {
		t.Fatalf("AgeKeyPath = %q", paths.AgeKeyPath)
	}
}

func TestResolveRespectsOverrides(t *testing.T) {
	t.Setenv("XDG_DATA_HOME", filepath.Join(string(filepath.Separator), "tmp", "xdg"))
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
