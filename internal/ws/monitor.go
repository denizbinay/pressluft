package ws

import (
	"context"
	"log/slog"
	"time"

	"pressluft/internal/observability"
	"pressluft/internal/platform"
)

type ServerStore interface {
	UpdateNodeStatus(ctx context.Context, serverID int64, status platform.NodeStatus, lastSeen, version string) error
	MarkNodesOfflineBefore(ctx context.Context, cutoff time.Time) (int64, error)
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
		unhealthyThreshold: time.Duration(platform.NodeUnhealthyThresholdSeconds) * time.Second,
		offlineThreshold:   time.Duration(platform.NodeOfflineThresholdSeconds) * time.Second,
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
			m.logger.Info("node health transitioned", observability.Correlation{ServerID: serverID}.LogArgs("node_status", platform.NodeStatusOffline, "reason", "heartbeat_timeout", "elapsed", elapsed)...)
			if m.store != nil {
				_ = m.store.UpdateNodeStatus(context.Background(), serverID, platform.NodeStatusOffline, lastSeen.UTC().Format(time.RFC3339), version)
			}
			conn.Close()
			m.hub.Unregister(serverID)
		} else if elapsed > m.unhealthyThreshold {
			m.logger.Debug("node health transitioned", observability.Correlation{ServerID: serverID}.LogArgs("node_status", platform.NodeStatusUnhealthy, "reason", "heartbeat_degraded", "elapsed", elapsed)...)
			if m.store != nil {
				_ = m.store.UpdateNodeStatus(context.Background(), serverID, platform.NodeStatusUnhealthy, lastSeen.UTC().Format(time.RFC3339), version)
			}
		}

		return true
	})

	if m.store != nil {
		if marked, err := m.store.MarkNodesOfflineBefore(context.Background(), now.Add(-m.offlineThreshold)); err != nil {
			m.logger.Error("stale node offline sweep failed", "error", err)
		} else if marked > 0 {
			m.logger.Info("stale node offline sweep completed", "marked_offline", marked)
		}
	}
}
