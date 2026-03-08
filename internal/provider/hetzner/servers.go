package hetzner

import (
	"context"
	"fmt"
	"slices"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"pressluft/internal/provider"
)

func (h *Hetzner) ListServerCatalog(ctx context.Context, token string) (*provider.ServerCatalog, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("api token must not be empty")
	}

	client := newClient(token)

	// Fetch datacenters - this is the authoritative source for availability.
	// dc.ServerTypes.Available contains only server types that can be created NOW.
	datacenters, err := client.Datacenter.All(ctx)
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}

	// Build availability map: serverTypeID -> []locationName
	// Note: dc.ServerTypes.Available only has ID populated, not Name
	availability := make(map[int64][]string)
	locationSet := make(map[string]*hcloud.Location)

	for _, dc := range datacenters {
		if dc.Location == nil {
			continue
		}
		locationSet[dc.Location.Name] = dc.Location

		for _, st := range dc.ServerTypes.Available {
			if st == nil {
				continue
			}
			// Deduplicate: multiple datacenters can share a location
			locs := availability[st.ID]
			found := false
			for _, loc := range locs {
				if loc == dc.Location.Name {
					found = true
					break
				}
			}
			if !found {
				availability[st.ID] = append(locs, dc.Location.Name)
			}
		}
	}

	// Fetch all server types for full metadata (cores, memory, pricing)
	serverTypes, err := client.ServerType.All(ctx)
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}

	// Convert locations
	locations := make([]provider.ServerLocation, 0, len(locationSet))
	for _, loc := range locationSet {
		locations = append(locations, provider.ServerLocation{
			Name:        loc.Name,
			Description: loc.Description,
			Country:     loc.Country,
			City:        loc.City,
			NetworkZone: string(loc.NetworkZone),
		})
	}

	// Convert server types, only including those with availability
	serverTypeOptions := make([]provider.ServerTypeOption, 0, len(serverTypes))
	for _, st := range serverTypes {
		if st == nil {
			continue
		}
		availableAt, hasAvailability := availability[st.ID]
		if !hasAvailability || len(availableAt) == 0 {
			// Skip server types that aren't available anywhere
			continue
		}

		prices := make([]provider.ServerTypePrice, 0, len(st.Pricings))
		for _, p := range st.Pricings {
			if p.Location == nil {
				continue
			}
			prices = append(prices, provider.ServerTypePrice{
				LocationName: p.Location.Name,
				HourlyGross:  p.Hourly.Gross,
				MonthlyGross: p.Monthly.Gross,
				Currency:     p.Hourly.Currency,
			})
		}

		slices.Sort(availableAt)
		serverTypeOptions = append(serverTypeOptions, provider.ServerTypeOption{
			Name:         st.Name,
			Description:  st.Description,
			Cores:        st.Cores,
			MemoryGB:     float64(st.Memory),
			DiskGB:       st.Disk,
			Architecture: string(st.Architecture),
			AvailableAt:  availableAt,
			Prices:       prices,
		})
	}

	catalog := &provider.ServerCatalog{
		Locations:   locations,
		ServerTypes: serverTypeOptions,
	}

	slices.SortFunc(catalog.Locations, func(a, b provider.ServerLocation) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(catalog.ServerTypes, func(a, b provider.ServerTypeOption) int {
		return strings.Compare(a.Name, b.Name)
	})

	return catalog, nil
}

func newClient(token string) *hcloud.Client {
	return hcloud.NewClient(
		hcloud.WithToken(token),
		hcloud.WithApplication("pressluft", "1.0.0"),
	)
}

func mapHetznerAPIError(err error) error {
	switch {
	case hcloud.IsError(err, hcloud.ErrorCodeUnauthorized):
		return fmt.Errorf("invalid Hetzner API token")
	case hcloud.IsError(err, hcloud.ErrorCodeForbidden):
		return fmt.Errorf("insufficient Hetzner permissions")
	case hcloud.IsError(err, hcloud.ErrorCodeRateLimitExceeded):
		return fmt.Errorf("Hetzner API rate limit exceeded")
	case hcloud.IsError(err, hcloud.ErrorCodeInvalidInput):
		return fmt.Errorf("invalid Hetzner server configuration")
	case hcloud.IsError(err, hcloud.ErrorCodeConflict):
		return fmt.Errorf("conflicting Hetzner action in progress")
	default:
		return fmt.Errorf("hetzner api error: %w", err)
	}
}
