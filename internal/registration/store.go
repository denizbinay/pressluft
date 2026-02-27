package registration

import (
	"database/sql"
	"errors"
	"fmt"
	"time"
)

type Store struct {
	db *sql.DB
}

type Token struct {
	ID         int64
	ServerID   int64
	TokenHash  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	ConsumedAt *time.Time
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(serverID int64, expiresIn time.Duration) (string, error) {
	plaintext, err := GenerateToken()
	if err != nil {
		return "", err
	}

	hash := HashToken(plaintext)
	expiresAt := time.Now().Add(expiresIn)

	_, err = s.db.Exec(`
		INSERT INTO registration_tokens (server_id, token_hash, expires_at)
		VALUES (?, ?, ?)
	`, serverID, hash, expiresAt.Format(time.RFC3339))

	if err != nil {
		return "", fmt.Errorf("insert token: %w", err)
	}

	return plaintext, nil
}

func (s *Store) Consume(plaintext string, serverID int64) error {
	hash := HashToken(plaintext)

	result, err := s.db.Exec(`
		UPDATE registration_tokens
		SET consumed_at = datetime('now')
		WHERE token_hash = ?
		  AND server_id = ?
		  AND consumed_at IS NULL
		  AND expires_at > datetime('now')
	`, hash, serverID)
	if err != nil {
		return fmt.Errorf("consume token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows != 1 {
		return errors.New("invalid, expired, or already consumed token")
	}

	return nil
}

func (s *Store) CleanupExpired() (int64, error) {
	result, err := s.db.Exec(`
		DELETE FROM registration_tokens
		WHERE expires_at < datetime('now')
		  AND consumed_at IS NULL
	`)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired tokens: %w", err)
	}

	return result.RowsAffected()
}
