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
	if !IsKnownJobKind(in.Kind) {
		return Job{}, fmt.Errorf("unsupported job kind: %s", in.Kind)
	}

	now := nowRFC3339().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx,
		`INSERT INTO jobs (server_id, kind, status, current_step, retry_count, last_error, payload, started_at, finished_at, timeout_at, created_at, updated_at)
		 VALUES (?, ?, ?, '', 0, NULL, ?, NULL, NULL, NULL, ?, ?)`,
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

	row := s.db.QueryRowContext(ctx,
		`SELECT id, server_id, kind, status, current_step, retry_count, last_error, payload, started_at, finished_at, timeout_at, created_at, updated_at, command_id
		 FROM jobs
		 WHERE id = ?`,
		id,
	)
	job, err := scanJob(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return Job{}, fmt.Errorf("job %d not found", id)
		}
		return Job{}, fmt.Errorf("query job: %w", err)
	}
	return job, nil
}

func (s *Store) SetCommandID(ctx context.Context, jobID int64, commandID string) error {
	_, err := s.db.ExecContext(ctx,
		"UPDATE jobs SET command_id = ? WHERE id = ?",
		commandID, jobID,
	)
	return err
}

func (s *Store) GetJobByCommandID(ctx context.Context, commandID string) (*Job, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT id, server_id, kind, status, current_step, retry_count, last_error, payload, started_at, finished_at, timeout_at, created_at, updated_at, command_id
		 FROM jobs
		 WHERE command_id = ?`,
		commandID,
	)
	job, err := scanJob(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("job not found for command %s", commandID)
		}
		return nil, fmt.Errorf("query job by command_id: %w", err)
	}
	return &job, nil
}

func (s *Store) TransitionJob(ctx context.Context, id int64, in TransitionInput) (Job, error) {
	current, err := s.GetJob(ctx, id)
	if err != nil {
		return Job{}, err
	}
	if err := ValidateTransition(current.Status, in.ToStatus); err != nil {
		return Job{}, err
	}

	spec, ok := JobKindPolicy(current.Kind)
	if !ok {
		return Job{}, fmt.Errorf("unsupported job kind: %s", current.Kind)
	}

	now := nowRFC3339()
	nowText := now.Format(time.RFC3339)
	currentStep := strings.TrimSpace(in.CurrentStep)
	if currentStep == "" {
		currentStep = current.CurrentStep
	}
	lastError := strings.TrimSpace(in.LastError)
	startedAt := nullableString(current.StartedAt)
	finishedAt := nullableString(current.FinishedAt)
	timeoutAt := nullableString(current.TimeoutAt)

	switch in.ToStatus {
	case JobStatusRunning:
		if startedAt == nil {
			startedAt = nowText
		}
		finishedAt = nil
		if spec.Timeout > 0 {
			timeoutAt = now.Add(spec.Timeout).UTC().Format(time.RFC3339)
		} else {
			timeoutAt = nil
		}
		lastError = ""
	case JobStatusSucceeded, JobStatusFailed:
		finishedAt = nowText
		timeoutAt = nil
	}

	_, err = s.db.ExecContext(ctx,
		`UPDATE jobs
		 SET status = ?, current_step = ?, retry_count = ?, last_error = ?, started_at = ?, finished_at = ?, timeout_at = ?, updated_at = ?
		 WHERE id = ?`,
		in.ToStatus,
		currentStep,
		in.RetryCount,
		nullableString(lastError),
		startedAt,
		finishedAt,
		timeoutAt,
		nowText,
		id,
	)
	if err != nil {
		return Job{}, fmt.Errorf("update job status: %w", err)
	}

	return s.GetJob(ctx, id)
}

func (s *Store) MarkJobTimedOut(ctx context.Context, id int64, message string) (Job, bool, error) {
	return s.failActiveJob(ctx, id, JobEventTypeTimedOut, "error", message)
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

	event, err := appendEventTx(ctx, tx, jobID, in)
	if err != nil {
		return JobEvent{}, err
	}
	if err := tx.Commit(); err != nil {
		return JobEvent{}, fmt.Errorf("commit event tx: %w", err)
	}
	return event, nil
}

// ClaimNextJob atomically claims the oldest queued job by transitioning it to running.
// Returns nil, nil if no jobs are available.
func (s *Store) ClaimNextJob(ctx context.Context) (*Job, error) {
	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("begin claim tx: %w", err)
	}
	defer tx.Rollback()

	row := tx.QueryRowContext(ctx,
		`SELECT id, server_id, kind, status, current_step, retry_count, last_error, payload, started_at, finished_at, timeout_at, created_at, updated_at, command_id
		 FROM jobs
		 WHERE status = ?
		 ORDER BY created_at ASC
		 LIMIT 1`,
		JobStatusQueued,
	)
	job, err := scanJob(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("select queued job: %w", err)
	}

	spec, ok := JobKindPolicy(job.Kind)
	if !ok {
		return nil, fmt.Errorf("unsupported job kind: %s", job.Kind)
	}

	now := time.Now().UTC()
	timeoutAt := sql.NullString{}
	if spec.Timeout > 0 {
		timeoutAt = sql.NullString{String: now.Add(spec.Timeout).Format(time.RFC3339), Valid: true}
	}
	res, err := tx.ExecContext(ctx,
		`UPDATE jobs
		 SET status = ?, started_at = COALESCE(started_at, ?), finished_at = NULL, timeout_at = ?, updated_at = ?
		 WHERE id = ? AND status = ?`,
		JobStatusRunning,
		now.Format(time.RFC3339),
		nullableNullString(timeoutAt),
		now.Format(time.RFC3339),
		job.ID,
		JobStatusQueued,
	)
	if err != nil {
		return nil, fmt.Errorf("claim next job: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		return nil, nil
	}
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("commit claim tx: %w", err)
	}

	claimed, err := s.GetJob(ctx, job.ID)
	if err != nil {
		return nil, err
	}
	return &claimed, nil
}

// ListAllJobs returns all jobs, ordered by created_at DESC.
func (s *Store) ListAllJobs(ctx context.Context) ([]Job, error) {
	rows, err := s.db.QueryContext(ctx,
		`SELECT id, server_id, kind, status, current_step, retry_count, last_error, payload, started_at, finished_at, timeout_at, created_at, updated_at, command_id
		 FROM jobs
		 ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list all jobs: %w", err)
	}
	defer rows.Close()

	return scanJobs(rows)
}

// ListJobsByServer returns all jobs for a given server, ordered by created_at DESC.
func (s *Store) ListJobsByServer(ctx context.Context, serverID int64) ([]Job, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server_id must be greater than zero")
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT id, server_id, kind, status, current_step, retry_count, last_error, payload, started_at, finished_at, timeout_at, created_at, updated_at, command_id
		 FROM jobs
		 WHERE server_id = ?
		 ORDER BY created_at DESC`,
		serverID,
	)
	if err != nil {
		return nil, fmt.Errorf("list jobs by server: %w", err)
	}
	defer rows.Close()

	return scanJobs(rows)
}

// GetLatestJobForServer returns the most recent job for a server.
func (s *Store) GetLatestJobForServer(ctx context.Context, serverID int64) (*Job, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server_id must be greater than zero")
	}

	row := s.db.QueryRowContext(ctx,
		`SELECT id, server_id, kind, status, current_step, retry_count, last_error, payload, started_at, finished_at, timeout_at, created_at, updated_at, command_id
		 FROM jobs
		 WHERE server_id = ?
		 ORDER BY created_at DESC
		 LIMIT 1`,
		serverID,
	)
	job, err := scanJob(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get latest job for server: %w", err)
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

	return scanEvents(rows)
}

// ListAllEvents returns all events for a job, ordered by sequence.
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

	return scanEvents(rows)
}

// RecoverStuckJobs marks previously in-flight jobs as failed with an explicit recovery event.
func (s *Store) RecoverStuckJobs(ctx context.Context) (int64, error) {
	const message = "worker restarted before job completion; outcome may be incomplete; inspect state before retrying"

	rows, err := s.db.QueryContext(ctx,
		`SELECT id FROM jobs WHERE status = ? ORDER BY created_at ASC`,
		JobStatusRunning,
	)
	if err != nil {
		return 0, fmt.Errorf("list recoverable jobs: %w", err)
	}
	defer rows.Close()

	var ids []int64
	for rows.Next() {
		var id int64
		if err := rows.Scan(&id); err != nil {
			return 0, fmt.Errorf("scan recoverable job id: %w", err)
		}
		ids = append(ids, id)
	}
	if err := rows.Err(); err != nil {
		return 0, fmt.Errorf("iterate recoverable jobs: %w", err)
	}

	var count int64
	for _, id := range ids {
		if _, changed, err := s.failActiveJob(ctx, id, JobEventTypeRecovered, "error", message); err != nil {
			return count, err
		} else if changed {
			count++
		}
	}
	return count, nil
}

func (s *Store) failActiveJob(ctx context.Context, id int64, eventType, level, message string) (Job, bool, error) {
	job, err := s.GetJob(ctx, id)
	if err != nil {
		return Job{}, false, err
	}
	if IsTerminalStatus(job.Status) {
		return job, false, nil
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return Job{}, false, fmt.Errorf("begin job failure tx: %w", err)
	}
	defer tx.Rollback()

	now := nowRFC3339().Format(time.RFC3339)
	res, err := tx.ExecContext(ctx,
		`UPDATE jobs
		 SET status = ?, last_error = ?, finished_at = ?, timeout_at = NULL, updated_at = ?
		 WHERE id = ? AND status IN (?, ?)`,
		JobStatusFailed,
		message,
		now,
		now,
		id,
		JobStatusQueued,
		JobStatusRunning,
	)
	if err != nil {
		return Job{}, false, fmt.Errorf("mark job failed: %w", err)
	}
	if rows, _ := res.RowsAffected(); rows == 0 {
		current, getErr := s.GetJob(ctx, id)
		if getErr != nil {
			return Job{}, false, getErr
		}
		return current, false, nil
	}

	if _, err := appendEventTx(ctx, tx, id, CreateEventInput{
		EventType: eventType,
		Level:     level,
		StepKey:   job.CurrentStep,
		Status:    string(JobStatusFailed),
		Message:   message,
	}); err != nil {
		return Job{}, false, err
	}

	if err := tx.Commit(); err != nil {
		return Job{}, false, fmt.Errorf("commit job failure tx: %w", err)
	}
	updated, err := s.GetJob(ctx, id)
	if err != nil {
		return Job{}, false, err
	}
	return updated, true, nil
}

func appendEventTx(ctx context.Context, tx *sql.Tx, jobID int64, in CreateEventInput) (JobEvent, error) {
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(seq), 0) + 1 FROM job_events WHERE job_id = ?`,
		jobID,
	).Scan(&seq); err != nil {
		return JobEvent{}, fmt.Errorf("next event seq: %w", err)
	}

	now := nowRFC3339().Format(time.RFC3339)
	_, err := tx.ExecContext(ctx,
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

func scanJobs(rows *sql.Rows) ([]Job, error) {
	out := make([]Job, 0)
	for rows.Next() {
		job, err := scanJob(rows)
		if err != nil {
			return nil, fmt.Errorf("scan job: %w", err)
		}
		out = append(out, job)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate jobs: %w", err)
	}
	return out, nil
}

func scanEvents(rows *sql.Rows) ([]JobEvent, error) {
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

type rowScanner interface {
	Scan(dest ...any) error
}

func scanJob(scanner rowScanner) (Job, error) {
	var (
		job        Job
		serverID   sql.NullInt64
		lastErr    sql.NullString
		payload    sql.NullString
		startedAt  sql.NullString
		finishedAt sql.NullString
		timeoutAt  sql.NullString
		commandID  sql.NullString
	)
	err := scanner.Scan(
		&job.ID,
		&serverID,
		&job.Kind,
		&job.Status,
		&job.CurrentStep,
		&job.RetryCount,
		&lastErr,
		&payload,
		&startedAt,
		&finishedAt,
		&timeoutAt,
		&job.CreatedAt,
		&job.UpdatedAt,
		&commandID,
	)
	if err != nil {
		return Job{}, err
	}
	if serverID.Valid {
		job.ServerID = serverID.Int64
	}
	job.LastError = nullString(lastErr)
	job.Payload = nullString(payload)
	job.StartedAt = nullString(startedAt)
	job.FinishedAt = nullString(finishedAt)
	job.TimeoutAt = nullString(timeoutAt)
	if commandID.Valid {
		job.CommandID = &commandID.String
	}
	return job, nil
}

func nowRFC3339() time.Time {
	return time.Now().UTC()
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

func nullableNullString(v sql.NullString) any {
	if !v.Valid || strings.TrimSpace(v.String) == "" {
		return nil
	}
	return strings.TrimSpace(v.String)
}

func nullString(v sql.NullString) string {
	if !v.Valid {
		return ""
	}
	return v.String
}
