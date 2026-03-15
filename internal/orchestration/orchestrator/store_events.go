package orchestrator

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/shared/idutil"
)

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
