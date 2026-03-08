package worker

import (
	"context"

	"pressluft/internal/platform"
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

func (a *ServerStoreAdapter) GetByID(ctx context.Context, id int64) (*server.StoredServer, error) {
	return a.store.GetByID(ctx, id)
}

func (a *ServerStoreAdapter) UpdateStatus(ctx context.Context, id int64, status platform.ServerStatus) error {
	return a.store.UpdateStatus(ctx, id, status)
}

func (a *ServerStoreAdapter) UpdateSetupState(ctx context.Context, id int64, setupState platform.SetupState, setupLastError string) error {
	return a.store.UpdateSetupState(ctx, id, setupState, setupLastError)
}

func (a *ServerStoreAdapter) UpdateProvisioning(ctx context.Context, id int64, providerServerID, actionID, actionStatus string, status platform.ServerStatus, ipv4, ipv6 string) error {
	return a.store.UpdateProvisioning(ctx, id, providerServerID, actionID, actionStatus, status, ipv4, ipv6)
}

func (a *ServerStoreAdapter) UpdateServerType(ctx context.Context, id int64, serverType string) error {
	return a.store.UpdateServerType(ctx, id, serverType)
}

func (a *ServerStoreAdapter) UpdateImage(ctx context.Context, id int64, image string) error {
	return a.store.UpdateImage(ctx, id, image)
}

func (a *ServerStoreAdapter) GetKey(ctx context.Context, serverID int64) (*server.StoredServerKey, error) {
	return a.store.GetKey(ctx, serverID)
}

func (a *ServerStoreAdapter) CreateKey(ctx context.Context, in server.CreateServerKeyInput) error {
	return a.store.CreateKey(ctx, in)
}

// ProviderStoreAdapter wraps provider.Store to implement the worker.ProviderStore interface.
type ProviderStoreAdapter struct {
	store *provider.Store
}

// NewProviderStoreAdapter creates an adapter for the provider store.
func NewProviderStoreAdapter(store *provider.Store) *ProviderStoreAdapter {
	return &ProviderStoreAdapter{store: store}
}

func (a *ProviderStoreAdapter) GetByID(ctx context.Context, id int64) (*provider.StoredProvider, error) {
	return a.store.GetByID(ctx, id)
}
