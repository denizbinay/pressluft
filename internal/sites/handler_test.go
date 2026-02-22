package sites

import (
	"context"
	"errors"
	"log"
	"os"
	"testing"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/nodes"
	"pressluft/internal/store"
)

type mockSiteCreateExecutor struct {
	runErr     error
	calledWith SiteCreateVars
}

func (m *mockSiteCreateExecutor) RunSiteCreate(_ context.Context, _ nodes.Node, vars SiteCreateVars) error {
	m.calledWith = vars
	return m.runErr
}

func TestSiteCreateHandler_Handle(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	logger := log.New(os.Stderr, "test: ", 0)

	t.Run("success", func(t *testing.T) {
		siteStore := store.NewInMemorySiteStoreWithSeed(nil)
		nodeStore := nodes.NewInMemoryStore(nil)

		providerNode, _ := nodeStore.Create(context.Background(), nodes.CreateInput{
			ProviderID: "hetzner",
			Name:       "provider-node",
			Hostname:   "192.0.2.40",
			PublicIP:   "192.0.2.40",
			SSHPort:    22,
			SSHUser:    "ubuntu",
			IsLocal:    false,
			Now:        now,
		})
		site, env, _ := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
			Name:       "Test Site",
			Slug:       "test-site",
			NodeID:     providerNode.ID,
			NodePublic: providerNode.PublicIP,
			Now:        now,
		})

		executor := &mockSiteCreateExecutor{}
		handler := NewSiteCreateHandler(siteStore, nodeStore, executor, logger)

		job := jobs.Job{
			ID:            "job-1",
			JobType:       "site_create",
			SiteID:        &site.ID,
			EnvironmentID: &env.ID,
			NodeID:        &providerNode.ID,
		}

		err := handler.Handle(context.Background(), job)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if executor.calledWith.SiteID != site.ID {
			t.Errorf("expected site_id=%s, got %s", site.ID, executor.calledWith.SiteID)
		}
		if executor.calledWith.EnvironmentID != env.ID {
			t.Errorf("expected environment_id=%s, got %s", env.ID, executor.calledWith.EnvironmentID)
		}
		if executor.calledWith.PreviewURL != env.PreviewURL {
			t.Errorf("expected preview_url=%s, got %s", env.PreviewURL, executor.calledWith.PreviewURL)
		}
	})

	t.Run("missing site_id", func(t *testing.T) {
		siteStore := store.NewInMemorySiteStoreWithSeed(nil)
		nodeStore := nodes.NewInMemoryStore(nil)
		executor := &mockSiteCreateExecutor{}
		handler := NewSiteCreateHandler(siteStore, nodeStore, executor, logger)

		job := jobs.Job{
			ID:      "job-1",
			JobType: "site_create",
		}

		err := handler.Handle(context.Background(), job)
		if err == nil {
			t.Fatal("expected error for missing site_id")
		}

		var execErr jobs.ExecutionError
		if !errors.As(err, &execErr) {
			t.Fatalf("expected ExecutionError, got %T", err)
		}
		if execErr.Code != "ANSIBLE_UNKNOWN_EXIT" {
			t.Errorf("expected code=ANSIBLE_UNKNOWN_EXIT, got %s", execErr.Code)
		}
	})

	t.Run("site not found", func(t *testing.T) {
		siteStore := store.NewInMemorySiteStoreWithSeed(nil)
		nodeStore := nodes.NewInMemoryStore(nil)
		providerNode, _ := nodeStore.Create(context.Background(), nodes.CreateInput{
			ProviderID: "hetzner",
			Name:       "provider-node",
			Hostname:   "192.0.2.41",
			PublicIP:   "192.0.2.41",
			SSHPort:    22,
			SSHUser:    "ubuntu",
			IsLocal:    false,
			Now:        now,
		})

		executor := &mockSiteCreateExecutor{}
		handler := NewSiteCreateHandler(siteStore, nodeStore, executor, logger)

		missingID := "missing-site-id"
		envID := "env-id"
		job := jobs.Job{
			ID:            "job-1",
			JobType:       "site_create",
			SiteID:        &missingID,
			EnvironmentID: &envID,
			NodeID:        &providerNode.ID,
		}

		err := handler.Handle(context.Background(), job)
		if err == nil {
			t.Fatal("expected error for missing site")
		}

		var execErr jobs.ExecutionError
		if !errors.As(err, &execErr) {
			t.Fatalf("expected ExecutionError, got %T", err)
		}
	})

	t.Run("executor failure is propagated", func(t *testing.T) {
		siteStore := store.NewInMemorySiteStoreWithSeed(nil)
		nodeStore := nodes.NewInMemoryStore(nil)

		providerNode, _ := nodeStore.Create(context.Background(), nodes.CreateInput{
			ProviderID: "hetzner",
			Name:       "provider-node",
			Hostname:   "192.0.2.42",
			PublicIP:   "192.0.2.42",
			SSHPort:    22,
			SSHUser:    "ubuntu",
			IsLocal:    false,
			Now:        now,
		})
		site, env, _ := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
			Name:       "Test Site",
			Slug:       "test-site-2",
			NodeID:     providerNode.ID,
			NodePublic: providerNode.PublicIP,
			Now:        now,
		})

		executor := &mockSiteCreateExecutor{
			runErr: jobs.ExecutionError{Code: "ANSIBLE_HOST_FAILED", Message: "connection refused", Retryable: true},
		}
		handler := NewSiteCreateHandler(siteStore, nodeStore, executor, logger)

		job := jobs.Job{
			ID:            "job-2",
			JobType:       "site_create",
			SiteID:        &site.ID,
			EnvironmentID: &env.ID,
			NodeID:        &providerNode.ID,
		}

		err := handler.Handle(context.Background(), job)
		if err == nil {
			t.Fatal("expected error from executor failure")
		}

		var execErr jobs.ExecutionError
		if !errors.As(err, &execErr) {
			t.Fatalf("expected ExecutionError, got %T", err)
		}
		if execErr.Code != "ANSIBLE_HOST_FAILED" {
			t.Errorf("expected code=ANSIBLE_HOST_FAILED, got %s", execErr.Code)
		}
		if !execErr.Retryable {
			t.Error("expected retryable=true")
		}
	})
}

func TestBuildInventory(t *testing.T) {
	t.Run("remote node", func(t *testing.T) {
		node := nodes.Node{
			ID:                "node-1",
			Hostname:          "192.168.1.10",
			SSHPort:           22,
			SSHUser:           "deploy",
			SSHPrivateKeyPath: "/path/to/key",
			IsLocal:           false,
		}

		inventory := buildInventory(node)
		if !contains(inventory, "[target]") {
			t.Error("expected [target] group")
		}
		if !contains(inventory, "192.168.1.10") {
			t.Error("expected hostname")
		}
		if !contains(inventory, "ansible_port=22") {
			t.Error("expected ansible_port")
		}
		if !contains(inventory, "ansible_user=deploy") {
			t.Error("expected ansible_user")
		}
		if !contains(inventory, "ansible_ssh_private_key_file=/path/to/key") {
			t.Error("expected ansible_ssh_private_key_file")
		}
		if contains(inventory, "ansible_connection=local") {
			t.Error("expected no ansible_connection=local for remote node")
		}
	})

	t.Run("local node", func(t *testing.T) {
		node := nodes.Node{
			ID:       nodes.SelfNodeID,
			Hostname: "127.0.0.1",
			SSHPort:  22,
			SSHUser:  "pressluft",
			IsLocal:  true,
		}

		inventory := buildInventory(node)
		if contains(inventory, "ansible_connection=local") {
			t.Error("expected local nodes to use SSH inventory")
		}
	})
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
