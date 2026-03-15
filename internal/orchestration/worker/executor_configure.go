package worker

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	serverpkg "pressluft/internal/controlplane/server"
	"pressluft/internal/controlplane/server/profiles"
	"pressluft/internal/infra/runner"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/shared/security"

	"pressluft/internal/controlplane/activity"
)

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
