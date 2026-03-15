package apitypes

import (
	"fmt"
	"strings"

	"pressluft/internal/platform"
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
