package provider

import "context"

// ServerImageOption describes an image usable for rebuild actions.
type ServerImageOption struct {
	ID           int64  `json:"id"`
	Name         string `json:"name"`
	Type         string `json:"type"`
	Architecture string `json:"architecture"`
	Deprecated   bool   `json:"deprecated"`
	Status       string `json:"status"`
}

// FirewallOption describes a firewall that can be attached to servers.
type FirewallOption struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
}

// VolumeOption describes a storage volume available to attach or manage.
type VolumeOption struct {
	ID       int64  `json:"id"`
	Name     string `json:"name"`
	SizeGB   int    `json:"size_gb"`
	Location string `json:"location"`
	Status   string `json:"status"`
	ServerID int64  `json:"server_id,omitempty"`
}

// ServerImageProvider lists images available for rebuild operations.
type ServerImageProvider interface {
	Provider
	ListServerImages(ctx context.Context, token, architecture string) ([]ServerImageOption, error)
}

// FirewallProvider lists firewalls available for server actions.
type FirewallProvider interface {
	Provider
	ListFirewalls(ctx context.Context, token string) ([]FirewallOption, error)
}

// VolumeProvider lists volumes available for server actions.
type VolumeProvider interface {
	Provider
	ListVolumes(ctx context.Context, token string) ([]VolumeOption, error)
}

// GetServerImageProvider returns a provider that supports image listing.
func GetServerImageProvider(providerType string) (ServerImageProvider, bool) {
	p := Get(providerType)
	if p == nil {
		return nil, false
	}
	sp, ok := p.(ServerImageProvider)
	return sp, ok
}

// GetFirewallProvider returns a provider that supports firewall listing.
func GetFirewallProvider(providerType string) (FirewallProvider, bool) {
	p := Get(providerType)
	if p == nil {
		return nil, false
	}
	fp, ok := p.(FirewallProvider)
	return fp, ok
}

// GetVolumeProvider returns a provider that supports volume listing.
func GetVolumeProvider(providerType string) (VolumeProvider, bool) {
	p := Get(providerType)
	if p == nil {
		return nil, false
	}
	vp, ok := p.(VolumeProvider)
	return vp, ok
}
