package worker

import (
	"context"
	"fmt"
	"strconv"
	"strings"

	"pressluft/internal/controlplane/activity"
	"pressluft/internal/infra/provider"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/platform"
)

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
