package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveAddr(t *testing.T) {
	t.Run("defaults when PORT is empty", func(t *testing.T) {
		t.Setenv("PORT", "")

		if got := resolveAddr(); got != ":8080" {
			t.Fatalf("resolveAddr() = %q, want %q", got, ":8080")
		}
	})

	t.Run("uses PORT when already prefixed", func(t *testing.T) {
		t.Setenv("PORT", ":9090")

		if got := resolveAddr(); got != ":9090" {
			t.Fatalf("resolveAddr() = %q, want %q", got, ":9090")
		}
	})

	t.Run("normalizes PORT without prefix", func(t *testing.T) {
		t.Setenv("PORT", "7070")

		if got := resolveAddr(); got != ":7070" {
			t.Fatalf("resolveAddr() = %q, want %q", got, ":7070")
		}
	})
}

func TestResolveProductionTLSConfig(t *testing.T) {
	t.Run("requires tls files in production bootstrap", func(t *testing.T) {
		if err := resolveProductionTLSConfig("https://control.example.test", "", ""); err == nil {
			t.Fatal("expected missing TLS files to fail")
		}
	})

	t.Run("requires https control plane url", func(t *testing.T) {
		if err := resolveProductionTLSConfig("http://control.example.test", "/tmp/control.crt", "/tmp/control.key"); err == nil {
			t.Fatal("expected non-https control plane URL to fail")
		}
	})

	t.Run("accepts configured tls files", func(t *testing.T) {
		if err := resolveProductionTLSConfig("https://control.example.test", "/tmp/control.crt", "/tmp/control.key"); err != nil {
			t.Fatalf("resolveProductionTLSConfig() error = %v", err)
		}
	})
}

func TestResolveAnsibleBinary(t *testing.T) {
	t.Run("resolves repo-relative binary", func(t *testing.T) {
		root := t.TempDir()
		binDir := filepath.Join(root, ".venv", "bin")
		if err := os.MkdirAll(binDir, 0o755); err != nil {
			t.Fatalf("MkdirAll() error = %v", err)
		}
		binary := filepath.Join(binDir, "ansible-playbook")
		if err := os.WriteFile(binary, []byte("#!/bin/sh\nexit 0\n"), 0o755); err != nil {
			t.Fatalf("WriteFile() error = %v", err)
		}
		resolved, err := resolveAnsibleBinary(filepath.Join(".venv", "bin", "ansible-playbook"), root)
		if err != nil {
			t.Fatalf("resolveAnsibleBinary() error = %v", err)
		}
		if resolved != binary {
			t.Fatalf("resolveAnsibleBinary() = %q, want %q", resolved, binary)
		}
	})

	t.Run("rejects missing binary", func(t *testing.T) {
		if _, err := resolveAnsibleBinary(filepath.Join(".venv", "bin", "ansible-playbook"), t.TempDir()); err == nil {
			t.Fatal("expected missing ansible-playbook to fail")
		}
	})
}
