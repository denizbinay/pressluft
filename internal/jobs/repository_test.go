package jobs

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestClaimNextRunnableEnforcesSiteAndNodeConcurrency(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 5, 0, 0, time.UTC)

	siteA := "site-a"
	siteB := "site-b"
	nodeA := "node-a"
	nodeB := "node-b"
	nodeC := "node-c"

	repo := NewInMemoryRepository([]Job{
		{
			ID:           "running-lock",
			JobType:      "site_create",
			Status:       StatusRunning,
			SiteID:       &siteA,
			NodeID:       &nodeA,
			AttemptCount: 1,
			MaxAttempts:  3,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "blocked-site",
			JobType:      "node_provision",
			Status:       StatusQueued,
			SiteID:       &siteA,
			NodeID:       &nodeB,
			AttemptCount: 0,
			MaxAttempts:  3,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "blocked-node",
			JobType:      "node_provision",
			Status:       StatusQueued,
			SiteID:       &siteB,
			NodeID:       &nodeA,
			AttemptCount: 0,
			MaxAttempts:  3,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
		{
			ID:           "eligible",
			JobType:      "node_provision",
			Status:       StatusQueued,
			SiteID:       &siteB,
			NodeID:       &nodeC,
			AttemptCount: 0,
			MaxAttempts:  3,
			CreatedAt:    now,
			UpdatedAt:    now,
		},
	})

	claimed, ok, err := repo.ClaimNextRunnable(context.Background(), "worker-1", now)
	if err != nil {
		t.Fatalf("ClaimNextRunnable() error = %v", err)
	}
	if !ok {
		t.Fatal("ClaimNextRunnable() ok = false, want true")
	}
	if claimed.ID != "eligible" {
		t.Fatalf("claimed job = %q, want eligible", claimed.ID)
	}
}

func TestRequeueClearsLockAndSchedulesRunAfter(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 6, 0, 0, time.UTC)
	runAfter := now.Add(time.Minute)

	repo := NewInMemoryRepository([]Job{{
		ID:           "job-1",
		Status:       StatusRunning,
		JobType:      "node_provision",
		AttemptCount: 1,
		MaxAttempts:  3,
		CreatedAt:    now,
		UpdatedAt:    now,
	}})

	updated, err := repo.Requeue(context.Background(), "job-1", runAfter, "ANSIBLE_HOST_UNREACHABLE", "timeout", now)
	if err != nil {
		t.Fatalf("Requeue() error = %v", err)
	}

	if updated.Status != StatusQueued {
		t.Fatalf("status = %s, want queued", updated.Status)
	}
	if updated.LockedAt != nil || updated.LockedBy != nil {
		t.Fatal("lock fields should be cleared on requeue")
	}
	if updated.RunAfter == nil || !updated.RunAfter.Equal(runAfter) {
		t.Fatalf("run_after = %v, want %v", updated.RunAfter, runAfter)
	}
}

func TestEnqueueBlocksConflictingQueuedOrRunningMutations(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 7, 0, 0, time.UTC)
	siteID := "site-a"
	nodeID := "node-a"

	repo := NewInMemoryRepository([]Job{{
		ID:           "job-running",
		JobType:      "site_create",
		Status:       StatusRunning,
		SiteID:       &siteID,
		NodeID:       &nodeID,
		AttemptCount: 1,
		MaxAttempts:  3,
		CreatedAt:    now,
		UpdatedAt:    now,
	}})

	_, err := repo.Enqueue(context.Background(), EnqueueInput{
		ID:          "job-new",
		JobType:     "env_create",
		SiteID:      &siteID,
		NodeID:      &nodeID,
		MaxAttempts: 3,
		CreatedAt:   now,
	})
	if !errors.Is(err, ErrConflict) {
		t.Fatalf("Enqueue() error = %v, want ErrConflict", err)
	}
}
