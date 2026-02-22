package backups

import (
	"context"
	"crypto/rand"
	"errors"
	"fmt"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

const (
	defaultJobType        = "backup_create"
	defaultStorageType    = "s3"
	defaultRetentionDays  = 30
	defaultStoragePathFmt = "s3://pressluft/backups/%s/%s.tar.zst"
)

var ErrMutationConflict = errors.New("backup mutation conflict")

type JobQueue interface {
	Enqueue(ctx context.Context, input jobs.EnqueueInput) (jobs.Job, error)
}

type Service struct {
	sites   store.SiteStore
	backups store.BackupStore
	queue   JobQueue
	now     func() time.Time
	jobType string
}

type CreateInput struct {
	EnvironmentID string
	BackupScope   string
}

func NewService(siteStore store.SiteStore, backupStore store.BackupStore, queue JobQueue) *Service {
	return &Service{
		sites:   siteStore,
		backups: backupStore,
		queue:   queue,
		now:     func() time.Time { return time.Now().UTC() },
		jobType: defaultJobType,
	}
}

func (s *Service) ListByEnvironmentID(ctx context.Context, environmentID string) ([]store.Backup, error) {
	if _, err := s.sites.GetEnvironmentByID(ctx, environmentID); err != nil {
		return nil, fmt.Errorf("get environment by id: %w", err)
	}

	items, err := s.backups.ListBackupsByEnvironmentID(ctx, environmentID, s.now())
	if err != nil {
		return nil, fmt.Errorf("list backups by environment id: %w", err)
	}
	return items, nil
}

func (s *Service) Create(ctx context.Context, input CreateInput) (string, error) {
	if !isValidScope(input.BackupScope) {
		return "", store.ErrInvalidBackupScope
	}

	environment, err := s.sites.GetEnvironmentByID(ctx, input.EnvironmentID)
	if err != nil {
		return "", fmt.Errorf("get environment by id: %w", err)
	}

	now := s.now()
	backupID, err := generateUUIDv4()
	if err != nil {
		return "", fmt.Errorf("generate backup id: %w", err)
	}

	if _, err := s.queue.Enqueue(ctx, jobs.EnqueueInput{
		ID:            backupID,
		JobType:       s.jobType,
		SiteID:        &environment.SiteID,
		EnvironmentID: &environment.ID,
		NodeID:        &environment.NodeID,
		MaxAttempts:   3,
		CreatedAt:     now,
	}); err != nil {
		if errors.Is(err, jobs.ErrConflict) {
			return "", ErrMutationConflict
		}
		return "", fmt.Errorf("enqueue backup create job: %w", err)
	}

	retention := now.AddDate(0, 0, defaultRetentionDays)
	storagePath := fmt.Sprintf(defaultStoragePathFmt, environment.ID, backupID)
	if _, err := s.backups.CreateBackup(ctx, store.CreateBackupInput{
		ID:             backupID,
		EnvironmentID:  environment.ID,
		BackupScope:    input.BackupScope,
		StorageType:    defaultStorageType,
		StoragePath:    storagePath,
		RetentionUntil: retention,
		CreatedAt:      now,
	}); err != nil {
		return "", fmt.Errorf("create backup record: %w", err)
	}

	return backupID, nil
}

func isValidScope(scope string) bool {
	return scope == "db" || scope == "files" || scope == "full"
}

func generateUUIDv4() (string, error) {
	buf := make([]byte, 16)
	if _, err := rand.Read(buf); err != nil {
		return "", err
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
