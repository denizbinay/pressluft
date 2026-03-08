package security

import (
	"crypto/rand"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func EnsureRandomSecret(path string, allowGenerate bool) error {
	if strings.TrimSpace(path) == "" {
		return fmt.Errorf("secret path is empty")
	}
	info, err := os.Stat(path)
	if err == nil {
		if info.IsDir() {
			return fmt.Errorf("secret path is a directory: %s", path)
		}
		return nil
	}
	if !os.IsNotExist(err) {
		return fmt.Errorf("stat secret: %w", err)
	}
	if !allowGenerate {
		return fmt.Errorf("secret missing: %s", path)
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return fmt.Errorf("create secret dir: %w", err)
	}
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return fmt.Errorf("generate secret: %w", err)
	}
	if err := os.WriteFile(path, buf, 0o600); err != nil {
		return fmt.Errorf("write secret: %w", err)
	}
	return nil
}
