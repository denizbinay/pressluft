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
