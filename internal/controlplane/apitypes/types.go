package apitypes

import (
	"fmt"
	"strings"

	"pressluft/internal/agent/agentcommand"
	"pressluft/internal/controlplane/auth"
	"pressluft/internal/controlplane/server/profiles"
	"pressluft/internal/infra/provider"
	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/shared/idutil"
	"pressluft/internal/shared/ws"
)

func FormatAppID(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return ""
	}
	normalized, err := idutil.Normalize(id)
	if err != nil {
		return id
	}
	return normalized
}

func ParseAppID(raw string) (string, error) {
	id, err := idutil.Normalize(raw)
	if err != nil {
		return "", fmt.Errorf("app id: %w", err)
	}
	return id, nil
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
	"CreateSiteRequest":       CreateSiteRequest{},
	"CreateDomainRequest":     CreateDomainRequest{},
	"ServerCatalogResponse":   ServerCatalogResponse{},
	"CreateServerResponse":    CreateServerResponse{},
	"StoredSite":              StoredSite{},
	"SiteHealthCheck":         agentcommand.SiteHealthCheck{},
	"SiteHealthSnapshot":      agentcommand.SiteHealthSnapshot{},
	"SiteHealthResponse":      SiteHealthResponse{},
	"StoredDomain":            StoredDomain{},
	"DeleteSiteResponse":      DeleteSiteResponse{},
	"DeleteDomainResponse":    DeleteDomainResponse{},
	"DeleteServerResponse":    DeleteServerResponse{},
	"UpdateSiteRequest":       UpdateSiteRequest{},
	"UpdateDomainRequest":     UpdateDomainRequest{},
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
	"Job":                     Job{},
	"JobEvent":                orchestrator.JobEvent{},
	"Activity":                Activity{},
	"ActivityListResponse":    ActivityListResponse{},
	"UnreadCountResponse":     UnreadCountResponse{},
}
