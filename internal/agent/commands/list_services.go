package commands

import (
	"bufio"
	"bytes"
	"context"
	"strings"

	"pressluft/internal/agentcommand"
	"pressluft/internal/ws"
)

// ListServices returns a list of running systemd services.
func ListServices(ctx context.Context, cmd ws.Command) ws.CommandResult {
	// Get running services using systemctl
	out, err := commandContext(ctx,
		"systemctl", "list-units",
		"--type=service",
		"--state=running",
		"--no-legend",
		"--no-pager",
	).Output()
	if err != nil {
		return ws.FailureResult(cmd.ID, agentcommand.ErrorCodeExecutionFailed, "failed to list services", nil, err.Error())
	}

	services := parseSystemctlOutput(out)
	return ws.SuccessResult(cmd.ID, agentcommand.ListServicesResult{Services: services}, "")
}

// parseSystemctlOutput parses the output of systemctl list-units.
// Each line format: UNIT LOAD ACTIVE SUB DESCRIPTION
func parseSystemctlOutput(output []byte) []agentcommand.Service {
	var services []agentcommand.Service
	scanner := bufio.NewScanner(bytes.NewReader(output))

	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" {
			continue
		}

		// Split into fields: UNIT LOAD ACTIVE SUB DESCRIPTION...
		fields := strings.Fields(line)
		if len(fields) < 5 {
			continue
		}

		unit := fields[0]
		loadState := fields[1]
		activeState := fields[2]
		// SUB is fields[3], we skip it
		description := strings.Join(fields[4:], " ")

		// Extract service name (remove .service suffix)
		name := strings.TrimSuffix(unit, ".service")

		services = append(services, agentcommand.Service{
			Name:        name,
			Description: description,
			LoadState:   loadState,
			ActiveState: activeState,
		})
	}

	return services
}
