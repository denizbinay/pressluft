//go:build !dev

package agent

import (
	"context"
	"crypto/tls"
	"encoding/json"
	"fmt"
	"net/http"

	"pressluft/internal/ws"

	"nhooyr.io/websocket"
)

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
