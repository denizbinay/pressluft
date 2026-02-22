package jobs

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrNotFound = errors.New("job not found")
var ErrInvalidTransition = errors.New("invalid job state transition")

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

type QueueStore interface {
	Reader
	ClaimNextRunnable(ctx context.Context, workerID string, now time.Time) (Job, bool, error)
	CompleteSuccess(ctx context.Context, id string, now time.Time) (Job, error)
	CompleteFailure(ctx context.Context, id string, errorCode string, errorMessage string, now time.Time) (Job, error)
	Requeue(ctx context.Context, id string, runAfter time.Time, errorCode string, errorMessage string, now time.Time) (Job, error)
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

func (r *InMemoryRepository) ClaimNextRunnable(_ context.Context, workerID string, now time.Time) (Job, bool, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	for _, id := range r.order {
		job := r.byID[id]
		if !isRunnable(job, now) {
			continue
		}
		if !r.canRunConcurrently(job) {
			continue
		}

		job.Status = StatusRunning
		job.AttemptCount++
		job.LockedAt = timePtr(now)
		job.LockedBy = stringPtr(workerID)
		job.StartedAt = timePtr(now)
		job.FinishedAt = nil
		job.UpdatedAt = now
		r.byID[id] = job

		return job, true, nil
	}

	return Job{}, false, nil
}

func (r *InMemoryRepository) CompleteSuccess(_ context.Context, id string, now time.Time) (Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.byID[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	if job.Status != StatusRunning {
		return Job{}, fmt.Errorf("complete success: %w", ErrInvalidTransition)
	}

	job.Status = StatusSucceeded
	job.FinishedAt = timePtr(now)
	job.UpdatedAt = now
	r.byID[id] = job

	return job, nil
}

func (r *InMemoryRepository) CompleteFailure(_ context.Context, id string, errorCode string, errorMessage string, now time.Time) (Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.byID[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	if job.Status != StatusRunning {
		return Job{}, fmt.Errorf("complete failure: %w", ErrInvalidTransition)
	}

	job.Status = StatusFailed
	job.ErrorCode = stringPtr(errorCode)
	job.ErrorMessage = stringPtr(errorMessage)
	job.FinishedAt = timePtr(now)
	job.UpdatedAt = now
	r.byID[id] = job

	return job, nil
}

func (r *InMemoryRepository) Requeue(_ context.Context, id string, runAfter time.Time, errorCode string, errorMessage string, now time.Time) (Job, error) {
	r.mu.Lock()
	defer r.mu.Unlock()

	job, ok := r.byID[id]
	if !ok {
		return Job{}, ErrNotFound
	}
	if job.Status != StatusRunning {
		return Job{}, fmt.Errorf("requeue: %w", ErrInvalidTransition)
	}

	job.Status = StatusQueued
	job.RunAfter = timePtr(runAfter)
	job.LockedAt = nil
	job.LockedBy = nil
	job.StartedAt = nil
	job.FinishedAt = nil
	job.ErrorCode = stringPtr(errorCode)
	job.ErrorMessage = stringPtr(errorMessage)
	job.UpdatedAt = now
	r.byID[id] = job

	return job, nil
}

func isRunnable(job Job, now time.Time) bool {
	if job.Status != StatusQueued {
		return false
	}
	if job.RunAfter == nil {
		return true
	}
	return !job.RunAfter.After(now)
}

func (r *InMemoryRepository) canRunConcurrently(job Job) bool {
	for _, otherID := range r.order {
		other := r.byID[otherID]
		if other.Status != StatusRunning {
			continue
		}

		if job.SiteID != nil && other.SiteID != nil && *job.SiteID == *other.SiteID {
			return false
		}

		if job.NodeID != nil && other.NodeID != nil && *job.NodeID == *other.NodeID {
			return false
		}
	}

	return true
}

func timePtr(v time.Time) *time.Time {
	t := v
	return &t
}

func stringPtr(v string) *string {
	s := v
	return &s
}
