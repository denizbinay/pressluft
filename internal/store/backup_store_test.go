package store

import (
	"context"
	"testing"
	"time"
)

func TestBackupStoreLifecycleTransitions(t *testing.T) {
	ctx := context.Background()
	store := NewInMemoryBackupStore()
	now := time.Date(2026, 2, 22, 0, 0, 0, 0, time.UTC)

	_, err := store.CreateBackup(ctx, CreateBackupInput{
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

	if _, err := store.MarkBackupRunning(ctx, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", now.Add(time.Minute)); err != nil {
		t.Fatalf("MarkBackupRunning() error = %v", err)
	}
	if _, err := store.MarkBackupCompleted(ctx, "aaaaaaaa-aaaa-aaaa-aaaa-aaaaaaaaaaaa", "sha256:test", 42, now.Add(2*time.Minute)); err != nil {
		t.Fatalf("MarkBackupCompleted() error = %v", err)
	}

	items, err := store.ListBackupsByEnvironmentID(ctx, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", now.AddDate(0, 0, 31))
	if err != nil {
		t.Fatalf("ListBackupsByEnvironmentID() error = %v", err)
	}
	if len(items) != 1 {
		t.Fatalf("len(items) = %d, want 1", len(items))
	}
	if items[0].Status != "expired" {
		t.Fatalf("status = %q, want expired", items[0].Status)
	}
}
