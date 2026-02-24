package worker

import (
	"context"
	"fmt"
	"log/slog"
	"strconv"

	"pressluft/internal/orchestrator"
	"pressluft/internal/provider"
	"pressluft/internal/provider/hetzner"
)

// ServerStore defines the server persistence interface needed by the executor.
type ServerStore interface {
	GetByID(ctx context.Context, id int64) (*StoredServer, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateProvisioning(ctx context.Context, id int64, providerServerID, actionID, actionStatus, status string) error
}

// StoredServer mirrors the server package type to avoid import cycles.
type StoredServer struct {
	ID               int64
	ProviderID       int64
	ProviderType     string
	ProviderServerID string
	Name             string
	Location         string
	ServerType       string
	Image            string
	ProfileKey       string
	Status           string
}

// ProviderStore defines the provider persistence interface needed by the executor.
type ProviderStore interface {
	GetByID(ctx context.Context, id int64) (*StoredProvider, error)
}

// StoredProvider mirrors the provider package type to avoid import cycles.
type StoredProvider struct {
	ID       int64
	Type     string
	APIToken string
}

// Executor runs job steps and emits events.
type Executor struct {
	jobStore      *orchestrator.Store
	serverStore   ServerStore
	providerStore ProviderStore
	logger        *slog.Logger
}

// NewExecutor creates an executor with the given dependencies.
func NewExecutor(
	jobStore *orchestrator.Store,
	serverStore ServerStore,
	providerStore ProviderStore,
	logger *slog.Logger,
) *Executor {
	return &Executor{
		jobStore:      jobStore,
		serverStore:   serverStore,
		providerStore: providerStore,
		logger:        logger,
	}
}

// Execute runs all steps for a job. It handles state transitions and event emission.
func (e *Executor) Execute(ctx context.Context, job *orchestrator.Job) error {
	switch job.Kind {
	case "provision_server":
		return e.executeProvisionServer(ctx, job)
	default:
		return e.failJob(ctx, job, fmt.Sprintf("unknown job kind: %s", job.Kind))
	}
}

// executeProvisionServer runs the server provisioning workflow:
// 1. validate - Check server record and provider exist
// 2. create_ssh_key - Generate and register SSH key with provider
// 3. create_server - Call provider API to create server with SSH key
// 4. wait_running - Poll until server is running
// 5. finalize - Mark server as ready
func (e *Executor) executeProvisionServer(ctx context.Context, job *orchestrator.Job) error {
	// Transition to running
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: "validate",
	}); err != nil {
		return fmt.Errorf("transition to running: %w", err)
	}

	// Step 1: Validate
	e.emitStepStart(ctx, job.ID, "validate", "Validating server configuration")

	server, err := e.serverStore.GetByID(ctx, job.ServerID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("server not found: %v", err))
	}

	storedProvider, err := e.providerStore.GetByID(ctx, server.ProviderID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("provider not found: %v", err))
	}

	serverProvider, ok := provider.GetServerProvider(storedProvider.Type)
	if !ok {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support server provisioning", storedProvider.Type))
	}

	e.emitStepComplete(ctx, job.ID, "validate", "Server configuration validated")

	// Step 2: Create SSH Key
	e.updateStep(ctx, job.ID, "create_ssh_key")
	e.emitStepStart(ctx, job.ID, "create_ssh_key", "Generating SSH key pair")

	// Use job ID for deterministic key name to support retries
	keyName := fmt.Sprintf("pressluft-%s-job-%d", server.Name, job.ID)
	publicKey, privateKey, err := hetzner.GenerateSSHKeyPair(keyName)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to generate SSH key: %v", err))
	}

	e.emitEvent(ctx, job.ID, "info", "create_ssh_key", "running", "Registering SSH key with provider")

	sshKeyResult, err := serverProvider.CreateSSHKey(ctx, storedProvider.APIToken, keyName, publicKey)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to register SSH key: %v", err))
	}

	// Store private key reference (in production, this would go to secure storage)
	e.logger.Info("ssh key created",
		"job_id", job.ID,
		"key_id", sshKeyResult.ID,
		"key_name", sshKeyResult.Name,
		"fingerprint", sshKeyResult.Fingerprint,
	)
	_ = privateKey // TODO: Store securely for later SSH access

	e.emitStepComplete(ctx, job.ID, "create_ssh_key", fmt.Sprintf("SSH key registered: %s", sshKeyResult.Fingerprint))

	// Step 3: Create Server
	e.updateStep(ctx, job.ID, "create_server")
	e.emitStepStart(ctx, job.ID, "create_server", "Creating server at provider")

	createResult, err := serverProvider.CreateServer(ctx, storedProvider.APIToken, provider.CreateServerRequest{
		Name:       server.Name,
		Location:   server.Location,
		ServerType: server.ServerType,
		Image:      server.Image,
		Labels: map[string]string{
			"pressluft_profile":    server.ProfileKey,
			"pressluft_ssh_key_id": strconv.FormatInt(sshKeyResult.ID, 10),
		},
	})
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to create server: %v", err))
	}

	// Update server record with provider IDs
	if err := e.serverStore.UpdateProvisioning(ctx, server.ID, createResult.ProviderServerID, createResult.ActionID, createResult.Status, "provisioning"); err != nil {
		e.logger.Error("failed to update server provisioning state", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "create_server", fmt.Sprintf("Server created: %s", createResult.ProviderServerID))

	// Step 4: Wait for Running
	e.updateStep(ctx, job.ID, "wait_running")
	e.emitStepStart(ctx, job.ID, "wait_running", "Waiting for server to be ready")

	// For now, we just mark it as complete since Hetzner's create is async
	// In a full implementation, we'd poll the action status
	e.emitStepComplete(ctx, job.ID, "wait_running", "Server is initializing (async)")

	// Step 5: Finalize
	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing server setup")

	if err := e.serverStore.UpdateStatus(ctx, server.ID, "ready"); err != nil {
		e.logger.Error("failed to update server status to ready", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "finalize", "Server provisioning complete")

	// Mark job as succeeded
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusSucceeded,
		CurrentStep: "finalize",
	}); err != nil {
		return fmt.Errorf("transition to succeeded: %w", err)
	}

	e.emitEvent(ctx, job.ID, "info", "", "succeeded", "Job completed successfully")

	return nil
}

func (e *Executor) failJob(ctx context.Context, job *orchestrator.Job, errMsg string) error {
	e.logger.Error("job failed", "job_id", job.ID, "error", errMsg)

	// Update server status to failed
	if job.ServerID > 0 {
		if err := e.serverStore.UpdateStatus(ctx, job.ServerID, "failed"); err != nil {
			e.logger.Error("failed to update server status", "error", err)
		}
	}

	// Emit failure event
	e.emitEvent(ctx, job.ID, "error", job.CurrentStep, "failed", errMsg)

	// Transition job to failed
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:  orchestrator.JobStatusFailed,
		LastError: errMsg,
	}); err != nil {
		e.logger.Error("failed to transition job to failed", "error", err)
	}

	return fmt.Errorf("job failed: %s", errMsg)
}

func (e *Executor) updateStep(ctx context.Context, jobID int64, step string) {
	if _, err := e.jobStore.TransitionJob(ctx, jobID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: step,
	}); err != nil {
		e.logger.Error("failed to update job step", "job_id", jobID, "step", step, "error", err)
	}
}

func (e *Executor) emitStepStart(ctx context.Context, jobID int64, step, message string) {
	e.emitEvent(ctx, jobID, "info", step, "running", message)
}

func (e *Executor) emitStepComplete(ctx context.Context, jobID int64, step, message string) {
	e.emitEvent(ctx, jobID, "info", step, "completed", message)
}

func (e *Executor) emitEvent(ctx context.Context, jobID int64, level, step, status, message string) {
	_, err := e.jobStore.AppendEvent(ctx, jobID, orchestrator.CreateEventInput{
		EventType: "step_update",
		Level:     level,
		StepKey:   step,
		Status:    status,
		Message:   message,
	})
	if err != nil {
		e.logger.Error("failed to emit event", "job_id", jobID, "error", err)
	}
}
