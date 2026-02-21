package backups

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

const BackupCleanupScheduleInterval = 6 * time.Hour

type cleanupCandidate struct {
	BackupID       string
	SiteID         string
	EnvironmentID  string
	NodeID         string
	StoragePath    string
	RetentionUntil string
}

type backupCleanupPayload struct {
	BackupID      string `json:"backup_id"`
	EnvironmentID string `json:"environment_id"`
	StoragePath   string `json:"storage_path"`
}

func (s *Service) EnqueueExpiredCleanup(ctx context.Context) (int, error) {
	now := time.Now().UTC().Format(time.RFC3339)
	queuedCount := 0

	err := store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		activeSites, activeBackups, err := loadActiveBackupCleanupScope(ctx, tx)
		if err != nil {
			return err
		}

		candidates, err := listExpiredCleanupCandidates(ctx, tx, now)
		if err != nil {
			return err
		}

		for _, candidate := range candidates {
			if _, exists := activeSites[candidate.SiteID]; exists {
				continue
			}
			if _, exists := activeBackups[candidate.BackupID]; exists {
				continue
			}

			jobID, err := newUUIDv4()
			if err != nil {
				return err
			}

			payload, err := json.Marshal(backupCleanupPayload{
				BackupID:      candidate.BackupID,
				EnvironmentID: candidate.EnvironmentID,
				StoragePath:   candidate.StoragePath,
			})
			if err != nil {
				return fmt.Errorf("marshal backup_cleanup payload: %w", err)
			}

			err = jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
				JobID:         jobID,
				JobType:       "backup_cleanup",
				SiteID:        sql.NullString{String: candidate.SiteID, Valid: true},
				EnvironmentID: sql.NullString{String: candidate.EnvironmentID, Valid: true},
				NodeID:        sql.NullString{String: candidate.NodeID, Valid: true},
				PayloadJSON:   string(payload),
			})
			if err != nil {
				if errors.Is(err, jobs.ErrConcurrencyConflict) {
					continue
				}
				return fmt.Errorf("enqueue backup_cleanup job: %w", err)
			}

			queuedCount++
			activeSites[candidate.SiteID] = struct{}{}
			activeBackups[candidate.BackupID] = struct{}{}
		}

		return nil
	})
	if err != nil {
		return 0, err
	}

	return queuedCount, nil
}

func RunCleanupScheduler(ctx context.Context, service *Service, interval time.Duration) error {
	if service == nil {
		return errors.New("backup cleanup scheduler service is nil")
	}

	if interval <= 0 {
		interval = BackupCleanupScheduleInterval
	}

	if _, err := service.EnqueueExpiredCleanup(ctx); err != nil {
		return fmt.Errorf("enqueue scheduled backup cleanup: %w", err)
	}

	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			if _, err := service.EnqueueExpiredCleanup(ctx); err != nil {
				return fmt.Errorf("enqueue scheduled backup cleanup: %w", err)
			}
		}
	}
}

func loadActiveBackupCleanupScope(ctx context.Context, tx *sql.Tx) (map[string]struct{}, map[string]struct{}, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT site_id, payload_json
		FROM jobs
		WHERE job_type = 'backup_cleanup'
		  AND status IN ('queued', 'running')
	`)
	if err != nil {
		return nil, nil, fmt.Errorf("list active backup_cleanup jobs: %w", err)
	}
	defer rows.Close()

	activeSites := make(map[string]struct{})
	activeBackups := make(map[string]struct{})
	for rows.Next() {
		var siteID sql.NullString
		var payloadJSON string
		if err := rows.Scan(&siteID, &payloadJSON); err != nil {
			return nil, nil, fmt.Errorf("scan active backup_cleanup job: %w", err)
		}
		if siteID.Valid {
			activeSites[siteID.String] = struct{}{}
		}

		var payload backupCleanupPayload
		if err := json.Unmarshal([]byte(payloadJSON), &payload); err == nil && payload.BackupID != "" {
			activeBackups[payload.BackupID] = struct{}{}
		}
	}

	if err := rows.Err(); err != nil {
		return nil, nil, fmt.Errorf("iterate active backup_cleanup jobs: %w", err)
	}

	return activeSites, activeBackups, nil
}

func listExpiredCleanupCandidates(ctx context.Context, tx *sql.Tx, now string) ([]cleanupCandidate, error) {
	rows, err := tx.QueryContext(ctx, `
		SELECT b.id, e.site_id, b.environment_id, e.node_id, b.storage_path, b.retention_until
		FROM backups b
		JOIN environments e ON e.id = b.environment_id
		WHERE b.retention_until < ?
		  AND b.status IN ('completed', 'failed')
		ORDER BY b.retention_until ASC, b.created_at ASC
	`, now)
	if err != nil {
		return nil, fmt.Errorf("query expired cleanup candidates: %w", err)
	}
	defer rows.Close()

	candidates := make([]cleanupCandidate, 0)
	for rows.Next() {
		var candidate cleanupCandidate
		if err := rows.Scan(
			&candidate.BackupID,
			&candidate.SiteID,
			&candidate.EnvironmentID,
			&candidate.NodeID,
			&candidate.StoragePath,
			&candidate.RetentionUntil,
		); err != nil {
			return nil, fmt.Errorf("scan expired cleanup candidate: %w", err)
		}
		candidates = append(candidates, candidate)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate expired cleanup candidates: %w", err)
	}

	return candidates, nil
}
