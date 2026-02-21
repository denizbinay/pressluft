package releases

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

const (
	HealthCheckErrorCodeFailed  = "HEALTH_CHECK_FAILED"
	HealthCheckErrorCodeTimeout = "HEALTH_CHECK_TIMEOUT"

	ReleaseRollbackErrorCodeFailed  = "RELEASE_ROLLBACK_FAILED"
	ReleaseRollbackErrorCodeTimeout = "RELEASE_ROLLBACK_TIMEOUT"
)

var ErrInvalidInput = errors.New("invalid input")
var ErrEnvironmentNotFound = errors.New("environment not found")
var ErrReleaseNotFound = errors.New("release not found")
var ErrNoRollbackRelease = errors.New("no rollback release available")

type Service struct {
	db *sql.DB
}

type HealthCheckTriggerInput struct {
	SiteID          string
	EnvironmentID   string
	NodeID          string
	MutationJobType string
}

type HealthCheckTriggerResult struct {
	JobID     string
	ReleaseID string
}

type HealthCheckFailureInput struct {
	SiteID           string
	EnvironmentID    string
	NodeID           string
	HealthCheckJobID string
	Err              error
}

type HealthCheckFailureResult struct {
	RollbackJobID     string
	FailedReleaseID   string
	RestoredReleaseID string
}

type RollbackSuccessInput struct {
	SiteID            string
	EnvironmentID     string
	RollbackJobID     string
	RestoredReleaseID string
}

type RollbackFailureInput struct {
	SiteID        string
	EnvironmentID string
	RollbackJobID string
	Err           error
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) TriggerPostMutationHealthCheck(ctx context.Context, input HealthCheckTriggerInput) (HealthCheckTriggerResult, error) {
	if strings.TrimSpace(input.SiteID) == "" || strings.TrimSpace(input.EnvironmentID) == "" || strings.TrimSpace(input.NodeID) == "" {
		return HealthCheckTriggerResult{}, ErrInvalidInput
	}
	if !isHealthTriggerJobType(strings.TrimSpace(input.MutationJobType)) {
		return HealthCheckTriggerResult{}, ErrInvalidInput
	}

	jobID, err := newUUIDv4()
	if err != nil {
		return HealthCheckTriggerResult{}, err
	}

	result := HealthCheckTriggerResult{JobID: jobID}
	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		releaseID, err := loadCurrentRelease(ctx, tx, input.SiteID, input.EnvironmentID, input.NodeID)
		if err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]string{
			"environment_id":   input.EnvironmentID,
			"release_id":       releaseID,
			"trigger_job_type": input.MutationJobType,
		})
		if err != nil {
			return fmt.Errorf("marshal health_check payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "health_check",
			SiteID:        sql.NullString{String: input.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: input.EnvironmentID, Valid: true},
			NodeID:        sql.NullString{String: input.NodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue health_check job: %w", err)
		}

		result.ReleaseID = releaseID
		return nil
	})
	if err != nil {
		return HealthCheckTriggerResult{}, err
	}

	return result, nil
}

func (s *Service) HandleHealthCheckSuccess(ctx context.Context, healthCheckJobID, environmentID, releaseID string) error {
	if strings.TrimSpace(healthCheckJobID) == "" || strings.TrimSpace(environmentID) == "" || strings.TrimSpace(releaseID) == "" {
		return ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE releases
			SET health_status = 'healthy'
			WHERE id = ? AND environment_id = ?
		`, releaseID, environmentID); err != nil {
			return fmt.Errorf("mark release healthy: %w", err)
		}

		if err := markJobSucceeded(ctx, tx, healthCheckJobID, now); err != nil {
			return err
		}

		return nil
	})
}

func (s *Service) HandleHealthCheckFailure(ctx context.Context, input HealthCheckFailureInput) (HealthCheckFailureResult, error) {
	if strings.TrimSpace(input.SiteID) == "" || strings.TrimSpace(input.EnvironmentID) == "" || strings.TrimSpace(input.NodeID) == "" || strings.TrimSpace(input.HealthCheckJobID) == "" {
		return HealthCheckFailureResult{}, ErrInvalidInput
	}

	rollbackJobID, err := newUUIDv4()
	if err != nil {
		return HealthCheckFailureResult{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result := HealthCheckFailureResult{RollbackJobID: rollbackJobID}
	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		failedReleaseID, err := loadCurrentRelease(ctx, tx, input.SiteID, input.EnvironmentID, input.NodeID)
		if err != nil {
			return err
		}

		restoredReleaseID, err := loadPreviousRelease(ctx, tx, input.EnvironmentID, failedReleaseID)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE releases
			SET health_status = 'unhealthy'
			WHERE id = ?
		`, failedReleaseID); err != nil {
			return fmt.Errorf("mark failed release unhealthy: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'restoring', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, input.EnvironmentID); err != nil {
			return fmt.Errorf("mark environment restoring: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'restoring', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, input.SiteID); err != nil {
			return fmt.Errorf("mark site restoring: %w", err)
		}

		if err := markJobFailed(ctx, tx, input.HealthCheckJobID, classifyHealthCheckErrorCode(input.Err), messageFromError(input.Err), now); err != nil {
			return err
		}

		payload, err := json.Marshal(map[string]string{
			"environment_id":      input.EnvironmentID,
			"failed_release_id":   failedReleaseID,
			"restored_release_id": restoredReleaseID,
			"health_check_job_id": input.HealthCheckJobID,
		})
		if err != nil {
			return fmt.Errorf("marshal release_rollback payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         rollbackJobID,
			JobType:       "release_rollback",
			SiteID:        sql.NullString{String: input.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: input.EnvironmentID, Valid: true},
			NodeID:        sql.NullString{String: input.NodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue release_rollback job: %w", err)
		}

		result.FailedReleaseID = failedReleaseID
		result.RestoredReleaseID = restoredReleaseID
		return nil
	})
	if err != nil {
		return HealthCheckFailureResult{}, err
	}

	return result, nil
}

func (s *Service) ApplyRollbackSuccess(ctx context.Context, input RollbackSuccessInput) error {
	if strings.TrimSpace(input.SiteID) == "" || strings.TrimSpace(input.EnvironmentID) == "" || strings.TrimSpace(input.RollbackJobID) == "" || strings.TrimSpace(input.RestoredReleaseID) == "" {
		return ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET current_release_id = ?, status = 'active', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, input.RestoredReleaseID, now, input.EnvironmentID); err != nil {
			return fmt.Errorf("update environment after rollback: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'active', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, input.SiteID); err != nil {
			return fmt.Errorf("update site after rollback: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE releases
			SET health_status = 'healthy'
			WHERE id = ?
		`, input.RestoredReleaseID); err != nil {
			return fmt.Errorf("mark restored release healthy: %w", err)
		}

		if err := markJobSucceeded(ctx, tx, input.RollbackJobID, now); err != nil {
			return err
		}

		return nil
	})
}

func (s *Service) ApplyRollbackFailure(ctx context.Context, input RollbackFailureInput) error {
	if strings.TrimSpace(input.SiteID) == "" || strings.TrimSpace(input.EnvironmentID) == "" || strings.TrimSpace(input.RollbackJobID) == "" {
		return ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'failed', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, input.EnvironmentID); err != nil {
			return fmt.Errorf("mark environment failed after rollback failure: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'failed', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, input.SiteID); err != nil {
			return fmt.Errorf("mark site failed after rollback failure: %w", err)
		}

		if err := markJobFailed(ctx, tx, input.RollbackJobID, classifyRollbackErrorCode(input.Err), messageFromError(input.Err), now); err != nil {
			return err
		}

		return nil
	})
}

func loadCurrentRelease(ctx context.Context, tx *sql.Tx, siteID, environmentID, nodeID string) (string, error) {
	var releaseID sql.NullString
	err := tx.QueryRowContext(ctx, `
		SELECT current_release_id
		FROM environments
		WHERE id = ? AND site_id = ? AND node_id = ?
		LIMIT 1
	`, environmentID, siteID, nodeID).Scan(&releaseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrEnvironmentNotFound
		}
		return "", fmt.Errorf("query environment current release: %w", err)
	}
	if !releaseID.Valid || strings.TrimSpace(releaseID.String) == "" {
		return "", ErrReleaseNotFound
	}

	return strings.TrimSpace(releaseID.String), nil
}

func loadPreviousRelease(ctx context.Context, tx *sql.Tx, environmentID, failedReleaseID string) (string, error) {
	var releaseID string
	err := tx.QueryRowContext(ctx, `
		SELECT id
		FROM releases
		WHERE environment_id = ? AND id != ?
		ORDER BY created_at DESC
		LIMIT 1
	`, environmentID, failedReleaseID).Scan(&releaseID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrNoRollbackRelease
		}
		return "", fmt.Errorf("query previous release: %w", err)
	}

	return releaseID, nil
}

func markJobSucceeded(ctx context.Context, tx *sql.Tx, jobID, now string) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE jobs
		SET status = 'succeeded', finished_at = ?, error_code = NULL, error_message = NULL, updated_at = ?
		WHERE id = ?
	`, now, now, jobID); err != nil {
		return fmt.Errorf("mark job succeeded: %w", err)
	}

	return nil
}

func markJobFailed(ctx context.Context, tx *sql.Tx, jobID, errorCode, message, now string) error {
	if _, err := tx.ExecContext(ctx, `
		UPDATE jobs
		SET status = 'failed', finished_at = ?, error_code = ?, error_message = ?, updated_at = ?
		WHERE id = ?
	`, now, errorCode, truncateMessage(message), now, jobID); err != nil {
		return fmt.Errorf("mark job failed: %w", err)
	}

	return nil
}

func classifyHealthCheckErrorCode(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return HealthCheckErrorCodeTimeout
	}
	return HealthCheckErrorCodeFailed
}

func classifyRollbackErrorCode(err error) string {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return ReleaseRollbackErrorCodeTimeout
	}
	return ReleaseRollbackErrorCodeFailed
}

func messageFromError(err error) string {
	if err == nil {
		return "operation failed"
	}
	return err.Error()
}

func truncateMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "operation failed"
	}
	if len(message) > 512 {
		return message[:512]
	}
	return message
}

func isHealthTriggerJobType(jobType string) bool {
	switch jobType {
	case "env_deploy", "env_restore", "env_promote":
		return true
	default:
		return false
	}
}

func newUUIDv4() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate uuid: %w", err)
	}

	buf[6] = (buf[6] & 0x0f) | 0x40
	buf[8] = (buf[8] & 0x3f) | 0x80

	return fmt.Sprintf(
		"%08x-%04x-%04x-%04x-%012x",
		buf[0:4],
		buf[4:6],
		buf[6:8],
		buf[8:10],
		buf[10:16],
	), nil
}
