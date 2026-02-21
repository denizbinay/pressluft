package backups

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

const defaultRetentionDays = 30

var ErrInvalidInput = errors.New("invalid input")
var ErrEnvironmentNotFound = errors.New("environment not found")
var ErrInvalidTransition = errors.New("invalid transition")

type Backup struct {
	ID             string  `json:"id"`
	EnvironmentID  string  `json:"environment_id"`
	BackupScope    string  `json:"backup_scope"`
	Status         string  `json:"status"`
	StorageType    string  `json:"storage_type"`
	StoragePath    string  `json:"storage_path"`
	RetentionUntil string  `json:"retention_until"`
	Checksum       *string `json:"checksum"`
	SizeBytes      *int64  `json:"size_bytes"`
	CreatedAt      string  `json:"created_at"`
	CompletedAt    *string `json:"completed_at"`
}

type CreateInput struct {
	EnvironmentID string
	BackupScope   string
}

type CreateResult struct {
	BackupID string
	JobID    string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Create(ctx context.Context, input CreateInput) (CreateResult, error) {
	if err := validateCreateInput(input); err != nil {
		return CreateResult{}, err
	}

	now := time.Now().UTC()
	nowStr := now.Format(time.RFC3339)
	retentionUntil := now.Add(defaultRetentionDays * 24 * time.Hour).Format(time.RFC3339)

	backupID, err := newUUIDv4()
	if err != nil {
		return CreateResult{}, err
	}
	jobID, err := newUUIDv4()
	if err != nil {
		return CreateResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		siteID, nodeID, err := loadEnvironmentRefs(ctx, tx, input.EnvironmentID)
		if err != nil {
			return err
		}

		storagePath := buildStoragePath(input.EnvironmentID, backupID)
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO backups (
				id, environment_id, backup_scope, status, storage_type, storage_path,
				retention_until, checksum, size_bytes, created_at, completed_at
			)
			VALUES (?, ?, ?, 'pending', 's3', ?, ?, NULL, NULL, ?, NULL)
		`, backupID, input.EnvironmentID, input.BackupScope, storagePath, retentionUntil, nowStr); err != nil {
			return fmt.Errorf("insert backup: %w", err)
		}

		payload, err := json.Marshal(map[string]string{
			"backup_id":      backupID,
			"environment_id": input.EnvironmentID,
			"backup_scope":   input.BackupScope,
			"storage_path":   storagePath,
		})
		if err != nil {
			return fmt.Errorf("marshal backup_create payload: %w", err)
		}

		if err := jobs.EnqueueMutationJob(ctx, tx, jobs.MutationJobInput{
			JobID:         jobID,
			JobType:       "backup_create",
			SiteID:        sql.NullString{String: siteID, Valid: true},
			EnvironmentID: sql.NullString{String: input.EnvironmentID, Valid: true},
			NodeID:        sql.NullString{String: nodeID, Valid: true},
			PayloadJSON:   string(payload),
		}); err != nil {
			return fmt.Errorf("enqueue backup_create job: %w", err)
		}

		return nil
	})
	if err != nil {
		return CreateResult{}, err
	}

	return CreateResult{BackupID: backupID, JobID: jobID}, nil
}

func (s *Service) ListByEnvironment(ctx context.Context, environmentID string) ([]Backup, error) {
	if strings.TrimSpace(environmentID) == "" {
		return nil, ErrInvalidInput
	}

	if _, _, err := loadEnvironmentRefs(ctx, s.db, environmentID); err != nil {
		return nil, err
	}

	rows, err := s.db.QueryContext(ctx, `
		SELECT id, environment_id, backup_scope, status, storage_type, storage_path,
		       retention_until, checksum, size_bytes, created_at, completed_at
		FROM backups
		WHERE environment_id = ?
		ORDER BY created_at DESC
	`, environmentID)
	if err != nil {
		return nil, fmt.Errorf("query backups for environment: %w", err)
	}
	defer rows.Close()

	backups := make([]Backup, 0)
	for rows.Next() {
		backup, err := scanBackup(rows)
		if err != nil {
			return nil, err
		}
		backups = append(backups, backup)
	}

	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate backups: %w", err)
	}

	return backups, nil
}

func (s *Service) MarkRunning(ctx context.Context, backupID string) error {
	if strings.TrimSpace(backupID) == "" {
		return ErrInvalidInput
	}

	updated, err := updateBackupStatus(ctx, s.db, backupID, "pending", "running")
	if err != nil {
		return err
	}
	if !updated {
		return ErrInvalidTransition
	}
	return nil
}

func (s *Service) MarkCompleted(ctx context.Context, backupID, checksum string, sizeBytes int64) error {
	if strings.TrimSpace(backupID) == "" {
		return ErrInvalidInput
	}
	if strings.TrimSpace(checksum) == "" || sizeBytes < 0 {
		return ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
		UPDATE backups
		SET status = 'completed', checksum = ?, size_bytes = ?, completed_at = ?
		WHERE id = ? AND status = 'running'
	`, strings.TrimSpace(checksum), sizeBytes, now, backupID)
	if err != nil {
		return fmt.Errorf("mark backup completed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected for complete backup: %w", err)
	}
	if rows == 0 {
		return ErrInvalidTransition
	}

	return nil
}

func (s *Service) MarkFailed(ctx context.Context, backupID string) error {
	if strings.TrimSpace(backupID) == "" {
		return ErrInvalidInput
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
		UPDATE backups
		SET status = 'failed', completed_at = ?
		WHERE id = ? AND status = 'running'
	`, now, backupID)
	if err != nil {
		return fmt.Errorf("mark backup failed: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected for fail backup: %w", err)
	}
	if rows == 0 {
		return ErrInvalidTransition
	}

	return nil
}

func (s *Service) MarkExpired(ctx context.Context, backupID string) error {
	if strings.TrimSpace(backupID) == "" {
		return ErrInvalidInput
	}

	result, err := s.db.ExecContext(ctx, `
		UPDATE backups
		SET status = 'expired'
		WHERE id = ? AND status IN ('completed', 'failed')
	`, backupID)
	if err != nil {
		return fmt.Errorf("mark backup expired: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read rows affected for expire backup: %w", err)
	}
	if rows == 0 {
		return ErrInvalidTransition
	}

	return nil
}

func loadEnvironmentRefs(ctx context.Context, q interface {
	QueryRowContext(context.Context, string, ...any) *sql.Row
}, environmentID string) (string, string, error) {
	var siteID string
	var nodeID string
	err := q.QueryRowContext(ctx, `
		SELECT site_id, node_id
		FROM environments
		WHERE id = ?
		LIMIT 1
	`, environmentID).Scan(&siteID, &nodeID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", "", ErrEnvironmentNotFound
		}
		return "", "", fmt.Errorf("query environment refs: %w", err)
	}

	return siteID, nodeID, nil
}

func validateCreateInput(input CreateInput) error {
	if strings.TrimSpace(input.EnvironmentID) == "" {
		return ErrInvalidInput
	}
	if input.BackupScope != "db" && input.BackupScope != "files" && input.BackupScope != "full" {
		return ErrInvalidInput
	}
	return nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanBackup(scanner scanner) (Backup, error) {
	var backup Backup
	var checksum sql.NullString
	var sizeBytes sql.NullInt64
	var completedAt sql.NullString

	if err := scanner.Scan(
		&backup.ID,
		&backup.EnvironmentID,
		&backup.BackupScope,
		&backup.Status,
		&backup.StorageType,
		&backup.StoragePath,
		&backup.RetentionUntil,
		&checksum,
		&sizeBytes,
		&backup.CreatedAt,
		&completedAt,
	); err != nil {
		return Backup{}, err
	}

	if checksum.Valid {
		backup.Checksum = &checksum.String
	}
	if sizeBytes.Valid {
		value := sizeBytes.Int64
		backup.SizeBytes = &value
	}
	if completedAt.Valid {
		backup.CompletedAt = &completedAt.String
	}

	return backup, nil
}

func updateBackupStatus(ctx context.Context, db *sql.DB, backupID, fromStatus, toStatus string) (bool, error) {
	result, err := db.ExecContext(ctx, `
		UPDATE backups
		SET status = ?
		WHERE id = ? AND status = ?
	`, toStatus, backupID, fromStatus)
	if err != nil {
		return false, fmt.Errorf("update backup status to %s: %w", toStatus, err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("read rows affected for backup status %s: %w", toStatus, err)
	}

	return rows > 0, nil
}

func buildStoragePath(environmentID, backupID string) string {
	return fmt.Sprintf("s3://pressluft/backups/%s/%s.tar.zst", environmentID, backupID)
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
