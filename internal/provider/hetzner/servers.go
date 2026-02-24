package hetzner

import (
	"context"
	"crypto/ed25519"
	"crypto/rand"
	"encoding/pem"
	"fmt"
	"slices"
	"strconv"
	"strings"

	"github.com/hetznercloud/hcloud-go/v2/hcloud"
	"golang.org/x/crypto/ssh"

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
		if hcloud.IsError(err, hcloud.ErrorCodeUniquenessError) {
			// Idempotency: if server already exists, retrieve it
			existingServer, _, errGet := client.Server.Get(ctx, req.Name)
			if errGet == nil && existingServer != nil {
				return &provider.CreateServerResult{
					ProviderServerID: strconv.FormatInt(existingServer.ID, 10),
					Status:           string(existingServer.Status),
				}, nil
			}
		}
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

// CreateSSHKey registers a public SSH key with Hetzner Cloud.
func (h *Hetzner) CreateSSHKey(ctx context.Context, token, name, publicKey string) (*provider.SSHKeyResult, error) {
	if strings.TrimSpace(token) == "" {
		return nil, fmt.Errorf("api token must not be empty")
	}
	if strings.TrimSpace(name) == "" {
		return nil, fmt.Errorf("ssh key name must not be empty")
	}
	if strings.TrimSpace(publicKey) == "" {
		return nil, fmt.Errorf("public key must not be empty")
	}

	client := newClient(token)

	sshKey, _, err := client.SSHKey.Create(ctx, hcloud.SSHKeyCreateOpts{
		Name:      name,
		PublicKey: publicKey,
	})
	if err != nil {
		if hcloud.IsError(err, hcloud.ErrorCodeUniquenessError) {
			// Idempotency: if key already exists, try to retrieve it
			existingKey, _, errGet := client.SSHKey.Get(ctx, name)
			if errGet == nil && existingKey != nil {
				return &provider.SSHKeyResult{
					ID:          existingKey.ID,
					Name:        existingKey.Name,
					Fingerprint: existingKey.Fingerprint,
				}, nil
			}
		}
		return nil, mapHetznerAPIError(err)
	}

	return &provider.SSHKeyResult{
		ID:          sshKey.ID,
		Name:        sshKey.Name,
		Fingerprint: sshKey.Fingerprint,
	}, nil
}

// DeleteSSHKey removes an SSH key from Hetzner Cloud by its ID.
func (h *Hetzner) DeleteSSHKey(ctx context.Context, token string, keyID int64) error {
	if strings.TrimSpace(token) == "" {
		return fmt.Errorf("api token must not be empty")
	}
	if keyID <= 0 {
		return fmt.Errorf("ssh key id must be positive")
	}

	client := newClient(token)

	_, err := client.SSHKey.Delete(ctx, &hcloud.SSHKey{ID: keyID})
	if err != nil {
		return mapHetznerAPIError(err)
	}

	return nil
}

// GenerateSSHKeyPair creates a new Ed25519 SSH key pair.
// Returns the public key in OpenSSH authorized_keys format and the private key in PEM format.
func GenerateSSHKeyPair(comment string) (publicKey, privateKey string, err error) {
	// Generate Ed25519 key pair
	pubKey, privKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return "", "", fmt.Errorf("generate ed25519 key: %w", err)
	}

	// Convert public key to OpenSSH authorized_keys format
	sshPubKey, err := ssh.NewPublicKey(pubKey)
	if err != nil {
		return "", "", fmt.Errorf("create ssh public key: %w", err)
	}
	authorizedKey := ssh.MarshalAuthorizedKey(sshPubKey)
	publicKeyStr := strings.TrimSpace(string(authorizedKey))
	if comment != "" {
		publicKeyStr = publicKeyStr + " " + comment
	}

	// Convert private key to PEM format
	pemBlock, err := ssh.MarshalPrivateKey(privKey, comment)
	if err != nil {
		return "", "", fmt.Errorf("marshal private key: %w", err)
	}
	privateKeyPEM := pem.EncodeToMemory(pemBlock)

	return publicKeyStr, string(privateKeyPEM), nil
}
