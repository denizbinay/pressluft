# Spec Index

This index defines the minimal, explicit specification set for Pressluft MVP. The goal is to remove implementation ambiguity without overengineering.

## Documents

- docs/data-model.md
- docs/state-machines.md
- docs/job-execution.md
- docs/promotion-and-drift.md
- docs/backups-and-restore.md
- docs/health-checks.md
- docs/migration-spec.md
- docs/provisioning-spec.md
- docs/api-contract.md
- docs/ui-flows.md
- docs/security-and-secrets.md
- docs/ansible-execution.md
- docs/domain-and-routing.md
- docs/technical-architecture.md
- docs/user-requirements-and-workflows.md
- docs/vision-and-purpose.md

## Locked Decisions (MVP)

- Single admin user (no multi-user roles).
- Ubuntu 24.04 LTS only.
- Job execution guarantee: at-least-once with idempotent jobs.
- Concurrency: 1 job per site, 1 job per node.
- Backup retention: time-based only.
- Off-site backups required using S3-compatible storage.
- Promotion presets: content-protect and commerce-protect.
- Updates in scope: WordPress core, plugins, themes.
- Domain/TLS: LetsEncrypt HTTP-01 for custom domains, DNS-01 for preview wildcard cert. See `docs/domain-and-routing.md`.
- Observability: logs + basic metrics.
- UI/CLI: Web UI + API only.
- Migrations: all-in-one import archive.
- Rollback: automatic on health check failure.
- Health checks: HTTP 200 + wp-cli status + DB connectivity.

## Ownership Boundaries

- Go core: control plane, state machine, jobs, API, UI integration.
- Ansible playbooks: all node-targeted operations including provisioning, deployment, backups, restores, health checks, and drift checks. Invoked exclusively by the Go control plane as local subprocesses. See `docs/ansible-execution.md`.
- Nuxt webapp: web UI implementation that consumes the API.

## Glossary

- Site: A WordPress project managed by Pressluft.
- Environment: An isolated runtime for a site (production, staging, clone).
- Node: A managed Linux host accessed via SSH.
- Release: An immutable deployment snapshot for an environment.
- Backup: A stored copy of files and/or database.
- Job: A transactional operation executed by the control plane.
