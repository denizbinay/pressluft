package backups

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

const backupCreatePlaybookPath = "ansible/playbooks/backup-create.yml"

type Executor interface {
	RunBackupCreate(ctx context.Context, input ExecutionInput) error
}

type ExecutionInput struct {
	BackupID      string
	EnvironmentID string
	BackupScope   string
	StoragePath   string
}

type AnsibleExecutor struct {
	runCommand func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func NewAnsibleExecutor() *AnsibleExecutor {
	return &AnsibleExecutor{runCommand: runCommand}
}

func (e *AnsibleExecutor) RunBackupCreate(ctx context.Context, input ExecutionInput) error {
	tmpDir, err := os.MkdirTemp("", "pressluft-backup-create-*")
	if err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("create temp dir: %v", err), Retryable: false}
	}
	defer os.RemoveAll(tmpDir)

	inventoryPath := filepath.Join(tmpDir, "inventory.ini")
	extraVarsPath := filepath.Join(tmpDir, "extra-vars.json")

	if err := os.WriteFile(inventoryPath, []byte("[target]\nlocalhost ansible_connection=local\n"), 0o600); err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("write inventory: %v", err), Retryable: false}
	}

	extraVars, err := json.Marshal(map[string]string{
		"backup_id":      input.BackupID,
		"environment_id": input.EnvironmentID,
		"backup_scope":   input.BackupScope,
		"storage_path":   input.StoragePath,
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
		backupCreatePlaybookPath,
	}

	output, err := e.runCommand(ctx, "ansible-playbook", args...)
	if err == nil {
		return nil
	}

	return mapAnsibleError(ctx, err, string(output))
}

type Handler struct {
	backups  store.BackupStore
	executor Executor
	now      func() time.Time
}

func NewHandler(backupStore store.BackupStore, executor Executor) *Handler {
	return &Handler{
		backups:  backupStore,
		executor: executor,
		now:      func() time.Time { return time.Now().UTC() },
	}
}

func (h *Handler) Handle(ctx context.Context, job jobs.Job) error {
	if job.ID == "" {
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "backup_create requires backup job id", Retryable: false}
	}
	if job.EnvironmentID == nil || *job.EnvironmentID == "" {
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "backup_create requires environment_id", Retryable: false}
	}

	backup, err := h.backups.GetBackupByID(ctx, job.ID)
	if err != nil {
		if errors.Is(err, store.ErrBackupNotFound) {
			return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "backup record not found for backup_create", Retryable: false}
		}
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: err.Error(), Retryable: false}
	}

	if _, err := h.backups.MarkBackupRunning(ctx, backup.ID, h.now()); err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: err.Error(), Retryable: false}
	}

	if err := h.executor.RunBackupCreate(ctx, ExecutionInput{
		BackupID:      backup.ID,
		EnvironmentID: backup.EnvironmentID,
		BackupScope:   backup.BackupScope,
		StoragePath:   backup.StoragePath,
	}); err != nil {
		_, _ = h.backups.MarkBackupFailed(ctx, backup.ID, h.now())
		return forceNonRetryable(err)
	}

	checksum := checksumForBackup(backup.ID)
	if _, err := h.backups.MarkBackupCompleted(ctx, backup.ID, checksum, 0, h.now()); err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: err.Error(), Retryable: false}
	}

	return nil
}

func forceNonRetryable(err error) error {
	var execErr jobs.ExecutionError
	if errors.As(err, &execErr) {
		execErr.Retryable = false
		return execErr
	}

	return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: err.Error(), Retryable: false}
}

func checksumForBackup(backupID string) string {
	hash := sha256.Sum256([]byte(backupID))
	return "sha256:" + hex.EncodeToString(hash[:])
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
