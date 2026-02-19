Status: active
Owner: platform
Last Reviewed: 2026-02-20
Depends On: docs/spec-index.md, docs/state-machines.md, docs/job-execution.md
Supersedes: none

# Promotion and Drift

This document defines promotion rules, drift detection, and pushback safety for MVP.

## Goals

- Changes move up from non-production to production.
- Live data is protected by default.
- Pushbacks require drift validation and a fresh backup.

## Drift Detection

Drift is defined as production changes since the source environment was created.

Detection methods:

- Database drift: compare production table row counts and checksums for protected tables.
- Files drift: compare checksums for protected paths (uploads are always protected).

Drift statuses:

- `unknown` when no check has been performed.
- `clean` when protected data matches expectations.
- `drifted` when protected data differs.

Drift checks are recorded in `drift_checks` and `environments.last_drift_check_id` is updated.

## Promotion Presets

Presets define protected data during pushback.

### content-protect

Protect:

- Database tables: `wp_posts`, `wp_postmeta`, `wp_terms`, `wp_term_taxonomy`, `wp_term_relationships`, `wp_comments`, `wp_commentmeta`.
- Files: `wp-content/uploads`.

### commerce-protect

Protect:

- All `content-protect` tables and files.
- Database tables: `wp_woocommerce_orders`, `wp_woocommerce_order_items`, `wp_woocommerce_order_itemmeta`, `wp_wc_customer_lookup`, `wp_wc_order_stats`.

Notes:

- Table names use the environment's table prefix.
- Protected tables are never overwritten during promotion.

## Promotion Workflow

1. Validate drift against production using the preset.
2. Require a fresh production backup (full scope) before any pushback.
3. Apply promotion with selective data sync:
   - Files: exclude protected paths.
   - Database: exclude protected tables.
4. Run health checks (see docs/health-checks.md).
5. If health checks fail, rollback automatically.

## Warnings and Blocks

- If drift status is `drifted`, UI must display a blocking warning.
- If drift gate or backup gate is unmet, promotion must be blocked.
- No override path exists in MVP.
