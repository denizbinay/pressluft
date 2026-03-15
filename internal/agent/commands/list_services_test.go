package commands

import (
	"context"
	"encoding/json"
	"os/exec"
	"testing"

	"pressluft/internal/agent/agentcommand"
	"pressluft/internal/shared/ws"
)

func TestParseSystemctlOutput_MultipleServices(t *testing.T) {
	input := []byte(
		"nginx.service loaded active running A high performance web server\n" +
			"php8.3-fpm.service loaded active running The PHP FastCGI Process Manager\n" +
			"mariadb.service loaded active running MariaDB database server\n",
	)

	services := parseSystemctlOutput(input)
	if len(services) != 3 {
		t.Fatalf("got %d services, want 3", len(services))
	}

	tests := []struct {
		name        string
		description string
		loadState   string
		activeState string
	}{
		{"nginx", "A high performance web server", "loaded", "active"},
		{"php8.3-fpm", "The PHP FastCGI Process Manager", "loaded", "active"},
		{"mariadb", "MariaDB database server", "loaded", "active"},
	}

	for i, tt := range tests {
		if services[i].Name != tt.name {
			t.Errorf("services[%d].Name = %q, want %q", i, services[i].Name, tt.name)
		}
		if services[i].Description != tt.description {
			t.Errorf("services[%d].Description = %q, want %q", i, services[i].Description, tt.description)
		}
		if services[i].LoadState != tt.loadState {
			t.Errorf("services[%d].LoadState = %q, want %q", i, services[i].LoadState, tt.loadState)
		}
		if services[i].ActiveState != tt.activeState {
			t.Errorf("services[%d].ActiveState = %q, want %q", i, services[i].ActiveState, tt.activeState)
		}
	}
}

func TestParseSystemctlOutput_EmptyInput(t *testing.T) {
	services := parseSystemctlOutput([]byte(""))
	if len(services) != 0 {
		t.Fatalf("got %d services, want 0", len(services))
	}
}

func TestParseSystemctlOutput_BlankLines(t *testing.T) {
	input := []byte("\n  \n\nnginx.service loaded active running Web Server\n\n")
	services := parseSystemctlOutput(input)
	if len(services) != 1 {
		t.Fatalf("got %d services, want 1", len(services))
	}
	if services[0].Name != "nginx" {
		t.Errorf("Name = %q, want nginx", services[0].Name)
	}
}

func TestParseSystemctlOutput_SkipsShortLines(t *testing.T) {
	input := []byte("too few fields\nnginx.service loaded active running Web Server\n")
	services := parseSystemctlOutput(input)
	if len(services) != 1 {
		t.Fatalf("got %d services, want 1", len(services))
	}
}

func TestParseSystemctlOutput_StripsServiceSuffix(t *testing.T) {
	input := []byte("redis-server.service loaded active running Redis\n")
	services := parseSystemctlOutput(input)
	if len(services) != 1 {
		t.Fatalf("got %d services, want 1", len(services))
	}
	if services[0].Name != "redis-server" {
		t.Errorf("Name = %q, want redis-server", services[0].Name)
	}
}

func TestParseSystemctlOutput_NoServiceSuffix(t *testing.T) {
	input := []byte("dbus loaded active running D-Bus System Message Bus\n")
	services := parseSystemctlOutput(input)
	if len(services) != 1 {
		t.Fatalf("got %d services, want 1", len(services))
	}
	if services[0].Name != "dbus" {
		t.Errorf("Name = %q, want dbus", services[0].Name)
	}
}

func TestListServices_Success(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("echo", "nginx.service loaded active running Web Server")
	}

	result := ListServices(context.Background(), ws.Command{ID: "cmd-ls-1"})
	if !result.Success {
		t.Fatalf("expected success, got error: %s", result.Error)
	}

	var lr agentcommand.ListServicesResult
	if err := json.Unmarshal(result.Payload, &lr); err != nil {
		t.Fatalf("unmarshal payload: %v", err)
	}
	if len(lr.Services) != 1 {
		t.Fatalf("got %d services, want 1", len(lr.Services))
	}
	if lr.Services[0].Name != "nginx" {
		t.Errorf("Name = %q, want nginx", lr.Services[0].Name)
	}
}

func TestListServices_CommandFailure(t *testing.T) {
	original := commandContext
	defer func() { commandContext = original }()

	commandContext = func(ctx context.Context, name string, args ...string) *exec.Cmd {
		return exec.Command("false")
	}

	result := ListServices(context.Background(), ws.Command{ID: "cmd-ls-fail"})
	if result.Success {
		t.Fatal("expected failure")
	}
	if result.ErrorCode != agentcommand.ErrorCodeExecutionFailed {
		t.Errorf("ErrorCode = %q, want %q", result.ErrorCode, agentcommand.ErrorCodeExecutionFailed)
	}
}
