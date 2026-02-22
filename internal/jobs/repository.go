package jobs

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrNotFound = errors.New("job not found")

type Status string

const (
	StatusQueued    Status = "queued"
	StatusRunning   Status = "running"
	StatusSucceeded Status = "succeeded"
	StatusFailed    Status = "failed"
	StatusCancelled Status = "cancelled"
)

type Job struct {
	ID            string     `json:"id"`
	JobType       string     `json:"job_type,omitempty"`
	Status        Status     `json:"status"`
	SiteID        *string    `json:"site_id"`
	EnvironmentID *string    `json:"environment_id"`
	NodeID        *string    `json:"node_id"`
	AttemptCount  int        `json:"attempt_count"`
	MaxAttempts   int        `json:"max_attempts"`
	RunAfter      *time.Time `json:"run_after"`
	LockedAt      *time.Time `json:"locked_at"`
	LockedBy      *string    `json:"locked_by"`
	StartedAt     *time.Time `json:"started_at"`
	FinishedAt    *time.Time `json:"finished_at"`
	ErrorCode     *string    `json:"error_code"`
	ErrorMessage  *string    `json:"error_message"`
	CreatedAt     time.Time  `json:"created_at"`
	UpdatedAt     time.Time  `json:"updated_at"`
}

type Reader interface {
	List(ctx context.Context) ([]Job, error)
	GetByID(ctx context.Context, id string) (Job, error)
	CountByStatus(ctx context.Context, status Status) (int, error)
}

type InMemoryRepository struct {
	mu    sync.RWMutex
	order []string
	byID  map[string]Job
}

func NewInMemoryRepository(seed []Job) *InMemoryRepository {
	byID := make(map[string]Job, len(seed))
	order := make([]string, 0, len(seed))
	for _, job := range seed {
		if job.ID == "" {
			continue
		}
		order = append(order, job.ID)
		byID[job.ID] = job
	}

	return &InMemoryRepository{order: order, byID: byID}
}

func (r *InMemoryRepository) List(context.Context) ([]Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	list := make([]Job, 0, len(r.order))
	for _, id := range r.order {
		list = append(list, r.byID[id])
	}
	return list, nil
}

func (r *InMemoryRepository) GetByID(_ context.Context, id string) (Job, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	job, ok := r.byID[id]
	if !ok {
		return Job{}, ErrNotFound
	}

	return job, nil
}

func (r *InMemoryRepository) CountByStatus(_ context.Context, status Status) (int, error) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, id := range r.order {
		if r.byID[id].Status == status {
			count++
		}
	}

	return count, nil
}
