package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pressluft/internal/activity"
	"pressluft/internal/orchestrator"
	"pressluft/internal/provider/hetzner"
	"pressluft/internal/runner"
	"pressluft/internal/security"
	"pressluft/internal/server/profiles"
)

// ServerStore defines the server persistence interface needed by the executor.
type ServerStore interface {
	GetByID(ctx context.Context, id int64) (*StoredServer, error)
	UpdateStatus(ctx context.Context, id int64, status string) error
	UpdateProvisioning(ctx context.Context, id int64, providerServerID, actionID, actionStatus, status, ipv4, ipv6 string) error
	UpdateServerType(ctx context.Context, id int64, serverType string) error
	GetKey(ctx context.Context, serverID int64) (*StoredServerKey, error)
	CreateKey(ctx context.Context, in CreateServerKeyInput) error
}

// StoredServer mirrors the server package type to avoid import cycles.
type StoredServer struct {
	ID               int64
	ProviderID       int64
	ProviderType     string
	ProviderServerID string
	IPv4             string
	IPv6             string
	Name             string
	Location         string
	ServerType       string
	Image            string
	ProfileKey       string
	Status           string
}

// StoredServerKey mirrors the server package key type to avoid import cycles.
type StoredServerKey struct {
	ServerID            int64
	PublicKey           string
	PrivateKeyEncrypted string
	EncryptionKeyID     string
	CreatedAt           string
	RotatedAt           string
}

// CreateServerKeyInput mirrors the server package input type to avoid import cycles.
type CreateServerKeyInput struct {
	ServerID            int64
	PublicKey           string
	PrivateKeyEncrypted string
	EncryptionKeyID     string
	RotatedAt           string
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
	jobStore        *orchestrator.Store
	serverStore     ServerStore
	providerStore   ProviderStore
	activityStore   *activity.Store
	runner          runner.Runner
	playbookPath    string
	configurePath   string
	deletePath      string
	rebuildPath     string
	resizePath      string
	firewallsPath   string
	volumePath      string
	controlPlaneURL string
	logger          *slog.Logger
}

// ExecutorConfig defines runner configuration.
type ExecutorConfig struct {
	ProvisionPlaybookPath string
	ConfigurePlaybookPath string
	DeletePlaybookPath    string
	RebuildPlaybookPath   string
	ResizePlaybookPath    string
	FirewallsPlaybookPath string
	VolumePlaybookPath    string
	ControlPlaneURL       string
}

// NewExecutor creates an executor with the given dependencies.
func NewExecutor(
	jobStore *orchestrator.Store,
	serverStore ServerStore,
	providerStore ProviderStore,
	activityStore *activity.Store,
	runner runner.Runner,
	config ExecutorConfig,
	logger *slog.Logger,
) *Executor {
	return &Executor{
		jobStore:        jobStore,
		serverStore:     serverStore,
		providerStore:   providerStore,
		activityStore:   activityStore,
		runner:          runner,
		playbookPath:    strings.TrimSpace(config.ProvisionPlaybookPath),
		configurePath:   strings.TrimSpace(config.ConfigurePlaybookPath),
		deletePath:      strings.TrimSpace(config.DeletePlaybookPath),
		rebuildPath:     strings.TrimSpace(config.RebuildPlaybookPath),
		resizePath:      strings.TrimSpace(config.ResizePlaybookPath),
		firewallsPath:   strings.TrimSpace(config.FirewallsPlaybookPath),
		volumePath:      strings.TrimSpace(config.VolumePlaybookPath),
		controlPlaneURL: strings.TrimSpace(config.ControlPlaneURL),
		logger:          logger,
	}
}

// Execute runs all steps for a job. It handles state transitions and event emission.
func (e *Executor) Execute(ctx context.Context, job *orchestrator.Job) error {
	switch job.Kind {
	case "provision_server":
		return e.executeProvisionServer(ctx, job)
	case "delete_server":
		return e.executeDeleteServer(ctx, job)
	case "rebuild_server":
		return e.executeRebuildServer(ctx, job)
	case "resize_server":
		return e.executeResizeServer(ctx, job)
	case "update_firewalls":
		return e.executeUpdateFirewalls(ctx, job)
	case "manage_volume":
		return e.executeManageVolume(ctx, job)
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
		Title:              fmt.Sprintf("%s started", jobKindLabel(job.Kind)),
	})

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

	if storedProvider.Type != "hetzner" {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible provisioning", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required for provisioning")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.playbookPath) == "" {
		return e.failJob(ctx, job, "provision playbook path not configured")
	}
	if strings.TrimSpace(e.configurePath) == "" {
		return e.failJob(ctx, job, "configure playbook path not configured")
	}

	e.emitStepComplete(ctx, job.ID, "validate", "Server configuration validated")

	// Step 2: Provision with Ansible
	e.updateStep(ctx, job.ID, "provision")
	e.emitStepStart(ctx, job.ID, "provision", "Running provision playbook")

	keyName := fmt.Sprintf("pressluft-server-%d", server.ID)
	storedKey, err := e.serverStore.GetKey(ctx, server.ID)
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to read SSH key: %v", err))
	}

	var publicKey string
	var privateKey string
	if storedKey != nil {
		publicKey = storedKey.PublicKey
		e.logSSHPublicKey("stored", publicKey)
	} else {
		publicKey, privateKey, err = hetzner.GenerateSSHKeyPair(keyName)
		if err != nil {
			return e.failJob(ctx, job, fmt.Sprintf("failed to generate SSH key: %v", err))
		}
		e.logSSHPublicKey("generated", publicKey)
		encryptedKey, keyID, err := security.Encrypt([]byte(privateKey))
		if err != nil {
			return e.failJob(ctx, job, fmt.Sprintf("failed to encrypt SSH key: %v", err))
		}
		if err := e.serverStore.CreateKey(ctx, CreateServerKeyInput{
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
			privateKey = ""
			e.logSSHPublicKey("stored_after_conflict", publicKey)
		} else {
			storedKey = &StoredServerKey{
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
		PlaybookPath:  e.playbookPath,
		ExtraVars: map[string]string{
			"hcloud_token":    storedProvider.APIToken,
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
	if err := e.serverStore.UpdateProvisioning(ctx, server.ID, providerServerID, "", "", "provisioning", result.IPv4, result.IPv6); err != nil {
		e.logger.Error("failed to update server provisioning state", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "provision", fmt.Sprintf("Server created: %s", providerServerID))

	// Step 3: Configure
	e.updateStep(ctx, job.ID, "configure")
	e.emitStepStart(ctx, job.ID, "configure", "Running configure playbook")

	profile, ok := profiles.Get(server.ProfileKey)
	if !ok {
		return e.failJob(ctx, job, fmt.Sprintf("unknown profile key: %s", server.ProfileKey))
	}

	if privateKey == "" {
		if storedKey == nil {
			storedKey, err = e.serverStore.GetKey(ctx, server.ID)
			if err != nil {
				return e.failJob(ctx, job, fmt.Sprintf("failed to read SSH key: %v", err))
			}
			if storedKey == nil {
				return e.failJob(ctx, job, "missing SSH key for server")
			}
		}
		decryptedKey, err := security.Decrypt(storedKey.PrivateKeyEncrypted)
		if err != nil {
			return e.failJob(ctx, job, fmt.Sprintf("failed to decrypt SSH key: %v", err))
		}
		privateKey = string(decryptedKey)
	}

	privateKeyPath := filepath.Join(workspace, "server.key")
	if err := os.WriteFile(privateKeyPath, []byte(privateKey), 0o600); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to write private key: %v", err))
	}

	configureInventoryPath := filepath.Join(workspace, "configure.ini")
	configureInventory := fmt.Sprintf("server ansible_host=%s ansible_user=root ansible_ssh_private_key_file=%s ansible_ssh_common_args='-o StrictHostKeyChecking=no -o UserKnownHostsFile=/dev/null'\n", result.IPv4, privateKeyPath)
	if err := os.WriteFile(configureInventoryPath, []byte(configureInventory), 0o600); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to write configure inventory: %v", err))
	}

	agentBinaryPath, err := filepath.Abs("bin/pressluft-agent")
	if err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("failed to resolve agent binary path: %v", err))
	}
	if _, err := os.Stat(agentBinaryPath); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("agent binary not found at %q; ensure bin/pressluft-agent exists in the project root", agentBinaryPath))
	}

	configureRequest := runner.Request{
		JobID:         job.ID,
		InventoryPath: configureInventoryPath,
		PlaybookPath:  e.configurePath,
		ExtraVars: map[string]string{
			"server_id":         strconv.FormatInt(server.ID, 10),
			"control_plane_url": e.controlPlaneURL,
			"profile_path":      profile.ArtifactPath,
			"agent_binary_path": agentBinaryPath,
		},
	}

	if err := e.runner.Run(ctx, configureRequest, &runnerEventSink{jobStore: e.jobStore, jobID: job.ID, logger: e.logger}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible configure failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "configure", "Configure playbook completed")

	// Step 4: Finalize
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

	// Emit job completed activity
	e.emitActivity(ctx, activity.EmitInput{
		EventType:          activity.EventJobCompleted,
		Category:           activity.CategoryJob,
		Level:              activity.LevelSuccess,
		ResourceType:       activity.ResourceJob,
		ResourceID:         job.ID,
		ParentResourceType: activity.ResourceServer,
		ParentResourceID:   job.ServerID,
		ActorType:          activity.ActorSystem,
		Title:              fmt.Sprintf("%s completed", jobKindLabel(job.Kind)),
	})

	// Emit server provisioned activity (special case for provision jobs)
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

type deleteServerPayload struct {
	ServerName string `json:"server_name"`
}

type rebuildServerPayload struct {
	ServerName  string `json:"server_name"`
	ServerImage string `json:"server_image"`
}

type resizeServerPayload struct {
	ServerType  string `json:"server_type"`
	UpgradeDisk *bool  `json:"upgrade_disk"`
}

type updateFirewallsPayload struct {
	Firewalls []string `json:"firewalls"`
}

type manageVolumePayload struct {
	VolumeName string `json:"volume_name"`
	SizeGB     int    `json:"size_gb"`
	Location   string `json:"location"`
	State      string `json:"state"`
	Automount  *bool  `json:"automount"`
}

func (e *Executor) executeDeleteServer(ctx context.Context, job *orchestrator.Job) error {
	if job.ServerID <= 0 {
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
		Title:              fmt.Sprintf("%s started", jobKindLabel(job.Kind)),
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
	if storedProvider.Type != "hetzner" {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible workflows", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.deletePath) == "" {
		return e.failJob(ctx, job, "delete playbook path not configured")
	}

	var payload deleteServerPayload
	if err := parseJobPayload(job, &payload); err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	serverName := strings.TrimSpace(payload.ServerName)
	if serverName == "" {
		serverName = strings.TrimSpace(server.Name)
	}
	if serverName == "" {
		return e.failJob(ctx, job, "server_name is required for delete_server job")
	}

	e.emitStepComplete(ctx, job.ID, "validate", "Delete request validated")

	e.updateStep(ctx, job.ID, "delete")
	e.emitStepStart(ctx, job.ID, "delete", "Running delete playbook")

	if err := e.runLocalPlaybook(ctx, job.ID, e.deletePath, map[string]string{
		"api_token":   storedProvider.APIToken,
		"server_name": serverName,
	}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible delete failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "delete", "Delete playbook completed")

	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing delete")

	if err := e.serverStore.UpdateStatus(ctx, server.ID, "ready"); err != nil {
		e.logger.Error("failed to update server status to ready", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "finalize", "Server delete complete")

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) executeRebuildServer(ctx context.Context, job *orchestrator.Job) error {
	if job.ServerID <= 0 {
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
		Title:              fmt.Sprintf("%s started", jobKindLabel(job.Kind)),
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
	if storedProvider.Type != "hetzner" {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible workflows", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.rebuildPath) == "" {
		return e.failJob(ctx, job, "rebuild playbook path not configured")
	}

	var payload rebuildServerPayload
	if err := parseJobPayload(job, &payload); err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	serverName := strings.TrimSpace(payload.ServerName)
	if serverName == "" {
		serverName = strings.TrimSpace(server.Name)
	}
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

	e.emitStepComplete(ctx, job.ID, "validate", "Rebuild request validated")

	e.updateStep(ctx, job.ID, "rebuild")
	e.emitStepStart(ctx, job.ID, "rebuild", "Running rebuild playbook")

	if err := e.runLocalPlaybook(ctx, job.ID, e.rebuildPath, map[string]string{
		"api_token":    storedProvider.APIToken,
		"server_name":  serverName,
		"server_image": serverImage,
	}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible rebuild failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "rebuild", "Rebuild playbook completed")

	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing rebuild")

	if err := e.serverStore.UpdateStatus(ctx, server.ID, "ready"); err != nil {
		e.logger.Error("failed to update server status to ready", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "finalize", "Server rebuild complete")

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) executeResizeServer(ctx context.Context, job *orchestrator.Job) error {
	if job.ServerID <= 0 {
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
		Title:              fmt.Sprintf("%s started", jobKindLabel(job.Kind)),
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
	if storedProvider.Type != "hetzner" {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible workflows", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.resizePath) == "" {
		return e.failJob(ctx, job, "resize playbook path not configured")
	}

	var payload resizeServerPayload
	if err := parseJobPayload(job, &payload); err != nil {
		return e.failJob(ctx, job, err.Error())
	}
	serverType := strings.TrimSpace(payload.ServerType)
	if serverType == "" {
		return e.failJob(ctx, job, "server_type is required for resize_server job")
	}
	if payload.UpgradeDisk == nil {
		return e.failJob(ctx, job, "upgrade_disk is required for resize_server job")
	}

	e.emitStepComplete(ctx, job.ID, "validate", "Resize request validated")

	e.updateStep(ctx, job.ID, "resize")
	e.emitStepStart(ctx, job.ID, "resize", "Running resize playbook")

	if err := e.runLocalPlaybook(ctx, job.ID, e.resizePath, map[string]string{
		"api_token":    storedProvider.APIToken,
		"server_name":  server.Name,
		"server_type":  serverType,
		"upgrade_disk": strconv.FormatBool(*payload.UpgradeDisk),
	}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible resize failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "resize", "Resize playbook completed")

	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing resize")

	if err := e.serverStore.UpdateServerType(ctx, server.ID, serverType); err != nil {
		e.logger.Error("failed to update server type", "error", err)
	}
	if err := e.serverStore.UpdateStatus(ctx, server.ID, "ready"); err != nil {
		e.logger.Error("failed to update server status to ready", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "finalize", "Server resize complete")

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) executeUpdateFirewalls(ctx context.Context, job *orchestrator.Job) error {
	if job.ServerID <= 0 {
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
		Title:              fmt.Sprintf("%s started", jobKindLabel(job.Kind)),
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
	if storedProvider.Type != "hetzner" {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible workflows", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.firewallsPath) == "" {
		return e.failJob(ctx, job, "firewalls playbook path not configured")
	}

	var payload updateFirewallsPayload
	if err := parseJobPayload(job, &payload); err != nil {
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

	if err := e.runLocalPlaybook(ctx, job.ID, e.firewallsPath, map[string]string{
		"api_token":     storedProvider.APIToken,
		"server_name":   server.Name,
		"firewalls_csv": firewallsCSV,
	}); err != nil {
		return e.failJob(ctx, job, fmt.Sprintf("ansible firewall update failed: %v", err))
	}

	e.emitStepComplete(ctx, job.ID, "update_firewalls", "Firewall update playbook completed")

	e.updateStep(ctx, job.ID, "finalize")
	e.emitStepStart(ctx, job.ID, "finalize", "Finalizing firewall update")

	if err := e.serverStore.UpdateStatus(ctx, server.ID, "ready"); err != nil {
		e.logger.Error("failed to update server status to ready", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "finalize", "Firewall update complete")

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) executeManageVolume(ctx context.Context, job *orchestrator.Job) error {
	if job.ServerID <= 0 {
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
		Title:              fmt.Sprintf("%s started", jobKindLabel(job.Kind)),
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
	if storedProvider.Type != "hetzner" {
		return e.failJob(ctx, job, fmt.Sprintf("provider %s does not support ansible workflows", storedProvider.Type))
	}
	if strings.TrimSpace(storedProvider.APIToken) == "" {
		return e.failJob(ctx, job, "provider token is required")
	}
	if e.runner == nil {
		return e.failJob(ctx, job, "ansible runner not configured")
	}
	if strings.TrimSpace(e.volumePath) == "" {
		return e.failJob(ctx, job, "volume playbook path not configured")
	}

	var payload manageVolumePayload
	if err := parseJobPayload(job, &payload); err != nil {
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

	if err := e.runLocalPlaybook(ctx, job.ID, e.volumePath, map[string]string{
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

	if err := e.serverStore.UpdateStatus(ctx, server.ID, "ready"); err != nil {
		e.logger.Error("failed to update server status to ready", "error", err)
	}

	e.emitStepComplete(ctx, job.ID, "finalize", "Volume workflow complete")

	return e.completeJob(ctx, job, "finalize")
}

func (e *Executor) runLocalPlaybook(ctx context.Context, jobID int64, playbookPath string, extraVars map[string]string) error {
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

func parseJobPayload(job *orchestrator.Job, target any) error {
	raw := strings.TrimSpace(job.Payload)
	if raw == "" {
		return nil
	}
	if err := json.Unmarshal([]byte(raw), target); err != nil {
		return fmt.Errorf("invalid job payload: %w", err)
	}
	return nil
}

func (e *Executor) completeJob(ctx context.Context, job *orchestrator.Job, step string) error {
	if _, err := e.jobStore.TransitionJob(ctx, job.ID, orchestrator.TransitionInput{
		ToStatus:    orchestrator.JobStatusSucceeded,
		CurrentStep: step,
	}); err != nil {
		return fmt.Errorf("transition to succeeded: %w", err)
	}

	e.emitEvent(ctx, job.ID, "info", "", "succeeded", "Job completed successfully")

	// Emit job completed activity
	input := activity.EmitInput{
		EventType:    activity.EventJobCompleted,
		Category:     activity.CategoryJob,
		Level:        activity.LevelSuccess,
		ResourceType: activity.ResourceJob,
		ResourceID:   job.ID,
		ActorType:    activity.ActorSystem,
		Title:        fmt.Sprintf("%s completed", jobKindLabel(job.Kind)),
	}
	if job.ServerID > 0 {
		input.ParentResourceType = activity.ResourceServer
		input.ParentResourceID = job.ServerID
	}
	e.emitActivity(ctx, input)

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

	// Emit job failed activity with requires_attention flag
	input := activity.EmitInput{
		EventType:         activity.EventJobFailed,
		Category:          activity.CategoryJob,
		Level:             activity.LevelError,
		ResourceType:      activity.ResourceJob,
		ResourceID:        job.ID,
		ActorType:         activity.ActorSystem,
		Title:             fmt.Sprintf("%s failed", jobKindLabel(job.Kind)),
		Message:           errMsg,
		RequiresAttention: true,
	}
	if job.ServerID > 0 {
		input.ParentResourceType = activity.ResourceServer
		input.ParentResourceID = job.ServerID
	}
	e.emitActivity(ctx, input)

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
	e.logger.Info("ssh public key", "source", source, "value", publicKey, "field_count", len(fields))
}

type runnerEventSink struct {
	jobStore *orchestrator.Store
	jobID    int64
	logger   *slog.Logger
}

func (s *runnerEventSink) Emit(ctx context.Context, event runner.Event) error {
	stepKey := strings.TrimSpace(event.StepKey)
	if stepKey == "" {
		stepKey = "ansible"
	}
	_, err := s.jobStore.AppendEvent(ctx, s.jobID, orchestrator.CreateEventInput{
		EventType: "step_update",
		Level:     event.Level,
		StepKey:   stepKey,
		Status:    event.Type,
		Message:   event.Message,
		Payload:   event.Payload,
	})
	if err != nil && s.logger != nil {
		s.logger.Error("failed to emit runner event", "job_id", s.jobID, "error", err)
	}
	return err
}

// emitActivity emits an activity event if the activity store is configured.
func (e *Executor) emitActivity(ctx context.Context, input activity.EmitInput) {
	if e.activityStore == nil {
		return
	}
	if _, err := e.activityStore.Emit(ctx, input); err != nil {
		e.logger.Error("failed to emit activity", "event_type", input.EventType, "error", err)
	}
}

// jobKindLabel returns a human-readable label for a job kind.
func jobKindLabel(kind string) string {
	labels := map[string]string{
		"provision_server": "Server provisioning",
		"delete_server":    "Server deletion",
		"rebuild_server":   "Server rebuild",
		"resize_server":    "Server resize",
		"update_firewalls": "Firewall update",
		"manage_volume":    "Volume management",
	}
	if label, ok := labels[kind]; ok {
		return label
	}
	return kind
}
