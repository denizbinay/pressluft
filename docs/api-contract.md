# API Contract

This document defines the MVP API surface. All mutating actions are async and return a job.

## Auth

- Single admin user.
- Session-based auth using a secure session token.

## Conventions

- All responses are JSON.
- Errors include `code` and `message`.
- Mutations return `{ job_id }`.

## Endpoints

### Auth

- `POST /api/login`
  - Body: `{ email, password }`
  - Response: `{ session_token }`

- `POST /api/logout`
  - Response: `{ success: true }`

### Sites

- `GET /api/sites`
- `POST /api/sites`
  - Body: `{ name, slug }`
  - Response: `{ job_id }`

- `GET /api/sites/:id`
- `GET /api/sites/:id/environments`

- `POST /api/sites/:id/import`
  - Body: `{ archive_url }`
  - Response: `{ job_id }`

### Environments

- `POST /api/sites/:id/environments`
  - Body: `{ name, slug, type, source_environment_id, promotion_preset }`
  - Response: `{ job_id }`

- `GET /api/environments/:id`
- `POST /api/environments/:id/drift-check`
  - Response: `{ job_id }`
- `POST /api/environments/:id/deploy`
  - Body: `{ source_type, source_ref }`
  - source_type: `git | upload`
  - Response: `{ job_id }`

- `POST /api/environments/:id/updates`
  - Body: `{ scope }`
  - scope: `core | plugins | themes | all`
  - Response: `{ job_id }`

- `POST /api/environments/:id/restore`
  - Body: `{ backup_id }`
  - Response: `{ job_id }`

- `POST /api/environments/:id/promote`
  - Body: `{ target_environment_id }`
  - Response: `{ job_id }`

### Backups

- `POST /api/environments/:id/backups`
  - Body: `{ backup_scope }`
  - backup_scope: `db | files | full`
  - Response: `{ job_id }`

- `GET /api/environments/:id/backups`

### Domains

- `GET /api/environments/:id/domains`

- `POST /api/environments/:id/domains`
  - Body: `{ hostname }`
  - Response: `{ job_id }`

- `DELETE /api/domains/:id`
  - Response: `{ job_id }`

### Jobs

- `GET /api/jobs`
- `GET /api/jobs/:id`
  - Response: `{ id, status, error_code, error_message, started_at, finished_at }`

### Metrics

- `GET /api/metrics`
  - Response: `{ jobs_running, jobs_queued, nodes_active, sites_total }`
