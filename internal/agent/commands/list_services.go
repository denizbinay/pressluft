package commands

import (
	"bufio"
	"bytes"
	"context"
	"encoding/json"
	"os/exec"
	"strings"

	"pressluft/internal/ws"
)

// Service represents a systemd service.
type Service struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	ActiveState string `json:"active_state"`
	LoadState   string `json:"load_state"`
}

// ListServicesResult is the response payload for list_services command.
type ListServicesResult struct {
	Services []Service `json:"services"`
}

// ListServices returns a list of running systemd services.
func ListServices(ctx context.Context, cmd ws.Command) ws.CommandResult {
	// Get running services using systemctl
	out, err := exec.CommandContext(ctx,
		"systemctl", "list-units",
		"--type=service",
		"--state=running",
		"--no-legend",
		"--no-pager",
	).Output()
	if err != nil {
		return ws.CommandResult{
			CommandID: cmd.ID,
			Success:   false,
			Error:     "failed to list services: " + err.Error(),
		}
	}

	services := parseSystemctlOutput(out)

	payload, _ := json.Marshal(ListServicesResult{Services: services})

	return ws.CommandResult{
		CommandID: cmd.ID,
		Success:   true,
		Output:    string(payload),
	}
}

// parseSystemctlOutput parses the output of systemctl list-units.
// Each line format: UNIT LOAD ACTIVE SUB DESCRIPTION
func parseSystemctlOutput(output []byte) []Service {
	var services []Service
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

		services = append(services, Service{
			Name:        name,
			Description: description,
			LoadState:   loadState,
			ActiveState: activeState,
		})
	}

	return services
}
