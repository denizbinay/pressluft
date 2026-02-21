package metrics

import (
	"context"
	"database/sql"
	"fmt"
)

type Snapshot struct {
	JobsRunning int `json:"jobs_running"`
	JobsQueued  int `json:"jobs_queued"`
	NodesActive int `json:"nodes_active"`
	SitesTotal  int `json:"sites_total"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Snapshot(ctx context.Context) (Snapshot, error) {
	jobsRunning, err := countByQuery(ctx, s.db, `SELECT COUNT(1) FROM jobs WHERE status = 'running'`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("count running jobs: %w", err)
	}

	jobsQueued, err := countByQuery(ctx, s.db, `SELECT COUNT(1) FROM jobs WHERE status = 'queued'`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("count queued jobs: %w", err)
	}

	nodesActive, err := countByQuery(ctx, s.db, `SELECT COUNT(1) FROM nodes WHERE status = 'active'`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("count active nodes: %w", err)
	}

	sitesTotal, err := countByQuery(ctx, s.db, `SELECT COUNT(1) FROM sites`)
	if err != nil {
		return Snapshot{}, fmt.Errorf("count sites: %w", err)
	}

	return Snapshot{
		JobsRunning: jobsRunning,
		JobsQueued:  jobsQueued,
		NodesActive: nodesActive,
		SitesTotal:  sitesTotal,
	}, nil
}

func countByQuery(ctx context.Context, db *sql.DB, query string) (int, error) {
	var count int
	if err := db.QueryRowContext(ctx, query).Scan(&count); err != nil {
		return 0, err
	}
	return count, nil
}
