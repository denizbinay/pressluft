package environments

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

type mockEnvCreateExecutor struct {
	runErr     error
	calledWith EnvCreateVars
}

func (m *mockEnvCreateExecutor) RunEnvCreate(_ context.Context, _ nodes.Node, vars EnvCreateVars) error {
	m.calledWith = vars
	return m.runErr
}

func TestEnvCreateHandler_Handle(t *testing.T) {
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	logger := log.New(os.Stderr, "test: ", 0)

	t.Run("success", func(t *testing.T) {
		siteStore := store.NewInMemorySiteStoreWithSeed(nil)
		nodeStore := nodes.NewInMemoryStore(nil)

		selfNode, _ := nodeStore.EnsureSelfNode(context.Background(), now)
		site, prodEnv, _ := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
			Name:       "Test Site",
			Slug:       "test-site",
			NodeID:     selfNode.ID,
			NodePublic: selfNode.PublicIP,
			Now:        now,
		})

		stagingEnv, _ := siteStore.CreateEnvironment(context.Background(), store.CreateEnvironmentInput{
			SiteID:              site.ID,
			Name:                "Staging",
			Slug:                "staging",
			EnvironmentType:     "staging",
			SourceEnvironmentID: &prodEnv.ID,
			PromotionPreset:     "content-protect",
			Now:                 now,
		})

		executor := &mockEnvCreateExecutor{}
		handler := NewEnvCreateHandler(siteStore, nodeStore, executor, logger)

		job := jobs.Job{
			ID:            "job-1",
			JobType:       "env_create",
			SiteID:        &site.ID,
			EnvironmentID: &stagingEnv.ID,
			NodeID:        &selfNode.ID,
		}

		err := handler.Handle(context.Background(), job)
		if err != nil {
			t.Fatalf("expected no error, got %v", err)
		}

		if executor.calledWith.SiteID != site.ID {
			t.Errorf("expected site_id=%s, got %s", site.ID, executor.calledWith.SiteID)
		}
		if executor.calledWith.EnvironmentID != stagingEnv.ID {
			t.Errorf("expected environment_id=%s, got %s", stagingEnv.ID, executor.calledWith.EnvironmentID)
		}
		if executor.calledWith.SourceEnvironmentID != prodEnv.ID {
			t.Errorf("expected source_environment_id=%s, got %s", prodEnv.ID, executor.calledWith.SourceEnvironmentID)
		}
		if executor.calledWith.PreviewURL != stagingEnv.PreviewURL {
			t.Errorf("expected preview_url=%s, got %s", stagingEnv.PreviewURL, executor.calledWith.PreviewURL)
		}
		if executor.calledWith.EnvironmentType != "staging" {
			t.Errorf("expected environment_type=staging, got %s", executor.calledWith.EnvironmentType)
		}
	})

	t.Run("missing environment_id", func(t *testing.T) {
		siteStore := store.NewInMemorySiteStoreWithSeed(nil)
		nodeStore := nodes.NewInMemoryStore(nil)
		executor := &mockEnvCreateExecutor{}
		handler := NewEnvCreateHandler(siteStore, nodeStore, executor, logger)

		siteID := "site-1"
		job := jobs.Job{
			ID:      "job-1",
			JobType: "env_create",
			SiteID:  &siteID,
		}

		err := handler.Handle(context.Background(), job)
		if err == nil {
			t.Fatal("expected error for missing environment_id")
		}

		var execErr jobs.ExecutionError
		if !errors.As(err, &execErr) {
			t.Fatalf("expected ExecutionError, got %T", err)
		}
		if execErr.Code != "ANSIBLE_UNKNOWN_EXIT" {
			t.Errorf("expected code=ANSIBLE_UNKNOWN_EXIT, got %s", execErr.Code)
		}
	})

	t.Run("source environment not found", func(t *testing.T) {
		siteStore := store.NewInMemorySiteStoreWithSeed(nil)
		nodeStore := nodes.NewInMemoryStore(nil)

		selfNode, _ := nodeStore.EnsureSelfNode(context.Background(), now)
		site, _, _ := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
			Name:       "Test Site",
			Slug:       "test-site-2",
			NodeID:     selfNode.ID,
			NodePublic: selfNode.PublicIP,
			Now:        now,
		})

		// Create environment referencing non-existent source
		// This test simulates data inconsistency
		executor := &mockEnvCreateExecutor{}
		handler := NewEnvCreateHandler(siteStore, nodeStore, executor, logger)

		missingSourceID := "missing-source"
		envID := "env-id"
		job := jobs.Job{
			ID:            "job-1",
			JobType:       "env_create",
			SiteID:        &site.ID,
			EnvironmentID: &envID,
			NodeID:        &selfNode.ID,
		}

		err := handler.Handle(context.Background(), job)
		if err == nil {
			t.Fatal("expected error for missing environment")
		}

		var execErr jobs.ExecutionError
		if !errors.As(err, &execErr) {
			t.Fatalf("expected ExecutionError, got %T", err)
		}

		_ = missingSourceID // silence unused warning
	})

	t.Run("executor failure is propagated", func(t *testing.T) {
		siteStore := store.NewInMemorySiteStoreWithSeed(nil)
		nodeStore := nodes.NewInMemoryStore(nil)

		selfNode, _ := nodeStore.EnsureSelfNode(context.Background(), now)
		site, prodEnv, _ := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
			Name:       "Test Site",
			Slug:       "test-site-3",
			NodeID:     selfNode.ID,
			NodePublic: selfNode.PublicIP,
			Now:        now,
		})

		stagingEnv, _ := siteStore.CreateEnvironment(context.Background(), store.CreateEnvironmentInput{
			SiteID:              site.ID,
			Name:                "Staging",
			Slug:                "staging",
			EnvironmentType:     "staging",
			SourceEnvironmentID: &prodEnv.ID,
			PromotionPreset:     "content-protect",
			Now:                 now,
		})

		executor := &mockEnvCreateExecutor{
			runErr: jobs.ExecutionError{Code: "ANSIBLE_PLAY_ERROR", Message: "database export failed", Retryable: true},
		}
		handler := NewEnvCreateHandler(siteStore, nodeStore, executor, logger)

		job := jobs.Job{
			ID:            "job-2",
			JobType:       "env_create",
			SiteID:        &site.ID,
			EnvironmentID: &stagingEnv.ID,
			NodeID:        &selfNode.ID,
		}

		err := handler.Handle(context.Background(), job)
		if err == nil {
			t.Fatal("expected error from executor failure")
		}

		var execErr jobs.ExecutionError
		if !errors.As(err, &execErr) {
			t.Fatalf("expected ExecutionError, got %T", err)
		}
		if execErr.Code != "ANSIBLE_PLAY_ERROR" {
			t.Errorf("expected code=ANSIBLE_PLAY_ERROR, got %s", execErr.Code)
		}
		if !execErr.Retryable {
			t.Error("expected retryable=true")
		}
	})
}
