package sshutil

import (
	"crypto/ed25519"
	"strings"
	"testing"

	"golang.org/x/crypto/ssh"
)

func TestGenerateKeyPairReturnsValidKeys(t *testing.T) {
	pub, priv, err := GenerateKeyPair("test-key")
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}
	if pub == "" {
		t.Fatal("public key is empty")
	}
	if priv == "" {
		t.Fatal("private key is empty")
	}
}

func TestGenerateKeyPairPublicKeyFormat(t *testing.T) {
	pub, _, err := GenerateKeyPair("my-comment")
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	if !strings.HasPrefix(pub, "ssh-ed25519 ") {
		t.Fatalf("public key prefix = %q, want ssh-ed25519 prefix", pub[:20])
	}

	if !strings.HasSuffix(pub, " my-comment") {
		t.Fatalf("public key does not end with comment: %q", pub)
	}

	// Parse the key portion (without comment) to verify it's a valid authorized key
	parts := strings.SplitN(pub, " ", 3)
	if len(parts) < 2 {
		t.Fatalf("expected at least 2 parts in public key, got %d", len(parts))
	}
	keyLine := parts[0] + " " + parts[1]
	_, _, _, _, err = ssh.ParseAuthorizedKey([]byte(keyLine))
	if err != nil {
		t.Fatalf("public key is not a valid authorized key: %v", err)
	}
}

func TestGenerateKeyPairEmptyComment(t *testing.T) {
	pub, _, err := GenerateKeyPair("")
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	// No trailing comment when comment is empty
	parts := strings.SplitN(pub, " ", 3)
	if len(parts) != 2 {
		t.Fatalf("expected exactly 2 parts (no comment) in public key, got %d: %q", len(parts), pub)
	}
}

func TestGenerateKeyPairPrivateKeyIsPEM(t *testing.T) {
	_, priv, err := GenerateKeyPair("test")
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	if !strings.Contains(priv, "-----BEGIN OPENSSH PRIVATE KEY-----") {
		t.Fatal("private key missing PEM header")
	}
	if !strings.Contains(priv, "-----END OPENSSH PRIVATE KEY-----") {
		t.Fatal("private key missing PEM footer")
	}
}

func TestGenerateKeyPairPrivateKeyIsEd25519(t *testing.T) {
	_, priv, err := GenerateKeyPair("test")
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	rawKey, err := ssh.ParseRawPrivateKey([]byte(priv))
	if err != nil {
		t.Fatalf("ParseRawPrivateKey() error = %v", err)
	}

	privKey, ok := rawKey.(*ed25519.PrivateKey)
	if !ok {
		t.Fatalf("private key type = %T, want *ed25519.PrivateKey", rawKey)
	}
	if len(*privKey) != ed25519.PrivateKeySize {
		t.Fatalf("private key size = %d, want %d", len(*privKey), ed25519.PrivateKeySize)
	}
}

func TestGenerateKeyPairUniqueness(t *testing.T) {
	pub1, priv1, err := GenerateKeyPair("key1")
	if err != nil {
		t.Fatalf("first GenerateKeyPair() error = %v", err)
	}
	pub2, priv2, err := GenerateKeyPair("key2")
	if err != nil {
		t.Fatalf("second GenerateKeyPair() error = %v", err)
	}

	// Strip comments for comparison
	pubKey1 := strings.SplitN(pub1, " ", 3)[1]
	pubKey2 := strings.SplitN(pub2, " ", 3)[1]
	if pubKey1 == pubKey2 {
		t.Fatal("two generated key pairs have identical public keys")
	}
	if priv1 == priv2 {
		t.Fatal("two generated key pairs have identical private keys")
	}
}

func TestGenerateKeyPairPublicMatchesPrivate(t *testing.T) {
	pub, priv, err := GenerateKeyPair("verify")
	if err != nil {
		t.Fatalf("GenerateKeyPair() error = %v", err)
	}

	// Parse private key
	rawKey, err := ssh.ParseRawPrivateKey([]byte(priv))
	if err != nil {
		t.Fatalf("ParseRawPrivateKey() error = %v", err)
	}
	privKey := rawKey.(*ed25519.PrivateKey)
	derivedPub := privKey.Public().(ed25519.PublicKey)

	sshDerivedPub, err := ssh.NewPublicKey(derivedPub)
	if err != nil {
		t.Fatalf("NewPublicKey() error = %v", err)
	}
	derivedAuthorized := strings.TrimSpace(string(ssh.MarshalAuthorizedKey(sshDerivedPub)))

	// The public key (without comment) should match the derived one
	pubWithoutComment := strings.SplitN(pub, " ", 3)
	pubKeyOnly := pubWithoutComment[0] + " " + pubWithoutComment[1]

	if pubKeyOnly != derivedAuthorized {
		t.Fatalf("public key does not match private key.\ngot:  %q\nwant: %q", pubKeyOnly, derivedAuthorized)
	}
}
