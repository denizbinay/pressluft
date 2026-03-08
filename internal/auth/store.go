package auth

import (
	"context"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"database/sql"
	"encoding/base64"
	"errors"
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	"golang.org/x/crypto/bcrypt"
)

const (
	DefaultSessionIdleTimeout     = 12 * time.Hour
	DefaultSessionAbsoluteTimeout = 7 * 24 * time.Hour
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrUnauthenticated    = errors.New("unauthenticated")
)

type User struct {
	ID           int64
	Email        string
	PasswordHash string
	Role         Role
	Status       string
	CreatedAt    string
	UpdatedAt    string
	LastLoginAt  string
}

type Session struct {
	ID                int64
	UserID            int64
	SessionID         string
	CreatedAt         string
	ExpiresAt         string
	AbsoluteExpiresAt string
	RevokedAt         string
	LastUsedAt        string
	UserAgent         string
	IP                string
}

type Store struct {
	db *sql.DB
}

func NewStore(db *sql.DB) *Store {
	return &Store{db: db}
}

func (s *Store) UserCount(ctx context.Context) (int64, error) {
	var count int64
	if err := s.db.QueryRowContext(ctx, `SELECT COUNT(*) FROM users`).Scan(&count); err != nil {
		return 0, fmt.Errorf("count users: %w", err)
	}
	return count, nil
}

func (s *Store) CreateUser(ctx context.Context, email, password string, role Role) (*User, error) {
	email = strings.TrimSpace(strings.ToLower(email))
	if email == "" || strings.TrimSpace(password) == "" {
		return nil, fmt.Errorf("email and password are required")
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return nil, fmt.Errorf("hash password: %w", err)
	}

	now := time.Now().UTC().Format(time.RFC3339)
	res, err := s.db.ExecContext(ctx, `
		INSERT INTO users (email, password_hash, role, status, created_at, updated_at)
		VALUES (?, ?, ?, 'active', ?, ?)
	`, email, string(hash), string(role), now, now)
	if err != nil {
		return nil, fmt.Errorf("insert user: %w", err)
	}
	id, err := res.LastInsertId()
	if err != nil {
		return nil, fmt.Errorf("user insert id: %w", err)
	}
	return s.GetUserByID(ctx, id)
}

func (s *Store) GetUserByID(ctx context.Context, id int64) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, status, created_at, updated_at, COALESCE(last_login_at, '')
		FROM users WHERE id = ?
	`, id)
	var user User
	var role string
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &role, &user.Status, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by id: %w", err)
	}
	user.Role = Role(role)
	return &user, nil
}

func (s *Store) GetUserByEmail(ctx context.Context, email string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT id, email, password_hash, role, status, created_at, updated_at, COALESCE(last_login_at, '')
		FROM users WHERE email = ?
	`, strings.TrimSpace(strings.ToLower(email)))
	var user User
	var role string
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &role, &user.Status, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrInvalidCredentials
		}
		return nil, fmt.Errorf("get user by email: %w", err)
	}
	user.Role = Role(role)
	return &user, nil
}

func (s *Store) UpdateLastLogin(ctx context.Context, userID int64) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE users
		SET last_login_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
		    updated_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE id = ?
	`, userID)
	if err != nil {
		return fmt.Errorf("update last login: %w", err)
	}
	return nil
}

func (s *Store) CreateSession(ctx context.Context, userID int64, tokenHash string, expiresAt, absoluteExpiresAt time.Time, userAgent, ip string) error {
	_, err := s.db.ExecContext(ctx, `
		INSERT INTO sessions (user_id, session_hash, created_at, expires_at, absolute_expires_at, last_used_at, user_agent, ip)
		VALUES (?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), ?, ?, strftime('%Y-%m-%dT%H:%M:%SZ', 'now'), ?, ?)
	`, userID, tokenHash, expiresAt.UTC().Format(time.RFC3339), absoluteExpiresAt.UTC().Format(time.RFC3339), nullableString(userAgent), nullableString(ip))
	if err != nil {
		return fmt.Errorf("insert session: %w", err)
	}
	return nil
}

func (s *Store) GetSessionUserByHash(ctx context.Context, tokenHash string) (*User, error) {
	row := s.db.QueryRowContext(ctx, `
		SELECT u.id, u.email, u.password_hash, u.role, u.status, u.created_at, u.updated_at, COALESCE(u.last_login_at, '')
		FROM sessions s
		JOIN users u ON u.id = s.user_id
		WHERE s.session_hash = ?
		  AND s.revoked_at IS NULL
		  AND datetime(s.expires_at) > datetime('now')
		  AND datetime(COALESCE(s.absolute_expires_at, s.expires_at)) > datetime('now')
		  AND u.status = 'active'
		LIMIT 1
	`, tokenHash)
	var user User
	var role string
	if err := row.Scan(&user.ID, &user.Email, &user.PasswordHash, &role, &user.Status, &user.CreatedAt, &user.UpdatedAt, &user.LastLoginAt); err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, ErrUnauthenticated
		}
		return nil, fmt.Errorf("lookup session user: %w", err)
	}
	user.Role = Role(role)
	return &user, nil
}

func (s *Store) TouchSession(ctx context.Context, tokenHash string, expiresAt time.Time) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET last_used_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now'),
		    expires_at = CASE
		        WHEN datetime(?) < datetime(COALESCE(absolute_expires_at, expires_at)) THEN ?
		        ELSE COALESCE(absolute_expires_at, expires_at)
		    END
		WHERE session_hash = ?
		  AND revoked_at IS NULL
		  AND datetime(expires_at) > datetime('now')
		  AND datetime(COALESCE(absolute_expires_at, expires_at)) > datetime('now')
	`, expiresAt.UTC().Format(time.RFC3339), expiresAt.UTC().Format(time.RFC3339), tokenHash)
	if err != nil {
		return fmt.Errorf("touch session: %w", err)
	}
	return nil
}

func (s *Store) RevokeSession(ctx context.Context, tokenHash string) error {
	_, err := s.db.ExecContext(ctx, `
		UPDATE sessions
		SET revoked_at = strftime('%Y-%m-%dT%H:%M:%SZ', 'now')
		WHERE session_hash = ?
		  AND revoked_at IS NULL
	`, tokenHash)
	if err != nil {
		return fmt.Errorf("revoke session: %w", err)
	}
	return nil
}

func VerifyPassword(hash, password string) error {
	if err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password)); err != nil {
		return ErrInvalidCredentials
	}
	return nil
}

func GenerateOpaqueToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", fmt.Errorf("generate random token: %w", err)
	}
	return base64.RawURLEncoding.EncodeToString(buf), nil
}

func HashOpaqueToken(secret []byte, token string) string {
	mac := hmac.New(sha256.New, secret)
	_, _ = mac.Write([]byte(token))
	return base64.RawURLEncoding.EncodeToString(mac.Sum(nil))
}

func nullableString(s string) any {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return s
}

func extractRequestIP(r *http.Request) string {
	if r == nil {
		return ""
	}
	host, _, err := net.SplitHostPort(strings.TrimSpace(r.RemoteAddr))
	if err == nil {
		return host
	}
	return strings.TrimSpace(r.RemoteAddr)
}
