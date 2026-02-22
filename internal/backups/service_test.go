package backups

import (
	"context"
	"errors"
	"strings"
	"testing"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

func TestCreateBackupEnqueuesJobAndCreatesPendingRecord(t *testing.T) {
	ctx := context.Background()
	siteStore, environment := seedSiteAndEnvironment(t)
	backupStore := store.NewInMemoryBackupStore()
	queue := jobs.NewInMemoryRepository(nil)

	svc := NewService(siteStore, backupStore, queue)
	jobID, err := svc.Create(ctx, CreateInput{EnvironmentID: environment.ID, BackupScope: "full"})
	if err != nil {
		t.Fatalf("Create() error = %v", err)
	}
	if jobID == "" {
		t.Fatal("jobID is empty")
	}

	queuedJob, err := queue.GetByID(ctx, jobID)
	if err != nil {
		t.Fatalf("GetByID() error = %v", err)
	}
	if queuedJob.JobType != "backup_create" {
		t.Fatalf("job type = %q, want backup_create", queuedJob.JobType)
	}

	items, err := svc.ListByEnvironmentID(ctx, environment.ID)
	if err != nil {
		t.Fatalf("ListByEnvironmentID() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(backups) = %d, want 1", len(items))
	}

	backup := items[0]
	if backup.ID != jobID {
		t.Fatalf("backup id = %q, want %q", backup.ID, jobID)
	}
	if backup.Status != "pending" {
		t.Fatalf("backup status = %q, want pending", backup.Status)
	}
	if backup.StorageType != "s3" {
		t.Fatalf("backup storage_type = %q, want s3", backup.StorageType)
	}
	if !strings.Contains(backup.StoragePath, environment.ID) {
		t.Fatalf("backup storage_path = %q, expected environment id", backup.StoragePath)
	}
	if backup.RetentionUntil.Sub(backup.CreatedAt) < (29*24*time.Hour) || backup.RetentionUntil.Sub(backup.CreatedAt) > (31*24*time.Hour) {
		t.Fatalf("backup retention delta = %s, want ~30d", backup.RetentionUntil.Sub(backup.CreatedAt))
	}
}

func TestCreateBackupRejectsInvalidScope(t *testing.T) {
	ctx := context.Background()
	siteStore, environment := seedSiteAndEnvironment(t)
	backupStore := store.NewInMemoryBackupStore()
	queue := jobs.NewInMemoryRepository(nil)

	svc := NewService(siteStore, backupStore, queue)
	_, err := svc.Create(ctx, CreateInput{EnvironmentID: environment.ID, BackupScope: "invalid"})
	if !errors.Is(err, store.ErrInvalidBackupScope) {
		t.Fatalf("Create() error = %v, want ErrInvalidBackupScope", err)
	}
}

func TestCreateBackupReturnsConflictWhenMutationJobExists(t *testing.T) {
	ctx := context.Background()
	siteStore, environment := seedSiteAndEnvironment(t)
	backupStore := store.NewInMemoryBackupStore()

	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)
	queue := jobs.NewInMemoryRepository([]jobs.Job{{
		ID:            "job-running",
		JobType:       "site_create",
		Status:        jobs.StatusRunning,
		SiteID:        &environment.SiteID,
		EnvironmentID: &environment.ID,
		NodeID:        &environment.NodeID,
		AttemptCount:  1,
		MaxAttempts:   3,
		CreatedAt:     now,
		UpdatedAt:     now,
	}})

	svc := NewService(siteStore, backupStore, queue)
	_, err := svc.Create(ctx, CreateInput{EnvironmentID: environment.ID, BackupScope: "full"})
	if !errors.Is(err, ErrMutationConflict) {
		t.Fatalf("Create() error = %v, want ErrMutationConflict", err)
	}
}

func seedSiteAndEnvironment(t *testing.T) (*store.InMemorySiteStore, store.Environment) {
	t.Helper()
	siteStore := store.NewInMemorySiteStore(0)
	_, environment, err := siteStore.CreateSiteWithProductionEnvironment(context.Background(), store.CreateSiteInput{
		Name:       "Acme Co",
		Slug:       "acme",
		NodeID:     "44444444-4444-4444-4444-444444444444",
		NodePublic: "127.0.0.1",
		Now:        time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC),
	})
	if err != nil {
		t.Fatalf("CreateSiteWithProductionEnvironment() error = %v", err)
	}

	return siteStore, environment
}
