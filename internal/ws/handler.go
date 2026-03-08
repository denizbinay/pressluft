package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"pressluft/internal/observability"
	"pressluft/internal/platform"
)

type Completer interface {
	HandleResult(CommandResult) error
	HandleLogEntry(LogEntry) error
}

type Handler struct {
	hub       *Hub
	completer Completer
	waiter    *ResultWaiter
	store     NodeStatusStore
	logger    *slog.Logger
}

type NodeStatusStore interface {
	UpdateNodeStatus(ctx context.Context, serverID int64, status platform.NodeStatus, lastSeen, version string) error
}

func NewHandler(hub *Hub, completer Completer, waiter *ResultWaiter, store NodeStatusStore, logger *slog.Logger) *Handler {
	return &Handler{
		hub:       hub,
		completer: completer,
		waiter:    waiter,
		store:     store,
		logger:    logger,
	}
}

func (h *Handler) HandleConnection(ctx context.Context, conn *Conn) {
	corr := observability.Correlation{ServerID: conn.ServerID()}
	h.logger.Info("agent websocket session opened", corr.LogArgs("node_status", platform.NodeStatusOnline)...)
	h.persistNodeStatus(context.Background(), conn.ServerID(), platform.NodeStatusOnline, conn.LastSeen(), conn.Version(), "connect")
	defer func() {
		if recovered := recover(); recovered != nil {
			h.logger.Error("agent websocket session panicked", corr.LogArgs("panic", recovered)...)
		}
		h.persistNodeStatus(context.Background(), conn.ServerID(), platform.NodeStatusUnhealthy, conn.LastSeen(), conn.Version(), "disconnect")
		h.hub.Unregister(conn.ServerID())
		if err := conn.Close(); err != nil {
			h.logger.Debug("agent websocket close failed", corr.LogArgs("error", err)...)
		}
		h.logger.Info("agent websocket session closed", corr.LogArgs("node_status", platform.NodeStatusUnhealthy)...)
	}()

	for {
		env, err := conn.Receive(ctx)
		if err != nil {
			h.logger.Debug("agent websocket receive failed", corr.LogArgs("error", err)...)
			return
		}

		h.handleMessage(ctx, conn, env)
	}
}

func (h *Handler) handleMessage(ctx context.Context, conn *Conn, env Envelope) {
	switch env.Type {
	case TypeHeartbeat:
		h.handleHeartbeat(ctx, conn, env)
	case TypeCommandResult:
		h.handleCommandResult(ctx, conn, env)
	case TypeLogEntry:
		h.handleLogEntry(ctx, conn, env)
	default:
		h.logger.Debug("agent websocket message ignored", "server_id", conn.ServerID(), "message_type", env.Type)
	}
}

func (h *Handler) handleHeartbeat(ctx context.Context, conn *Conn, env Envelope) {
	var hb Heartbeat
	if err := json.Unmarshal(env.Payload, &hb); err != nil {
		h.logger.Debug("agent heartbeat decode failed", "server_id", conn.ServerID(), "error", err)
		return
	}

	// Update connection state with heartbeat data including metrics
	conn.UpdateFromHeartbeat(hb)

	ack := Envelope{
		Type:    TypeHeartbeatAck,
		Payload: nil,
	}
	ackPayload, err := marshalJSON(Heartbeat{Timestamp: time.Now()})
	if err != nil {
		h.logger.Error("agent heartbeat ack encode failed", "server_id", conn.ServerID(), "error", err)
		return
	}
	ack.Payload = ackPayload
	h.persistNodeStatus(context.Background(), conn.ServerID(), platform.NodeStatusOnline, conn.LastSeen(), conn.Version(), "heartbeat")
	h.logger.Debug("agent heartbeat received", "server_id", conn.ServerID(), "node_status", platform.NodeStatusOnline, "version", conn.Version())

	_ = conn.Send(ctx, ack)
}

func (h *Handler) handleCommandResult(ctx context.Context, conn *Conn, env Envelope) {
	var result CommandResult
	if err := json.Unmarshal(env.Payload, &result); err != nil {
		h.logger.Debug("command result decode failed", "server_id", conn.ServerID(), "error", err)
		return
	}
	if result.ServerID == 0 {
		result.ServerID = conn.ServerID()
	}
	h.logger.Info("command result received", observability.Correlation{JobID: result.JobID, ServerID: result.ServerID, CommandID: result.CommandID}.LogArgs("success", result.Success, "error_code", result.ErrorCode)...)

	if h.waiter != nil && h.waiter.Resolve(result) {
		return
	}

	if h.completer == nil {
		return
	}

	if err := h.completer.HandleResult(result); err != nil {
		h.logger.Error("command result handling failed", observability.Correlation{JobID: result.JobID, ServerID: result.ServerID, CommandID: result.CommandID}.LogArgs("error", err)...)
	}
}

func (h *Handler) handleLogEntry(ctx context.Context, conn *Conn, env Envelope) {
	var entry LogEntry
	if err := json.Unmarshal(env.Payload, &entry); err != nil {
		h.logger.Debug("command log entry decode failed", "server_id", conn.ServerID(), "error", err)
		return
	}
	if entry.ServerID == 0 {
		entry.ServerID = conn.ServerID()
	}
	h.logger.Debug("command log entry received", observability.Correlation{JobID: entry.JobID, ServerID: entry.ServerID, CommandID: entry.CommandID}.LogArgs("level", entry.Level, "message", entry.Message)...)

	if h.completer == nil {
		return
	}

	if err := h.completer.HandleLogEntry(entry); err != nil {
		h.logger.Error("command log entry handling failed", observability.Correlation{JobID: entry.JobID, ServerID: entry.ServerID, CommandID: entry.CommandID}.LogArgs("error", err)...)
	}
}

func (h *Handler) persistNodeStatus(ctx context.Context, serverID int64, status platform.NodeStatus, lastSeen time.Time, version string, reason string) {
	if h.store == nil {
		return
	}
	if err := h.store.UpdateNodeStatus(ctx, serverID, status, lastSeen.UTC().Format(time.RFC3339), version); err != nil {
		h.logger.Error("node status persistence failed", "server_id", serverID, "node_status", status, "reason", reason, "error", err)
		return
	}
	h.logger.Debug("node status persisted", "server_id", serverID, "node_status", status, "reason", reason, "version", version)
}
