package environments

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

const defaultJobType = "env_create"

var ErrMutationConflict = errors.New("environment mutation conflict")

type JobQueue interface {
	Enqueue(ctx context.Context, input jobs.EnqueueInput) (jobs.Job, error)
}

type Service struct {
	store   store.SiteStore
	queue   JobQueue
	now     func() time.Time
	jobType string
}

type CreateInput struct {
	SiteID              string
	Name                string
	Slug                string
	EnvironmentType     string
	SourceEnvironmentID *string
	PromotionPreset     string
}

func NewService(siteStore store.SiteStore, queue JobQueue) *Service {
	return &Service{
		store:   siteStore,
		queue:   queue,
		now:     func() time.Time { return time.Now().UTC() },
		jobType: defaultJobType,
	}
}

func (s *Service) ListBySiteID(ctx context.Context, siteID string) ([]store.Environment, error) {
	items, err := s.store.ListEnvironmentsBySiteID(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list site environments: %w", err)
	}
	return items, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (store.Environment, error) {
	item, err := s.store.GetEnvironmentByID(ctx, id)
	if err != nil {
		return store.Environment{}, fmt.Errorf("get environment by id: %w", err)
	}
	return item, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (string, error) {
	now := s.now()
	environment, err := s.store.CreateEnvironment(ctx, store.CreateEnvironmentInput{
		SiteID:              input.SiteID,
		Name:                input.Name,
		Slug:                input.Slug,
		EnvironmentType:     input.EnvironmentType,
		SourceEnvironmentID: input.SourceEnvironmentID,
		PromotionPreset:     input.PromotionPreset,
		Now:                 now,
	})
	if err != nil {
		return "", fmt.Errorf("create environment: %w", err)
	}

	job, err := s.queue.Enqueue(ctx, jobs.EnqueueInput{
		JobType:       s.jobType,
		SiteID:        &environment.SiteID,
		EnvironmentID: &environment.ID,
		NodeID:        &environment.NodeID,
		MaxAttempts:   3,
		CreatedAt:     now,
	})
	if err != nil {
		if errors.Is(err, jobs.ErrConflict) {
			return "", ErrMutationConflict
		}
		return "", fmt.Errorf("enqueue environment create job: %w", err)
	}

	return job.ID, nil
}
