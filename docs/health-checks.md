Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/job-execution.md, docs/technical-architecture.md
Supersedes: none

# Health Checks

This document defines the minimal health check contract for MVP.

## Goals

- Confirm an environment is usable after deploy, restore, or promotion.
- Provide a deterministic signal for automatic rollback.

## Required Checks

1. HTTP check: GET the environment primary domain or preview URL and require HTTP 200.
2. WP-CLI check: `wp core is-installed` must succeed in the environment context.
3. Database connectivity: a simple query must succeed (via WP-CLI or direct DB ping).

## Rules

- All required checks must pass for the release to be considered healthy.
- Each check must complete within a configured timeout (default: 10 seconds).
- Each check may be retried once before marking the release unhealthy.
- If checks fail during deploy, restore, or promotion, rollback is triggered automatically.
