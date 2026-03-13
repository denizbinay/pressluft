package worker

import (
	"context"
	"crypto/rand"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/mail"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/activity"
	"pressluft/internal/observability"
	"pressluft/internal/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/provider"
	"pressluft/internal/runner"
	"pressluft/internal/security"
	serverpkg "pressluft/internal/server"
	"pressluft/internal/server/profiles"
	"pressluft/internal/sshutil"
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

func (e *Executor) executeAgentJob(ctx context.Context, job *orchestrator.Job) error {
	if strings.TrimSpace(job.ServerID) == "" {
		return e.failJob(ctx, job, "server_id is required for agent job")
	}
	if e.agentRunner == nil {
		return e.failJob(ctx, job, "agent runner not configured")
	}

	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobStarted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   job.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s started", orchestrator.JobKindLabel(job.Kind)),
	})

	if err := e.agentRunner.Run(ctx, *job); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("agent dispatch failed: %v", err))
	}

	return nil
}

// executeProvisionServer provisions provider infrastructure and then queues setup.
func (e *Executor) executeProvisionServer(ctx context.Context, job *orchestrator.Job) error {
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: "validate",
	}); err != nil {
		return fmt.Errorf("transition to running: %w", err)
	}

	// Emit job started activity
	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobStarted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   job.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s started", orchestrator.JobKindLabel(job.Kind)),
	})

	e.emitStepStart(ctx, job.ID, "validate", "Validating server configuration")

	server, err := e.serverStore.GetByID(ctx, job.ServerID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("server not found: %v", err))
	}

	storedProvider, err := e.providerStore.GetByID(ctx, server.ProviderID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("provider not found: %v", err))
	}

	if !provider.SupportsProvisioningWorkflow(storedProvider.Type) {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible provisioning", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required for provisioning")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.playbookBasePath) == "" {
		return e.failJob(ctx, job, "playbook base path not configured")
	}

	e.emitStepComplete(ctx, job.ID, "validate", "Server configuration validated")
	e.setServerStatus(ctx, server.ID, platform.ServerStatusProvisioning)
	e.setSetupState(ctx, server.ID, platform.SetupStateNotStarted, "")

	e.updateStep(ctx, job.ID, "provision")
	e.emitStepStart(ctx, job.ID, "provision", "Running provision playbook")

	keyName := fmt.Sprintf("pressluft-server-%s", server.ID)
	storedKey, err := e.serverStore.GetKey(ctx, server.ID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to read SSH key: %v", err))
	}

	var publicKey string
	if storedKey != nil {
		publicKey = storedKey.PublicKey
		e.logSSHPublicKey("stored", publicKey)
	} else {
		var generatedPrivateKey string
		publicKey, generatedPrivateKey, err = sshutil.GenerateKeyPair(keyName)
		if err != nil {
			return e.failJob(ctx, job, fmt.Sprintf("failed to generate SSH key: %v", err))
		}
		e.logSSHPublicKey("generated", publicKey)
		encryptedKey, keyID, err := security.Encrypt([]byte(generatedPrivateKey))
		if err != nil {
			return e.failJob(ctx, job, fmt.Sprintf("failed to encrypt SSH key: %v", err))
		}
		if err := e.serverStore.CreateKey(ctx, serverpkg.CreateServerKeyInput{
			ServerID:            server.ID,
			PublicKey:           publicKey,
			PrivateKeyEncrypted: encryptedKey,
			EncryptionKeyID:     keyID,
		}); err != nil {
			storedKey, err = e.serverStore.GetKey(ctx, server.ID)
			if err != nil {
				return e.failJob(ctx, job, fmt.Sprintf("failed to read existing SSH key: %v", err))
			}
			if storedKey == nil {
				return e.failJob(ctx, job, fmt.Sprintf("failed to store SSH key: %v", err))
			}
			publicKey = storedKey.PublicKey
			e.logSSHPublicKey("stored_after_conflict", publicKey)
		} else {
			storedKey = &serverpkg.StoredServerKey{
				ServerID:            server.ID,
				PublicKey:           publicKey,
				PrivateKeyEncrypted: encryptedKey,
				EncryptionKeyID:     keyID,
			}
		}
	}
	if err := validateSSHPublicKey(publicKey); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("invalid SSH public key: %v", err))
	}
	e.logSSHPublicKey("pre_ansible", publicKey)

	workspace, err := os.MkdirTemp("", "pressluft-ansible-")
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to create ansible workspace: %v", err))
	}
	defer os.RemoveAll(workspace)

	inventoryPath := filepath.Join(workspace, "inventory.ini")
	if err := os.WriteFile(inventoryPath, []byte("localhost ansible_connection=local\n"), 0o600); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to write ansible inventory: %v", err))
	}

	artifactPath := filepath.Join(workspace, "provision-result.json")
	request := runner.Request{
		JobID:         job.ID,
		InventoryPath: inventoryPath,
		PlaybookPath:  e.providerPlaybook(storedProvider.Type, playbookProvision),
		ExtraVars: map[string]string{
			"api_token":       storedProvider.APIToken,
			"server_name":     server.Name,
			"server_location": server.Location,
			"server_type":     server.ServerType,
			"server_image":    server.Image,
			"ssh_key_name":    keyName,
			"ssh_public_key":  publicKey,
			"artifact_path":   artifactPath,
		},
	}

	if err := e.runner.Run(ctx, request, &runnerEventSink{jobStore: e.jobStore, jobID: job.ID, logger: e.logger}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible provision failed: %v", err))
	}

	result, err := readProvisionArtifact(artifactPath)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to read provision result: %v", err))
	}
	if result.ID == 0 {
		return e.failJob(ctx, job, "provision result missing server id")
	}
	if strings.TrimSpace(result.IPv4) == "" {
		return e.failJob(ctx, job, "provision result missing IPv4")
	}

	providerServerID := strconv.FormatInt(result.ID, 10)
	if err := e.serverStore.UpdateProvisioning(ctx, server.ID, providerServerID, "", "", platform.ServerStatusProvisioning, result.IPv4, result.IPv6); err != nil {
		e.logger.Error("failed to update server provisioning state", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "provision", fmt.Sprintf("Server created: %s", providerServerID))
	e.setServerStatus(ctx, server.ID, platform.ServerStatusConfiguring)
	e.setSetupState(ctx, server.ID, platform.SetupStateRunning, "")

	payloadBytes, err := json.Marshal(orchestrator.ConfigureServerPayload{IPv4: result.IPv4})
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("marshal configure payload: %v", err))
	}
	configureJob, err := e.jobStore.CreateJob(ctx, orchestrator.CreateJobInput{
		Kind:     string(orchestrator.JobKindConfigureServer),
		ServerID: server.ID,
		Payload:  string(payloadBytes),
	})
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("queue configure job: %v", err))
	}
	e.emitEvent(ctx, configureJob.ID, orchestrator.JobEventTypeCreated, "info", "", string(configureJob.Status), "Setup job accepted and queued")

	if err := e.completeJob(ctx, job, "provision"); err != nil {
		return err
	}

	e.emitActivity(ctx, activity.EmitInput{
		EventType:    activity.EventServerProvisioned,
		Category:     activity.CategoryServer,
		Level:        activity.LevelSuccess,
		ResourceType: activity.ResourceServer,
		ResourceID:   job.ServerID,
		ActorType:    activity.ActorSystem,
		Title:        fmt.Sprintf("Server '%s' provisioned", server.Name),
	})
	return nil
}

func (e *Executor) executeConfigureServer(ctx context.Context, job *orchestrator.Job) error {
	if strings.TrimSpace(job.ServerID) == "" {
		return e.failJob(ctx, job, "server_id is required for configure_server job")
	}
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: "validate",
	}); err != nil {
		return fmt.Errorf("transition to running: %w", err)
	}

	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobStarted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   job.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s started", orchestrator.JobKindLabel(job.Kind)),
	})

	e.emitStepStart(ctx, job.ID, "validate", "Validating server setup request")
	server, err := e.serverStore.GetByID(ctx, job.ServerID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("server not found: %v", err))
	}
	if strings.TrimSpace(e.configurePath) == "" {
		return e.failJob(ctx, job, "configure playbook path not configured")
	}

	payload, err := orchestrator.UnmarshalConfigureServerPayload(job.Payload)
	if err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	ipv4 := strings.TrimSpace(payload.IPv4)
	if ipv4 == "" {
		ipv4 = strings.TrimSpace(server.IPv4)
	}
	if ipv4 == "" {
		return e.failJob(ctx, job, "server IPv4 is required for configure_server job")
	}

	e.emitStepComplete(ctx, job.ID, "validate", "Server setup request validated")
	e.updateStep(ctx, job.ID, "configure")
	e.emitStepStart(ctx, job.ID, "configure", "Running configure playbook")
	e.setServerStatus(ctx, server.ID, platform.ServerStatusConfiguring)
	e.setSetupState(ctx, server.ID, platform.SetupStateRunning, "")

	if err := e.runConfigurePlaybook(ctx, job.ID, server, ipv4, "", nil); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible configure failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "configure", "Configure playbook completed")
	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing server setup")
	e.setSetupState(ctx, server.ID, platform.SetupStateReady, "")
	e.setServerStatus(ctx, server.ID, platform.ServerStatusReady)
	e.emitStepComplete(ctx, job.ID, "finalize", "Server setup complete")

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) executeDeleteServer(ctx context.Context, job *orchestrator.Job) error {
	if strings.TrimSpace(job.ServerID) == "" {
		return e.failJob(ctx, job, "server_id is required for delete_server job")
	}
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: "validate",
	}); err != nil {
		return fmt.Errorf("transition to running: %w", err)
	}

	// Emit job started activity
	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobStarted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   job.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s started", orchestrator.JobKindLabel(job.Kind)),
	})

	e.emitStepStart(ctx, job.ID, "validate", "Validating server delete request")

	server, err := e.serverStore.GetByID(ctx, job.ServerID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("server not found: %v", err))
	}

	storedProvider, err := e.providerStore.GetByID(ctx, server.ProviderID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("provider not found: %v", err))
	}
	if !provider.SupportsServerMutationWorkflow(storedProvider.Type) {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible workflows", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.playbookBasePath) == "" {
		return e.failJob(ctx, job, "playbook base path not configured")
	}

	if _, err := orchestrator.UnmarshalDeleteServerPayload(job.Payload); err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	serverName := strings.TrimSpace(server.Name)
	if serverName == "" {
		return e.failJob(ctx, job, "server_name is required for delete_server job")
	}

	e.emitStepComplete(ctx, job.ID, "validate", "Delete request validated")

	e.updateStep(ctx, job.ID, "delete")
	e.emitStepStart(ctx, job.ID, "delete", "Running delete playbook")

	if err := e.runLocalPlaybook(ctx, job.ID, e.providerPlaybook(storedProvider.Type, playbookDelete), map[string]string{
		"api_token":   storedProvider.APIToken,
		"server_name": serverName,
	}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible delete failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "delete", "Delete playbook completed")

	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing delete")

	if err := e.serverStore.UpdateStatus(ctx, server.ID, platform.ServerStatusDeleted); err != nil {
		e.logger.Error("failed to update server status to deleted", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "finalize", "Server delete complete")
	e.emitActivity(ctx, activity.EmitInput{
		EventType:    activity.EventServerDeleted,
		Category:     activity.CategoryServer,
		Level:        activity.LevelSuccess,
		ResourceType: activity.ResourceServer,
		ResourceID:   server.ID,
		ActorType:    activity.ActorSystem,
		Title:        fmt.Sprintf("Server '%s' deleted", server.Name),
	})

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) executeRebuildServer(ctx context.Context, job *orchestrator.Job) error {
	if strings.TrimSpace(job.ServerID) == "" {
		return e.failJob(ctx, job, "server_id is required for rebuild_server job")
	}
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: "validate",
	}); err != nil {
		return fmt.Errorf("transition to running: %w", err)
	}

	// Emit job started activity
	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobStarted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   job.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s started", orchestrator.JobKindLabel(job.Kind)),
	})

	e.emitStepStart(ctx, job.ID, "validate", "Validating server rebuild request")

	server, err := e.serverStore.GetByID(ctx, job.ServerID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("server not found: %v", err))
	}

	storedProvider, err := e.providerStore.GetByID(ctx, server.ProviderID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("provider not found: %v", err))
	}
	if !provider.SupportsServerMutationWorkflow(storedProvider.Type) {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible workflows", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.playbookBasePath) == "" {
		return e.failJob(ctx, job, "playbook base path not configured")
	}

	payload, err := orchestrator.UnmarshalRebuildServerPayload(job.Payload)
	if err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	serverName := strings.TrimSpace(server.Name)
	serverImage := strings.TrimSpace(payload.ServerImage)
	if serverImage == "" {
		serverImage = strings.TrimSpace(server.Image)
	}
	if serverName == "" {
		return e.failJob(ctx, job, "server_name is required for rebuild_server job")
	}
	if serverImage == "" {
		return e.failJob(ctx, job, "server_image is required for rebuild_server job")
	}
	if _, err := validateSelectableProfile(server.ProfileKey); err != nil {
		return e.failJob(ctx, job, err.Error())
	}

	e.emitStepComplete(ctx, job.ID, "validate", "Rebuild request validated")

	e.updateStep(ctx, job.ID, "rebuild")
	e.emitStepStart(ctx, job.ID, "rebuild", "Running rebuild playbook")

	if err := e.runLocalPlaybook(ctx, job.ID, e.providerPlaybook(storedProvider.Type, playbookRebuild), map[string]string{
		"api_token":    storedProvider.APIToken,
		"server_name":  serverName,
		"server_image": serverImage,
	}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible rebuild failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "rebuild", "Rebuild playbook completed")
	e.setServerStatus(ctx, server.ID, platform.ServerStatusConfiguring)
	e.setSetupState(ctx, server.ID, platform.SetupStateRunning, "")
	if err := e.serverStore.UpdateImage(ctx, server.ID, serverImage); err != nil {
		e.logger.Error("failed to update server image", "error", err)
	}
	if strings.TrimSpace(server.IPv4) == "" {
		return e.failJob(ctx, job, "server IPv4 is required for rebuild follow-up setup")
	}
	configurePayload, err := json.Marshal(orchestrator.ConfigureServerPayload{IPv4: server.IPv4})
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("marshal rebuild configure payload: %v", err))
	}
	configureJob, err := e.jobStore.CreateJob(ctx, orchestrator.CreateJobInput{
		Kind:     string(orchestrator.JobKindConfigureServer),
		ServerID: server.ID,
		Payload:  string(configurePayload),
	})
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("queue configure job after rebuild: %v", err))
	}
	e.emitEvent(ctx, configureJob.ID, orchestrator.JobEventTypeCreated, "info", "", string(configureJob.Status), "Setup job accepted and queued")

	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Queued follow-up server setup")
	e.emitStepComplete(ctx, job.ID, "finalize", "Server rebuild complete")

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) executeResizeServer(ctx context.Context, job *orchestrator.Job) error {
	if strings.TrimSpace(job.ServerID) == "" {
		return e.failJob(ctx, job, "server_id is required for resize_server job")
	}
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: "validate",
	}); err != nil {
		return fmt.Errorf("transition to running: %w", err)
	}

	// Emit job started activity
	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobStarted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   job.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s started", orchestrator.JobKindLabel(job.Kind)),
	})

	e.emitStepStart(ctx, job.ID, "validate", "Validating server resize request")

	server, err := e.serverStore.GetByID(ctx, job.ServerID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("server not found: %v", err))
	}

	storedProvider, err := e.providerStore.GetByID(ctx, server.ProviderID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("provider not found: %v", err))
	}
	if !provider.SupportsServerMutationWorkflow(storedProvider.Type) {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible workflows", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.playbookBasePath) == "" {
		return e.failJob(ctx, job, "playbook base path not configured")
	}

	payload, err := orchestrator.UnmarshalResizeServerPayload(job.Payload)
	if err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	serverType := strings.TrimSpace(payload.ServerType)
	if serverType == "" {
		return e.failJob(ctx, job, "server_type is required for resize_server job")
	}

	e.emitStepComplete(ctx, job.ID, "validate", "Resize request validated")

	e.updateStep(ctx, job.ID, "resize")
	e.emitStepStart(ctx, job.ID, "resize", "Running resize playbook")

	if err := e.runLocalPlaybook(ctx, job.ID, e.providerPlaybook(storedProvider.Type, playbookResize), map[string]string{
		"api_token":    storedProvider.APIToken,
		"server_name":  server.Name,
		"server_type":  serverType,
		"upgrade_disk": strconv.FormatBool(payload.UpgradeDisk),
	}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible resize failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "resize", "Resize playbook completed")

	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing resize")

	if err := e.serverStore.UpdateServerType(ctx, server.ID, serverType); err != nil {
		e.logger.Error("failed to update server type", "error", err)
	}
	if err := e.serverStore.UpdateStatus(ctx, server.ID, platform.ServerStatusReady); err != nil {
		e.logger.Error("failed to update server status to ready", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "finalize", "Server resize complete")

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) executeUpdateFirewalls(ctx context.Context, job *orchestrator.Job) error {
	if strings.TrimSpace(job.ServerID) == "" {
		return e.failJob(ctx, job, "server_id is required for update_firewalls job")
	}
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: "validate",
	}); err != nil {
		return fmt.Errorf("transition to running: %w", err)
	}

	// Emit job started activity
	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobStarted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   job.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s started", orchestrator.JobKindLabel(job.Kind)),
	})

	e.emitStepStart(ctx, job.ID, "validate", "Validating firewall update request")

	server, err := e.serverStore.GetByID(ctx, job.ServerID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("server not found: %v", err))
	}

	storedProvider, err := e.providerStore.GetByID(ctx, server.ProviderID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("provider not found: %v", err))
	}
	if !provider.SupportsServerMutationWorkflow(storedProvider.Type) {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible workflows", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.playbookBasePath) == "" {
		return e.failJob(ctx, job, "playbook base path not configured")
	}

	payload, err := orchestrator.UnmarshalUpdateFirewallsPayload(job.Payload)
	if err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	firewalls := make([]string, 0, len(payload.Firewalls))
	for _, fw := range payload.Firewalls {
		fw = strings.TrimSpace(fw)
		if fw != "" {
			firewalls = append(firewalls, fw)
		}
	}
	if len(firewalls) == 0 {
		return e.failJob(ctx, job, "firewalls payload must contain at least one firewall")
	}
	firewallsCSV := strings.Join(firewalls, ",")

	e.emitStepComplete(ctx, job.ID, "validate", "Firewall update validated")

	e.updateStep(ctx, job.ID, "update_firewalls")
	e.emitStepStart(ctx, job.ID, "update_firewalls", "Running firewall update playbook")

	if err := e.runLocalPlaybook(ctx, job.ID, e.providerPlaybook(storedProvider.Type, playbookFirewalls), map[string]string{
		"api_token":     storedProvider.APIToken,
		"server_name":   server.Name,
		"firewalls_csv": firewallsCSV,
	}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible firewall update failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "update_firewalls", "Firewall update playbook completed")

	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing firewall update")

	if err := e.serverStore.UpdateStatus(ctx, server.ID, platform.ServerStatusReady); err != nil {
		e.logger.Error("failed to update server status to ready", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "finalize", "Firewall update complete")

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) executeManageVolume(ctx context.Context, job *orchestrator.Job) error {
	if strings.TrimSpace(job.ServerID) == "" {
		return e.failJob(ctx, job, "server_id is required for manage_volume job")
	}
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusRunning,
		CurrentStep: "validate",
	}); err != nil {
		return fmt.Errorf("transition to running: %w", err)
	}

	// Emit job started activity
	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobStarted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   job.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s started", orchestrator.JobKindLabel(job.Kind)),
	})

	e.emitStepStart(ctx, job.ID, "validate", "Validating volume request")

	server, err := e.serverStore.GetByID(ctx, job.ServerID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("server not found: %v", err))
	}

	storedProvider, err := e.providerStore.GetByID(ctx, server.ProviderID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("provider not found: %v", err))
	}
	if !provider.SupportsServerMutationWorkflow(storedProvider.Type) {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible workflows", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.playbookBasePath) == "" {
		return e.failJob(ctx, job, "playbook base path not configured")
	}

	payload, err := orchestrator.UnmarshalManageVolumePayload(job.Payload)
	if err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	volumeName := strings.TrimSpace(payload.VolumeName)
	state := strings.TrimSpace(payload.State)
	location := strings.TrimSpace(payload.Location)
	if volumeName == "" {
		return e.failJob(ctx, job, "volume_name is required for manage_volume job")
	}
	if state != "present" && state != "absent" {
		return e.failJob(ctx, job, "state must be present or absent for manage_volume job")
	}
	if state == "present" {
		if payload.Automount == nil {
			return e.failJob(ctx, job, "automount is required for manage_volume job when state=present")
		}
		if payload.SizeGB <= 0 {
			return e.failJob(ctx, job, "size_gb is required for manage_volume job when state=present")
		}
	}

	e.emitStepComplete(ctx, job.ID, "validate", "Volume request validated")

	e.updateStep(ctx, job.ID, "manage_volume")
	e.emitStepStart(ctx, job.ID, "manage_volume", "Running volume playbook")
	automountValue := "false"
	if payload.Automount != nil {
		automountValue = strconv.FormatBool(*payload.Automount)
	}

	if err := e.runLocalPlaybook(ctx, job.ID, e.providerPlaybook(storedProvider.Type, playbookVolume), map[string]string{
		"api_token":   storedProvider.APIToken,
		"server_name": server.Name,
		"volume_name": volumeName,
		"size_gb":     strconv.Itoa(payload.SizeGB),
		"location":    location,
		"state":       state,
		"automount":   automountValue,
	}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible volume management failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "manage_volume", "Volume playbook completed")

	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing volume workflow")

	if err := e.serverStore.UpdateStatus(ctx, server.ID, platform.ServerStatusReady); err != nil {
		e.logger.Error("failed to update server status to ready", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "finalize", "Volume workflow complete")

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) executeDeploySite(ctx context.Context, job *orchestrator.Job) error {
	if strings.TrimSpace(job.ServerID) == "" {
		return e.failJob(ctx, job, "server_id is required for site deployment job")
	}
	if e.siteStore == nil {
		return e.failJob(ctx, job, "site store not configured")
	}
	if e.domainStore == nil {
		return e.failJob(ctx, job, "domain store not configured")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}

	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{ToStatus: orchestrator.JobStatusRunning, CurrentStep: "validate"}); err != nil {
		return fmt.Errorf("transition to running: %w", err)
	}

	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobStarted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelInfo,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceSite,
		ParentResourceID:   e.siteIDForJob(*job),
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s started", orchestrator.JobKindLabel(job.Kind)),
	})

	payload, err := orchestrator.UnmarshalDeploySitePayload(job.Payload)
	if err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	site, err := e.siteStore.GetByID(ctx, payload.SiteID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("site not found: %v", err))
	}
	server, err := e.serverStore.GetByID(ctx, job.ServerID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("server not found: %v", err))
	}
	primaryDomain, err := e.primaryDomainForSite(ctx, site.ID)
	if err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	if strings.TrimSpace(site.PrimaryDomain) == "" {
		return e.failJob(ctx, job, "site primary hostname is required for deployment")
	}

	_ = e.siteStore.UpdateDeployment(ctx, site.ID, serverpkg.SiteDeploymentStateDeploying, fmt.Sprintf("Deploying WordPress to %s.", primaryDomain.Hostname), job.ID, "")
	_ = e.siteStore.UpdateRuntimeHealth(ctx, site.ID, serverpkg.SiteRuntimeHealthStatePending, "Deploy is running. Runtime health will be verified before the site is marked live.", "")
	_ = e.domainStore.UpdateRoutingStatus(ctx, primaryDomain.ID, serverpkg.DomainRoutingStatePending, "Applying server routing for this hostname.", time.Now().UTC())

	e.emitStepStart(ctx, job.ID, "validate", "Validating site deployment inputs")
	if strings.TrimSpace(server.ProfileKey) != "nginx-stack" {
		return e.failJob(ctx, job, fmt.Sprintf("server profile %q is not supported for site deployment", server.ProfileKey))
	}
	if server.Status != platform.ServerStatusReady {
		return e.failJob(ctx, job, "server must be ready before deploying a site")
	}
	if server.SetupState != platform.SetupStateReady {
		return e.failJob(ctx, job, "server setup must be ready before deploying a site")
	}
	storedKey, err := e.serverStore.GetKey(ctx, server.ID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to read SSH key: %v", err))
	}
	if storedKey == nil {
		return e.failJob(ctx, job, "missing SSH key for server")
	}
	decryptedKey, err := security.Decrypt(storedKey.PrivateKeyEncrypted)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to decrypt SSH key: %v", err))
	}
	e.emitStepComplete(ctx, job.ID, "validate", "Site deployment request validated")

	e.updateStep(ctx, job.ID, "deploy")
	e.emitStepStart(ctx, job.ID, "deploy", "Creating site files, database, and WordPress config")
	if err := e.runSiteDeployPlaybook(ctx, job.ID, server, site, primaryDomain, string(decryptedKey), payload.TLSContactEmail); err != nil {
		_ = e.domainStore.UpdateRoutingStatus(ctx, primaryDomain.ID, serverpkg.DomainRoutingStateIssue, err.Error(), time.Now().UTC())
		_ = e.siteStore.UpdateDeployment(ctx, site.ID, serverpkg.SiteDeploymentStateFailed, err.Error(), job.ID, "")
		_ = e.siteStore.UpdateRuntimeHealth(ctx, site.ID, serverpkg.SiteRuntimeHealthStateIssue, err.Error(), time.Now().UTC().Format(time.RFC3339))
		return e.failJob(ctx, job, fmt.Sprintf("site deploy failed: %v", err))
	}
	e.emitStepComplete(ctx, job.ID, "deploy", "Site files and WordPress installation applied")

	e.updateStep(ctx, job.ID, "verify")
	e.emitStepStart(ctx, job.ID, "verify", "Verifying deployed hostname and WordPress runtime")
	if err := e.verifySiteDeployment(ctx, *site, *primaryDomain); err != nil {
		_ = e.domainStore.UpdateRoutingStatus(ctx, primaryDomain.ID, serverpkg.DomainRoutingStateIssue, err.Error(), time.Now().UTC())
		_ = e.siteStore.UpdateDeployment(ctx, site.ID, serverpkg.SiteDeploymentStateFailed, err.Error(), job.ID, "")
		_ = e.siteStore.UpdateRuntimeHealth(ctx, site.ID, serverpkg.SiteRuntimeHealthStateIssue, err.Error(), time.Now().UTC().Format(time.RFC3339))
		return e.failJob(ctx, job, fmt.Sprintf("site verification failed: %v", err))
	}
	_ = e.domainStore.UpdateRoutingStatus(ctx, primaryDomain.ID, serverpkg.DomainRoutingStateReady, "Hostname routing verified over HTTPS.", time.Now().UTC())
	_ = e.siteStore.UpdateRuntimeHealth(ctx, site.ID, serverpkg.SiteRuntimeHealthStateHealthy, fmt.Sprintf("WordPress rendered successfully at https://%s/.", primaryDomain.Hostname), time.Now().UTC().Format(time.RFC3339))
	e.emitStepComplete(ctx, job.ID, "verify", "Hostname routing and WordPress runtime verified")

	now := time.Now().UTC().Format(time.RFC3339)
	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing site deployment")
	_ = e.siteStore.UpdateDeployment(ctx, site.ID, serverpkg.SiteDeploymentStateReady, fmt.Sprintf("Site is live at https://%s/.", primaryDomain.Hostname), job.ID, now)
	e.emitStepComplete(ctx, job.ID, "finalize", "Site deployment complete")
	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventSiteDeployed,
		Category:           activity.CategorySite,
		Level:              activity.LevelSuccess,
		ResourceType:       activity.ResourceSite,
		ResourceID:         site.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   site.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("Site '%s' deployed", site.Name),
		Message:            fmt.Sprintf("WordPress is live at https://%s/.", primaryDomain.Hostname),
	})
	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) primaryDomainForSite(ctx context.Context, siteID string) (*serverpkg.StoredDomain, error) {
	domains, err := e.domainStore.ListBySite(ctx, siteID)
	if err != nil {
		return nil, fmt.Errorf("list site domains: %w", err)
	}
	for i := range domains {
		if domains[i].IsPrimary {
			return &domains[i], nil
		}
	}
	if len(domains) == 0 {
		return nil, fmt.Errorf("site has no assigned hostname")
	}
	return &domains[0], nil
}

func (e *Executor) runSiteDeployPlaybook(ctx context.Context, jobID string, server *serverpkg.StoredServer, site *serverpkg.StoredSite, primaryDomain *serverpkg.StoredDomain, privateKey, tlsContactEmail string) error {
	if server == nil || site == nil || primaryDomain == nil {
		return fmt.Errorf("site deployment context is incomplete")
	}
	effectiveTLSContactEmail, err := e.resolveACMEContactEmail(strings.TrimSpace(tlsContactEmail), strings.TrimSpace(site.WordPressAdminEmail))
	if err != nil {
		return err
	}
	workspace, err := os.MkdirTemp("", "pressluft-site-deploy-")
	if err != nil {
		return fmt.Errorf("failed to create deploy workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	privateKeyPath := filepath.Join(workspace, "server.key")
	if err := os.WriteFile(privateKeyPath, []byte(privateKey), 0o600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}
	inventoryPath := filepath.Join(workspace, "deploy.ini")
	inventory := fmt.Sprintf("server ansible_host=%s ansible_user=root ansible_ssh_private_key_file=%s ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'\n", server.IPv4, privateKeyPath)
	if err := os.WriteFile(inventoryPath, []byte(inventory), 0o600); err != nil {
		return fmt.Errorf("failed to write deploy inventory: %w", err)
	}

	dbSuffix := strings.ReplaceAll(site.ID[:8], "-", "")
	dbName := "pl_" + dbSuffix
	dbUser := "pl_" + dbSuffix
	dbPassword, err := randomHex(24)
	if err != nil {
		return fmt.Errorf("generate database password: %w", err)
	}
	adminPassword, err := randomHex(24)
	if err != nil {
		return fmt.Errorf("generate admin password: %w", err)
	}
	secretKey, err := randomHex(32)
	if err != nil {
		return fmt.Errorf("generate secret key: %w", err)
	}
	deployPath := effectiveWordPressPath(*site)
	request := runner.Request{
		JobID:         jobID,
		InventoryPath: inventoryPath,
		PlaybookPath:  e.siteDeployPlaybook(),
		ExtraVars: map[string]string{
			"profile_key":       server.ProfileKey,
			"site_id":           site.ID,
			"site_name":         site.Name,
			"hostname":          primaryDomain.Hostname,
			"site_path":         deployPath,
			"php_version":       firstNonEmpty(site.PHPVersion, "8.3"),
			"wordpress_version": firstNonEmpty(site.WordPressVersion, "6.8"),
			"db_name":           dbName,
			"db_user":           dbUser,
			"db_password":       dbPassword,
			"admin_user":        "pressluft",
			"admin_password":    adminPassword,
			"admin_email":       firstNonEmpty(site.WordPressAdminEmail, fmt.Sprintf("admin@%s", primaryDomain.Hostname)),
			"tls_contact_email": effectiveTLSContactEmail,
			"secret_key":        secretKey,
		},
	}
	return e.runner.Run(ctx, request, &runnerEventSink{jobStore: e.jobStore, jobID: jobID, logger: e.logger})
}

func (e *Executor) resolveACMEContactEmail(operatorEmail, siteAdminEmail string) (string, error) {
	if isUsableACMEContactEmail(operatorEmail) {
		return operatorEmail, nil
	}
	if e.executionMode == platform.ExecutionModeDev && isUsableACMEContactEmail(siteAdminEmail) {
		return siteAdminEmail, nil
	}
	if e.executionMode == platform.ExecutionModeDev {
		return "", fmt.Errorf("no usable ACME contact email available: update the site WordPress admin email to a real address")
	}
	return "", fmt.Errorf("no usable ACME contact email available for certificate issuance")
}

func isUsableACMEContactEmail(email string) bool {
	address, err := mail.ParseAddress(strings.TrimSpace(email))
	if err != nil {
		return false
	}
	parts := strings.Split(address.Address, "@")
	if len(parts) != 2 {
		return false
	}
	domain := strings.ToLower(strings.TrimSpace(parts[1]))
	if domain == "" || !strings.Contains(domain, ".") {
		return false
	}
	if domain == "localhost" || domain == "localdomain" {
		return false
	}
	if domain == "example.com" || domain == "example.net" || domain == "example.org" {
		return false
	}
	if strings.HasSuffix(domain, ".example") || strings.HasSuffix(domain, ".invalid") || strings.HasSuffix(domain, ".localhost") || strings.HasSuffix(domain, ".test") {
		return false
	}
	return true
}

func (e *Executor) verifySiteDeployment(ctx context.Context, site serverpkg.StoredSite, primaryDomain serverpkg.StoredDomain) error {
	if err := serverpkg.VerifyPublicSiteRouting(ctx, site.ID, primaryDomain.Hostname); err != nil {
		return err
	}
	if err := serverpkg.VerifyPublicWordPressRuntime(ctx, primaryDomain.Hostname); err != nil {
		return err
	}
	return nil
}

func effectiveWordPressPath(site serverpkg.StoredSite) string {
	path := strings.TrimSpace(site.WordPressPath)
	if path == "" || path == "/srv/www/" {
		return filepath.ToSlash(filepath.Join("/srv/www/pressluft/sites", site.ID, "current"))
	}
	return path
}

func randomHex(length int) (string, error) {
	if length <= 0 {
		return "", nil
	}
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	const alphabet = "0123456789abcdef"
	out := make([]byte, length)
	for i := range out {
		out[i] = alphabet[int(buf[i])%len(alphabet)]
	}
	return string(out), nil
}

func firstNonEmpty(values ...string) string {
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value != "" {
			return value
		}
	}
	return ""
}

func (e *Executor) runConfigurePlaybook(ctx context.Context, jobID string, server *serverpkg.StoredServer, ipv4, privateKey string, storedKey *serverpkg.StoredServerKey) error {
	if server == nil {
		return fmt.Errorf("server is required")
	}
	if strings.TrimSpace(ipv4) == "" {
		return fmt.Errorf("server IPv4 is required")
	}
	if strings.TrimSpace(e.configurePath) == "" {
		return fmt.Errorf("configure playbook path not configured")
	}

	profile, err := validateSelectableProfile(server.ProfileKey)
	if err != nil {
		return err
	}

	if privateKey == "" {
		if storedKey == nil {
			var err error
			storedKey, err = e.serverStore.GetKey(ctx, server.ID)
			if err != nil {
				return fmt.Errorf("failed to read SSH key: %w", err)
			}
		}
		if storedKey == nil {
			return fmt.Errorf("missing SSH key for server")
		}
		decryptedKey, err := security.Decrypt(storedKey.PrivateKeyEncrypted)
		if err != nil {
			return fmt.Errorf("failed to decrypt SSH key: %w", err)
		}
		privateKey = string(decryptedKey)
	}

	workspace, err := os.MkdirTemp("", "pressluft-configure-")
	if err != nil {
		return fmt.Errorf("failed to create configure workspace: %w", err)
	}
	defer os.RemoveAll(workspace)

	privateKeyPath := filepath.Join(workspace, "server.key")
	if err := os.WriteFile(privateKeyPath, []byte(privateKey), 0o600); err != nil {
		return fmt.Errorf("failed to write private key: %w", err)
	}

	configureInventoryPath := filepath.Join(workspace, "configure.ini")
	configureInventory := fmt.Sprintf("server ansible_host=%s ansible_user=root ansible_ssh_private_key_file=%s ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'\n", ipv4, privateKeyPath)
	if err := os.WriteFile(configureInventoryPath, []byte(configureInventory), 0o600); err != nil {
		return fmt.Errorf("failed to write configure inventory: %w", err)
	}

	agentBinaryPath, err := filepath.Abs("bin/pressluft-agent")
	if err != nil {
		return fmt.Errorf("failed to resolve agent binary path: %w", err)
	}
	if _, err := os.Stat(agentBinaryPath); err != nil {
		return fmt.Errorf("agent binary not found at %q; ensure bin/pressluft-agent exists in the project root", agentBinaryPath)
	}

	extraVars := ConfigureContract{
		ServerID:           server.ID,
		ControlPlaneURL:    e.controlPlaneURL,
		ExecutionMode:      e.executionMode,
		ProfileKey:         profile.Key,
		ProfileArtifact:    profile.ArtifactPath,
		ProfileSupport:     profile.SupportLevel,
		ConfigureGuarantee: profile.ConfigureGuarantee,
		AgentBinaryPath:    agentBinaryPath,
	}.ExtraVars()

	devVars, err := e.extraAgentVars(ctx, server.ID)
	if err != nil {
		return fmt.Errorf("failed to prepare agent config: %w", err)
	}
	for key, value := range devVars {
		extraVars[key] = value
	}

	configureRequest := runner.Request{
		JobID:         jobID,
		InventoryPath: configureInventoryPath,
		PlaybookPath:  e.configurePath,
		ExtraVars:     extraVars,
	}

	return e.runner.Run(ctx, configureRequest, &runnerEventSink{jobStore: e.jobStore, jobID: jobID, logger: e.logger})
}

func validateSelectableProfile(profileKey string) (profiles.Profile, error) {
	profile, ok := profiles.Get(profileKey)
	if !ok {
		return profiles.Profile{}, fmt.Errorf("unknown profile key: %s", profileKey)
	}
	if profile.Selectable() {
		return profile, nil
	}

	reason := strings.TrimSpace(profile.SupportReason)
	if reason == "" {
		reason = "profile is not selectable in the current platform contract"
	}
	return profiles.Profile{}, fmt.Errorf("profile %q is %s: %s", profile.Key, profile.SupportLevel, reason)
}

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

type provisionArtifact struct {
	ID   int64  `json:"id"`
	IPv4 string `json:"ipv4"`
	IPv6 string `json:"ipv6"`
}

func readProvisionArtifact(path string) (provisionArtifact, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return provisionArtifact{}, err
	}

	var out provisionArtifact
	if err := json.Unmarshal(data, &out); err != nil {
		return provisionArtifact{}, err
	}
	return out, nil
}

func validateSSHPublicKey(publicKey string) error {
	fields := strings.Fields(publicKey)
	if len(fields) < 2 {
		return fmt.Errorf("expected OpenSSH public key format")
	}
	return nil
}

func (e *Executor) logSSHPublicKey(source, publicKey string) {
	if e.logger == nil || strings.TrimSpace(publicKey) == "" {
		return
	}
	fields := strings.Fields(publicKey)
	e.logger.Debug("ssh public key prepared", "source", source, "key_type", fields[0], "field_count", len(fields))
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
