Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: docs/spec-index.md, docs/technical-architecture.md, docs/job-execution.md, docs/ansible-execution.md, docs/security-and-secrets.md, docs/config-matrix.md, docs/features/feature-install-bootstrap.md
Supersedes: none

# FEATURE: installation-packaging

## Problem

Operators need a true zero-manual installation path on a fresh Ubuntu 24.04 host.
The installer must download prebuilt release artifacts (no on-host build toolchains), install runtime prerequisites, configure systemd services, and start a runnable Pressluft control plane.

Developers also need a disposable local sandbox mode that does not leave persistent DB/secrets artifacts unless explicitly requested.

## Scope

- In scope:
  - Prebuilt release artifacts (GitHub Releases) for linux/amd64 and linux/arm64.
  - Curlable `install.sh` that downloads artifacts, verifies checksums, installs runtime prerequisites, installs systemd units, runs migrations, and starts services.
  - Control plane runtime layout under `/opt/pressluft` and state under `/var/lib/pressluft`.
  - Long-running worker process (`pressluft worker`) that polls and executes queued jobs.
  - `pressluft migrate up` command so installs/upgrades do not require `go run`.
  - Local sandbox script that runs Pressluft with ephemeral DB + secrets directories.
- Out of scope:
  - Non-Ubuntu installation flows.
  - Marketplace images.
  - Multi-node production hardening (remote SSH keys, inventory encryption).

## Allowed Change Paths

- `install.sh`
- `packaging/**`
- `.github/workflows/**`
- `cmd/pressluft/**`
- `internal/**`
- `migrations/**`
- `ansible/**`
- `scripts/**`
- `docs/testing.md`
- `docs/config-matrix.md`
- `docs/features/feature-*.md`
- `PLAN.md`
- `PROGRESS.md`

## Contract Impact

- API: `none`
- DB schema: `none`
- Infra/playbooks: `update-required`

Contract/spec files:

- `docs/technical-architecture.md`
- `docs/ansible-execution.md`
- `docs/job-execution.md`
- `docs/config-matrix.md`

## Acceptance Criteria

1. Running `curl -fsSL https://raw.githubusercontent.com/denizbinay/pressluft/main/install.sh | sudo bash` on a fresh Ubuntu 24.04 host installs a runnable Pressluft control plane without requiring Go/Node.
2. The installer downloads prebuilt artifacts from GitHub Releases, verifies integrity via SHA-256, and installs to a versioned `/opt/pressluft/releases/<version>/` path with an atomic `current` symlink.
3. systemd services (`pressluft-api`, `pressluft-worker`) are installed, enabled, and started; services run as non-root user `pressluft`.
4. `pressluft migrate up` is idempotent and is executed automatically before service start.
5. Installer reruns are idempotent: no duplicate systemd units, no duplicate local node records, stable `current` symlink.
6. A local sandbox script exists that runs against ephemeral DB/secrets paths by default and cleans up on exit.

## Scenarios (WHEN/THEN)

1. WHEN `install.sh` runs on a fresh Ubuntu 24.04 host THEN Pressluft installs successfully and services start.
2. WHEN `PRESSLUFT_VERSION` is set THEN `install.sh` installs the pinned release version.
3. WHEN `install.sh` is re-run THEN the installation converges to the expected state without duplicating services or node records.
4. WHEN checksum validation fails THEN installation aborts without starting services and prints safe rollback guidance.
5. WHEN the local sandbox is started THEN Pressluft runs with disposable DB + secrets paths and leaves no artifacts on normal exit.

## Verification

- Required commands:
  - `bash scripts/check-readiness.sh`
  - `go build -o ./bin/pressluft ./cmd/pressluft`
  - `go vet ./...`
  - `go test ./internal/... -v`
  - `ansible-playbook --syntax-check ansible/playbooks/node-provision.yml`
- Required tests:
  - Worker loop unit tests for polling and shutdown behavior.
  - Install script smoke on a fresh Ubuntu 24.04 host (manual/CI).

## Risks and Rollback

- Risk: partial install can leave `current` symlink or systemd units inconsistent.
- Rollback: stop services, remove `/opt/pressluft/current` symlink and the installed release directory, remove `/etc/systemd/system/pressluft-*.service`, and re-run the installer.
