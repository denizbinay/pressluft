Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/templates/feature-spec-template.md
Supersedes: none

# Feature Specs

This directory contains implementation-bound feature specs.

## Naming

- Use `feature-<slug>.md`.
- Keep one feature spec per coherent unit of delivery.

## Required Rule

- No implementation task starts without a corresponding feature spec in this directory.

## Current Specs

- `docs/features/feature-auth-session.md`
- `docs/features/feature-install-bootstrap.md`
- `docs/features/feature-node-provision.md`
- `docs/features/feature-site-create.md`
- `docs/features/feature-site-import.md`
- `docs/features/feature-settings-domain-config.md`
- `docs/features/feature-environment-create-clone.md`
- `docs/features/feature-environment-deploy-updates.md`
- `docs/features/feature-environment-restore.md`
- `docs/features/feature-promotion-drift.md`
- `docs/features/feature-backups.md`
- `docs/features/feature-backup-retention-cleanup.md`
- `docs/features/feature-domains-and-tls.md`
- `docs/features/feature-cache-controls.md`
- `docs/features/feature-magic-login.md`
- `docs/features/feature-jobs-and-metrics.md`
- `docs/features/feature-wave4-dashboard-create-flows.md`
- `docs/features/feature-dashboard-ia-overhaul.md`
- `docs/features/feature-health-checks-and-rollback.md`
- `docs/features/feature-audit-logging.md`
- `docs/features/feature-job-control.md`
- `docs/features/feature-wp-first-runtime.md`

## How to Create

1. Copy `docs/templates/feature-spec-template.md`.
2. Fill all sections before implementation.
3. Reference governing docs from `docs/spec-index.md`.
