package store

import (
	"context"
)

type InMemoryNodeStore struct {
	activeCount int
}

func NewInMemoryNodeStore(activeCount int) *InMemoryNodeStore {
	return &InMemoryNodeStore{activeCount: activeCount}
}

func (s *InMemoryNodeStore) CountActiveNodes(context.Context) (int, error) {
	return s.activeCount, nil
}
