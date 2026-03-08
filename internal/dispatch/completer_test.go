package dispatch

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"
	"time"

	"pressluft/internal/activity"
	"pressluft/internal/orchestrator"
	"pressluft/internal/ws"

	_ "modernc.org/sqlite"
)

func TestCompleterHandleResultMarksJobSucceededAndEmitsActivity(t *testing.T) {
	db := newCompleterDB(t)
	jobStore := orchestrator.NewStore(db)
	activityStore := activity.NewStore(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	job, err := jobStore.CreateJob(context.Background(), orchestrator.CreateJobInput{Kind: string(orchestrator.JobKindRestartService), ServerID: 3})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := jobStore.TransitionJob(context.Background(), job.ID, orchestrator.TransitionInput{ToStatus: orchestrator.JobStatusRunning}); err != nil {
		t.Fatalf("transition job: %v", err)
	}
	if err := jobStore.SetCommandID(context.Background(), job.ID, "cmd-1"); err != nil {
		t.Fatalf("set command id: %v", err)
	}

	completer := NewCompleter(jobStore, activityStore, logger)
	if err := completer.HandleResult(ws.CommandResult{CommandID: "cmd-1", Success: true, Output: "ok"}); err != nil {
		t.Fatalf("handle result: %v", err)
	}

	updated, err := jobStore.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if updated.Status != orchestrator.JobStatusSucceeded {
		t.Fatalf("status = %q, want %q", updated.Status, orchestrator.JobStatusSucceeded)
	}

	events, err := jobStore.ListAllEvents(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].EventType != orchestrator.JobEventTypeSucceeded {
		t.Fatalf("unexpected events: %+v", events)
	}
	if events[0].Payload == "" {
		t.Fatal("expected correlated job event payload")
	}

	activities, _, err := activityStore.List(context.Background(), activity.ListFilter{Limit: 10})
	if err != nil {
		t.Fatalf("list activity: %v", err)
	}
	if len(activities) != 1 || activities[0].EventType != activity.EventJobCompleted {
		t.Fatalf("unexpected activity: %+v", activities)
	}
	if activities[0].Payload == "" {
		t.Fatal("expected correlated activity payload")
	}
}

func TestCompleterHandleLogEntryAppendsTimelineEvent(t *testing.T) {
	db := newCompleterDB(t)
	jobStore := orchestrator.NewStore(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	job, err := jobStore.CreateJob(context.Background(), orchestrator.CreateJobInput{Kind: string(orchestrator.JobKindRestartService), ServerID: 5})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := jobStore.TransitionJob(context.Background(), job.ID, orchestrator.TransitionInput{ToStatus: orchestrator.JobStatusRunning}); err != nil {
		t.Fatalf("transition job: %v", err)
	}
	if err := jobStore.SetCommandID(context.Background(), job.ID, "cmd-2"); err != nil {
		t.Fatalf("set command id: %v", err)
	}

	completer := NewCompleter(jobStore, nil, logger)
	if err := completer.HandleLogEntry(ws.LogEntry{CommandID: "cmd-2", Level: "info", Message: "restarting service", Timestamp: time.Now()}); err != nil {
		t.Fatalf("handle log entry: %v", err)
	}

	events, err := jobStore.ListAllEvents(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 1 || events[0].EventType != orchestrator.JobEventTypeCommandLog {
		t.Fatalf("unexpected events: %+v", events)
	}
	if events[0].Payload == "" {
		t.Fatal("expected correlated log event payload")
	}
}

func newCompleterDB(t *testing.T) *sql.DB {
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
		CREATE TABLE activity (
			id                 INTEGER PRIMARY KEY AUTOINCREMENT,
			event_type         TEXT NOT NULL,
			category           TEXT NOT NULL,
			level              TEXT NOT NULL,
			resource_type      TEXT,
			resource_id        INTEGER,
			parent_resource_type TEXT,
			parent_resource_id INTEGER,
			actor_type         TEXT NOT NULL,
			actor_id           TEXT,
			title              TEXT NOT NULL,
			message            TEXT,
			payload            TEXT,
			requires_attention INTEGER NOT NULL DEFAULT 0,
			read_at            TEXT,
			created_at         TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}
	return db
}
