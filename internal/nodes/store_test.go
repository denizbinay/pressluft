package nodes

import (
	"context"
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
