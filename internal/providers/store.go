package providers

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrProviderNotFound = errors.New("provider not found")

type Status string

const (
	StatusConnected    Status = "connected"
	StatusDegraded     Status = "degraded"
	StatusDisconnected Status = "disconnected"
)

type Connection struct {
	ProviderID         string
	Status             Status
	SecretToken        string
	SecretConfigured   bool
	Guidance           []string
	Capabilities       []string
	LastStatusMessage  string
	ConnectedAt        *time.Time
	LastCheckedAt      *time.Time
	UpdatedAt          time.Time
}

type Store interface {
	List(ctx context.Context) ([]Connection, error)
	GetByProviderID(ctx context.Context, providerID string) (Connection, error)
	Upsert(ctx context.Context, connection Connection) (Connection, error)
}

type InMemoryStore struct {
	mu     sync.RWMutex
	byName map[string]Connection
}

func NewInMemoryStore(seed []Connection) *InMemoryStore {
	byName := make(map[string]Connection, len(seed))
	for _, item := range seed {
		if item.ProviderID == "" {
			continue
		}
		byName[item.ProviderID] = item
	}

	return &InMemoryStore{byName: byName}
}

func (s *InMemoryStore) List(_ context.Context) ([]Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	items := make([]Connection, 0, len(s.byName))
	for _, item := range s.byName {
		items = append(items, item)
	}

	return items, nil
}

func (s *InMemoryStore) GetByProviderID(_ context.Context, providerID string) (Connection, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	item, ok := s.byName[providerID]
	if !ok {
		return Connection{}, ErrProviderNotFound
	}

	return item, nil
}

func (s *InMemoryStore) Upsert(_ context.Context, connection Connection) (Connection, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.byName[connection.ProviderID] = connection
	return connection, nil
}
