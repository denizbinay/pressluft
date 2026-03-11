package idutil

import (
	"fmt"
	"strings"

	"github.com/google/uuid"
)

// New generates a UUIDv7 application identifier.
func New() (string, error) {
	id, err := uuid.NewV7()
	if err != nil {
		return "", fmt.Errorf("generate uuidv7: %w", err)
	}
	return id.String(), nil
}

// MustNew generates a UUIDv7 application identifier and panics on failure.
func MustNew() string {
	id, err := New()
	if err != nil {
		panic(err)
	}
	return id
}

// Normalize validates a UUIDv7 identifier and returns its canonical form.
func Normalize(raw string) (string, error) {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return "", fmt.Errorf("id is required")
	}
	parsed, err := uuid.Parse(raw)
	if err != nil {
		return "", fmt.Errorf("invalid id: %w", err)
	}
	if parsed.Version() != 7 {
		return "", fmt.Errorf("invalid id version: want uuidv7")
	}
	return parsed.String(), nil
}

// IsValid reports whether raw is a valid UUIDv7 identifier.
func IsValid(raw string) bool {
	_, err := Normalize(raw)
	return err == nil
}
