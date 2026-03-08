package auth

import (
	"context"
	"database/sql"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	_ "modernc.org/sqlite"
)

func TestAuthenticateRequestCapsIdleRefreshAtAbsoluteExpiry(t *testing.T) {
	service, store, db, user := newSessionServiceTestHarness(t, time.Hour, 2*time.Minute)

	token := "session-token"
	hash := HashOpaqueToken(service.sessionSecret, token)
	now := time.Now().UTC()
	absoluteExpiresAt := now.Add(2 * time.Minute)
	if err := store.CreateSession(context.Background(), user.ID, hash, now.Add(30*time.Second), absoluteExpiresAt, "test-agent", "192.0.2.10"); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})

	actor, err := service.AuthenticateRequest(req)
	if err != nil {
		t.Fatalf("AuthenticateRequest() error = %v", err)
	}
	if actor.ID != "1" {
		t.Fatalf("actor.ID = %q, want %q", actor.ID, "1")
	}

	var expiresAt string
	if err := db.QueryRow(`SELECT expires_at FROM sessions WHERE session_hash = ?`, hash).Scan(&expiresAt); err != nil {
		t.Fatalf("query expires_at: %v", err)
	}
	if got := expiresAt; got != absoluteExpiresAt.Format(time.RFC3339) {
		t.Fatalf("expires_at = %q, want capped absolute expiry %q", got, absoluteExpiresAt.Format(time.RFC3339))
	}
}

func TestAuthenticateRequestRejectsAbsoluteExpiredSession(t *testing.T) {
	service, store, _, user := newSessionServiceTestHarness(t, time.Hour, 2*time.Hour)

	token := "expired-session"
	hash := HashOpaqueToken(service.sessionSecret, token)
	now := time.Now().UTC()
	if err := store.CreateSession(context.Background(), user.ID, hash, now.Add(time.Hour), now.Add(-time.Minute), "test-agent", "192.0.2.10"); err != nil {
		t.Fatalf("CreateSession() error = %v", err)
	}

	req := httptest.NewRequest(http.MethodGet, "/api/auth/me", nil)
	req.AddCookie(&http.Cookie{Name: SessionCookieName, Value: token})

	_, err := service.AuthenticateRequest(req)
	if !errors.Is(err, ErrUnauthenticated) {
		t.Fatalf("AuthenticateRequest() error = %v, want %v", err, ErrUnauthenticated)
	}
}

func newSessionServiceTestHarness(t *testing.T, idleTimeout, absoluteTimeout time.Duration) (*Service, *Store, *sql.DB, *User) {
	t.Helper()

	db, err := sql.Open("sqlite", "file::memory:?cache=shared")
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	for _, statement := range []string{
		`CREATE TABLE users (id INTEGER PRIMARY KEY AUTOINCREMENT, email TEXT NOT NULL UNIQUE, password_hash TEXT NOT NULL, role TEXT NOT NULL, status TEXT NOT NULL DEFAULT 'active', created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')), updated_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')), last_login_at TEXT)`,
		`CREATE TABLE sessions (id INTEGER PRIMARY KEY AUTOINCREMENT, user_id INTEGER NOT NULL REFERENCES users(id) ON DELETE CASCADE, session_hash TEXT NOT NULL UNIQUE, created_at TEXT NOT NULL DEFAULT (strftime('%Y-%m-%dT%H:%M:%SZ', 'now')), expires_at TEXT NOT NULL, absolute_expires_at TEXT, revoked_at TEXT, last_used_at TEXT, user_agent TEXT, ip TEXT)`,
	} {
		if _, err := db.Exec(statement); err != nil {
			t.Fatalf("exec %q: %v", statement, err)
		}
	}

	store := NewStore(db)
	user, err := store.CreateUser(context.Background(), "admin@example.test", "correct horse battery staple", RoleAdmin)
	if err != nil {
		t.Fatalf("CreateUser() error = %v", err)
	}

	service := NewService(store, []byte("test-session-secret"), idleTimeout, absoluteTimeout, true)
	return service, store, db, user
}
