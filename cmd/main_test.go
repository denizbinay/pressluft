package main

import "testing"

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
