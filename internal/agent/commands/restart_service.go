package commands

import (
	"context"
	"encoding/json"
	"os/exec"
	"regexp"

	"pressluft/internal/ws"
)

type RestartServiceParams struct {
	ServiceName string `json:"service_name"`
}

func RestartService(ctx context.Context, cmd ws.Command) ws.CommandResult {
	var params RestartServiceParams
	if err := json.Unmarshal(cmd.Payload, &params); err != nil {
		return ws.CommandResult{CommandID: cmd.ID, Success: false, Error: "invalid payload"}
	}

	if !isValidServiceName(params.ServiceName) {
		return ws.CommandResult{CommandID: cmd.ID, Success: false, Error: "invalid service name"}
	}

	out, err := exec.CommandContext(ctx, "systemctl", "restart", params.ServiceName).CombinedOutput()
	if err != nil {
		return ws.CommandResult{CommandID: cmd.ID, Success: false, Error: err.Error(), Output: string(out)}
	}

	return ws.CommandResult{CommandID: cmd.ID, Success: true, Output: string(out)}
}

func isValidServiceName(name string) bool {
	matched, _ := regexp.MatchString(`^[a-zA-Z0-9._-]+$`, name)
	return matched && len(name) > 0 && len(name) < 256
}
