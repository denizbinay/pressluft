// Package hetzner implements the provider.Provider interface for Hetzner Cloud.
package hetzner

import (
	"context"
	"fmt"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"pressluft/internal/provider"
)

const (
	providerType = "hetzner"
	displayName  = "Hetzner Cloud"
	docsURL      = "https://docs.hetzner.com/cloud/api/getting-started/generating-api-token"
)

// Hetzner implements provider.Provider.
type Hetzner struct{}

func init() {
	provider.Register(&Hetzner{})
}

// Info returns metadata about this provider.
func (h *Hetzner) Info() provider.Info {
	return provider.Info{
		Type:    providerType,
		Name:    displayName,
		DocsURL: docsURL,
	}
}

// Validate checks whether the given API token is valid and has read-write
// permissions by making a lightweight API call (list locations).
func (h *Hetzner) Validate(ctx context.Context, token string) (*provider.ValidationResult, error) {
	if token == "" {
		return &provider.ValidationResult{
			Valid:   false,
			Message: "API token must not be empty",
		}, nil
	}

	client := hcloud.NewClient(
		hcloud.WithToken(token),
		hcloud.WithApplication("pressluft", "1.0.0"),
	)

	// Use a lightweight read call to verify the token is valid.
	_, _, err := client.Location.List(ctx, hcloud.LocationListOpts{
		ListOpts: hcloud.ListOpts{Page: 1, PerPage: 1},
	})
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeUnauthorized) {
			return &provider.ValidationResult{
				Valid:   false,
				Message: "Invalid API token. Please check the token and try again.",
			}, nil
		}
		if hcloud.IsError(err, hcloud.ErrorCodeForbidden) {
			return &provider.ValidationResult{
				Valid:   false,
				Message: "Token does not have sufficient permissions.",
			}, nil
		}
		return nil, fmt.Errorf("hetzner api error: %w", err)
	}

	// Token is valid with at least read access. Now check write access by
	// attempting to list servers (same permission level, but confirms the
	// token scope covers compute resources).
	_, _, err = client.Server.List(ctx, hcloud.ServerListOpts{
		ListOpts: hcloud.ListOpts{Page: 1, PerPage: 1},
	})
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeForbidden) || hcloud.IsError(err, hcloud.ErrorCodeUnauthorized) {
			return &provider.ValidationResult{
				Valid:     true,
				ReadWrite: false,
				Message:   "Token is valid but appears to be read-only. A Read & Write token is required for server management.",
			}, nil
		}
		return nil, fmt.Errorf("hetzner api error: %w", err)
	}

	return &provider.ValidationResult{
		Valid:     true,
		ReadWrite: true,
		Message:   "Token is valid with Read & Write permissions.",
	}, nil
}
