package apitypes

import (
	"encoding/json"
	"fmt"
	"strings"

	"pressluft/internal/activity"
	"pressluft/internal/agentcommand"
	"pressluft/internal/auth"
	"pressluft/internal/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/provider"
	"pressluft/internal/server/profiles"
	"pressluft/internal/ws"
)

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

func (r *LoginRequest) Validate() error {
	r.Email = strings.TrimSpace(r.Email)
	if r.Email == "" || strings.TrimSpace(r.Password) == "" {
		return fmt.Errorf("email and password are required")
	}
	return nil
}

type StatusResponse struct {
	Status string `json:"status"`
}

type HealthResponse struct {
	Status             string                   `json:"status"`
	CallbackURLMode    platform.CallbackURLMode `json:"callback_url_mode,omitempty"`
	CallbackURLWarning string                   `json:"callback_url_warning,omitempty"`
}

type CreateProviderRequest struct {
	Type     string `json:"type"`
	Name     string `json:"name"`
	APIToken string `json:"api_token"`
}

func (r *CreateProviderRequest) Validate() error {
	r.Type = strings.TrimSpace(r.Type)
	r.Name = strings.TrimSpace(r.Name)
	if r.Type == "" || r.Name == "" || strings.TrimSpace(r.APIToken) == "" {
		return fmt.Errorf("type, name, and api_token are required")
	}
	return nil
}

type ValidateProviderRequest struct {
	Type     string `json:"type"`
	APIToken string `json:"api_token"`
}

func (r *ValidateProviderRequest) Validate() error {
	r.Type = strings.TrimSpace(r.Type)
	if r.Type == "" || strings.TrimSpace(r.APIToken) == "" {
		return fmt.Errorf("type and api_token are required")
	}
	return nil
}

type CreateProviderResponse struct {
	ID         int64                     `json:"id"`
	Validation provider.ValidationResult `json:"validation"`
}

type CreateServerRequest struct {
	ProviderID int64  `json:"provider_id"`
	Name       string `json:"name"`
	Location   string `json:"location"`
	ServerType string `json:"server_type"`
	ProfileKey string `json:"profile_key"`
}

type ServerCatalogResponse struct {
	Catalog  provider.ServerCatalog `json:"catalog"`
	Profiles []profiles.Profile     `json:"profiles"`
}

type CreateServerResponse struct {
	ServerID int64                 `json:"server_id"`
	JobID    int64                 `json:"job_id"`
	Status   platform.ServerStatus `json:"status"`
}

type StoredServer struct {
	ID               int64                 `json:"id"`
	ProviderID       int64                 `json:"provider_id"`
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
	ServerID    int64                  `json:"server_id"`
	JobID       int64                  `json:"job_id"`
	Status      platform.ServerStatus  `json:"status"`
	JobStatus   orchestrator.JobStatus `json:"job_status"`
	Async       bool                   `json:"async"`
	Description string                 `json:"description"`
}

type RebuildOptionsResponse struct {
	ServerID     int64                        `json:"server_id"`
	ServerType   string                       `json:"server_type"`
	Architecture string                       `json:"architecture"`
	Images       []provider.ServerImageOption `json:"images"`
}

type ResizeOptionsResponse struct {
	ServerID     int64                       `json:"server_id"`
	Location     string                      `json:"location"`
	ServerType   string                      `json:"server_type"`
	Architecture string                      `json:"architecture"`
	ServerTypes  []provider.ServerTypeOption `json:"server_types"`
}

type FirewallsResponse struct {
	ServerID  int64                     `json:"server_id"`
	Firewalls []provider.FirewallOption `json:"firewalls"`
}

type VolumesResponse struct {
	ServerID int64                   `json:"server_id"`
	Volumes  []provider.VolumeOption `json:"volumes"`
}

type AgentStatusMapResponse map[int64]ws.AgentInfo

type ServicesResponse struct {
	ServerID       int64                  `json:"server_id"`
	AgentConnected bool                   `json:"agent_connected"`
	Services       []agentcommand.Service `json:"services"`
}

type CreateJobRequest struct {
	Kind     string          `json:"kind"`
	ServerID int64           `json:"server_id"`
	Payload  json.RawMessage `json:"payload"`
}

func (r *CreateJobRequest) Validate() error {
	r.Kind = strings.TrimSpace(r.Kind)
	if r.Kind == "" {
		return fmt.Errorf("kind is required")
	}
	return nil
}

type ActivityListResponse struct {
	Data       []activity.Activity `json:"data"`
	NextCursor string              `json:"next_cursor,omitempty"`
}

type UnreadCountResponse struct {
	Count int64 `json:"count"`
}

var PublishedTypes = map[string]any{
	"LoginRequest":            LoginRequest{},
	"StatusResponse":          StatusResponse{},
	"HealthResponse":          HealthResponse{},
	"CreateProviderRequest":   CreateProviderRequest{},
	"ValidateProviderRequest": ValidateProviderRequest{},
	"CreateProviderResponse":  CreateProviderResponse{},
	"ProviderType":            provider.Info{},
	"StoredProvider":          provider.StoredProvider{},
	"ValidationResult":        provider.ValidationResult{},
	"CreateServerRequest":     CreateServerRequest{},
	"ServerCatalogResponse":   ServerCatalogResponse{},
	"CreateServerResponse":    CreateServerResponse{},
	"DeleteServerResponse":    DeleteServerResponse{},
	"RebuildOptionsResponse":  RebuildOptionsResponse{},
	"ResizeOptionsResponse":   ResizeOptionsResponse{},
	"FirewallsResponse":       FirewallsResponse{},
	"VolumesResponse":         VolumesResponse{},
	"ServerProfile":           profiles.Profile{},
	"ServerCatalog":           provider.ServerCatalog{},
	"ServerLocation":          provider.ServerLocation{},
	"ServerTypePrice":         provider.ServerTypePrice{},
	"ServerTypeOption":        provider.ServerTypeOption{},
	"StoredServer":            StoredServer{},
	"AgentInfo":               ws.AgentInfo{},
	"AgentStatusMapResponse":  AgentStatusMapResponse{},
	"Service":                 agentcommand.Service{},
	"ServicesResponse":        ServicesResponse{},
	"AuthActor":               auth.Actor{},
	"CreateJobRequest":        CreateJobRequest{},
	"Job":                     orchestrator.Job{},
	"JobEvent":                orchestrator.JobEvent{},
	"Activity":                activity.Activity{},
	"ActivityListResponse":    ActivityListResponse{},
	"UnreadCountResponse":     UnreadCountResponse{},
}
