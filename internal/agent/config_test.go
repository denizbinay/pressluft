package agent

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestLoadConfigValidatesRequiredFields(t *testing.T) {
	path := writeAgentConfig(t, `
control_plane: https://control.example.test
cert_file: /etc/pressluft/client.crt
key_file: /etc/pressluft/client.key
ca_cert_file: /etc/pressluft/ca.crt
data_dir: /var/lib/pressluft
`)

	if _, err := LoadConfig(path); err == nil || !strings.Contains(err.Error(), "server_id") {
		t.Fatalf("LoadConfig() error = %v, want server_id validation", err)
	}
}

func TestLoadConfigRejectsMutuallyExclusiveTokenSources(t *testing.T) {
	path := writeAgentConfig(t, `
server_id: 42
control_plane: https://control.example.test
cert_file: /etc/pressluft/client.crt
key_file: /etc/pressluft/client.key
ca_cert_file: /etc/pressluft/ca.crt
data_dir: /var/lib/pressluft
registration_token: inline-token
registration_token_file: /etc/pressluft/token
`)

	if _, err := LoadConfig(path); err == nil || !strings.Contains(err.Error(), "mutually exclusive") {
		t.Fatalf("LoadConfig() error = %v, want mutual exclusion validation", err)
	}
}

func TestLoadConfigAcceptsValidConfig(t *testing.T) {
	path := writeAgentConfig(t, `
server_id: 42
control_plane: https://control.example.test
cert_file: /etc/pressluft/client.crt
key_file: /etc/pressluft/client.key
ca_cert_file: /etc/pressluft/ca.crt
data_dir: /var/lib/pressluft
registration_token_file: /etc/pressluft/token
`)

	cfg, err := LoadConfig(path)
	if err != nil {
		t.Fatalf("LoadConfig() error = %v", err)
	}
	if cfg.ServerID != 42 {
		t.Fatalf("server_id = %d, want 42", cfg.ServerID)
	}
}

func writeAgentConfig(t *testing.T, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), "agent.yaml")
	if err := os.WriteFile(path, []byte(strings.TrimSpace(content)+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile() error = %v", err)
	}
	return path
}
