package apitypes

import (
	"fmt"
	"strings"

	"pressluft/internal/infra/provider"
)

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
	ID         string                    `json:"id"`
	Validation provider.ValidationResult `json:"validation"`
}
