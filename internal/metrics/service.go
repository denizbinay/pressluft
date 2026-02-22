package metrics

import (
	"context"
	"fmt"

	"pressluft/internal/jobs"
)

type NodeCounter interface {
	CountActiveNodes(ctx context.Context) (int, error)
}

type SiteCounter interface {
	CountSites(ctx context.Context) (int, error)
}

type Service struct {
	jobs  jobs.Reader
	nodes NodeCounter
	sites SiteCounter
}

type Snapshot struct {
	JobsRunning int `json:"jobs_running"`
	JobsQueued  int `json:"jobs_queued"`
	NodesActive int `json:"nodes_active"`
	SitesTotal  int `json:"sites_total"`
}

func NewService(jobsReader jobs.Reader, nodeCounter NodeCounter, siteCounter SiteCounter) *Service {
	return &Service{jobs: jobsReader, nodes: nodeCounter, sites: siteCounter}
}

func (s *Service) GetSnapshot(ctx context.Context) (Snapshot, error) {
	running, err := s.jobs.CountByStatus(ctx, jobs.StatusRunning)
	if err != nil {
		return Snapshot{}, fmt.Errorf("count running jobs: %w", err)
	}

	queued, err := s.jobs.CountByStatus(ctx, jobs.StatusQueued)
	if err != nil {
		return Snapshot{}, fmt.Errorf("count queued jobs: %w", err)
	}

	activeNodes, err := s.nodes.CountActiveNodes(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("count active nodes: %w", err)
	}

	totalSites, err := s.sites.CountSites(ctx)
	if err != nil {
		return Snapshot{}, fmt.Errorf("count total sites: %w", err)
	}

	return Snapshot{
		JobsRunning: clampNonNegative(running),
		JobsQueued:  clampNonNegative(queued),
		NodesActive: clampNonNegative(activeNodes),
		SitesTotal:  clampNonNegative(totalSites),
	}, nil
}

func clampNonNegative(value int) int {
	if value < 0 {
		return 0
	}
	return value
}
