package apitypes

import (
	"pressluft/internal/agent/agentcommand"
	"pressluft/internal/controlplane/server/profiles"
	"pressluft/internal/infra/provider"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/shared/ws"
)

type CreateServerRequest struct {
	ProviderID string `json:"provider_id"`
	Name       string `json:"name"`
	Location   string `json:"location"`
	ServerType string `json:"server_type"`
	ProfileKey string `json:"profile_key"`
}

type CreateServerResponse struct {
	ServerID string                `json:"server_id"`
	JobID    string                `json:"job_id"`
	Status   platform.ServerStatus `json:"status"`
}

type StoredServer struct {
	ID               string                `json:"id"`
	ProviderID       string                `json:"provider_id"`
	ProviderType     string                `json:"provider_type"`
	ProviderServerID string                `json:"provider_server_id,omitempty"`
	IPv4             string                `json:"ipv4,omitempty"`
	IPv6             string                `json:"ipv6,omitempty"`
	Name             string                `json:"name"`
	Location         string                `json:"location"`
	ServerType       string                `json:"server_type"`
	Image            string                `json:"image"`
	ProfileKey       string                `json:"profile_key"`
	Status           platform.ServerStatus `json:"status"`
	SetupState       platform.SetupState   `json:"setup_state"`
	SetupLastError   string                `json:"setup_last_error,omitempty"`
	ActionID         string                `json:"action_id,omitempty"`
	ActionStatus     string                `json:"action_status,omitempty"`
	HasKey           bool                  `json:"has_key"`
	NodeStatus       platform.NodeStatus   `json:"node_status,omitempty"`
	NodeLastSeen     string                `json:"node_last_seen,omitempty"`
	NodeVersion      string                `json:"node_version,omitempty"`
	CreatedAt        string                `json:"created_at"`
	UpdatedAt        string                `json:"updated_at"`
}

type DeleteServerResponse struct {
	ServerID    string                 `json:"server_id"`
	JobID       string                 `json:"job_id"`
	Status      platform.ServerStatus  `json:"status"`
	JobStatus   orchestrator.JobStatus `json:"job_status"`
	Async       bool                   `json:"async"`
	Description string                 `json:"description"`
}

type ServerCatalogResponse struct {
	Catalog  provider.ServerCatalog `json:"catalog"`
	Profiles []profiles.Profile     `json:"profiles"`
}

type RebuildOptionsResponse struct {
	ServerID     string                       `json:"server_id"`
	ServerType   string                       `json:"server_type"`
	Architecture string                       `json:"architecture"`
	Images       []provider.ServerImageOption `json:"images"`
}

type ResizeOptionsResponse struct {
	ServerID     string                      `json:"server_id"`
	Location     string                      `json:"location"`
	ServerType   string                      `json:"server_type"`
	Architecture string                      `json:"architecture"`
	ServerTypes  []provider.ServerTypeOption `json:"server_types"`
}

type FirewallsResponse struct {
	ServerID  string                    `json:"server_id"`
	Firewalls []provider.FirewallOption `json:"firewalls"`
}

type VolumesResponse struct {
	ServerID string                  `json:"server_id"`
	Volumes  []provider.VolumeOption `json:"volumes"`
}

type AgentStatusMapResponse map[string]ws.AgentInfo

type ServicesResponse struct {
	ServerID       string                 `json:"server_id"`
	AgentConnected bool                   `json:"agent_connected"`
	Services       []agentcommand.Service `json:"services"`
}
