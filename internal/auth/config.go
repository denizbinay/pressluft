package auth

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"pressluft/internal/platform"
	"pressluft/internal/security"
)

func LoadSessionSecret(path string) ([]byte, string, error) {
	if strings.TrimSpace(path) == "" {
		path = DefaultSessionSecretPath()
	}
	if err := security.EnsureRandomSecret(path, true); err != nil {
		return nil, "", err
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, "", fmt.Errorf("read session secret: %w", err)
	}
	return data, path, nil
}

func DefaultSessionSecretPath() string {
	return filepath.Join(defaultDataDir(), "pressluft", "session.key")
}

func defaultDataDir() string {
	dataDir := strings.TrimSpace(os.Getenv("XDG_DATA_HOME"))
	if dataDir == "" {
		home, _ := os.UserHomeDir()
		dataDir = filepath.Join(home, ".local", "share")
	}
	return dataDir
}

func BootstrapCredentials(mode platform.ExecutionMode) (string, string, error) {
	email := strings.TrimSpace(os.Getenv("PRESSLUFT_BOOTSTRAP_ADMIN_EMAIL"))
	password := strings.TrimSpace(os.Getenv("PRESSLUFT_BOOTSTRAP_ADMIN_PASSWORD"))
	passwordFile := strings.TrimSpace(os.Getenv("PRESSLUFT_BOOTSTRAP_ADMIN_PASSWORD_FILE"))
	if password == "" && passwordFile != "" {
		data, err := os.ReadFile(passwordFile)
		if err != nil {
			return "", "", fmt.Errorf("read bootstrap admin password file: %w", err)
		}
		password = strings.TrimSpace(string(data))
	}
	if mode == platform.ExecutionModeDev {
		return email, password, nil
	}
	if email == "" || password == "" {
		return "", "", fmt.Errorf("PRESSLUFT_BOOTSTRAP_ADMIN_EMAIL and PRESSLUFT_BOOTSTRAP_ADMIN_PASSWORD or PRESSLUFT_BOOTSTRAP_ADMIN_PASSWORD_FILE are required in %s mode", mode)
	}
	return email, password, nil
}
