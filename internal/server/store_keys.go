package server

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"
)

// StoredServerKey is a persisted SSH key for a server.
type StoredServerKey struct {
	ServerID            int64  `json:"server_id"`
	PublicKey           string `json:"public_key"`
	PrivateKeyEncrypted string `json:"private_key_encrypted"`
	EncryptionKeyID     string `json:"encryption_key_id"`
	CreatedAt           string `json:"created_at"`
	RotatedAt           string `json:"rotated_at,omitempty"`
}

// CreateServerKeyInput is required to store a server key.
type CreateServerKeyInput struct {
	ServerID            int64
	PublicKey           string
	PrivateKeyEncrypted string
	EncryptionKeyID     string
	RotatedAt           string
}

// GetKey returns the SSH key for a server, if present.
func (s *ServerStore) GetKey(ctx context.Context, serverID int64) (*StoredServerKey, error) {
	if serverID <= 0 {
		return nil, fmt.Errorf("server_id must be greater than zero")
	}

	var (
		key       StoredServerKey
		rotatedAt sql.NullString
	)
	err := s.db.QueryRowContext(ctx,
		`SELECT server_id, public_key, private_key_encrypted, encryption_key_id, created_at, rotated_at
         FROM server_keys
         WHERE server_id = ?`,
		serverID,
	).Scan(
		&key.ServerID,
		&key.PublicKey,
		&key.PrivateKeyEncrypted,
		&key.EncryptionKeyID,
		&key.CreatedAt,
		&rotatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, nil
		}
		return nil, fmt.Errorf("get server key: %w", err)
	}
	key.RotatedAt = nullStringValue(rotatedAt)
	return &key, nil
}

// CreateKey stores a new SSH key for a server.
func (s *ServerStore) CreateKey(ctx context.Context, in CreateServerKeyInput) error {
	if err := validateCreateServerKeyInput(in); err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	rotatedAt := sql.NullString{String: strings.TrimSpace(in.RotatedAt), Valid: strings.TrimSpace(in.RotatedAt) != ""}
	_, err := s.db.ExecContext(ctx,
		`INSERT INTO server_keys (server_id, public_key, private_key_encrypted, encryption_key_id, created_at, rotated_at)
         VALUES (?, ?, ?, ?, ?, ?)`,
		in.ServerID,
		in.PublicKey,
		in.PrivateKeyEncrypted,
		in.EncryptionKeyID,
		now,
		rotatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert server key: %w", err)
	}
	return nil
}

func validateCreateServerKeyInput(in CreateServerKeyInput) error {
	if in.ServerID <= 0 {
		return fmt.Errorf("server_id must be greater than zero")
	}
	if strings.TrimSpace(in.PublicKey) == "" {
		return fmt.Errorf("public_key is required")
	}
	if strings.TrimSpace(in.PrivateKeyEncrypted) == "" {
		return fmt.Errorf("private_key_encrypted is required")
	}
	if strings.TrimSpace(in.EncryptionKeyID) == "" {
		return fmt.Errorf("encryption_key_id is required")
	}
	return nil
}
