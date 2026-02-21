package ansible

import (
	"fmt"
	"os"
	"strings"
)

func WriteTempLocalInventory(hostname string) (string, func(), error) {
	host := strings.TrimSpace(hostname)
	if host == "" {
		host = "localhost"
	}

	f, err := os.CreateTemp("", "pressluft-inventory-*.ini")
	if err != nil {
		return "", nil, fmt.Errorf("create temp inventory: %w", err)
	}
	cleanup := func() { _ = os.Remove(f.Name()) }

	content := fmt.Sprintf("[target]\n%s ansible_connection=local\n", host)
	if _, err := f.WriteString(content); err != nil {
		f.Close()
		cleanup()
		return "", nil, fmt.Errorf("write inventory: %w", err)
	}
	if err := f.Close(); err != nil {
		cleanup()
		return "", nil, fmt.Errorf("close inventory: %w", err)
	}

	return f.Name(), cleanup, nil
}
