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

// CreateServerRequest is the provider-agnostic server creation payload.
// ProfileKey is used by Pressluft orchestration and not forwarded directly
// unless needed by the provider adapter.
type CreateServerRequest struct {
	Name       string            `json:"name"`
	Location   string            `json:"location"`
	ServerType string            `json:"server_type"`
	Image      string            `json:"image"`
	UserData   string            `json:"user_data,omitempty"`
	Labels     map[string]string `json:"labels,omitempty"`
}

// CreateServerResult contains identifiers needed for asynchronous tracking.
type CreateServerResult struct {
	ProviderServerID string `json:"provider_server_id"`
	ActionID         string `json:"action_id,omitempty"`
	Status           string `json:"status"`
}

// SSHKeyResult contains the result of creating an SSH key at the provider.
type SSHKeyResult struct {
	ID          int64  `json:"id"`
	Name        string `json:"name"`
	Fingerprint string `json:"fingerprint"`
}

// ServerProvider is implemented by providers that support server lifecycle
// operations in addition to credential validation.
type ServerProvider interface {
	Provider
	ListServerCatalog(ctx context.Context, token string) (*ServerCatalog, error)
	CreateServer(ctx context.Context, token string, req CreateServerRequest) (*CreateServerResult, error)
	CreateSSHKey(ctx context.Context, token, name, publicKey string) (*SSHKeyResult, error)
	DeleteSSHKey(ctx context.Context, token string, keyID int64) error
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
