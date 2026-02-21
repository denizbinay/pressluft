package ssh

import (
	"context"
	"database/sql"
	"errors"
	"path/filepath"
	"testing"
	"time"

	"pressluft/internal/store"
)

func TestCreateMagicLoginReturnsURLAndExpiry(t *testing.T) {
	t.Parallel()

	db := openSSHTestDB(t)
	seedMagicLoginEnvironment(t, db, "active")
	runner := &stubRunner{output: "https://example.test/wp-admin/?token=abc"}
	svc := NewService(db, runner)

	result, err := svc.CreateMagicLogin(context.Background(), "env-1")
	if err != nil {
		t.Fatalf("create magic login: %v", err)
	}
	if result.LoginURL != "https://example.test/wp-admin/?token=abc" {
		t.Fatalf("unexpected login url: %s", result.LoginURL)
	}
	if _, err := time.Parse(time.RFC3339, result.ExpiresAt); err != nil {
		t.Fatalf("expires_at is not RFC3339: %v", err)
	}

	if runner.called != 1 {
		t.Fatalf("expected one ssh call, got %d", runner.called)
	}
	if runner.host != "node.local" || runner.port != 22 || runner.user != "root" {
		t.Fatalf("unexpected ssh target host=%s port=%d user=%s", runner.host, runner.port, runner.user)
	}
}

func TestCreateMagicLoginRejectsInactiveEnvironment(t *testing.T) {
	t.Parallel()

	db := openSSHTestDB(t)
	seedMagicLoginEnvironment(t, db, "deploying")
	svc := NewService(db, &stubRunner{output: "https://example.test"})

	_, err := svc.CreateMagicLogin(context.Background(), "env-1")
	if !errors.Is(err, ErrEnvironmentNotActive) {
		t.Fatalf("expected environment not active, got %v", err)
	}
}

func TestCreateMagicLoginReturnsNodeUnreachableOnTimeout(t *testing.T) {
	t.Parallel()

	db := openSSHTestDB(t)
	seedMagicLoginEnvironment(t, db, "active")
	runner := &stubRunner{
		runFn: func(ctx context.Context, _ string, _ int, _ string, _ ...string) (string, error) {
			deadline, ok := ctx.Deadline()
			if !ok {
				t.Fatalf("expected deadline on context")
			}
			remaining := time.Until(deadline)
			if remaining > magicLoginNodeQueryTimeout || remaining < 9*time.Second {
				t.Fatalf("unexpected timeout window remaining=%s", remaining)
			}
			<-ctx.Done()
			return "", ctx.Err()
		},
	}
	svc := NewService(db, runner)

	_, err := svc.CreateMagicLogin(context.Background(), "env-1")
	if !errors.Is(err, ErrNodeUnreachable) {
		t.Fatalf("expected node unreachable, got %v", err)
	}
}

func TestCreateMagicLoginReturnsWPCliError(t *testing.T) {
	t.Parallel()

	db := openSSHTestDB(t)
	seedMagicLoginEnvironment(t, db, "active")
	runner := &stubRunner{output: "Error: wp failed", err: errors.New("exit status 1")}
	svc := NewService(db, runner)

	_, err := svc.CreateMagicLogin(context.Background(), "env-1")
	if !errors.Is(err, ErrWPCliError) {
		t.Fatalf("expected wp-cli error, got %v", err)
	}
}

type stubRunner struct {
	output string
	err    error
	called int
	host   string
	port   int
	user   string
	runFn  func(ctx context.Context, host string, port int, user string, remoteArgs ...string) (string, error)
}

func (s *stubRunner) Run(ctx context.Context, host string, port int, user string, remoteArgs ...string) (string, error) {
	s.called++
	s.host = host
	s.port = port
	s.user = user
	if s.runFn != nil {
		return s.runFn(ctx, host, port, user, remoteArgs...)
	}
	return s.output, s.err
}

func openSSHTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "ssh-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE nodes (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			hostname TEXT NOT NULL,
			public_ip TEXT NULL,
			ssh_port INTEGER NOT NULL,
			ssh_user TEXT NOT NULL,
			status TEXT NOT NULL,
			is_local INTEGER NOT NULL,
			last_seen_at TEXT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL
		);
		CREATE TABLE environments (
			id TEXT PRIMARY KEY,
			site_id TEXT NOT NULL,
			name TEXT NOT NULL,
			slug TEXT NOT NULL,
			environment_type TEXT NOT NULL,
			status TEXT NOT NULL,
			node_id TEXT NOT NULL,
			source_environment_id TEXT NULL,
			promotion_preset TEXT NOT NULL,
			preview_url TEXT NOT NULL,
			primary_domain_id TEXT NULL,
			current_release_id TEXT NULL,
			drift_status TEXT NOT NULL,
			drift_checked_at TEXT NULL,
			last_drift_check_id TEXT NULL,
			fastcgi_cache_enabled INTEGER NOT NULL,
			redis_cache_enabled INTEGER NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL,
			UNIQUE(site_id, slug)
		);
	`); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return db
}

func seedMagicLoginEnvironment(t *testing.T, db *sql.DB, status string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO nodes (id, name, hostname, public_ip, ssh_port, ssh_user, status, is_local, last_seen_at, created_at, updated_at, state_version)
		VALUES ('node-1', 'Node', 'node.local', '203.0.113.10', 22, 'root', 'active', 1, NULL, datetime('now'), datetime('now'), 1);
		INSERT INTO environments (
			id, site_id, name, slug, environment_type, status, node_id, source_environment_id,
			promotion_preset, preview_url, primary_domain_id, current_release_id, drift_status,
			drift_checked_at, last_drift_check_id, fastcgi_cache_enabled, redis_cache_enabled,
			created_at, updated_at, state_version
		)
		VALUES ('env-1', 'site-1', 'Production', 'production', 'production', ?, 'node-1', NULL, 'content-protect', 'http://preview.test', NULL, NULL, 'unknown', NULL, NULL, 1, 1, datetime('now'), datetime('now'), 1);
	`, status); err != nil {
		t.Fatalf("seed magic login environment: %v", err)
	}
}
