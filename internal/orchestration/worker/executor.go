package worker

import (
	"context"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strings"
	"time"

	"pressluft/internal/controlplane/activity"
	serverpkg "pressluft/internal/controlplane/server"
	"pressluft/internal/infra/provider"
	"pressluft/internal/infra/runner"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/shared/observability"
)

// ServerStore defines the server persistence interface needed by the executor.
type ServerStore interface {
	GetByID(ctx context.Context, id string) (*serverpkg.StoredServer, error)
	UpdateStatus(ctx context.Context, id string, status platform.ServerStatus) error
	UpdateSetupState(ctx context.Context, id string, setupState platform.SetupState, setupLastError string) error
	UpdateProvisioning(ctx context.Context, id string, providerServerID, actionID, actionStatus string, status platform.ServerStatus, ipv4, ipv6 string) error
	UpdateServerType(ctx context.Context, id string, serverType string) error
	UpdateImage(ctx context.Context, id string, image string) error
	GetKey(ctx context.Context, serverID string) (*serverpkg.StoredServerKey, error)
	CreateKey(ctx context.Context, in serverpkg.CreateServerKeyInput) error
}

// ProviderStore defines the provider persistence interface needed by the executor.
type ProviderStore interface {
	GetByID(ctx context.Context, id string) (*provider.StoredProvider, error)
}

type SiteStore interface {
	GetByID(ctx context.Context, id string) (*serverpkg.StoredSite, error)
	UpdateDeployment(ctx context.Context, siteID, deploymentState, deploymentStatus, lastDeployJobID, lastDeployedAt string) error
	UpdateRuntimeHealth(ctx context.Context, siteID, runtimeHealthState, runtimeHealthStatus, lastHealthCheckAt string) error
}

type DomainStore interface {
	ListBySite(ctx context.Context, siteID string) ([]serverpkg.StoredDomain, error)
	UpdateRoutingStatus(ctx context.Context, domainID, routingState, routingStatusMessage string, checkedAt time.Time) error
}

// Executor runs job steps and emits events.
type Executor struct {
	jobStore          *orchestrator.Store
	serverStore       ServerStore
	providerStore     ProviderStore
	siteStore         SiteStore
	domainStore       DomainStore
	activityStore     *activity.Store
	runner            runner.Runner
	agentRunner       AgentJobRunner
	devTokenStore     DevTokenStore
	registrationStore RegistrationTokenStore
	executionMode     platform.ExecutionMode
	playbookBasePath  string
	configurePath     string
	controlPlaneURL   string
	logger            *slog.Logger
}

// Conventional playbook filenames inside ops/ansible/playbooks/<provider-type>/.
// Each provider supplies its own set of playbooks following this naming
// convention, making it obvious where to add playbooks for a new provider.
const (
	playbookProvision  = "provision.yml"
	playbookDelete     = "delete.yml"
	playbookRebuild    = "rebuild.yml"
	playbookResize     = "resize.yml"
	playbookFirewalls  = "firewalls.yml"
	playbookVolume     = "volume.yml"
	playbookSiteDeploy = "deploy-site.yml"
)

// ExecutorConfig defines runner configuration.
type ExecutorConfig struct {
	// PlaybookBasePath is the root directory containing per-provider playbook
	// subdirectories. The executor resolves provider-specific playbooks as
	// <PlaybookBasePath>/<provider-type>/<action>.yml.
	PlaybookBasePath      string
	ConfigurePlaybookPath string
	ControlPlaneURL       string
	ExecutionMode         platform.ExecutionMode
	DevTokenStore         DevTokenStore
	RegistrationStore     RegistrationTokenStore
	AgentRunner           AgentJobRunner
}

type DevTokenStore interface {
	Create(serverID string, expiresIn time.Duration) (string, error)
}

type RegistrationTokenStore interface {
	Create(serverID string, expiresIn time.Duration) (string, error)
}

type AgentJobRunner interface {
	Run(ctx context.Context, job orchestrator.Job) error
}

// NewExecutor creates an executor with the given dependencies.
func NewExecutor(
	jobStore *orchestrator.Store,
	serverStore ServerStore,
	providerStore ProviderStore,
	siteStore SiteStore,
	domainStore DomainStore,
	activityStore *activity.Store,
	runner runner.Runner,
	config ExecutorConfig,
	logger *slog.Logger,
) *Executor {
	return &Executor{
		jobStore:          jobStore,
		serverStore:       serverStore,
		providerStore:     providerStore,
		siteStore:         siteStore,
		domainStore:       domainStore,
		activityStore:     activityStore,
		runner:            runner,
		agentRunner:       config.AgentRunner,
		devTokenStore:     config.DevTokenStore,
		registrationStore: config.RegistrationStore,
		executionMode:     config.ExecutionMode,
		playbookBasePath:  strings.TrimSpace(config.PlaybookBasePath),
		configurePath:     strings.TrimSpace(config.ConfigurePlaybookPath),
		controlPlaneURL:   strings.TrimSpace(config.ControlPlaneURL),
		logger:            logger,
	}
}

// providerPlaybook resolves a playbook path for the given provider type.
// The convention is: <playbookBasePath>/<providerType>/<filename>.
func (e *Executor) providerPlaybook(providerType, filename string) string {
	return filepath.Join(e.playbookBasePath, providerType, filename)
}

func (e *Executor) siteDeployPlaybook() string {
	return filepath.Join(e.playbookBasePath, playbookSiteDeploy)
}

// Execute runs all steps for a job. It handles state transitions and event emission.
func (e *Executor) Execute(ctx context.Context, job *orchestrator.Job) error {
	switch job.Kind {
	case string(orchestrator.JobKindProvisionServer):
		return e.executeProvisionServer(ctx, job)
	case string(orchestrator.JobKindConfigureServer):
		return e.executeConfigureServer(ctx, job)
	case string(orchestrator.JobKindDeleteServer):
		return e.executeDeleteServer(ctx, job)
	case string(orchestrator.JobKindRebuildServer):
		return e.executeRebuildServer(ctx, job)
	case string(orchestrator.JobKindResizeServer):
		return e.executeResizeServer(ctx, job)
	case string(orchestrator.JobKindUpdateFirewalls):
		return e.executeUpdateFirewalls(ctx, job)
	case string(orchestrator.JobKindManageVolume):
		return e.executeManageVolume(ctx, job)
	case string(orchestrator.JobKindRestartService):
		return e.executeAgentJob(ctx, job)
	case string(orchestrator.JobKindDeploySite):
		return e.executeDeploySite(ctx, job)
	default:
		return e.failJob(ctx, job, fmt.Sprintf("unknown job kind: %s", job.Kind))
	}
}

// --- Shared helpers ---

func (e *Executor) setServerStatus(ctx context.Context, serverID string, status platform.ServerStatus) {
	if strings.TrimSpace(serverID) == "" || strings.TrimSpace(string(status)) == "" {
		return
	}
	if err := e.serverStore.UpdateStatus(ctx, serverID, status); err != nil {
		e.logger.Error("server status persistence failed", "server_id", serverID, "server_status", status, "error", err)
		return
	}
	e.logger.Info("server status updated", "server_id", serverID, "server_status", status)
}

func (e *Executor) setSetupState(ctx context.Context, serverID string, setupState platform.SetupState, setupLastError string) {
	if strings.TrimSpace(serverID) == "" || strings.TrimSpace(string(setupState)) == "" {
		return
	}
	if err := e.serverStore.UpdateSetupState(ctx, serverID, setupState, setupLastError); err != nil {
		e.logger.Error("server setup state persistence failed", "server_id", serverID, "setup_state", setupState, "error", err)
		return
	}
	e.logger.Info("server setup state updated", "server_id", serverID, "setup_state", setupState)
}

func (e *Executor) runLocalPlaybook(ctx context.Context, jobID string, playbookPath string, extraVars map[string]string) error {
	workspace, err := os.MkdirTemp("", "pressluft-ansible-")
	if err != nil {
		return fmt.Errorf("failed to create ansible workspace: %v", err)
	}
	defer os.RemoveAll(workspace)

	inventoryPath := filepath.Join(workspace, "inventory.ini")
	if err := os.WriteFile(inventoryPath, []byte("localhost ansible_connection=local\n"), 0o600); err != nil {
		return fmt.Errorf("failed to write ansible inventory: %v", err)
	}

	request := runner.Request{
		JobID:         jobID,
		InventoryPath: inventoryPath,
		PlaybookPath:  playbookPath,
		ExtraVars:     extraVars,
	}

	return e.runner.Run(ctx, request, &runnerEventSink{jobStore: e.jobStore, jobID: jobID, logger: e.logger})
}

func (e *Executor) completeJob(ctx context.Context, job *orchestrator.Job, step string) error {
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusSucceeded,
		CurrentStep: step,
	}); err != nil {
		return fmt.Errorf("transition to succeeded: %w", err)
	}

	e.emitEvent(ctx, job.ID, orchestrator.JobEventTypeSucceeded, "info", "", string(orchestrator.JobStatusSucceeded), "Job completed successfully")

	// Emit job completed activity
	input := activity.EmitInput{
		EventType:    activity.EventJobCompleted,
		Category:     activity.CategoryJob,
		Level:        activity.LevelSuccess,
		ResourceType: activity.ResourceJob,
		ResourceID:   job.ID,
		ActorType:    activity.ActorSystem,
		Title:        fmt.Sprintf("%s completed", orchestrator.JobKindLabel(job.Kind)),
	}
	if siteID := e.siteIDForJob(*job); siteID != "" {
		input.ParentResourceType = activity.ResourceSite
		input.ParentResourceID = siteID
	} else if job.ServerID != "" {
		input.ParentResourceType = activity.ResourceServer
		input.ParentResourceID = job.ServerID
	}
	e.emitActivity(ctx, input)

	return nil
}

func (e *Executor) failJob(ctx context.Context, job *orchestrator.Job, errMsg string) error {
	corr := observability.Correlation{JobID: job.ID, ServerID: job.ServerID, CommandID: derefString(job.CommandID)}
	e.logger.Error("job failed", corr.LogArgs("error", errMsg)...)

	if job.ServerID != "" && job.Kind != string(orchestrator.JobKindDeploySite) {
		if job.Kind == string(orchestrator.JobKindConfigureServer) {
			e.setSetupState(ctx, job.ServerID, platform.SetupStateDegraded, errMsg)
		} else if err := e.serverStore.UpdateStatus(ctx, job.ServerID, platform.ServerStatusFailed); err != nil {
			e.logger.Error("server failure status persistence failed", corr.LogArgs("server_status", platform.ServerStatusFailed, "error", err)...)
		}
	}

	// Emit failure event
	e.emitEvent(ctx, job.ID, orchestrator.JobEventTypeFailed, "error", job.CurrentStep, string(orchestrator.JobStatusFailed), errMsg)

	// Transition job to failed
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:  orchestrator.JobStatusFailed,
		LastError: errMsg,
	}); err != nil {
		e.logger.Error("job failure transition persistence failed", corr.LogArgs("error", err)...)
	}

	// Emit job failed activity with requires_attention flag
	input := activity.EmitInput{
		EventType:         activity.EventJobFailed,
		Category:          activity.CategoryJob,
		Level:             activity.LevelError,
		ResourceType:      activity.ResourceJob,
		ResourceID:        job.ID,
		ActorType:         activity.ActorSystem,
		Title:             fmt.Sprintf("%s failed", orchestrator.JobKindLabel(job.Kind)),
		Message:           errMsg,
		RequiresAttention: true,
	}
	if siteID := e.siteIDForJob(*job); siteID != "" {
		input.ParentResourceType = activity.ResourceSite
		input.ParentResourceID = siteID
	} else if job.ServerID != "" {
		input.ParentResourceType = activity.ResourceServer
		input.ParentResourceID = job.ServerID
	}
	e.emitActivity(ctx, input)

	return fmt.Errorf("job failed: %s", errMsg)
}

func (e *Executor) updateStep(ctx context.Context, jobID string, step string) {
	if _, err := e.jobStore.TransitionJob(ctx, jobID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: step,
	}); err != nil {
		e.logger.Error("job step transition persistence failed", "job_id", jobID, "step", step, "error", err)
	}
}

func (e *Executor) siteIDForJob(job orchestrator.Job) string {
	if job.Kind != string(orchestrator.JobKindDeploySite) {
		return ""
	}
	payload, err := orchestrator.UnmarshalDeploySitePayload(job.Payload)
	if err != nil {
		return ""
	}
	return strings.TrimSpace(payload.SiteID)
}

func (e *Executor) emitStepStart(ctx context.Context, jobID string, step, message string) {
	e.emitEvent(ctx, jobID, orchestrator.JobEventTypeStepStarted, "info", step, string(orchestrator.JobStatusRunning), message)
}

func (e *Executor) emitStepComplete(ctx context.Context, jobID string, step, message string) {
	e.emitEvent(ctx, jobID, orchestrator.JobEventTypeStepComplete, "info", step, "completed", message)
}

func (e *Executor) emitEvent(ctx context.Context, jobID string, eventType, level, step, status, message string) {
	event, err := e.jobStore.AppendEvent(ctx, jobID, orchestrator.CreateEventInput{
		EventType: eventType,
		Level:     level,
		StepKey:   step,
		Status:    status,
		Message:   message,
	})
	if err != nil {
		e.logger.Error("job event append failed", "job_id", jobID, "event_type", eventType, "error", err)
		return
	}
	e.logger.Debug("job event appended", "job_id", jobID, "event_seq", event.Seq, "event_type", event.EventType, "status", status)
}

type runnerEventSink struct {
	jobStore *orchestrator.Store
	jobID    string
	logger   *slog.Logger
}

func (s *runnerEventSink) Emit(ctx context.Context, event runner.Event) error {
	stepKey := strings.TrimSpace(event.StepKey)
	if stepKey == "" {
		stepKey = "ansible"
	}
	_, err := s.jobStore.AppendEvent(ctx, s.jobID, orchestrator.CreateEventInput{
		EventType: runnerEventType(event.Type),
		Level:     event.Level,
		StepKey:   stepKey,
		Status:    event.Type,
		Message:   event.Message,
		Payload:   event.Payload,
	})
	if err != nil && s.logger != nil {
		s.logger.Error("runner event append failed", "job_id", s.jobID, "event_type", event.Type, "step_key", stepKey, "error", err)
	}
	return err
}

func runnerEventType(status string) string {
	switch strings.TrimSpace(status) {
	case "running", "started":
		return orchestrator.JobEventTypeStepStarted
	default:
		return orchestrator.JobEventTypeStepComplete
	}
}

// emitActivity emits an activity event if the activity store is configured.
func (e *Executor) emitActivity(ctx context.Context, input activity.EmitInput) {
	if e.activityStore == nil {
		return
	}
	activityEntry, err := e.activityStore.Emit(ctx, input)
	if err != nil {
		e.logger.Error("activity emit failed", "event_type", input.EventType, "resource_type", input.ResourceType, "resource_id", input.ResourceID, "parent_resource_type", input.ParentResourceType, "parent_resource_id", input.ParentResourceID, "error", err)
		return
	}
	e.logger.Debug("activity emitted", "activity_id", activityEntry.ID, "event_type", activityEntry.EventType, "resource_type", activityEntry.ResourceType, "resource_id", activityEntry.ResourceID, "parent_resource_type", activityEntry.ParentResourceType, "parent_resource_id", activityEntry.ParentResourceID)
}

func derefString(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}
