package ws

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"
)

type Completer interface {
	HandleResult(CommandResult) error
	HandleLogEntry(LogEntry) error
}

type Handler struct {
	hub       *Hub
	completer Completer
	logger    *slog.Logger
}

func NewHandler(hub *Hub, completer Completer, logger *slog.Logger) *Handler {
	return &Handler{
		hub:       hub,
		completer: completer,
		logger:    logger,
	}
}

func (h *Handler) HandleConnection(ctx context.Context, conn *Conn) {
	defer func() {
		h.hub.Unregister(conn.ServerID())
		conn.Close()
	}()

	for {
		env, err := conn.Receive(ctx)
		if err != nil {
			h.logger.Debug("connection receive error", "server_id", conn.ServerID(), "error", err)
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
		h.logger.Debug("unknown message type", "type", env.Type)
	}
}

func (h *Handler) handleHeartbeat(ctx context.Context, conn *Conn, env Envelope) {
	conn.UpdateLastSeen()

	var hb Heartbeat
	if err := json.Unmarshal(env.Payload, &hb); err != nil {
		h.logger.Debug("unmarshal heartbeat error", "error", err)
		return
	}

	ack := Envelope{
		Type:    TypeHeartbeatAck,
		Payload: mustMarshal(Heartbeat{Timestamp: time.Now()}),
	}

	_ = conn.Send(ctx, ack)
}

func (h *Handler) handleCommandResult(ctx context.Context, conn *Conn, env Envelope) {
	var result CommandResult
	if err := json.Unmarshal(env.Payload, &result); err != nil {
		h.logger.Debug("unmarshal command result error", "error", err)
		return
	}

	if err := h.completer.HandleResult(result); err != nil {
		h.logger.Error("handle command result error", "error", err)
	}
}

func (h *Handler) handleLogEntry(ctx context.Context, conn *Conn, env Envelope) {
	var entry LogEntry
	if err := json.Unmarshal(env.Payload, &entry); err != nil {
		h.logger.Debug("unmarshal log entry error", "error", err)
		return
	}

	if err := h.completer.HandleLogEntry(entry); err != nil {
		h.logger.Error("handle log entry error", "error", err)
	}
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
