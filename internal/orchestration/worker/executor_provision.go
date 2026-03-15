package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strconv"
	"strings"

	"pressluft/internal/controlplane/activity"
	serverpkg "pressluft/internal/controlplane/server"
	"pressluft/internal/infra/provider"
	"pressluft/internal/infra/runner"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/shared/security"
	"pressluft/internal/shared/sshutil"
)

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
