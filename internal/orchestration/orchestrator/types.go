package orchestrator

import (
	"encoding/json"
	"fmt"
	"time"

	"pressluft/internal/platform"
)

// JobStatus is the durable lifecycle state for orchestration jobs.
type JobStatus string

const (
	JobStatusQueued    JobStatus = "queued"
	JobStatusRunning   JobStatus = "running"
	JobStatusSucceeded JobStatus = "succeeded"
	JobStatusFailed    JobStatus = "failed"
)

const (
	JobEventTypeCreated      = "job_created"
	JobEventTypeStepStarted  = "step_started"
	JobEventTypeStepComplete = "step_completed"
	JobEventTypeCommandLog   = "command_log"
	JobEventTypeSucceeded    = "job_succeeded"
	JobEventTypeFailed       = "job_failed"
	JobEventTypeRecovered    = "job_recovered"
	JobEventTypeTimedOut     = "job_timed_out"
)

// JobKind is the canonical identifier for a supported orchestration workflow.
type JobKind string

const (
	JobKindProvisionServer JobKind = "provision_server"
	JobKindConfigureServer JobKind = "configure_server"
	JobKindDeleteServer    JobKind = "delete_server"
	JobKindRebuildServer   JobKind = "rebuild_server"
	JobKindResizeServer    JobKind = "resize_server"
	JobKindUpdateFirewalls JobKind = "update_firewalls"
	JobKindManageVolume    JobKind = "manage_volume"
	JobKindRestartService  JobKind = "restart_service"
	JobKindDeploySite      JobKind = "deploy_site"
)

type JobKindSpec struct {
	Kind            JobKind
	Label           string
	AllowedStatuses []JobStatus
	Destructive     bool
	Experimental    bool
	ExecutionPath   string
	DispatchPolicy  DispatchPolicy
	Timeout         time.Duration
	RetryLimit      int
	Recovery        string
	QueuedStatus    platform.ServerStatus
	Steps           []WorkflowStep
	ValidatePayload JobPayloadValidator
}

type DispatchPolicy struct {
	QueueServer bool
}

type WorkflowStep struct {
	Key   string
	Label string
}

type JobPayloadValidator func(json.RawMessage, string) (string, error)

var supportedJobKinds = []JobKindSpec{
	{Kind: JobKindProvisionServer, Label: "Server infrastructure provisioning", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: false}, Timeout: 30 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; inspect provider state before retrying manually", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "provision", Label: "Provisioning infrastructure"}}, ValidatePayload: validateProvisionServerPayload},
	{Kind: JobKindConfigureServer, Label: "Server setup", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: false}, Timeout: 30 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; retry setup manually after inspection", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "configure", Label: "Configuring server"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateConfigureServerPayload},
	{Kind: JobKindDeleteServer, Label: "Server deletion", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Destructive: true, Experimental: true, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 20 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; verify provider-side deletion before retrying manually", QueuedStatus: platform.ServerStatusDeleting, Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "delete", Label: "Deleting server"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateDeleteServerPayload},
	{Kind: JobKindRebuildServer, Label: "Server rebuild", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Destructive: true, Experimental: true, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 45 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; inspect machine state before retrying manually", QueuedStatus: platform.ServerStatusRebuilding, Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "rebuild", Label: "Rebuilding server"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateRebuildServerPayload},
	{Kind: JobKindResizeServer, Label: "Server resize", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Destructive: true, Experimental: true, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 20 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; inspect provider-side resize state before retrying manually", QueuedStatus: platform.ServerStatusResizing, Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "resize", Label: "Resizing server"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateResizeServerPayload},
	{Kind: JobKindUpdateFirewalls, Label: "Firewall update", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Experimental: true, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 15 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; retry manually after inspection", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "update_firewalls", Label: "Updating firewalls"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateUpdateFirewallsPayload},
	{Kind: JobKindManageVolume, Label: "Volume management", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Experimental: true, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 20 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; retry manually after inspection", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "manage_volume", Label: "Managing volume"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateManageVolumePayload},
	{Kind: JobKindRestartService, Label: "Service restart", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Experimental: true, ExecutionPath: "agent", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 2 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption or timeout; late agent results are ignored", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "restart_service", Label: "Restarting service"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateRestartServicePayload},
	{Kind: JobKindDeploySite, Label: "Site deployment", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: false}, Timeout: 25 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; inspect site files, database, and routing before retrying manually", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "deploy", Label: "Deploying site"}, {Key: "verify", Label: "Verifying site routing"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateDeploySitePayload},
}

// SupportedJobKinds returns the current canonical job-kind contract.
func SupportedJobKinds() []JobKindSpec {
	out := make([]JobKindSpec, len(supportedJobKinds))
	copy(out, supportedJobKinds)
	return out
}

// IsKnownJobKind reports whether kind is part of the current runtime contract.
func IsKnownJobKind(kind string) bool {
	_, ok := JobKindPolicy(kind)
	return ok
}

// JobKindLabel returns a human-readable label for a supported job kind.
func JobKindLabel(kind string) string {
	spec, ok := JobKindPolicy(kind)
	if !ok {
		return kind
	}
	return spec.Label
}

// AllowedStatusesForKind returns the lifecycle states currently used by the runtime for kind.
func AllowedStatusesForKind(kind string) []JobStatus {
	spec, ok := JobKindPolicy(kind)
	if !ok {
		return nil
	}
	out := make([]JobStatus, len(spec.AllowedStatuses))
	copy(out, spec.AllowedStatuses)
	return out
}

func JobKindPolicy(kind string) (JobKindSpec, bool) {
	for _, spec := range supportedJobKinds {
		if string(spec.Kind) == kind {
			return spec, true
		}
	}
	return JobKindSpec{}, false
}

func DispatchPolicyForKind(kind string) (DispatchPolicy, bool) {
	spec, ok := JobKindPolicy(kind)
	if !ok {
		return DispatchPolicy{}, false
	}
	return spec.DispatchPolicy, true
}

func WorkflowStepsForKind(kind string) []WorkflowStep {
	spec, ok := JobKindPolicy(kind)
	if !ok || len(spec.Steps) == 0 {
		return nil
	}
	out := make([]WorkflowStep, len(spec.Steps))
	copy(out, spec.Steps)
	return out
}

func QueuedServerStatusForKind(kind string) (platform.ServerStatus, bool) {
	spec, ok := JobKindPolicy(kind)
	if !ok || spec.QueuedStatus == "" {
		return "", false
	}
	return spec.QueuedStatus, true
}

func ValidatePayload(kind string, payload json.RawMessage, serverID string) (string, error) {
	spec, ok := JobKindPolicy(kind)
	if !ok {
		return "", fmt.Errorf("unsupported job kind: %s", kind)
	}
	if spec.ValidatePayload == nil {
		return normalizeArbitraryPayload(payload), nil
	}
	return spec.ValidatePayload(payload, serverID)
}

// Job is the persisted orchestration unit.
type Job struct {
	ID          string    `json:"id"`
	ServerID    string    `json:"server_id,omitempty"`
	Kind        string    `json:"kind"`
	Status      JobStatus `json:"status"`
	CurrentStep string    `json:"current_step"`
	RetryCount  int       `json:"retry_count"`
	LastError   string    `json:"last_error,omitempty"`
	Payload     string    `json:"payload,omitempty"`
	StartedAt   string    `json:"started_at,omitempty"`
	FinishedAt  string    `json:"finished_at,omitempty"`
	TimeoutAt   string    `json:"timeout_at,omitempty"`
	CreatedAt   string    `json:"created_at"`
	UpdatedAt   string    `json:"updated_at"`
	CommandID   *string   `json:"command_id,omitempty"`
}

// JobEvent is an ordered event entry consumed by the dashboard.
type JobEvent struct {
	ID         string `json:"id"`
	JobID      string `json:"job_id"`
	Seq        int64  `json:"seq"`
	EventType  string `json:"event_type"`
	Level      string `json:"level"`
	StepKey    string `json:"step_key,omitempty"`
	Status     string `json:"status,omitempty"`
	Message    string `json:"message"`
	Payload    string `json:"payload,omitempty"`
	OccurredAt string `json:"occurred_at"`
}

// CreateJobInput is the job creation payload.
type CreateJobInput struct {
	Kind     string
	ServerID string
	Payload  string
}

// TransitionInput updates a job lifecycle state.
type TransitionInput struct {
	ToStatus    JobStatus
	CurrentStep string
	LastError   string
	RetryCount  int
}

// CreateEventInput appends an event for a job timeline.
type CreateEventInput struct {
	EventType string
	Level     string
	StepKey   string
	Status    string
	Message   string
	Payload   string
}
