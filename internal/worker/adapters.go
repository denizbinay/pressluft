package worker

import (
	"context"

	"pressluft/internal/provider"
	"pressluft/internal/server"
)

// ServerStoreAdapter wraps server.ServerStore to implement the worker.ServerStore interface.
type ServerStoreAdapter struct {
	store *server.ServerStore
}

// NewServerStoreAdapter creates an adapter for the server store.
func NewServerStoreAdapter(store *server.ServerStore) *ServerStoreAdapter {
	return &ServerStoreAdapter{store: store}
}

func (a *ServerStoreAdapter) GetByID(ctx context.Context, id int64) (*StoredServer, error) {
	s, err := a.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &StoredServer{
		ID:               s.ID,
		ProviderID:       s.ProviderID,
		ProviderType:     s.ProviderType,
		ProviderServerID: s.ProviderServerID,
		IPv4:             s.IPv4,
		IPv6:             s.IPv6,
		Name:             s.Name,
		Location:         s.Location,
		ServerType:       s.ServerType,
		Image:            s.Image,
		ProfileKey:       s.ProfileKey,
		Status:           s.Status,
	}, nil
}

func (a *ServerStoreAdapter) UpdateStatus(ctx context.Context, id int64, status string) error {
	return a.store.UpdateStatus(ctx, id, status)
}

func (a *ServerStoreAdapter) UpdateProvisioning(ctx context.Context, id int64, providerServerID, actionID, actionStatus, status, ipv4, ipv6 string) error {
	return a.store.UpdateProvisioning(ctx, id, providerServerID, actionID, actionStatus, status, ipv4, ipv6)
}

func (a *ServerStoreAdapter) UpdateServerType(ctx context.Context, id int64, serverType string) error {
	return a.store.UpdateServerType(ctx, id, serverType)
}

func (a *ServerStoreAdapter) GetKey(ctx context.Context, serverID int64) (*StoredServerKey, error) {
	key, err := a.store.GetKey(ctx, serverID)
	if err != nil || key == nil {
		return nil, err
	}
	return &StoredServerKey{
		ServerID:            key.ServerID,
		PublicKey:           key.PublicKey,
		PrivateKeyEncrypted: key.PrivateKeyEncrypted,
		EncryptionKeyID:     key.EncryptionKeyID,
		CreatedAt:           key.CreatedAt,
		RotatedAt:           key.RotatedAt,
	}, nil
}

func (a *ServerStoreAdapter) CreateKey(ctx context.Context, in CreateServerKeyInput) error {
	return a.store.CreateKey(ctx, server.CreateServerKeyInput{
		ServerID:            in.ServerID,
		PublicKey:           in.PublicKey,
		PrivateKeyEncrypted: in.PrivateKeyEncrypted,
		EncryptionKeyID:     in.EncryptionKeyID,
		RotatedAt:           in.RotatedAt,
	})
}

// ProviderStoreAdapter wraps provider.Store to implement the worker.ProviderStore interface.
type ProviderStoreAdapter struct {
	store *provider.Store
}

// NewProviderStoreAdapter creates an adapter for the provider store.
func NewProviderStoreAdapter(store *provider.Store) *ProviderStoreAdapter {
	return &ProviderStoreAdapter{store: store}
}

func (a *ProviderStoreAdapter) GetByID(ctx context.Context, id int64) (*StoredProvider, error) {
	p, err := a.store.GetByID(ctx, id)
	if err != nil {
		return nil, err
	}
	return &StoredProvider{
		ID:       p.ID,
		Type:     p.Type,
		APIToken: p.APIToken,
	}, nil
}
