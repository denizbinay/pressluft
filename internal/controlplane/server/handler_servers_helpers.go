package server

import (
	"fmt"
	"net/http"
	"strings"
	"time"

	"pressluft/internal/controlplane/apitypes"
	"pressluft/internal/infra/provider"
	"pressluft/internal/platform"
	"pressluft/internal/shared/ws"
)

func apiStoredServer(in StoredServer) apitypes.StoredServer {
	return apitypes.StoredServer{
		ID:               apitypes.FormatAppID(in.ID),
		ProviderID:       in.ProviderID,
		ProviderType:     in.ProviderType,
		ProviderServerID: in.ProviderServerID,
		IPv4:             in.IPv4,
		IPv6:             in.IPv6,
		Name:             in.Name,
		Location:         in.Location,
		ServerType:       in.ServerType,
		Image:            in.Image,
		ProfileKey:       in.ProfileKey,
		Status:           in.Status,
		SetupState:       in.SetupState,
		SetupLastError:   in.SetupLastError,
		ActionID:         in.ActionID,
		ActionStatus:     in.ActionStatus,
		HasKey:           in.HasKey,
		NodeStatus:       in.NodeStatus,
		NodeLastSeen:     in.NodeLastSeen,
		NodeVersion:      in.NodeVersion,
		CreatedAt:        in.CreatedAt,
		UpdatedAt:        in.UpdatedAt,
	}
}

func storedAgentInfo(server StoredServer) ws.AgentInfo {
	status := platform.NodeStatusUnknown
	if server.NodeStatus != platform.NodeStatus("") {
		status = server.NodeStatus
	}
	info := ws.AgentInfo{
		Connected: status == platform.NodeStatusOnline,
		Status:    status,
		Version:   server.NodeVersion,
	}
	if server.NodeLastSeen != "" {
		if parsed, err := time.Parse(time.RFC3339, server.NodeLastSeen); err == nil {
			info.LastSeen = parsed
		}
	}
	return info
}

func resolveServerArchitecture(serverType string, catalog *provider.ServerCatalog) (string, error) {
	if catalog == nil {
		return "", fmt.Errorf("provider catalog is empty")
	}
	for _, option := range catalog.ServerTypes {
		if option.Name == serverType {
			arch := strings.TrimSpace(option.Architecture)
			if arch == "" {
				return "", fmt.Errorf("architecture not available for server type %q", serverType)
			}
			return arch, nil
		}
	}
	return "", fmt.Errorf("server type %q not found in provider catalog", serverType)
}

func filterServerTypes(options []provider.ServerTypeOption, location, architecture string) []provider.ServerTypeOption {
	if len(options) == 0 {
		return nil
	}
	location = strings.TrimSpace(location)
	architecture = strings.TrimSpace(architecture)
	filtered := make([]provider.ServerTypeOption, 0, len(options))
	for _, option := range options {
		if option.Architecture != architecture {
			continue
		}
		if location != "" && !containsString(option.AvailableAt, location) {
			continue
		}
		filtered = append(filtered, option)
	}
	return filtered
}

func containsString(values []string, target string) bool {
	for _, value := range values {
		if value == target {
			return true
		}
	}
	return false
}

func parseProviderIDQuery(r *http.Request) (string, error) {
	raw := strings.TrimSpace(r.URL.Query().Get("provider_id"))
	if raw == "" {
		return "", fmt.Errorf("provider_id is required")
	}
	id, err := apitypes.ParseAppID(raw)
	if err != nil {
		return "", fmt.Errorf("provider_id must be a valid app id")
	}
	return id, nil
}

func validateCreateServerHTTPRequest(req apitypes.CreateServerRequest) error {
	if strings.TrimSpace(req.ProviderID) == "" {
		return fmt.Errorf("provider_id is required")
	}
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.Location) == "" {
		return fmt.Errorf("location is required")
	}
	if strings.TrimSpace(req.ServerType) == "" {
		return fmt.Errorf("server_type is required")
	}
	if strings.TrimSpace(req.ProfileKey) == "" {
		return fmt.Errorf("profile_key is required")
	}
	return nil
}
