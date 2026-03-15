//go:build dev

package envconfig

import (
	"os"
	"path/filepath"
	"strings"
)

const Mode = "dev"

func defaultDataDir() string {
	root := repoRoot()
	return filepath.Join(root, ".pressluft")
}

func defaultAgeKeyPath() string {
	return filepath.Join(defaultDataDir(), "age.key")
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
