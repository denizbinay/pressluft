package ws

import (
	"context"
	"sync"
)

type Hub struct {
	conns map[int64]*Conn
	mu    sync.RWMutex
}

func NewHub() *Hub {
	return &Hub{
		conns: make(map[int64]*Conn),
	}
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
	defer h.mu.RUnlock()

	for id, conn := range h.conns {
		if !fn(id, conn) {
			break
		}
	}
}
