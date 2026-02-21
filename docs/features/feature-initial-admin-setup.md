Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/security-and-secrets.md, docs/features/feature-auth-session.md, docs/features/feature-installation-packaging.md
Supersedes: none

# FEATURE: initial-admin-setup

## Problem

Operators need a zero-manual way to access the control plane after install.
Developers need a local sandbox that "just works" with a predictable admin user.
On WSL2, local browser access should work out-of-the-box on `127.0.0.1`.

## Scope

- In scope:
  - Add CLI support for initializing the first (and only) admin user.
  - Seed a deterministic dev admin (`admin@local` / `0000`) in the local sandbox script.
  - Installer initializes the admin user on first install (idempotent).
  - Local sandbox listens in a WSL2-friendly way and prints correct URLs.
  - Provide a non-stuck fallback path from `/` to `/login` for the embedded dashboard.
- Out of scope:
  - Public web signup/registration endpoint.
  - Multi-user roles.
  - Password reset flows.

## Allowed Change Paths

- `cmd/pressluft/**`
- `internal/auth/**`
- `internal/admin/**`
- `scripts/dev-sandbox.sh`
- `install.sh`
- `web/pages/index.vue`
- `docs/features/feature-initial-admin-setup.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `none`

## Acceptance Criteria

1. On a fresh DB, `pressluft admin init` creates the first admin user exactly once and reports credentials (generated or provided).
2. Re-running `pressluft admin init` is idempotent and does not create additional users.
3. `scripts/dev-sandbox.sh` starts a runnable control plane and seeds `admin@local` / `0000` automatically.
4. On WSL2, after running `scripts/dev-sandbox.sh`, a Windows browser can reach the dashboard at `http://127.0.0.1:18080/`.
5. Visiting `/` never results in an indefinite dead-end: there is always a visible path to `/login`.

## Verification

- Required commands:
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `cd web && pnpm lint`
  - `cd web && pnpm build`
