package registration

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/idutil"
)

var (
	ErrInvalidToken  = errors.New("invalid token")
	ErrExpiredToken  = errors.New("expired token")
	ErrConsumedToken = errors.New("token already consumed")
)

type Store struct {
	db *sql.DB
}

type Token struct {
	ID         string
	ServerID   string
	TokenHash  string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	ConsumedAt *time.Time
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) Create(serverID string, expiresIn time.Duration) (string, error) {
	serverID, err := lookupServerID(s.db, serverID)
	if err != nil {
		return "", err
	}
	plaintext, err := GenerateToken()
	if err != nil {
		return "", err
	}

	hash := HashToken(plaintext)
	expiresAt := time.Now().Add(expiresIn)

	tokenID, err := idutil.New()
	if err != nil {
		return "", err
	}
	_, err = s.db.Exec(`
		INSERT INTO registration_tokens (id, server_id, token_hash, expires_at)
		VALUES (?, ?, ?, ?)
	`, tokenID, serverID, hash, expiresAt.Format(time.RFC3339))

	if err != nil {
		return "", fmt.Errorf("insert token: %w", err)
	}

	return plaintext, nil
}

func (s *Store) Consume(plaintext string, serverID string) error {
	return s.consume(context.Background(), s.db, plaintext, serverID)
}

func (s *Store) Validate(plaintext string, serverID string) error {
	return s.validate(context.Background(), s.db, plaintext, serverID)
}

func (s *Store) ConsumeTx(ctx context.Context, tx *sql.Tx, plaintext string, serverID string) error {
	return s.consume(ctx, tx, plaintext, serverID)
}

type queryExecutor interface {
	ExecContext(ctx context.Context, query string, args ...any) (sql.Result, error)
	QueryRowContext(ctx context.Context, query string, args ...any) *sql.Row
}

func (s *Store) validate(ctx context.Context, db queryExecutor, plaintext string, serverID string) error {
	serverID, err := lookupServerID(db, serverID)
	if err != nil {
		return err
	}
	hash := HashToken(plaintext)

	var expiresAt string
	var consumedAt sql.NullString
	err = db.QueryRowContext(ctx, `
		SELECT expires_at, consumed_at
		FROM registration_tokens
		WHERE token_hash = ?
		  AND server_id = ?
		ORDER BY id DESC
		LIMIT 1
	`, hash, serverID).Scan(&expiresAt, &consumedAt)
	if errors.Is(err, sql.ErrNoRows) {
		return ErrInvalidToken
	}
	if err != nil {
		return fmt.Errorf("lookup token: %w", err)
	}
	if consumedAt.Valid {
		return ErrConsumedToken
	}
	expires, err := time.Parse(time.RFC3339, expiresAt)
	if err != nil {
		return fmt.Errorf("parse token expiry: %w", err)
	}
	if !expires.After(time.Now().UTC()) {
		return ErrExpiredToken
	}
	return nil
}

func (s *Store) consume(ctx context.Context, db queryExecutor, plaintext string, serverID string) error {
	serverID, err := lookupServerID(db, serverID)
	if err != nil {
		return err
	}
	if err := s.validate(ctx, db, plaintext, serverID); err != nil {
		return err
	}

	hash := HashToken(plaintext)

	result, err := db.ExecContext(ctx, `
		UPDATE registration_tokens
		SET consumed_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE token_hash = ?
		  AND server_id = ?
		  AND consumed_at IS NULL
		  AND datetime(expires_at) > datetime('now')
	`, hash, serverID)
	if err != nil {
		return fmt.Errorf("consume token: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("get rows affected: %w", err)
	}

	if rows != 1 {
		if err := s.validate(ctx, db, plaintext, serverID); err != nil {
			return err
		}
		return ErrConsumedToken
	}

	return nil
}

func lookupServerID(db queryExecutor, serverID string) (string, error) {
	serverID = strings.TrimSpace(serverID)
	if serverID == "" {
		return "", fmt.Errorf("server_id is required")
	}
	var storedID string
	if err := db.QueryRowContext(context.Background(), `SELECT id FROM servers WHERE id = ?`, serverID).Scan(&storedID); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", fmt.Errorf("server %s not found", serverID)
		}
		return "", fmt.Errorf("lookup server id: %w", err)
	}
	return storedID, nil
}

func (s *Store) CleanupExpired() (int64, error) {
	result, err := s.db.Exec(`
		DELETE FROM registration_tokens
		WHERE datetime(expires_at) < datetime('now')
		  AND consumed_at IS NULL
	`)
	if err != nil {
		return 0, fmt.Errorf("cleanup expired tokens: %w", err)
	}

	return result.RowsAffected()
}
