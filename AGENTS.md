# AGENTS.md

Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md
Supersedes: none

This is the execution contract for coding agents in this repository.

## 1. Core Workflow (Required)

Use this loop for every non-trivial task:

1. Spec: identify governing docs from `docs/spec-index.md`.
2. Plan: state files to touch and decisions to make.
3. Act: implement minimal scoped changes.
4. Verify: run relevant commands and map results to acceptance criteria.

Do not skip this loop.

## 2. Hard Constraints

- Database is the source of truth for state.
- Infrastructure mutations must run through job queue and Ansible.
- Concurrency invariant: max 1 mutation job per site and max 1 per node.
- Nodes, including localhost, are managed through SSH.
- Releases are immutable and deployed via atomic symlink switch.
- Mutable resources use optimistic concurrency via `state_version`.
- State transitions happen inside DB transactions.

## 3. Commands (Execute Exactly)

```sh
# Backend
go build -o ./bin/pressluft ./cmd/pressluft
go test ./internal/... -v
go vet ./...

# Frontend (from repo root)
cd web && pnpm install
cd web && pnpm lint
cd web && pnpm build
cd web && pnpm dev

# Database
go run ./migrations/migrate.go up
```

Validation gates:
- Before commit: `go build` and `go vet` must pass.
- Before PR: `go test` must pass.

## 4. Stack and Versions

- Go 1.22
- Router: `github.com/go-chi/chi/v5`
- SQLite via `modernc.org/sqlite` (no CGo)
- Nuxt 3 + Vue 3 + TypeScript
- Node.js >= 20 LTS
- pnpm >= 9
- Ansible >= 2.16
- Ubuntu 24.04 LTS target only

Go uses stdlib-first approach. Frontend uses `<script setup lang="ts">` only.

## 5. Canonical Specs

Start at `docs/spec-index.md`. For implementation, load only needed docs:

- Root routers: `SPEC.md`, `ARCHITECTURE.md`, `CONTRACTS.md`
- Core: `docs/technical-architecture.md`, `docs/data-model.md`, `docs/job-execution.md`
- Contracts: `contracts/openapi.yaml`, `docs/api-contract.md`, `docs/contract-guardrails.md`, `docs/contract-traceability.md`, `docs/error-codes.md`, `docs/job-types.md`
- Infra: `docs/ansible-execution.md`, `docs/provisioning-spec.md`
- Security: `docs/security-and-secrets.md`
- Schema authority: `docs/schema-authority.md`, `docs/migrations-guidelines.md`
- Decision records: `docs/adr/README.md`
- Session guide: `docs/agent-session-playbook.md`

## 6. When to Ask First

- Adding new dependencies (Go or npm)
- Changing DB schema or files in `migrations/`
- Adding new API endpoints
- Changing Ansible playbook structure

## 7. Never Do

- Do not bypass job queue for infrastructure mutations.
- Do not introduce CGo.
- Do not commit secrets, keys, or `.env` files.
- Do not do large refactors without a written spec.
- Do not invent endpoints/schema behavior outside specs.

## 8. Conventions

Go:
- `gofmt` required
- no `init()`
- return wrapped errors (`fmt.Errorf("context: %w", err)`)
- tests beside implementation files

TypeScript/Vue:
- typed props/emits
- avoid `any`
- `kebab-case.vue` components, `camelCase.ts` composables

SQL migrations:
- sequential timestamped names
- reversible or explicitly marked irreversible
- no data-dependent DDL

## 9. Session Output Requirements

For substantial tasks, agent output must include:
- governing specs used
- plan of touched paths
- what changed and why
- verification commands and result summary
