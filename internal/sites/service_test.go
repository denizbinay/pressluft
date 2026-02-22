package sites

import (
	"context"
	"errors"
	"testing"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

func TestCreatePersistsSiteEnvironmentAndEnqueuesJob(t *testing.T) {
	siteStore := store.NewInMemorySiteStore(0)
	queue := jobs.NewInMemoryRepository(nil)
	svc := NewService(siteStore, queue)
	svc.now = func() time.Time { return time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC) }

	jobID, err := svc.Create(context.Background(), "Acme", "acme")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if jobID == "" {
		t.Fatal("Create() jobID empty")
	}

	items, err := svc.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("sites count = %d, want 1", len(items))
	}
	site := items[0]
	if site.Status != "active" {
		t.Fatalf("site status = %s, want active", site.Status)
	}
	if site.PrimaryEnvironmentID == nil || *site.PrimaryEnvironmentID == "" {
		t.Fatal("site primary_environment_id missing")
	}

	env, err := siteStore.GetEnvironmentByID(context.Background(), *site.PrimaryEnvironmentID)
	if err != nil {
		t.Fatalf("GetEnvironmentByID() error = %v", err)
	}
	if env.EnvironmentType != "production" {
		t.Fatalf("environment_type = %s, want production", env.EnvironmentType)
	}
	if env.Status != "active" {
		t.Fatalf("environment status = %s, want active", env.Status)
	}
	if env.PreviewURL == "" {
		t.Fatal("preview_url empty")
	}

	job, err := queue.GetByID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetByID(job) error = %v", err)
	}
	if job.JobType != "site_create" {
		t.Fatalf("job_type = %s, want site_create", job.JobType)
	}
	if job.SiteID == nil || *job.SiteID != site.ID {
		t.Fatalf("job site_id = %v, want %s", job.SiteID, site.ID)
	}
}

func TestCreateReturnsConflictWhenSlugAlreadyExists(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	siteID := "a27651d9-cd74-407f-9fd6-4b73ec803d75"
	nodeID := "33333333-3333-3333-3333-333333333333"

	queue := jobs.NewInMemoryRepository([]jobs.Job{{
		ID:           "job-existing",
		JobType:      "site_create",
		Status:       jobs.StatusQueued,
		SiteID:       &siteID,
		NodeID:       &nodeID,
		AttemptCount: 0,
		MaxAttempts:  3,
		CreatedAt:    now,
		UpdatedAt:    now,
	}})

	siteStore := store.NewInMemorySiteStoreWithSeed([]store.Site{{
		ID:                   siteID,
		Name:                 "Existing",
		Slug:                 "existing",
		Status:               "active",
		PrimaryEnvironmentID: nil,
		CreatedAt:            now,
		UpdatedAt:            now,
		StateVersion:         1,
	}})

	svc := NewService(siteStore, queue)
	svc.now = func() time.Time { return now }

	_, err := svc.Create(context.Background(), "Existing", "existing")
	if !errors.Is(err, store.ErrSiteSlugConflict) {
		t.Fatalf("Create() error = %v, want site slug conflict", err)
	}
}
