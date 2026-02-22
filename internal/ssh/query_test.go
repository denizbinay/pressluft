package ssh

import (
	"context"
	"os/exec"
	"testing"
	"time"
)

func TestRunnerWordPressVersion_Success(t *testing.T) {
	runner := &Runner{
		timeout: 5 * time.Second,
		execCmd: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "echo", "6.4.3")
		},
	}

	version, err := runner.WordPressVersion(context.Background(), "127.0.0.1", 22, "pressluft", "mysite", "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if version != "6.4.3" {
		t.Errorf("expected version 6.4.3, got %s", version)
	}
}

func TestRunnerWordPressVersion_EmptyOutput(t *testing.T) {
	runner := &Runner{
		timeout: 5 * time.Second,
		execCmd: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "echo", "")
		},
	}

	_, err := runner.WordPressVersion(context.Background(), "127.0.0.1", 22, "pressluft", "mysite", "production")
	if err == nil {
		t.Fatal("expected error for empty output")
	}
}

func TestRunnerWordPressVersion_CommandFailure(t *testing.T) {
	runner := &Runner{
		timeout: 5 * time.Second,
		execCmd: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "false")
		},
	}

	_, err := runner.WordPressVersion(context.Background(), "127.0.0.1", 22, "pressluft", "mysite", "production")
	if err == nil {
		t.Fatal("expected error for failed command")
	}
}

func TestLocalRunnerWordPressVersion_Success(t *testing.T) {
	runner := &LocalRunner{
		timeout: 5 * time.Second,
		execCmd: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "echo", "6.5.0")
		},
	}

	version, err := runner.WordPressVersion(context.Background(), "127.0.0.1", 22, "pressluft", "mysite", "production")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if version != "6.5.0" {
		t.Errorf("expected version 6.5.0, got %s", version)
	}
}

func TestLocalRunnerWordPressVersion_CommandFailure(t *testing.T) {
	runner := &LocalRunner{
		timeout: 5 * time.Second,
		execCmd: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "false")
		},
	}

	_, err := runner.WordPressVersion(context.Background(), "127.0.0.1", 22, "pressluft", "mysite", "production")
	if err == nil {
		t.Fatal("expected error for failed command")
	}
}

func TestRunnerCheckNodePrerequisitesUnreachableNode(t *testing.T) {
	runner := &Runner{
		timeout: 5 * time.Second,
		execCmd: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "false")
		},
	}

	reasons, err := runner.CheckNodePrerequisites(context.Background(), "127.0.0.1", 22, "pressluft", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reasons) != 1 || reasons[0] != "node_unreachable" {
		t.Fatalf("reasons = %v, want [node_unreachable]", reasons)
	}
}

func TestLocalRunnerCheckNodePrerequisites_MissingTools(t *testing.T) {
	runner := &LocalRunner{
		timeout: 5 * time.Second,
		execCmd: func(ctx context.Context, name string, args ...string) *exec.Cmd {
			return exec.CommandContext(ctx, "false")
		},
	}

	reasons, err := runner.CheckNodePrerequisites(context.Background(), "127.0.0.1", 22, "pressluft", true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(reasons) != 2 {
		t.Fatalf("reasons len = %d, want 2", len(reasons))
	}
}
