package sites

import (
	"context"
	"errors"
	"fmt"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

const (
	defaultNodeID  = "44444444-4444-4444-4444-444444444444"
	defaultNodeIP  = "127.0.0.1"
	defaultJobType = "site_create"
)

var ErrMutationConflict = errors.New("site mutation conflict")

type JobQueue interface {
	Enqueue(ctx context.Context, input jobs.EnqueueInput) (jobs.Job, error)
}

type Service struct {
	store   store.SiteStore
	queue   JobQueue
	now     func() time.Time
	nodeID  string
	nodeIP  string
	jobType string
}

func NewService(siteStore store.SiteStore, queue JobQueue) *Service {
	return &Service{
		store:   siteStore,
		queue:   queue,
		now:     func() time.Time { return time.Now().UTC() },
		nodeID:  defaultNodeID,
		nodeIP:  defaultNodeIP,
		jobType: defaultJobType,
	}
}

func (s *Service) List(ctx context.Context) ([]store.Site, error) {
	items, err := s.store.ListSites(ctx)
	if err != nil {
		return nil, fmt.Errorf("list sites: %w", err)
	}
	return items, nil
}

func (s *Service) GetByID(ctx context.Context, id string) (store.Site, error) {
	site, err := s.store.GetSiteByID(ctx, id)
	if err != nil {
		return store.Site{}, fmt.Errorf("get site by id: %w", err)
	}
	return site, nil
}

func (s *Service) Create(ctx context.Context, name string, slug string) (string, error) {
	now := s.now()
	site, _, err := s.store.CreateSiteWithProductionEnvironment(ctx, store.CreateSiteInput{
		Name:       name,
		Slug:       slug,
		NodeID:     s.nodeID,
		NodePublic: s.nodeIP,
		Now:        now,
	})
	if err != nil {
		return "", fmt.Errorf("create site and environment: %w", err)
	}

	job, err := s.queue.Enqueue(ctx, jobs.EnqueueInput{
		JobType:       s.jobType,
		SiteID:        &site.ID,
		EnvironmentID: site.PrimaryEnvironmentID,
		NodeID:        &s.nodeID,
		MaxAttempts:   3,
		CreatedAt:     now,
	})
	if err != nil {
		if errors.Is(err, jobs.ErrConflict) {
			return "", ErrMutationConflict
		}
		return "", fmt.Errorf("enqueue site create job: %w", err)
	}

	return job.ID, nil
}
