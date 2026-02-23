package hetzner

import (
	"context"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"

	"pressluft/internal/provider"
)

func (h *Hetzner) ListServerCatalog(ctx context.Context, token string) (*provider.ServerCatalog, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("api token must not be empty")
	}

	client := newClient(token)

	locations, err := client.Location.All(ctx)
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}

	serverTypes, err := client.ServerType.All(ctx)
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}

	images, _, err := client.Image.List(ctx, hcloud.ImageListOpts{
		Type:     []hcloud.ImageType{hcloud.ImageTypeSystem},
		ListOpts: hcloud.ListOpts{PerPage: 50},
	})
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}

	catalog := &provider.ServerCatalog{
		Locations:   mapLocations(locations),
		ServerTypes: mapServerTypes(serverTypes),
		Images:      mapImages(images),
	}

	slices.SortFunc(catalog.Locations, func(a, b provider.ServerLocation) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(catalog.ServerTypes, func(a, b provider.ServerTypeOption) int {
		return strings.Compare(a.Name, b.Name)
	})
	slices.SortFunc(catalog.Images, func(a, b provider.ServerImageOption) int {
		return strings.Compare(a.Name, b.Name)
	})

	return catalog, nil
}

func (h *Hetzner) CreateServer(ctx context.Context, token string, req provider.CreateServerRequest) (*provider.CreateServerResult, error) {
	if err := validateCreateServerRequest(req); err != nil {
		return nil, err
	}
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("api token must not be empty")
	}

	client := newClient(token)

	serverType, _, err := client.ServerType.GetByName(ctx, req.ServerType)
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}
	if serverType == nil {
		return nil, fmt.Errorf("server type %q not found", req.ServerType)
	}

	location, _, err := client.Location.GetByName(ctx, req.Location)
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}
	if location == nil {
		return nil, fmt.Errorf("location %q not found", req.Location)
	}

	image, _, err := client.Image.GetForArchitecture(ctx, req.Image, serverType.Architecture)
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}
	if image == nil {
		return nil, fmt.Errorf("image %q not found for architecture %q", req.Image, serverType.Architecture)
	}

	result, _, err := client.Server.Create(ctx, hcloud.ServerCreateOpts{
		Name:       req.Name,
		ServerType: serverType,
		Image:      image,
		Location:   location,
		UserData:   req.UserData,
		Labels:     req.Labels,
	})
	if err != nil {
		return nil, mapHetznerAPIError(err)
	}

	createResult := &provider.CreateServerResult{
		Status: "provisioning",
	}
	if result.Server != nil {
		createResult.ProviderServerID = strconv.FormatInt(result.Server.ID, 10)
	}
	if result.Action != nil {
		createResult.ActionID = strconv.FormatInt(result.Action.ID, 10)
		createResult.Status = string(result.Action.Status)
	}

	return createResult, nil
}

func newClient(token string) *hcloud.Client {
	return hcloud.NewClient(
		hcloud.WithToken(token),
		hcloud.WithApplication("pressluft", "1.0.0"),
	)
}

func mapLocations(in []*hcloud.Location) []provider.ServerLocation {
	out := make([]provider.ServerLocation, 0, len(in))
	for _, loc := range in {
		if loc == nil {
			continue
		}
		out = append(out, provider.ServerLocation{
			Name:        loc.Name,
			Description: loc.Description,
			Country:     loc.Country,
			City:        loc.City,
			NetworkZone: string(loc.NetworkZone),
		})
	}
	return out
}

func mapServerTypes(in []*hcloud.ServerType) []provider.ServerTypeOption {
	out := make([]provider.ServerTypeOption, 0, len(in))
	for _, st := range in {
		if st == nil {
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

		out = append(out, provider.ServerTypeOption{
			Name:         st.Name,
			Description:  st.Description,
			Cores:        st.Cores,
			MemoryGB:     float64(st.Memory),
			DiskGB:       st.Disk,
			Architecture: string(st.Architecture),
			Prices:       prices,
		})
	}
	return out
}

func mapImages(in []*hcloud.Image) []provider.ServerImageOption {
	out := make([]provider.ServerImageOption, 0, len(in))
	for _, img := range in {
		if img == nil || img.IsDeprecated() || img.IsDeleted() {
			continue
		}
		out = append(out, provider.ServerImageOption{
			Name:         img.Name,
			Description:  img.Description,
			Type:         string(img.Type),
			OSFlavor:     img.OSFlavor,
			OSVersion:    img.OSVersion,
			Architecture: string(img.Architecture),
		})
	}
	return out
}

func validateCreateServerRequest(req provider.CreateServerRequest) error {
	if strings.TrimSpace(req.Name) == "" {
		return fmt.Errorf("name is required")
	}
	if strings.TrimSpace(req.Location) == "" {
		return fmt.Errorf("location is required")
	}
	if strings.TrimSpace(req.ServerType) == "" {
		return fmt.Errorf("server_type is required")
	}
	if strings.TrimSpace(req.Image) == "" {
		return fmt.Errorf("image is required")
	}
	if strings.TrimSpace(req.ProfileKey) == "" {
		return fmt.Errorf("profile_key is required")
	}
	return nil
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
