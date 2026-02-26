package ansible

import (
	"encoding/json"
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
