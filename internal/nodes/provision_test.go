package nodes

import (
	"context"
	"errors"
	"io"
	"log"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"pressluft/internal/jobs"
)

func TestProvisionHandlerSuccessMarksNodeActive(t *testing.T) {
	now := time.Date(2026, 2, 22, 1, 10, 0, 0, time.UTC)
	nodeID := "node-1"
	store := NewInMemoryStore([]Node{{
		ID:        nodeID,
		Hostname:  "127.0.0.1",
		SSHPort:   22,
		SSHUser:   "ubuntu",
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}})

	handler := NewProvisionHandler(store, stubExecutor{err: nil}, log.New(io.Discard, "", 0))
	handler.now = func() time.Time { return now }

	err := handler.Handle(context.Background(), jobs.Job{ID: "job-1", JobType: "node_provision", NodeID: &nodeID})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	node, err := store.GetByID(context.Background(), nodeID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if node.Status != StatusActive {
		t.Fatalf("status = %s, want active", node.Status)
	}
}

func TestProvisionHandlerFailureMarksNodeUnreachable(t *testing.T) {
	now := time.Date(2026, 2, 22, 1, 11, 0, 0, time.UTC)
	nodeID := "node-1"
	store := NewInMemoryStore([]Node{{
		ID:        nodeID,
		Hostname:  "127.0.0.1",
		SSHPort:   22,
		SSHUser:   "ubuntu",
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}})

	handler := NewProvisionHandler(store, stubExecutor{err: jobs.ExecutionError{Code: "ANSIBLE_HOST_UNREACHABLE", Message: "timeout", Retryable: true}}, log.New(io.Discard, "", 0))
	handler.now = func() time.Time { return now }

	err := handler.Handle(context.Background(), jobs.Job{ID: "job-1", JobType: "node_provision", NodeID: &nodeID})
	if err == nil {
		t.Fatal("Handle() error = nil, want non-nil")
	}

	node, getErr := store.GetByID(context.Background(), nodeID)
	if getErr != nil {
		t.Fatalf("GetByID() error = %v", getErr)
	}
	if node.Status != StatusUnreachable {
		t.Fatalf("status = %s, want unreachable", node.Status)
	}
}

func TestMapAnsibleErrorDeadlineExceeded(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), time.Nanosecond)
	defer cancel()
	time.Sleep(time.Millisecond)

	err := mapAnsibleError(ctx, context.DeadlineExceeded, "timeout")
	var execErr jobs.ExecutionError
	if !errors.As(err, &execErr) {
		t.Fatalf("error type = %T, want jobs.ExecutionError", err)
	}
	if execErr.Code != "ANSIBLE_TIMEOUT" {
		t.Fatalf("code = %s, want ANSIBLE_TIMEOUT", execErr.Code)
	}
}

func TestAnsibleExecutorBuildsInventory(t *testing.T) {
	node := Node{
		ID:                "node-1",
		Hostname:          "example.local",
		SSHPort:           2202,
		SSHUser:           "ubuntu",
		SSHPrivateKeyPath: "/tmp/key",
	}

	inventory := buildInventory(node)
	for _, token := range []string{"[target]", "example.local", "ansible_port=2202", "ansible_user=ubuntu", "ansible_ssh_private_key_file=/tmp/key"} {
		if !strings.Contains(inventory, token) {
			t.Fatalf("inventory %q missing %q", inventory, token)
		}
	}
}

func TestAnsibleExecutorMapsExitCode(t *testing.T) {
	executor := NewAnsibleExecutor()
	executor.runCommand = func(context.Context, string, ...string) ([]byte, error) {
		cmd := exec.Command(os.Args[0], "-test.run=TestHelperProcess", "--", "exit4")
		cmd.Env = append(os.Environ(), "GO_WANT_HELPER_PROCESS=1")
		output, err := cmd.CombinedOutput()
		return output, err
	}

	err := executor.RunNodeProvision(context.Background(), Node{ID: "node-1", Hostname: "127.0.0.1", SSHPort: 22, SSHUser: "ubuntu"})
	var execErr jobs.ExecutionError
	if !errors.As(err, &execErr) {
		t.Fatalf("error type = %T, want jobs.ExecutionError", err)
	}
	if execErr.Code != "ANSIBLE_HOST_UNREACHABLE" {
		t.Fatalf("code = %s, want ANSIBLE_HOST_UNREACHABLE", execErr.Code)
	}
	if !execErr.Retryable {
		t.Fatal("retryable = false, want true")
	}
}

func TestHelperProcess(t *testing.T) {
	if os.Getenv("GO_WANT_HELPER_PROCESS") != "1" {
		return
	}
	if len(os.Args) < 4 || os.Args[2] != "--" {
		os.Exit(2)
	}
	if os.Args[3] == "exit4" {
		_, _ = os.Stdout.WriteString("host unreachable\n")
		os.Exit(4)
	}
	os.Exit(0)
}

type stubExecutor struct {
	err error
}

func (s stubExecutor) RunNodeProvision(context.Context, Node) error {
	return s.err
}
