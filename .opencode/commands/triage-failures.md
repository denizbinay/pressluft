---
description: Triage failures after an unattended attempt
agent: general
subtask: true
---
Triage the current repository state after a failed unattended attempt.

Process:
1. Run `/readiness`.
2. Run `/backend-gates` and/or `/frontend-gates` as applicable.
3. Inspect `git status` and `git diff` to identify likely culprit changes.
4. Produce a minimal fix plan and the next verification command to run.

Return:
1. Failing command(s) and key error excerpt(s).
2. Suspected cause (files/changes).
3. Minimal remediation sequence.
