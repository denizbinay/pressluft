package worker

import "pressluft/internal/platform"

// ConfigureContract is the worker-to-configure-playbook boundary.
type ConfigureContract struct {
	ServerID           string
	ControlPlaneURL    string
	ExecutionMode      platform.ExecutionMode
	ProfileKey         string
	ProfileArtifact    string
	ProfileSupport     platform.SupportLevel
	ConfigureGuarantee string
	AgentBinaryPath    string
}

func (c ConfigureContract) ExtraVars() map[string]string {
	return map[string]string{
		"server_id":                   c.ServerID,
		"control_plane_url":           c.ControlPlaneURL,
		"pressluft_execution_mode":    string(c.ExecutionMode),
		"profile_key":                 c.ProfileKey,
		"profile_path":                c.ProfileArtifact,
		"profile_support_level":       string(c.ProfileSupport),
		"profile_configure_guarantee": c.ConfigureGuarantee,
		"agent_binary_path":           c.AgentBinaryPath,
	}
}
