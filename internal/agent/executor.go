package agent

import (
	"context"
	"errors"

	"pressluft/internal/agent/commands"
	"pressluft/internal/agentcommand"
	"pressluft/internal/ws"
)

type commandFunc func(context.Context, ws.Command) ws.CommandResult

type Executor struct {
	restartService commandFunc
	listServices   commandFunc
	siteHealth     commandFunc
}

func NewExecutor() *Executor {
	return &Executor{
		restartService: commands.RestartService,
		listServices:   commands.ListServices,
		siteHealth:     commands.SiteHealthSnapshot,
	}
}

func (e *Executor) Execute(ctx context.Context, cmd ws.Command) ws.CommandResult {
	normalizedPayload, err := agentcommand.Validate(cmd.Type, cmd.Payload)
	if err != nil {
		var validationErr *agentcommand.ValidationError
		if errors.As(err, &validationErr) {
			return ws.FailureResult(cmd.ID, validationErr.Code, validationErr.Message, nil, "")
		}
		return ws.FailureResult(cmd.ID, agentcommand.ErrorCodeInvalidPayload, err.Error(), nil, "")
	}
	cmd.Payload = normalizedPayload

	if timeout := agentcommand.Timeout(cmd.Type); timeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, timeout)
		defer cancel()
	}

	switch cmd.Type {
	case agentcommand.TypeRestartService:
		return e.restartService(ctx, cmd)
	case agentcommand.TypeListServices:
		return e.listServices(ctx, cmd)
	case agentcommand.TypeSiteHealth:
		return e.siteHealth(ctx, cmd)
	default:
		return ws.FailureResult(cmd.ID, agentcommand.ErrorCodeUnknownCommand, "unknown command", nil, "")
	}
}
