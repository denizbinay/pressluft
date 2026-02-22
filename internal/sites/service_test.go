package sites

import (
	"context"
	"errors"
	"testing"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/nodes"
	"pressluft/internal/store"
)

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

func TestCreatePersistsSiteEnvironmentAndEnqueuesJob(t *testing.T) {
	siteStore := store.NewInMemorySiteStore(0)
	nodeStore := nodes.NewInMemoryStore(nil)
	queue := jobs.NewInMemoryRepository(nil)
	svc := NewService(siteStore, nodeStore, queue, nil)
	svc.now = func() time.Time { return time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC) }
	_, _ = nodeStore.Create(context.Background(), nodes.CreateInput{
		ProviderID: "hetzner",
		Name:       "provider-node",
		Hostname:   "192.0.2.20",
		PublicIP:   "192.0.2.20",
		SSHPort:    22,
		SSHUser:    "ubuntu",
		IsLocal:    false,
		Now:        svc.now(),
	})

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

	nodeStore := nodes.NewInMemoryStore(nil)
	_, _ = nodeStore.Create(context.Background(), nodes.CreateInput{
		ProviderID: "hetzner",
		Name:       "provider-node",
		Hostname:   "192.0.2.30",
		PublicIP:   "192.0.2.30",
		SSHPort:    22,
		SSHUser:    "ubuntu",
		IsLocal:    false,
		Now:        now,
	})
	svc := NewService(siteStore, nodeStore, queue, nil)
	svc.now = func() time.Time { return now }

	_, err := svc.Create(context.Background(), "Existing", "existing")
	if !errors.Is(err, store.ErrSiteSlugConflict) {
		t.Fatalf("Create() error = %v, want site slug conflict", err)
	}
}

func TestCreateUsesProviderBackedNodeAsTarget(t *testing.T) {
	siteStore := store.NewInMemorySiteStore(0)
	nodeStore := nodes.NewInMemoryStore(nil)
	queue := jobs.NewInMemoryRepository(nil)
	svc := NewService(siteStore, nodeStore, queue, nil)
	svc.now = func() time.Time { return time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC) }
	providerNode, _ := nodeStore.Create(context.Background(), nodes.CreateInput{
		ProviderID: "hetzner",
		Name:       "provider-node",
		Hostname:   "192.0.2.21",
		PublicIP:   "192.0.2.21",
		SSHPort:    22,
		SSHUser:    "ubuntu",
		IsLocal:    false,
		Now:        svc.now(),
	})

	jobID, err := svc.Create(context.Background(), "Test Site", "test-site")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	job, err := queue.GetByID(context.Background(), jobID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}

	if job.NodeID == nil || *job.NodeID != providerNode.ID {
		t.Fatalf("job.NodeID = %v, want %s", job.NodeID, providerNode.ID)
	}
}

func TestCreateFailsWhenNoProviderNodeRegistered(t *testing.T) {
	siteStore := store.NewInMemorySiteStore(0)
	nodeStore := nodes.NewInMemoryStore(nil)
	queue := jobs.NewInMemoryRepository(nil)
	svc := NewService(siteStore, nodeStore, queue, nil)

	_, err := svc.Create(context.Background(), "Acme", "acme")
	if !errors.Is(err, ErrNodeNotReady) {
		t.Fatalf("Create() error = %v, want ErrNodeNotReady", err)
	}
}

func TestCreateFailsWhenTargetNodeNotReady(t *testing.T) {
	siteStore := store.NewInMemorySiteStore(0)
	nodeStore := nodes.NewInMemoryStore(nil)
	queue := jobs.NewInMemoryRepository(nil)
	readiness := fakeReadinessChecker{report: nodes.ReadinessReport{IsReady: false, ReasonCodes: []string{nodes.ReasonSudoUnavailable}, Guidance: []string{"configure sudo"}}}
	svc := NewService(siteStore, nodeStore, queue, readiness)
	_, _ = nodeStore.Create(context.Background(), nodes.CreateInput{
		ProviderID: "hetzner",
		Name:       "provider-node",
		Hostname:   "192.0.2.22",
		PublicIP:   "192.0.2.22",
		SSHPort:    22,
		SSHUser:    "ubuntu",
		IsLocal:    false,
		Now:        time.Now().UTC(),
	})

	_, err := svc.Create(context.Background(), "Acme", "acme")
	if !errors.Is(err, ErrNodeNotReady) {
		t.Fatalf("Create() error = %v, want ErrNodeNotReady", err)
	}
}
