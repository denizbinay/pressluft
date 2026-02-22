package environments

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"pressluft/internal/backups"
	"pressluft/internal/jobs"
	"pressluft/internal/nodes"
	"pressluft/internal/store"
)

const envRestorePlaybookPath = "ansible/playbooks/env-restore.yml"

type EnvRestoreExecutor interface {
	RunEnvRestore(ctx context.Context, node nodes.Node, vars EnvRestoreVars) error
}

type EnvRestoreVars struct {
	SiteID                 string `json:"site_id"`
	SiteSlug               string `json:"site_slug"`
	EnvironmentID          string `json:"environment_id"`
	EnvironmentSlug        string `json:"environment_slug"`
	BackupID               string `json:"backup_id"`
	BackupArtifactPath     string `json:"backup_artifact_path"`
	PreRestoreArtifactPath string `json:"pre_restore_artifact_path"`
	RestoreMarkerPath      string `json:"restore_marker_path"`
}

type AnsibleEnvRestoreExecutor struct {
	runCommand func(ctx context.Context, name string, args ...string) ([]byte, error)
}

func NewAnsibleEnvRestoreExecutor() *AnsibleEnvRestoreExecutor {
	return &AnsibleEnvRestoreExecutor{runCommand: runCommand}
}

func (e *AnsibleEnvRestoreExecutor) RunEnvRestore(ctx context.Context, node nodes.Node, vars EnvRestoreVars) error {
	tmpDir, err := os.MkdirTemp("", "pressluft-env-restore-*")
	if err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("create temp dir: %v", err), Retryable: false}
	}
	defer os.RemoveAll(tmpDir)

	inventoryPath := filepath.Join(tmpDir, "inventory.ini")
	extraVarsPath := filepath.Join(tmpDir, "extra-vars.json")

	if err := os.WriteFile(inventoryPath, []byte(buildRestoreInventory(node)), 0o600); err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("write inventory: %v", err), Retryable: false}
	}

	extraVars, err := json.Marshal(vars)
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
		envRestorePlaybookPath,
	}

	output, err := e.runCommand(ctx, "ansible-playbook", args...)
	if err == nil {
		return nil
	}

	return mapAnsibleError(ctx, err, string(output))
}

type EnvRestoreHandler struct {
	siteStore    store.SiteStore
	nodeStore    nodes.Store
	backupStore  store.BackupStore
	restoreStore store.RestoreRequestStore
	executor     EnvRestoreExecutor
	logger       *log.Logger
	now          func() time.Time
}

func NewEnvRestoreHandler(siteStore store.SiteStore, nodeStore nodes.Store, backupStore store.BackupStore, restoreStore store.RestoreRequestStore, executor EnvRestoreExecutor, logger *log.Logger) *EnvRestoreHandler {
	return &EnvRestoreHandler{
		siteStore:    siteStore,
		nodeStore:    nodeStore,
		backupStore:  backupStore,
		restoreStore: restoreStore,
		executor:     executor,
		logger:       logger,
		now:          func() time.Time { return time.Now().UTC() },
	}
}

func (h *EnvRestoreHandler) Handle(ctx context.Context, job jobs.Job) error {
	if job.SiteID == nil || *job.SiteID == "" || job.EnvironmentID == nil || *job.EnvironmentID == "" || job.NodeID == nil || *job.NodeID == "" {
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "env_restore requires site_id, environment_id, and node_id", Retryable: false}
	}

	request, err := h.restoreStore.GetRestoreRequestByJobID(ctx, job.ID)
	if err != nil {
		_, _, _ = h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, false, h.now())
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "restore request not found", Retryable: false}
	}

	site, err := h.siteStore.GetSiteByID(ctx, *job.SiteID)
	if err != nil {
		_, _, _ = h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, false, h.now())
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "site not found for env_restore job", Retryable: false}
	}

	environment, err := h.siteStore.GetEnvironmentByID(ctx, *job.EnvironmentID)
	if err != nil {
		_, _, _ = h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, false, h.now())
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "environment not found for env_restore job", Retryable: false}
	}

	node, err := h.nodeStore.GetByID(ctx, *job.NodeID)
	if err != nil {
		_, _, _ = h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, false, h.now())
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "node not found for env_restore job", Retryable: false}
	}

	backup, err := h.backupStore.GetBackupByID(ctx, request.BackupID)
	if err != nil {
		_, _, _ = h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, false, h.now())
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "backup not found for env_restore job", Retryable: false}
	}
	if backup.EnvironmentID != environment.ID || backup.Status != "completed" || backup.Checksum == nil || backup.SizeBytes == nil || *backup.SizeBytes <= 0 {
		_, _, _ = h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, false, h.now())
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: "backup is not eligible for restore", Retryable: false}
	}

	preRestoreBackupID := fmt.Sprintf("pre-%s-%d", environment.ID[:8], h.now().Unix())
	preRestoreStoragePath := fmt.Sprintf("s3://pressluft/backups/%s/%s.tar.zst", environment.ID, preRestoreBackupID)
	preRestoreArtifactPath := backups.LocalArtifactPath(preRestoreStoragePath)
	if err := os.MkdirAll(filepath.Dir(preRestoreArtifactPath), 0o755); err != nil {
		_, _, _ = h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, false, h.now())
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("create pre-restore backup dir: %v", err), Retryable: false}
	}
	preRestoreBody := []byte("pre_restore_backup environment_id=" + environment.ID + " generated_at=" + h.now().Format(time.RFC3339) + "\n")
	if err := os.WriteFile(preRestoreArtifactPath, preRestoreBody, 0o600); err != nil {
		_, _, _ = h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, false, h.now())
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("write pre-restore backup artifact: %v", err), Retryable: false}
	}
	checksum, size, err := backups.ChecksumAndSize(preRestoreArtifactPath)
	if err != nil {
		_, _, _ = h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, false, h.now())
		return jobs.ExecutionError{Code: "ANSIBLE_UNEXPECTED_ERROR", Message: fmt.Sprintf("checksum pre-restore artifact: %v", err), Retryable: false}
	}

	if _, err := h.backupStore.CreateBackup(ctx, store.CreateBackupInput{
		ID:             preRestoreBackupID,
		EnvironmentID:  environment.ID,
		BackupScope:    "full",
		StorageType:    "s3",
		StoragePath:    preRestoreStoragePath,
		RetentionUntil: h.now().AddDate(0, 0, 30),
		CreatedAt:      h.now(),
	}); err == nil {
		_, _ = h.backupStore.MarkBackupRunning(ctx, preRestoreBackupID, h.now())
		_, _ = h.backupStore.MarkBackupCompleted(ctx, preRestoreBackupID, checksum, size, h.now())
	}

	restoreMarkerPath := filepath.Join(os.TempDir(), "pressluft-restores", environment.SiteID, environment.ID+".restore.txt")
	h.logger.Printf("event=env_restore stage=start site_id=%s environment_id=%s backup_id=%s node_id=%s job_id=%s", site.ID, environment.ID, backup.ID, node.ID, job.ID)

	err = h.executor.RunEnvRestore(ctx, node, EnvRestoreVars{
		SiteID:                 site.ID,
		SiteSlug:               site.Slug,
		EnvironmentID:          environment.ID,
		EnvironmentSlug:        environment.Slug,
		BackupID:               backup.ID,
		BackupArtifactPath:     backups.LocalArtifactPath(backup.StoragePath),
		PreRestoreArtifactPath: preRestoreArtifactPath,
		RestoreMarkerPath:      restoreMarkerPath,
	})
	if err != nil {
		_, _, _ = h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, false, h.now())
		_ = h.restoreStore.DeleteRestoreRequest(ctx, job.ID)
		h.logger.Printf("event=env_restore stage=failed site_id=%s environment_id=%s backup_id=%s node_id=%s job_id=%s error=%v", site.ID, environment.ID, backup.ID, node.ID, job.ID, err)
		return forceNonRetryableRestore(err)
	}

	if _, _, err := h.siteStore.MarkEnvironmentRestoreResult(ctx, *job.EnvironmentID, true, h.now()); err != nil {
		return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: err.Error(), Retryable: false}
	}
	_ = h.restoreStore.DeleteRestoreRequest(ctx, job.ID)
	h.logger.Printf("event=env_restore stage=succeeded site_id=%s environment_id=%s backup_id=%s node_id=%s job_id=%s", site.ID, environment.ID, backup.ID, node.ID, job.ID)
	return nil
}

func buildRestoreInventory(node nodes.Node) string {
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

func forceNonRetryableRestore(err error) error {
	var execErr jobs.ExecutionError
	if errors.As(err, &execErr) {
		execErr.Retryable = false
		return execErr
	}
	return jobs.ExecutionError{Code: "ANSIBLE_UNKNOWN_EXIT", Message: err.Error(), Retryable: false}
}
