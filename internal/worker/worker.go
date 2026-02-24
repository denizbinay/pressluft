package worker

import (
	"context"
	"log/slog"
	"time"

	"pressluft/internal/orchestrator"
)

// Config holds worker configuration.
type Config struct {
	PollInterval time.Duration
}

// DefaultConfig returns sensible defaults.
func DefaultConfig() Config {
	return Config{
		PollInterval: 2 * time.Second,
	}
}

// Worker polls for queued jobs and executes them.
type Worker struct {
	jobStore *orchestrator.Store
	executor *Executor
	config   Config
	logger   *slog.Logger
}

// New creates a worker with the given dependencies.
func New(jobStore *orchestrator.Store, executor *Executor, logger *slog.Logger, config Config) *Worker {
	if config.PollInterval <= 0 {
		config.PollInterval = DefaultConfig().PollInterval
	}
	return &Worker{
		jobStore: jobStore,
		executor: executor,
		config:   config,
		logger:   logger,
	}
}

// Run starts the polling loop. It blocks until ctx is cancelled.
func (w *Worker) Run(ctx context.Context) {
	w.logger.Info("worker started", "poll_interval", w.config.PollInterval)

	ticker := time.NewTicker(w.config.PollInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			w.logger.Info("worker shutting down")
			return
		case <-ticker.C:
			w.poll(ctx)
		}
	}
}

func (w *Worker) poll(ctx context.Context) {
	job, err := w.jobStore.ClaimNextJob(ctx)
	if err != nil {
		w.logger.Error("failed to claim job", "error", err)
		return
	}
	if job == nil {
		// No jobs available
		return
	}

	w.logger.Info("claimed job", "job_id", job.ID, "kind", job.Kind, "server_id", job.ServerID)

	// Execute the job (blocking)
	if err := w.executor.Execute(ctx, job); err != nil {
		w.logger.Error("job execution failed", "job_id", job.ID, "error", err)
		// Executor handles marking the job as failed
		return
	}

	w.logger.Info("job completed", "job_id", job.ID)
}
