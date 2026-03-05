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
	token := strings.TrimSpace(a.config.DevWSToken)
	if token == "" {
		return fmt.Errorf("dev_ws_token is required for dev agent")
	}

	conn, _, err := websocket.Dial(ctx, a.config.ControlPlane+"/ws/agent", &websocket.DialOptions{
		HTTPHeader: http.Header{
			"Origin":                {a.config.ControlPlane},
			"X-Pressluft-Dev-Token": {token},
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
