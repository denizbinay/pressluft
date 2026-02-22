package audit

import (
	"context"
	"fmt"
	"sync"
	"time"
)

type Entry struct {
	ID           string
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	Result       string
	CreatedAt    time.Time
}

type Store interface {
	Create(ctx context.Context, entry Entry) error
	UpsertByResource(ctx context.Context, entry Entry) error
	List(ctx context.Context) ([]Entry, error)
}

type Recorder interface {
	Record(ctx context.Context, entry Entry) error
	RecordAsyncAccepted(ctx context.Context, entry Entry) error
	UpdateAsyncResult(ctx context.Context, action string, resourceType string, resourceID string, result string) error
}

type Service struct {
	store   Store
	now     func() time.Time
	mu      sync.Mutex
	seqByNs map[int64]uint64
}

func NewService(store Store) *Service {
	return &Service{
		store:   store,
		now:     func() time.Time { return time.Now().UTC() },
		seqByNs: make(map[int64]uint64),
	}
}

func (s *Service) Record(ctx context.Context, entry Entry) error {
	normalized, err := s.normalizeEntry(entry)
	if err != nil {
		return err
	}
	return s.store.Create(ctx, normalized)
}

func (s *Service) RecordAsyncAccepted(ctx context.Context, entry Entry) error {
	if entry.Result == "" {
		entry.Result = "accepted"
	}
	normalized, err := s.normalizeEntry(entry)
	if err != nil {
		return err
	}
	return s.store.UpsertByResource(ctx, normalized)
}

func (s *Service) UpdateAsyncResult(ctx context.Context, action string, resourceType string, resourceID string, result string) error {
	if action == "" || resourceType == "" || resourceID == "" || result == "" {
		return fmt.Errorf("update async result: missing required audit fields")
	}

	now := s.now()
	return s.store.UpsertByResource(ctx, Entry{
		ID:           s.newID(now),
		UserID:       "admin",
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Result:       result,
		CreatedAt:    now,
	})
}

func (s *Service) normalizeEntry(entry Entry) (Entry, error) {
	if entry.UserID == "" || entry.Action == "" || entry.ResourceType == "" || entry.ResourceID == "" || entry.Result == "" {
		return Entry{}, fmt.Errorf("record audit: missing required fields")
	}

	now := s.now()
	if entry.CreatedAt.IsZero() {
		entry.CreatedAt = now
	}
	if entry.ID == "" {
		entry.ID = s.newID(now)
	}

	return entry, nil
}

func (s *Service) newID(now time.Time) string {
	ns := now.UnixNano()
	s.mu.Lock()
	defer s.mu.Unlock()
	s.seqByNs[ns]++
	return fmt.Sprintf("audit-%d-%d", ns, s.seqByNs[ns])
}

type InMemoryStore struct {
	mu      sync.RWMutex
	entries []Entry
	byKey   map[string]int
}

func NewInMemoryStore() *InMemoryStore {
	return &InMemoryStore{byKey: make(map[string]int)}
}

func (s *InMemoryStore) Create(_ context.Context, entry Entry) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.entries = append(s.entries, entry)
	return nil
}

func (s *InMemoryStore) UpsertByResource(_ context.Context, entry Entry) error {
	key := correlationKey(entry.Action, entry.ResourceType, entry.ResourceID)

	s.mu.Lock()
	defer s.mu.Unlock()
	if idx, ok := s.byKey[key]; ok {
		existing := s.entries[idx]
		existing.Result = entry.Result
		s.entries[idx] = existing
		return nil
	}

	s.entries = append(s.entries, entry)
	s.byKey[key] = len(s.entries) - 1
	return nil
}

func (s *InMemoryStore) List(_ context.Context) ([]Entry, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	out := make([]Entry, len(s.entries))
	copy(out, s.entries)
	return out, nil
}

func correlationKey(action string, resourceType string, resourceID string) string {
	return action + "|" + resourceType + "|" + resourceID
}
