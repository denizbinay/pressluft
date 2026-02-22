package nodes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/jobs"
)

const nodeProvisionPlaybookPath = "ansible/playbooks/node-provision.yml"

type Executor interface {
	RunNodeProvision(ctx context.Context, node Node) error
}

type AnsibleExecutor struct {
	runCommand func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func NewAnsibleExecutor() *AnsibleExecutor {
	return &AnsibleExecutor{runCommand: runCommand}
}

func (e *AnsibleExecutor) RunNodeProvision(ctx context.Context, node Node) error {
	tmpDir, err := os.MkdirTemp("", "pressluft-node-provision-*")
	if err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("create temp dir: %v", err), Retryable: false}
	}
	defer os.RemoveAll(tmpDir)

	inventoryPath := filepath.Join(tmpDir, "inventory.ini")
	extraVarsPath := filepath.Join(tmpDir, "extra-vars.json")

	if err := os.WriteFile(inventoryPath, []byte(buildInventory(node)), 0o600); err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("write inventory: %v", err), Retryable: false}
	}

	extraVars, err := json.Marshal(map[string]string{
		"node_id":        node.ID,
		"node_hostname":  node.Hostname,
		"node_public_ip": node.PublicIP,
	})
	if err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("marshal extra vars: %v", err), Retryable: false}
	}

	if err := os.WriteFile(extraVarsPath, extraVars, 0o600); err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("write extra vars: %v", err), Retryable: false}
	}

	args := []string{
		"-i", inventoryPath,
		"-e", "@" + extraVarsPath,
		"--ssh-extra-args=-o StrictHostKeyChecking=accept-new",
		nodeProvisionPlaybookPath,
	}

	output, err := e.runCommand(ctx, "ansible-playbook", args...)
	if err == nil {
		return nil
	}

	return mapAnsibleError(ctx, err, string(output))
}

type ProvisionHandler struct {
	store    Store
	executor Executor
	logger   *log.Logger
	now      func() time.Time
}

func NewProvisionHandler(store Store, executor Executor, logger *log.Logger) *ProvisionHandler {
	return &ProvisionHandler{
		store:    store,
		executor: executor,
		logger:   logger,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (h *ProvisionHandler) Handle(ctx context.Context, job jobs.Job) error {
	if job.NodeID == nil || *job.NodeID == "" {
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "node_provision requires node_id", Retryable: false}
	}

	now := h.now()
	node, err := h.store.MarkProvisioning(ctx, *job.NodeID, now)
	if err != nil {
		if errors.Is(err, ErrNotFound) {
			return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "node not found for node_provision job", Retryable: false}
		}
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: err.Error(), Retryable: false}
	}

	h.logger.Printf("event=node_provision stage=start node_id=%s job_id=%s", node.ID, job.ID)

	if err := h.executor.RunNodeProvision(ctx, node); err != nil {
		if _, markErr := h.store.MarkUnreachable(ctx, node.ID, h.now()); markErr != nil {
			return fmt.Errorf("mark node unreachable: %w", markErr)
		}
		h.logger.Printf("event=node_provision stage=failed node_id=%s job_id=%s", node.ID, job.ID)
		return err
	}

	if _, err := h.store.MarkActive(ctx, node.ID, h.now()); err != nil {
		return fmt.Errorf("mark node active: %w", err)
	}

	h.logger.Printf("event=node_provision stage=succeeded node_id=%s job_id=%s", node.ID, job.ID)
	return nil
}

func buildInventory(node Node) string {
	fields := []string{node.Hostname}
	if node.SSHPort > 0 {
		fields = append(fields, "ansible_port="+strconv.Itoa(node.SSHPort))
	}
	if node.SSHUser != "" {
		fields = append(fields, "ansible_user="+node.SSHUser)
	}
	if node.SSHPrivateKeyPath != "" {
		fields = append(fields, "ansible_ssh_private_key_file="+node.SSHPrivateKeyPath)
	}

	return "[target]\n" + strings.Join(fields, " ") + "\n"
}

func runCommand(ctx context.Context, name string, args ...string) ([]byte, error) {
	cmd := exec.CommandContext(ctx, name, args...)
	return cmd.CombinedOutput()
}

func mapAnsibleError(ctx context.Context, err error, output string) error {
	message := strings.TrimSpace(output)
	if message == "" {
		message = err.Error()
	}

	if errors.Is(ctx.Err(), context.DeadlineExceeded) {
		return jobs.ExecutionError{Code: "ANSIBLE_TIMEOUT", Message: message, Retryable: true}
	}

	var exitErr *exec.ExitError
	if !errors.As(err, &exitErr) {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: message, Retryable: false}
	}

	switch exitErr.ExitCode() {
	case 1:
		return jobs.ExecutionError{Code: "ANSIBLE_PLAY_ERROR", Message: message, Retryable: true}
	case 2:
		return jobs.ExecutionError{Code: "ANSIBLE_HOST_FAILED", Message: message, Retryable: true}
	case 4:
		return jobs.ExecutionError{Code: "ANSIBLE_HOST_UNREACHABLE", Message: message, Retryable: true}
	case 5:
		return jobs.ExecutionError{Code: "ANSIBLE_SYNTAX_ERROR", Message: message, Retryable: false}
	case 250:
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: message, Retryable: false}
	default:
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: message, Retryable: false}
	}
}
