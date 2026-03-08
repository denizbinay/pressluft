package provider

import "context"

// ServerLocation is a provider region/location option.
type ServerLocation struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	Country     string `json:"country,omitempty"`
	City        string `json:"city,omitempty"`
	NetworkZone string `json:"network_zone,omitempty"`
}

// ServerTypePrice describes size pricing in a specific location.
type ServerTypePrice struct {
	LocationName string `json:"location_name"`
	HourlyGross  string `json:"hourly_gross"`
	MonthlyGross string `json:"monthly_gross"`
	Currency     string `json:"currency"`
}

// ServerTypeOption describes a provisionable size.
type ServerTypeOption struct {
	Name         string            `json:"name"`
	Description  string            `json:"description"`
	Cores        int               `json:"cores"`
	MemoryGB     float64           `json:"memory_gb"`
	DiskGB       int               `json:"disk_gb"`
	Architecture string            `json:"architecture"`
	AvailableAt  []string          `json:"available_at"`
	Prices       []ServerTypePrice `json:"prices"`
}

// ServerCatalog is the data required by the guided create-server UI.
// Images are intentionally omitted - they are defined by the server profile.
type ServerCatalog struct {
	Locations   []ServerLocation   `json:"locations"`
	ServerTypes []ServerTypeOption `json:"server_types"`
}

// ServerProvider is implemented by providers that expose a catalog of
// available server locations and types for the provisioning UI.
// Actual server lifecycle operations (create, delete, rebuild, resize) are
// handled by provider-specific Ansible playbooks under
// ops/ansible/playbooks/<provider-type>/.
type ServerProvider interface {
	Provider
	ListServerCatalog(ctx context.Context, token string) (*ServerCatalog, error)
}

// GetServerProvider returns a provider that supports server operations.
func GetServerProvider(providerType string) (ServerProvider, bool) {
	p := Get(providerType)
	if p == nil {
		return nil, false
	}
	sp, ok := p.(ServerProvider)
	return sp, ok
}
