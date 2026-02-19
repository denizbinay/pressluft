Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: contracts/openapi.yaml, contracts/schemas/error.json, docs/api-contract.md, docs/job-execution.md, docs/ansible-execution.md
Supersedes: none

# Error Codes

This document defines the canonical error code registry for Pressluft MVP.

## Scope

- API errors use the shared `Error` shape from `contracts/schemas/error.json`.
- Non-generic API failure paths must use stable `code` values listed below.
- Job execution failures use `jobs.error_code` values listed below.

## Generic API Error Codes

Use these for cross-endpoint validation/auth/resource failures where no domain-specific code is required.

| Code | HTTP Status | Meaning |
|---|---|---|
| `bad_request` | 400 | Request payload or params are invalid. |
| `unauthorized` | 401 | Missing, expired, or revoked session. |
| `not_found` | 404 | Referenced resource does not exist. |
| `conflict` | 409 | Resource state/concurrency conflict blocks operation. |

## Endpoint-Specific API Error Codes

| Endpoint | Code | HTTP Status | Meaning |
|---|---|---|---|
| `POST /api/environments/:id/magic-login` | `environment_not_active` | 409 | Environment is not in `active` state. |
| `POST /api/environments/:id/magic-login` | `node_unreachable` | 502 | SSH connection failed or timed out. |
| `POST /api/environments/:id/magic-login` | `wp_cli_error` | 502 | WP-CLI command failed on target environment. |

## Job Error Codes (`jobs.error_code`)

| Code | Retryable | Source | Meaning |
|---|---|---|---|
| `JOB_TIMEOUT` | no | job executor | Job exceeded total timeout window. |
| `ANSIBLE_TIMEOUT` | yes | ansible subprocess | Ansible subprocess exceeded timeout and was terminated. |
| `ANSIBLE_PLAY_ERROR` | yes | ansible exit code 1 | General play error. |
| `ANSIBLE_HOST_FAILED` | yes | ansible exit code 2 | Host task failure. |
| `ANSIBLE_HOST_UNREACHABLE` | yes | ansible exit code 4 | Host unreachable. |
| `ANSIBLE_SYNTAX_ERROR` | no | ansible exit code 5 | Invalid playbook/options/syntax. |
| `ANSIBLE_UNEXPECTED_ERROR` | no | ansible exit code 250 | Unexpected Ansible runtime error. |
| `ANSIBLE_UNKNOWN_EXIT` | no | ansible other non-zero | Unknown Ansible failure. |

## Rules

- Do not introduce new API or job error codes without updating this registry in the same PR.
- Error codes are append-only for MVP stability. Do not rename existing codes.
- API handlers must not leak raw infrastructure output to clients.
