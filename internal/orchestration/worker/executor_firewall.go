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
