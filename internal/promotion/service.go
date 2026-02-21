package promotion

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

const backupFreshnessWindow = 60 * time.Minute

var ErrInvalidInput = errors.New("invalid input")
var ErrEnvironmentNotFound = errors.New("environment not found")
var ErrEnvironmentNotActive = errors.New("environment is not active")
var ErrDriftGateNotMet = errors.New("drift gate not met")
var ErrBackupGateNotMet = errors.New("backup gate not met")

type DriftCheckInput struct {
	EnvironmentID string
}

type DriftCheckResult struct {
	JobID        string
	DriftCheckID string
}

type PromoteInput struct {
	EnvironmentID       string
	TargetEnvironmentID string
}

type PromoteResult struct {
	JobID string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) DriftCheck(ctx context.Context, input DriftCheckInput) (DriftCheckResult, error) {
	if strings.TrimSpace(input.EnvironmentID) == "" {
		return DriftCheckResult{}, ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	driftCheckID, err := newUUIDv4()
	if err != nil {
		return DriftCheckResult{}, err
	}
	jobID, err := newUUIDv4()
	if err != nil {
		return DriftCheckResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		envRef, err := loadEnvironmentRef(ctx, tx, strings.TrimSpace(input.EnvironmentID))
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			INSERT INTO drift_checks (id, environment_id, promotion_preset, status, db_checksums_json, file_checksums_json, checked_at)
			VALUES (?, ?, ?, 'clean', '{}', '{}', ?)
		`, driftCheckID, envRef.EnvironmentID, envRef.PromotionPreset, now); err != nil {
			return fmt.Errorf("insert drift check: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET drift_status = 'clean', drift_checked_at = ?, last_drift_check_id = ?, updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, driftCheckID, now, envRef.EnvironmentID); err != nil {
			return fmt.Errorf("update environment drift status: %w", err)
		}

		payload, err := json.Marshal(map[string]string{
			"environment_id":   envRef.EnvironmentID,
			"drift_check_id":   driftCheckID,
			"promotion_preset": envRef.PromotionPreset,
		})
		if err != nil {
			return fmt.Errorf("marshal drift_check payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "drift_check",
			SiteID:        sql.NullString{String: envRef.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: envRef.EnvironmentID, Valid: true},
			NodeID:        sql.NullString{String: envRef.NodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue drift_check job: %w", err)
		}

		return nil
	})
	if err != nil {
		return DriftCheckResult{}, err
	}

	return DriftCheckResult{JobID: jobID, DriftCheckID: driftCheckID}, nil
}

func (s *Service) Promote(ctx context.Context, input PromoteInput) (PromoteResult, error) {
	if strings.TrimSpace(input.EnvironmentID) == "" || strings.TrimSpace(input.TargetEnvironmentID) == "" {
		return PromoteResult{}, ErrInvalidInput
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	jobID, err := newUUIDv4()
	if err != nil {
		return PromoteResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		sourceRef, err := loadEnvironmentRef(ctx, tx, strings.TrimSpace(input.EnvironmentID))
		if err != nil {
			return err
		}
		targetRef, err := loadEnvironmentRef(ctx, tx, strings.TrimSpace(input.TargetEnvironmentID))
		if err != nil {
			return err
		}

		if sourceRef.SiteID != targetRef.SiteID {
			return ErrInvalidInput
		}

		if sourceRef.DriftStatus != "clean" || !sourceRef.LastDriftCheckID.Valid {
			return ErrDriftGateNotMet
		}

		backupID, err := loadFreshCompletedFullBackup(ctx, tx, targetRef.EnvironmentID, now)
		if err != nil {
			return err
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'deploying', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, nowStr, targetRef.EnvironmentID); err != nil {
			return fmt.Errorf("mark promote target deploying: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'deploying', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, nowStr, targetRef.SiteID); err != nil {
			return fmt.Errorf("mark site deploying for promote: %w", err)
		}

		payload, err := json.Marshal(map[string]string{
			"source_environment_id": sourceRef.EnvironmentID,
			"target_environment_id": targetRef.EnvironmentID,
			"promotion_preset":      sourceRef.PromotionPreset,
			"drift_check_id":        sourceRef.LastDriftCheckID.String,
			"pre_promote_backup_id": backupID,
		})
		if err != nil {
			return fmt.Errorf("marshal env_promote payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "env_promote",
			SiteID:        sql.NullString{String: targetRef.SiteID, Valid: true},
			EnvironmentID: sql.NullString{String: targetRef.EnvironmentID, Valid: true},
			NodeID:        sql.NullString{String: targetRef.NodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue env_promote job: %w", err)
		}

		return nil
	})
	if err != nil {
		return PromoteResult{}, err
	}

	return PromoteResult{JobID: jobID}, nil
}

func (s *Service) MarkPromoteSucceeded(ctx context.Context, siteID, targetEnvironmentID, jobID string) error {
	if strings.TrimSpace(siteID) == "" || strings.TrimSpace(targetEnvironmentID) == "" || strings.TrimSpace(jobID) == "" {
		return ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'active', updated_at = ?, state_version = state_version + 1
			WHERE id = ? AND site_id = ?
		`, now, targetEnvironmentID, siteID); err != nil {
			return fmt.Errorf("mark promoted environment active: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'active', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, siteID); err != nil {
			return fmt.Errorf("mark promoted site active: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'succeeded', finished_at = ?, error_code = NULL, error_message = NULL, updated_at = ?
			WHERE id = ?
		`, now, now, jobID); err != nil {
			return fmt.Errorf("mark promote job succeeded: %w", err)
		}

		return nil
	})
}

func (s *Service) MarkPromoteFailed(ctx context.Context, siteID, targetEnvironmentID, jobID, errorCode, errorMessage string) error {
	if strings.TrimSpace(siteID) == "" || strings.TrimSpace(targetEnvironmentID) == "" || strings.TrimSpace(jobID) == "" {
		return ErrInvalidInput
	}

	if strings.TrimSpace(errorCode) == "" {
		errorCode = "ENV_PROMOTE_FAILED"
	}
	if strings.TrimSpace(errorMessage) == "" {
		errorMessage = "environment promotion failed"
	}

	now := time.Now().UTC().Format(time.RFC3339)
	return store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			UPDATE environments
			SET status = 'failed', updated_at = ?, state_version = state_version + 1
			WHERE id = ? AND site_id = ?
		`, now, targetEnvironmentID, siteID); err != nil {
			return fmt.Errorf("mark promoted environment failed: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE sites
			SET status = 'failed', updated_at = ?, state_version = state_version + 1
			WHERE id = ?
		`, now, siteID); err != nil {
			return fmt.Errorf("mark promoted site failed: %w", err)
		}

		if _, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'failed', finished_at = ?, error_code = ?, error_message = ?, updated_at = ?
			WHERE id = ?
		`, now, strings.TrimSpace(errorCode), truncateMessage(errorMessage), now, jobID); err != nil {
			return fmt.Errorf("mark promote job failed: %w", err)
		}

		return nil
	})
}

type environmentRef struct {
	EnvironmentID    string
	SiteID           string
	NodeID           string
	Status           string
	PromotionPreset  string
	DriftStatus      string
	LastDriftCheckID sql.NullString
}

func loadEnvironmentRef(ctx context.Context, tx *sql.Tx, environmentID string) (environmentRef, error) {
	var ref environmentRef
	err := tx.QueryRowContext(ctx, `
		SELECT id, site_id, node_id, status, promotion_preset, drift_status, last_drift_check_id
		FROM environments
		WHERE id = ?
		LIMIT 1
	`, environmentID).Scan(
		&ref.EnvironmentID,
		&ref.SiteID,
		&ref.NodeID,
		&ref.Status,
		&ref.PromotionPreset,
		&ref.DriftStatus,
		&ref.LastDriftCheckID,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return environmentRef{}, ErrEnvironmentNotFound
		}
		return environmentRef{}, fmt.Errorf("query environment for promotion: %w", err)
	}

	if ref.Status != "active" {
		return environmentRef{}, ErrEnvironmentNotActive
	}

	return ref, nil
}

func loadFreshCompletedFullBackup(ctx context.Context, tx *sql.Tx, environmentID string, now time.Time) (string, error) {
	var backupID string
	var completedAt string
	err := tx.QueryRowContext(ctx, `
		SELECT id, completed_at
		FROM backups
		WHERE environment_id = ? AND backup_scope = 'full' AND status = 'completed'
		ORDER BY completed_at DESC
		LIMIT 1
	`, environmentID).Scan(&backupID, &completedAt)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrBackupGateNotMet
		}
		return "", fmt.Errorf("query latest completed full backup: %w", err)
	}

	parsedCompletedAt, parseErr := time.Parse(time.RFC3339, strings.TrimSpace(completedAt))
	if parseErr != nil {
		return "", ErrBackupGateNotMet
	}
	if now.Sub(parsedCompletedAt) > backupFreshnessWindow {
		return "", ErrBackupGateNotMet
	}

	return backupID, nil
}

func truncateMessage(message string) string {
	message = strings.TrimSpace(message)
	if message == "" {
		message = "environment promotion failed"
	}
	if len(message) > 512 {
		return message[:512]
	}
	return message
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
