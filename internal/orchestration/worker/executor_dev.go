//go:build dev

package worker

import (
	"context"
	"fmt"
	"time"
)

func (e *Executor) extraAgentVars(ctx context.Context, serverID string) (map[string]string, error) {
	if e.devTokenStore == nil {
		return nil, nil
	}

	token, err := e.devTokenStore.Create(serverID, 365*24*time.Hour)
	if err != nil {
		return nil, fmt.Errorf("create dev agent token: %w", err)
	}

	return map[string]string{
		"dev_ws_token": token,
	}, nil
}
