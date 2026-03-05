package dispatch

import (
	"context"
	"fmt"
	"log/slog"

	"pressluft/internal/orchestrator"
	"pressluft/internal/ws"
)

type Completer struct {
	store    JobStore
	activity ActivityLogger
	logger   *slog.Logger
}

func NewCompleter(store JobStore, activity ActivityLogger, logger *slog.Logger) *Completer {
	return &Completer{
		store:    store,
		activity: activity,
		logger:   logger,
	}
}

func (c *Completer) HandleResult(result ws.CommandResult) error {
	ctx := context.Background()

	job, err := c.store.GetJobByCommandID(ctx, result.CommandID)
	if err != nil {
		return fmt.Errorf("job not found for command %s: %w", result.CommandID, err)
	}

	transitionInput := orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusSucceeded,
		CurrentStep: "",
		LastError:   "",
	}

	if result.Success {
		transitionInput.ToStatus = orchestrator.JobStatusSucceeded
	} else {
		transitionInput.ToStatus = orchestrator.JobStatusFailed
		transitionInput.LastError = result.Error
	}

	_, err = c.store.TransitionJob(ctx, job.ID, transitionInput)
	if err != nil {
		return err
	}

	message := "Command completed successfully"
	level := "info"
	status := "succeeded"
	if !result.Success {
		level = "error"
		status = "failed"
		if result.Error != "" {
			message = result.Error
		} else {
			message = "Command failed"
		}
	}

	_, _ = c.store.AppendEvent(ctx, job.ID, orchestrator.CreateEventInput{
		EventType: "step_update",
		Level:     level,
		StepKey:   "command",
		Status:    status,
		Message:   message,
	})

	return nil
}

func (c *Completer) HandleLogEntry(entry ws.LogEntry) error {
	ctx := context.Background()

	job, err := c.store.GetJobByCommandID(ctx, entry.CommandID)
	if err != nil {
		return err
	}

	return c.activity.Log(ctx, job.ServerID, "agent_log", entry.Message)
}
