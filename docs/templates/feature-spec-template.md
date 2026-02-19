Status: active
Owner: platform
Last Reviewed: 2026-02-19
Depends On: docs/spec-index.md, docs/spec-lifecycle.md
Supersedes: none

# Feature Spec Template

Use this file as the required template for implementation-bound work.

---

Status: draft
Owner: <team or person>
Last Reviewed: YYYY-MM-DD
Depends On: <comma-separated paths>
Supersedes: none

# FEATURE: <short-name>

## Problem

Describe user-visible problem and why this work exists.

## Scope

- In scope:
  - <item>
- Out of scope:
  - <item>

## Allowed Change Paths

- `<path/glob-1>`
- `<path/glob-2>`

## Contract Impact

- API: `none | update-required`
- DB schema: `none | update-required`
- Infra/playbooks: `none | update-required`

If update is required, list exact contract/spec files.

## Acceptance Criteria

1. <behavioral criterion>
2. <behavioral criterion>
3. <behavioral criterion>

## Verification

- Required commands:
  - `<command>`
- Required tests:
  - `<test file or suite>`

## Risks and Rollback

- Risk: <item>
- Rollback: <how to revert safely>
