---
description: Resume plan execution (alias)
agent: general
subtask: true
---
Resume unattended plan execution by reloading state from the repository.

Optional argument: a hint like `wave-1` or a `PLAN.md` task label.

Process:
1. Read `PLAN.md` and `PROGRESS.md` to find the next incomplete work.
2. Run `/readiness`.
3. Continue with the same loop as `/run-plan`.

Return:
1. What you resumed from (wave/task).
2. What you completed in this window.
3. Any new blocker and the exact failing verification command.
