package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"pressluft/internal/ansible"
	"pressluft/internal/store"
)

const nodeProvisionErrorCodeFailed = "NODE_PROVISION_FAILED"
const nodeProvisionErrorCodeTimeout = "NODE_PROVISION_TIMEOUT"

type CommandRunner interface {
	Run(ctx context.Context, command string, args ...string) (string, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), fmt.Errorf("run %s: %w", command, err)
	}
	return string(output), nil
}

func ExecuteQueuedNodeProvision(ctx context.Context, db *sql.DB, runner CommandRunner, playbookPath string) (bool, error) {
	job, ok, err := lockNextQueuedNodeProvision(ctx, db)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	inventoryPath, cleanup, err := ansible.WriteTempLocalInventory(job.Hostname)
	if err != nil {
		if err2 := completeNodeProvisionFailure(ctx, db, job.JobID, job.NodeID, nodeProvisionErrorCodeFailed, err.Error()); err2 != nil {
			return true, err2
		}
		return true, err
	}
	defer cleanup()

	output, runErr := runner.Run(ctx, "ansible-playbook", "-i", inventoryPath, playbookPath)
	if runErr != nil {
		errorCode := classifyNodeProvisionError(runErr)
		truncated := truncateError(output, runErr)
		if err := completeNodeProvisionFailure(ctx, db, job.JobID, job.NodeID, errorCode, truncated); err != nil {
			return true, err
		}
		return true, fmt.Errorf("execute node_provision job %s: %w", job.JobID, runErr)
	}

	if err := completeNodeProvisionSuccess(ctx, db, job.JobID, job.NodeID); err != nil {
		return true, err
	}

	return true, nil
}

type queuedNodeProvision struct {
	JobID    string
	NodeID   string
	Hostname string
}

func lockNextQueuedNodeProvision(ctx context.Context, db *sql.DB) (queuedNodeProvision, bool, error) {
	var selected queuedNodeProvision
	err := store.WithTx(ctx, db, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, `
			SELECT j.id, n.id, n.hostname
			FROM jobs j
			JOIN nodes n ON n.id = j.node_id
			WHERE j.job_type = 'node_provision'
			  AND j.status = 'queued'
			ORDER BY j.created_at ASC
			LIMIT 1
		`)

		var hostname string
		if err := row.Scan(&selected.JobID, &selected.NodeID, &hostname); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return nil
			}
			return fmt.Errorf("select queued node_provision job: %w", err)
		}

		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'running', started_at = ?, updated_at = ?
			WHERE id = ?
		`, now, now, selected.JobID); err != nil {
			return fmt.Errorf("mark job running: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE nodes
			SET status = 'provisioning', updated_at = ?
			WHERE id = ?
		`, now, selected.NodeID); err != nil {
			return fmt.Errorf("mark node provisioning: %w", err)
		}

		selected.Hostname = hostname
		return nil
	})
	if err != nil {
		return queuedNodeProvision{}, false, fmt.Errorf("lock node_provision job: %w", err)
	}

	if selected.JobID == "" {
		return queuedNodeProvision{}, false, nil
	}

	return selected, true, nil
}

func completeNodeProvisionSuccess(ctx context.Context, db *sql.DB, jobID, nodeID string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'succeeded', finished_at = ?, error_code = NULL, error_message = NULL, updated_at = ?
			WHERE id = ?
		`, now, now, jobID); err != nil {
			return fmt.Errorf("mark job succeeded: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE nodes
			SET status = 'active', updated_at = ?
			WHERE id = ?
		`, now, nodeID); err != nil {
			return fmt.Errorf("mark node active: %w", err)
		}

		return nil
	})
}

func completeNodeProvisionFailure(ctx context.Context, db *sql.DB, jobID, nodeID, errorCode, errorMessage string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'failed', finished_at = ?, error_code = ?, error_message = ?, updated_at = ?
			WHERE id = ?
		`, now, errorCode, errorMessage, now, jobID); err != nil {
			return fmt.Errorf("mark job failed: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE nodes
			SET status = 'unreachable', updated_at = ?
			WHERE id = ?
		`, now, nodeID); err != nil {
			return fmt.Errorf("mark node unreachable: %w", err)
		}

		return nil
	})
}

func classifyNodeProvisionError(err error) string {
	if errors.Is(err, context.DeadlineExceeded) || errors.Is(err, context.Canceled) {
		return nodeProvisionErrorCodeTimeout
	}
	return nodeProvisionErrorCodeFailed
}

func truncateError(output string, runErr error) string {
	message := strings.TrimSpace(output)
	if message == "" {
		message = runErr.Error()
	}
	if len(message) > 512 {
		return message[:512]
	}
	return message
}
