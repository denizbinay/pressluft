package store

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrBackupNotFound      = errors.New("backup not found")
	ErrBackupConflict      = errors.New("backup conflict")
	ErrInvalidBackupScope  = errors.New("invalid backup scope")
	ErrInvalidBackupStatus = errors.New("invalid backup status transition")
)

type Backup struct {
	ID             string     `json:"id"`
	EnvironmentID  string     `json:"environment_id"`
	BackupScope    string     `json:"backup_scope"`
	Status         string     `json:"status"`
	StorageType    string     `json:"storage_type"`
	StoragePath    string     `json:"storage_path"`
	RetentionUntil time.Time  `json:"retention_until"`
	Checksum       *string    `json:"checksum"`
	SizeBytes      *int64     `json:"size_bytes"`
	CreatedAt      time.Time  `json:"created_at"`
	CompletedAt    *time.Time `json:"completed_at"`
}

type BackupStore interface {
	ListBackupsByEnvironmentID(ctx context.Context, environmentID string, now time.Time) ([]Backup, error)
	GetBackupByID(ctx context.Context, id string) (Backup, error)
	CreateBackup(ctx context.Context, input CreateBackupInput) (Backup, error)
	MarkBackupRunning(ctx context.Context, id string, now time.Time) (Backup, error)
	MarkBackupCompleted(ctx context.Context, id string, checksum string, sizeBytes int64, now time.Time) (Backup, error)
	MarkBackupFailed(ctx context.Context, id string, now time.Time) (Backup, error)
}

type CreateBackupInput struct {
	ID             string
	EnvironmentID  string
	BackupScope    string
	StorageType    string
	StoragePath    string
	RetentionUntil time.Time
	CreatedAt      time.Time
}

type InMemoryBackupStore struct {
	mu              sync.RWMutex
	backupsByID     map[string]Backup
	backupOrderByID []string
	envBackupIDs    map[string][]string
}

var (
	globalBackupStoreMu sync.RWMutex
	globalBackupStore   BackupStore
)

func NewInMemoryBackupStore() *InMemoryBackupStore {
	store := &InMemoryBackupStore{
		backupsByID:     make(map[string]Backup),
		backupOrderByID: make([]string, 0),
		envBackupIDs:    make(map[string][]string),
	}
	setDefaultBackupStore(store)
	return store
}

func DefaultBackupStore() BackupStore {
	globalBackupStoreMu.RLock()
	current := globalBackupStore
	globalBackupStoreMu.RUnlock()
	if current != nil {
		return current
	}
	return NewInMemoryBackupStore()
}

func setDefaultBackupStore(backupStore BackupStore) {
	globalBackupStoreMu.Lock()
	defer globalBackupStoreMu.Unlock()
	globalBackupStore = backupStore
}

func (s *InMemoryBackupStore) ListBackupsByEnvironmentID(_ context.Context, environmentID string, now time.Time) ([]Backup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.expireCompletedBackupsLocked(now)

	ids := s.envBackupIDs[environmentID]
	result := make([]Backup, 0, len(ids))
	for _, id := range ids {
		result = append(result, s.backupsByID[id])
	}

	return result, nil
}

func (s *InMemoryBackupStore) GetBackupByID(_ context.Context, id string) (Backup, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	backup, ok := s.backupsByID[id]
	if !ok {
		return Backup{}, ErrBackupNotFound
	}

	return backup, nil
}

func (s *InMemoryBackupStore) CreateBackup(_ context.Context, input CreateBackupInput) (Backup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.backupsByID[input.ID]; exists {
		return Backup{}, ErrBackupConflict
	}
	if !isValidBackupScope(input.BackupScope) {
		return Backup{}, ErrInvalidBackupScope
	}

	createdAt := input.CreatedAt.UTC()
	if createdAt.IsZero() {
		createdAt = time.Now().UTC()
	}

	backup := Backup{
		ID:             input.ID,
		EnvironmentID:  input.EnvironmentID,
		BackupScope:    input.BackupScope,
		Status:         "pending",
		StorageType:    input.StorageType,
		StoragePath:    input.StoragePath,
		RetentionUntil: input.RetentionUntil.UTC(),
		Checksum:       nil,
		SizeBytes:      nil,
		CreatedAt:      createdAt,
		CompletedAt:    nil,
	}

	s.backupsByID[backup.ID] = backup
	s.backupOrderByID = append(s.backupOrderByID, backup.ID)
	s.envBackupIDs[backup.EnvironmentID] = append(s.envBackupIDs[backup.EnvironmentID], backup.ID)

	return backup, nil
}

func (s *InMemoryBackupStore) MarkBackupRunning(_ context.Context, id string, _ time.Time) (Backup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	backup, ok := s.backupsByID[id]
	if !ok {
		return Backup{}, ErrBackupNotFound
	}
	if backup.Status != "pending" {
		return Backup{}, fmt.Errorf("mark backup running: %w", ErrInvalidBackupStatus)
	}

	backup.Status = "running"
	s.backupsByID[id] = backup

	return backup, nil
}

func (s *InMemoryBackupStore) MarkBackupCompleted(_ context.Context, id string, checksum string, sizeBytes int64, now time.Time) (Backup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	backup, ok := s.backupsByID[id]
	if !ok {
		return Backup{}, ErrBackupNotFound
	}
	if backup.Status != "running" {
		return Backup{}, fmt.Errorf("mark backup completed: %w", ErrInvalidBackupStatus)
	}

	backup.Status = "completed"
	backup.Checksum = stringPtr(checksum)
	backup.SizeBytes = int64Ptr(sizeBytes)
	completedAt := now.UTC()
	if completedAt.IsZero() {
		completedAt = time.Now().UTC()
	}
	backup.CompletedAt = &completedAt
	s.backupsByID[id] = backup

	return backup, nil
}

func (s *InMemoryBackupStore) MarkBackupFailed(_ context.Context, id string, _ time.Time) (Backup, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	backup, ok := s.backupsByID[id]
	if !ok {
		return Backup{}, ErrBackupNotFound
	}
	if backup.Status != "running" {
		return Backup{}, fmt.Errorf("mark backup failed: %w", ErrInvalidBackupStatus)
	}

	backup.Status = "failed"
	backup.Checksum = nil
	backup.SizeBytes = nil
	backup.CompletedAt = nil
	s.backupsByID[id] = backup

	return backup, nil
}

func (s *InMemoryBackupStore) expireCompletedBackupsLocked(now time.Time) {
	if now.IsZero() {
		now = time.Now().UTC()
	}
	for _, id := range s.backupOrderByID {
		backup := s.backupsByID[id]
		if backup.Status != "completed" {
			continue
		}
		if backup.RetentionUntil.After(now) {
			continue
		}
		backup.Status = "expired"
		s.backupsByID[id] = backup
	}
}

func isValidBackupScope(scope string) bool {
	return scope == "db" || scope == "files" || scope == "full"
}

func int64Ptr(v int64) *int64 {
	vv := v
	return &vv
}
