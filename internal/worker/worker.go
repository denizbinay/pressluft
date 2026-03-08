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
	JobTimeouts  map[string]time.Duration
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
	executor jobExecutor
	config   Config
	logger   *slog.Logger
}

type jobExecutor interface {
	Execute(ctx context.Context, job *orchestrator.Job) error
}

// New creates a worker with the given dependencies.
func New(jobStore *orchestrator.Store, executor jobExecutor, logger *slog.Logger, config Config) *Worker {
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

	// Recover any jobs that were interrupted by a previous shutdown.
	if recovered, err := w.jobStore.RecoverStuckJobs(ctx); err != nil {
		w.logger.Error("job recovery failed", "error", err)
	} else if recovered > 0 {
		w.logger.Info("job recovery completed", "recovered_jobs", recovered)
	}

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
		w.logger.Error("job claim failed", "error", err)
		return
	}
	if job == nil {
		// No jobs available
		return
	}

	w.logger.Info("job claimed", "job_id", job.ID, "job_kind", job.Kind, "server_id", job.ServerID)

	policy, ok := orchestrator.JobKindPolicy(job.Kind)
	if !ok {
		w.logger.Error("claimed job kind unsupported", "job_id", job.ID, "job_kind", job.Kind, "server_id", job.ServerID)
		return
	}

	jobCtx := ctx
	cancel := func() {}
	timeout := policy.Timeout
	if override, ok := w.config.JobTimeouts[job.Kind]; ok && override > 0 {
		timeout = override
	}
	if timeout > 0 {
		jobCtx, cancel = context.WithTimeout(ctx, timeout)
	}
	defer cancel()

	if err := w.executor.Execute(jobCtx, job); err != nil {
		w.logger.Error("job execution failed", "job_id", job.ID, "job_kind", job.Kind, "server_id", job.ServerID, "error", err)
		if jobCtx.Err() == context.DeadlineExceeded {
			message := "job timed out before completion"
			if _, changed, markErr := w.jobStore.MarkJobTimedOut(ctx, job.ID, message); markErr != nil {
				w.logger.Error("job timeout persistence failed", "job_id", job.ID, "job_kind", job.Kind, "server_id", job.ServerID, "error", markErr)
			} else if changed {
				w.logger.Warn("job timed out", "job_id", job.ID, "job_kind", job.Kind, "server_id", job.ServerID, "timeout", timeout)
			}
		}
		return
	}

	w.logger.Info("job execution returned", "job_id", job.ID, "job_kind", job.Kind, "server_id", job.ServerID)
}
