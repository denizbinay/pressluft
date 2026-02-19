Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/job-execution.md, docs/provisioning-spec.md
Supersedes: none

# Ansible Execution

This document defines how the Go control plane invokes Ansible playbooks for all infrastructure operations.

## Principles

- Ansible is the execution mechanism for all node-targeted **mutations**. Non-mutating **node queries** use direct SSH from the Go binary (see `docs/technical-architecture.md` Section 7).
- The Go control plane invokes `ansible-playbook` as a local subprocess on the control plane host.
- Each job type maps to exactly one playbook. Playbooks are not shared across job types.
- The database is the single source of truth. Inventory and variables are derived from DB state at invocation time.
- Ansible inherits concurrency guarantees from the job queue. No additional Ansible-level concurrency controls are required.

## Runtime Dependency

- Ansible >= 2.16 must be installed on the control plane host.
- The `install.sh` bootstrap script is responsible for installing Ansible.
- The Go binary does not embed or bundle Ansible. It expects `ansible-playbook` to be available on `$PATH`.

## Configuration

A minimal `ansible.cfg` is committed to the `ansible/` directory. It locks down runtime behavior and avoids dependence on system-wide or user-level Ansible configuration.

Required settings:

```ini
[defaults]
gathering = explicit
stdout_callback = yaml
nocows = true
retry_files_enabled = false
interpreter_python = auto_silent

[ssh_connection]
pipelining = true
```

- `gathering = explicit` disables automatic fact gathering. Playbooks that need facts must use an explicit `ansible.builtin.setup` task. This reduces execution time since all required variables are passed via extra-vars.
- `pipelining = true` reduces the number of SSH operations per task.
- No other configuration files are used. All connection parameters are passed via CLI arguments and dynamic inventory.

## Invocation Model

The Go job executor invokes Ansible as a local subprocess:

1. The job executor acquires a job from the queue (see `docs/job-execution.md`).
2. It resolves the target playbook from the job type (see Playbook Registry below).
3. It builds a dynamic inventory and extra-vars from the job's `payload_json` and related DB records.
4. It executes `ansible-playbook` as a child process with the constructed arguments.
5. It captures stdout and stderr from the subprocess.
6. On process exit, it maps the exit code to a job outcome (see Exit Code Contract below).
7. It updates the job and related resource states in a single DB transaction.

The subprocess is executed synchronously within the job executor goroutine. The job timeout (default 30 minutes, see `docs/job-execution.md`) applies to the entire subprocess lifetime. If the timeout is reached, the subprocess is killed via process signal.

### Command Structure

```
ansible-playbook \
  -i <dynamic-inventory-path> \
  -e @<extra-vars-file-path> \
  --ssh-extra-args='-o StrictHostKeyChecking=accept-new' \
  ansible/playbooks/<playbook>.yml
```

- The dynamic inventory file and extra-vars file are written to a temporary directory before invocation and cleaned up after completion.
- The `--ssh-extra-args` flag is used to pass SSH options. Key-based auth is configured via Ansible connection variables, not the system SSH config.

## Inventory Generation

Inventory is dynamic and DB-driven. No static inventory files are committed to the repository.

At invocation time, the job executor:

1. Reads the target `node` record from the database (resolved via `jobs.node_id`).
2. Generates an INI-format inventory file containing a single host entry with connection variables:

```ini
[target]
<node.hostname> ansible_port=<node.ssh_port> ansible_user=<node.ssh_user> ansible_ssh_private_key_file=<key_path>
```

3. Writes this file to a temporary path scoped to the job execution.

The `ansible/inventories/` directory in the repository is reserved for inventory plugins or shared group variables if needed in the future. It must not contain static host definitions.

### SSH Key Resolution

- SSH private keys are stored by the control plane in `/var/lib/pressluft/secrets` (see `docs/security-and-secrets.md`).
- The key path for a node is resolved from the secrets store and passed as `ansible_ssh_private_key_file` in the inventory.

## Parameter Passing

All job parameters are passed to Ansible via `--extra-vars` using a JSON file.

The extra-vars file is constructed by:

1. Deserializing `jobs.payload_json`.
2. Enriching it with resolved DB state:
   - `site_id`, `site_slug`
   - `environment_id`, `environment_slug`, `environment_type`
   - `node_hostname`, `node_ssh_user`
   - `release_path` (for deploy/rollback jobs)
   - `backup_storage_path` (for backup/restore jobs)
   - `domain_hostname` (for domain jobs)
   - `preview_url` (for site_create, env_create, domain_remove jobs)
   - `preview_domain` (for node_provision, when wildcard cert is needed)
   - `node_public_ip` (for domain_add DNS verification, node_provision)
    - `dns01_provider`, `dns01_credentials` (for node_provision, when wildcard cert is needed)
    - `fastcgi_cache_enabled`, `redis_cache_enabled` (for site_create, env_create, env_cache_toggle, and any playbook that generates Nginx server blocks)
    - `redis_prefix` (for site_create, env_create, env_cache_toggle — value: `pressluft_{environment_id}:`)
    - Other fields as required by the specific playbook.
3. Writing the merged JSON object to a temporary file.

The playbook accesses these values as standard Ansible variables (e.g., `{{ site_slug }}`, `{{ environment_id }}`).

### Variable Namespacing

All Pressluft-provided variables use flat, underscore-separated names prefixed where needed for clarity. No nested objects. This avoids ambiguity in Ansible's variable precedence.

### Ansible Vault

Ansible Vault is not used. The Go control plane is the sole secret store (see `docs/security-and-secrets.md`). Secrets are injected into playbooks via extra-vars at invocation time. Introducing Ansible Vault would create a second source of truth for secrets and is prohibited.

## Playbook Registry

Each job type maps to exactly one playbook. Playbook files live in `ansible/playbooks/`.

| Job Type | Playbook | Description |
|----------|----------|-------------|
| `node_provision` | `node-provision.yml` | Bootstrap server stack on a node (includes Redis, Fail2Ban, 7G WAF, PHP hardening, security headers — see `docs/provisioning-spec.md`) |
| `site_create` | `site-create.yml` | Create system user, DB, PHP-FPM pool, Nginx block for preview URL, initial release, configure Redis Object Cache and `wp-config.php` Redis settings |
| `site_import` | `site-import.yml` | Import archive, restore DB, copy files, URL rewrite, configure Redis Object Cache |
| `env_create` | `env-create.yml` | Clone environment (files, DB, config), create Nginx block for preview URL, WordPress URL rewrite to new preview URL, configure Redis Object Cache |
| `env_deploy` | `env-deploy.yml` | Create release, install deps, symlink switch, reload, purge FastCGI cache |
| `env_update` | `env-update.yml` | Apply WordPress core/plugin/theme updates, purge FastCGI cache |
| `env_restore` | `env-restore.yml` | Restore from backup archive, purge FastCGI cache |
| `env_promote` | `env-promote.yml` | Selective sync with drift-protected tables/files, purge FastCGI cache on target |
| `env_cache_toggle` | `env-cache-toggle.yml` | Enable or disable FastCGI page cache and/or Redis Object Cache for an environment. Regenerates Nginx server block, toggles Redis Object Cache drop-in via WP-CLI. |
| `cache_purge` | `cache-purge.yml` | Purge FastCGI page cache and/or Redis Object Cache for an environment on demand |
| `backup_create` | `backup-create.yml` | Export DB, archive files (excluding cache directories), upload to S3 |
| `domain_add` | `domain-add.yml` | Verify DNS, configure Nginx server block (with WAF include and conditional cache directives), provision TLS certificate, WordPress URL rewrite, purge FastCGI cache (see `docs/domain-and-routing.md`) |
| `domain_remove` | `domain-remove.yml` | Remove Nginx server block and TLS certificate, revert WordPress URL to preview URL, purge FastCGI cache (see `docs/domain-and-routing.md`) |
| `drift_check` | `drift-check.yml` | Compute checksums for drift comparison |
| `health_check` | `health-check.yml` | HTTP check, WP-CLI check, DB connectivity check |
| `backup_cleanup` | `backup-cleanup.yml` | Remove expired backups from S3 storage |
| `release_rollback` | `release-rollback.yml` | Revert current symlink to previous release, reload, purge FastCGI cache |

Playbooks may use shared roles from `ansible/roles/` for common tasks (e.g., Nginx reload, PHP-FPM pool management, symlink operations). Roles are internal implementation details of the playbooks and do not constitute a separate interface.

### Role Structure

Roles must follow the standard Ansible directory layout (`tasks/`, `handlers/`, `templates/`, `files/`, `vars/`, `defaults/`, `meta/`). Only include directories that the role actually uses.

Roles that accept variables from playbooks must define `meta/argument_specs.yml` for input validation. This ensures that missing or malformed variables from the Go control plane are caught at role entry, not mid-execution.

### Expected Shared Roles

| Role | Used By | Purpose |
|------|---------|---------|
| `nginx-reload` | Any playbook that modifies Nginx config | Handler-based Nginx reload |
| `nginx-cache-purge` | `env-deploy`, `env-update`, `env-promote`, `env-restore`, `cache-purge`, `domain-add`, `domain-remove`, `release-rollback` | Purge FastCGI cache files for a specific environment |
| `nginx-waf` | `node-provision` | Deploy vendored 7G WAF rules to `/etc/nginx/conf.d/` |
| `redis-object-cache` | `site-create`, `site-import`, `env-create`, `env-cache-toggle` | Install/remove Redis Object Cache drop-in via WP-CLI, configure `wp-config.php` Redis constants |
| `php-fpm-pool` | `site-create`, `env-create` | Create/configure PHP-FPM pool with hardened settings (including `open_basedir`) |
| `fail2ban` | `node-provision` | Deploy Fail2Ban jails for WordPress protection |

## Exit Code Contract

Ansible exit codes are mapped to job outcomes as follows:

| Exit Code | Meaning | Job Action |
|-----------|---------|------------|
| 0 | Success | Mark job `succeeded`. Transition resource to success state. |
| 1 | General error (play error) | Retryable. Re-queue with backoff if attempts remain. |
| 2 | One or more hosts failed | Retryable. Re-queue with backoff if attempts remain. |
| 4 | One or more hosts unreachable | Retryable. Re-queue with backoff if attempts remain. |
| 5 | Invalid options / syntax error | Non-retryable. Mark job `failed` with `error_code = ANSIBLE_SYNTAX_ERROR`. |
| 250 | Unexpected error | Non-retryable. Mark job `failed` with `error_code = ANSIBLE_UNEXPECTED_ERROR`. |
| Other | Unknown | Non-retryable. Mark job `failed` with `error_code = ANSIBLE_UNKNOWN_EXIT`. |

For exit codes 1, 2, and 4: if `attempt_count >= max_attempts`, the job is marked `failed` instead of re-queued.

### Error Capture

- Ansible stdout and stderr are captured in memory during subprocess execution.
- On failure, the last 10 KB of combined output is stored in `jobs.error_message`.
- `jobs.error_code` is set to a structured code derived from the exit code (e.g., `ANSIBLE_HOST_FAILED`, `ANSIBLE_HOST_UNREACHABLE`, `ANSIBLE_PLAY_ERROR`).

### Error Code Reference

| error_code | Trigger |
|------------|---------|
| `ANSIBLE_PLAY_ERROR` | Exit code 1 |
| `ANSIBLE_HOST_FAILED` | Exit code 2 |
| `ANSIBLE_HOST_UNREACHABLE` | Exit code 4 |
| `ANSIBLE_SYNTAX_ERROR` | Exit code 5 |
| `ANSIBLE_UNEXPECTED_ERROR` | Exit code 250 |
| `ANSIBLE_UNKNOWN_EXIT` | Any other non-zero exit code |
| `ANSIBLE_TIMEOUT` | Subprocess killed due to job timeout |
| `JOB_TIMEOUT` | Job exceeded total timeout (set by `docs/job-execution.md`) |

## Output Handling

For all job executions:

- Stdout is streamed to the control plane log file in real time (for operational debugging).
- On job completion (success or failure), a truncated copy of the output is persisted with the job record.
- Sensitive values (passwords, keys) must never appear in Ansible output. Playbooks must use `no_log: true` on tasks that handle secrets.

## Concurrency

Ansible concurrency is fully governed by the job queue:

- The job queue guarantees at most one running job per site and one running job per node (see `docs/job-execution.md`).
- Each `ansible-playbook` subprocess targets exactly one node via the dynamic inventory.
- No Ansible-level fork or serial configuration is required for concurrency safety.
- The job executor does not maintain a pool of concurrent Ansible processes per site or per node.

This means Ansible's default `forks` setting is irrelevant (single host per invocation) and no `serial` directive is needed.

## Long-Running Operations

For operations that may exceed SSH connection stability (large file transfers, database imports):

- Playbooks should use Ansible's `async` and `poll` directives to launch detached processes on the remote host.
- This aligns with `docs/technical-architecture.md` Section 5: "Long-running operations execute via detached processes (systemd-run or equivalent) with status polling."
- The Ansible task launches the operation asynchronously and polls for completion, tolerating transient SSH interruptions.

## Idempotency

All playbooks must be idempotent (safe to re-run):

- This is required by the at-least-once job execution guarantee.
- Playbooks must detect existing state and converge, not blindly create or overwrite.
- Resource creation tasks must use conditional checks (e.g., `creates:`, `when: not exists`).
- Database operations must be wrapped in idempotent patterns (e.g., `CREATE DATABASE IF NOT EXISTS`).

## Playbook Development Rules

### Structural Rules

- Playbooks target a single host. Multi-host plays are not used.
- All plays, tasks, and blocks must have a descriptive `name:`. Unnamed tasks produce unreadable output in error logs.
- All modules must be referenced by their fully qualified collection name (FQCN). Use `ansible.builtin.copy`, not `copy`. This avoids ambiguity when collections are present.
- Modules that support a `state` parameter must set it explicitly (e.g., `state: present`, `state: absent`). Do not rely on module defaults.
- Use handlers for service reloads (Nginx, PHP-FPM) triggered by configuration changes, rather than inline reload tasks. This prevents unnecessary reloads when configuration has not changed.
- Playbooks set `gather_facts: false` unless they explicitly need host facts. All required data is passed via extra-vars.

### Security Rules

- All tasks that handle secrets use `no_log: true`.
- Playbooks must not hardcode paths, users, or credentials. All values come from extra-vars.

### Scope Rules

- Playbooks do not modify files outside the environment's directory scope (`/var/www/sites/<site_id>/`) except for shared system configuration (Nginx, PHP-FPM, MariaDB).
- Roles are shared across playbooks via `ansible/roles/` but playbook-to-role mapping is an implementation detail, not a contract.

### Development Workflow

- Run `ansible-playbook --syntax-check` on all playbooks before committing. This catches YAML and Ansible syntax errors without executing against a host.
- Run `ansible-lint` if available to catch common anti-patterns.
