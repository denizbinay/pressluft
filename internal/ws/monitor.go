package ws

import (
	"context"
	"log/slog"
	"time"
)

type ServerStore interface {
	UpdateNodeStatus(ctx context.Context, serverID int64, status, lastSeen, version string) error
}

type Monitor struct {
	hub                *Hub
	store              ServerStore
	unhealthyThreshold time.Duration
	offlineThreshold   time.Duration
	logger             *slog.Logger
}

func NewMonitor(hub *Hub, store ServerStore, logger *slog.Logger) *Monitor {
	return &Monitor{
		hub:                hub,
		store:              store,
		unhealthyThreshold: 45 * time.Second,
		offlineThreshold:   150 * time.Second,
		logger:             logger,
	}
}

func (m *Monitor) Start(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			m.checkConnections()
		}
	}
}

func (m *Monitor) checkConnections() {
	now := time.Now()

	m.hub.Range(func(serverID int64, conn *Conn) bool {
		lastSeen := conn.LastSeen()
		elapsed := now.Sub(lastSeen)
		version := conn.Version()

		if elapsed > m.offlineThreshold {
			m.logger.Info("node offline, closing connection", "server_id", serverID, "elapsed", elapsed)
			_ = m.store.UpdateNodeStatus(context.Background(), serverID, "offline", lastSeen.UTC().Format(time.RFC3339), version)
			conn.Close()
			m.hub.Unregister(serverID)
		} else if elapsed > m.unhealthyThreshold {
			m.logger.Debug("node unhealthy", "server_id", serverID, "elapsed", elapsed)
			_ = m.store.UpdateNodeStatus(context.Background(), serverID, "unhealthy", lastSeen.UTC().Format(time.RFC3339), version)
		}

		return true
	})
}
