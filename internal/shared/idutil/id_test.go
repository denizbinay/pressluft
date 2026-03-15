package idutil

import (
	"testing"
)

func TestNewReturnsValidUUIDv7(t *testing.T) {
	id, err := New()
	if err != nil {
		t.Fatalf("New() error = %v", err)
	}
	if id == "" {
		t.Fatal("New() returned empty string")
	}
	if !IsValid(id) {
		t.Fatalf("New() returned invalid id: %q", id)
	}
}

func TestNewReturnsUniqueIDs(t *testing.T) {
	seen := make(map[string]bool)
	for i := 0; i < 100; i++ {
		id, err := New()
		if err != nil {
			t.Fatalf("New() error on iteration %d: %v", i, err)
		}
		if seen[id] {
			t.Fatalf("New() returned duplicate id %q on iteration %d", id, i)
		}
		seen[id] = true
	}
}

func TestMustNewDoesNotPanic(t *testing.T) {
	id := MustNew()
	if id == "" {
		t.Fatal("MustNew() returned empty string")
	}
	if !IsValid(id) {
		t.Fatalf("MustNew() returned invalid id: %q", id)
	}
}

func TestNormalizeValidUUIDv7(t *testing.T) {
	id, _ := New()
	normalized, err := Normalize(id)
	if err != nil {
		t.Fatalf("Normalize(%q) error = %v", id, err)
	}
	if normalized != id {
		t.Fatalf("Normalize(%q) = %q, want %q", id, normalized, id)
	}
}

func TestNormalizeTrimsWhitespace(t *testing.T) {
	id, _ := New()
	normalized, err := Normalize("  " + id + "  ")
	if err != nil {
		t.Fatalf("Normalize() error = %v", err)
	}
	if normalized != id {
		t.Fatalf("Normalize() = %q, want %q", normalized, id)
	}
}

func TestNormalizeEmptyString(t *testing.T) {
	_, err := Normalize("")
	if err == nil {
		t.Fatal("Normalize(\"\") expected error, got nil")
	}
}

func TestNormalizeWhitespaceOnly(t *testing.T) {
	_, err := Normalize("   ")
	if err == nil {
		t.Fatal("Normalize(\"   \") expected error, got nil")
	}
}

func TestNormalizeInvalidUUID(t *testing.T) {
	_, err := Normalize("not-a-uuid")
	if err == nil {
		t.Fatal("Normalize(\"not-a-uuid\") expected error, got nil")
	}
}

func TestNormalizeRejectsNonV7UUID(t *testing.T) {
	// UUIDv4
	_, err := Normalize("550e8400-e29b-41d4-a716-446655440000")
	if err == nil {
		t.Fatal("Normalize() expected error for UUIDv4, got nil")
	}
}

func TestIsValidAcceptsValidID(t *testing.T) {
	id, _ := New()
	if !IsValid(id) {
		t.Fatalf("IsValid(%q) = false, want true", id)
	}
}

func TestIsValidRejectsInvalidInput(t *testing.T) {
	cases := []string{
		"",
		"   ",
		"not-a-uuid",
		"550e8400-e29b-41d4-a716-446655440000", // v4
	}
	for _, tc := range cases {
		if IsValid(tc) {
			t.Errorf("IsValid(%q) = true, want false", tc)
		}
	}
}
