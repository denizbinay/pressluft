package hetzner

import (
	"testing"
	"time"

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
		ProfileKey: "nginx-stack",
	})
	if err != nil {
		t.Fatalf("unexpected validation error: %v", err)
	}
}

func TestMapServerTypesIncludesPricing(t *testing.T) {
	serverTypes := []*hcloud.ServerType{
		{
			Name:         "cx22",
			Description:  "Standard",
			Cores:        2,
			Memory:       4,
			Disk:         40,
			Architecture: hcloud.ArchitectureX86,
			Pricings: []hcloud.ServerTypeLocationPricing{
				{
					Location: &hcloud.Location{Name: "fsn1"},
					Hourly:   hcloud.Price{Currency: "EUR", Gross: "0.0120"},
					Monthly:  hcloud.Price{Currency: "EUR", Gross: "7.19"},
				},
			},
		},
	}

	options := mapServerTypes(serverTypes)
	if len(options) != 1 {
		t.Fatalf("server type count = %d, want %d", len(options), 1)
	}
	if len(options[0].Prices) != 1 {
		t.Fatalf("price count = %d, want %d", len(options[0].Prices), 1)
	}
	if options[0].Prices[0].MonthlyGross != "7.19" {
		t.Fatalf("monthly gross = %q, want %q", options[0].Prices[0].MonthlyGross, "7.19")
	}
}

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

func TestMapImagesSkipsDeprecated(t *testing.T) {
	images := []*hcloud.Image{
		{
			Name:         "ubuntu-24.04",
			Description:  "Ubuntu",
			Type:         hcloud.ImageTypeSystem,
			Architecture: hcloud.ArchitectureX86,
		},
		{
			Name:       "old-image",
			Deprecated: mustParseTime(t, "2025-01-01T00:00:00Z"),
		},
	}

	mapped := mapImages(images)
	if len(mapped) != 1 {
		t.Fatalf("image count = %d, want %d", len(mapped), 1)
	}
	if mapped[0].Name != "ubuntu-24.04" {
		t.Fatalf("image name = %q, want %q", mapped[0].Name, "ubuntu-24.04")
	}
}

func mustParseTime(t *testing.T, value string) time.Time {
	t.Helper()
	parsed, err := time.Parse(time.RFC3339, value)
	if err != nil {
		t.Fatalf("parse time: %v", err)
	}
	return parsed
}
