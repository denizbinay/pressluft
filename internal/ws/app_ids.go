package ws

import (
	"fmt"
	"strings"

	"pressluft/internal/idutil"
)

func FormatAppID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	normalized, err := idutil.Normalize(id)
	if err != nil {
		return id
	}
	return normalized
}

func ParseAppID(raw string) (string, error) {
	id, err := idutil.Normalize(raw)
	if err != nil {
		return "", fmt.Errorf("app id: %w", err)
	}
	return id, nil
}
