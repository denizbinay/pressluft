package orchestrator

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// Store persists orchestration jobs and timeline events.
type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) CreateJob(ctx context.Context, in CreateJobInput) (Job, error) {
	if strings.TrimSpace(in.Kind) == "" {
		return Job{}, fmt.Errorf("kind is required")
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs (server_id, kind, status, current_step, retry_count, payload, created_at, updated_at)
		 VALUES (?, ?, ?, '', 0, ?, ?, ?)`,
		nullableInt64(in.ServerID),
		in.Kind,
		JobStatusQueued,
		nullableString(in.Payload),
		now,
		now,
	)
	if err != nil {
		return Job{}, fmt.Errorf("insert job: %w", err)
	}

	id, err := res.LastInsertId()
	if err != nil {
		return Job{}, fmt.Errorf("job insert id: %w", err)
	}

	return s.GetJob(ctx, id)
}

func (s *Store) GetJob(ctx context.Context, id int64) (Job, error) {
	if id <= 0 {
		return Job{}, fmt.Errorf("id must be greater than zero")
	}

	var (
		job      Job
		serverID sql.NullInt64
		lastErr  sql.NullString
		payload  sql.NullString
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, server_id, kind, status, current_step, retry_count, last_error, payload, created_at, updated_at
		 FROM jobs
		 WHERE id = ?`,
		id,
	).Scan(
		&job.ID,
		&serverID,
		&job.Kind,
		&job.Status,
		&job.CurrentStep,
		&job.RetryCount,
		&lastErr,
		&payload,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return Job{}, fmt.Errorf("job %d not found", id)
		}
		return Job{}, fmt.Errorf("query job: %w", err)
	}
	if serverID.Valid {
		job.ServerID = serverID.Int64
	}
	if lastErr.Valid {
		job.LastError = lastErr.String
	}
	if payload.Valid {
		job.Payload = payload.String
	}
	return job, nil
}

func (s *Store) TransitionJob(ctx context.Context, id int64, in TransitionInput) (Job, error) {
	current, err := s.GetJob(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if err := ValidateTransition(current.Status, in.ToStatus); err != nil {
		return Job{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = s.db.ExecContext(ctx,
		`UPDATE jobs
		 SET status = ?, current_step = ?, retry_count = ?, last_error = ?, updated_at = ?
		 WHERE id = ?`,
		in.ToStatus,
		strings.TrimSpace(in.CurrentStep),
		in.RetryCount,
		nullableString(in.LastError),
		now,
		id,
	)
	if err != nil {
		return Job{}, fmt.Errorf("update job status: %w", err)
	}

	return s.GetJob(ctx, id)
}

func (s *Store) AppendEvent(ctx context.Context, jobID int64, in CreateEventInput) (JobEvent, error) {
	if jobID <= 0 {
		return JobEvent{}, fmt.Errorf("job_id must be greater than zero")
	}
	if strings.TrimSpace(in.EventType) == "" {
		return JobEvent{}, fmt.Errorf("event_type is required")
	}
	if strings.TrimSpace(in.Level) == "" {
		return JobEvent{}, fmt.Errorf("level is required")
	}
	if strings.TrimSpace(in.Message) == "" {
		return JobEvent{}, fmt.Errorf("message is required")
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return JobEvent{}, fmt.Errorf("begin event tx: %w", err)
	}
	defer tx.Rollback()

	var seq int64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(seq), 0) + 1 FROM job_events WHERE job_id = ?`,
		jobID,
	).Scan(&seq); err != nil {
		return JobEvent{}, fmt.Errorf("next event seq: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO job_events (job_id, seq, event_type, level, step_key, status, message, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		jobID,
		seq,
		in.EventType,
		in.Level,
		nullableString(in.StepKey),
		nullableString(in.Status),
		in.Message,
		nullableString(in.Payload),
		now,
	)
	if err != nil {
		return JobEvent{}, fmt.Errorf("insert event: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return JobEvent{}, fmt.Errorf("commit event tx: %w", err)
	}

	return JobEvent{
		JobID:      jobID,
		Seq:        seq,
		EventType:  in.EventType,
		Level:      in.Level,
		StepKey:    strings.TrimSpace(in.StepKey),
		Status:     strings.TrimSpace(in.Status),
		Message:    in.Message,
		Payload:    strings.TrimSpace(in.Payload),
		OccurredAt: now,
	}, nil
}

// ClaimNextJob atomically claims the oldest queued job by transitioning it to "preparing".
// Returns nil, nil if no jobs are available (not an error).
// Uses SQLite-compatible atomic UPDATE with subquery to prevent race conditions.
func (s *Store) ClaimNextJob(ctx context.Context) (*Job, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	// Atomic claim: UPDATE with subquery selects oldest queued job and transitions to preparing.
	// SQLite executes this atomically, preventing race conditions between workers.
	var (
		job      Job
		serverID sql.NullInt64
		lastErr  sql.NullString
		payload  sql.NullString
	)
	err := s.db.QueryRowContext(ctx,
		`UPDATE jobs 
		 SET status = ?, updated_at = ?
		 WHERE id = (
		   SELECT id FROM jobs 
		   WHERE status = ? 
		   ORDER BY created_at ASC 
		   LIMIT 1
		 )
		 RETURNING id, server_id, kind, status, current_step, retry_count, last_error, payload, created_at, updated_at`,
		JobStatusPreparing,
		now,
		JobStatusQueued,
	).Scan(
		&job.ID,
		&serverID,
		&job.Kind,
		&job.Status,
		&job.CurrentStep,
		&job.RetryCount,
		&lastErr,
		&payload,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			// No queued jobs available - not an error
			return nil, nil
		}
		return nil, fmt.Errorf("claim next job: %w", err)
	}

	if serverID.Valid {
		job.ServerID = serverID.Int64
	}
	if lastErr.Valid {
		job.LastError = lastErr.String
	}
	if payload.Valid {
		job.Payload = payload.String
	}

	return &job, nil
}

// ListAllJobs returns all jobs, ordered by created_at DESC.
// Returns an empty slice (not nil) when no jobs exist.
func (s *Store) ListAllJobs(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, server_id, kind, status, current_step, retry_count, last_error, payload, created_at, updated_at
		 FROM jobs
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list all jobs: %w", err)
	}
	defer rows.Close()

	out := make([]Job, 0)
	for rows.Next() {
		var (
			job       Job
			serverIDN sql.NullInt64
			lastErr   sql.NullString
			payload   sql.NullString
		)
		if err := rows.Scan(
			&job.ID,
			&serverIDN,
			&job.Kind,
			&job.Status,
			&job.CurrentStep,
			&job.RetryCount,
			&lastErr,
			&payload,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		if serverIDN.Valid {
			job.ServerID = serverIDN.Int64
		}
		if lastErr.Valid {
			job.LastError = lastErr.String
		}
		if payload.Valid {
			job.Payload = payload.String
		}
		out = append(out, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs: %w", err)
	}

	return out, nil
}

// ListJobsByServer returns all jobs for a given server, ordered by created_at DESC.
// Returns an empty slice (not nil) when no jobs exist for the server.
func (s *Store) ListJobsByServer(ctx context.Context, serverID int64) ([]Job, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server_id must be greater than zero")
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, server_id, kind, status, current_step, retry_count, last_error, payload, created_at, updated_at
		 FROM jobs
		 WHERE server_id = ?
		 ORDER BY created_at DESC`,
		serverID,
	)
	if err != nil {
		return nil, fmt.Errorf("list jobs by server: %w", err)
	}
	defer rows.Close()

	out := make([]Job, 0)
	for rows.Next() {
		var (
			job       Job
			serverIDN sql.NullInt64
			lastErr   sql.NullString
			payload   sql.NullString
		)
		if err := rows.Scan(
			&job.ID,
			&serverIDN,
			&job.Kind,
			&job.Status,
			&job.CurrentStep,
			&job.RetryCount,
			&lastErr,
			&payload,
			&job.CreatedAt,
			&job.UpdatedAt,
		); err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		if serverIDN.Valid {
			job.ServerID = serverIDN.Int64
		}
		if lastErr.Valid {
			job.LastError = lastErr.String
		}
		if payload.Valid {
			job.Payload = payload.String
		}
		out = append(out, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs: %w", err)
	}

	return out, nil
}

// GetLatestJobForServer returns the most recent job for a server.
// Returns nil, nil if no jobs exist for the server (not an error).
func (s *Store) GetLatestJobForServer(ctx context.Context, serverID int64) (*Job, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server_id must be greater than zero")
	}

	var (
		job       Job
		serverIDN sql.NullInt64
		lastErr   sql.NullString
		payload   sql.NullString
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT id, server_id, kind, status, current_step, retry_count, last_error, payload, created_at, updated_at
		 FROM jobs
		 WHERE server_id = ?
		 ORDER BY created_at DESC
		 LIMIT 1`,
		serverID,
	).Scan(
		&job.ID,
		&serverIDN,
		&job.Kind,
		&job.Status,
		&job.CurrentStep,
		&job.RetryCount,
		&lastErr,
		&payload,
		&job.CreatedAt,
		&job.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest job for server: %w", err)
	}
	if serverIDN.Valid {
		job.ServerID = serverIDN.Int64
	}
	if lastErr.Valid {
		job.LastError = lastErr.String
	}
	if payload.Valid {
		job.Payload = payload.String
	}
	return &job, nil
}

func (s *Store) ListEvents(ctx context.Context, jobID int64, afterSeq int64, limit int) ([]JobEvent, error) {
	if jobID <= 0 {
		return nil, fmt.Errorf("job_id must be greater than zero")
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT job_id, seq, event_type, level, step_key, status, message, payload, created_at
		 FROM job_events
		 WHERE job_id = ? AND seq > ?
		 ORDER BY seq ASC
		 LIMIT ?`,
		jobID,
		afterSeq,
		limit,
	)
	if err != nil {
		return nil, fmt.Errorf("list events: %w", err)
	}
	defer rows.Close()

	out := make([]JobEvent, 0, limit)
	for rows.Next() {
		var (
			e       JobEvent
			stepKey sql.NullString
			status  sql.NullString
			payload sql.NullString
		)
		if err := rows.Scan(
			&e.JobID,
			&e.Seq,
			&e.EventType,
			&e.Level,
			&stepKey,
			&status,
			&e.Message,
			&payload,
			&e.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.StepKey = nullString(stepKey)
		e.Status = nullString(status)
		e.Payload = nullString(payload)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return out, nil
}

// ListAllEvents returns all events for a job, ordered by sequence.
// Used for historical view of completed jobs.
func (s *Store) ListAllEvents(ctx context.Context, jobID int64) ([]JobEvent, error) {
	if jobID <= 0 {
		return nil, fmt.Errorf("job_id must be greater than zero")
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT job_id, seq, event_type, level, step_key, status, message, payload, created_at
		 FROM job_events
		 WHERE job_id = ?
		 ORDER BY seq ASC`,
		jobID,
	)
	if err != nil {
		return nil, fmt.Errorf("list all events: %w", err)
	}
	defer rows.Close()

	out := make([]JobEvent, 0)
	for rows.Next() {
		var (
			e       JobEvent
			stepKey sql.NullString
			status  sql.NullString
			payload sql.NullString
		)
		if err := rows.Scan(
			&e.JobID,
			&e.Seq,
			&e.EventType,
			&e.Level,
			&stepKey,
			&status,
			&e.Message,
			&payload,
			&e.OccurredAt,
		); err != nil {
			return nil, fmt.Errorf("scan event: %w", err)
		}
		e.StepKey = nullString(stepKey)
		e.Status = nullString(status)
		e.Payload = nullString(payload)
		out = append(out, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate events: %w", err)
	}

	return out, nil
}

// RecoverStuckJobs resets jobs that were interrupted mid-execution (e.g., server restart).
// Jobs in "preparing" or "running" status are transitioned back to "queued" so they can be re-claimed.
// Returns the number of jobs recovered.
func (s *Store) RecoverStuckJobs(ctx context.Context) (int64, error) {
	now := time.Now().UTC().Format(time.RFC3339)

	res, err := s.db.ExecContext(ctx,
		`UPDATE jobs 
		 SET status = ?, updated_at = ?
		 WHERE status IN (?, ?)`,
		JobStatusQueued,
		now,
		JobStatusPreparing,
		JobStatusRunning,
	)
	if err != nil {
		return 0, fmt.Errorf("recover stuck jobs: %w", err)
	}

	count, _ := res.RowsAffected()
	return count, nil
}

func nullableInt64(v int64) any {
	if v <= 0 {
		return nil
	}
	return v
}

func nullableString(v string) any {
	v = strings.TrimSpace(v)
	if v == "" {
		return nil
	}
	return v
}

func nullString(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}
