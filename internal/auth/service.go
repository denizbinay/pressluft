package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/store"
)

var ErrInvalidCredentials = errors.New("invalid credentials")
var ErrUnauthorized = errors.New("unauthorized")

type Service struct {
	db *sql.DB
}

type LoginResult struct {
	UserID       string
	SessionToken string
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Login(ctx context.Context, email, password string) (LoginResult, error) {
	var userID string
	var passwordHash string
	var active int
	err := s.db.QueryRowContext(ctx, `
		SELECT id, password_hash, is_active
		FROM users
		WHERE email = ?
		LIMIT 1
	`, strings.ToLower(strings.TrimSpace(email))).Scan(&userID, &passwordHash, &active)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, fmt.Errorf("query user: %w", err)
	}

	if active != 1 || !passwordMatches(passwordHash, password) {
		return LoginResult{}, ErrInvalidCredentials
	}

	token, err := randomToken(32)
	if err != nil {
		return LoginResult{}, err
	}

	now := time.Now().UTC()
	expiresAt := now.Add(24 * time.Hour)
	sessionID, err := randomID("session")
	if err != nil {
		return LoginResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		if _, err := tx.ExecContext(ctx, `
			INSERT INTO auth_sessions (id, user_id, session_token, expires_at, created_at, revoked_at)
			VALUES (?, ?, ?, ?, ?, NULL)
		`, sessionID, userID, token, expiresAt.Format(time.RFC3339), now.Format(time.RFC3339)); err != nil {
			return fmt.Errorf("insert auth session: %w", err)
		}
		return nil
	})
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{UserID: userID, SessionToken: token}, nil
}

func (s *Service) Logout(ctx context.Context, token string) error {
	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
		UPDATE auth_sessions
		SET revoked_at = ?
		WHERE session_token = ?
		  AND revoked_at IS NULL
	`, now, token)
	if err != nil {
		return fmt.Errorf("revoke auth session: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read revoked rows: %w", err)
	}
	if rows == 0 {
		return ErrUnauthorized
	}

	return nil
}

func (s *Service) ValidateSession(ctx context.Context, token string) (string, error) {
	var userID string
	err := s.db.QueryRowContext(ctx, `
		SELECT s.user_id
		FROM auth_sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.session_token = ?
		  AND s.revoked_at IS NULL
		  AND s.expires_at > ?
		  AND u.is_active = 1
		LIMIT 1
	`, token, time.Now().UTC().Format(time.RFC3339)).Scan(&userID)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return "", ErrUnauthorized
		}
		return "", fmt.Errorf("validate session: %w", err)
	}

	return userID, nil
}

func passwordMatches(stored, plain string) bool {
	if strings.HasPrefix(stored, "sha256:") {
		hash := sha256.Sum256([]byte(plain))
		candidate := "sha256:" + hex.EncodeToString(hash[:])
		return subtle.ConstantTimeCompare([]byte(candidate), []byte(stored)) == 1
	}

	return subtle.ConstantTimeCompare([]byte(stored), []byte(plain)) == 1
}

func randomToken(length int) (string, error) {
	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate token: %w", err)
	}
	return hex.EncodeToString(buf), nil
}

func randomID(prefix string) (string, error) {
	buf := make([]byte, 8)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate id: %w", err)
	}
	return fmt.Sprintf("%s_%s", prefix, hex.EncodeToString(buf)), nil
}
