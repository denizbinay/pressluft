package ws

import (
	"context"
	"sync"
	"time"

	"pressluft/internal/platform"
)

type Hub struct {
	conns  map[int64]*Conn
	mu     sync.RWMutex
	waiter *ResultWaiter
}

func NewHub() *Hub {
	return &Hub{
		conns: make(map[int64]*Conn),
	}
}

func (h *Hub) SetResultWaiter(waiter *ResultWaiter) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.waiter = waiter
}

func (h *Hub) resultWaiter() *ResultWaiter {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return h.waiter
}

func (h *Hub) Register(conn *Conn) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.conns[conn.ServerID()] = conn
}

func (h *Hub) Unregister(serverID int64) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.conns, serverID)
}

func (h *Hub) Get(serverID int64) (*Conn, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	conn, ok := h.conns[serverID]
	return conn, ok
}

func (h *Hub) Send(serverID int64, env Envelope) error {
	h.mu.RLock()
	conn, ok := h.conns[serverID]
	h.mu.RUnlock()

	if !ok {
		return nil
	}

	return conn.Send(context.Background(), env)
}

func (h *Hub) Broadcast(env Envelope) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, conn := range h.conns {
		_ = conn.Send(context.Background(), env)
	}
}

func (h *Hub) ConnectedServerIDs() []int64 {
	h.mu.RLock()
	defer h.mu.RUnlock()

	ids := make([]int64, 0, len(h.conns))
	for id := range h.conns {
		ids = append(ids, id)
	}
	return ids
}

func (h *Hub) Range(fn func(serverID int64, conn *Conn) bool) {
	h.mu.RLock()
	snapshot := make(map[int64]*Conn, len(h.conns))
	for id, conn := range h.conns {
		snapshot[id] = conn
	}
	h.mu.RUnlock()

	for id, conn := range snapshot {
		if !fn(id, conn) {
			break
		}
	}
}

// GetAgentInfo returns the real-time status and metrics for a server's agent.
// If the agent is not connected, it returns a disconnected status.
func (h *Hub) GetAgentInfo(serverID int64) AgentInfo {
	h.mu.RLock()
	conn, ok := h.conns[serverID]
	h.mu.RUnlock()

	if !ok {
		return AgentInfo{
			Connected: false,
			Status:    platform.NodeStatusOffline,
		}
	}

	lastSeen := conn.LastSeen()
	elapsed := time.Since(lastSeen)
	cpuPercent, memUsedMB, memTotalMB := conn.Metrics()

	info := AgentInfo{
		Connected:  true,
		LastSeen:   lastSeen,
		Version:    conn.Version(),
		CPUPercent: cpuPercent,
		MemUsedMB:  memUsedMB,
		MemTotalMB: memTotalMB,
	}

	// Determine status based on time since last heartbeat
	switch {
	case elapsed > time.Duration(platform.NodeOfflineThresholdSeconds)*time.Second:
		info.Status = platform.NodeStatusOffline
		info.Connected = false
	case elapsed > time.Duration(platform.NodeUnhealthyThresholdSeconds)*time.Second:
		info.Status = platform.NodeStatusUnhealthy
	default:
		info.Status = platform.NodeStatusOnline
	}

	return info
}

// GetAllAgentInfo returns agent info for all connected servers.
func (h *Hub) GetAllAgentInfo() map[int64]AgentInfo {
	h.mu.RLock()
	defer h.mu.RUnlock()

	result := make(map[int64]AgentInfo, len(h.conns))
	now := time.Now()

	for serverID, conn := range h.conns {
		lastSeen := conn.LastSeen()
		elapsed := now.Sub(lastSeen)
		cpuPercent, memUsedMB, memTotalMB := conn.Metrics()

		info := AgentInfo{
			Connected:  true,
			LastSeen:   lastSeen,
			Version:    conn.Version(),
			CPUPercent: cpuPercent,
			MemUsedMB:  memUsedMB,
			MemTotalMB: memTotalMB,
		}

		switch {
		case elapsed > time.Duration(platform.NodeOfflineThresholdSeconds)*time.Second:
			info.Status = platform.NodeStatusOffline
			info.Connected = false
		case elapsed > time.Duration(platform.NodeUnhealthyThresholdSeconds)*time.Second:
			info.Status = platform.NodeStatusUnhealthy
		default:
			info.Status = platform.NodeStatusOnline
		}

		result[serverID] = info
	}

	return result
}
