package providers

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"sort"
	"strings"
	"time"
)

var ErrUnknownProvider = errors.New("unknown provider")
var ErrMissingSecret = errors.New("provider secret is required")

type CatalogEntry struct {
	ID           string
	DisplayName  string
	Capabilities []string
}

type PublicConnection struct {
	ProviderID        string     `json:"provider_id"`
	DisplayName       string     `json:"display_name"`
	Status            Status     `json:"status"`
	Capabilities      []string   `json:"capabilities"`
	SecretConfigured  bool       `json:"secret_configured"`
	Guidance          []string   `json:"guidance"`
	LastStatusMessage string     `json:"last_status_message"`
	ConnectedAt       *time.Time `json:"connected_at"`
	LastCheckedAt     *time.Time `json:"last_checked_at"`
	UpdatedAt         time.Time  `json:"updated_at"`
}

type ConnectInput struct {
	ProviderID string
	Secret     string
}

type Service struct {
	store             Store
	now               func() time.Time
	catalog           map[string]CatalogEntry
	hetznerAPIBaseURL string
	hetznerHTTPClient *http.Client
}

func NewService(store Store) *Service {
	return &Service{
		store: store,
		now:   func() time.Time { return time.Now().UTC() },
		catalog: map[string]CatalogEntry{
			"hetzner": {
				ID:          "hetzner",
				DisplayName: "Hetzner Cloud",
				Capabilities: []string{
					"node_acquisition",
					"provider_health",
				},
			},
		},
		hetznerAPIBaseURL: "https://api.hetzner.cloud/v1",
		hetznerHTTPClient: &http.Client{Timeout: 10 * time.Second},
	}
}

func (s *Service) List(ctx context.Context) ([]PublicConnection, error) {
	persisted, err := s.store.List(ctx)
	if err != nil {
		return nil, err
	}

	persistedByID := make(map[string]Connection, len(persisted))
	for _, item := range persisted {
		persistedByID[item.ProviderID] = item
	}

	ids := make([]string, 0, len(s.catalog))
	for id := range s.catalog {
		ids = append(ids, id)
	}
	sort.Strings(ids)

	items := make([]PublicConnection, 0, len(ids))
	for _, providerID := range ids {
		entry := s.catalog[providerID]
		connection, ok := persistedByID[providerID]
		if !ok {
			items = append(items, PublicConnection{
				ProviderID:        providerID,
				DisplayName:       entry.DisplayName,
				Status:            StatusDisconnected,
				Capabilities:      append([]string(nil), entry.Capabilities...),
				SecretConfigured:  false,
				Guidance:          []string{"Connect provider credentials to enable node acquisition."},
				LastStatusMessage: "Provider not connected",
				UpdatedAt:         time.Time{},
			})
			continue
		}

		if len(connection.Capabilities) == 0 {
			connection.Capabilities = append([]string(nil), entry.Capabilities...)
		}
		items = append(items, toPublicConnection(entry, connection))
	}

	return items, nil
}

func (s *Service) Connect(ctx context.Context, input ConnectInput) (PublicConnection, error) {
	providerID := strings.TrimSpace(strings.ToLower(input.ProviderID))
	entry, ok := s.catalog[providerID]
	if !ok {
		return PublicConnection{}, ErrUnknownProvider
	}

	secret := strings.TrimSpace(input.Secret)
	if secret == "" {
		return PublicConnection{}, ErrMissingSecret
	}

	now := s.now()
	health := s.validateProviderCredential(ctx, providerID, secret)

	connection := Connection{
		ProviderID:        providerID,
		Status:            health.Status,
		SecretToken:       secret,
		SecretConfigured:  true,
		Guidance:          health.Guidance,
		Capabilities:      append([]string(nil), entry.Capabilities...),
		LastStatusMessage: health.Message,
		ConnectedAt:       &now,
		LastCheckedAt:     &now,
		UpdatedAt:         now,
	}

	persisted, err := s.store.Upsert(ctx, connection)
	if err != nil {
		return PublicConnection{}, err
	}

	return toPublicConnection(entry, persisted), nil
}

type credentialHealth struct {
	Status   Status
	Message  string
	Guidance []string
}

func (s *Service) validateProviderCredential(ctx context.Context, providerID string, secret string) credentialHealth {
	if os.Getenv("PRESSLUFT_DISABLE_RUNTIME_PROBES") == "1" {
		return credentialHealth{
			Status:  StatusConnected,
			Message: "Provider connection healthy",
		}
	}

	switch providerID {
	case "hetzner":
		return s.validateHetznerCredential(ctx, secret)
	default:
		return credentialHealth{
			Status:  StatusConnected,
			Message: "Provider connection healthy",
		}
	}
}

func (s *Service) validateHetznerCredential(ctx context.Context, secret string) credentialHealth {
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, s.hetznerAPIBaseURL+"/actions?per_page=1", nil)
	if err != nil {
		return credentialHealth{
			Status:   StatusDegraded,
			Message:  "Unable to verify provider credential with Hetzner API",
			Guidance: []string{"Retry provider connect. If this persists, verify outbound access to api.hetzner.cloud."},
		}
	}
	req.Header.Set("Authorization", "Bearer "+secret)

	resp, err := s.hetznerHTTPClient.Do(req)
	if err != nil {
		return credentialHealth{
			Status:   StatusDegraded,
			Message:  fmt.Sprintf("Unable to verify provider credential with Hetzner API: %v", err),
			Guidance: []string{"Retry provider connect. If this persists, verify outbound access to api.hetzner.cloud."},
		}
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return credentialHealth{
			Status:  StatusConnected,
			Message: "Provider connection healthy",
		}
	}

	if resp.StatusCode == http.StatusUnauthorized || resp.StatusCode == http.StatusForbidden {
		return credentialHealth{
			Status:   StatusDegraded,
			Message:  "Credential rejected by Hetzner API",
			Guidance: []string{"Generate a valid bearer token in Hetzner Cloud and reconnect provider."},
		}
	}

	if resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode >= 500 {
		return credentialHealth{
			Status:   StatusDegraded,
			Message:  fmt.Sprintf("Hetzner API unavailable during credential validation (status %d)", resp.StatusCode),
			Guidance: []string{"Retry provider connect after provider/API health recovers."},
		}
	}

	return credentialHealth{
		Status:   StatusDegraded,
		Message:  fmt.Sprintf("Unexpected Hetzner API status during credential validation (%d)", resp.StatusCode),
		Guidance: []string{"Confirm token permissions and retry provider connect."},
	}
}

func toPublicConnection(entry CatalogEntry, connection Connection) PublicConnection {
	guidance := append([]string(nil), connection.Guidance...)
	capabilities := append([]string(nil), connection.Capabilities...)
	if len(capabilities) == 0 {
		capabilities = append([]string(nil), entry.Capabilities...)
	}

	return PublicConnection{
		ProviderID:        connection.ProviderID,
		DisplayName:       entry.DisplayName,
		Status:            connection.Status,
		Capabilities:      capabilities,
		SecretConfigured:  connection.SecretConfigured,
		Guidance:          guidance,
		LastStatusMessage: connection.LastStatusMessage,
		ConnectedAt:       connection.ConnectedAt,
		LastCheckedAt:     connection.LastCheckedAt,
		UpdatedAt:         connection.UpdatedAt,
	}
}
