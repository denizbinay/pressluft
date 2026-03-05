package ws

import (
	"context"
	"errors"
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

	ch := waiter.Register(cmd.ID)
	env := Envelope{
		Type:    TypeCommand,
		Payload: mustMarshal(cmd),
	}

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
