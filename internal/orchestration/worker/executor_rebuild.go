package worker

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"pressluft/internal/controlplane/activity"
	"pressluft/internal/infra/provider"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/platform"
)

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
