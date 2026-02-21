package domains

import (
	"context"
	"database/sql"
	"path/filepath"
	"strings"
	"testing"

	"pressluft/internal/store"
)

func TestAddQueuesDomainAddJobAndCreatesPendingDomain(t *testing.T) {
	t.Parallel()

	db := openDomainsTestDB(t)
	seedDomainEnvironment(t, db)
	svc := NewService(db)

	result, err := svc.Add(context.Background(), AddInput{EnvironmentID: "env-1", Hostname: "Example.com"})
	if err != nil {
		t.Fatalf("add domain: %v", err)
	}
	if strings.TrimSpace(result.JobID) == "" || strings.TrimSpace(result.DomainID) == "" {
		t.Fatalf("expected domain and job ids")
	}

	assertDomainString(t, db, "SELECT tls_status FROM domains WHERE id = ?", "pending", result.DomainID)
	assertDomainString(t, db, "SELECT hostname FROM domains WHERE id = ?", "example.com", result.DomainID)
	assertDomainString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "domain_add", result.JobID)
}

func TestListByEnvironmentReturnsAuthoritativeTLSState(t *testing.T) {
	t.Parallel()

	db := openDomainsTestDB(t)
	seedDomainEnvironment(t, db)
	if _, err := db.Exec(`
		INSERT INTO domains (id, environment_id, hostname, tls_status, tls_issuer, created_at, updated_at)
		VALUES ('domain-1', 'env-1', 'example.com', 'active', 'letsencrypt', datetime('now'), datetime('now'))
	`); err != nil {
		t.Fatalf("seed domain: %v", err)
	}
	svc := NewService(db)

	listed, err := svc.ListByEnvironment(context.Background(), "env-1")
	if err != nil {
		t.Fatalf("list domains: %v", err)
	}
	if len(listed) != 1 {
		t.Fatalf("expected one domain, got %d", len(listed))
	}
	if listed[0].TLSStatus != "active" {
		t.Fatalf("expected tls status active, got %s", listed[0].TLSStatus)
	}
}

func TestRemoveQueuesDomainRemoveJob(t *testing.T) {
	t.Parallel()

	db := openDomainsTestDB(t)
	seedDomainEnvironment(t, db)
	seedDomain(t, db, "domain-1", "env-1", "example.com", "active")
	svc := NewService(db)

	result, err := svc.Remove(context.Background(), "domain-1")
	if err != nil {
		t.Fatalf("remove domain: %v", err)
	}
	if strings.TrimSpace(result.JobID) == "" {
		t.Fatalf("expected job id")
	}

	assertDomainString(t, db, "SELECT job_type FROM jobs WHERE id = ?", "domain_remove", result.JobID)
}

func TestMarkRemoveSucceededDeletesDomainAndClearsPrimary(t *testing.T) {
	t.Parallel()

	db := openDomainsTestDB(t)
	seedDomainEnvironment(t, db)
	seedDomain(t, db, "domain-1", "env-1", "example.com", "active")
	if _, err := db.Exec(`UPDATE environments SET primary_domain_id = 'domain-1' WHERE id = 'env-1'`); err != nil {
		t.Fatalf("set primary domain: %v", err)
	}
	seedRunningJob(t, db, "job-domain-remove", "domain_remove")
	svc := NewService(db)

	if err := svc.MarkRemoveSucceeded(context.Background(), "domain-1", "env-1", "job-domain-remove"); err != nil {
		t.Fatalf("mark remove succeeded: %v", err)
	}

	var count int
	if err := db.QueryRow("SELECT COUNT(1) FROM domains WHERE id = 'domain-1'").Scan(&count); err != nil {
		t.Fatalf("count domains: %v", err)
	}
	if count != 0 {
		t.Fatalf("expected domain deleted")
	}

	var primary sql.NullString
	if err := db.QueryRow("SELECT primary_domain_id FROM environments WHERE id = 'env-1'").Scan(&primary); err != nil {
		t.Fatalf("query primary domain: %v", err)
	}
	if primary.Valid {
		t.Fatalf("expected primary domain to be cleared")
	}

	assertDomainString(t, db, "SELECT status FROM jobs WHERE id = ?", "succeeded", "job-domain-remove")
}

func TestDNSMismatchErrorUsesDeterministicCodeAndMessage(t *testing.T) {
	t.Parallel()

	code, message := DNSMismatchError("Example.com", "203.0.113.10")
	if code != DNSMismatchErrorCode {
		t.Fatalf("unexpected code: %s", code)
	}
	if message != "dns mismatch: example.com does not resolve to node ip 203.0.113.10" {
		t.Fatalf("unexpected message: %s", message)
	}
}

func openDomainsTestDB(t *testing.T) *sql.DB {
	t.Helper()

	path := filepath.Join(t.TempDir(), "domains-test.db")
	db, err := store.OpenSQLite(path)
	if err != nil {
		t.Fatalf("open sqlite db: %v", err)
	}
	t.Cleanup(func() { _ = db.Close() })

	if _, err := db.Exec(`
		CREATE TABLE sites (
			id TEXT PRIMARY KEY,
			name TEXT NOT NULL,
			slug TEXT NOT NULL UNIQUE,
			status TEXT NOT NULL,
			primary_environment_id TEXT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL,
			state_version INTEGER NOT NULL
		);
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
		CREATE TABLE domains (
			id TEXT PRIMARY KEY,
			environment_id TEXT NOT NULL,
			hostname TEXT NOT NULL UNIQUE,
			tls_status TEXT NOT NULL,
			tls_issuer TEXT NOT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
		CREATE TABLE jobs (
			id TEXT PRIMARY KEY,
			job_type TEXT NOT NULL,
			status TEXT NOT NULL,
			site_id TEXT NULL,
			environment_id TEXT NULL,
			node_id TEXT NULL,
			payload_json TEXT NOT NULL,
			attempt_count INTEGER NOT NULL,
			max_attempts INTEGER NOT NULL,
			run_after TEXT NULL,
			locked_at TEXT NULL,
			locked_by TEXT NULL,
			started_at TEXT NULL,
			finished_at TEXT NULL,
			error_code TEXT NULL,
			error_message TEXT NULL,
			created_at TEXT NOT NULL,
			updated_at TEXT NOT NULL
		);
	`); err != nil {
		t.Fatalf("create test schema: %v", err)
	}

	return db
}

func seedDomainEnvironment(t *testing.T, db *sql.DB) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO sites (id, name, slug, status, primary_environment_id, created_at, updated_at, state_version)
		VALUES ('site-1', 'Acme', 'acme', 'active', 'env-1', datetime('now'), datetime('now'), 1);
		INSERT INTO nodes (id, name, hostname, public_ip, ssh_port, ssh_user, status, is_local, last_seen_at, created_at, updated_at, state_version)
		VALUES ('node-1', 'Node', 'node.local', '203.0.113.10', 22, 'root', 'active', 1, NULL, datetime('now'), datetime('now'), 1);
		INSERT INTO environments (
			id, site_id, name, slug, environment_type, status, node_id, source_environment_id,
			promotion_preset, preview_url, primary_domain_id, current_release_id, drift_status,
			drift_checked_at, last_drift_check_id, fastcgi_cache_enabled, redis_cache_enabled,
			created_at, updated_at, state_version
		)
		VALUES ('env-1', 'site-1', 'Production', 'production', 'production', 'active', 'node-1', NULL, 'content-protect', 'http://preview.test', NULL, NULL, 'unknown', NULL, NULL, 1, 1, datetime('now'), datetime('now'), 1);
	`); err != nil {
		t.Fatalf("seed domain env: %v", err)
	}
}

func seedDomain(t *testing.T, db *sql.DB, id, environmentID, hostname, tlsStatus string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO domains (id, environment_id, hostname, tls_status, tls_issuer, created_at, updated_at)
		VALUES (?, ?, ?, ?, 'letsencrypt', datetime('now'), datetime('now'))
	`, id, environmentID, hostname, tlsStatus); err != nil {
		t.Fatalf("seed domain: %v", err)
	}
}

func seedRunningJob(t *testing.T, db *sql.DB, jobID, jobType string) {
	t.Helper()

	if _, err := db.Exec(`
		INSERT INTO jobs (
			id, job_type, status, site_id, environment_id, node_id, payload_json,
			attempt_count, max_attempts, run_after, locked_at, locked_by,
			started_at, finished_at, error_code, error_message, created_at, updated_at
		)
		VALUES (?, ?, 'running', 'site-1', 'env-1', 'node-1', '{}', 1, 3, NULL, datetime('now'), 'worker', datetime('now'), NULL, NULL, NULL, datetime('now'), datetime('now'))
	`, jobID, jobType); err != nil {
		t.Fatalf("seed running job: %v", err)
	}
}

func assertDomainString(t *testing.T, db *sql.DB, query, expected, arg string) {
	t.Helper()

	var got string
	if err := db.QueryRow(query, arg).Scan(&got); err != nil {
		t.Fatalf("query %q: %v", query, err)
	}
	if got != expected {
		t.Fatalf("unexpected value for %q: got %q want %q", query, got, expected)
	}
}
