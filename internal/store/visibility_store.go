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

type InMemorySiteStore struct {
	totalCount int
}

func NewInMemorySiteStore(totalCount int) *InMemorySiteStore {
	return &InMemorySiteStore{totalCount: totalCount}
}

func (s *InMemorySiteStore) CountSites(context.Context) (int, error) {
	return s.totalCount, nil
}
