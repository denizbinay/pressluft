package worker

import (
	"context"
	"database/sql"
	"errors"
	"io"
	"log/slog"
	"testing"
	"time"

	"pressluft/internal/orchestrator"

	_ "modernc.org/sqlite"
)

func TestWorkerRecoversInterruptedJobsOnStartup(t *testing.T) {
	store := newWorkerJobStore(t)
	job, err := store.CreateJob(context.Background(), orchestrator.CreateJobInput{Kind: string(orchestrator.JobKindDeleteServer), ServerID: 1})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := store.TransitionJob(context.Background(), job.ID, orchestrator.TransitionInput{ToStatus: orchestrator.JobStatusRunning, CurrentStep: "delete"}); err != nil {
		t.Fatalf("transition job: %v", err)
	}

	w := New(store, noopExecutor{}, testWorkerLogger(), Config{PollInterval: 10 * time.Millisecond})
	ctx, cancel := context.WithCancel(context.Background())
	done := make(chan struct{})
	go func() {
		defer close(done)
		w.Run(ctx)
	}()

	time.Sleep(25 * time.Millisecond)
	cancel()
	<-done

	updated, err := store.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if updated.Status != orchestrator.JobStatusFailed {
		t.Fatalf("status = %q, want %q", updated.Status, orchestrator.JobStatusFailed)
	}
	if updated.LastError == "" {
		t.Fatal("expected recovery reason")
	}
}

func TestWorkerMarksTimedOutJobFailed(t *testing.T) {
	store := newWorkerJobStore(t)
	job, err := store.CreateJob(context.Background(), orchestrator.CreateJobInput{Kind: string(orchestrator.JobKindRestartService), ServerID: 2})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	executor := blockingExecutor{wait: 50 * time.Millisecond}
	w := New(store, executor, testWorkerLogger(), Config{
		PollInterval: 5 * time.Millisecond,
		JobTimeouts: map[string]time.Duration{
			string(orchestrator.JobKindRestartService): 10 * time.Millisecond,
		},
	})
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	w.poll(ctx)

	updated, err := store.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if updated.Status != orchestrator.JobStatusFailed {
		t.Fatalf("status = %q, want %q", updated.Status, orchestrator.JobStatusFailed)
	}
	if updated.LastError != "job timed out before completion" {
		t.Fatalf("last_error = %q", updated.LastError)
	}

	events, err := store.ListAllEvents(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].EventType != orchestrator.JobEventTypeTimedOut {
		t.Fatalf("unexpected timeout events: %+v", events)
	}
}

type noopExecutor struct{}

func (noopExecutor) Execute(context.Context, *orchestrator.Job) error { return nil }

type blockingExecutor struct {
	wait time.Duration
}

func (e blockingExecutor) Execute(ctx context.Context, _ *orchestrator.Job) error {
	select {
	case <-time.After(e.wait):
		return errors.New("unexpected completion")
	case <-ctx.Done():
		return ctx.Err()
	}
}

func newWorkerJobStore(t *testing.T) *orchestrator.Store {
	t.Helper()
	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })
	if _, err := db.Exec(`
		CREATE TABLE jobs (
			id           INTEGER PRIMARY KEY AUTOINCREMENT,
			server_id    INTEGER,
			kind         TEXT    NOT NULL,
			status       TEXT    NOT NULL,
			current_step TEXT    NOT NULL DEFAULT '',
			retry_count  INTEGER NOT NULL DEFAULT 0,
			last_error   TEXT,
			payload      TEXT,
			started_at   TEXT,
			finished_at  TEXT,
			timeout_at   TEXT,
			command_id   TEXT,
			created_at   TEXT    NOT NULL,
			updated_at   TEXT    NOT NULL
		);
		CREATE TABLE job_events (
			id         INTEGER PRIMARY KEY AUTOINCREMENT,
			job_id     INTEGER NOT NULL,
			seq        INTEGER NOT NULL,
			event_type TEXT    NOT NULL,
			level      TEXT    NOT NULL,
			step_key   TEXT,
			status     TEXT,
			message    TEXT    NOT NULL,
			payload    TEXT,
			created_at TEXT    NOT NULL
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return orchestrator.NewStore(db)
}

func testWorkerLogger() *slog.Logger {
	return slog.New(slog.NewTextHandler(io.Discard, nil))
}
