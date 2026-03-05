package dispatch

import (
	"context"
	"encoding/json"

	"github.com/google/uuid"
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

type ActivityLogger interface {
	Log(ctx context.Context, serverID int64, action string, details string) error
}

type AgentRunner struct {
	hub   *ws.Hub
	store JobStore
}

func NewAgentRunner(hub *ws.Hub, store JobStore) *AgentRunner {
	return &AgentRunner{hub: hub, store: store}
}

func (r *AgentRunner) Run(ctx context.Context, job orchestrator.Job) error {
	commandID := uuid.New().String()

	if err := r.store.SetCommandID(ctx, job.ID, commandID); err != nil {
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
		EventType: "step_update",
		Level:     "info",
		StepKey:   "dispatch",
		Status:    "running",
		Message:   "Dispatching command to agent",
	})

	cmd := ws.Command{
		ID:      commandID,
		JobID:   job.ID,
		Type:    job.Kind,
		Payload: []byte(job.Payload),
	}

	conn, ok := r.hub.Get(job.ServerID)
	if !ok {
		transitionInput = orchestrator.TransitionInput{
			ToStatus:  orchestrator.JobStatusFailed,
			LastError: "agent not connected",
		}
		_, _ = r.store.TransitionJob(ctx, job.ID, transitionInput)
		r.appendEvent(ctx, job.ID, orchestrator.CreateEventInput{
			EventType: "step_update",
			Level:     "error",
			StepKey:   "dispatch",
			Status:    "failed",
			Message:   "Agent not connected",
		})
		return nil
	}

	env := ws.Envelope{
		Type:    ws.TypeCommand,
		Payload: mustMarshal(cmd),
	}

	if err := conn.Send(ctx, env); err != nil {
		transitionInput = orchestrator.TransitionInput{
			ToStatus:  orchestrator.JobStatusFailed,
			LastError: "failed to send command",
		}
		_, _ = r.store.TransitionJob(ctx, job.ID, transitionInput)
		r.appendEvent(ctx, job.ID, orchestrator.CreateEventInput{
			EventType: "step_update",
			Level:     "error",
			StepKey:   "dispatch",
			Status:    "failed",
			Message:   "Failed to send command to agent",
		})
		return nil
	}

	r.appendEvent(ctx, job.ID, orchestrator.CreateEventInput{
		EventType: "step_update",
		Level:     "info",
		StepKey:   "dispatch",
		Status:    "running",
		Message:   "Command sent to agent",
	})

	return nil
}

func (r *AgentRunner) appendEvent(ctx context.Context, jobID int64, input orchestrator.CreateEventInput) {
	if r.store == nil {
		return
	}
	_, _ = r.store.AppendEvent(ctx, jobID, input)
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
