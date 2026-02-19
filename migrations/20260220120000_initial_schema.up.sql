CREATE TABLE IF NOT EXISTS schema_migrations (
  version TEXT PRIMARY KEY,
  applied_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS sites (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  slug TEXT NOT NULL UNIQUE,
  status TEXT NOT NULL CHECK (status IN ('active', 'cloning', 'deploying', 'restoring', 'failed')),
  primary_environment_id TEXT NULL REFERENCES environments(id),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  state_version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS nodes (
  id TEXT PRIMARY KEY,
  name TEXT NOT NULL,
  hostname TEXT NOT NULL,
  public_ip TEXT NULL,
  ssh_port INTEGER NOT NULL,
  ssh_user TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('active', 'provisioning', 'unreachable', 'decommissioned')),
  is_local INTEGER NOT NULL CHECK (is_local IN (0, 1)),
  last_seen_at TEXT NULL,
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  state_version INTEGER NOT NULL DEFAULT 1
);

CREATE TABLE IF NOT EXISTS environments (
  id TEXT PRIMARY KEY,
  site_id TEXT NOT NULL REFERENCES sites(id),
  name TEXT NOT NULL,
  slug TEXT NOT NULL,
  environment_type TEXT NOT NULL CHECK (environment_type IN ('production', 'staging', 'clone')),
  status TEXT NOT NULL CHECK (status IN ('active', 'cloning', 'deploying', 'restoring', 'failed')),
  node_id TEXT NOT NULL REFERENCES nodes(id),
  source_environment_id TEXT NULL REFERENCES environments(id),
  promotion_preset TEXT NOT NULL CHECK (promotion_preset IN ('content-protect', 'commerce-protect')),
  preview_url TEXT NOT NULL,
  primary_domain_id TEXT NULL REFERENCES domains(id),
  current_release_id TEXT NULL REFERENCES releases(id),
  drift_status TEXT NOT NULL CHECK (drift_status IN ('unknown', 'clean', 'drifted')),
  drift_checked_at TEXT NULL,
  last_drift_check_id TEXT NULL REFERENCES drift_checks(id),
  fastcgi_cache_enabled INTEGER NOT NULL DEFAULT 1 CHECK (fastcgi_cache_enabled IN (0, 1)),
  redis_cache_enabled INTEGER NOT NULL DEFAULT 1 CHECK (redis_cache_enabled IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL,
  state_version INTEGER NOT NULL DEFAULT 1,
  UNIQUE(site_id, slug)
);

CREATE TABLE IF NOT EXISTS releases (
  id TEXT PRIMARY KEY,
  environment_id TEXT NOT NULL REFERENCES environments(id),
  source_type TEXT NOT NULL CHECK (source_type IN ('git', 'upload')),
  source_ref TEXT NOT NULL,
  path TEXT NOT NULL,
  health_status TEXT NOT NULL CHECK (health_status IN ('unknown', 'healthy', 'unhealthy')),
  notes TEXT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS backups (
  id TEXT PRIMARY KEY,
  environment_id TEXT NOT NULL REFERENCES environments(id),
  backup_scope TEXT NOT NULL CHECK (backup_scope IN ('db', 'files', 'full')),
  status TEXT NOT NULL CHECK (status IN ('pending', 'running', 'completed', 'failed', 'expired')),
  storage_type TEXT NOT NULL CHECK (storage_type IN ('s3')),
  storage_path TEXT NOT NULL,
  retention_until TEXT NOT NULL,
  checksum TEXT NULL,
  size_bytes INTEGER NULL,
  created_at TEXT NOT NULL,
  completed_at TEXT NULL
);

CREATE TABLE IF NOT EXISTS domains (
  id TEXT PRIMARY KEY,
  environment_id TEXT NOT NULL REFERENCES environments(id),
  hostname TEXT NOT NULL UNIQUE,
  tls_status TEXT NOT NULL CHECK (tls_status IN ('pending', 'active', 'failed', 'disabled')),
  tls_issuer TEXT NOT NULL CHECK (tls_issuer IN ('letsencrypt')),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS jobs (
  id TEXT PRIMARY KEY,
  job_type TEXT NOT NULL,
  status TEXT NOT NULL CHECK (status IN ('queued', 'running', 'succeeded', 'failed', 'cancelled')),
  site_id TEXT NULL REFERENCES sites(id),
  environment_id TEXT NULL REFERENCES environments(id),
  node_id TEXT NULL REFERENCES nodes(id),
  payload_json TEXT NOT NULL,
  attempt_count INTEGER NOT NULL,
  max_attempts INTEGER NOT NULL DEFAULT 3,
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

CREATE INDEX IF NOT EXISTS idx_jobs_status_run_after ON jobs(status, run_after);
CREATE INDEX IF NOT EXISTS idx_jobs_site_id_status ON jobs(site_id, status);

CREATE TABLE IF NOT EXISTS users (
  id TEXT PRIMARY KEY,
  email TEXT NOT NULL UNIQUE,
  display_name TEXT NOT NULL,
  role TEXT NOT NULL CHECK (role IN ('admin')),
  password_hash TEXT NOT NULL,
  is_active INTEGER NOT NULL CHECK (is_active IN (0, 1)),
  created_at TEXT NOT NULL,
  updated_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS auth_sessions (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  session_token TEXT NOT NULL UNIQUE,
  expires_at TEXT NOT NULL,
  created_at TEXT NOT NULL,
  revoked_at TEXT NULL
);

CREATE TABLE IF NOT EXISTS drift_checks (
  id TEXT PRIMARY KEY,
  environment_id TEXT NOT NULL REFERENCES environments(id),
  promotion_preset TEXT NOT NULL CHECK (promotion_preset IN ('content-protect', 'commerce-protect')),
  status TEXT NOT NULL CHECK (status IN ('unknown', 'clean', 'drifted')),
  db_checksums_json TEXT NULL,
  file_checksums_json TEXT NULL,
  checked_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS audit_logs (
  id TEXT PRIMARY KEY,
  user_id TEXT NOT NULL REFERENCES users(id),
  action TEXT NOT NULL,
  resource_type TEXT NOT NULL,
  resource_id TEXT NOT NULL,
  result TEXT NOT NULL,
  created_at TEXT NOT NULL
);

CREATE TABLE IF NOT EXISTS settings (
  key TEXT PRIMARY KEY,
  value TEXT NOT NULL,
  updated_at TEXT NOT NULL
);
