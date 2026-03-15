package cliutil

import (
	"fmt"
	"os"
	"path/filepath"
)

func FindRepoRoot() (string, error) {
	dir, err := os.Getwd()
	if err != nil {
		return "", err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir, nil
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return "", fmt.Errorf("could not find go.mod in any parent directory")
		}
		dir = parent
	}
}

func GoCmd() string {
	if v := os.Getenv("GO"); v != "" {
		return v
	}
	return "go"
}

func NpmCmd() string {
	if v := os.Getenv("NPM"); v != "" {
		return v
	}
	return "pnpm"
}
