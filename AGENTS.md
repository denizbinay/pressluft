# AGENTS.md

## 1. Commands (Execute Exactly)

```sh
# Backend
go build -o ./bin/pressluft ./cmd/pressluft   # Build
go test ./internal/... -v                      # Test
go vet ./...                                   # Static analysis

# Frontend (from repo root)
cd web && pnpm install                         # Install deps
cd web && pnpm lint                            # Lint
cd web && pnpm build                           # Production build
cd web && pnpm dev                             # Dev server

# Database
go run ./migrations/migrate.go up              # Run migrations
```

Validation gate: `go build` and `go vet` must pass before any commit. `go test` must pass before any PR.

## 2. Tech Stack (Exact Versions)

| Layer       | Technology                 | Version / Constraint          |
|-------------|----------------------------|-------------------------------|
| Backend     | Go                         | 1.22                          |
| Router      | go-chi/chi                 | v5                            |
| Database    | SQLite (pure Go driver)    | modernc.org/sqlite            |
| Frontend    | Nuxt                       | 3.x (Vue 3, TypeScript)      |
| Runtime     | Node.js                    | >= 20 LTS                     |
| Pkg Manager | pnpm                       | >= 9                          |
| Infra       | Ansible                    | >= 2.16                       |
| Target OS   | Ubuntu                     | 24.04 LTS only                |

Go code uses the standard library wherever possible. Chi is the only router dependency. No CGo.

Frontend uses the Composition API (`<script setup lang="ts">`) exclusively. No Options API.

## 3. Spec-Driven Workflow

**You are the gatekeeper for spec integrity.**

The authoritative spec index is `docs/spec-index.md`. All specs live in `/docs`. Every implementation decision must trace to a spec. If it doesn't, the spec is written first.

### The Loop: Spec → Plan → Act → Verify

1. **Spec:** Identify the spec(s) in `/docs` that govern the work. Read them. If no spec exists or the request contradicts a spec, stop and resolve the gap before writing code.
2. **Plan:** State which spec sections you are implementing and outline the approach (files to create/modify, key decisions). This can be brief but must be explicit.
3. **Act:** Implement. Keep changes minimal and predictable. Prefer boring over clever.
4. **Verify:** Confirm the implementation satisfies the spec. Run the relevant commands from Section 1.

Skipping steps is not permitted. If the user asks to "just do it," push back and explain the loop.

### Key Architectural Invariants (from specs)

These are load-bearing rules. Violating any of them requires a spec change first.

- The database is the single source of truth for all state.
- All infrastructure mutations go through the job queue. No ad-hoc SSH.
- One mutation per site at a time (concurrency: 1 job/site, 1 job/node).
- All nodes (including localhost) are accessed via SSH. No special local-execution path.
- Releases are immutable. Deployment uses atomic symlink switching.
- State transitions occur inside database transactions.
- Optimistic concurrency via `state_version` on mutable resources.

## 4. Boundaries

### Always

- Read recent `git log` / `git diff` before starting work to understand repo state.
- Run `go build` and `go vet` before committing.
- Write commit messages explaining *why*, not just *what*.
- Reference the governing spec document in commit messages or PR descriptions.
- Update `/docs` when architecture decisions change.
- Keep the whole application in mind, not just the immediate request.

### Ask First

- Before adding any new dependency (Go module or npm package).
- Before modifying database schemas or migration files (`/migrations`).
- Before creating new API endpoints (the API contract is specced in `docs/api-contract.md`).
- Before changing Ansible playbook structure.

### Never

- Never change specs unilaterally. Propose changes, explain the reasoning, get confirmation.
- Never perform large refactors without a written spec in `/docs`.
- Never commit secrets, `.env` files, private keys, or credentials.
- Never introduce CGo dependencies.
- Never bypass the job queue for infrastructure mutations.
- Never generate code without first identifying the governing spec and stating a plan.

## 5. Repository Structure

Directories marked with `*` are planned but may not yet exist.

```
pressluft/
├── cmd/pressluft/           # Main entrypoint (single binary)
│   └── main.go*
├── internal/                # All Go application code (not importable externally)
│   ├── api/                 #   HTTP handlers and middleware (Chi)
│   ├── model/               #   Domain types, enums, validation
│   ├── store/               #   SQLite repository layer
│   ├── job/                 #   Job queue, executor, scheduling
│   ├── ssh/                 #   SSH execution layer
│   └── service/             #   Business logic orchestrating store + jobs
├── web/                     # Nuxt 3 frontend
│   ├── pages/               #   File-based routing
│   ├── components/          #   Vue components
│   ├── composables/         #   Shared composition functions
│   ├── layouts/             #   Layout templates
│   ├── server/              #   Nuxt server routes (if any)
│   └── nuxt.config.ts       #   Nuxt configuration
├── ansible/                 # Ansible playbooks and roles
│   ├── playbooks/           #   Playbook files
│   ├── roles/               #   Reusable roles
│   └── inventories/         #   Host inventories
├── docs/                    # Source of truth: specs and architecture
│   ├── spec-index.md        #   START HERE — index of all specs
│   ├── technical-architecture.md
│   ├── data-model.md
│   ├── state-machines.md
│   ├── api-contract.md
│   ├── job-execution.md
│   └── ...                  #   (see spec-index.md for full list)
├── migrations/              # SQL migration files
├── scripts/                 # Install and bootstrap scripts
├── configs/                 # Example/default configuration files
├── AGENTS.md                # This file
└── README.md
```

## 6. Code Conventions

**Go:**
- Format with `gofmt`. No exceptions.
- Errors are returned, not panicked. Wrap with `fmt.Errorf("context: %w", err)`.
- Package names are short, lowercase, singular (`store`, not `stores`).
- No `init()` functions. Explicit initialization in `main.go`.
- Tests live next to the code they test (`foo_test.go` beside `foo.go`).

**TypeScript / Vue:**
- `<script setup lang="ts">` only. No Options API.
- Props and emits are typed. No `any` unless unavoidable and commented.
- File names: `kebab-case.vue` for components, `camelCase.ts` for composables.

**SQL Migrations:**
- Sequential, timestamped filenames.
- Every migration must be reversible or documented as irreversible.
- No data-dependent DDL (don't hardcode generated IDs).
