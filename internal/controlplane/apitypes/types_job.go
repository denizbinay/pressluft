package apitypes

import (
	"encoding/json"
	"fmt"
	"strings"

	"pressluft/internal/orchestration/orchestrator"
)

type CreateJobRequest struct {
	Kind     string          `json:"kind"`
	ServerID string          `json:"server_id,omitempty"`
	Payload  json.RawMessage `json:"payload"`
}

func (r *CreateJobRequest) Validate() error {
	r.Kind = strings.TrimSpace(r.Kind)
	if r.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	return nil
}

type Job struct {
	ID          string                 `json:"id"`
	ServerID    string                 `json:"server_id,omitempty"`
	Kind        string                 `json:"kind"`
	Status      orchestrator.JobStatus `json:"status"`
	CurrentStep string                 `json:"current_step"`
	RetryCount  int                    `json:"retry_count"`
	LastError   string                 `json:"last_error,omitempty"`
	Payload     string                 `json:"payload,omitempty"`
	StartedAt   string                 `json:"started_at,omitempty"`
	FinishedAt  string                 `json:"finished_at,omitempty"`
	TimeoutAt   string                 `json:"timeout_at,omitempty"`
	CreatedAt   string                 `json:"created_at"`
	UpdatedAt   string                 `json:"updated_at"`
	CommandID   *string                `json:"command_id,omitempty"`
}

func APIJob(in orchestrator.Job) Job {
	return Job{
		ID:          in.ID,
		ServerID:    FormatAppID(in.ServerID),
		Kind:        in.Kind,
		Status:      in.Status,
		CurrentStep: in.CurrentStep,
		RetryCount:  in.RetryCount,
		LastError:   in.LastError,
		Payload:     in.Payload,
		StartedAt:   in.StartedAt,
		FinishedAt:  in.FinishedAt,
		TimeoutAt:   in.TimeoutAt,
		CreatedAt:   in.CreatedAt,
		UpdatedAt:   in.UpdatedAt,
		CommandID:   in.CommandID,
	}
}

func APIJobs(in []orchestrator.Job) []Job {
	out := make([]Job, 0, len(in))
	for _, item := range in {
		out = append(out, APIJob(item))
	}
	return out
}
