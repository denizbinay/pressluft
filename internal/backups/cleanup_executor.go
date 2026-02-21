package backups

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"pressluft/internal/store"
)

const backupCleanupInternalErrorCode = "BACKUP_CLEANUP_INTERNAL"
const maxErrorMessageSize = 10 * 1024

type CommandRunner interface {
	Run(ctx context.Context, command string, args ...string) (string, error)
}

type queuedBackupCleanup struct {
	JobID         string
	SiteID        string
	EnvironmentID string
	NodeID        string
	BackupID      string
	StoragePath   string
	Inventory     string
	AttemptCount  int
	MaxAttempts   int
}

func ExecuteQueuedBackupCleanup(ctx context.Context, db *sql.DB, runner CommandRunner, playbookPath string) (bool, error) {
	job, ok, err := lockNextQueuedBackupCleanup(ctx, db)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	extraVars, err := json.Marshal(map[string]string{
		"site_id":        job.SiteID,
		"environment_id": job.EnvironmentID,
		"node_id":        job.NodeID,
		"backup_id":      job.BackupID,
		"storage_path":   job.StoragePath,
	})
	if err != nil {
		if markErr := markBackupCleanupFailed(ctx, db, job.JobID, backupCleanupInternalErrorCode, "failed to marshal ansible vars"); markErr != nil {
			return true, markErr
		}
		return true, fmt.Errorf("marshal backup_cleanup vars: %w", err)
	}

	output, runErr := runner.Run(ctx, "ansible-playbook", "-i", job.Inventory, playbookPath, "--extra-vars", string(extraVars))
	if runErr != nil {
		errorCode, retryable := classifyAnsibleError(runErr)
		message := truncateErrorMessage(output, runErr)
		if retryable && job.AttemptCount < job.MaxAttempts {
			if retryErr := requeueBackupCleanup(ctx, db, job.JobID, job.AttemptCount, errorCode, message); retryErr != nil {
				return true, retryErr
			}
			return true, fmt.Errorf("execute backup_cleanup job %s: %w", job.JobID, runErr)
		}

		if failErr := markBackupCleanupFailed(ctx, db, job.JobID, errorCode, message); failErr != nil {
			return true, failErr
		}
		return true, fmt.Errorf("execute backup_cleanup job %s: %w", job.JobID, runErr)
	}

	if err := markBackupCleanupSucceeded(ctx, db, job); err != nil {
		message := truncateErrorMessage("", err)
		if failErr := markBackupCleanupFailed(ctx, db, job.JobID, backupCleanupInternalErrorCode, message); failErr != nil {
			return true, failErr
		}
		return true, fmt.Errorf("complete backup_cleanup job %s: %w", job.JobID, err)
	}

	return true, nil
}

func lockNextQueuedBackupCleanup(ctx context.Context, db *sql.DB) (queuedBackupCleanup, bool, error) {
	var selected queuedBackupCleanup
	err := store.WithTx(ctx, db, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, `
			SELECT j.id, j.site_id, j.environment_id, j.node_id, j.payload_json, j.attempt_count, j.max_attempts, n.hostname
			FROM jobs j
			JOIN nodes n ON n.id = j.node_id
			WHERE j.job_type = 'backup_cleanup'
			  AND j.status = 'queued'
			  AND (j.run_after IS NULL OR j.run_after <= ?)
			ORDER BY j.created_at ASC
			LIMIT 1
		`, time.Now().UTC().Format(time.RFC3339))

		var payloadJSON string
		var hostname string
		if err := row.Scan(&selected.JobID, &selected.SiteID, &selected.EnvironmentID, &selected.NodeID, &payloadJSON, &selected.AttemptCount, &selected.MaxAttempts, &hostname); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("select queued backup_cleanup job: %w", err)
		}

		var payload backupCleanupPayload
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			return fmt.Errorf("decode backup_cleanup payload: %w", err)
		}

		selected.BackupID = strings.TrimSpace(payload.BackupID)
		selected.StoragePath = strings.TrimSpace(payload.StoragePath)
		if selected.BackupID == "" || selected.StoragePath == "" {
			return errors.New("backup_cleanup payload missing required fields")
		}

		selected.Inventory = fmt.Sprintf("%s ansible_connection=local", hostname)

		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'running', attempt_count = attempt_count + 1, locked_at = ?, locked_by = 'worker-backup-cleanup', started_at = ?, updated_at = ?, run_after = NULL
			WHERE id = ?
		`, now, now, now, selected.JobID); err != nil {
			return fmt.Errorf("mark backup_cleanup job running: %w", err)
		}

		selected.AttemptCount++
		return nil
	})
	if err != nil {
		return queuedBackupCleanup{}, false, fmt.Errorf("lock backup_cleanup job: %w", err)
	}

	if selected.JobID == "" {
		return queuedBackupCleanup{}, false, nil
	}

	return selected, true, nil
}

func requeueBackupCleanup(ctx context.Context, db *sql.DB, jobID string, attemptCount int, errorCode, errorMessage string) error {
	now := time.Now().UTC()
	runAfter := now.Add(retryBackoff(attemptCount)).Format(time.RFC3339)
	return store.WithTx(ctx, db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'queued', run_after = ?, error_code = ?, error_message = ?, locked_at = NULL, locked_by = NULL, updated_at = ?
			WHERE id = ?
		`, runAfter, errorCode, errorMessage, now.Format(time.RFC3339), jobID); err != nil {
			return fmt.Errorf("requeue backup_cleanup job: %w", err)
		}
		return nil
	})
}

func markBackupCleanupSucceeded(ctx context.Context, db *sql.DB, job queuedBackupCleanup) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE backups
			SET status = 'expired'
			WHERE id = ?
			  AND retention_until < ?
			  AND status IN ('completed', 'failed')
		`, job.BackupID, now); err != nil {
			return fmt.Errorf("mark backup expired: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'succeeded', finished_at = ?, error_code = NULL, error_message = NULL, locked_at = NULL, locked_by = NULL, updated_at = ?
			WHERE id = ?
		`, now, now, job.JobID); err != nil {
			return fmt.Errorf("mark backup_cleanup job succeeded: %w", err)
		}

		return nil
	})
}

func markBackupCleanupFailed(ctx context.Context, db *sql.DB, jobID, errorCode, errorMessage string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'failed', finished_at = ?, error_code = ?, error_message = ?, locked_at = NULL, locked_by = NULL, updated_at = ?
			WHERE id = ?
		`, now, errorCode, errorMessage, now, jobID); err != nil {
			return fmt.Errorf("mark backup_cleanup job failed: %w", err)
		}
		return nil
	})
}

func classifyAnsibleError(err error) (string, bool) {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return "ANSIBLE_TIMEOUT", false
	}

	var exitErr *exec.ExitError
	if errors.As(err, &exitErr) {
		switch exitErr.ExitCode() {
		case 1:
			return "ANSIBLE_PLAY_ERROR", true
		case 2:
			return "ANSIBLE_HOST_FAILED", true
		case 4:
			return "ANSIBLE_HOST_UNREACHABLE", true
		case 5:
			return "ANSIBLE_SYNTAX_ERROR", false
		case 250:
			return "ANSIBLE_UNEXPECTED_ERROR", false
		default:
			return "ANSIBLE_UNKNOWN_EXIT", false
		}
	}

	return "ANSIBLE_UNEXPECTED_ERROR", false
}

func retryBackoff(attemptCount int) time.Duration {
	switch attemptCount {
	case 1:
		return 1 * time.Minute
	case 2:
		return 5 * time.Minute
	default:
		return 15 * time.Minute
	}
}

func truncateErrorMessage(output string, runErr error) string {
	message := strings.TrimSpace(output)
	if message == "" {
		message = runErr.Error()
	}
	if len(message) <= maxErrorMessageSize {
		return message
	}
	return message[len(message)-maxErrorMessageSize:]
}
