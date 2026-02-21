package jobs

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/store"
)

var ErrNotFound = errors.New("job not found")
var ErrInvalidInput = errors.New("invalid input")
var ErrNotCancellable = errors.New("job not cancellable")

type Job struct {
	ID            string  `json:"id"`
	JobType       string  `json:"job_type"`
	Status        string  `json:"status"`
	SiteID        *string `json:"site_id"`
	EnvironmentID *string `json:"environment_id"`
	NodeID        *string `json:"node_id"`
	AttemptCount  int     `json:"attempt_count"`
	MaxAttempts   int     `json:"max_attempts"`
	RunAfter      *string `json:"run_after"`
	LockedAt      *string `json:"locked_at"`
	LockedBy      *string `json:"locked_by"`
	StartedAt     *string `json:"started_at"`
	FinishedAt    *string `json:"finished_at"`
	ErrorCode     *string `json:"error_code"`
	ErrorMessage  *string `json:"error_message"`
	CreatedAt     string  `json:"created_at"`
	UpdatedAt     string  `json:"updated_at"`
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) List(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx, `
		SELECT id, job_type, status, site_id, environment_id, node_id,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		FROM jobs
		ORDER BY created_at DESC
	`)
	if err != nil {
		return nil, fmt.Errorf("query jobs list: %w", err)
	}
	defer rows.Close()

	jobsList := make([]Job, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, err
		}
		jobsList = append(jobsList, job)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs list: %w", err)
	}

	return jobsList, nil
}

func (s *Service) Get(ctx context.Context, id string) (Job, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Job{}, ErrInvalidInput
	}

	row := s.db.QueryRowContext(ctx, `
		SELECT id, job_type, status, site_id, environment_id, node_id,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		FROM jobs
		WHERE id = ?
		LIMIT 1
	`, id)

	job, err := scanJob(row)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return Job{}, ErrNotFound
		}
		return Job{}, err
	}

	return job, nil
}

func (s *Service) Cancel(ctx context.Context, id string) (Job, error) {
	id = strings.TrimSpace(id)
	if id == "" {
		return Job{}, ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	err := store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var status string
		if err := tx.QueryRowContext(ctx, `
			SELECT status
			FROM jobs
			WHERE id = ?
			LIMIT 1
		`, id).Scan(&status); err != nil {
			if errors.Is(err, sql.ErrNoRows) {
				return ErrNotFound
			}
			return fmt.Errorf("query job for cancel: %w", err)
		}

		if status != "queued" && status != "running" {
			return ErrNotCancellable
		}

		result, err := tx.ExecContext(ctx, `
			UPDATE jobs
			SET status = 'cancelled',
				finished_at = ?,
				locked_at = NULL,
				locked_by = NULL,
				error_code = NULL,
				error_message = NULL,
				updated_at = ?
			WHERE id = ?
			  AND status IN ('queued', 'running')
		`, now, now, id)
		if err != nil {
			return fmt.Errorf("cancel job: %w", err)
		}

		affected, err := result.RowsAffected()
		if err != nil {
			return fmt.Errorf("read cancel rows affected: %w", err)
		}
		if affected != 1 {
			return ErrNotCancellable
		}

		return nil
	})
	if err != nil {
		return Job{}, err
	}

	return s.Get(ctx, id)
}

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(scanner rowScanner) (Job, error) {
	var job Job
	var siteID sql.NullString
	var environmentID sql.NullString
	var nodeID sql.NullString
	var runAfter sql.NullString
	var lockedAt sql.NullString
	var lockedBy sql.NullString
	var startedAt sql.NullString
	var finishedAt sql.NullString
	var errorCode sql.NullString
	var errorMessage sql.NullString

	err := scanner.Scan(
		&job.ID,
		&job.JobType,
		&job.Status,
		&siteID,
		&environmentID,
		&nodeID,
		&job.AttemptCount,
		&job.MaxAttempts,
		&runAfter,
		&lockedAt,
		&lockedBy,
		&startedAt,
		&finishedAt,
		&errorCode,
		&errorMessage,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		return Job{}, fmt.Errorf("scan job row: %w", err)
	}

	job.SiteID = nullStringPtr(siteID)
	job.EnvironmentID = nullStringPtr(environmentID)
	job.NodeID = nullStringPtr(nodeID)
	job.RunAfter = nullStringPtr(runAfter)
	job.LockedAt = nullStringPtr(lockedAt)
	job.LockedBy = nullStringPtr(lockedBy)
	job.StartedAt = nullStringPtr(startedAt)
	job.FinishedAt = nullStringPtr(finishedAt)
	job.ErrorCode = nullStringPtr(errorCode)
	job.ErrorMessage = nullStringPtr(errorMessage)

	return job, nil
}

func nullStringPtr(value sql.NullString) *string {
	if !value.Valid {
		return nil
	}
	copy := value.String
	return &copy
}
