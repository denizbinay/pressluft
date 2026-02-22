package backups

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pressluft/internal/jobs"
	"pressluft/internal/store"
)

func TestHandlerTransitionsBackupToCompleted(t *testing.T) {
	ctx := context.Background()
	backupStore := store.NewInMemoryBackupStore()
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)

	_, err := backupStore.CreateBackup(ctx, store.CreateBackupInput{
		ID:             "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa",
		EnvironmentID:  "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		BackupScope:    "full",
		StorageType:    "s3",
		StoragePath:    "s3://pressluft/backups/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb/aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa.tar.zst",
		RetentionUntil: now.AddDate(0, 0, 30),
		CreatedAt:      now,
	})
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	handler := NewHandler(backupStore, stubExecutor{})
	handler.now = func() time.Time { return now.Add(time.Minute) }

	environmentID := "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb"
	err = handler.Handle(ctx, jobs.Job{ID: "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", JobType: "backup_create", EnvironmentID: &environmentID})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	backup, err := backupStore.GetBackupByID(ctx, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa")
	if err != nil {
		t.Fatalf("GetBackupByID() error = %v", err)
	}
	if backup.Status != "completed" {
		t.Fatalf("status = %q, want completed", backup.Status)
	}
	if backup.Checksum == nil || *backup.Checksum == "" {
		t.Fatal("checksum is empty")
	}
	if backup.SizeBytes == nil {
		t.Fatal("size_bytes is nil")
	}
}

func TestHandlerTransitionsBackupToFailedAndReturnsExecutionError(t *testing.T) {
	ctx := context.Background()
	backupStore := store.NewInMemoryBackupStore()
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)

	_, err := backupStore.CreateBackup(ctx, store.CreateBackupInput{
		ID:             "cccccccc-cccc-cccc-cccc-cccccccccccc",
		EnvironmentID:  "dddddddd-dddd-dddd-dddd-dddddddddddd",
		BackupScope:    "db",
		StorageType:    "s3",
		StoragePath:    "s3://pressluft/backups/dddddddd-dddd-dddd-dddd-dddddddddddd/cccccccc-cccc-cccc-cccc-cccccccccccc.tar.zst",
		RetentionUntil: now.AddDate(0, 0, 30),
		CreatedAt:      now,
	})
	if err != nil {
		t.Fatalf("CreateBackup() error = %v", err)
	}

	handler := NewHandler(backupStore, stubExecutor{err: jobs.ExecutionError{Code: "ANSIBLE_HOST_UNREACHABLE", Message: "timeout", Retryable: true}})
	handler.now = func() time.Time { return now.Add(time.Minute) }

	environmentID := "dddddddd-dddd-dddd-dddd-dddddddddddd"
	err = handler.Handle(ctx, jobs.Job{ID: "cccccccc-cccc-cccc-cccc-cccccccccccc", JobType: "backup_create", EnvironmentID: &environmentID})
	if err == nil {
		t.Fatal("Handle() error = nil, want non-nil")
	}

	var execErr jobs.ExecutionError
	if !errors.As(err, &execErr) {
		t.Fatalf("error type = %T, want jobs.ExecutionError", err)
	}
	if execErr.Code != "ANSIBLE_HOST_UNREACHABLE" {
		t.Fatalf("code = %s, want ANSIBLE_HOST_UNREACHABLE", execErr.Code)
	}
	if execErr.Retryable {
		t.Fatal("retryable = true, want false")
	}

	backup, err := backupStore.GetBackupByID(ctx, "cccccccc-cccc-cccc-cccc-cccccccccccc")
	if err != nil {
		t.Fatalf("GetBackupByID() error = %v", err)
	}
	if backup.Status != "failed" {
		t.Fatalf("status = %q, want failed", backup.Status)
	}
}

type stubExecutor struct {
	err error
}

func (s stubExecutor) RunBackupCreate(_ context.Context, input ExecutionInput) error {
	if s.err != nil {
		return s.err
	}
	if err := os.MkdirAll(filepath.Dir(input.ArtifactPath), 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(input.ArtifactPath, []byte("backup artifact payload"), 0o600); err != nil {
		return err
	}
	return s.err
}
