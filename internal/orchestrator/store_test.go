package orchestrator

import (
	"context"
	"database/sql"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestClaimNextJobMarksRunningWithTimeout(t *testing.T) {
	store := newTestStore(t)
	created, err := store.CreateJob(context.Background(), CreateJobInput{Kind: string(JobKindRestartService), ServerID: 7})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	claimed, err := store.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if claimed == nil {
		t.Fatal("expected claimed job")
	}
	if claimed.ID != created.ID {
		t.Fatalf("claimed id = %d, want %d", claimed.ID, created.ID)
	}
	if claimed.Status != JobStatusRunning {
		t.Fatalf("status = %q, want %q", claimed.Status, JobStatusRunning)
	}
	if claimed.StartedAt == "" || claimed.TimeoutAt == "" {
		t.Fatalf("expected started_at and timeout_at, got %+v", claimed)
	}
	deadline, err := time.Parse(time.RFC3339, claimed.TimeoutAt)
	if err != nil {
		t.Fatalf("parse timeout_at: %v", err)
	}
	if time.Until(deadline) <= 0 {
		t.Fatal("expected timeout_at in the future")
	}
}

func TestRecoverStuckJobsFailsRunningJobAndAppendsRecoveryEvent(t *testing.T) {
	store := newTestStore(t)
	job := mustCreateAndClaimJob(t, store, CreateJobInput{Kind: string(JobKindDeleteServer), ServerID: 9})

	recovered, err := store.RecoverStuckJobs(context.Background())
	if err != nil {
		t.Fatalf("recover stuck jobs: %v", err)
	}
	if recovered != 1 {
		t.Fatalf("recovered = %d, want 1", recovered)
	}

	updated, err := store.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if updated.Status != JobStatusFailed {
		t.Fatalf("status = %q, want %q", updated.Status, JobStatusFailed)
	}
	if updated.LastError == "" || updated.FinishedAt == "" {
		t.Fatalf("expected last_error and finished_at, got %+v", updated)
	}

	events, err := store.ListAllEvents(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	if events[0].EventType != JobEventTypeRecovered {
		t.Fatalf("event_type = %q, want %q", events[0].EventType, JobEventTypeRecovered)
	}
}

func TestMarkJobTimedOutAppendsTimeoutEvent(t *testing.T) {
	store := newTestStore(t)
	job := mustCreateAndClaimJob(t, store, CreateJobInput{Kind: string(JobKindRestartService), ServerID: 11})

	updated, changed, err := store.MarkJobTimedOut(context.Background(), job.ID, "job timed out before completion")
	if err != nil {
		t.Fatalf("mark job timed out: %v", err)
	}
	if !changed {
		t.Fatal("expected timeout change")
	}
	if updated.Status != JobStatusFailed {
		t.Fatalf("status = %q, want %q", updated.Status, JobStatusFailed)
	}

	events, err := store.ListAllEvents(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 {
		t.Fatalf("event count = %d, want 1", len(events))
	}
	if events[0].Seq != 1 || events[0].EventType != JobEventTypeTimedOut {
		t.Fatalf("unexpected event: %+v", events[0])
	}
}

func newTestStore(t *testing.T) *Store {
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
	return NewStore(db)
}

func mustCreateAndClaimJob(t *testing.T, store *Store, in CreateJobInput) Job {
	t.Helper()
	if _, err := store.CreateJob(context.Background(), in); err != nil {
		t.Fatalf("create job: %v", err)
	}
	job, err := store.ClaimNextJob(context.Background())
	if err != nil {
		t.Fatalf("claim job: %v", err)
	}
	if job == nil {
		t.Fatal("expected claimed job")
	}
	return *job
}
