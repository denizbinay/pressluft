package environments

import (
	"context"
	"errors"
	"testing"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/nodes"
	"pressluft/internal/store"
)

type fakeNodeStore struct {
	node nodes.Node
	err  error
}

func (f fakeNodeStore) GetByID(context.Context, string) (nodes.Node, error) {
	if f.err != nil {
		return nodes.Node{}, f.err
	}
	return f.node, nil
}

type fakeReadinessChecker struct {
	report nodes.ReadinessReport
	err    error
}

func (f fakeReadinessChecker) Evaluate(context.Context, nodes.Node) (nodes.ReadinessReport, error) {
	if f.err != nil {
		return nodes.ReadinessReport{}, f.err
	}
	return f.report, nil
}

func TestCreatePersistsEnvironmentAndEnqueuesJob(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	siteStore := store.NewInMemorySiteStore(0)
	queue := jobs.NewInMemoryRepository(nil)
	site, production, err := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
		Name:       "Acme",
		Slug:       "acme",
		NodeID:     "44444444-4444-4444-4444-444444444444",
		NodePublic: "127.0.0.1",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	svc := NewService(siteStore, queue, nil, nil)
	svc.now = func() time.Time { return now }

	jobID, err := svc.Create(context.Background(), CreateInput{
		SiteID:              site.ID,
		Name:                "Staging",
		Slug:                "staging",
		EnvironmentType:     "staging",
		SourceEnvironmentID: &production.ID,
		PromotionPreset:     "commerce-protect",
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	if jobID == "" {
		t.Fatal("Create() jobID empty")
	}

	environments, err := svc.ListBySiteID(context.Background(), site.ID)
	if err != nil {
		t.Fatalf("ListBySiteID() error = %v", err)
	}
	if len(environments) != 2 {
		t.Fatalf("environments count = %d, want 2", len(environments))
	}

	created := environments[1]
	if created.Status != "cloning" {
		t.Fatalf("status = %q, want cloning", created.Status)
	}
	if created.SourceEnvironmentID == nil || *created.SourceEnvironmentID != production.ID {
		t.Fatalf("source_environment_id = %v, want %s", created.SourceEnvironmentID, production.ID)
	}

	job, err := queue.GetByID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetByID(job) error = %v", err)
	}
	if job.JobType != "env_create" {
		t.Fatalf("job_type = %q, want env_create", job.JobType)
	}
	if job.EnvironmentID == nil || *job.EnvironmentID != created.ID {
		t.Fatalf("job environment_id = %v, want %s", job.EnvironmentID, created.ID)
	}

	updatedSite, err := siteStore.GetSiteByID(context.Background(), site.ID)
	if err != nil {
		t.Fatalf("GetSiteByID() error = %v", err)
	}
	if updatedSite.Status != "cloning" {
		t.Fatalf("site status = %q, want cloning", updatedSite.Status)
	}
}

func TestCreateReturnsConflictWhenMutationAlreadyQueued(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	siteStore := store.NewInMemorySiteStore(0)
	site, production, err := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
		Name:       "Acme",
		Slug:       "acme",
		NodeID:     "44444444-4444-4444-4444-444444444444",
		NodePublic: "127.0.0.1",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	queue := jobs.NewInMemoryRepository([]jobs.Job{{
		ID:            "job-existing",
		JobType:       "env_create",
		Status:        jobs.StatusQueued,
		SiteID:        &site.ID,
		EnvironmentID: &production.ID,
		NodeID:        &production.NodeID,
		AttemptCount:  0,
		MaxAttempts:   3,
		CreatedAt:     now,
		UpdatedAt:     now,
	}})

	svc := NewService(siteStore, queue, nil, nil)
	svc.now = func() time.Time { return now }

	_, err = svc.Create(context.Background(), CreateInput{
		SiteID:              site.ID,
		Name:                "Clone",
		Slug:                "clone-1",
		EnvironmentType:     "clone",
		SourceEnvironmentID: &production.ID,
		PromotionPreset:     "content-protect",
	})
	if !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("Create() error = %v, want ErrMutationConflict", err)
	}
}

func TestCreateRejectsMissingSourceEnvironment(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	siteStore := store.NewInMemorySiteStore(0)
	queue := jobs.NewInMemoryRepository(nil)
	site, _, err := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
		Name:       "Acme",
		Slug:       "acme",
		NodeID:     "44444444-4444-4444-4444-444444444444",
		NodePublic: "127.0.0.1",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	svc := NewService(siteStore, queue, nil, nil)
	_, err = svc.Create(context.Background(), CreateInput{
		SiteID:          site.ID,
		Name:            "Staging",
		Slug:            "staging",
		EnvironmentType: "staging",
		PromotionPreset: "content-protect",
	})
	if !errors.Is(err, store.ErrInvalidEnvironmentCreate) {
		t.Fatalf("Create() error = %v, want ErrInvalidEnvironmentCreate", err)
	}
}

func TestCreateFailsWhenTargetNodeNotReady(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	siteStore := store.NewInMemorySiteStore(0)
	queue := jobs.NewInMemoryRepository(nil)
	site, production, err := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
		Name:       "Acme",
		Slug:       "acme",
		NodeID:     nodes.SelfNodeID,
		NodePublic: "127.0.0.1",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	readiness := fakeReadinessChecker{report: nodes.ReadinessReport{IsReady: false, ReasonCodes: []string{nodes.ReasonRuntimeMissing}}}
	svc := NewService(siteStore, queue, fakeNodeStore{node: nodes.Node{ID: nodes.SelfNodeID}}, readiness)

	_, err = svc.Create(context.Background(), CreateInput{
		SiteID:              site.ID,
		Name:                "Staging",
		Slug:                "staging",
		EnvironmentType:     "staging",
		SourceEnvironmentID: &production.ID,
		PromotionPreset:     "commerce-protect",
	})
	if !errors.Is(err, ErrNodeNotReady) {
		t.Fatalf("Create() error = %v, want ErrNodeNotReady", err)
	}
}
