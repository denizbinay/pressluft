package store

import (
	"context"
	"errors"
	"sync"
	"time"
)

var ErrRestoreRequestNotFound = errors.New("restore request not found")

type RestoreRequest struct {
	JobID         string
	EnvironmentID string
	BackupID      string
	CreatedAt     time.Time
}

type RestoreRequestStore interface {
	SaveRestoreRequest(ctx context.Context, request RestoreRequest) error
	GetRestoreRequestByJobID(ctx context.Context, jobID string) (RestoreRequest, error)
	DeleteRestoreRequest(ctx context.Context, jobID string) error
}

type InMemoryRestoreRequestStore struct {
	mu      sync.RWMutex
	byJobID map[string]RestoreRequest
}

var (
	globalRestoreRequestStoreMu sync.RWMutex
	globalRestoreRequestStore   RestoreRequestStore
)

func NewInMemoryRestoreRequestStore() *InMemoryRestoreRequestStore {
	store := &InMemoryRestoreRequestStore{byJobID: make(map[string]RestoreRequest)}
	setDefaultRestoreRequestStore(store)
	return store
}

func DefaultRestoreRequestStore() RestoreRequestStore {
	globalRestoreRequestStoreMu.RLock()
	current := globalRestoreRequestStore
	globalRestoreRequestStoreMu.RUnlock()
	if current != nil {
		return current
	}
	return NewInMemoryRestoreRequestStore()
}

func setDefaultRestoreRequestStore(restoreStore RestoreRequestStore) {
	globalRestoreRequestStoreMu.Lock()
	defer globalRestoreRequestStoreMu.Unlock()
	globalRestoreRequestStore = restoreStore
}

func (s *InMemoryRestoreRequestStore) SaveRestoreRequest(_ context.Context, request RestoreRequest) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.byJobID[request.JobID] = request
	return nil
}

func (s *InMemoryRestoreRequestStore) GetRestoreRequestByJobID(_ context.Context, jobID string) (RestoreRequest, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	request, ok := s.byJobID[jobID]
	if !ok {
		return RestoreRequest{}, ErrRestoreRequestNotFound
	}
	return request, nil
}

func (s *InMemoryRestoreRequestStore) DeleteRestoreRequest(_ context.Context, jobID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.byJobID, jobID)
	return nil
}
