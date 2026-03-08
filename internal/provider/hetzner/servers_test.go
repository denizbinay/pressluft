package hetzner

import (
	"testing"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
)

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
