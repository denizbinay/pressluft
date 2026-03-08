package dispatch

import (
	"context"
	"database/sql"
	"io"
	"log/slog"
	"testing"

	"pressluft/internal/orchestrator"
	"pressluft/internal/ws"

	_ "modernc.org/sqlite"
)

func TestAgentRunnerFailsInvalidPayloadBeforeDispatch(t *testing.T) {
	db := newCompleterDB(t)
	jobStore := orchestrator.NewStore(db)
	runner := NewAgentRunner(ws.NewHub(), jobStore, nil)

	job, err := jobStore.CreateJob(context.Background(), orchestrator.CreateJobInput{
		Kind:     string(orchestrator.JobKindRestartService),
		ServerID: 7,
		Payload:  `{"service_name":"../../etc/passwd"}`,
	})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}

	if err := runner.Run(context.Background(), job); err != nil {
		t.Fatalf("run: %v", err)
	}

	updated, err := jobStore.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if updated.Status != orchestrator.JobStatusFailed {
		t.Fatalf("status = %q, want %q", updated.Status, orchestrator.JobStatusFailed)
	}
	if updated.LastError == "" {
		t.Fatal("expected validation error to be persisted")
	}
	events, err := jobStore.ListAllEvents(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) == 0 || events[len(events)-1].Payload == "" {
		t.Fatalf("expected correlated event payload, got %+v", events)
	}
}

func TestCompleterIgnoresLateResultForTerminalJob(t *testing.T) {
	db := newCompleterDB(t)
	jobStore := orchestrator.NewStore(db)
	logger := slog.New(slog.NewTextHandler(io.Discard, nil))

	job, err := jobStore.CreateJob(context.Background(), orchestrator.CreateJobInput{Kind: string(orchestrator.JobKindRestartService), ServerID: 5})
	if err != nil {
		t.Fatalf("create job: %v", err)
	}
	if _, err := jobStore.TransitionJob(context.Background(), job.ID, orchestrator.TransitionInput{ToStatus: orchestrator.JobStatusRunning}); err != nil {
		t.Fatalf("transition running: %v", err)
	}
	if err := jobStore.SetCommandID(context.Background(), job.ID, "cmd-late"); err != nil {
		t.Fatalf("set command id: %v", err)
	}
	if _, err := jobStore.TransitionJob(context.Background(), job.ID, orchestrator.TransitionInput{ToStatus: orchestrator.JobStatusFailed, LastError: "job timed out before completion"}); err != nil {
		t.Fatalf("transition failed: %v", err)
	}

	completer := NewCompleter(jobStore, nil, logger)
	if err := completer.HandleResult(ws.CommandResult{CommandID: "cmd-late", Success: true, Output: "late"}); err != nil {
		t.Fatalf("handle late result: %v", err)
	}

	updated, err := jobStore.GetJob(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("get job: %v", err)
	}
	if updated.Status != orchestrator.JobStatusFailed {
		t.Fatalf("status = %q, want %q", updated.Status, orchestrator.JobStatusFailed)
	}

	events, err := jobStore.ListAllEvents(context.Background(), job.ID)
	if err != nil {
		t.Fatalf("list events: %v", err)
	}
	if len(events) != 0 {
		t.Fatalf("events = %+v, want none for ignored late result", events)
	}
}

func newAgentRunnerDB(t *testing.T) *sql.DB {
	t.Helper()
	return newCompleterDB(t)
}
