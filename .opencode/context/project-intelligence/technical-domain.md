<!-- Context: project-intelligence/technical | Priority: critical | Version: 1.6 | Updated: 2026-02-25 -->

# Technical Domain

Pressluft is a single-binary Go application that embeds a static Nuxt dashboard and exposes provisioning APIs. The architecture now includes an `ops/` workspace plus orchestration scaffolding for job lifecycle, event streaming, and safe execution boundaries.

## Quick Reference

**Type**: Monolith (Go API + embedded Nuxt SPA)
**Core flow**: `make build` -> `nuxt generate` -> copy to `internal/server/dist/` -> Go build
**Current focus**: Provider-backed server creation + ops/orchestration foundation

## Primary Stack

| Layer | Technology | Version | Rationale |
|-------|-----------|---------|-----------|
| Backend | Go | 1.22 | Single binary runtime and predictable deployment |
| Frontend | Nuxt 4 + Vue 3 | ^4.3.1 / ^3.5.28 | Fast dashboard iteration and typed composables |
| Styling | Tailwind CSS v4 | ^4.2.0 | CSS-first tokens and utility workflows |
| Data | SQLite + Goose | modernc + v3.24.1 | Embedded, migration-backed local persistence |
| Cloud SDK | hcloud-go | v2.19.0 | Hetzner API support compatible with Go 1.22 |
| Ops assets | YAML + Ansible scaffold | internal | Auditable profile intent and convergence path |

## Architecture Pattern

- **API boundary**: `internal/server/*` handlers expose JSON endpoints; stores isolate persistence logic.
- **Provider model**: interface + registry (`Register/Get/All`) keeps provider extension simple.
- **Ops model**: `ops/` holds profile intent and convergence artifacts for ops contributors.
- **Orchestration model**: job state machine + persisted events/checkpoints under `internal/orchestrator` with SSE streaming.
- **Worker model**: `internal/worker` polls queued jobs and runs provisioning steps via the executor.
- **Execution boundary**: runner abstraction (`internal/runner`) isolates external tool invocation (Ansible guardrails).

## Project Structure (Core)

```text
pressluft/
├── cmd/main.go
├── internal/
│   ├── database/{database.go,migrations/*.sql}
│   ├── provider/{provider.go,provider_servers.go,hetzner/*}
│   ├── server/{handler*.go,store_servers.go,profiles/registry.go,dist/}
│   ├── orchestrator/{types.go,state_machine.go,store.go}
│   ├── worker/{worker.go,executor.go,adapters.go}
│   ├── runner/{runner.go,ansible/adapter.go}
│   ├── agentproto/types.go
│   └── events/types.go
├── ops/
│   ├── profiles/*/profile.yaml
│   ├── ansible/{playbooks,roles}
│   ├── schemas/profile.schema.json
│   ├── scripts/
│   └── tests/
└── web/app/{pages,components,composables}
```

## API Surface

| Method | Path | Description |
|--------|------|-------------|
| GET | `/api/health` | Service health status |
| GET/POST | `/api/providers` | Provider list/create |
| DELETE | `/api/providers/{id}` | Remove provider |
| POST | `/api/providers/validate` | Validate provider token |
| GET | `/api/providers/types` | List available provider types |
| GET/POST | `/api/servers` | Server list/create (creates provisioning job) |
| GET | `/api/servers/catalog?provider_id={id}` | Provider server catalog + profiles |
| GET | `/api/servers/profiles` | Profile registry metadata |
| GET | `/api/servers/{id}` | Server detail |
| DELETE | `/api/servers/{id}` | Remove server (and its jobs) |
| GET | `/api/servers/{id}/jobs` | Jobs tied to a server |
| GET/POST | `/api/jobs` | List/create orchestration jobs |
| GET | `/api/jobs/{id}` | Read orchestration job state |
| GET | `/api/jobs/{id}/events` | Stream job events (SSE) |
| GET | `/api/jobs/{id}/events/history` | Job event history (JSON) |

All APIs return JSON; errors use `{"error":"..."}`.

## Provisioning and Ops Patterns

### Profile Contract

- Canonical profiles live in `ops/profiles/*/profile.yaml`.
- `internal/server/profiles/registry.go` exposes API-safe profile metadata.
- Profiles include `base_image`, `image_policy`, service/hardening intent, hooks, and artifact references.

### Job Lifecycle Contract

State model (v1):

```text
queued -> preparing -> running -> waiting_reboot -> resuming -> verifying -> succeeded
running|resuming -> retrying -> running
active -> failed|cancelled|timed_out
```

Provisioning steps (current): `validate` → `create_ssh_key` → `create_server` → `wait_running` → `finalize`.

### Runner Guardrails

- Use explicit command args (`exec.CommandContext`), never shell strings.
- Pin working directory and allowlist playbooks.
- Run syntax-check path before apply path.
- Emit structured runner events for orchestration persistence.

## Frontend Patterns

- Nuxt 4 app under `web/app/` with composable-first state access.
- Servers UX uses guided modal, provider catalog, profile selection, and a provisioning timeline.
- Price labels are normalized to 2 decimal places for readable pricing.
- Jobs composable (`useJobs`) streams SSE events with polling fallback for timelines.

## Naming and Conventions

| Area | Convention | Example |
|------|------------|---------|
| Go packages | short lowercase | `provider`, `orchestrator` |
| API payload fields | snake_case JSON | `provider_id`, `profile_key` |
| Ops profile keys | kebab-case | `woocommerce-optimized` |
| Vue composables | `use*` camelCase | `useServers`, `useJobs` |

## Security and Reliability Baselines

- Provider API tokens are not serialized in API responses.
- Input validation is enforced at handler and store boundaries.
- SQLite uses WAL, foreign keys, and bounded connections.
- Orchestration state and events are persisted for auditability.
- Runner design enforces command safety boundaries.

## Development Commands

| Command | Purpose |
|---------|---------|
| `make dev` | Run Go API + Nuxt dev UI with proxy |
| `make test` | Run Go tests |
| `make build` | Generate UI + embed + build binary |
| `make check` | format + lint + test + build |

## 📂 Codebase References

- `internal/server/handler.go` - API route registration including jobs routes
- `internal/server/handler_servers.go` - server catalog/profile/create behavior
- `internal/server/handler_jobs.go` - orchestration job create/read/SSE handlers
- `internal/server/store_servers.go` - server persistence and status updates
- `internal/server/profiles/registry.go` - profile metadata exposed to API clients
- `internal/orchestrator/state_machine.go` - lifecycle transition rules
- `internal/orchestrator/store.go` - jobs/events persistence access
- `internal/worker/executor.go` - provisioning workflow execution
- `internal/worker/worker.go` - job polling and recovery loop
- `internal/provider/hetzner/servers.go` - Hetzner catalog + create server + SSH keys
- `internal/runner/ansible/adapter.go` - guardrailed Ansible execution scaffold
- `internal/database/migrations/00003_create_jobs.sql` - orchestration tables
- `ops/profiles/README.md` - profile authoring conventions
- `ops/schemas/profile.schema.json` - profile contract schema
- `web/app/components/SettingsServers.vue` - server create UX and price formatting
- `web/app/composables/useJobs.ts` - job API + event stream client scaffold

## Related Files

- `business-domain.md` - product purpose and user value
- `business-tech-bridge.md` - business-to-technical mapping
- `decisions-log.md` - architectural decision rationale
- `living-notes.md` - active priorities and known constraints
