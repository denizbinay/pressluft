package agent

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"pressluft/internal/agentcommand"
	"pressluft/internal/ws"
)

func TestExecutorRejectsDisallowedRestartService(t *testing.T) {
	executor := NewExecutor()
	payload, err := json.Marshal(agentcommand.RestartServiceParams{ServiceName: "sshd"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	result := executor.Execute(context.Background(), ws.Command{ID: "cmd-1", Type: agentcommand.TypeRestartService, Payload: payload})
	if result.Success {
		t.Fatal("expected failure for disallowed service")
	}
	if result.ErrorCode != agentcommand.ErrorCodeServiceNotAllowed {
		t.Fatalf("error code = %q, want %q", result.ErrorCode, agentcommand.ErrorCodeServiceNotAllowed)
	}
}

func TestExecutorAppliesCommandTimeout(t *testing.T) {
	deadlineSeen := make(chan time.Time, 1)
	executor := &Executor{
		restartService: func(ctx context.Context, cmd ws.Command) ws.CommandResult {
			deadline, ok := ctx.Deadline()
			if !ok {
				return ws.FailureResult(cmd.ID, "missing_deadline", "missing deadline", nil, "")
			}
			deadlineSeen <- deadline
			return ws.SuccessResult(cmd.ID, nil, "")
		},
		listServices: func(ctx context.Context, cmd ws.Command) ws.CommandResult {
			return ws.SuccessResult(cmd.ID, nil, "")
		},
		siteHealth: func(ctx context.Context, cmd ws.Command) ws.CommandResult {
			return ws.SuccessResult(cmd.ID, nil, "")
		},
	}
	payload, err := json.Marshal(agentcommand.RestartServiceParams{ServiceName: "nginx"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	started := time.Now()
	result := executor.Execute(context.Background(), ws.Command{ID: "cmd-2", Type: agentcommand.TypeRestartService, Payload: payload})
	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}

	select {
	case deadline := <-deadlineSeen:
		remaining := time.Until(deadline)
		if remaining <= time.Minute || deadline.Before(started.Add(90*time.Second)) {
			t.Fatalf("deadline = %v, want command timeout near 2 minutes", deadline)
		}
	case <-time.After(time.Second):
		t.Fatal("expected restart service function to observe deadline")
	}
}

func TestExecutorDispatchesSiteHealthCommand(t *testing.T) {
	called := false
	executor := &Executor{
		restartService: func(ctx context.Context, cmd ws.Command) ws.CommandResult {
			return ws.SuccessResult(cmd.ID, nil, "")
		},
		listServices: func(ctx context.Context, cmd ws.Command) ws.CommandResult {
			return ws.SuccessResult(cmd.ID, nil, "")
		},
		siteHealth: func(ctx context.Context, cmd ws.Command) ws.CommandResult {
			called = true
			return ws.SuccessResult(cmd.ID, agentcommand.SiteHealthSnapshot{SiteID: "site-1", Healthy: true}, "")
		},
	}
	payload, err := json.Marshal(agentcommand.SiteHealthSnapshotParams{SiteID: "site-1", Hostname: "example.testable.io", SitePath: "/srv/www/site"})
	if err != nil {
		t.Fatalf("marshal payload: %v", err)
	}

	result := executor.Execute(context.Background(), ws.Command{ID: "cmd-site-health", Type: agentcommand.TypeSiteHealth, Payload: payload})
	if !called {
		t.Fatal("expected site health handler to be called")
	}
	if !result.Success {
		t.Fatalf("expected success, got %+v", result)
	}
}
