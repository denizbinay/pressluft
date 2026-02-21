---
description: Run frontend install, lint, and build
agent: general
subtask: true
---
Run frontend validation gates from repository root.

Gate output:
!`if [ ! -d "web" ]; then echo "SKIP: web directory not present"; else cd web && pnpm install && pnpm lint && pnpm build; fi`

Return:
1. Overall status (pass/fail/skip).
2. Failing command and first actionable error.
3. Minimal next fix sequence.
