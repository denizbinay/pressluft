package ws

import (
	"context"
	"encoding/json"
	"io"
	"log/slog"
	"sync"
	"testing"

	"pressluft/internal/platform"
)

type nodeStatusUpdate struct {
	serverID int64
	status   platform.NodeStatus
	lastSeen string
	version  string
}

type recordingNodeStatusStore struct {
	mu      sync.Mutex
	updates []nodeStatusUpdate
}

func (s *recordingNodeStatusStore) UpdateNodeStatus(_ context.Context, serverID int64, status platform.NodeStatus, lastSeen, version string) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.updates = append(s.updates, nodeStatusUpdate{serverID: serverID, status: status, lastSeen: lastSeen, version: version})
	return nil
}

func (s *recordingNodeStatusStore) latest() nodeStatusUpdate {
	s.mu.Lock()
	defer s.mu.Unlock()
	if len(s.updates) == 0 {
		return nodeStatusUpdate{}
	}
	return s.updates[len(s.updates)-1]
}

func TestHandlerHeartbeatPersistsOnlineAndUpdatesState(t *testing.T) {
	conn := NewConn(nil, 42)
	store := &recordingNodeStatusStore{}
	handler := NewHandler(NewHub(), nil, nil, store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	payload, err := json.Marshal(Heartbeat{Version: "1.2.3", CPUPercent: 10, MemUsedMB: 64, MemTotalMB: 128})
	if err != nil {
		t.Fatalf("marshal heartbeat: %v", err)
	}
	handler.handleHeartbeat(context.Background(), conn, Envelope{Type: TypeHeartbeat, Payload: payload})

	if got := conn.Version(); got != "1.2.3" {
		t.Fatalf("connection version = %q, want %q", got, "1.2.3")
	}
	cpuPercent, memUsedMB, memTotalMB := conn.Metrics()
	if cpuPercent != 10 || memUsedMB != 64 || memTotalMB != 128 {
		t.Fatalf("connection metrics = (%v, %d, %d), want (10, 64, 128)", cpuPercent, memUsedMB, memTotalMB)
	}
	if got := store.latest(); got.status != platform.NodeStatusOnline || got.version != "1.2.3" {
		t.Fatalf("latest update = %+v, want online with version", got)
	}
}

func TestHandleConnectionMarksNodeUnhealthyOnDisconnect(t *testing.T) {
	hub := NewHub()
	conn := NewConn(nil, 9)
	hub.Register(conn)
	store := &recordingNodeStatusStore{}
	handler := NewHandler(hub, nil, nil, store, slog.New(slog.NewTextHandler(io.Discard, nil)))

	handler.HandleConnection(context.Background(), conn)

	if _, ok := hub.Get(9); ok {
		t.Fatal("expected connection to be unregistered")
	}
	if got := store.latest(); got.status != platform.NodeStatusUnhealthy {
		t.Fatalf("latest update = %+v, want unhealthy", got)
	}
}
