// Package provider defines the interface that every cloud provider must
// implement and a registry for looking them up by type key.
package provider

import (
	"context"
	"fmt"
)

// ValidationResult is returned after a token is validated against the
// provider's API. It tells the caller whether the token is valid, what
// permission level it has, and an optional human-readable message.
type ValidationResult struct {
	Valid       bool   `json:"valid"`
	ReadWrite   bool   `json:"read_write"`
	Message     string `json:"message"`
	ProjectName string `json:"project_name,omitempty"`
}

// Info describes a provider type for the frontend (display name, docs URL, etc.).
type Info struct {
	Type    string `json:"type"`
	Name    string `json:"name"`
	DocsURL string `json:"docs_url"`
}

// Provider is the interface every cloud provider adapter must satisfy.
// Validate checks whether the given API token is valid and has the
// required permissions.
type Provider interface {
	Info() Info
	Validate(ctx context.Context, token string) (*ValidationResult, error)
}

// registry holds all registered provider implementations keyed by type.
var registry = map[string]Provider{}

// Register adds a provider to the global registry. It panics on duplicate keys.
func Register(p Provider) {
	key := p.Info().Type
	if _, exists := registry[key]; exists {
		panic(fmt.Sprintf("provider %q already registered", key))
	}
	registry[key] = p
}

// Get returns the provider for the given type key, or nil if not found.
func Get(providerType string) Provider {
	return registry[providerType]
}

// All returns Info for every registered provider.
func All() []Info {
	out := make([]Info, 0, len(registry))
	for _, p := range registry {
		out = append(out, p.Info())
	}
	return out
}
