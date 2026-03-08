package commands

import (
	"context"
	"errors"
	"os/exec"

	"pressluft/internal/agentcommand"
	"pressluft/internal/ws"
)

var commandContext = exec.CommandContext

func RestartService(ctx context.Context, cmd ws.Command) ws.CommandResult {
	params, err := agentcommand.DecodeRestartServicePayload(cmd.Payload)
	if err != nil {
		var validationErr *agentcommand.ValidationError
		if errors.As(err, &validationErr) {
			return ws.FailureResult(cmd.ID, validationErr.Code, validationErr.Message, nil, "")
		}
		return ws.FailureResult(cmd.ID, agentcommand.ErrorCodeInvalidPayload, "invalid restart_service payload", nil, "")
	}

	out, err := commandContext(ctx, "systemctl", "restart", params.ServiceName).CombinedOutput()
	if err != nil {
		code := agentcommand.ErrorCodeExecutionFailed
		message := err.Error()
		if errors.Is(err, context.DeadlineExceeded) || errors.Is(ctx.Err(), context.DeadlineExceeded) {
			code = agentcommand.ErrorCodeCommandTimedOut
			message = "command timed out"
		}
		return ws.FailureResult(cmd.ID, code, message, agentcommand.RestartServiceResult{ServiceName: params.ServiceName, Action: "restart_failed"}, string(out))
	}

	return ws.SuccessResult(cmd.ID, agentcommand.RestartServiceResult{ServiceName: params.ServiceName, Action: "restarted"}, string(out))
}
