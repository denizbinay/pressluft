package orchestrator

import (
	"bytes"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/agentcommand"
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

type JobPayloadValidator func(json.RawMessage, int64) (string, error)

var supportedJobKinds = []JobKindSpec{
	{Kind: JobKindProvisionServer, Label: "Server infrastructure provisioning", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: false}, Timeout: 30 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; inspect provider state before retrying manually", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "provision", Label: "Provisioning infrastructure"}}, ValidatePayload: validateProvisionServerPayload},
	{Kind: JobKindConfigureServer, Label: "Server setup", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: false}, Timeout: 30 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; retry setup manually after inspection", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "configure", Label: "Configuring server"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateConfigureServerPayload},
	{Kind: JobKindDeleteServer, Label: "Server deletion", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Destructive: true, Experimental: true, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 20 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; verify provider-side deletion before retrying manually", QueuedStatus: platform.ServerStatusDeleting, Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "delete", Label: "Deleting server"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateDeleteServerPayload},
	{Kind: JobKindRebuildServer, Label: "Server rebuild", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Destructive: true, Experimental: true, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 45 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; inspect machine state before retrying manually", QueuedStatus: platform.ServerStatusRebuilding, Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "rebuild", Label: "Rebuilding server"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateRebuildServerPayload},
	{Kind: JobKindResizeServer, Label: "Server resize", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Destructive: true, Experimental: true, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 20 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; inspect provider-side resize state before retrying manually", QueuedStatus: platform.ServerStatusResizing, Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "resize", Label: "Resizing server"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateResizeServerPayload},
	{Kind: JobKindUpdateFirewalls, Label: "Firewall update", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Experimental: true, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 15 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; retry manually after inspection", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "update_firewalls", Label: "Updating firewalls"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateUpdateFirewallsPayload},
	{Kind: JobKindManageVolume, Label: "Volume management", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Experimental: true, ExecutionPath: "worker", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 20 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption; retry manually after inspection", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "manage_volume", Label: "Managing volume"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateManageVolumePayload},
	{Kind: JobKindRestartService, Label: "Service restart", AllowedStatuses: []JobStatus{JobStatusQueued, JobStatusRunning, JobStatusSucceeded, JobStatusFailed}, Experimental: true, ExecutionPath: "agent", DispatchPolicy: DispatchPolicy{QueueServer: true}, Timeout: 2 * time.Minute, RetryLimit: 0, Recovery: "mark failed on worker interruption or timeout; late agent results are ignored", Steps: []WorkflowStep{{Key: "validate", Label: "Validating request"}, {Key: "restart_service", Label: "Restarting service"}, {Key: "finalize", Label: "Finalizing"}}, ValidatePayload: validateRestartServicePayload},
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

func ValidatePayload(kind string, payload json.RawMessage, serverID int64) (string, error) {
	spec, ok := JobKindPolicy(kind)
	if !ok {
		return "", fmt.Errorf("unsupported job kind: %s", kind)
	}
	if spec.ValidatePayload == nil {
		return normalizeArbitraryPayload(payload), nil
	}
	return spec.ValidatePayload(payload, serverID)
}

type ConfigureServerPayload struct {
	IPv4 string `json:"ipv4,omitempty"`
}

type DeleteServerPayload struct{}

type RebuildServerPayload struct {
	ServerImage string `json:"server_image,omitempty"`
}

type ResizeServerPayload struct {
	ServerType  string `json:"server_type"`
	UpgradeDisk bool   `json:"upgrade_disk"`
}

type UpdateFirewallsPayload struct {
	Firewalls []string `json:"firewalls"`
}

type ManageVolumePayload struct {
	VolumeName string `json:"volume_name"`
	SizeGB     int    `json:"size_gb,omitempty"`
	Location   string `json:"location,omitempty"`
	State      string `json:"state"`
	Automount  *bool  `json:"automount,omitempty"`
}

func MarshalConfigureServerPayload(in ConfigureServerPayload) (string, error) {
	return marshalNormalizedPayload(in)
}

func UnmarshalConfigureServerPayload(raw string) (ConfigureServerPayload, error) {
	var out ConfigureServerPayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return ConfigureServerPayload{}, err
	}
	out.IPv4 = strings.TrimSpace(out.IPv4)
	return out, nil
}

func MarshalDeleteServerPayload() (string, error) {
	return marshalNormalizedPayload(DeleteServerPayload{})
}

func UnmarshalDeleteServerPayload(raw string) (DeleteServerPayload, error) {
	var out DeleteServerPayload
	return out, unmarshalNormalizedPayload(raw, &out)
}

func MarshalRebuildServerPayload(in RebuildServerPayload) (string, error) {
	in.ServerImage = strings.TrimSpace(in.ServerImage)
	if in.ServerImage == "" {
		return marshalNormalizedPayload(struct{}{})
	}
	return marshalNormalizedPayload(in)
}

func UnmarshalRebuildServerPayload(raw string) (RebuildServerPayload, error) {
	var out RebuildServerPayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return RebuildServerPayload{}, err
	}
	out.ServerImage = strings.TrimSpace(out.ServerImage)
	return out, nil
}

func MarshalResizeServerPayload(in ResizeServerPayload) (string, error) {
	in.ServerType = strings.TrimSpace(in.ServerType)
	return marshalNormalizedPayload(in)
}

func UnmarshalResizeServerPayload(raw string) (ResizeServerPayload, error) {
	var out ResizeServerPayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return ResizeServerPayload{}, err
	}
	out.ServerType = strings.TrimSpace(out.ServerType)
	return out, nil
}

func MarshalUpdateFirewallsPayload(in UpdateFirewallsPayload) (string, error) {
	firewalls := make([]string, 0, len(in.Firewalls))
	for _, firewall := range in.Firewalls {
		firewall = strings.TrimSpace(firewall)
		if firewall != "" {
			firewalls = append(firewalls, firewall)
		}
	}
	in.Firewalls = firewalls
	return marshalNormalizedPayload(in)
}

func UnmarshalUpdateFirewallsPayload(raw string) (UpdateFirewallsPayload, error) {
	var out UpdateFirewallsPayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return UpdateFirewallsPayload{}, err
	}
	firewalls := make([]string, 0, len(out.Firewalls))
	for _, firewall := range out.Firewalls {
		firewall = strings.TrimSpace(firewall)
		if firewall != "" {
			firewalls = append(firewalls, firewall)
		}
	}
	out.Firewalls = firewalls
	return out, nil
}

func MarshalManageVolumePayload(in ManageVolumePayload) (string, error) {
	in.VolumeName = strings.TrimSpace(in.VolumeName)
	in.Location = strings.TrimSpace(in.Location)
	in.State = strings.TrimSpace(in.State)
	return marshalNormalizedPayload(in)
}

func UnmarshalManageVolumePayload(raw string) (ManageVolumePayload, error) {
	var out ManageVolumePayload
	if err := unmarshalNormalizedPayload(raw, &out); err != nil {
		return ManageVolumePayload{}, err
	}
	out.VolumeName = strings.TrimSpace(out.VolumeName)
	out.Location = strings.TrimSpace(out.Location)
	out.State = strings.TrimSpace(out.State)
	return out, nil
}

func marshalNormalizedPayload(value any) (string, error) {
	data, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("marshal normalized payload: %w", err)
	}
	if string(data) == "null" {
		return "", nil
	}
	return string(data), nil
}

func unmarshalNormalizedPayload(raw string, target any) error {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "{}"
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("invalid normalized job payload: %w", err)
	}
	return nil
}

func normalizeArbitraryPayload(payload json.RawMessage) string {
	trimmed := strings.TrimSpace(string(bytes.TrimSpace(payload)))
	if trimmed == "" || trimmed == "null" {
		return ""
	}
	return trimmed
}

func requireServerID(serverID int64, kind JobKind) error {
	if serverID <= 0 {
		return fmt.Errorf("server_id is required for %s job", kind)
	}
	return nil
}

func validateProvisionServerPayload(payload json.RawMessage, _ int64) (string, error) {
	return normalizeArbitraryPayload(payload), nil
}

func validateConfigureServerPayload(payload json.RawMessage, serverID int64) (string, error) {
	if err := requireServerID(serverID, JobKindConfigureServer); err != nil {
		return "", err
	}
	if normalizeArbitraryPayload(payload) == "" {
		return MarshalConfigureServerPayload(ConfigureServerPayload{})
	}
	var parsed struct {
		IPv4 string `json:"ipv4"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(payload), &parsed); err != nil {
		return "", fmt.Errorf("invalid configure_server payload: %w", err)
	}
	return MarshalConfigureServerPayload(ConfigureServerPayload{IPv4: parsed.IPv4})
}

func validateDeleteServerPayload(payload json.RawMessage, serverID int64) (string, error) {
	if err := requireServerID(serverID, JobKindDeleteServer); err != nil {
		return "", err
	}
	if normalizeArbitraryPayload(payload) == "" {
		return "", nil
	}
	var parsed DeleteServerPayload
	if err := json.Unmarshal(bytes.TrimSpace(payload), &parsed); err != nil {
		return "", fmt.Errorf("invalid delete_server payload: %w", err)
	}
	return MarshalDeleteServerPayload()
}

func validateRebuildServerPayload(payload json.RawMessage, serverID int64) (string, error) {
	if err := requireServerID(serverID, JobKindRebuildServer); err != nil {
		return "", err
	}
	if normalizeArbitraryPayload(payload) == "" {
		return MarshalRebuildServerPayload(RebuildServerPayload{})
	}
	var parsed RebuildServerPayload
	if err := json.Unmarshal(bytes.TrimSpace(payload), &parsed); err != nil {
		return "", fmt.Errorf("invalid rebuild_server payload: %w", err)
	}
	return MarshalRebuildServerPayload(parsed)
}

func validateResizeServerPayload(payload json.RawMessage, serverID int64) (string, error) {
	if err := requireServerID(serverID, JobKindResizeServer); err != nil {
		return "", err
	}
	var parsed struct {
		ServerType  string `json:"server_type"`
		UpgradeDisk *bool  `json:"upgrade_disk"`
	}
	if err := json.Unmarshal(bytes.TrimSpace(defaultPayloadObject(payload)), &parsed); err != nil {
		return "", fmt.Errorf("invalid resize_server payload: %w", err)
	}
	if strings.TrimSpace(parsed.ServerType) == "" {
		return "", fmt.Errorf("server_type is required for resize_server job")
	}
	if parsed.UpgradeDisk == nil {
		return "", fmt.Errorf("upgrade_disk is required for resize_server job")
	}
	return MarshalResizeServerPayload(ResizeServerPayload{
		ServerType:  parsed.ServerType,
		UpgradeDisk: *parsed.UpgradeDisk,
	})
}

func validateUpdateFirewallsPayload(payload json.RawMessage, serverID int64) (string, error) {
	if err := requireServerID(serverID, JobKindUpdateFirewalls); err != nil {
		return "", err
	}
	var parsed UpdateFirewallsPayload
	if err := json.Unmarshal(bytes.TrimSpace(defaultPayloadObject(payload)), &parsed); err != nil {
		return "", fmt.Errorf("invalid update_firewalls payload: %w", err)
	}
	firewalls := make([]string, 0, len(parsed.Firewalls))
	for _, firewall := range parsed.Firewalls {
		firewall = strings.TrimSpace(firewall)
		if firewall != "" {
			firewalls = append(firewalls, firewall)
		}
	}
	if len(firewalls) == 0 {
		return "", fmt.Errorf("firewalls payload must contain at least one firewall")
	}
	return MarshalUpdateFirewallsPayload(UpdateFirewallsPayload{Firewalls: firewalls})
}

func validateManageVolumePayload(payload json.RawMessage, serverID int64) (string, error) {
	if err := requireServerID(serverID, JobKindManageVolume); err != nil {
		return "", err
	}
	var parsed ManageVolumePayload
	if err := json.Unmarshal(bytes.TrimSpace(defaultPayloadObject(payload)), &parsed); err != nil {
		return "", fmt.Errorf("invalid manage_volume payload: %w", err)
	}
	volumeName := strings.TrimSpace(parsed.VolumeName)
	state := strings.TrimSpace(parsed.State)
	if volumeName == "" {
		return "", fmt.Errorf("volume_name is required for manage_volume job")
	}
	if state != "present" && state != "absent" {
		return "", fmt.Errorf("state must be present or absent for manage_volume job")
	}
	if state == "present" {
		if parsed.Automount == nil {
			return "", fmt.Errorf("automount is required for manage_volume job when state=present")
		}
		if parsed.SizeGB <= 0 {
			return "", fmt.Errorf("size_gb is required for manage_volume job when state=present")
		}
	}
	return MarshalManageVolumePayload(parsed)
}

func validateRestartServicePayload(payload json.RawMessage, serverID int64) (string, error) {
	if err := requireServerID(serverID, JobKindRestartService); err != nil {
		return "", err
	}
	normalizedPayload, err := agentcommand.Validate(string(JobKindRestartService), bytes.TrimSpace(defaultPayloadObject(payload)))
	if err != nil {
		return "", fmt.Errorf("invalid restart_service payload: %w", err)
	}
	return string(normalizedPayload), nil
}

func defaultPayloadObject(payload json.RawMessage) []byte {
	if normalizeArbitraryPayload(payload) == "" {
		return []byte("{}")
	}
	return payload
}

// Job is the persisted orchestration unit.
type Job struct {
	ID          int64     `json:"id"`
	ServerID    int64     `json:"server_id,omitempty"`
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
	JobID      int64  `json:"job_id"`
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
	ServerID int64
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
