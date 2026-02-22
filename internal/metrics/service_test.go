package metrics

import (
	"context"
	"errors"
	"testing"
	"time"

	"pressluft/internal/jobs"
)

func TestGetSnapshotAggregatesExpectedCounters(t *testing.T) {
	now := time.Now().UTC()
	jobStore := jobs.NewInMemoryRepository([]jobs.Job{
		{ID: "job-1", Status: jobs.StatusQueued, AttemptCount: 0, MaxAttempts: 3, CreatedAt: now, UpdatedAt: now},
		{ID: "job-2", Status: jobs.StatusRunning, AttemptCount: 1, MaxAttempts: 3, CreatedAt: now, UpdatedAt: now},
		{ID: "job-3", Status: jobs.StatusFailed, AttemptCount: 1, MaxAttempts: 3, CreatedAt: now, UpdatedAt: now},
	})

	svc := NewService(jobStore, stubNodeCounter{count: 4}, stubSiteCounter{count: 9})

	snapshot, err := svc.GetSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetSnapshot() error = %v", err)
	}

	if snapshot.JobsQueued != 1 {
		t.Fatalf("JobsQueued = %d, want 1", snapshot.JobsQueued)
	}
	if snapshot.JobsRunning != 1 {
		t.Fatalf("JobsRunning = %d, want 1", snapshot.JobsRunning)
	}
	if snapshot.NodesActive != 4 {
		t.Fatalf("NodesActive = %d, want 4", snapshot.NodesActive)
	}
	if snapshot.SitesTotal != 9 {
		t.Fatalf("SitesTotal = %d, want 9", snapshot.SitesTotal)
	}
}

func TestGetSnapshotClampsNegativeCounters(t *testing.T) {
	now := time.Now().UTC()
	jobStore := jobs.NewInMemoryRepository([]jobs.Job{
		{ID: "job-1", Status: jobs.StatusQueued, AttemptCount: 0, MaxAttempts: 3, CreatedAt: now, UpdatedAt: now},
	})

	svc := NewService(jobStore, stubNodeCounter{count: -3}, stubSiteCounter{count: -1})

	snapshot, err := svc.GetSnapshot(context.Background())
	if err != nil {
		t.Fatalf("GetSnapshot() error = %v", err)
	}

	if snapshot.NodesActive != 0 {
		t.Fatalf("NodesActive = %d, want 0", snapshot.NodesActive)
	}
	if snapshot.SitesTotal != 0 {
		t.Fatalf("SitesTotal = %d, want 0", snapshot.SitesTotal)
	}
}

func TestGetSnapshotPropagatesCounterError(t *testing.T) {
	now := time.Now().UTC()
	jobStore := jobs.NewInMemoryRepository([]jobs.Job{
		{ID: "job-1", Status: jobs.StatusQueued, AttemptCount: 0, MaxAttempts: 3, CreatedAt: now, UpdatedAt: now},
	})

	svc := NewService(jobStore, stubNodeCounter{err: errors.New("boom")}, stubSiteCounter{count: 1})

	if _, err := svc.GetSnapshot(context.Background()); err == nil {
		t.Fatal("GetSnapshot() error = nil, want non-nil")
	}
}

type stubNodeCounter struct {
	count int
	err   error
}

func (s stubNodeCounter) CountActiveNodes(context.Context) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.count, nil
}

type stubSiteCounter struct {
	count int
	err   error
}

func (s stubSiteCounter) CountSites(context.Context) (int, error) {
	if s.err != nil {
		return 0, s.err
	}
	return s.count, nil
}
