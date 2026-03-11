package ansible

import (
	"encoding/json"
	"os"
	"os/exec"
	"strings"
	"testing"
)

func TestFlattenExtraVarsJSONPreservesSpaces(t *testing.T) {
	extra := map[string]string{
		"ssh_public_key": "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIOiL6xwz0x base-comment",
		"simple":         "value",
	}

	args := flattenExtraVars(extra)
	if len(args) != 2 {
		t.Fatalf("expected 2 args, got %d", len(args))
	}
	if args[0] != "--extra-vars" {
		t.Fatalf("expected --extra-vars flag, got %q", args[0])
	}

	var decoded map[string]string
	if err := json.Unmarshal([]byte(args[1]), &decoded); err != nil {
		t.Fatalf("failed to decode extra vars JSON: %v", err)
	}
	if decoded["ssh_public_key"] != extra["ssh_public_key"] {
		t.Fatalf("ssh_public_key mismatch: got %q", decoded["ssh_public_key"])
	}
}

func TestConfigureCommandStripsRemoteTempOverride(t *testing.T) {
	t.Setenv("ANSIBLE_REMOTE_TEMP", "/tmp/ansible-remote")
	t.Setenv("PATH", "/usr/bin")

	adapter := NewAdapter("/repo/.venv/bin/ansible-playbook", "/repo", nil)
	cmd := exec.Command("true")

	adapter.configureCommand(cmd)

	for _, entry := range cmd.Env {
		if strings.HasPrefix(entry, "ANSIBLE_REMOTE_TEMP=") {
			t.Fatalf("ANSIBLE_REMOTE_TEMP leaked into ansible command env: %q", entry)
		}
	}

	if got := envValue(cmd.Env, "ANSIBLE_STDOUT_CALLBACK"); got != "json" {
		t.Fatalf("ANSIBLE_STDOUT_CALLBACK = %q, want json", got)
	}
	if got := envValue(cmd.Env, "ANSIBLE_ROLES_PATH"); got != "ops/ansible/roles" {
		t.Fatalf("ANSIBLE_ROLES_PATH = %q, want ops/ansible/roles", got)
	}
	if got := envValue(cmd.Env, "VIRTUAL_ENV"); got != "/repo/.venv" {
		t.Fatalf("VIRTUAL_ENV = %q, want /repo/.venv", got)
	}
	if got := envValue(cmd.Env, "PATH"); !strings.HasPrefix(got, "/repo/.venv/bin"+string(os.PathListSeparator)) {
		t.Fatalf("PATH = %q, want ansible bin prepended", got)
	}
}

func envValue(env []string, key string) string {
	prefix := key + "="
	for _, entry := range env {
		if strings.HasPrefix(entry, prefix) {
			return strings.TrimPrefix(entry, prefix)
		}
	}
	return ""
}
