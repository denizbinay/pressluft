package dispatch

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"

	"github.com/google/uuid"
	"pressluft/internal/agentcommand"
	"pressluft/internal/observability"
	"pressluft/internal/orchestrator"
	"pressluft/internal/ws"
)

type JobStore interface {
	SetCommandID(ctx context.Context, jobID int64, commandID string) error
	TransitionJob(ctx context.Context, id int64, in orchestrator.TransitionInput) (orchestrator.Job, error)
	GetJob(ctx context.Context, id int64) (orchestrator.Job, error)
	GetJobByCommandID(ctx context.Context, commandID string) (*orchestrator.Job, error)
	AppendEvent(ctx context.Context, jobID int64, in orchestrator.CreateEventInput) (orchestrator.JobEvent, error)
}

type AgentRunner struct {
	hub    *ws.Hub
	store  JobStore
	logger *slog.Logger
}

func NewAgentRunner(hub *ws.Hub, store JobStore, logger *slog.Logger) *AgentRunner {
	if logger == nil {
		logger = slog.Default()
	}
	return &AgentRunner{hub: hub, store: store, logger: logger}
}

func (r *AgentRunner) Run(ctx context.Context, job orchestrator.Job) error {
	commandID := uuid.New().String()
	corr := observability.Correlation{JobID: job.ID, ServerID: job.ServerID, CommandID: commandID}
	r.logger.Info("command dispatch started", corr.LogArgs("command_type", job.Kind)...)

	if err := r.store.SetCommandID(ctx, job.ID, commandID); err != nil {
		r.logger.Error("command dispatch command id persistence failed", corr.LogArgs("error", err)...)
		return err
	}

	transitionInput := orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: "",
	}

	if _, err := r.store.TransitionJob(ctx, job.ID, transitionInput); err != nil {
		return err
	}

	r.appendEvent(ctx, job.ID, orchestrator.CreateEventInput{
		EventType: orchestrator.JobEventTypeStepStarted,
		Level:     "info",
		StepKey:   "dispatch",
		Status:    string(orchestrator.JobStatusRunning),
		Message:   "Dispatching command to agent",
		Payload:   corr.Payload(map[string]any{"phase": "dispatch_started", "command_type": job.Kind}),
	})

	cmd := ws.Command{
		ID:       commandID,
		JobID:    job.ID,
		ServerID: job.ServerID,
		Type:     job.Kind,
		Payload:  []byte(job.Payload),
	}

	normalizedPayload, err := agentcommand.Validate(cmd.Type, cmd.Payload)
	if err != nil {
		transitionInput = orchestrator.TransitionInput{
			ToStatus:  orchestrator.JobStatusFailed,
			LastError: err.Error(),
		}
		_, _ = r.store.TransitionJob(ctx, job.ID, transitionInput)
		r.appendEvent(ctx, job.ID, orchestrator.CreateEventInput{
			EventType: orchestrator.JobEventTypeFailed,
			Level:     "error",
			StepKey:   "dispatch",
			Status:    string(orchestrator.JobStatusFailed),
			Message:   "Command validation failed before dispatch",
			Payload:   corr.Payload(map[string]any{"phase": "validation_failed", "error": err.Error()}),
		})
		r.logger.Warn("command dispatch validation failed", corr.LogArgs("error", err)...)
		return nil
	}
	cmd.Payload = normalizedPayload

	conn, ok := r.hub.Get(job.ServerID)
	if !ok {
		transitionInput = orchestrator.TransitionInput{
			ToStatus:  orchestrator.JobStatusFailed,
			LastError: "agent not connected",
		}
		_, _ = r.store.TransitionJob(ctx, job.ID, transitionInput)
		r.appendEvent(ctx, job.ID, orchestrator.CreateEventInput{
			EventType: orchestrator.JobEventTypeFailed,
			Level:     "error",
			StepKey:   "dispatch",
			Status:    string(orchestrator.JobStatusFailed),
			Message:   "Agent not connected",
			Payload:   corr.Payload(map[string]any{"phase": "dispatch_blocked", "error": "agent not connected"}),
		})
		r.logger.Warn("command dispatch blocked", corr.LogArgs("error", "agent not connected")...)
		return nil
	}

	env := ws.Envelope{
		Type:    ws.TypeCommand,
		Payload: normalizedPayload,
	}
	cmdPayload, err := json.Marshal(cmd)
	if err != nil {
		transitionInput = orchestrator.TransitionInput{
			ToStatus:  orchestrator.JobStatusFailed,
			LastError: "failed to serialize command",
		}
		_, _ = r.store.TransitionJob(ctx, job.ID, transitionInput)
		r.appendEvent(ctx, job.ID, orchestrator.CreateEventInput{
			EventType: orchestrator.JobEventTypeFailed,
			Level:     "error",
			StepKey:   "dispatch",
			Status:    string(orchestrator.JobStatusFailed),
			Message:   "Failed to serialize command for agent dispatch",
			Payload:   corr.Payload(map[string]any{"phase": "serialization_failed"}),
		})
		r.logger.Error("command dispatch serialization failed", corr.LogArgs("error", err)...)
		return nil
	}
	env.Payload = cmdPayload

	if err := conn.Send(ctx, env); err != nil {
		lastError := "failed to send command"
		if errors.Is(err, context.DeadlineExceeded) {
			lastError = "command dispatch timed out"
		}
		transitionInput = orchestrator.TransitionInput{
			ToStatus:  orchestrator.JobStatusFailed,
			LastError: lastError,
		}
		_, _ = r.store.TransitionJob(ctx, job.ID, transitionInput)
		r.appendEvent(ctx, job.ID, orchestrator.CreateEventInput{
			EventType: orchestrator.JobEventTypeFailed,
			Level:     "error",
			StepKey:   "dispatch",
			Status:    string(orchestrator.JobStatusFailed),
			Message:   "Failed to send command to agent",
			Payload:   corr.Payload(map[string]any{"phase": "send_failed", "error": lastError}),
		})
		r.logger.Error("command dispatch failed", corr.LogArgs("error", err)...)
		return nil
	}

	r.appendEvent(ctx, job.ID, orchestrator.CreateEventInput{
		EventType: orchestrator.JobEventTypeStepComplete,
		Level:     "info",
		StepKey:   "dispatch",
		Status:    "sent",
		Message:   "Command sent to agent",
		Payload:   corr.Payload(map[string]any{"phase": "sent", "command_type": job.Kind}),
	})
	r.logger.Info("command dispatch completed", corr.LogArgs("command_type", job.Kind, "dispatch_status", "sent")...)

	return nil
}

func (r *AgentRunner) appendEvent(ctx context.Context, jobID int64, input orchestrator.CreateEventInput) {
	if r.store == nil {
		return
	}
	event, err := r.store.AppendEvent(ctx, jobID, input)
	if err != nil {
		r.logger.Error("job event append failed", "job_id", jobID, "event_type", input.EventType, "error", err)
		return
	}
	r.logger.Debug("job event appended", "job_id", jobID, "event_seq", event.Seq, "event_type", event.EventType)
}
