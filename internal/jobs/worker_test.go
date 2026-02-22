package jobs

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"pressluft/internal/audit"
)

func TestWorkerProcessNextMarksSucceeded(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 10, 0, 0, time.UTC)
	repo := NewInMemoryRepository([]Job{{
		ID:           "job-success",
		JobType:      "node_provision",
		Status:       StatusQueued,
		AttemptCount: 0,
		MaxAttempts:  3,
		CreatedAt:    now,
		UpdatedAt:    now,
	}})

	worker := NewWorker(repo, "worker-1", map[string]Handler{
		"node_provision": func(context.Context, Job) error { return nil },
	}, nil)
	worker.now = func() time.Time { return now }

	processed, err := worker.ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessNext() processed = false, want true")
	}

	job, err := repo.GetByID(context.Background(), "job-success")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if job.Status != StatusSucceeded {
		t.Fatalf("status = %s, want succeeded", job.Status)
	}
	if job.AttemptCount != 1 {
		t.Fatalf("attempt_count = %d, want 1", job.AttemptCount)
	}
}

func TestWorkerProcessNextRetryableErrorRequeuesWithBackoff(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 12, 0, 0, time.UTC)
	repo := NewInMemoryRepository([]Job{{
		ID:           "job-retry",
		JobType:      "node_provision",
		Status:       StatusQueued,
		AttemptCount: 0,
		MaxAttempts:  3,
		CreatedAt:    now,
		UpdatedAt:    now,
	}})

	worker := NewWorker(repo, "worker-1", map[string]Handler{
		"node_provision": func(context.Context, Job) error {
			return ExecutionError{Code: "ANSIBLE_HOST_UNREACHABLE", Message: "ssh timeout", Retryable: true}
		},
	}, nil)
	worker.now = func() time.Time { return now }

	processed, err := worker.ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessNext() processed = false, want true")
	}

	job, err := repo.GetByID(context.Background(), "job-retry")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if job.Status != StatusQueued {
		t.Fatalf("status = %s, want queued", job.Status)
	}
	if job.AttemptCount != 1 {
		t.Fatalf("attempt_count = %d, want 1", job.AttemptCount)
	}
	if job.RunAfter == nil {
		t.Fatal("run_after is nil, want scheduled retry")
	}
	want := now.Add(time.Minute)
	if !job.RunAfter.Equal(want) {
		t.Fatalf("run_after = %s, want %s", job.RunAfter.Format(time.RFC3339), want.Format(time.RFC3339))
	}
}

func TestWorkerProcessNextRetryableErrorAtMaxAttemptsFails(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 14, 0, 0, time.UTC)
	repo := NewInMemoryRepository([]Job{{
		ID:           "job-max",
		JobType:      "node_provision",
		Status:       StatusQueued,
		AttemptCount: 2,
		MaxAttempts:  3,
		CreatedAt:    now,
		UpdatedAt:    now,
	}})

	worker := NewWorker(repo, "worker-1", map[string]Handler{
		"node_provision": func(context.Context, Job) error {
			return ExecutionError{Code: "ANSIBLE_HOST_UNREACHABLE", Message: "still unreachable", Retryable: true}
		},
	}, nil)
	worker.now = func() time.Time { return now }

	processed, err := worker.ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessNext() processed = false, want true")
	}

	job, err := repo.GetByID(context.Background(), "job-max")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if job.Status != StatusFailed {
		t.Fatalf("status = %s, want failed", job.Status)
	}
	if job.AttemptCount != 3 {
		t.Fatalf("attempt_count = %d, want 3", job.AttemptCount)
	}
	if job.ErrorCode == nil || *job.ErrorCode != "ANSIBLE_HOST_UNREACHABLE" {
		t.Fatalf("error_code = %v, want ANSIBLE_HOST_UNREACHABLE", job.ErrorCode)
	}
}

func TestWorkerProcessNextTruncatesLongErrorMessage(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 16, 0, 0, time.UTC)
	repo := NewInMemoryRepository([]Job{{
		ID:           "job-truncate",
		JobType:      "node_provision",
		Status:       StatusQueued,
		AttemptCount: 2,
		MaxAttempts:  3,
		CreatedAt:    now,
		UpdatedAt:    now,
	}})

	worker := NewWorker(repo, "worker-1", map[string]Handler{
		"node_provision": func(context.Context, Job) error {
			return errors.New(strings.Repeat("x", 12*1024))
		},
	}, nil)
	worker.now = func() time.Time { return now }

	processed, err := worker.ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessNext() processed = false, want true")
	}

	job, err := repo.GetByID(context.Background(), "job-truncate")
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if job.ErrorMessage == nil {
		t.Fatal("error_message is nil")
	}
	if len(*job.ErrorMessage) != 10*1024 {
		t.Fatalf("error_message length = %d, want %d", len(*job.ErrorMessage), 10*1024)
	}
}

func TestWorkerProcessNextWritesAndUpdatesAsyncAuditEntry(t *testing.T) {
	now := time.Date(2026, 2, 22, 0, 18, 0, 0, time.UTC)
	repo := NewInMemoryRepository([]Job{{
		ID:           "job-audit-success",
		JobType:      "node_provision",
		Status:       StatusQueued,
		AttemptCount: 0,
		MaxAttempts:  3,
		CreatedAt:    now,
		UpdatedAt:    now,
	}})
	auditStore := audit.NewInMemoryStore()
	auditService := audit.NewService(auditStore)

	worker := NewWorker(repo, "worker-1", map[string]Handler{
		"node_provision": func(context.Context, Job) error { return nil },
	}, auditService)
	worker.now = func() time.Time { return now }

	processed, err := worker.ProcessNext(context.Background())
	if err != nil {
		t.Fatalf("ProcessNext() error = %v", err)
	}
	if !processed {
		t.Fatal("ProcessNext() processed = false, want true")
	}

	entries, err := auditStore.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(entries) != 1 {
		t.Fatalf("audit entries len = %d, want 1", len(entries))
	}
	if entries[0].Result != "succeeded" {
		t.Fatalf("audit result = %s, want succeeded", entries[0].Result)
	}
}
