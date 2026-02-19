Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/data-model.md, docs/domain-and-routing.md, docs/security-and-secrets.md
Supersedes: none

# Config Matrix

This document is the canonical configuration inventory for control plane and node operations.

## Rules

- Do not hardcode environment-specific values in code.
- New config keys must be documented here before implementation.
- Secrets must never be stored in plaintext in DB rows; only encrypted references or encrypted blobs in the secrets store.

## Control Plane Settings (settings table)

| Key | Type | Required | Default | Secret | Notes |
|-----|------|----------|---------|--------|-------|
| `control_plane_domain` | string or null | no | null | no | Enables HTTPS endpoint configuration for panel/API |
| `preview_domain` | string or null | no | null | no | Enables wildcard preview domain mode |
| `dns01_provider` | string or null | conditional | null | no | Required when `preview_domain` is set |
| `dns01_credentials_json` | encrypted JSON reference | conditional | null | yes | Required when `preview_domain` is set |

## Runtime Environment Variables (Control Plane Process)

| Name | Type | Required | Default | Secret | Notes |
|------|------|----------|---------|--------|-------|
| `PRESSLUFT_LISTEN_ADDR` | string | no | `:8080` | no | Bind address for UI/API |
| `PRESSLUFT_DB_PATH` | string | no | `/var/lib/pressluft/pressluft.db` | no | SQLite DB path |
| `PRESSLUFT_SECRETS_DIR` | string | no | `/var/lib/pressluft/secrets` | yes | Encrypted secret store location |
| `PRESSLUFT_LOG_DIR` | string | no | `/var/log/pressluft` | no | Control plane logs |

## Node-Level Defaults

| Setting | Value | Notes |
|---------|-------|-------|
| OS | Ubuntu 24.04 LTS | Only supported target |
| Redis bind | `127.0.0.1` | No external exposure |
| Redis DB | `0` | Environment isolation via key prefix |
| FastCGI cache path | `/var/cache/nginx/fastcgi` | Global Nginx `http` config |

## Validation Requirements

- If `preview_domain` is set, both `dns01_provider` and `dns01_credentials_json` must be present.
- If `control_plane_domain` is unset, HTTP on IP:port is allowed.
- Invalid or incomplete config must return structured API errors and block related operations.
