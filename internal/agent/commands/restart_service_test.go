package commands

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"pressluft/internal/agent/agentcommand"
	"pressluft/internal/shared/ws"
)

func TestRestartService_Success(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	payload, _ := json.Marshal(agentcommand.RestartServiceParams{ServiceName: "nginx"})
	result := RestartService(context.Background(), ws.Command{ID: "cmd-rs-1", Payload: payload})
	if !result.Success {
		t.Fatalf("expected success, got error: %s (code: %s)", result.Error, result.ErrorCode)
	}

	var rr agentcommand.RestartServiceResult
	if err := json.Unmarshal(result.Payload, &rr); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if rr.ServiceName != "nginx" {
		t.Errorf("ServiceName = %q, want nginx", rr.ServiceName)
	}
	if rr.Action != "restarted" {
		t.Errorf("Action = %q, want restarted", rr.Action)
	}
}

func TestRestartService_DisallowedService(t *testing.T) {
	payload, _ := json.Marshal(agentcommand.RestartServiceParams{ServiceName: "sshd"})
	result := RestartService(context.Background(), ws.Command{ID: "cmd-rs-2", Payload: payload})
	if result.Success {
		t.Fatal("expected failure for disallowed service")
	}
	if result.ErrorCode != agentcommand.ErrorCodeServiceNotAllowed {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, agentcommand.ErrorCodeServiceNotAllowed)
	}
}

func TestRestartService_EmptyPayload(t *testing.T) {
	result := RestartService(context.Background(), ws.Command{ID: "cmd-rs-3", Payload: nil})
	if result.Success {
		t.Fatal("expected failure for empty payload")
	}
	if result.ErrorCode != agentcommand.ErrorCodeInvalidPayload {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, agentcommand.ErrorCodeInvalidPayload)
	}
}

func TestRestartService_InvalidJSON(t *testing.T) {
	result := RestartService(context.Background(), ws.Command{ID: "cmd-rs-4", Payload: json.RawMessage(`{invalid}`)})
	if result.Success {
		t.Fatal("expected failure for invalid JSON")
	}
	if result.ErrorCode != agentcommand.ErrorCodeInvalidPayload {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, agentcommand.ErrorCodeInvalidPayload)
	}
}

func TestRestartService_MissingServiceName(t *testing.T) {
	payload, _ := json.Marshal(agentcommand.RestartServiceParams{ServiceName: ""})
	result := RestartService(context.Background(), ws.Command{ID: "cmd-rs-5", Payload: payload})
	if result.Success {
		t.Fatal("expected failure for missing service name")
	}
	if result.ErrorCode != agentcommand.ErrorCodeInvalidPayload {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, agentcommand.ErrorCodeInvalidPayload)
	}
}

func TestRestartService_InvalidServiceNameFormat(t *testing.T) {
	// Service name with slashes should fail the regex
	payload := json.RawMessage(`{"service_name":"../../etc/passwd"}`)
	result := RestartService(context.Background(), ws.Command{ID: "cmd-rs-6", Payload: payload})
	if result.Success {
		t.Fatal("expected failure for invalid service name format")
	}
	if result.ErrorCode != agentcommand.ErrorCodeInvalidServiceName {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, agentcommand.ErrorCodeInvalidServiceName)
	}
}

func TestRestartService_CommandFailure(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	payload, _ := json.Marshal(agentcommand.RestartServiceParams{ServiceName: "nginx"})
	result := RestartService(context.Background(), ws.Command{ID: "cmd-rs-7", Payload: payload})
	if result.Success {
		t.Fatal("expected failure when systemctl fails")
	}
	if result.ErrorCode != agentcommand.ErrorCodeExecutionFailed {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, agentcommand.ErrorCodeExecutionFailed)
	}

	var rr agentcommand.RestartServiceResult
	if err := json.Unmarshal(result.Payload, &rr); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if rr.Action != "restart_failed" {
		t.Errorf("Action = %q, want restart_failed", rr.Action)
	}
}

func TestRestartService_ContextTimeout(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // pre-cancel

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.CommandContext(ctx, "sleep", "10")
	}

	payload, _ := json.Marshal(agentcommand.RestartServiceParams{ServiceName: "nginx"})
	result := RestartService(ctx, ws.Command{ID: "cmd-rs-timeout", Payload: payload})
	if result.Success {
		t.Fatal("expected failure for cancelled context")
	}
}

func TestRestartService_AllAllowedServices(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	for _, svc := range agentcommand.AllowedServiceNames() {
		payload, _ := json.Marshal(agentcommand.RestartServiceParams{ServiceName: svc})
		result := RestartService(context.Background(), ws.Command{ID: "cmd-allowed-" + svc, Payload: payload})
		if !result.Success {
			t.Errorf("service %q: expected success, got error: %s (code: %s)", svc, result.Error, result.ErrorCode)
		}
	}
}

func TestRestartService_PreservesCommandID(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("true")
	}

	payload, _ := json.Marshal(agentcommand.RestartServiceParams{ServiceName: "nginx"})
	result := RestartService(context.Background(), ws.Command{ID: "my-unique-id", Payload: payload})
	if result.CommandID != "my-unique-id" {
		t.Errorf("CommandID = %q, want my-unique-id", result.CommandID)
	}
}
