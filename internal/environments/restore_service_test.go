package environments

import (
	"context"
	"errors"
	"testing"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

func TestRestoreServiceCreateEnqueuesRestoreJob(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	siteStore := store.NewInMemorySiteStore(0)
	backupStore := store.NewInMemoryBackupStore()
	restoreStore := store.NewInMemoryRestoreRequestStore()
	queue := jobs.NewInMemoryRepository(nil)

	site, environment, err := siteStore.CreateSiteWithProductionEnvironment(ctx, store.CreateSiteInput{
		Name:       "Acme",
		Slug:       "acme",
		NodeID:     "44444444-4444-4444-4444-444444444444",
		NodePublic: "127.0.0.1",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	_, err = backupStore.CreateBackup(ctx, store.CreateBackupInput{
		ID:             "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		EnvironmentID:  environment.ID,
		BackupScope:    "full",
		StorageType:    "s3",
		StoragePath:    "s3://pressluft/backups/" + environment.ID + "/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.tar.zst",
		RetentionUntil: now.AddDate(0, 0, 30),
		CreatedAt:      now,
	})
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}
	_, _ = backupStore.MarkBackupRunning(ctx, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", now)
	_, _ = backupStore.MarkBackupCompleted(ctx, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", "sha256:test", 128, now)

	svc := NewRestoreService(siteStore, backupStore, restoreStore, queue)
	svc.now = func() time.Time { return now }

	jobID, err := svc.Create(ctx, environment.ID, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb")
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}

	job, err := queue.GetByID(ctx, jobID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if job.JobType != "env_restore" {
		t.Fatalf("job_type = %q, want env_restore", job.JobType)
	}

	req, err := restoreStore.GetRestoreRequestByJobID(ctx, jobID)
	if err != nil {
		t.Fatalf("GetRestoreRequestByJobID() error = %v", err)
	}
	if req.BackupID != "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb" {
		t.Fatalf("backup_id = %q, want target backup", req.BackupID)
	}

	updatedEnv, err := siteStore.GetEnvironmentByID(ctx, environment.ID)
	if err != nil {
		t.Fatalf("GetEnvironmentByID() error = %v", err)
	}
	if updatedEnv.Status != "restoring" {
		t.Fatalf("environment status = %q, want restoring", updatedEnv.Status)
	}

	updatedSite, err := siteStore.GetSiteByID(ctx, site.ID)
	if err != nil {
		t.Fatalf("GetSiteByID() error = %v", err)
	}
	if updatedSite.Status != "restoring" {
		t.Fatalf("site status = %q, want restoring", updatedSite.Status)
	}
}

func TestRestoreServiceRejectsInvalidBackup(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	siteStore := store.NewInMemorySiteStore(0)
	backupStore := store.NewInMemoryBackupStore()
	restoreStore := store.NewInMemoryRestoreRequestStore()
	queue := jobs.NewInMemoryRepository(nil)

	_, environment, err := siteStore.CreateSiteWithProductionEnvironment(ctx, store.CreateSiteInput{
		Name:       "Acme",
		Slug:       "acme",
		NodeID:     "44444444-4444-4444-4444-444444444444",
		NodePublic: "127.0.0.1",
		Now:        now,
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	svc := NewRestoreService(siteStore, backupStore, restoreStore, queue)
	_, err = svc.Create(ctx, environment.ID, "missing-backup")
	if !errors.Is(err, store.ErrBackupNotFound) {
		t.Fatalf("Create() error = %v, want ErrBackupNotFound", err)
	}
}
