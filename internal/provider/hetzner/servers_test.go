package hetzner

import (
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"pressluft/internal/provider"
)

func TestValidateCreateServerRequest(t *testing.T) {
	err := validateCreateServerRequest(provider.CreateServerRequest{})
	if err == nil {
		t.Fatal("expected validation error, got nil")
	}

	err = validateCreateServerRequest(provider.CreateServerRequest{
		Name:       "agency-prod-01",
		Location:   "fsn1",
		ServerType: "cx22",
		Image:      "ubuntu-24.04",
	})
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

// TestMapHetznerAPIErrorMapping is kept for error mapping coverage.
// The mapServerTypes function was removed as availability is now derived from datacenters.

func TestMapHetznerAPIErrorMapping(t *testing.T) {
	err := mapHetznerAPIError(hcloud.Error{Code: hcloud.ErrorCodeRateLimitExceeded, Message: "limit"})
	if err == nil {
		t.Fatal("expected mapped error, got nil")
	}
	if got, want := err.Error(), "Hetzner API rate limit exceeded"; got != want {
		t.Fatalf("mapped error = %q, want %q", got, want)
	}

	err = mapHetznerAPIError(hcloud.Error{Code: hcloud.ErrorCodeInvalidInput, Message: "bad input"})
	if got, want := err.Error(), "invalid Hetzner server configuration"; got != want {
		t.Fatalf("mapped error = %q, want %q", got, want)
	}
}

// Image mapping was removed - images are now defined by server profiles.
