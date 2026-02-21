Status: active
Owner: platform
Last Reviewed: 2026-02-21
Depends On: AGENTS.md, PLAN.md, docs/plan-dependency-matrix.md, docs/testing.md
Supersedes: none

# Unattended Orchestration

This document defines how to execute the full MVP plan without user interaction.

## Objective

- Start one run and allow autonomous progression through plan waves.
- Remove runtime permission prompts for unattended operation.
- Preserve deterministic checkpoints via tracked repo state.

## Runtime Baseline

The repository has one canonical runtime config: `opencode.json`.

- It enables unattended execution with a minimal destructive-command denylist.
- It is loaded automatically when OpenCode runs from this repository.

## Execution Model

Unattended execution is performed inside an OpenCode session using command presets and subagents.

Progress persistence is done via tracked repo state:

- `PLAN.md` checkboxes
- `PROGRESS.md` stage notes
- Session handoff notes when substantial work is done (see `docs/templates/session-handoff-template.md`)

## Standard Execution

1. Start OpenCode from repository root (or open this repo in Desktop).
2. Start unattended execution: `/run-plan`.
3. If interrupted, start a new session and resume with `/resume-run`.
4. If a gate fails, run `/triage-failures`.

## Completion Criteria

- All `PLAN.md` tasks required for MVP are completed.
- Required gates pass for each implemented slice.
- No unresolved failed gate remains.

## Safety Model

- Runtime guardrails block catastrophic local shell patterns (force push/hard reset/root-level recursive deletes).
- Hard spec constraints remain mandatory in all sessions.
