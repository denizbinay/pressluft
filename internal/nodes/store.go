package nodes

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrNotFound = errors.New("node not found")

type Status string

const (
	StatusActive       Status = "active"
	StatusProvisioning Status = "provisioning"
	StatusUnreachable  Status = "unreachable"
	StatusDecommission Status = "decommissioned"
)

type Node struct {
	ID                string
	Hostname          string
	PublicIP          string
	SSHPort           int
	SSHUser           string
	SSHPrivateKeyPath string
	Status            Status
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Store interface {
	GetByID(ctx context.Context, id string) (Node, error)
	MarkProvisioning(ctx context.Context, id string, now time.Time) (Node, error)
	MarkActive(ctx context.Context, id string, now time.Time) (Node, error)
	MarkUnreachable(ctx context.Context, id string, now time.Time) (Node, error)
	CountActiveNodes(ctx context.Context) (int, error)
}

type InMemoryStore struct {
	mu   sync.RWMutex
	byID map[string]Node
}

func NewInMemoryStore(seed []Node) *InMemoryStore {
	byID := make(map[string]Node, len(seed))
	for _, node := range seed {
		if node.ID == "" {
			continue
		}
		byID[node.ID] = node
	}

	return &InMemoryStore{byID: byID}
}

func (s *InMemoryStore) GetByID(_ context.Context, id string) (Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	node, ok := s.byID[id]
	if !ok {
		return Node{}, ErrNotFound
	}

	return node, nil
}

func (s *InMemoryStore) MarkProvisioning(_ context.Context, id string, now time.Time) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, ok := s.byID[id]
	if !ok {
		return Node{}, ErrNotFound
	}

	switch node.Status {
	case StatusActive, StatusProvisioning, StatusUnreachable:
		node.Status = StatusProvisioning
		node.UpdatedAt = now
		s.byID[id] = node
		return node, nil
	default:
		return Node{}, fmt.Errorf("mark provisioning from %s: %w", node.Status, ErrInvalidTransition)
	}
}

func (s *InMemoryStore) MarkActive(_ context.Context, id string, now time.Time) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, ok := s.byID[id]
	if !ok {
		return Node{}, ErrNotFound
	}

	switch node.Status {
	case StatusProvisioning, StatusActive:
		node.Status = StatusActive
		node.UpdatedAt = now
		s.byID[id] = node
		return node, nil
	default:
		return Node{}, fmt.Errorf("mark active from %s: %w", node.Status, ErrInvalidTransition)
	}
}

func (s *InMemoryStore) MarkUnreachable(_ context.Context, id string, now time.Time) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, ok := s.byID[id]
	if !ok {
		return Node{}, ErrNotFound
	}

	switch node.Status {
	case StatusProvisioning, StatusUnreachable:
		node.Status = StatusUnreachable
		node.UpdatedAt = now
		s.byID[id] = node
		return node, nil
	default:
		return Node{}, fmt.Errorf("mark unreachable from %s: %w", node.Status, ErrInvalidTransition)
	}
}

func (s *InMemoryStore) CountActiveNodes(_ context.Context) (int, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	count := 0
	for _, node := range s.byID {
		if node.Status == StatusActive {
			count++
		}
	}

	return count, nil
}

var ErrInvalidTransition = errors.New("invalid node state transition")
