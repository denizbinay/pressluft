//go:build dev

package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"

	"pressluft/internal/ws"

	"nhooyr.io/websocket"
)

func (a *Agent) connectAndRun(ctx context.Context) error {
	token := strings.TrimSpace(a.config.ResolveDevWSToken())
	if token == "" {
		return fmt.Errorf("dev_ws_token is required for dev agent")
	}
	wsURL, err := a.config.websocketURL()
	if err != nil {
		return err
	}

	conn, _, err := websocket.Dial(ctx, wsURL, &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin":                {a.config.ControlPlane},
			"X-Pressluft-Dev-Token": {token},
		},
	})
	if err != nil {
		return fmt.Errorf("dial websocket: %w", err)
	}
	a.conn = conn

	a.logger.Info("agent websocket connected", "server_id", a.config.ServerID, "control_plane", a.config.ControlPlane, "transport", "ws")

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
