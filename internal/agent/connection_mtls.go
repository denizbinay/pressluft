//go:build !dev

package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"pressluft/internal/ws"

	"nhooyr.io/websocket"
)

func (a *Agent) connectAndRun(ctx context.Context) error {
	wsURL, err := a.config.websocketURL()
	if err != nil {
		return err
	}
	if !strings.HasPrefix(wsURL, "wss://") {
		return fmt.Errorf("production agent transport requires wss control_plane URL")
	}

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

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPClient: httpClient,
		HTTPHeader: http.Header{
			"Origin": {a.config.ControlPlane},
		},
	})
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}
	a.conn = conn

	a.logger.Info("agent websocket connected", "server_id", a.config.ServerID, "control_plane", a.config.ControlPlane, "transport", "wss+mTLS")

	go a.sendHeartbeats(ctx)

	for {
		_, data, err := conn.Read(ctx)
		if err != nil {
			return fmt.Errorf("read: %w", err)
		}

		var env ws.Envelope
		if err := json.Unmarshal(data, &env); err != nil {
			a.logger.Debug("agent websocket envelope decode failed", "server_id", a.config.ServerID, "error", err)
			continue
		}

		a.handleMessage(ctx, env)
	}
}
