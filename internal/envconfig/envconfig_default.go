//go:build !dev

package envconfig

import (
	"os"
	"path/filepath"
	"strings"
)

const Mode = "standard"

func defaultDataDir() string {
	dataHome := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataHome == "" {
		home, err := os.UserHomeDir()
		if err != nil || strings.TrimSpace(home) == "" {
			return filepath.Join(string(os.PathSeparator), "etc", "pressluft")
		}
		dataHome = filepath.Join(home, ".local", "share")
	}
	return filepath.Join(dataHome, "pressluft")
}

func defaultAgeKeyPath() string {
	home, err := os.UserHomeDir()
	if err != nil || strings.TrimSpace(home) == "" {
		return filepath.Join(string(os.PathSeparator), "etc", "pressluft", "age.key")
	}
	return filepath.Join(home, ".pressluft", "age.key")
}
