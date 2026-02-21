package migration

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"os/exec"
	"strings"
	"time"

	"pressluft/internal/ansible"
	"pressluft/internal/store"
)

const maxErrorMessageSize = 10 * 1024

type CommandRunner interface {
	Run(ctx context.Context, command string, args ...string) (string, error)
}

type ExecRunner struct{}

func (ExecRunner) Run(ctx context.Context, command string, args ...string) (string, error) {
	cmd := exec.CommandContext(ctx, command, args...)
	output, err := cmd.CombinedOutput()
	if err != nil {
		return string(output), err
	}
	return string(output), nil
}

type queuedSiteImport struct {
	JobID         string
	SiteID        string
	EnvironmentID string
	NodeID        string
	ReleaseID     string
	ArchiveURL    string
	TargetURL     string
	Hostname      string
	AttemptCount  int
	MaxAttempts   int
}

func ExecuteQueuedSiteImport(ctx context.Context, db *sql.DB, runner CommandRunner, playbookPath string) (bool, error) {
	job, ok, err := lockNextQueuedSiteImport(ctx, db)
	if err != nil {
		return false, err
	}
	if !ok {
		return false, nil
	}

	inventoryPath, cleanup, err := ansible.WriteTempLocalInventory(job.Hostname)
	if err != nil {
		if markErr := markSiteImportFailed(ctx, db, job, "ANSIBLE_UNEXPECTED_ERROR", err.Error()); markErr != nil {
			return true, markErr
		}
		return true, err
	}
	defer cleanup()

	extraVars, err := json.Marshal(map[string]string{
		"site_id":        job.SiteID,
		"environment_id": job.EnvironmentID,
		"node_id":        job.NodeID,
		"archive_url":    job.ArchiveURL,
		"release_id":     job.ReleaseID,
		"target_url":     job.TargetURL,
	})
	if err != nil {
		if markErr := markSiteImportFailed(ctx, db, job, "ANSIBLE_UNEXPECTED_ERROR", "failed to marshal ansible vars"); markErr != nil {
			return true, markErr
		}
		return true, fmt.Errorf("marshal site_import vars: %w", err)
	}

	output, runErr := runner.Run(ctx, "ansible-playbook", "-i", inventoryPath, playbookPath, "--extra-vars", string(extraVars))
	if runErr != nil {
		errorCode, retryable := classifyAnsibleError(runErr)
		message := truncateErrorMessage(output, runErr)
		if retryable && job.AttemptCount < job.MaxAttempts {
			if retryErr := requeueSiteImport(ctx, db, job.JobID, job.AttemptCount, errorCode, message); retryErr != nil {
				return true, retryErr
			}
			return true, fmt.Errorf("execute site_import job %s: %w", job.JobID, runErr)
		}

		if failErr := markSiteImportFailed(ctx, db, job, errorCode, message); failErr != nil {
			return true, failErr
		}
		return true, fmt.Errorf("execute site_import job %s: %w", job.JobID, runErr)
	}

	if err := markSiteImportSucceeded(ctx, db, job); err != nil {
		return true, err
	}

	return true, nil
}

func lockNextQueuedSiteImport(ctx context.Context, db *sql.DB) (queuedSiteImport, bool, error) {
	var selected queuedSiteImport
	err := store.WithTx(ctx, db, func(tx *sql.Tx) error {
		row := tx.QueryRowContext(ctx, `
			SELECT j.id, j.site_id, j.environment_id, j.node_id, j.payload_json, j.attempt_count, j.max_attempts, n.hostname
			FROM jobs j
			JOIN nodes n ON n.id = j.node_id
			WHERE j.job_type = 'site_import'
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
			return fmt.Errorf("select queued site_import job: %w", err)
		}

		var payload struct {
			ReleaseID  string `json:"release_id"`
			ArchiveURL string `json:"archive_url"`
			TargetURL  string `json:"target_url"`
		}
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err != nil {
			return fmt.Errorf("decode site_import payload: %w", err)
		}

		selected.ReleaseID = strings.TrimSpace(payload.ReleaseID)
		selected.ArchiveURL = strings.TrimSpace(payload.ArchiveURL)
		selected.TargetURL = strings.TrimSpace(payload.TargetURL)
		selected.Hostname = hostname

		now := time.Now().UTC().Format(time.RFC3339)
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'running', attempt_count = attempt_count + 1, locked_at = ?, locked_by = 'worker-site-import', started_at = ?, updated_at = ?, run_after = NULL
			WHERE id = ?
		`, now, now, now, selected.JobID); err != nil {
			return fmt.Errorf("mark site_import job running: %w", err)
		}

		selected.AttemptCount++
		return nil
	})
	if err != nil {
		return queuedSiteImport{}, false, fmt.Errorf("lock site_import job: %w", err)
	}

	if selected.JobID == "" {
		return queuedSiteImport{}, false, nil
	}

	return selected, true, nil
}

func requeueSiteImport(ctx context.Context, db *sql.DB, jobID string, attemptCount int, errorCode, errorMessage string) error {
	now := time.Now().UTC()
	runAfter := now.Add(retryBackoff(attemptCount)).Format(time.RFC3339)
	return store.WithTx(ctx, db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'queued', run_after = ?, error_code = ?, error_message = ?, locked_at = NULL, locked_by = NULL, updated_at = ?
			WHERE id = ?
		`, runAfter, errorCode, errorMessage, now.Format(time.RFC3339), jobID); err != nil {
			return fmt.Errorf("requeue site_import job: %w", err)
		}
		return nil
	})
}

func markSiteImportSucceeded(ctx context.Context, db *sql.DB, job queuedSiteImport) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'active', current_release_id = ?, updated_at = ?, state_version = state_version + 1
			WHERE id = ? AND site_id = ?
		`, job.ReleaseID, now, job.EnvironmentID, job.SiteID); err != nil {
			return fmt.Errorf("mark imported environment active: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'active', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, job.SiteID); err != nil {
			return fmt.Errorf("mark imported site active: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'succeeded', finished_at = ?, error_code = NULL, error_message = NULL, locked_at = NULL, locked_by = NULL, updated_at = ?
			WHERE id = ?
		`, now, now, job.JobID); err != nil {
			return fmt.Errorf("mark site_import job succeeded: %w", err)
		}

		return nil
	})
}

func markSiteImportFailed(ctx context.Context, db *sql.DB, job queuedSiteImport, errorCode, errorMessage string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'failed', updated_at = ?, state_version = state_version + 1
			WHERE id = ? AND site_id = ?
		`, now, job.EnvironmentID, job.SiteID); err != nil {
			return fmt.Errorf("mark imported environment failed: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'failed', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, job.SiteID); err != nil {
			return fmt.Errorf("mark imported site failed: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'failed', finished_at = ?, error_code = ?, error_message = ?, locked_at = NULL, locked_by = NULL, updated_at = ?
			WHERE id = ?
		`, now, errorCode, errorMessage, now, job.JobID); err != nil {
			return fmt.Errorf("mark site_import job failed: %w", err)
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
