package nodes

import (
	"context"
	"errors"
	"testing"
	"time"
)

func TestMarkProvisioningAndActive(t *testing.T) {
	now := time.Date(2026, 2, 22, 1, 0, 0, 0, time.UTC)
	store := NewInMemoryStore([]Node{{
		ID:        "node-1",
		Hostname:  "127.0.0.1",
		SSHPort:   22,
		SSHUser:   "ubuntu",
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}})

	node, err := store.MarkProvisioning(context.Background(), "node-1", now)
	if err != nil {
		t.Fatalf("MarkProvisioning() error = %v", err)
	}
	if node.Status != StatusProvisioning {
		t.Fatalf("status = %s, want provisioning", node.Status)
	}

	node, err = store.MarkActive(context.Background(), "node-1", now.Add(time.Minute))
	if err != nil {
		t.Fatalf("MarkActive() error = %v", err)
	}
	if node.Status != StatusActive {
		t.Fatalf("status = %s, want active", node.Status)
	}
}

func TestMarkUnreachableFromProvisioning(t *testing.T) {
	now := time.Date(2026, 2, 22, 1, 5, 0, 0, time.UTC)
	store := NewInMemoryStore([]Node{{
		ID:        "node-1",
		Hostname:  "127.0.0.1",
		SSHPort:   22,
		SSHUser:   "ubuntu",
		Status:    StatusProvisioning,
		CreatedAt: now,
		UpdatedAt: now,
	}})

	node, err := store.MarkUnreachable(context.Background(), "node-1", now)
	if err != nil {
		t.Fatalf("MarkUnreachable() error = %v", err)
	}
	if node.Status != StatusUnreachable {
		t.Fatalf("status = %s, want unreachable", node.Status)
	}
}

func TestCountActiveNodes(t *testing.T) {
	store := NewInMemoryStore([]Node{
		{ID: "node-1", Status: StatusActive},
		{ID: "node-2", Status: StatusProvisioning},
		{ID: "node-3", Status: StatusActive},
	})

	count, err := store.CountActiveNodes(context.Background())
	if err != nil {
		t.Fatalf("CountActiveNodes() error = %v", err)
	}
	if count != 2 {
		t.Fatalf("count = %d, want 2", count)
	}
}

func TestGetSelfNodeReturnsErrorWhenNotExists(t *testing.T) {
	store := NewInMemoryStore([]Node{
		{ID: "node-1", Status: StatusActive, IsLocal: false},
	})

	_, err := store.GetSelfNode(context.Background())
	if err != ErrSelfNodeNotFound {
		t.Fatalf("GetSelfNode() error = %v, want ErrSelfNodeNotFound", err)
	}
}

func TestGetSelfNodeReturnsLocalNode(t *testing.T) {
	store := NewInMemoryStore([]Node{
		{ID: "node-1", Status: StatusActive, IsLocal: false},
		{ID: SelfNodeID, Status: StatusActive, IsLocal: true, Hostname: "127.0.0.1"},
	})

	node, err := store.GetSelfNode(context.Background())
	if err != nil {
		t.Fatalf("GetSelfNode() error = %v", err)
	}
	if node.ID != SelfNodeID {
		t.Fatalf("node.ID = %s, want %s", node.ID, SelfNodeID)
	}
	if !node.IsLocal {
		t.Fatal("node.IsLocal = false, want true")
	}
}

func TestEnsureSelfNodeCreatesWhenMissing(t *testing.T) {
	now := time.Date(2026, 2, 22, 2, 0, 0, 0, time.UTC)
	store := NewInMemoryStore(nil)

	node, err := store.EnsureSelfNode(context.Background(), now)
	if err != nil {
		t.Fatalf("EnsureSelfNode() error = %v", err)
	}
	if node.ID != SelfNodeID {
		t.Fatalf("node.ID = %s, want %s", node.ID, SelfNodeID)
	}
	if !node.IsLocal {
		t.Fatal("node.IsLocal = false, want true")
	}
	if node.Status != StatusActive {
		t.Fatalf("node.Status = %s, want active", node.Status)
	}
	if node.Hostname != "127.0.0.1" {
		t.Fatalf("node.Hostname = %s, want 127.0.0.1", node.Hostname)
	}
}

func TestEnsureSelfNodeReturnsExistingWhenPresent(t *testing.T) {
	now := time.Date(2026, 2, 22, 2, 0, 0, 0, time.UTC)
	store := NewInMemoryStore([]Node{
		{ID: SelfNodeID, Name: SelfNodeName, Hostname: "127.0.0.1", Status: StatusActive, IsLocal: true, CreatedAt: now},
	})

	node, err := store.EnsureSelfNode(context.Background(), now.Add(time.Hour))
	if err != nil {
		t.Fatalf("EnsureSelfNode() error = %v", err)
	}
	if node.ID != SelfNodeID {
		t.Fatalf("node.ID = %s, want %s", node.ID, SelfNodeID)
	}
	if node.CreatedAt != now {
		t.Fatalf("node.CreatedAt changed unexpectedly")
	}
}

func TestListNodesReturnsAllNodes(t *testing.T) {
	now := time.Date(2026, 2, 22, 2, 0, 0, 0, time.UTC)
	store := NewInMemoryStore([]Node{
		{ID: "node-1", Name: "test-1", Hostname: "192.168.1.1", Status: StatusActive, CreatedAt: now, UpdatedAt: now},
		{ID: "node-2", Name: "test-2", Hostname: "192.168.1.2", Status: StatusProvisioning, CreatedAt: now, UpdatedAt: now},
		{ID: SelfNodeID, Name: SelfNodeName, Hostname: "127.0.0.1", Status: StatusActive, IsLocal: true, CreatedAt: now, UpdatedAt: now},
	})

	nodes, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(nodes) != 3 {
		t.Fatalf("len(nodes) = %d, want 3", len(nodes))
	}
}

func TestListNodesReturnsEmptyWhenNoNodes(t *testing.T) {
	store := NewInMemoryStore(nil)

	nodes, err := store.List(context.Background())
	if err != nil {
		t.Fatalf("List() error = %v", err)
	}
	if len(nodes) != 0 {
		t.Fatalf("len(nodes) = %d, want 0", len(nodes))
	}
}

func TestCreateNodePersistsRemoteTarget(t *testing.T) {
	now := time.Date(2026, 2, 22, 3, 0, 0, 0, time.UTC)
	store := NewInMemoryStore(nil)

	node, err := store.Create(context.Background(), CreateInput{
		Name:     "remote-1",
		Hostname: "203.0.113.10",
		PublicIP: "203.0.113.10",
		SSHPort:  2222,
		SSHUser:  "ubuntu",
		IsLocal:  false,
		Now:      now,
	})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if node.ID == "" {
		t.Fatal("node.ID is empty")
	}
	if node.Status != StatusActive {
		t.Fatalf("node.Status = %s, want active", node.Status)
	}
	if node.IsLocal {
		t.Fatal("node.IsLocal = true, want false")
	}

	stored, err := store.GetByID(context.Background(), node.ID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if stored.Hostname != "203.0.113.10" {
		t.Fatalf("stored.Hostname = %s, want 203.0.113.10", stored.Hostname)
	}
	if stored.SSHPort != 2222 {
		t.Fatalf("stored.SSHPort = %d, want 2222", stored.SSHPort)
	}
}

func TestCreateLocalNodeFailsWhenLocalNodeAlreadyExists(t *testing.T) {
	now := time.Date(2026, 2, 22, 3, 5, 0, 0, time.UTC)
	store := NewInMemoryStore([]Node{{
		ID:        SelfNodeID,
		Name:      SelfNodeName,
		Hostname:  "127.0.0.1",
		Status:    StatusActive,
		IsLocal:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}})

	_, err := store.Create(context.Background(), CreateInput{
		Name:     "local-2",
		Hostname: "127.0.0.1",
		PublicIP: "127.0.0.1",
		SSHUser:  "pressluft",
		IsLocal:  true,
		Now:      now,
	})
	if !errors.Is(err, ErrLocalNodeExists) {
		t.Fatalf("Create() error = %v, want ErrLocalNodeExists", err)
	}
}

func TestUpsertLocalUpdatesExistingLocalNode(t *testing.T) {
	now := time.Date(2026, 2, 22, 3, 10, 0, 0, time.UTC)
	store := NewInMemoryStore([]Node{{
		ID:        SelfNodeID,
		Name:      SelfNodeName,
		Hostname:  "127.0.0.1",
		PublicIP:  "127.0.0.1",
		SSHPort:   22,
		SSHUser:   "pressluft",
		Status:    StatusActive,
		IsLocal:   true,
		CreatedAt: now,
		UpdatedAt: now,
	}})

	updated, err := store.UpsertLocal(context.Background(), CreateInput{
		Name:              "local-vm",
		Hostname:          "192.0.2.10",
		PublicIP:          "192.0.2.10",
		SSHPort:           2222,
		SSHUser:           "ubuntu",
		SSHPrivateKeyPath: "/tmp/local-vm-key",
		Now:               now.Add(5 * time.Minute),
	})
	if err != nil {
		t.Fatalf("UpsertLocal() error = %v", err)
	}
	if updated.ID != SelfNodeID {
		t.Fatalf("updated.ID = %s, want %s", updated.ID, SelfNodeID)
	}
	if updated.Hostname != "192.0.2.10" {
		t.Fatalf("updated.Hostname = %s, want 192.0.2.10", updated.Hostname)
	}
	if updated.SSHUser != "ubuntu" {
		t.Fatalf("updated.SSHUser = %s, want ubuntu", updated.SSHUser)
	}
}

func TestUpsertLocalCreatesWhenMissing(t *testing.T) {
	now := time.Date(2026, 2, 22, 3, 15, 0, 0, time.UTC)
	store := NewInMemoryStore(nil)

	created, err := store.UpsertLocal(context.Background(), CreateInput{
		Name:     "local-vm",
		Hostname: "198.51.100.12",
		PublicIP: "198.51.100.12",
		SSHUser:  "ubuntu",
		Now:      now,
	})
	if err != nil {
		t.Fatalf("UpsertLocal() error = %v", err)
	}
	if created.ID == "" {
		t.Fatal("created.ID is empty")
	}
	if !created.IsLocal {
		t.Fatal("created.IsLocal = false, want true")
	}
	if created.SSHPort != 22 {
		t.Fatalf("created.SSHPort = %d, want 22", created.SSHPort)
	}
}
