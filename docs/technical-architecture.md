Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/job-execution.md, docs/ansible-execution.md
Supersedes: none

# Technical Architecture

Pressluft is a single-codebase WordPress orchestration control plane designed for deterministic site lifecycle management. Deployment topology does not alter internal architecture.

## 1. Control Plane

Implemented as a monolithic Go application with an embedded web UI, running as a non-root system service.

Responsibilities:

- Explicit site lifecycle state machine: active, cloning, deploying, restoring, failed
- Persistent state storage (SQLite for MVP)
- DB-backed job queue (tables in the primary database)
- Web UI/API for site creation and node registry
- Ansible execution layer: the job executor invokes `ansible-playbook` as a local subprocess for all node-targeted operations. Inventory is generated dynamically from the database. Each job type maps to one playbook. See `docs/ansible-execution.md` for the full contract.
- Concurrency control via row-level versioning

The database is the source of truth. All infrastructure mutations are modeled as transactional jobs. Only one mutation per site may execute at a time. State transitions occur inside database transactions to prevent race conditions. Non-mutating, ephemeral operations on nodes (node queries) use direct SSH from the Go binary, bypassing the job queue and Ansible (see Section 7).

## 2. Node Model

A Node is a first-class resource in the database representing any managed Linux host.

Required stack:

- Ubuntu 24.04 LTS
- Nginx
- PHP-FPM
- MariaDB
- Redis
- WP-CLI

Nodes are agentless and accessed exclusively over SSH. Even in single-server mode, the host running Pressluft is registered as a Node and managed via SSH to localhost. This ensures identical execution paths for local and remote management and enables future multi-node expansion without redesign.

## 3. Provisioning

For Hetzner-based setups:

1. User provisions a fresh Ubuntu server
2. `curl install.sh` installs Pressluft and prerequisites
3. Idempotent bootstrap executed via Ansible over SSH
4. Local node registered in control plane

Bootstrap installs the web stack, configures firewall rules, creates system users, and enforces baseline security. All bootstrap steps are idempotent to tolerate retries.

## 4. Isolation Model

Each site environment receives:

- Dedicated Linux user (unique UID)
- Dedicated PHP-FPM pool running as that user
- Dedicated MariaDB database and restricted DB user
- Dedicated Nginx server block

Filesystem layout:

/var/www/sites/<site_id>  
  releases/<timestamp>/  
  current -> releases/<timestamp>  
  shared/uploads/  
  shared/wp-config.php  

Releases are immutable. Uploads are writable and non-executable. Configuration and uploads are symlinked into each release.

## 5. Deployment Model

Deployments use atomic symlink switching:

1. Create new release directory
2. Install dependencies
3. Run health checks (see docs/health-checks.md)
4. Switch current symlink
5. Reload PHP-FPM

Rollback reverts the symlink.

Long-running operations execute via detached processes (systemd-run or equivalent) with status polling to tolerate SSH interruptions.

## 6. Backups and Promotion

Before destructive operations:

- Database exported using single-transaction mode
- Files archived
- Off-site storage via S3-compatible storage

Promotion workflows include drift validation and mandatory backup enforcement.

## 7. Node Queries

Not all node-targeted operations are mutations. Some operations are lightweight, non-destructive, and time-sensitive. These are classified as **node queries** and execute via direct SSH from the Go binary (`internal/ssh` package), bypassing both the job queue and Ansible.

**Invariant (refined):** All infrastructure **mutations** go through the job queue and Ansible. Non-mutating, ephemeral **node queries** use direct SSH from the Go binary.

A node query must satisfy all of the following:

1. **Non-destructive** — no filesystem, database, or configuration changes that persist beyond the session (ephemeral tokens and transients are acceptable).
2. **Single-command** — one SSH round-trip, one command execution.
3. **Time-bounded** — hard timeout of 10 seconds. If the command does not complete within this window, the SSH connection is terminated and an error is returned.
4. **Concurrency-safe** — safe to execute while a job is running against the same site or node. Node queries do not acquire job locks or check concurrency constraints.

Node queries are executed synchronously: the API handler opens an SSH connection to the target node, runs the command as the appropriate Linux user, and returns the result in the HTTP response. There is no `job_id` — the response contains the query result directly.

### Current Node Queries

| Query | Command | Purpose |
|-------|---------|---------|
| Magic login | `wp eval '...'` (as environment Linux user) | Generate a one-time WordPress admin session token |

### Future Candidates (Not Yet Specced)

- PHP version detection
- Disk usage per environment
- WordPress version and plugin inventory
- MariaDB status checks

This architecture prioritizes determinism, isolation, and operational simplicity while remaining multi-node capable.
