package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"time"

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
	if a.config.RegistrationToken != "" && !a.config.IsRegistered() {
		a.logger.Info("registering agent with control plane")
		if err := Register(a.config, ""); err != nil {
			return fmt.Errorf("registration failed: %w", err)
		}
		a.logger.Info("registration successful")
	}

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		if err := a.connectAndRun(ctx); err != nil {
			a.logger.Error("connection error", "error", err)
		}

		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-time.After(time.Second):
		}
	}
}

func (a *Agent) connectAndRun(ctx context.Context) error {
	cert, err := LoadClientCert(a.config)
	if err != nil {
		return fmt.Errorf("load client cert: %w", err)
	}

	caPool, err := LoadCACertPool(a.config)
	if err != nil {
		return fmt.Errorf("load CA cert: %w", err)
	}

	tlsConfig := &tls.Config{
		Certificates: []tls.Certificate{cert},
		RootCAs:      caPool,
	}

	httpClient := &http.Client{
		Transport: &http.Transport{
			TLSClientConfig: tlsConfig,
		},
	}

	conn, _, err := websocket.Dial(ctx, a.config.ControlPlane+"/ws/agent", &websocket.DialOptions{
		HTTPClient: httpClient,
		HTTPHeader: http.Header{
			"Origin": {a.config.ControlPlane},
		},
	})
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}
	a.conn = conn

	a.logger.Info("connected to control plane")

	go a.sendHeartbeats(ctx)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var env ws.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			a.logger.Debug("unmarshal error", "error", err)
			continue
		}

		a.handleMessage(ctx, env)
	}
}

func (a *Agent) sendHeartbeats(ctx context.Context) {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			env := ws.Envelope{
				Type: ws.TypeHeartbeat,
				Payload: mustMarshal(ws.Heartbeat{
					Timestamp: time.Now(),
					Version:   "1.0.0",
				}),
			}
			if err := a.conn.Write(ctx, websocket.MessageText, mustMarshal(env)); err != nil {
				a.logger.Debug("heartbeat error", "error", err)
			}
		}
	}
}

func (a *Agent) handleMessage(ctx context.Context, env ws.Envelope) {
	switch env.Type {
	case ws.TypeCommand:
		var cmd ws.Command
		if err := json.Unmarshal(env.Payload, &cmd); err != nil {
			a.logger.Error("unmarshal command", "error", err)
			return
		}

		result := a.executor.Execute(ctx, cmd)

		resultEnv := ws.Envelope{
			Type:    ws.TypeCommandResult,
			Payload: mustMarshal(result),
		}

		_ = a.conn.Write(ctx, websocket.MessageText, mustMarshal(resultEnv))
	}
}

func mustMarshal(v any) []byte {
	data, err := json.Marshal(v)
	if err != nil {
		panic(err)
	}
	return data
}
