Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/job-execution.md, docs/ansible-execution.md, docs/contract-traceability.md
Supersedes: none

# Job Types

This document defines the canonical `jobs.job_type` registry for Pressluft MVP.

## Rules

- `jobs.job_type` must be one of the values listed in this document.
- All infrastructure mutation job types execute through Ansible.
- Node queries are synchronous and do not create job rows.
- New job types must be added here, in `docs/ansible-execution.md`, and in owning feature specs in the same PR.

## Mutation Job Types (Ansible)

| Job Type | Playbook | Owning Feature |
|---|---|---|
| `node_provision` | `ansible/playbooks/node-provision.yml` | `docs/features/feature-node-provision.md` |
| `site_create` | `ansible/playbooks/site-create.yml` | `docs/features/feature-site-create.md` |
| `site_import` | `ansible/playbooks/site-import.yml` | `docs/features/feature-site-import.md` |
| `env_create` | `ansible/playbooks/env-create.yml` | `docs/features/feature-environment-create-clone.md` |
| `env_deploy` | `ansible/playbooks/env-deploy.yml` | `docs/features/feature-environment-deploy-updates.md` |
| `env_update` | `ansible/playbooks/env-update.yml` | `docs/features/feature-environment-deploy-updates.md` |
| `env_restore` | `ansible/playbooks/env-restore.yml` | `docs/features/feature-environment-restore.md` |
| `env_promote` | `ansible/playbooks/env-promote.yml` | `docs/features/feature-promotion-drift.md` |
| `env_cache_toggle` | `ansible/playbooks/env-cache-toggle.yml` | `docs/features/feature-cache-controls.md` |
| `cache_purge` | `ansible/playbooks/cache-purge.yml` | `docs/features/feature-cache-controls.md` |
| `backup_create` | `ansible/playbooks/backup-create.yml` | `docs/features/feature-backups.md` |
| `domain_add` | `ansible/playbooks/domain-add.yml` | `docs/features/feature-domains-and-tls.md` |
| `domain_remove` | `ansible/playbooks/domain-remove.yml` | `docs/features/feature-domains-and-tls.md` |
| `drift_check` | `ansible/playbooks/drift-check.yml` | `docs/features/feature-promotion-drift.md` |
| `health_check` | `ansible/playbooks/health-check.yml` | `docs/features/feature-health-checks-and-rollback.md` |
| `backup_cleanup` | `ansible/playbooks/backup-cleanup.yml` | `docs/features/feature-backup-retention-cleanup.md` |
| `release_rollback` | `ansible/playbooks/release-rollback.yml` | `docs/features/feature-health-checks-and-rollback.md` |

## Synchronous Node Queries (No Job Row)

| Operation | Endpoint | Execution Model |
|---|---|---|
| magic login | `POST /api/environments/:id/magic-login` | direct SSH node query |
