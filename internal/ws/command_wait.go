package ws

import (
	"context"
	"errors"

	"pressluft/internal/agentcommand"
)

func (h *Hub) SendCommandAndWait(ctx context.Context, serverID int64, cmd Command) (CommandResult, error) {
	if cmd.ID == "" {
		return CommandResult{}, errors.New("command id is required")
	}

	waiter := h.resultWaiter()
	if waiter == nil {
		return CommandResult{}, errors.New("command waiter not configured")
	}

	conn, ok := h.Get(serverID)
	if !ok {
		return CommandResult{}, errors.New("agent not connected")
	}

	normalizedPayload, err := agentcommand.Validate(cmd.Type, cmd.Payload)
	if err != nil {
		return CommandResult{}, err
	}
	cmd.Payload = normalizedPayload

	ch := waiter.Register(cmd.ID)
	payload, err := marshalJSON(cmd)
	if err != nil {
		waiter.Cancel(cmd.ID)
		return CommandResult{}, err
	}
	env := Envelope{Type: TypeCommand, Payload: payload}

	if err := conn.Send(ctx, env); err != nil {
		waiter.Cancel(cmd.ID)
		return CommandResult{}, err
	}

	select {
	case result := <-ch:
		return result, nil
	case <-ctx.Done():
		waiter.Cancel(cmd.ID)
		return CommandResult{}, ctx.Err()
	}
}
