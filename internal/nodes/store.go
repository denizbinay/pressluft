package nodes

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"sync"
	"time"
)

var ErrNotFound = errors.New("node not found")
var ErrSelfNodeNotFound = errors.New("self-node not found")
var ErrLocalNodeExists = errors.New("local node already exists")

const (
	SelfNodeID   = "00000000-0000-0000-0000-000000000001"
	SelfNodeName = "self"
)

type Status string

const (
	StatusActive       Status = "active"
	StatusProvisioning Status = "provisioning"
	StatusUnreachable  Status = "unreachable"
	StatusDecommission Status = "decommissioned"
)

type Node struct {
	ID                string
	ProviderID        string
	Name              string
	Hostname          string
	PublicIP          string
	SSHPort           int
	SSHUser           string
	SSHPrivateKeyPath string
	Status            Status
	IsLocal           bool
	LastSeenAt        *time.Time
	CreatedAt         time.Time
	UpdatedAt         time.Time
}

type Store interface {
	List(ctx context.Context) ([]Node, error)
	GetByID(ctx context.Context, id string) (Node, error)
	GetSelfNode(ctx context.Context) (Node, error)
	Create(ctx context.Context, input CreateInput) (Node, error)
	UpsertLocal(ctx context.Context, input CreateInput) (Node, error)
	EnsureSelfNode(ctx context.Context, now time.Time) (Node, error)
	UpdateConnection(ctx context.Context, id string, input ConnectionInput) (Node, error)
	MarkProvisioning(ctx context.Context, id string, now time.Time) (Node, error)
	MarkActive(ctx context.Context, id string, now time.Time) (Node, error)
	MarkUnreachable(ctx context.Context, id string, now time.Time) (Node, error)
	CountActiveNodes(ctx context.Context) (int, error)
}

type CreateInput struct {
	ProviderID        string
	Name              string
	Hostname          string
	PublicIP          string
	SSHPort           int
	SSHUser           string
	SSHPrivateKeyPath string
	IsLocal           bool
	Now               time.Time
}

type ConnectionInput struct {
	Hostname          string
	PublicIP          string
	SSHPort           int
	SSHUser           string
	SSHPrivateKeyPath string
	Now               time.Time
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

func (s *InMemoryStore) List(_ context.Context) ([]Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	result := make([]Node, 0, len(s.byID))
	for _, node := range s.byID {
		result = append(result, node)
	}

	return result, nil
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

func (s *InMemoryStore) GetSelfNode(_ context.Context) (Node, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for _, node := range s.byID {
		if node.IsLocal {
			return node, nil
		}
	}

	return Node{}, ErrSelfNodeNotFound
}

func (s *InMemoryStore) Create(_ context.Context, input CreateInput) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if input.IsLocal {
		for _, existing := range s.byID {
			if existing.IsLocal {
				return Node{}, ErrLocalNodeExists
			}
		}
	}

	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	id, err := generateUUIDv4()
	if err != nil {
		return Node{}, fmt.Errorf("generate node id: %w", err)
	}

	sshPort := input.SSHPort
	if sshPort <= 0 {
		sshPort = 22
	}

	status := StatusActive
	node := Node{
		ID:                id,
		ProviderID:        input.ProviderID,
		Name:              input.Name,
		Hostname:          input.Hostname,
		PublicIP:          input.PublicIP,
		SSHPort:           sshPort,
		SSHUser:           input.SSHUser,
		SSHPrivateKeyPath: input.SSHPrivateKeyPath,
		Status:            status,
		IsLocal:           input.IsLocal,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	s.byID[node.ID] = node
	return node, nil
}

func (s *InMemoryStore) UpsertLocal(_ context.Context, input CreateInput) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	sshPort := input.SSHPort
	if sshPort <= 0 {
		sshPort = 22
	}

	for id, existing := range s.byID {
		if !existing.IsLocal {
			continue
		}
		existing.Name = input.Name
		existing.Hostname = input.Hostname
		existing.PublicIP = input.PublicIP
		existing.SSHPort = sshPort
		existing.SSHUser = input.SSHUser
		existing.SSHPrivateKeyPath = input.SSHPrivateKeyPath
		existing.IsLocal = true
		existing.UpdatedAt = now
		s.byID[id] = existing
		return existing, nil
	}

	id, err := generateUUIDv4()
	if err != nil {
		return Node{}, fmt.Errorf("generate node id: %w", err)
	}

	node := Node{
		ID:                id,
		ProviderID:        input.ProviderID,
		Name:              input.Name,
		Hostname:          input.Hostname,
		PublicIP:          input.PublicIP,
		SSHPort:           sshPort,
		SSHUser:           input.SSHUser,
		SSHPrivateKeyPath: input.SSHPrivateKeyPath,
		Status:            StatusActive,
		IsLocal:           true,
		CreatedAt:         now,
		UpdatedAt:         now,
	}

	s.byID[node.ID] = node
	return node, nil
}

func (s *InMemoryStore) EnsureSelfNode(_ context.Context, now time.Time) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, node := range s.byID {
		if node.IsLocal {
			return node, nil
		}
	}

	selfNode := Node{
		ID:        SelfNodeID,
		ProviderID: "local",
		Name:      SelfNodeName,
		Hostname:  "127.0.0.1",
		PublicIP:  "127.0.0.1",
		SSHPort:   22,
		SSHUser:   "pressluft",
		Status:    StatusActive,
		IsLocal:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}

	s.byID[selfNode.ID] = selfNode
	return selfNode, nil
}

func (s *InMemoryStore) UpdateConnection(_ context.Context, id string, input ConnectionInput) (Node, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	node, ok := s.byID[id]
	if !ok {
		return Node{}, ErrNotFound
	}

	now := input.Now.UTC()
	if now.IsZero() {
		now = time.Now().UTC()
	}

	sshPort := input.SSHPort
	if sshPort <= 0 {
		sshPort = 22
	}

	node.Hostname = input.Hostname
	node.PublicIP = input.PublicIP
	node.SSHPort = sshPort
	node.SSHUser = input.SSHUser
	node.SSHPrivateKeyPath = input.SSHPrivateKeyPath
	node.UpdatedAt = now
	s.byID[id] = node

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

func generateUUIDv4() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80
	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16],
	), nil
}

var ErrInvalidTransition = errors.New("invalid node state transition")
