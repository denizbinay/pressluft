//go:build !dev

package worker

import (
	"context"
	"fmt"
	"net/url"
	"strings"
	"time"

	"pressluft/internal/platform"
)

func (e *Executor) extraAgentVars(ctx context.Context, serverID int64) (map[string]string, error) {
	if e.executionMode != platform.ExecutionModeProductionBootstrap {
		return nil, fmt.Errorf("agent bootstrap requires %q control-plane mode; current mode is %q", platform.ExecutionModeProductionBootstrap, e.executionMode)
	}
	if e.registrationStore == nil {
		return nil, fmt.Errorf("registration token store not configured")
	}
	if err := validateProductionControlPlaneURL(e.controlPlaneURL); err != nil {
		return nil, err
	}
	token, err := e.registrationStore.Create(serverID, 30*time.Minute)
	if err != nil {
		return nil, fmt.Errorf("create registration token: %w", err)
	}
	return map[string]string{
		"agent_registration_token": token,
	}, nil
}

func validateProductionControlPlaneURL(raw string) error {
	parsed, err := url.Parse(strings.TrimSpace(raw))
	if err != nil {
		return fmt.Errorf("parse control plane URL: %w", err)
	}
	if parsed.Scheme != "https" || parsed.Host == "" {
		return fmt.Errorf("production bootstrap requires PRESSLUFT_CONTROL_PLANE_URL with an https URL")
	}
	return nil
}
