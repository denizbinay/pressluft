package agent

import (
	"context"
	"encoding/json"
	"os/exec"

	"pressluft/internal/ws"
)

type Executor struct{}

func NewExecutor() *Executor {
	return &Executor{}
}

func (e *Executor) Execute(ctx context.Context, cmd ws.Command) ws.CommandResult {
	switch cmd.Type {
	case "restart_service":
		return e.restartService(ctx, cmd)
	default:
		return ws.CommandResult{CommandID: cmd.ID, Success: false, Error: "unknown command"}
	}
}

func (e *Executor) restartService(ctx context.Context, cmd ws.Command) ws.CommandResult {
	var params struct {
		ServiceName string `json:"service_name"`
	}
	if err := json.Unmarshal(cmd.Payload, &params); err != nil {
		return ws.CommandResult{CommandID: cmd.ID, Success: false, Error: "invalid payload"}
	}

	out, err := exec.CommandContext(ctx, "systemctl", "restart", params.ServiceName).CombinedOutput()
	if err != nil {
		return ws.CommandResult{CommandID: cmd.ID, Success: false, Error: err.Error(), Output: string(out)}
	}

	return ws.CommandResult{CommandID: cmd.ID, Success: true, Output: string(out)}
}
