package store

import (
	"errors"
	"strconv"
	"sync"
	"time"
)

var ErrSessionNotFound = errors.New("session not found")

type AuthSession struct {
	ID           string
	UserID       string
	SessionToken string
	ExpiresAt    time.Time
	CreatedAt    time.Time
	RevokedAt    *time.Time
}

type SessionStore interface {
	Create(session AuthSession) error
	GetByToken(token string) (AuthSession, error)
	RevokeByToken(token string, revokedAt time.Time) error
}

type InMemorySessionStore struct {
	mu       sync.RWMutex
	byToken  map[string]AuthSession
	nextID   int
	idPrefix string
}

func NewInMemorySessionStore() *InMemorySessionStore {
	return &InMemorySessionStore{
		byToken:  make(map[string]AuthSession),
		nextID:   1,
		idPrefix: "session",
	}
}

func (s *InMemorySessionStore) Create(session AuthSession) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if session.ID == "" {
		session.ID = s.idPrefix + "-" + strconv.Itoa(s.nextID)
	}
	s.byToken[session.SessionToken] = session
	s.nextID++
	return nil
}

func (s *InMemorySessionStore) GetByToken(token string) (AuthSession, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	session, ok := s.byToken[token]
	if !ok {
		return AuthSession{}, ErrSessionNotFound
	}

	return session, nil
}

func (s *InMemorySessionStore) RevokeByToken(token string, revokedAt time.Time) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	session, ok := s.byToken[token]
	if !ok {
		return ErrSessionNotFound
	}

	session.RevokedAt = &revokedAt
	s.byToken[token] = session
	return nil
}
