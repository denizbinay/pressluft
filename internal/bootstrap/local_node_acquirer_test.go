package bootstrap

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestEnvLocalNodeAcquirerAcquireSuccess(t *testing.T) {
	t.Setenv("PRESSLUFT_LOCAL_NODE_HOST", "192.0.2.20")
	t.Setenv("PRESSLUFT_LOCAL_NODE_PUBLIC_IP", "198.51.100.20")
	t.Setenv("PRESSLUFT_LOCAL_NODE_SSH_PORT", "2222")
	t.Setenv("PRESSLUFT_LOCAL_NODE_SSH_USER", "ubuntu")
	t.Setenv("PRESSLUFT_LOCAL_NODE_SSH_PRIVATE_KEY_PATH", "/tmp/key")

	acquirer := NewEnvLocalNodeAcquirer()
	target, err := acquirer.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if target.Hostname != "192.0.2.20" {
		t.Fatalf("Hostname = %s, want 192.0.2.20", target.Hostname)
	}
	if target.PublicIP != "198.51.100.20" {
		t.Fatalf("PublicIP = %s, want 198.51.100.20", target.PublicIP)
	}
	if target.SSHPort != 2222 {
		t.Fatalf("SSHPort = %d, want 2222", target.SSHPort)
	}
	if target.SSHUser != "ubuntu" {
		t.Fatalf("SSHUser = %s, want ubuntu", target.SSHUser)
	}
	if target.SSHPrivateKeyPath != "/tmp/key" {
		t.Fatalf("SSHPrivateKeyPath = %s, want /tmp/key", target.SSHPrivateKeyPath)
	}
}

func TestEnvLocalNodeAcquirerAcquireUnavailable(t *testing.T) {
	_ = os.Unsetenv("PRESSLUFT_LOCAL_NODE_HOST")

	acquirer := NewEnvLocalNodeAcquirer()
	_, err := acquirer.Acquire(context.Background())
	if !errors.Is(err, ErrLocalAcquisitionUnavailable) {
		t.Fatalf("Acquire() error = %v, want ErrLocalAcquisitionUnavailable", err)
	}
}

func TestNewLocalNodeAcquirerUsesEnvWhenHostConfigured(t *testing.T) {
	t.Setenv("PRESSLUFT_LOCAL_NODE_HOST", "192.0.2.20")

	acquirer := NewLocalNodeAcquirer()
	target, err := acquirer.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if target.Hostname != "192.0.2.20" {
		t.Fatalf("Hostname = %s, want 192.0.2.20", target.Hostname)
	}
}

func TestMultipassLocalNodeAcquirerAcquireLaunchesAndReturnsTarget(t *testing.T) {
	keyDir := t.TempDir()
	privateKeyPath := filepath.Join(keyDir, "local-node-ed25519")
	if err := os.WriteFile(privateKeyPath, []byte("private"), 0o600); err != nil {
		t.Fatalf("WriteFile(private) error = %v", err)
	}
	publicKey := "ssh-ed25519 AAAAC3NzaC1lZDI1NTE5AAAAIKk1test pressluft@test"
	if err := os.WriteFile(privateKeyPath+".pub", []byte(publicKey+"\n"), 0o600); err != nil {
		t.Fatalf("WriteFile(public) error = %v", err)
	}
	t.Setenv("PRESSLUFT_LOCAL_NODE_SSH_PRIVATE_KEY_PATH", privateKeyPath)
	t.Setenv("PRESSLUFT_LOCAL_NODE_NAME", "pressluft-local-node")

	calls := 0
	acquirer := &MultipassLocalNodeAcquirer{
		lookPath: func(file string) (string, error) {
			if file != "multipass" {
				t.Fatalf("lookPath file = %s, want multipass", file)
			}
			return "/usr/bin/multipass", nil
		},
		run: func(ctx context.Context, name string, args ...string) ([]byte, error) {
			_ = ctx
			if name == "multipass" {
				calls++
				switch {
				case len(args) >= 2 && args[0] == "info" && args[1] == "pressluft-local-node":
					if calls == 1 {
						return []byte("instance does not exist"), fmt.Errorf("not found")
					}
					return []byte(`{"info":{"pressluft-local-node":{"state":"Running","ipv4":["192.0.2.44"]}}}`), nil
				case len(args) >= 4 && args[0] == "launch" && args[1] == "24.04" && args[2] == "--name" && args[3] == "pressluft-local-node":
					return []byte(""), nil
				case len(args) >= 3 && args[0] == "exec" && args[1] == "pressluft-local-node":
					if !strings.Contains(strings.Join(args, " "), publicKey) {
						t.Fatalf("multipass exec args missing public key: %v", args)
					}
					return []byte(""), nil
				default:
					t.Fatalf("unexpected multipass args: %v", args)
					return nil, nil
				}
			}
			t.Fatalf("command = %s, want multipass", name)
			return nil, nil
		},
	}

	target, err := acquirer.Acquire(context.Background())
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}
	if target.Hostname != "192.0.2.44" {
		t.Fatalf("Hostname = %s, want 192.0.2.44", target.Hostname)
	}
	if target.SSHUser != "ubuntu" {
		t.Fatalf("SSHUser = %s, want ubuntu", target.SSHUser)
	}
	if target.SSHPrivateKeyPath != privateKeyPath {
		t.Fatalf("SSHPrivateKeyPath = %s, want %s", target.SSHPrivateKeyPath, privateKeyPath)
	}
}

func TestMultipassLocalNodeAcquirerAcquireUnavailableWhenMissingBinary(t *testing.T) {
	path := os.Getenv("PATH")
	t.Setenv("PATH", "")
	t.Cleanup(func() {
		_ = os.Setenv("PATH", path)
	})

	acquirer := NewMultipassLocalNodeAcquirer()
	_, err := acquirer.Acquire(context.Background())
	if !errors.Is(err, ErrLocalAcquisitionUnavailable) {
		t.Fatalf("Acquire() error = %v, want ErrLocalAcquisitionUnavailable", err)
	}
}
