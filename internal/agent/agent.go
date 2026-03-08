package agent

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"math/rand"
	"time"

	"pressluft/internal/observability"
	"pressluft/internal/ws"

	"nhooyr.io/websocket"
)

type Agent struct {
	config   *Config
	conn     *websocket.Conn
	executor *Executor
	logger   *slog.Logger
}

func New(config *Config, logger *slog.Logger) *Agent {
	return &Agent{
		config:   config,
		executor: NewExecutor(),
		logger:   logger,
	}
}

func (a *Agent) Run(ctx context.Context) error {
	a.logger.Info("agent runtime started", "server_id", a.config.ServerID, "control_plane", a.config.ControlPlane)
	if err := a.bootstrap(ctx); err != nil {
		return err
	}

	attempt := 0
	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := a.connectAndRun(ctx); err != nil {
			if ctx.Err() != nil {
				return ctx.Err()
			}
			a.logger.Error("agent connection loop failed", "server_id", a.config.ServerID, "error", err)
			attempt++
			delay := retryDelay(attempt)
			a.logger.Info("agent reconnect scheduled", "server_id", a.config.ServerID, "attempt", attempt, "delay", delay)
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
			}
			continue
		}
		attempt = 0
	}
}

func (a *Agent) sendHeartbeats(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	// Send initial heartbeat immediately on connect
	a.sendHeartbeat(ctx)

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			a.sendHeartbeat(ctx)
		}
	}
}

func (a *Agent) sendHeartbeat(ctx context.Context) {
	metrics := CollectMetrics()

	payload, err := json.Marshal(ws.Heartbeat{
		Timestamp:  time.Now(),
		Version:    "1.0.0",
		CPUPercent: metrics.CPUPercent,
		MemUsedMB:  metrics.MemUsedMB,
		MemTotalMB: metrics.MemTotalMB,
	})
	if err != nil {
		a.logger.Error("agent heartbeat encode failed", "server_id", a.config.ServerID, "error", err)
		return
	}

	env := ws.Envelope{
		Type:    ws.TypeHeartbeat,
		Payload: payload,
	}
	message, err := json.Marshal(env)
	if err != nil {
		a.logger.Error("agent heartbeat envelope encode failed", "server_id", a.config.ServerID, "error", err)
		return
	}
	if err := a.conn.Write(ctx, websocket.MessageText, message); err != nil {
		a.logger.Debug("agent heartbeat send failed", "server_id", a.config.ServerID, "error", err)
	}
}

func (a *Agent) handleMessage(ctx context.Context, env ws.Envelope) {
	switch env.Type {
	case ws.TypeCommand:
		var cmd ws.Command
		if err := json.Unmarshal(env.Payload, &cmd); err != nil {
			a.logger.Error("command decode failed", "server_id", a.config.ServerID, "error", err)
			return
		}
		if cmd.ServerID == 0 {
			cmd.ServerID = a.config.ServerID
		}
		corr := observability.Correlation{JobID: cmd.JobID, ServerID: cmd.ServerID, CommandID: cmd.ID}
		a.logger.Info("command execution started", corr.LogArgs("command_type", cmd.Type)...)

		result := a.executor.Execute(ctx, cmd)
		if result.JobID == 0 {
			result.JobID = cmd.JobID
		}
		if result.ServerID == 0 {
			result.ServerID = cmd.ServerID
		}
		payload, err := json.Marshal(result)
		if err != nil {
			a.logger.Error("command result encode failed", corr.LogArgs("error", err)...)
			return
		}

		resultEnv := ws.Envelope{
			Type:    ws.TypeCommandResult,
			Payload: payload,
		}

		message, err := json.Marshal(resultEnv)
		if err != nil {
			a.logger.Error("command result envelope encode failed", corr.LogArgs("error", err)...)
			return
		}
		if err := a.conn.Write(ctx, websocket.MessageText, message); err != nil && !errors.Is(err, context.Canceled) {
			a.logger.Debug("command result send failed", corr.LogArgs("error", err)...)
			return
		}
		a.logger.Info("command execution finished", observability.Correlation{JobID: result.JobID, ServerID: result.ServerID, CommandID: result.CommandID}.LogArgs("success", result.Success, "error_code", result.ErrorCode)...)
	case ws.TypeHeartbeatAck:
		return
	}
}

func retryDelay(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	base := time.Second << min(attempt-1, 5)
	if base > 30*time.Second {
		base = 30 * time.Second
	}
	jitter := time.Duration(rand.Int63n(int64(base / 2)))
	return base + jitter
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
