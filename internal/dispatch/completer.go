package dispatch

import (
	"context"
	"fmt"
	"log/slog"

	"pressluft/internal/activity"
	"pressluft/internal/observability"
	"pressluft/internal/orchestrator"
	"pressluft/internal/ws"
)

type Completer struct {
	store    JobStore
	activity *activity.Store
	logger   *slog.Logger
}

func NewCompleter(store JobStore, activityStore *activity.Store, logger *slog.Logger) *Completer {
	return &Completer{
		store:    store,
		activity: activityStore,
		logger:   logger,
	}
}

func (c *Completer) HandleResult(result ws.CommandResult) error {
	ctx := context.Background()

	job, err := c.store.GetJobByCommandID(ctx, result.CommandID)
	if err != nil {
		c.logger.Error("command result job lookup failed", observability.Correlation{JobID: result.JobID, ServerID: result.ServerID, CommandID: result.CommandID}.LogArgs("error", err)...)
		return fmt.Errorf("job not found for command %s: %w", result.CommandID, err)
	}
	if result.JobID == 0 {
		result.JobID = job.ID
	}
	if result.ServerID == 0 {
		result.ServerID = job.ServerID
	}
	corr := observability.Correlation{JobID: job.ID, ServerID: job.ServerID, CommandID: result.CommandID}
	if job.Status == orchestrator.JobStatusSucceeded || job.Status == orchestrator.JobStatusFailed {
		c.logger.Debug("stale command result ignored", corr.LogArgs("job_status", job.Status)...)
		return nil
	}

	transitionInput := orchestrator.TransitionInput{ToStatus: orchestrator.JobStatusSucceeded}
	message := "Command completed successfully"
	level := "info"
	eventType := orchestrator.JobEventTypeSucceeded
	status := string(orchestrator.JobStatusSucceeded)

	if !result.Success {
		transitionInput.ToStatus = orchestrator.JobStatusFailed
		transitionInput.LastError = result.Error
		level = "error"
		eventType = orchestrator.JobEventTypeFailed
		status = string(orchestrator.JobStatusFailed)
		if result.Error != "" {
			message = result.Error
		} else {
			message = "Command failed"
		}
	}

	updated, err := c.store.TransitionJob(ctx, job.ID, transitionInput)
	if err != nil {
		return err
	}

	if _, err := c.store.AppendEvent(ctx, job.ID, orchestrator.CreateEventInput{
		EventType: eventType,
		Level:     level,
		StepKey:   "command",
		Status:    status,
		Message:   message,
		Payload:   corr.Payload(map[string]any{"result": commandResultEventPayload(result), "success": result.Success, "error_code": result.ErrorCode}),
	}); err != nil {
		c.logger.Error("command result event append failed", corr.LogArgs("error", err)...)
	}

	c.logger.Info("command result recorded", corr.LogArgs("success", result.Success, "error_code", result.ErrorCode)...)
	c.emitTerminalActivity(ctx, updated, message, result.Success)
	return nil
}

func commandResultEventPayload(result ws.CommandResult) string {
	if len(result.Payload) > 0 {
		return string(result.Payload)
	}
	return result.Output
}

func (c *Completer) HandleLogEntry(entry ws.LogEntry) error {
	ctx := context.Background()

	job, err := c.store.GetJobByCommandID(ctx, entry.CommandID)
	if err != nil {
		c.logger.Error("command log job lookup failed", observability.Correlation{JobID: entry.JobID, ServerID: entry.ServerID, CommandID: entry.CommandID}.LogArgs("error", err)...)
		return err
	}
	if entry.JobID == 0 {
		entry.JobID = job.ID
	}
	if entry.ServerID == 0 {
		entry.ServerID = job.ServerID
	}
	corr := observability.Correlation{JobID: job.ID, ServerID: job.ServerID, CommandID: entry.CommandID}

	if _, err := c.store.AppendEvent(ctx, job.ID, orchestrator.CreateEventInput{
		EventType: orchestrator.JobEventTypeCommandLog,
		Level:     entry.Level,
		StepKey:   "command",
		Status:    string(job.Status),
		Message:   entry.Message,
		Payload:   corr.Payload(map[string]any{"timestamp": entry.Timestamp.UTC().Format("2006-01-02T15:04:05Z07:00")}),
	}); err != nil {
		c.logger.Error("command log event append failed", corr.LogArgs("error", err)...)
	}
	c.logger.Debug("command log recorded", corr.LogArgs("level", entry.Level, "message", entry.Message)...)

	return nil
}

func (c *Completer) emitTerminalActivity(ctx context.Context, job orchestrator.Job, message string, success bool) {
	if c.activity == nil {
		return
	}

	input := activity.EmitInput{
		Category:     activity.CategoryJob,
		ResourceType: activity.ResourceJob,
		ResourceID:   job.ID,
		ActorType:    activity.ActorSystem,
		Message:      message,
	}
	if job.ServerID > 0 {
		input.ParentResourceType = activity.ResourceServer
		input.ParentResourceID = job.ServerID
	}
	if success {
		input.EventType = activity.EventJobCompleted
		input.Level = activity.LevelSuccess
		input.Title = fmt.Sprintf("%s completed", orchestrator.JobKindLabel(job.Kind))
	} else {
		input.EventType = activity.EventJobFailed
		input.Level = activity.LevelError
		input.Title = fmt.Sprintf("%s failed", orchestrator.JobKindLabel(job.Kind))
		input.RequiresAttention = true
	}
	input.Payload = observability.Correlation{JobID: job.ID, ServerID: job.ServerID, CommandID: derefCommandID(job.CommandID)}.Payload(map[string]any{"success": success})
	activityEntry, err := c.activity.Emit(ctx, input)
	if err != nil {
		c.logger.Error("command completion activity emit failed", observability.Correlation{JobID: job.ID, ServerID: job.ServerID, CommandID: derefCommandID(job.CommandID)}.LogArgs("error", err)...)
		return
	}
	c.logger.Info("command completion activity emitted", observability.Correlation{JobID: job.ID, ServerID: job.ServerID, CommandID: derefCommandID(job.CommandID)}.LogArgs("activity_id", activityEntry.ID, "success", success)...)
}

func derefCommandID(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
