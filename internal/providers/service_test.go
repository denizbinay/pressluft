package providers

import (
	"context"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"testing"
)

func TestListIncludesDisconnectedCatalogEntry(t *testing.T) {
	service := NewService(NewInMemoryStore(nil))

	connections, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(connections) != 1 {
		t.Fatalf("len(connections) = %d, want 1", len(connections))
	}
	if connections[0].ProviderID != "hetzner" {
		t.Fatalf("provider_id = %q, want hetzner", connections[0].ProviderID)
	}
	if connections[0].Status != StatusDisconnected {
		t.Fatalf("status = %q, want disconnected", connections[0].Status)
	}
	if connections[0].SecretConfigured {
		t.Fatal("secret_configured = true, want false")
	}
}

func TestConnectPersistsWithoutExposingSecret(t *testing.T) {
	if err := os.Setenv("PRESSLUFT_DISABLE_RUNTIME_PROBES", "1"); err != nil {
		t.Fatalf("Setenv error: %v", err)
	}
	t.Cleanup(func() {
		_ = os.Unsetenv("PRESSLUFT_DISABLE_RUNTIME_PROBES")
	})

	service := NewService(NewInMemoryStore(nil))

	connection, err := service.Connect(context.Background(), ConnectInput{ProviderID: "hetzner", Secret: "hetzner_bearer_token"})
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if connection.Status != StatusConnected {
		t.Fatalf("status = %q, want connected", connection.Status)
	}

	persisted, err := service.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(persisted) != 1 {
		t.Fatalf("len(persisted) = %d, want 1", len(persisted))
	}
	if !persisted[0].SecretConfigured {
		t.Fatal("secret_configured = false, want true")
	}
}

func TestConnectDegradesWhenHetznerCredentialRejected(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actions" {
			http.Error(w, "unexpected route", http.StatusNotFound)
			return
		}
		http.Error(w, "forbidden", http.StatusForbidden)
	}))
	defer server.Close()

	service := NewService(NewInMemoryStore(nil))
	service.hetznerAPIBaseURL = server.URL
	service.hetznerHTTPClient = server.Client()

	connection, err := service.Connect(context.Background(), ConnectInput{ProviderID: "hetzner", Secret: "bearer-token"})
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if connection.Status != StatusDegraded {
		t.Fatalf("status = %q, want degraded", connection.Status)
	}
	if !strings.Contains(connection.LastStatusMessage, "Credential rejected") {
		t.Fatalf("last_status_message = %q, want credential rejected guidance", connection.LastStatusMessage)
	}
}

func TestConnectUsesLiveValidationWithoutPrefixHeuristics(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/actions" {
			http.Error(w, "unexpected route", http.StatusNotFound)
			return
		}
		if got := r.Header.Get("Authorization"); got != "Bearer bearer-token" {
			t.Fatalf("Authorization header = %q, want Bearer bearer-token", got)
		}
		_, _ = w.Write([]byte(`{"actions":[]}`))
	}))
	defer server.Close()

	service := NewService(NewInMemoryStore(nil))
	service.hetznerAPIBaseURL = server.URL
	service.hetznerHTTPClient = server.Client()

	connection, err := service.Connect(context.Background(), ConnectInput{ProviderID: "hetzner", Secret: "bearer-token"})
	if err != nil {
		t.Fatalf("Connect() error = %v", err)
	}
	if connection.Status != StatusConnected {
		t.Fatalf("status = %q, want connected", connection.Status)
	}
}
