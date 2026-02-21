package jobs

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"
)

var ErrConcurrencyConflict = errors.New("mutation concurrency conflict")

type MutationJobInput struct {
	JobID         string
	JobType       string
	SiteID        sql.NullString
	EnvironmentID sql.NullString
	NodeID        sql.NullString
	PayloadJSON   string
}

func EnqueueMutationJob(ctx context.Context, tx *sql.Tx, input MutationJobInput) error {
	if input.SiteID.Valid {
		blocked, err := hasActiveMutation(ctx, tx, "site_id", input.SiteID.String)
		if err != nil {
			return err
		}
		if blocked {
			return fmt.Errorf("site %s: %w", input.SiteID.String, ErrConcurrencyConflict)
		}
	}

	if input.NodeID.Valid {
		blocked, err := hasActiveMutation(ctx, tx, "node_id", input.NodeID.String)
		if err != nil {
			return err
		}
		if blocked {
			return fmt.Errorf("node %s: %w", input.NodeID.String, ErrConcurrencyConflict)
		}
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if _, err := tx.ExecContext(ctx, `
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (?, ?, 'queued', ?, ?, ?, ?, 0, 3, NULL, NULL, NULL, NULL, NULL, NULL, NULL, ?, ?)
	`, input.JobID, input.JobType, input.SiteID, input.EnvironmentID, input.NodeID, input.PayloadJSON, now, now); err != nil {
		return fmt.Errorf("insert mutation job: %w", err)
	}

	return nil
}

func hasActiveMutation(ctx context.Context, tx *sql.Tx, column, id string) (bool, error) {
	query := fmt.Sprintf(`
		SELECT COUNT(1)
		FROM jobs
		WHERE %s = ?
		  AND status IN ('queued', 'running')
	`, column)

	var count int
	if err := tx.QueryRowContext(ctx, query, id).Scan(&count); err != nil {
		return false, fmt.Errorf("query active mutations for %s %s: %w", column, id, err)
	}

	return count > 0, nil
}

func EnsureNodeProvisionQueued(ctx context.Context, tx *sql.Tx, nodeID string) (bool, error) {
	payload, err := json.Marshal(map[string]string{"node_id": nodeID})
	if err != nil {
		return false, fmt.Errorf("marshal node_provision payload: %w", err)
	}

	jobID := fmt.Sprintf("job_node_provision_%d", time.Now().UTC().UnixNano())
	err = EnqueueMutationJob(ctx, tx, MutationJobInput{
		JobID:       jobID,
		JobType:     "node_provision",
		NodeID:      sql.NullString{String: nodeID, Valid: true},
		PayloadJSON: string(payload),
	})
	if err != nil {
		if errors.Is(err, ErrConcurrencyConflict) {
			return false, nil
		}
		return false, fmt.Errorf("enqueue node_provision job: %w", err)
	}

	return true, nil
}
