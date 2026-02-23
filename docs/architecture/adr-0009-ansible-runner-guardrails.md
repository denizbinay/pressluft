# ADR-0009: Ansible Runner Guardrails in Go

- Status: accepted
- Date: 2026-02-23

## Context

Running external tooling from Go introduces command injection and secret-handling risks if execution boundaries are weak.

## Decision

Pressluft runner integrations must follow strict guardrails:
- Use `exec.CommandContext` with explicit args (no shell strings).
- Pin executable path and working directory.
- Allowlist playbook and inventory paths.
- Keep host key checking enabled.
- Support `--syntax-check` (and optional check-mode) before apply.
- Treat logs/events as structured output for dashboard use.

## Consequences

### Positive
- Safer execution defaults for provisioning and maintenance jobs.
- Better diagnostics and auditability.

### Trade-offs
- Additional validation and plumbing in runner adapters.
- Requires careful test coverage around process execution boundaries.
