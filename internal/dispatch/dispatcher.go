package dispatch

import (
	"context"

	"pressluft/internal/orchestrator"
	"pressluft/internal/ws"
)

// JobRunner executes jobs using the agent protocol.
type JobRunner interface {
	Run(ctx context.Context, job orchestrator.Job) error
}

type Dispatcher struct {
	agentRunner *AgentRunner
	hub         *ws.Hub
}

func NewDispatcher(agentRunner *AgentRunner, hub *ws.Hub) *Dispatcher {
	return &Dispatcher{
		agentRunner: agentRunner,
		hub:         hub,
	}
}

// Dispatch sends a job to the appropriate runner.
// Currently only supports agent jobs; Ansible jobs are handled by the worker.
func (d *Dispatcher) Dispatch(ctx context.Context, job orchestrator.Job) error {
	if d.isAgentJob(job.Kind) && d.isAgentConnected(job.ServerID) {
		return d.agentRunner.Run(ctx, job)
	}
	// Non-agent jobs should not go through the dispatcher
	return nil
}

// CanHandleJob returns true if this job can be handled by the agent.
func (d *Dispatcher) CanHandleJob(job orchestrator.Job) bool {
	return d.isAgentJob(job.Kind) && d.isAgentConnected(job.ServerID)
}

func (d *Dispatcher) isAgentJob(kind string) bool {
	agentJobs := map[string]bool{
		"restart_service": true,
	}
	return agentJobs[kind]
}

func (d *Dispatcher) isAgentConnected(serverID int64) bool {
	if d.hub == nil {
		return false
	}
	_, ok := d.hub.Get(serverID)
	return ok
}
