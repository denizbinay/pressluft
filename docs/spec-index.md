Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: none
Supersedes: none

# Spec Index

This index defines the minimal, explicit specification set for Pressluft MVP. The goal is to remove implementation ambiguity without overengineering.

## Metadata Standard

All new and updated specs should include this header block at top of file:

```md
Status: draft | active | deprecated
Owner: <team or person>
Last Reviewed: YYYY-MM-DD
Depends On: <comma-separated paths or none>
Supersedes: <comma-separated paths or none>
```

Progressive adoption rule: existing specs may add this metadata during normal edits; no bulk rewrite required.

## Documents

### Root Routers

- SPEC.md
- ARCHITECTURE.md
- CONTRACTS.md

### Core (Always Load)

- docs/data-model.md
- docs/technical-architecture.md
- docs/job-execution.md
- docs/state-machines.md
- docs/security-and-secrets.md

### Contracts

- contracts/openapi.yaml
- contracts/schemas/error.json
- docs/api-contract.md
- docs/contract-guardrails.md
- docs/contract-traceability.md
- docs/error-codes.md
- docs/job-types.md
- docs/schema-authority.md

### Infrastructure and Operations

- docs/ansible-execution.md
- docs/provisioning-spec.md
- docs/domain-and-routing.md
- docs/migration-spec.md
- docs/backups-and-restore.md
- docs/health-checks.md
- docs/promotion-and-drift.md
- docs/migrations-guidelines.md
- docs/config-matrix.md
- docs/testing.md

### Product and UX

- docs/ui-flows.md
- docs/user-requirements-and-workflows.md
- docs/vision-and-purpose.md

### Agent Workflow

- docs/agent-governance.md
- docs/parallel-execution.md
- docs/spec-lifecycle.md
- docs/changes-workflow.md
- docs/agent-session-playbook.md
- docs/templates/feature-spec-template.md
- docs/templates/session-handoff-template.md
- docs/features/README.md
- docs/pre-plan-readiness.md

### Decision Records

- docs/adr/README.md

## Locked Decisions (MVP)

- Single admin user (no multi-user roles).
- Ubuntu 24.04 LTS only.
- Job execution guarantee: at-least-once with idempotent jobs.
- Concurrency: 1 job per site, 1 job per node.
- Node queries (non-mutating): direct SSH from Go binary, bypass job queue. See `docs/technical-architecture.md`.
- Backup retention: time-based only.
- Off-site backups required using S3-compatible storage.
- Promotion presets: content-protect and commerce-protect.
- Updates in scope: WordPress core, plugins, themes.
- Domain/TLS: LetsEncrypt HTTP-01 for custom domains, DNS-01 for preview wildcard cert. See `docs/domain-and-routing.md`.
- Auth transport: cookie session (`session_token`) for protected API access.
- Caching: Redis object cache + Nginx FastCGI page cache, per-environment toggles, enabled by default. See `docs/provisioning-spec.md` and `docs/domain-and-routing.md`.
- Security hardening: Fail2Ban + 7G WAF + PHP hardening + security headers, always-on at node level. See `docs/provisioning-spec.md` and `docs/security-and-secrets.md`.
- Magic login: one-time WordPress admin URL via node query. See `docs/security-and-secrets.md`.
- Observability: logs + basic metrics.
- UI/CLI: Web UI + API only.
- Migrations: all-in-one import archive.
- Rollback: automatic on health check failure.
- Health checks: HTTP 200 + wp-cli status + DB connectivity.

## Ownership Boundaries

- Go core: control plane, state machine, jobs, API, UI integration, direct SSH for node queries (`internal/ssh`).
- Ansible playbooks: all node-targeted **mutations** including provisioning, deployment, backups, restores, health checks, and drift checks. Invoked exclusively by the Go control plane as local subprocesses. See `docs/ansible-execution.md`.
- Nuxt webapp: web UI implementation that consumes the API.

## Session Rules

- Every implementation task must reference one feature spec under `docs/features/`.
- Every feature spec must declare in-scope paths, out-of-scope changes, acceptance criteria, and test requirements.
- Session startup should load `PLAN.md` and `PROGRESS.md` before implementation.
- API behavior is canonical in `contracts/openapi.yaml`; `docs/api-contract.md` is explanatory.
- Every OpenAPI endpoint must map to exactly one owning feature spec in `docs/contract-traceability.md`.
- Keep per-session context minimal: core docs + one feature spec + relevant domain docs.
- Substantial sessions must produce a structured handoff using `docs/templates/session-handoff-template.md`.
- Major changes must use `changes/<slug>/` as defined in `docs/changes-workflow.md`.

## Glossary

- Site: A WordPress project managed by Pressluft.
- Environment: An isolated runtime for a site (production, staging, clone).
- Node: A managed Linux host accessed via SSH.
- Release: An immutable deployment snapshot for an environment.
- Backup: A stored copy of files and/or database.
- Job: A transactional operation executed by the control plane.
