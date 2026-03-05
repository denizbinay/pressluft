//go:build !dev

package worker

import "context"

func (e *Executor) extraAgentVars(ctx context.Context, serverID int64) (map[string]string, error) {
	return nil, nil
}
