Status: active
Owner: platform
Last Reviewed: 2026-02-20
Depends On: docs/job-execution.md, docs/ansible-execution.md, docs/data-model.md
Supersedes: none

# Job Types

This document defines the canonical `jobs.job_type` registry for Pressluft MVP.

## Rules

- `jobs.job_type` must be one of the values listed in this document.
- All infrastructure mutation job types execute through Ansible.
- Node queries are synchronous and do not create job rows.
- Job scope must align with `docs/data-model.md`: site-scoped jobs require `site_id`; global jobs must set `site_id = NULL`.
- New job types must be added here, in `docs/ansible-execution.md`, and in owning feature specs in the same PR.

## Mutation Job Types (Ansible)

| Job Type | Scope | Playbook | Owning Feature |
|---|---|---|---|
| `node_provision` | global (`site_id = NULL`) | `ansible/playbooks/node-provision.yml` | `docs/features/feature-node-provision.md` |
| `site_create` | site | `ansible/playbooks/site-create.yml` | `docs/features/feature-site-create.md` |
| `site_import` | site | `ansible/playbooks/site-import.yml` | `docs/features/feature-site-import.md` |
| `env_create` | site | `ansible/playbooks/env-create.yml` | `docs/features/feature-environment-create-clone.md` |
| `env_restore` | site | `ansible/playbooks/env-restore.yml` | `docs/features/feature-environment-restore.md` |
| `env_promote` | site | `ansible/playbooks/env-promote.yml` | `docs/features/feature-promotion-drift.md` |
| `env_cache_toggle` | site | `ansible/playbooks/env-cache-toggle.yml` | `docs/features/feature-cache-controls.md` |
| `cache_purge` | site | `ansible/playbooks/cache-purge.yml` | `docs/features/feature-cache-controls.md` |
| `backup_create` | site | `ansible/playbooks/backup-create.yml` | `docs/features/feature-backups.md` |
| `domain_add` | site | `ansible/playbooks/domain-add.yml` | `docs/features/feature-domains-and-tls.md` |
| `domain_remove` | site | `ansible/playbooks/domain-remove.yml` | `docs/features/feature-domains-and-tls.md` |
| `drift_check` | site | `ansible/playbooks/drift-check.yml` | `docs/features/feature-promotion-drift.md` |
| `health_check` | site | `ansible/playbooks/health-check.yml` | `docs/features/feature-health-checks-and-rollback.md` |
| `backup_cleanup` | site | `ansible/playbooks/backup-cleanup.yml` | `docs/features/feature-backup-retention-cleanup.md` |
| `release_rollback` | site | `ansible/playbooks/release-rollback.yml` | `docs/features/feature-health-checks-and-rollback.md` |

## Synchronous Node Queries (No Job Row)

| Operation | Endpoint | Execution Model |
|---|---|---|
| magic login | `POST /api/environments/{id}/magic-login` | direct SSH node query |
