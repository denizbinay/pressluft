package environments

import (
	"context"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

const restoreJobType = "env_restore"

var ErrInvalidRestoreRequest = errors.New("invalid restore request")

type RestoreService struct {
	sites          store.SiteStore
	backups        store.BackupStore
	restoreStore   store.RestoreRequestStore
	queue          JobQueue
	now            func() time.Time
	restoreJobType string
}

func NewRestoreService(siteStore store.SiteStore, backupStore store.BackupStore, restoreStore store.RestoreRequestStore, queue JobQueue) *RestoreService {
	return &RestoreService{
		sites:          siteStore,
		backups:        backupStore,
		restoreStore:   restoreStore,
		queue:          queue,
		now:            func() time.Time { return time.Now().UTC() },
		restoreJobType: restoreJobType,
	}
}

func (s *RestoreService) Create(ctx context.Context, environmentID string, backupID string) (string, error) {
	environmentID = strings.TrimSpace(environmentID)
	backupID = strings.TrimSpace(backupID)
	if environmentID == "" || backupID == "" {
		return "", ErrInvalidRestoreRequest
	}

	environment, err := s.sites.GetEnvironmentByID(ctx, environmentID)
	if err != nil {
		return "", fmt.Errorf("get environment by id: %w", err)
	}

	backup, err := s.backups.GetBackupByID(ctx, backupID)
	if err != nil {
		return "", fmt.Errorf("get backup by id: %w", err)
	}
	if backup.EnvironmentID != environmentID {
		return "", ErrInvalidRestoreRequest
	}
	if backup.Status != "completed" || backup.Checksum == nil || backup.SizeBytes == nil || *backup.SizeBytes <= 0 {
		return "", ErrInvalidRestoreRequest
	}

	now := s.now()
	if _, _, err := s.sites.MarkEnvironmentRestoring(ctx, environmentID, now); err != nil {
		return "", fmt.Errorf("mark environment restoring: %w", err)
	}

	job, err := s.queue.Enqueue(ctx, jobs.EnqueueInput{
		JobType:       s.restoreJobType,
		SiteID:        &environment.SiteID,
		EnvironmentID: &environment.ID,
		NodeID:        &environment.NodeID,
		MaxAttempts:   1,
		CreatedAt:     now,
	})
	if err != nil {
		_, _, _ = s.sites.MarkEnvironmentRestoreResult(ctx, environmentID, true, s.now())
		if errors.Is(err, jobs.ErrConflict) {
			return "", ErrMutationConflict
		}
		return "", fmt.Errorf("enqueue restore job: %w", err)
	}

	if err := s.restoreStore.SaveRestoreRequest(ctx, store.RestoreRequest{
		JobID:         job.ID,
		EnvironmentID: environmentID,
		BackupID:      backupID,
		CreatedAt:     now,
	}); err != nil {
		_, _, _ = s.sites.MarkEnvironmentRestoreResult(ctx, environmentID, true, s.now())
		return "", fmt.Errorf("save restore request: %w", err)
	}

	return job.ID, nil
}
