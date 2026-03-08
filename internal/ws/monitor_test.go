package ws

import (
	"context"
	"io"
	"log/slog"
	"testing"
	"time"

	"pressluft/internal/platform"
)

type monitorStore struct {
	updates      []nodeStatusUpdate
	offlineCalls int
}

func (s *monitorStore) UpdateNodeStatus(_ context.Context, serverID int64, status platform.NodeStatus, lastSeen, version string) error {
	s.updates = append(s.updates, nodeStatusUpdate{serverID: serverID, status: status, lastSeen: lastSeen, version: version})
	return nil
}

func (s *monitorStore) MarkNodesOfflineBefore(_ context.Context, cutoff time.Time) (int64, error) {
	s.offlineCalls++
	return 0, nil
}

func TestMonitorMarksStaleConnectionsUnhealthyAndOffline(t *testing.T) {
	hub := NewHub()
	store := &monitorStore{}
	monitor := NewMonitor(hub, store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	unhealthyConn := &Conn{serverID: 1, lastSeen: time.Now().Add(-50 * time.Second), version: "v1"}
	offlineConn := &Conn{serverID: 2, lastSeen: time.Now().Add(-3 * time.Minute), version: "v2"}
	hub.Register(unhealthyConn)
	hub.Register(offlineConn)

	monitor.checkConnections()

	if len(store.updates) < 2 {
		t.Fatalf("updates = %+v, want unhealthy and offline transitions", store.updates)
	}
	if _, ok := hub.Get(2); ok {
		t.Fatal("expected offline connection to be removed from hub")
	}
	if store.offlineCalls != 1 {
		t.Fatalf("offlineCalls = %d, want 1", store.offlineCalls)
	}
}
