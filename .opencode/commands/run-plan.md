---
description: Run full unattended plan with default config
agent: general
subtask: true
---
Execute `PLAN.md` end-to-end in this session with no user interaction.

Constraints:
- Use repository default runtime guardrails from `opencode.json`.
- Follow the repo loop: Spec -> Plan -> Act -> Verify.

Process:
1. Run `/readiness` and fix any failures.
2. Read `PLAN.md` and `PROGRESS.md`, then execute remaining work in dependency order.
3. For each non-trivial task:
   - Ask the Architect subagent for a path-scoped plan.
   - Delegate execution to the Implementer subagent (minimal, scoped changes).
   - Run `/backend-gates` and/or `/frontend-gates` as applicable.
   - Ask the Reviewer subagent to review against specs and acceptance criteria.
4. Persist progress as you go by updating `PLAN.md` checkboxes and `PROGRESS.md` stage notes.

Stop conditions:
- A gate fails repeatedly with no new information.
- A change would violate hard constraints in `AGENTS.md`.

Return:
1. Current completed milestone (PLAN wave/task).
2. Any blocking failures (command + error excerpt + implicated files).
3. Next concrete step to resume.
