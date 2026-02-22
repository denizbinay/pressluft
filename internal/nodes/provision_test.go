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
	"pressluft/internal/providers"
	"pressluft/internal/providers/hetzner"
)

func TestProvisionHandlerSuccessMarksNodeActive(t *testing.T) {
	now := time.Date(2026, 2, 22, 1, 10, 0, 0, time.UTC)
	nodeID := "node-1"
	store := NewInMemoryStore([]Node{{
		ID:        nodeID,
		Hostname:  "127.0.0.1",
		PublicIP:  "127.0.0.1",
		SSHPort:   22,
		SSHUser:   "ubuntu",
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}})

	handler := NewProvisionHandler(store, stubExecutor{err: nil}, nil, nil, log.New(io.Discard, "", 0))
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
		PublicIP:  "127.0.0.1",
		SSHPort:   22,
		SSHUser:   "ubuntu",
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}})

	handler := NewProvisionHandler(store, stubExecutor{err: jobs.ExecutionError{Code: "ANSIBLE_HOST_UNREACHABLE", Message: "timeout", Retryable: true}}, nil, nil, log.New(io.Discard, "", 0))
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

func TestProvisionHandlerExecutorTimeoutMarksNodeUnreachable(t *testing.T) {
	now := time.Date(2026, 2, 22, 1, 16, 0, 0, time.UTC)
	nodeID := "node-1"
	store := NewInMemoryStore([]Node{{
		ID:        nodeID,
		Hostname:  "127.0.0.1",
		PublicIP:  "127.0.0.1",
		SSHPort:   22,
		SSHUser:   "ubuntu",
		Status:    StatusActive,
		CreatedAt: now,
		UpdatedAt: now,
	}})

	handler := NewProvisionHandler(store, blockingExecutor{}, nil, nil, log.New(io.Discard, "", 0))
	handler.now = func() time.Time { return now }
	handler.timeout = 5 * time.Millisecond

	err := handler.Handle(context.Background(), jobs.Job{ID: "job-1", JobType: "node_provision", NodeID: &nodeID})
	var execErr jobs.ExecutionError
	if !errors.As(err, &execErr) {
		t.Fatalf("error type = %T, want jobs.ExecutionError", err)
	}
	if execErr.Code != "ANSIBLE_TIMEOUT" {
		t.Fatalf("code = %s, want ANSIBLE_TIMEOUT", execErr.Code)
	}

	node, getErr := store.GetByID(context.Background(), nodeID)
	if getErr != nil {
		t.Fatalf("GetByID() error = %v", getErr)
	}
	if node.Status != StatusUnreachable {
		t.Fatalf("status = %s, want unreachable", node.Status)
	}
}

func TestProvisionHandlerProviderAcquiresTargetBeforeProvision(t *testing.T) {
	now := time.Date(2026, 2, 22, 1, 20, 0, 0, time.UTC)
	nodeID := "node-provider"
	store := NewInMemoryStore([]Node{{
		ID:         nodeID,
		ProviderID: "hetzner",
		Name:       "edge-1",
		Hostname:   "pending.provider",
		PublicIP:   "pending.provider",
		SSHPort:    22,
		SSHUser:    "ubuntu",
		Status:     StatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}})

	handler := NewProvisionHandler(
		store,
		stubExecutor{err: nil},
		stubProviderStore{connection: providers.Connection{ProviderID: "hetzner", SecretToken: "bearer-token"}},
		stubHetznerAcquirer{target: hetzner.AcquireTarget{Hostname: "203.0.113.20", PublicIP: "203.0.113.20", SSHPort: 22, SSHUser: "root", SSHPrivateKeyPath: "/tmp/provider-key", ServerID: 101, ActionID: 909}},
		log.New(io.Discard, "", 0),
	)
	handler.now = func() time.Time { return now }

	err := handler.Handle(context.Background(), jobs.Job{ID: "job-provider", JobType: "node_provision", NodeID: &nodeID})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	updated, err := store.GetByID(context.Background(), nodeID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if updated.Hostname != "203.0.113.20" {
		t.Fatalf("Hostname = %s, want 203.0.113.20", updated.Hostname)
	}
	if updated.SSHPrivateKeyPath != "/tmp/provider-key" {
		t.Fatalf("SSHPrivateKeyPath = %s, want /tmp/provider-key", updated.SSHPrivateKeyPath)
	}
}

func TestProvisionHandlerProviderAcquisitionTimeoutRetryable(t *testing.T) {
	now := time.Date(2026, 2, 22, 1, 21, 0, 0, time.UTC)
	nodeID := "node-provider"
	store := NewInMemoryStore([]Node{{
		ID:         nodeID,
		ProviderID: "hetzner",
		Name:       "edge-1",
		Hostname:   "pending.provider",
		PublicIP:   "pending.provider",
		SSHPort:    22,
		SSHUser:    "ubuntu",
		Status:     StatusActive,
		CreatedAt:  now,
		UpdatedAt:  now,
	}})

	handler := NewProvisionHandler(
		store,
		stubExecutor{err: nil},
		stubProviderStore{connection: providers.Connection{ProviderID: "hetzner", SecretToken: "bearer-token"}},
		stubHetznerAcquirer{err: hetzner.ErrActionTimeout},
		log.New(io.Discard, "", 0),
	)

	err := handler.Handle(context.Background(), jobs.Job{ID: "job-provider", JobType: "node_provision", NodeID: &nodeID})
	var execErr jobs.ExecutionError
	if !errors.As(err, &execErr) {
		t.Fatalf("error type = %T, want jobs.ExecutionError", err)
	}
	if execErr.Code != "PROVIDER_ACQUISITION_TIMEOUT" {
		t.Fatalf("code = %s, want PROVIDER_ACQUISITION_TIMEOUT", execErr.Code)
	}
	if !execErr.Retryable {
		t.Fatal("retryable = false, want true")
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

type blockingExecutor struct{}

func (blockingExecutor) RunNodeProvision(ctx context.Context, _ Node) error {
	<-ctx.Done()
	return ctx.Err()
}

type stubProviderStore struct {
	connection providers.Connection
	err        error
}

func (s stubProviderStore) List(context.Context) ([]providers.Connection, error) {
	return nil, nil
}

func (s stubProviderStore) GetByProviderID(context.Context, string) (providers.Connection, error) {
	if s.err != nil {
		return providers.Connection{}, s.err
	}
	return s.connection, nil
}

func (s stubProviderStore) Upsert(context.Context, providers.Connection) (providers.Connection, error) {
	return providers.Connection{}, nil
}

type stubHetznerAcquirer struct {
	target hetzner.AcquireTarget
	err    error
}

func (s stubHetznerAcquirer) Acquire(context.Context, hetzner.AcquireInput) (hetzner.AcquireTarget, error) {
	if s.err != nil {
		return hetzner.AcquireTarget{}, s.err
	}
	return s.target, nil
}
