package audit

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"fmt"
	"time"
)

type Entry struct {
	UserID       string
	Action       string
	ResourceType string
	ResourceID   string
	Result       string
}

type Service struct {
	db *sql.DB
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Record(ctx context.Context, entry Entry) error {
	id, err := randomID("audit")
	if err != nil {
		return err
	}

	if _, err := s.db.ExecContext(ctx, `
		INSERT INTO audit_logs (id, user_id, action, resource_type, resource_id, result, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)
	`, id, entry.UserID, entry.Action, entry.ResourceType, entry.ResourceID, entry.Result, time.Now().UTC().Format(time.RFC3339)); err != nil {
		return fmt.Errorf("insert audit log: %w", err)
	}

	return nil
}

func randomID(prefix string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf)), nil
}
