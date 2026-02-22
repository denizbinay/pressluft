package environments

import (
	"context"
	"errors"
	"log"
	"os"
	"path/filepath"
	"testing"
	"time"

	"pressluft/internal/backups"
	"pressluft/internal/jobs"
	"pressluft/internal/nodes"
	"pressluft/internal/store"
)

type mockEnvRestoreExecutor struct {
	err error
}

func (m *mockEnvRestoreExecutor) RunEnvRestore(context.Context, nodes.Node, EnvRestoreVars) error {
	return m.err
}

func TestEnvRestoreHandlerHandleSuccess(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	logger := log.New(os.Stderr, "test: ", 0)
	siteStore := store.NewInMemorySiteStore(0)
	nodeStore := nodes.NewInMemoryStore(nil)
	backupStore := store.NewInMemoryBackupStore()
	restoreStore := store.NewInMemoryRestoreRequestStore()

	selfNode, _ := nodeStore.EnsureSelfNode(ctx, now)
	site, environment, _ := siteStore.CreateSiteWithProductionEnvironment(ctx, store.CreateSiteInput{
		Name:       "Acme",
		Slug:       "acme",
		NodeID:     selfNode.ID,
		NodePublic: selfNode.PublicIP,
		Now:        now,
	})
	_, _, _ = siteStore.MarkEnvironmentRestoring(ctx, environment.ID, now)

	_, _ = backupStore.CreateBackup(ctx, store.CreateBackupInput{
		ID:             "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb",
		EnvironmentID:  environment.ID,
		BackupScope:    "full",
		StorageType:    "s3",
		StoragePath:    "s3://pressluft/backups/" + environment.ID + "/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.tar.zst",
		RetentionUntil: now.AddDate(0, 0, 30),
		CreatedAt:      now,
	})
	_, _ = backupStore.MarkBackupRunning(ctx, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", now)
	artifactPath := backups.LocalArtifactPath("s3://pressluft/backups/" + environment.ID + "/bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb.tar.zst")
	_ = os.MkdirAll(filepath.Dir(artifactPath), 0o755)
	_ = os.WriteFile(artifactPath, []byte("backup payload"), 0o600)
	checksum, size, _ := backups.ChecksumAndSize(artifactPath)
	_, _ = backupStore.MarkBackupCompleted(ctx, "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", checksum, size, now)

	_ = restoreStore.SaveRestoreRequest(ctx, store.RestoreRequest{JobID: "job-restore", EnvironmentID: environment.ID, BackupID: "bbbbbbbb-bbbb-bbbb-bbbb-bbbbbbbbbbbb", CreatedAt: now})

	handler := NewEnvRestoreHandler(siteStore, nodeStore, backupStore, restoreStore, &mockEnvRestoreExecutor{}, logger)
	handler.now = func() time.Time { return now.Add(time.Minute) }

	err := handler.Handle(ctx, jobs.Job{ID: "job-restore", JobType: "env_restore", SiteID: &site.ID, EnvironmentID: &environment.ID, NodeID: &selfNode.ID})
	if err != nil {
		t.Fatalf("Handle() error = %v", err)
	}

	updatedEnv, _ := siteStore.GetEnvironmentByID(ctx, environment.ID)
	if updatedEnv.Status != "active" {
		t.Fatalf("environment status = %q, want active", updatedEnv.Status)
	}
}

func TestEnvRestoreHandlerHandleFailureMarksFailed(t *testing.T) {
	ctx := context.Background()
	now := time.Date(2026, 2, 22, 12, 0, 0, 0, time.UTC)
	logger := log.New(os.Stderr, "test: ", 0)
	siteStore := store.NewInMemorySiteStore(0)
	nodeStore := nodes.NewInMemoryStore(nil)
	backupStore := store.NewInMemoryBackupStore()
	restoreStore := store.NewInMemoryRestoreRequestStore()

	selfNode, _ := nodeStore.EnsureSelfNode(ctx, now)
	site, environment, _ := siteStore.CreateSiteWithProductionEnvironment(ctx, store.CreateSiteInput{
		Name:       "Acme",
		Slug:       "acme",
		NodeID:     selfNode.ID,
		NodePublic: selfNode.PublicIP,
		Now:        now,
	})
	_, _, _ = siteStore.MarkEnvironmentRestoring(ctx, environment.ID, now)

	_, _ = backupStore.CreateBackup(ctx, store.CreateBackupInput{
		ID:             "cccccccc-cccc-cccc-cccc-cccccccccccc",
		EnvironmentID:  environment.ID,
		BackupScope:    "full",
		StorageType:    "s3",
		StoragePath:    "s3://pressluft/backups/" + environment.ID + "/cccccccc-cccc-cccc-cccc-cccccccccccc.tar.zst",
		RetentionUntil: now.AddDate(0, 0, 30),
		CreatedAt:      now,
	})
	_, _ = backupStore.MarkBackupRunning(ctx, "cccccccc-cccc-cccc-cccc-cccccccccccc", now)
	artifactPath := backups.LocalArtifactPath("s3://pressluft/backups/" + environment.ID + "/cccccccc-cccc-cccc-cccc-cccccccccccc.tar.zst")
	_ = os.MkdirAll(filepath.Dir(artifactPath), 0o755)
	_ = os.WriteFile(artifactPath, []byte("backup payload"), 0o600)
	checksum, size, _ := backups.ChecksumAndSize(artifactPath)
	_, _ = backupStore.MarkBackupCompleted(ctx, "cccccccc-cccc-cccc-cccc-cccccccccccc", checksum, size, now)

	_ = restoreStore.SaveRestoreRequest(ctx, store.RestoreRequest{JobID: "job-restore-fail", EnvironmentID: environment.ID, BackupID: "cccccccc-cccc-cccc-cccc-cccccccccccc", CreatedAt: now})

	handler := NewEnvRestoreHandler(siteStore, nodeStore, backupStore, restoreStore, &mockEnvRestoreExecutor{err: jobs.ExecutionError{Code: "ANSIBLE_PLAY_ERROR", Message: "restore failed", Retryable: true}}, logger)

	err := handler.Handle(ctx, jobs.Job{ID: "job-restore-fail", JobType: "env_restore", SiteID: &site.ID, EnvironmentID: &environment.ID, NodeID: &selfNode.ID})
	if err == nil {
		t.Fatal("Handle() error = nil, want error")
	}
	var execErr jobs.ExecutionError
	if !errors.As(err, &execErr) {
		t.Fatalf("error type = %T, want ExecutionError", err)
	}
	if execErr.Retryable {
		t.Fatal("retryable = true, want false")
	}

	updatedEnv, _ := siteStore.GetEnvironmentByID(ctx, environment.ID)
	if updatedEnv.Status != "failed" {
		t.Fatalf("environment status = %q, want failed", updatedEnv.Status)
	}
}
