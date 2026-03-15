package worker

import (
	"context"
	"fmt"
	"strings"

	"pressluft/internal/controlplane/activity"
	"pressluft/internal/infra/provider"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/platform"
)

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
