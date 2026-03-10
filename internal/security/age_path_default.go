//go:build !dev

package security

import (
	"os"
	"path/filepath"
	"strings"
)

func defaultAgeKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(string(os.PathSeparator), "etc", "pressluft", "age.key")
	}
	return filepath.Join(home, ".pressluft", "age.key")
}
