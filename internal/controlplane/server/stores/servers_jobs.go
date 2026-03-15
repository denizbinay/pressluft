package stores

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"pressluft/internal/orchestration/orchestrator"
	"pressluft/internal/platform"
	"pressluft/internal/shared/idutil"
)

func (s *ServerStore) QueueServerJob(ctx context.Context, in QueueServerJobInput) (StoredServer, orchestrator.Job, error) {
	serverPublicID, err := idutil.Normalize(in.ServerID)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("server_id: %w", err)
	}
	if !orchestrator.IsKnownJobKind(in.Kind) {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("unsupported job kind: %s", in.Kind)
	}

	tx, err := s.db.BeginTx(ctx, nil)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("begin transaction: %w", err)
	}
	defer tx.Rollback()

	var serverStatusRaw string
	if err := tx.QueryRowContext(ctx, `SELECT status FROM servers WHERE id = ?`, serverPublicID).Scan(&serverStatusRaw); err != nil {
		if err == sql.ErrNoRows {
			return StoredServer{}, orchestrator.Job{}, fmt.Errorf("server %s not found", serverPublicID)
		}
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("read server status: %w", err)
	}

	serverStatus, err := platform.NormalizeServerStatus(serverStatusRaw)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("normalize server status: %w", err)
	}
	if platform.IsDeletingOrDeletedServerStatus(string(serverStatus)) {
		if serverStatus == platform.ServerStatusDeleting {
			return StoredServer{}, orchestrator.Job{}, ErrServerDeleting
		}
		return StoredServer{}, orchestrator.Job{}, ErrServerDeleted
	}

	now := time.Now().UTC().Format(time.RFC3339)
	if queuedStatus, ok := orchestrator.QueuedServerStatusForKind(in.Kind); ok {
		var activeJobID string
		var activeJobKind string
		err := tx.QueryRowContext(ctx,
			`SELECT id, kind
				 FROM jobs
				 WHERE server_id = ?
				   AND kind IN (?, ?, ?)
				   AND status IN (?, ?)
				 ORDER BY created_at DESC
				 LIMIT 1`,
			serverPublicID,
			string(orchestrator.JobKindDeleteServer),
			string(orchestrator.JobKindRebuildServer),
			string(orchestrator.JobKindResizeServer),
			orchestrator.JobStatusQueued,
			orchestrator.JobStatusRunning,
		).Scan(&activeJobID, &activeJobKind)
		if err != nil && err != sql.ErrNoRows {
			return StoredServer{}, orchestrator.Job{}, fmt.Errorf("check active destructive jobs: %w", err)
		}
		if err == nil {
			return StoredServer{}, orchestrator.Job{}, fmt.Errorf("%w: job %s (%s)", ErrServerActionConflict, activeJobID, activeJobKind)
		}

		res, err := tx.ExecContext(ctx,
			`UPDATE servers SET status = ?, updated_at = ? WHERE id = ?`,
			string(queuedStatus),
			now,
			serverPublicID,
		)
		if err != nil {
			return StoredServer{}, orchestrator.Job{}, fmt.Errorf("update queued lifecycle status: %w", err)
		}
		if rows, _ := res.RowsAffected(); rows == 0 {
			return StoredServer{}, orchestrator.Job{}, fmt.Errorf("server %s not found", serverPublicID)
		}
	}
	jobPublicID, err := idutil.New()
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, err
	}

	_, err = tx.ExecContext(ctx,
		`INSERT INTO jobs (id, server_id, kind, status, current_step, retry_count, payload, created_at, updated_at)
		 VALUES (?, ?, ?, ?, '', 0, ?, ?, ?)`,
		jobPublicID,
		serverPublicID,
		in.Kind,
		orchestrator.JobStatusQueued,
		nullableString(in.Payload),
		now,
		now,
	)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("insert job: %w", err)
	}

	if err := tx.Commit(); err != nil {
		return StoredServer{}, orchestrator.Job{}, fmt.Errorf("commit transaction: %w", err)
	}

	server, err := s.GetByID(ctx, serverPublicID)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, err
	}
	job, err := orchestrator.NewStore(s.db).GetJob(ctx, jobPublicID)
	if err != nil {
		return StoredServer{}, orchestrator.Job{}, err
	}

	return *server, job, nil
}
