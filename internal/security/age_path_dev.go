//go:build dev

package security

import (
	"os"
	"path/filepath"
	"strings"
)

func defaultAgeKeyPath() string {
	return filepath.Join(repoRoot(), ".pressluft", "age.key")
}

func repoRoot() string {
	wd, err := os.Getwd()
	if err != nil || strings.TrimSpace(wd) == "" {
		return "."
	}

	current := wd
	for {
		if _, err := os.Stat(filepath.Join(current, "go.mod")); err == nil {
			return current
		}

		parent := filepath.Dir(current)
		if parent == current {
			return wd
		}
		current = parent
	}
}
