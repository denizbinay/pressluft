package hetzner

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"pressluft/internal/provider"
)

// ListServerImages returns images compatible with the given architecture.
func (h *Hetzner) ListServerImages(ctx context.Context, token, architecture string) ([]provider.ServerImageOption, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("api token must not be empty")
	}
	arch := strings.TrimSpace(architecture)
	if arch == "" {
		return nil, fmt.Errorf("architecture is required")
	}

	client := newClient(token)
	images, err := client.Image.AllWithOpts(ctx, hcloud.ImageListOpts{
		ListOpts:          hcloud.ListOpts{PerPage: 50},
		Architecture:      []hcloud.Architecture{hcloud.Architecture(arch)},
		IncludeDeprecated: true,
	})
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}

	options := make([]provider.ServerImageOption, 0, len(images))
	for _, image := range images {
		if image == nil || image.IsDeleted() {
			continue
		}
		options = append(options, provider.ServerImageOption{
			ID:           image.ID,
			Name:         image.Name,
			Type:         string(image.Type),
			Architecture: string(image.Architecture),
			Deprecated:   image.IsDeprecated(),
			Status:       string(image.Status),
		})
	}

	slices.SortFunc(options, func(a, b provider.ServerImageOption) int {
		if cmp := strings.Compare(a.Name, b.Name); cmp != 0 {
			return cmp
		}
		if a.ID < b.ID {
			return -1
		}
		if a.ID > b.ID {
			return 1
		}
		return 0
	})

	return options, nil
}

// ListFirewalls returns all firewalls in the Hetzner project.
func (h *Hetzner) ListFirewalls(ctx context.Context, token string) ([]provider.FirewallOption, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("api token must not be empty")
	}

	client := newClient(token)
	firewalls, err := client.Firewall.All(ctx)
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}

	options := make([]provider.FirewallOption, 0, len(firewalls))
	for _, fw := range firewalls {
		if fw == nil {
			continue
		}
		options = append(options, provider.FirewallOption{
			ID:   fw.ID,
			Name: fw.Name,
		})
	}

	slices.SortFunc(options, func(a, b provider.FirewallOption) int {
		return strings.Compare(a.Name, b.Name)
	})

	return options, nil
}

// ListVolumes returns all volumes in the Hetzner project.
func (h *Hetzner) ListVolumes(ctx context.Context, token string) ([]provider.VolumeOption, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("api token must not be empty")
	}

	client := newClient(token)
	volumes, err := client.Volume.All(ctx)
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}

	options := make([]provider.VolumeOption, 0, len(volumes))
	for _, vol := range volumes {
		if vol == nil {
			continue
		}
		opt := provider.VolumeOption{
			ID:     vol.ID,
			Name:   vol.Name,
			SizeGB: vol.Size,
			Status: string(vol.Status),
		}
		if vol.Location != nil {
			opt.Location = vol.Location.Name
		}
		if vol.Server != nil {
			opt.ServerID = vol.Server.ID
		}
		options = append(options, opt)
	}

	slices.SortFunc(options, func(a, b provider.VolumeOption) int {
		return strings.Compare(a.Name, b.Name)
	})

	return options, nil
}
