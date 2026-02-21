package admin

import (
	"context"
	"crypto/rand"
	"database/sql"
	"encoding/hex"
	"errors"
	"fmt"
	"strings"
	"time"

	"pressluft/internal/auth"
	"pressluft/internal/store"
)

var ErrAlreadyInitialized = errors.New("admin already initialized")
var ErrInvalidInput = errors.New("invalid input")

type Service struct {
	db *sql.DB
}

type InitOptions struct {
	Email       string
	DisplayName string
	Password    string
}

type InitResult struct {
	Created           bool
	UserID            string
	Email             string
	DisplayName       string
	GeneratedPassword string
}

func NewService(db *sql.DB) *Service {
	return &Service{db: db}
}

func (s *Service) Init(ctx context.Context, opts InitOptions) (InitResult, error) {
	email := strings.ToLower(strings.TrimSpace(opts.Email))
	if email == "" {
		return InitResult{}, ErrInvalidInput
	}
	displayName := strings.TrimSpace(opts.DisplayName)
	if displayName == "" {
		return InitResult{}, ErrInvalidInput
	}

	var userCount int
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&userCount); err != nil {
		return InitResult{}, fmt.Errorf("count users: %w", err)
	}
	if userCount > 0 {
		return InitResult{Created: false}, ErrAlreadyInitialized
	}

	password := opts.Password
	generated := ""
	if strings.TrimSpace(password) == "" {
		pw, err := randomPassword(12)
		if err != nil {
			return InitResult{}, err
		}
		password = pw
		generated = pw
	}

	hash, err := auth.HashPassword(password)
	if err != nil {
		return InitResult{}, err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	userID, err := randomID("user")
	if err != nil {
		return InitResult{}, err
	}

	err = store.WithTx(ctx, s.db, func(tx *sql.Tx) error {
		var existing int
		if err := tx.QueryRowContext(ctx, `SELECT COUNT(1) FROM users`).Scan(&existing); err != nil {
			return fmt.Errorf("count users in tx: %w", err)
		}
		if existing > 0 {
			return ErrAlreadyInitialized
		}

		if _, err := tx.ExecContext(ctx, `
 			INSERT INTO users (id, email, display_name, role, password_hash, is_active, created_at, updated_at)
 			VALUES (?, ?, ?, 'admin', ?, 1, ?, ?)
 		`, userID, email, displayName, hash, now, now); err != nil {
			return fmt.Errorf("insert user: %w", err)
		}
		return nil
	})
	if err != nil {
		if errors.Is(err, ErrAlreadyInitialized) {
			return InitResult{Created: false}, ErrAlreadyInitialized
		}
		return InitResult{}, err
	}

	return InitResult{
		Created:           true,
		UserID:            userID,
		Email:             email,
		DisplayName:       displayName,
		GeneratedPassword: generated,
	}, nil
}

func (s *Service) SetPassword(ctx context.Context, email, newPassword string) error {
	email = strings.ToLower(strings.TrimSpace(email))
	if email == "" || strings.TrimSpace(newPassword) == "" {
		return ErrInvalidInput
	}

	hash, err := auth.HashPassword(newPassword)
	if err != nil {
		return err
	}

	now := time.Now().UTC().Format(time.RFC3339)
	result, err := s.db.ExecContext(ctx, `
 		UPDATE users
 		SET password_hash = ?, updated_at = ?
 		WHERE email = ?
 	`, hash, now, email)
	if err != nil {
		return fmt.Errorf("update password: %w", err)
	}

	rows, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("read updated rows: %w", err)
	}
	if rows == 0 {
		return sql.ErrNoRows
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

func randomPassword(length int) (string, error) {
	if length < 8 {
		length = 8
	}

	buf := make([]byte, length)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate password: %w", err)
	}

	// Hex is ASCII and avoids shell escaping surprises.
	return hex.EncodeToString(buf)[:length], nil
}
