Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: contracts/openapi.yaml, docs/features/README.md, docs/job-execution.md, docs/state-machines.md, docs/ansible-execution.md, docs/job-types.md
Supersedes: none

# Contract Traceability

This matrix maps API endpoints to owning feature specs and execution contracts.

Path note: matrix endpoints include the runtime `/api` prefix. OpenAPI paths omit `/api` because the OpenAPI server base URL is `/api`.

## Endpoint Ownership Matrix

| Endpoint | Feature Spec | Execution Model | Job Type / Query | Primary State Surface |
|---|---|---|---|---|
| `POST /login` | `docs/features/feature-auth-session.md` | sync | none | auth session |
| `POST /logout` | `docs/features/feature-auth-session.md` | sync | none | auth session |
| `GET /sites` | `docs/features/feature-site-create.md` | sync | none | site read model |
| `POST /sites` | `docs/features/feature-site-create.md` | async | `site_create` | site/environment |
| `GET /sites/{id}` | `docs/features/feature-site-create.md` | sync | none | site read model |
| `GET /sites/{id}/environments` | `docs/features/feature-environment-create-clone.md` | sync | none | environment read model |
| `POST /sites/{id}/environments` | `docs/features/feature-environment-create-clone.md` | async | `env_create` | site/environment |
| `POST /sites/{id}/import` | `docs/features/feature-site-import.md` | async | `site_import` | site/environment |
| `GET /environments/{id}` | `docs/features/feature-environment-create-clone.md` | sync | none | environment read model |
| `POST /environments/{id}/drift-check` | `docs/features/feature-promotion-drift.md` | async | `drift_check` | environment drift |
| `POST /environments/{id}/deploy` | `docs/features/feature-environment-deploy-updates.md` | async | `env_deploy` | site/environment |
| `POST /environments/{id}/updates` | `docs/features/feature-environment-deploy-updates.md` | async | `env_update` | site/environment |
| `POST /environments/{id}/restore` | `docs/features/feature-environment-restore.md` | async | `env_restore` | site/environment |
| `POST /environments/{id}/promote` | `docs/features/feature-promotion-drift.md` | async | `env_promote` | environment drift/promotion |
| `PATCH /environments/{id}/cache` | `docs/features/feature-cache-controls.md` | async | `env_cache_toggle` | environment cache settings |
| `POST /environments/{id}/cache/purge` | `docs/features/feature-cache-controls.md` | async | `cache_purge` | environment cache runtime |
| `POST /environments/{id}/magic-login` | `docs/features/feature-magic-login.md` | sync | node query | environment access |
| `GET /environments/{id}/backups` | `docs/features/feature-backups.md` | sync | none | backup read model |
| `POST /environments/{id}/backups` | `docs/features/feature-backups.md` | async | `backup_create` | backup lifecycle |
| `GET /environments/{id}/domains` | `docs/features/feature-domains-and-tls.md` | sync | none | domain read model |
| `POST /environments/{id}/domains` | `docs/features/feature-domains-and-tls.md` | async | `domain_add` | domain/tls status |
| `DELETE /domains/{id}` | `docs/features/feature-domains-and-tls.md` | async | `domain_remove` | domain/tls status |
| `GET /jobs` | `docs/features/feature-jobs-and-metrics.md` | sync | none | job read model |
| `GET /jobs/{id}` | `docs/features/feature-jobs-and-metrics.md` | sync | none | job read model |
| `GET /metrics` | `docs/features/feature-jobs-and-metrics.md` | sync | none | metrics read model |

## Invariants

- All async mutation endpoints must map to exactly one `job_type`.
- Synchronous node-query endpoints must be explicitly listed and justified.
- No endpoint is implementation-ready unless it has an owning feature spec.
- All canonical job types in `docs/job-types.md` must have an owning feature spec, including internal orchestration job types not directly exposed as endpoints.
