package admin

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"

	"pressluft/internal/auth"
	"pressluft/internal/store"
)

func TestInitCreatesFirstAdminAndIsIdempotent(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	svc := NewService(db)

	res, err := svc.Init(context.Background(), InitOptions{
		Email:       "admin@local",
		DisplayName: "Admin",
		Password:    "0000",
	})
	if err != nil {
		t.Fatalf("init admin: %v", err)
	}
	if !res.Created {
		t.Fatalf("expected created")
	}
	if res.GeneratedPassword != "" {
		t.Fatalf("expected no generated password when provided")
	}

	_, err = svc.Init(context.Background(), InitOptions{Email: "admin@local", DisplayName: "Admin", Password: "0000"})
	if !errors.Is(err, ErrAlreadyInitialized) {
		t.Fatalf("expected ErrAlreadyInitialized, got %v", err)
	}

	assertCount(t, db, "SELECT COUNT(1) FROM users", 1)
}

func TestInitGeneratedPasswordAllowsLogin(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	adminSvc := NewService(db)
	authSvc := auth.NewService(db)

	res, err := adminSvc.Init(context.Background(), InitOptions{Email: "admin@local", DisplayName: "Admin"})
	if err != nil {
		t.Fatalf("init admin: %v", err)
	}
	if res.GeneratedPassword == "" {
		t.Fatalf("expected generated password")
	}

	if _, err := authSvc.Login(context.Background(), "admin@local", res.GeneratedPassword); err != nil {
		t.Fatalf("login with generated password: %v", err)
	}
}

func TestSetPasswordUpdatesHashAndLogin(t *testing.T) {
	t.Parallel()

	db := openTestDB(t)
	adminSvc := NewService(db)
	authSvc := auth.NewService(db)

	if _, err := adminSvc.Init(context.Background(), InitOptions{Email: "admin@local", DisplayName: "Admin", Password: "old"}); err != nil {
		t.Fatalf("init admin: %v", err)
	}

	if _, err := authSvc.Login(context.Background(), "admin@local", "old"); err != nil {
		t.Fatalf("login old: %v", err)
	}

	if err := adminSvc.SetPassword(context.Background(), "admin@local", "new"); err != nil {
		t.Fatalf("set password: %v", err)
	}

	if _, err := authSvc.Login(context.Background(), "admin@local", "old"); !errors.Is(err, auth.ErrInvalidCredentials) {
		t.Fatalf("expected invalid credentials for old password, got %v", err)
	}
	if _, err := authSvc.Login(context.Background(), "admin@local", "new"); err != nil {
		t.Fatalf("login new: %v", err)
	}
}

func openTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "admin-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite: %v", err)
	}
	t.Cleanup(func() {
		_ = db.Close()
	})

	if _, err := db.Exec(`
		CREATE TABLE IF NOT EXISTS users (
			id TEXT PRIMARY KEY,
			email TEXT NOT NULL UNIQUE,
			display_name TEXT NOT NULL,
			role TEXT NOT NULL,
			password_hash TEXT NOT NULL,
			is_active INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE IF NOT EXISTS auth_sessions (
			id TEXT PRIMARY KEY,
			user_id TEXT NOT NULL,
			session_token TEXT NOT NULL UNIQUE,
			expires_at TEXT NOT NULL,
			created_at TEXT NOT NULL,
			revoked_at TEXT NULL
		);
	`); err != nil {
		t.Fatalf("create schema: %v", err)
	}

	return db
}

func assertCount(t *testing.T, db *sql.DB, query string, expected int) {
	t.Helper()
	var got int
	if err := db.QueryRow(query).Scan(&got); err != nil {
		t.Fatalf("query count: %v", err)
	}
	if got != expected {
		t.Fatalf("expected %d, got %d", expected, got)
	}
}
