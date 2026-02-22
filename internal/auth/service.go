package auth

import (
	"crypto/rand"
	"crypto/subtle"
	"encoding/hex"
	"errors"
	"fmt"
	"time"

	"pressluft/internal/store"
)

const (
	CookieName    = "session_token"
	DefaultEmail  = "admin@pressluft.local"
	DefaultUserID = "admin"
)

var (
	ErrInvalidCredentials = errors.New("invalid credentials")
	ErrSessionExpired     = errors.New("session expired")
	ErrSessionRevoked     = errors.New("session revoked")
	ErrUnauthorized       = errors.New("unauthorized")
)

type Service struct {
	adminEmail    string
	adminPassword string
	adminUserID   string
	sessionTTL    time.Duration
	store         store.SessionStore
}

func NewService(sessionStore store.SessionStore, adminEmail string, adminPassword string, sessionTTL time.Duration) *Service {
	if adminEmail == "" {
		adminEmail = DefaultEmail
	}
	if adminPassword == "" {
		adminPassword = "pressluft-dev-password"
	}
	if sessionTTL <= 0 {
		sessionTTL = 24 * time.Hour
	}

	return &Service{
		adminEmail:    adminEmail,
		adminPassword: adminPassword,
		adminUserID:   DefaultUserID,
		sessionTTL:    sessionTTL,
		store:         sessionStore,
	}
}

func (s *Service) Login(email string, password string, now time.Time) (store.AuthSession, error) {
	if !secureEqual(email, s.adminEmail) || !secureEqual(password, s.adminPassword) {
		return store.AuthSession{}, ErrInvalidCredentials
	}

	token, err := generateToken()
	if err != nil {
		return store.AuthSession{}, fmt.Errorf("generate token: %w", err)
	}

	session := store.AuthSession{
		UserID:       s.adminUserID,
		SessionToken: token,
		CreatedAt:    now.UTC(),
		ExpiresAt:    now.UTC().Add(s.sessionTTL),
	}

	if err := s.store.Create(session); err != nil {
		return store.AuthSession{}, fmt.Errorf("create session: %w", err)
	}

	created, err := s.store.GetByToken(token)
	if err != nil {
		return store.AuthSession{}, fmt.Errorf("fetch session: %w", err)
	}

	return created, nil
}

func (s *Service) Validate(token string, now time.Time) (store.AuthSession, error) {
	session, err := s.store.GetByToken(token)
	if err != nil {
		if errors.Is(err, store.ErrSessionNotFound) {
			return store.AuthSession{}, ErrUnauthorized
		}
		return store.AuthSession{}, fmt.Errorf("lookup session: %w", err)
	}

	if session.RevokedAt != nil {
		return store.AuthSession{}, ErrSessionRevoked
	}

	if now.UTC().After(session.ExpiresAt) {
		return store.AuthSession{}, ErrSessionExpired
	}

	return session, nil
}

func (s *Service) Revoke(token string, now time.Time) error {
	if err := s.store.RevokeByToken(token, now.UTC()); err != nil {
		if errors.Is(err, store.ErrSessionNotFound) {
			return ErrUnauthorized
		}
		return fmt.Errorf("revoke session: %w", err)
	}

	return nil
}

func generateToken() (string, error) {
	buf := make([]byte, 32)
	if _, err := rand.Read(buf); err != nil {
		return "", err
	}
	return hex.EncodeToString(buf), nil
}

func secureEqual(left string, right string) bool {
	if len(left) != len(right) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(left), []byte(right)) == 1
}
