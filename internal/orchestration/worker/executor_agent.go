package worker

import (
	"context"
	"fmt"
	"strings"

	"pressluft/internal/controlplane/activity"
	"pressluft/internal/orchestration/orchestrator"
)

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
