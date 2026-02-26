package security

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"filippo.io/age"
)

func TestEncryptDecryptRoundTrip(t *testing.T) {
	identity, err := age.GenerateX25519Identity()
	if err != nil {
		t.Fatalf("generate age identity: %v", err)
	}

	keyPath := filepath.Join(t.TempDir(), "age.key")
	if err := os.WriteFile(keyPath, []byte(identity.String()+"\n"), 0o600); err != nil {
		t.Fatalf("write age key: %v", err)
	}
	t.Setenv("PRESSLUFT_AGE_KEY_PATH", keyPath)

	plaintext := []byte("round-trip-secret")
	ciphertext, keyID, err := Encrypt(plaintext)
	if err != nil {
		t.Fatalf("encrypt: %v", err)
	}
	if !strings.HasPrefix(keyID, "sha256:") {
		t.Fatalf("keyID = %q, want sha256 prefix", keyID)
	}

	decrypted, err := Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("decrypt: %v", err)
	}
	if !bytes.Equal(decrypted, plaintext) {
		t.Fatalf("decrypted = %q, want %q", decrypted, plaintext)
	}
}

func TestEnsureAgeKeyGeneratesWhenAllowed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "age.key")

	generated, err := EnsureAgeKey(path, true)
	if err != nil {
		t.Fatalf("ensure age key: %v", err)
	}
	if !generated {
		t.Fatalf("expected generated=true")
	}

	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat age key: %v", err)
	}
	if info.IsDir() {
		t.Fatalf("age key path is a directory")
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("age key permissions = %v, want 600", info.Mode().Perm())
	}

	generated, err = EnsureAgeKey(path, true)
	if err != nil {
		t.Fatalf("ensure age key again: %v", err)
	}
	if generated {
		t.Fatalf("expected generated=false when key exists")
	}
}

func TestEnsureAgeKeyFailsWhenMissingAndDisallowed(t *testing.T) {
	path := filepath.Join(t.TempDir(), "age.key")
	_, err := EnsureAgeKey(path, false)
	if err == nil {
		t.Fatalf("expected error when key is missing and generation disallowed")
	}
}
