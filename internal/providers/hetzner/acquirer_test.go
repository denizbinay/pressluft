package hetzner

import (
	"context"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestAcquireSuccess(t *testing.T) {
	var actionCalls int
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1")
		switch {
		case r.Method == http.MethodGet && (path == "/ssh_keys" || path == "/ssh_keys/pressluft-managed"):
			_, _ = w.Write([]byte(`{"ssh_keys":[]}`))
		case r.Method == http.MethodPost && path == "/ssh_keys":
			_, _ = w.Write([]byte(`{"ssh_key":{"id":11}}`))
		case r.Method == http.MethodPost && path == "/servers":
			_, _ = w.Write([]byte(`{"server":{"id":101},"action":{"id":909}}`))
		case r.Method == http.MethodGet && path == "/actions/909":
			actionCalls++
			if actionCalls == 1 {
				_, _ = w.Write([]byte(`{"action":{"id":909,"status":"running"}}`))
				return
			}
			_, _ = w.Write([]byte(`{"action":{"id":909,"status":"success"}}`))
		case r.Method == http.MethodGet && path == "/servers/101":
			_, _ = w.Write([]byte(`{"server":{"name":"edge-1","public_net":{"ipv4":{"ip":"203.0.113.12"}}}}`))
		default:
			http.Error(w, "unexpected route", http.StatusNotFound)
		}
	}))
	defer server.Close()

	privateKeyPath, publicKeyPath := writeManagedKeys(t)
	acquirer := NewAcquirer()
	acquirer.baseURL = server.URL
	acquirer.httpClient = server.Client()
	acquirer.privateKeyPath = privateKeyPath
	acquirer.publicKeyPath = publicKeyPath
	acquirer.pollInterval = time.Millisecond
	acquirer.pollTimeout = 50 * time.Millisecond

	target, err := acquirer.Acquire(context.Background(), AcquireInput{Token: "bearer-token", Name: "edge-1"})
	if err != nil {
		t.Fatalf("Acquire() error = %v", err)
	}

	if target.PublicIP != "203.0.113.12" {
		t.Fatalf("PublicIP = %q, want 203.0.113.12", target.PublicIP)
	}
	if target.ServerID != 101 {
		t.Fatalf("ServerID = %d, want 101", target.ServerID)
	}
	if target.ActionID != 909 {
		t.Fatalf("ActionID = %d, want 909", target.ActionID)
	}
}

func TestAcquireMissingManagedKey(t *testing.T) {
	acquirer := NewAcquirer()
	acquirer.publicKeyPath = filepath.Join(t.TempDir(), "missing.pub")

	_, err := acquirer.Acquire(context.Background(), AcquireInput{Token: "bearer-token", Name: "edge-1"})
	if err == nil {
		t.Fatal("Acquire() error = nil, want non-nil")
	}
	if !strings.Contains(err.Error(), ErrManagedKeyMissing.Error()) {
		t.Fatalf("error = %v, want ErrManagedKeyMissing", err)
	}
}

func TestAcquireActionTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := strings.TrimPrefix(r.URL.Path, "/v1")
		switch {
		case r.Method == http.MethodGet && (path == "/ssh_keys" || path == "/ssh_keys/pressluft-managed"):
			_, _ = w.Write([]byte(`{"ssh_keys":[{"id":11}]}`))
		case r.Method == http.MethodPost && path == "/servers":
			_, _ = w.Write([]byte(`{"server":{"id":101},"action":{"id":909}}`))
		case r.Method == http.MethodGet && path == "/actions/909":
			_, _ = w.Write([]byte(`{"action":{"id":909,"status":"running"}}`))
		default:
			http.Error(w, "unexpected route", http.StatusNotFound)
		}
	}))
	defer server.Close()

	privateKeyPath, publicKeyPath := writeManagedKeys(t)
	acquirer := NewAcquirer()
	acquirer.baseURL = server.URL
	acquirer.httpClient = server.Client()
	acquirer.privateKeyPath = privateKeyPath
	acquirer.publicKeyPath = publicKeyPath
	acquirer.pollInterval = time.Millisecond
	acquirer.pollTimeout = 4 * time.Millisecond

	_, err := acquirer.Acquire(context.Background(), AcquireInput{Token: "bearer-token", Name: "edge-1"})
	if err == nil {
		t.Fatal("Acquire() error = nil, want non-nil")
	}
	if err != ErrActionTimeout {
		t.Fatalf("error = %v, want %v", err, ErrActionTimeout)
	}
}

func writeManagedKeys(t *testing.T) (string, string) {
	t.Helper()
	dir := t.TempDir()
	privateKeyPath := filepath.Join(dir, "managed")
	publicKeyPath := privateKeyPath + ".pub"
	if err := os.WriteFile(privateKeyPath, []byte("PRIVATE"), 0o600); err != nil {
		t.Fatalf("write private key: %v", err)
	}
	if err := os.WriteFile(publicKeyPath, []byte(fmt.Sprintf("ssh-ed25519 %s", strings.Repeat("a", 24))), 0o600); err != nil {
		t.Fatalf("write public key: %v", err)
	}
	return privateKeyPath, publicKeyPath
}
