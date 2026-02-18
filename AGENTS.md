# AGENTS.md

## 1. Project Context & Stack
- **Backend:** Go 1.22 (Standard Library + Chi router).
- **Frontend:** Nuxt 3 (Vue 3, TypeScript).
- **Infra:** Ansible (Playbooks in `/ansible`).
- **Docs:** Markdown-based specs in `/docs`.

## 2. Operational Commands (Execute these exactly)
- **Backend Build:** `go build -o ./bin/pressluft ./cmd/pressluft`
- **Backend Test:** `go test ./internal/... -v`
- **Frontend Dev:** `cd web && pnpm dev`
- **Frontend Lint:** `cd web && pnpm lint`
- **Database Migrations:** `go run ./migrations/migrate.go up`

## 3. Work Style (Authoritative)
**You are the gatekeeper for spec integrity.**
- **Spec-First:** There are only two valid paths: implement after specs, or change specs to enable work.
- **Source of Truth:** Always compare decisions against `/docs`. If a user request conflicts with `/docs`, push back and explain the contradiction.
- **Minimalism:** Keep implementation minimal and explicit. Prefer boring/predictable over clever.
- **Scope:** Keep the whole app in mind, not just the single request.

## 4. Boundaries & Workflow
### Always
- Read the last couple of git diffs before starting to understand the repo state.
- Write complete git commit messages explaining *why* a change was made.
- Update `/docs` if architecture decisions change.

### Ask First
- Before adding new dependencies (Go or NPM).
- Before modifying database schemas (`/migrations`).

### Never
- Never change specs unilaterally.
- Never perform large refactors without a written spec in `/docs`.

## 5. Repository Structure
- `cmd/pressluft/` (Main entrypoint)
- `internal/` (Go core logic)
- `web/` (Nuxt 3 app)
- `ansible/` (Playbooks, roles, inventories)
- `docs/` (Source of truth: specs and architecture)
- `scripts/` (Installers)
- `migrations/` (SQL migrations)
- `configs/` (Example config files)
