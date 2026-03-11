package orchestrator

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/idutil"
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
	publicID, err := idutil.New()
	if err != nil {
		return Job{}, err
	}
	_, err = s.db.ExecContext(ctx,
		`INSERT INTO jobs (id, server_id, kind, status, current_step, retry_count, last_error, payload, started_at, finished_at, timeout_at, created_at, updated_at)
		 VALUES (?, ?, ?, ?, '', 0, NULL, ?, NULL, NULL, NULL, ?, ?)`,
		publicID,
		nullableString(in.ServerID),
		in.Kind,
		JobStatusQueued,
		nullableString(in.Payload),
		now,
		now,
	)
	if err != nil {
		return Job{}, fmt.Errorf("insert job: %w", err)
	}
	return s.GetJob(ctx, publicID)
}

func (s *Store) GetJob(ctx context.Context, id string) (Job, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return Job{}, err
	}

	row := s.db.QueryRowContext(ctx,
		`SELECT j.id, COALESCE(s.id, ''), j.kind, j.status, j.current_step, j.retry_count, j.last_error, j.payload, j.started_at, j.finished_at, j.timeout_at, j.created_at, j.updated_at, j.command_id
		 FROM jobs j
		 LEFT JOIN servers s ON s.id = j.server_id
		 WHERE j.id = ?`,
		publicID,
	)
	job, err := scanJob(row)
	if err != nil {
		if err == sql.ErrNoRows {
			return Job{}, fmt.Errorf("job %s not found", publicID)
		}
		return Job{}, fmt.Errorf("query job: %w", err)
	}
	return job, nil
}

func (s *Store) SetCommandID(ctx context.Context, jobID string, commandID string) error {
	jobID, err := s.lookupJobID(ctx, jobID)
	if err != nil {
		return err
	}
	_, err = s.db.ExecContext(ctx,
		"UPDATE jobs SET command_id = ? WHERE id = ?",
		commandID, jobID,
	)
	return err
}

func (s *Store) GetJobByCommandID(ctx context.Context, commandID string) (*Job, error) {
	row := s.db.QueryRowContext(ctx,
		`SELECT j.id, COALESCE(s.id, ''), j.kind, j.status, j.current_step, j.retry_count, j.last_error, j.payload, j.started_at, j.finished_at, j.timeout_at, j.created_at, j.updated_at, j.command_id
		 FROM jobs j
		 LEFT JOIN servers s ON s.id = j.server_id
		 WHERE j.command_id = ?`,
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

func (s *Store) TransitionJob(ctx context.Context, id string, in TransitionInput) (Job, error) {
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
		current.ID,
	)
	if err != nil {
		return Job{}, fmt.Errorf("update job status: %w", err)
	}

	return s.GetJob(ctx, id)
}

func (s *Store) MarkJobTimedOut(ctx context.Context, id string, message string) (Job, bool, error) {
	return s.failActiveJob(ctx, id, JobEventTypeTimedOut, "error", message)
}

func (s *Store) AppendEvent(ctx context.Context, jobID string, in CreateEventInput) (JobEvent, error) {
	jobID, err := s.lookupJobID(ctx, jobID)
	if err != nil {
		return JobEvent{}, err
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
		`SELECT j.id, COALESCE(s.id, ''), j.kind, j.status, j.current_step, j.retry_count, j.last_error, j.payload, j.started_at, j.finished_at, j.timeout_at, j.created_at, j.updated_at, j.command_id
		 FROM jobs j
		 LEFT JOIN servers s ON s.id = j.server_id
		 WHERE j.status = ?
		 ORDER BY j.created_at ASC
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
		`SELECT j.id, COALESCE(s.id, ''), j.kind, j.status, j.current_step, j.retry_count, j.last_error, j.payload, j.started_at, j.finished_at, j.timeout_at, j.created_at, j.updated_at, j.command_id
		 FROM jobs j
		 LEFT JOIN servers s ON s.id = j.server_id
		 ORDER BY j.created_at DESC`,
	)
	if err != nil {
		return nil, fmt.Errorf("list all jobs: %w", err)
	}
	defer rows.Close()

	return scanJobs(rows)
}

// ListJobsByServer returns all jobs for a given server, ordered by created_at DESC.
func (s *Store) ListJobsByServer(ctx context.Context, serverID string) ([]Job, error) {
	serverID, err := lookupServerID(ctx, s.db, serverID)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT j.id, COALESCE(s.id, ''), j.kind, j.status, j.current_step, j.retry_count, j.last_error, j.payload, j.started_at, j.finished_at, j.timeout_at, j.created_at, j.updated_at, j.command_id
		 FROM jobs j
		 LEFT JOIN servers s ON s.id = j.server_id
		 WHERE j.server_id = ?
		 ORDER BY j.created_at DESC`,
		serverID,
	)
	if err != nil {
		return nil, fmt.Errorf("list jobs by server: %w", err)
	}
	defer rows.Close()

	return scanJobs(rows)
}

// GetLatestJobForServer returns the most recent job for a server.
func (s *Store) GetLatestJobForServer(ctx context.Context, serverID string) (*Job, error) {
	serverID, err := lookupServerID(ctx, s.db, serverID)
	if err != nil {
		return nil, err
	}

	row := s.db.QueryRowContext(ctx,
		`SELECT j.id, COALESCE(s.id, ''), j.kind, j.status, j.current_step, j.retry_count, j.last_error, j.payload, j.started_at, j.finished_at, j.timeout_at, j.created_at, j.updated_at, j.command_id
		 FROM jobs j
		 LEFT JOIN servers s ON s.id = j.server_id
		 WHERE j.server_id = ?
		 ORDER BY j.created_at DESC
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

func (s *Store) ListEvents(ctx context.Context, jobID string, afterSeq int64, limit int) ([]JobEvent, error) {
	jobID, err := s.lookupJobID(ctx, jobID)
	if err != nil {
		return nil, err
	}
	if limit <= 0 {
		limit = 100
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT e.id, j.id, e.seq, e.event_type, e.level, e.step_key, e.status, e.message, e.payload, e.created_at
		 FROM job_events e
		 JOIN jobs j ON j.id = e.job_id
		 WHERE e.job_id = ? AND e.seq > ?
		 ORDER BY e.seq ASC
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
func (s *Store) ListAllEvents(ctx context.Context, jobID string) ([]JobEvent, error) {
	jobID, err := s.lookupJobID(ctx, jobID)
	if err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx,
		`SELECT e.id, j.id, e.seq, e.event_type, e.level, e.step_key, e.status, e.message, e.payload, e.created_at
		 FROM job_events e
		 JOIN jobs j ON j.id = e.job_id
		 WHERE e.job_id = ?
		 ORDER BY e.seq ASC`,
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

	var ids []string
	for rows.Next() {
		var id string
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

func (s *Store) failActiveJob(ctx context.Context, id string, eventType, level, message string) (Job, bool, error) {
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
		job.ID,
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

	if _, err := appendEventTx(ctx, tx, job.ID, CreateEventInput{
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

func appendEventTx(ctx context.Context, tx *sql.Tx, jobID string, in CreateEventInput) (JobEvent, error) {
	var seq int64
	if err := tx.QueryRowContext(ctx,
		`SELECT COALESCE(MAX(seq), 0) + 1 FROM job_events WHERE job_id = ?`,
		jobID,
	).Scan(&seq); err != nil {
		return JobEvent{}, fmt.Errorf("next event seq: %w", err)
	}

	eventID, err := idutil.New()
	if err != nil {
		return JobEvent{}, err
	}
	now := nowRFC3339().Format(time.RFC3339)
	_, err = tx.ExecContext(ctx,
		`INSERT INTO job_events (id, job_id, seq, event_type, level, step_key, status, message, payload, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		eventID,
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
		ID:         eventID,
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
			&e.ID,
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
		serverID   sql.NullString
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
	job.ServerID = nullString(serverID)
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

func (s *Store) lookupJobID(ctx context.Context, id string) (string, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return "", err
	}
	var jobID string
	if err := s.db.QueryRowContext(ctx, `SELECT id FROM jobs WHERE id = ?`, publicID).Scan(&jobID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("job %s not found", publicID)
		}
		return "", fmt.Errorf("lookup job id: %w", err)
	}
	return jobID, nil
}

func lookupServerID(ctx context.Context, db *sql.DB, id string) (string, error) {
	publicID, err := idutil.Normalize(id)
	if err != nil {
		return "", err
	}
	var serverID string
	if err := db.QueryRowContext(ctx, `SELECT id FROM servers WHERE id = ?`, publicID).Scan(&serverID); err != nil {
		if err == sql.ErrNoRows {
			return "", fmt.Errorf("server %s not found", publicID)
		}
		return "", fmt.Errorf("lookup server id: %w", err)
	}
	return serverID, nil
}
