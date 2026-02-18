# Technical Architecture

Pressluft is a single-codebase WordPress orchestration control plane designed for deterministic site lifecycle management. Deployment topology does not alter internal architecture.

## 1. Control Plane

Implemented as a monolithic Go application with an embedded web UI, running as a non-root system service.

Responsibilities:

- Explicit site lifecycle state machine: ACTIVE, CLONING, DEPLOYING, RESTORING, FAILED
- Persistent state storage (SQLite for MVP)
- DB-backed job queue (tables in the primary database)
- Web UI/API for site creation and node registry
- SSH execution layer that invokes external Ansible playbooks
- Concurrency control via row-level versioning

The database is the source of truth. All infrastructure mutations are modeled as transactional jobs. Only one mutation per site may execute at a time. State transitions occur inside database transactions to prevent race conditions.

## 2. Node Model

A Node is a first-class resource in the database representing any managed Linux host.

Required stack:

- Ubuntu LTS
- Nginx
- PHP-FPM
- MariaDB
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
3. Run health checks
4. Switch current symlink
5. Reload PHP-FPM

Rollback reverts the symlink.

Long-running operations execute via detached processes (systemd-run or equivalent) with status polling to tolerate SSH interruptions.

## 6. Backups and Promotion

Before destructive operations:

- Database exported using single-transaction mode
- Files archived
- Optional off-site storage via restic

Promotion workflows include drift validation and mandatory backup enforcement.

This architecture prioritizes determinism, isolation, and operational simplicity while remaining multi-node capable.
