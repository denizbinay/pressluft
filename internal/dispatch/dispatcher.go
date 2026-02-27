package dispatch

import (
	"context"

	"pressluft/internal/orchestrator"
	"pressluft/internal/runner"
	"pressluft/internal/ws"
)

type Dispatcher struct {
	ansibleRunner runner.Runner
	agentRunner   runner.Runner
	hub           *ws.Hub
}

func NewDispatcher(ansible, agent runner.Runner, hub *ws.Hub) *Dispatcher {
	return &Dispatcher{
		ansibleRunner: ansible,
		agentRunner:   agent,
		hub:           hub,
	}
}

func (d *Dispatcher) Dispatch(ctx context.Context, job orchestrator.Job) error {
	if d.isAgentJob(job.Kind) && d.isAgentConnected(job.ServerID) {
		return d.agentRunner.Run(ctx, job)
	}
	return d.ansibleRunner.Run(ctx, job)
}

func (d *Dispatcher) isAgentJob(kind string) bool {
	agentJobs := map[string]bool{
		"restart_service": true,
	}
	return agentJobs[kind]
}

func (d *Dispatcher) isAgentConnected(serverID int64) bool {
	_, ok := d.hub.Get(serverID)
	return ok
}
